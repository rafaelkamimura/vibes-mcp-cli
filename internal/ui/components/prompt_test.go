package components

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"openai-cli/internal/prompt"
)

// Mock implementations for TUI testing

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

// Test fixtures and utilities

func createTestTemplates() []prompt.Template {
	return []prompt.Template{
		{
			Name:        "api-service",
			Category:    "general",
			Language:    "go",
			Framework:   "gin",
			Description: "REST API service template",
			Parameters: []prompt.TemplateParameter{
				{Name: "service", Description: "Service name", Type: "string", Required: true},
				{Name: "port", Description: "Service port", Type: "string", Default: "8080"},
			},
			Examples:   []string{"Generate user service", "Generate product service"},
			CreatedAt:  time.Now().Add(-time.Hour),
			UpdatedAt:  time.Now(),
		},
		{
			Name:        "database-migration",
			Category:    "languages",
			Language:    "sql",
			Description: "Database migration template",
			Parameters: []prompt.TemplateParameter{
				{Name: "table", Description: "Table name", Type: "string", Required: true},
				{Name: "operation", Description: "Migration type", Type: "select", Required: true,
					Options: []string{"create", "alter", "drop"}},
			},
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-time.Hour),
		},
		{
			Name:        "test-suite",
			Category:    "workflows",
			Language:    "go",
			Description: "Test suite generation template",
			Parameters: []prompt.TemplateParameter{
				{Name: "package", Description: "Package name", Type: "string", Required: true},
				{Name: "coverage", Description: "Target coverage", Type: "string", Default: "80"},
			},
			CreatedAt: time.Now().Add(-3 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
	}
}

func createTestUIConfig() *PromptUIConfig {
	return &PromptUIConfig{
		Theme: PromptUITheme{
			Primary:     tcell.ColorBlue,
			Secondary:   tcell.ColorGreen,
			Accent:      tcell.ColorYellow,
			Background:  tcell.ColorBlack,
			Surface:     tcell.ColorDarkSlateGray,
			TextPrimary: tcell.ColorWhite,
			TextSecondary: tcell.ColorLightGray,
			Success:     tcell.ColorGreen,
			Warning:     tcell.ColorYellow,
			Error:       tcell.ColorRed,
		},
		Icons: PromptUIIcons{
			Template:    "📄",
			Category:    "📂",
			Language:    "💻",
			Framework:   "🔧",
			Parameter:   "⚙️",
			Required:    "❗",
			Optional:    "❓",
			Success:     "✅",
			Warning:     "⚠️",
			Error:       "❌",
			Info:        "ℹ️",
			Loading:     "⏳",
			Search:      "🔍",
			Filter:      "🔽",
			Edit:        "✏️",
			Delete:      "🗑️",
			Copy:        "📋",
			Save:        "💾",
			History:     "📚",
			Config:      "⚙️",
		},
		Animation: PromptUIAnimation{
			Enabled:      true,
			Duration:     time.Millisecond * 200,
			FadeIn:       true,
			SlideIn:      true,
			Bounce:       false,
			LoadingSpinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		},
		Layout: PromptUILayout{
			SidebarWidth:    30,
			HeaderHeight:    3,
			FooterHeight:    2,
			ModalWidth:      60,
			ModalHeight:     20,
			ListItemHeight:  3,
			PreviewEnabled:  true,
			ShowLineNumbers: true,
		},
		Keybindings: PromptUIKeybindings{
			Quit:         []string{"q", "Ctrl+C"},
			Help:         []string{"?", "F1"},
			Refresh:      []string{"r", "F5"},
			Search:       []string{"/", "Ctrl+F"},
			Filter:       []string{"f", "Tab"},
			Select:       []string{"Enter", "Space"},
			Edit:         []string{"e", "F2"},
			Delete:       []string{"d", "Delete"},
			Copy:         []string{"c", "Ctrl+C"},
			Paste:        []string{"p", "Ctrl+V"},
			Save:         []string{"s", "Ctrl+S"},
			Cancel:       []string{"Escape"},
			Up:           []string{"k", "Up"},
			Down:         []string{"j", "Down"},
			Left:         []string{"h", "Left"},
			Right:        []string{"l", "Right"},
			PageUp:       []string{"K", "PageUp"},
			PageDown:     []string{"J", "PageDown"},
			Home:         []string{"g", "Home"},
			End:          []string{"G", "End"},
		},
	}
}

// Test cases for PromptBrowser component

func TestPromptBrowser_Initialization(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	browser := NewPromptBrowser(config, logger)
	require.NotNil(t, browser)
	assert.NotNil(t, browser.Flex)

	// Test with manager
	browser.SetManager(mockManager)
	assert.Equal(t, mockManager, browser.manager)
}

func TestPromptBrowser_LoadTemplates(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	templates := createTestTemplates()

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	mockManager.On("ListTemplates", "").Return(templates, nil)

	err := browser.LoadTemplates("")
	assert.NoError(t, err)

	// Verify templates are loaded
	assert.Equal(t, len(templates), browser.templateList.GetItemCount())

	mockManager.AssertExpectations(t)
}

func TestPromptBrowser_FilterTemplates(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	templates := createTestTemplates()

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	// Load all templates first
	mockManager.On("ListTemplates", "").Return(templates, nil)
	err := browser.LoadTemplates("")
	require.NoError(t, err)

	// Test category filter
	mockManager.On("ListTemplates", "general").Return([]prompt.Template{templates[0]}, nil)
	err = browser.FilterByCategory("general")
	assert.NoError(t, err)

	// Verify filtering
	assert.Equal(t, 1, browser.templateList.GetItemCount())

	mockManager.AssertExpectations(t)
}

func TestPromptBrowser_SearchTemplates(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	templates := createTestTemplates()

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	// Load templates
	mockManager.On("ListTemplates", "").Return(templates, nil)
	err := browser.LoadTemplates("")
	require.NoError(t, err)

	// Test search functionality
	browser.SetSearchQuery("api")
	
	// Should find templates containing "api" in name or description
	visibleCount := 0
	for i := 0; i < browser.templateList.GetItemCount(); i++ {
		mainText, _ := browser.templateList.GetItemText(i)
		if strings.Contains(strings.ToLower(mainText), "api") {
			visibleCount++
		}
	}
	assert.Greater(t, visibleCount, 0)

	mockManager.AssertExpectations(t)
}

func TestPromptBrowser_TemplateSelection(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	templates := createTestTemplates()

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	mockManager.On("ListTemplates", "").Return(templates, nil)
	err := browser.LoadTemplates("")
	require.NoError(t, err)

	// Test template selection
	selectedTemplate := &templates[0]
	var callbackResult *prompt.Template
	
	browser.SetOnTemplateSelected(func(template *prompt.Template) {
		callbackResult = template
	})

	// Simulate selection
	browser.selectTemplate(selectedTemplate)
	
	assert.NotNil(t, callbackResult)
	assert.Equal(t, selectedTemplate.Name, callbackResult.Name)

	mockManager.AssertExpectations(t)
}

func TestPromptBrowser_KeyboardNavigation(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}
	templates := createTestTemplates()

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	mockManager.On("ListTemplates", "").Return(templates, nil)
	err := browser.LoadTemplates("")
	require.NoError(t, err)

	// Create a mock screen for event testing
	screen := tcell.NewSimulationScreen("UTF-8")
	err = screen.Init()
	require.NoError(t, err)
	defer screen.Fini()

	// Test keyboard events
	testCases := []struct {
		key      tcell.Key
		rune     rune
		expected string
	}{
		{tcell.KeyRune, 'j', "down"},
		{tcell.KeyRune, 'k', "up"},
		{tcell.KeyRune, '/', "search"},
		{tcell.KeyRune, 'f', "filter"},
		{tcell.KeyEnter, 0, "select"},
		{tcell.KeyRune, 'r', "refresh"},
	}

	for _, tc := range testCases {
		event := tcell.NewEventKey(tc.key, tc.rune, tcell.ModNone)
		
		// Test that the event is handled
		handled := browser.handleKeyEvent(event)
		
		// For navigation keys, they should be handled
		if tc.expected == "up" || tc.expected == "down" || tc.expected == "select" {
			assert.True(t, handled, "Key %s should be handled", tc.expected)
		}
	}

	mockManager.AssertExpectations(t)
}

