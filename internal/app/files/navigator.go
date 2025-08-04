package files

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileNode represents a node in the file tree
type FileNode struct {
	Path       string      // Full path to the file/directory
	Name       string      // File/directory name
	IsDir      bool        // Whether this is a directory
	Size       int64       // File size in bytes
	ModTime    time.Time   // Last modification time
	FileType   FileType    // Detected file type
	Children   []*FileNode // Child nodes (for directories)
	Parent     *FileNode   // Parent node
	IsExpanded bool        // Whether directory is expanded in UI
	IsLoaded   bool        // Whether children have been loaded
	IsSelected bool        // Whether this node is currently selected
	Depth      int         // Depth in the tree (for indentation)
}

// IsRoot returns true if this is a root node
func (fn *FileNode) IsRoot() bool {
	return fn.Parent == nil
}

// GetIcon returns the appropriate icon for this file node
func (fn *FileNode) GetIcon() string {
	if fn.IsDir {
		if fn.IsExpanded {
			return "📂"
		}
		return "📁"
	}
	return fn.FileType.Icon()
}

// GetDisplayName returns the name to display in the UI
func (fn *FileNode) GetDisplayName() string {
	if fn.IsDir && len(fn.Children) > 0 {
		return fmt.Sprintf("%s (%d)", fn.Name, len(fn.Children))
	}
	return fn.Name
}

// NavigationHistory manages the navigation history for back/forward functionality
type NavigationHistory struct {
	history []string // Stack of visited paths
	current int      // Current position in history
	maxSize int      // Maximum history size
}

// NewNavigationHistory creates a new navigation history
func NewNavigationHistory(maxSize int) *NavigationHistory {
	if maxSize <= 0 {
		maxSize = 50
	}
	return &NavigationHistory{
		history: make([]string, 0, maxSize),
		current: -1,
		maxSize: maxSize,
	}
}

// Push adds a new path to the history
func (nh *NavigationHistory) Push(path string) {
	// Don't add duplicate consecutive entries
	if len(nh.history) > 0 && nh.current >= 0 && nh.history[nh.current] == path {
		return
	}

	// Remove forward history when pushing new path
	if nh.current < len(nh.history)-1 {
		nh.history = nh.history[:nh.current+1]
	}

	// Add new path
	nh.history = append(nh.history, path)
	nh.current = len(nh.history) - 1

	// Trim history if it exceeds max size
	if len(nh.history) > nh.maxSize {
		nh.history = nh.history[1:]
		nh.current--
	}
}

// Back returns the previous path in history
func (nh *NavigationHistory) Back() (string, bool) {
	if nh.current > 0 {
		nh.current--
		return nh.history[nh.current], true
	}
	return "", false
}

// Forward returns the next path in history
func (nh *NavigationHistory) Forward() (string, bool) {
	if nh.current < len(nh.history)-1 {
		nh.current++
		return nh.history[nh.current], true
	}
	return "", false
}

// CanGoBack returns true if there are previous paths in history
func (nh *NavigationHistory) CanGoBack() bool {
	return nh.current > 0
}

// CanGoForward returns true if there are forward paths in history
func (nh *NavigationHistory) CanGoForward() bool {
	return nh.current < len(nh.history)-1
}

// Navigator provides file system navigation capabilities
type Navigator struct {
	validator   *SecurityValidator
	detector    *SyntaxDetector
	searcher    *FileSearcher
	history     *NavigationHistory
	currentPath string
	rootNode    *FileNode
}

// NewNavigator creates a new file navigator with security validation
func NewNavigator(config *SecurityConfig) *Navigator {
	validator := NewSecurityValidator(config)

	return &Navigator{
		validator: validator,
		detector:  NewSyntaxDetector(),
		searcher:  NewFileSearcher(validator),
		history:   NewNavigationHistory(50),
	}
}

