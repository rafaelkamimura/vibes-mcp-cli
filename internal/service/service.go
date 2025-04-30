package service

import (
	"context"

	"openai-cli/internal/client"
)

// APIClient defines methods used by Service.
type APIClient interface {
	CreateCompletion(ctx context.Context, req client.CompletionsRequest) (*client.CompletionsResponse, error)
	CreateChatCompletion(ctx context.Context, req client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error)
	CreateEmbedding(ctx context.Context, req client.EmbeddingRequest) (*client.EmbeddingResponse, error)
}

// Service orchestrates API calls.
type Service struct {
	client APIClient
}

// NewService creates a new Service.
func NewService(client APIClient) *Service {
	return &Service{client: client}
}

// CreateCompletion wraps client.CreateCompletion.
func (s *Service) CreateCompletion(ctx context.Context, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
	return s.client.CreateCompletion(ctx, req)
}

// CreateChatCompletion wraps client.CreateChatCompletion.
func (s *Service) CreateChatCompletion(ctx context.Context, req client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error) {
	return s.client.CreateChatCompletion(ctx, req)
}

// CreateEmbedding wraps client.CreateEmbedding.
func (s *Service) CreateEmbedding(ctx context.Context, req client.EmbeddingRequest) (*client.EmbeddingResponse, error) {
	return s.client.CreateEmbedding(ctx, req)
}
