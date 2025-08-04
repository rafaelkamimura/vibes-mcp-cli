package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/prompt"
)

// MockPromptManager implements the prompt.Manager interface for testing
type MockPromptManager struct {
	templates map[string]prompt.Template
	config    *prompt.Config
	history   []prompt.HistoryEntry
}

func NewMockPromptManager() *MockPromptManager {
	return &MockPromptManager{
		templates: make(map[string]prompt.Template),
		config: &prompt.Config{
			DefaultRepository:  "test-repo",
			PreferredLanguage:  "go",
			PreferredFramework: "gin",
		},
		history: []prompt.HistoryEntry{},
	}
}

// Implement prompt.Manager interface methods
func (m *MockPromptManager) ListTemplates(category string) ([]prompt.Template, error) {
	var templates []prompt.Template
	for _, template := range m.templates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}
	return templates, nil
}

func (m *MockPromptManager) GetTemplate(name string) (prompt.Template, error) {
	template, exists := m.templates[name]
	if !exists {
		return prompt.Template{}, fmt.Errorf("template not found: %s", name)
	}
	return template, nil
}

func (m *MockPromptManager) CreateTemplateInteractive(name string) error {
	return fmt.Errorf("interactive creation not supported in mock")
}

func (m *MockPromptManager) CreateTemplateFromFile(name, filePath string) error {
	return fmt.Errorf("file creation not supported in mock")
}

func (m *MockPromptManager) UpdateTemplate(name string, interactive, validate bool) error {
	if _, exists := m.templates[name]; !exists {
		return fmt.Errorf("template not found: %s", name)
	}
	return nil
}

func (m *MockPromptManager) DeleteTemplate(name string) error {
	delete(m.templates, name)
	return nil
}

func (m *MockPromptManager) GeneratePrompt(config *prompt.GenerationConfig) (*prompt.GenerationResult, error) {
	template, err := m.GetTemplate(config.TemplateName)
	if err != nil {
		return nil, err
	}
	
	// Simple mock generation
	content := template.Content
	for key, value := range config.Parameters {
		content = strings.ReplaceAll(content, fmt.Sprintf("{{%s}}", key), value)
	}
	
	return &prompt.GenerationResult{
		Content:     content,
		Template:    template,
		Parameters:  config.Parameters,
		GeneratedAt: time.Now(),
		Context:     config.Context,
		ValidationStatus: prompt.ValidationStatus{
			Valid: true,
			Score: 95,
		},
		WordCount: len(strings.Fields(content)),
		CharCount: len(content),
	}, nil
}

func (m *MockPromptManager) DetectWorkspaceContext() (*prompt.WorkspaceContext, error) {
	return &prompt.WorkspaceContext{
		WorkingDirectory:   "/test/workspace",
		Repository:         "test-repo",
		Language:           "go",
		Framework:          "gin",
		AvailableLanguages: []string{"go", "javascript"},
		RecentFiles:        []string{"main.go", "README.md"},
		GitBranch:          "main",
		GitStatus:          "clean",
		Dependencies: []prompt.Dependency{
			{Name: "gin", Version: "1.9.0", Type: "prod", Manager: "go"},
		},
		LastModified: time.Now(),
	}, nil
}

func (m *MockPromptManager) SuggestTemplates(context *prompt.WorkspaceContext) []prompt.TemplateSuggestion {
	return []prompt.TemplateSuggestion{
		{
			Name:      "golang-function",
			Reason:    "Go language detected",
			Relevance: 0.9,
			Category:  "languages",
		},
		{
			Name:      "api-endpoint",
			Reason:    "Gin framework detected",
			Relevance: 0.8,
			Category:  "workflows",
		},
	}
}

func (m *MockPromptManager) ValidateTemplate(name string) (bool, []string) {
	template, exists := m.templates[name]
	if !exists {
		return false, []string{"Template not found"}
	}
	
	if len(template.Content) == 0 {
		return false, []string{"Template content is empty"}
	}
	
	return true, []string{}
}

func (m *MockPromptManager) ValidateAllTemplates() (*prompt.ValidationReport, error) {
	return &prompt.ValidationReport{
		Total:        len(m.templates),
		Valid:        len(m.templates),
		Invalid:      0,
		AverageScore: 95,
		Issues:       make(map[string][]string),
		GeneratedAt:  time.Now(),
	}, nil
}