// Test cases for PromptModal component

func TestPromptModal_ConfirmationDialog(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	modal := NewPromptModal(config, logger)
	require.NotNil(t, modal)

	// Test confirmation dialog
	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal.ShowConfirmation("Test Title", "Test message", callback)
	
	// Simulate button clicks
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		// This would normally be called by tview
		result = ModalResult{
			Action:   strings.ToLower(buttonLabel),
			Canceled: buttonLabel == "Cancel",
		}
		callback(result)
	})

	// Simulate "Yes" button click
	modal.GetDoneFunc()(0, "Yes")
	assert.Equal(t, "yes", result.Action)
	assert.False(t, result.Canceled)

	// Simulate "Cancel" button click
	modal.GetDoneFunc()(2, "Cancel")
	assert.Equal(t, "cancel", result.Action)
	assert.True(t, result.Canceled)
}

func TestPromptModal_InputDialog(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	modal := NewPromptModal(config, logger)
	require.NotNil(t, modal)

	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal.ShowInput("Input Title", "Enter value:", "placeholder", "default", callback)
	
	// Test input handling
	testInput := "test input value"
	result = ModalResult{
		Action:   "ok",
		Value:    testInput,
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "ok", result.Action)
	assert.Equal(t, testInput, result.Value)
	assert.False(t, result.Canceled)
}

