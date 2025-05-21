package mcp

import (
   "context"
   "encoding/json"
   "io"
   "net/http"
   "net/http/httptest"
   "strings"
   "testing"
)

func TestCallTool_Success(t *testing.T) {
   server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
       if r.Method != http.MethodPost {
           t.Errorf("Expected POST, got %s", r.Method)
       }
       if !strings.HasSuffix(r.URL.Path, "/mcp") {
           t.Errorf("Expected path /mcp, got %s", r.URL.Path)
       }
       bodyBytes, _ := io.ReadAll(r.Body)
       var reqObj RPCRequest
       if err := json.Unmarshal(bodyBytes, &reqObj); err != nil {
           t.Errorf("Failed to unmarshal request: %v", err)
       }
       if reqObj.Method != "call_tool" {
           t.Errorf("Expected method call_tool, got %s", reqObj.Method)
       }
       respObj := RPCResponse{
           JSONRPC: "2.0",
           ID:      reqObj.ID,
           // Result must be a raw JSON string literal
           Result:  json.RawMessage(`"output-data"`),
       }
       w.Header().Set("Content-Type", "application/json")
       json.NewEncoder(w).Encode(respObj)
   }))
   defer server.Close()

   client := NewClient(server.URL, "")
   result, err := client.CallTool(context.Background(), "input-data", "trace-id")
   if err != nil {
       t.Fatalf("Expected no error, got %v", err)
   }
   if result != "output-data" {
       t.Errorf("Expected output-data, got %s", result)
   }
}

func TestCallTool_RPCError(t *testing.T) {
   server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
       respObj := RPCResponse{
           JSONRPC: "2.0",
           ID:      "trace-id",
           Error: &RPCError{
               Code:    123,
               Message: "something went wrong",
           },
       }
       w.Header().Set("Content-Type", "application/json")
       json.NewEncoder(w).Encode(respObj)
   }))
   defer server.Close()

   client := NewClient(server.URL, "")
   _, err := client.CallTool(context.Background(), "input", "trace-id")
   if err == nil {
       t.Fatalf("Expected error, got nil")
   }
   if !strings.Contains(err.Error(), "rpc error") {
       t.Errorf("Expected rpc error, got %v", err)
   }
}

func TestCallTool_Non200Status(t *testing.T) {
   server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
       http.Error(w, "internal error", http.StatusInternalServerError)
   }))
   defer server.Close()

   client := NewClient(server.URL, "token123")
   _, err := client.CallTool(context.Background(), "input", "trace-id")
   if err == nil {
       t.Fatalf("Expected error, got nil")
   }
   if !strings.Contains(err.Error(), "non-200 status code") {
       t.Errorf("Expected non-200 status code error, got %v", err)
   }
}

func TestCallTool_InvalidJSON(t *testing.T) {
   server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
       w.Header().Set("Content-Type", "application/json")
       w.Write([]byte("invalid-json"))
   }))
   defer server.Close()

   client := NewClient(server.URL, "")
   _, err := client.CallTool(context.Background(), "input", "trace-id")
   if err == nil {
       t.Fatalf("Expected error, got nil")
   }
}