// SetRoot sets the root directory for navigation
func (n *Navigator) SetRoot(rootPath string) error {
	// Validate and sanitize the root path
	cleanPath, err := n.validator.SanitizePath(rootPath)
	if err != nil {
		return fmt.Errorf("invalid root path: %w", err)
	}

	// Verify it's a directory
	if err := n.validator.ValidateDirectory(cleanPath); err != nil {
		return fmt.Errorf("root path validation failed: %w", err)
	}

	// Create root node
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to stat root path: %w", err)
	}

	n.rootNode = &FileNode{
		Path:       cleanPath,
		Name:       filepath.Base(cleanPath),
		IsDir:      true,
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		FileType:   FileTypeUnknown,
		IsExpanded: true,
		Depth:      0,
	}

	n.currentPath = cleanPath
	n.history.Push(cleanPath)

	// Load initial children
	return n.LoadChildren(n.rootNode)
}

// GetRoot returns the root node
func (n *Navigator) GetRoot() *FileNode {
	return n.rootNode
}

// GetCurrentPath returns the current navigation path
func (n *Navigator) GetCurrentPath() string {
	return n.currentPath
}

// Navigate changes the current directory
func (n *Navigator) Navigate(path string) error {
	// Validate the target path
	cleanPath, err := n.validator.SanitizePath(path)
	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	if err := n.validator.ValidateDirectory(cleanPath); err != nil {
		return fmt.Errorf("navigation validation failed: %w", err)
	}

	n.currentPath = cleanPath
	n.history.Push(cleanPath)

	return nil
}

// NavigateUp moves to the parent directory
func (n *Navigator) NavigateUp() error {
	if n.currentPath == "" {
		return fmt.Errorf("no current path set")
	}

	parent := filepath.Dir(n.currentPath)
	if parent == n.currentPath {
		return fmt.Errorf("already at root directory")
	}

	return n.Navigate(parent)
}

// NavigateBack goes back in navigation history
func (n *Navigator) NavigateBack() error {
	if path, ok := n.history.Back(); ok {
		n.currentPath = path
		return nil
	}
	return fmt.Errorf("no previous path in history")
}

// NavigateForward goes forward in navigation history
func (n *Navigator) NavigateForward() error {
	if path, ok := n.history.Forward(); ok {
		n.currentPath = path
		return nil
	}
	return fmt.Errorf("no forward path in history")
}

// CanNavigateBack returns true if navigation back is possible
func (n *Navigator) CanNavigateBack() bool {
	return n.history.CanGoBack()
}

// CanNavigateForward returns true if navigation forward is possible
func (n *Navigator) CanNavigateForward() bool {
	return n.history.CanGoForward()
}