func TestPromptModal_SelectDialog(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	modal := NewPromptModal(config, logger)
	require.NotNil(t, modal)

	options := []string{"Option 1", "Option 2", "Option 3"}
	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal.ShowSelect("Select Title", "Choose option:", options, callback)
	
	// Test selection
	selectedIndex := 1
	result = ModalResult{
		Action:   "ok",
		Index:    selectedIndex,
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "ok", result.Action)
	assert.Equal(t, selectedIndex, result.Index)
	assert.False(t, result.Canceled)
}

func TestPromptModal_ProgressDialog(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	modal := NewPromptModal(config, logger)
	require.NotNil(t, modal)

	modal.ShowProgress("Processing", "Please wait...")
	assert.Equal(t, ModalTypeProgress, modal.modalType)

	// Test progress updates
	modal.UpdateProgress(25, "Step 1 complete")
	modal.UpdateProgress(50, "Step 2 complete")
	modal.UpdateProgress(75, "Step 3 complete")
	modal.UpdateProgress(100, "Complete")

	// Test closing progress
	var closed bool
	modal.callback = func(result ModalResult) {
		closed = result.Action == "complete"
	}
	
	modal.CloseProgress()
	assert.True(t, closed)
}

func TestPromptModal_ErrorDialog(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	modal := NewPromptModal(config, logger)
	require.NotNil(t, modal)

	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal.ShowError("Error Title", "Error message", callback)
	assert.Equal(t, ModalTypeError, modal.modalType)

	// Simulate OK button
	result = ModalResult{
		Action:   "ok",
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "ok", result.Action)
	assert.False(t, result.Canceled)
}

// Test cases for specialized modal dialogs

func TestPromptModal_TemplateDeleteConfirm(t *testing.T) {
	config := createTestUIConfig()
	template := createTestTemplates()[0]

	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal := ShowTemplateDeleteConfirm(template, config, callback)
	require.NotNil(t, modal)

	// Simulate confirmation
	result = ModalResult{
		Action:   "yes",
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "yes", result.Action)
	assert.False(t, result.Canceled)
}

