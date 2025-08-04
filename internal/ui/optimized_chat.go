package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openai-cli/internal/client"
	contextpkg "openai-cli/internal/context"
	"openai-cli/internal/mcp"
	"openai-cli/internal/subagent"
	"openai-cli/internal/telemetry"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

// OptimizedChatConfig holds configuration for the optimized chat component
type OptimizedChatConfig struct {
	MaxTokens           int                      `json:"max_tokens"`
	SummaryThreshold    int                      `json:"summary_threshold"`
	RetainRecent        int                      `json:"retain_recent"`
	UseSubagents        bool                     `json:"use_subagents"`
	EnableOptimization  bool                     `json:"enable_optimization"`
	CompressionRatio    float64                  `json:"compression_ratio"`
	TelemetryClient     telemetry.Client         `json:"-"`
	Logger              *telemetry.TelemetryLogger `json:"-"`
}

// OptimizedChatComponent provides an enhanced chat interface with context optimization
type OptimizedChatComponent struct {
	// UI Components
	chatView     *tview.TextView
	input        *tview.InputField
	statsView    *tview.TextView
	subagentView *tview.TextView
	container    *tview.Flex

	// Core components
	mcpClient           *mcp.Client
	contextOptimizer    *contextpkg.ContextOptimizer
	subagentManager     *subagent.SubagentManager
	config             *OptimizedChatConfig

	// State
	conversation       []client.ChatMessage
	optimizationStats  OptimizationStats
	availableSubagents []mcp.SubagentInfo
}

// OptimizationStats tracks optimization performance metrics
type OptimizationStats struct {
	TotalMessages       int           `json:"total_messages"`
	TotalOptimizations  int           `json:"total_optimizations"`
	OriginalTokens      int           `json:"original_tokens"`
	OptimizedTokens     int           `json:"optimized_tokens"`
	TokensSaved         int           `json:"tokens_saved"`
	CompressionRatio    float64       `json:"compression_ratio"`
	AverageProcessTime  time.Duration `json:"average_process_time"`
	SubagentUsage       map[string]int `json:"subagent_usage"`
	LastOptimization    time.Time     `json:"last_optimization"`
}

// NewOptimizedChatComponent creates a new optimized chat component
func NewOptimizedChatComponent(mcpClient *mcp.Client, config *OptimizedChatConfig) *OptimizedChatComponent {
	occ := &OptimizedChatComponent{
		mcpClient:      mcpClient,
		config:         config,
		conversation:   make([]client.ChatMessage, 0),
		optimizationStats: OptimizationStats{
			SubagentUsage: make(map[string]int),
		},
	}

	// Initialize context optimizer
	strategy := mcp.OptimizationStrategy{
		MaxTokens:         config.MaxTokens,
		SummaryThreshold:  config.SummaryThreshold,
		CompressionRatio:  config.CompressionRatio,
		RetainRecent:      config.RetainRecent,
		PriorityWeighting: true,
		UseSubagents:      config.UseSubagents,
	}
	occ.contextOptimizer = contextpkg.NewContextOptimizer(mcpClient, strategy)

	// Initialize subagent manager
	managerConfig := &subagent.ManagerConfig{
		MaxConcurrentTasks:  5,
		DefaultTimeout:      30 * time.Second,
		RetryAttempts:       2,
		RetryDelay:          1 * time.Second,
		HealthCheckInterval: 5 * time.Minute,
		EnableLoadBalancing: true,
		EnableFallback:      true,
	}
	occ.subagentManager = subagent.NewSubagentManager(mcpClient, managerConfig)

	// Initialize UI components
	occ.initializeUI()

	// Load available subagents
	go occ.loadSubagents()

	return occ
}

// initializeUI sets up the UI components
func (occ *OptimizedChatComponent) initializeUI() {
	// Chat view with optimization indicators
	occ.chatView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	occ.chatView.SetBorder(true).
		SetTitle("Optimized Chat (Context Aware)").
		SetTitleAlign(tview.AlignLeft)

	// Input field
	occ.input = tview.NewInputField().
		SetLabel("You: ").
		SetFieldWidth(0)
	
	// Stats view showing optimization metrics
	occ.statsView = tview.NewTextView().
		SetDynamicColors(true)
	occ.statsView.SetBorder(true).
		SetTitle("Optimization Stats").
		SetTitleAlign(tview.AlignLeft)
	occ.updateStatsDisplay()

	// Subagent status view
	occ.subagentView = tview.NewTextView().
		SetDynamicColors(true)
	occ.subagentView.SetBorder(true).
		SetTitle("Active Subagents").
		SetTitleAlign(tview.AlignLeft)

	// Layout
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(occ.statsView, 0, 1, false).
		AddItem(occ.subagentView, 0, 1, false)

	occ.container = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(occ.chatView, 0, 1, false).
			AddItem(occ.input, 1, 0, true), 0, 3, true).
		AddItem(rightPanel, 40, 0, false)

	// Input handler
	occ.input.SetDoneFunc(occ.handleUserInput)

	// Keyboard shortcuts
	occ.setupKeyboardShortcuts()
}

