package components

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/prompt"
)

// PromptBrowserMode represents different modes of operation
type PromptBrowserMode int

const (
	BrowserModeNormal PromptBrowserMode = iota
	BrowserModeSearch
	BrowserModeFilter
)

// PromptBrowserCallbacks defines callback functions for browser events
type PromptBrowserCallbacks struct {
	OnTemplateSelect   func(template prompt.Template)
	OnTemplateGenerate func(template prompt.Template)
	OnTemplateEdit     func(template prompt.Template)
	OnTemplateDelete   func(template prompt.Template)
	OnError            func(err error)
}

// PromptBrowser provides a TUI for browsing and managing prompt templates
type PromptBrowser struct {
	*tview.Flex

	// Configuration
	config    *PromptUIConfig
	callbacks *PromptBrowserCallbacks
	manager   prompt.Manager
	logger    *zap.Logger

	// UI components
	categoryList   *tview.List
	templateList   *tview.List
	previewPane    *tview.TextView
	searchInput    *tview.InputField
	statusBar      *tview.TextView
	helpText       *tview.TextView

	// State
	mode           PromptBrowserMode
	categories     []string
	templates      []prompt.Template
	filteredItems  []*TemplateListItem
	selectedCategory string
	selectedTemplate *prompt.Template
	searchQuery      string
	lastError        error

	// Key bindings
	keyBindings *KeyBindings

	// Context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPromptBrowser creates a new prompt browser component
func NewPromptBrowser(manager prompt.Manager, config *PromptUIConfig, callbacks *PromptBrowserCallbacks, logger *zap.Logger) *PromptBrowser {
	if config == nil {
		config = DefaultPromptUIConfig()
	}

	if callbacks == nil {
		callbacks = &PromptBrowserCallbacks{}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pb := &PromptBrowser{
		Flex:      tview.NewFlex(),
		config:    config,
		callbacks: callbacks,
		manager:   manager,
		logger:    logger,
		mode:      BrowserModeNormal,
		ctx:       ctx,
		cancel:    cancel,
	}

	pb.initKeyBindings()
	pb.initUI()
	pb.refreshData()

	return pb
}

// initKeyBindings sets up keyboard shortcuts
func (pb *PromptBrowser) initKeyBindings() {
	pb.keyBindings = NewKeyBindings()

	// Navigation
	pb.keyBindings.AddRune('j', "Down", func() { pb.navigateDown() })
	pb.keyBindings.AddRune('k', "Up", func() { pb.navigateUp() })
	pb.keyBindings.AddKey(tcell.KeyEnter, "Select/Generate", func() { pb.selectOrGenerate() })

	// Actions
	pb.keyBindings.AddRune('g', "Generate", func() { pb.generateTemplate() })
	pb.keyBindings.AddRune('e', "Edit", func() { pb.editTemplate() })
	pb.keyBindings.AddRune('d', "Delete", func() { pb.deleteTemplate() })
	pb.keyBindings.AddRune('c', "Copy", func() { pb.copyTemplate() })

	// Search and filter
	pb.keyBindings.AddRune('/', "Search", func() { pb.activateSearch() })
	pb.keyBindings.AddRune('f', "Filter", func() { pb.activateFilter() })
	pb.keyBindings.AddKey(tcell.KeyEsc, "Clear/Exit", func() { pb.clearSearchOrExit() })

	// Refresh and navigation
	pb.keyBindings.AddRune('r', "Refresh", func() { pb.refreshData() })
	pb.keyBindings.AddKey(tcell.KeyF5, "Refresh", func() { pb.refreshData() })
	pb.keyBindings.AddKey(tcell.KeyTab, "Next Panel", func() { pb.nextPanel() })
	pb.keyBindings.AddKey(tcell.KeyBacktab, "Prev Panel", func() { pb.prevPanel() })

	// Category navigation
	pb.keyBindings.AddRune('1', "All Categories", func() { pb.selectCategory("") })
	pb.keyBindings.AddRune('2', "General", func() { pb.selectCategory("general") })
	pb.keyBindings.AddRune('3', "Languages", func() { pb.selectCategory("languages") })
	pb.keyBindings.AddRune('4', "Workflows", func() { pb.selectCategory("workflows") })
	pb.keyBindings.AddRune('5', "Workspace", func() { pb.selectCategory("workspace") })
	pb.keyBindings.AddRune('6', "Custom", func() { pb.selectCategory("custom") })
}

// initUI initializes the user interface components
func (pb *PromptBrowser) initUI() {
	// Create category list
	pb.categoryList = CreateStyledList("Categories", pb.config.Theme, pb.config.Icons)
	pb.categoryList.SetChangedFunc(pb.onCategoryChanged)
	pb.categoryList.SetSelectedFunc(pb.onCategorySelected)

	// Create template list
	pb.templateList = CreateStyledList("Templates", pb.config.Theme, pb.config.Icons)
	pb.templateList.SetChangedFunc(pb.onTemplateChanged)
	pb.templateList.SetSelectedFunc(pb.onTemplateSelected)

	// Create preview pane
	pb.previewPane = CreateStyledTextView("Preview", pb.config.Theme, pb.config.Icons)
	pb.previewPane.SetScrollable(true)

	// Create search input (initially hidden)
	pb.searchInput = CreateStyledInputField(fmt.Sprintf("%s Search: ", pb.config.Icons.Search), pb.config.Theme)
	pb.searchInput.SetBorder(true)
	pb.searchInput.SetTitle(fmt.Sprintf(" %s Search Templates ", pb.config.Icons.Search))
	pb.searchInput.SetDoneFunc(pb.onSearchDone)
	pb.searchInput.SetChangedFunc(pb.onSearchChanged)

	// Create status bar
	pb.statusBar = tview.NewTextView()
	pb.statusBar.SetDynamicColors(true)
	pb.statusBar.SetTextAlign(tview.AlignLeft)

	// Create help text
	pb.helpText = tview.NewTextView()
	pb.helpText.SetDynamicColors(true)
	pb.helpText.SetTextAlign(tview.AlignCenter)

	// Setup layout
	pb.setupLayout()

	// Initialize content
	pb.loadCategories()
	pb.updateHelpText()
	pb.updateStatusBar()
}

// setupLayout arranges the UI components
func (pb *PromptBrowser) setupLayout() {
	// Left panel: categories
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(pb.categoryList, 0, 1, true)

	// Middle panel: templates
	middlePanel := tview.NewFlex().SetDirection(tview.FlexRow)
	middlePanel.AddItem(pb.templateList, 0, 1, false)

	// Right panel: preview
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPanel.AddItem(pb.previewPane, 0, 1, false)

	// Main content area
	contentArea := tview.NewFlex().SetDirection(tview.FlexColumn)
	contentArea.AddItem(leftPanel, 0, 1, true)
	contentArea.AddItem(middlePanel, 0, 2, false)
	contentArea.AddItem(rightPanel, 0, 2, false)

	// Bottom panel: status and help
	bottomPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	bottomPanel.AddItem(pb.statusBar, 1, 0, false)
	bottomPanel.AddItem(pb.helpText, 1, 0, false)

	// Main layout
	pb.SetDirection(tview.FlexRow)
	pb.AddItem(contentArea, 0, 1, true)
	pb.AddItem(bottomPanel, 2, 0, false)

	// Set up input capture
	pb.SetInputCapture(pb.handleKeyPress)
}

// handleKeyPress processes keyboard input
func (pb *PromptBrowser) handleKeyPress(event *tcell.EventKey) *tcell.EventKey {
	// Handle search mode separately
	if pb.mode == BrowserModeSearch {
		return pb.handleSearchKeyPress(event)
	}

	// Try key bindings first
	if pb.keyBindings.Handle(event) {
		return nil
	}

	return event
}

// handleSearchKeyPress handles key presses in search mode
func (pb *PromptBrowser) handleSearchKeyPress(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		pb.deactivateSearch()
		return nil
	}
	return event
}

// loadCategories populates the category list
func (pb *PromptBrowser) loadCategories() {
	pb.categoryList.Clear()

	// Add "All" category
	allText := fmt.Sprintf("%s All Templates", pb.config.Icons.Template)
	pb.categoryList.AddItem(allText, "Show all templates", '1', nil)

	// Load categories from templates
	categoryMap := make(map[string]int)
	for _, template := range pb.templates {
		if template.Category != "" {
			categoryMap[template.Category]++
		}
	}

	// Sort categories
	pb.categories = make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		pb.categories = append(pb.categories, category)
	}
	sort.Strings(pb.categories)

	// Add category items
	for i, category := range pb.categories {
		count := categoryMap[category]
		icon := pb.config.Icons.Category
		color := pb.config.Theme.GetCategoryColor(category)
		
		colorStart, colorEnd := pb.config.Theme.ColorToTags(color)
		mainText := fmt.Sprintf("%s %s%s%s", icon, colorStart, strings.Title(category), colorEnd)
		secondaryText := fmt.Sprintf("%d templates", count)
		
		shortcut := rune('2' + i)
		if i < 5 { // Only assign shortcuts to first 5 categories
			pb.categoryList.AddItem(mainText, secondaryText, shortcut, nil)
		} else {
			pb.categoryList.AddItem(mainText, secondaryText, 0, nil)
		}
	}
}

