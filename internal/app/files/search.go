package files

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SearchResult represents a single search result
type SearchResult struct {
	Path         string    // Full path to the file
	RelativePath string    // Path relative to search root
	Name         string    // File name
	Size         int64     // File size in bytes
	ModTime      time.Time // Last modification time
	IsDir        bool      // Whether this is a directory
	FileType     FileType  // Detected file type
	MatchType    MatchType // How this result matched the search
}

// MatchType indicates how a search result matched the query
type MatchType int

const (
	MatchFileName  MatchType = iota // Matched by filename
	MatchPath                       // Matched by path
	MatchContent                    // Matched by file content
	MatchExtension                  // Matched by file extension
)

// String returns the string representation of the match type
func (mt MatchType) String() string {
	switch mt {
	case MatchFileName:
		return "filename"
	case MatchPath:
		return "path"
	case MatchContent:
		return "content"
	case MatchExtension:
		return "extension"
	default:
		return "unknown"
	}
}

// SearchOptions configures the search behavior
type SearchOptions struct {
	// Pattern is the search pattern (can be regex or glob)
	Pattern string
	// IsRegex indicates if the pattern should be treated as a regular expression
	IsRegex bool
	// CaseSensitive determines if the search is case-sensitive
	CaseSensitive bool
	// IncludeHidden determines if hidden files should be included
	IncludeHidden bool
	// MaxResults limits the maximum number of results
	MaxResults int
	// MaxDepth limits the maximum directory depth to search
	MaxDepth int
	// FileTypes filters results to specific file types
	FileTypes []FileType
	// Extensions filters results to specific file extensions
	Extensions []string
	// MinSize filters files by minimum size in bytes
	MinSize int64
	// MaxSize filters files by maximum size in bytes
	MaxSize int64
	// ModifiedAfter filters files modified after this time
	ModifiedAfter *time.Time
	// ModifiedBefore filters files modified before this time
	ModifiedBefore *time.Time
	// SearchContent determines if file contents should be searched
	SearchContent bool
	// ContentPattern is the pattern to search for in file contents
	ContentPattern string
}

// DefaultSearchOptions returns sensible default search options
func DefaultSearchOptions() *SearchOptions {
	return &SearchOptions{
		IsRegex:       false,
		CaseSensitive: false,
		IncludeHidden: false,
		MaxResults:    1000,
		MaxDepth:      10,
		MaxSize:       10 * 1024 * 1024, // 10MB
		SearchContent: false,
	}
}

// FileSearcher provides file search capabilities with security validation
type FileSearcher struct {
	validator *SecurityValidator
	detector  *SyntaxDetector
}

// NewFileSearcher creates a new file searcher with the given security validator
func NewFileSearcher(validator *SecurityValidator) *FileSearcher {
	if validator == nil {
		validator = NewSecurityValidator(nil)
	}

	return &FileSearcher{
		validator: validator,
		detector:  NewSyntaxDetector(),
	}
}

