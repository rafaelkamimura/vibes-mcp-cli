package prompt

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// GeneratorImpl implements the Generator interface
type GeneratorImpl struct {
	logger         *zap.Logger
	templateParser TemplateParser
	renderer       *TemplateRenderer
}

// NewGenerator creates a new generator
func NewGenerator(logger *zap.Logger) Generator {
	return &GeneratorImpl{
		logger:         logger,
		templateParser: NewTemplateParser(logger),
		renderer:       NewTemplateRenderer(logger),
	}
}

// Generate generates a prompt from a template configuration
func (g *GeneratorImpl) Generate(ctx context.Context, config *GenerationConfig) (*GenerationResult, error) {
	g.logger.Info("Starting prompt generation",
		zap.String("template", config.TemplateName),
		zap.Bool("interactive", config.Interactive))

	startTime := time.Now()

	// Load template
	template, err := g.loadTemplate(config.TemplateName)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Prepare parameters
	parameters := make(map[string]string)
	for k, v := range config.Parameters {
		if v != "" {
			parameters[k] = v
		}
	}

	// Handle interactive mode
	if config.Interactive {
		interactiveParams, err := g.ProcessInteractive(template, config.Context)
		if err != nil {
			return nil, fmt.Errorf("interactive processing failed: %w", err)
		}
		
		// Merge interactive parameters with provided ones (interactive takes precedence)
		for k, v := range interactiveParams {
			parameters[k] = v
		}
	}

	// Fill in missing required parameters with defaults or context
	if err := g.fillMissingParameters(template, parameters, config.Context); err != nil {
		return nil, fmt.Errorf("failed to fill missing parameters: %w", err)
	}

	// Generate content
	content, err := g.FillTemplate(template, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to fill template: %w", err)
	}

	// Format output if specified
	if config.OutputFormat != "" {
		content, err = g.FormatOutput(content, config.OutputFormat)
		if err != nil {
			g.logger.Warn("Failed to format output, using original content", zap.Error(err))
		}
	}

	// Calculate statistics
	wordCount, charCount := g.CalculateStats(content)

	// Create result
	result := &GenerationResult{
		Content:     content,
		Template:    *template,
		Parameters:  parameters,
		GeneratedAt: startTime,
		Context:     config.Context,
		WordCount:   wordCount,
		CharCount:   charCount,
		Metadata:    config.Metadata,
		ValidationStatus: ValidationStatus{
			Valid: true,
			Score: 100,
		},
	}

	g.logger.Info("Prompt generation completed",
		zap.String("template", config.TemplateName),
		zap.Int("word_count", wordCount),
		zap.Int("char_count", charCount),
		zap.Duration("duration", time.Since(startTime)))

	return result, nil
}

// FillTemplate fills a template with parameters
func (g *GeneratorImpl) FillTemplate(template *Template, parameters map[string]string) (string, error) {
	g.logger.Debug("Filling template",
		zap.String("name", template.Name),
		zap.Int("parameters", len(parameters)))

	// Use the template renderer to fill the template
	context := &WorkspaceContext{} // Empty context for now
	if repoName, exists := parameters["repo"]; exists {
		context.Repository = repoName
	}
	if language, exists := parameters["language"]; exists {
		context.Language = language
	}
	if framework, exists := parameters["framework"]; exists {
		context.Framework = framework
	}

	content, err := g.renderer.Render(template, parameters, context)
	if err != nil {
		return "", &PromptError{
			Type:    ErrorTypeGeneration,
			Message: "failed to render template",
			Cause:   err,
		}
	}

	// Post-process content
	content = g.postProcessContent(content, parameters)

	return content, nil
}

