package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"openai-cli/internal/client"
	"time"
)

// AnthropicClient wraps interaction with Anthropic Claude API.
type AnthropicClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Anthropic client. If baseURL is empty, defaults to api.anthropic.com
func NewClient(apiKey, baseURL string) *AnthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateCompletion is not supported for Anthropic provider.
func (c *AnthropicClient) CreateCompletion(ctx context.Context, reqData client.CompletionsRequest) (*client.CompletionsResponse, error) {
	return nil, fmt.Errorf("completion endpoint not supported by provider anthropic")
}

// CreateChatCompletion sends a chat-like completion request to Anthropic.
func (c *AnthropicClient) CreateChatCompletion(ctx context.Context, reqData client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error) {
	// Build prompt from messages
	prompt := ""
	for _, msg := range reqData.Messages {
		switch msg.Role {
		case "user":
			prompt += "Human: " + msg.Content + "\n\n"
		case "assistant":
			prompt += "Assistant: " + msg.Content + "\n\n"
		default:
			prompt += msg.Content + "\n\n"
		}
	}
	// Instruct model to respond
	prompt += "Assistant:"

	payload := map[string]interface{}{
		"model":                reqData.Model,
		"prompt":               prompt,
		"max_tokens_to_sample": 512,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v1/complete", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic non-200 status: %d, body: %s", resp.StatusCode, string(data))
	}
	var ar struct {
		ID         string `json:"id"`
		Model      string `json:"model"`
		Completion string `json:"completion"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	result := &client.ChatCompletionsResponse{
		ID:      ar.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   ar.Model,
		Choices: []client.ChatCompletionChoice{
			{
				Index:        0,
				Message:      client.ChatMessage{Role: "assistant", Content: ar.Completion},
				FinishReason: ar.StopReason,
			},
		},
	}
	return result, nil
}

// CreateEmbedding is not supported for Anthropic provider.
func (c *AnthropicClient) CreateEmbedding(ctx context.Context, reqData client.EmbeddingRequest) (*client.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embedding endpoint not supported by provider anthropic")
}
