package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// TemplateParserImpl implements the TemplateParser interface
type TemplateParserImpl struct {
	logger *zap.Logger
}

// NewTemplateParser creates a new template parser
func NewTemplateParser(logger *zap.Logger) TemplateParser {
	return &TemplateParserImpl{
		logger: logger,
	}
}

// LoadTemplate loads a template from a file
func (p *TemplateParserImpl) LoadTemplate(filePath string) (*Template, error) {
	p.logger.Debug("Loading template", zap.String("path", filePath))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: fmt.Sprintf("template file not found: %s", filePath),
			Cause:   err,
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to read template file",
			Cause:   err,
		}
	}

	// Determine file format and parse
	ext := strings.ToLower(filepath.Ext(filePath))
	var template *Template

	switch ext {
	case ".yaml", ".yml":
		template, err = p.parseYAMLTemplate(content, filePath)
	case ".json":
		template, err = p.parseJSONTemplate(content, filePath)
	case ".md":
		template, err = p.parseMarkdownTemplate(content, filePath)
	default:
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: fmt.Sprintf("unsupported template format: %s", ext),
		}
	}

	if err != nil {
		return nil, err
	}

	// Set file path and update metadata
	template.FilePath = filePath
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}
	if template.UpdatedAt.IsZero() {
		template.UpdatedAt = time.Now()
	}

	// Validate template structure
	if err := p.ValidateStructure(template); err != nil {
		return nil, err
	}

	p.logger.Info("Template loaded successfully",
		zap.String("name", template.Name),
		zap.String("category", template.Category),
		zap.String("path", filePath))

	return template, nil
}

// SaveTemplate saves a template to a file
func (p *TemplateParserImpl) SaveTemplate(template *Template, filePath string) error {
	p.logger.Debug("Saving template",
		zap.String("name", template.Name),
		zap.String("path", filePath))

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Update metadata
	template.UpdatedAt = time.Now()
	template.FilePath = filePath

	// Determine format from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	var content []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		content, err = yaml.Marshal(template)
	case ".json":
		content, err = json.MarshalIndent(template, "", "  ")
	default:
		return &PromptError{
			Type:    ErrorTypeTemplate,
			Message: fmt.Sprintf("unsupported save format: %s", ext),
		}
	}

	if err != nil {
		return &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to marshal template",
			Cause:   err,
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to write template file",
			Cause:   err,
		}
	}

	p.logger.Info("Template saved successfully",
		zap.String("name", template.Name),
		zap.String("path", filePath))

	return nil
}