// handleUserInput processes user input and handles optimization
func (occ *OptimizedChatComponent) handleUserInput(key tcell.Key) {
	if key != tcell.KeyEnter {
		return
	}

	userInput := strings.TrimSpace(occ.input.GetText())
	if userInput == "" {
		return
	}

	occ.input.SetText("")
	
	// Add user message to conversation
	userMessage := client.ChatMessage{
		Role:    "user",
		Content: userInput,
	}
	occ.conversation = append(occ.conversation, userMessage)
	occ.optimizationStats.TotalMessages++

	// Display user message
	fmt.Fprintf(occ.chatView, "[blue]You:[white] %s\n", userInput)

	// Log user action
	if occ.config.TelemetryClient != nil {
		telemetry.LogUserAction(occ.config.TelemetryClient, "optimized_chat_message", map[string]interface{}{
			"message_length":     len(userInput),
			"conversation_length": len(occ.conversation),
			"optimization_enabled": occ.config.EnableOptimization,
		})
	}

	// Process with optimization if enabled
	go occ.processOptimizedResponse(userInput)
}

// processOptimizedResponse handles response generation with context optimization
func (occ *OptimizedChatComponent) processOptimizedResponse(userInput string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	startTime := time.Now()

	// Classify the input to determine if subagent routing is beneficial
	inputType := occ.classifyInput(userInput)
	
	var optimizationResult *mcp.OptimizationResult
	var err error

	if occ.config.EnableOptimization && len(occ.conversation) > 3 {
		// Create context chunks from conversation
		additionalContext := occ.createContextChunks()

		// Optimize conversation context
		_, optimizationResult, err = occ.contextOptimizer.OptimizeContext(
			ctx, 
			occ.conversation, 
			additionalContext,
		)

		if err != nil {
			if occ.config.Logger != nil {
				occ.config.Logger.Error("Context optimization failed", 
					zap.Error(err))
			}
			// Fallback to original conversation
		} else {
			// Update optimization stats
			occ.updateOptimizationStats(optimizationResult)
			
			// Display optimization info
			compressionRatio := float64(optimizationResult.OptimizedTokens) / float64(optimizationResult.OriginalTokens) * 100
			fmt.Fprintf(occ.chatView, "[green]ⓘ Context optimized: %d→%d tokens (%.1f%% compression)[white]\n",
				optimizationResult.OriginalTokens,
				optimizationResult.OptimizedTokens,
				100-compressionRatio)
		}
	}

	// Route to subagent if beneficial
	if occ.config.UseSubagents && occ.shouldUseSubagent(inputType, userInput) {
		response, subagentErr := occ.routeToSubagent(ctx, inputType, userInput)
		if subagentErr == nil {
			// Display subagent response
			fmt.Fprintf(occ.chatView, "[yellow]🤖 %s:[white] %s\n", response.SubagentID, response.Summary)
			
			// Add to conversation
			assistantMessage := client.ChatMessage{
				Role:    "assistant",
				Content: response.Summary,
			}
			occ.conversation = append(occ.conversation, assistantMessage)
			
			// Update stats
			occ.optimizationStats.SubagentUsage[response.SubagentID]++
			
			processingTime := time.Since(startTime)
			occ.updateProcessingTime(processingTime)
			occ.updateStatsDisplay()
			occ.updateSubagentDisplay()
			return
		} else if occ.config.Logger != nil {
			occ.config.Logger.Warn("Subagent routing failed, falling back to standard processing", 
				zap.Error(subagentErr))
		}
	}

	// Standard LLM processing (fallback or primary path)
	// This would integrate with your existing chat completion logic
	responseContent := "This would integrate with your LLM provider for the actual response generation."
	
	// Display response
	fmt.Fprintf(occ.chatView, "[green]Assistant:[white] %s\n", responseContent)
	
	// Add to conversation
	assistantMessage := client.ChatMessage{
		Role:    "assistant",
		Content: responseContent,
	}
	occ.conversation = append(occ.conversation, assistantMessage)
	
	processingTime := time.Since(startTime)
	occ.updateProcessingTime(processingTime)
	occ.updateStatsDisplay()
	
	// Scroll to bottom
	occ.chatView.ScrollToEnd()
}

