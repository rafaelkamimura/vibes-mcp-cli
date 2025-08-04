package prompt

import (
	"context"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/config"
)

// Manager defines the core prompt management interface
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

// Template represents a prompt template
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

// TemplateParameter defines a parameter within a template
type TemplateParameter struct {
	Name         string   `json:"name" yaml:"name"`
	Description  string   `json:"description" yaml:"description"`
	Type         string   `json:"type" yaml:"type"` // string, int, bool, select
	Required     bool     `json:"required" yaml:"required"`
	Default      string   `json:"default,omitempty" yaml:"default,omitempty"`
	Options      []string `json:"options,omitempty" yaml:"options,omitempty"` // for select type
	Placeholder  string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Validation   string   `json:"validation,omitempty" yaml:"validation,omitempty"` // regex pattern
}

// GenerationConfig defines configuration for prompt generation
type GenerationConfig struct {
	TemplateName string                 `json:"template_name"`
	Interactive  bool                   `json:"interactive"`
	Context      *WorkspaceContext      `json:"context,omitempty"`
	Parameters   map[string]string      `json:"parameters"`
	Validate     bool                   `json:"validate"`
	OutputFormat string                 `json:"output_format,omitempty"` // markdown, text, json
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// GenerationResult contains the result of prompt generation
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

// WorkspaceContext contains detected workspace information
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

// Dependency represents a project dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // dev, prod, peer
	Manager string `json:"manager"` // npm, go, pip, etc.
}

// TemplateSuggestion contains suggested templates based on context
type TemplateSuggestion struct {
	Name     string  `json:"name"`
	Reason   string  `json:"reason"`
	Relevance float64 `json:"relevance"` // 0.0 to 1.0
	Category string  `json:"category"`
}

// ValidationStatus represents validation results
type ValidationStatus struct {
	Valid    bool     `json:"valid"`
	Score    int      `json:"score"` // 0-100
	Issues   []string `json:"issues,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ValidationReport contains validation results for all templates
type ValidationReport struct {
	Total        int                    `json:"total"`
	Valid        int                    `json:"valid"`
	Invalid      int                    `json:"invalid"`
	AverageScore int                    `json:"average_score"`
	Issues       map[string][]string    `json:"issues"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

// HistoryEntry represents a prompt generation history entry
type HistoryEntry struct {
	ID           string            `json:"id"`
	Template     string            `json:"template"`
	Repository   string            `json:"repository"`
	Language     string            `json:"language"`
	Framework    string            `json:"framework,omitempty"`
	Parameters   map[string]string `json:"parameters"`
	OutputMethod string            `json:"output_method"` // clipboard, file, stdout
	AITool       string            `json:"ai_tool,omitempty"` // claude, context7, beastmode
	Success      bool              `json:"success"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Duration     time.Duration     `json:"duration"`
	WordCount    int               `json:"word_count"`
}

// Config represents prompt system configuration
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

// ManagerImpl is the concrete implementation of Manager
type ManagerImpl struct {
	cfg    *config.Config
	logger *zap.Logger
	config *Config

	// Service dependencies
	templateParser  TemplateParser
	workspaceDetector WorkspaceDetector
	generator       Generator
	validator       Validator
	historyTracker  HistoryTracker
	integrator      Integrator
}

// TemplateParser handles template parsing and loading
type TemplateParser interface {
	LoadTemplate(filePath string) (*Template, error)
	SaveTemplate(template *Template, filePath string) error
	ParseContent(content, templateName string) (*Template, error)
	ValidateStructure(template *Template) error
	ListTemplateFiles(directory string) ([]string, error)
}

// WorkspaceDetector handles workspace context detection
type WorkspaceDetector interface {
	DetectContext(ctx context.Context) (*WorkspaceContext, error)
	DetectLanguage(directory string) (string, error)
	DetectFramework(directory, language string) (string, error)
	GetRecentFiles(directory string, limit int) ([]string, error)
	GetGitStatus(directory string) (string, string, error) // branch, status
	GetDependencies(directory, language string) ([]Dependency, error)
	GetProjectStructure(directory string) ([]string, error)
}

// Generator handles prompt generation logic
type Generator interface {
	Generate(ctx context.Context, config *GenerationConfig) (*GenerationResult, error)
	FillTemplate(template *Template, parameters map[string]string) (string, error)
	ProcessInteractive(template *Template, context *WorkspaceContext) (map[string]string, error)
	FormatOutput(content, format string) (string, error)
	CalculateStats(content string) (int, int) // words, chars
}

// Validator handles template validation
type Validator interface {
	ValidateTemplate(template *Template) (bool, []string, error)
	ValidateParameters(template *Template, parameters map[string]string) (bool, []string)
	ValidateContent(content string) (int, []string, []string) // score, issues, warnings
	ValidateStructure(template *Template) []string
	GetQualityScore(template *Template) int
}

// HistoryTracker handles generation history
type HistoryTracker interface {
	Record(entry *HistoryEntry) error
	GetHistory(limit int, filter string) ([]HistoryEntry, error)
	GetStats() (*HistoryStats, error)
	Cleanup(olderThan time.Duration) error
}

// HistoryStats contains usage statistics
type HistoryStats struct {
	TotalGenerations int                    `json:"total_generations"`
	TopTemplates     []TemplateUsage        `json:"top_templates"`
	TopLanguages     []LanguageUsage        `json:"top_languages"`
	TopRepositories  []RepositoryUsage      `json:"top_repositories"`
	SuccessRate      float64                `json:"success_rate"`
	AverageWordCount int                    `json:"average_word_count"`
	PeriodStats      map[string]int         `json:"period_stats"` // daily, weekly, monthly
}

// TemplateUsage tracks template usage statistics
type TemplateUsage struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// LanguageUsage tracks language usage statistics
type LanguageUsage struct {
	Language string `json:"language"`
	Count    int    `json:"count"`
}

// RepositoryUsage tracks repository usage statistics
type RepositoryUsage struct {
	Repository string `json:"repository"`
	Count      int    `json:"count"`
}

// Integrator handles AI tool integrations
type Integrator interface {
	CopyToClipboard(content string) error
	SaveToFile(content, filePath string) error
	SendToClaude(ctx context.Context, content string) error
	UseContext7(ctx context.Context, result *GenerationResult) error
	TriggerBeastmode(ctx context.Context, result *GenerationResult) error
	TestIntegration(tool string) error
}

// Error types for the prompt system
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

// Error type constants
const (
	ErrorTypeTemplate   = "template"
	ErrorTypeValidation = "validation"
	ErrorTypeGeneration = "generation"
	ErrorTypeWorkspace  = "workspace"
	ErrorTypeIntegration = "integration"
	ErrorTypeHistory    = "history"
	ErrorTypeConfig     = "config"
)

// Template categories
const (
	CategoryGeneral   = "general"
	CategoryLanguages = "languages"
	CategoryWorkflows = "workflows"
	CategoryWorkspace = "workspace"
	CategoryCustom    = "custom"
)

// Output formats
const (
	FormatMarkdown = "markdown"
	FormatText     = "text"
	FormatJSON     = "json"
	FormatHTML     = "html"
)

// AI tool identifiers
const (
	AIToolClaude    = "claude"
	AIToolContext7  = "context7"
	AIToolBeastmode = "beastmode"
)