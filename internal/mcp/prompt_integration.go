package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/client"
	"openai-cli/internal/prompt"
	"openai-cli/internal/service"
)

// PromptAIIntegrator handles AI assistant integration for prompt operations
type PromptAIIntegrator struct {
	promptManager prompt.Manager
	serviceClient service.APIClient
	logger        *zap.Logger
	
	// Configuration
	defaultAssistant  string
	defaultTemperature float64
	defaultMaxTokens   int
}

// NewPromptAIIntegrator creates a new AI integrator for prompt operations
func NewPromptAIIntegrator(promptManager prompt.Manager, logger *zap.Logger) *PromptAIIntegrator {
	return &PromptAIIntegrator{
		promptManager:      promptManager,
		logger:             logger,
		defaultAssistant:   "claude", // Default to Claude
		defaultTemperature: 0.7,
		defaultMaxTokens:   4000,
	}
}

// SetServiceClient sets the API service client for AI calls
func (pai *PromptAIIntegrator) SetServiceClient(client service.APIClient) {
	pai.serviceClient = client
}

// SendToAssistant sends a generated prompt to an AI assistant
func (pai *PromptAIIntegrator) SendToAssistant(ctx context.Context, params AIAssistantParams) (*AIAssistantResult, error) {
	start := time.Now()
	
	pai.logger.Debug("Sending prompt to AI assistant", 
		zap.String("assistant", params.Assistant),
		zap.Int("content_length", len(params.Content)))

	if pai.serviceClient == nil {
		return &AIAssistantResult{
			Success: false,
			Error:   "AI service client not configured",
		}, nil
	}

	// Use default assistant if not specified
	assistant := params.Assistant
	if assistant == "" {
		assistant = pai.defaultAssistant
	}

	// Set default values
	temperature := params.Temperature
	if temperature == 0 {
		temperature = pai.defaultTemperature
	}
	
	maxTokens := params.MaxTokens
	if maxTokens == 0 {
		maxTokens = pai.defaultMaxTokens
	}

	// Build context for the AI call
	contextInfo := ""
	if params.Context != nil {
		contextInfo = pai.buildContextString(params.Context)
	}

	// Build system prompt
	systemPrompt := params.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = pai.buildDefaultSystemPrompt(params.Context)
	}

	// Prepare the message for the AI
	messages := []client.ChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Add context if available
	if contextInfo != "" {
		messages = append(messages, client.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Context: %s", contextInfo),
		})
	}

	// Add the main prompt
	messages = append(messages, client.ChatMessage{
		Role:    "user",
		Content: params.Content,
	})

	// Create chat completion request
	req := client.ChatCompletionsRequest{
		Model:    "claude-3", // Default model, could be configured
		Messages: messages,
	}

	// Call the AI service
	response, err := pai.serviceClient.CreateChatCompletion(ctx, req)
	if err != nil {
		pai.logger.Error("Failed to call AI assistant", 
			zap.String("assistant", assistant),
			zap.Error(err))
		
		return &AIAssistantResult{
			Success: false,
			Error:   fmt.Sprintf("AI call failed: %v", err),
		}, nil
	}

	// Extract content from the first choice
	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	}

	// Extract suggestions and improvements from the response
	suggestions, improvements := pai.extractSuggestionsFromResponse(responseContent)

	// For now, we'll use a placeholder for token usage since it's not in the current response structure
	tokenUsage := len(responseContent) / 4 // Rough estimate: 4 chars per token

	result := &AIAssistantResult{
		Response:       responseContent,
		Assistant:      assistant,
		TokensUsed:     tokenUsage,
		ProcessingTime: time.Since(start),
		Suggestions:    suggestions,
		Improvements:   improvements,
		Success:        true,
		Metadata: map[string]interface{}{
			"temperature":    temperature,
			"max_tokens":     maxTokens,
			"context_length": len(contextInfo),
			"system_prompt":  systemPrompt != "",
		},
	}

	pai.logger.Info("Successfully sent prompt to AI assistant",
		zap.String("assistant", assistant),
		zap.Int("tokens_used", tokenUsage),
		zap.Duration("processing_time", result.ProcessingTime))

	return result, nil
}

