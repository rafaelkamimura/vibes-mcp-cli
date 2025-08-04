package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"openai-cli/internal/client"
	"openai-cli/internal/prompt"
	"openai-cli/internal/service"
)

// Mock implementations for MCP testing

type MockPromptManager struct {
	mock.Mock
}

func (m *MockPromptManager) ListTemplates(category string) ([]prompt.Template, error) {
	args := m.Called(category)
	return args.Get(0).([]prompt.Template), args.Error(1)
}

func (m *MockPromptManager) GetTemplate(name string) (prompt.Template, error) {
	args := m.Called(name)
	return args.Get(0).(prompt.Template), args.Error(1)
}

func (m *MockPromptManager) GeneratePrompt(config *prompt.GenerationConfig) (*prompt.GenerationResult, error) {
	args := m.Called(config)
	return args.Get(0).(*prompt.GenerationResult), args.Error(1)
}

func (m *MockPromptManager) DetectWorkspaceContext() (*prompt.WorkspaceContext, error) {
	args := m.Called()
	return args.Get(0).(*prompt.WorkspaceContext), args.Error(1)
}

func (m *MockPromptManager) SuggestTemplates(context *prompt.WorkspaceContext) []prompt.TemplateSuggestion {
	args := m.Called(context)
	return args.Get(0).([]prompt.TemplateSuggestion)
}

func (m *MockPromptManager) ValidateTemplate(name string) (bool, []string) {
	args := m.Called(name)
	return args.Bool(0), args.Get(1).([]string)
}

func (m *MockPromptManager) ValidateAllTemplates() (*prompt.ValidationReport, error) {
	args := m.Called()
	return args.Get(0).(*prompt.ValidationReport), args.Error(1)
}

func (m *MockPromptManager) GetConfig() *prompt.Config {
	args := m.Called()
	return args.Get(0).(*prompt.Config)
}

func (m *MockPromptManager) SetConfig(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockPromptManager) GetConfigValue(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockPromptManager) GetHistory(limit int, filter string) ([]prompt.HistoryEntry, error) {
	args := m.Called(limit, filter)
	return args.Get(0).([]prompt.HistoryEntry), args.Error(1)
}

func (m *MockPromptManager) RecordGeneration(entry *prompt.HistoryEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

func (m *MockPromptManager) CopyToClipboard(content string) error {
	args := m.Called(content)
	return args.Error(0)
}

func (m *MockPromptManager) SaveToFile(content, filePath string) error {
	args := m.Called(content, filePath)
	return args.Error(0)
}

func (m *MockPromptManager) UseContext7(result *prompt.GenerationResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *MockPromptManager) TriggerBeastmode(result *prompt.GenerationResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *MockPromptManager) CreateTemplateInteractive(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockPromptManager) CreateTemplateFromFile(name, filePath string) error {
	args := m.Called(name, filePath)
	return args.Error(0)
}

func (m *MockPromptManager) UpdateTemplate(name string, interactive, validate bool) error {
	args := m.Called(name, interactive, validate)
	return args.Error(0)
}

func (m *MockPromptManager) DeleteTemplate(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

type MockServiceClient struct {
	mock.Mock
}

func (m *MockServiceClient) CreateCompletion(ctx context.Context, req client.CompletionsRequest) (*client.CompletionsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.CompletionsResponse), args.Error(1)
}

func (m *MockServiceClient) CreateChatCompletion(ctx context.Context, req client.ChatCompletionsRequest) (*client.ChatCompletionsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.ChatCompletionsResponse), args.Error(1)
}

func (m *MockServiceClient) CreateEmbedding(ctx context.Context, req client.EmbeddingRequest) (*client.EmbeddingResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.EmbeddingResponse), args.Error(1)
}

// Test fixtures and utilities

