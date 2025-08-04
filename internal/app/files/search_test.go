package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/testutil"
)

func TestNewFileSearcher(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)

	assert.NotNil(t, searcher)
	assert.NotNil(t, searcher.validator)
	assert.NotNil(t, searcher.detector)
}

func TestFileSearcher_Search(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("basic filename search", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*.go",
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// All results should be Go files
		for _, result := range results {
			assert.True(t, strings.HasSuffix(result.Name, ".go"))
			assert.Equal(t, MatchFileName, result.MatchType)
		}
	})

	t.Run("case insensitive search", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:       "README",
			CaseSensitive: false,
			MaxResults:    10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// Should find README.md
		found := false
		for _, result := range results {
			if strings.Contains(strings.ToLower(result.Name), "readme") {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("case sensitive search", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:       "readme", // lowercase
			CaseSensitive: true,
			MaxResults:    10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should not find README.md (uppercase)
		for _, result := range results {
			assert.NotEqual(t, "README.md", result.Name)
		}
	})

	t.Run("regex search", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "main\\.(go|txt)",
			IsRegex:    true,
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// All results should match the pattern
		for _, result := range results {
			matched := strings.HasSuffix(result.Name, "main.go") || strings.HasSuffix(result.Name, "main.txt")
			assert.True(t, matched, "Result should match regex: %s", result.Name)
		}
	})

	t.Run("file type filter", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			FileTypes:  []FileType{FileTypeGo},
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// All results should be Go files
		for _, result := range results {
			assert.Equal(t, FileTypeGo, result.FileType)
		}
	})

	t.Run("extension filter", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			Extensions: []string{".md", ".txt"},
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// All results should have .md or .txt extension
		for _, result := range results {
			ext := strings.ToLower(filepath.Ext(result.Name))
			assert.True(t, ext == ".md" || ext == ".txt", "Unexpected extension: %s", ext)
		}
	})

	t.Run("size filter", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			MinSize:    100,  // At least 100 bytes
			MaxSize:    1000, // At most 1000 bytes
			MaxResults: 50,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// All results should be within size range
		for _, result := range results {
			if !result.IsDir {
				assert.GreaterOrEqual(t, result.Size, int64(100))
				assert.LessOrEqual(t, result.Size, int64(1000))
			}
		}
	})

	t.Run("max results limit", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			MaxResults: 3,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(results), 3)
	})

	t.Run("max depth limit", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			MaxDepth:   1, // Only search one level deep
			MaxResults: 50,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should only find files in root and first level subdirectories
		for _, result := range results {
			depth := strings.Count(strings.TrimPrefix(result.RelativePath, "./"), string(os.PathSeparator))
			assert.LessOrEqual(t, depth, 1, "File too deep: %s", result.RelativePath)
		}
	})

	t.Run("hidden files inclusion", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:       ".*",
			IncludeHidden: true,
			MaxResults:    10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should find hidden files
		foundHidden := false
		for _, result := range results {
			if strings.HasPrefix(result.Name, ".") {
				foundHidden = true
				break
			}
		}
		assert.True(t, foundHidden, "Should find hidden files when included")
	})

	t.Run("hidden files exclusion", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:       "*",
			IncludeHidden: false,
			MaxResults:    10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should not find hidden files
		for _, result := range results {
			assert.False(t, strings.HasPrefix(result.Name, "."), "Should not find hidden files: %s", result.Name)
		}
	})
}

