package components

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
	"openai-cli/internal/app/session"
)

// SimpleSessionView provides a simplified TUI for managing Claude Code sessions
type SimpleSessionView struct {
	*tview.Flex
	manager *session.Manager
	logger  *zap.Logger

	// UI components
	sessionList    *tview.Table
	sessionDetails *tview.TextView
	statusBar      *tview.TextView
	helpText       *tview.TextView

	// State
	currentSessionID string
	sessions         []*claude.Session
}

// NewSimpleSessionView creates a new simplified session management view
func NewSimpleSessionView(manager *session.Manager, logger *zap.Logger) *SimpleSessionView {
	if logger == nil {
		logger = zap.NewNop()
	}

	sv := &SimpleSessionView{
		Flex:    tview.NewFlex(),
		manager: manager,
		logger:  logger,
	}

	sv.setupUI()
	sv.refreshSessions()

	return sv
}

// setupUI initializes the UI components
func (sv *SimpleSessionView) setupUI() {
	// Create main layout
	sv.SetDirection(tview.FlexRow)

	// Create session list table
	sv.sessionList = tview.NewTable()
	sv.sessionList.
		SetBorder(true).
		SetTitle(" Claude Code Sessions ").
		SetTitleAlign(tview.AlignLeft)
	sv.sessionList.SetSelectable(true, false)

	// Set headers
	sv.sessionList.SetCell(0, 0, tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow))
	sv.sessionList.SetCell(0, 1, tview.NewTableCell("Name").SetTextColor(tcell.ColorYellow))
	sv.sessionList.SetCell(0, 2, tview.NewTableCell("State").SetTextColor(tcell.ColorYellow))
	sv.sessionList.SetCell(0, 3, tview.NewTableCell("Created").SetTextColor(tcell.ColorYellow))
	sv.sessionList.SetFixed(1, 0)

	// Create session details
	sv.sessionDetails = tview.NewTextView()
	sv.sessionDetails.
		SetBorder(true).
		SetTitle(" Session Details ").
		SetTitleAlign(tview.AlignLeft)
	sv.sessionDetails.SetDynamicColors(true)

	// Create help text
	sv.helpText = tview.NewTextView()
	sv.helpText.SetBorder(true).SetTitle(" Help ")
	sv.helpText.SetDynamicColors(true)
	sv.helpText.SetText(`[yellow]Commands:[white]
  n - New session
  s - Start session  
  t - Terminate session
  d - Delete session
  r - Refresh list
  q - Quit to menu`)

	// Create status bar
	sv.statusBar = tview.NewTextView()
	sv.statusBar.SetDynamicColors(true)
	sv.statusBar.SetTextAlign(tview.AlignLeft)

	// Layout components
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(sv.sessionList, 0, 2, true)
	leftPanel.AddItem(sv.sessionDetails, 0, 1, false)

	mainContent := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainContent.AddItem(leftPanel, 0, 3, true)
	mainContent.AddItem(sv.helpText, 30, 0, false)

	sv.AddItem(mainContent, 0, 1, true)
	sv.AddItem(sv.statusBar, 1, 0, false)

	// Set up selection handler
	sv.sessionList.SetSelectionChangedFunc(func(row, column int) {
		if row > 0 && row <= len(sv.sessions) {
			sv.selectSession(row - 1)
		}
	})

	// Set up input capture
	sv.SetInputCapture(sv.handleInput)
}

// handleInput handles keyboard input
func (sv *SimpleSessionView) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'n', 'N':
		sv.createSession()
		return nil
	case 's', 'S':
		sv.startSession()
		return nil
	case 't', 'T':
		sv.terminateSession()
		return nil
	case 'd', 'D':
		sv.deleteSession()
		return nil
	case 'r', 'R':
		sv.refreshSessions()
		return nil
	case 'q', 'Q':
		// Let parent handle quit
		return event
	}

	switch event.Key() {
	case tcell.KeyF5:
		sv.refreshSessions()
		return nil
	}

	return event
}

// refreshSessions refreshes the session list
func (sv *SimpleSessionView) refreshSessions() {
	sv.sessions = sv.manager.ListSessions()
	
	// Clear table (except header)
	rowCount := sv.sessionList.GetRowCount()
	for row := 1; row < rowCount; row++ {
		for col := 0; col < 4; col++ {
			sv.sessionList.SetCell(row, col, tview.NewTableCell(""))
		}
	}

	// Populate table
	for i, session := range sv.sessions {
		row := i + 1
		metadata := session.GetMetadata()
		
		// Safely get state color
		stateColor := tcell.ColorWhite
		switch metadata.State {
		case claude.SessionStateActive:
			stateColor = tcell.ColorGreen
		case claude.SessionStatePaused:
			stateColor = tcell.ColorYellow
		case claude.SessionStateTerminated:
			stateColor = tcell.ColorRed
		case claude.SessionStateError:
			stateColor = tcell.ColorDarkRed
		}

		// Truncate long IDs for display
		displayID := metadata.ID
		if len(displayID) > 20 {
			displayID = displayID[:17] + "..."
		}

		sv.sessionList.SetCell(row, 0, tview.NewTableCell(displayID))
		sv.sessionList.SetCell(row, 1, tview.NewTableCell(metadata.Name))
		sv.sessionList.SetCell(row, 2, tview.NewTableCell(metadata.State.String()).SetTextColor(stateColor))
		sv.sessionList.SetCell(row, 3, tview.NewTableCell(metadata.CreatedAt.Format("15:04:05")))
	}

	sv.updateStatusBar()
	
	// Select first session if available
	if len(sv.sessions) > 0 {
		if sv.currentSessionID == "" || sv.getSessionIndex(sv.currentSessionID) == -1 {
			sv.sessionList.Select(1, 0)
			sv.selectSession(0)
		}
	} else {
		sv.currentSessionID = ""
		sv.sessionDetails.SetText("No sessions available")
	}
}

