package client

// CompletionsRequest represents a completion request payload.
type CompletionsRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// Token represents the access token returned by the auth endpoint.
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// UserCreate is the registration request payload for creating a new user.
type UserCreate struct {
	Username string `json:"username"`
	Password string `json:"password"`
	FullName string `json:"full_name,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Text         string      `json:"text"`
	Index        int         `json:"index"`
	LogProbs     interface{} `json:"logprobs,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// CompletionsResponse is the response from completion API.
type CompletionsResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// ChatMessage represents a message in chat.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionsRequest payload for chat API.
type ChatCompletionsRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatCompletionChoice represents a single chat choice.
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatCompletionsResponse is the response from chat API.
type ChatCompletionsResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

// EmbeddingRequest payload for embeddings API.
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// Embedding represents a single embedding.
type Embedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// EmbeddingResponse is the response from embeddings API.
type EmbeddingResponse struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