// onCategoryChanged handles category selection changes
func (pb *PromptBrowser) onCategoryChanged(index int, mainText, secondaryText string, shortcut rune) {
	if index == 0 {
		pb.selectedCategory = ""
	} else if index-1 < len(pb.categories) {
		pb.selectedCategory = pb.categories[index-1]
	}

	pb.filterTemplates()
	pb.updateStatusBar()
}

// onCategorySelected handles category selection
func (pb *PromptBrowser) onCategorySelected(index int, mainText, secondaryText string, shortcut rune) {
	pb.onCategoryChanged(index, mainText, secondaryText, shortcut)
}

// onTemplateChanged handles template selection changes
func (pb *PromptBrowser) onTemplateChanged(index int, mainText, secondaryText string, shortcut rune) {
	if index >= 0 && index < len(pb.filteredItems) {
		item := pb.filteredItems[index]
		pb.selectedTemplate = &item.Template
		pb.updatePreview()
		
		if pb.callbacks.OnTemplateSelect != nil {
			pb.callbacks.OnTemplateSelect(item.Template)
		}
	}
}

// onTemplateSelected handles template selection
func (pb *PromptBrowser) onTemplateSelected(index int, mainText, secondaryText string, shortcut rune) {
	pb.onTemplateChanged(index, mainText, secondaryText, shortcut)
	pb.generateTemplate()
}

