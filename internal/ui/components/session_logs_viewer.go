package components

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
	"openai-cli/internal/app/session"
	"openai-cli/internal/telemetry"
)

// SessionLogsViewer provides a comprehensive UI for viewing and managing session logs
type SessionLogsViewer struct {
	*tview.Flex
	manager         *session.Manager
	telemetryClient telemetry.Client
	logger          *zap.Logger

	// UI components
	sessionList     *tview.Table
	conversationView *tview.TextView
	searchField     *tview.InputField
	filterDropdown  *tview.DropDown
	detailsView     *tview.TextView
	statusBar       *tview.TextView
	helpText        *tview.TextView

	// State
	allSessions       []*claude.Session
	filteredSessions  []*claude.Session
	currentSessionID  string
	searchTerm        string
	filterType        string
	currentPage       int
	itemsPerPage      int
	refreshTicker     *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
}

// SessionLogsViewerConfig holds configuration for the session logs viewer
type SessionLogsViewerConfig struct {
	RefreshInterval time.Duration
	ItemsPerPage    int
	MaxSearchHistory int
}

// DefaultSessionLogsViewerConfig returns default configuration
func DefaultSessionLogsViewerConfig() *SessionLogsViewerConfig {
	return &SessionLogsViewerConfig{
		RefreshInterval:  5 * time.Second,
		ItemsPerPage:     20,
		MaxSearchHistory: 10,
	}
}

// NewSessionLogsViewer creates a new session logs viewer
func NewSessionLogsViewer(manager *session.Manager, telemetryClient telemetry.Client, logger *zap.Logger) *SessionLogsViewer {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	viewer := &SessionLogsViewer{
		Flex:            tview.NewFlex(),
		manager:         manager,
		telemetryClient: telemetryClient,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		itemsPerPage:    DefaultSessionLogsViewerConfig().ItemsPerPage,
		filterType:      "all",
	}

	viewer.setupUI()
	viewer.refreshSessions()
	viewer.startAutoRefresh()

	return viewer
}

// setupUI initializes the UI components
func (sv *SessionLogsViewer) setupUI() {
	sv.SetDirection(tview.FlexRow)

	// Create search and filter controls
	searchFilterFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	
	// Search field
	sv.searchField = tview.NewInputField()
	sv.searchField.SetLabel("Search: ").SetFieldWidth(0)
	sv.searchField.SetChangedFunc(func(text string) {
		sv.searchTerm = text
		sv.applyFilters()
	})
	sv.searchField.SetBorder(true).SetTitle(" Search Sessions ")

	// Filter dropdown
	filterOptions := []string{"all", "active", "paused", "terminated", "error", "today", "week", "month"}
	sv.filterDropdown = tview.NewDropDown()
	sv.filterDropdown.SetLabel("Filter: ").SetOptions(filterOptions, func(option string, index int) {
		sv.filterType = option
		sv.applyFilters()
	})
	sv.filterDropdown.SetBorder(true).SetTitle(" Filter ")

	searchFilterFlex.AddItem(sv.searchField, 0, 2, false)
	searchFilterFlex.AddItem(sv.filterDropdown, 30, 0, false)

	// Create session list table
	sv.sessionList = tview.NewTable()
	sv.sessionList.SetBorder(true).SetTitle(" Session History ")
	sv.sessionList.SetSelectable(true, false)
	sv.sessionList.SetFixed(1, 0)

	// Set table headers
	headers := []string{"ID", "Name", "State", "Created", "Duration", "Messages", "Tokens", "Status"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		sv.sessionList.SetCell(0, i, cell)
	}

	// Create conversation view
	sv.conversationView = tview.NewTextView()
	sv.conversationView.SetBorder(true).SetTitle(" Conversation History ")
	sv.conversationView.SetScrollable(true).SetWordWrap(true)
	sv.conversationView.SetDynamicColors(true)

	// Create session details view
	sv.detailsView = tview.NewTextView()
	sv.detailsView.SetBorder(true).SetTitle(" Session Details ")
	sv.detailsView.SetScrollable(true)
	sv.detailsView.SetDynamicColors(true)

	// Create help text
	sv.helpText = tview.NewTextView()
	sv.helpText.SetBorder(true).SetTitle(" Commands ")
	sv.helpText.SetDynamicColors(true)
	sv.helpText.SetText(`[yellow]Navigation:[white]
  ↑/↓ - Navigate sessions
  Enter - View conversation
  Tab - Switch panels
  
[yellow]Search & Filter:[white]
  / - Focus search
  f - Focus filter
  c - Clear filters
  
[yellow]Actions:[white]
  r - Refresh
  e - Export session
  d - Delete session
  
[yellow]Pagination:[white]
  PgUp/PgDn - Navigate pages
  Home/End - First/Last page
  
[yellow]Other:[white]
  h - Toggle help
  q - Quit to menu`)

	// Create status bar
	sv.statusBar = tview.NewTextView()
	sv.statusBar.SetDynamicColors(true)

	// Create main content area
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(sv.sessionList, 0, 2, true)
	leftPanel.AddItem(sv.detailsView, 0, 1, false)

	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPanel.AddItem(sv.conversationView, 0, 3, false)
	rightPanel.AddItem(sv.helpText, 12, 0, false)

	mainContent := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainContent.AddItem(leftPanel, 0, 1, true)
	mainContent.AddItem(rightPanel, 0, 2, false)

	// Add components to main flex
	sv.AddItem(searchFilterFlex, 3, 0, false)
	sv.AddItem(mainContent, 0, 1, true)
	sv.AddItem(sv.statusBar, 1, 0, false)

	// Set up event handlers
	sv.setupEventHandlers()
}

