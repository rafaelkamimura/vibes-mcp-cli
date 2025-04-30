package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubTransport implements HTTPClient for testing retry and header propagation.
type stubTransport struct {
	responses []httpResponse
	calls     int
	lastReq   *http.Request
}

type httpResponse struct {
	resp *http.Response
	err  error
}

func (s *stubTransport) Do(req *http.Request) (*http.Response, error) {
	s.lastReq = req
	if s.calls < len(s.responses) {
		r := s.responses[s.calls]
		s.calls++
		return r.resp, r.err
	}
	// default: no response
	return nil, errors.New("no more stub responses")
}

func makeJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestCreateCompletion_Success(t *testing.T) {
	// Prepare stub transport: single successful response
	jsonBody := `{"id":"1","object":"text_completion","created":1,"model":"m","choices":[{"text":"ok","index":0}]}`
	stub := &stubTransport{responses: []httpResponse{{resp: makeJSONResponse(200, jsonBody), err: nil}}}
	cli := NewClient("secret-key", "http://example.com")
	cli.httpClient = stub

	req := CompletionsRequest{Model: "m", Prompt: "hi"}
	resp, err := cli.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ok", resp.Choices[0].Text)
	// Authorization header check
	auth := stub.lastReq.Header.Get("Authorization")
	assert.Equal(t, "Bearer secret-key", auth)
}

func TestCreateCompletion_RetryOn5xx(t *testing.T) {
	// First a 500 error, then success
	jsonBody := `{"id":"1","object":"text_completion","created":1,"model":"m","choices":[{"text":"retry","index":0}]}`
	resp500 := makeJSONResponse(500, "server error")
	resp200 := makeJSONResponse(200, jsonBody)
	stub := &stubTransport{responses: []httpResponse{{resp: resp500, err: nil}, {resp: resp200, err: nil}}}
	cli := NewClient("key", "url")
	cli.httpClient = stub

	req := CompletionsRequest{Model: "m", Prompt: "hi"}
	resp, err := cli.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "retry", resp.Choices[0].Text)
	assert.Equal(t, 2, stub.calls)
}

func TestCreateCompletion_RateLimitRetry(t *testing.T) {
	// 429 then success
	bodyOK := `{"id":"1","object":"text_completion","created":1,"model":"m","choices":[{"text":"rl","index":0}]}`
	r429 := makeJSONResponse(429, "limit")
	r200 := makeJSONResponse(200, bodyOK)
	r429.Header.Set("Retry-After", "0")
	stub := &stubTransport{responses: []httpResponse{{resp: r429, err: nil}, {resp: r200, err: nil}}}
	cli := NewClient("k", "u")
	cli.httpClient = stub

	req := CompletionsRequest{Model: "m", Prompt: "p"}
	resp, err := cli.CreateCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "rl", resp.Choices[0].Text)
	assert.Equal(t, 2, stub.calls)
}

func TestCreateCompletion_Fail400(t *testing.T) {
	resp400 := makeJSONResponse(400, "bad req")
	stub := &stubTransport{responses: []httpResponse{{resp: resp400, err: nil}}}
	cli := NewClient("k", "u")
	cli.httpClient = stub

	req := CompletionsRequest{Model: "m", Prompt: "p"}
	resp, err := cli.CreateCompletion(context.Background(), req)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-200 status code: 400")
}

func TestCreateChatCompletion_Success(t *testing.T) {
	jsonBody := `{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`
	stub := &stubTransport{responses: []httpResponse{{resp: makeJSONResponse(200, jsonBody), err: nil}}}
	cli := NewClient("k", "u")
	cli.httpClient = stub

	req := ChatCompletionsRequest{Model: "m", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	resp, err := cli.CreateChatCompletion(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "hi", resp.Choices[0].Message.Content)
}

func TestCreateEmbedding_Success(t *testing.T) {
	jsonBody := `{"object":"list","data":[{"index":0,"embedding":[0.1,0.2]}],"model":"m","usage":{"prompt_tokens":1,"total_tokens":1}}`
	stub := &stubTransport{responses: []httpResponse{{resp: makeJSONResponse(200, jsonBody), err: nil}}}
	cli := NewClient("k", "u")
	cli.httpClient = stub

	req := EmbeddingRequest{Model: "m", Input: []string{"a"}}
	resp, err := cli.CreateEmbedding(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 0.1, resp.Data[0].Embedding[0])
}
