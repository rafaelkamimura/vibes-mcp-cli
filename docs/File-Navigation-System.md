# Enhanced File Navigation System

This document describes the enhanced file navigation system implemented for Claude Code integration in the Vibes MCP CLI. The system provides secure, performant, and user-friendly file exploration capabilities with built-in Claude Code integration.

## Architecture Overview

The file navigation system consists of five main components:

```
internal/app/files/
├── security.go      # Secure file access and validation
├── navigator.go     # Main file navigation logic  
├── syntax.go        # File type detection and syntax highlighting
├── search.go        # File search functionality
└── (examples/)      # Usage examples and integration guides

internal/ui/components/
└── file_explorer.go # TUI file explorer component
```

## Key Features

### 🔒 Security Features
- **Path traversal protection** with multiple validation layers
- **Allowlist-based access control** for directories
- **Symlink target validation** to prevent security bypass
- **Resource limits** to prevent DoS attacks
- **File size limits** and content validation
- **Hidden file access control**

### 📂 File Navigation
- **Tree-based navigation** with lazy loading
- **Breadcrumb navigation** with history support
- **Back/forward navigation** similar to web browsers
- **Directory expansion/collapse** with keyboard shortcuts
- **File type detection** with icons and syntax highlighting

### 🔍 Search Capabilities
- **Quick filename search** with glob patterns
- **Regular expression search** with ReDoS protection
- **File type filtering** (Go, Python, JavaScript, etc.)
- **Content search** within text files
- **Advanced search options** with size and date filters

### 🤖 Claude Code Integration
- **Send files to Claude Code** with 'M' key
- **Add files to session** with 'A' key  
- **Multiple file selection** support
- **Automatic file type detection** for context
- **Syntax highlighting detection** for code files

## Component Details

### SecurityValidator (`security.go`)

Provides comprehensive security validation for file system access:

```go
type SecurityConfig struct {
    AllowedPaths   []string // Allowed base paths
    ForbiddenPaths []string // Explicitly forbidden paths  
    MaxDepth       int      // Maximum directory depth
    AllowHidden    bool     // Allow hidden files/directories
    MaxFileSize    int64    // Maximum file size for reading
}
```

**Security Features:**
- Path traversal detection (including URL-encoded attempts)
- Symlink target validation
- Directory depth limits
- File size restrictions
- Hidden file access control

### Navigator (`navigator.go`)

Core navigation logic with secure file access:

```go
type Navigator struct {
    validator   *SecurityValidator
    detector    *SyntaxDetector  
    searcher    *FileSearcher
    history     *NavigationHistory
    currentPath string
    rootNode    *FileNode
}
```

**Key Methods:**
- `SetRoot(path)` - Set navigation root with validation
- `Navigate(path)` - Navigate to directory with security checks
- `LoadChildren(node)` - Lazy load directory contents
- `ReadFile(path)` - Secure file reading with TOCTOU protection
- `Search(options)` - File search with security validation

### SyntaxDetector (`syntax.go`)

File type detection and syntax highlighting support:

```go
type FileType int

const (
    FileTypeGo FileType = iota
    FileTypePython
    FileTypeJavaScript  
    FileTypeJSON
    // ... 20+ file types supported
)
```

**Features:**
- Extension-based detection
- Filename-based detection (Dockerfile, Makefile, etc.)
- Programming language identification
- Binary file detection
- Icon assignment for UI display

### FileSearcher (`search.go`)

Advanced file search with security validation:

```go
type SearchOptions struct {
    Pattern       string    // Search pattern (regex or glob)
    IsRegex       bool      // Treat pattern as regex
    CaseSensitive bool      // Case-sensitive search
    MaxResults    int       // Limit number of results
    FileTypes     []FileType // Filter by file types
    SearchContent bool      // Search within file contents
}
```

**Search Types:**
- Filename matching
- Path matching  
- Content searching
- Extension filtering
- File type filtering

### FileExplorer (`file_explorer.go`)

TUI component built with tview:

```go
type FileExplorer struct {
    *tview.Flex
    navigator   *files.Navigator
    treeView    *tview.TreeView
    previewView *tview.TextView
    searchInput *tview.InputField
}
```

**UI Features:**
- Tree view with file icons
- File preview panel
- Search interface
- Status bar with navigation info
- Keyboard shortcuts
- Context-sensitive actions

## Usage Examples

### Basic Navigation

```go
// Create navigator with security config
config := &files.SecurityConfig{
    AllowedPaths: []string{"/home/user/projects"},
    MaxFileSize:  10 * 1024 * 1024, // 10MB
    AllowHidden:  false,
}

navigator := files.NewNavigator(config)
err := navigator.SetRoot("/home/user/projects")
```

### File Search

```go
// Quick filename search
results, err := navigator.QuickSearch(ctx, "*.go")

// Advanced search with options
options := &files.SearchOptions{
    Pattern:    "config",
    FileTypes:  []files.FileType{files.FileTypeJSON, files.FileTypeYAML},
    MaxResults: 50,
}
results, err := navigator.Search(ctx, options)
```