// ProcessInteractive handles interactive parameter collection
func (g *GeneratorImpl) ProcessInteractive(template *Template, context *WorkspaceContext) (map[string]string, error) {
	g.logger.Info("Starting interactive parameter collection",
		zap.String("template", template.Name),
		zap.Int("parameters", len(template.Parameters)))

	parameters := make(map[string]string)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("\n🎯 Interactive Template: %s\n", template.Name)
	fmt.Printf("📝 Description: %s\n\n", template.Description)

	// Collect parameters interactively
	for _, param := range template.Parameters {
		value, err := g.collectParameterInteractively(param, context, scanner)
		if err != nil {
			return nil, fmt.Errorf("failed to collect parameter %s: %w", param.Name, err)
		}
		if value != "" {
			parameters[param.Name] = value
		}
	}

	// Ask for any additional context
	fmt.Println("\n💡 Any additional context or requirements? (press Enter to skip)")
	fmt.Print("Additional context: ")
	if scanner.Scan() {
		additionalContext := strings.TrimSpace(scanner.Text())
		if additionalContext != "" {
			parameters["additional_context"] = additionalContext
		}
	}

	fmt.Printf("\n✅ Collected %d parameters\n\n", len(parameters))

	return parameters, nil
}

// FormatOutput formats the output according to the specified format
func (g *GeneratorImpl) FormatOutput(content, format string) (string, error) {
	g.logger.Debug("Formatting output", zap.String("format", format))

	switch strings.ToLower(format) {
	case FormatMarkdown:
		return g.formatAsMarkdown(content), nil
	case FormatText:
		return g.formatAsText(content), nil
	case FormatJSON:
		return g.formatAsJSON(content), nil
	case FormatHTML:
		return g.formatAsHTML(content), nil
	default:
		return content, &PromptError{
			Type:    ErrorTypeGeneration,
			Message: fmt.Sprintf("unsupported output format: %s", format),
		}
	}
}

// CalculateStats calculates word and character counts
func (g *GeneratorImpl) CalculateStats(content string) (int, int) {
	charCount := len(content)
	
	// Count words (split by whitespace and filter empty strings)
	words := strings.Fields(content)
	wordCount := len(words)

	return wordCount, charCount
}

// Helper methods

