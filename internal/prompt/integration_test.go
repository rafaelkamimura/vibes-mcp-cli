package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"openai-cli/internal/config"
)

// Mock implementations for integration testing

type MockTemplateParser struct {
	mock.Mock
}

func (m *MockTemplateParser) LoadTemplate(filePath string) (*Template, error) {
	args := m.Called(filePath)
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockTemplateParser) SaveTemplate(template *Template, filePath string) error {
	args := m.Called(template, filePath)
	return args.Error(0)
}

func (m *MockTemplateParser) ParseContent(content, templateName string) (*Template, error) {
	args := m.Called(content, templateName)
	return args.Get(0).(*Template), args.Error(1)
}

func (m *MockTemplateParser) ValidateStructure(template *Template) error {
	args := m.Called(template)
	return args.Error(0)
}

func (m *MockTemplateParser) ListTemplateFiles(directory string) ([]string, error) {
	args := m.Called(directory)
	return args.Get(0).([]string), args.Error(1)
}

type MockWorkspaceDetector struct {
	mock.Mock
}

func (m *MockWorkspaceDetector) DetectContext(ctx context.Context) (*WorkspaceContext, error) {
	args := m.Called(ctx)
	return args.Get(0).(*WorkspaceContext), args.Error(1)
}

func (m *MockWorkspaceDetector) DetectLanguage(directory string) (string, error) {
	args := m.Called(directory)
	return args.String(0), args.Error(1)
}

func (m *MockWorkspaceDetector) DetectFramework(directory, language string) (string, error) {
	args := m.Called(directory, language)
	return args.String(0), args.Error(1)
}

func (m *MockWorkspaceDetector) GetRecentFiles(directory string, limit int) ([]string, error) {
	args := m.Called(directory, limit)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockWorkspaceDetector) GetGitStatus(directory string) (string, string, error) {
	args := m.Called(directory)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockWorkspaceDetector) GetDependencies(directory, language string) ([]Dependency, error) {
	args := m.Called(directory, language)
	return args.Get(0).([]Dependency), args.Error(1)
}

func (m *MockWorkspaceDetector) GetProjectStructure(directory string) ([]string, error) {
	args := m.Called(directory)
	return args.Get(0).([]string), args.Error(1)
}

type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) Generate(ctx context.Context, config *GenerationConfig) (*GenerationResult, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*GenerationResult), args.Error(1)
}

func (m *MockGenerator) FillTemplate(template *Template, parameters map[string]string) (string, error) {
	args := m.Called(template, parameters)
	return args.String(0), args.Error(1)
}

func (m *MockGenerator) ProcessInteractive(template *Template, context *WorkspaceContext) (map[string]string, error) {
	args := m.Called(template, context)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockGenerator) FormatOutput(content, format string) (string, error) {
	args := m.Called(content, format)
	return args.String(0), args.Error(1)
}

func (m *MockGenerator) CalculateStats(content string) (int, int) {
	args := m.Called(content)
	return args.Int(0), args.Int(1)
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) ValidateTemplate(template *Template) (bool, []string, error) {
	args := m.Called(template)
	return args.Bool(0), args.Get(1).([]string), args.Error(2)
}

func (m *MockValidator) ValidateParameters(template *Template, parameters map[string]string) (bool, []string) {
	args := m.Called(template, parameters)
	return args.Bool(0), args.Get(1).([]string)
}

func (m *MockValidator) ValidateContent(content string) (int, []string, []string) {
	args := m.Called(content)
	return args.Int(0), args.Get(1).([]string), args.Get(2).([]string)
}

func (m *MockValidator) ValidateStructure(template *Template) []string {
	args := m.Called(template)
	return args.Get(0).([]string)
}

func (m *MockValidator) GetQualityScore(template *Template) int {
	args := m.Called(template)
	return args.Int(0)
}

type MockHistoryTracker struct {
	mock.Mock
}