func (m *MockPromptManager) GetConfig() *prompt.Config {
	return m.config
}

func (m *MockPromptManager) SetConfig(key, value string) error {
	// Simple mock implementation
	return nil
}

func (m *MockPromptManager) GetConfigValue(key string) string {
	return ""
}

func (m *MockPromptManager) GetHistory(limit int, filter string) ([]prompt.HistoryEntry, error) {
	return m.history, nil
}

func (m *MockPromptManager) RecordGeneration(entry *prompt.HistoryEntry) error {
	m.history = append(m.history, *entry)
	return nil
}

func (m *MockPromptManager) CopyToClipboard(content string) error {
	return nil
}

func (m *MockPromptManager) SaveToFile(content, filePath string) error {
	return nil
}

func (m *MockPromptManager) UseContext7(result *prompt.GenerationResult) error {
	return nil
}

func (m *MockPromptManager) TriggerBeastmode(result *prompt.GenerationResult) error {
	return nil
}

// Helper method to add templates for testing
func (m *MockPromptManager) AddTemplate(template prompt.Template) {
	m.templates[template.Name] = template
}

// Test setup helper
func setupTestData(manager *MockPromptManager) {
	manager.AddTemplate(prompt.Template{
		Name:        "golang-function",
		Category:    "languages",
		Description: "Generate a Go function",
		Content:     "func {{function_name}}() {\n    // TODO: Implement {{description}}\n}",
		Parameters: []prompt.TemplateParameter{
			{
				Name:        "function_name",
				Description: "Name of the function",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "description",
				Description: "Function description",
				Type:        "string",
				Required:    false,
				Default:     "function logic",
			},
		},
		Tags:      []string{"go", "function"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	
	manager.AddTemplate(prompt.Template{
		Name:        "api-endpoint",
		Category:    "workflows",
		Description: "Create an API endpoint",
		Content:     "// {{method}} {{path}}\nfunc {{handler_name}}(c *gin.Context) {\n    // Handle {{description}}\n}",
		Parameters: []prompt.TemplateParameter{
			{
				Name:     "method",
				Type:     "select",
				Required: true,
				Options:  []string{"GET", "POST", "PUT", "DELETE"},
			},
			{
				Name:     "path",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "handler_name",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "description",
				Type:     "string",
				Required: false,
				Default:  "request",
			},
		},
		Tags:      []string{"api", "gin", "endpoint"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

// Tests for PromptResourceManager
func TestPromptResourceManager_ListResources(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	resourceManager := NewPromptResourceManager(manager, logger)
	
	resources, err := resourceManager.ListResources(context.Background())
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}
	
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
	
	// Check resource properties
	for _, resource := range resources {
		if resource.URI == "" {
			t.Error("Resource URI should not be empty")
		}
		if resource.Name == "" {
			t.Error("Resource name should not be empty")
		}
		if resource.MimeType != "text/plain" {
			t.Errorf("Expected mime type 'text/plain', got '%s'", resource.MimeType)
		}
	}
}

func TestPromptResourceManager_GetResource(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	resourceManager := NewPromptResourceManager(manager, logger)
	
	uri := "prompt://templates/golang-function"
	resource, err := resourceManager.GetResource(context.Background(), uri)
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}
	
	if resource.Name != "golang-function" {
		t.Errorf("Expected name 'golang-function', got '%s'", resource.Name)
	}
	
	if resource.Category != "languages" {
		t.Errorf("Expected category 'languages', got '%s'", resource.Category)
	}
	
	if len(resource.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(resource.Parameters))
	}
}

func TestPromptResourceManager_SearchResources(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	resourceManager := NewPromptResourceManager(manager, logger)
	
	// Search by language
	resources, err := resourceManager.SearchResources(context.Background(), "go")
	if err != nil {
		t.Fatalf("Failed to search resources: %v", err)
	}
	
	if len(resources) == 0 {
		t.Error("Expected to find resources with 'go' search")
	}
	
	// Search by category
	resources, err = resourceManager.SearchResources(context.Background(), "workflow")
	if err != nil {
		t.Fatalf("Failed to search resources: %v", err)
	}
	
	found := false
	for _, resource := range resources {
		if resource.Category == "workflows" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected to find workflow category resources")
	}
}

// Tests for PromptToolManager
func TestPromptToolManager_GetTools(t *testing.T) {
	manager := NewMockPromptManager()
	logger := zap.NewNop()
	toolManager := NewPromptToolManager(manager, logger)
	
	tools := toolManager.GetTools()
	
	if len(tools) == 0 {
		t.Error("Expected to get tools, got none")
	}
	
	// Check for expected tools
	expectedTools := []string{
		"generate_prompt",
		"validate_template",
		"detect_context",
		"suggest_templates",
		"get_history",
	}
	
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' not found", expected)
		}
	}
}

func TestPromptToolManager_CallTool_GeneratePrompt(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	toolManager := NewPromptToolManager(manager, logger)
	
	args := map[string]interface{}{
		"template_name": "golang-function",
		"parameters": map[string]interface{}{
			"function_name": "calculateSum",
			"description":   "calculate sum of two numbers",
		},
	}
	
	result, err := toolManager.CallTool(context.Background(), "generate_prompt", args)
	if err != nil {
		t.Fatalf("Failed to call generate_prompt tool: %v", err)
	}
	
	genResult, ok := result.(PromptGenerateResult)
	if !ok {
		t.Fatalf("Expected PromptGenerateResult, got %T", result)
	}
	
	if !genResult.Success {
		t.Errorf("Expected successful generation, got error: %s", genResult.Error)
	}
	
	if !strings.Contains(genResult.Content, "calculateSum") {
		t.Error("Generated content should contain function name")
	}
	
	if !strings.Contains(genResult.Content, "calculate sum of two numbers") {
		t.Error("Generated content should contain description")
	}
}

func TestPromptToolManager_CallTool_DetectContext(t *testing.T) {
	manager := NewMockPromptManager()
	logger := zap.NewNop()
	toolManager := NewPromptToolManager(manager, logger)
	
	result, err := toolManager.CallTool(context.Background(), "detect_context", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to call detect_context tool: %v", err)
	}
	
	context, ok := result.(*WorkspaceContext)
	if !ok {
		t.Fatalf("Expected *WorkspaceContext, got %T", result)
	}
	
	if context.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", context.Language)
	}
	
	if context.Repository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", context.Repository)
	}
}