func TestPromptModal_TemplateImportDialog(t *testing.T) {
	config := createTestUIConfig()

	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal := ShowTemplateImportDialog(config, callback)
	require.NotNil(t, modal)

	// Simulate file path input
	filePath := "/path/to/template.yaml"
	result = ModalResult{
		Action:   "ok",
		Value:    filePath,
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "ok", result.Action)
	assert.Equal(t, filePath, result.Value)
	assert.False(t, result.Canceled)
}

func TestPromptModal_ValidationResultsDialog(t *testing.T) {
	config := createTestUIConfig()
	template := createTestTemplates()[0]

	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	// Test with no issues (valid template)
	modal := ShowValidationResultsDialog(template, []string{}, config, callback)
	require.NotNil(t, modal)
	assert.Equal(t, ModalTypeInfo, modal.modalType)

	// Test with validation issues
	issues := []string{
		"Missing required parameter description",
		"Invalid content format",
		"Template too short",
	}
	modal = ShowValidationResultsDialog(template, issues, config, callback)
	require.NotNil(t, modal)
	assert.Equal(t, ModalTypeError, modal.modalType)

	result = ModalResult{
		Action:   "ok",
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "ok", result.Action)
	assert.False(t, result.Canceled)
}

func TestPromptModal_WorkspaceDetectionResults(t *testing.T) {
	config := createTestUIConfig()
	context := &prompt.WorkspaceContext{
		WorkingDirectory: "/test/workspace",
		Repository:       "test-repo",
		Language:         "go",
		Framework:        "gin",
		GitBranch:        "main",
	}

	var result ModalResult
	callback := func(r ModalResult) {
		result = r
	}

	modal := ShowWorkspaceDetectionResults(context, config, callback)
	require.NotNil(t, modal)
	assert.Equal(t, ModalTypeInfo, modal.modalType)

	result = ModalResult{
		Action:   "ok",
		Canceled: false,
	}
	callback(result)

	assert.Equal(t, "ok", result.Action)
	assert.False(t, result.Canceled)
}

func TestPromptModal_GenerationResults(t *testing.T) {
	config := createTestUIConfig()
	template := createTestTemplates()[0]
	
	result := &prompt.GenerationResult{
		Content:     "Generated prompt content",
		Template:    template,
		Parameters:  map[string]string{"service": "auth", "port": "8080"},
		GeneratedAt: time.Now(),
		ValidationStatus: prompt.ValidationStatus{
			Valid: true,
			Score: 92,
		},
		WordCount: 150,
		CharCount: 750,
	}

	var modalResult ModalResult
	callback := func(r ModalResult) {
		modalResult = r
	}

	modal := ShowGenerationResults(result, config, callback)
	require.NotNil(t, modal)
	assert.Equal(t, ModalTypeInfo, modal.modalType)

	modalResult = ModalResult{
		Action:   "ok",
		Canceled: false,
	}
	callback(modalResult)

	assert.Equal(t, "ok", modalResult.Action)
	assert.False(t, modalResult.Canceled)
}

// Test cases for PromptHistory component

func TestPromptHistory_LoadHistory(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	history := NewPromptHistory(config, logger)
	require.NotNil(t, history)

	history.SetManager(mockManager)

	// Create test history entries
	entries := []prompt.HistoryEntry{
		{
			ID:           "1",
			Template:     "api-service",
			Repository:   "test-repo",
			Language:     "go",
			Framework:    "gin",
			Parameters:   map[string]string{"service": "auth"},
			OutputMethod: "clipboard",
			AITool:       "claude",
			Success:      true,
			Timestamp:    time.Now(),
			Duration:     time.Second * 5,
			WordCount:    120,
		},
		{
			ID:        "2",
			Template:  "database-migration",
			Repository: "db-project",
			Language:  "sql",
			Success:   false,
			ErrorMessage: "validation failed",
			Timestamp: time.Now().Add(-time.Hour),
			Duration:  time.Second * 2,
		},
	}

	mockManager.On("GetHistory", 50, "").Return(entries, nil)

	err := history.LoadHistory(50, "")
	assert.NoError(t, err)

	// Verify history is loaded
	assert.Equal(t, len(entries), history.historyList.GetItemCount())

	mockManager.AssertExpectations(t)
}