func (m *MockHistoryTracker) Record(entry *HistoryEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

func (m *MockHistoryTracker) GetHistory(limit int, filter string) ([]HistoryEntry, error) {
	args := m.Called(limit, filter)
	return args.Get(0).([]HistoryEntry), args.Error(1)
}

func (m *MockHistoryTracker) GetStats() (*HistoryStats, error) {
	args := m.Called()
	return args.Get(0).(*HistoryStats), args.Error(1)
}

func (m *MockHistoryTracker) Cleanup(olderThan time.Duration) error {
	args := m.Called(olderThan)
	return args.Error(0)
}

type MockIntegrator struct {
	mock.Mock
}

func (m *MockIntegrator) CopyToClipboard(content string) error {
	args := m.Called(content)
	return args.Error(0)
}

func (m *MockIntegrator) SaveToFile(content, filePath string) error {
	args := m.Called(content, filePath)
	return args.Error(0)
}

func (m *MockIntegrator) SendToClaude(ctx context.Context, content string) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockIntegrator) UseContext7(ctx context.Context, result *GenerationResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockIntegrator) TriggerBeastmode(ctx context.Context, result *GenerationResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockIntegrator) TestIntegration(tool string) error {
	args := m.Called(tool)
	return args.Error(0)
}

// Test fixtures for integration tests

func createTestManagerWithMocks(t *testing.T) (*ManagerImpl, *MockTemplateParser, *MockWorkspaceDetector, *MockGenerator, *MockValidator, *MockHistoryTracker, *MockIntegrator) {
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
	}
	logger := zaptest.NewLogger(t)

	mockParser := &MockTemplateParser{}
	mockDetector := &MockWorkspaceDetector{}
	mockGenerator := &MockGenerator{}
	mockValidator := &MockValidator{}
	mockHistory := &MockHistoryTracker{}
	mockIntegrator := &MockIntegrator{}

	// Create default config
	promptConfig := &Config{
		DefaultRepository:   "test-repo",
		PreferredLanguage:   "go",
		AutoClipboard:       false,
		AutoValidate:        true,
		PreferredAITool:     AIToolClaude,
		OutputFormat:        FormatMarkdown,
		HistoryLimit:        100,
		ValidationEnabled:   true,
		BackupEnabled:       true,
		IntegrationSettings: make(map[string]string),
		LastUpdated:         time.Now(),
	}

	manager := &ManagerImpl{
		cfg:               cfg,
		logger:            logger,
		config:            promptConfig,
		templateParser:    mockParser,
		workspaceDetector: mockDetector,
		generator:         mockGenerator,
		validator:         mockValidator,
		historyTracker:    mockHistory,
		integrator:        mockIntegrator,
	}

	return manager, mockParser, mockDetector, mockGenerator, mockValidator, mockHistory, mockIntegrator
}

func createComplexTestTemplate() *Template {
	return &Template{
		Name:        "complex-template",
		Category:    CategoryGeneral,
		Language:    "go",
		Framework:   "gin",
		Description: "A complex template for integration testing",
		Content: `# {{.title}}

Repository: {{.repo}}
Language: {{.language}}
Framework: {{.framework}}

## Task Description
{{.description}}

## Requirements
{{range .requirements}}
- {{.}}
{{end}}

## Implementation Plan
1. Setup project structure
2. Implement core functionality
3. Add tests
4. Documentation

## Code Examples
` + "```{{.language}}\n" + `
{{.code_example}}
` + "```" + `

## Success Criteria
- All tests pass
- Code coverage > 80%
- Documentation complete`,
		Parameters: []TemplateParameter{
			{Name: "title", Description: "Project title", Type: "string", Required: true},
			{Name: "repo", Description: "Repository name", Type: "string", Required: true},
			{Name: "language", Description: "Programming language", Type: "select", Required: true, Options: []string{"go", "python", "javascript"}},
			{Name: "framework", Description: "Framework", Type: "string", Required: false},
			{Name: "description", Description: "Task description", Type: "string", Required: true},
			{Name: "requirements", Description: "Requirements list", Type: "string", Required: false},
			{Name: "code_example", Description: "Code example", Type: "string", Required: false},
		},
		Examples: []string{
			"Generate API service implementation",
			"Create authentication system",
		},
		Tags:      []string{"api", "service", "backend"},
		Author:    "test-author",
		Version:   "1.0.0",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	}
}

