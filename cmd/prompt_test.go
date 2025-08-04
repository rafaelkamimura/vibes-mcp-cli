package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"openai-cli/internal/config"
	"openai-cli/internal/prompt"
)

// Mock implementations for testing

type MockManager struct {
	mock.Mock
}

func (m *MockManager) ListTemplates(category string) ([]prompt.Template, error) {
	args := m.Called(category)
	return args.Get(0).([]prompt.Template), args.Error(1)
}

func (m *MockManager) GetTemplate(name string) (prompt.Template, error) {
	args := m.Called(name)
	return args.Get(0).(prompt.Template), args.Error(1)
}

func (m *MockManager) GeneratePrompt(config *prompt.GenerationConfig) (*prompt.GenerationResult, error) {
	args := m.Called(config)
	return args.Get(0).(*prompt.GenerationResult), args.Error(1)
}

func (m *MockManager) DetectWorkspaceContext() (*prompt.WorkspaceContext, error) {
	args := m.Called()
	return args.Get(0).(*prompt.WorkspaceContext), args.Error(1)
}

func (m *MockManager) SuggestTemplates(context *prompt.WorkspaceContext) []prompt.TemplateSuggestion {
	args := m.Called(context)
	return args.Get(0).([]prompt.TemplateSuggestion)
}

func (m *MockManager) ValidateTemplate(name string) (bool, []string) {
	args := m.Called(name)
	return args.Bool(0), args.Get(1).([]string)
}

func (m *MockManager) ValidateAllTemplates() (*prompt.ValidationReport, error) {
	args := m.Called()
	return args.Get(0).(*prompt.ValidationReport), args.Error(1)
}

func (m *MockManager) GetConfig() *prompt.Config {
	args := m.Called()
	return args.Get(0).(*prompt.Config)
}

func (m *MockManager) SetConfig(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockManager) GetConfigValue(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockManager) GetHistory(limit int, filter string) ([]prompt.HistoryEntry, error) {
	args := m.Called(limit, filter)
	return args.Get(0).([]prompt.HistoryEntry), args.Error(1)
}

func (m *MockManager) RecordGeneration(entry *prompt.HistoryEntry) error {
	args := m.Called(entry)
	return args.Error(0)
}

func (m *MockManager) CopyToClipboard(content string) error {
	args := m.Called(content)
	return args.Error(0)
}

func (m *MockManager) SaveToFile(content, filePath string) error {
	args := m.Called(content, filePath)
	return args.Error(0)
}

func (m *MockManager) UseContext7(result *prompt.GenerationResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *MockManager) TriggerBeastmode(result *prompt.GenerationResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *MockManager) CreateTemplateInteractive(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockManager) CreateTemplateFromFile(name, filePath string) error {
	args := m.Called(name, filePath)
	return args.Error(0)
}

func (m *MockManager) UpdateTemplate(name string, interactive, validate bool) error {
	args := m.Called(name, interactive, validate)
	return args.Error(0)
}