// ParseContent parses template content from a string
func (p *TemplateParserImpl) ParseContent(content, templateName string) (*Template, error) {
	p.logger.Debug("Parsing template content", zap.String("name", templateName))

	// Try to determine format from content
	content = strings.TrimSpace(content)
	
	var template *Template
	var err error

	// Try YAML first
	if strings.HasPrefix(content, "name:") || strings.HasPrefix(content, "---") {
		template, err = p.parseYAMLTemplate([]byte(content), "")
	} else if strings.HasPrefix(content, "{") {
		// Try JSON
		template, err = p.parseJSONTemplate([]byte(content), "")
	} else {
		// Assume it's plain content and create a basic template
		template = &Template{
			Name:        templateName,
			Category:    CategoryGeneral,
			Description: "Generated template",
			Content:     content,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	if err != nil {
		return nil, err
	}

	// Override name if provided
	if templateName != "" {
		template.Name = templateName
	}

	return template, nil
}

// ValidateStructure validates template structure
func (p *TemplateParserImpl) ValidateStructure(template *Template) error {
	p.logger.Debug("Validating template structure", zap.String("name", template.Name))

	var issues []string

	// Required fields
	if template.Name == "" {
		issues = append(issues, "template name is required")
	}
	if template.Category == "" {
		issues = append(issues, "template category is required")
	}
	if template.Description == "" {
		issues = append(issues, "template description is required")
	}
	if template.Content == "" {
		issues = append(issues, "template content is required")
	}

	// Validate category
	validCategories := []string{CategoryGeneral, CategoryLanguages, CategoryWorkflows, CategoryWorkspace, CategoryCustom}
	validCategory := false
	for _, cat := range validCategories {
		if template.Category == cat {
			validCategory = true
			break
		}
	}
	if !validCategory {
		issues = append(issues, fmt.Sprintf("invalid category '%s', must be one of: %s",
			template.Category, strings.Join(validCategories, ", ")))
	}

	// Validate parameters
	paramNames := make(map[string]bool)
	for i, param := range template.Parameters {
		if param.Name == "" {
			issues = append(issues, fmt.Sprintf("parameter %d is missing name", i))
			continue
		}
		
		if paramNames[param.Name] {
			issues = append(issues, fmt.Sprintf("duplicate parameter name: %s", param.Name))
		}
		paramNames[param.Name] = true

		if param.Description == "" {
			issues = append(issues, fmt.Sprintf("parameter '%s' is missing description", param.Name))
		}

		// Validate parameter type
		validTypes := []string{"string", "int", "bool", "select"}
		validType := false
		for _, t := range validTypes {
			if param.Type == t {
				validType = true
				break
			}
		}
		if !validType {
			issues = append(issues, fmt.Sprintf("parameter '%s' has invalid type '%s'", param.Name, param.Type))
		}

		// For select type, options are required
		if param.Type == "select" && len(param.Options) == 0 {
			issues = append(issues, fmt.Sprintf("parameter '%s' of type 'select' requires options", param.Name))
		}

		// Validate regex if provided
		if param.Validation != "" {
			if _, err := regexp.Compile(param.Validation); err != nil {
				issues = append(issues, fmt.Sprintf("parameter '%s' has invalid validation regex: %s", param.Name, err.Error()))
			}
		}
	}

	// Check for placeholder consistency
	placeholderPattern := regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)
	placeholders := placeholderPattern.FindAllStringSubmatch(template.Content, -1)
	
	for _, match := range placeholders {
		paramName := match[1]
		if !paramNames[paramName] && !isBuiltinPlaceholder(paramName) {
			issues = append(issues, fmt.Sprintf("content references undefined parameter: %s", paramName))
		}
	}

	if len(issues) > 0 {
		return &PromptError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("template structure validation failed: %s", strings.Join(issues, "; ")),
		}
	}

	return nil
}

// ListTemplateFiles lists template files in a directory
func (p *TemplateParserImpl) ListTemplateFiles(directory string) ([]string, error) {
	p.logger.Debug("Listing template files", zap.String("directory", directory))

	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".md" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to list template files",
			Cause:   err,
		}
	}

	p.logger.Debug("Template files listed",
		zap.String("directory", directory),
		zap.Int("count", len(files)))

	return files, nil
}

// Helper methods for parsing different formats

func (p *TemplateParserImpl) parseYAMLTemplate(content []byte, filePath string) (*Template, error) {
	var template Template
	if err := yaml.Unmarshal(content, &template); err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to parse YAML template",
			Cause:   err,
		}
	}
	return &template, nil
}

func (p *TemplateParserImpl) parseJSONTemplate(content []byte, filePath string) (*Template, error) {
	var template Template
	if err := json.Unmarshal(content, &template); err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to parse JSON template",
			Cause:   err,
		}
	}
	return &template, nil
}

func (p *TemplateParserImpl) parseMarkdownTemplate(content []byte, filePath string) (*Template, error) {
	// Parse markdown template with frontmatter
	contentStr := string(content)
	
	// Check for frontmatter
	if !strings.HasPrefix(contentStr, "---") {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "markdown template must start with YAML frontmatter",
		}
	}

	// Split frontmatter and content
	parts := strings.SplitN(contentStr, "---", 3)
	if len(parts) < 3 {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "invalid markdown template format",
		}
	}

	// Parse frontmatter
	frontmatter := strings.TrimSpace(parts[1])
	var template Template
	if err := yaml.Unmarshal([]byte(frontmatter), &template); err != nil {
		return nil, &PromptError{
			Type:    ErrorTypeTemplate,
			Message: "failed to parse template frontmatter",
			Cause:   err,
		}
	}

	// Set content (everything after second ---)
	template.Content = strings.TrimSpace(parts[2])

	return &template, nil
}