// Search performs a file search with the given options
func (fs *FileSearcher) Search(ctx context.Context, rootPath string, options *SearchOptions) ([]*SearchResult, error) {
	if options == nil {
		options = DefaultSearchOptions()
	}

	// Validate the root path
	if err := fs.validator.ValidateDirectory(rootPath); err != nil {
		return nil, fmt.Errorf("invalid search root: %w", err)
	}

	// Compile regex pattern if needed with complexity limits
	var pattern *regexp.Regexp
	var err error
	if options.IsRegex {
		// Prevent ReDoS attacks by limiting pattern complexity
		if len(options.Pattern) > 1000 {
			return nil, fmt.Errorf("regex pattern too long (max 1000 characters)")
		}

		flags := ""
		if !options.CaseSensitive {
			flags = "(?i)"
		}
		pattern, err = regexp.Compile(flags + options.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	results := make([]*SearchResult, 0)
	visited := make(map[string]bool)

	err = fs.walkDirectory(ctx, rootPath, rootPath, 0, options, pattern, &results, visited)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Sort results by relevance and name
	sort.Slice(results, func(i, j int) bool {
		// Prefer exact filename matches over path matches
		if results[i].MatchType != results[j].MatchType {
			return results[i].MatchType < results[j].MatchType
		}
		// Then sort by name
		return results[i].Name < results[j].Name
	})

	// Limit results if specified
	if options.MaxResults > 0 && len(results) > options.MaxResults {
		results = results[:options.MaxResults]
	}

	return results, nil
}

// walkDirectory recursively walks through directories and searches for matches
func (fs *FileSearcher) walkDirectory(
	ctx context.Context,
	rootPath, currentPath string,
	depth int,
	options *SearchOptions,
	pattern *regexp.Regexp,
	results *[]*SearchResult,
	visited map[string]bool,
) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check depth limits
	if depth > options.MaxDepth {
		return nil
	}

	// Prevent infinite loops from symlinks and validate symlink targets
	absPath, err := filepath.Abs(currentPath)
	if err != nil {
		return err
	}
	if visited[absPath] {
		return nil
	}

	// Validate symlink targets before following them
	if info, err := os.Lstat(currentPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(currentPath)
		if err != nil {
			return nil // Skip broken symlinks
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(currentPath), target)
		}
		// Validate that symlink target is within allowed paths
		if err := fs.validator.ValidatePath(target); err != nil {
			return nil // Skip symlinks pointing outside allowed paths
		}
	}

	visited[absPath] = true

	// Read directory entries
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		// Skip directories we can't read instead of failing
		return nil
	}

	for _, entry := range entries {
		fullPath := filepath.Join(currentPath, entry.Name())

		// Skip hidden files if not included
		if !options.IncludeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Validate path access
		if err := fs.validator.ValidatePath(fullPath); err != nil {
			continue // Skip inaccessible paths
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Check if this matches our search criteria
		if match := fs.checkMatch(rootPath, fullPath, entry.Name(), info, options, pattern); match != nil {
			*results = append(*results, match)

			// Check if we've reached the result limit
			if options.MaxResults > 0 && len(*results) >= options.MaxResults {
				return nil
			}
		}

		// Recursively search subdirectories
		if entry.IsDir() {
			err := fs.walkDirectory(ctx, rootPath, fullPath, depth+1, options, pattern, results, visited)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// checkMatch determines if a file/directory matches the search criteria
func (fs *FileSearcher) checkMatch(
	rootPath, fullPath, name string,
	info fs.FileInfo,
	options *SearchOptions,
	pattern *regexp.Regexp,
) *SearchResult {
	// Apply size filters
	if !info.IsDir() {
		if options.MinSize > 0 && info.Size() < options.MinSize {
			return nil
		}
		if options.MaxSize > 0 && info.Size() > options.MaxSize {
			return nil
		}
	}

	// Apply time filters
	if options.ModifiedAfter != nil && info.ModTime().Before(*options.ModifiedAfter) {
		return nil
	}
	if options.ModifiedBefore != nil && info.ModTime().After(*options.ModifiedBefore) {
		return nil
	}

	// Detect file type and apply filters
	fileType := fs.detector.DetectFileType(name)
	if len(options.FileTypes) > 0 {
		found := false
		for _, ft := range options.FileTypes {
			if fileType == ft {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// Apply extension filters
	if len(options.Extensions) > 0 {
		ext := strings.ToLower(filepath.Ext(name))
		found := false
		for _, allowedExt := range options.Extensions {
			if ext == strings.ToLower(allowedExt) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// Check pattern match
	matchType := fs.getMatchType(name, fullPath, options, pattern)
	if matchType == -1 {
		// Check content search if enabled and this is a text file
		if options.SearchContent && !info.IsDir() && fileType.IsReadable() {
			if fs.searchFileContent(fullPath, options) {
				matchType = int(MatchContent)
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	// Create relative path
	relPath, err := filepath.Rel(rootPath, fullPath)
	if err != nil {
		relPath = fullPath
	}

	return &SearchResult{
		Path:         fullPath,
		RelativePath: relPath,
		Name:         name,
		Size:         info.Size(),
		ModTime:      info.ModTime(),
		IsDir:        info.IsDir(),
		FileType:     fileType,
		MatchType:    MatchType(matchType),
	}
}

// getMatchType determines how the file matches the search pattern
func (fs *FileSearcher) getMatchType(name, fullPath string, options *SearchOptions, pattern *regexp.Regexp) int {
	searchName := name
	searchPath := fullPath
	searchPattern := options.Pattern

	if !options.CaseSensitive {
		searchName = strings.ToLower(searchName)
		searchPath = strings.ToLower(searchPath)
		searchPattern = strings.ToLower(searchPattern)
	}

	if options.IsRegex {
		// Regex matching
		if pattern.MatchString(searchName) {
			return int(MatchFileName)
		}
		if pattern.MatchString(searchPath) {
			return int(MatchPath)
		}
	} else {
		// Simple string/glob matching
		if fs.matchesPattern(searchName, searchPattern) {
			return int(MatchFileName)
		}
		if fs.matchesPattern(searchPath, searchPattern) {
			return int(MatchPath)
		}

		// Check extension match
		ext := strings.ToLower(filepath.Ext(name))
		if ext != "" && strings.Contains(searchPattern, ext[1:]) { // Remove the dot
			return int(MatchExtension)
		}
	}

	return -1 // No match
}

// matchesPattern performs simple pattern matching with basic glob support
func (fs *FileSearcher) matchesPattern(text, pattern string) bool {
	// Simple substring match
	if strings.Contains(text, pattern) {
		return true
	}

	// Basic glob pattern matching
	matched, err := filepath.Match(pattern, text)
	if err == nil && matched {
		return true
	}

	return false
}

// searchFileContent searches for a pattern within file contents
func (fs *FileSearcher) searchFileContent(filePath string, options *SearchOptions) bool {
	if options.ContentPattern == "" {
		return false
	}

	// Validate file for reading
	if err := fs.validator.ValidateRead(filePath); err != nil {
		return false
	}

	// Read file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	searchContent := string(content)
	searchPattern := options.ContentPattern

	if !options.CaseSensitive {
		searchContent = strings.ToLower(searchContent)
		searchPattern = strings.ToLower(searchPattern)
	}

	return strings.Contains(searchContent, searchPattern)
}

// QuickSearch performs a fast filename-only search
func (fs *FileSearcher) QuickSearch(ctx context.Context, rootPath, pattern string) ([]*SearchResult, error) {
	options := DefaultSearchOptions()
	options.Pattern = pattern
	options.MaxResults = 50 // Limit for quick searches
	options.MaxDepth = 5    // Shallow search for speed
	options.SearchContent = false

	return fs.Search(ctx, rootPath, options)
}

// SearchByType searches for files of specific types
func (fs *FileSearcher) SearchByType(ctx context.Context, rootPath string, fileTypes []FileType) ([]*SearchResult, error) {
	options := DefaultSearchOptions()
	options.Pattern = "*"
	options.FileTypes = fileTypes
	options.SearchContent = false

	return fs.Search(ctx, rootPath, options)
}

// SearchByExtension searches for files with specific extensions
func (fs *FileSearcher) SearchByExtension(ctx context.Context, rootPath string, extensions []string) ([]*SearchResult, error) {
	options := DefaultSearchOptions()
	options.Pattern = "*"
	options.Extensions = extensions
	options.SearchContent = false

	return fs.Search(ctx, rootPath, options)
}
