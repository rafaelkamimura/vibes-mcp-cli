package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ConfigManager handles prompt system configuration persistence and management
type ConfigManager struct {
	logger     *zap.Logger
	configPath string
	config     *Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string, logger *zap.Logger) *ConfigManager {
	return &ConfigManager{
		logger:     logger,
		configPath: configPath,
	}
}

// LoadConfig loads configuration from file or creates default
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	cm.logger.Debug("Loading configuration", zap.String("path", cm.configPath))

	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		cm.logger.Info("Configuration file not found, creating default config")
		config := cm.createDefaultConfig()
		if err := cm.SaveConfig(config); err != nil {
			cm.logger.Warn("Failed to save default config", zap.Error(err))
		}
		cm.config = config
		return config, nil
	}

	// Read existing config
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to read configuration file",
			Cause:   err,
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to parse configuration file",
			Cause:   err,
		}
	}

	// Validate and migrate config if needed
	if err := cm.validateAndMigrateConfig(&config); err != nil {
		cm.logger.Warn("Configuration validation failed", zap.Error(err))
		// Continue with possibly invalid config rather than failing
	}

	cm.config = &config
	cm.logger.Info("Configuration loaded successfully",
		zap.String("preferred_language", config.PreferredLanguage),
		zap.String("default_repository", config.DefaultRepository))

	return &config, nil
}

// SaveConfig saves configuration to file
func (cm *ConfigManager) SaveConfig(config *Config) error {
	cm.logger.Debug("Saving configuration", zap.String("path", cm.configPath))

	// Update last modified time
	config.LastUpdated = time.Now()

	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to create configuration directory",
			Cause:   err,
		}
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to marshal configuration",
			Cause:   err,
		}
	}

	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to write configuration file",
			Cause:   err,
		}
	}

	cm.config = config
	cm.logger.Info("Configuration saved successfully")
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	if cm.config == nil {
		// Load config if not loaded
		config, err := cm.LoadConfig()
		if err != nil {
			cm.logger.Error("Failed to load config, using defaults", zap.Error(err))
			return cm.createDefaultConfig()
		}
		return config
	}
	return cm.config
}

// UpdateConfig updates a configuration value
func (cm *ConfigManager) UpdateConfig(key, value string) error {
	cm.logger.Debug("Updating configuration",
		zap.String("key", key),
		zap.String("value", value))

	config := cm.GetConfig()

	switch key {
	case "default_repository", "default-repository":
		config.DefaultRepository = value
	case "preferred_language", "preferred-language":
		if !cm.isValidLanguage(value) {
			return &PromptError{
				Type:    ErrorTypeConfig,
				Message: fmt.Sprintf("invalid language: %s", value),
			}
		}
		config.PreferredLanguage = value
	case "preferred_framework", "preferred-framework":
		config.PreferredFramework = value
	case "auto_clipboard", "auto-clipboard":
		config.AutoClipboard = cm.parseBool(value)
	case "auto_validate", "auto-validate":
		config.AutoValidate = cm.parseBool(value)
	case "preferred_ai_tool", "preferred-ai-tool":
		if !cm.isValidAITool(value) {
			return &PromptError{
				Type:    ErrorTypeConfig,
				Message: fmt.Sprintf("invalid AI tool: %s", value),
			}
		}
		config.PreferredAITool = value
	case "output_format", "output-format":
		if !cm.isValidOutputFormat(value) {
			return &PromptError{
				Type:    ErrorTypeConfig,
				Message: fmt.Sprintf("invalid output format: %s", value),
			}
		}
		config.OutputFormat = value
	case "history_limit", "history-limit":
		if limit, err := cm.parseInt(value); err != nil {
			return &PromptError{
				Type:    ErrorTypeConfig,
				Message: fmt.Sprintf("invalid history limit: %s", value),
				Cause:   err,
			}
		} else {
			config.HistoryLimit = limit
		}
	case "validation_enabled", "validation-enabled":
		config.ValidationEnabled = cm.parseBool(value)
	case "backup_enabled", "backup-enabled":
		config.BackupEnabled = cm.parseBool(value)
	case "custom_templates_path", "custom-templates-path":
		if !filepath.IsAbs(value) {
			return &PromptError{
				Type:    ErrorTypeConfig,
				Message: "custom templates path must be absolute",
			}
		}
		config.CustomTemplatesPath = value
	default:
		// Handle integration settings
		if key[:12] == "integration." {
			integrationKey := key[12:]
			if config.IntegrationSettings == nil {
				config.IntegrationSettings = make(map[string]string)
			}
			config.IntegrationSettings[integrationKey] = value
		} else {
			return &PromptError{
				Type:    ErrorTypeConfig,
				Message: fmt.Sprintf("unknown configuration key: %s", key),
			}
		}
	}

	return cm.SaveConfig(config)
}