func TestFileSearcher_ContentSearch(t *testing.T) {
	tempDir := testutil.TempDir(t)

	// Create files with specific content
	testutil.CreateTestFile(t, tempDir, "file1.txt", "This file contains the search term")
	testutil.CreateTestFile(t, tempDir, "file2.txt", "This file does not contain the term")
	testutil.CreateTestFile(t, tempDir, "file3.go", "package main\n\nfunc search() {\n\t// search function\n}")

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("content search finds matching files", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:        "*",
			SearchContent:  true,
			ContentPattern: "search",
			MaxResults:     10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)

		// Should find files containing "search"
		foundFile1 := false
		foundFile3 := false
		for _, result := range results {
			if result.Name == "file1.txt" && result.MatchType == MatchContent {
				foundFile1 = true
			}
			if result.Name == "file3.go" && result.MatchType == MatchContent {
				foundFile3 = true
			}
		}
		assert.True(t, foundFile1, "Should find file1.txt with content match")
		assert.True(t, foundFile3, "Should find file3.go with content match")
	})

	t.Run("content search with case sensitivity", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:        "*",
			SearchContent:  true,
			ContentPattern: "SEARCH", // uppercase
			CaseSensitive:  true,
			MaxResults:     10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should not find any matches (all content has lowercase "search")
		contentMatches := 0
		for _, result := range results {
			if result.MatchType == MatchContent {
				contentMatches++
			}
		}
		assert.Zero(t, contentMatches, "Should not find content matches with case sensitive uppercase search")
	})
}

func TestFileSearcher_SecurityValidation(t *testing.T) {
	tempDir := testutil.TempDir(t)
	forbiddenDir := testutil.CreateTestDir(t, tempDir, "forbidden")
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")

	testutil.CreateTestFile(t, forbiddenDir, "secret.txt", "secret content")
	testutil.CreateTestFile(t, allowedDir, "public.txt", "public content")

	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{forbiddenDir},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("search respects security boundaries", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, allowedDir, options)
		assert.NoError(t, err)

		// Should only find files in allowed directory
		for _, result := range results {
			assert.True(t, strings.HasPrefix(result.Path, allowedDir),
				"Result should be within allowed directory: %s", result.Path)
		}
	})

	t.Run("search in forbidden directory fails", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			MaxResults: 10,
		}

		_, err := searcher.Search(ctx, forbiddenDir, options)
		assert.Error(t, err)
	})
}

func TestFileSearcher_SymlinkHandling(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	outsideDir := testutil.CreateTestDir(t, tempDir, "outside")

	// Create files
	allowedFile := testutil.CreateTestFile(t, allowedDir, "file.txt", "content")
	outsideFile := testutil.CreateTestFile(t, outsideDir, "secret.txt", "secret")

	// Create symlinks
	symlinkToAllowed := filepath.Join(allowedDir, "link_to_allowed.txt")
	symlinkToOutside := filepath.Join(allowedDir, "link_to_outside.txt")

	testutil.CreateSymlink(t, allowedFile, symlinkToAllowed)
	testutil.CreateSymlink(t, outsideFile, symlinkToOutside)

	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{outsideDir},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("symlink validation", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*",
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, allowedDir, options)
		assert.NoError(t, err)

		// Check that only safe symlinks are included
		for _, result := range results {
			if strings.Contains(result.Name, "link_to_outside") {
				t.Errorf("Unsafe symlink should not be included: %s", result.Name)
			}
		}
	})
}

