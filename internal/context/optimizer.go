package context

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openai-cli/internal/client"
	"openai-cli/internal/mcp"
)



// ContextOptimizer manages context optimization using subagents
type ContextOptimizer struct {
	mcpClient    *mcp.Client
	strategy     mcp.OptimizationStrategy
	tokenCounter func(string) int
	subagents    map[string]SubagentConfig
}

// SubagentConfig defines configuration for specialized subagents
type SubagentConfig struct {
	Name        string   `json:"name"`
	TaskTypes   []string `json:"task_types"`
	Endpoint    string   `json:"endpoint"`
	MaxTokens   int      `json:"max_tokens"`
	Specializes []string `json:"specializes"` // code, docs, data, etc.
}

// NewContextOptimizer creates a new context optimizer
func NewContextOptimizer(mcpClient *mcp.Client, strategy mcp.OptimizationStrategy) *ContextOptimizer {
	return &ContextOptimizer{
		mcpClient:    mcpClient,
		strategy:     strategy,
		tokenCounter: estimateTokenCount,
		subagents: map[string]SubagentConfig{
			"code_analyzer": {
				Name:        "Code Analyzer",
				TaskTypes:   []string{"code_review", "code_summary", "bug_analysis"},
				Specializes: []string{"go", "python", "javascript", "typescript"},
				MaxTokens:   2000,
			},
			"doc_summarizer": {
				Name:        "Document Summarizer",
				TaskTypes:   []string{"documentation", "readme", "markdown"},
				Specializes: []string{"markdown", "text", "documentation"},
				MaxTokens:   1500,
			},
			"data_processor": {
				Name:        "Data Processor",
				TaskTypes:   []string{"json_analysis", "csv_processing", "log_analysis"},
				Specializes: []string{"json", "csv", "logs", "structured_data"},
				MaxTokens:   1000,
			},
		},
	}
}

