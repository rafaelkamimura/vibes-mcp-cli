package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// rawCall sends a generic JSON-RPC request with the given method and params,
// returning the raw JSON result.
func (c *Client) rawCall(ctx context.Context, method string, params interface{}, traceID string) (json.RawMessage, error) {
	// Construct JSON-RPC request
	reqObj := struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      string      `json:"id"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}{
		JSONRPC: "2.0",
		ID:      traceID,
		Method:  method,
		Params:  params,
	}
	bodyBytes, err := json.Marshal(reqObj)
	if err != nil {
		return nil, err
	}
	url := c.BaseURL + "/mcp"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}
	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: code %d message %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

// ReadFile retrieves the content of the file at the given path via JSON-RPC "read_file" method.
func (c *Client) ReadFile(ctx context.Context, path string) (ReadFileResult, error) {
	// Generate a trace ID for this call
	traceID := fmt.Sprintf("%d", time.Now().UnixNano())
	params := map[string]interface{}{"path": path, "offset_lines": 0}
	raw, err := c.rawCall(ctx, "read_file", params, traceID)
	if err != nil {
		return ReadFileResult{}, err
	}
	var res ReadFileResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return ReadFileResult{}, err
	}
	return res, nil
}

// Client interacts with the MCP JSON-RPC endpoint.
type Client struct {
	httpClient HTTPClient
	BaseURL    string
	AuthToken  string
}

// NewClient creates a new MCP client with default HTTP client.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		httpClient: &http.Client{},
		BaseURL:    strings.TrimRight(baseURL, "/"),
		AuthToken:  authToken,
	}
}

// NewClientWithHTTPClient allows injecting a custom HTTPClient (e.g., for testing).
func NewClientWithHTTPClient(httpClient HTTPClient, baseURL, authToken string) *Client {
	return &Client{
		httpClient: httpClient,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		AuthToken:  authToken,
	}
}

// CallTool sends a JSON-RPC call_tool request and returns the result.
func (c *Client) CallTool(ctx context.Context, input, traceID string) (string, error) {
	reqObj := RPCRequest{
		JSONRPC: "2.0",
		ID:      traceID,
		Method:  "call_tool",
		Params:  RPCParams{Input: input},
	}
	bodyBytes, err := json.Marshal(reqObj)
	if err != nil {
		return "", err
	}
	url := c.BaseURL + "/mcp"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}
	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return "", err
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("rpc error: code %d message %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	// Unmarshal result (JSON string) into Go string
	var resultStr string
	if err := json.Unmarshal(rpcResp.Result, &resultStr); err != nil {
		return "", err
	}
	return resultStr, nil
}

// CallSubagent sends a JSON-RPC request to a specific subagent and returns the response.
func (c *Client) CallSubagent(ctx context.Context, params SubagentCallParams, traceID string) (*SubagentResponse, error) {
	raw, err := c.rawCall(ctx, "call_subagent", params, traceID)
	if err != nil {
		return nil, err
	}
	
	var response SubagentResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subagent response: %w", err)
	}
	
	return &response, nil
}

// OptimizeContext sends a JSON-RPC request to optimize conversation context.
func (c *Client) OptimizeContext(ctx context.Context, params ContextOptimizeParams, traceID string) (*OptimizationResult, error) {
	raw, err := c.rawCall(ctx, "optimize_context", params, traceID)
	if err != nil {
		return nil, err
	}
	
	var result OptimizationResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal optimization result: %w", err)
	}
	
	return &result, nil
}

// ListSubagents retrieves the list of available subagents.
func (c *Client) ListSubagents(ctx context.Context, traceID string) (*SubagentListResult, error) {
	raw, err := c.rawCall(ctx, "list_subagents", map[string]interface{}{}, traceID)
	if err != nil {
		return nil, err
	}
	
	var result SubagentListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subagent list: %w", err)
	}
	
	return &result, nil
}

// GetSubagentInfo retrieves detailed information about a specific subagent.
func (c *Client) GetSubagentInfo(ctx context.Context, subagentID, traceID string) (*SubagentInfo, error) {
	params := map[string]interface{}{
		"subagent_id": subagentID,
	}
	
	raw, err := c.rawCall(ctx, "get_subagent_info", params, traceID)
	if err != nil {
		return nil, err
	}
	
	var info SubagentInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subagent info: %w", err)
	}
	
	return &info, nil
}

// RouteTask intelligently routes a task to the best available subagent.
func (c *Client) RouteTask(ctx context.Context, taskType, input string, context []ContextChunk, traceID string) (*SubagentResponse, error) {
	params := map[string]interface{}{
		"task_type": taskType,
		"input":     input,
		"context":   context,
	}
	
	raw, err := c.rawCall(ctx, "route_task", params, traceID)
	if err != nil {
		return nil, err
	}
	
	var response SubagentResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task routing response: %w", err)
	}
	
	return &response, nil
}

// BatchOptimizeContext optimizes multiple conversation contexts in parallel.
func (c *Client) BatchOptimizeContext(ctx context.Context, requests []ContextOptimizeParams, traceID string) ([]*OptimizationResult, error) {
	params := map[string]interface{}{
		"requests": requests,
	}
	
	raw, err := c.rawCall(ctx, "batch_optimize_context", params, traceID)
	if err != nil {
		return nil, err
	}
	
	var results []*OptimizationResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch optimization results: %w", err)
	}
	
	return results, nil
}

// GetOptimizationMetrics retrieves metrics about context optimization performance.
func (c *Client) GetOptimizationMetrics(ctx context.Context, traceID string) (map[string]interface{}, error) {
	raw, err := c.rawCall(ctx, "get_optimization_metrics", map[string]interface{}{}, traceID)
	if err != nil {
		return nil, err
	}
	
	var metrics map[string]interface{}
	if err := json.Unmarshal(raw, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal optimization metrics: %w", err)
	}
	
	return metrics, nil
}

// Prompt-specific MCP client methods

// ListPromptResources retrieves all available prompt template resources.
func (c *Client) ListPromptResources(ctx context.Context, traceID string) ([]PromptResource, error) {
	raw, err := c.rawCall(ctx, "resources/list", map[string]interface{}{
		"type": "prompt",
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var resources []PromptResource
	if err := json.Unmarshal(raw, &resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt resources: %w", err)
	}
	
	return resources, nil
}

// GetPromptResource retrieves a specific prompt template resource.
func (c *Client) GetPromptResource(ctx context.Context, uri, traceID string) (*PromptResource, error) {
	params := map[string]interface{}{
		"uri": uri,
	}
	
	raw, err := c.rawCall(ctx, "resources/read", params, traceID)
	if err != nil {
		return nil, err
	}
	
	var resource PromptResource
	if err := json.Unmarshal(raw, &resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt resource: %w", err)
	}
	
	return &resource, nil
}

// ListPromptTools retrieves all available prompt operation tools.
func (c *Client) ListPromptTools(ctx context.Context, traceID string) ([]PromptTool, error) {
	raw, err := c.rawCall(ctx, "tools/list", map[string]interface{}{
		"type": "prompt",
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var tools []PromptTool
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt tools: %w", err)
	}
	
	return tools, nil
}

// GeneratePrompt generates a prompt from a template via MCP tool call.
func (c *Client) GeneratePrompt(ctx context.Context, params PromptGenerateParams, traceID string) (*PromptGenerateResult, error) {
	raw, err := c.rawCall(ctx, "tools/call", map[string]interface{}{
		"name":      "generate_prompt",
		"arguments": params,
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var result PromptGenerateResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt generation result: %w", err)
	}
	
	return &result, nil
}

// ValidatePromptTemplate validates a prompt template via MCP tool call.
func (c *Client) ValidatePromptTemplate(ctx context.Context, params PromptValidateParams, traceID string) (*PromptValidateResult, error) {
	raw, err := c.rawCall(ctx, "tools/call", map[string]interface{}{
		"name":      "validate_template",
		"arguments": params,
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var result PromptValidateResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation result: %w", err)
	}
	
	return &result, nil
}

// DetectWorkspaceContext detects current workspace context via MCP tool call.
func (c *Client) DetectWorkspaceContext(ctx context.Context, traceID string) (*WorkspaceContext, error) {
	raw, err := c.rawCall(ctx, "tools/call", map[string]interface{}{
		"name":      "detect_context",
		"arguments": map[string]interface{}{},
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var context WorkspaceContext
	if err := json.Unmarshal(raw, &context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspace context: %w", err)
	}
	
	return &context, nil
}

// SuggestPromptTemplates suggests relevant templates based on context via MCP tool call.
func (c *Client) SuggestPromptTemplates(ctx context.Context, params PromptSuggestParams, traceID string) (*PromptSuggestResult, error) {
	raw, err := c.rawCall(ctx, "tools/call", map[string]interface{}{
		"name":      "suggest_templates",
		"arguments": params,
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var result PromptSuggestResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template suggestions: %w", err)
	}
	
	return &result, nil
}

// GetPromptHistory retrieves prompt generation history via MCP tool call.
func (c *Client) GetPromptHistory(ctx context.Context, params PromptHistoryParams, traceID string) (*PromptHistoryResult, error) {
	raw, err := c.rawCall(ctx, "tools/call", map[string]interface{}{
		"name":      "get_history",
		"arguments": params,
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var result PromptHistoryResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt history: %w", err)
	}
	
	return &result, nil
}

// SendToAIAssistant sends generated prompt to AI assistant via MCP tool call.
func (c *Client) SendToAIAssistant(ctx context.Context, params AIAssistantParams, traceID string) (*AIAssistantResult, error) {
	raw, err := c.rawCall(ctx, "tools/call", map[string]interface{}{
		"name":      "send_to_ai",
		"arguments": params,
	}, traceID)
	if err != nil {
		return nil, err
	}
	
	var result AIAssistantResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI assistant result: %w", err)
	}
	
	return &result, nil
}

// GetPromptMetrics retrieves prompt system metrics and statistics.
func (c *Client) GetPromptMetrics(ctx context.Context, traceID string) (map[string]interface{}, error) {
	raw, err := c.rawCall(ctx, "get_prompt_metrics", map[string]interface{}{}, traceID)
	if err != nil {
		return nil, err
	}
	
	var metrics map[string]interface{}
	if err := json.Unmarshal(raw, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prompt metrics: %w", err)
	}
	
	return metrics, nil
}