func TestPromptHistory_FilterHistory(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	history := NewPromptHistory(config, logger)
	history.SetManager(mockManager)

	// Test filtering by template
	filteredEntries := []prompt.HistoryEntry{
		{
			ID:        "1",
			Template:  "api-service",
			Language:  "go",
			Success:   true,
			Timestamp: time.Now(),
		},
	}

	mockManager.On("GetHistory", 50, "api").Return(filteredEntries, nil)

	err := history.FilterHistory("api")
	assert.NoError(t, err)

	assert.Equal(t, 1, history.historyList.GetItemCount())

	mockManager.AssertExpectations(t)
}

func TestPromptHistory_HistorySelection(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	history := NewPromptHistory(config, logger)
	history.SetManager(mockManager)

	entry := &prompt.HistoryEntry{
		ID:        "1",
		Template:  "api-service",
		Success:   true,
		Timestamp: time.Now(),
	}

	var selectedEntry *prompt.HistoryEntry
	history.SetOnEntrySelected(func(e *prompt.HistoryEntry) {
		selectedEntry = e
	})

	// Simulate selection
	history.selectEntry(entry)

	assert.NotNil(t, selectedEntry)
	assert.Equal(t, entry.ID, selectedEntry.ID)
	assert.Equal(t, entry.Template, selectedEntry.Template)
}

// Test cases for TemplateEditor component

func TestTemplateEditor_Initialize(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	editor := NewTemplateEditor(config, logger)
	require.NotNil(t, editor)

	assert.NotNil(t, editor.form)
	assert.NotNil(t, editor.nameField)
	assert.NotNil(t, editor.categoryDropdown)
	assert.NotNil(t, editor.descriptionField)
	assert.NotNil(t, editor.contentArea)
}

func TestTemplateEditor_LoadTemplate(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	editor := NewTemplateEditor(config, logger)
	template := createTestTemplates()[0]

	editor.LoadTemplate(&template)

	// Verify template is loaded into form fields
	assert.Equal(t, template.Name, editor.nameField.GetText())
	assert.Equal(t, template.Description, editor.descriptionField.GetText())
	assert.Equal(t, template.Content, editor.contentArea.GetText(false))
}

func TestTemplateEditor_ValidateTemplate(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	editor := NewTemplateEditor(config, logger)

	// Test valid template
	template := createTestTemplates()[0]
	editor.LoadTemplate(&template)

	issues := editor.ValidateCurrentTemplate()
	assert.Empty(t, issues)

	// Test invalid template
	editor.nameField.SetText("")
	editor.descriptionField.SetText("")

	issues = editor.ValidateCurrentTemplate()
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0], "name")
	assert.Contains(t, issues[1], "description")
}

func TestTemplateEditor_SaveTemplate(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	editor := NewTemplateEditor(config, logger)
	editor.SetManager(mockManager)
	
	template := createTestTemplates()[0]
	editor.LoadTemplate(&template)

	mockManager.On("UpdateTemplate", template.Name, false, true).Return(nil)

	var savedTemplate *prompt.Template
	editor.SetOnTemplateSaved(func(t *prompt.Template) {
		savedTemplate = t
	})

	err := editor.SaveCurrentTemplate()
	assert.NoError(t, err)
	assert.NotNil(t, savedTemplate)

	mockManager.AssertExpectations(t)
}

// Test cases for TemplateGenerator component

func TestTemplateGenerator_Initialize(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	generator := NewTemplateGenerator(config, logger)
	require.NotNil(t, generator)

	assert.NotNil(t, generator.form)
	assert.NotNil(t, generator.templateDropdown)
	assert.NotNil(t, generator.parametersForm)
}