func TestPromptToolManager_CallTool_SuggestTemplates(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	toolManager := NewPromptToolManager(manager, logger)
	
	args := map[string]interface{}{
		"language": "go",
		"max_results": 5,
	}
	
	result, err := toolManager.CallTool(context.Background(), "suggest_templates", args)
	if err != nil {
		t.Fatalf("Failed to call suggest_templates tool: %v", err)
	}
	
	suggestResult, ok := result.(PromptSuggestResult)
	if !ok {
		t.Fatalf("Expected PromptSuggestResult, got %T", result)
	}
	
	if !suggestResult.Success {
		t.Errorf("Expected successful suggestion, got error: %s", suggestResult.Error)
	}
	
	if len(suggestResult.Suggestions) == 0 {
		t.Error("Expected suggestions, got none")
	}
	
	// Check that suggestions are relevant
	for _, suggestion := range suggestResult.Suggestions {
		if suggestion.Relevance <= 0 || suggestion.Relevance > 1 {
			t.Errorf("Invalid relevance score: %f", suggestion.Relevance)
		}
	}
}

// Tests for PromptMCPHandler
func TestPromptMCPHandler_HandleRequest(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	handler := NewPromptMCPHandler(manager, logger)
	
	// Test resources/list
	result, err := handler.HandleRequest(context.Background(), "resources/list", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Failed to handle resources/list: %v", err)
	}
	
	resourcesResult, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}
	
	if resourcesResult["total"].(int) == 0 {
		t.Error("Expected resources to be returned")
	}
}

func TestPromptMCPHandler_ServeHTTP(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	handler := NewPromptMCPHandler(manager, logger)
	
	// Create a test request
	reqBody := `{
		"jsonrpc": "2.0",
		"id": "test-1",
		"method": "resources/list",
		"params": {}
	}`
	
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response RPCResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", response.ID)
	}
	
	if response.Error != nil {
		t.Errorf("Expected no error, got: %v", response.Error)
	}
}

