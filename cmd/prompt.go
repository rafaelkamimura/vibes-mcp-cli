package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"openai-cli/internal/client"
	"openai-cli/internal/prompt"
	"openai-cli/internal/service"
)

var (
	// Template selection flags
	promptCategory   string
	promptRepo       string
	promptLanguage   string
	promptFramework  string
	promptComponent  string
	promptSeverity   string
	promptPriority   string

	// Generation flags  
	promptInteractive bool
	promptAutoDetect  bool
	promptValidate    bool

	// Output flags
	promptOutput     string
	promptClipboard  bool
	promptStdout     bool

	// AI integration flags
	promptSendToClaude bool
	promptUseContext7  bool
	promptBeastmode    bool

	// Template management flags
	promptTemplatePath string
	promptForce        bool
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "AI prompt template management and generation",
	Long: `Generate structured AI prompts using predefined templates.

Templates are organized by category:
- general: Repository-agnostic development tasks  
- languages: Technology-specific optimizations
- workflows: Multi-step orchestration patterns
- workspace: Vibes workspace specific optimizations

Examples:
  vibes-mcp-cli prompt list
  vibes-mcp-cli prompt generate feature-development --interactive
  vibes-mcp-cli prompt generate bug-investigation --repo vibes-mcp-cli --severity critical
  vibes-mcp-cli prompt workspace-status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to list if no subcommand
		if len(args) == 0 {
			return runPromptList(cmd, args)
		}
		return fmt.Errorf("unknown subcommand: %s", args[0])
	},
}

var promptListCmd = &cobra.Command{
	Use:   "list [category]",
	Short: "List available prompt templates",
	Long: `List all available prompt templates, optionally filtered by category.

Categories:
- general: Feature development, bug investigation, documentation
- languages: Go, Python, TypeScript specific templates
- workflows: Testing, optimization, multi-step processes  
- workspace: Vibes workspace specific templates

Examples:
  vibes-mcp-cli prompt list
  vibes-mcp-cli prompt list general
  vibes-mcp-cli prompt list languages`,
	RunE: runPromptList,
}

var promptShowCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Show template details and structure",
	Long: `Display detailed information about a specific template including:
- Template description and purpose
- Required parameters and placeholders
- Example usage patterns
- Success criteria and validation steps

Examples:
  vibes-mcp-cli prompt show feature-development
  vibes-mcp-cli prompt show go/service-implementation`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptShow,
}

var promptGenerateCmd = &cobra.Command{
	Use:   "generate <template-name>",
	Short: "Generate a prompt from template",
	Long: `Generate a structured AI prompt from the specified template.

The command supports multiple modes:
- Interactive mode: Guided prompt generation with step-by-step input
- Direct mode: Generate using command-line flags
- Auto-detect mode: Automatically detect workspace context

Output options:
- Clipboard: Copy directly to system clipboard
- File: Save to specified file path
- Stdout: Print to terminal
- UI integration: Send to existing chat interface

AI tool integration:
- Claude: Send directly to Claude API
- Context7: Use with real-time documentation
- Beastmode: Trigger autonomous development mode

Examples:
  vibes-mcp-cli prompt generate feature-development --interactive
  vibes-mcp-cli prompt generate bug-investigation --repo vibes-mcp-cli --severity critical --clipboard
  vibes-mcp-cli prompt generate testing --language python --framework pytest --send-to-claude`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptGenerate,
}

var promptValidateCmd = &cobra.Command{
	Use:   "validate [template-name]",
	Short: "Validate prompt templates",
	Long: `Validate prompt templates for quality, completeness, and consistency.

Validation includes:
- Structure and required sections
- Code example quality and syntax
- Placeholder consistency
- Success criteria completeness
- Integration with workspace patterns

Examples:
  vibes-mcp-cli prompt validate                    # Validate all templates
  vibes-mcp-cli prompt validate feature-development # Validate specific template`,
	RunE: runPromptValidate,
}

