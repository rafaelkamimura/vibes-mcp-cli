# Prompt System API Reference

This document provides comprehensive API reference documentation for the vibes-mcp-cli prompt system, including Go package APIs, MCP protocol endpoints, and error handling patterns.

## Table of Contents

- [Go Package API](#go-package-api)
- [MCP Protocol Endpoints](#mcp-protocol-endpoints)
- [Configuration API](#configuration-api)
- [Event System](#event-system)
- [Error Handling](#error-handling)
- [Data Types](#data-types)

## Go Package API

### Manager Interface

The core interface for all prompt system operations.

```go
package prompt

type Manager interface {
    // Template operations
    ListTemplates(category string) ([]Template, error)
    GetTemplate(name string) (Template, error)
    CreateTemplateInteractive(name string) error
    CreateTemplateFromFile(name, filePath string) error
    UpdateTemplate(name string, interactive, validate bool) error
    DeleteTemplate(name string) error

    // Generation operations
    GeneratePrompt(config *GenerationConfig) (*GenerationResult, error)
    DetectWorkspaceContext() (*WorkspaceContext, error)
    SuggestTemplates(context *WorkspaceContext) []TemplateSuggestion

    // Validation operations
    ValidateTemplate(name string) (bool, []string)
    ValidateAllTemplates() (*ValidationReport, error)

    // Configuration operations
    GetConfig() *Config
    SetConfig(key, value string) error
    GetConfigValue(key string) string

    // History operations
    GetHistory(limit int, filter string) ([]HistoryEntry, error)
    RecordGeneration(entry *HistoryEntry) error

    // Integration operations
    CopyToClipboard(content string) error
    SaveToFile(content, filePath string) error
    UseContext7(result *GenerationResult) error
    TriggerBeastmode(result *GenerationResult) error
}
```

#### NewManager

Creates a new prompt manager instance.

```go
func NewManager(cfg *config.Config, logger *zap.Logger) (Manager, error)
```

**Parameters:**
- `cfg`: Application configuration
- `logger`: Structured logger instance

**Returns:**
- `Manager`: Prompt manager instance
- `error`: Initialization error if any

**Example:**
```go
manager, err := prompt.NewManager(cfg, logger)
if err != nil {
    log.Fatal("Failed to initialize prompt manager:", err)
}
```

#### ListTemplates

Lists available templates, optionally filtered by category.

```go
func (m *ManagerImpl) ListTemplates(category string) ([]Template, error)
```

**Parameters:**
- `category`: Template category filter (empty string for all)

**Returns:**
- `[]Template`: List of matching templates
- `error`: Operation error if any

**Categories:**
- `general`: Repository-agnostic development tasks
- `languages`: Technology-specific optimizations
- `workflows`: Multi-step orchestration patterns
- `workspace`: Vibes workspace specific optimizations

**Example:**
```go
// List all templates
templates, err := manager.ListTemplates("")

// List templates by category
goTemplates, err := manager.ListTemplates("languages")
```

#### GetTemplate

Retrieves a specific template by name.

```go
func (m *ManagerImpl) GetTemplate(name string) (Template, error)
```

**Parameters:**
- `name`: Template name (e.g., "feature-development")

**Returns:**
- `Template`: Template object
- `error`: Error if template not found

**Example:**
```go
template, err := manager.GetTemplate("feature-development")
if err != nil {
    log.Printf("Template not found: %v", err)
    return
}
fmt.Printf("Template: %s - %s\n", template.Name, template.Description)
```

#### GeneratePrompt

Generates a prompt from a template with given configuration.

```go
func (m *ManagerImpl) GeneratePrompt(config *GenerationConfig) (*GenerationResult, error)
```

**Parameters:**
- `config`: Generation configuration

**Returns:**
- `*GenerationResult`: Generated prompt with metadata
- `error`: Generation error if any

**Example:**
```go
config := &prompt.GenerationConfig{
    TemplateName: "feature-development",
    Interactive:  false,
    Parameters: map[string]string{
        "repo":      "myproject",
        "language":  "go",
        "component": "authentication",
    },
    Validate: true,
}

result, err := manager.GeneratePrompt(config)
if err != nil {
    log.Printf("Generation failed: %v", err)
    return
}

fmt.Println("Generated prompt:")
fmt.Println(result.Content)
```

#### DetectWorkspaceContext

Detects the current workspace context for intelligent template suggestions.

```go
func (m *ManagerImpl) DetectWorkspaceContext() (*WorkspaceContext, error)
```

**Returns:**
- `*WorkspaceContext`: Detected workspace information
- `error`: Detection error if any

**Example:**
```go
context, err := manager.DetectWorkspaceContext()
if err != nil {
    log.Printf("Context detection failed: %v", err)
    return
}

fmt.Printf("Repository: %s\n", context.Repository)
fmt.Printf("Language: %s\n", context.Language)
fmt.Printf("Framework: %s\n", context.Framework)
```

### Template Parser Interface

Handles template file operations and parsing.

```go
type TemplateParser interface {
    LoadTemplate(filePath string) (*Template, error)
    SaveTemplate(template *Template, filePath string) error
    ParseContent(content, templateName string) (*Template, error)
    ValidateStructure(template *Template) error
    ListTemplateFiles(directory string) ([]string, error)
}
```

#### NewTemplateParser

Creates a new template parser instance.

```go
func NewTemplateParser(logger *zap.Logger) TemplateParser
```

**Example:**
```go
parser := prompt.NewTemplateParser(logger)
template, err := parser.LoadTemplate("./templates/feature-development.yaml")
```

### Workspace Detector Interface

Analyzes project context and structure.

```go
type WorkspaceDetector interface {
    DetectContext(ctx context.Context) (*WorkspaceContext, error)
    DetectLanguage(directory string) (string, error)
    DetectFramework(directory, language string) (string, error)
    GetRecentFiles(directory string, limit int) ([]string, error)
    GetGitStatus(directory string) (string, string, error)
    GetDependencies(directory, language string) ([]Dependency, error)
    GetProjectStructure(directory string) ([]string, error)
}
```

#### NewWorkspaceDetector

Creates a new workspace detector instance.

```go
func NewWorkspaceDetector(logger *zap.Logger) WorkspaceDetector
```

### Generator Interface

Handles prompt generation and processing.

```go
type Generator interface {
    Generate(ctx context.Context, config *GenerationConfig) (*GenerationResult, error)
    FillTemplate(template *Template, parameters map[string]string) (string, error)
    ProcessInteractive(template *Template, context *WorkspaceContext) (map[string]string, error)
    FormatOutput(content, format string) (string, error)
    CalculateStats(content string) (int, int) // words, chars
}
```

#### NewGenerator

Creates a new prompt generator instance.

```go
func NewGenerator(logger *zap.Logger) Generator
```

### Validator Interface

Provides template validation capabilities.

```go
type Validator interface {
    ValidateTemplate(template *Template) (bool, []string, error)
    ValidateParameters(template *Template, parameters map[string]string) (bool, []string)
    ValidateContent(content string) (int, []string, []string) // score, issues, warnings
    ValidateStructure(template *Template) []string
    GetQualityScore(template *Template) int
}
```

#### NewValidator

Creates a new template validator instance.

```go
func NewValidator(logger *zap.Logger) Validator
```

### Integrator Interface

Manages AI tool integrations.

```go
type Integrator interface {
    CopyToClipboard(content string) error
    SaveToFile(content, filePath string) error
    SendToClaude(ctx context.Context, content string) error
    UseContext7(ctx context.Context, result *GenerationResult) error
    TriggerBeastmode(ctx context.Context, result *GenerationResult) error
    TestIntegration(tool string) error
}
```

#### NewIntegrator

Creates a new AI integrator instance.

```go
func NewIntegrator(cfg *config.Config, logger *zap.Logger) Integrator
```

## MCP Protocol Endpoints

The MCP (Model Context Protocol) server provides RESTful endpoints for remote access to the prompt system.

### Base URL

```
http://localhost:8080/v1/prompts
```

### Authentication

All endpoints require authentication via API key:

```http
Authorization: Bearer <api-key>
```

### Content Type

All requests and responses use JSON:

```http
Content-Type: application/json
```

### Endpoints

#### GET /templates

Lists available templates.

**Query Parameters:**
- `category` (optional): Filter by category
- `language` (optional): Filter by programming language
- `validate` (optional): Include validation status

**Response:**
```json
{
  "templates": [
    {
      "name": "feature-development",
      "category": "general",
      "language": "go",
      "framework": "cobra",
      "description": "Generate comprehensive feature development prompts",
      "parameters": [
        {
          "name": "repo",
          "description": "Repository name",
          "type": "string",
          "required": true
        }
      ],
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 1,
  "category": "general"
}
```

**Example:**
```bash
curl -H "Authorization: Bearer <api-key>" \
     "http://localhost:8080/v1/prompts/templates?category=general"
```

#### GET /templates/{name}

Retrieves a specific template.

**Path Parameters:**
- `name`: Template name

**Response:**
```json
{
  "template": {
    "name": "feature-development",
    "category": "general",
    "description": "Generate comprehensive feature development prompts",
    "content": "# Feature Development Request\n\n## Context\nRepository: {{repo}}\n...",
    "parameters": [
      {
        "name": "repo",
        "description": "Repository name",
        "type": "string",
        "required": true,
        "placeholder": "project-name"
      }
    ],
    "examples": [
      "vibes-mcp-cli prompt generate feature-development --repo myproject"
    ],
    "validation_status": {
      "valid": true,
      "score": 95,
      "issues": [],
      "warnings": []
    }
  }
}
```

#### POST /generate

Generates a prompt from a template.

**Request Body:**
```json
{
  "template_name": "feature-development",
  "parameters": {
    "repo": "myproject",
    "language": "go",
    "component": "authentication"
  },
  "interactive": false,
  "validate": true,
  "output_format": "markdown",
  "context": {
    "auto_detect": true
  }
}
```

**Response:**
```json
{
  "result": {
    "content": "# Feature Development Request\n\n## Context\nRepository: myproject\n...",
    "template": {
      "name": "feature-development",
      "category": "general"
    },
    "parameters": {
      "repo": "myproject",
      "language": "go",
      "component": "authentication"
    },
    "generated_at": "2024-01-15T14:30:00Z",
    "validation_status": {
      "valid": true,
      "score": 92
    },
    "word_count": 247,
    "char_count": 1543
  }
}
```

#### POST /validate

Validates one or more templates.

**Request Body:**
```json
{
  "template_name": "feature-development",
  "validate_all": false
}
```

**Response:**
```json
{
  "validation_report": {
    "template_name": "feature-development",
    "valid": true,
    "score": 95,
    "issues": [],
    "warnings": [
      "Consider adding more usage examples"
    ],
    "validated_at": "2024-01-15T14:30:00Z"
  }
}
```

#### GET /history

Retrieves prompt generation history.

**Query Parameters:**
- `limit` (optional): Maximum number of entries (default: 20)
- `filter` (optional): Filter by template category
- `since` (optional): ISO 8601 timestamp for filtering

**Response:**
```json
{
  "history": [
    {
      "id": "1642248600000000000",
      "template": "feature-development",
      "repository": "myproject",
      "language": "go",
      "parameters": {
        "repo": "myproject",
        "component": "authentication"
      },
      "output_method": "clipboard",
      "ai_tool": "claude",
      "success": true,
      "timestamp": "2024-01-15T14:30:00Z",
      "duration": "2.5s",
      "word_count": 247
    }
  ],
  "total": 1,
  "limit": 20
}
```

#### GET /context

Retrieves current workspace context.

**Response:**
```json
{
  "context": {
    "working_directory": "/Users/dev/myproject",
    "repository": "myproject",
    "language": "go",
    "framework": "cobra",
    "available_languages": ["go", "yaml", "markdown"],
    "recent_files": [
      "cmd/prompt.go",
      "internal/prompt/manager.go",
      "README.md"
    ],
    "git_branch": "feature/prompt-system",
    "git_status": "clean",
    "dependencies": [
      {
        "name": "cobra",
        "version": "1.7.0",
        "type": "prod",
        "manager": "go"
      }
    ],
    "last_modified": "2024-01-15T14:25:00Z"
  },
  "suggestions": [
    {
      "name": "feature-development",
      "reason": "matches go language, specific to myproject repository",
      "relevance": 0.9,
      "category": "general"
    }
  ]
}
```

#### POST /templates

Creates a new template.

**Request Body:**
```json
{
  "name": "custom-template",
  "category": "custom",
  "description": "My custom template",
  "content": "# Custom Template\n\nRepository: {{repo}}\nTask: {{task}}",
  "parameters": [
    {
      "name": "repo",
      "description": "Repository name",
      "type": "string",
      "required": true
    },
    {
      "name": "task",
      "description": "Task description",
      "type": "string",
      "required": true
    }
  ]
}
```

**Response:**
```json
{
  "template": {
    "name": "custom-template",
    "category": "custom",
    "description": "My custom template",
    "created_at": "2024-01-15T14:30:00Z",
    "file_path": "~/.vibes-mcp-cli/custom-templates/custom-template.yaml"
  }
}
```

#### PUT /templates/{name}

Updates an existing template.

**Path Parameters:**
- `name`: Template name

**Request Body:** Same as POST /templates

#### DELETE /templates/{name}

Deletes a custom template.

**Path Parameters:**
- `name`: Template name

**Response:**
```json
{
  "message": "Template 'custom-template' deleted successfully"
}
```

### WebSocket Endpoints

#### ws://localhost:8080/v1/prompts/events

Real-time event stream for prompt system events.

**Events:**
- `template_created`
- `template_updated`
- `template_deleted`
- `prompt_generated`
- `validation_completed`

**Example Event:**
```json
{
  "event": "prompt_generated",
  "timestamp": "2024-01-15T14:30:00Z",
  "data": {
    "template": "feature-development",
    "repository": "myproject",
    "word_count": 247,
    "success": true
  }
}
```

## Configuration API

### Configuration Structure

```go
type Config struct {
    DefaultRepository     string            `json:"default_repository"`
    PreferredLanguage     string            `json:"preferred_language"`
    PreferredFramework    string            `json:"preferred_framework,omitempty"`
    AutoClipboard         bool              `json:"auto_clipboard"`
    AutoValidate          bool              `json:"auto_validate"`
    PreferredAITool       string            `json:"preferred_ai_tool"`
    OutputFormat          string            `json:"output_format"`
    HistoryLimit          int               `json:"history_limit"`
    TemplateDirectories   []string          `json:"template_directories"`
    CustomTemplatesPath   string            `json:"custom_templates_path"`
    ValidationEnabled     bool              `json:"validation_enabled"`
    BackupEnabled         bool              `json:"backup_enabled"`
    IntegrationSettings   map[string]string `json:"integration_settings"`
    LastUpdated           time.Time         `json:"last_updated"`
}
```

### Configuration Methods

#### GetConfig

```go
func (m *ManagerImpl) GetConfig() *Config
```

Returns the current configuration.

#### SetConfig

```go
func (m *ManagerImpl) SetConfig(key, value string) error
```

Sets a configuration value.

**Supported Keys:**
- `default-repository`
- `preferred-language`
- `preferred-framework`
- `auto-clipboard`
- `auto-validate`
- `preferred-ai-tool`
- `output-format`

#### GetConfigValue

```go
func (m *ManagerImpl) GetConfigValue(key string) string
```

Retrieves a specific configuration value.

## Event System

### Event Types

```go
const (
    EventTemplateLoaded     = "template_loaded"
    EventTemplateCreated    = "template_created"
    EventTemplateUpdated    = "template_updated"
    EventTemplateDeleted    = "template_deleted"
    EventPromptGenerated    = "prompt_generated"
    EventValidationComplete = "validation_completed"
    EventContextDetected    = "context_detected"
    EventAIResponse         = "ai_response"
)
```

### Event Structure

```go
type Event struct {
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

### Event Handler Interface

```go
type EventHandler interface {
    HandleEvent(ctx context.Context, event *Event) error
    SupportedEvents() []string
}
```

### Event Publisher

```go
type EventPublisher interface {
    Publish(ctx context.Context, event *Event) error
    Subscribe(handler EventHandler) error
    Unsubscribe(handler EventHandler) error
}
```

## Error Handling

### Error Types

```go
type PromptError struct {
    Type    string
    Message string
    Cause   error
}

func (e *PromptError) Error() string {
    if e.Cause != nil {
        return e.Message + ": " + e.Cause.Error()
    }
    return e.Message
}

func (e *PromptError) Unwrap() error {
    return e.Cause
}
```

### Error Type Constants

```go
const (
    ErrorTypeTemplate    = "template"
    ErrorTypeValidation  = "validation"
    ErrorTypeGeneration  = "generation"
    ErrorTypeWorkspace   = "workspace"
    ErrorTypeIntegration = "integration"
    ErrorTypeHistory     = "history"
    ErrorTypeConfig      = "config"
)
```

### Common Errors

#### Template Errors

```go
// Template not found
&PromptError{
    Type:    ErrorTypeTemplate,
    Message: "template 'feature-development' not found",
}

// Invalid template structure
&PromptError{
    Type:    ErrorTypeTemplate,
    Message: "invalid template structure: missing required field 'name'",
}
```

#### Validation Errors

```go
// Parameter validation failed
&PromptError{
    Type:    ErrorTypeValidation,
    Message: "required parameter 'repo' not provided",
}

// Content validation failed
&PromptError{
    Type:    ErrorTypeValidation,
    Message: "template content validation failed",
}
```

#### Generation Errors

```go
// Generation timeout
&PromptError{
    Type:    ErrorTypeGeneration,
    Message: "prompt generation timed out after 30s",
}

// Parameter processing error
&PromptError{
    Type:    ErrorTypeGeneration,
    Message: "failed to process template parameters",
    Cause:   originalError,
}
```

#### Integration Errors

```go
// AI service error
&PromptError{
    Type:    ErrorTypeIntegration,
    Message: "Claude API returned status 429",
}

// Clipboard error
&PromptError{
    Type:    ErrorTypeIntegration,
    Message: "failed to copy to clipboard: no clipboard tool found",
}
```

### Error Handling Patterns

#### Manager Methods

```go
// Check error type
result, err := manager.GeneratePrompt(config)
if err != nil {
    var promptErr *PromptError
    if errors.As(err, &promptErr) {
        switch promptErr.Type {
        case ErrorTypeTemplate:
            // Handle template errors
        case ErrorTypeValidation:
            // Handle validation errors
        default:
            // Handle other errors
        }
    }
}
```

#### MCP Endpoints

HTTP status codes for different error types:

- `400 Bad Request`: Validation errors, invalid parameters
- `404 Not Found`: Template not found
- `422 Unprocessable Entity`: Generation errors
- `500 Internal Server Error`: System errors
- `502 Bad Gateway`: Integration errors
- `503 Service Unavailable`: Service overload

**Error Response Format:**
```json
{
  "error": {
    "type": "template",
    "message": "template 'invalid-template' not found",
    "code": "TEMPLATE_NOT_FOUND",
    "details": {
      "template_name": "invalid-template",
      "searched_directories": [
        "~/.vibes-mcp-cli/custom-templates",
        "./prompt-templates"
      ]
    }
  }
}
```

## Data Types

### Core Types

#### Template

```go
type Template struct {
    Name        string              `json:"name" yaml:"name"`
    Category    string              `json:"category" yaml:"category"`
    Language    string              `json:"language,omitempty" yaml:"language,omitempty"`
    Framework   string              `json:"framework,omitempty" yaml:"framework,omitempty"`
    Description string              `json:"description" yaml:"description"`
    Content     string              `json:"content" yaml:"content"`
    Parameters  []TemplateParameter `json:"parameters" yaml:"parameters"`
    Examples    []string            `json:"examples,omitempty" yaml:"examples,omitempty"`
    Tags        []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
    Author      string              `json:"author,omitempty" yaml:"author,omitempty"`
    Version     string              `json:"version,omitempty" yaml:"version,omitempty"`
    CreatedAt   time.Time           `json:"created_at" yaml:"created_at"`
    UpdatedAt   time.Time           `json:"updated_at" yaml:"updated_at"`
    FilePath    string              `json:"file_path,omitempty" yaml:"file_path,omitempty"`
}
```

#### TemplateParameter

```go
type TemplateParameter struct {
    Name         string   `json:"name" yaml:"name"`
    Description  string   `json:"description" yaml:"description"`
    Type         string   `json:"type" yaml:"type"` // string, int, bool, select, text
    Required     bool     `json:"required" yaml:"required"`
    Default      string   `json:"default,omitempty" yaml:"default,omitempty"`
    Options      []string `json:"options,omitempty" yaml:"options,omitempty"`
    Placeholder  string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
    Validation   string   `json:"validation,omitempty" yaml:"validation,omitempty"`
}
```

#### GenerationConfig

```go
type GenerationConfig struct {
    TemplateName string                 `json:"template_name"`
    Interactive  bool                   `json:"interactive"`
    Context      *WorkspaceContext      `json:"context,omitempty"`
    Parameters   map[string]string      `json:"parameters"`
    Validate     bool                   `json:"validate"`
    OutputFormat string                 `json:"output_format,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
```

#### GenerationResult

```go
type GenerationResult struct {
    Content          string                 `json:"content"`
    Template         Template               `json:"template"`
    Parameters       map[string]string      `json:"parameters"`
    GeneratedAt      time.Time              `json:"generated_at"`
    Context          *WorkspaceContext      `json:"context,omitempty"`
    ValidationStatus ValidationStatus       `json:"validation_status"`
    Metadata         map[string]interface{} `json:"metadata,omitempty"`
    WordCount        int                    `json:"word_count"`
    CharCount        int                    `json:"char_count"`
}
```

#### WorkspaceContext

```go
type WorkspaceContext struct {
    WorkingDirectory    string            `json:"working_directory"`
    Repository          string            `json:"repository"`
    Language            string            `json:"language"`
    Framework           string            `json:"framework,omitempty"`
    AvailableLanguages  []string          `json:"available_languages"`
    RecentFiles         []string          `json:"recent_files"`
    GitBranch           string            `json:"git_branch,omitempty"`
    GitStatus           string            `json:"git_status,omitempty"`
    Dependencies        []Dependency      `json:"dependencies,omitempty"`
    ProjectStructure    []string          `json:"project_structure,omitempty"`
    Environment         map[string]string `json:"environment,omitempty"`
    LastModified        time.Time         `json:"last_modified"`
}
```

#### ValidationStatus

```go
type ValidationStatus struct {
    Valid    bool     `json:"valid"`
    Score    int      `json:"score"` // 0-100
    Issues   []string `json:"issues,omitempty"`
    Warnings []string `json:"warnings,omitempty"`
}
```

#### HistoryEntry

```go
type HistoryEntry struct {
    ID           string            `json:"id"`
    Template     string            `json:"template"`
    Repository   string            `json:"repository"`
    Language     string            `json:"language"`
    Framework    string            `json:"framework,omitempty"`
    Parameters   map[string]string `json:"parameters"`
    OutputMethod string            `json:"output_method"`
    AITool       string            `json:"ai_tool,omitempty"`
    Success      bool              `json:"success"`
    ErrorMessage string            `json:"error_message,omitempty"`
    Timestamp    time.Time         `json:"timestamp"`
    Duration     time.Duration     `json:"duration"`
    WordCount    int               `json:"word_count"`
}
```

### Constants

#### Template Categories

```go
const (
    CategoryGeneral   = "general"
    CategoryLanguages = "languages"
    CategoryWorkflows = "workflows"
    CategoryWorkspace = "workspace"
    CategoryCustom    = "custom"
)
```

#### Output Formats

```go
const (
    FormatMarkdown = "markdown"
    FormatText     = "text"
    FormatJSON     = "json"
    FormatHTML     = "html"
)
```

#### AI Tool Identifiers

```go
const (
    AIToolClaude    = "claude"
    AIToolContext7  = "context7"
    AIToolBeastmode = "beastmode"
)
```

### Type Validation

#### Parameter Types

- `string`: Single-line text input
- `text`: Multi-line text input  
- `int`: Integer number
- `bool`: Boolean true/false
- `select`: Dropdown with predefined options
- `file`: File path input

#### Validation Rules

```go
func ValidateParameterType(paramType, value string) error {
    switch paramType {
    case "string":
        return validateString(value)
    case "text":
        return validateText(value)
    case "int":
        return validateInt(value)
    case "bool":
        return validateBool(value)
    case "select":
        return validateSelect(value, options)
    case "file":
        return validateFile(value)
    default:
        return fmt.Errorf("unknown parameter type: %s", paramType)
    }
}
```

For implementation examples and usage patterns, see the [Development Guide](Prompt-Development.md).