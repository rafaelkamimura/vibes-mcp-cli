# Prompt System Development Guide

This guide covers setting up the development environment, extending the prompt system, testing strategies, and contributing guidelines for the vibes-mcp-cli prompt system.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)  
- [Extending the System](#extending-the-system)
- [Adding New Template Types](#adding-new-template-types)
- [Extending AI Integrations](#extending-ai-integrations)
- [Testing Strategies](#testing-strategies)
- [Contributing Guidelines](#contributing-guidelines)
- [Code Standards](#code-standards)

## Development Environment Setup

### Prerequisites

- **Go 1.23+** with toolchain go1.24.0
- **Git** for version control
- **Make** for build automation
- **Docker** (optional, for containerized development)

### Initial Setup

1. **Clone and Setup**
   ```bash
   git clone https://github.com/your-org/vibes-mcp-cli.git
   cd vibes-mcp-cli
   
   # Initialize development environment
   make init
   ```

2. **Environment Configuration**
   ```bash
   # Copy example environment file
   cp .env_example .env
   
   # Edit with your configuration
   vim .env
   ```
   
   Required environment variables:
   ```bash
   OPENAI_CLI_API_KEY=your-anthropic-api-key
   OPENAI_CLI_BASE_URL=https://api.anthropic.com
   OPENAI_CLI_PROVIDER=anthropic
   ```

3. **Build and Test**
   ```bash
   # Build the project
   make build
   
   # Run tests
   make test
   
   # Run linting
   make lint
   ```

4. **Verify Installation**
   ```bash
   ./vibes-mcp-cli prompt list
   ./vibes-mcp-cli prompt workspace-status
   ```

### Development Workflow

1. **Start Development Server**
   ```bash
   # Run in development mode with hot reload
   make dev
   
   # Or run HTTP server for API testing
   make serve
   ```

2. **Test Changes**
   ```bash
   # Run specific tests
   go test ./internal/prompt/...
   
   # Run integration tests
   make test-integration
   
   # Run end-to-end tests
   make test-e2e
   ```

3. **Debug Mode**
   ```bash
   export LOG_LEVEL=debug
   ./vibes-mcp-cli prompt generate feature-development --interactive
   ```

### IDE Setup

#### VS Code Configuration

Create `.vscode/settings.json`:
```json
{
    "go.testFlags": ["-v"],
    "go.buildFlags": ["-v"],
    "go.lintTool": "golangci-lint",
    "go.formatTool": "goimports",
    "files.exclude": {
        "**/.git": true,
        "**/vendor": true,
        "**/*.exe": true
    },
    "go.testTimeout": "30s"
}
```

Create `.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug CLI",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/main.go",
            "args": ["prompt", "generate", "feature-development", "--interactive"],
            "env": {
                "LOG_LEVEL": "debug"
            }
        },
        {
            "name": "Debug Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/internal/prompt",
            "env": {
                "LOG_LEVEL": "debug"
            }
        }
    ]
}
```

## Extending the System

### Adding New Components

The prompt system follows a modular architecture. To add new components:

1. **Define the Interface**
   ```go
   // internal/prompt/interfaces.go
   type NewComponent interface {
       Process(ctx context.Context, input *Input) (*Output, error)
       Configure(config map[string]interface{}) error
       Validate() error
   }
   ```

2. **Implement the Component**
   ```go
   // internal/prompt/new_component.go
   package prompt
   
   import (
       "context"
       "go.uber.org/zap"
   )
   
   type NewComponentImpl struct {
       logger *zap.Logger
       config map[string]interface{}
   }
   
   func NewNewComponent(logger *zap.Logger) NewComponent {
       return &NewComponentImpl{
           logger: logger,
           config: make(map[string]interface{}),
       }
   }
   
   func (c *NewComponentImpl) Process(ctx context.Context, input *Input) (*Output, error) {
       c.logger.Debug("Processing input", zap.Any("input", input))
       
       // Implementation logic here
       
       return &Output{}, nil
   }
   
   func (c *NewComponentImpl) Configure(config map[string]interface{}) error {
       c.config = config
       return nil
   }
   
   func (c *NewComponentImpl) Validate() error {
       // Validation logic
       return nil
   }
   ```

3. **Add Tests**
   ```go
   // internal/prompt/new_component_test.go
   package prompt
   
   import (
       "context"
       "testing"
       "go.uber.org/zap/zaptest"
   )
   
   func TestNewComponent_Process(t *testing.T) {
       logger := zaptest.NewLogger(t)
       component := NewNewComponent(logger)
       
       input := &Input{
           // Test input
       }
       
       output, err := component.Process(context.Background(), input)
       if err != nil {
           t.Fatalf("Process failed: %v", err)
       }
       
       // Assertions
       if output == nil {
           t.Error("Expected output, got nil")
       }
   }
   ```

4. **Integrate with Manager**
   ```go
   // internal/prompt/manager.go
   func NewManager(cfg *config.Config, logger *zap.Logger) (Manager, error) {
       // ... existing code ...
       
       newComponent := NewNewComponent(logger)
       
       manager := &ManagerImpl{
           // ... existing fields ...
           newComponent: newComponent,
       }
       
       return manager, nil
   }
   ```

## Adding New Template Types

### 1. Define Template Type

```go
// internal/prompt/types.go
const (
    TemplateTypeStandard    = "standard"
    TemplateTypeMultiFile   = "multi-file"
    TemplateTypeInteractive = "interactive"
    TemplateTypeCustom      = "custom"     // New type
)

type CustomTemplate struct {
    Template
    CustomField string `json:"custom_field" yaml:"custom_field"`
    // Add custom fields
}
```

### 2. Extend Template Parser

```go
// internal/prompt/template.go
func (p *TemplateParserImpl) LoadTemplate(filePath string) (*Template, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    
    // Determine template type
    var baseTemplate struct {
        Type string `yaml:"type"`
    }
    
    if err := yaml.Unmarshal(data, &baseTemplate); err != nil {
        return nil, err
    }
    
    switch baseTemplate.Type {
    case TemplateTypeCustom:
        return p.parseCustomTemplate(data)
    default:
        return p.parseStandardTemplate(data)
    }
}

func (p *TemplateParserImpl) parseCustomTemplate(data []byte) (*Template, error) {
    var customTemplate CustomTemplate
    if err := yaml.Unmarshal(data, &customTemplate); err != nil {
        return nil, err
    }
    
    // Custom validation and processing
    if err := p.validateCustomTemplate(&customTemplate); err != nil {
        return nil, err
    }
    
    return &customTemplate.Template, nil
}
```

### 3. Extend Generator

```go
// internal/prompt/generator.go
func (g *GeneratorImpl) Generate(ctx context.Context, config *GenerationConfig) (*GenerationResult, error) {
    template, err := g.getTemplate(config.TemplateName)
    if err != nil {
        return nil, err
    }
    
    switch template.Type {
    case TemplateTypeCustom:
        return g.generateCustomTemplate(ctx, template, config)
    default:
        return g.generateStandardTemplate(ctx, template, config)
    }
}

func (g *GeneratorImpl) generateCustomTemplate(ctx context.Context, template *Template, config *GenerationConfig) (*GenerationResult, error) {
    // Custom generation logic
    return &GenerationResult{}, nil
}
```

### 4. Update CLI Commands

```go
// cmd/prompt.go
func runPromptGenerate(cmd *cobra.Command, args []string) error {
    // ... existing code ...
    
    // Handle custom template types
    if template.Type == prompt.TemplateTypeCustom {
        return handleCustomTemplateGeneration(template, config)
    }
    
    // ... existing code ...
}

func handleCustomTemplateGeneration(template *prompt.Template, config *prompt.GenerationConfig) error {
    // Custom CLI handling
    return nil
}
```

### 5. Create Template Examples

```yaml
# templates/custom/example-custom.yaml
type: "custom"
name: "custom-workflow"
category: "workflows"
description: "Custom workflow template with special processing"
custom_field: "special-value"

content: |
  # Custom Workflow: {{workflow_name}}
  
  ## Custom Processing
  This template uses custom processing logic.
  
  Custom Field: {{custom_field}}

parameters:
  - name: "workflow_name"
    description: "Name of the workflow"
    type: "string"
    required: true
    
  - name: "custom_field"
    description: "Custom field value"
    type: "string"
    required: false
    default: "default-value"
```

## Extending AI Integrations

### 1. Define AI Provider Interface

```go
// internal/prompt/ai_provider.go
package prompt

import "context"

type AIProvider interface {
    Name() string
    ProcessPrompt(ctx context.Context, request *AIRequest) (*AIResponse, error)
    ValidateConfig(config map[string]interface{}) error
    GetCapabilities() []string
}

type AIRequest struct {
    Content     string                 `json:"content"`
    Temperature float64                `json:"temperature"`
    MaxTokens   int                    `json:"max_tokens"`
    Context     *WorkspaceContext      `json:"context,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type AIResponse struct {
    Content        string                 `json:"content"`
    TokensUsed     int                    `json:"tokens_used"`
    ProcessingTime time.Duration          `json:"processing_time"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
```

### 2. Implement Custom AI Provider

```go
// internal/prompt/custom_ai_provider.go
package prompt

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
)

type CustomAIProvider struct {
    name     string
    apiKey   string
    baseURL  string
    client   *http.Client
}

func NewCustomAIProvider(config map[string]interface{}) (*CustomAIProvider, error) {
    apiKey, ok := config["api_key"].(string)
    if !ok {
        return nil, fmt.Errorf("api_key required for CustomAI provider")
    }
    
    baseURL, ok := config["base_url"].(string)
    if !ok {
        baseURL = "https://api.customai.com"
    }
    
    return &CustomAIProvider{
        name:    "custom-ai",
        apiKey:  apiKey,
        baseURL: baseURL,
        client:  &http.Client{Timeout: 60 * time.Second},
    }, nil
}

func (p *CustomAIProvider) Name() string {
    return p.name
}

func (p *CustomAIProvider) ProcessPrompt(ctx context.Context, request *AIRequest) (*AIResponse, error) {
    // Prepare API request
    payload := map[string]interface{}{
        "prompt":      request.Content,
        "temperature": request.Temperature,
        "max_tokens":  request.MaxTokens,
    }
    
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }
    
    // Make HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/generate", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)
    
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // Parse response
    var result struct {
        Content    string `json:"content"`
        TokensUsed int    `json:"tokens_used"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &AIResponse{
        Content:    result.Content,
        TokensUsed: result.TokensUsed,
    }, nil
}

func (p *CustomAIProvider) ValidateConfig(config map[string]interface{}) error {
    if _, ok := config["api_key"]; !ok {
        return fmt.Errorf("api_key is required")
    }
    return nil
}

func (p *CustomAIProvider) GetCapabilities() []string {
    return []string{"text-generation", "content-analysis"}
}
```

### 3. Register Provider

```go
// internal/prompt/ai_registry.go
package prompt

var aiProviders = make(map[string]func(map[string]interface{}) (AIProvider, error))

func RegisterAIProvider(name string, factory func(map[string]interface{}) (AIProvider, error)) {
    aiProviders[name] = factory
}

func CreateAIProvider(name string, config map[string]interface{}) (AIProvider, error) {
    factory, exists := aiProviders[name]
    if !exists {
        return nil, fmt.Errorf("unknown AI provider: %s", name)
    }
    
    return factory(config)
}

func init() {
    // Register built-in providers
    RegisterAIProvider("claude", NewClaudeProvider)
    RegisterAIProvider("custom-ai", func(config map[string]interface{}) (AIProvider, error) {
        return NewCustomAIProvider(config)
    })
}
```

### 4. Update Integration Layer

```go
// internal/prompt/integration.go
func (i *IntegratorImpl) processWithAI(ctx context.Context, providerName string, content string) (*AIResponse, error) {
    config := i.getProviderConfig(providerName)
    
    provider, err := CreateAIProvider(providerName, config)
    if err != nil {
        return nil, err
    }
    
    request := &AIRequest{
        Content:     content,
        Temperature: 0.7,
        MaxTokens:   4000,
    }
    
    return provider.ProcessPrompt(ctx, request)
}
```

## Testing Strategies

### Unit Testing

#### Template Parser Tests

```go
// internal/prompt/template_test.go
package prompt

import (
    "testing"
    "go.uber.org/zap/zaptest"
)

func TestTemplateParser_LoadTemplate(t *testing.T) {
    tests := []struct {
        name     string
        content  string
        expected *Template
        wantErr  bool
    }{
        {
            name: "valid template",
            content: `
name: "test-template"
category: "general"
description: "Test template"
content: "Hello {{name}}"
parameters:
  - name: "name"
    description: "Name parameter"
    type: "string"
    required: true
`,
            expected: &Template{
                Name:        "test-template",
                Category:    "general",
                Description: "Test template",
                Content:     "Hello {{name}}",
                Parameters: []TemplateParameter{
                    {
                        Name:        "name",
                        Description: "Name parameter",
                        Type:        "string",
                        Required:    true,
                    },
                },
            },
            wantErr: false,
        },
        {
            name:     "invalid yaml",
            content:  "invalid: yaml: content:",
            expected: nil,
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewTemplateParser(zaptest.NewLogger(t))
            
            // Create temporary file
            tmpFile := createTempFile(t, tt.content)
            defer os.Remove(tmpFile)
            
            template, err := parser.LoadTemplate(tmpFile)
            
            if tt.wantErr {
                if err == nil {
                    t.Error("Expected error, got nil")
                }
                return
            }
            
            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }
            
            // Compare templates
            if template.Name != tt.expected.Name {
                t.Errorf("Expected name %s, got %s", tt.expected.Name, template.Name)
            }
            
            // ... more assertions
        })
    }
}

func createTempFile(t *testing.T, content string) string {
    tmpFile, err := os.CreateTemp("", "template-*.yaml")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    
    if _, err := tmpFile.WriteString(content); err != nil {
        t.Fatalf("Failed to write temp file: %v", err)
    }
    
    if err := tmpFile.Close(); err != nil {
        t.Fatalf("Failed to close temp file: %v", err)
    }
    
    return tmpFile.Name()
}
```

#### Generator Tests

```go
// internal/prompt/generator_test.go
func TestGenerator_Generate(t *testing.T) {
    logger := zaptest.NewLogger(t)
    generator := NewGenerator(logger)
    
    template := &Template{
        Name:        "test-template",
        Content:     "Hello {{name}}, welcome to {{project}}!",
        Parameters: []TemplateParameter{
            {Name: "name", Type: "string", Required: true},
            {Name: "project", Type: "string", Required: true},
        },
    }
    
    config := &GenerationConfig{
        TemplateName: "test-template",
        Parameters: map[string]string{
            "name":    "John",
            "project": "vibes-mcp-cli",
        },
    }
    
    result, err := generator.Generate(context.Background(), config)
    if err != nil {
        t.Fatalf("Generation failed: %v", err)
    }
    
    expected := "Hello John, welcome to vibes-mcp-cli!"
    if result.Content != expected {
        t.Errorf("Expected content %q, got %q", expected, result.Content)
    }
    
    if result.WordCount != 6 {
        t.Errorf("Expected word count 6, got %d", result.WordCount)
    }
}
```

### Integration Testing

#### MCP Endpoint Tests

```go
// internal/mcp/prompt_integration_test.go
package mcp

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestPromptServer_GenerateEndpoint(t *testing.T) {
    // Setup test server
    server := NewPromptServer(testManager, testLogger)
    
    // Prepare request
    reqBody := map[string]interface{}{
        "template_name": "test-template",
        "parameters": map[string]string{
            "name": "test",
        },
    }
    
    reqData, _ := json.Marshal(reqBody)
    req := httptest.NewRequest("POST", "/v1/prompts/generate", bytes.NewBuffer(reqData))
    req.Header.Set("Content-Type", "application/json")
    
    // Create response recorder
    w := httptest.NewRecorder()
    
    // Call handler
    server.handleGenerate(w, req)
    
    // Check response
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    var response map[string]interface{}
    if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
        t.Fatalf("Failed to decode response: %v", err)
    }
    
    if _, exists := response["result"]; !exists {
        t.Error("Expected 'result' field in response")
    }
}
```

### End-to-End Testing

```go
// e2e/prompt_workflow_test.go
package e2e

import (
    "os/exec"
    "testing"
)

func TestPromptWorkflow(t *testing.T) {
    tests := []struct {
        name     string
        command  []string
        wantErr  bool
        contains string
    }{
        {
            name:     "list templates",
            command:  []string{"prompt", "list"},
            wantErr:  false,
            contains: "Available Prompt Templates",
        },
        {
            name:     "generate prompt",
            command:  []string{"prompt", "generate", "feature-development", "--repo", "test", "--language", "go"},
            wantErr:  false,
            contains: "Feature Development Request",
        },
        {
            name:     "validate templates",
            command:  []string{"prompt", "validate"},
            wantErr:  false,
            contains: "Template Validation Report",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := exec.Command("./vibes-mcp-cli", tt.command...)
            output, err := cmd.CombinedOutput()
            
            if tt.wantErr && err == nil {
                t.Error("Expected error, got nil")
            }
            
            if !tt.wantErr && err != nil {
                t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
            }
            
            if tt.contains != "" && !strings.Contains(string(output), tt.contains) {
                t.Errorf("Expected output to contain %q, got: %s", tt.contains, output)
            }
        })
    }
}
```

### Performance Testing

```go
// internal/prompt/performance_test.go
func BenchmarkTemplateGeneration(b *testing.B) {
    manager, err := prompt.NewManager(testConfig, testLogger)
    if err != nil {
        b.Fatal(err)
    }
    
    config := &prompt.GenerationConfig{
        TemplateName: "feature-development",
        Parameters: map[string]string{
            "repo":     "benchmark-repo",
            "language": "go",
        },
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := manager.GeneratePrompt(config)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkTemplateLoading(b *testing.B) {
    parser := prompt.NewTemplateParser(testLogger)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := parser.LoadTemplate("templates/feature-development.yaml")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Testing Utilities

```go
// internal/prompt/testutil/helpers.go
package testutil

import (
    "os"
    "testing"
    "go.uber.org/zap/zaptest"
    "openai-cli/internal/prompt"
)

func CreateTestManager(t *testing.T) prompt.Manager {
    logger := zaptest.NewLogger(t)
    config := &config.Config{
        APIKey:  "test-key",
        BaseURL: "https://api.example.com",
    }
    
    manager, err := prompt.NewManager(config, logger)
    if err != nil {
        t.Fatal(err)
    }
    
    return manager
}

func CreateTestTemplate(name, content string) *prompt.Template {
    return &prompt.Template{
        Name:        name,
        Category:    "test",
        Description: "Test template",
        Content:     content,
        Parameters: []prompt.TemplateParameter{
            {
                Name:     "test_param",
                Type:     "string",
                Required: true,
            },
        },
    }
}

func CreateTempDir(t *testing.T) string {
    dir, err := os.MkdirTemp("", "prompt-test-*")
    if err != nil {
        t.Fatal(err)
    }
    
    t.Cleanup(func() {
        os.RemoveAll(dir)
    })
    
    return dir
}
```

## Contributing Guidelines

### Code Review Process

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/prompt-enhancement
   ```

2. **Make Changes**
   - Follow coding standards
   - Add comprehensive tests
   - Update documentation

3. **Test Changes**
   ```bash
   make test
   make test-integration
   make lint
   ```

4. **Commit Changes**
   ```bash
   git add .
   git commit -m "feat(prompt): add custom template support

   - Add CustomTemplate type with additional fields
   - Extend parser to handle custom template format
   - Add CLI support for custom template generation
   - Include comprehensive tests and documentation
   
   Closes #123"
   ```

5. **Push and Create PR**
   ```bash
   git push origin feature/prompt-enhancement
   # Create pull request via GitHub/GitLab
   ```

### Commit Message Format

Follow conventional commits:

```
<type>(<scope>): <description>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

**Examples:**
```
feat(prompt): add multi-file template support

Add support for templates that generate multiple files
with coordinated content and structure.

Closes #456

fix(template): handle missing parameter defaults

Ensure template parameters work correctly when no
default value is specified.

Fixes #789

docs(api): update prompt generation examples

Add comprehensive examples for new template types
and integration patterns.
```

### Pull Request Guidelines

1. **PR Title**: Use conventional commit format
2. **Description**: Include:
   - What changes were made
   - Why the changes were necessary
   - How to test the changes
   - Any breaking changes

3. **Checklist**:
   - [ ] Tests added/updated
   - [ ] Documentation updated
   - [ ] Changelog updated
   - [ ] Breaking changes noted
   - [ ] Performance impact considered

### Release Process

1. **Version Bumping**
   ```bash
   # Update version in relevant files
   vim VERSION
   vim go.mod
   vim docs/API-Reference.md
   ```

2. **Changelog Update**
   ```bash
   # Add changes to CHANGELOG.md
   vim CHANGELOG.md
   ```

3. **Create Release**
   ```bash
   git tag -a v1.2.0 -m "Release v1.2.0"
   git push origin v1.2.0
   ```

## Code Standards

### Go Code Style

Follow standard Go conventions:

1. **Formatting**
   ```bash
   # Use gofmt and goimports
   gofmt -s -w .
   goimports -w .
   ```

2. **Naming Conventions**
   ```go
   // Use camelCase for private functions
   func parseTemplate() {}
   
   // Use PascalCase for public functions
   func ParseTemplate() {}
   
   // Use descriptive names
   func generatePromptFromTemplate() {} // Good
   func genPrmpt() {}                  // Bad
   ```

3. **Error Handling**
   ```go
   // Always handle errors
   result, err := manager.GeneratePrompt(config)
   if err != nil {
       return nil, fmt.Errorf("failed to generate prompt: %w", err)
   }
   
   // Use custom error types for specific errors
   if errors.Is(err, prompt.ErrTemplateNotFound) {
       // Handle specific error
   }
   ```

4. **Logging**
   ```go
   // Use structured logging
   logger.Info("Processing template",
       zap.String("template", templateName),
       zap.Int("parameters", len(parameters)))
   
   // Log errors with context
   logger.Error("Template generation failed",
       zap.String("template", templateName),
       zap.Error(err))
   ```

### Documentation Standards

1. **Package Documentation**
   ```go
   // Package prompt provides comprehensive prompt generation
   // and template management capabilities for the vibes-mcp-cli.
   //
   // The package supports multiple template formats, workspace
   // context detection, AI integration, and extensible validation.
   package prompt
   ```

2. **Function Documentation**
   ```go
   // GeneratePrompt creates a prompt from the specified template
   // with the given configuration.
   //
   // The generation process includes parameter validation,
   // template processing, and optional AI integration.
   //
   // Parameters:
   //   config: Generation configuration including template name and parameters
   //
   // Returns:
   //   result: Generated prompt with metadata and statistics
   //   error: Generation error if any step fails
   func (m *ManagerImpl) GeneratePrompt(config *GenerationConfig) (*GenerationResult, error) {
   ```

3. **Example Code**
   ```go
   // Example usage:
   //
   //   manager, err := prompt.NewManager(cfg, logger)
   //   if err != nil {
   //       log.Fatal(err)
   //   }
   //
   //   config := &prompt.GenerationConfig{
   //       TemplateName: "feature-development",
   //       Parameters: map[string]string{
   //           "repo": "myproject",
   //       },
   //   }
   //
   //   result, err := manager.GeneratePrompt(config)
   //   if err != nil {
   //       log.Fatal(err)
   //   }
   //
   //   fmt.Println(result.Content)
   ```

### Testing Standards

1. **Test Structure**
   ```go
   func TestComponent_Method(t *testing.T) {
       // Arrange
       manager := createTestManager(t)
       config := &Config{...}
       
       // Act
       result, err := manager.Method(config)
       
       // Assert
       if err != nil {
           t.Fatalf("Method failed: %v", err)
       }
       
       if result.Field != expected {
           t.Errorf("Expected %v, got %v", expected, result.Field)
       }
   }
   ```

2. **Table-Driven Tests**
   ```go
   func TestValidator_ValidateTemplate(t *testing.T) {
       tests := []struct {
           name     string
           template *Template
           expected bool
           wantErr  bool
       }{
           {
               name: "valid template",
               template: &Template{
                   Name: "test",
                   Description: "Test template",
               },
               expected: true,
               wantErr: false,
           },
           // More test cases...
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Test implementation
           })
       }
   }
   ```

3. **Mock Usage**
   ```go
   //go:generate mockgen -source=interfaces.go -destination=mocks/mock_interfaces.go
   
   func TestWithMock(t *testing.T) {
       ctrl := gomock.NewController(t)
       defer ctrl.Finish()
       
       mockProvider := mocks.NewMockAIProvider(ctrl)
       mockProvider.EXPECT().
           ProcessPrompt(gomock.Any(), gomock.Any()).
           Return(&AIResponse{Content: "test"}, nil)
       
       // Test with mock
   }
   ```

This development guide provides a comprehensive foundation for extending and contributing to the vibes-mcp-cli prompt system. For specific API details, refer to the [API Reference](Prompt-API.md), and for architectural insights, see the [Architecture Overview](Prompt-Architecture.md).