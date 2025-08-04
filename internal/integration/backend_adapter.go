package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	contextpkg "openai-cli/internal/context"
	"openai-cli/internal/mcp"
	"openai-cli/internal/metrics"
	"openai-cli/internal/subagent"
)

// BackendAdapter integrates with the vibes-agent-backend for subagent management
type BackendAdapter struct {
	baseURL      string
	authToken    string
	httpClient   *http.Client
	mcpClient    *mcp.Client
}

// SubagentRegistration represents a subagent registration request
type SubagentRegistration struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Capabilities []string               `json:"capabilities"`
	Endpoint     string                 `json:"endpoint"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// SubagentResponse represents a subagent execution response from the backend
type SubagentResponse struct {
	ID           string                 `json:"id"`
	SubagentID   string                 `json:"subagent_id"`
	Summary      string                 `json:"summary"`
	KeyPoints    []string               `json:"key_points"`
	Analysis     string                 `json:"analysis"`
	TokensSaved  int                    `json:"tokens_saved"`
	Confidence   float64                `json:"confidence"`
	ProcessingTime time.Duration        `json:"processing_time"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// BackendSubagentConfig represents subagent configuration from the backend
type BackendSubagentConfig struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	TaskTypes    []string `json:"task_types"`
	Languages    []string `json:"languages"`
	MaxTokens    int      `json:"max_tokens"`
	Active       bool     `json:"active"`
}

// NewBackendAdapter creates a new backend adapter
func NewBackendAdapter(baseURL, authToken string) *BackendAdapter {
	return &BackendAdapter{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		mcpClient: mcp.NewClient(baseURL, authToken),
	}
}

// RegisterSubagentCollection registers subagents from the claude-code-subagents-collection
func (ba *BackendAdapter) RegisterSubagentCollection(ctx context.Context, collectionPath string) error {
	// This would integrate with the subagents collection
	// For now, we'll register some example subagents
	
	subagents := []SubagentRegistration{
		{
			Name:        "code-analyzer",
			Description: "Analyzes code structure, patterns, and quality metrics",
			Version:     "1.0.0",
			Capabilities: []string{"code_analysis", "code_review", "refactoring_suggestions"},
			Endpoint:    "/subagents/code-analyzer",
			Metadata: map[string]interface{}{
				"languages": []string{"go", "python", "javascript", "typescript", "java"},
				"max_file_size": 50000,
				"specializes": []string{"static_analysis", "code_metrics", "security_scan"},
			},
		},
		{
			Name:        "doc-summarizer", 
			Description: "Summarizes documentation, README files, and technical content",
			Version:     "1.0.0",
			Capabilities: []string{"document_summary", "content_extraction", "key_points"},
			Endpoint:    "/subagents/doc-summarizer",
			Metadata: map[string]interface{}{
				"formats": []string{"markdown", "rst", "txt", "html"},
				"max_length": 10000,
				"compression_ratio": 0.3,
			},
		},
		{
			Name:        "data-processor",
			Description: "Processes structured data, logs, and configuration files",
			Version:     "1.0.0", 
			Capabilities: []string{"data_analysis", "log_parsing", "config_validation"},
			Endpoint:    "/subagents/data-processor",
			Metadata: map[string]interface{}{
				"formats": []string{"json", "yaml", "csv", "xml", "log"},
				"max_records": 1000,
				"analytics": true,
			},
		},
		{
			Name:        "context-compressor",
			Description: "Intelligently compresses conversation context while preserving meaning",
			Version:     "1.0.0",
			Capabilities: []string{"context_compression", "semantic_preservation", "priority_ranking"},
			Endpoint:    "/subagents/context-compressor",
			Metadata: map[string]interface{}{
				"compression_algorithms": []string{"semantic", "priority", "temporal"},
				"preserve_relationships": true,
				"max_compression": 0.8,
			},
		},
	}

	for _, subagent := range subagents {
		if err := ba.registerSubagent(ctx, subagent); err != nil {
			return fmt.Errorf("failed to register subagent %s: %w", subagent.Name, err)
		}
	}

	return nil
}