// LoadChildren loads the children of a directory node
func (n *Navigator) LoadChildren(node *FileNode) error {
	if !node.IsDir {
		return fmt.Errorf("cannot load children of non-directory")
	}

	// Validate directory access
	if err := n.validator.ValidateDirectory(node.Path); err != nil {
		return fmt.Errorf("access denied: %w", err)
	}

	// Read directory contents
	entries, err := os.ReadDir(node.Path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Prevent resource exhaustion from large directories
	const maxDirectoryEntries = 10000
	if len(entries) > maxDirectoryEntries {
		return fmt.Errorf("directory too large: %d entries (max: %d)", len(entries), maxDirectoryEntries)
	}

	// Clear existing children
	node.Children = make([]*FileNode, 0, len(entries))

	// Create child nodes
	for _, entry := range entries {
		childPath := filepath.Join(node.Path, entry.Name())

		// Skip if path validation fails
		if err := n.validator.ValidatePath(childPath); err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		fileType := n.detector.DetectFileType(entry.Name())

		child := &FileNode{
			Path:     childPath,
			Name:     entry.Name(),
			IsDir:    entry.IsDir(),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			FileType: fileType,
			Parent:   node,
			Depth:    node.Depth + 1,
		}

		node.Children = append(node.Children, child)
	}

	// Sort children: directories first, then by name
	sort.Slice(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir // Directories first
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	node.IsLoaded = true
	return nil
}

// ToggleExpanded toggles the expanded state of a directory node
func (n *Navigator) ToggleExpanded(node *FileNode) error {
	if !node.IsDir {
		return fmt.Errorf("cannot expand non-directory")
	}

	if !node.IsExpanded {
		// Expanding: load children if not already loaded
		if !node.IsLoaded {
			if err := n.LoadChildren(node); err != nil {
				return err
			}
		}
		node.IsExpanded = true
	} else {
		// Collapsing
		node.IsExpanded = false
	}

	return nil
}

// RefreshNode refreshes a node by reloading its children
func (n *Navigator) RefreshNode(node *FileNode) error {
	if !node.IsDir {
		return fmt.Errorf("cannot refresh non-directory")
	}

	node.IsLoaded = false
	return n.LoadChildren(node)
}

// FindNode finds a node by its path
func (n *Navigator) FindNode(path string) *FileNode {
	if n.rootNode == nil {
		return nil
	}
	return n.findNodeRecursive(n.rootNode, path)
}

// findNodeRecursive recursively searches for a node with the given path
func (n *Navigator) findNodeRecursive(node *FileNode, path string) *FileNode {
	if node.Path == path {
		return node
	}

	for _, child := range node.Children {
		if found := n.findNodeRecursive(child, path); found != nil {
			return found
		}
	}

	return nil
}

// GetFlattenedNodes returns a flattened list of visible nodes (respecting expansion state)
func (n *Navigator) GetFlattenedNodes() []*FileNode {
	if n.rootNode == nil {
		return nil
	}

	var nodes []*FileNode
	n.flattenNodesRecursive(n.rootNode, &nodes)
	return nodes
}

// flattenNodesRecursive recursively flattens the tree structure
func (n *Navigator) flattenNodesRecursive(node *FileNode, nodes *[]*FileNode) {
	*nodes = append(*nodes, node)

	if node.IsDir && node.IsExpanded {
		for _, child := range node.Children {
			n.flattenNodesRecursive(child, nodes)
		}
	}
}

// ReadFile safely reads a file's contents with TOCTOU protection
func (n *Navigator) ReadFile(path string) ([]byte, error) {
	if err := n.validator.ValidateRead(path); err != nil {
		return nil, fmt.Errorf("read access denied: %w", err)
	}

	// Check if it's a text file
	if n.detector.IsBinaryFile(path) {
		return nil, fmt.Errorf("cannot read binary file as text")
	}

	// Open file to get a handle for atomic operations
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Re-validate using file descriptor to prevent TOCTOU attacks
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check file size limit using actual file descriptor
	if info.Size() > n.validator.config.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds limit")
	}

	// Read from the file descriptor we validated
	content := make([]byte, info.Size())
	_, err = file.Read(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// GetFileInfo returns detailed information about a file
func (n *Navigator) GetFileInfo(path string) (*FileNode, error) {
	if err := n.validator.ValidatePath(path); err != nil {
		return nil, fmt.Errorf("access denied: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	fileType := n.detector.DetectFileType(filepath.Base(path))

	return &FileNode{
		Path:     path,
		Name:     filepath.Base(path),
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		FileType: fileType,
	}, nil
}

// Search performs a file search using the configured searcher
func (n *Navigator) Search(ctx context.Context, options *SearchOptions) ([]*SearchResult, error) {
	if n.currentPath == "" {
		return nil, fmt.Errorf("no current path set for search")
	}

	return n.searcher.Search(ctx, n.currentPath, options)
}

// QuickSearch performs a quick filename search
func (n *Navigator) QuickSearch(ctx context.Context, pattern string) ([]*SearchResult, error) {
	if n.currentPath == "" {
		return nil, fmt.Errorf("no current path set for search")
	}

	return n.searcher.QuickSearch(ctx, n.currentPath, pattern)
}

// GetBreadcrumb returns the breadcrumb path components
func (n *Navigator) GetBreadcrumb() []string {
	if n.currentPath == "" {
		return nil
	}

	var parts []string
	path := n.currentPath

	for path != "/" && path != "." {
		parts = append([]string{filepath.Base(path)}, parts...)
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}

	if len(parts) == 0 {
		parts = append(parts, "/")
	}

	return parts
}

// IsTextFile returns true if the file can be displayed as text
func (n *Navigator) IsTextFile(path string) bool {
	return n.detector.IsTextFile(path)
}

// GetFileType returns the detected file type
func (n *Navigator) GetFileType(path string) FileType {
	return n.detector.DetectFileType(path)
}

// GetLanguage returns the programming language for syntax highlighting
func (n *Navigator) GetLanguage(path string) string {
	return n.detector.GetLanguage(path)
}