func (m *MockManager) DeleteTemplate(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// Test fixtures

func createTestTemplate(name, category string) prompt.Template {
	return prompt.Template{
		Name:        name,
		Category:    category,
		Description: fmt.Sprintf("Test template for %s", name),
		Content:     fmt.Sprintf("# %s\n\nTest content for {{.param}}", name),
		Parameters: []prompt.TemplateParameter{
			{
				Name:        "param",
				Description: "Test parameter",
				Type:        "string",
				Required:    true,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestGenerationResult(templateName string) *prompt.GenerationResult {
	return &prompt.GenerationResult{
		Content:     fmt.Sprintf("Generated content for %s", templateName),
		Template:    createTestTemplate(templateName, "general"),
		Parameters:  map[string]string{"param": "test-value"},
		GeneratedAt: time.Now(),
		ValidationStatus: prompt.ValidationStatus{
			Valid: true,
			Score: 85,
		},
		WordCount: 10,
		CharCount: 50,
	}
}

func createTestWorkspaceContext() *prompt.WorkspaceContext {
	return &prompt.WorkspaceContext{
		WorkingDirectory: "/test/workspace",
		Repository:       "test-repo",
		Language:         "go",
		Framework:        "gin",
		AvailableLanguages: []string{"go", "javascript"},
		RecentFiles:      []string{"main.go", "config.go"},
		GitBranch:        "main",
		LastModified:     time.Now(),
	}
}

// Test utilities

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

func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func setupTestCommand() (*cobra.Command, *MockManager) {
	// Reset global variables
	promptCategory = ""
	promptRepo = ""
	promptLanguage = ""
	promptFramework = ""
	promptComponent = ""
	promptSeverity = ""
	promptPriority = ""
	promptInteractive = false
	promptAutoDetect = false
	promptValidate = true
	promptOutput = ""
	promptClipboard = false
	promptStdout = true
	promptSendToClaude = false
	promptUseContext7 = false
	promptBeastmode = false
	promptTemplatePath = ""
	promptForce = false

	// Setup test config and logger
	cfg = &config.Config{
		APIKey:   "test-key",
		BaseURL:  "https://api.test.com",
		Provider: "test",
		Model:    "test-model",
	}
	logger = zaptest.NewLogger(&testing.T{})

	// Create a fresh command for each test
	cmd := &cobra.Command{
		Use: "prompt",
	}

	return cmd, &MockManager{}
}

// Test Cases

func TestPromptCommand_DefaultBehavior(t *testing.T) {
	cmd, mockManager := setupTestCommand()
	
	// Mock list templates call
	templates := []prompt.Template{
		createTestTemplate("test-template", "general"),
	}
	mockManager.On("ListTemplates", "").Return(templates, nil)

	// Test default behavior (should list templates)
	err := runPromptList(cmd, []string{})
	assert.NoError(t, err)
	mockManager.AssertExpectations(t)
}

func TestPromptListCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		templates   []prompt.Template
		setupMock   func(*MockManager)
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name: "list all templates",
			args: []string{},
			templates: []prompt.Template{
				createTestTemplate("template1", "general"),
				createTestTemplate("template2", "languages"),
			},
			setupMock: func(m *MockManager) {
				m.On("ListTemplates", "").Return([]prompt.Template{
					createTestTemplate("template1", "general"),
					createTestTemplate("template2", "languages"),
				}, nil)
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Available Prompt Templates")
				assert.Contains(t, output, "template1")
				assert.Contains(t, output, "template2")
			},
		},
		{
			name: "list templates by category",
			args: []string{"general"},
			setupMock: func(m *MockManager) {
				m.On("ListTemplates", "general").Return([]prompt.Template{
					createTestTemplate("general-template", "general"),
				}, nil)
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Category: general")
				assert.Contains(t, output, "general-template")
			},
		},
		{
			name: "no templates found",
			args: []string{"nonexistent"},
			setupMock: func(m *MockManager) {
				m.On("ListTemplates", "nonexistent").Return([]prompt.Template{}, nil)
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "No templates found")
				assert.Contains(t, output, "nonexistent")
			},
		},
		{
			name: "manager error",
			args: []string{},
			setupMock: func(m *MockManager) {
				m.On("ListTemplates", "").Return([]prompt.Template{}, fmt.Errorf("manager error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			tt.setupMock(mockManager)

			output := captureOutput(func() {
				err := runPromptList(&cobra.Command{}, tt.args)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})

			if tt.validate != nil {
				tt.validate(t, output)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPromptShowCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupMock   func(*MockManager)
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name: "show existing template",
			args: []string{"test-template"},
			setupMock: func(m *MockManager) {
				template := createTestTemplate("test-template", "general")
				template.Language = "go"
				template.Framework = "gin"
				template.Examples = []string{"example usage"}
				m.On("GetTemplate", "test-template").Return(template, nil)
				m.On("ValidateTemplate", "test-template").Return(true, []string{})
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Template: test-template")
				assert.Contains(t, output, "Category: general")
				assert.Contains(t, output, "Language: go")
				assert.Contains(t, output, "Framework: gin")
				assert.Contains(t, output, "Required Parameters:")
				assert.Contains(t, output, "Usage Examples:")
				assert.Contains(t, output, "Template validation: Passed")
			},
		},
		{
			name: "show non-existent template",
			args: []string{"nonexistent"},
			setupMock: func(m *MockManager) {
				m.On("GetTemplate", "nonexistent").Return(prompt.Template{}, fmt.Errorf("template not found"))
			},
			expectError: true,
		},
		{
			name: "show template with validation issues",
			args: []string{"invalid-template"},
			setupMock: func(m *MockManager) {
				template := createTestTemplate("invalid-template", "general")
				m.On("GetTemplate", "invalid-template").Return(template, nil)
				m.On("ValidateTemplate", "invalid-template").Return(false, []string{"missing description", "invalid parameter"})
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Validation Issues:")
				assert.Contains(t, output, "missing description")
				assert.Contains(t, output, "invalid parameter")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			tt.setupMock(mockManager)

			output := captureOutput(func() {
				err := runPromptShow(&cobra.Command{}, tt.args)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})

			if tt.validate != nil {
				tt.validate(t, output)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPromptGenerateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]interface{}
		setupMock   func(*MockManager)
		expectError bool
		validate    func(*testing.T, string)
	}{
		{
			name: "generate prompt with parameters",
			args: []string{"test-template"},
			flags: map[string]interface{}{
				"repo":     "test-repo",
				"language": "go",
				"task":     "user authentication",
			},
			setupMock: func(m *MockManager) {
				result := createTestGenerationResult("test-template")
				m.On("GeneratePrompt", mock.MatchedBy(func(config *prompt.GenerationConfig) bool {
					return config.TemplateName == "test-template" &&
						config.Parameters["repo"] == "test-repo" &&
						config.Parameters["language"] == "go"
				})).Return(result, nil)
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Generated content for test-template")
			},
		},
		{
			name: "generate with auto-detect",
			args: []string{"test-template"},
			flags: map[string]interface{}{
				"auto-detect": true,
			},
			setupMock: func(m *MockManager) {
				context := createTestWorkspaceContext()
				m.On("DetectWorkspaceContext").Return(context, nil)
				
				result := createTestGenerationResult("test-template")
				m.On("GeneratePrompt", mock.MatchedBy(func(config *prompt.GenerationConfig) bool {
					return config.TemplateName == "test-template" && config.Context != nil
				})).Return(result, nil)
			},
			expectError: false,
		},
		{
			name: "generate with clipboard output",
			args: []string{"test-template"},
			flags: map[string]interface{}{
				"clipboard": true,
			},
			setupMock: func(m *MockManager) {
				result := createTestGenerationResult("test-template")
				m.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)
				m.On("CopyToClipboard", result.Content).Return(nil)
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Copied to clipboard")
			},
		},
		{
			name: "generate with file output",
			args: []string{"test-template"},
			flags: map[string]interface{}{
				"output": "/tmp/test-output.txt",
			},
			setupMock: func(m *MockManager) {
				result := createTestGenerationResult("test-template")
				m.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)
				m.On("SaveToFile", result.Content, "/tmp/test-output.txt").Return(nil)
			},
			expectError: false,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Saved to /tmp/test-output.txt")
			},
		},
		{
			name: "generate with AI integration",
			args: []string{"test-template"},
			flags: map[string]interface{}{
				"send-to-claude": true,
			},
			setupMock: func(m *MockManager) {
				result := createTestGenerationResult("test-template")
				m.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)
			},
			expectError: false,
		},
		{
			name: "generation error",
			args: []string{"nonexistent-template"},
			setupMock: func(m *MockManager) {
				m.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(&prompt.GenerationResult{}, fmt.Errorf("template not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			// Set flags
			if repo, ok := tt.flags["repo"]; ok {
				promptRepo = repo.(string)
			}
			if language, ok := tt.flags["language"]; ok {
				promptLanguage = language.(string)
			}
			if autoDetect, ok := tt.flags["auto-detect"]; ok {
				promptAutoDetect = autoDetect.(bool)
			}
			if clipboard, ok := tt.flags["clipboard"]; ok {
				promptClipboard = clipboard.(bool)
			}
			if output, ok := tt.flags["output"]; ok {
				promptOutput = output.(string)
			}
			if sendToClaude, ok := tt.flags["send-to-claude"]; ok {
				promptSendToClaude = sendToClaude.(bool)
			}

			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			tt.setupMock(mockManager)

			output := captureOutput(func() {
				err := runPromptGenerate(&cobra.Command{}, tt.args)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})

			if tt.validate != nil {
				tt.validate(t, output)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestPromptValidateCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setupMock func(*MockManager)
		validate  func(*testing.T, string)
	}{
		{
			name: "validate specific template - passed",
			args: []string{"test-template"},
			setupMock: func(m *MockManager) {
				m.On("ValidateTemplate", "test-template").Return(true, []string{})
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Template validation: PASSED")
			},
		},
		{
			name: "validate specific template - failed",
			args: []string{"invalid-template"},
			setupMock: func(m *MockManager) {
				m.On("ValidateTemplate", "invalid-template").Return(false, []string{"missing description", "invalid content"})
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Template validation: FAILED")
				assert.Contains(t, output, "missing description")
				assert.Contains(t, output, "invalid content")
			},
		},
		{
			name: "validate all templates",
			args: []string{},
			setupMock: func(m *MockManager) {
				report := &prompt.ValidationReport{
					Total:        3,
					Valid:        2,
					Invalid:      1,
					AverageScore: 85,
					Issues: map[string][]string{
						"bad-template": {"bad content"},
					},
					GeneratedAt: time.Now(),
				}
				m.On("ValidateAllTemplates").Return(report, nil)
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Template Validation Report")
				assert.Contains(t, output, "Total Templates: 3")
				assert.Contains(t, output, "Valid Templates: 2")
				assert.Contains(t, output, "Invalid Templates: 1")
				assert.Contains(t, output, "Average Score: 85/100")
				assert.Contains(t, output, "bad-template")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			tt.setupMock(mockManager)

			output := captureOutput(func() {
				err := runPromptValidate(&cobra.Command{}, tt.args)
				assert.NoError(t, err)
			})

			tt.validate(t, output)
			mockManager.AssertExpectations(t)
		})
	}
}

func TestPromptWorkspaceStatusCommand(t *testing.T) {
	_, mockManager := setupTestCommand()
	
	context := createTestWorkspaceContext()
	context.AvailableLanguages = []string{"go", "javascript", "python"}
	context.RecentFiles = []string{"main.go", "config.go", "test.go"}
	
	suggestions := []prompt.TemplateSuggestion{
		{Name: "go-service", Reason: "matches go language", Relevance: 0.8},
		{Name: "api-endpoint", Reason: "recent API development", Relevance: 0.7},
	}

	mockManager.On("DetectWorkspaceContext").Return(context, nil)
	mockManager.On("SuggestTemplates", context).Return(suggestions)

	// Mock NewManager to return our mock
	originalNewManager := prompt.NewManager
	prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
		return mockManager, nil
	}
	defer func() { prompt.NewManager = originalNewManager }()

	output := captureOutput(func() {
		err := runPromptWorkspaceStatus(&cobra.Command{}, []string{})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Workspace Context Status")
	assert.Contains(t, output, "test-repo")
	assert.Contains(t, output, "Language: go")
	assert.Contains(t, output, "Framework: gin")
	assert.Contains(t, output, "Available Languages: go, javascript, python")
	assert.Contains(t, output, "Recent Activity:")
	assert.Contains(t, output, "main.go")
	assert.Contains(t, output, "Suggested Templates:")
	assert.Contains(t, output, "go-service - matches go language")

	mockManager.AssertExpectations(t)
}

func TestPromptCreateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		interactive bool
		templatePath string
		setupMock   func(*MockManager)
		expectError bool
	}{
		{
			name:        "create interactive template",
			args:        []string{"new-template"},
			interactive: true,
			setupMock: func(m *MockManager) {
				m.On("CreateTemplateInteractive", "new-template").Return(nil)
			},
			expectError: false,
		},
		{
			name:         "create from file",
			args:         []string{"new-template"},
			templatePath: "/path/to/template.yaml",
			setupMock: func(m *MockManager) {
				m.On("CreateTemplateFromFile", "new-template", "/path/to/template.yaml").Return(nil)
			},
			expectError: false,
		},
		{
			name:        "create without options",
			args:        []string{"new-template"},
			expectError: true,
		},
		{
			name:        "create interactive error",
			args:        []string{"new-template"},
			interactive: true,
			setupMock: func(m *MockManager) {
				m.On("CreateTemplateInteractive", "new-template").Return(fmt.Errorf("creation failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			promptInteractive = tt.interactive
			promptTemplatePath = tt.templatePath

			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			if tt.setupMock != nil {
				tt.setupMock(mockManager)
			}

			err := runPromptCreate(&cobra.Command{}, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.setupMock != nil {
				mockManager.AssertExpectations(t)
			}
		})
	}
}

func TestPromptDeleteCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		force     bool
		input     string
		setupMock func(*MockManager)
		expectError bool
	}{
		{
			name:  "delete with force",
			args:  []string{"test-template"},
			force: true,
			setupMock: func(m *MockManager) {
				m.On("DeleteTemplate", "test-template").Return(nil)
			},
			expectError: false,
		},
		{
			name:  "delete with confirmation - yes",
			args:  []string{"test-template"},
			force: false,
			input: "y\n",
			setupMock: func(m *MockManager) {
				m.On("DeleteTemplate", "test-template").Return(nil)
			},
			expectError: false,
		},
		{
			name:  "delete with confirmation - no",
			args:  []string{"test-template"},
			force: false,
			input: "n\n",
			expectError: false, // Should not error, just cancel
		},
		{
			name:  "delete error",
			args:  []string{"nonexistent"},
			force: true,
			setupMock: func(m *MockManager) {
				m.On("DeleteTemplate", "nonexistent").Return(fmt.Errorf("template not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			promptForce = tt.force

			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			if tt.setupMock != nil {
				tt.setupMock(mockManager)
			}

			// Mock stdin for confirmation input
			if tt.input != "" {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				go func() {
					defer w.Close()
					w.Write([]byte(tt.input))
				}()
				defer func() { os.Stdin = oldStdin }()
			}

			err := runPromptDelete(&cobra.Command{}, tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.setupMock != nil {
				mockManager.AssertExpectations(t)
			}
		})
	}
}

func TestPromptHistoryCommand(t *testing.T) {
	_, mockManager := setupTestCommand()
	
	history := []prompt.HistoryEntry{
		{
			ID:           "1",
			Template:     "test-template",
			Repository:   "test-repo",
			Language:     "go",
			Parameters:   map[string]string{"task": "test"},
			OutputMethod: "clipboard",
			AITool:       "claude",
			Success:      true,
			Timestamp:    time.Now(),
			WordCount:    100,
		},
		{
			ID:        "2",
			Template:  "another-template",
			Repository: "another-repo",
			Language:  "python",
			Success:   false,
			Timestamp: time.Now().Add(-time.Hour),
			WordCount: 50,
		},
	}

	mockManager.On("GetHistory", 20, "").Return(history, nil)

	// Mock NewManager to return our mock
	originalNewManager := prompt.NewManager
	prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
		return mockManager, nil
	}
	defer func() { prompt.NewManager = originalNewManager }()

	output := captureOutput(func() {
		err := runPromptHistory(&cobra.Command{}, []string{})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Prompt Generation History")
	assert.Contains(t, output, "test-template")
	assert.Contains(t, output, "another-template")
	assert.Contains(t, output, "Repository: test-repo")
	assert.Contains(t, output, "Language: go")
	assert.Contains(t, output, "Output: clipboard")
	assert.Contains(t, output, "AI Tool: claude")

	mockManager.AssertExpectations(t)
}

func TestPromptConfigCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setupMock func(*MockManager)
		validate  func(*testing.T, string)
	}{
		{
			name: "show config",
			args: []string{},
			setupMock: func(m *MockManager) {
				config := &prompt.Config{
					DefaultRepository:  "test-repo",
					PreferredLanguage:  "go",
					PreferredFramework: "gin",
					AutoClipboard:      true,
					AutoValidate:       true,
					PreferredAITool:    "claude",
				}
				m.On("GetConfig").Return(config)
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Prompt Configuration")
				assert.Contains(t, output, "Default Repository: test-repo")
				assert.Contains(t, output, "Preferred Language: go")
				assert.Contains(t, output, "Auto Clipboard: true")
			},
		},
		{
			name: "set config value",
			args: []string{"set", "preferred-language=python"},
			setupMock: func(m *MockManager) {
				m.On("SetConfig", "preferred-language", "python").Return(nil)
			},
		},
		{
			name: "get config value",
			args: []string{"get", "preferred-language"},
			setupMock: func(m *MockManager) {
				m.On("GetConfigValue", "preferred-language").Return("go")
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "preferred-language = go")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockManager := setupTestCommand()
			
			// Mock NewManager to return our mock
			originalNewManager := prompt.NewManager
			prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
				return mockManager, nil
			}
			defer func() { prompt.NewManager = originalNewManager }()

			tt.setupMock(mockManager)

			output := captureOutput(func() {
				err := runPromptConfig(&cobra.Command{}, tt.args)
				assert.NoError(t, err)
			})

			if tt.validate != nil {
				tt.validate(t, output)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

// Integration tests for flag parsing and validation

func TestPromptCommand_FlagValidation(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		flags     []string
		expectError bool
		errorMsg  string
	}{
		{
			name:    "valid generate flags",
			command: "generate",
			flags:   []string{"--repo", "test-repo", "--language", "go", "--interactive"},
		},
		{
			name:    "valid list flags",
			command: "list",
			flags:   []string{"--category", "general", "--validate"},
		},
		{
			name:    "invalid output combination",
			command: "generate",
			flags:   []string{"--output", "/tmp/test", "--clipboard", "--stdout=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test flag parsing by creating commands and checking if they parse correctly
			cmd := promptCmd
			cmd.SetArgs(append([]string{tt.command, "test-template"}, tt.flags...))
			
			err := cmd.ParseFlags(append([]string{tt.command, "test-template"}, tt.flags...))
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Error handling tests

func TestPromptCommand_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func() prompt.Manager
		command   string
		args      []string
		expectError bool
	}{
		{
			name: "manager initialization error",
			setupMock: func() prompt.Manager {
				originalNewManager := prompt.NewManager
				prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
					return nil, fmt.Errorf("initialization failed")
				}
				defer func() { prompt.NewManager = originalNewManager }()
				return nil
			},
			command:     "list",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestCommand()
			
			if tt.setupMock != nil {
				tt.setupMock()
			}

			var err error
			switch tt.command {
			case "list":
				err = runPromptList(&cobra.Command{}, tt.args)
			case "show":
				err = runPromptShow(&cobra.Command{}, tt.args)
			case "generate":
				err = runPromptGenerate(&cobra.Command{}, tt.args)
			}

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Performance tests

func BenchmarkPromptList(b *testing.B) {
	_, mockManager := setupTestCommand()
	
	templates := make([]prompt.Template, 100)
	for i := 0; i < 100; i++ {
		templates[i] = createTestTemplate(fmt.Sprintf("template-%d", i), "general")
	}
	
	mockManager.On("ListTemplates", "").Return(templates, nil)

	// Mock NewManager to return our mock
	originalNewManager := prompt.NewManager
	prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
		return mockManager, nil
	}
	defer func() { prompt.NewManager = originalNewManager }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runPromptList(&cobra.Command{}, []string{})
	}
}

func BenchmarkPromptGenerate(b *testing.B) {
	_, mockManager := setupTestCommand()
	
	result := createTestGenerationResult("benchmark-template")
	mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)

	// Mock NewManager to return our mock
	originalNewManager := prompt.NewManager
	prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
		return mockManager, nil
	}
	defer func() { prompt.NewManager = originalNewManager }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runPromptGenerate(&cobra.Command{}, []string{"benchmark-template"})
	}
}

// Concurrent access tests

func TestPromptCommand_ConcurrentAccess(t *testing.T) {
	_, mockManager := setupTestCommand()
	
	result := createTestGenerationResult("concurrent-template")
	mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil).Times(10)

	// Mock NewManager to return our mock
	originalNewManager := prompt.NewManager
	prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
		return mockManager, nil
	}
	defer func() { prompt.NewManager = originalNewManager }()

	// Run multiple prompt generations concurrently
	const numGoroutines = 10
	errCh := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			err := runPromptGenerate(&cobra.Command{}, []string{"concurrent-template"})
			errCh <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errCh
		assert.NoError(t, err)
	}

	mockManager.AssertExpectations(t)
}

// Memory usage tests

func TestPromptCommand_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	_, mockManager := setupTestCommand()
	
	// Create large templates to test memory handling
	largeTemplates := make([]prompt.Template, 1000)
	for i := 0; i < 1000; i++ {
		template := createTestTemplate(fmt.Sprintf("large-template-%d", i), "general")
		template.Content = strings.Repeat("Large content ", 1000) // ~13KB per template
		largeTemplates[i] = template
	}
	
	mockManager.On("ListTemplates", "").Return(largeTemplates, nil)

	// Mock NewManager to return our mock
	originalNewManager := prompt.NewManager
	prompt.NewManager = func(cfg *config.Config, logger *zap.Logger) (prompt.Manager, error) {
		return mockManager, nil
	}
	defer func() { prompt.NewManager = originalNewManager }()

	// Test that large template lists don't cause memory issues
	err := runPromptList(&cobra.Command{}, []string{})
	assert.NoError(t, err)

	mockManager.AssertExpectations(t)
}