### Secure File Reading

```go
// Read file with security validation
content, err := navigator.ReadFile("/path/to/file.go")
if err != nil {
    // Handle security or I/O error
}

// Check if file is suitable for text display
if navigator.IsTextFile(filePath) {
    language := navigator.GetLanguage(filePath)
    // Use language for syntax highlighting
}
```

### TUI Integration

```go
// Create file explorer component
config := &components.FileExplorerConfig{
    RootPath:        "/home/user/projects",
    ShowHiddenFiles: false,
    EnableSearch:    true,
    EnablePreview:   true,
}

callbacks := &components.FileExplorerCallbacks{
    OnFileAction: func(action components.FileExplorerAction, path string) {
        switch action {
        case components.ActionSendToClaude:
            sendToClaudeCode(path)
        case components.ActionAddToSession:
            addToSession(path)
        }
    },
}

fileExplorer := components.NewFileExplorer(config, callbacks)
```

## Keyboard Shortcuts

### Navigation
- `↑↓` - Navigate files/directories
- `Enter` - Open directory or preview file
- `Backspace` - Navigate up to parent directory
- `Ctrl+R` - Refresh current directory

### File Actions  
- `M` - Send file to Claude Code
- `A` - Add file to current session
- `V` - Preview file content
- `C` - Copy file path
- `R` - Refresh directory

### Search
- `/` - Activate search mode
- `Esc` - Exit search mode
- `Enter` - Execute search

### Display Options
- `H` - Toggle hidden files visibility
- `Q` - Quit file explorer

## Security Considerations

### Path Validation
- All paths are validated against allowlist/forbiddenlist
- Path traversal attempts (../, ~/, %2e%2e) are blocked
- Symlink targets are validated before following

### Resource Protection
- Directory size limits prevent memory exhaustion
- File size limits prevent large file attacks  
- Search complexity limits prevent ReDoS attacks
- Operation timeouts prevent hanging

### Information Disclosure
- Error messages don't leak sensitive path information
- Hidden files are protected by default
- Binary files are detected and handled appropriately

## Performance Optimizations

### Lazy Loading
- Directory contents loaded on-demand
- Large directories paginated/limited
- Background loading for smooth UI

### Caching
- File type detection results cached
- Search results cached temporarily
- Navigation history maintained efficiently

### Memory Management
- Limited result sets prevent memory bloat
- Streaming file reads for large files
- Garbage collection friendly data structures

## Integration with Existing UI

The file explorer integrates seamlessly with the existing tview-based UI:

1. **Consistent styling** with existing components
2. **Keyboard navigation** matching vim-like patterns
3. **Modal dialogs** for confirmations and errors
4. **Status updates** integrated with application state
5. **Theme support** through tview color schemes

## Claude Code Integration

### File Selection Workflow
1. User navigates to desired file
2. Presses 'M' to send to Claude Code
3. File content is validated and prepared
4. Context information (file type, language) added
5. Sent to Claude Code session with formatting

### Session Management
- Files can be added to current session with 'A' key
- Multiple files supported for batch operations
- Session state maintained across navigation
- File list can be cleared or modified

### Content Preparation
- Syntax highlighting detection for code context
- Binary file detection prevents invalid sends
- File size validation ensures reasonable content
- Error handling for inaccessible files

## Testing and Validation

### Security Testing
- Path traversal attack vectors tested
- Symlink manipulation attempts blocked
- Resource exhaustion scenarios handled
- Permission boundary enforcement verified

### Performance Testing
- Large directory handling validated
- Memory usage profiled and optimized
- Search performance benchmarked
- UI responsiveness maintained under load

### Integration Testing
- tview component integration verified
- Keyboard shortcuts tested thoroughly
- Error handling and edge cases covered
- Cross-platform compatibility ensured

## Deployment Notes

### Configuration
- Security settings should be environment-specific
- File size limits based on available memory
- Search limits based on expected usage patterns
- Allowed paths configured per deployment

### Monitoring
- File access patterns should be logged
- Security violations tracked and alerted
- Performance metrics collected
- Error rates monitored

### Maintenance
- Regular security audits recommended
- Dependency updates for tview/tcell
- Performance tuning based on usage patterns
- User feedback integration for UX improvements

## Future Enhancements

### Potential Features
- File editing capabilities
- Git integration and status display  
- Plugin system for custom file types
- Network file system support
- Collaborative features

### Performance Improvements
- Virtual scrolling for very large directories
- Background indexing for faster search
- Thumbnail generation for images
- Syntax highlighting in preview

### Security Enhancements
- File integrity verification
- Digital signature validation
- Audit logging for compliance
- Role-based access control

---

This enhanced file navigation system provides a secure, performant, and user-friendly foundation for file operations in the Vibes MCP CLI, with seamless integration to Claude Code workflows.