func createTestTemplate() prompt.Template {
	return prompt.Template{
		Name:        "mcp-test-template",
		Category:    "general",
		Language:    "go",
		Framework:   "gin",
		Description: "Template for MCP testing",
		Content: `# {{.title}}

Repository: {{.repo}}
Language: {{.language}}
Framework: {{.framework}}

## Task
{{.description}}

## Implementation
{{.implementation}}

## Tests
{{.tests}}`,
		Parameters: []prompt.TemplateParameter{
			{Name: "title", Description: "Project title", Type: "string", Required: true},
			{Name: "repo", Description: "Repository name", Type: "string", Required: true},
			{Name: "language", Description: "Programming language", Type: "string", Required: true},
			{Name: "framework", Description: "Framework", Type: "string", Required: false},
			{Name: "description", Description: "Task description", Type: "string", Required: true},
			{Name: "implementation", Description: "Implementation details", Type: "string", Required: false},
			{Name: "tests", Description: "Test requirements", Type: "string", Required: false},
		},
		Examples: []string{
			"Generate API service",
			"Create data model",
		},
		Tags:      []string{"api", "service", "testing"},
		Author:    "mcp-test",
		Version:   "1.0.0",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	}
}

func createTestWorkspaceContext() *WorkspaceContext {
	return &WorkspaceContext{
		WorkingDirectory:   "/test/mcp-workspace",
		Repository:         "mcp-test-repo",
		Language:           "go",
		Framework:          "gin",
		AvailableLanguages: []string{"go", "javascript", "python"},
		RecentFiles:        []string{"main.go", "handler.go", "model.go"},
		GitBranch:          "feature/mcp-integration",
		GitStatus:          "clean",
		Dependencies: []prompt.Dependency{
			{Name: "gin-gonic/gin", Version: "v1.9.1", Type: "prod", Manager: "go"},
			{Name: "stretchr/testify", Version: "v1.8.1", Type: "dev", Manager: "go"},
		},
		ProjectStructure: []string{
			"cmd/",
			"internal/",
			"pkg/",
			"go.mod",
			"README.md",
		},
		LastModified: time.Now(),
	}
}

func createTestGenerationResult() *prompt.GenerationResult {
	template := createTestTemplate()
	return &prompt.GenerationResult{
		Content: `# MCP Test Project

Repository: mcp-test-repo
Language: go
Framework: gin

## Task
Create MCP integration for prompt management

## Implementation
- Implement MCP protocol handlers
- Add resource management
- Create tool definitions

## Tests
- Unit tests for all handlers
- Integration tests for protocol compliance
- Performance tests for concurrent operations`,
		Template:    template,
		Parameters:  map[string]string{"title": "MCP Test Project", "repo": "mcp-test-repo", "language": "go", "framework": "gin", "description": "Create MCP integration for prompt management"},
		GeneratedAt: time.Now(),
		Context:     createTestWorkspaceContext(),
		ValidationStatus: prompt.ValidationStatus{
			Valid: true,
			Score: 92,
		},
		WordCount: 45,
		CharCount: 280,
	}
}

// MCP Resource Tests

func TestMCPResources_ListTemplates(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	require.NotNil(t, server)

	templates := []prompt.Template{
		createTestTemplate(),
		{Name: "template2", Category: "languages", Description: "Another template"},
	}

	mockManager.On("ListTemplates", "").Return(templates, nil)

	resources, err := server.ListResources(context.Background())
	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Verify resource structure
	assert.Equal(t, "template://mcp-test-template", resources[0].URI)
	assert.Equal(t, "Template: mcp-test-template", resources[0].Name)
	assert.Equal(t, "Template for MCP testing", resources[0].Description)
	assert.Equal(t, "text/markdown", resources[0].MimeType)

	mockManager.AssertExpectations(t)
}

func TestMCPResources_GetTemplate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	template := createTestTemplate()

	mockManager.On("GetTemplate", "mcp-test-template").Return(template, nil)

	resource, err := server.GetResource(context.Background(), "template://mcp-test-template")
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	assert.Equal(t, "template://mcp-test-template", resource.URI)
	assert.Contains(t, resource.Contents, template.Name)
	assert.Contains(t, resource.Contents, template.Description)
	assert.Contains(t, resource.Contents, template.Content)

	mockManager.AssertExpectations(t)
}

func TestMCPResources_GetWorkspaceContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	context := createTestWorkspaceContext()

	mockManager.On("DetectWorkspaceContext").Return(context, nil)

	resource, err := server.GetResource(context.Background(), "workspace://context")
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	assert.Equal(t, "workspace://context", resource.URI)
	assert.Contains(t, resource.Contents, context.Repository)
	assert.Contains(t, resource.Contents, context.Language)
	assert.Contains(t, resource.Contents, context.Framework)

	mockManager.AssertExpectations(t)
}

