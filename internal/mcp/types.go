package mcp

import (
	"encoding/json"
	"time"
)

// RPCRequest represents a JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// RPCParams holds parameters for call_tool.
type RPCParams struct {
	Input string `json:"input"`
}

// SubagentCallParams holds parameters for subagent routing calls
type SubagentCallParams struct {
	SubagentID string                 `json:"subagent_id"`
	TaskType   string                 `json:"task_type"`
	Input      string                 `json:"input"`
	Context    []ContextChunk        `json:"context,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	MaxTokens  int                   `json:"max_tokens,omitempty"`
	Timeout    int                   `json:"timeout,omitempty"`
}

// ContextOptimizeParams holds parameters for context optimization calls
type ContextOptimizeParams struct {
	Messages   []ChatMessage        `json:"messages"`
	Context    []ContextChunk       `json:"context,omitempty"`
	Strategy   OptimizationStrategy `json:"strategy"`
	MaxTokens  int                  `json:"max_tokens"`
	TaskType   string               `json:"task_type,omitempty"`
}

// ChatMessage represents a chat message for optimization
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ContextChunk represents a piece of context with metadata
type ContextChunk struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Content        string            `json:"content"`
	Summary        string            `json:"summary,omitempty"`
	TokenCount     int               `json:"token_count"`
	Priority       int               `json:"priority"`
	Timestamp      time.Time         `json:"timestamp"`
	Source         string            `json:"source"`
	Tags           []string          `json:"tags,omitempty"`
	CompressedSize int               `json:"compressed_size,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// OptimizationStrategy defines how context should be optimized
type OptimizationStrategy struct {
	MaxTokens         int     `json:"max_tokens"`
	SummaryThreshold  int     `json:"summary_threshold"`
	CompressionRatio  float64 `json:"compression_ratio"`
	RetainRecent      int     `json:"retain_recent"`
	PriorityWeighting bool    `json:"priority_weighting"`
	UseSubagents      bool    `json:"use_subagents"`
}

// RPCResponse represents a JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadFileResult holds the result of a read_file JSON-RPC call.
type ReadFileResult struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	OffsetLines int    `json:"offset_lines"`
	LimitLines  *int   `json:"limit_lines"`
	TotalLines  int    `json:"total_lines"`
}

