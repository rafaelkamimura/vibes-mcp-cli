package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/config"
)

// IntegratorImpl implements the Integrator interface
type IntegratorImpl struct {
	logger *zap.Logger
	config *config.Config
}

// NewIntegrator creates a new integrator
func NewIntegrator(cfg *config.Config, logger *zap.Logger) Integrator {
	return &IntegratorImpl{
		logger: logger,
		config: cfg,
	}
}

// CopyToClipboard copies content to the system clipboard
func (i *IntegratorImpl) CopyToClipboard(content string) error {
	i.logger.Debug("Copying content to clipboard", zap.Int("length", len(content)))

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return &PromptError{
				Type:    ErrorTypeIntegration,
				Message: "no clipboard tool found (install xclip or xsel)",
			}
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: fmt.Sprintf("clipboard not supported on %s", runtime.GOOS),
		}
	}

	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to copy to clipboard",
			Cause:   err,
		}
	}

	i.logger.Info("Content copied to clipboard successfully")
	return nil
}

// SaveToFile saves content to a file
func (i *IntegratorImpl) SaveToFile(content, filePath string) error {
	i.logger.Debug("Saving content to file",
		zap.String("path", filePath),
		zap.Int("length", len(content)))

	// Create directory if it doesn't exist
	if err := os.MkdirAll(strings.TrimSuffix(filePath, "/"), 0755); err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to create directory",
			Cause:   err,
		}
	}

	// Write content to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to write file",
			Cause:   err,
		}
	}

	i.logger.Info("Content saved to file successfully", zap.String("path", filePath))
	return nil
}

// SendToClaude sends content directly to Claude API
func (i *IntegratorImpl) SendToClaude(ctx context.Context, content string) error {
	i.logger.Info("Sending content to Claude", zap.Int("length", len(content)))

	// Prepare API request
	requestBody := map[string]interface{}{
		"model":      "claude-3-sonnet-20240229",
		"max_tokens": 4000,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": content,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to marshal request",
			Cause:   err,
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", i.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to create request",
			Cause:   err,
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+i.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to send request to Claude",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: fmt.Sprintf("Claude API returned status %d", resp.StatusCode),
		}
	}

	// Parse response
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to parse Claude response",
			Cause:   err,
		}
	}

	// Display response
	if len(response.Content) > 0 {
		fmt.Printf("\n🤖 Claude Response:\n")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(response.Content[0].Text)
		fmt.Println()
	}

	i.logger.Info("Content sent to Claude successfully")
	return nil
}

// UseContext7 integrates with Context7 for real-time documentation
func (i *IntegratorImpl) UseContext7(ctx context.Context, result *GenerationResult) error {
	i.logger.Info("Activating Context7 integration", zap.String("template", result.Template.Name))

	// Context7 integration would involve:
	// 1. Analyzing the generated prompt for documentation needs
	// 2. Making API calls to Context7 service
	// 3. Enriching the prompt with real-time documentation

	// For now, this is a placeholder implementation
	context7URL := i.getIntegrationSetting("context7.url", "https://api.context7.com")
	context7APIKey := i.getIntegrationSetting("context7.api_key", "")

	if context7APIKey == "" {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "Context7 API key not configured",
		}
	}

	// Analyze prompt content for documentation opportunities
	docQueries := i.extractDocumentationQueries(result.Content)
	if len(docQueries) == 0 {
		i.logger.Info("No documentation queries found in prompt")
		return nil
	}

	// Fetch documentation for each query
	for _, query := range docQueries {
		if err := i.fetchContext7Documentation(ctx, context7URL, context7APIKey, query); err != nil {
			i.logger.Warn("Failed to fetch documentation for query",
				zap.String("query", query),
				zap.Error(err))
		}
	}

	i.logger.Info("Context7 integration completed",
		zap.Int("queries", len(docQueries)))

	return nil
}