// GetConfigValue gets a configuration value
func (cm *ConfigManager) GetConfigValue(key string) string {
	config := cm.GetConfig()

	switch key {
	case "default_repository", "default-repository":
		return config.DefaultRepository
	case "preferred_language", "preferred-language":
		return config.PreferredLanguage
	case "preferred_framework", "preferred-framework":
		return config.PreferredFramework
	case "auto_clipboard", "auto-clipboard":
		return cm.formatBool(config.AutoClipboard)
	case "auto_validate", "auto-validate":
		return cm.formatBool(config.AutoValidate)
	case "preferred_ai_tool", "preferred-ai-tool":
		return config.PreferredAITool
	case "output_format", "output-format":
		return config.OutputFormat
	case "history_limit", "history-limit":
		return fmt.Sprintf("%d", config.HistoryLimit)
	case "validation_enabled", "validation-enabled":
		return cm.formatBool(config.ValidationEnabled)
	case "backup_enabled", "backup-enabled":
		return cm.formatBool(config.BackupEnabled)
	case "custom_templates_path", "custom-templates-path":
		return config.CustomTemplatesPath
	default:
		// Handle integration settings
		if len(key) > 12 && key[:12] == "integration." {
			integrationKey := key[12:]
			if config.IntegrationSettings != nil {
				if value, exists := config.IntegrationSettings[integrationKey]; exists {
					return value
				}
			}
		}
		return ""
	}
}

// ListAllSettings returns all configuration settings
func (cm *ConfigManager) ListAllSettings() map[string]string {
	config := cm.GetConfig()
	settings := make(map[string]string)

	settings["default_repository"] = config.DefaultRepository
	settings["preferred_language"] = config.PreferredLanguage
	settings["preferred_framework"] = config.PreferredFramework
	settings["auto_clipboard"] = cm.formatBool(config.AutoClipboard)
	settings["auto_validate"] = cm.formatBool(config.AutoValidate)
	settings["preferred_ai_tool"] = config.PreferredAITool
	settings["output_format"] = config.OutputFormat
	settings["history_limit"] = fmt.Sprintf("%d", config.HistoryLimit)
	settings["validation_enabled"] = cm.formatBool(config.ValidationEnabled)
	settings["backup_enabled"] = cm.formatBool(config.BackupEnabled)
	settings["custom_templates_path"] = config.CustomTemplatesPath

	// Add integration settings
	for key, value := range config.IntegrationSettings {
		settings["integration."+key] = value
	}

	return settings
}

// ResetToDefaults resets configuration to default values
func (cm *ConfigManager) ResetToDefaults() error {
	cm.logger.Info("Resetting configuration to defaults")

	defaultConfig := cm.createDefaultConfig()
	return cm.SaveConfig(defaultConfig)
}

// ValidateConfig validates the current configuration
func (cm *ConfigManager) ValidateConfig() (bool, []string) {
	config := cm.GetConfig()
	return cm.validateConfigStruct(config)
}

// AddTemplateDirectory adds a template directory to the search paths
func (cm *ConfigManager) AddTemplateDirectory(directory string) error {
	config := cm.GetConfig()

	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: fmt.Sprintf("directory does not exist: %s", directory),
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to resolve absolute path",
			Cause:   err,
		}
	}

	// Check if already in list
	for _, existingDir := range config.TemplateDirectories {
		if existingDir == absPath {
			return nil // Already exists
		}
	}

	// Add to list
	config.TemplateDirectories = append(config.TemplateDirectories, absPath)
	return cm.SaveConfig(config)
}

// RemoveTemplateDirectory removes a template directory from search paths
func (cm *ConfigManager) RemoveTemplateDirectory(directory string) error {
	config := cm.GetConfig()

	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(directory)
	if err != nil {
		absPath = directory // Use as-is if conversion fails
	}

	// Find and remove directory
	var newDirs []string
	removed := false
	for _, existingDir := range config.TemplateDirectories {
		if existingDir != absPath && existingDir != directory {
			newDirs = append(newDirs, existingDir)
		} else {
			removed = true
		}
	}

	if !removed {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "directory not found in template directories",
		}
	}

	config.TemplateDirectories = newDirs
	return cm.SaveConfig(config)
}

// ExportConfig exports configuration to a file
func (cm *ConfigManager) ExportConfig(filePath string) error {
	config := cm.GetConfig()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to marshal configuration for export",
			Cause:   err,
		}
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to write exported configuration",
			Cause:   err,
		}
	}

	cm.logger.Info("Configuration exported successfully", zap.String("path", filePath))
	return nil
}

// ImportConfig imports configuration from a file
func (cm *ConfigManager) ImportConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to read configuration file for import",
			Cause:   err,
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: "failed to parse imported configuration",
			Cause:   err,
		}
	}

	// Validate imported config
	if valid, issues := cm.validateConfigStruct(&config); !valid {
		return &PromptError{
			Type:    ErrorTypeConfig,
			Message: fmt.Sprintf("imported configuration is invalid: %s", issues[0]),
		}
	}

	// Save imported config
	if err := cm.SaveConfig(&config); err != nil {
		return err
	}

	cm.logger.Info("Configuration imported successfully", zap.String("path", filePath))
	return nil
}