// setupEventHandlers sets up keyboard and selection event handlers
func (sv *SessionLogsViewer) setupEventHandlers() {
	// Session list selection handler
	sv.sessionList.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 && row-1 < len(sv.filteredSessions) {
			session := sv.filteredSessions[row-1]
			sv.currentSessionID = session.GetID()
			sv.loadSessionDetails(session)
			sv.loadConversationHistory(session)
		}
	})

	// Main input capture
	sv.SetInputCapture(sv.handleInput)
}

// handleInput handles keyboard input
func (sv *SessionLogsViewer) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		// Cycle through focusable components
		return event
	case tcell.KeyEnter:
		if sv.currentSessionID != "" {
			sv.viewFullConversation()
		}
		return nil
	case tcell.KeyF5:
		sv.refreshSessions()
		return nil
	case tcell.KeyPgUp:
		sv.previousPage()
		return nil
	case tcell.KeyPgDn:
		sv.nextPage()
		return nil
	case tcell.KeyHome:
		sv.firstPage()
		return nil
	case tcell.KeyEnd:
		sv.lastPage()
		return nil
	}

	switch event.Rune() {
	case '/':
		// Focus search field
		sv.searchField.SetText("")
		return event
	case 'f', 'F':
		// Focus filter dropdown
		return event
	case 'c', 'C':
		sv.clearFilters()
		return nil
	case 'r', 'R':
		sv.refreshSessions()
		return nil
	case 'e', 'E':
		sv.exportSession()
		return nil
	case 'd', 'D':
		sv.deleteCurrentSession()
		return nil
	case 'h', 'H':
		sv.toggleHelp()
		return nil
	case 'q', 'Q':
		// Let parent handle quit
		return event
	}

	return event
}

// refreshSessions refreshes the session list from the manager
func (sv *SessionLogsViewer) refreshSessions() {
	if sv.manager == nil {
		sv.logger.Warn("Session manager is nil, cannot refresh sessions")
		return
	}

	sv.allSessions = sv.manager.ListSessions()
	
	// Sort sessions by creation time (newest first)
	sort.Slice(sv.allSessions, func(i, j int) bool {
		return sv.allSessions[i].GetMetadata().CreatedAt.After(sv.allSessions[j].GetMetadata().CreatedAt)
	})

	sv.applyFilters()
	sv.updateStatusBar()

	// Log telemetry
	if sv.telemetryClient != nil && sv.telemetryClient.IsEnabled() {
		telemetry.LogUserAction(sv.telemetryClient, "session_logs_refresh", map[string]interface{}{
			"total_sessions": len(sv.allSessions),
		})
	}
}

// applyFilters applies current search and filter criteria
func (sv *SessionLogsViewer) applyFilters() {
	sv.filteredSessions = make([]*claude.Session, 0)

	for _, session := range sv.allSessions {
		if sv.matchesFilters(session) {
			sv.filteredSessions = append(sv.filteredSessions, session)
		}
	}

	sv.updateSessionTable()
	sv.updateStatusBar()
}

