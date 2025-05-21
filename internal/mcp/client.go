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