// Helper methods

func (cm *ConfigManager) createDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	
	return &Config{
		DefaultRepository:     "vibes-mcp-cli",
		PreferredLanguage:     "go",
		PreferredFramework:    "",
		AutoClipboard:         false,
		AutoValidate:          true,
		PreferredAITool:       AIToolClaude,
		OutputFormat:          FormatMarkdown,
		HistoryLimit:          100,
		ValidationEnabled:     true,
		BackupEnabled:         true,
		IntegrationSettings:   make(map[string]string),
		LastUpdated:           time.Now(),
		TemplateDirectories: []string{
			filepath.Join(homeDir, ".vibes-mcp-cli", "templates"),
			"./prompt-templates",
			"./templates",
		},
		CustomTemplatesPath: filepath.Join(homeDir, ".vibes-mcp-cli", "custom-templates"),
	}
}

func (cm *ConfigManager) validateAndMigrateConfig(config *Config) error {
	var migrated bool

	// Ensure required fields have defaults
	if config.DefaultRepository == "" {
		config.DefaultRepository = "vibes-mcp-cli"
		migrated = true
	}

	if config.PreferredLanguage == "" {
		config.PreferredLanguage = "go"
		migrated = true
	}

	if config.PreferredAITool == "" {
		config.PreferredAITool = AIToolClaude
		migrated = true
	}

	if config.OutputFormat == "" {
		config.OutputFormat = FormatMarkdown
		migrated = true
	}

	if config.HistoryLimit <= 0 {
		config.HistoryLimit = 100
		migrated = true
	}

	if config.IntegrationSettings == nil {
		config.IntegrationSettings = make(map[string]string)
		migrated = true
	}

	if len(config.TemplateDirectories) == 0 {
		homeDir, _ := os.UserHomeDir()
		config.TemplateDirectories = []string{
			filepath.Join(homeDir, ".vibes-mcp-cli", "templates"),
			"./prompt-templates",
			"./templates",
		}
		migrated = true
	}

	if config.CustomTemplatesPath == "" {
		homeDir, _ := os.UserHomeDir()
		config.CustomTemplatesPath = filepath.Join(homeDir, ".vibes-mcp-cli", "custom-templates")
		migrated = true
	}

	// Save migrated config
	if migrated {
		cm.logger.Info("Configuration migrated to current version")
		return cm.SaveConfig(config)
	}

	return nil
}

func (cm *ConfigManager) validateConfigStruct(config *Config) (bool, []string) {
	var issues []string

	// Validate required fields
	if config.DefaultRepository == "" {
		issues = append(issues, "default repository is required")
	}

	if config.PreferredLanguage != "" && !cm.isValidLanguage(config.PreferredLanguage) {
		issues = append(issues, fmt.Sprintf("invalid preferred language: %s", config.PreferredLanguage))
	}

	if config.PreferredAITool != "" && !cm.isValidAITool(config.PreferredAITool) {
		issues = append(issues, fmt.Sprintf("invalid preferred AI tool: %s", config.PreferredAITool))
	}

	if config.OutputFormat != "" && !cm.isValidOutputFormat(config.OutputFormat) {
		issues = append(issues, fmt.Sprintf("invalid output format: %s", config.OutputFormat))
	}

	if config.HistoryLimit < 0 || config.HistoryLimit > 10000 {
		issues = append(issues, "history limit must be between 0 and 10000")
	}

	if config.CustomTemplatesPath != "" && !filepath.IsAbs(config.CustomTemplatesPath) {
		issues = append(issues, "custom templates path must be absolute")
	}

	// Validate template directories
	for _, dir := range config.TemplateDirectories {
		if !filepath.IsAbs(dir) && !strings.HasPrefix(dir, "./") {
			issues = append(issues, fmt.Sprintf("template directory must be absolute or relative: %s", dir))
		}
	}

	return len(issues) == 0, issues
}

func (cm *ConfigManager) isValidLanguage(language string) bool {
	validLanguages := []string{
		"go", "javascript", "typescript", "python", "java", "rust",
		"c", "cpp", "php", "ruby", "swift", "kotlin", "csharp",
		"scala", "clojure", "perl", "shell", "bash",
	}

	for _, valid := range validLanguages {
		if language == valid {
			return true
		}
	}
	return false
}

func (cm *ConfigManager) isValidAITool(tool string) bool {
	validTools := []string{AIToolClaude, AIToolContext7, AIToolBeastmode}
	for _, valid := range validTools {
		if tool == valid {
			return true
		}
	}
	return false
}

func (cm *ConfigManager) isValidOutputFormat(format string) bool {
	validFormats := []string{FormatMarkdown, FormatText, FormatJSON, FormatHTML}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}

func (cm *ConfigManager) parseBool(value string) bool {
	switch strings.ToLower(value) {
	case "true", "yes", "y", "1", "on", "enabled":
		return true
	default:
		return false
	}
}

func (cm *ConfigManager) formatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (cm *ConfigManager) parseInt(value string) (int, error) {
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err
}