func createComplexWorkspaceContext() *WorkspaceContext {
	return &WorkspaceContext{
		WorkingDirectory:   "/test/workspace",
		Repository:         "complex-project",
		Language:           "go",
		Framework:          "gin",
		AvailableLanguages: []string{"go", "javascript", "yaml"},
		RecentFiles:        []string{"main.go", "handlers.go", "models.go", "config.yaml"},
		GitBranch:          "feature/integration-tests",
		GitStatus:          "modified",
		Dependencies: []Dependency{
			{Name: "gin-gonic/gin", Version: "v1.9.1", Type: "prod", Manager: "go"},
			{Name: "stretchr/testify", Version: "v1.8.1", Type: "dev", Manager: "go"},
		},
		ProjectStructure: []string{
			"cmd/",
			"internal/",
			"pkg/",
			"test/",
			"go.mod",
			"README.md",
		},
		Environment: map[string]string{
			"GO_VERSION": "1.21",
			"CGO_ENABLED": "0",
		},
		LastModified: time.Now(),
	}
}

// Integration Tests

func TestManagerIntegration_CompleteWorkflow(t *testing.T) {
	manager, mockParser, mockDetector, mockGenerator, mockValidator, mockHistory, mockIntegrator := createTestManagerWithMocks(t)

	// Setup test template and context
	template := createComplexTestTemplate()
	context := createComplexWorkspaceContext()

	// Mock template loading
	mockParser.On("ListTemplateFiles", mock.AnythingOfType("string")).Return([]string{"/templates/complex-template.yaml"}, nil)
	mockParser.On("LoadTemplate", "/templates/complex-template.yaml").Return(template, nil)

	// Mock workspace detection
	mockDetector.On("DetectContext", mock.AnythingOfType("*context.Context")).Return(context, nil)

	// Mock generation
	generationResult := &GenerationResult{
		Content:     "Generated complex content with all parameters filled",
		Template:    *template,
		Parameters:  map[string]string{"title": "Test Project", "repo": "complex-project", "language": "go"},
		GeneratedAt: time.Now(),
		Context:     context,
		ValidationStatus: ValidationStatus{
			Valid: true,
			Score: 92,
		},
		WordCount: 150,
		CharCount: 800,
	}
	mockGenerator.On("Generate", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("*GenerationConfig")).Return(generationResult, nil)

	// Mock validation
	mockValidator.On("ValidateTemplate", template).Return(true, []string{}, nil)

	// Mock history recording
	mockHistory.On("Record", mock.AnythingOfType("*HistoryEntry")).Return(nil)

	// Mock integrations
	mockIntegrator.On("CopyToClipboard", generationResult.Content).Return(nil)
	mockIntegrator.On("SaveToFile", generationResult.Content, "/tmp/output.md").Return(nil)

	// Test complete workflow
	t.Run("list templates", func(t *testing.T) {
		templates, err := manager.ListTemplates("")
		assert.NoError(t, err)
		assert.Len(t, templates, 1)
		assert.Equal(t, "complex-template", templates[0].Name)
	})

	t.Run("get template", func(t *testing.T) {
		tmpl, err := manager.GetTemplate("complex-template")
		assert.NoError(t, err)
		assert.Equal(t, "complex-template", tmpl.Name)
		assert.Equal(t, CategoryGeneral, tmpl.Category)
		assert.Len(t, tmpl.Parameters, 7)
	})

	t.Run("detect workspace context", func(t *testing.T) {
		ctx, err := manager.DetectWorkspaceContext()
		assert.NoError(t, err)
		assert.Equal(t, "complex-project", ctx.Repository)
		assert.Equal(t, "go", ctx.Language)
		assert.Equal(t, "gin", ctx.Framework)
		assert.Len(t, ctx.Dependencies, 2)
	})

	t.Run("generate prompt", func(t *testing.T) {
		config := &GenerationConfig{
			TemplateName: "complex-template",
			Interactive:  false,
			Context:      context,
			Parameters: map[string]string{
				"title":       "Test Project",
				"repo":        "complex-project",
				"language":    "go",
				"framework":   "gin",
				"description": "Integration test project",
			},
			Validate: true,
		}

		result, err := manager.GeneratePrompt(config)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Generated complex content with all parameters filled", result.Content)
		assert.True(t, result.ValidationStatus.Valid)
		assert.Equal(t, 92, result.ValidationStatus.Score)
		assert.Greater(t, result.WordCount, 0)
		assert.Greater(t, result.CharCount, 0)
	})

	t.Run("validate template", func(t *testing.T) {
		valid, issues := manager.ValidateTemplate("complex-template")
		assert.True(t, valid)
		assert.Empty(t, issues)
	})

	t.Run("copy to clipboard", func(t *testing.T) {
		err := manager.CopyToClipboard("test content")
		assert.NoError(t, err)
	})

	t.Run("save to file", func(t *testing.T) {
		err := manager.SaveToFile("test content", "/tmp/output.md")
		assert.NoError(t, err)
	})

	// Verify all mocks were called as expected
	mockParser.AssertExpectations(t)
	mockDetector.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
	mockHistory.AssertExpectations(t)
	mockIntegrator.AssertExpectations(t)
}