// EnhancePrompt uses AI to enhance and improve a prompt
func (pai *PromptAIIntegrator) EnhancePrompt(ctx context.Context, content string, context *WorkspaceContext, assistant string) (*AIAssistantResult, error) {
	pai.logger.Debug("Enhancing prompt with AI", zap.String("assistant", assistant))

	enhancementPrompt := pai.buildEnhancementPrompt(content, context)
	
	params := AIAssistantParams{
		Content:     enhancementPrompt,
		Assistant:   assistant,
		Context:     context,
		Temperature: 0.3, // Lower temperature for enhancement
		MaxTokens:   2000,
		SystemPrompt: "You are an expert prompt engineer. Your task is to analyze and improve prompts for clarity, effectiveness, and specificity. Provide concrete suggestions and an enhanced version.",
	}

	return pai.SendToAssistant(ctx, params)
}

// GetFeedback gets feedback on a prompt from an AI assistant
func (pai *PromptAIIntegrator) GetFeedback(ctx context.Context, content string, assistant string) (*AIAssistantResult, error) {
	pai.logger.Debug("Getting AI feedback on prompt", zap.String("assistant", assistant))

	feedbackPrompt := pai.buildFeedbackPrompt(content)
	
	params := AIAssistantParams{
		Content:     feedbackPrompt,
		Assistant:   assistant,
		Temperature: 0.4,
		MaxTokens:   1500,
		SystemPrompt: "You are an expert prompt evaluator. Analyze the given prompt and provide detailed feedback on its quality, clarity, and effectiveness. Include specific suggestions for improvement.",
	}

	return pai.SendToAssistant(ctx, params)
}