func TestMCPResources_GetHistory(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	
	historyEntries := []prompt.HistoryEntry{
		{
			ID:           "1",
			Template:     "mcp-test-template",
			Repository:   "test-repo",
			Language:     "go",
			Success:      true,
			Timestamp:    time.Now(),
			WordCount:    100,
		},
		{
			ID:        "2",
			Template:  "another-template",
			Success:   false,
			ErrorMessage: "validation failed",
			Timestamp: time.Now().Add(-time.Hour),
		},
	}

	mockManager.On("GetHistory", 10, "").Return(historyEntries, nil)

	resource, err := server.GetResource(context.Background(), "history://recent?limit=10")
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	assert.Equal(t, "history://recent?limit=10", resource.URI)
	assert.Contains(t, resource.Contents, "mcp-test-template")
	assert.Contains(t, resource.Contents, "another-template")
	assert.Contains(t, resource.Contents, "validation failed")

	mockManager.AssertExpectations(t)
}

func TestMCPResources_InvalidResource(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	_, err := server.GetResource(context.Background(), "invalid://resource")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")
}

// MCP Tool Tests

func TestMCPTools_GeneratePrompt(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	result := createTestGenerationResult()

	mockManager.On("GeneratePrompt", mock.MatchedBy(func(config *prompt.GenerationConfig) bool {
		return config.TemplateName == "mcp-test-template" &&
			config.Parameters["title"] == "Test Project" &&
			config.Parameters["repo"] == "test-repo"
	})).Return(result, nil)

	tools := server.ListTools(context.Background())
	assert.Len(t, tools, 6) // Should have 6 tools: generate, validate, suggest, create, update, delete

	// Find generate tool
	var generateTool *MCPTool
	for _, tool := range tools {
		if tool.Name == "generate_prompt" {
			generateTool = &tool
			break
		}
	}
	require.NotNil(t, generateTool)

	// Test tool execution
	args := map[string]interface{}{
		"template_name": "mcp-test-template",
		"parameters": map[string]interface{}{
			"title": "Test Project",
			"repo":  "test-repo",
		},
	}

	toolResult, err := server.ExecuteTool(context.Background(), "generate_prompt", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "MCP Test Project")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_ValidateTemplate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	mockManager.On("ValidateTemplate", "mcp-test-template").Return(true, []string{})

	args := map[string]interface{}{
		"template_name": "mcp-test-template",
	}

	toolResult, err := server.ExecuteTool(context.Background(), "validate_template", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "valid")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_ValidateTemplateWithIssues(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	issues := []string{
		"Missing required parameter",
		"Invalid content format",
	}
	mockManager.On("ValidateTemplate", "invalid-template").Return(false, issues)

	args := map[string]interface{}{
		"template_name": "invalid-template",
	}

	toolResult, err := server.ExecuteTool(context.Background(), "validate_template", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.False(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "Missing required parameter")
	assert.Contains(t, toolResult.Content, "Invalid content format")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_SuggestTemplates(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	context := createTestWorkspaceContext()

	suggestions := []prompt.TemplateSuggestion{
		{Name: "go-service", Reason: "matches go language", Relevance: 0.8, Category: "languages"},
		{Name: "gin-api", Reason: "matches gin framework", Relevance: 0.7, Category: "frameworks"},
	}

	mockManager.On("DetectWorkspaceContext").Return(context, nil)
	mockManager.On("SuggestTemplates", context).Return(suggestions)

	toolResult, err := server.ExecuteTool(context.Background(), "suggest_templates", map[string]interface{}{})
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "go-service")
	assert.Contains(t, toolResult.Content, "gin-api")
	assert.Contains(t, toolResult.Content, "matches go language")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_CreateTemplate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	mockManager.On("CreateTemplateFromFile", "new-template", "/path/to/template.yaml").Return(nil)

	args := map[string]interface{}{
		"name":      "new-template",
		"file_path": "/path/to/template.yaml",
	}

	toolResult, err := server.ExecuteTool(context.Background(), "create_template", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "created successfully")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_UpdateTemplate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	mockManager.On("UpdateTemplate", "existing-template", false, true).Return(nil)

	args := map[string]interface{}{
		"name":        "existing-template",
		"interactive": false,
		"validate":    true,
	}

	toolResult, err := server.ExecuteTool(context.Background(), "update_template", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "updated successfully")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_DeleteTemplate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	mockManager.On("DeleteTemplate", "template-to-delete").Return(nil)

	args := map[string]interface{}{
		"name": "template-to-delete",
	}

	toolResult, err := server.ExecuteTool(context.Background(), "delete_template", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)
	assert.Contains(t, toolResult.Content, "deleted successfully")

	mockManager.AssertExpectations(t)
}

func TestMCPTools_InvalidTool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	_, err := server.ExecuteTool(context.Background(), "invalid_tool", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

// MCP Protocol Compliance Tests

func TestMCPProtocol_JSONRPCCompliance(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	handler := server.GetHTTPHandler()

	// Test valid JSON-RPC request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "resources/list",
		"params":  map[string]interface{}{},
		"id":      1,
	}

	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	mockManager.On("ListTemplates", "").Return([]prompt.Template{createTestTemplate()}, nil)

	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.Equal(t, float64(1), response["id"])
	assert.NotNil(t, response["result"])

	mockManager.AssertExpectations(t)
}

func TestMCPProtocol_InvalidJSONRPC(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	handler := server.GetHTTPHandler()

	// Test invalid JSON-RPC (missing jsonrpc field)
	request := map[string]interface{}{
		"method": "resources/list",
		"params": map[string]interface{}{},
		"id":     1,
	}

	requestBody, err := json.Marshal(request)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "2.0", response["jsonrpc"])
	assert.Equal(t, float64(1), response["id"])
	assert.NotNil(t, response["error"])
}

func TestMCPProtocol_BatchRequests(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	handler := server.GetHTTPHandler()

	// Test batch request
	batchRequest := []map[string]interface{}{
		{
			"jsonrpc": "2.0",
			"method":  "resources/list",
			"params":  map[string]interface{}{},
			"id":      1,
		},
		{
			"jsonrpc": "2.0",
			"method":  "tools/list",
			"params":  map[string]interface{}{},
			"id":      2,
		},
	}

	requestBody, err := json.Marshal(batchRequest)
	require.NoError(t, err)

	mockManager.On("ListTemplates", "").Return([]prompt.Template{createTestTemplate()}, nil)

	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(requestBody)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var responses []map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &responses)
	assert.NoError(t, err)

	assert.Len(t, responses, 2)
	assert.Equal(t, float64(1), responses[0]["id"])
	assert.Equal(t, float64(2), responses[1]["id"])

	mockManager.AssertExpectations(t)
}

