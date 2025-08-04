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