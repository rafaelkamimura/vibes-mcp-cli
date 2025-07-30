# Developer Guide

This comprehensive developer guide covers extending the vibes-mcp-cli system, implementing new features, testing patterns, and best practices for contributing to the codebase. Whether you're adding new UI components, extending session management, or implementing custom providers, this guide will help you navigate the system architecture effectively.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)
- [Architecture Overview](#architecture-overview)
- [Extending Core Components](#extending-core-components)
- [Adding UI Components](#adding-ui-components)
- [Implementing Custom Providers](#implementing-custom-providers)
- [Testing Patterns](#testing-patterns)
- [Security Considerations](#security-considerations)
- [Performance Guidelines](#performance-guidelines)
- [Contribution Guidelines](#contribution-guidelines)

## Development Environment Setup

### Prerequisites

**Required Tools**:
- Go 1.21 or later
- Git 2.30 or later
- Make (for build automation)
- Docker (for containerized testing)

**Recommended Tools**:
- Visual Studio Code with Go extension
- golangci-lint for code linting
- gopls language server
- dlv debugger

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/your-org/vibes-mcp-cli.git
cd vibes-mcp-cli

# Install dependencies
go mod download

# Install development tools
make install-tools

# Run initial setup
make init

# Verify setup
make test
make lint
```

### Development Configuration

Create a development configuration file:

```yaml
# .dev-config.yaml
api_key: "dev-api-key"
provider: "openai"
log_level: "debug"

security:
  allowed_paths:
    - "/tmp/dev-workspace"
    - "~/projects"
  max_file_size: 1048576  # 1MB for dev
  allow_hidden: true  # Allow hidden files in dev

session_manager:
  storage_path: "./dev-sessions"
  max_sessions: 3
  auto_cleanup: false  # Manual cleanup in dev

ui:
  theme: "dark"
  enable_mouse: true
  vim_mode: false
```

### Project Structure

```
vibes-mcp-cli/
├── cmd/                    # CLI entry points
├── internal/               # Internal packages
│   ├── app/               # Core application logic
│   │   ├── claude/        # Claude Code integration
│   │   ├── files/         # File navigation system
│   │   ├── session/       # Session management
│   │   └── testutil/      # Testing utilities
│   ├── client/            # HTTP clients
│   ├── config/            # Configuration management
│   ├── mcp/               # MCP protocol
│   ├── providers/         # LLM providers
│   ├── service/           # Service layer
│   └── ui/                # UI components
├── docs/                  # Documentation
├── scripts/               # Build and development scripts
├── testdata/              # Test data and fixtures
├── .github/               # GitHub workflows
├── Makefile               # Build automation
├── go.mod                 # Go modules
└── go.sum                 # Go module checksums
```

## Architecture Overview

### Design Patterns

The system uses several key design patterns:

#### 1. Layered Architecture

```go
// Presentation Layer
type UIComponent interface {
    Render() error
    HandleInput(event *tcell.EventKey) bool
    Focus() bool
}

// Application Layer  
type SessionManager interface {
    CreateSession(name string, config *SessionConfig) (*Session, error)
    // ... other methods
}

// Infrastructure Layer
type Navigator interface {
    SetRoot(path string) error
    Navigate(path string) error
    // ... other methods
}
```

#### 2. Dependency Injection

```go
// Constructor pattern with dependency injection
func NewFileExplorer(
    config *FileExplorerConfig,
    navigator files.Navigator,
    callbacks *FileExplorerCallbacks,
) *FileExplorer {
    return &FileExplorer{
        config:    config,
        navigator: navigator,
        callbacks: callbacks,
    }
}
```

#### 3. Observer Pattern

```go
// Event-driven architecture
type EventSubscriber interface {
    OnEvent(event Event) error
}

type EventPublisher struct {
    subscribers []EventSubscriber
}

func (ep *EventPublisher) Publish(event Event) {
    for _, subscriber := range ep.subscribers {
        go subscriber.OnEvent(event)
    }
}
```

## Extending Core Components

### Adding New File Types

#### 1. Define File Type

```go
// internal/app/files/syntax.go
const (
    // ... existing types
    FileTypeRust FileType = iota + 100  // Avoid conflicts
    FileTypeSwift
    FileTypeKotlin
)

func (ft FileType) String() string {
    switch ft {
    // ... existing cases
    case FileTypeRust:
        return "Rust"
    case FileTypeSwift:
        return "Swift"
    case FileTypeKotlin:
        return "Kotlin"
    default:
        return "Unknown"
    }
}
```

#### 2. Add Detection Logic

```go
func (sd *SyntaxDetector) detectFileType(filename string) FileType {
    ext := strings.ToLower(filepath.Ext(filename))
    
    switch ext {
    // ... existing cases
    case ".rs":
        return FileTypeRust
    case ".swift":
        return FileTypeSwift
    case ".kt", ".kts":
        return FileTypeKotlin
    default:
        return sd.detectByContent(filename)
    }
}
```

#### 3. Add Icon Support

```go
func (ft FileType) Icon() string {
    switch ft {
    // ... existing cases
    case FileTypeRust:
        return "🦀"
    case FileTypeSwift:  
        return "🐦"
    case FileTypeKotlin:
        return "🅺"
    default:
        return "📄"
    }
}
```

#### 4. Add Tests

```go
// internal/app/files/syntax_test.go
func TestSyntaxDetector_DetectRustFiles(t *testing.T) {
    detector := NewSyntaxDetector()
    
    tests := []struct {
        filename string
        expected FileType
    }{
        {"main.rs", FileTypeRust},
        {"lib.rs", FileTypeRust},
        {"Cargo.toml", FileTypeTOML},
    }
    
    for _, tt := range tests {
        t.Run(tt.filename, func(t *testing.T) {
            result := detector.DetectFileType(tt.filename)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Extending Session Management

#### 1. Custom Session Types

```go
// internal/app/session/types.go
type SessionType int

const (
    SessionTypeStandard SessionType = iota
    SessionTypeDebug
    SessionTypeTraining
    SessionTypeCustom
)

type CustomSessionConfig struct {
    *SessionConfig
    Type        SessionType           `json:"type"`
    CustomArgs  []string             `json:"custom_args"`
    Hooks       map[string]string    `json:"hooks"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

#### 2. Session Hooks

```go
type SessionHooks struct {
    OnStart    func(*Session) error
    OnInput    func(*Session, string) (string, error)
    OnOutput   func(*Session, []byte) ([]byte, error)
    OnPause    func(*Session) error
    OnResume   func(*Session) error
    OnTerminate func(*Session) error
}

func (s *Session) executeHook(hookName string, args ...interface{}) error {
    if s.hooks == nil {
        return nil
    }
    
    hook, exists := s.hooks.GetHook(hookName)
    if !exists {
        return nil
    }
    
    return hook.Execute(s, args...)
}
```

#### 3. Session Plugins

```go
type SessionPlugin interface {
    Name() string
    Version() string
    Initialize(session *Session) error
    ProcessInput(input string) (string, error)
    ProcessOutput(output []byte) ([]byte, error)
    Cleanup() error
}

type PluginManager struct {
    plugins map[string]SessionPlugin
    mu      sync.RWMutex
}

func (pm *PluginManager) RegisterPlugin(plugin SessionPlugin) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    if _, exists := pm.plugins[plugin.Name()]; exists {
        return fmt.Errorf("plugin %s already registered", plugin.Name())
    }
    
    pm.plugins[plugin.Name()] = plugin
    return nil
}
```

### Custom Security Validators

#### 1. Validator Interface

```go
type CustomValidator interface {
    Name() string
    ValidatePath(path string, context *ValidationContext) error
    ValidateContent(content []byte, context *ValidationContext) error
    Priority() int  // Higher priority validators run first
}

type ValidationContext struct {
    UserID    string
    SessionID string
    Operation string
    Metadata  map[string]interface{}
}
```

#### 2. Example Custom Validator

```go
type ProjectBoundaryValidator struct {
    projectRoots []string
}

func (pbv *ProjectBoundaryValidator) Name() string {
    return "project-boundary"
}

func (pbv *ProjectBoundaryValidator) ValidatePath(path string, ctx *ValidationContext) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    
    for _, root := range pbv.projectRoots {
        if strings.HasPrefix(absPath, root) {
            return nil  // Path is within allowed project
        }
    }
    
    return &SecurityError{
        Path:   path,
        Reason: "path outside project boundaries",
    }
}

func (pbv *ProjectBoundaryValidator) Priority() int {
    return 100  // High priority
}
```

## Adding UI Components

### Component Architecture

#### 1. Component Interface

```go
// internal/ui/component.go
type Component interface {
    tview.Primitive
    
    // Lifecycle methods
    Initialize() error
    Start() error
    Stop() error
    
    // Event handling
    HandleEvent(event Event) bool
    
    // Focus management
    CanFocus() bool
    SetFocusFunc(func(tview.Primitive))
}

type BaseComponent struct {
    *tview.Box
    name      string
    focused   bool
    callbacks ComponentCallbacks
}
```

#### 2. Creating a Custom Component

```go
// internal/ui/components/log_viewer.go
type LogViewer struct {
    *BaseComponent
    textView  *tview.TextView
    logFile   string
    following bool
    filter    *regexp.Regexp
}

func NewLogViewer(config *LogViewerConfig) *LogViewer {
    lv := &LogViewer{
        BaseComponent: NewBaseComponent("log-viewer"),
        textView:      tview.NewTextView(),
        logFile:       config.LogFile,
        following:     config.Follow,
    }
    
    lv.setupUI()
    lv.setupKeyBindings()
    
    return lv
}

func (lv *LogViewer) setupUI() {
    lv.textView.SetBorder(true)
    lv.textView.SetTitle("📄 Log Viewer")
    lv.textView.SetScrollable(true)
    lv.textView.SetWrap(true)
    lv.textView.SetChangedFunc(func() {
        if lv.following {
            lv.textView.ScrollToEnd()
        }
    })
}

func (lv *LogViewer) setupKeyBindings() {
    lv.textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Key() {
        case tcell.KeyRune:
            switch event.Rune() {
            case 'f', 'F':
                lv.toggleFollow()
                return nil
            case '/':
                lv.showSearchDialog()
                return nil
            }
        }
        return event
    })
}
```

#### 3. Component Configuration

```go
type LogViewerConfig struct {
    LogFile     string   `yaml:"log_file"`
    Follow      bool     `yaml:"follow"`
    MaxLines    int      `yaml:"max_lines"`
    FilterLevel string   `yaml:"filter_level"`
    ShowTime    bool     `yaml:"show_time"`
    Patterns    []string `yaml:"highlight_patterns"`
}

func DefaultLogViewerConfig() *LogViewerConfig {
    return &LogViewerConfig{
        Follow:      true,
        MaxLines:    1000,
        FilterLevel: "info",
        ShowTime:    true,
    }
}
```

### Component Registration

#### 1. Component Registry

```go
// internal/ui/registry.go
type ComponentRegistry struct {
    factories map[string]ComponentFactory
    mu        sync.RWMutex
}

type ComponentFactory func(config interface{}) (Component, error)

func (cr *ComponentRegistry) Register(name string, factory ComponentFactory) {
    cr.mu.Lock()
    defer cr.mu.Unlock()
    cr.factories[name] = factory
}

func (cr *ComponentRegistry) Create(name string, config interface{}) (Component, error) {
    cr.mu.RLock()
    factory, exists := cr.factories[name]
    cr.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("component %s not registered", name)
    }
    
    return factory(config)
}
```

#### 2. Auto-Registration

```go
// internal/ui/components/init.go
func init() {
    registry := GetComponentRegistry()
    
    // Register built-in components
    registry.Register("file-explorer", func(config interface{}) (Component, error) {
        cfg := config.(*FileExplorerConfig)
        return NewFileExplorer(cfg, nil), nil
    })
    
    registry.Register("log-viewer", func(config interface{}) (Component, error) {
        cfg := config.(*LogViewerConfig)
        return NewLogViewer(cfg), nil
    })
}
```

## Implementing Custom Providers

### Provider Interface

#### 1. Base Provider Interface

```go
// internal/providers/provider.go
type Provider interface {
    Name() string
    Completions(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    Embeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
    Models(ctx context.Context) ([]Model, error)
    Close() error
}

type ProviderConfig struct {
    Name      string                 `json:"name"`
    BaseURL   string                 `json:"base_url"`
    APIKey    string                 `json:"api_key"`
    Options   map[string]interface{} `json:"options"`
    RateLimit *RateLimitConfig       `json:"rate_limit,omitempty"`
    Retry     *RetryConfig           `json:"retry,omitempty"`
}
```

#### 2. Custom Provider Implementation

```go
// internal/providers/custom/custom.go
type CustomProvider struct {
    config *ProviderConfig
    client *http.Client
    logger *zap.Logger
}

func NewCustomProvider(config *ProviderConfig, logger *zap.Logger) (*CustomProvider, error) {
    if config.APIKey == "" {
        return nil, errors.New("API key is required")
    }
    
    client := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        10,
            IdleConnTimeout:     90 * time.Second,
            DisableCompression:  true,
        },
    }
    
    return &CustomProvider{
        config: config,
        client: client,
        logger: logger,
    }, nil
}

func (cp *CustomProvider) Name() string {
    return cp.config.Name
}

func (cp *CustomProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // Convert request to provider format
    providerReq := cp.convertChatRequest(req)
    
    // Make HTTP request
    resp, err := cp.makeRequest(ctx, "POST", "/chat/completions", providerReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // Parse response
    var providerResp CustomChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&providerResp); err != nil {
        return nil, err
    }
    
    // Convert to standard format
    return cp.convertChatResponse(providerResp), nil
}
```

#### 3. Provider Registration

```go
// internal/providers/registry.go
type ProviderRegistry struct {
    factories map[string]ProviderFactory
    mu        sync.RWMutex
}

type ProviderFactory func(*ProviderConfig, *zap.Logger) (Provider, error)

func (pr *ProviderRegistry) Register(name string, factory ProviderFactory) {
    pr.mu.Lock()
    defer pr.mu.Unlock()
    pr.factories[name] = factory
}

// Register custom provider
func init() {
    registry := GetProviderRegistry()
    registry.Register("custom", func(config *ProviderConfig, logger *zap.Logger) (Provider, error) {
        return NewCustomProvider(config, logger)
    })
}
```

## Testing Patterns

### Unit Testing

#### 1. Test Structure

```go
// internal/app/files/navigator_test.go
func TestNavigator_SetRoot(t *testing.T) {
    tests := []struct {
        name        string
        path        string
        wantErr     bool
        expectedErr string
    }{
        {
            name:    "valid path",
            path:    "/tmp",
            wantErr: false,
        },
        {
            name:        "forbidden path",
            path:        "/etc",
            wantErr:     true,
            expectedErr: "security violation",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config := &SecurityConfig{
                AllowedPaths:   []string{"/tmp"},
                ForbiddenPaths: []string{"/etc"},
            }
            
            navigator := NewNavigator(config)
            err := navigator.SetRoot(tt.path)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 2. Mocking

```go
// internal/app/testutil/mocks.go
type MockNavigator struct {
    mock.Mock
}

func (m *MockNavigator) SetRoot(path string) error {
    args := m.Called(path)
    return args.Error(0)
}

func (m *MockNavigator) Navigate(path string) error {
    args := m.Called(path)
    return args.Error(0)
}

// Usage in tests
func TestFileExplorer_SetCurrentPath(t *testing.T) {
    mockNav := &MockNavigator{}
    mockNav.On("Navigate", "/test/path").Return(nil)
    
    explorer := &FileExplorer{navigator: mockNav}
    err := explorer.SetCurrentPath("/test/path")
    
    assert.NoError(t, err)
    mockNav.AssertExpectations(t)
}
```

#### 3. Test Utilities

```go
// internal/app/testutil/testutil.go
func CreateTempDir(t *testing.T) string {
    dir, err := ioutil.TempDir("", "vibes-test-*")
    require.NoError(t, err)
    
    t.Cleanup(func() {
        os.RemoveAll(dir)
    })
    
    return dir
}

func CreateTestFile(t *testing.T, dir, filename, content string) string {
    path := filepath.Join(dir, filename)
    err := ioutil.WriteFile(path, []byte(content), 0644)
    require.NoError(t, err)
    return path
}

func CreateTestSession(t *testing.T) *Session {
    config := DefaultSessionConfig()
    config.Name = "test-session"
    
    session := NewSession("test-id", "test", config, nil, nil, CreateTempDir(t))
    return session
}
```

### Integration Testing

#### 1. Component Integration Tests

```go
// internal/ui/components/file_explorer_integration_test.go
func TestFileExplorer_Integration(t *testing.T) {
    // Setup test environment
    tempDir := testutil.CreateTempDir(t)
    testutil.CreateTestFile(t, tempDir, "test.go", "package main")
    testutil.CreateTestFile(t, tempDir, "README.md", "# Test")
    
    // Create navigator with test config
    config := &files.SecurityConfig{
        AllowedPaths: []string{tempDir},
        MaxFileSize:  1024,
    }
    navigator := files.NewNavigator(config)
    
    // Create file explorer
    explorerConfig := &FileExplorerConfig{
        RootPath:     tempDir,
        EnableSearch: true,
    }
    
    var actionPath string
    callbacks := &FileExplorerCallbacks{
        OnFileAction: func(action FileExplorerAction, path string) {
            actionPath = path
        },
    }
    
    explorer := NewFileExplorer(explorerConfig, callbacks)
    
    // Test navigation and file actions
    err := explorer.SetCurrentPath(tempDir)
    assert.NoError(t, err)
    
    // Simulate file selection and action
    // This would require more complex TUI testing framework
    // For now, test the underlying logic
    assert.Equal(t, tempDir, explorer.GetCurrentPath())
}
```

#### 2. End-to-End Testing

```go
// test/e2e/session_management_test.go
func TestSessionManagement_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test in short mode")
    }
    
    // Start test server
    server := startTestServer(t)
    defer server.Close()
    
    // Create session manager
    config := session.DefaultManagerConfig()
    config.StoragePath = testutil.CreateTempDir(t)
    
    manager, err := session.NewManager(config, zap.NewNop())
    require.NoError(t, err)
    defer manager.Close()
    
    // Test full session lifecycle
    sess, err := manager.CreateSession("e2e-test", nil)
    require.NoError(t, err)
    
    err = manager.StartSession(sess.GetID())
    require.NoError(t, err)
    
    err = manager.SendInput(sess.GetID(), "help\n")
    require.NoError(t, err)
    
    // Wait for output
    time.Sleep(time.Second)
    
    output, err := manager.GetOutput(sess.GetID())
    require.NoError(t, err)
    assert.Contains(t, string(output), "help")
    
    err = manager.TerminateSession(sess.GetID())
    require.NoError(t, err)
}
```

### Performance Testing

#### 1. Benchmark Tests

```go
// internal/app/files/search_benchmark_test.go
func BenchmarkFileSearcher_Search(b *testing.B) {
    // Setup large directory structure
    tempDir := createLargeDirectoryStructure(b, 1000) // 1000 files
    
    searcher := NewFileSearcher(&SecurityValidator{})
    options := &SearchOptions{
        Pattern:    "*.go",
        MaxResults: 100,
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := searcher.Search(context.Background(), tempDir, options)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkNavigator_LoadChildren(b *testing.B) {
    tempDir := createLargeDirectory(b, 500) // 500 files per directory
    
    config := &SecurityConfig{
        AllowedPaths: []string{tempDir},
        MaxFileSize:  1024,
    }
    navigator := NewNavigator(config)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        node := &FileNode{Path: tempDir, IsLoaded: false}
        err := navigator.LoadChildren(node)
        if err != nil {
            b.Fatal(err)
        }
        node.IsLoaded = false // Reset for next iteration
    }
}
```

#### 2. Memory Testing

```go
// internal/app/session/memory_test.go
func TestSessionManager_MemoryUsage(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping memory test in short mode")
    }
    
    // Measure baseline memory
    var m1 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    // Create and use session manager
    config := DefaultManagerConfig()
    config.MaxSessions = 100
    
    manager, err := NewManager(config, zap.NewNop())
    require.NoError(t, err)
    
    // Create many sessions
    for i := 0; i < 50; i++ {
        sess, err := manager.CreateSession(fmt.Sprintf("test-%d", i), nil)
        require.NoError(t, err)
        
        err = manager.StartSession(sess.GetID())
        require.NoError(t, err)
    }
    
    // Measure memory after session creation
    var m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    memoryUsed := m2.Alloc - m1.Alloc
    t.Logf("Memory used for 50 sessions: %d bytes", memoryUsed)
    
    // Memory usage should be reasonable (< 100MB for 50 sessions)
    assert.Less(t, memoryUsed, uint64(100*1024*1024))
    
    manager.Close()
    
    // Verify cleanup
    var m3 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m3)
    
    // Memory should be mostly cleaned up
    assert.Less(t, m3.Alloc-m1.Alloc, memoryUsed/2)
}
```

## Security Considerations

### Secure Development Practices

#### 1. Input Validation

```go
// Always validate user inputs
func validateSessionName(name string) error {
    if len(name) == 0 {
        return errors.New("session name cannot be empty")
    }
    
    if len(name) > 100 {
        return errors.New("session name too long")
    }
    
    // Check for dangerous characters
    if matched, _ := regexp.MatchString(`[<>&"'/\\]`, name); matched {
        return errors.New("session name contains invalid characters")
    }
    
    return nil
}
```

#### 2. Error Handling

```go
// Don't leak sensitive information in errors
func (n *Navigator) ReadFile(path string) ([]byte, error) {
    if err := n.validator.ValidatePath(path); err != nil {
        // Log the actual error but return generic message
        n.logger.Warn("file access denied", zap.String("path", path), zap.Error(err))
        return nil, errors.New("file access denied")
    }
    
    // ... rest of implementation
}
```

#### 3. Resource Management

```go
// Always clean up resources
func (s *Session) processLargeFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close() // Always close
    
    // Use limited reader to prevent memory exhaustion
    reader := io.LimitReader(file, maxFileSize)
    
    // Process with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    return s.processWithContext(ctx, reader)
}
```

### Security Testing

#### 1. Vulnerability Testing

```go
// internal/app/files/security_test.go
func TestNavigator_PathTraversalPrevention(t *testing.T) {
    config := &SecurityConfig{
        AllowedPaths: []string{"/tmp"},
    }
    navigator := NewNavigator(config)
    
    vulnerablePaths := []string{
        "../../../etc/passwd",
        "..%2F..%2F..%2Fetc%2Fpasswd",
        "....//....//etc/passwd",
        "/tmp/../../../etc/passwd",
        "~/../../etc/passwd",
    }
    
    for _, path := range vulnerablePaths {
        t.Run(path, func(t *testing.T) {
            err := navigator.ValidatePath(path)
            assert.Error(t, err, "path traversal should be blocked: %s", path)
        })
    }
}
```

#### 2. Fuzzing

```go
// internal/app/files/fuzz_test.go
func FuzzNavigator_ValidatePath(f *testing.F) {
    f.Add("/tmp/test")
    f.Add("../../../etc/passwd")
    f.Add("normal/path/file.txt")
    
    config := &SecurityConfig{
        AllowedPaths: []string{"/tmp"},
    }
    navigator := NewNavigator(config)
    
    f.Fuzz(func(t *testing.T, path string) {
        // Should not panic regardless of input
        navigator.ValidatePath(path)
    })
}
```

## Performance Guidelines

### Optimization Patterns

#### 1. Lazy Loading

```go
// Load data only when needed
type LazyFileTree struct {
    root     *FileNode
    loader   func(path string) ([]*FileNode, error)
    cache    map[string][]*FileNode
    cacheMu  sync.RWMutex
}

func (lft *LazyFileTree) GetChildren(node *FileNode) ([]*FileNode, error) {
    if node.IsLoaded {
        return node.Children, nil
    }
    
    // Check cache first
    lft.cacheMu.RLock()
    if cached, exists := lft.cache[node.Path]; exists {
        lft.cacheMu.RUnlock()
        node.Children = cached
        node.IsLoaded = true
        return cached, nil
    }
    lft.cacheMu.RUnlock()
    
    // Load children
    children, err := lft.loader(node.Path)
    if err != nil {
        return nil, err
    }
    
    // Cache results
    lft.cacheMu.Lock()
    lft.cache[node.Path] = children
    lft.cacheMu.Unlock()
    
    node.Children = children
    node.IsLoaded = true
    
    return children, nil
}
```

#### 2. Connection Pooling

```go
// Reuse connections for better performance
type HTTPClientPool struct {
    clients chan *http.Client
    factory func() *http.Client
}

func NewHTTPClientPool(size int, factory func() *http.Client) *HTTPClientPool {
    pool := &HTTPClientPool{
        clients: make(chan *http.Client, size),
        factory: factory,
    }
    
    // Pre-populate pool
    for i := 0; i < size; i++ {
        pool.clients <- factory()
    }
    
    return pool
}

func (p *HTTPClientPool) Get() *http.Client {
    select {
    case client := <-p.clients:
        return client
    default:
        // Pool empty, create new client
        return p.factory()
    }
}

func (p *HTTPClientPool) Put(client *http.Client) {
    select {
    case p.clients <- client:
        // Successfully returned to pool
    default:
        // Pool full, discard client
    }
}
```

#### 3. Memory Optimization

```go
// Use object pools for frequently allocated objects
var fileNodePool = sync.Pool{
    New: func() interface{} {
        return &FileNode{}
    },
}

func NewFileNode(path, name string) *FileNode {
    node := fileNodePool.Get().(*FileNode)
    node.Path = path
    node.Name = name
    node.IsLoaded = false
    node.Children = nil
    return node
}

func (fn *FileNode) Release() {
    // Clear references to prevent memory leaks
    fn.Path = ""
    fn.Name = ""
    fn.Children = nil
    fn.Parent = nil
    
    fileNodePool.Put(fn)
}
```

## Contribution Guidelines

### Code Standards

#### 1. Code Style

Follow the official Go code style:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Check for common issues
go vet ./...
```

#### 2. Documentation

```go
// Package documentation
// Package files provides secure file navigation and search capabilities
// for the vibes-mcp-cli system.
package files

// Public function documentation
// NewNavigator creates a new secure file navigator with the provided configuration.
// The navigator validates all file operations against the security configuration
// to prevent path traversal and unauthorized access.
//
// Parameters:
//   - config: Security configuration defining allowed/forbidden paths and limits
//
// Returns:
//   - *Navigator: Configured navigator instance
//
// Example:
//   config := &SecurityConfig{
//       AllowedPaths: []string{"/home/user/projects"},
//       MaxFileSize:  10 * 1024 * 1024,
//   }
//   navigator := NewNavigator(config)
func NewNavigator(config *SecurityConfig) *Navigator {
    // Implementation...
}
```

#### 3. Error Handling

```go
// Use custom error types for better error handling
type FileNavigationError struct {
    Path      string
    Operation string
    Cause     error
}

func (e *FileNavigationError) Error() string {
    return fmt.Sprintf("file navigation error: %s operation on %s failed: %v", 
        e.Operation, e.Path, e.Cause)
}

func (e *FileNavigationError) Unwrap() error {
    return e.Cause
}

// Usage
func (n *Navigator) Navigate(path string) error {
    if err := n.validatePath(path); err != nil {
        return &FileNavigationError{
            Path:      path,
            Operation: "navigate",
            Cause:     err,
        }
    }
    
    // ... implementation
}
```

### Pull Request Process

#### 1. Branch Naming

```bash
# Feature branches
git checkout -b feature/add-rust-support

# Bug fixes
git checkout -b fix/session-memory-leak

# Documentation
git checkout -b docs/update-api-reference
```

#### 2. Commit Messages

Follow conventional commit format:

```
feat(files): add Rust file type support

- Add FileTypeRust constant and detection logic
- Include Rust icon (🦀) and syntax highlighting
- Add comprehensive tests for Rust file detection
- Update documentation with new file type

Closes #123
```

#### 3. Testing Requirements

Before submitting a PR:

```bash
# Run all tests
make test

# Run benchmarks
make bench

# Check test coverage
make coverage

# Lint code
make lint

# Build for all platforms
make build-all
```

#### 4. Documentation Updates

Update relevant documentation:
- API documentation for new interfaces
- User guide for new features
- Developer guide for new extension points
- Security guide for security-related changes

---

This comprehensive developer guide provides everything needed to extend and contribute to the vibes-mcp-cli system. The modular architecture makes it easy to add new features while maintaining security and performance standards. For specific implementation details, refer to the existing codebase and the [API Reference](API-Reference.md).