package prompt

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"openai-cli/internal/config"
)

// Test data
const testTemplateYAML = `
name: test-template
category: general
description: A test template for unit testing
content: |
  # Test Prompt
  
  Repository: {{.repo}}
  Language: {{.language}}
  Task: {{.task}}
  
  ## Requirements
  - Implement {{.task}} functionality
  - Use {{.language}} programming language
  - Follow best practices
  
  ## Expected Output
  - Working code implementation
  - Unit tests
  - Documentation
parameters:
  - name: repo
    description: Repository name
    type: string
    required: true
    default: test-repo
  - name: language
    description: Programming language
    type: select
    required: true
    options: ["go", "python", "javascript"]
  - name: task
    description: Task description
    type: string
    required: true
    placeholder: Enter task description
examples:
  - "vibes-mcp-cli prompt generate test-template --repo myrepo --language go --task 'user authentication'"
tags: ["test", "example"]
version: "1.0.0"
`

func TestNewManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
	}

	tests := []struct {
		name        string
		cfg         *config.Config
		logger      *zap.Logger
		expectError bool
	}{
		{
			name:        "valid config and logger",
			cfg:         cfg,
			logger:      logger,
			expectError: false,
		},
		{
			name:        "nil config",
			cfg:         nil,
			logger:      logger,
			expectError: true,
		},
		{
			name:        "nil logger",
			cfg:         cfg,
			logger:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.cfg, tt.logger)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestTemplateParser(t *testing.T) {
	logger := zaptest.NewLogger(t)
	parser := NewTemplateParser(logger)

	t.Run("parse YAML template", func(t *testing.T) {
		template, err := parser.ParseContent(testTemplateYAML, "test-template")
		require.NoError(t, err)
		
		assert.Equal(t, "test-template", template.Name)
		assert.Equal(t, "general", template.Category)
		assert.Equal(t, "A test template for unit testing", template.Description)
		assert.Contains(t, template.Content, "# Test Prompt")
		assert.Len(t, template.Parameters, 3)
		assert.Len(t, template.Examples, 1)
		assert.Equal(t, "1.0.0", template.Version)
	})

	t.Run("validate template structure", func(t *testing.T) {
		template, err := parser.ParseContent(testTemplateYAML, "test-template")
		require.NoError(t, err)
		
		err = parser.ValidateStructure(template)
		assert.NoError(t, err)
	})

	t.Run("invalid template structure", func(t *testing.T) {
		invalidYAML := `
name: ""
category: invalid-category
description: ""
content: ""
`
		template, err := parser.ParseContent(invalidYAML, "")
		require.NoError(t, err)
		
		err = parser.ValidateStructure(template)
		assert.Error(t, err)
	})
}

func TestWorkspaceDetector(t *testing.T) {
	logger := zaptest.NewLogger(t)
	detector := NewWorkspaceDetector(logger)

	t.Run("detect context", func(t *testing.T) {
		ctx := context.Background()
		workspaceContext, err := detector.DetectContext(ctx)
		
		assert.NoError(t, err)
		assert.NotNil(t, workspaceContext)
		assert.NotEmpty(t, workspaceContext.WorkingDirectory)
		assert.NotEmpty(t, workspaceContext.Repository)
	})

	t.Run("detect Go language", func(t *testing.T) {
		// Create temporary directory with Go files
		tempDir := t.TempDir()
		goModContent := "module test\n\ngo 1.21\n"
		err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
		require.NoError(t, err)
		
		language, err := detector.DetectLanguage(tempDir)
		assert.NoError(t, err)
		assert.Equal(t, "go", language)
	})

	t.Run("detect recent files", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create test files
		testFiles := []string{"main.go", "config.go", "test.go"}
		for _, file := range testFiles {
			err := os.WriteFile(filepath.Join(tempDir, file), []byte("package main"), 0644)
			require.NoError(t, err)
		}
		
		files, err := detector.GetRecentFiles(tempDir, 5)
		assert.NoError(t, err)
		assert.Len(t, files, 3)
	})
}