func TestFileSearcher_MatchTypes(t *testing.T) {
	tempDir := testutil.TempDir(t)

	// Create test structure for different match types
	testutil.CreateTestFile(t, tempDir, "search_in_name.txt", "content")
	testutil.CreateTestDir(t, tempDir, "search_in_path")
	testutil.CreateTestFile(t, tempDir, "search_in_path/file.txt", "content")
	testutil.CreateTestFile(t, tempDir, "content_file.txt", "This contains search term")
	testutil.CreateTestFile(t, tempDir, "example.js", "javascript content")

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("filename match type", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "search_in_name",
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		found := false
		for _, result := range results {
			if result.Name == "search_in_name.txt" {
				assert.Equal(t, MatchFileName, result.MatchType)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("path match type", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "search_in_path",
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		foundDir := false
		foundFile := false
		for _, result := range results {
			if result.Name == "search_in_path" && result.IsDir {
				assert.Equal(t, MatchFileName, result.MatchType) // Directory name match
				foundDir = true
			}
			if result.Name == "file.txt" && strings.Contains(result.Path, "search_in_path") {
				assert.Equal(t, MatchPath, result.MatchType) // Path contains the pattern
				foundFile = true
			}
		}
		assert.True(t, foundDir)
		assert.True(t, foundFile)
	})

	t.Run("extension match type", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "js", // Should match .js extension
			MaxResults: 10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		found := false
		for _, result := range results {
			if result.Name == "example.js" {
				assert.Equal(t, MatchExtension, result.MatchType)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("content match type", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:        "*",
			SearchContent:  true,
			ContentPattern: "search term",
			MaxResults:     10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		found := false
		for _, result := range results {
			if result.Name == "content_file.txt" && result.MatchType == MatchContent {
				found = true
			}
		}
		assert.True(t, found)
	})
}

func TestFileSearcher_QuickSearch(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("quick search limits results and depth", func(t *testing.T) {
		results, err := searcher.QuickSearch(ctx, tempDir, "test")
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(results), 50) // QuickSearch limits to 50 results

		// Results should be from shallow depth (max 5 levels)
		for _, result := range results {
			depth := strings.Count(strings.TrimPrefix(result.RelativePath, "./"), string(os.PathSeparator))
			assert.LessOrEqual(t, depth, 5)
		}
	})
}

func TestFileSearcher_SearchByType(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("search by Go file type", func(t *testing.T) {
		results, err := searcher.SearchByType(ctx, tempDir, []FileType{FileTypeGo})
		assert.NoError(t, err)

		// All results should be Go files
		for _, result := range results {
			assert.Equal(t, FileTypeGo, result.FileType)
		}
	})

	t.Run("search by multiple file types", func(t *testing.T) {
		results, err := searcher.SearchByType(ctx, tempDir, []FileType{FileTypeGo, FileTypeMarkdown})
		assert.NoError(t, err)

		// All results should be Go or Markdown files
		for _, result := range results {
			isValid := result.FileType == FileTypeGo || result.FileType == FileTypeMarkdown
			assert.True(t, isValid, "Unexpected file type: %s", result.FileType.String())
		}
	})
}

func TestFileSearcher_SearchByExtension(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("search by .go extension", func(t *testing.T) {
		results, err := searcher.SearchByExtension(ctx, tempDir, []string{".go"})
		assert.NoError(t, err)

		// All results should have .go extension
		for _, result := range results {
			assert.True(t, strings.HasSuffix(result.Name, ".go"))
		}
	})

	t.Run("search by multiple extensions", func(t *testing.T) {
		results, err := searcher.SearchByExtension(ctx, tempDir, []string{".go", ".md"})
		assert.NoError(t, err)

		// All results should have .go or .md extension
		for _, result := range results {
			hasValidExt := strings.HasSuffix(result.Name, ".go") || strings.HasSuffix(result.Name, ".md")
			assert.True(t, hasValidExt, "Unexpected extension for file: %s", result.Name)
		}
	})
}

func TestFileSearcher_TimeFilters(t *testing.T) {
	tempDir := testutil.TempDir(t)

	// Create files with different timestamps
	oldFile := testutil.CreateTestFile(t, tempDir, "old.txt", "old content")
	newFile := testutil.CreateTestFile(t, tempDir, "new.txt", "new content")

	// Modify timestamps
	oldTime := time.Now().Add(-24 * time.Hour)
	newTime := time.Now()

	os.Chtimes(oldFile, oldTime, oldTime)
	os.Chtimes(newFile, newTime, newTime)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	t.Run("modified after filter", func(t *testing.T) {
		cutoffTime := time.Now().Add(-12 * time.Hour) // 12 hours ago
		options := &SearchOptions{
			Pattern:       "*",
			ModifiedAfter: &cutoffTime,
			MaxResults:    10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should only find new file
		foundNew := false
		for _, result := range results {
			if result.Name == "new.txt" {
				foundNew = true
			}
			if result.Name == "old.txt" {
				t.Errorf("Old file should be filtered out by ModifiedAfter")
			}
		}
		assert.True(t, foundNew)
	})

	t.Run("modified before filter", func(t *testing.T) {
		cutoffTime := time.Now().Add(-12 * time.Hour) // 12 hours ago
		options := &SearchOptions{
			Pattern:        "*",
			ModifiedBefore: &cutoffTime,
			MaxResults:     10,
		}

		results, err := searcher.Search(ctx, tempDir, options)
		assert.NoError(t, err)

		// Should only find old file
		foundOld := false
		for _, result := range results {
			if result.Name == "old.txt" {
				foundOld = true
			}
			if result.Name == "new.txt" {
				t.Errorf("New file should be filtered out by ModifiedBefore")
			}
		}
		assert.True(t, foundOld)
	})
}

func TestFileSearcher_ConcurrentSearch(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)

	// Run multiple searches concurrently
	const numGoroutines = 10
	resultChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx := context.Background()
			options := &SearchOptions{
				Pattern:    "*.go",
				MaxResults: 5,
			}

			_, err := searcher.Search(ctx, tempDir, options)
			resultChan <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-resultChan
		assert.NoError(t, err)
	}
}

func TestFileSearcher_ContextCancellation(t *testing.T) {
	tempDir := testutil.TempDir(t)

	// Create many files to make search take longer
	for i := 0; i < 1000; i++ {
		testutil.CreateTestFile(t, tempDir, filepath.Join("many", "file_"+string(rune(i))+".txt"), "content")
	}

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)

	t.Run("context cancellation stops search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		options := &SearchOptions{
			Pattern:    "*",
			MaxResults: 10000, // Request many results
		}

		_, err := searcher.Search(ctx, tempDir, options)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestSearchOptions_Defaults(t *testing.T) {
	options := DefaultSearchOptions()

	assert.NotNil(t, options)
	assert.False(t, options.IsRegex)
	assert.False(t, options.CaseSensitive)
	assert.False(t, options.IncludeHidden)
	assert.Equal(t, 1000, options.MaxResults)
	assert.Equal(t, 10, options.MaxDepth)
	assert.Equal(t, int64(10*1024*1024), options.MaxSize)
	assert.False(t, options.SearchContent)
}

func TestMatchType_String(t *testing.T) {
	tests := []struct {
		matchType MatchType
		expected  string
	}{
		{MatchFileName, "filename"},
		{MatchPath, "path"},
		{MatchContent, "content"},
		{MatchExtension, "extension"},
		{MatchType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.matchType.String())
		})
	}
}

// Benchmark tests for search performance
func BenchmarkFileSearcher_BasicSearch(b *testing.B) {
	tempDir := testutil.TempDir(b)
	testutil.CreateTestFileStructure(b, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	options := &SearchOptions{
		Pattern:    "*.go",
		MaxResults: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := searcher.Search(ctx, tempDir, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileSearcher_RegexSearch(b *testing.B) {
	tempDir := testutil.TempDir(b)
	testutil.CreateTestFileStructure(b, tempDir)

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	options := &SearchOptions{
		Pattern:    "main\\.(go|txt)",
		IsRegex:    true,
		MaxResults: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := searcher.Search(ctx, tempDir, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileSearcher_ContentSearch(b *testing.B) {
	tempDir := testutil.TempDir(b)

	// Create files with searchable content
	for i := 0; i < 100; i++ {
		content := "This is file " + string(rune(i)) + " with some searchable content"
		testutil.CreateTestFile(b, tempDir, "file_"+string(rune(i))+".txt", content)
	}

	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()

	options := &SearchOptions{
		Pattern:        "*",
		SearchContent:  true,
		ContentPattern: "searchable",
		MaxResults:     10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := searcher.Search(ctx, tempDir, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}