// MCP Client Tests

func TestMCPClient_ConnectAndDisconnect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	// Create mock server
	server := NewPromptMCPServer(mockManager, logger)
	testServer := httptest.NewServer(server.GetHTTPHandler())
	defer testServer.Close()

	// Create client
	client := NewPromptMCPClient(testServer.URL, logger)
	require.NotNil(t, client)

	// Test connection
	err := client.Connect(context.Background())
	assert.NoError(t, err)
	assert.True(t, client.IsConnected())

	// Test disconnection
	err = client.Disconnect()
	assert.NoError(t, err)
	assert.False(t, client.IsConnected())
}

func TestMCPClient_ListResources(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	// Setup server
	server := NewPromptMCPServer(mockManager, logger)
	testServer := httptest.NewServer(server.GetHTTPHandler())
	defer testServer.Close()

	templates := []prompt.Template{createTestTemplate()}
	mockManager.On("ListTemplates", "").Return(templates, nil)

	// Setup client
	client := NewPromptMCPClient(testServer.URL, logger)
	err := client.Connect(context.Background())
	require.NoError(t, err)
	defer client.Disconnect()

	// Test list resources
	resources, err := client.ListResources(context.Background())
	assert.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "template://mcp-test-template", resources[0].URI)

	mockManager.AssertExpectations(t)
}

