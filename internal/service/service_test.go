package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"openai-cli/internal/client"
)

// stubAPIClient implements APIClient for testing purposes.
type stubAPIClient struct {
	// For CreateCompletion
	lastCompletionReq client.CompletionsRequest
	completionResp    *client.CompletionsResponse
	completionErr     error
	// For CreateChatCompletion
	lastChatReq client.ChatCompletionsRequest
	chatResp    *client.ChatCompletionsResponse
	chatErr     error
	// For CreateEmbedding
	lastEmbedReq client.EmbeddingRequest
	embedResp    *client.EmbeddingResponse
	embedErr     error
}

func (s *stubAPIClient) CreateCompletion(ctx context.Context, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
	s.lastCompletionReq = req
	return s.completionResp, s.completionErr
}

func (s *stubAPIClient) CreateChatCompletion(ctx context.Context, req client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error) {
	s.lastChatReq = req
	return s.chatResp, s.chatErr
}

func (s *stubAPIClient) CreateEmbedding(ctx context.Context, req client.EmbeddingRequest) (*client.EmbeddingResponse, error) {
	s.lastEmbedReq = req
	return s.embedResp, s.embedErr
}

func TestService_CreateCompletion_Success(t *testing.T) {
	stub := &stubAPIClient{
		completionResp: &client.CompletionsResponse{
			Choices: []client.Choice{{Text: "hello"}},
		},
		completionErr: nil,
	}
	svc := NewService(stub)
	req := client.CompletionsRequest{Model: "m", Prompt: "p"}
	resp, err := svc.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, stub.completionResp, resp)
	assert.Equal(t, req, stub.lastCompletionReq)
}

func TestService_CreateCompletion_Error(t *testing.T) {
	stub := &stubAPIClient{completionResp: nil, completionErr: errors.New("fail")}
	svc := NewService(stub)
	resp, err := svc.CreateCompletion(context.Background(), client.CompletionsRequest{})
	assert.Nil(t, resp)
	assert.EqualError(t, err, "fail")
}

func TestService_CreateChatCompletion_Success(t *testing.T) {
	stub := &stubAPIClient{
		chatResp: &client.ChatCompletionsResponse{
			Choices: []client.ChatCompletionChoice{{Message: client.ChatMessage{Content: "hi"}}},
		},
		chatErr: nil,
	}
	svc := NewService(stub)
	req := client.ChatCompletionsRequest{Model: "m", Messages: []client.ChatMessage{{Role: "user", Content: "p"}}}
	resp, err := svc.CreateChatCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, stub.chatResp, resp)
	assert.Equal(t, req, stub.lastChatReq)
}

func TestService_CreateChatCompletion_Error(t *testing.T) {
	stub := &stubAPIClient{chatResp: nil, chatErr: errors.New("chat fail")}
	svc := NewService(stub)
	resp, err := svc.CreateChatCompletion(context.Background(), client.ChatCompletionsRequest{})
	assert.Nil(t, resp)
	assert.EqualError(t, err, "chat fail")
}

func TestService_CreateEmbedding_Success(t *testing.T) {
	stub := &stubAPIClient{
		embedResp: &client.EmbeddingResponse{
			Data: []client.Embedding{{Embedding: []float64{0.1}}},
		},
		embedErr: nil,
	}
	svc := NewService(stub)
	req := client.EmbeddingRequest{Model: "m", Input: []string{"x"}}
	resp, err := svc.CreateEmbedding(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, stub.embedResp, resp)
	assert.Equal(t, req, stub.lastEmbedReq)
}

func TestService_CreateEmbedding_Error(t *testing.T) {
	stub := &stubAPIClient{embedResp: nil, embedErr: errors.New("embed fail")}
	svc := NewService(stub)
	resp, err := svc.CreateEmbedding(context.Background(), client.EmbeddingRequest{})
	assert.Nil(t, resp)
	assert.EqualError(t, err, "embed fail")
}
