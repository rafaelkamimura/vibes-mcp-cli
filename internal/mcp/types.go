package mcp

import "encoding/json"

// RPCRequest represents a JSON-RPC 2.0 request.
type RPCRequest struct {
   JSONRPC string    `json:"jsonrpc"`
   ID      string    `json:"id"`
   Method  string    `json:"method"`
   Params  RPCParams `json:"params"`
}

// RPCParams holds parameters for call_tool.
type RPCParams struct {
   Input string `json:"input"`
}

// RPCResponse represents a JSON-RPC 2.0 response.
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