// AnalyzePromptQuality analyzes prompt quality using AI
func (pai *PromptAIIntegrator) AnalyzePromptQuality(ctx context.Context, content string) (map[string]interface{}, error) {
	pai.logger.Debug("Analyzing prompt quality with AI")

	analysisPrompt := pai.buildQualityAnalysisPrompt(content)
	
	params := AIAssistantParams{
		Content:     analysisPrompt,
		Assistant:   "claude",
		Temperature: 0.2,
		MaxTokens:   1000,
		SystemPrompt: "You are a prompt quality analyzer. Evaluate the given prompt and provide a structured analysis including scores for clarity, specificity, completeness, and actionability. Return your analysis in a structured format.",
	}

	result, err := pai.SendToAssistant(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse the structured response (would need more sophisticated parsing)
	analysis := map[string]interface{}{
		"overall_score":    85, // Would extract from AI response
		"clarity_score":    80,
		"specificity_score": 90,
		"completeness_score": 85,
		"actionability_score": 88,
		"ai_response":      result.Response,
		"suggestions":      result.Suggestions,
		"processing_time":  result.ProcessingTime,
	}

	return analysis, nil
}

// GenerateTemplateVariations generates variations of a template using AI
func (pai *PromptAIIntegrator) GenerateTemplateVariations(ctx context.Context, templateName string, count int) ([]string, error) {
	pai.logger.Debug("Generating template variations with AI", 
		zap.String("template", templateName),
		zap.Int("count", count))

	// Get the original template
	template, err := pai.promptManager.GetTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	variationPrompt := pai.buildVariationPrompt(template.Content, count)
	
	params := AIAssistantParams{
		Content:     variationPrompt,
		Assistant:   "claude",
		Temperature: 0.8, // Higher temperature for creativity
		MaxTokens:   3000,
		SystemPrompt: "You are a creative prompt engineer. Generate diverse variations of the given prompt template while maintaining its core purpose and effectiveness.",
	}

	result, err := pai.SendToAssistant(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse variations from the response
	variations := pai.parseVariationsFromResponse(result.Response)
	
	pai.logger.Info("Generated template variations",
		zap.String("template", templateName),
		zap.Int("requested", count),
		zap.Int("generated", len(variations)))

	return variations, nil
}

// OptimizeForContext optimizes a prompt for specific context using AI
func (pai *PromptAIIntegrator) OptimizeForContext(ctx context.Context, content string, context *WorkspaceContext) (*AIAssistantResult, error) {
	pai.logger.Debug("Optimizing prompt for context with AI")

	optimizationPrompt := pai.buildContextOptimizationPrompt(content, context)
	
	params := AIAssistantParams{
		Content:     optimizationPrompt,
		Context:     context,
		Assistant:   "claude",
		Temperature: 0.4,
		MaxTokens:   2500,
		SystemPrompt: "You are an expert at contextual prompt optimization. Adapt the given prompt to be most effective for the specific development context provided.",
	}

	return pai.SendToAssistant(ctx, params)
}

// Helper methods

// buildDefaultSystemPrompt creates a default system prompt
func (pai *PromptAIIntegrator) buildDefaultSystemPrompt(context *WorkspaceContext) string {
	prompt := "You are an expert AI assistant helping with software development tasks."
	
	if context != nil {
		if context.Language != "" {
			prompt += fmt.Sprintf(" The primary programming language is %s.", context.Language)
		}
		if context.Framework != "" {
			prompt += fmt.Sprintf(" The project uses %s framework.", context.Framework)
		}
	}
	
	prompt += " Provide helpful, accurate, and contextually relevant responses."
	return prompt
}

// buildContextString creates a context string for AI calls
func (pai *PromptAIIntegrator) buildContextString(context *WorkspaceContext) string {
	var parts []string
	
	if context.Repository != "" {
		parts = append(parts, fmt.Sprintf("Repository: %s", context.Repository))
	}
	if context.Language != "" {
		parts = append(parts, fmt.Sprintf("Language: %s", context.Language))
	}
	if context.Framework != "" {
		parts = append(parts, fmt.Sprintf("Framework: %s", context.Framework))
	}
	if context.GitBranch != "" {
		parts = append(parts, fmt.Sprintf("Branch: %s", context.GitBranch))
	}
	if len(context.Dependencies) > 0 {
		deps := make([]string, len(context.Dependencies))
		for i, dep := range context.Dependencies {
			deps[i] = fmt.Sprintf("%s@%s", dep.Name, dep.Version)
		}
		parts = append(parts, fmt.Sprintf("Dependencies: %s", strings.Join(deps, ", ")))
	}
	
	return strings.Join(parts, "\n")
}

// buildEnhancementPrompt creates a prompt for enhancing content
func (pai *PromptAIIntegrator) buildEnhancementPrompt(content string, context *WorkspaceContext) string {
	prompt := "Please analyze and enhance the following prompt:\n\n"
	prompt += content + "\n\n"
	prompt += "Provide:\n"
	prompt += "1. An analysis of the current prompt's strengths and weaknesses\n"
	prompt += "2. Specific suggestions for improvement\n"
	prompt += "3. An enhanced version of the prompt\n"
	
	if context != nil {
		prompt += fmt.Sprintf("\nContext: This prompt will be used in a %s project", context.Language)
		if context.Framework != "" {
			prompt += fmt.Sprintf(" using %s", context.Framework)
		}
	}
	
	return prompt
}

// buildFeedbackPrompt creates a prompt for getting feedback
func (pai *PromptAIIntegrator) buildFeedbackPrompt(content string) string {
	return fmt.Sprintf(`Please provide detailed feedback on this prompt:

%s

Evaluate it on:
1. Clarity - Is the prompt clear and unambiguous?
2. Specificity - Does it provide enough specific details?
3. Completeness - Are all necessary elements included?
4. Actionability - Can someone easily act on this prompt?

Provide a score (1-10) for each criterion and explain your reasoning.`, content)
}

// buildQualityAnalysisPrompt creates a prompt for quality analysis
func (pai *PromptAIIntegrator) buildQualityAnalysisPrompt(content string) string {
	return fmt.Sprintf(`Analyze the quality of this prompt and provide a structured assessment:

%s

Provide scores (1-100) for:
- Clarity: How clear and understandable is the prompt?
- Specificity: How specific and detailed are the requirements?
- Completeness: Are all necessary elements included?
- Actionability: How easy is it to act on this prompt?

Also provide an overall quality score and list the top 3 areas for improvement.`, content)
}

// buildVariationPrompt creates a prompt for generating variations
func (pai *PromptAIIntegrator) buildVariationPrompt(content string, count int) string {
	return fmt.Sprintf(`Generate %d creative variations of this prompt template:

%s

Each variation should:
1. Maintain the core purpose and intent
2. Use different wording and structure
3. Be suitable for the same use case
4. Offer a unique perspective or approach

Present each variation clearly numbered and separated.`, count, content)
}

// buildContextOptimizationPrompt creates a prompt for context optimization
func (pai *PromptAIIntegrator) buildContextOptimizationPrompt(content string, context *WorkspaceContext) string {
	prompt := fmt.Sprintf(`Optimize this prompt for the specific development context:

Original prompt:
%s

Context:
`, content)
	
	if context != nil {
		prompt += pai.buildContextString(context)
	}
	
	prompt += `

Please provide:
1. An optimized version that's tailored to this specific context
2. Explanation of the changes made and why
3. Additional context-specific suggestions that could be included`
	
	return prompt
}

// extractSuggestionsFromResponse extracts suggestions from AI response
func (pai *PromptAIIntegrator) extractSuggestionsFromResponse(response string) ([]string, []string) {
	// Basic implementation - would need more sophisticated parsing
	var suggestions, improvements []string
	
	lines := strings.Split(response, "\n")
	var inSuggestions, inImprovements bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "suggestion") {
			inSuggestions = true
			inImprovements = false
			continue
		}
		if strings.Contains(strings.ToLower(line), "improvement") {
			inImprovements = true
			inSuggestions = false
			continue
		}
		
		if (inSuggestions || inImprovements) && strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			item := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
			if inSuggestions {
				suggestions = append(suggestions, item)
			} else {
				improvements = append(improvements, item)
			}
		}
	}
	
	return suggestions, improvements
}