// Tests for PromptMCPServer
func TestPromptMCPServer_Configuration(t *testing.T) {
	manager := NewMockPromptManager()
	logger := zap.NewNop()
	server := NewPromptMCPServer(manager, logger)
	
	server.SetConfiguration("0.0.0.0", 9999, false, true)
	
	if server.host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", server.host)
	}
	
	if server.port != 9999 {
		t.Errorf("Expected port 9999, got %d", server.port)
	}
	
	if server.enableWS {
		t.Error("Expected WebSocket to be disabled")
	}
	
	if !server.enableAuth {
		t.Error("Expected auth to be enabled")
	}
}

func TestPromptMCPServer_HealthEndpoint(t *testing.T) {
	manager := NewMockPromptManager()
	logger := zap.NewNop()
	server := NewPromptMCPServer(manager, logger)
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	server.handleHealth(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var health map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}
	
	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", health["status"])
	}
}

func TestPromptMCPServer_MetricsEndpoint(t *testing.T) {
	manager := NewMockPromptManager()
	logger := zap.NewNop()
	server := NewPromptMCPServer(manager, logger)
	
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	server.handleMetrics(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var metrics map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&metrics); err != nil {
		t.Fatalf("Failed to decode metrics response: %v", err)
	}
	
	expectedFields := []string{"start_time", "uptime", "request_count", "error_count", "active_clients"}
	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Expected metrics field '%s' not found", field)
		}
	}
}

func TestPromptMCPServer_InfoEndpoint(t *testing.T) {
	manager := NewMockPromptManager()
	logger := zap.NewNop()
	server := NewPromptMCPServer(manager, logger)
	
	req := httptest.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()
	
	server.handleInfo(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var info map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("Failed to decode info response: %v", err)
	}
	
	if info["name"] != "Prompt MCP Server" {
		t.Errorf("Expected name 'Prompt MCP Server', got '%v'", info["name"])
	}
	
	capabilities, ok := info["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected capabilities to be a map")
	}
	
	if _, exists := capabilities["resources"]; !exists {
		t.Error("Expected capabilities to include resources")
	}
	
	if _, exists := capabilities["tools"]; !exists {
		t.Error("Expected capabilities to include tools")
	}
}

// Benchmark tests
func BenchmarkPromptResourceManager_ListResources(b *testing.B) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	resourceManager := NewPromptResourceManager(manager, logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resourceManager.ListResources(context.Background())
		if err != nil {
			b.Fatalf("Failed to list resources: %v", err)
		}
	}
}

func BenchmarkPromptToolManager_CallTool(b *testing.B) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	toolManager := NewPromptToolManager(manager, logger)
	
	args := map[string]interface{}{
		"template_name": "golang-function",
		"parameters": map[string]interface{}{
			"function_name": "testFunc",
			"description":   "test function",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := toolManager.CallTool(context.Background(), "generate_prompt", args)
		if err != nil {
			b.Fatalf("Failed to call tool: %v", err)
		}
	}
}

// Integration test
func TestMCPIntegration_EndToEnd(t *testing.T) {
	manager := NewMockPromptManager()
	setupTestData(manager)
	
	logger := zap.NewNop()
	handler := NewPromptMCPHandler(manager, logger)
	
	// Test direct handler calls (simulating what the HTTP server would do)
	
	// Test listing resources
	result, err := handler.HandleRequest(context.Background(), "resources/list", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}
	
	resourcesResult, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}
	
	if resourcesResult["total"].(int) == 0 {
		t.Error("Expected resources to be returned")
	}
	
	// Test generating prompt via tool call
	genParams := map[string]interface{}{
		"name": "generate_prompt",
		"arguments": map[string]interface{}{
			"template_name": "golang-function",
			"parameters": map[string]interface{}{
				"function_name": "testFunction",
				"description":   "a test function",
			},
		},
	}
	
	genParamsBytes, _ := json.Marshal(genParams)
	result, err = handler.HandleRequest(context.Background(), "tools/call", genParamsBytes)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}
	
	genResult, ok := result.(PromptGenerateResult)
	if !ok {
		t.Fatalf("Expected PromptGenerateResult, got %T", result)
	}
	
	if !genResult.Success {
		t.Errorf("Expected successful generation, got error: %s", genResult.Error)
	}
	
	if !strings.Contains(genResult.Content, "testFunction") {
		t.Error("Generated content should contain function name")
	}
	
	t.Log("End-to-end integration test completed successfully")
}