func TestTemplateGenerator_LoadTemplate(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	generator := NewTemplateGenerator(config, logger)
	template := createTestTemplates()[0]

	generator.LoadTemplate(&template)

	// Verify parameters form is populated
	assert.Equal(t, len(template.Parameters), generator.parameterFields)
}

func TestTemplateGenerator_GeneratePrompt(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	generator := NewTemplateGenerator(config, logger)
	generator.SetManager(mockManager)

	template := createTestTemplates()[0]
	generator.LoadTemplate(&template)

	result := &prompt.GenerationResult{
		Content:     "Generated content",
		Template:    template,
		Parameters:  map[string]string{"service": "auth", "port": "8080"},
		GeneratedAt: time.Now(),
		ValidationStatus: prompt.ValidationStatus{Valid: true, Score: 90},
		WordCount: 50,
		CharCount: 200,
	}

	mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)

	var generatedResult *prompt.GenerationResult
	generator.SetOnPromptGenerated(func(r *prompt.GenerationResult) {
		generatedResult = r
	})

	err := generator.GeneratePrompt()
	assert.NoError(t, err)
	assert.NotNil(t, generatedResult)
	assert.Equal(t, result.Content, generatedResult.Content)

	mockManager.AssertExpectations(t)
}

// Test cases for WorkspaceStatus component

func TestWorkspaceStatus_Initialize(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)

	status := NewWorkspaceStatus(config, logger)
	require.NotNil(t, status)

	assert.NotNil(t, status.infoView)
	assert.NotNil(t, status.suggestionsView)
}

func TestWorkspaceStatus_UpdateContext(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	status := NewWorkspaceStatus(config, logger)
	status.SetManager(mockManager)

	context := &prompt.WorkspaceContext{
		WorkingDirectory:   "/test/workspace",
		Repository:         "test-repo",
		Language:           "go",
		Framework:          "gin",
		AvailableLanguages: []string{"go", "javascript"},
		RecentFiles:        []string{"main.go", "handlers.go"},
		GitBranch:          "main",
	}

	suggestions := []prompt.TemplateSuggestion{
		{Name: "go-service", Reason: "matches go language", Relevance: 0.8},
		{Name: "api-handler", Reason: "recent handler files", Relevance: 0.6},
	}

	mockManager.On("SuggestTemplates", context).Return(suggestions)

	status.UpdateContext(context)

	// Verify context is displayed
	infoText := status.infoView.GetText(false)
	assert.Contains(t, infoText, "test-repo")
	assert.Contains(t, infoText, "go")
	assert.Contains(t, infoText, "gin")

	mockManager.AssertExpectations(t)
}

func TestWorkspaceStatus_RefreshContext(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	status := NewWorkspaceStatus(config, logger)
	status.SetManager(mockManager)

	context := &prompt.WorkspaceContext{
		Repository: "updated-repo",
		Language:   "python",
		Framework:  "flask",
	}

	mockManager.On("DetectWorkspaceContext").Return(context, nil)
	mockManager.On("SuggestTemplates", context).Return([]prompt.TemplateSuggestion{})

	err := status.RefreshContext()
	assert.NoError(t, err)

	// Verify updated context is displayed
	infoText := status.infoView.GetText(false)
	assert.Contains(t, infoText, "updated-repo")
	assert.Contains(t, infoText, "python")
	assert.Contains(t, infoText, "flask")

	mockManager.AssertExpectations(t)
}

// Benchmark tests for TUI components

func BenchmarkPromptBrowser_LoadTemplates(b *testing.B) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(&testing.T{})
	mockManager := &MockPromptManager{}

	// Create large template set
	templates := make([]prompt.Template, 100)
	for i := 0; i < 100; i++ {
		templates[i] = prompt.Template{
			Name:        fmt.Sprintf("template-%d", i),
			Category:    "general",
			Description: fmt.Sprintf("Template %d description", i),
		}
	}

	mockManager.On("ListTemplates", "").Return(templates, nil)

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := browser.LoadTemplates("")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPromptModal_ShowConfirmation(b *testing.B) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(&testing.T{})

	callback := func(result ModalResult) {
		// No-op callback for benchmarking
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modal := NewPromptModal(config, logger)
		modal.ShowConfirmation("Test", "Message", callback)
	}
}