func TestGenerator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	generator := NewGenerator(logger)

	t.Run("generate prompt", func(t *testing.T) {
		ctx := context.Background()
		
		config := &GenerationConfig{
			TemplateName: "test-template",
			Interactive:  false,
			Parameters: map[string]string{
				"repo":     "test-repo",
				"language": "go",
				"task":     "user authentication",
			},
			Validate: true,
		}
		
		result, err := generator.Generate(ctx, config)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
		assert.Greater(t, result.WordCount, 0)
		assert.Greater(t, result.CharCount, 0)
	})

	t.Run("fill template", func(t *testing.T) {
		template := &Template{
			Name:    "test",
			Content: "Hello {{.name}}, welcome to {{.repo}}!",
		}
		
		parameters := map[string]string{
			"name": "John",
			"repo": "test-repo",
		}
		
		content, err := generator.FillTemplate(template, parameters)
		assert.NoError(t, err)
		assert.Contains(t, content, "Hello John")
		assert.Contains(t, content, "welcome to test-repo")
	})

	t.Run("calculate stats", func(t *testing.T) {
		content := "This is a test content with multiple words."
		wordCount, charCount := generator.CalculateStats(content)
		
		assert.Equal(t, 8, wordCount)
		assert.Equal(t, len(content), charCount)
	})

	t.Run("format output", func(t *testing.T) {
		content := "# Test\nThis is **bold** text."
		
		// Test markdown format (should be unchanged)
		formatted, err := generator.FormatOutput(content, FormatMarkdown)
		assert.NoError(t, err)
		assert.Equal(t, content, formatted)
		
		// Test text format (should remove markdown)
		formatted, err = generator.FormatOutput(content, FormatText)
		assert.NoError(t, err)
		assert.NotContains(t, formatted, "**")
		assert.NotContains(t, formatted, "#")
	})
}

func TestValidator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validator := NewValidator(logger)

	t.Run("validate valid template", func(t *testing.T) {
		parser := NewTemplateParser(logger)
		template, err := parser.ParseContent(testTemplateYAML, "test-template")
		require.NoError(t, err)
		
		valid, issues, err := validator.ValidateTemplate(template)
		assert.NoError(t, err)
		assert.True(t, valid)
		assert.Empty(t, issues)
	})

	t.Run("validate invalid template", func(t *testing.T) {
		template := &Template{
			Name:        "", // Invalid: empty name
			Category:    "invalid", // Invalid: not a valid category
			Description: "", // Invalid: empty description
			Content:     "", // Invalid: empty content
		}
		
		valid, issues, err := validator.ValidateTemplate(template)
		assert.NoError(t, err)
		assert.False(t, valid)
		assert.NotEmpty(t, issues)
	})

	t.Run("validate content quality", func(t *testing.T) {
		goodContent := `# Test Template

This is a well-structured template with:
- Clear headers
- Proper formatting
- Meaningful content
- Code examples: {{.code}}

## Requirements
- Implement functionality
- Write tests
- Add documentation

The template provides {{.feature}} implementation.`

		score, issues, _ := validator.ValidateContent(goodContent)
		assert.Greater(t, score, 70) // Should have good score
		assert.Empty(t, issues) // Should have no critical issues
		
		// Bad content
		badContent := "TODO: write content"
		score, issues, _ = validator.ValidateContent(badContent)
		assert.Less(t, score, 50) // Should have poor score
		assert.NotEmpty(t, issues) // Should have issues
	})

	t.Run("get quality score", func(t *testing.T) {
		parser := NewTemplateParser(logger)
		template, err := parser.ParseContent(testTemplateYAML, "test-template")
		require.NoError(t, err)
		
		score := validator.GetQualityScore(template)
		assert.Greater(t, score, 70) // Should be high quality
		assert.LessOrEqual(t, score, 100)
	})
}