var promptWorkspaceStatusCmd = &cobra.Command{
	Use:   "workspace-status",
	Short: "Show workspace context for prompt generation",
	Long: `Display current workspace context that can be used for automatic 
prompt generation, including:
- Detected repositories and languages
- Current working directory context
- Available frameworks and tools
- Recent development activity
- Suggested templates based on context

This information helps with auto-detection and context-aware prompt generation.`,
	RunE: runPromptWorkspaceStatus,
}

var promptCreateCmd = &cobra.Command{
	Use:   "create <template-name>",
	Short: "Create a new custom template",
	Long: `Create a new custom prompt template interactively.

The creation process guides you through:
- Template metadata and description
- Required parameters and placeholders
- Code examples and patterns
- Success criteria and validation steps
- Integration with existing templates

Examples:
  vibes-mcp-cli prompt create custom-optimization --interactive
  vibes-mcp-cli prompt create team-specific-pattern`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptCreate,
}

var promptUpdateCmd = &cobra.Command{
	Use:   "update <template-name>",
	Short: "Update an existing template",
	Long: `Update an existing prompt template with new content or improvements.

Supports:
- Interactive editing mode
- Validation of changes
- Backup of original template
- Version tracking and history

Examples:
  vibes-mcp-cli prompt update feature-development
  vibes-mcp-cli prompt update custom-template --interactive`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptUpdate,
}

var promptDeleteCmd = &cobra.Command{
	Use:   "delete <template-name>",
	Short: "Delete a custom template",
	Long: `Delete a custom prompt template. Built-in templates cannot be deleted.

Includes safety features:
- Confirmation prompt
- Backup before deletion
- Undo capability
- Usage tracking and warnings

Examples:
  vibes-mcp-cli prompt delete custom-template
  vibes-mcp-cli prompt delete old-pattern --force`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptDelete,
}

var promptHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show prompt generation history",
	Long: `Display history of generated prompts including:
- Generation timestamp and context
- Template used and parameters
- Output method and success status
- AI tool integration results
- Reuse and modification tracking

Examples:
  vibes-mcp-cli prompt history
  vibes-mcp-cli prompt history --limit 10`,
	RunE: runPromptHistory,
}

var promptConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage prompt configuration",
	Long: `Configure prompt generation preferences and defaults.

Configuration includes:
- Default repositories and languages
- Preferred AI tools and integration
- Output preferences and formats
- Template customization options
- Quality and validation settings

Examples:
  vibes-mcp-cli prompt config --set preferred-language=go
  vibes-mcp-cli prompt config --set default-repo=vibes-mcp-cli
  vibes-mcp-cli prompt config --list`,
	RunE: runPromptConfig,
}

