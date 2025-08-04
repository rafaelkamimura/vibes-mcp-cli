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

// HistoryFilterType represents different ways to filter history
type HistoryFilterType int

const (
	FilterAll HistoryFilterType = iota
	FilterByTemplate
	FilterByLanguage
	FilterByRepository
	FilterByStatus
	FilterByDate
)

// HistoryViewMode represents different view modes
type HistoryViewMode int

const (
	ViewModeList HistoryViewMode = iota
	ViewModeTable
	ViewModeTimeline
)

// HistoryFilter represents a filter configuration
type HistoryFilter struct {
	Type  HistoryFilterType
	Value string
	Label string
}

// PromptHistoryCallbacks defines callback functions for history events
type PromptHistoryCallbacks struct {
	OnEntrySelect    func(entry prompt.HistoryEntry)
	OnEntryReuse     func(entry prompt.HistoryEntry)
	OnEntryDelete    func(entry prompt.HistoryEntry)
	OnStatsRequest   func()
	OnError          func(err error)
}

// PromptHistory provides a TUI for viewing and managing prompt generation history
type PromptHistory struct {
	*tview.Flex

	// Configuration
	config    *PromptUIConfig
	callbacks *PromptHistoryCallbacks
	manager   prompt.Manager
	logger    *zap.Logger

	// UI components
	filterBar     *tview.TextView
	historyList   *tview.List
	detailsPane   *tview.TextView
	statsPane     *tview.TextView
	searchInput   *tview.InputField
	statusBar     *tview.TextView

	// State
	viewMode        HistoryViewMode
	entries         []prompt.HistoryEntry
	filteredEntries []prompt.HistoryEntry
	selectedEntry   *prompt.HistoryEntry
	activeFilters   []HistoryFilter
	searchQuery     string
	stats           *prompt.HistoryStats
	isSearchActive  bool
	keyBindings     *KeyBindings

	// Context for operations
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPromptHistory creates a new prompt history component
func NewPromptHistory(manager prompt.Manager, config *PromptUIConfig, callbacks *PromptHistoryCallbacks, logger *zap.Logger) *PromptHistory {
	if config == nil {
		config = DefaultPromptUIConfig()
	}

	if callbacks == nil {
		callbacks = &PromptHistoryCallbacks{}
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	ph := &PromptHistory{
		Flex:        tview.NewFlex(),
		config:      config,
		callbacks:   callbacks,
		manager:     manager,
		logger:      logger,
		viewMode:    ViewModeList,
		ctx:         ctx,
		cancel:      cancel,
	}

	ph.initKeyBindings()
	ph.initUI()
	ph.refreshHistory()

	return ph
}

// initKeyBindings sets up keyboard shortcuts
func (ph *PromptHistory) initKeyBindings() {
	ph.keyBindings = NewKeyBindings()

	// Navigation
	ph.keyBindings.AddKey(tcell.KeyEnter, "Select/View", func() { ph.selectEntry() })
	ph.keyBindings.AddKey(tcell.KeyTab, "Next Panel", func() { ph.nextPanel() })
	ph.keyBindings.AddKey(tcell.KeyBacktab, "Prev Panel", func() { ph.prevPanel() })

	// Actions
	ph.keyBindings.AddRune('r', "Reuse Entry", func() { ph.reuseEntry() })
	ph.keyBindings.AddRune('d', "Delete Entry", func() { ph.deleteEntry() })
	ph.keyBindings.AddRune('c', "Copy Content", func() { ph.copyEntryContent() })
	ph.keyBindings.AddRune('e', "Export Entry", func() { ph.exportEntry() })

	// View modes
	ph.keyBindings.AddRune('1', "List View", func() { ph.setViewMode(ViewModeList) })
	ph.keyBindings.AddRune('2', "Table View", func() { ph.setViewMode(ViewModeTable) })
	ph.keyBindings.AddRune('3', "Timeline View", func() { ph.setViewMode(ViewModeTimeline) })

	// Filtering and search
	ph.keyBindings.AddRune('/', "Search", func() { ph.activateSearch() })
	ph.keyBindings.AddRune('f', "Filter", func() { ph.showFilterMenu() })
	ph.keyBindings.AddRune('x', "Clear Filters", func() { ph.clearFilters() })
	ph.keyBindings.AddKey(tcell.KeyEsc, "Clear/Exit", func() { ph.clearSearchOrExit() })

	// Refresh and stats
	ph.keyBindings.AddKey(tcell.KeyF5, "Refresh", func() { ph.refreshHistory() })
	ph.keyBindings.AddRune('s', "Show Stats", func() { ph.showStats() })

	// Bulk operations
	ph.keyBindings.AddKey(tcell.KeyCtrlA, "Select All", func() { ph.selectAll() })
	ph.keyBindings.AddKey(tcell.KeyCtrlD, "Delete Selected", func() { ph.deleteSelected() })
}

// initUI initializes the user interface components
func (ph *PromptHistory) initUI() {
	ph.SetDirection(tview.FlexRow)

	// Create filter bar
	ph.filterBar = tview.NewTextView()
	ph.filterBar.SetDynamicColors(true)
	ph.filterBar.SetTextAlign(tview.AlignLeft)
	ph.filterBar.SetBorder(true)
	ph.filterBar.SetTitle(" Active Filters ")

	// Create main content area
	contentArea := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Left panel: history list
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	ph.historyList = CreateStyledList("Generation History", ph.config.Theme, ph.config.Icons)
	ph.historyList.SetChangedFunc(ph.onEntryChanged)
	ph.historyList.SetSelectedFunc(ph.onEntrySelected)
	leftPanel.AddItem(ph.historyList, 0, 1, true)

	// Right panel: details and stats
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	ph.detailsPane = CreateStyledTextView("Entry Details", ph.config.Theme, ph.config.Icons)
	ph.detailsPane.SetScrollable(true)
	rightPanel.AddItem(ph.detailsPane, 0, 2, false)

	ph.statsPane = CreateStyledTextView("Statistics", ph.config.Theme, ph.config.Icons)
	ph.statsPane.SetScrollable(true)
	rightPanel.AddItem(ph.statsPane, 0, 1, false)

	// Add panels to content area
	contentArea.AddItem(leftPanel, 0, 1, true)
	contentArea.AddItem(rightPanel, 0, 1, false)

	// Create search input (initially hidden)
	ph.searchInput = CreateStyledInputField(fmt.Sprintf("%s Search: ", ph.config.Icons.Search), ph.config.Theme)
	ph.searchInput.SetBorder(true)
	ph.searchInput.SetTitle(fmt.Sprintf(" %s Search History ", ph.config.Icons.Search))
	ph.searchInput.SetDoneFunc(ph.onSearchDone)
	ph.searchInput.SetChangedFunc(ph.onSearchChanged)

	// Create status bar
	ph.statusBar = tview.NewTextView()
	ph.statusBar.SetDynamicColors(true)
	ph.statusBar.SetTextAlign(tview.AlignLeft)

	// Layout
	ph.AddItem(ph.filterBar, 3, 0, false)
	ph.AddItem(contentArea, 0, 1, true)
	ph.AddItem(ph.statusBar, 1, 0, false)

	// Set up input capture
	ph.SetInputCapture(ph.handleKeyPress)

	// Initial update
	ph.updateUI()
}

// handleKeyPress processes keyboard input
func (ph *PromptHistory) handleKeyPress(event *tcell.EventKey) *tcell.EventKey {
	// Handle search mode separately
	if ph.isSearchActive {
		return ph.handleSearchKeyPress(event)
	}

	// Try key bindings first
	if ph.keyBindings.Handle(event) {
		return nil
	}

	return event
}

// handleSearchKeyPress handles key presses in search mode
func (ph *PromptHistory) handleSearchKeyPress(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		ph.deactivateSearch()
		return nil
	}
	return event
}

// Data loading and refresh methods
func (ph *PromptHistory) refreshHistory() {
	ph.showStatus("Loading history...", "info")

	go func() {
		// Load history entries
		entries, err := ph.manager.GetHistory(ph.config.MaxHistoryItems, "")
		if err != nil {
			ph.handleError(fmt.Errorf("failed to load history: %w", err))
			return
		}

		ph.entries = entries
		ph.applyFilters()

		// Load statistics
		stats, err := ph.manager.GetHistory(0, "") // Get all for stats
		if err == nil {
			ph.calculateStats(stats)
		}

		ph.updateUI()
		ph.showStatus(fmt.Sprintf("Loaded %d history entries", len(entries)), "success")
	}()
}

func (ph *PromptHistory) calculateStats(allEntries []prompt.HistoryEntry) {
	if len(allEntries) == 0 {
		return
	}

	stats := &prompt.HistoryStats{
		TotalGenerations: len(allEntries),
	}

	// Calculate success rate
	successCount := 0
	totalWordCount := 0
	templateCounts := make(map[string]int)
	languageCounts := make(map[string]int)
	repositoryCounts := make(map[string]int)

	for _, entry := range allEntries {
		if entry.Success {
			successCount++
		}
		totalWordCount += entry.WordCount

		// Track template usage
		templateCounts[entry.Template]++

		// Track language usage
		if entry.Language != "" {
			languageCounts[entry.Language]++
		}

		// Track repository usage
		if entry.Repository != "" {
			repositoryCounts[entry.Repository]++
		}
	}

	stats.SuccessRate = float64(successCount) / float64(len(allEntries))
	stats.AverageWordCount = totalWordCount / len(allEntries)

	// Convert to sorted slices
	stats.TopTemplates = ph.mapToTemplateUsage(templateCounts)
	stats.TopLanguages = ph.mapToLanguageUsage(languageCounts)
	stats.TopRepositories = ph.mapToRepositoryUsage(repositoryCounts)

	ph.stats = stats
}

func (ph *PromptHistory) mapToTemplateUsage(counts map[string]int) []prompt.TemplateUsage {
	usage := make([]prompt.TemplateUsage, 0, len(counts))
	for name, count := range counts {
		usage = append(usage, prompt.TemplateUsage{Name: name, Count: count})
	}
	sort.Slice(usage, func(i, j int) bool {
		return usage[i].Count > usage[j].Count
	})
	if len(usage) > 10 {
		usage = usage[:10]
	}
	return usage
}

func (ph *PromptHistory) mapToLanguageUsage(counts map[string]int) []prompt.LanguageUsage {
	usage := make([]prompt.LanguageUsage, 0, len(counts))
	for lang, count := range counts {
		usage = append(usage, prompt.LanguageUsage{Language: lang, Count: count})
	}
	sort.Slice(usage, func(i, j int) bool {
		return usage[i].Count > usage[j].Count
	})
	if len(usage) > 10 {
		usage = usage[:10]
	}
	return usage
}

func (ph *PromptHistory) mapToRepositoryUsage(counts map[string]int) []prompt.RepositoryUsage {
	usage := make([]prompt.RepositoryUsage, 0, len(counts))
	for repo, count := range counts {
		usage = append(usage, prompt.RepositoryUsage{Repository: repo, Count: count})
	}
	sort.Slice(usage, func(i, j int) bool {
		return usage[i].Count > usage[j].Count
	})
	if len(usage) > 10 {
		usage = usage[:10]
	}
	return usage
}

// Filtering methods
func (ph *PromptHistory) applyFilters() {
	ph.filteredEntries = make([]prompt.HistoryEntry, 0)

	for _, entry := range ph.entries {
		if ph.matchesFilters(entry) && ph.matchesSearch(entry) {
			ph.filteredEntries = append(ph.filteredEntries, entry)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(ph.filteredEntries, func(i, j int) bool {
		return ph.filteredEntries[i].Timestamp.After(ph.filteredEntries[j].Timestamp)
	})
}

func (ph *PromptHistory) matchesFilters(entry prompt.HistoryEntry) bool {
	for _, filter := range ph.activeFilters {
		switch filter.Type {
		case FilterByTemplate:
			if entry.Template != filter.Value {
				return false
			}
		case FilterByLanguage:
			if entry.Language != filter.Value {
				return false
			}
		case FilterByRepository:
			if entry.Repository != filter.Value {
				return false
			}
		case FilterByStatus:
			success := entry.Success
			if (filter.Value == "success" && !success) || (filter.Value == "failed" && success) {
				return false
			}
		case FilterByDate:
			// Simple date filtering (today, this week, this month)
			now := time.Now()
			switch filter.Value {
			case "today":
				if !ph.isSameDay(entry.Timestamp, now) {
					return false
				}
			case "week":
				if entry.Timestamp.Before(now.AddDate(0, 0, -7)) {
					return false
				}
			case "month":
				if entry.Timestamp.Before(now.AddDate(0, -1, 0)) {
					return false
				}
			}
		}
	}
	return true
}

func (ph *PromptHistory) matchesSearch(entry prompt.HistoryEntry) bool {
	if ph.searchQuery == "" {
		return true
	}

	query := strings.ToLower(ph.searchQuery)
	
	searchFields := []string{
		strings.ToLower(entry.Template),
		strings.ToLower(entry.Language),
		strings.ToLower(entry.Framework),
		strings.ToLower(entry.Repository),
		strings.ToLower(entry.OutputMethod),
		strings.ToLower(entry.ErrorMessage),
	}

	// Search in parameter values
	for _, value := range entry.Parameters {
		searchFields = append(searchFields, strings.ToLower(value))
	}

	for _, field := range searchFields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

func (ph *PromptHistory) isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// UI update methods
func (ph *PromptHistory) updateUI() {
	ph.updateHistoryList()
	ph.updateDetailsPane()
	ph.updateStatsPane()
	ph.updateFilterBar()
	ph.updateStatusBar()
}

func (ph *PromptHistory) updateHistoryList() {
	ph.historyList.Clear()

	for i, entry := range ph.filteredEntries {
		icon := ph.getEntryIcon(entry)
		statusColor := ph.getEntryStatusColor(entry)
		
		colorStart, colorEnd := ph.config.Theme.ColorToTags(statusColor)
		mainText := fmt.Sprintf("%s %s%s%s", icon, colorStart, entry.Template, colorEnd)
		
		// Build secondary text with key information
		var secondary strings.Builder
		secondary.WriteString(FormatTime(entry.Timestamp))
		
		if entry.Language != "" {
			secondary.WriteString(fmt.Sprintf(" • %s", entry.Language))
		}
		
		if entry.WordCount > 0 {
			secondary.WriteString(fmt.Sprintf(" • %d words", entry.WordCount))
		}
		
		if entry.Duration > 0 {
			secondary.WriteString(fmt.Sprintf(" • %s", FormatDuration(entry.Duration)))
		}
		
		if !entry.Success && entry.ErrorMessage != "" {
			secondary.WriteString(fmt.Sprintf(" • Error: %s", TruncateText(entry.ErrorMessage, 30)))
		}

		shortcut := rune(0)
		if i < 9 {
			shortcut = rune('1' + i)
		}

		ph.historyList.AddItem(mainText, secondary.String(), shortcut, nil)
	}

	// Update selection if we have entries
	if len(ph.filteredEntries) > 0 && ph.selectedEntry == nil {
		ph.selectedEntry = &ph.filteredEntries[0]
	}
}

func (ph *PromptHistory) getEntryIcon(entry prompt.HistoryEntry) string {
	if entry.Success {
		return ph.config.Icons.Success
	} else {
		return ph.config.Icons.Error
	}
}

func (ph *PromptHistory) getEntryStatusColor(entry prompt.HistoryEntry) tcell.Color {
	if entry.Success {
		return ph.config.Theme.Success
	} else {
		return ph.config.Theme.Error
	}
}

func (ph *PromptHistory) updateDetailsPane() {
	if ph.selectedEntry == nil {
		ph.detailsPane.SetText("No entry selected")
		return
	}

	entry := *ph.selectedEntry
	var details strings.Builder

	// Header
	colorStart, colorEnd := ph.config.Theme.ColorToTags(ph.config.Theme.Primary)
	details.WriteString(fmt.Sprintf("%s%s%s\n\n", colorStart, entry.Template, colorEnd))

	// Basic information
	details.WriteString("[yellow]Basic Information:[white]\n")
	details.WriteString(fmt.Sprintf("  ID: %s\n", entry.ID))
	details.WriteString(fmt.Sprintf("  Timestamp: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05")))
	details.WriteString(fmt.Sprintf("  Success: %t\n", entry.Success))
	if entry.Language != "" {
		details.WriteString(fmt.Sprintf("  Language: %s\n", entry.Language))
	}
	if entry.Framework != "" {
		details.WriteString(fmt.Sprintf("  Framework: %s\n", entry.Framework))
	}
	if entry.Repository != "" {
		details.WriteString(fmt.Sprintf("  Repository: %s\n", entry.Repository))
	}

	// Performance metrics
	details.WriteString("\n[yellow]Performance:[white]\n")
	details.WriteString(fmt.Sprintf("  Duration: %s\n", FormatDuration(entry.Duration)))
	details.WriteString(fmt.Sprintf("  Word Count: %d\n", entry.WordCount))

	// Output information
	details.WriteString("\n[yellow]Output:[white]\n")
	details.WriteString(fmt.Sprintf("  Method: %s\n", entry.OutputMethod))
	if entry.AITool != "" {
		details.WriteString(fmt.Sprintf("  AI Tool: %s\n", entry.AITool))
	}

	// Parameters
	if len(entry.Parameters) > 0 {
		details.WriteString("\n[yellow]Parameters:[white]\n")
		for key, value := range entry.Parameters {
			details.WriteString(fmt.Sprintf("  %s: %s\n", key, TruncateText(value, 50)))
		}
	}

	// Error information
	if !entry.Success && entry.ErrorMessage != "" {
		details.WriteString("\n[red]Error:[white]\n")
		details.WriteString(fmt.Sprintf("  %s\n", entry.ErrorMessage))
	}

	ph.detailsPane.SetText(details.String())
}

func (ph *PromptHistory) updateStatsPane() {
	if ph.stats == nil {
		ph.statsPane.SetText("No statistics available")
		return
	}

	var stats strings.Builder

	// Header
	colorStart, colorEnd := ph.config.Theme.ColorToTags(ph.config.Theme.Secondary)
	stats.WriteString(fmt.Sprintf("%sGeneration Statistics%s\n\n", colorStart, colorEnd))

	// Overall stats
	stats.WriteString("[yellow]Overall:[white]\n")
	stats.WriteString(fmt.Sprintf("  Total Generations: %d\n", ph.stats.TotalGenerations))
	stats.WriteString(fmt.Sprintf("  Success Rate: %.1f%%\n", ph.stats.SuccessRate*100))
	stats.WriteString(fmt.Sprintf("  Average Words: %d\n", ph.stats.AverageWordCount))

	// Top templates
	if len(ph.stats.TopTemplates) > 0 {
		stats.WriteString("\n[yellow]Most Used Templates:[white]\n")
		for i, template := range ph.stats.TopTemplates {
			if i >= 5 {
				break
			}
			stats.WriteString(fmt.Sprintf("  %d. %s (%d times)\n", i+1, template.Name, template.Count))
		}
	}

	// Top languages
	if len(ph.stats.TopLanguages) > 0 {
		stats.WriteString("\n[yellow]Most Used Languages:[white]\n")
		for i, language := range ph.stats.TopLanguages {
			if i >= 5 {
				break
			}
			stats.WriteString(fmt.Sprintf("  %d. %s (%d times)\n", i+1, language.Language, language.Count))
		}
	}

	// Current filters summary
	if len(ph.filteredEntries) != len(ph.entries) {
		stats.WriteString(fmt.Sprintf("\n[yellow]Current View:[white]\n"))
		stats.WriteString(fmt.Sprintf("  Showing: %d / %d entries\n", len(ph.filteredEntries), len(ph.entries)))
	}

	ph.statsPane.SetText(stats.String())
}

func (ph *PromptHistory) updateFilterBar() {
	if len(ph.activeFilters) == 0 && ph.searchQuery == "" {
		ph.filterBar.SetText("No active filters")
		return
	}

	var filters strings.Builder

	// Active filters
	for i, filter := range ph.activeFilters {
		if i > 0 {
			filters.WriteString(" • ")
		}
		colorStart, colorEnd := ph.config.Theme.ColorToTags(ph.config.Theme.Accent)
		filters.WriteString(fmt.Sprintf("%s%s: %s%s", colorStart, filter.Label, filter.Value, colorEnd))
	}

	// Search query
	if ph.searchQuery != "" {
		if filters.Len() > 0 {
			filters.WriteString(" • ")
		}
		colorStart, colorEnd := ph.config.Theme.ColorToTags(ph.config.Theme.Info)
		filters.WriteString(fmt.Sprintf("%sSearch: %s%s", colorStart, ph.searchQuery, colorEnd))
	}

	ph.filterBar.SetText(filters.String())
}

func (ph *PromptHistory) updateStatusBar() {
	var status strings.Builder

	status.WriteString(fmt.Sprintf("Entries: %d", len(ph.filteredEntries)))
	
	if len(ph.filteredEntries) != len(ph.entries) {
		status.WriteString(fmt.Sprintf(" (filtered from %d)", len(ph.entries)))
	}

	if ph.selectedEntry != nil {
		status.WriteString(fmt.Sprintf(" | Selected: %s", ph.selectedEntry.Template))
	}

	if len(ph.activeFilters) > 0 {
		status.WriteString(fmt.Sprintf(" | Filters: %d", len(ph.activeFilters)))
	}

	if ph.config.EnableKeyHelp {
		status.WriteString(" | ")
		status.WriteString(ph.keyBindings.GetHelpText(ph.config.Theme))
	}

	ph.statusBar.SetText(status.String())
}

// Event handlers
func (ph *PromptHistory) onEntryChanged(index int, mainText, secondaryText string, shortcut rune) {
	if index >= 0 && index < len(ph.filteredEntries) {
		ph.selectedEntry = &ph.filteredEntries[index]
		ph.updateDetailsPane()
		ph.updateStatusBar()
	}
}

func (ph *PromptHistory) onEntrySelected(index int, mainText, secondaryText string, shortcut rune) {
	ph.selectEntry()
}

func (ph *PromptHistory) onSearchChanged(text string) {
	ph.searchQuery = text
	ph.applyFilters()
	ph.updateUI()
}

func (ph *PromptHistory) onSearchDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		ph.deactivateSearch()
	}
}

// Action methods
func (ph *PromptHistory) selectEntry() {
	if ph.selectedEntry == nil {
		return
	}

	if ph.callbacks.OnEntrySelect != nil {
		ph.callbacks.OnEntrySelect(*ph.selectedEntry)
	}
}

func (ph *PromptHistory) reuseEntry() {
	if ph.selectedEntry == nil {
		ph.showStatus("No entry selected", "warning")
		return
	}

	if ph.callbacks.OnEntryReuse != nil {
		ph.callbacks.OnEntryReuse(*ph.selectedEntry)
	}
}

func (ph *PromptHistory) deleteEntry() {
	if ph.selectedEntry == nil {
		ph.showStatus("No entry selected", "warning")
		return
	}

	// Show confirmation (in a real implementation, this would be a modal)
	if ph.callbacks.OnEntryDelete != nil {
		ph.callbacks.OnEntryDelete(*ph.selectedEntry)
	}

	// Remove from local lists
	ph.removeEntryFromLists(*ph.selectedEntry)
	ph.updateUI()
}

func (ph *PromptHistory) copyEntryContent() {
	if ph.selectedEntry == nil {
		ph.showStatus("No entry selected", "warning")
		return
	}

	// Create a text representation
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Template: %s\n", ph.selectedEntry.Template))
	content.WriteString(fmt.Sprintf("Timestamp: %s\n", ph.selectedEntry.Timestamp.Format("2006-01-02 15:04:05")))
	
	if len(ph.selectedEntry.Parameters) > 0 {
		content.WriteString("\nParameters:\n")
		for key, value := range ph.selectedEntry.Parameters {
			content.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	err := ph.manager.CopyToClipboard(content.String())
	if err != nil {
		ph.handleError(fmt.Errorf("failed to copy entry: %w", err))
		return
	}

	ph.showStatus("Entry copied to clipboard", "success")
}

func (ph *PromptHistory) exportEntry() {
	if ph.selectedEntry == nil {
		ph.showStatus("No entry selected", "warning")
		return
	}

	filename := fmt.Sprintf("history_entry_%s_%d.json", 
		strings.ReplaceAll(ph.selectedEntry.Template, " ", "_"),
		ph.selectedEntry.Timestamp.Unix())

	// Export as JSON (simplified)
	content := fmt.Sprintf(`{
  "id": "%s",
  "template": "%s",
  "timestamp": "%s",
  "success": %t,
  "language": "%s",
  "framework": "%s",
  "repository": "%s",
  "parameters": %v,
  "duration": "%s",
  "word_count": %d
}`, 
		ph.selectedEntry.ID,
		ph.selectedEntry.Template,
		ph.selectedEntry.Timestamp.Format(time.RFC3339),
		ph.selectedEntry.Success,
		ph.selectedEntry.Language,
		ph.selectedEntry.Framework,
		ph.selectedEntry.Repository,
		ph.selectedEntry.Parameters,
		ph.selectedEntry.Duration.String(),
		ph.selectedEntry.WordCount)

	err := ph.manager.SaveToFile(content, filename)
	if err != nil {
		ph.handleError(fmt.Errorf("failed to export entry: %w", err))
		return
	}

	ph.showStatus(fmt.Sprintf("Entry exported to %s", filename), "success")
}

func (ph *PromptHistory) showStats() {
	if ph.callbacks.OnStatsRequest != nil {
		ph.callbacks.OnStatsRequest()
	}
}

// View mode methods
func (ph *PromptHistory) setViewMode(mode HistoryViewMode) {
	ph.viewMode = mode
	ph.updateUI()
	
	var modeStr string
	switch mode {
	case ViewModeList:
		modeStr = "List"
	case ViewModeTable:
		modeStr = "Table"
	case ViewModeTimeline:
		modeStr = "Timeline"
	}
	
	ph.showStatus(fmt.Sprintf("Switched to %s view", modeStr), "info")
}

// Search methods
func (ph *PromptHistory) activateSearch() {
	ph.isSearchActive = true
	
	// Replace filter bar with search input
	ph.RemoveItem(ph.filterBar)
	ph.InsertItem(0, ph.searchInput, 3, 0, false)
	
	// Focus on search input
	app := ph.GetApplication()
	if app != nil {
		app.SetFocus(ph.searchInput)
	}
}

func (ph *PromptHistory) deactivateSearch() {
	ph.isSearchActive = false
	
	// Restore original layout
	ph.RemoveItem(ph.searchInput)
	ph.InsertItem(0, ph.filterBar, 3, 0, false)
	
	// Focus back on history list
	app := ph.GetApplication()
	if app != nil {
		app.SetFocus(ph.historyList)
	}
}

func (ph *PromptHistory) clearSearchOrExit() {
	if ph.isSearchActive {
		if ph.searchQuery != "" {
			ph.searchInput.SetText("")
			ph.searchQuery = ""
			ph.applyFilters()
			ph.updateUI()
		} else {
			ph.deactivateSearch()
		}
	}
}

// Filter methods
func (ph *PromptHistory) showFilterMenu() {
	// This would typically show a filter selection dialog
	ph.showStatus("Filter menu not implemented yet", "info")
}

func (ph *PromptHistory) clearFilters() {
	ph.activeFilters = []HistoryFilter{}
	ph.searchQuery = ""
	ph.applyFilters()
	ph.updateUI()
	ph.showStatus("All filters cleared", "info")
}

// Bulk operations
func (ph *PromptHistory) selectAll() {
	ph.showStatus("Select all not implemented yet", "info")
}

func (ph *PromptHistory) deleteSelected() {
	ph.showStatus("Delete selected not implemented yet", "info")
}

// Navigation methods
func (ph *PromptHistory) nextPanel() {
	if ph.historyList.HasFocus() {
		ph.setFocus(ph.detailsPane)
	} else if ph.detailsPane.HasFocus() {
		ph.setFocus(ph.statsPane)
	} else {
		ph.setFocus(ph.historyList)
	}
}

func (ph *PromptHistory) prevPanel() {
	if ph.statsPane.HasFocus() {
		ph.setFocus(ph.detailsPane)
	} else if ph.detailsPane.HasFocus() {
		ph.setFocus(ph.historyList)
	} else {
		ph.setFocus(ph.statsPane)
	}
}

func (ph *PromptHistory) setFocus(primitive tview.Primitive) {
	app := ph.GetApplication()
	if app != nil {
		app.SetFocus(primitive)
	}
}

// Utility methods
func (ph *PromptHistory) removeEntryFromLists(entry prompt.HistoryEntry) {
	// Remove from main entries
	for i, e := range ph.entries {
		if e.ID == entry.ID {
			ph.entries = append(ph.entries[:i], ph.entries[i+1:]...)
			break
		}
	}

	// Remove from filtered entries
	for i, e := range ph.filteredEntries {
		if e.ID == entry.ID {
			ph.filteredEntries = append(ph.filteredEntries[:i], ph.filteredEntries[i+1:]...)
			break
		}
	}

	// Update selected entry
	if ph.selectedEntry != nil && ph.selectedEntry.ID == entry.ID {
		if len(ph.filteredEntries) > 0 {
			ph.selectedEntry = &ph.filteredEntries[0]
		} else {
			ph.selectedEntry = nil
		}
	}
}

func (ph *PromptHistory) showStatus(message, msgType string) {
	statusMsg := NewStatusMessage(message, msgType, 3*time.Second)
	formattedMsg := statusMsg.FormatForDisplay(ph.config.Theme)
	
	// Temporarily update status bar
	originalText := ph.statusBar.GetText(false)
	ph.statusBar.SetText(formattedMsg)
	
	// Restore after duration
	go func() {
		time.Sleep(statusMsg.Duration)
		ph.statusBar.SetText(originalText)
	}()
}

func (ph *PromptHistory) handleError(err error) {
	ph.logger.Error("prompt history error", zap.Error(err))
	ph.showStatus(err.Error(), "error")
	
	if ph.callbacks.OnError != nil {
		ph.callbacks.OnError(err)
	}
}

// GetApplication returns the tview application
func (ph *PromptHistory) GetApplication() *tview.Application {
	// This would need to be set externally or passed in during initialization
	return nil
}

// Public interface methods

// GetSelectedEntry returns the currently selected history entry
func (ph *PromptHistory) GetSelectedEntry() *prompt.HistoryEntry {
	return ph.selectedEntry
}

// GetFilteredEntries returns the currently filtered entries
func (ph *PromptHistory) GetFilteredEntries() []prompt.HistoryEntry {
	return ph.filteredEntries
}

// AddFilter adds a filter to the active filters
func (ph *PromptHistory) AddFilter(filterType HistoryFilterType, value, label string) {
	filter := HistoryFilter{
		Type:  filterType,
		Value: value,
		Label: label,
	}
	ph.activeFilters = append(ph.activeFilters, filter)
	ph.applyFilters()
	ph.updateUI()
}

// RemoveFilter removes a filter from the active filters
func (ph *PromptHistory) RemoveFilter(filterType HistoryFilterType, value string) {
	for i, filter := range ph.activeFilters {
		if filter.Type == filterType && filter.Value == value {
			ph.activeFilters = append(ph.activeFilters[:i], ph.activeFilters[i+1:]...)
			break
		}
	}
	ph.applyFilters()
	ph.updateUI()
}

// SetSearchQuery sets the search query
func (ph *PromptHistory) SetSearchQuery(query string) {
	ph.searchQuery = query
	ph.applyFilters()
	ph.updateUI()
}

// Focus sets focus to the prompt history component
func (ph *PromptHistory) Focus(delegate func(p tview.Primitive)) {
	delegate(ph.historyList)
}

// HasFocus returns true if the prompt history has focus
func (ph *PromptHistory) HasFocus() bool {
	return ph.historyList.HasFocus() || ph.detailsPane.HasFocus() || ph.statsPane.HasFocus()
}

// Close cleans up the prompt history component
func (ph *PromptHistory) Close() {
	ph.cancel()
}