func isBuiltinPlaceholder(name string) bool {
	builtinPlaceholders := []string{
		"timestamp",
		"date",
		"time",
		"user",
		"hostname",
		"pwd",
		"repo",
		"branch",
		"language",
		"framework",
	}

	for _, builtin := range builtinPlaceholders {
		if name == builtin {
			return true
		}
	}
	return false
}

// TemplateRenderer handles template rendering with parameters
type TemplateRenderer struct {
	logger *zap.Logger
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(logger *zap.Logger) *TemplateRenderer {
	return &TemplateRenderer{
		logger: logger,
	}
}

// Render renders a template with the given parameters
func (r *TemplateRenderer) Render(template *Template, parameters map[string]string, context *WorkspaceContext) (string, error) {
	r.logger.Debug("Rendering template",
		zap.String("name", template.Name),
		zap.Int("parameters", len(parameters)))

	content := template.Content

	// Add builtin parameters
	builtinParams := r.getBuiltinParameters(context)
	for key, value := range builtinParams {
		parameters[key] = value
	}

	// Replace placeholders
	placeholderPattern := regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)
	
	content = placeholderPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract parameter name
		paramName := placeholderPattern.FindStringSubmatch(match)[1]
		
		if value, exists := parameters[paramName]; exists {
			return value
		}
		
		// Log missing parameter but don't fail
		r.logger.Warn("Missing parameter value", zap.String("parameter", paramName))
		return match // Keep original placeholder
	})

	// Handle conditional sections
	content = r.processConditionals(content, parameters)

	// Handle loops/iterations
	content = r.processLoops(content, parameters)

	r.logger.Debug("Template rendered successfully",
		zap.String("name", template.Name),
		zap.Int("length", len(content)))

	return content, nil
}

func (r *TemplateRenderer) getBuiltinParameters(context *WorkspaceContext) map[string]string {
	params := map[string]string{
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"date":      time.Now().Format("2006-01-02"),
		"time":      time.Now().Format("15:04:05"),
	}

	if hostname, err := os.Hostname(); err == nil {
		params["hostname"] = hostname
	}

	if pwd, err := os.Getwd(); err == nil {
		params["pwd"] = pwd
	}

	if context != nil {
		if context.Repository != "" {
			params["repo"] = context.Repository
		}
		if context.Language != "" {
			params["language"] = context.Language
		}
		if context.Framework != "" {
			params["framework"] = context.Framework
		}
		if context.GitBranch != "" {
			params["branch"] = context.GitBranch
		}
	}

	return params
}

func (r *TemplateRenderer) processConditionals(content string, parameters map[string]string) string {
	// Process {{#if condition}} ... {{/if}} blocks
	ifPattern := regexp.MustCompile(`\{\{#if\s+(\w+)\}\}(.*?)\{\{/if\}\}`)
	
	return ifPattern.ReplaceAllStringFunc(content, func(match string) string {
		matches := ifPattern.FindStringSubmatch(match)
		condition := matches[1]
		block := matches[2]
		
		// Check if condition parameter exists and is truthy
		if value, exists := parameters[condition]; exists && r.isTruthy(value) {
			return block
		}
		
		return "" // Remove block if condition is false
	})
}

func (r *TemplateRenderer) processLoops(content string, parameters map[string]string) string {
	// Process {{#each array}} ... {{/each}} blocks
	// This is a simplified implementation
	eachPattern := regexp.MustCompile(`\{\{#each\s+(\w+)\}\}(.*?)\{\{/each\}\}`)
	
	return eachPattern.ReplaceAllStringFunc(content, func(match string) string {
		matches := eachPattern.FindStringSubmatch(match)
		arrayName := matches[1]
		block := matches[2]
		
		// For now, just return the block once
		// In a full implementation, this would iterate over array values
		if _, exists := parameters[arrayName]; exists {
			return block
		}
		
		return ""
	})
}

func (r *TemplateRenderer) isTruthy(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower != "" && lower != "false" && lower != "0" && lower != "no"
}