// matchesFilters checks if a session matches current filter criteria
func (sv *SessionLogsViewer) matchesFilters(session *claude.Session) bool {
	metadata := session.GetMetadata()

	// Search term filter
	if sv.searchTerm != "" {
		searchLower := strings.ToLower(sv.searchTerm)
		if !strings.Contains(strings.ToLower(metadata.Name), searchLower) &&
			!strings.Contains(strings.ToLower(metadata.ID), searchLower) {
			return false
		}
	}

	// State filter
	switch sv.filterType {
	case "active":
		if metadata.State != claude.SessionStateActive {
			return false
		}
	case "paused":
		if metadata.State != claude.SessionStatePaused {
			return false
		}
	case "terminated":
		if metadata.State != claude.SessionStateTerminated {
			return false
		}
	case "error":
		if metadata.State != claude.SessionStateError {
			return false
		}
	case "today":
		if !sv.isToday(metadata.CreatedAt) {
			return false
		}
	case "week":
		if !sv.isThisWeek(metadata.CreatedAt) {
			return false
		}
	case "month":
		if !sv.isThisMonth(metadata.CreatedAt) {
			return false
		}
	}

	return true
}

// Time filter helpers
func (sv *SessionLogsViewer) isToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}

func (sv *SessionLogsViewer) isThisWeek(t time.Time) bool {
	now := time.Now()
	_, week := now.ISOWeek()
	_, tWeek := t.ISOWeek()
	return now.Year() == t.Year() && week == tWeek
}

func (sv *SessionLogsViewer) isThisMonth(t time.Time) bool {
	now := time.Now()
	return now.Year() == t.Year() && now.Month() == t.Month()
}

// updateSessionTable updates the session table with filtered results
func (sv *SessionLogsViewer) updateSessionTable() {
	// Clear existing rows (except header)
	rowCount := sv.sessionList.GetRowCount()
	for row := 1; row < rowCount; row++ {
		sv.sessionList.RemoveRow(row)
	}

	// Calculate pagination
	startIdx := sv.currentPage * sv.itemsPerPage
	endIdx := startIdx + sv.itemsPerPage
	if endIdx > len(sv.filteredSessions) {
		endIdx = len(sv.filteredSessions)
	}

	// Add sessions for current page
	for i := startIdx; i < endIdx; i++ {
		session := sv.filteredSessions[i]
		metadata := session.GetMetadata()
		row := (i - startIdx) + 1

		// Session ID (truncated)
		displayID := metadata.ID
		if len(displayID) > 12 {
			displayID = displayID[:9] + "..."
		}
		sv.sessionList.SetCell(row, 0, tview.NewTableCell(displayID))

		// Name
		sv.sessionList.SetCell(row, 1, tview.NewTableCell(metadata.Name))

		// State with color
		stateColor := sv.getStateColor(metadata.State)
		stateCell := tview.NewTableCell(metadata.State.String()).SetTextColor(stateColor)
		sv.sessionList.SetCell(row, 2, stateCell)

		// Created time
		sv.sessionList.SetCell(row, 3, tview.NewTableCell(metadata.CreatedAt.Format("01/02 15:04")))

		// Duration
		duration := sv.calculateDuration(metadata)
		sv.sessionList.SetCell(row, 4, tview.NewTableCell(duration))

		// Message count
		messageCount := "0"
		tokenCount := "0"
		statusIndicator := "●"
		
		if metadata.Stats != nil {
			messageCount = strconv.Itoa(int(metadata.Stats.InputCount))
			tokenCount = fmt.Sprintf("%.1fk", float64(metadata.Stats.OutputBytes)/1000)
			
			if metadata.Stats.ErrorCount > 0 {
				statusIndicator = "⚠"
			} else if metadata.State == claude.SessionStateActive {
				statusIndicator = "●"
			} else {
				statusIndicator = "○"
			}
		}

		sv.sessionList.SetCell(row, 5, tview.NewTableCell(messageCount))
		sv.sessionList.SetCell(row, 6, tview.NewTableCell(tokenCount))
		sv.sessionList.SetCell(row, 7, tview.NewTableCell(statusIndicator))
	}

	// Select first session if none selected
	if len(sv.filteredSessions) > 0 && sv.currentSessionID == "" {
		sv.sessionList.Select(1, 0)
	}
}