// classifyInput determines the type of user input for optimal routing
func (occ *OptimizedChatComponent) classifyInput(input string) string {
	input = strings.ToLower(input)
	
	// Code-related keywords
	codeKeywords := []string{"code", "function", "bug", "error", "debug", "programming", "script", "syntax"}
	for _, keyword := range codeKeywords {
		if strings.Contains(input, keyword) {
			return "code_analysis"
		}
	}
	
	// Documentation keywords
	docKeywords := []string{"explain", "documentation", "readme", "how to", "tutorial", "guide"}
	for _, keyword := range docKeywords {
		if strings.Contains(input, keyword) {
			return "doc_summary"
		}
	}
	
	// Data analysis keywords
	dataKeywords := []string{"data", "analyze", "json", "csv", "logs", "metrics", "statistics"}
	for _, keyword := range dataKeywords {
		if strings.Contains(input, keyword) {
			return "data_analysis"
		}
	}
	
	return "general_summary"
}

// shouldUseSubagent determines if subagent routing would be beneficial
func (occ *OptimizedChatComponent) shouldUseSubagent(inputType, input string) bool {
	// Use subagents for specialized tasks and longer inputs
	if inputType != "general_summary" && len(input) > 50 {
		return true
	}
	
	// Use subagents if conversation context is complex
	if len(occ.conversation) > 10 {
		return true
	}
	
	return false
}

// routeToSubagent routes a task to the appropriate subagent
func (occ *OptimizedChatComponent) routeToSubagent(ctx context.Context, taskType, input string) (*mcp.SubagentResponse, error) {
	// Convert conversation to context chunks
	contextChunks := occ.createContextChunks()
	
	// Route via context optimizer
	return occ.contextOptimizer.RouteTaskToSubagent(ctx, taskType, input, contextChunks)
}

