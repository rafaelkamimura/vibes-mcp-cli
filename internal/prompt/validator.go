package prompt

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

// ValidatorImpl implements the Validator interface
type ValidatorImpl struct {
	logger *zap.Logger
}

// NewValidator creates a new validator
func NewValidator(logger *zap.Logger) Validator {
	return &ValidatorImpl{
		logger: logger,
	}
}

// ValidateTemplate validates a template for quality and completeness
func (v *ValidatorImpl) ValidateTemplate(template *Template) (bool, []string, error) {
	v.logger.Debug("Validating template", zap.String("name", template.Name))

	var issues []string

	// Validate basic structure
	structureIssues := v.ValidateStructure(template)
	issues = append(issues, structureIssues...)

	// Validate content quality
	score, contentIssues, _ := v.ValidateContent(template.Content)
	issues = append(issues, contentIssues...)

	// Validate parameters
	parameterValid, parameterIssues := v.ValidateParameters(template, nil)
	if !parameterValid {
		issues = append(issues, parameterIssues...)
	}

	// Additional template-specific validations
	additionalIssues := v.validateTemplateSpecifics(template)
	issues = append(issues, additionalIssues...)

	// Template is valid if no critical issues and reasonable quality score
	isValid := len(issues) == 0 && score >= 60

	v.logger.Info("Template validation completed",
		zap.String("name", template.Name),
		zap.Bool("valid", isValid),
		zap.Int("issues", len(issues)),
		zap.Int("score", score))

	return isValid, issues, nil
}

// ValidateParameters validates template parameters against provided values
func (v *ValidatorImpl) ValidateParameters(template *Template, parameters map[string]string) (bool, []string) {
	v.logger.Debug("Validating parameters",
		zap.String("template", template.Name),
		zap.Int("provided", len(parameters)))

	var issues []string

	// Check each template parameter
	for _, param := range template.Parameters {
		issues = append(issues, v.validateParameter(param)...)

		// If parameters are provided, validate values
		if parameters != nil {
			if value, exists := parameters[param.Name]; exists {
				if paramIssues := v.validateParameterValue(param, value); len(paramIssues) > 0 {
					issues = append(issues, paramIssues...)
				}
			} else if param.Required {
				issues = append(issues, fmt.Sprintf("required parameter '%s' is missing", param.Name))
			}
		}
	}

	// Check for unused parameters
	if parameters != nil {
		for paramName := range parameters {
			if !v.isValidParameterName(paramName, template) {
				issues = append(issues, fmt.Sprintf("unknown parameter '%s'", paramName))
			}
		}
	}

	return len(issues) == 0, issues
}