func init() {
	rootCmd.AddCommand(promptCmd)

	// Add subcommands
	promptCmd.AddCommand(promptListCmd)
	promptCmd.AddCommand(promptShowCmd)
	promptCmd.AddCommand(promptGenerateCmd)
	promptCmd.AddCommand(promptValidateCmd)
	promptCmd.AddCommand(promptWorkspaceStatusCmd)
	promptCmd.AddCommand(promptCreateCmd)
	promptCmd.AddCommand(promptUpdateCmd)
	promptCmd.AddCommand(promptDeleteCmd)
	promptCmd.AddCommand(promptHistoryCmd)
	promptCmd.AddCommand(promptConfigCmd)

	// Template selection flags (for generate command)
	promptGenerateCmd.Flags().StringVarP(&promptCategory, "category", "c", "", "Template category")
	promptGenerateCmd.Flags().StringVarP(&promptRepo, "repo", "r", "", "Repository name")
	promptGenerateCmd.Flags().StringVarP(&promptLanguage, "language", "l", "", "Programming language")
	promptGenerateCmd.Flags().StringVarP(&promptFramework, "framework", "f", "", "Framework name")
	promptGenerateCmd.Flags().StringVar(&promptComponent, "component", "", "Component name")
	promptGenerateCmd.Flags().StringVarP(&promptSeverity, "severity", "s", "medium", "Issue severity (critical, high, medium, low)")
	promptGenerateCmd.Flags().StringVarP(&promptPriority, "priority", "p", "p2", "Issue priority (p0, p1, p2, p3)")

	// Generation mode flags
	promptGenerateCmd.Flags().BoolVarP(&promptInteractive, "interactive", "i", false, "Interactive template filling")
	promptGenerateCmd.Flags().BoolVar(&promptAutoDetect, "auto-detect", false, "Auto-detect workspace context")
	promptGenerateCmd.Flags().BoolVarP(&promptValidate, "validate", "v", true, "Validate generated prompt")

	// Output flags
	promptGenerateCmd.Flags().StringVarP(&promptOutput, "output", "o", "", "Output file path")
	promptGenerateCmd.Flags().BoolVar(&promptClipboard, "clipboard", false, "Copy to system clipboard")
	promptGenerateCmd.Flags().BoolVar(&promptStdout, "stdout", true, "Print to stdout (default)")

	// AI integration flags
	promptGenerateCmd.Flags().BoolVar(&promptSendToClaude, "send-to-claude", false, "Send directly to Claude API")
	promptGenerateCmd.Flags().BoolVar(&promptUseContext7, "use-context7", false, "Use Context7 for documentation")
	promptGenerateCmd.Flags().BoolVar(&promptBeastmode, "beastmode", false, "Trigger Beastmode autonomous development")

	// List command flags
	promptListCmd.Flags().StringVarP(&promptCategory, "category", "c", "", "Filter by category")
	promptListCmd.Flags().BoolVar(&promptValidate, "validate", false, "Show validation status")

	// Create/Update command flags
	promptCreateCmd.Flags().BoolVarP(&promptInteractive, "interactive", "i", true, "Interactive creation mode")
	promptCreateCmd.Flags().StringVar(&promptTemplatePath, "from-file", "", "Create from existing template file")
	
	promptUpdateCmd.Flags().BoolVarP(&promptInteractive, "interactive", "i", true, "Interactive update mode")
	promptUpdateCmd.Flags().BoolVar(&promptValidate, "validate", true, "Validate changes")

	// Delete command flags
	promptDeleteCmd.Flags().BoolVar(&promptForce, "force", false, "Force delete without confirmation")

	// History command flags
	promptHistoryCmd.Flags().IntVar(&promptSeverity, "limit", 20, "Limit number of history entries")
	promptHistoryCmd.Flags().StringVar(&promptCategory, "filter", "", "Filter by template category")
}

func runPromptList(cmd *cobra.Command, args []string) error {
	logger.Info("Running prompt list command", zap.String("category", promptCategory))

	// Initialize prompt manager
	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	// Determine category filter
	category := promptCategory
	if len(args) > 0 {
		category = args[0]
	}

	// List templates
	templates, err := manager.ListTemplates(category)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Display results
	if len(templates) == 0 {
		fmt.Printf("No templates found")
		if category != "" {
			fmt.Printf(" in category '%s'", category)
		}
		fmt.Println()
		return nil
	}

	fmt.Printf("📋 Available Prompt Templates")
	if category != "" {
		fmt.Printf(" (Category: %s)", category)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))

	// Group by category for display
	categorized := make(map[string][]prompt.Template)
	for _, template := range templates {
		categorized[template.Category] = append(categorized[template.Category], template)
	}

	for cat, tmpls := range categorized {
		fmt.Printf("\n📂 %s:\n", strings.Title(cat))
		for _, tmpl := range tmpls {
			status := "✅"
			if promptValidate {
				if valid, _ := manager.ValidateTemplate(tmpl.Name); !valid {
					status = "❌"
				}
			}
			fmt.Printf("  %s %s - %s\n", status, tmpl.Name, tmpl.Description)
			if tmpl.Language != "" {
				fmt.Printf("      Language: %s", tmpl.Language)
				if tmpl.Framework != "" {
					fmt.Printf(", Framework: %s", tmpl.Framework)
				}
				fmt.Println()
			}
		}
	}

	fmt.Printf("\n💡 Use 'vibes-mcp-cli prompt show <template>' for details\n")
	fmt.Printf("💡 Use 'vibes-mcp-cli prompt generate <template>' to create prompts\n")

	return nil
}

func runPromptShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	logger.Info("Running prompt show command", zap.String("template", templateName))

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	template, err := manager.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Display template details
	fmt.Printf("📄 Template: %s\n", template.Name)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Category: %s\n", template.Category)
	if template.Language != "" {
		fmt.Printf("Language: %s\n", template.Language)
	}
	if template.Framework != "" {
		fmt.Printf("Framework: %s\n", template.Framework)
	}
	fmt.Printf("Description: %s\n", template.Description)

	// Show parameters
	if len(template.Parameters) > 0 {
		fmt.Printf("\n📝 Required Parameters:\n")
		for _, param := range template.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}
			fmt.Printf("  • %s: %s%s\n", param.Name, param.Description, required)
			if param.Default != "" {
				fmt.Printf("    Default: %s\n", param.Default)
			}
		}
	}

	// Show usage examples
	if len(template.Examples) > 0 {
		fmt.Printf("\n💡 Usage Examples:\n")
		for i, example := range template.Examples {
			fmt.Printf("  %d. %s\n", i+1, example)
		}
	}

	// Show validation status
	if valid, issues := manager.ValidateTemplate(templateName); !valid {
		fmt.Printf("\n⚠️  Validation Issues:\n")
		for _, issue := range issues {
			fmt.Printf("  • %s\n", issue)
		}
	} else {
		fmt.Printf("\n✅ Template validation: Passed\n")
	}

	fmt.Printf("\n🚀 Generate: vibes-mcp-cli prompt generate %s --interactive\n", templateName)

	return nil
}

func runPromptGenerate(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	logger.Info("Running prompt generate command", 
		zap.String("template", templateName),
		zap.Bool("interactive", promptInteractive),
		zap.Bool("auto_detect", promptAutoDetect))

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	// Auto-detect workspace context if requested
	var context *prompt.WorkspaceContext
	if promptAutoDetect {
		context, err = manager.DetectWorkspaceContext()
		if err != nil {
			logger.Warn("Failed to auto-detect context, continuing without", zap.Error(err))
		} else {
			logger.Info("Auto-detected workspace context", 
				zap.String("repo", context.Repository),
				zap.String("language", context.Language))
		}
	}

	// Prepare generation config
	config := &prompt.GenerationConfig{
		TemplateName: templateName,
		Interactive:  promptInteractive,
		Context:      context,
		Parameters: map[string]string{
			"repo":      promptRepo,
			"language":  promptLanguage,
			"framework": promptFramework,
			"component": promptComponent,
			"severity":  promptSeverity,
			"priority":  promptPriority,
		},
		Validate: promptValidate,
	}

	// Generate prompt
	result, err := manager.GeneratePrompt(config)
	if err != nil {
		return fmt.Errorf("failed to generate prompt: %w", err)
	}

	// Handle output options
	if err := handlePromptOutput(result, manager); err != nil {
		return fmt.Errorf("failed to handle output: %w", err)
	}

	// Handle AI tool integration
	if err := handleAIIntegration(result, manager); err != nil {
		return fmt.Errorf("failed to handle AI integration: %w", err)
	}

	logger.Info("Prompt generated successfully", 
		zap.String("template", templateName),
		zap.Int("length", len(result.Content)))

	return nil
}

func runPromptValidate(cmd *cobra.Command, args []string) error {
	logger.Info("Running prompt validate command")

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	// Validate specific template or all templates
	if len(args) > 0 {
		templateName := args[0]
		valid, issues := manager.ValidateTemplate(templateName)
		
		fmt.Printf("🔍 Validation Results for '%s'\n", templateName)
		fmt.Println(strings.Repeat("=", 50))
		
		if valid {
			fmt.Println("✅ Template validation: PASSED")
		} else {
			fmt.Println("❌ Template validation: FAILED")
			fmt.Println("\nIssues found:")
			for _, issue := range issues {
				fmt.Printf("  • %s\n", issue)
			}
		}
	} else {
		// Validate all templates
		report, err := manager.ValidateAllTemplates()
		if err != nil {
			return fmt.Errorf("failed to validate templates: %w", err)
		}

		fmt.Printf("🔍 Template Validation Report\n")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Total Templates: %d\n", report.Total)
		fmt.Printf("Valid Templates: %d\n", report.Valid)
		fmt.Printf("Invalid Templates: %d\n", report.Invalid)
		fmt.Printf("Average Score: %d/100\n", report.AverageScore)

		if len(report.Issues) > 0 {
			fmt.Println("\n❌ Templates with Issues:")
			for template, issues := range report.Issues {
				fmt.Printf("  %s:\n", template)
				for _, issue := range issues {
					fmt.Printf("    • %s\n", issue)
				}
			}
		}
	}

	return nil
}