func TestManagerIntegration_TemplateSuggestions(t *testing.T) {
	manager, mockParser, _, _, _, _, _ := createTestManagerWithMocks(t)

	context := createComplexWorkspaceContext()

	// Create templates with different relevance scores
	templates := []Template{
		{
			Name:        "go-api-service",
			Category:    CategoryLanguages,
			Language:    "go",
			Framework:   "gin",
			Description: "Go API service template",
		},
		{
			Name:        "python-web-app",
			Category:    CategoryLanguages,
			Language:    "python",
			Framework:   "flask",
			Description: "Python web application template",
		},
		{
			Name:        "generic-dockerfile",
			Category:    CategoryGeneral,
			Description: "Generic Dockerfile template",
		},
		{
			Name:        "complex-project-setup",
			Category:    CategoryWorkspace,
			Description: "Setup for complex-project repository",
		},
	}

	mockParser.On("ListTemplateFiles", mock.AnythingOfType("string")).Return([]string{
		"/templates/go-api-service.yaml",
		"/templates/python-web-app.yaml",
		"/templates/generic-dockerfile.yaml",
		"/templates/complex-project-setup.yaml",
	}, nil)

	for _, template := range templates {
		mockParser.On("LoadTemplate", mock.AnythingOfType("string")).Return(&template, nil).Once()
	}

	suggestions := manager.SuggestTemplates(context)

	// Should suggest templates in order of relevance
	assert.NotEmpty(t, suggestions)
	
	// First suggestion should be the Go + Gin template (highest relevance)
	assert.Equal(t, "go-api-service", suggestions[0].Name)
	assert.Contains(t, suggestions[0].Reason, "go")
	assert.Contains(t, suggestions[0].Reason, "gin")
	assert.Greater(t, suggestions[0].Relevance, 0.5)

	// Should include project-specific template
	projectSpecific := false
	for _, suggestion := range suggestions {
		if suggestion.Name == "complex-project-setup" {
			projectSpecific = true
			assert.Contains(t, suggestion.Reason, "complex-project")
		}
	}
	assert.True(t, projectSpecific, "Should suggest project-specific templates")

	mockParser.AssertExpectations(t)
}

