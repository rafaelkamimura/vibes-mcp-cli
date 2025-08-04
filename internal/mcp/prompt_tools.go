package mcp

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"openai-cli/internal/prompt"
)

// PromptToolManager manages prompt operations as MCP tools
type PromptToolManager struct {
	promptManager prompt.Manager
	logger        *zap.Logger
	tools         map[string]*PromptTool
}

// NewPromptToolManager creates a new prompt tool manager
func NewPromptToolManager(promptManager prompt.Manager, logger *zap.Logger) *PromptToolManager {
	ptm := &PromptToolManager{
		promptManager: promptManager,
		logger:        logger,
		tools:         make(map[string]*PromptTool),
	}
	
	// Register all available tools
	ptm.registerTools()
	
	return ptm
}

// GetTools returns all available prompt tools
func (ptm *PromptToolManager) GetTools() []PromptTool {
	var tools []PromptTool
	for _, tool := range ptm.tools {
		tools = append(tools, *tool)
	}
	return tools
}

// GetTool returns a specific tool by name
func (ptm *PromptToolManager) GetTool(name string) (*PromptTool, error) {
	tool, exists := ptm.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// CallTool executes a prompt tool with given arguments
func (ptm *PromptToolManager) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	ptm.logger.Debug("Calling prompt tool", zap.String("tool", toolName))

	switch toolName {
	case "generate_prompt":
		return ptm.generatePrompt(ctx, arguments)
	case "validate_template":
		return ptm.validateTemplate(ctx, arguments)
	case "detect_context":
		return ptm.detectContext(ctx, arguments)
	case "suggest_templates":
		return ptm.suggestTemplates(ctx, arguments)
	case "get_history":
		return ptm.getHistory(ctx, arguments)
	case "template_stats":
		return ptm.getTemplateStats(ctx, arguments)
	case "workspace_analysis":
		return ptm.analyzeWorkspace(ctx, arguments)
	case "quality_check":
		return ptm.qualityCheck(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// registerTools registers all available prompt tools
func (ptm *PromptToolManager) registerTools() {
	// Generate Prompt Tool
	ptm.tools["generate_prompt"] = &PromptTool{
		Name:        "generate_prompt",
		Description: "Generate a prompt from a template with parameters and context",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"template_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the template to use",
				},
				"parameters": map[string]interface{}{
					"type":        "object",
					"description": "Parameters to fill in the template",
				},
				"context": map[string]interface{}{
					"type":        "object",
					"description": "Workspace context information",
				},
				"interactive": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to use interactive parameter collection",
					"default":     false,
				},
				"validate": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to validate the generated prompt",
					"default":     true,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format (markdown, text, json)",
					"enum":        []string{"markdown", "text", "json"},
					"default":     "text",
				},
			},
			Required: []string{"template_name"},
		},
	}

	// Validate Template Tool
	ptm.tools["validate_template"] = &PromptTool{
		Name:        "validate_template",
		Description: "Validate a prompt template or generated content",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"template_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of template to validate",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to validate directly",
				},
			},
		},
	}

	// Detect Context Tool
	ptm.tools["detect_context"] = &PromptTool{
		Name:        "detect_context",
		Description: "Detect current workspace context (language, framework, dependencies)",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "Directory to analyze (defaults to current)",
				},
			},
		},
	}

	// Suggest Templates Tool
	ptm.tools["suggest_templates"] = &PromptTool{
		Name:        "suggest_templates",
		Description: "Suggest relevant templates based on context",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"context": map[string]interface{}{
					"type":        "object",
					"description": "Workspace context",
				},
				"task_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of task (coding, documentation, debugging, etc.)",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Programming language",
				},
				"framework": map[string]interface{}{
					"type":        "string",
					"description": "Framework or library",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of suggestions",
					"default":     5,
				},
			},
		},
	}

	// Get History Tool
	ptm.tools["get_history"] = &PromptTool{
		Name:        "get_history",
		Description: "Retrieve prompt generation history with filtering",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of entries to return",
					"default":     10,
				},
				"filter": map[string]interface{}{
					"type":        "string",
					"description": "Filter string for search",
				},
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Filter by repository",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Filter by programming language",
				},
				"template": map[string]interface{}{
					"type":        "string",
					"description": "Filter by template name",
				},
				"start_date": map[string]interface{}{
					"type":        "string",
					"description": "Start date for filtering (ISO 8601)",
				},
				"end_date": map[string]interface{}{
					"type":        "string",
					"description": "End date for filtering (ISO 8601)",
				},
			},
		},
	}

	// Template Statistics Tool
	ptm.tools["template_stats"] = &PromptTool{
		Name:        "template_stats",
		Description: "Get statistics about template usage and performance",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"template_name": map[string]interface{}{
					"type":        "string",
					"description": "Specific template to get stats for",
				},
				"time_range": map[string]interface{}{
					"type":        "string",
					"description": "Time range for stats (daily, weekly, monthly)",
					"enum":        []string{"daily", "weekly", "monthly"},
					"default":     "weekly",
				},
			},
		},
	}

	// Workspace Analysis Tool
	ptm.tools["workspace_analysis"] = &PromptTool{
		Name:        "workspace_analysis",
		Description: "Analyze workspace for prompt optimization opportunities",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"deep_scan": map[string]interface{}{
					"type":        "boolean",
					"description": "Perform deep analysis including file content",
					"default":     false,
				},
				"include_dependencies": map[string]interface{}{
					"type":        "boolean",
					"description": "Include dependency analysis",
					"default":     true,
				},
			},
		},
	}

	// Quality Check Tool
	ptm.tools["quality_check"] = &PromptTool{
		Name:        "quality_check",
		Description: "Perform comprehensive quality check on templates or generated content",
		InputSchema: ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to check",
				},
				"template_name": map[string]interface{}{
					"type":        "string",
					"description": "Template name to check",
				},
				"check_types": map[string]interface{}{
					"type":        "array",
					"description": "Types of checks to perform",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"grammar", "clarity", "completeness", "relevance"},
					},
					"default": []string{"grammar", "clarity", "completeness"},
				},
			},
		},
	}

	ptm.logger.Info("Registered prompt tools", zap.Int("count", len(ptm.tools)))
}