// ValidateContent validates the content quality and structure
func (v *ValidatorImpl) ValidateContent(content string) (int, []string, []string) {
	v.logger.Debug("Validating content", zap.Int("length", len(content)))

	var issues []string
	var warnings []string
	score := 100

	// Basic content checks
	if len(content) == 0 {
		issues = append(issues, "content is empty")
		return 0, issues, warnings
	}

	// Length checks
	if len(content) < 50 {
		issues = append(issues, "content is too short (minimum 50 characters)")
		score -= 30
	} else if len(content) < 100 {
		warnings = append(warnings, "content is quite short (consider adding more detail)")
		score -= 10
	}

	if len(content) > 10000 {
		warnings = append(warnings, "content is very long (consider breaking into sections)")
		score -= 5
	}

	// Structure checks
	structureScore, structureIssues, structureWarnings := v.validateContentStructure(content)
	score += structureScore - 50 // Adjust from base score
	issues = append(issues, structureIssues...)
	warnings = append(warnings, structureWarnings...)

	// Language quality checks
	languageScore, languageIssues, languageWarnings := v.validateLanguageQuality(content)
	score += languageScore - 50 // Adjust from base score
	issues = append(issues, languageIssues...)
	warnings = append(warnings, languageWarnings...)

	// Placeholder validation
	placeholderIssues := v.validatePlaceholders(content)
	issues = append(issues, placeholderIssues...)
	if len(placeholderIssues) > 0 {
		score -= len(placeholderIssues) * 5
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score, issues, warnings
}

// ValidateStructure validates the template structure
func (v *ValidatorImpl) ValidateStructure(template *Template) []string {
	var issues []string

	// Required fields
	if template.Name == "" {
		issues = append(issues, "template name is required")
	} else {
		// Name format validation
		if !regexp.MustCompile(`^[a-z0-9-_/]+$`).MatchString(template.Name) {
			issues = append(issues, "template name should contain only lowercase letters, numbers, hyphens, underscores, and forward slashes")
		}
	}

	if template.Category == "" {
		issues = append(issues, "template category is required")
	} else {
		// Validate category
		validCategories := []string{CategoryGeneral, CategoryLanguages, CategoryWorkflows, CategoryWorkspace, CategoryCustom}
		if !v.contains(validCategories, template.Category) {
			issues = append(issues, fmt.Sprintf("invalid category '%s', must be one of: %s",
				template.Category, strings.Join(validCategories, ", ")))
		}
	}

	if template.Description == "" {
		issues = append(issues, "template description is required")
	} else {
		// Description quality
		if len(template.Description) < 10 {
			issues = append(issues, "template description is too short")
		}
		if len(template.Description) > 200 {
			issues = append(issues, "template description is too long (max 200 characters)")
		}
	}

	if template.Content == "" {
		issues = append(issues, "template content is required")
	}

	// Validate language and framework
	if template.Language != "" {
		validLanguages := []string{
			"go", "javascript", "typescript", "python", "java", "rust",
			"c", "cpp", "php", "ruby", "swift", "kotlin", "csharp",
		}
		if !v.contains(validLanguages, template.Language) {
			issues = append(issues, fmt.Sprintf("unsupported language '%s'", template.Language))
		}
	}

	// Validate version format if provided
	if template.Version != "" {
		if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(template.Version) {
			issues = append(issues, "version should follow semantic versioning (e.g., 1.0.0)")
		}
	}

	return issues
}

// GetQualityScore calculates an overall quality score for a template
func (v *ValidatorImpl) GetQualityScore(template *Template) int {
	score := 100

	// Structure score
	structureIssues := v.ValidateStructure(template)
	score -= len(structureIssues) * 10

	// Content score
	contentScore, _, _ := v.ValidateContent(template.Content)
	score = (score + contentScore) / 2

	// Parameter quality
	for _, param := range template.Parameters {
		paramIssues := v.validateParameter(param)
		score -= len(paramIssues) * 2
	}

	// Bonus points for good practices
	if len(template.Examples) > 0 {
		score += 5
	}
	if len(template.Tags) > 0 {
		score += 3
	}
	if template.Author != "" {
		score += 2
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// Helper methods

func (v *ValidatorImpl) validateParameter(param TemplateParameter) []string {
	var issues []string

	if param.Name == "" {
		issues = append(issues, "parameter name is required")
		return issues
	}

	// Parameter name format
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(param.Name) {
		issues = append(issues, fmt.Sprintf("parameter name '%s' should start with lowercase letter and contain only lowercase letters, numbers, and underscores", param.Name))
	}

	if param.Description == "" {
		issues = append(issues, fmt.Sprintf("parameter '%s' is missing description", param.Name))
	}

	// Type validation
	validTypes := []string{"string", "int", "bool", "select"}
	if param.Type == "" {
		param.Type = "string" // Default type
	} else if !v.contains(validTypes, param.Type) {
		issues = append(issues, fmt.Sprintf("parameter '%s' has invalid type '%s'", param.Name, param.Type))
	}

	// Select type requires options
	if param.Type == "select" {
		if len(param.Options) == 0 {
			issues = append(issues, fmt.Sprintf("parameter '%s' of type 'select' requires options", param.Name))
		} else {
			// Validate options
			for i, option := range param.Options {
				if strings.TrimSpace(option) == "" {
					issues = append(issues, fmt.Sprintf("parameter '%s' has empty option at index %d", param.Name, i))
				}
			}
		}
	}

	// Validate regex if provided
	if param.Validation != "" {
		if _, err := regexp.Compile(param.Validation); err != nil {
			issues = append(issues, fmt.Sprintf("parameter '%s' has invalid validation regex: %s", param.Name, err.Error()))
		}
	}

	// Default value should match type
	if param.Default != "" {
		if paramIssues := v.validateParameterValue(param, param.Default); len(paramIssues) > 0 {
			issues = append(issues, fmt.Sprintf("parameter '%s' default value is invalid: %s", param.Name, strings.Join(paramIssues, ", ")))
		}
	}

	return issues
}

func (v *ValidatorImpl) validateParameterValue(param TemplateParameter, value string) []string {
	var issues []string

	// Type validation
	switch param.Type {
	case "int":
		if !regexp.MustCompile(`^-?\d+$`).MatchString(value) {
			issues = append(issues, fmt.Sprintf("value '%s' is not a valid integer", value))
		}
	case "bool":
		validBoolValues := []string{"true", "false", "yes", "no", "y", "n", "1", "0"}
		if !v.contains(validBoolValues, strings.ToLower(value)) {
			issues = append(issues, fmt.Sprintf("value '%s' is not a valid boolean", value))
		}
	case "select":
		if !v.contains(param.Options, value) {
			issues = append(issues, fmt.Sprintf("value '%s' is not a valid option", value))
		}
	}

	// Regex validation
	if param.Validation != "" {
		if matched, err := regexp.MatchString(param.Validation, value); err == nil {
			if !matched {
				issues = append(issues, fmt.Sprintf("value '%s' does not match required pattern", value))
			}
		}
	}

	return issues
}

func (v *ValidatorImpl) validateContentStructure(content string) (int, []string, []string) {
	var issues []string
	var warnings []string
	score := 50 // Base score

	// Check for basic structure elements
	hasHeaders := regexp.MustCompile(`^#+\s+`).MatchString(content)
	if hasHeaders {
		score += 10
	} else {
		warnings = append(warnings, "content lacks structure (consider adding headers)")
	}

	// Check for sections
	sections := regexp.MustCompile(`(?m)^#+\s+(.+)$`).FindAllString(content, -1)
	if len(sections) > 1 {
		score += 10
	}

	// Check for code blocks
	hasCodeBlocks := strings.Contains(content, "```")
	if hasCodeBlocks {
		score += 5
	}

	// Check for lists
	hasLists := regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`).MatchString(content) ||
		regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`).MatchString(content)
	if hasLists {
		score += 5
	}

	// Check for placeholders
	placeholders := regexp.MustCompile(`\{\{[^}]+\}\}`).FindAllString(content, -1)
	if len(placeholders) > 0 {
		score += 10
	} else {
		warnings = append(warnings, "content has no placeholders (consider making it more dynamic)")
	}

	// Check for excessive repetition
	words := strings.Fields(strings.ToLower(content))
	wordCount := make(map[string]int)
	for _, word := range words {
		wordCount[word]++
	}
	
	for word, count := range wordCount {
		if len(word) > 3 && count > len(words)/10 { // Word appears more than 10% of the time
			warnings = append(warnings, fmt.Sprintf("word '%s' appears very frequently (%d times)", word, count))
			score -= 5
		}
	}

	return score, issues, warnings
}

func (v *ValidatorImpl) validateLanguageQuality(content string) (int, []string, []string) {
	var issues []string
	var warnings []string
	score := 50 // Base score

	// Check for basic grammar and spelling issues
	grammarScore, grammarIssues := v.checkBasicGrammar(content)
	score += grammarScore - 50
	issues = append(issues, grammarIssues...)

	// Check readability
	readabilityScore := v.calculateReadabilityScore(content)
	score += readabilityScore - 50

	if readabilityScore < 30 {
		warnings = append(warnings, "content may be difficult to read (consider shorter sentences)")
	}

	// Check for professional tone
	if v.hasCasualLanguage(content) {
		warnings = append(warnings, "content contains casual language (consider more professional tone)")
		score -= 5
	}

	// Check for completeness indicators
	if v.hasIncompleteIndicators(content) {
		issues = append(issues, "content appears incomplete (contains TODO, FIXME, etc.)")
		score -= 20
	}

	return score, issues, warnings
}

func (v *ValidatorImpl) validatePlaceholders(content string) []string {
	var issues []string

	// Find all placeholders
	placeholderPattern := regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)
	placeholders := placeholderPattern.FindAllStringSubmatch(content, -1)

	for _, match := range placeholders {
		placeholder := strings.TrimSpace(match[1])
		
		// Check placeholder format
		if !regexp.MustCompile(`^\.?\w+$`).MatchString(placeholder) {
			issues = append(issues, fmt.Sprintf("invalid placeholder format: %s", match[0]))
			continue
		}

		// Remove leading dot if present
		paramName := strings.TrimPrefix(placeholder, ".")
		
		// Check for valid parameter name format
		if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(paramName) {
			issues = append(issues, fmt.Sprintf("invalid parameter name in placeholder: %s", paramName))
		}
	}

	return issues
}

func (v *ValidatorImpl) validateTemplateSpecifics(template *Template) []string {
	var issues []string

	// Category-specific validations
	switch template.Category {
	case CategoryLanguages:
		if template.Language == "" {
			issues = append(issues, "language templates should specify a language")
		}
	case CategoryWorkflows:
		if !strings.Contains(strings.ToLower(template.Content), "step") &&
			!strings.Contains(strings.ToLower(template.Content), "process") {
			issues = append(issues, "workflow templates should describe steps or processes")
		}
	case CategoryWorkspace:
		if !strings.Contains(strings.ToLower(template.Content), "vibes") {
			issues = append(issues, "workspace templates should reference Vibes workspace")
		}
	}

	// Check consistency between metadata and content
	if template.Language != "" {
		langPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(template.Language) + `\b`)
		if !langPattern.MatchString(template.Content) && !langPattern.MatchString(template.Description) {
			issues = append(issues, fmt.Sprintf("template specifies language '%s' but doesn't mention it in content or description", template.Language))
		}
	}

	if template.Framework != "" {
		frameworkPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(template.Framework) + `\b`)
		if !frameworkPattern.MatchString(template.Content) && !frameworkPattern.MatchString(template.Description) {
			issues = append(issues, fmt.Sprintf("template specifies framework '%s' but doesn't mention it in content or description", template.Framework))
		}
	}

	return issues
}