func TestMCPClient_GetResource(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	// Setup server
	server := NewPromptMCPServer(mockManager, logger)
	testServer := httptest.NewServer(server.GetHTTPHandler())
	defer testServer.Close()

	template := createTestTemplate()
	mockManager.On("GetTemplate", "mcp-test-template").Return(template, nil)

	// Setup client
	client := NewPromptMCPClient(testServer.URL, logger)
	err := client.Connect(context.Background())
	require.NoError(t, err)
	defer client.Disconnect()

	// Test get resource
	resource, err := client.GetResource(context.Background(), "template://mcp-test-template")
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	assert.Equal(t, "template://mcp-test-template", resource.URI)
	assert.Contains(t, resource.Contents, template.Name)

	mockManager.AssertExpectations(t)
}

func TestMCPClient_ExecuteTool(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	// Setup server
	server := NewPromptMCPServer(mockManager, logger)
	testServer := httptest.NewServer(server.GetHTTPHandler())
	defer testServer.Close()

	result := createTestGenerationResult()
	mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)

	// Setup client
	client := NewPromptMCPClient(testServer.URL, logger)
	err := client.Connect(context.Background())
	require.NoError(t, err)
	defer client.Disconnect()

	// Test execute tool
	args := map[string]interface{}{
		"template_name": "mcp-test-template",
		"parameters": map[string]interface{}{
			"title": "Test Project",
			"repo":  "test-repo",
		},
	}

	toolResult, err := client.ExecuteTool(context.Background(), "generate_prompt", args)
	assert.NoError(t, err)
	assert.NotNil(t, toolResult)
	assert.True(t, toolResult.Success)

	mockManager.AssertExpectations(t)
}

// AI Integration Tests

func TestPromptAIIntegrator_SendToAssistant(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	mockService := &MockServiceClient{}

	integrator := NewPromptAIIntegrator(mockManager, logger)
	integrator.SetServiceClient(mockService)

	// Mock AI response
	aiResponse := &client.ChatCompletionsResponse{
		Choices: []client.ChatCompletionChoice{
			{
				Message: client.ChatMessage{
					Role:    "assistant",
					Content: "Here's an enhanced version of your prompt:\n\n# Enhanced Project\n\nThis is an improved prompt with better structure and clarity.",
				},
			},
		},
	}

	mockService.On("CreateChatCompletion", mock.AnythingOfType("*context.Context"), mock.MatchedBy(func(req client.ChatCompletionsRequest) bool {
		return req.Model == "claude-3" && len(req.Messages) >= 2
	})).Return(aiResponse, nil)

	// Test AI integration
	params := AIAssistantParams{
		Content:     "Original prompt content",
		Assistant:   "claude",
		Context:     createTestWorkspaceContext(),
		Temperature: 0.7,
		MaxTokens:   2000,
		SystemPrompt: "You are a helpful assistant.",
	}

	result, err := integrator.SendToAssistant(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "claude", result.Assistant)
	assert.Contains(t, result.Response, "Enhanced Project")
	assert.Greater(t, result.TokensUsed, 0)
	assert.Greater(t, result.ProcessingTime, time.Duration(0))

	mockService.AssertExpectations(t)
}

func TestPromptAIIntegrator_EnhancePrompt(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	mockService := &MockServiceClient{}

	integrator := NewPromptAIIntegrator(mockManager, logger)
	integrator.SetServiceClient(mockService)

	// Mock AI response
	aiResponse := &client.ChatCompletionsResponse{
		Choices: []client.ChatCompletionChoice{
			{
				Message: client.ChatMessage{
					Role: "assistant",
					Content: `Analysis of the prompt:

Strengths:
- Clear structure
- Good parameter usage

Suggestions:
- Add more specific examples
- Include error handling requirements

Enhanced version:
# Enhanced API Service Template

This template creates a robust API service with comprehensive error handling and monitoring.`,
				},
			},
		},
	}

	mockService.On("CreateChatCompletion", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("client.ChatCompletionsRequest")).Return(aiResponse, nil)

	result, err := integrator.EnhancePrompt(context.Background(), "Original prompt", createTestWorkspaceContext(), "claude")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Response, "Enhanced API Service Template")
	assert.NotEmpty(t, result.Suggestions)
	assert.NotEmpty(t, result.Improvements)

	mockService.AssertExpectations(t)
}