// SubagentResponse represents a response from a subagent call
type SubagentResponse struct {
	ID             string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	SubagentID     string                 `json:"subagent_id"`
	Summary        string                 `json:"summary"`
	KeyPoints      []string               `json:"key_points,omitempty"`
	Analysis       string                 `json:"analysis,omitempty"`
	TokensProcessed int                   `json:"tokens_processed"`
	TokensSaved    int                   `json:"tokens_saved"`
	Confidence     float64                `json:"confidence"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Success        bool                   `json:"success"`
	Error          string                 `json:"error,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// OptimizationResult contains the results of context optimization
type OptimizationResult struct {
	OriginalTokens    int                    `json:"original_tokens"`
	OptimizedTokens   int                    `json:"optimized_tokens"`
	TokensSaved       int                    `json:"tokens_saved"`
	CompressionRatio  float64                `json:"compression_ratio"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	Strategy          OptimizationStrategy   `json:"strategy"`
	SubagentUsage     map[string]int         `json:"subagent_usage,omitempty"`
	OptimizedMessages []ChatMessage          `json:"optimized_messages"`
	OptimizedContext  []ContextChunk         `json:"optimized_context,omitempty"`
	Success           bool                   `json:"success"`
	Error             string                 `json:"error,omitempty"`
}

// SubagentListResult represents the result of listing available subagents
type SubagentListResult struct {
	Subagents []SubagentInfo `json:"subagents"`
	Total     int            `json:"total"`
}

// SubagentInfo represents information about a subagent
type SubagentInfo struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Version      string                `json:"version"`
	Capabilities []SubagentCapability  `json:"capabilities"`
	Status       string                `json:"status"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SubagentCapability defines what a subagent can do
type SubagentCapability struct {
	TaskType       string        `json:"task_type"`
	Languages      []string      `json:"languages,omitempty"`
	MaxTokens      int           `json:"max_tokens"`
	MinTokens      int           `json:"min_tokens"`
	Confidence     float64       `json:"confidence"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// Prompt-specific MCP types for template operations and AI integration

// PromptResource represents a prompt template as an MCP resource
type PromptResource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	MimeType    string                 `json:"mimeType"`
	Text        string                 `json:"text,omitempty"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags,omitempty"`
	Parameters  []TemplateParameter    `json:"parameters,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateParameter represents a template parameter for MCP
type TemplateParameter struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Required     bool     `json:"required"`
	Default      string   `json:"default,omitempty"`
	Options      []string `json:"options,omitempty"`
	Placeholder  string   `json:"placeholder,omitempty"`
	Validation   string   `json:"validation,omitempty"`
}

// PromptTool represents a prompt operation as an MCP tool
type PromptTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema ToolInputSchema        `json:"inputSchema"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolInputSchema defines the input schema for prompt tools
type ToolInputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// PromptGenerateParams holds parameters for prompt generation via MCP
type PromptGenerateParams struct {
	TemplateName string                 `json:"template_name"`
	Parameters   map[string]string      `json:"parameters,omitempty"`
	Context      *WorkspaceContext      `json:"context,omitempty"`
	Interactive  bool                   `json:"interactive,omitempty"`
	Validate     bool                   `json:"validate,omitempty"`
	OutputFormat string                 `json:"output_format,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PromptGenerateResult contains the result of prompt generation via MCP
type PromptGenerateResult struct {
	Content          string                 `json:"content"`
	Template         string                 `json:"template"`
	Parameters       map[string]string      `json:"parameters"`
	GeneratedAt      time.Time              `json:"generated_at"`
	Context          *WorkspaceContext      `json:"context,omitempty"`
	ValidationStatus ValidationStatus       `json:"validation_status"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	WordCount        int                    `json:"word_count"`
	CharCount        int                    `json:"char_count"`
	Success          bool                   `json:"success"`
	Error            string                 `json:"error,omitempty"`
}

// WorkspaceContext represents workspace information for MCP
type WorkspaceContext struct {
	WorkingDirectory   string            `json:"working_directory"`
	Repository         string            `json:"repository"`
	Language           string            `json:"language"`
	Framework          string            `json:"framework,omitempty"`
	AvailableLanguages []string          `json:"available_languages"`
	RecentFiles        []string          `json:"recent_files"`
	GitBranch          string            `json:"git_branch,omitempty"`
	GitStatus          string            `json:"git_status,omitempty"`
	Dependencies       []Dependency      `json:"dependencies,omitempty"`
	ProjectStructure   []string          `json:"project_structure,omitempty"`
	Environment        map[string]string `json:"environment,omitempty"`
	LastModified       time.Time         `json:"last_modified"`
}

// Dependency represents a project dependency for MCP
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
	Manager string `json:"manager"`
}

// ValidationStatus represents validation results for MCP
type ValidationStatus struct {
	Valid    bool     `json:"valid"`
	Score    int      `json:"score"`
	Issues   []string `json:"issues,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// PromptValidateParams holds parameters for template validation via MCP
type PromptValidateParams struct {
	TemplateName string `json:"template_name,omitempty"`
	Content      string `json:"content,omitempty"`
}

// PromptValidateResult contains validation results via MCP
type PromptValidateResult struct {
	Valid    bool     `json:"valid"`
	Score    int      `json:"score"`
	Issues   []string `json:"issues,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Template string   `json:"template,omitempty"`
	Success  bool     `json:"success"`
	Error    string   `json:"error,omitempty"`
}

// PromptSuggestParams holds parameters for template suggestions via MCP
type PromptSuggestParams struct {
	Context     *WorkspaceContext `json:"context,omitempty"`
	TaskType    string            `json:"task_type,omitempty"`
	Language    string            `json:"language,omitempty"`
	Framework   string            `json:"framework,omitempty"`
	MaxResults  int               `json:"max_results,omitempty"`
}

// PromptSuggestResult contains template suggestions via MCP
type PromptSuggestResult struct {
	Suggestions []TemplateSuggestion `json:"suggestions"`
	Context     *WorkspaceContext    `json:"context,omitempty"`
	Success     bool                 `json:"success"`
	Error       string               `json:"error,omitempty"`
}

// TemplateSuggestion represents a suggested template for MCP
type TemplateSuggestion struct {
	Name      string  `json:"name"`
	Reason    string  `json:"reason"`
	Relevance float64 `json:"relevance"`
	Category  string  `json:"category"`
}

// AIAssistantParams holds parameters for AI assistant integration via MCP
type AIAssistantParams struct {
	Content     string            `json:"content"`
	Assistant   string            `json:"assistant"` // claude, gpt, etc.
	Context     *WorkspaceContext `json:"context,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	SystemPrompt string           `json:"system_prompt,omitempty"`
}

// AIAssistantResult contains AI assistant response via MCP
type AIAssistantResult struct {
	Response        string                 `json:"response"`
	Assistant       string                 `json:"assistant"`
	TokensUsed      int                    `json:"tokens_used"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Suggestions     []string               `json:"suggestions,omitempty"`
	Improvements    []string               `json:"improvements,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Success         bool                   `json:"success"`
	Error           string                 `json:"error,omitempty"`
}

// PromptHistoryParams holds parameters for history operations via MCP
type PromptHistoryParams struct {
	Limit       int    `json:"limit,omitempty"`
	Filter      string `json:"filter,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Language    string `json:"language,omitempty"`
	Template    string `json:"template,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
}

// PromptHistoryResult contains prompt generation history via MCP
type PromptHistoryResult struct {
	Entries []HistoryEntry `json:"entries"`
	Total   int            `json:"total"`
	Stats   *HistoryStats  `json:"stats,omitempty"`
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
}

// HistoryEntry represents a prompt generation history entry for MCP
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

// HistoryStats contains usage statistics for MCP
type HistoryStats struct {
	TotalGenerations int                    `json:"total_generations"`
	TopTemplates     []TemplateUsage        `json:"top_templates"`
	TopLanguages     []LanguageUsage        `json:"top_languages"`
	TopRepositories  []RepositoryUsage      `json:"top_repositories"`
	SuccessRate      float64                `json:"success_rate"`
	AverageWordCount int                    `json:"average_word_count"`
	PeriodStats      map[string]int         `json:"period_stats"`
}

// TemplateUsage tracks template usage statistics for MCP
type TemplateUsage struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// LanguageUsage tracks language usage statistics for MCP
type LanguageUsage struct {
	Language string `json:"language"`
	Count    int    `json:"count"`
}

// RepositoryUsage tracks repository usage statistics for MCP
type RepositoryUsage struct {
	Repository string `json:"repository"`
	Count      int    `json:"count"`
}