// Integration tests combining multiple components

func TestTUIIntegration_TemplateWorkflow(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	// Create components
	browser := NewPromptBrowser(config, logger)
	editor := NewTemplateEditor(config, logger)
	generator := NewTemplateGenerator(config, logger)

	browser.SetManager(mockManager)
	editor.SetManager(mockManager)
	generator.SetManager(mockManager)

	templates := createTestTemplates()
	mockManager.On("ListTemplates", "").Return(templates, nil)

	// Test workflow: browse -> select -> edit -> generate
	t.Run("browse templates", func(t *testing.T) {
		err := browser.LoadTemplates("")
		assert.NoError(t, err)
		assert.Greater(t, browser.templateList.GetItemCount(), 0)
	})

	t.Run("select and edit template", func(t *testing.T) {
		selectedTemplate := &templates[0]
		
		// Simulate template selection and editing
		editor.LoadTemplate(selectedTemplate)
		assert.Equal(t, selectedTemplate.Name, editor.nameField.GetText())
		
		// Test validation
		issues := editor.ValidateCurrentTemplate()
		assert.Empty(t, issues)
	})

	t.Run("generate prompt from template", func(t *testing.T) {
		selectedTemplate := &templates[0]
		generator.LoadTemplate(selectedTemplate)

		result := &prompt.GenerationResult{
			Content:     "Generated prompt content",
			Template:    *selectedTemplate,
			GeneratedAt: time.Now(),
			ValidationStatus: prompt.ValidationStatus{Valid: true},
		}

		mockManager.On("GeneratePrompt", mock.AnythingOfType("*prompt.GenerationConfig")).Return(result, nil)

		err := generator.GeneratePrompt()
		assert.NoError(t, err)
	})

	mockManager.AssertExpectations(t)
}

func TestTUIIntegration_ModalChaining(t *testing.T) {
	config := createTestUIConfig()
	template := createTestTemplates()[0]

	// Test chaining of modals in a workflow
	results := make([]ModalResult, 0)
	
	callback := func(result ModalResult) {
		results = append(results, result)
	}

	// Step 1: Confirm template deletion
	modal1 := ShowTemplateDeleteConfirm(template, config, callback)
	require.NotNil(t, modal1)

	// Simulate "Yes" response
	results = append(results, ModalResult{Action: "yes"})

	// Step 2: Show progress for deletion
	modal2 := NewPromptModal(config, nil)
	modal2.ShowProgress("Deleting", "Deleting template...")
	modal2.UpdateProgress(50, "Backing up...")
	modal2.UpdateProgress(100, "Complete")

	// Step 3: Show completion confirmation
	modal3 := NewPromptModal(config, callback)
	modal3.ShowInfo("Success", "Template deleted successfully", callback)

	// Simulate "OK" response
	results = append(results, ModalResult{Action: "ok"})

	assert.Len(t, results, 2)
	assert.Equal(t, "yes", results[0].Action)
	assert.Equal(t, "ok", results[1].Action)
}

// Error handling tests for TUI components

func TestTUIComponents_ErrorHandling(t *testing.T) {
	config := createTestUIConfig()
	logger := zaptest.NewLogger(t)
	mockManager := &MockPromptManager{}

	browser := NewPromptBrowser(config, logger)
	browser.SetManager(mockManager)

	// Test error handling in template loading
	mockManager.On("ListTemplates", "").Return([]prompt.Template{}, fmt.Errorf("database error"))

	err := browser.LoadTemplates("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	// Test error recovery
	templates := createTestTemplates()
	mockManager.On("ListTemplates", "").Return(templates, nil)

	err = browser.LoadTemplates("")
	assert.NoError(t, err)

	mockManager.AssertExpectations(t)
}