// createContextChunks converts conversation to context chunks
func (occ *OptimizedChatComponent) createContextChunks() []mcp.ContextChunk {
	chunks := make([]mcp.ContextChunk, 0, len(occ.conversation))
	
	for i, msg := range occ.conversation {
		chunk := mcp.ContextChunk{
			ID:         fmt.Sprintf("msg_%d", i),
			Type:       "text",
			Content:    msg.Content,
			TokenCount: len(msg.Content) / 4, // Rough estimation
			Priority:   5, // Default priority
			Timestamp:  time.Now().Add(-time.Duration(len(occ.conversation)-i) * time.Minute),
			Source:     msg.Role,
			Tags:       []string{msg.Role},
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks
}

// updateOptimizationStats updates optimization performance metrics
func (occ *OptimizedChatComponent) updateOptimizationStats(result *mcp.OptimizationResult) {
	occ.optimizationStats.TotalOptimizations++
	occ.optimizationStats.OriginalTokens += result.OriginalTokens
	occ.optimizationStats.OptimizedTokens += result.OptimizedTokens
	occ.optimizationStats.TokensSaved += result.TokensSaved
	occ.optimizationStats.LastOptimization = time.Now()
	
	// Update compression ratio
	if occ.optimizationStats.OriginalTokens > 0 {
		occ.optimizationStats.CompressionRatio = float64(occ.optimizationStats.OptimizedTokens) / 
			float64(occ.optimizationStats.OriginalTokens)
	}
	
	// Update subagent usage
	for subagent, count := range result.SubagentUsage {
		occ.optimizationStats.SubagentUsage[subagent] += count
	}
}

// updateProcessingTime updates average processing time
func (occ *OptimizedChatComponent) updateProcessingTime(duration time.Duration) {
	if occ.optimizationStats.TotalMessages > 0 {
		occ.optimizationStats.AverageProcessTime = time.Duration(
			(int64(occ.optimizationStats.AverageProcessTime)*int64(occ.optimizationStats.TotalMessages-1) + 
			 int64(duration)) / int64(occ.optimizationStats.TotalMessages))
	} else {
		occ.optimizationStats.AverageProcessTime = duration
	}
}

// updateStatsDisplay refreshes the optimization stats display
func (occ *OptimizedChatComponent) updateStatsDisplay() {
	stats := &occ.optimizationStats
	
	occ.statsView.Clear()
	fmt.Fprintf(occ.statsView, "[yellow]Messages:[white] %d\n", stats.TotalMessages)
	fmt.Fprintf(occ.statsView, "[yellow]Optimizations:[white] %d\n", stats.TotalOptimizations)
	
	if stats.OriginalTokens > 0 {
		fmt.Fprintf(occ.statsView, "[yellow]Token Savings:[white] %d/%d (%.1f%%)\n", 
			stats.TokensSaved, stats.OriginalTokens, 
			float64(stats.TokensSaved)/float64(stats.OriginalTokens)*100)
		fmt.Fprintf(occ.statsView, "[yellow]Compression:[white] %.1f%%\n", 
			stats.CompressionRatio*100)
	}
	
	fmt.Fprintf(occ.statsView, "[yellow]Avg Process Time:[white] %v\n", 
		stats.AverageProcessTime.Truncate(time.Millisecond))
	
	if len(stats.SubagentUsage) > 0 {
		fmt.Fprintf(occ.statsView, "\n[yellow]Subagent Usage:[white]\n")
		for subagent, count := range stats.SubagentUsage {
			fmt.Fprintf(occ.statsView, "  %s: %d\n", subagent, count)
		}
	}
}

// updateSubagentDisplay refreshes the subagent status display
func (occ *OptimizedChatComponent) updateSubagentDisplay() {
	occ.subagentView.Clear()
	
	if len(occ.availableSubagents) == 0 {
		fmt.Fprintf(occ.subagentView, "[red]No subagents available[white]")
		return
	}
	
	fmt.Fprintf(occ.subagentView, "[yellow]Available:[white] %d\n\n", len(occ.availableSubagents))
	
	for _, subagent := range occ.availableSubagents {
		status := "[green]●[white]"
		if subagent.Status != "active" {
			status = "[red]●[white]"
		}
		
		fmt.Fprintf(occ.subagentView, "%s [cyan]%s[white]\n", status, subagent.Name)
		fmt.Fprintf(occ.subagentView, "  %s\n", subagent.Description)
		
		if len(subagent.Capabilities) > 0 {
			taskTypes := make([]string, len(subagent.Capabilities))
			for i, cap := range subagent.Capabilities {
				taskTypes[i] = cap.TaskType
			}
			fmt.Fprintf(occ.subagentView, "  [dim]Tasks: %s[white]\n", strings.Join(taskTypes, ", "))
		}
		fmt.Fprintf(occ.subagentView, "\n")
	}
}

// loadSubagents loads the list of available subagents
func (occ *OptimizedChatComponent) loadSubagents() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := occ.contextOptimizer.GetAvailableSubagents(ctx)
	if err != nil {
		if occ.config.Logger != nil {
			occ.config.Logger.Error("Failed to load subagents", 
				zap.Error(err))
		}
		return
	}
	
	occ.availableSubagents = result.Subagents
	occ.updateSubagentDisplay()
}

// setupKeyboardShortcuts sets up keyboard shortcuts for the component
func (occ *OptimizedChatComponent) setupKeyboardShortcuts() {
	occ.container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlO:
			// Toggle optimization
			occ.config.EnableOptimization = !occ.config.EnableOptimization
			title := "Optimized Chat"
			if occ.config.EnableOptimization {
				title += " (Context Aware)"
			} else {
				title += " (Standard)"
			}
			occ.chatView.SetTitle(title)
			return nil
		case tcell.KeyCtrlS:
			// Toggle subagent usage
			occ.config.UseSubagents = !occ.config.UseSubagents
			return nil
		case tcell.KeyCtrlR:
			// Refresh subagents
			go occ.loadSubagents()
			return nil
		}
		return event
	})
}

// GetContainer returns the main UI container
func (occ *OptimizedChatComponent) GetContainer() tview.Primitive {
	return occ.container
}

// GetStats returns current optimization statistics
func (occ *OptimizedChatComponent) GetStats() OptimizationStats {
	return occ.optimizationStats
}

// Reset clears the conversation and resets stats
func (occ *OptimizedChatComponent) Reset() {
	occ.conversation = make([]client.ChatMessage, 0)
	occ.optimizationStats = OptimizationStats{
		SubagentUsage: make(map[string]int),
	}
	occ.chatView.Clear()
	occ.updateStatsDisplay()
	occ.updateSubagentDisplay()
}