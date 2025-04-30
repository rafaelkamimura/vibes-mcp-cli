package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// HTTPClient defines the interface for HTTP operations.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client wraps HTTP interactions with OpenAI API.
type Client struct {
	httpClient HTTPClient
	apiKey     string
	baseURL    string
}

// NewClient creates a new OpenAI API client.
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		baseURL:    baseURL,
	}
}

// doRequest wraps http.Client.Do with retry, backoff, and rate-limit handling.
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	const maxRetries = 3
	backoff := 500 * time.Millisecond
	var resp *http.Response
	var err error
	for i := 0; i < maxRetries; i++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			// network or context error, retry
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		// handle 429 Too Many Requests
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			resp.Body.Close()
			if secs, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
				time.Sleep(time.Duration(secs) * time.Second)
			} else {
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}
		// handle 5xx server errors
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		// other status codes (including 2xx, 4xx) propagate
		return resp, nil
	}
	// retries exhausted
	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries, err)
	}
	return resp, nil
}

// CreateCompletion sends a completions request.
func (c *Client) CreateCompletion(ctx context.Context, reqData CompletionsRequest) (*CompletionsResponse, error) {
	url := fmt.Sprintf("%s/v1/completions", c.baseURL)
	bodyBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 status code: %d, body: %s", resp.StatusCode, string(data))
	}
	var cResp CompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
		return nil, err
	}
	return &cResp, nil
}

// CreateChatCompletion sends a chat completion request.
func (c *Client) CreateChatCompletion(ctx context.Context, reqData ChatCompletionsRequest) (*ChatCompletionsResponse, error) {
	url := fmt.Sprintf("%s/v1/chat/completions", c.baseURL)
	bodyBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 status code: %d, body: %s", resp.StatusCode, string(data))
	}
	var cResp ChatCompletionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
		return nil, err
	}
	return &cResp, nil
}

// CreateEmbedding sends an embedding request.
func (c *Client) CreateEmbedding(ctx context.Context, reqData EmbeddingRequest) (*EmbeddingResponse, error) {
	url := fmt.Sprintf("%s/v1/embeddings", c.baseURL)
	bodyBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 status code: %d, body: %s", resp.StatusCode, string(data))
	}
	var eResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&eResp); err != nil {
		return nil, err
	}
	return &eResp, nil
}