func TestHistoryTracker(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tempDir := t.TempDir()
	tracker := NewHistoryTracker(tempDir, logger)

	t.Run("record and retrieve history", func(t *testing.T) {
		entry := &HistoryEntry{
			Template:   "test-template",
			Repository: "test-repo",
			Language:   "go",
			Parameters: map[string]string{"task": "test"},
			Success:    true,
			Timestamp:  time.Now(),
			WordCount:  100,
		}
		
		err := tracker.Record(entry)
		assert.NoError(t, err)
		assert.NotEmpty(t, entry.ID) // Should generate ID
		
		history, err := tracker.GetHistory(10, "")
		assert.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, "test-template", history[0].Template)
	})

	t.Run("filter history", func(t *testing.T) {
		// Record entries with different templates
		entries := []*HistoryEntry{
			{Template: "go-template", Language: "go", Success: true},
			{Template: "python-template", Language: "python", Success: true},
			{Template: "go-service", Language: "go", Success: true},
		}
		
		for _, entry := range entries {
			err := tracker.Record(entry)
			assert.NoError(t, err)
		}
		
		// Filter by "go"
		history, err := tracker.GetHistory(10, "go")
		assert.NoError(t, err)
		assert.Len(t, history, 2) // Should match go-template and go-service
	})

	t.Run("get statistics", func(t *testing.T) {
		stats, err := tracker.GetStats()
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Greater(t, stats.TotalGenerations, 0)
		assert.GreaterOrEqual(t, stats.SuccessRate, 0.0)
		assert.LessOrEqual(t, stats.SuccessRate, 1.0)
	})

	t.Run("cleanup old entries", func(t *testing.T) {
		// Record old entry
		oldEntry := &HistoryEntry{
			Template:  "old-template",
			Success:   true,
			Timestamp: time.Now().Add(-48 * time.Hour), // 2 days ago
		}
		err := tracker.Record(oldEntry)
		assert.NoError(t, err)
		
		// Cleanup entries older than 1 day
		err = tracker.Cleanup(24 * time.Hour)
		assert.NoError(t, err)
		
		// Old entry should be gone
		history, err := tracker.GetHistory(100, "old-template")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})
}

func TestIntegrator(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
	}
	integrator := NewIntegrator(cfg, logger)

	t.Run("save to file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test-output.txt")
		content := "Test content for file output"
		
		err := integrator.SaveToFile(content, filePath)
		assert.NoError(t, err)
		
		// Verify file was created
		savedContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(savedContent))
	})

	t.Run("test integration", func(t *testing.T) {
		// Test clipboard (should work on most systems)
		err := integrator.TestIntegration("clipboard")
		// Don't assert error since clipboard might not be available in CI
		t.Logf("Clipboard test result: %v", err)
	})
}

func TestConfigManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompt-config.json")
	
	configManager := NewConfigManager(configPath, logger)

	t.Run("load default config", func(t *testing.T) {
		config, err := configManager.LoadConfig()
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "vibes-mcp-cli", config.DefaultRepository)
		assert.Equal(t, "go", config.PreferredLanguage)
		assert.True(t, config.AutoValidate)
	})

	t.Run("update and save config", func(t *testing.T) {
		err := configManager.UpdateConfig("preferred_language", "python")
		assert.NoError(t, err)
		
		config := configManager.GetConfig()
		assert.Equal(t, "python", config.PreferredLanguage)
		
		// Verify config was saved
		newManager := NewConfigManager(configPath, logger)
		config2, err := newManager.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, "python", config2.PreferredLanguage)
	})

	t.Run("validate config", func(t *testing.T) {
		valid, issues := configManager.ValidateConfig()
		assert.True(t, valid)
		assert.Empty(t, issues)
		
		// Test invalid config
		err := configManager.UpdateConfig("history_limit", "invalid")
		assert.Error(t, err)
	})

	t.Run("export and import config", func(t *testing.T) {
		exportPath := filepath.Join(tempDir, "exported-config.json")
		
		err := configManager.ExportConfig(exportPath)
		assert.NoError(t, err)
		
		// Verify export file exists
		_, err = os.Stat(exportPath)
		assert.NoError(t, err)
		
		// Test import
		newConfigPath := filepath.Join(tempDir, "imported-config.json")
		newManager := NewConfigManager(newConfigPath, logger)
		
		err = newManager.ImportConfig(exportPath)
		assert.NoError(t, err)
		
		config := newManager.GetConfig()
		assert.Equal(t, "python", config.PreferredLanguage) // Should preserve changed value
	})
}

func TestManagerIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
	}

	// Create temporary directory for templates
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates")
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create test template file
	templateFile := filepath.Join(templateDir, "test-template.yaml")
	err = os.WriteFile(templateFile, []byte(testTemplateYAML), 0644)
	require.NoError(t, err)

	// Override template directories for testing
	manager, err := NewManager(cfg, logger)
	require.NoError(t, err)
	
	// Update config to use test directory
	managerImpl := manager.(*ManagerImpl)
	managerImpl.config.TemplateDirectories = []string{templateDir}

	t.Run("end-to-end prompt generation", func(t *testing.T) {
		// List templates
		templates, err := manager.ListTemplates("")
		assert.NoError(t, err)
		assert.Len(t, templates, 1)
		assert.Equal(t, "test-template", templates[0].Name)

		// Get specific template
		template, err := manager.GetTemplate("test-template")
		assert.NoError(t, err)
		assert.Equal(t, "test-template", template.Name)

		// Generate prompt
		config := &GenerationConfig{
			TemplateName: "test-template",
			Parameters: map[string]string{
				"repo":     "my-repo",
				"language": "go",
				"task":     "API endpoint",
			},
			Validate: true,
		}

		result, err := manager.GeneratePrompt(config)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.Content, "my-repo")
		assert.Contains(t, result.Content, "go")
		assert.Contains(t, result.Content, "API endpoint")

		// Validate template
		valid, issues := manager.ValidateTemplate("test-template")
		assert.True(t, valid)
		assert.Empty(t, issues)
	})

	t.Run("workspace context detection", func(t *testing.T) {
		context, err := manager.DetectWorkspaceContext()
		assert.NoError(t, err)
		assert.NotNil(t, context)
		assert.NotEmpty(t, context.WorkingDirectory)

		// Test template suggestions
		suggestions := manager.SuggestTemplates(context)
		assert.NotNil(t, suggestions)
		// Suggestions might be empty if no templates match context
	})

	t.Run("template validation", func(t *testing.T) {
		report, err := manager.ValidateAllTemplates()
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, 1, report.Total)
		assert.Greater(t, report.AverageScore, 0)
	})
}

// Benchmark tests
func BenchmarkTemplateGeneration(b *testing.B) {
	logger := zap.NewNop()
	generator := NewGenerator(logger)
	
	template := &Template{
		Name:    "benchmark-template",
		Content: "Test content with {{.param1}} and {{.param2}} placeholders.",
	}
	
	parameters := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.FillTemplate(template, parameters)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWorkspaceDetection(b *testing.B) {
	logger := zap.NewNop()
	detector := NewWorkspaceDetector(logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.DetectContext(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTemplateValidation(b *testing.B) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	parser := NewTemplateParser(logger)
	
	template, err := parser.ParseContent(testTemplateYAML, "benchmark-template")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := validator.ValidateTemplate(template)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test helpers
func createTestTemplate(name, category, content string) *Template {
	return &Template{
		Name:        name,
		Category:    category,
		Description: "Test template for " + name,
		Content:     content,
		Parameters:  []TemplateParameter{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func createTestGenerationConfig(templateName string) *GenerationConfig {
	return &GenerationConfig{
		TemplateName: templateName,
		Interactive:  false,
		Parameters: map[string]string{
			"repo":     "test-repo",
			"language": "go",
		},
		Validate: true,
	}
}

func assertValidTemplate(t *testing.T, template *Template) {
	assert.NotEmpty(t, template.Name)
	assert.NotEmpty(t, template.Category)
	assert.NotEmpty(t, template.Description)
	assert.NotEmpty(t, template.Content)
	assert.NotZero(t, template.CreatedAt)
	assert.NotZero(t, template.UpdatedAt)
}

func assertValidGenerationResult(t *testing.T, result *GenerationResult) {
	assert.NotEmpty(t, result.Content)
	assert.NotZero(t, result.GeneratedAt)
	assert.Greater(t, result.WordCount, 0)
	assert.Greater(t, result.CharCount, 0)
	assert.NotEmpty(t, result.Template.Name)
}