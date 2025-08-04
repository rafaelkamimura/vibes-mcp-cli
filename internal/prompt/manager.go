package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/config"
)

// NewManager creates a new prompt manager instance
func NewManager(cfg *config.Config, logger *zap.Logger) (Manager, error) {
	if cfg == nil {
		return nil, &PromptError{
			Type:    ErrorTypeConfig,
			Message: "config cannot be nil",
		}
	}
	if logger == nil {
		return nil, &PromptError{
			Type:    ErrorTypeConfig,
			Message: "logger cannot be nil",
		}
	}

	// Initialize default config
	promptConfig := &Config{
		DefaultRepository:   "vibes-mcp-cli",
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

	// Set template directories
	homeDir, _ := os.UserHomeDir()
	promptConfig.TemplateDirectories = []string{
		filepath.Join(homeDir, ".vibes-mcp-cli", "templates"),
		"./prompt-templates",
		"./templates",
	}
	promptConfig.CustomTemplatesPath = filepath.Join(homeDir, ".vibes-mcp-cli", "custom-templates")

	// Initialize service dependencies
	templateParser := NewTemplateParser(logger)
	workspaceDetector := NewWorkspaceDetector(logger)
	generator := NewGenerator(logger)
	validator := NewValidator(logger)
	historyTracker := NewHistoryTracker(promptConfig.CustomTemplatesPath, logger)
	integrator := NewIntegrator(cfg, logger)

	manager := &ManagerImpl{
		cfg:               cfg,
		logger:            logger,
		config:            promptConfig,
		templateParser:    templateParser,
		workspaceDetector: workspaceDetector,
		generator:         generator,
		validator:         validator,
		historyTracker:    historyTracker,
		integrator:        integrator,
	}

	// Ensure directories exist
	if err := manager.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Load existing config if available
	if err := manager.loadConfig(); err != nil {
		logger.Warn("Failed to load existing config, using defaults", zap.Error(err))
	}

	logger.Info("Prompt manager initialized successfully",
		zap.String("custom_templates_path", promptConfig.CustomTemplatesPath),
		zap.Int("template_directories", len(promptConfig.TemplateDirectories)))

	return manager, nil
}

// ListTemplates returns templates filtered by category
func (m *ManagerImpl) ListTemplates(category string) ([]Template, error) {
	ctx := context.Background()
	m.logger.Debug("Listing templates", zap.String("category", category))

	var allTemplates []Template

	// Search in all template directories
	for _, dir := range m.config.TemplateDirectories {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		templates, err := m.loadTemplatesFromDirectory(ctx, dir, category)
		if err != nil {
			m.logger.Warn("Failed to load templates from directory",
				zap.String("directory", dir),
				zap.Error(err))
			continue
		}

		allTemplates = append(allTemplates, templates...)
	}

	m.logger.Info("Templates listed successfully",
		zap.Int("count", len(allTemplates)),
		zap.String("category", category))

	return allTemplates, nil
}

// GetTemplate retrieves a specific template by name
func (m *ManagerImpl) GetTemplate(name string) (Template, error) {
	m.logger.Debug("Getting template", zap.String("name", name))

	// Search in all template directories
	for _, dir := range m.config.TemplateDirectories {
		templatePath := m.findTemplateFile(dir, name)
		if templatePath == "" {
			continue
		}

		template, err := m.templateParser.LoadTemplate(templatePath)
		if err != nil {
			m.logger.Warn("Failed to load template",
				zap.String("path", templatePath),
				zap.Error(err))
			continue
		}

		m.logger.Info("Template loaded successfully",
			zap.String("name", name),
			zap.String("path", templatePath))

		return *template, nil
	}

	return Template{}, &PromptError{
		Type:    ErrorTypeTemplate,
		Message: fmt.Sprintf("template '%s' not found", name),
	}
}

// GeneratePrompt generates a prompt from a template with given configuration
func (m *ManagerImpl) GeneratePrompt(config *GenerationConfig) (*GenerationResult, error) {
	ctx := context.Background()
	startTime := time.Now()

	m.logger.Info("Starting prompt generation",
		zap.String("template", config.TemplateName),
		zap.Bool("interactive", config.Interactive))

	// Get template
	template, err := m.GetTemplate(config.TemplateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Generate prompt
	result, err := m.generator.Generate(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate prompt: %w", err)
	}

	// Validate if requested
	if config.Validate && m.config.ValidationEnabled {
		validationStatus := m.validateGeneratedPrompt(result.Content, &template)
		result.ValidationStatus = validationStatus

		if !validationStatus.Valid {
			m.logger.Warn("Generated prompt failed validation",
				zap.String("template", config.TemplateName),
				zap.Strings("issues", validationStatus.Issues))
		}
	}

	// Record generation in history
	historyEntry := &HistoryEntry{
		ID:         generateID(),
		Template:   config.TemplateName,
		Repository: config.Parameters["repo"],
		Language:   config.Parameters["language"],
		Framework:  config.Parameters["framework"],
		Parameters: config.Parameters,
		Success:    true,
		Timestamp:  startTime,
		Duration:   time.Since(startTime),
		WordCount:  result.WordCount,
	}

	if config.Context != nil {
		historyEntry.Repository = config.Context.Repository
		historyEntry.Language = config.Context.Language
		historyEntry.Framework = config.Context.Framework
	}

	if err := m.historyTracker.Record(historyEntry); err != nil {
		m.logger.Warn("Failed to record generation history", zap.Error(err))
	}

	m.logger.Info("Prompt generated successfully",
		zap.String("template", config.TemplateName),
		zap.Int("word_count", result.WordCount),
		zap.Duration("duration", time.Since(startTime)))

	return result, nil
}

// DetectWorkspaceContext detects the current workspace context
func (m *ManagerImpl) DetectWorkspaceContext() (*WorkspaceContext, error) {
	ctx := context.Background()
	m.logger.Debug("Detecting workspace context")

	context, err := m.workspaceDetector.DetectContext(ctx)
	if err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeWorkspace,
			Message: "failed to detect workspace context",
			Cause:   err,
		}
	}

	m.logger.Info("Workspace context detected",
		zap.String("repository", context.Repository),
		zap.String("language", context.Language),
		zap.String("framework", context.Framework))

	return context, nil
}