func TestManagerIntegration_ConfigurationPersistence(t *testing.T) {
	manager, _, _, _, _, _, _ := createTestManagerWithMocks(t)

	// Test configuration operations
	t.Run("get initial config", func(t *testing.T) {
		config := manager.GetConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "test-repo", config.DefaultRepository)
		assert.Equal(t, "go", config.PreferredLanguage)
		assert.True(t, config.AutoValidate)
	})

	t.Run("set configuration values", func(t *testing.T) {
		err := manager.SetConfig("preferred-language", "python")
		assert.NoError(t, err)

		value := manager.GetConfigValue("preferred-language")
		assert.Equal(t, "python", value)

		err = manager.SetConfig("auto-clipboard", "true")
		assert.NoError(t, err)

		value = manager.GetConfigValue("auto-clipboard")
		assert.Equal(t, "true", value)
	})

	t.Run("invalid configuration key", func(t *testing.T) {
		err := manager.SetConfig("invalid-key", "value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown config key")
	})

	t.Run("get all config", func(t *testing.T) {
		config := manager.GetConfig()
		assert.Equal(t, "python", config.PreferredLanguage)
		assert.True(t, config.AutoClipboard)
	})
}

func TestManagerIntegration_HistoryManagement(t *testing.T) {
	manager, _, _, _, _, mockHistory, _ := createTestManagerWithMocks(t)

	// Create test history entries
	entries := []HistoryEntry{
		{
			ID:           "1",
			Template:     "go-service",
			Repository:   "test-repo",
			Language:     "go",
			Framework:    "gin",
			Parameters:   map[string]string{"service": "auth"},
			OutputMethod: "clipboard",
			AITool:       "claude",
			Success:      true,
			Timestamp:    time.Now().Add(-2 * time.Hour),
			Duration:     time.Second * 5,
			WordCount:    120,
		},
		{
			ID:           "2",
			Template:     "python-api",
			Repository:   "python-project",
			Language:     "python",
			Framework:    "fastapi",
			Parameters:   map[string]string{"endpoint": "users"},
			OutputMethod: "file",
			Success:      true,
			Timestamp:    time.Now().Add(-time.Hour),
			Duration:     time.Second * 3,
			WordCount:    85,
		},
		{
			ID:           "3",
			Template:     "go-service",
			Repository:   "test-repo",
			Language:     "go",
			Success:      false,
			ErrorMessage: "template validation failed",
			Timestamp:    time.Now().Add(-30 * time.Minute),
			Duration:     time.Second * 1,
		},
	}

	mockHistory.On("GetHistory", 10, "").Return(entries, nil)
	mockHistory.On("GetHistory", 10, "go").Return([]HistoryEntry{entries[0], entries[2]}, nil)
	mockHistory.On("GetStats").Return(&HistoryStats{
		TotalGenerations:  3,
		SuccessRate:      0.67,
		AverageWordCount: 102,
		TopTemplates: []TemplateUsage{
			{Name: "go-service", Count: 2},
			{Name: "python-api", Count: 1},
		},
		TopLanguages: []LanguageUsage{
			{Language: "go", Count: 2},
			{Language: "python", Count: 1},
		},
		TopRepositories: []RepositoryUsage{
			{Repository: "test-repo", Count: 2},
			{Repository: "python-project", Count: 1},
		},
	}, nil)

	t.Run("get all history", func(t *testing.T) {
		history, err := manager.GetHistory(10, "")
		assert.NoError(t, err)
		assert.Len(t, history, 3)
		assert.Equal(t, "go-service", history[0].Template)
		assert.Equal(t, "python-api", history[1].Template)
		assert.True(t, history[1].Success)
		assert.False(t, history[2].Success)
	})

	t.Run("get filtered history", func(t *testing.T) {
		history, err := manager.GetHistory(10, "go")
		assert.NoError(t, err)
		assert.Len(t, history, 2)
		for _, entry := range history {
			assert.Equal(t, "go-service", entry.Template)
		}
	})

	mockHistory.AssertExpectations(t)
}