// getStateColor returns appropriate color for session state
func (sv *SessionLogsViewer) getStateColor(state claude.SessionState) tcell.Color {
	switch state {
	case claude.SessionStateActive:
		return tcell.ColorGreen
	case claude.SessionStatePaused:
		return tcell.ColorYellow
	case claude.SessionStateTerminated:
		return tcell.ColorGray
	case claude.SessionStateError:
		return tcell.ColorRed
	default:
		return tcell.ColorWhite
	}
}

// calculateDuration calculates session duration string
func (sv *SessionLogsViewer) calculateDuration(metadata *claude.SessionMetadata) string {
	var endTime time.Time
	if metadata.State == claude.SessionStateActive || metadata.State == claude.SessionStatePaused {
		endTime = time.Now()
	} else {
		endTime = metadata.UpdatedAt
	}

	duration := endTime.Sub(metadata.CreatedAt)
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else {
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
}

// loadSessionDetails loads detailed information for a session
func (sv *SessionLogsViewer) loadSessionDetails(session *claude.Session) {
	metadata := session.GetMetadata()
	
	details := fmt.Sprintf(`[yellow]Session Details[white]

[yellow]ID:[white] %s
[yellow]Name:[white] %s
[yellow]State:[white] %s
[yellow]Created:[white] %s
[yellow]Updated:[white] %s
[yellow]Duration:[white] %s`,
		metadata.ID,
		metadata.Name,
		metadata.State.String(),
		metadata.CreatedAt.Format("2006-01-02 15:04:05"),
		metadata.UpdatedAt.Format("2006-01-02 15:04:05"),
		sv.calculateDuration(metadata))

	if metadata.Config != nil {
		details += fmt.Sprintf(`

[yellow]Configuration:[white]
  Max History: %d
  Auto Save: %v`,
			metadata.Config.MaxHistory,
			metadata.Config.AutoSave)
	}

	if metadata.Stats != nil {
		details += fmt.Sprintf(`

[yellow]Statistics:[white]
  Input Count: %d
  Output Bytes: %d
  Error Count: %d
  Last Activity: %s`,
			metadata.Stats.InputCount,
			metadata.Stats.OutputBytes,
			metadata.Stats.ErrorCount,
			metadata.Stats.LastActive.Format("15:04:05"))
	}

	sv.detailsView.SetText(details)
}

// loadConversationHistory loads conversation history for a session
func (sv *SessionLogsViewer) loadConversationHistory(session *claude.Session) {
	// Try to get conversation history
	output, err := session.GetOutput()
	if err != nil {
		sv.conversationView.SetText(fmt.Sprintf("[red]Error loading conversation: %v[white]", err))
		return
	}

	if len(output) == 0 {
		sv.conversationView.SetText("[gray]No conversation history available[white]")
		return
	}

	// Format the conversation for display
	conversation := sv.formatConversation(string(output))
	sv.conversationView.SetText(conversation)
	sv.conversationView.ScrollToBeginning()
}

// formatConversation formats raw conversation data for display
func (sv *SessionLogsViewer) formatConversation(raw string) string {
	// Simple formatting - could be enhanced based on actual format
	lines := strings.Split(raw, "\n")
	var formatted strings.Builder

	for i, line := range lines {
		if strings.HasPrefix(line, "User:") {
			formatted.WriteString(fmt.Sprintf("[blue]%s[white]\n", line))
		} else if strings.HasPrefix(line, "Assistant:") || strings.HasPrefix(line, "Claude:") {
			formatted.WriteString(fmt.Sprintf("[green]%s[white]\n", line))
		} else if strings.HasPrefix(line, "System:") {
			formatted.WriteString(fmt.Sprintf("[yellow]%s[white]\n", line))
		} else if strings.HasPrefix(line, "Error:") {
			formatted.WriteString(fmt.Sprintf("[red]%s[white]\n", line))
		} else {
			formatted.WriteString(line + "\n")
		}

		// Add separator between messages
		if i < len(lines)-1 && (strings.HasPrefix(lines[i+1], "User:") || strings.HasPrefix(lines[i+1], "Assistant:")) {
			formatted.WriteString("[gray]────────────────────────────────────────[white]\n")
		}
	}

	return formatted.String()
}

// Pagination methods
func (sv *SessionLogsViewer) nextPage() {
	maxPage := (len(sv.filteredSessions) - 1) / sv.itemsPerPage
	if sv.currentPage < maxPage {
		sv.currentPage++
		sv.updateSessionTable()
		sv.updateStatusBar()
	}
}

func (sv *SessionLogsViewer) previousPage() {
	if sv.currentPage > 0 {
		sv.currentPage--
		sv.updateSessionTable()
		sv.updateStatusBar()
	}
}

func (sv *SessionLogsViewer) firstPage() {
	sv.currentPage = 0
	sv.updateSessionTable()
	sv.updateStatusBar()
}

func (sv *SessionLogsViewer) lastPage() {
	maxPage := (len(sv.filteredSessions) - 1) / sv.itemsPerPage
	if maxPage >= 0 {
		sv.currentPage = maxPage
		sv.updateSessionTable()
		sv.updateStatusBar()
	}
}

// Action methods
func (sv *SessionLogsViewer) clearFilters() {
	sv.searchTerm = ""
	sv.filterType = "all"
	sv.searchField.SetText("")
	sv.filterDropdown.SetCurrentOption(0)
	sv.currentPage = 0
	sv.applyFilters()
}

func (sv *SessionLogsViewer) viewFullConversation() {
	// This could open a modal or new page with full conversation
	sv.showStatus("Full conversation view not implemented yet", tcell.ColorYellow)
}

func (sv *SessionLogsViewer) exportSession() {
	if sv.currentSessionID == "" {
		sv.showStatus("No session selected", tcell.ColorRed)
		return
	}
	sv.showStatus("Export functionality not implemented yet", tcell.ColorYellow)
}

func (sv *SessionLogsViewer) deleteCurrentSession() {
	if sv.currentSessionID == "" {
		sv.showStatus("No session selected", tcell.ColorRed)
		return
	}

	// For safety, require confirmation in a real implementation
	err := sv.manager.DeleteSession(sv.currentSessionID, true)
	if err != nil {
		sv.showStatus(fmt.Sprintf("Error deleting session: %v", err), tcell.ColorRed)
		return
	}

	sv.currentSessionID = ""
	sv.showStatus("Session deleted", tcell.ColorGreen)
	sv.refreshSessions()
}

func (sv *SessionLogsViewer) toggleHelp() {
	// Toggle help visibility
	if sv.helpText.HasFocus() {
		sv.showStatus("Help hidden", tcell.ColorGray)
	} else {
		sv.showStatus("Help shown", tcell.ColorGray)
	}
}

// updateStatusBar updates the status bar with current information
func (sv *SessionLogsViewer) updateStatusBar() {
	totalPages := (len(sv.filteredSessions) + sv.itemsPerPage - 1) / sv.itemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	status := fmt.Sprintf(" Sessions: %d/%d | Page: %d/%d | Filter: %s",
		len(sv.filteredSessions),
		len(sv.allSessions),
		sv.currentPage+1,
		totalPages,
		sv.filterType)

	if sv.searchTerm != "" {
		status += fmt.Sprintf(" | Search: '%s'", sv.searchTerm)
	}

	sv.statusBar.SetText(status)
}

// showStatus shows a temporary status message
func (sv *SessionLogsViewer) showStatus(message string, color tcell.Color) {
	colorName := "white"
	switch color {
	case tcell.ColorRed:
		colorName = "red"
	case tcell.ColorGreen:
		colorName = "green"
	case tcell.ColorYellow:
		colorName = "yellow"
	case tcell.ColorGray:
		colorName = "gray"
	}

	originalText := sv.statusBar.GetText(false)
	sv.statusBar.SetText(fmt.Sprintf("[%s]%s[white]", colorName, message))

	// Reset after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		sv.statusBar.SetText(originalText)
	}()
}

// startAutoRefresh starts automatic refresh timer
func (sv *SessionLogsViewer) startAutoRefresh() {
	sv.refreshTicker = time.NewTicker(DefaultSessionLogsViewerConfig().RefreshInterval)
	
	go func() {
		for {
			select {
			case <-sv.ctx.Done():
				return
			case <-sv.refreshTicker.C:
				// Only refresh if we have active sessions
				if sv.hasActiveSessions() {
					sv.refreshSessions()
				}
			}
		}
	}()
}

// hasActiveSessions checks if there are any active sessions
func (sv *SessionLogsViewer) hasActiveSessions() bool {
	for _, session := range sv.allSessions {
		if session.GetMetadata().State == claude.SessionStateActive {
			return true
		}
	}
	return false
}

// Close cleans up resources
func (sv *SessionLogsViewer) Close() {
	if sv.refreshTicker != nil {
		sv.refreshTicker.Stop()
	}
	sv.cancel()
}