func runPromptWorkspaceStatus(cmd *cobra.Command, args []string) error {
	logger.Info("Running prompt workspace-status command")

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	context, err := manager.DetectWorkspaceContext()
	if err != nil {
		return fmt.Errorf("failed to detect workspace context: %w", err)
	}

	fmt.Printf("🏗️  Workspace Context Status\n")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Current Directory: %s\n", context.WorkingDirectory)
	fmt.Printf("Detected Repository: %s\n", context.Repository)
	fmt.Printf("Primary Language: %s\n", context.Language)
	if context.Framework != "" {
		fmt.Printf("Framework: %s\n", context.Framework)
	}

	if len(context.AvailableLanguages) > 0 {
		fmt.Printf("Available Languages: %s\n", strings.Join(context.AvailableLanguages, ", "))
	}

	if len(context.RecentFiles) > 0 {
		fmt.Printf("\n📁 Recent Activity:\n")
		for i, file := range context.RecentFiles {
			if i >= 5 { // Limit to 5 recent files
				break
			}
			fmt.Printf("  • %s\n", file)
		}
	}

	// Suggest templates based on context
	suggestions := manager.SuggestTemplates(context)
	if len(suggestions) > 0 {
		fmt.Printf("\n💡 Suggested Templates:\n")
		for _, suggestion := range suggestions {
			fmt.Printf("  • %s - %s\n", suggestion.Name, suggestion.Reason)
		}
	}

	return nil
}

func runPromptCreate(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	logger.Info("Running prompt create command", zap.String("template", templateName))

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	if promptInteractive {
		return manager.CreateTemplateInteractive(templateName)
	}

	if promptTemplatePath != "" {
		return manager.CreateTemplateFromFile(templateName, promptTemplatePath)
	}

	return fmt.Errorf("either --interactive or --from-file must be specified")
}

func runPromptUpdate(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	logger.Info("Running prompt update command", zap.String("template", templateName))

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	return manager.UpdateTemplate(templateName, promptInteractive, promptValidate)
}

