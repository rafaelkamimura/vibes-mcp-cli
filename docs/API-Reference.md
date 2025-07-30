# API Reference

This document provides comprehensive API documentation for the vibes-mcp-cli Claude Code session manager, covering all interfaces, methods, and configuration options for the new modular architecture.

## Table of Contents

- [Core Interfaces](#core-interfaces)
- [Session Management API](#session-management-api)
- [File Navigation API](#file-navigation-api)
- [UI Components API](#ui-components-api)
- [Configuration API](#configuration-api)
- [Error Handling](#error-handling)
- [Examples](#examples)

## Core Interfaces

### SessionManager Interface

The primary interface for managing Claude Code sessions.

```go
type SessionManager interface {
    // Session Lifecycle
    CreateSession(name string, config *SessionConfig) (*Session, error)
    GetSession(sessionID string) (*Session, error)
    GetSessionByName(name string) (*Session, error)
    DeleteSession(sessionID string, force bool) error
    
    // Session Operations  
    StartSession(sessionID string) error
    PauseSession(sessionID string) error
    ResumeSession(sessionID string) error
    TerminateSession(sessionID string) error
    
    // Session I/O
    SendInput(sessionID, input string) error
    GetOutput(sessionID string) ([]byte, error)
    SubscribeToOutput(sessionID string) (<-chan []byte, error)
    
    // Session Persistence
    SaveSession(sessionID string) error
    SaveAllSessions() error
    
    // Session Discovery
    ListSessions() []*Session
    ListActiveSessions() []*Session
    GetSessionCount() int
    GetActiveSessionCount() int
    
    // Statistics and Monitoring
    GetStats() *ManagerStats
    
    // Lifecycle
    Close() error
}
```

### Navigator Interface

Interface for secure file system navigation and operations.

```go
type Navigator interface {
    // Navigation
    SetRoot(path string) error
    Navigate(path string) error
    NavigateUp() error
    GetCurrentPath() string
    GetRoot() *FileNode
    
    // File Operations
    ReadFile(path string) ([]byte, error)
    GetFileInfo(path string) (*FileNode, error)
    GetFileType(path string) FileType
    IsTextFile(path string) bool
    
    // Directory Operations
    LoadChildren(node *FileNode) error
    RefreshNode(node *FileNode) error
    FindNode(path string) *FileNode
    
    // Search Operations
    Search(ctx context.Context, options *SearchOptions) ([]*SearchResult, error)
    QuickSearch(ctx context.Context, pattern string) ([]*SearchResult, error)
    
    // Navigation History
    CanNavigateBack() bool
    CanNavigateForward() bool
    GetBreadcrumb() []string
}
```

### Executor Interface

Interface for executing Claude Code processes and managing I/O.

```go
type Executor interface {
    // Process Management
    StartProcess(config *ProcessConfig) (*Process, error)
    KillProcess(pid int) error
    ListProcesses() []*ProcessInfo
    
    // Resource Monitoring
    GetResourceUsage(pid int) (*ResourceUsage, error)
    SetResourceLimits(limits *ResourceLimits) error
    
    // I/O Operations
    SendInput(pid int, input []byte) error
    GetOutput(pid int) []byte
    SubscribeToOutput(pid int) <-chan []byte
    
    // Lifecycle
    Close() error
}
```

## Session Management API

### SessionConfig Structure

Configuration for creating new sessions.

```go
type SessionConfig struct {
    Name        string            `json:"name"`        // Session display name
    WorkingDir  string            `json:"working_dir"` // Working directory for Claude Code
    Environment map[string]string `json:"environment"` // Environment variables
    Args        []string          `json:"args"`        // Command line arguments
    AutoSave    bool              `json:"auto_save"`   // Enable automatic saving
    MaxHistory  int               `json:"max_history"` // Maximum history entries
}

// Default configuration
func DefaultSessionConfig() *SessionConfig {
    return &SessionConfig{
        Environment: make(map[string]string),
        Args:        []string{},
        AutoSave:    true,
        MaxHistory:  1000,
    }
}
```

### Session States

Sessions progress through defined states during their lifecycle.

```go
type SessionState int

const (
    SessionStateCreated    SessionState = iota // Just created
    SessionStateActive                         // Running and accepting input
    SessionStatePaused                         // Paused but can be resumed
    SessionStateTerminated                     // Terminated and cannot be restarted
    SessionStateError                          // Error state requiring intervention
)
```

### SessionMetadata Structure

Metadata and statistics for sessions.

```go
type SessionMetadata struct {
    ID        string        `json:"id"`         // Unique session identifier
    Name      string        `json:"name"`       // Display name
    State     SessionState  `json:"state"`      // Current state
    Config    *SessionConfig `json:"config"`    // Session configuration
    CreatedAt time.Time     `json:"created_at"` // Creation timestamp
    UpdatedAt time.Time     `json:"updated_at"` // Last update timestamp
    Tags      []string      `json:"tags"`       // User-defined tags
    Stats     *SessionStats `json:"stats,omitempty"` // Usage statistics
}

type SessionStats struct {
    InputCount    int           `json:"input_count"`   // Number of inputs sent
    OutputBytes   int64         `json:"output_bytes"`  // Total output bytes
    Duration      time.Duration `json:"duration"`      // Total active duration
    LastActive    time.Time     `json:"last_active"`   // Last activity timestamp
    ProcessCount  int           `json:"process_count"` // Number of processes spawned
    ErrorCount    int           `json:"error_count"`   // Number of errors encountered
}
```

### Manager Configuration

Configuration for the session manager itself.

```go
type ManagerConfig struct {
    StoragePath     string        `json:"storage_path"`     // Session storage directory
    MaxSessions     int           `json:"max_sessions"`     // Maximum concurrent sessions
    DefaultTimeout  time.Duration `json:"default_timeout"`  // Default session timeout
    CleanupInterval time.Duration `json:"cleanup_interval"` // Cleanup interval
    AutoCleanup     bool          `json:"auto_cleanup"`     // Enable automatic cleanup
    ClaudePath      string        `json:"claude_path"`      // Path to Claude executable
}

func DefaultManagerConfig() *ManagerConfig {
    return &ManagerConfig{
        StoragePath:     "./sessions",
        MaxSessions:     10,
        DefaultTimeout:  time.Hour * 2,
        CleanupInterval: time.Hour,
        AutoCleanup:     true,
        ClaudePath:      "claude",
    }
}
```

## File Navigation API

### SecurityConfig Structure

Configuration for secure file access and validation.

```go
type SecurityConfig struct {
    AllowedPaths   []string `json:"allowed_paths"`   // Allowed base paths
    ForbiddenPaths []string `json:"forbidden_paths"` // Explicitly forbidden paths
    MaxDepth       int      `json:"max_depth"`       // Maximum directory depth
    AllowHidden    bool     `json:"allow_hidden"`    // Allow hidden files/directories
    MaxFileSize    int64    `json:"max_file_size"`   // Maximum file size for reading
}
```

### FileNode Structure

Represents a file or directory in the navigation tree.

```go
type FileNode struct {
    Path     string    `json:"path"`      // Full file path
    Name     string    `json:"name"`      // File/directory name
    IsDir    bool      `json:"is_dir"`    // True if directory
    Size     int64     `json:"size"`      // File size in bytes
    ModTime  time.Time `json:"mod_time"`  // Last modification time
    FileType FileType  `json:"file_type"` // Detected file type
    IsLoaded bool      `json:"is_loaded"` // True if children are loaded
    Children []*FileNode `json:"children,omitempty"` // Child nodes
    Parent   *FileNode `json:"-"`         // Parent node (not serialized)
}

// Display methods
func (fn *FileNode) GetDisplayName() string
func (fn *FileNode) GetIcon() string
```

### File Types

Comprehensive file type detection and classification.

```go
type FileType int

const (
    FileTypeUnknown FileType = iota
    FileTypeText
    FileTypeCode
    FileTypeGo
    FileTypePython
    FileTypeJavaScript
    FileTypeTypeScript
    FileTypeJava
    FileTypeC
    FileTypeCPP
    FileTypeRust
    FileTypeMarkdown
    FileTypeJSON
    FileTypeYAML
    FileTypeXML
    FileTypeHTML
    FileTypeCSS
    FileTypeSQL
    FileTypeShell
    FileTypeDockerfile
    FileTypeMakefile
    FileTypeImage
    FileTypeArchive
    FileTypeBinary
    FileTypeExecutable
)

// Methods
func (ft FileType) String() string
func (ft FileType) Icon() string
func (ft FileType) IsTextFile() bool
func (ft FileType) IsCodeFile() bool
```

### Search Operations

Advanced search functionality with security validation.

```go
type SearchOptions struct {
    Pattern       string     `json:"pattern"`        // Search pattern (regex or glob)
    IsRegex       bool       `json:"is_regex"`       // Treat pattern as regex
    CaseSensitive bool       `json:"case_sensitive"` // Case-sensitive search
    MaxResults    int        `json:"max_results"`    // Limit number of results
    FileTypes     []FileType `json:"file_types"`     // Filter by file types
    SearchContent bool       `json:"search_content"` // Search within file contents
    MaxFileSize   int64      `json:"max_file_size"`  // Maximum file size to search
    IncludeHidden bool       `json:"include_hidden"` // Include hidden files
}

type SearchResult struct {
    Path      string          `json:"path"`       // File path
    Name      string          `json:"name"`       // File name
    FileType  FileType        `json:"file_type"`  // File type
    MatchType SearchMatchType `json:"match_type"` // Type of match
    Snippet   string          `json:"snippet"`    // Content snippet (if content search)
    LineNum   int             `json:"line_num"`   // Line number (if content search)
}

type SearchMatchType int

const (
    MatchTypeName SearchMatchType = iota
    MatchTypePath
    MatchTypeContent
    MatchTypeExtension
)

func DefaultSearchOptions() *SearchOptions {
    return &SearchOptions{
        MaxResults:    100,
        CaseSensitive: false,
        MaxFileSize:   1024 * 1024, // 1MB
        IncludeHidden: false,
    }
}
```

## UI Components API

### FileExplorer Component

The main file exploration TUI component.

```go
type FileExplorer struct {
    *tview.Flex
    // ... (internal fields)
}

type FileExplorerConfig struct {
    RootPath        string   `json:"root_path"`         // Root directory path
    AllowedPaths    []string `json:"allowed_paths"`     // Allowed base paths
    ForbiddenPaths  []string `json:"forbidden_paths"`   // Forbidden paths
    ShowHiddenFiles bool     `json:"show_hidden_files"` // Show hidden files
    MaxFileSize     int64    `json:"max_file_size"`     // Maximum file size
    EnableSearch    bool     `json:"enable_search"`     // Enable search functionality
    EnablePreview   bool     `json:"enable_preview"`    // Enable file preview
}

type FileExplorerCallbacks struct {
    OnFileSelect      func(path string, fileType FileType)
    OnFileAction      func(action FileExplorerAction, path string)
    OnDirectoryChange func(path string)
    OnError           func(err error)
}

// Constructor
func NewFileExplorer(config *FileExplorerConfig, callbacks *FileExplorerCallbacks) *FileExplorer

// Public methods
func (fe *FileExplorer) SetCurrentPath(path string) error
func (fe *FileExplorer) GetCurrentPath() string
func (fe *FileExplorer) GetSelectedFiles() []string
func (fe *FileExplorer) ClearSelectedFiles()
```

### File Actions

Actions that can be performed on files in the explorer.

```go
type FileExplorerAction int

const (
    ActionView         FileExplorerAction = iota // View file content
    ActionEdit                                   // Edit file
    ActionSendToClaude                          // Send file to Claude Code
    ActionAddToSession                          // Add file to current session
    ActionCopy                                  // Copy file path
    ActionDelete                                // Delete file
    ActionRename                                // Rename file
    ActionRefresh                               // Refresh directory
)
```

### Explorer Modes

Different operational modes for the file explorer.

```go
type FileExplorerMode int

const (
    ModeBrowse FileExplorerMode = iota // Normal browsing mode
    ModeSearch                         // Search mode
    ModeSelect                         // File selection mode
)
```

## Configuration API

### Application Configuration

Main application configuration structure.

```go
type Config struct {
    // API Configuration
    APIKey      string `yaml:"api_key" env:"OPENAI_CLI_API_KEY"`
    BaseURL     string `yaml:"base_url" env:"OPENAI_CLI_BASE_URL"`
    Provider    string `yaml:"provider" env:"OPENAI_CLI_PROVIDER"`
    
    // Logging Configuration
    LogLevel    string `yaml:"log_level" env:"OPENAI_CLI_LOG_LEVEL"`
    
    // Authentication
    AuthToken   string `yaml:"auth_token"`
    
    // Session Management
    SessionConfig *ManagerConfig `yaml:"session_manager"`
    
    // File Navigation Security
    SecurityConfig *SecurityConfig `yaml:"security"`
    
    // UI Configuration
    UIConfig *UIConfig `yaml:"ui"`
    
    // Templates and Shortcuts
    Templates []string `yaml:"templates"`
}

type UIConfig struct {
    Theme           string `yaml:"theme"`            // UI theme
    ShowLineNumbers bool   `yaml:"show_line_numbers"` // Show line numbers in preview
    TabSize         int    `yaml:"tab_size"`         // Tab size for code display
    EnableMouse     bool   `yaml:"enable_mouse"`     // Enable mouse support
    VimMode         bool   `yaml:"vim_mode"`         // Enable vim-style navigation
}
```

### Environment Variables

Configuration can be provided via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_CLI_API_KEY` | OpenAI/Claude API key | - |
| `OPENAI_CLI_BASE_URL` | API base URL | `https://api.openai.com` |
| `OPENAI_CLI_PROVIDER` | LLM provider | `openai` |
| `OPENAI_CLI_LOG_LEVEL` | Logging level | `info` |
| `VIBES_SESSION_STORAGE` | Session storage path | `./sessions` |
| `VIBES_MAX_SESSIONS` | Maximum concurrent sessions | `10` |
| `VIBES_CLAUDE_PATH` | Path to Claude executable | `claude` |

## Error Handling

### Error Types

The system defines specific error types for different categories of failures.

```go
// Security-related errors
type SecurityError struct {
    Path   string
    Reason string
}

func (e *SecurityError) Error() string {
    return fmt.Sprintf("security violation: %s - %s", e.Path, e.Reason)
}

// Session-related errors
type SessionError struct {
    SessionID string
    Operation string
    Cause     error
}

func (e *SessionError) Error() string {
    return fmt.Sprintf("session %s operation %s failed: %v", e.SessionID, e.Operation, e.Cause)
}

// File operation errors
type FileOperationError struct {
    Path      string
    Operation string
    Cause     error
}

func (e *FileOperationError) Error() string {
    return fmt.Sprintf("file operation %s on %s failed: %v", e.Operation, e.Path, e.Cause)
}
```

### Error Handling Patterns

#### Session Operations

```go
session, err := manager.CreateSession("my-session", config)
if err != nil {
    switch e := err.(type) {
    case *SessionError:
        log.Printf("Session operation failed: %v", e)
    default:
        log.Printf("Unexpected error: %v", err)
    }
    return err
}
```

#### File Operations

```go
content, err := navigator.ReadFile(path)
if err != nil {
    switch e := err.(type) {
    case *SecurityError:
        log.Printf("Security violation: %v", e)
        // Handle security violation
    case *FileOperationError:
        log.Printf("File operation failed: %v", e)
        // Handle file operation failure
    default:
        log.Printf("Unexpected error: %v", err)
    }
    return nil, err
}
```

## Examples

### Basic Session Management

```go
// Create manager
config := session.DefaultManagerConfig()
config.MaxSessions = 5
config.StoragePath = "/tmp/sessions"

logger, _ := zap.NewDevelopment()
manager, err := session.NewManager(config, logger)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Create and start session
sessionConfig := session.DefaultSessionConfig()
sessionConfig.Name = "my-claude-session"
sessionConfig.WorkingDir = "/home/user/projects"

sess, err := manager.CreateSession("main", sessionConfig)
if err != nil {
    log.Fatal(err)
}

if err := manager.StartSession(sess.GetID()); err != nil {
    log.Fatal(err)
}

// Send input
if err := manager.SendInput(sess.GetID(), "help"); err != nil {
    log.Fatal(err)
}

// Get output
output, err := manager.GetOutput(sess.GetID())
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(output))
```

### File Navigation with Security

```go
// Configure security
securityConfig := &files.SecurityConfig{
    AllowedPaths:   []string{"/home/user", "/tmp"},
    ForbiddenPaths: []string{"/etc", "/root"},
    MaxFileSize:    10 * 1024 * 1024, // 10MB
    AllowHidden:    false,
    MaxDepth:       10,
}

// Create navigator
navigator := files.NewNavigator(securityConfig)
if err := navigator.SetRoot("/home/user/projects"); err != nil {
    log.Fatal(err)
}

// Navigate and search
if err := navigator.Navigate("/home/user/projects/myapp"); err != nil {
    log.Fatal(err)
}

searchOptions := &files.SearchOptions{
    Pattern:    "*.go",
    MaxResults: 50,
    FileTypes:  []files.FileType{files.FileTypeGo},
}

results, err := navigator.Search(context.Background(), searchOptions)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Found: %s (%s)\n", result.Path, result.FileType.String())
}
```

### File Explorer Component

```go
// Configure file explorer
config := &components.FileExplorerConfig{
    RootPath:        "/home/user/projects",
    ShowHiddenFiles: false,
    EnableSearch:    true,
    EnablePreview:   true,
    MaxFileSize:     10 * 1024 * 1024,
}

// Set up callbacks
callbacks := &components.FileExplorerCallbacks{
    OnFileAction: func(action components.FileExplorerAction, path string) {
        switch action {
        case components.ActionSendToClaude:
            // Send file to Claude Code session
            content, err := ioutil.ReadFile(path)
            if err != nil {
                log.Printf("Failed to read file: %v", err)
                return
            }
            // Send to active session...
            
        case components.ActionAddToSession:
            // Add file to current session file list
            sessionFiles = append(sessionFiles, path)
            
        case components.ActionCopy:
            // Copy path to clipboard
            copyToClipboard(path)
        }
    },
    OnError: func(err error) {
        log.Printf("File explorer error: %v", err)
    },
}

// Create component
fileExplorer := components.NewFileExplorer(config, callbacks)

// Integrate with tview application
app := tview.NewApplication()
app.SetRoot(fileExplorer, true)
if err := app.Run(); err != nil {
    log.Fatal(err)
}
```

### Resource Monitoring

```go
// Monitor session resources
stats := manager.GetStats()
fmt.Printf("Active sessions: %d/%d\n", stats.ActiveSessions, stats.TotalSessions)

// Monitor individual session
sessionMeta := session.GetMetadata()
fmt.Printf("Session %s:\n", sessionMeta.Name)
fmt.Printf("  State: %s\n", sessionMeta.State.String())
fmt.Printf("  Inputs: %d\n", sessionMeta.Stats.InputCount)
fmt.Printf("  Output: %d bytes\n", sessionMeta.Stats.OutputBytes)
fmt.Printf("  Last active: %s\n", sessionMeta.Stats.LastActive.Format(time.RFC3339))
```

---

This API reference provides comprehensive documentation of all interfaces, types, and usage patterns in the vibes-mcp-cli Claude Code session manager. For implementation details and architectural patterns, see the [Architecture Documentation](Architecture.md).