// onSearchChanged handles search input changes
func (pb *PromptBrowser) onSearchChanged(text string) {
	pb.searchQuery = text
	if len(text) >= pb.config.SearchMinLength {
		pb.filterTemplates()
	} else if text == "" {
		pb.filterTemplates()
	}
}

// onSearchDone handles search completion
func (pb *PromptBrowser) onSearchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		pb.deactivateSearch()
	}
}

// refreshData refreshes all data from the manager
func (pb *PromptBrowser) refreshData() {
	// Load templates
	templates, err := pb.manager.ListTemplates("")
	if err != nil {
		pb.handleError(fmt.Errorf("failed to load templates: %w", err))
		return
	}

	pb.templates = templates
	pb.loadCategories()
	pb.filterTemplates()
	pb.updateStatusBar()
}

// filterTemplates filters templates based on category and search query
func (pb *PromptBrowser) filterTemplates() {
	pb.templateList.Clear()
	pb.filteredItems = make([]*TemplateListItem, 0)

	for _, template := range pb.templates {
		// Apply category filter
		if pb.selectedCategory != "" && template.Category != pb.selectedCategory {
			continue
		}

		item := NewTemplateListItem(template)

		// Apply search filter
		if pb.searchQuery != "" && !item.FilterMatch(pb.searchQuery) {
			continue
		}

		pb.filteredItems = append(pb.filteredItems, item)
	}

	// Sort filtered items
	sort.Slice(pb.filteredItems, func(i, j int) bool {
		// Sort by category first, then by name
		if pb.filteredItems[i].Category != pb.filteredItems[j].Category {
			return pb.filteredItems[i].Category < pb.filteredItems[j].Category
		}
		return pb.filteredItems[i].Template.Name < pb.filteredItems[j].Template.Name
	})

	// Add items to list
	for _, item := range pb.filteredItems {
		mainText, secondaryText := item.FormatForList(pb.config)
		pb.templateList.AddItem(mainText, secondaryText, 0, nil)
	}

	// Update preview if we have items
	if len(pb.filteredItems) > 0 {
		pb.selectedTemplate = &pb.filteredItems[0].Template
		pb.updatePreview()
	} else {
		pb.selectedTemplate = nil
		pb.previewPane.Clear()
		pb.previewPane.SetText("No templates found")
	}
}

