package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"openai-cli/internal/client"
	"openai-cli/internal/providers"
)

// stubTestClient implements service.APIClient for CLI integration tests.
type stubTestClient struct {
	completionResp *client.CompletionsResponse
	chatResp       *client.ChatCompletionsResponse
	embedResp      *client.EmbeddingResponse
}

func (s stubTestClient) CreateCompletion(ctx context.Context, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
	return s.completionResp, nil
}
func (s stubTestClient) CreateChatCompletion(ctx context.Context, req client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error) {
	return s.chatResp, nil
}
func (s stubTestClient) CreateEmbedding(ctx context.Context, req client.EmbeddingRequest) (*client.EmbeddingResponse, error) {
	return s.embedResp, nil
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestCLI_CompletionIntegration(t *testing.T) {
	// Setup env and stub provider
	os.Setenv("OPENAI_CLI_API_KEY", "token")
	os.Setenv("OPENAI_CLI_PROVIDER", "test")
	providers.TestClient = stubTestClient{
		completionResp: &client.CompletionsResponse{Choices: []client.Choice{{Text: "hi"}}},
	}
	// Execute command and capture output
	RootCmd.SetArgs([]string{"completion", "--prompt", "foo"})
	output := captureOutput(func() {
		cmd, err := RootCmd.ExecuteC()
		assert.NoError(t, err)
		_ = cmd
	})
	assert.Equal(t, "hi\n", output)
}

func TestCLI_ChatIntegration(t *testing.T) {
	os.Setenv("OPENAI_CLI_API_KEY", "token")
	os.Setenv("OPENAI_CLI_PROVIDER", "test")
	providers.TestClient = stubTestClient{
		chatResp: &client.ChatCompletionsResponse{Choices: []client.ChatCompletionChoice{{Message: client.ChatMessage{Content: "hey"}}}},
	}
	RootCmd.SetArgs([]string{"chat", "--message", "foo"})
	output := captureOutput(func() {
		cmd, err := RootCmd.ExecuteC()
		assert.NoError(t, err)
		_ = cmd
	})
	assert.Equal(t, "hey\n", output)
}

func TestCLI_EmbedIntegration(t *testing.T) {
	os.Setenv("OPENAI_CLI_API_KEY", "token")
	os.Setenv("OPENAI_CLI_PROVIDER", "test")
	providers.TestClient = stubTestClient{
		embedResp: &client.EmbeddingResponse{Data: []client.Embedding{{Index: 0, Embedding: []float64{3.14}}}},
	}
	RootCmd.SetArgs([]string{"embed", "--input", "a"})
	output := captureOutput(func() {
		cmd, err := RootCmd.ExecuteC()
		assert.NoError(t, err)
		_ = cmd
	})
	expected := "Index: 0, Embedding: [3.14]\n"
	assert.Equal(t, expected, output)
}