// SuggestTemplates suggests templates based on workspace context
func (m *ManagerImpl) SuggestTemplates(context *WorkspaceContext) []TemplateSuggestion {
	m.logger.Debug("Suggesting templates", zap.String("language", context.Language))

	var suggestions []TemplateSuggestion

	// Get all templates
	templates, err := m.ListTemplates("")
	if err != nil {
		m.logger.Warn("Failed to list templates for suggestions", zap.Error(err))
		return suggestions
	}

	// Score templates based on context relevance
	for _, template := range templates {
		relevance := m.calculateTemplateRelevance(template, context)
		if relevance > 0.3 { // Only suggest relevant templates
			reason := m.generateSuggestionReason(template, context)
			suggestions = append(suggestions, TemplateSuggestion{
				Name:     template.Name,
				Reason:   reason,
				Relevance: relevance,
				Category: template.Category,
			})
		}
	}

	// Sort by relevance (simple bubble sort for small arrays)
	for i := 0; i < len(suggestions)-1; i++ {
		for j := 0; j < len(suggestions)-i-1; j++ {
			if suggestions[j].Relevance < suggestions[j+1].Relevance {
				suggestions[j], suggestions[j+1] = suggestions[j+1], suggestions[j]
			}
		}
	}

	// Limit to top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	m.logger.Info("Template suggestions generated",
		zap.Int("count", len(suggestions)))

	return suggestions
}

// ValidateTemplate validates a specific template
func (m *ManagerImpl) ValidateTemplate(name string) (bool, []string) {
	m.logger.Debug("Validating template", zap.String("name", name))

	template, err := m.GetTemplate(name)
	if err != nil {
		return false, []string{fmt.Sprintf("Template not found: %s", err.Error())}
	}

	valid, issues, err := m.validator.ValidateTemplate(&template)
	if err != nil {
		return false, []string{fmt.Sprintf("Validation error: %s", err.Error())}
	}

	m.logger.Info("Template validation completed",
		zap.String("name", name),
		zap.Bool("valid", valid),
		zap.Int("issues", len(issues)))

	return valid, issues
}