// TriggerBeastmode activates autonomous development mode
func (i *IntegratorImpl) TriggerBeastmode(ctx context.Context, result *GenerationResult) error {
	i.logger.Info("Triggering Beastmode", zap.String("template", result.Template.Name))

	// Beastmode integration would involve:
	// 1. Analyzing the prompt for actionable development tasks
	// 2. Creating automated development workflows
	// 3. Executing code generation and testing pipelines

	beastmodeURL := i.getIntegrationSetting("beastmode.url", "http://localhost:8080")
	beastmodeToken := i.getIntegrationSetting("beastmode.token", "")

	if beastmodeToken == "" {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "Beastmode token not configured",
		}
	}

	// Extract development tasks from the prompt
	tasks := i.extractDevelopmentTasks(result.Content)
	if len(tasks) == 0 {
		i.logger.Info("No development tasks found in prompt")
		return nil
	}

	// Create Beastmode workflow
	workflow := i.createBeastmodeWorkflow(result, tasks)

	// Submit workflow to Beastmode
	if err := i.submitBeastmodeWorkflow(ctx, beastmodeURL, beastmodeToken, workflow); err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to submit Beastmode workflow",
			Cause:   err,
		}
	}

	i.logger.Info("Beastmode workflow submitted successfully",
		zap.Int("tasks", len(tasks)))

	return nil
}

// TestIntegration tests connectivity to an integration
func (i *IntegratorImpl) TestIntegration(tool string) error {
	i.logger.Debug("Testing integration", zap.String("tool", tool))

	switch tool {
	case AIToolClaude:
		return i.testClaudeIntegration()
	case AIToolContext7:
		return i.testContext7Integration()
	case AIToolBeastmode:
		return i.testBeastmodeIntegration()
	case "clipboard":
		return i.testClipboardIntegration()
	default:
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: fmt.Sprintf("unknown integration tool: %s", tool),
		}
	}
}

// Helper methods

func (i *IntegratorImpl) getIntegrationSetting(key, defaultValue string) string {
	// This would typically read from the integration settings in config
	// For now, check environment variables
	envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

func (i *IntegratorImpl) extractDocumentationQueries(content string) []string {
	var queries []string

	// Look for code blocks and API references
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for API calls or function references
		if strings.Contains(line, "API") ||
			strings.Contains(line, "function") ||
			strings.Contains(line, "method") ||
			strings.Contains(line, "class") {
			queries = append(queries, line)
		}

		// Look for technology/framework mentions
		if strings.Contains(line, "framework") ||
			strings.Contains(line, "library") ||
			strings.Contains(line, "package") {
			queries = append(queries, line)
		}
	}

	// Limit to avoid too many queries
	if len(queries) > 5 {
		queries = queries[:5]
	}

	return queries
}

func (i *IntegratorImpl) fetchContext7Documentation(ctx context.Context, baseURL, apiKey, query string) error {
	// Prepare request
	requestBody := map[string]interface{}{
		"query":   query,
		"context": "prompt-generation",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/search", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Context7 API returned status %d", resp.StatusCode)
	}

	// Process response (would typically cache and use results)
	i.logger.Debug("Context7 documentation fetched", zap.String("query", query))
	return nil
}

func (i *IntegratorImpl) extractDevelopmentTasks(content string) []string {
	var tasks []string

	// Look for action words and task indicators
	actionWords := []string{
		"implement", "create", "build", "develop", "add", "fix",
		"refactor", "optimize", "test", "deploy", "update",
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		lowerLine := strings.ToLower(strings.TrimSpace(line))
		
		for _, action := range actionWords {
			if strings.Contains(lowerLine, action) {
				tasks = append(tasks, strings.TrimSpace(line))
				break
			}
		}
	}

	// Limit tasks
	if len(tasks) > 10 {
		tasks = tasks[:10]
	}

	return tasks
}

func (i *IntegratorImpl) createBeastmodeWorkflow(result *GenerationResult, tasks []string) map[string]interface{} {
	workflow := map[string]interface{}{
		"name":        fmt.Sprintf("Generated from %s", result.Template.Name),
		"description": "Auto-generated workflow from prompt template",
		"tasks":       tasks,
		"context": map[string]interface{}{
			"repository": result.Context.Repository,
			"language":   result.Context.Language,
			"framework":  result.Context.Framework,
		},
		"metadata": map[string]interface{}{
			"source":       "vibes-mcp-cli",
			"template":     result.Template.Name,
			"generated_at": result.GeneratedAt.Format(time.RFC3339),
		},
	}

	return workflow
}