func TestManagerIntegration_ErrorHandling(t *testing.T) {
	manager, mockParser, mockDetector, mockGenerator, mockValidator, _, _ := createTestManagerWithMocks(t)

	t.Run("template not found", func(t *testing.T) {
		mockParser.On("ListTemplateFiles", mock.AnythingOfType("string")).Return([]string{}, nil)

		_, err := manager.GetTemplate("nonexistent-template")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("workspace detection failure", func(t *testing.T) {
		mockDetector.On("DetectContext", mock.AnythingOfType("*context.Context")).Return(&WorkspaceContext{}, fmt.Errorf("git not available"))

		_, err := manager.DetectWorkspaceContext()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detect workspace context")
	})

	t.Run("generation failure", func(t *testing.T) {
		template := createComplexTestTemplate()
		mockParser.On("LoadTemplate", mock.AnythingOfType("string")).Return(template, nil)
		mockGenerator.On("Generate", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("*GenerationConfig")).Return(&GenerationResult{}, fmt.Errorf("generation failed"))

		config := &GenerationConfig{
			TemplateName: "complex-template",
			Parameters:   map[string]string{},
		}

		_, err := manager.GeneratePrompt(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate prompt")
	})

	t.Run("validation failure", func(t *testing.T) {
		template := createComplexTestTemplate()
		mockValidator.On("ValidateTemplate", template).Return(false, []string{"missing required field"}, fmt.Errorf("validation error"))

		valid, issues := manager.ValidateTemplate("complex-template")
		assert.False(t, valid)
		assert.Contains(t, issues[0], "Validation error")
	})

	mockParser.AssertExpectations(t)
	mockDetector.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestManagerIntegration_ConcurrentOperations(t *testing.T) {
	manager, mockParser, mockDetector, mockGenerator, _, _, _ := createTestManagerWithMocks(t)

	template := createComplexTestTemplate()
	context := createComplexWorkspaceContext()

	// Setup mocks for concurrent access
	mockParser.On("LoadTemplate", mock.AnythingOfType("string")).Return(template, nil).Times(10)
	mockDetector.On("DetectContext", mock.AnythingOfType("*context.Context")).Return(context, nil).Times(5)

	generationResult := &GenerationResult{
		Content:     "Generated content",
		Template:    *template,
		Parameters:  map[string]string{},
		GeneratedAt: time.Now(),
		ValidationStatus: ValidationStatus{Valid: true, Score: 85},
		WordCount: 50,
		CharCount: 200,
	}
	mockGenerator.On("Generate", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("*GenerationConfig")).Return(generationResult, nil).Times(5)

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Test concurrent template retrieval
	wg.Add(numGoroutines / 2)
	for i := 0; i < numGoroutines/2; i++ {
		go func() {
			defer wg.Done()
			_, err := manager.GetTemplate("complex-template")
			errors <- err
		}()
	}

	// Test concurrent context detection
	wg.Add(numGoroutines / 4)
	for i := 0; i < numGoroutines/4; i++ {
		go func() {
			defer wg.Done()
			_, err := manager.DetectWorkspaceContext()
			errors <- err
		}()
	}

	// Test concurrent prompt generation
	wg.Add(numGoroutines / 4)
	for i := 0; i < numGoroutines/4; i++ {
		go func() {
			defer wg.Done()
			config := &GenerationConfig{
				TemplateName: "complex-template",
				Parameters:   map[string]string{"repo": "test"},
			}
			_, err := manager.GeneratePrompt(config)
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		assert.NoError(t, err)
	}

	mockParser.AssertExpectations(t)
	mockDetector.AssertExpectations(t)
	mockGenerator.AssertExpectations(t)
}

func TestManagerIntegration_LargeDatasets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	manager, mockParser, _, _, mockValidator, _, _ := createTestManagerWithMocks(t)

	// Create a large number of templates
	const numTemplates = 1000
	templates := make([]Template, numTemplates)
	templateFiles := make([]string, numTemplates)

	for i := 0; i < numTemplates; i++ {
		templates[i] = Template{
			Name:        fmt.Sprintf("template-%d", i),
			Category:    CategoryGeneral,
			Description: fmt.Sprintf("Test template %d", i),
			Content:     fmt.Sprintf("Template content %d with {{.param}}", i),
			Parameters: []TemplateParameter{
				{Name: "param", Type: "string", Required: true},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		templateFiles[i] = fmt.Sprintf("/templates/template-%d.yaml", i)
	}

	// Mock large template loading
	mockParser.On("ListTemplateFiles", mock.AnythingOfType("string")).Return(templateFiles, nil)
	for i, template := range templates {
		mockParser.On("LoadTemplate", templateFiles[i]).Return(&template, nil)
	}

	// Test validation of all templates
	validationReport := &ValidationReport{
		Total:        numTemplates,
		Valid:        numTemplates - 10, // 10 invalid templates
		Invalid:      10,
		AverageScore: 82,
		Issues: map[string][]string{
			"template-0": {"minor issue"},
		},
		GeneratedAt: time.Now(),
	}

	for i := 0; i < numTemplates; i++ {
		if i < 10 {
			mockValidator.On("ValidateTemplate", &templates[i]).Return(false, []string{"issue"}, nil)
			mockValidator.On("GetQualityScore", &templates[i]).Return(60)
		} else {
			mockValidator.On("ValidateTemplate", &templates[i]).Return(true, []string{}, nil)
			mockValidator.On("GetQualityScore", &templates[i]).Return(85)
		}
	}

	t.Run("list large number of templates", func(t *testing.T) {
		start := time.Now()
		templateList, err := manager.ListTemplates("")
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, templateList, numTemplates)
		assert.Less(t, duration, time.Second*5) // Should complete within 5 seconds
	})

	t.Run("validate all templates", func(t *testing.T) {
		start := time.Now()
		report, err := manager.ValidateAllTemplates()
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, numTemplates, report.Total)
		assert.Equal(t, numTemplates-10, report.Valid)
		assert.Equal(t, 10, report.Invalid)
		assert.Less(t, duration, time.Second*10) // Should complete within 10 seconds
	})

	mockParser.AssertExpectations(t)
	mockValidator.AssertExpectations(t)
}

func TestManagerIntegration_FileSystemOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	customDir := filepath.Join(tempDir, "custom")

	err := os.MkdirAll(templatesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(customDir, 0755)
	require.NoError(t, err)

	// Create real manager with real file system operations
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
	}
	logger := zaptest.NewLogger(t)

	manager, err := NewManager(cfg, logger)
	require.NoError(t, err)

	managerImpl := manager.(*ManagerImpl)
	managerImpl.config.TemplateDirectories = []string{templatesDir}
	managerImpl.config.CustomTemplatesPath = customDir

	// Create test template files
	templateContent := `name: file-test-template
category: general
description: Template for file system testing
content: |
  # File Test Template
  
  Repository: {{.repo}}
  Test content for file operations.
parameters:
  - name: repo
    description: Repository name
    type: string
    required: true
version: "1.0.0"
`

	templateFile := filepath.Join(templatesDir, "file-test-template.yaml")
	err = os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	t.Run("load templates from file system", func(t *testing.T) {
		templates, err := manager.ListTemplates("")
		assert.NoError(t, err)
		assert.Len(t, templates, 1)
		assert.Equal(t, "file-test-template", templates[0].Name)
		assert.Equal(t, "general", templates[0].Category)
	})

	t.Run("get template from file system", func(t *testing.T) {
		template, err := manager.GetTemplate("file-test-template")
		assert.NoError(t, err)
		assert.Equal(t, "file-test-template", template.Name)
		assert.Contains(t, template.Content, "File Test Template")
		assert.Len(t, template.Parameters, 1)
		assert.Equal(t, "repo", template.Parameters[0].Name)
	})

	t.Run("create custom template", func(t *testing.T) {
		customTemplateFile := filepath.Join(templatesDir, "source-template.yaml")
		err = os.WriteFile(customTemplateFile, []byte(templateContent), 0644)
		require.NoError(t, err)

		err = manager.CreateTemplateFromFile("custom-template", customTemplateFile)
		assert.NoError(t, err)

		// Verify custom template was created
		customPath := filepath.Join(customDir, "custom-template.yaml")
		_, err = os.Stat(customPath)
		assert.NoError(t, err)
		
		// Verify it can be loaded
		customTemplate, err := manager.GetTemplate("custom-template")
		assert.NoError(t, err)
		assert.Equal(t, "custom-template", customTemplate.Name)
	})

	t.Run("delete custom template", func(t *testing.T) {
		// Ensure template exists
		customPath := filepath.Join(customDir, "custom-template.yaml")
		_, err = os.Stat(customPath)
		require.NoError(t, err)

		// Delete template
		err = manager.DeleteTemplate("custom-template")
		assert.NoError(t, err)

		// Verify template was deleted
		_, err = os.Stat(customPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("backup on delete", func(t *testing.T) {
		// Create another custom template
		err = manager.CreateTemplateFromFile("backup-test", templateFile)
		require.NoError(t, err)

		customPath := filepath.Join(customDir, "backup-test.yaml")
		_, err = os.Stat(customPath)
		require.NoError(t, err)

		// Delete should create backup
		err = manager.DeleteTemplate("backup-test")
		assert.NoError(t, err)

		// Check for backup file
		backupPattern := filepath.Join(customDir, "backup-test.yaml.backup.*")
		matches, err := filepath.Glob(backupPattern)
		assert.NoError(t, err)
		assert.Len(t, matches, 1)
	})
}

// Benchmarks for integration testing

func BenchmarkManagerIntegration_ListTemplates(b *testing.B) {
	manager, mockParser, _, _, _, _, _ := createTestManagerWithMocks(&testing.T{})

	// Create mock templates
	templates := make([]Template, 100)
	templateFiles := make([]string, 100)
	for i := 0; i < 100; i++ {
		templates[i] = Template{
			Name:        fmt.Sprintf("bench-template-%d", i),
			Category:    CategoryGeneral,
			Description: fmt.Sprintf("Benchmark template %d", i),
		}
		templateFiles[i] = fmt.Sprintf("/templates/bench-template-%d.yaml", i)
	}

	mockParser.On("ListTemplateFiles", mock.AnythingOfType("string")).Return(templateFiles, nil)
	for i, template := range templates {
		mockParser.On("LoadTemplate", templateFiles[i]).Return(&template, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ListTemplates("")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManagerIntegration_GeneratePrompt(b *testing.B) {
	manager, _, _, mockGenerator, _, mockHistory, _ := createTestManagerWithMocks(&testing.T{})

	template := createComplexTestTemplate()
	result := &GenerationResult{
		Content:     "Benchmark generated content",
		Template:    *template,
		Parameters:  map[string]string{"repo": "bench-repo"},
		GeneratedAt: time.Now(),
		ValidationStatus: ValidationStatus{Valid: true, Score: 85},
		WordCount: 50,
		CharCount: 200,
	}

	mockGenerator.On("Generate", mock.AnythingOfType("*context.Context"), mock.AnythingOfType("*GenerationConfig")).Return(result, nil)
	mockHistory.On("Record", mock.AnythingOfType("*HistoryEntry")).Return(nil)

	config := &GenerationConfig{
		TemplateName: "complex-template",
		Parameters:   map[string]string{"repo": "bench-repo"},
		Validate:     false, // Skip validation for benchmarking
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GeneratePrompt(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManagerIntegration_WorkspaceDetection(b *testing.B) {
	manager, _, mockDetector, _, _, _, _ := createTestManagerWithMocks(&testing.T{})

	context := createComplexWorkspaceContext()
	mockDetector.On("DetectContext", mock.AnythingOfType("*context.Context")).Return(context, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.DetectWorkspaceContext()
		if err != nil {
			b.Fatal(err)
		}
	}
}