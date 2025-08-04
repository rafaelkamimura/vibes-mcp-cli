package components

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/prompt"
)

// WorkspaceStatusCallbacks defines callback functions for workspace status events
type WorkspaceStatusCallbacks struct {
	OnContextRefresh    func(context *prompt.WorkspaceContext)
	OnTemplatesSuggest  func(suggestions []prompt.TemplateSuggestion)
	OnQuickAction       func(action string, data interface{})
	OnError             func(err error)
}

// WorkspaceStatus provides a TUI component for displaying workspace context and status
type WorkspaceStatus struct {
	*tview.Flex

	// Configuration
	config    *PromptUIConfig
	callbacks *WorkspaceStatusCallbacks
	manager   prompt.Manager
	logger    *zap.Logger

	// UI components
	contextPanel      *tview.TextView
	detectedPanel     *tview.TextView
	suggestionsPanel  *tview.List
	quickActionsPanel *tview.List
	refreshButton     *tview.Button
	statusBar         *tview.TextView

	// State
	context          *prompt.WorkspaceContext
	suggestions      []prompt.TemplateSuggestion
	isRefreshing     bool
	lastRefresh      time.Time
	autoRefreshTimer *time.Ticker
	keyBindings      *KeyBindings

	// Context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWorkspaceStatus creates a new workspace status component
func NewWorkspaceStatus(manager prompt.Manager, config *PromptUIConfig, callbacks *WorkspaceStatusCallbacks, logger *zap.Logger) *WorkspaceStatus {
	if config == nil {
		config = DefaultPromptUIConfig()
	}

	if callbacks == nil {
		callbacks = &WorkspaceStatusCallbacks{}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	ws := &WorkspaceStatus{
		Flex:      tview.NewFlex(),
		config:    config,
		callbacks: callbacks,
		manager:   manager,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	ws.initKeyBindings()
	ws.initUI()
	ws.startAutoRefresh()
	ws.refreshContext()

	return ws
}

// initKeyBindings sets up keyboard shortcuts
func (ws *WorkspaceStatus) initKeyBindings() {
	ws.keyBindings = NewKeyBindings()

	// Refresh actions
	ws.keyBindings.AddRune('r', "Refresh", func() { ws.refreshContext() })
	ws.keyBindings.AddKey(tcell.KeyF5, "Refresh", func() { ws.refreshContext() })

	// Navigation
	ws.keyBindings.AddKey(tcell.KeyTab, "Next Panel", func() { ws.nextPanel() })
	ws.keyBindings.AddKey(tcell.KeyBacktab, "Prev Panel", func() { ws.prevPanel() })

	// Quick actions
	ws.keyBindings.AddKey(tcell.KeyEnter, "Execute Action", func() { ws.executeSelectedAction() })
	ws.keyBindings.AddRune('s', "Suggest Templates", func() { ws.suggestTemplates() })

	// Context actions
	ws.keyBindings.AddRune('c', "Copy Context", func() { ws.copyContextToClipboard() })
	ws.keyBindings.AddRune('e', "Export Context", func() { ws.exportContext() })
}

// initUI initializes the user interface components
func (ws *WorkspaceStatus) initUI() {
	ws.SetDirection(tview.FlexRow)

	// Create main content area
	contentArea := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Left column: context and detected info
	leftColumn := tview.NewFlex().SetDirection(tview.FlexRow)

	// Context panel
	ws.contextPanel = CreateStyledTextView("Workspace Context", ws.config.Theme, ws.config.Icons)
	ws.contextPanel.SetScrollable(true)
	leftColumn.AddItem(ws.contextPanel, 0, 1, true)

	// Detected info panel
	ws.detectedPanel = CreateStyledTextView("Auto-Detected", ws.config.Theme, ws.config.Icons)
	ws.detectedPanel.SetScrollable(true)
	leftColumn.AddItem(ws.detectedPanel, 0, 1, false)

	// Right column: suggestions and actions
	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow)

	// Suggestions panel
	ws.suggestionsPanel = CreateStyledList("Suggested Templates", ws.config.Theme, ws.config.Icons)
	ws.suggestionsPanel.SetSelectedFunc(ws.onSuggestionSelected)
	rightColumn.AddItem(ws.suggestionsPanel, 0, 1, false)

	// Quick actions panel
	ws.quickActionsPanel = CreateStyledList("Quick Actions", ws.config.Theme, ws.config.Icons)
	ws.quickActionsPanel.SetSelectedFunc(ws.onQuickActionSelected)
	ws.setupQuickActions()
	rightColumn.AddItem(ws.quickActionsPanel, 0, 1, false)

	// Add columns to content area
	contentArea.AddItem(leftColumn, 0, 1, true)
	contentArea.AddItem(rightColumn, 0, 1, false)

	// Create refresh button
	buttonArea := tview.NewFlex().SetDirection(tview.FlexColumn)
	ws.refreshButton = tview.NewButton("Refresh (r)")
	ws.refreshButton.SetSelectedFunc(ws.refreshContext)
	buttonArea.AddItem(ws.refreshButton, 0, 1, false)
	buttonArea.AddItem(tview.NewBox(), 0, 3, false) // Spacer

	// Create status bar
	ws.statusBar = tview.NewTextView()
	ws.statusBar.SetDynamicColors(true)
	ws.statusBar.SetTextAlign(tview.AlignLeft)

	// Layout
	ws.AddItem(contentArea, 0, 1, true)
	ws.AddItem(buttonArea, 3, 0, false)
	ws.AddItem(ws.statusBar, 1, 0, false)

	// Set up input capture
	ws.SetInputCapture(ws.handleKeyPress)

	// Initial update
	ws.updateUI()
}

// setupQuickActions populates the quick actions list
func (ws *WorkspaceStatus) setupQuickActions() {
	actions := []struct {
		name        string
		description string
		action      string
	}{
		{"Suggest Templates", "Get template suggestions for current context", "suggest"},
		{"Copy Context", "Copy workspace context to clipboard", "copy_context"},
		{"Export Context", "Export context to file", "export_context"},
		{"Refresh Context", "Refresh workspace detection", "refresh"},
		{"Browse Templates", "Open template browser", "browse"},
		{"Create Template", "Create new template", "create"},
	}

	for _, action := range actions {
		icon := ws.getActionIcon(action.action)
		mainText := fmt.Sprintf("%s %s", icon, action.name)
		ws.quickActionsPanel.AddItem(mainText, action.description, 0, nil)
	}
}

// getActionIcon returns an icon for the given action
func (ws *WorkspaceStatus) getActionIcon(action string) string {
	switch action {
	case "suggest":
		return ws.config.Icons.Lightning
	case "copy_context":
		return ws.config.Icons.Copy
	case "export_context":
		return ws.config.Icons.Save
	case "refresh":
		return ws.config.Icons.Settings
	case "browse":
		return ws.config.Icons.Search
	case "create":
		return ws.config.Icons.Generate
	default:
		return ws.config.Icons.Gear
	}
}

// handleKeyPress processes keyboard input
func (ws *WorkspaceStatus) handleKeyPress(event *tcell.EventKey) *tcell.EventKey {
	// Try key bindings first
	if ws.keyBindings.Handle(event) {
		return nil
	}

	return event
}

// Context refresh methods
func (ws *WorkspaceStatus) refreshContext() {
	if ws.isRefreshing {
		return
	}

	ws.isRefreshing = true
	ws.updateStatusBar("Refreshing workspace context...", "info")

	go func() {
		defer func() {
			ws.isRefreshing = false
			ws.lastRefresh = time.Now()
			ws.updateUI()
		}()

		// Detect workspace context
		context, err := ws.manager.DetectWorkspaceContext()
		if err != nil {
			ws.handleError(fmt.Errorf("failed to detect workspace context: %w", err))
			return
		}

		ws.context = context

		// Get template suggestions
		suggestions := ws.manager.SuggestTemplates(context)
		ws.suggestions = suggestions

		// Update UI on main thread
		ws.updateContextDisplay()
		ws.updateSuggestionsDisplay()
		ws.updateStatusBar("Context refreshed successfully", "success")

		// Call callbacks
		if ws.callbacks.OnContextRefresh != nil {
			ws.callbacks.OnContextRefresh(context)
		}

		if ws.callbacks.OnTemplatesSuggest != nil {
			ws.callbacks.OnTemplatesSuggest(suggestions)
		}
	}()
}

func (ws *WorkspaceStatus) startAutoRefresh() {
	if ws.config.AutoRefreshRate > 0 {
		ws.autoRefreshTimer = time.NewTicker(ws.config.AutoRefreshRate)
		go func() {
			for {
				select {
				case <-ws.autoRefreshTimer.C:
					ws.refreshContext()
				case <-ws.ctx.Done():
					ws.autoRefreshTimer.Stop()
					return
				}
			}
		}()
	}
}

// Display update methods
func (ws *WorkspaceStatus) updateContextDisplay() {
	if ws.context == nil {
		ws.contextPanel.SetText("No workspace context available")
		return
	}

	var content strings.Builder

	// Header
	colorStart, colorEnd := ws.config.Theme.ColorToTags(ws.config.Theme.Primary)
	content.WriteString(fmt.Sprintf("%sWorkspace Information%s\n\n", colorStart, colorEnd))

	// Basic info
	content.WriteString("[yellow]Directory:[white] ")
	content.WriteString(ws.context.WorkingDirectory)
	content.WriteString("\n")

	if ws.context.Repository != "" {
		content.WriteString("[yellow]Repository:[white] ")
		content.WriteString(filepath.Base(ws.context.Repository))
		content.WriteString("\n")
	}

	// Language and framework
	if ws.context.Language != "" {
		content.WriteString("[yellow]Primary Language:[white] ")
		content.WriteString(ws.context.Language)
		content.WriteString("\n")
	}

	if ws.context.Framework != "" {
		content.WriteString("[yellow]Framework:[white] ")
		content.WriteString(ws.context.Framework)
		content.WriteString("\n")
	}

	// Git information
	if ws.context.GitBranch != "" {
		content.WriteString("[yellow]Git Branch:[white] ")
		content.WriteString(ws.context.GitBranch)
		content.WriteString("\n")
	}

	if ws.context.GitStatus != "" && ws.context.GitStatus != "clean" {
		content.WriteString("[yellow]Git Status:[white] ")
		content.WriteString(ws.context.GitStatus)
		content.WriteString("\n")
	}

	// Available languages
	if len(ws.context.AvailableLanguages) > 0 {
		content.WriteString("\n[yellow]Available Languages:[white]\n")
		for _, lang := range ws.context.AvailableLanguages {
			content.WriteString(fmt.Sprintf("  • %s\n", lang))
		}
	}

	// Recent files
	if len(ws.context.RecentFiles) > 0 {
		content.WriteString("\n[yellow]Recent Files:[white]\n")
		maxFiles := 5
		if len(ws.context.RecentFiles) < maxFiles {
			maxFiles = len(ws.context.RecentFiles)
		}
		for i := 0; i < maxFiles; i++ {
			file := ws.context.RecentFiles[i]
			content.WriteString(fmt.Sprintf("  • %s\n", filepath.Base(file)))
		}
		if len(ws.context.RecentFiles) > maxFiles {
			content.WriteString(fmt.Sprintf("  ... and %d more\n", len(ws.context.RecentFiles)-maxFiles))
		}
	}

	// Dependencies
	if len(ws.context.Dependencies) > 0 {
		content.WriteString("\n[yellow]Dependencies:[white]\n")
		maxDeps := 5
		if len(ws.context.Dependencies) < maxDeps {
			maxDeps = len(ws.context.Dependencies)
		}
		for i := 0; i < maxDeps; i++ {
			dep := ws.context.Dependencies[i]
			content.WriteString(fmt.Sprintf("  • %s %s (%s)\n", dep.Name, dep.Version, dep.Type))
		}
		if len(ws.context.Dependencies) > maxDeps {
			content.WriteString(fmt.Sprintf("  ... and %d more\n", len(ws.context.Dependencies)-maxDeps))
		}
	}

	ws.contextPanel.SetText(content.String())
}

func (ws *WorkspaceStatus) updateDetectedDisplay() {
	if ws.context == nil {
		ws.detectedPanel.SetText("No detection data available")
		return
	}

	var content strings.Builder

	// Header
	colorStart, colorEnd := ws.config.Theme.ColorToTags(ws.config.Theme.Secondary)
	content.WriteString(fmt.Sprintf("%sAuto-Detection Results%s\n\n", colorStart, colorEnd))

	// Project structure summary
	if len(ws.context.ProjectStructure) > 0 {
		content.WriteString("[yellow]Project Structure:[white]\n")
		structureMap := make(map[string]int)
		for _, item := range ws.context.ProjectStructure {
			ext := filepath.Ext(item)
			if ext != "" {
				structureMap[ext]++
			} else if strings.Contains(item, "/") {
				structureMap["directories"]++
			}
		}

		for ext, count := range structureMap {
			if ext == "directories" {
				content.WriteString(fmt.Sprintf("  • Directories: %d\n", count))
			} else {
				content.WriteString(fmt.Sprintf("  • %s files: %d\n", ext, count))
			}
		}
	}

	// Environment variables (selective)
	if len(ws.context.Environment) > 0 {
		content.WriteString("\n[yellow]Relevant Environment:[white]\n")
		relevantVars := []string{"NODE_ENV", "GO_ENV", "RAILS_ENV", "PYTHON_ENV", "ENV"}
		for _, varName := range relevantVars {
			if value, exists := ws.context.Environment[varName]; exists {
				content.WriteString(fmt.Sprintf("  • %s: %s\n", varName, value))
			}
		}
	}

	// Detection confidence (simulated)
	content.WriteString("\n[yellow]Detection Confidence:[white]\n")
	if ws.context.Language != "" {
		content.WriteString("  • Language: [green]High[white]\n")
	}
	if ws.context.Framework != "" {
		content.WriteString("  • Framework: [green]High[white]\n")
	}
	if len(ws.context.Dependencies) > 0 {
		content.WriteString("  • Dependencies: [green]High[white]\n")
	}

	// Last updated
	content.WriteString(fmt.Sprintf("\n[yellow]Last Updated:[white] %s\n", FormatTime(ws.context.LastModified)))

	ws.detectedPanel.SetText(content.String())
}

func (ws *WorkspaceStatus) updateSuggestionsDisplay() {
	ws.suggestionsPanel.Clear()

	if len(ws.suggestions) == 0 {
		ws.suggestionsPanel.AddItem("No suggestions available", "Try refreshing or check your workspace", 0, nil)
		return
	}

	for i, suggestion := range ws.suggestions {
		relevanceColor := ws.getRelevanceColor(suggestion.Relevance)
		colorStart, colorEnd := ws.config.Theme.ColorToTags(relevanceColor)

		mainText := fmt.Sprintf("%s %s%s%s", ws.config.Icons.Template, colorStart, suggestion.Name, colorEnd)
		secondaryText := fmt.Sprintf("%.0f%% match - %s", suggestion.Relevance*100, suggestion.Reason)

		shortcut := rune(0)
		if i < 9 {
			shortcut = rune('1' + i)
		}

		ws.suggestionsPanel.AddItem(mainText, secondaryText, shortcut, nil)
	}
}

func (ws *WorkspaceStatus) getRelevanceColor(relevance float64) tcell.Color {
	if relevance >= 0.8 {
		return ws.config.Theme.Success
	} else if relevance >= 0.6 {
		return ws.config.Theme.Warning
	} else {
		return ws.config.Theme.TextMuted
	}
}

// Event handlers
func (ws *WorkspaceStatus) onSuggestionSelected(index int, mainText, secondaryText string, shortcut rune) {
	if index >= 0 && index < len(ws.suggestions) {
		suggestion := ws.suggestions[index]
		if ws.callbacks.OnQuickAction != nil {
			ws.callbacks.OnQuickAction("select_template", suggestion)
		}
	}
}

func (ws *WorkspaceStatus) onQuickActionSelected(index int, mainText, secondaryText string, shortcut rune) {
	actions := []string{"suggest", "copy_context", "export_context", "refresh", "browse", "create"}
	if index >= 0 && index < len(actions) {
		ws.executeQuickAction(actions[index])
	}
}

// Action execution methods
func (ws *WorkspaceStatus) executeSelectedAction() {
	// Get currently focused panel and execute appropriate action
	if ws.suggestionsPanel.HasFocus() {
		currentIndex := ws.suggestionsPanel.GetCurrentItem()
		if currentIndex >= 0 && currentIndex < len(ws.suggestions) {
			suggestion := ws.suggestions[currentIndex]
			if ws.callbacks.OnQuickAction != nil {
				ws.callbacks.OnQuickAction("select_template", suggestion)
			}
		}
	} else if ws.quickActionsPanel.HasFocus() {
		currentIndex := ws.quickActionsPanel.GetCurrentItem()
		actions := []string{"suggest", "copy_context", "export_context", "refresh", "browse", "create"}
		if currentIndex >= 0 && currentIndex < len(actions) {
			ws.executeQuickAction(actions[currentIndex])
		}
	}
}

func (ws *WorkspaceStatus) executeQuickAction(action string) {
	switch action {
	case "suggest":
		ws.suggestTemplates()
	case "copy_context":
		ws.copyContextToClipboard()
	case "export_context":
		ws.exportContext()
	case "refresh":
		ws.refreshContext()
	case "browse", "create":
		if ws.callbacks.OnQuickAction != nil {
			ws.callbacks.OnQuickAction(action, nil)
		}
	}
}

func (ws *WorkspaceStatus) suggestTemplates() {
	if ws.context == nil {
		ws.updateStatusBar("No workspace context available", "warning")
		return
	}

	ws.updateStatusBar("Getting template suggestions...", "info")
	
	go func() {
		suggestions := ws.manager.SuggestTemplates(ws.context)
		ws.suggestions = suggestions
		ws.updateSuggestionsDisplay()
		ws.updateStatusBar(fmt.Sprintf("Found %d template suggestions", len(suggestions)), "success")

		if ws.callbacks.OnTemplatesSuggest != nil {
			ws.callbacks.OnTemplatesSuggest(suggestions)
		}
	}()
}

func (ws *WorkspaceStatus) copyContextToClipboard() {
	if ws.context == nil {
		ws.updateStatusBar("No context to copy", "warning")
		return
	}

	// Create a text representation of the context
	var contextText strings.Builder
	contextText.WriteString(fmt.Sprintf("Workspace Context\n"))
	contextText.WriteString(fmt.Sprintf("=================\n\n"))
	contextText.WriteString(fmt.Sprintf("Directory: %s\n", ws.context.WorkingDirectory))
	
	if ws.context.Repository != "" {
		contextText.WriteString(fmt.Sprintf("Repository: %s\n", ws.context.Repository))
	}
	if ws.context.Language != "" {
		contextText.WriteString(fmt.Sprintf("Language: %s\n", ws.context.Language))
	}
	if ws.context.Framework != "" {
		contextText.WriteString(fmt.Sprintf("Framework: %s\n", ws.context.Framework))
	}
	if ws.context.GitBranch != "" {
		contextText.WriteString(fmt.Sprintf("Git Branch: %s\n", ws.context.GitBranch))
	}

	if len(ws.context.AvailableLanguages) > 0 {
		contextText.WriteString(fmt.Sprintf("\nAvailable Languages: %s\n", strings.Join(ws.context.AvailableLanguages, ", ")))
	}

	err := ws.manager.CopyToClipboard(contextText.String())
	if err != nil {
		ws.handleError(fmt.Errorf("failed to copy context: %w", err))
		return
	}

	ws.updateStatusBar("Context copied to clipboard", "success")
}

func (ws *WorkspaceStatus) exportContext() {
	if ws.context == nil {
		ws.updateStatusBar("No context to export", "warning")
		return
	}

	// Create filename based on current directory
	filename := fmt.Sprintf("workspace_context_%d.json", time.Now().Unix())

	// Export context as JSON (this would typically show a file dialog)
	contextJSON := ws.contextToJSON()
	err := ws.manager.SaveToFile(contextJSON, filename)
	if err != nil {
		ws.handleError(fmt.Errorf("failed to export context: %w", err))
		return
	}

	ws.updateStatusBar(fmt.Sprintf("Context exported to %s", filename), "success")
}

func (ws *WorkspaceStatus) contextToJSON() string {
	if ws.context == nil {
		return "{}"
	}

	// Simple JSON serialization (in a real implementation, use proper JSON marshaling)
	var json strings.Builder
	json.WriteString("{\n")
	json.WriteString(fmt.Sprintf("  \"working_directory\": \"%s\",\n", ws.context.WorkingDirectory))
	json.WriteString(fmt.Sprintf("  \"repository\": \"%s\",\n", ws.context.Repository))
	json.WriteString(fmt.Sprintf("  \"language\": \"%s\",\n", ws.context.Language))
	json.WriteString(fmt.Sprintf("  \"framework\": \"%s\",\n", ws.context.Framework))
	json.WriteString(fmt.Sprintf("  \"git_branch\": \"%s\",\n", ws.context.GitBranch))
	json.WriteString(fmt.Sprintf("  \"git_status\": \"%s\",\n", ws.context.GitStatus))
	json.WriteString(fmt.Sprintf("  \"available_languages\": %v,\n", ws.context.AvailableLanguages))
	json.WriteString(fmt.Sprintf("  \"recent_files\": %v,\n", ws.context.RecentFiles))
	json.WriteString(fmt.Sprintf("  \"last_modified\": \"%s\"\n", ws.context.LastModified.Format(time.RFC3339)))
	json.WriteString("}")
	return json.String()
}

// Navigation methods
func (ws *WorkspaceStatus) nextPanel() {
	// Cycle through focusable panels
	if ws.contextPanel.HasFocus() {
		ws.setFocus(ws.suggestionsPanel)
	} else if ws.suggestionsPanel.HasFocus() {
		ws.setFocus(ws.quickActionsPanel)
	} else {
		ws.setFocus(ws.contextPanel)
	}
}

func (ws *WorkspaceStatus) prevPanel() {
	// Cycle through focusable panels in reverse
	if ws.quickActionsPanel.HasFocus() {
		ws.setFocus(ws.suggestionsPanel)
	} else if ws.suggestionsPanel.HasFocus() {
		ws.setFocus(ws.contextPanel)
	} else {
		ws.setFocus(ws.quickActionsPanel)
	}
}

func (ws *WorkspaceStatus) setFocus(primitive tview.Primitive) {
	app := ws.GetApplication()
	if app != nil {
		app.SetFocus(primitive)
	}
}

// UI update methods
func (ws *WorkspaceStatus) updateUI() {
	ws.updateContextDisplay()
	ws.updateDetectedDisplay()
	ws.updateSuggestionsDisplay()
	ws.updateStatusBar("", "")
}

func (ws *WorkspaceStatus) updateStatusBar(message, msgType string) {
	var status strings.Builder

	if message != "" {
		if msgType != "" {
			statusMsg := NewStatusMessage(message, msgType, 3*time.Second)
			formattedMsg := statusMsg.FormatForDisplay(ws.config.Theme)
			status.WriteString(formattedMsg)
		} else {
			status.WriteString(message)
		}
	} else {
		// Default status information
		if ws.context != nil {
			status.WriteString(fmt.Sprintf("Context: %s", ws.context.Language))
			if ws.context.Framework != "" {
				status.WriteString(fmt.Sprintf(" (%s)", ws.context.Framework))
			}
		}

		if len(ws.suggestions) > 0 {
			if status.Len() > 0 {
				status.WriteString(" | ")
			}
			status.WriteString(fmt.Sprintf("Suggestions: %d", len(ws.suggestions)))
		}

		if !ws.lastRefresh.IsZero() {
			if status.Len() > 0 {
				status.WriteString(" | ")
			}
			status.WriteString(fmt.Sprintf("Last refresh: %s", FormatTime(ws.lastRefresh)))
		}

		if ws.isRefreshing {
			if status.Len() > 0 {
				status.WriteString(" | ")
			}
			status.WriteString("[yellow]Refreshing...[white]")
		}

		if ws.config.EnableKeyHelp {
			if status.Len() > 0 {
				status.WriteString(" | ")
			}
			status.WriteString(ws.keyBindings.GetHelpText(ws.config.Theme))
		}
	}

	ws.statusBar.SetText(status.String())
}

// Utility methods
func (ws *WorkspaceStatus) handleError(err error) {
	ws.logger.Error("workspace status error", zap.Error(err))
	ws.updateStatusBar(err.Error(), "error")
	
	if ws.callbacks.OnError != nil {
		ws.callbacks.OnError(err)
	}
}

// GetApplication returns the tview application
func (ws *WorkspaceStatus) GetApplication() *tview.Application {
	// This would need to be set externally or passed in during initialization
	return nil
}

// Public interface methods

// GetContext returns the current workspace context
func (ws *WorkspaceStatus) GetContext() *prompt.WorkspaceContext {
	return ws.context
}

// GetSuggestions returns the current template suggestions
func (ws *WorkspaceStatus) GetSuggestions() []prompt.TemplateSuggestion {
	return ws.suggestions
}

// SetContext sets the workspace context
func (ws *WorkspaceStatus) SetContext(context *prompt.WorkspaceContext) {
	ws.context = context
	ws.updateContextDisplay()
	ws.updateDetectedDisplay()
}

// SetSuggestions sets the template suggestions
func (ws *WorkspaceStatus) SetSuggestions(suggestions []prompt.TemplateSuggestion) {
	ws.suggestions = suggestions
	ws.updateSuggestionsDisplay()
}

// IsRefreshing returns true if context is currently being refreshed
func (ws *WorkspaceStatus) IsRefreshing() bool {
	return ws.isRefreshing
}

// Focus sets focus to the workspace status component
func (ws *WorkspaceStatus) Focus(delegate func(p tview.Primitive)) {
	delegate(ws.contextPanel)
}

// HasFocus returns true if the workspace status has focus
func (ws *WorkspaceStatus) HasFocus() bool {
	return ws.contextPanel.HasFocus() || ws.suggestionsPanel.HasFocus() || ws.quickActionsPanel.HasFocus()
}

// Close cleans up the workspace status component
func (ws *WorkspaceStatus) Close() {
	if ws.autoRefreshTimer != nil {
		ws.autoRefreshTimer.Stop()
	}
	ws.cancel()
}