// getSessionIndex returns the index of a session by ID
func (sv *SimpleSessionView) getSessionIndex(sessionID string) int {
	for i, session := range sv.sessions {
		if session.GetID() == sessionID {
			return i
		}
	}
	return -1
}

// selectSession selects a session by index
func (sv *SimpleSessionView) selectSession(index int) {
	if index < 0 || index >= len(sv.sessions) {
		return
	}

	session := sv.sessions[index]
	sv.currentSessionID = session.GetID()
	sv.updateDetails(session)
}

// updateDetails updates the session details view
func (sv *SimpleSessionView) updateDetails(session *claude.Session) {
	metadata := session.GetMetadata()
	
	details := fmt.Sprintf(`[yellow]Session:[white] %s
[yellow]Name:[white] %s
[yellow]State:[white] %s
[yellow]Created:[white] %s
[yellow]Updated:[white] %s`,
		metadata.ID,
		metadata.Name,
		metadata.State.String(),
		metadata.CreatedAt.Format("2006-01-02 15:04:05"),
		metadata.UpdatedAt.Format("2006-01-02 15:04:05"))

	if metadata.Stats != nil {
		details += fmt.Sprintf(`

[yellow]Statistics:[white]
  Input Count: %d
  Output Bytes: %d
  Error Count: %d`,
			metadata.Stats.InputCount,
			metadata.Stats.OutputBytes,
			metadata.Stats.ErrorCount)
	}

	sv.sessionDetails.SetText(details)
}

// createSession creates a new session
func (sv *SimpleSessionView) createSession() {
	sessionName := fmt.Sprintf("session-%d", time.Now().Unix())
	
	session, err := sv.manager.CreateSession(sessionName, nil)
	if err != nil {
		sv.showStatus(fmt.Sprintf("Error creating session: %v", err), tcell.ColorRed)
		return
	}

	sv.showStatus(fmt.Sprintf("Created session: %s", session.GetName()), tcell.ColorGreen)
	sv.refreshSessions()
}

// startSession starts the selected session
func (sv *SimpleSessionView) startSession() {
	if sv.currentSessionID == "" {
		sv.showStatus("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.StartSession(sv.currentSessionID)
	if err != nil {
		sv.showStatus(fmt.Sprintf("Error starting session: %v", err), tcell.ColorRed)
		return
	}

	sv.showStatus("Session started", tcell.ColorGreen)
	sv.refreshSessions()
}

// terminateSession terminates the selected session
func (sv *SimpleSessionView) terminateSession() {
	if sv.currentSessionID == "" {
		sv.showStatus("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.TerminateSession(sv.currentSessionID)
	if err != nil {
		sv.showStatus(fmt.Sprintf("Error terminating session: %v", err), tcell.ColorRed)
		return
	}

	sv.showStatus("Session terminated", tcell.ColorYellow)
	sv.refreshSessions()
}

// deleteSession deletes the selected session
func (sv *SimpleSessionView) deleteSession() {
	if sv.currentSessionID == "" {
		sv.showStatus("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.DeleteSession(sv.currentSessionID, true)
	if err != nil {
		sv.showStatus(fmt.Sprintf("Error deleting session: %v", err), tcell.ColorRed)
		return
	}

	sv.currentSessionID = ""
	sv.showStatus("Session deleted", tcell.ColorRed)
	sv.refreshSessions()
}

// updateStatusBar updates the status bar
func (sv *SimpleSessionView) updateStatusBar() {
	stats := sv.manager.GetStats()
	
	status := fmt.Sprintf(" Total: %d | Active: %d | Terminated: %d",
		stats.TotalSessions,
		stats.ActiveSessions,
		stats.TerminatedSessions)
	
	sv.statusBar.SetText(status)
}

// showStatus shows a temporary status message
func (sv *SimpleSessionView) showStatus(message string, color tcell.Color) {
	colorName := "white"
	switch color {
	case tcell.ColorRed:
		colorName = "red"
	case tcell.ColorGreen:
		colorName = "green"
	case tcell.ColorYellow:
		colorName = "yellow"
	}
	
	sv.statusBar.SetText(fmt.Sprintf("[%s]%s[white]", colorName, message))
	
	// Reset after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		sv.updateStatusBar()
	}()
}