// ValidateAllTemplates validates all available templates
func (m *ManagerImpl) ValidateAllTemplates() (*ValidationReport, error) {
	m.logger.Debug("Validating all templates")

	templates, err := m.ListTemplates("")
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	report := &ValidationReport{
		Total:       len(templates),
		Issues:      make(map[string][]string),
		GeneratedAt: time.Now(),
	}

	var totalScore int
	for _, template := range templates {
		valid, issues, err := m.validator.ValidateTemplate(&template)
		if err != nil {
			report.Issues[template.Name] = []string{fmt.Sprintf("Validation error: %s", err.Error())}
			continue
		}

		if valid {
			report.Valid++
		} else {
			report.Invalid++
			report.Issues[template.Name] = issues
		}

		score := m.validator.GetQualityScore(&template)
		totalScore += score
	}

	if report.Total > 0 {
		report.AverageScore = totalScore / report.Total
	}

	m.logger.Info("All templates validated",
		zap.Int("total", report.Total),
		zap.Int("valid", report.Valid),
		zap.Int("invalid", report.Invalid))

	return report, nil
}

// GetConfig returns the current configuration
func (m *ManagerImpl) GetConfig() *Config {
	return m.config
}

// SetConfig sets a configuration value
func (m *ManagerImpl) SetConfig(key, value string) error {
	m.logger.Debug("Setting config", zap.String("key", key), zap.String("value", value))

	switch key {
	case "default-repository":
		m.config.DefaultRepository = value
	case "preferred-language":
		m.config.PreferredLanguage = value
	case "preferred-framework":
		m.config.PreferredFramework = value
	case "auto-clipboard":
		m.config.AutoClipboard = value == "true"
	case "auto-validate":
		m.config.AutoValidate = value == "true"
	case "preferred-ai-tool":
		m.config.PreferredAITool = value
	case "output-format":
		m.config.OutputFormat = value
	default:
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: fmt.Sprintf("unknown config key: %s", key),
		}
	}

	m.config.LastUpdated = time.Now()

	// Save config
	if err := m.saveConfig(); err != nil {
		m.logger.Warn("Failed to save config", zap.Error(err))
	}

	return nil
}

// GetConfigValue returns a configuration value
func (m *ManagerImpl) GetConfigValue(key string) string {
	switch key {
	case "default-repository":
		return m.config.DefaultRepository
	case "preferred-language":
		return m.config.PreferredLanguage
	case "preferred-framework":
		return m.config.PreferredFramework
	case "auto-clipboard":
		if m.config.AutoClipboard {
			return "true"
		}
		return "false"
	case "auto-validate":
		if m.config.AutoValidate {
			return "true"
		}
		return "false"
	case "preferred-ai-tool":
		return m.config.PreferredAITool
	case "output-format":
		return m.config.OutputFormat
	default:
		return ""
	}
}

// GetHistory returns generation history
func (m *ManagerImpl) GetHistory(limit int, filter string) ([]HistoryEntry, error) {
	return m.historyTracker.GetHistory(limit, filter)
}

// RecordGeneration records a generation in history
func (m *ManagerImpl) RecordGeneration(entry *HistoryEntry) error {
	return m.historyTracker.Record(entry)
}

// CopyToClipboard copies content to system clipboard
func (m *ManagerImpl) CopyToClipboard(content string) error {
	return m.integrator.CopyToClipboard(content)
}

// SaveToFile saves content to a file
func (m *ManagerImpl) SaveToFile(content, filePath string) error {
	return m.integrator.SaveToFile(content, filePath)
}

// UseContext7 integrates with Context7
func (m *ManagerImpl) UseContext7(result *GenerationResult) error {
	ctx := context.Background()
	return m.integrator.UseContext7(ctx, result)
}

// TriggerBeastmode activates Beastmode
func (m *ManagerImpl) TriggerBeastmode(result *GenerationResult) error {
	ctx := context.Background()
	return m.integrator.TriggerBeastmode(ctx, result)
}

// CreateTemplateInteractive creates a template interactively
func (m *ManagerImpl) CreateTemplateInteractive(name string) error {
	m.logger.Info("Creating template interactively", zap.String("name", name))
	
	// This would typically involve prompting the user for input
	// For now, return a not implemented error
	return &PromptError{
		Type:    ErrorTypeTemplate,
		Message: "interactive template creation not yet implemented",
	}
}

// CreateTemplateFromFile creates a template from a file
func (m *ManagerImpl) CreateTemplateFromFile(name, filePath string) error {
	m.logger.Info("Creating template from file",
		zap.String("name", name),
		zap.String("file", filePath))

	// Load template from file
	template, err := m.templateParser.LoadTemplate(filePath)
	if err != nil {
		return fmt.Errorf("failed to load template from file: %w", err)
	}

	// Set name and update metadata
	template.Name = name
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	// Save to custom templates directory
	customPath := filepath.Join(m.config.CustomTemplatesPath, name+".yaml")
	if err := m.templateParser.SaveTemplate(template, customPath); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	m.logger.Info("Template created successfully",
		zap.String("name", name),
		zap.String("path", customPath))

	return nil
}