// updatePreview updates the preview pane with the selected template
func (pb *PromptBrowser) updatePreview() {
	if pb.selectedTemplate == nil {
		pb.previewPane.Clear()
		pb.previewPane.SetText("No template selected")
		return
	}

	template := *pb.selectedTemplate

	// Build preview content
	var preview strings.Builder

	// Header
	colorStart, colorEnd := pb.config.Theme.ColorToTags(pb.config.Theme.Primary)
	preview.WriteString(fmt.Sprintf("%s%s%s\n\n", colorStart, template.Name, colorEnd))

	// Description
	if template.Description != "" {
		preview.WriteString(fmt.Sprintf("[yellow]Description:[white] %s\n\n", template.Description))
	}

	// Metadata
	preview.WriteString("[yellow]Details:[white]\n")
	preview.WriteString(fmt.Sprintf("  Category: %s\n", template.Category))
	if template.Language != "" {
		preview.WriteString(fmt.Sprintf("  Language: %s\n", template.Language))
	}
	if template.Framework != "" {
		preview.WriteString(fmt.Sprintf("  Framework: %s\n", template.Framework))
	}
	if template.Author != "" {
		preview.WriteString(fmt.Sprintf("  Author: %s\n", template.Author))
	}
	if template.Version != "" {
		preview.WriteString(fmt.Sprintf("  Version: %s\n", template.Version))
	}
	preview.WriteString(fmt.Sprintf("  Created: %s\n", FormatTime(template.CreatedAt)))
	preview.WriteString(fmt.Sprintf("  Updated: %s\n", FormatTime(template.UpdatedAt)))

	// Parameters
	if len(template.Parameters) > 0 {
		preview.WriteString(fmt.Sprintf("\n[yellow]Parameters (%d):[white]\n", len(template.Parameters)))
		for _, param := range template.Parameters {
			required := ""
			if param.Required {
				required = " (required)"
			}
			preview.WriteString(fmt.Sprintf("  • %s%s: %s\n", param.Name, required, param.Description))
		}
	}

	// Tags
	if len(template.Tags) > 0 {
		preview.WriteString(fmt.Sprintf("\n[yellow]Tags:[white] %s\n", strings.Join(template.Tags, ", ")))
	}

	// Examples
	if len(template.Examples) > 0 {
		preview.WriteString(fmt.Sprintf("\n[yellow]Examples:[white]\n"))
		for i, example := range template.Examples {
			if i >= 3 { // Limit to 3 examples
				preview.WriteString("  ...\n")
				break
			}
			preview.WriteString(fmt.Sprintf("  %d. %s\n", i+1, example))
		}
	}

	// Content preview (first few lines)
	if template.Content != "" {
		preview.WriteString("\n[yellow]Content Preview:[white]\n")
		lines := strings.Split(template.Content, "\n")
		maxLines := pb.config.PreviewMaxLines
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			lines = append(lines, "...")
		}
		for _, line := range lines {
			preview.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	pb.previewPane.SetText(preview.String())
	pb.previewPane.ScrollToBeginning()
}

// updateStatusBar updates the status bar
func (pb *PromptBrowser) updateStatusBar() {
	totalTemplates := len(pb.templates)
	filteredTemplates := len(pb.filteredItems)

	var status strings.Builder
	status.WriteString(fmt.Sprintf("Templates: %d", totalTemplates))

	if pb.selectedCategory != "" {
		status.WriteString(fmt.Sprintf(" | Category: %s", pb.selectedCategory))
	}

	if pb.searchQuery != "" {
		status.WriteString(fmt.Sprintf(" | Search: '%s'", pb.searchQuery))
	}

	if filteredTemplates != totalTemplates {
		status.WriteString(fmt.Sprintf(" | Showing: %d", filteredTemplates))
	}

	if pb.selectedTemplate != nil {
		status.WriteString(fmt.Sprintf(" | Selected: %s", pb.selectedTemplate.Name))
	}

	pb.statusBar.SetText(status.String())
}

// updateHelpText updates the help text
func (pb *PromptBrowser) updateHelpText() {
	if pb.config.EnableKeyHelp {
		helpText := pb.keyBindings.GetHelpText(pb.config.Theme)
		pb.helpText.SetText(helpText)
	}
}

// Navigation methods
func (pb *PromptBrowser) navigateDown() {
	// Implementation depends on which panel has focus
	// This is a simplified version
	if pb.templateList.HasFocus() {
		currentIndex := pb.templateList.GetCurrentItem()
		if currentIndex < pb.templateList.GetItemCount()-1 {
			pb.templateList.SetCurrentItem(currentIndex + 1)
		}
	}
}

func (pb *PromptBrowser) navigateUp() {
	// Implementation depends on which panel has focus
	// This is a simplified version
	if pb.templateList.HasFocus() {
		currentIndex := pb.templateList.GetCurrentItem()
		if currentIndex > 0 {
			pb.templateList.SetCurrentItem(currentIndex - 1)
		}
	}
}

func (pb *PromptBrowser) nextPanel() {
	app := pb.GetApplication()
	if app != nil {
		app.SetFocus(pb.templateList)
	}
}

func (pb *PromptBrowser) prevPanel() {
	app := pb.GetApplication()
	if app != nil {
		app.SetFocus(pb.categoryList)
	}
}

// Action methods
func (pb *PromptBrowser) selectOrGenerate() {
	if pb.selectedTemplate != nil {
		pb.generateTemplate()
	}
}

func (pb *PromptBrowser) generateTemplate() {
	if pb.selectedTemplate == nil {
		pb.showStatus("No template selected", "warning")
		return
	}

	if pb.callbacks.OnTemplateGenerate != nil {
		pb.callbacks.OnTemplateGenerate(*pb.selectedTemplate)
	}
}

func (pb *PromptBrowser) editTemplate() {
	if pb.selectedTemplate == nil {
		pb.showStatus("No template selected", "warning")
		return
	}

	if pb.callbacks.OnTemplateEdit != nil {
		pb.callbacks.OnTemplateEdit(*pb.selectedTemplate)
	}
}

func (pb *PromptBrowser) deleteTemplate() {
	if pb.selectedTemplate == nil {
		pb.showStatus("No template selected", "warning")
		return
	}

	if pb.callbacks.OnTemplateDelete != nil {
		pb.callbacks.OnTemplateDelete(*pb.selectedTemplate)
	}
}

func (pb *PromptBrowser) copyTemplate() {
	if pb.selectedTemplate == nil {
		pb.showStatus("No template selected", "warning")
		return
	}

	// Copy template content to clipboard
	err := pb.manager.CopyToClipboard(pb.selectedTemplate.Content)
	if err != nil {
		pb.handleError(fmt.Errorf("failed to copy template: %w", err))
		return
	}

	pb.showStatus(fmt.Sprintf("Copied template '%s'", pb.selectedTemplate.Name), "success")
}

// Search methods
func (pb *PromptBrowser) activateSearch() {
	pb.mode = BrowserModeSearch
	
	// Replace status bar with search input
	pb.RemoveItem(pb.statusBar)
	pb.InsertItem(pb.GetItemCount()-1, pb.searchInput, 3, 0, false)
	
	// Focus on search input
	app := pb.GetApplication()
	if app != nil {
		app.SetFocus(pb.searchInput)
	}
}

func (pb *PromptBrowser) deactivateSearch() {
	pb.mode = BrowserModeNormal
	
	// Restore original layout
	pb.RemoveItem(pb.searchInput)
	pb.InsertItem(pb.GetItemCount()-1, pb.statusBar, 1, 0, false)
	
	// Focus back on template list
	app := pb.GetApplication()
	if app != nil {
		app.SetFocus(pb.templateList)
	}
}

func (pb *PromptBrowser) activateFilter() {
	// Similar to search but for filtering
	pb.activateSearch()
}

func (pb *PromptBrowser) clearSearchOrExit() {
	if pb.mode == BrowserModeSearch {
		if pb.searchQuery != "" {
			pb.searchInput.SetText("")
			pb.searchQuery = ""
			pb.filterTemplates()
		} else {
			pb.deactivateSearch()
		}
	}
}

func (pb *PromptBrowser) selectCategory(category string) {
	pb.selectedCategory = category
	
	// Find and select the category in the list
	for i := 0; i < pb.categoryList.GetItemCount(); i++ {
		if i == 0 && category == "" {
			pb.categoryList.SetCurrentItem(i)
			break
		} else if i > 0 && i-1 < len(pb.categories) && pb.categories[i-1] == category {
			pb.categoryList.SetCurrentItem(i)
			break
		}
	}
	
	pb.filterTemplates()
}

// Utility methods
func (pb *PromptBrowser) showStatus(message, msgType string) {
	statusMsg := NewStatusMessage(message, msgType, 3*time.Second)
	formattedMsg := statusMsg.FormatForDisplay(pb.config.Theme)
	
	// Temporarily update status bar
	originalText := pb.statusBar.GetText(false)
	pb.statusBar.SetText(formattedMsg)
	
	// Restore after duration
	go func() {
		time.Sleep(statusMsg.Duration)
		pb.statusBar.SetText(originalText)
	}()
}

func (pb *PromptBrowser) handleError(err error) {
	pb.lastError = err
	pb.logger.Error("prompt browser error", zap.Error(err))
	pb.showStatus(err.Error(), "error")
	
	if pb.callbacks.OnError != nil {
		pb.callbacks.OnError(err)
	}
}

// GetApplication returns the tview application
func (pb *PromptBrowser) GetApplication() *tview.Application {
	// This would need to be set externally or passed in during initialization
	return nil
}

// Focus sets focus to the prompt browser
func (pb *PromptBrowser) Focus(delegate func(p tview.Primitive)) {
	delegate(pb.categoryList)
}

// HasFocus returns true if the prompt browser has focus
func (pb *PromptBrowser) HasFocus() bool {
	return pb.categoryList.HasFocus() || pb.templateList.HasFocus() || pb.previewPane.HasFocus()
}

// GetSelectedTemplate returns the currently selected template
func (pb *PromptBrowser) GetSelectedTemplate() *prompt.Template {
	return pb.selectedTemplate
}

// SetSelectedTemplate sets the selected template
func (pb *PromptBrowser) SetSelectedTemplate(template *prompt.Template) {
	pb.selectedTemplate = template
	pb.updatePreview()
}

// GetFilteredCount returns the number of filtered templates
func (pb *PromptBrowser) GetFilteredCount() int {
	return len(pb.filteredItems)
}

// GetTotalCount returns the total number of templates
func (pb *PromptBrowser) GetTotalCount() int {
	return len(pb.templates)
}

// Close cleans up the prompt browser
func (pb *PromptBrowser) Close() {
	pb.cancel()
}