func (g *GeneratorImpl) loadTemplate(templateName string) (*Template, error) {
	// This would typically use the manager to load the template
	// For now, we'll create a simple implementation
	// In practice, this would call manager.GetTemplate(templateName)
	
	return &Template{
		Name:        templateName,
		Category:    CategoryGeneral,
		Description: fmt.Sprintf("Template for %s", templateName),
		Content:     "Default template content for {{.templateName}}",
		Parameters:  []TemplateParameter{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (g *GeneratorImpl) fillMissingParameters(template *Template, parameters map[string]string, context *WorkspaceContext) error {
	for _, param := range template.Parameters {
		// Skip if parameter already provided
		if _, exists := parameters[param.Name]; exists {
			continue
		}

		// Try to fill from context
		contextValue := g.getParameterFromContext(param.Name, context)
		if contextValue != "" {
			parameters[param.Name] = contextValue
			g.logger.Debug("Filled parameter from context",
				zap.String("parameter", param.Name),
				zap.String("value", contextValue))
			continue
		}

		// Use default value if available
		if param.Default != "" {
			parameters[param.Name] = param.Default
			g.logger.Debug("Used default parameter value",
				zap.String("parameter", param.Name),
				zap.String("value", param.Default))
			continue
		}

		// If required and still missing, return error
		if param.Required {
			return &PromptError{
				Type:    ErrorTypeGeneration,
				Message: fmt.Sprintf("required parameter '%s' is missing", param.Name),
			}
		}
	}

	return nil
}

func (g *GeneratorImpl) getParameterFromContext(paramName string, context *WorkspaceContext) string {
	if context == nil {
		return ""
	}

	switch paramName {
	case "repo", "repository":
		return context.Repository
	case "language", "lang":
		return context.Language
	case "framework":
		return context.Framework
	case "branch":
		return context.GitBranch
	case "pwd", "directory":
		return context.WorkingDirectory
	default:
		return ""
	}
}

func (g *GeneratorImpl) collectParameterInteractively(param TemplateParameter, context *WorkspaceContext, scanner *bufio.Scanner) (string, error) {
	// Show parameter info
	fmt.Printf("📋 Parameter: %s\n", param.Name)
	fmt.Printf("   Description: %s\n", param.Description)
	
	if param.Type != "" && param.Type != "string" {
		fmt.Printf("   Type: %s\n", param.Type)
	}

	// Show options for select type
	if param.Type == "select" && len(param.Options) > 0 {
		fmt.Printf("   Options: %s\n", strings.Join(param.Options, ", "))
	}

	// Show default value
	defaultValue := param.Default
	if defaultValue == "" {
		defaultValue = g.getParameterFromContext(param.Name, context)
	}
	
	if defaultValue != "" {
		fmt.Printf("   Default: %s\n", defaultValue)
	}

	// Show placeholder
	placeholder := param.Placeholder
	if placeholder == "" {
		placeholder = fmt.Sprintf("Enter %s", param.Name)
	}

	// Prompt for input
	if param.Required {
		fmt.Printf("   %s (required): ", placeholder)
	} else {
		fmt.Printf("   %s (optional): ", placeholder)
	}

	// Read input
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read input")
	}

	value := strings.TrimSpace(scanner.Text())

	// Use default if empty
	if value == "" {
		value = defaultValue
	}

	// Validate input
	if err := g.validateParameterValue(param, value); err != nil {
		fmt.Printf("   ❌ %s\n", err.Error())
		fmt.Println("   Please try again:")
		return g.collectParameterInteractively(param, context, scanner)
	}

	if value != "" {
		fmt.Printf("   ✅ Set %s = %s\n", param.Name, value)
	}
	fmt.Println()

	return value, nil
}

func (g *GeneratorImpl) validateParameterValue(param TemplateParameter, value string) error {
	// Skip validation for empty optional parameters
	if value == "" && !param.Required {
		return nil
	}

	// Check required parameters
	if param.Required && value == "" {
		return fmt.Errorf("parameter '%s' is required", param.Name)
	}

	// Type validation
	switch param.Type {
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("parameter '%s' must be an integer", param.Name)
		}
	case "bool":
		if !g.isBoolValue(value) {
			return fmt.Errorf("parameter '%s' must be a boolean (true/false, yes/no, 1/0)", param.Name)
		}
	case "select":
		if !g.isValidOption(value, param.Options) {
			return fmt.Errorf("parameter '%s' must be one of: %s", param.Name, strings.Join(param.Options, ", "))
		}
	}

	// Regex validation
	if param.Validation != "" {
		matched, err := regexp.MatchString(param.Validation, value)
		if err != nil {
			g.logger.Warn("Invalid validation regex", zap.String("parameter", param.Name), zap.Error(err))
		} else if !matched {
			return fmt.Errorf("parameter '%s' does not match required pattern", param.Name)
		}
	}

	return nil
}

func (g *GeneratorImpl) isBoolValue(value string) bool {
	lower := strings.ToLower(value)
	boolValues := []string{"true", "false", "yes", "no", "y", "n", "1", "0", "on", "off"}
	for _, boolVal := range boolValues {
		if lower == boolVal {
			return true
		}
	}
	return false
}

func (g *GeneratorImpl) isValidOption(value string, options []string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func (g *GeneratorImpl) postProcessContent(content string, parameters map[string]string) string {
	// Remove empty lines at the beginning and end
	content = strings.TrimSpace(content)

	// Fix double spacing
	content = regexp.MustCompile(`\n\n\n+`).ReplaceAllString(content, "\n\n")

	// Remove trailing whitespace from lines
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")

	// Add final newline if not present
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return content
}

func (g *GeneratorImpl) formatAsMarkdown(content string) string {
	// Content is already assumed to be markdown
	// Add proper markdown headers if not present
	if !strings.HasPrefix(content, "#") {
		content = "# Generated Prompt\n\n" + content
	}
	return content
}

func (g *GeneratorImpl) formatAsText(content string) string {
	// Remove markdown formatting
	content = regexp.MustCompile(`^#+\s*`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile(`\[(.*?)\]\(.*?\)`).ReplaceAllString(content, "$1")
	content = regexp.MustCompile("```[\\s\\S]*?```").ReplaceAllString(content, "[code block]")
	content = regexp.MustCompile("`([^`]*)`").ReplaceAllString(content, "$1")
	
	return content
}

func (g *GeneratorImpl) formatAsJSON(content string) string {
	// This would normally use json.Marshal but for simplicity:
	return fmt.Sprintf(`{
  "content": %q,
  "generated_at": %q,
  "format": "json"
}`, content, time.Now().Format(time.RFC3339))
}

func (g *GeneratorImpl) formatAsHTML(content string) string {
	// Convert markdown to HTML (simplified)
	html := content
	
	// Headers
	html = regexp.MustCompile(`^# (.*)`).ReplaceAllString(html, "<h1>$1</h1>")
	html = regexp.MustCompile(`^## (.*)`).ReplaceAllString(html, "<h2>$1</h2>")
	html = regexp.MustCompile(`^### (.*)`).ReplaceAllString(html, "<h3>$1</h3>")
	
	// Bold and italic
	html = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(html, "<strong>$1</strong>")
	html = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(html, "<em>$1</em>")
	
	// Code
	html = regexp.MustCompile("`([^`]*)`").ReplaceAllString(html, "<code>$1</code>")
	
	// Line breaks
	html = strings.ReplaceAll(html, "\n", "<br>\n")
	
	// Wrap in basic HTML structure
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Generated Prompt</title>
    <meta charset="utf-8">
</head>
<body>
%s
</body>
</html>`, html)
}

// PromptSuggestionEngine provides intelligent suggestions during interactive mode
type PromptSuggestionEngine struct {
	logger *zap.Logger
}

// NewPromptSuggestionEngine creates a new suggestion engine
func NewPromptSuggestionEngine(logger *zap.Logger) *PromptSuggestionEngine {
	return &PromptSuggestionEngine{
		logger: logger,
	}
}

// SuggestValue suggests a value for a parameter based on context
func (e *PromptSuggestionEngine) SuggestValue(param TemplateParameter, context *WorkspaceContext) string {
	switch param.Name {
	case "severity":
		return e.suggestSeverity(context)
	case "priority":
		return e.suggestPriority(context)
	case "component":
		return e.suggestComponent(context)
	case "language":
		if context != nil && context.Language != "" {
			return context.Language
		}
		return "go"
	case "framework":
		if context != nil && context.Framework != "" {
			return context.Framework
		}
		return ""
	case "repo", "repository":
		if context != nil && context.Repository != "" {
			return context.Repository
		}
		return "vibes-mcp-cli"
	default:
		return param.Default
	}
}

func (e *PromptSuggestionEngine) suggestSeverity(context *WorkspaceContext) string {
	// Analyze context to suggest severity
	if context == nil {
		return "medium"
	}

	// Check for error-related files or recent changes
	for _, file := range context.RecentFiles {
		if strings.Contains(strings.ToLower(file), "error") ||
			strings.Contains(strings.ToLower(file), "bug") ||
			strings.Contains(strings.ToLower(file), "fix") {
			return "high"
		}
	}

	// Check git status
	if context.GitStatus == "dirty" {
		return "medium"
	}

	return "low"
}

func (e *PromptSuggestionEngine) suggestPriority(context *WorkspaceContext) string {
	// Default priority mapping
	return "p2"
}

func (e *PromptSuggestionEngine) suggestComponent(context *WorkspaceContext) string {
	if context == nil {
		return ""
	}
	
	// Analyze recent files to suggest component
	for _, file := range context.RecentFiles {
		if strings.Contains(file, "/") {
			parts := strings.Split(file, "/")
			if len(parts) > 1 {
				return parts[0] // Return first directory as component
			}
		}
	}
	
	return ""
}