func runPromptDelete(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	logger.Info("Running prompt delete command", zap.String("template", templateName))

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	if !promptForce {
		fmt.Printf("Are you sure you want to delete template '%s'? (y/N): ", templateName)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response != "y" && response != "yes" {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	return manager.DeleteTemplate(templateName)
}

func runPromptHistory(cmd *cobra.Command, args []string) error {
	logger.Info("Running prompt history command")

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	// Parse limit from severity field (reused for history limit)
	limit := 20
	if promptSeverity != "" {
		if l, err := strconv.Atoi(promptSeverity); err == nil {
			limit = l
		}
	}

	history, err := manager.GetHistory(limit, promptCategory)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	fmt.Printf("📚 Prompt Generation History\n")
	fmt.Println(strings.Repeat("=", 50))

	if len(history) == 0 {
		fmt.Println("No prompt generation history found.")
		return nil
	}

	for i, entry := range history {
		fmt.Printf("%d. %s (%s)\n", i+1, entry.Template, entry.Timestamp.Format("2006-01-02 15:04"))
		fmt.Printf("   Repository: %s, Language: %s\n", entry.Repository, entry.Language)
		if entry.OutputMethod != "" {
			fmt.Printf("   Output: %s\n", entry.OutputMethod)
		}
		if entry.AITool != "" {
			fmt.Printf("   AI Tool: %s\n", entry.AITool)
		}
		fmt.Println()
	}

	return nil
}

func runPromptConfig(cmd *cobra.Command, args []string) error {
	logger.Info("Running prompt config command")

	manager, err := prompt.NewManager(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	// Handle configuration commands
	if len(args) == 0 {
		// Show current configuration
		config := manager.GetConfig()
		fmt.Printf("⚙️  Prompt Configuration\n")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Default Repository: %s\n", config.DefaultRepository)
		fmt.Printf("Preferred Language: %s\n", config.PreferredLanguage)
		fmt.Printf("Auto Clipboard: %t\n", config.AutoClipboard)
		fmt.Printf("Auto Validate: %t\n", config.AutoValidate)
		fmt.Printf("Preferred AI Tool: %s\n", config.PreferredAITool)
		return nil
	}

	// Handle set/get commands
	action := args[0]
	switch action {
	case "set":
		if len(args) != 2 {
			return fmt.Errorf("usage: vibes-mcp-cli prompt config set key=value")
		}
		parts := strings.SplitN(args[1], "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format, use key=value")
		}
		return manager.SetConfig(parts[0], parts[1])
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: vibes-mcp-cli prompt config get key")
		}
		value := manager.GetConfigValue(args[1])
		fmt.Printf("%s = %s\n", args[1], value)
		return nil
	case "list":
		return runPromptConfig(cmd, []string{}) // Show all config
	default:
		return fmt.Errorf("unknown config action: %s", action)
	}
}

func handlePromptOutput(result *prompt.GenerationResult, manager *prompt.Manager) error {
	// Handle stdout (default)
	if promptStdout || (!promptClipboard && promptOutput == "") {
		fmt.Println(result.Content)
	}

	// Handle clipboard
	if promptClipboard {
		if err := manager.CopyToClipboard(result.Content); err != nil {
			logger.Warn("Failed to copy to clipboard", zap.Error(err))
		} else {
			fmt.Println("✅ Copied to clipboard")
		}
	}

	// Handle file output
	if promptOutput != "" {
		if err := manager.SaveToFile(result.Content, promptOutput); err != nil {
			return fmt.Errorf("failed to save to file: %w", err)
		}
		fmt.Printf("✅ Saved to %s\n", promptOutput)
	}

	return nil
}

func handleAIIntegration(result *prompt.GenerationResult, manager *prompt.Manager) error {
	// Handle Claude integration
	if promptSendToClaude {
		if err := sendToClaude(result.Content); err != nil {
			logger.Warn("Failed to send to Claude", zap.Error(err))
		} else {
			fmt.Println("✅ Sent to Claude")
		}
	}

	// Handle Context7 integration
	if promptUseContext7 {
		if err := manager.UseContext7(result); err != nil {
			logger.Warn("Failed to use Context7", zap.Error(err))
		} else {
			fmt.Println("✅ Context7 integration active")
		}
	}

	// Handle Beastmode integration
	if promptBeastmode {
		if err := manager.TriggerBeastmode(result); err != nil {
			logger.Warn("Failed to trigger Beastmode", zap.Error(err))
		} else {
			fmt.Println("✅ Beastmode activated")
		}
	}

	return nil
}

func sendToClaude(content string) error {
	// Reuse existing chat functionality to send to Claude
	ctx := context.Background()
	
	// Create client using existing patterns
	clientImpl, err := client.New(cfg.Provider, cfg.APIKey, cfg.BaseURL, logger)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	svc := service.NewService(clientImpl, logger)

	// Create chat message
	messages := []client.ChatMessage{
		{
			Role:    "user",
			Content: content,
		},
	}

	// Send to API
	req := &client.ChatCompletionRequest{
		Model:    cfg.Model,
		Messages: messages,
	}

	resp, err := svc.CreateChatCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send to Claude: %w", err)
	}

	// Display response
	if len(resp.Choices) > 0 {
		fmt.Printf("\n🤖 Claude Response:\n")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(resp.Choices[0].Message.Content)
	}

	return nil
}