// UpdateTemplate updates an existing template
func (m *ManagerImpl) UpdateTemplate(name string, interactive, validate bool) error {
	m.logger.Info("Updating template",
		zap.String("name", name),
		zap.Bool("interactive", interactive))

	return &PromptError{
		Type:    ErrorTypeTemplate,
		Message: "template update not yet implemented",
	}
}

// DeleteTemplate deletes a template
func (m *ManagerImpl) DeleteTemplate(name string) error {
	m.logger.Info("Deleting template", zap.String("name", name))

	// Only allow deletion of custom templates
	customPath := filepath.Join(m.config.CustomTemplatesPath, name+".yaml")
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		return &PromptError{
			Type:    ErrorTypeTemplate,
			Message: fmt.Sprintf("custom template '%s' not found", name),
		}
	}

	// Create backup before deletion
	if m.config.BackupEnabled {
		backupPath := customPath + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(customPath, backupPath); err != nil {
			m.logger.Warn("Failed to create backup", zap.Error(err))
		}
	}

	// Delete the file
	if err := os.Remove(customPath); err != nil {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	m.logger.Info("Template deleted successfully", zap.String("name", name))
	return nil
}

// Helper methods

func (m *ManagerImpl) ensureDirectories() error {
	dirs := append(m.config.TemplateDirectories, m.config.CustomTemplatesPath)
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (m *ManagerImpl) loadTemplatesFromDirectory(ctx context.Context, dir, category string) ([]Template, error) {
	var templates []Template

	files, err := m.templateParser.ListTemplateFiles(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		template, err := m.templateParser.LoadTemplate(file)
		if err != nil {
			m.logger.Warn("Failed to load template", zap.String("file", file), zap.Error(err))
			continue
		}

		// Filter by category if specified
		if category != "" && template.Category != category {
			continue
		}

		templates = append(templates, *template)
	}

	return templates, nil
}

func (m *ManagerImpl) findTemplateFile(dir, name string) string {
	// Try different extensions
	extensions := []string{".yaml", ".yml", ".json", ".md"}
	
	for _, ext := range extensions {
		path := filepath.Join(dir, name+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Try in subdirectories by category
	categories := []string{CategoryGeneral, CategoryLanguages, CategoryWorkflows, CategoryWorkspace}
	for _, category := range categories {
		for _, ext := range extensions {
			path := filepath.Join(dir, category, name+ext)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

func (m *ManagerImpl) calculateTemplateRelevance(template Template, context *WorkspaceContext) float64 {
	var score float64

	// Language match
	if template.Language != "" && template.Language == context.Language {
		score += 0.4
	}

	// Framework match
	if template.Framework != "" && template.Framework == context.Framework {
		score += 0.3
	}

	// Repository-specific templates
	if strings.Contains(template.Name, context.Repository) ||
		strings.Contains(template.Description, context.Repository) {
		score += 0.2
	}

	// Recent file patterns
	for _, file := range context.RecentFiles {
		if strings.Contains(template.Description, filepath.Ext(file)) {
			score += 0.1
			break
		}
	}

	return score
}

func (m *ManagerImpl) generateSuggestionReason(template Template, context *WorkspaceContext) string {
	var reasons []string

	if template.Language == context.Language {
		reasons = append(reasons, fmt.Sprintf("matches %s language", context.Language))
	}

	if template.Framework == context.Framework {
		reasons = append(reasons, fmt.Sprintf("matches %s framework", context.Framework))
	}

	if strings.Contains(template.Name, context.Repository) {
		reasons = append(reasons, fmt.Sprintf("specific to %s repository", context.Repository))
	}

	if len(reasons) == 0 {
		return "general development template"
	}

	return strings.Join(reasons, ", ")
}

func (m *ManagerImpl) validateGeneratedPrompt(content string, template *Template) ValidationStatus {
	score, issues, warnings := m.validator.ValidateContent(content)
	
	return ValidationStatus{
		Valid:    len(issues) == 0,
		Score:    score,
		Issues:   issues,
		Warnings: warnings,
	}
}

func (m *ManagerImpl) loadConfig() error {
	// Implementation for loading saved configuration
	// This would typically read from a config file
	return nil
}

func (m *ManagerImpl) saveConfig() error {
	// Implementation for saving configuration
	// This would typically write to a config file
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}