func (i *IntegratorImpl) submitBeastmodeWorkflow(ctx context.Context, baseURL, token string, workflow map[string]interface{}) error {
	jsonData, err := json.Marshal(workflow)
	if err != nil {
		return err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/workflows", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Beastmode API returned status %d", resp.StatusCode)
	}

	return nil
}

func (i *IntegratorImpl) testClaudeIntegration() error {
	if i.config.APIKey == "" {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "Claude API key not configured",
		}
	}

	// Test with a simple request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return i.SendToClaude(ctx, "Hello, this is a test message.")
}

func (i *IntegratorImpl) testContext7Integration() error {
	context7APIKey := i.getIntegrationSetting("context7.api_key", "")
	if context7APIKey == "" {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "Context7 API key not configured",
		}
	}

	// Test with a simple query
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	context7URL := i.getIntegrationSetting("context7.url", "https://api.context7.com")
	return i.fetchContext7Documentation(ctx, context7URL, context7APIKey, "test query")
}

func (i *IntegratorImpl) testBeastmodeIntegration() error {
	beastmodeToken := i.getIntegrationSetting("beastmode.token", "")
	if beastmodeToken == "" {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "Beastmode token not configured",
		}
	}

	// Test connectivity
	beastmodeURL := i.getIntegrationSetting("beastmode.url", "http://localhost:8080")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", beastmodeURL+"/health", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+beastmodeToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: "failed to connect to Beastmode",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &PromptError{
			Type:    ErrorTypeIntegration,
			Message: fmt.Sprintf("Beastmode health check failed with status %d", resp.StatusCode),
		}
	}

	return nil
}

func (i *IntegratorImpl) testClipboardIntegration() error {
	testContent := "vibes-mcp-cli clipboard test"
	return i.CopyToClipboard(testContent)
}

// Advanced integration features

// WebhookIntegration handles webhook-based integrations
type WebhookIntegration struct {
	integrator *IntegratorImpl
	logger     *zap.Logger
}

// NewWebhookIntegration creates a new webhook integration
func NewWebhookIntegration(integrator *IntegratorImpl, logger *zap.Logger) *WebhookIntegration {
	return &WebhookIntegration{
		integrator: integrator,
		logger:     logger,
	}
}

// SendWebhook sends a webhook notification for prompt generation
func (w *WebhookIntegration) SendWebhook(ctx context.Context, webhookURL string, result *GenerationResult) error {
	w.logger.Debug("Sending webhook", zap.String("url", webhookURL))

	payload := map[string]interface{}{
		"event": "prompt_generated",
		"data": map[string]interface{}{
			"template":     result.Template.Name,
			"repository":   result.Context.Repository,
			"language":     result.Context.Language,
			"word_count":   result.WordCount,
			"generated_at": result.GeneratedAt.Format(time.RFC3339),
		},
		"metadata": map[string]interface{}{
			"source": "vibes-mcp-cli",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "vibes-mcp-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	w.logger.Info("Webhook sent successfully", zap.String("url", webhookURL))
	return nil
}

// SlackIntegration handles Slack notifications
type SlackIntegration struct {
	integrator *IntegratorImpl
	logger     *zap.Logger
}

// NewSlackIntegration creates a new Slack integration
func NewSlackIntegration(integrator *IntegratorImpl, logger *zap.Logger) *SlackIntegration {
	return &SlackIntegration{
		integrator: integrator,
		logger:     logger,
	}
}

// SendSlackNotification sends a Slack notification for prompt generation
func (s *SlackIntegration) SendSlackNotification(ctx context.Context, webhookURL string, result *GenerationResult) error {
	s.logger.Debug("Sending Slack notification")

	message := fmt.Sprintf("🎯 *Prompt Generated*\n"+
		"Template: `%s`\n"+
		"Repository: `%s`\n"+
		"Language: `%s`\n"+
		"Word Count: %d",
		result.Template.Name,
		result.Context.Repository,
		result.Context.Language,
		result.WordCount)

	payload := map[string]interface{}{
		"text": message,
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": message,
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	s.logger.Info("Slack notification sent successfully")
	return nil
}