func (v *ValidatorImpl) isValidParameterName(paramName string, template *Template) bool {
	// Check template parameters
	for _, param := range template.Parameters {
		if param.Name == paramName {
			return true
		}
	}

	// Check builtin parameters
	builtinParams := []string{
		"timestamp", "date", "time", "user", "hostname", "pwd",
		"repo", "branch", "language", "framework",
	}
	
	return v.contains(builtinParams, paramName)
}

func (v *ValidatorImpl) checkBasicGrammar(content string) (int, []string) {
	var issues []string
	score := 50 // Base score

	// Check for common grammar issues
	sentences := regexp.MustCompile(`[.!?]+`).Split(content, -1)
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Check sentence capitalization
		if len(sentence) > 0 && unicode.IsLower(rune(sentence[0])) {
			// Allow sentences starting with code or placeholders
			if !strings.HasPrefix(sentence, "`") && !strings.HasPrefix(sentence, "{{") {
				score -= 1
			}
		}

		// Check for very long sentences
		if len(sentence) > 200 {
			score -= 2
		}
	}

	// Check for repeated punctuation
	if regexp.MustCompile(`[.!?]{2,}`).MatchString(content) {
		issues = append(issues, "content contains repeated punctuation")
		score -= 5
	}

	// Check for missing spaces after punctuation
	if regexp.MustCompile(`[.!?][A-Z]`).MatchString(content) {
		issues = append(issues, "missing spaces after punctuation")
		score -= 5
	}

	return score, issues
}