// OptimizeContext optimizes conversation context using subagents
func (co *ContextOptimizer) OptimizeContext(ctx context.Context, messages []client.ChatMessage, additionalContext []mcp.ContextChunk) ([]client.ChatMessage, *mcp.OptimizationResult, error) {
	startTime := time.Now()
	
	// Convert client messages to MCP format
	mcpMessages := make([]mcp.ChatMessage, len(messages))
	for i, msg := range messages {
		mcpMessages[i] = mcp.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	
	// Calculate original token usage
	originalTokens := co.calculateMessagesTokens(messages)
	for _, chunk := range additionalContext {
		originalTokens += chunk.TokenCount
	}
	
	// If under token limit, return as-is
	if originalTokens <= co.strategy.MaxTokens {
		result := &mcp.OptimizationResult{
			OriginalTokens:    originalTokens,
			OptimizedTokens:   originalTokens,
			TokensSaved:       0,
			CompressionRatio:  1.0,
			ProcessingTime:    time.Since(startTime),
			Strategy:          co.strategy,
			OptimizedMessages: mcpMessages,
			OptimizedContext:  additionalContext,
			Success:           true,
		}
		return messages, result, nil
	}
	
	// Use MCP client for optimization
	params := mcp.ContextOptimizeParams{
		Messages:  mcpMessages,
		Context:   additionalContext,
		Strategy:  co.strategy,
		MaxTokens: co.strategy.MaxTokens,
	}
	
	traceID := fmt.Sprintf("context_opt_%d", time.Now().UnixNano())
	result, err := co.mcpClient.OptimizeContext(ctx, params, traceID)
	if err != nil {
		return messages, nil, fmt.Errorf("MCP context optimization failed: %w", err)
	}
	
	// Convert optimized messages back to client format
	optimizedMessages := make([]client.ChatMessage, len(result.OptimizedMessages))
	for i, msg := range result.OptimizedMessages {
		optimizedMessages[i] = client.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	
	return optimizedMessages, result, nil
}

// processChunksWithSubagents routes chunks to appropriate subagents for processing
func (co *ContextOptimizer) processChunksWithSubagents(ctx context.Context, chunks []mcp.ContextChunk) ([]mcp.ContextChunk, error) {
	optimizedChunks := make([]mcp.ContextChunk, 0, len(chunks))
	
	for _, chunk := range chunks {
		subagent := co.selectSubagentForChunk(chunk)
		
		if subagent == "" {
			// No specialized subagent, apply basic optimization
			optimizedChunk := co.applyBasicOptimization(chunk)
			optimizedChunks = append(optimizedChunks, optimizedChunk)
			continue
		}

		// Process with specialized subagent
		params := mcp.SubagentCallParams{
			SubagentID: subagent,
			TaskType:   co.getTaskTypeForChunk(chunk),
			Input:      chunk.Content,
			Context:    []mcp.ContextChunk{chunk},
			Parameters: map[string]interface{}{
				"max_summary_length": co.strategy.SummaryThreshold,
				"preserve_structure": chunk.Type == "code",
			},
			MaxTokens: co.subagents[subagent].MaxTokens,
		}

		traceID := fmt.Sprintf("%s_%s_%d", subagent, chunk.ID, time.Now().UnixNano())
		response, err := co.mcpClient.CallSubagent(ctx, params, traceID)
		if err != nil {
			// Fallback to basic optimization if subagent fails
			optimizedChunk := co.applyBasicOptimization(chunk)
			optimizedChunks = append(optimizedChunks, optimizedChunk)
			continue
		}

		// Create optimized chunk from subagent response
		optimizedChunk := mcp.ContextChunk{
			ID:         chunk.ID + "_optimized",
			Type:       chunk.Type,
			Content:    response.Summary,
			Summary:    strings.Join(response.KeyPoints, "; "),
			TokenCount: co.tokenCounter(response.Summary),
			Priority:   chunk.Priority,
			Timestamp:  time.Now(),
			Source:     chunk.Source + "_subagent_" + subagent,
			Tags:       append(chunk.Tags, "subagent_processed"),
			CompressedSize: chunk.TokenCount - response.TokensSaved,
		}

		optimizedChunks = append(optimizedChunks, optimizedChunk)
	}

	return optimizedChunks, nil
}

// selectSubagentForChunk determines the best subagent for processing a chunk
func (co *ContextOptimizer) selectSubagentForChunk(chunk mcp.ContextChunk) string {
	// Simple heuristic-based selection
	switch chunk.Type {
	case "code":
		for _, tag := range chunk.Tags {
			if contains(co.subagents["code_analyzer"].Specializes, tag) {
				return "code_analyzer"
			}
		}
		return "code_analyzer"
	case "text":
		if contains(chunk.Tags, "documentation") || contains(chunk.Tags, "markdown") {
			return "doc_summarizer"
		}
	case "data":
		return "data_processor"
	}
	
	return "" // No specialized subagent
}


// applyOptimizationStrategy applies final optimization based on strategy
func (co *ContextOptimizer) applyOptimizationStrategy(chunks []mcp.ContextChunk) []mcp.ContextChunk {
	if len(chunks) == 0 {
		return chunks
	}

	// Sort by priority and recency
	if co.strategy.PriorityWeighting {
		chunks = co.sortByPriorityAndRecency(chunks)
	}

	// Retain most recent messages as configured
	recentCount := co.strategy.RetainRecent
	if recentCount > len(chunks) {
		recentCount = len(chunks)
	}

	// Calculate tokens and apply limits
	totalTokens := 0
	result := make([]mcp.ContextChunk, 0, len(chunks))
	
	// Always include recent chunks first
	for i := len(chunks) - recentCount; i < len(chunks); i++ {
		chunk := chunks[i]
		if totalTokens + chunk.TokenCount <= co.strategy.MaxTokens {
			result = append(result, chunk)
			totalTokens += chunk.TokenCount
		}
	}

	// Add older chunks based on priority if space allows
	for i := len(chunks) - recentCount - 1; i >= 0; i-- {
		chunk := chunks[i]
		if totalTokens + chunk.TokenCount <= co.strategy.MaxTokens {
			result = append([]mcp.ContextChunk{chunk}, result...)
			totalTokens += chunk.TokenCount
		}
	}

	return result
}


// Helper functions
func (co *ContextOptimizer) calculateMessagesTokens(messages []client.ChatMessage) int {
	total := 0
	for _, msg := range messages {
		total += co.tokenCounter(msg.Content)
	}
	return total
}



func (co *ContextOptimizer) getTaskTypeForChunk(chunk mcp.ContextChunk) string {
	switch chunk.Type {
	case "code":
		return "code_summary"
	case "text":
		if contains(chunk.Tags, "documentation") {
			return "documentation"
		}
		return "text_summary"
	case "data":
		return "data_analysis"
	default:
		return "general_summary"
	}
}

func (co *ContextOptimizer) applyBasicOptimization(chunk mcp.ContextChunk) mcp.ContextChunk {
	// Simple truncation optimization
	if chunk.TokenCount > co.strategy.SummaryThreshold {
		content := chunk.Content
		if len(content) > co.strategy.SummaryThreshold*4 { // Rough estimate: 4 chars per token
			content = content[:co.strategy.SummaryThreshold*4] + "..."
		}
		
		return mcp.ContextChunk{
			ID:         chunk.ID + "_truncated",
			Type:       chunk.Type,
			Content:    content,
			Summary:    "Truncated for length",
			TokenCount: co.tokenCounter(content),
			Priority:   chunk.Priority,
			Timestamp:  time.Now(),
			Source:     chunk.Source + "_truncated",
			Tags:       append(chunk.Tags, "truncated"),
		}
	}
	return chunk
}

func (co *ContextOptimizer) sortByPriorityAndRecency(chunks []mcp.ContextChunk) []mcp.ContextChunk {
	// Sort by weighted score: priority * 0.7 + recency * 0.3
	// Implementation would use sort.Slice with custom comparison
	return chunks // Simplified for brevity
}

// RouteTaskToSubagent routes a task to the most appropriate subagent
func (co *ContextOptimizer) RouteTaskToSubagent(ctx context.Context, taskType, input string, context []mcp.ContextChunk) (*mcp.SubagentResponse, error) {
	traceID := fmt.Sprintf("route_%s_%d", taskType, time.Now().UnixNano())
	return co.mcpClient.RouteTask(ctx, taskType, input, context, traceID)
}

// GetAvailableSubagents retrieves the list of available subagents
func (co *ContextOptimizer) GetAvailableSubagents(ctx context.Context) (*mcp.SubagentListResult, error) {
	traceID := fmt.Sprintf("list_subagents_%d", time.Now().UnixNano())
	return co.mcpClient.ListSubagents(ctx, traceID)
}

// GetOptimizationMetrics retrieves optimization performance metrics
func (co *ContextOptimizer) GetOptimizationMetrics(ctx context.Context) (map[string]interface{}, error) {
	traceID := fmt.Sprintf("metrics_%d", time.Now().UnixNano())
	return co.mcpClient.GetOptimizationMetrics(ctx, traceID)
}

// Utility functions
func estimateTokenCount(text string) int {
	// Rough estimation: ~4 characters per token for English text
	return len(text) / 4
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