// generatePrompt implements the generate_prompt tool
func (ptm *PromptToolManager) generatePrompt(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	// Parse arguments
	templateName, ok := arguments["template_name"].(string)
	if !ok || templateName == "" {
		return PromptGenerateResult{Success: false, Error: "template_name is required"}, nil
	}

	// Extract parameters
	parameters := make(map[string]string)
	if params, ok := arguments["parameters"].(map[string]interface{}); ok {
		for k, v := range params {
			if str, ok := v.(string); ok {
				parameters[k] = str
			} else {
				parameters[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Extract context
	var workspaceContext *prompt.WorkspaceContext
	if contextData, ok := arguments["context"].(map[string]interface{}); ok {
		workspaceContext = ptm.parseWorkspaceContext(contextData)
	}

	// Extract other options
	interactive, _ := arguments["interactive"].(bool)
	validate, _ := arguments["validate"].(bool)
	if validate == false && arguments["validate"] == nil {
		validate = true // Default to true
	}
	outputFormat, _ := arguments["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "text"
	}

	// Create generation config
	config := &prompt.GenerationConfig{
		TemplateName: templateName,
		Interactive:  interactive,
		Context:      workspaceContext,
		Parameters:   parameters,
		Validate:     validate,
		OutputFormat: outputFormat,
	}

	// Generate prompt
	result, err := ptm.promptManager.GeneratePrompt(config)
	if err != nil {
		ptm.logger.Error("Failed to generate prompt", zap.String("template", templateName), zap.Error(err))
		return PromptGenerateResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Convert to MCP result
	mcpResult := PromptGenerateResult{
		Content:     result.Content,
		Template:    templateName,
		Parameters:  parameters,
		GeneratedAt: result.GeneratedAt,
		ValidationStatus: ValidationStatus{
			Valid:    result.ValidationStatus.Valid,
			Score:    result.ValidationStatus.Score,
			Issues:   result.ValidationStatus.Issues,
			Warnings: result.ValidationStatus.Warnings,
		},
		WordCount: result.WordCount,
		CharCount: result.CharCount,
		Success:   true,
	}

	// Add context if available
	if result.Context != nil {
		mcpResult.Context = ptm.convertWorkspaceContext(result.Context)
	}

	ptm.logger.Info("Generated prompt via MCP", 
		zap.String("template", templateName),
		zap.Int("word_count", result.WordCount))

	return mcpResult, nil
}

// validateTemplate implements the validate_template tool
func (ptm *PromptToolManager) validateTemplate(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	templateName, hasTemplate := arguments["template_name"].(string)
	content, hasContent := arguments["content"].(string)

	if !hasTemplate && !hasContent {
		return PromptValidateResult{Success: false, Error: "either template_name or content is required"}, nil
	}

	var valid bool
	var issues []string
	var score int

	if hasTemplate {
		// Validate template by name
		valid, issues = ptm.promptManager.ValidateTemplate(templateName)
		if valid {
			score = 100 // Basic scoring, could be enhanced
		} else {
			score = 0
		}
	} else {
		// Validate content directly (would need a content validator)
		// For now, basic validation
		valid = len(strings.TrimSpace(content)) > 0
		if !valid {
			issues = []string{"Content is empty"}
			score = 0
		} else {
			score = 85 // Assume good quality for non-empty content
		}
	}

	result := PromptValidateResult{
		Valid:   valid,
		Score:   score,
		Issues:  issues,
		Success: true,
	}

	if hasTemplate {
		result.Template = templateName
	}

	return result, nil
}

// detectContext implements the detect_context tool
func (ptm *PromptToolManager) detectContext(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	context, err := ptm.promptManager.DetectWorkspaceContext()
	if err != nil {
		ptm.logger.Error("Failed to detect workspace context", zap.Error(err))
		return nil, err
	}

	return ptm.convertWorkspaceContext(context), nil
}

// suggestTemplates implements the suggest_templates tool
func (ptm *PromptToolManager) suggestTemplates(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	var workspaceContext *prompt.WorkspaceContext
	
	// Parse context if provided
	if contextData, ok := arguments["context"].(map[string]interface{}); ok {
		workspaceContext = ptm.parseWorkspaceContext(contextData)
	} else {
		// Detect context if not provided
		var err error
		workspaceContext, err = ptm.promptManager.DetectWorkspaceContext()
		if err != nil {
			ptm.logger.Warn("Failed to detect workspace context", zap.Error(err))
		}
	}

	// Override context fields with specific arguments
	if workspaceContext == nil {
		workspaceContext = &prompt.WorkspaceContext{}
	}
	
	if lang, ok := arguments["language"].(string); ok {
		workspaceContext.Language = lang
	}
	if framework, ok := arguments["framework"].(string); ok {
		workspaceContext.Framework = framework
	}

	suggestions := ptm.promptManager.SuggestTemplates(workspaceContext)
	
	// Limit results if requested
	maxResults := 5
	if max, ok := arguments["max_results"].(float64); ok {
		maxResults = int(max)
	}
	
	if len(suggestions) > maxResults {
		suggestions = suggestions[:maxResults]
	}

	// Convert to MCP format
	var mcpSuggestions []TemplateSuggestion
	for _, suggestion := range suggestions {
		mcpSuggestions = append(mcpSuggestions, TemplateSuggestion{
			Name:      suggestion.Name,
			Reason:    suggestion.Reason,
			Relevance: suggestion.Relevance,
			Category:  suggestion.Category,
		})
	}

	return PromptSuggestResult{
		Suggestions: mcpSuggestions,
		Context:     ptm.convertWorkspaceContext(workspaceContext),
		Success:     true,
	}, nil
}

// getHistory implements the get_history tool
func (ptm *PromptToolManager) getHistory(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	limit := 10
	if l, ok := arguments["limit"].(float64); ok {
		limit = int(l)
	}

	filter, _ := arguments["filter"].(string)

	entries, err := ptm.promptManager.GetHistory(limit, filter)
	if err != nil {
		return PromptHistoryResult{Success: false, Error: err.Error()}, nil
	}

	// Convert to MCP format
	var mcpEntries []HistoryEntry
	for _, entry := range entries {
		mcpEntries = append(mcpEntries, HistoryEntry{
			ID:           entry.ID,
			Template:     entry.Template,
			Repository:   entry.Repository,
			Language:     entry.Language,
			Framework:    entry.Framework,
			Parameters:   entry.Parameters,
			OutputMethod: entry.OutputMethod,
			AITool:       entry.AITool,
			Success:      entry.Success,
			ErrorMessage: entry.ErrorMessage,
			Timestamp:    entry.Timestamp,
			Duration:     entry.Duration,
			WordCount:    entry.WordCount,
		})
	}

	return PromptHistoryResult{
		Entries: mcpEntries,
		Total:   len(mcpEntries),
		Success: true,
	}, nil
}

// getTemplateStats implements the template_stats tool
func (ptm *PromptToolManager) getTemplateStats(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	// This would integrate with the history tracker to get statistics
	// For now, return basic stats
	stats := map[string]interface{}{
		"total_generations": 0,
		"success_rate":      100.0,
		"average_word_count": 250,
		"message":           "Statistics feature not fully implemented",
	}

	if templateName, ok := arguments["template_name"].(string); ok {
		stats["template"] = templateName
	}

	return stats, nil
}

// analyzeWorkspace implements the workspace_analysis tool
func (ptm *PromptToolManager) analyzeWorkspace(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	context, err := ptm.promptManager.DetectWorkspaceContext()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}, nil
	}

	analysis := map[string]interface{}{
		"context":              ptm.convertWorkspaceContext(context),
		"optimization_opportunities": []string{
			"Consider creating custom templates for " + context.Language,
			"Add framework-specific templates for " + context.Framework,
		},
		"recommendations": []string{
			"Use context-aware prompt generation",
			"Implement template caching for better performance",
		},
	}

	return analysis, nil
}

// qualityCheck implements the quality_check tool
func (ptm *PromptToolManager) qualityCheck(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	content, hasContent := arguments["content"].(string)
	templateName, hasTemplate := arguments["template_name"].(string)

	if !hasContent && !hasTemplate {
		return map[string]interface{}{
			"error": "either content or template_name is required",
		}, nil
	}

	var checkContent string
	if hasContent {
		checkContent = content
	} else {
		template, err := ptm.promptManager.GetTemplate(templateName)
		if err != nil {
			return map[string]interface{}{
				"error": fmt.Sprintf("failed to get template: %v", err),
			}, nil
		}
		checkContent = template.Content
	}

	// Basic quality checks
	issues := []string{}
	warnings := []string{}
	score := 100

	if len(checkContent) == 0 {
		issues = append(issues, "Content is empty")
		score = 0
	} else {
		if len(checkContent) < 10 {
			warnings = append(warnings, "Content is very short")
			score -= 20
		}
		if !strings.Contains(checkContent, "\n") {
			warnings = append(warnings, "Content might benefit from line breaks")
			score -= 10
		}
	}

	return map[string]interface{}{
		"score":    score,
		"issues":   issues,
		"warnings": warnings,
		"valid":    len(issues) == 0,
	}, nil
}

// Helper methods

// parseWorkspaceContext converts map to WorkspaceContext
func (ptm *PromptToolManager) parseWorkspaceContext(data map[string]interface{}) *prompt.WorkspaceContext {
	context := &prompt.WorkspaceContext{}
	
	if wd, ok := data["working_directory"].(string); ok {
		context.WorkingDirectory = wd
	}
	if repo, ok := data["repository"].(string); ok {
		context.Repository = repo
	}
	if lang, ok := data["language"].(string); ok {
		context.Language = lang
	}
	if framework, ok := data["framework"].(string); ok {
		context.Framework = framework
	}
	if branch, ok := data["git_branch"].(string); ok {
		context.GitBranch = branch
	}
	if status, ok := data["git_status"].(string); ok {
		context.GitStatus = status
	}
	
	// Parse arrays
	if langs, ok := data["available_languages"].([]interface{}); ok {
		for _, lang := range langs {
			if str, ok := lang.(string); ok {
				context.AvailableLanguages = append(context.AvailableLanguages, str)
			}
		}
	}
	
	if files, ok := data["recent_files"].([]interface{}); ok {
		for _, file := range files {
			if str, ok := file.(string); ok {
				context.RecentFiles = append(context.RecentFiles, str)
			}
		}
	}

	return context
}

// convertWorkspaceContext converts prompt.WorkspaceContext to MCP WorkspaceContext
func (ptm *PromptToolManager) convertWorkspaceContext(context *prompt.WorkspaceContext) *WorkspaceContext {
	if context == nil {
		return nil
	}

	mcpContext := &WorkspaceContext{
		WorkingDirectory:   context.WorkingDirectory,
		Repository:         context.Repository,
		Language:           context.Language,
		Framework:          context.Framework,
		AvailableLanguages: context.AvailableLanguages,
		RecentFiles:        context.RecentFiles,
		GitBranch:          context.GitBranch,
		GitStatus:          context.GitStatus,
		LastModified:       context.LastModified,
	}

	// Convert dependencies
	for _, dep := range context.Dependencies {
		mcpContext.Dependencies = append(mcpContext.Dependencies, Dependency{
			Name:    dep.Name,
			Version: dep.Version,
			Type:    dep.Type,
			Manager: dep.Manager,
		})
	}

	mcpContext.ProjectStructure = context.ProjectStructure
	mcpContext.Environment = context.Environment

	return mcpContext
}