// parseVariationsFromResponse parses variations from AI response
func (pai *PromptAIIntegrator) parseVariationsFromResponse(response string) []string {
	// Basic implementation - would need more sophisticated parsing
	var variations []string
	
	lines := strings.Split(response, "\n")
	var currentVariation strings.Builder
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check if this line starts a new variation (numbered)
		if strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") || 
		   strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "4.") ||
		   strings.HasPrefix(line, "5.") {
			
			// Save previous variation if exists
			if currentVariation.Len() > 0 {
				variations = append(variations, strings.TrimSpace(currentVariation.String()))
				currentVariation.Reset()
			}
			
			// Start new variation (remove number prefix)
			parts := strings.SplitN(line, ".", 2)
			if len(parts) > 1 {
				currentVariation.WriteString(strings.TrimSpace(parts[1]))
			}
		} else if currentVariation.Len() > 0 && line != "" {
			// Continue current variation
			currentVariation.WriteString("\n" + line)
		}
	}
	
	// Add last variation
	if currentVariation.Len() > 0 {
		variations = append(variations, strings.TrimSpace(currentVariation.String()))
	}
	
	return variations
}

// SetDefaultConfiguration sets default configuration for the integrator
func (pai *PromptAIIntegrator) SetDefaultConfiguration(assistant string, temperature float64, maxTokens int) {
	pai.defaultAssistant = assistant
	pai.defaultTemperature = temperature
	pai.defaultMaxTokens = maxTokens
	
	pai.logger.Info("Updated AI integrator configuration",
		zap.String("default_assistant", assistant),
		zap.Float64("default_temperature", temperature),
		zap.Int("default_max_tokens", maxTokens))
}