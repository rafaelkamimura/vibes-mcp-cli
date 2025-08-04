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