func TestPromptAIIntegrator_GetFeedback(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	mockService := &MockServiceClient{}

	integrator := NewPromptAIIntegrator(mockManager, logger)
	integrator.SetServiceClient(mockService)

	// Mock AI response
	aiResponse := &client.ChatCompletionsResponse{
		Choices: []client.ChatCompletionChoice{
			{
				Message: client.ChatMessage{
					Role: "assistant",
					Content: `Feedback on your prompt:

Clarity: 8/10 - The prompt is generally clear but could benefit from more specific examples.
Specificity: 7/10 - Good level of detail, but some requirements are vague.
Completeness: 9/10 - Covers most necessary aspects.
Actionability: 8/10 - Easy to follow and implement.

Overall: This is a well-structured prompt with room for minor improvements in specificity.`,
				},
			},
		},
	}

	mockService.On("CreateChatCompletion", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("client.ChatCompletionsRequest")).Return(aiResponse, nil)

	result, err := integrator.GetFeedback(context.Background(), "Test prompt content", "claude")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Response, "Clarity: 8/10")
	assert.Contains(t, result.Response, "Specificity: 7/10")

	mockService.AssertExpectations(t)
}

func TestPromptAIIntegrator_AnalyzePromptQuality(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	mockService := &MockServiceClient{}

	integrator := NewPromptAIIntegrator(mockManager, logger)
	integrator.SetServiceClient(mockService)

	// Mock AI response
	aiResponse := &client.ChatCompletionsResponse{
		Choices: []client.ChatCompletionChoice{
			{
				Message: client.ChatMessage{
					Role: "assistant",
					Content: `Quality Analysis:

Overall Score: 85/100
Clarity: 80/100
Specificity: 90/100
Completeness: 85/100
Actionability: 88/100

Top 3 areas for improvement:
1. Add more concrete examples
2. Clarify success criteria
3. Include error handling guidelines`,
				},
			},
		},
	}

	mockService.On("CreateChatCompletion", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("client.ChatCompletionsRequest")).Return(aiResponse, nil)

	analysis, err := integrator.AnalyzePromptQuality(context.Background(), "Test prompt for quality analysis")
	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Equal(t, 85, analysis["overall_score"])
	assert.Equal(t, 80, analysis["clarity_score"])
	assert.Equal(t, 90, analysis["specificity_score"])
	assert.Contains(t, analysis["ai_response"], "Quality Analysis")

	mockService.AssertExpectations(t)
}

func TestPromptAIIntegrator_GenerateTemplateVariations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	mockService := &MockServiceClient{}

	integrator := NewPromptAIIntegrator(mockManager, logger)
	integrator.SetServiceClient(mockService)

	template := createTestTemplate()
	mockManager.On("GetTemplate", "mcp-test-template").Return(template, nil)

	// Mock AI response with variations
	aiResponse := &client.ChatCompletionsResponse{
		Choices: []client.ChatCompletionChoice{
			{
				Message: client.ChatMessage{
					Role: "assistant",
					Content: `Here are 3 variations of your template:

1. Simplified Version
# {{.title}} - Quick Start

Project: {{.repo}}
Tech Stack: {{.language}} + {{.framework}}

Quick implementation guide for {{.description}}.

2. Detailed Version
# Comprehensive {{.title}} Implementation

## Project Overview
Repository: {{.repo}}
Primary Language: {{.language}}
Framework: {{.framework}}

## Detailed Requirements
{{.description}}

## Step-by-step Implementation
{{.implementation}}

## Testing Strategy
{{.tests}}

3. Enterprise Version
# Enterprise-Grade {{.title}}

## Business Context
Project Repository: {{.repo}}
Technology Stack: {{.language}} with {{.framework}}

## Requirements Analysis
{{.description}}

## Architecture Design
{{.implementation}}

## Quality Assurance
{{.tests}}

## Deployment Strategy
Production-ready deployment with monitoring and scaling.`,
				},
			},
		},
	}

	mockService.On("CreateChatCompletion", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("client.ChatCompletionsRequest")).Return(aiResponse, nil)

	variations, err := integrator.GenerateTemplateVariations(context.Background(), "mcp-test-template", 3)
	assert.NoError(t, err)
	assert.Len(t, variations, 3)
	assert.Contains(t, variations[0], "Simplified Version")
	assert.Contains(t, variations[1], "Detailed Version")
	assert.Contains(t, variations[2], "Enterprise Version")

	mockManager.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