func (v *ValidatorImpl) calculateReadabilityScore(content string) int {
	// Simplified readability calculation based on sentence and word length
	sentences := regexp.MustCompile(`[.!?]+`).Split(content, -1)
	words := strings.Fields(content)
	
	if len(sentences) == 0 || len(words) == 0 {
		return 0
	}

	avgWordsPerSentence := float64(len(words)) / float64(len(sentences))
	avgCharsPerWord := float64(len(content)) / float64(len(words))

	// Simple scoring: penalize very long sentences and words
	score := 100
	if avgWordsPerSentence > 20 {
		score -= int((avgWordsPerSentence - 20) * 2)
	}
	if avgCharsPerWord > 6 {
		score -= int((avgCharsPerWord - 6) * 5)
	}

	if score < 0 {
		score = 0
	}
	
	return score
}

func (v *ValidatorImpl) hasCasualLanguage(content string) bool {
	casualWords := []string{
		"gonna", "wanna", "gotta", "kinda", "sorta",
		"yeah", "nope", "yep", "ok", "btw", "fyi",
	}
	
	lowerContent := strings.ToLower(content)
	for _, word := range casualWords {
		if strings.Contains(lowerContent, word) {
			return true
		}
	}
	
	return false
}

func (v *ValidatorImpl) hasIncompleteIndicators(content string) bool {
	indicators := []string{
		"TODO", "FIXME", "HACK", "XXX", "TEMP",
		"[placeholder]", "[TODO]", "[FIXME]",
		"coming soon", "to be implemented",
	}
	
	upperContent := strings.ToUpper(content)
	for _, indicator := range indicators {
		if strings.Contains(upperContent, strings.ToUpper(indicator)) {
			return true
		}
	}
	
	return false
}

func (v *ValidatorImpl) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}