// ExecuteSubagentTask executes a task using a specific subagent via the backend
func (ba *BackendAdapter) ExecuteSubagentTask(ctx context.Context, task *subagent.TaskRequest) (*SubagentResponse, error) {
	// Convert internal task to backend format
	backendTask := map[string]interface{}{
		"subagent_id": task.ID,
		"task_type":   string(task.TaskType),
		"input":       task.Input,
		"context":     task.Context,
		"parameters":  task.Parameters,
		"max_tokens":  task.MaxTokens,
		"priority":    task.Priority,
		"timeout":     task.Timeout.Seconds(),
	}

	// Execute via MCP call_tool
	taskJSON, err := json.Marshal(backendTask)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	traceID := fmt.Sprintf("backend_task_%s_%d", task.ID, time.Now().UnixNano())
	result, err := ba.mcpClient.CallTool(ctx, string(taskJSON), traceID)
	if err != nil {
		return nil, fmt.Errorf("MCP call failed: %w", err)
	}

	// Parse response
	var response SubagentResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAvailableSubagents retrieves the list of available subagents from the backend
func (ba *BackendAdapter) GetAvailableSubagents(ctx context.Context) ([]BackendSubagentConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ba.baseURL+"/subagents", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ba.authToken)
	req.Header.Set("Accept", "application/json")

	resp, err := ba.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var subagents []BackendSubagentConfig
	if err := json.NewDecoder(resp.Body).Decode(&subagents); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return subagents, nil
}