func TestPromptAIIntegrator_OptimizeForContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	mockService := &MockServiceClient{}

	integrator := NewPromptAIIntegrator(mockManager, logger)
	integrator.SetServiceClient(mockService)

	// Mock AI response
	aiResponse := &client.ChatCompletionsResponse{
		Choices: []client.ChatCompletionChoice{
			{
				Message: client.ChatMessage{
					Role: "assistant",
					Content: `Optimized prompt for Go + Gin context:

# Go Gin API Service Template

## Context-Specific Optimizations
- Leverages Gin's middleware system
- Follows Go project structure conventions
- Includes testify for testing (detected in dependencies)
- Uses go.mod for dependency management

## Go-Specific Implementation
- Proper package organization
- Interface-based design
- Error handling with Go idioms
- Struct tags for JSON serialization

## Gin Framework Integration
- Route handlers with proper HTTP methods
- Middleware for authentication and logging
- Request validation and response formatting

Changes made:
1. Added Go-specific project structure
2. Included Gin middleware examples  
3. Added testify testing patterns
4. Optimized for detected dependencies`,
				},
			},
		},
	}

	mockService.On("CreateChatCompletion", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("client.ChatCompletionsRequest")).Return(aiResponse, nil)

	result, err := integrator.OptimizeForContext(context.Background(), "Generic API template", createTestWorkspaceContext())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Response, "Go Gin API Service Template")
	assert.Contains(t, result.Response, "Context-Specific Optimizations")
	assert.Contains(t, result.Response, "testify")

	mockService.AssertExpectations(t)
}

// Error handling tests

func TestMCPIntegration_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	// Test resource not found
	mockManager.On("GetTemplate", "nonexistent").Return(prompt.Template{}, fmt.Errorf("template not found"))

	_, err := server.GetResource(context.Background(), "template://nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test tool execution error
	mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(&prompt.GenerationResult{}, fmt.Errorf("generation failed"))

	args := map[string]interface{}{
		"template_name": "failing-template",
	}

	toolResult, err := server.ExecuteTool(context.Background(), "generate_prompt", args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generation failed")
	assert.Nil(t, toolResult)

	mockManager.AssertExpectations(t)
}

// Performance and concurrent access tests

func TestMCPIntegration_ConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	template := createTestTemplate()

	// Setup mocks for concurrent access
	mockManager.On("GetTemplate", "mcp-test-template").Return(template, nil).Times(10)

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Test concurrent resource access
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := server.GetResource(context.Background(), "template://mcp-test-template")
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		assert.NoError(t, err)
	}

	mockManager.AssertExpectations(t)
}

func BenchmarkMCPServer_ListResources(b *testing.B) {
	logger := zap.NewNop()
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	// Create large template set
	templates := make([]prompt.Template, 100)
	for i := 0; i < 100; i++ {
		templates[i] = prompt.Template{
			Name:        fmt.Sprintf("template-%d", i),
			Category:    "general",
			Description: fmt.Sprintf("Template %d", i),
		}
	}

	mockManager.On("ListTemplates", "").Return(templates, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.ListResources(context.Background())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMCPServer_ExecuteTool(b *testing.B) {
	logger := zap.NewNop()
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)
	result := createTestGenerationResult()

	mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)

	args := map[string]interface{}{
		"template_name": "mcp-test-template",
		"parameters":    map[string]interface{}{"title": "Test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := server.ExecuteTool(context.Background(), "generate_prompt", args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Memory usage tests

func TestMCPIntegration_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	server := NewPromptMCPServer(mockManager, logger)

	// Create large content for memory testing
	largeContent := strings.Repeat("Large template content ", 10000) // ~250KB
	largeTemplate := createTestTemplate()
	largeTemplate.Content = largeContent

	mockManager.On("GetTemplate", "large-template").Return(largeTemplate, nil).Times(100)

	// Test memory usage with large content
	for i := 0; i < 100; i++ {
		resource, err := server.GetResource(context.Background(), "template://large-template")
		assert.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Contains(t, resource.Contents, "Large template content")
	}

	mockManager.AssertExpectations(t)
}