package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/prompt"
)

// PromptMCPHandler handles MCP JSON-RPC requests for prompt operations
type PromptMCPHandler struct {
	resourceManager *PromptResourceManager
	toolManager     *PromptToolManager
	aiIntegrator    *PromptAIIntegrator
	logger          *zap.Logger
}

// NewPromptMCPHandler creates a new prompt MCP handler
func NewPromptMCPHandler(
	promptManager prompt.Manager,
	logger *zap.Logger,
) *PromptMCPHandler {
	return &PromptMCPHandler{
		resourceManager: NewPromptResourceManager(promptManager, logger),
		toolManager:     NewPromptToolManager(promptManager, logger),
		aiIntegrator:    NewPromptAIIntegrator(promptManager, logger),
		logger:          logger,
	}
}

// HandleRequest processes MCP JSON-RPC requests for prompt operations
func (pmh *PromptMCPHandler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	pmh.logger.Debug("Handling prompt MCP request", zap.String("method", method))

	switch method {
	// Resource operations
	case "resources/list":
		return pmh.handleListResources(ctx, params)
	case "resources/read":
		return pmh.handleReadResource(ctx, params)
	case "resources/search":
		return pmh.handleSearchResources(ctx, params)
	case "resources/metadata":
		return pmh.handleGetResourceMetadata(ctx, params)

	// Tool operations
	case "tools/list":
		return pmh.handleListTools(ctx, params)
	case "tools/call":
		return pmh.handleCallTool(ctx, params)

	// Prompt-specific operations
	case "prompts/generate":
		return pmh.handleGeneratePrompt(ctx, params)
	case "prompts/validate":
		return pmh.handleValidatePrompt(ctx, params)
	case "prompts/suggest":
		return pmh.handleSuggestTemplates(ctx, params)
	case "prompts/history":
		return pmh.handleGetHistory(ctx, params)

	// AI Integration operations
	case "ai/send":
		return pmh.handleSendToAI(ctx, params)
	case "ai/enhance":
		return pmh.handleEnhancePrompt(ctx, params)
	case "ai/feedback":
		return pmh.handleGetAIFeedback(ctx, params)

	// Metrics and monitoring
	case "metrics/prompt":
		return pmh.handleGetPromptMetrics(ctx, params)
	case "metrics/usage":
		return pmh.handleGetUsageMetrics(ctx, params)

	// Context operations
	case "context/detect":
		return pmh.handleDetectContext(ctx, params)
	case "context/analyze":
		return pmh.handleAnalyzeWorkspace(ctx, params)

	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

// Resource handlers

func (pmh *PromptMCPHandler) handleListResources(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var reqParams struct {
		Type     string `json:"type,omitempty"`
		Category string `json:"category,omitempty"`
	}
	
	if len(params) > 0 {
		if err := json.Unmarshal(params, &reqParams); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	resources, err := pmh.resourceManager.ListResources(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by category if specified
	if reqParams.Category != "" {
		var filtered []PromptResource
		for _, resource := range resources {
			if resource.Category == reqParams.Category {
				filtered = append(filtered, resource)
			}
		}
		resources = filtered
	}

	return map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
	}, nil
}

func (pmh *PromptMCPHandler) handleReadResource(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var reqParams struct {
		URI string `json:"uri"`
	}
	
	if err := json.Unmarshal(params, &reqParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if reqParams.URI == "" {
		return nil, fmt.Errorf("uri parameter is required")
	}

	resource, err := pmh.resourceManager.GetResource(ctx, reqParams.URI)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (pmh *PromptMCPHandler) handleSearchResources(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var reqParams struct {
		Query string `json:"query"`
	}
	
	if err := json.Unmarshal(params, &reqParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if reqParams.Query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	resources, err := pmh.resourceManager.SearchResources(ctx, reqParams.Query)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
		"query":     reqParams.Query,
	}, nil
}

func (pmh *PromptMCPHandler) handleGetResourceMetadata(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var reqParams struct {
		URI string `json:"uri"`
	}
	
	if err := json.Unmarshal(params, &reqParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	metadata, err := pmh.resourceManager.GetResourceMetadata(ctx, reqParams.URI)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// Tool handlers

func (pmh *PromptMCPHandler) handleListTools(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var reqParams struct {
		Type string `json:"type,omitempty"`
	}
	
	if len(params) > 0 {
		if err := json.Unmarshal(params, &reqParams); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	tools := pmh.toolManager.GetTools()

	// Filter by type if needed (all our tools are prompt-related)
	if reqParams.Type != "" && reqParams.Type != "prompt" {
		tools = []PromptTool{} // Return empty if not prompt type
	}

	return map[string]interface{}{
		"tools": tools,
		"total": len(tools),
	}, nil
}

func (pmh *PromptMCPHandler) handleCallTool(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var reqParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	
	if err := json.Unmarshal(params, &reqParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if reqParams.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	result, err := pmh.toolManager.CallTool(ctx, reqParams.Name, reqParams.Arguments)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Prompt operation handlers

func (pmh *PromptMCPHandler) handleGeneratePrompt(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var genParams PromptGenerateParams
	
	if err := json.Unmarshal(params, &genParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Use the tool manager to generate the prompt
	args := map[string]interface{}{
		"template_name": genParams.TemplateName,
		"parameters":    genParams.Parameters,
		"context":       genParams.Context,
		"interactive":   genParams.Interactive,
		"validate":      genParams.Validate,
		"output_format": genParams.OutputFormat,
	}

	result, err := pmh.toolManager.CallTool(ctx, "generate_prompt", args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pmh *PromptMCPHandler) handleValidatePrompt(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var validateParams PromptValidateParams
	
	if err := json.Unmarshal(params, &validateParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	args := map[string]interface{}{
		"template_name": validateParams.TemplateName,
		"content":       validateParams.Content,
	}

	result, err := pmh.toolManager.CallTool(ctx, "validate_template", args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pmh *PromptMCPHandler) handleSuggestTemplates(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var suggestParams PromptSuggestParams
	
	if err := json.Unmarshal(params, &suggestParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	args := map[string]interface{}{
		"context":     suggestParams.Context,
		"task_type":   suggestParams.TaskType,
		"language":    suggestParams.Language,
		"framework":   suggestParams.Framework,
		"max_results": suggestParams.MaxResults,
	}

	result, err := pmh.toolManager.CallTool(ctx, "suggest_templates", args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pmh *PromptMCPHandler) handleGetHistory(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var historyParams PromptHistoryParams
	
	if err := json.Unmarshal(params, &historyParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	args := map[string]interface{}{
		"limit":      historyParams.Limit,
		"filter":     historyParams.Filter,
		"repository": historyParams.Repository,
		"language":   historyParams.Language,
		"template":   historyParams.Template,
		"start_date": historyParams.StartDate,
		"end_date":   historyParams.EndDate,
	}

	result, err := pmh.toolManager.CallTool(ctx, "get_history", args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// AI Integration handlers

func (pmh *PromptMCPHandler) handleSendToAI(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var aiParams AIAssistantParams
	
	if err := json.Unmarshal(params, &aiParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	result, err := pmh.aiIntegrator.SendToAssistant(ctx, aiParams)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pmh *PromptMCPHandler) handleEnhancePrompt(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var enhanceParams struct {
		Content   string            `json:"content"`
		Context   *WorkspaceContext `json:"context,omitempty"`
		Assistant string            `json:"assistant,omitempty"`
	}
	
	if err := json.Unmarshal(params, &enhanceParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	result, err := pmh.aiIntegrator.EnhancePrompt(ctx, enhanceParams.Content, enhanceParams.Context, enhanceParams.Assistant)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pmh *PromptMCPHandler) handleGetAIFeedback(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var feedbackParams struct {
		Content   string `json:"content"`
		Assistant string `json:"assistant,omitempty"`
	}
	
	if err := json.Unmarshal(params, &feedbackParams); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	feedback, err := pmh.aiIntegrator.GetFeedback(ctx, feedbackParams.Content, feedbackParams.Assistant)
	if err != nil {
		return nil, err
	}

	return feedback, nil
}

// Metrics handlers

func (pmh *PromptMCPHandler) handleGetPromptMetrics(ctx context.Context, params json.RawMessage) (interface{}, error) {
	stats, err := pmh.resourceManager.GetResourceStats(ctx)
	if err != nil {
		return nil, err
	}

	// Add additional metrics
	metrics := map[string]interface{}{
		"resources":    stats,
		"tools":        len(pmh.toolManager.tools),
		"timestamp":    time.Now(),
		"uptime":       time.Since(time.Now()), // Would track actual uptime
		"version":      "1.0.0",
	}

	return metrics, nil
}

func (pmh *PromptMCPHandler) handleGetUsageMetrics(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// This would integrate with telemetry/metrics system
	usage := map[string]interface{}{
		"total_requests":       0,
		"successful_requests":  0,
		"failed_requests":      0,
		"average_response_time": "0ms",
		"popular_templates":    []string{},
		"popular_tools":        []string{},
	}

	return usage, nil
}

// Context handlers

func (pmh *PromptMCPHandler) handleDetectContext(ctx context.Context, params json.RawMessage) (interface{}, error) {
	result, err := pmh.toolManager.CallTool(ctx, "detect_context", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pmh *PromptMCPHandler) handleAnalyzeWorkspace(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var analyzeParams struct {
		DeepScan             bool `json:"deep_scan,omitempty"`
		IncludeDependencies  bool `json:"include_dependencies,omitempty"`
	}
	
	if len(params) > 0 {
		if err := json.Unmarshal(params, &analyzeParams); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	args := map[string]interface{}{
		"deep_scan":             analyzeParams.DeepScan,
		"include_dependencies":  analyzeParams.IncludeDependencies,
	}

	result, err := pmh.toolManager.CallTool(ctx, "workspace_analysis", args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HTTP handler for serving MCP over HTTP
func (pmh *PromptMCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rpcReq RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		pmh.writeErrorResponse(w, "invalid JSON-RPC request", -32700, rpcReq.ID)
		return
	}

	// Convert params to JSON RawMessage
	var paramsRaw json.RawMessage
	if rpcReq.Params != nil {
		var err error
		paramsRaw, err = json.Marshal(rpcReq.Params)
		if err != nil {
			pmh.writeErrorResponse(w, "invalid parameters", -32602, rpcReq.ID)
			return
		}
	}

	// Handle the request
	result, err := pmh.HandleRequest(r.Context(), rpcReq.Method, paramsRaw)
	if err != nil {
		pmh.writeErrorResponse(w, err.Error(), -32000, rpcReq.ID)
		return
	}

	// Write successful response
	response := RPCResponse{
		JSONRPC: "2.0",
		ID:      rpcReq.ID,
	}

	if result != nil {
		resultBytes, err := json.Marshal(result)
		if err != nil {
			pmh.writeErrorResponse(w, "failed to marshal result", -32603, rpcReq.ID)
			return
		}
		response.Result = resultBytes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response
func (pmh *PromptMCPHandler) writeErrorResponse(w http.ResponseWriter, message string, code int, id string) {
	response := RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors are still HTTP 200
	json.NewEncoder(w).Encode(response)
}

// Middleware for logging and authentication
func (pmh *PromptMCPHandler) WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log request
		pmh.logger.Debug("MCP request received",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr))

		next.ServeHTTP(w, r)

		// Log response
		pmh.logger.Debug("MCP request completed",
			zap.Duration("duration", time.Since(start)))
	})
}

// WithAuth adds authentication middleware
func (pmh *PromptMCPHandler) WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			pmh.writeErrorResponse(w, "authorization required", -32001, "")
			return
		}

		// Validate token (this would integrate with the auth system)
		if !strings.HasPrefix(authHeader, "Bearer ") {
			pmh.writeErrorResponse(w, "invalid authorization format", -32001, "")
			return
		}

		// TODO: Integrate with actual auth validation
		next.ServeHTTP(w, r)
	})
}