// OptimizeContextWithBackend uses the backend's context optimization service
func (ba *BackendAdapter) OptimizeContextWithBackend(ctx context.Context, messages []string, strategy contextpkg.OptimizationStrategy) (*contextpkg.OptimizationResult, error) {
	request := map[string]interface{}{
		"messages": messages,
		"strategy": strategy,
		"operation": "context_optimization",
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	traceID := fmt.Sprintf("context_opt_%d", time.Now().UnixNano())
	result, err := ba.mcpClient.CallTool(ctx, string(requestJSON), traceID)
	if err != nil {
		return nil, fmt.Errorf("context optimization failed: %w", err)
	}

	var optimizationResult contextpkg.OptimizationResult
	if err := json.Unmarshal([]byte(result), &optimizationResult); err != nil {
		return nil, fmt.Errorf("failed to parse optimization result: %w", err)
	}

	return &optimizationResult, nil
}

// SyncSubagentMetrics synchronizes subagent performance metrics with the backend
func (ba *BackendAdapter) SyncSubagentMetrics(ctx context.Context, metrics map[string]*subagent.SubagentMetrics) error {
	request := map[string]interface{}{
		"operation": "sync_metrics",
		"metrics":   metrics,
		"timestamp": time.Now(),
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	traceID := fmt.Sprintf("sync_metrics_%d", time.Now().UnixNano())
	_, err = ba.mcpClient.CallTool(ctx, string(requestJSON), traceID)
	if err != nil {
		return fmt.Errorf("metrics sync failed: %w", err)
	}

	return nil
}

// GetOptimizationRecommendations gets optimization recommendations from the backend
func (ba *BackendAdapter) GetOptimizationRecommendations(ctx context.Context, userID string, conversationHistory []string) (*OptimizationRecommendations, error) {
	request := map[string]interface{}{
		"operation":     "get_recommendations",
		"user_id":       userID,
		"conversation":  conversationHistory,
		"request_time":  time.Now(),
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	traceID := fmt.Sprintf("recommendations_%s_%d", userID, time.Now().UnixNano())
	result, err := ba.mcpClient.CallTool(ctx, string(requestJSON), traceID)
	if err != nil {
		return nil, fmt.Errorf("recommendations request failed: %w", err)
	}

	var recommendations OptimizationRecommendations
	if err := json.Unmarshal([]byte(result), &recommendations); err != nil {
		return nil, fmt.Errorf("failed to parse recommendations: %w", err)
	}

	return &recommendations, nil
}

// SendTelemetryData sends usage and performance telemetry to the backend
func (ba *BackendAdapter) SendTelemetryData(ctx context.Context, events []metrics.TokenUsageEvent) error {
	request := map[string]interface{}{
		"operation": "telemetry",
		"events":    events,
		"timestamp": time.Now(),
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal telemetry: %w", err)
	}

	traceID := fmt.Sprintf("telemetry_%d", time.Now().UnixNano())
	_, err = ba.mcpClient.CallTool(ctx, string(requestJSON), traceID)
	if err != nil {
		return fmt.Errorf("telemetry upload failed: %w", err)
	}

	return nil
}

// StreamOptimizedChat streams chat with real-time optimization from the backend
func (ba *BackendAdapter) StreamOptimizedChat(ctx context.Context, messages []string, model string) (<-chan StreamMessage, <-chan error) {
	msgChan := make(chan StreamMessage, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(msgChan)
		defer close(errChan)

		request := map[string]interface{}{
			"operation": "stream_chat",
			"messages":  messages,
			"model":     model,
			"stream":    true,
		}

		requestJSON, _ := json.Marshal(request)
		traceID := fmt.Sprintf("stream_chat_%d", time.Now().UnixNano())

		// For streaming, we'd need to implement server-sent events or websockets
		// This is a simplified version that makes a regular call
		result, err := ba.mcpClient.CallTool(ctx, string(requestJSON), traceID)
		if err != nil {
			errChan <- err
			return
		}

		// Parse and send response
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(result), &response); err != nil {
			errChan <- err
			return
		}

		// Send optimization info
		msgChan <- StreamMessage{
			Type: "optimization",
			Data: response["optimization"],
		}

		// Send chat response
		msgChan <- StreamMessage{
			Type: "response", 
			Data: response["content"],
		}
	}()

	return msgChan, errChan
}

// Supporting types

// OptimizationRecommendations contains recommendations for improving optimization
type OptimizationRecommendations struct {
	UserID              string                 `json:"user_id"`
	RecommendedStrategy contextpkg.OptimizationStrategy `json:"recommended_strategy"`
	SuggestedSubagents  []string               `json:"suggested_subagents"`
	PotentialSavings    int                    `json:"potential_savings_tokens"`
	Confidence          float64                `json:"confidence"`
	Reasoning           []string               `json:"reasoning"`
	ImplementationTips  []string               `json:"implementation_tips"`
	EstimatedImprovement float64               `json:"estimated_improvement"`
}

// StreamMessage represents a message in a streaming response
type StreamMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Private methods

func (ba *BackendAdapter) registerSubagent(ctx context.Context, registration SubagentRegistration) error {
	requestBody, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("failed to marshal registration: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ba.baseURL+"/subagents/register", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ba.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := ba.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ContextOptimizationService provides a unified interface for context optimization
type ContextOptimizationService struct {
	localOptimizer  *contextpkg.ContextOptimizer
	backendAdapter  *BackendAdapter
	hybridMode      bool
	preferLocal     bool
}

// NewContextOptimizationService creates a new context optimization service
func NewContextOptimizationService(
	localOptimizer *contextpkg.ContextOptimizer,
	backendAdapter *BackendAdapter,
	hybridMode bool,
) *ContextOptimizationService {
	return &ContextOptimizationService{
		localOptimizer: localOptimizer,
		backendAdapter: backendAdapter,
		hybridMode:     hybridMode,
		preferLocal:    true,
	}
}

// OptimizeContext optimizes context using the best available method
func (cos *ContextOptimizationService) OptimizeContext(ctx context.Context, messages []string, strategy contextpkg.OptimizationStrategy) (*contextpkg.OptimizationResult, error) {
	if cos.hybridMode {
		// Try local first, fallback to backend
		if cos.preferLocal && cos.localOptimizer != nil {
			// Convert messages to ChatMessage format for local optimizer
			chatMessages := make([]client.ChatMessage, len(messages))
			for i, msg := range messages {
				chatMessages[i] = client.ChatMessage{
					Role:    "user", // Simplified - would need better role detection
					Content: msg,
				}
			}
			
			result, _, err := cos.localOptimizer.OptimizeContext(ctx, chatMessages, []contextpkg.ContextChunk{})
			if err == nil {
				// Convert back to result format
				return &contextpkg.OptimizationResult{
					OriginalTokens:   len(strings.Join(messages, " ")) / 4,
					OptimizedTokens:  len(strings.Join(cos.messagesToStrings(result), " ")) / 4,
					Strategy:         strategy,
					ProcessingTime:   time.Since(time.Now()),
				}, nil
			}
		}

		// Fallback to backend
		if cos.backendAdapter != nil {
			return cos.backendAdapter.OptimizeContextWithBackend(ctx, messages, strategy)
		}
	} else {
		// Use only one method based on preference
		if cos.preferLocal && cos.localOptimizer != nil {
			chatMessages := make([]client.ChatMessage, len(messages))
			for i, msg := range messages {
				chatMessages[i] = client.ChatMessage{
					Role:    "user",
					Content: msg,
				}
			}
			
			_, result, err := cos.localOptimizer.OptimizeContext(ctx, chatMessages, []contextpkg.ContextChunk{})
			return result, err
		} else if cos.backendAdapter != nil {
			return cos.backendAdapter.OptimizeContextWithBackend(ctx, messages, strategy)
		}
	}

	return nil, fmt.Errorf("no optimization method available")
}

func (cos *ContextOptimizationService) messagesToStrings(messages []client.ChatMessage) []string {
	strings := make([]string, len(messages))
	for i, msg := range messages {
		strings[i] = msg.Content
	}
	return strings
}

// Import for client package - add this to imports
import (
	"openai-cli/internal/client"
	"strings"
)