package cmd

import (
   "bytes"
   "context"
   "encoding/json"
   "net/http"
   "net/http/httptest"
   "testing"

   "openai-cli/internal/client"
   "openai-cli/internal/config"
   "openai-cli/internal/providers"
   "github.com/stretchr/testify/assert"
)

// stubAPIClient implements service.APIClient for endpoint tests.
type stubAPIClient struct {
   completionResp *client.CompletionsResponse
   chatResp       *client.ChatCompletionsResponse
   embedResp      *client.EmbeddingResponse
}

func (s stubAPIClient) CreateCompletion(ctx context.Context, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
   return s.completionResp, nil
}
func (s stubAPIClient) CreateChatCompletion(ctx context.Context, req client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error) {
   return s.chatResp, nil
}
func (s stubAPIClient) CreateEmbedding(ctx context.Context, req client.EmbeddingRequest) (*client.EmbeddingResponse, error) {
   return s.embedResp, nil
}

func setupTestServer() *httptest.Server {
   // configure global cfg and stub provider client
   cfg = &config.Config{APIKey: "", BaseURL: "", Provider: "test"}
   // prepare stub with sample responses
   providers.TestClient = stubAPIClient{
       completionResp: &client.CompletionsResponse{Choices: []client.Choice{{Text: "ok"}}},
       chatResp:       &client.ChatCompletionsResponse{Choices: []client.ChatCompletionChoice{{Message: client.ChatMessage{Content: "hi"}}}},
       embedResp:      &client.EmbeddingResponse{Data: []client.Embedding{{Index: 0, Embedding: []float64{1.23}}}},
   }
   handler := mcpHandler()
   server := httptest.NewServer(handler)
   return server
}

func TestCompletionsEndpoint(t *testing.T) {
   server := setupTestServer()
   defer server.Close()
   // send POST
   reqBody := client.CompletionsRequest{Model: "m", Prompt: "p"}
   b, _ := json.Marshal(reqBody)
   resp, err := http.Post(server.URL+"/v1/completions", "application/json", bytes.NewReader(b))
   assert.NoError(t, err)
   defer resp.Body.Close()
   assert.Equal(t, http.StatusOK, resp.StatusCode)
   var cResp client.CompletionsResponse
   err = json.NewDecoder(resp.Body).Decode(&cResp)
   assert.NoError(t, err)
   assert.Equal(t, "ok", cResp.Choices[0].Text)
}

func TestChatEndpoint(t *testing.T) {
   server := setupTestServer()
   defer server.Close()
   // send POST
   reqBody := client.ChatCompletionsRequest{Model: "m", Messages: []client.ChatMessage{{Role: "user", Content: "hi"}}}
   b, _ := json.Marshal(reqBody)
   resp, err := http.Post(server.URL+"/v1/chat/completions", "application/json", bytes.NewReader(b))
   assert.NoError(t, err)
   defer resp.Body.Close()
   assert.Equal(t, http.StatusOK, resp.StatusCode)
   var cResp client.ChatCompletionsResponse
   err = json.NewDecoder(resp.Body).Decode(&cResp)
   assert.NoError(t, err)
   assert.Equal(t, "hi", cResp.Choices[0].Message.Content)
}

func TestEmbeddingsEndpoint(t *testing.T) {
   server := setupTestServer()
   defer server.Close()
   // send POST
   reqBody := client.EmbeddingRequest{Model: "m", Input: []string{"a"}}
   b, _ := json.Marshal(reqBody)
   resp, err := http.Post(server.URL+"/v1/embeddings", "application/json", bytes.NewReader(b))
   assert.NoError(t, err)
   defer resp.Body.Close()
   assert.Equal(t, http.StatusOK, resp.StatusCode)
   var eResp client.EmbeddingResponse
   err = json.NewDecoder(resp.Body).Decode(&eResp)
   assert.NoError(t, err)
   assert.Equal(t, 1.23, eResp.Data[0].Embedding[0])
}