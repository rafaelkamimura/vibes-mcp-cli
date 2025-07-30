package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
	"openai-cli/internal/app/session"
)

// SessionView provides a TUI for managing Claude Code sessions
type SessionView struct {
	*tview.Flex
	manager         *session.Manager
	logger          *zap.Logger
	
	// UI components
	sessionList     *tview.List
	sessionDetails  *tview.TextView
	outputView      *tview.TextView
	inputField      *tview.InputField
	statusBar       *tview.TextView
	
	// State
	currentSession  *claude.Session
	outputChannel   <-chan []byte
	refreshTicker   *time.Ticker
	
	// Callbacks
	onSessionSelect func(session *claude.Session)
	onSessionStart  func(sessionID string)
	onSessionStop   func(sessionID string)
}

// NewSessionView creates a new session management view
func NewSessionView(manager *session.Manager, logger *zap.Logger) *SessionView {
	if logger == nil {
		logger = zap.NewNop()
	}

	sv := &SessionView{
		Flex:    tview.NewFlex(),
		manager: manager,
		logger:  logger,
	}

	sv.setupUI()
	sv.setupEventHandling()
	sv.startRefreshTimer()

	return sv
}

// setupUI initializes the UI components
func (sv *SessionView) setupUI() {
	// Create main layout
	sv.SetDirection(tview.FlexRow)

	// Create session list
	sv.sessionList = tview.NewList()
	sv.sessionList.
		SetBorder(true).
		SetTitle(" Sessions ").
		SetTitleAlign(tview.AlignLeft)
	sv.setupSessionList()

	// Create session details
	sv.sessionDetails = tview.NewTextView()
	sv.sessionDetails.
		SetBorder(true).
		SetTitle(" Session Details ").
		SetTitleAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWrap(true)

	// Create output view
	sv.outputView = tview.NewTextView()
	sv.outputView.
		SetBorder(true).
		SetTitle(" Output ").
		SetTitleAlign(tview.AlignLeft).
		SetScrollable(true).
		SetWrap(true).
		SetMaxLines(1000) // Limit output history

	// Create input field
	sv.inputField = tview.NewInputField()
	sv.inputField.
		SetBorder(true).
		SetTitle(" Input ").
		SetTitleAlign(tview.AlignLeft).
		SetPlaceholder("Enter command...")

	// Create status bar
	sv.statusBar = tview.NewTextView()
	sv.statusBar.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Layout components
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	leftPanel.AddItem(sv.sessionList, 0, 2, true)
	leftPanel.AddItem(sv.sessionDetails, 0, 1, false)

	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPanel.AddItem(sv.outputView, 0, 1, false)
	rightPanel.AddItem(sv.inputField, 3, 0, false)

	mainContent := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainContent.AddItem(leftPanel, 0, 1, true)
	mainContent.AddItem(rightPanel, 0, 2, false)

	sv.AddItem(mainContent, 0, 1, true)
	sv.AddItem(sv.statusBar, 1, 0, false)

	// Initial refresh
	sv.refreshSessionList()
	sv.updateStatusBar()
}

// setupSessionList configures the session list
func (sv *SessionView) setupSessionList() {
	sv.sessionList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		sv.selectSession(mainText)
	})

	sv.sessionList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		sv.selectSession(mainText)
	})
}

// setupEventHandling configures event handlers
func (sv *SessionView) setupEventHandling() {
	// Input field handling
	sv.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			sv.sendInput()
		}
	})

	// Global key bindings
	sv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'n':
			sv.createNewSession()
			return nil
		case 's':
			sv.startSelectedSession()
			return nil
		case 't':
			sv.terminateSelectedSession()
			return nil
		case 'p':
			sv.pauseSelectedSession()
			return nil
		case 'r':
			sv.resumeSelectedSession()
			return nil
		case 'd':
			sv.deleteSelectedSession()
			return nil
		case 'R':
			sv.refreshSessionList()
			return nil
		}

		switch event.Key() {
		case tcell.KeyF5:
			sv.refreshSessionList()
			return nil
		case tcell.KeyCtrlC:
			if sv.currentSession != nil && sv.currentSession.IsActive() {
				sv.terminateSelectedSession()
			}
			return nil
		}

		return event
	})
}

// startRefreshTimer starts the automatic refresh timer
func (sv *SessionView) startRefreshTimer() {
	sv.refreshTicker = time.NewTicker(time.Second * 5)
	go func() {
		for range sv.refreshTicker.C {
			sv.refreshSessionList()
			sv.updateSessionDetails()
		}
	}()
}

// refreshSessionList refreshes the session list
func (sv *SessionView) refreshSessionList() {
	sessions := sv.manager.ListSessions()
	
	sv.sessionList.Clear()
	
	for _, session := range sessions {
		metadata := session.GetMetadata()
		
		// Format session item
		mainText := metadata.Name
		if mainText == "" {
			mainText = metadata.ID
		}
		
		secondaryText := fmt.Sprintf("[%s] %s - %s",
			metadata.State.String(),
			metadata.CreatedAt.Format("Jan 2 15:04"),
			sv.formatDuration(session.GetDuration()))
		
		// Color code by state
		var color tcell.Color
		switch metadata.State {
		case claude.SessionStateActive:
			color = tcell.ColorGreen
		case claude.SessionStatePaused:
			color = tcell.ColorYellow
		case claude.SessionStateTerminated:
			color = tcell.ColorRed
		case claude.SessionStateError:
			color = tcell.ColorDarkRed
		default:
			color = tcell.ColorWhite
		}
		
		sv.sessionList.AddItem(mainText, secondaryText, 0, nil).
			SetSecondaryTextColor(color)
	}
	
	sv.updateStatusBar()
}

// selectSession selects a session by name
func (sv *SessionView) selectSession(sessionName string) {
	session, err := sv.manager.GetSessionByName(sessionName)
	if err != nil {
		sv.logger.Error("failed to get session", zap.Error(err))
		return
	}

	// Stop monitoring previous session
	if sv.outputChannel != nil {
		// Note: In a real implementation, you'd want to properly close the channel
		sv.outputChannel = nil
	}

	sv.currentSession = session
	sv.updateSessionDetails()

	// Start monitoring output if session is active
	if session.IsActive() {
		sv.startOutputMonitoring()
	}

	// Callback
	if sv.onSessionSelect != nil {
		sv.onSessionSelect(session)
	}
}

// updateSessionDetails updates the session details view
func (sv *SessionView) updateSessionDetails() {
	if sv.currentSession == nil {
		sv.sessionDetails.SetText("No session selected")
		return
	}

	metadata := sv.currentSession.GetMetadata()
	
	details := fmt.Sprintf(`[yellow]Session ID:[white] %s
[yellow]Name:[white] %s
[yellow]State:[white] %s
[yellow]Created:[white] %s
[yellow]Updated:[white] %s
[yellow]Working Dir:[white] %s
[yellow]Process ID:[white] %s

[yellow]Statistics:[white]
  Commands: %d
  Input Bytes: %s
  Output Bytes: %s
  Duration: %s
  Last Active: %s
  Peak Memory: %d MB
  Avg CPU: %.1f%%

[yellow]Tags:[white] %s
[yellow]Description:[white] %s`,
		metadata.ID,
		metadata.Name,
		metadata.State.String(),
		metadata.CreatedAt.Format("2006-01-02 15:04:05"),
		metadata.UpdatedAt.Format("2006-01-02 15:04:05"),
		metadata.WorkingDir,
		metadata.ProcessID,
		metadata.Stats.TotalCommands,
		sv.formatBytes(metadata.Stats.TotalInputBytes),
		sv.formatBytes(metadata.Stats.TotalOutputBytes),
		sv.formatDuration(metadata.Stats.Duration),
		sv.formatTime(metadata.Stats.LastActiveAt),
		metadata.Stats.PeakMemoryMB,
		metadata.Stats.AvgCPUPercent,
		strings.Join(metadata.Tags, ", "),
		metadata.Description)

	sv.sessionDetails.SetText(details)
}

// startOutputMonitoring starts monitoring session output
func (sv *SessionView) startOutputMonitoring() {
	if sv.currentSession == nil || !sv.currentSession.IsActive() {
		return
	}

	outputChan, err := sv.currentSession.SubscribeToOutput()
	if err != nil {
		sv.logger.Error("failed to subscribe to output", zap.Error(err))
		return
	}

	sv.outputChannel = outputChan

	go func() {
		for output := range outputChan {
			sv.outputView.Write(output)
		}
	}()
}

// sendInput sends input to the current session
func (sv *SessionView) sendInput() {
	if sv.currentSession == nil {
		sv.showMessage("No session selected", tcell.ColorRed)
		return
	}

	if !sv.currentSession.IsActive() {
		sv.showMessage("Session is not active", tcell.ColorRed)
		return
	}

	input := sv.inputField.GetText()
	if input == "" {
		return
	}

	err := sv.manager.SendInput(sv.currentSession.GetID(), input+"\n")
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to send input: %v", err), tcell.ColorRed)
		return
	}

	// Clear input field
	sv.inputField.SetText("")

	// Echo input to output view
	sv.outputView.Write([]byte(fmt.Sprintf("[yellow]> %s[white]\n", input)))
}

// createNewSession creates a new session
func (sv *SessionView) createNewSession() {
	// Simple implementation - in a real app, you'd show a dialog
	sessionName := fmt.Sprintf("session-%d", time.Now().Unix())
	
	session, err := sv.manager.CreateSession(sessionName, nil)
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to create session: %v", err), tcell.ColorRed)
		return
	}

	sv.refreshSessionList()
	sv.showMessage(fmt.Sprintf("Created session: %s", session.GetName()), tcell.ColorGreen)
}

// startSelectedSession starts the selected session
func (sv *SessionView) startSelectedSession() {
	if sv.currentSession == nil {
		sv.showMessage("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.StartSession(sv.currentSession.GetID())
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to start session: %v", err), tcell.ColorRed)
		return
	}

	sv.startOutputMonitoring()
	sv.refreshSessionList()
	sv.showMessage("Session started", tcell.ColorGreen)

	if sv.onSessionStart != nil {
		sv.onSessionStart(sv.currentSession.GetID())
	}
}

// terminateSelectedSession terminates the selected session  
func (sv *SessionView) terminateSelectedSession() {
	if sv.currentSession == nil {
		sv.showMessage("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.TerminateSession(sv.currentSession.GetID())
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to terminate session: %v", err), tcell.ColorRed)
		return
	}

	sv.refreshSessionList()
	sv.showMessage("Session terminated", tcell.ColorYellow)

	if sv.onSessionStop != nil {
		sv.onSessionStop(sv.currentSession.GetID())
	}
}

// pauseSelectedSession pauses the selected session
func (sv *SessionView) pauseSelectedSession() {
	if sv.currentSession == nil {
		sv.showMessage("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.PauseSession(sv.currentSession.GetID())
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to pause session: %v", err), tcell.ColorRed)
		return
	}

	sv.refreshSessionList()
	sv.showMessage("Session paused", tcell.ColorYellow)
}

// resumeSelectedSession resumes the selected session
func (sv *SessionView) resumeSelectedSession() {
	if sv.currentSession == nil {
		sv.showMessage("No session selected", tcell.ColorRed)
		return
	}

	err := sv.manager.ResumeSession(sv.currentSession.GetID())
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to resume session: %v", err), tcell.ColorRed)
		return
	}

	sv.refreshSessionList()
	sv.showMessage("Session resumed", tcell.ColorGreen)
}

// deleteSelectedSession deletes the selected session
func (sv *SessionView) deleteSelectedSession() {
	if sv.currentSession == nil {
		sv.showMessage("No session selected", tcell.ColorRed)
		return
	}

	// Simple confirmation - in a real app, you'd show a proper dialog
	err := sv.manager.DeleteSession(sv.currentSession.GetID(), true)
	if err != nil {
		sv.showMessage(fmt.Sprintf("Failed to delete session: %v", err), tcell.ColorRed)
		return
	}

	sv.currentSession = nil
	sv.outputChannel = nil
	sv.refreshSessionList()
	sv.updateSessionDetails()
	sv.showMessage("Session deleted", tcell.ColorRed)
}

// updateStatusBar updates the status bar
func (sv *SessionView) updateStatusBar() {
	stats := sv.manager.GetStats()
	
	status := fmt.Sprintf(" Sessions: %d | Active: %d | Paused: %d | Terminated: %d | Errors: %d | [yellow]F5[white] Refresh | [yellow]n[white] New | [yellow]s[white] Start | [yellow]t[white] Stop | [yellow]p[white] Pause | [yellow]r[white] Resume | [yellow]d[white] Delete",
		stats.TotalSessions,
		stats.ActiveSessions,
		stats.PausedSessions,
		stats.TerminatedSessions,
		stats.ErrorSessions)
	
	sv.statusBar.SetText(status)
}

// showMessage displays a temporary message
func (sv *SessionView) showMessage(message string, color tcell.Color) {
	// In a real implementation, you'd show this as a modal or notification
	sv.logger.Info(message)
	
	// Update status bar temporarily
	originalText := sv.statusBar.GetText(false)
	sv.statusBar.SetText(fmt.Sprintf("[%s]%s[white]", sv.colorToString(color), message))
	
	// Restore original status after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		sv.statusBar.SetText(originalText)
	}()
}

// colorToString converts tcell.Color to string
func (sv *SessionView) colorToString(color tcell.Color) string {
	switch color {
	case tcell.ColorRed:
		return "red"
	case tcell.ColorGreen:
		return "green"
	case tcell.ColorYellow:
		return "yellow"
	case tcell.ColorBlue:
		return "blue"
	default:
		return "white"
	}
}

// formatDuration formats a duration for display
func (sv *SessionView) formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	
	return fmt.Sprintf("%.1fh", d.Hours())
}

// formatBytes formats bytes for display
func (sv *SessionView) formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatTime formats a time for display
func (sv *SessionView) formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	
	if time.Since(t) < time.Hour*24 {
		return t.Format("15:04:05")
	}
	
	return t.Format("Jan 2 15:04")
}

// SetOnSessionSelect sets the session selection callback
func (sv *SessionView) SetOnSessionSelect(callback func(session *claude.Session)) {
	sv.onSessionSelect = callback
}

// SetOnSessionStart sets the session start callback
func (sv *SessionView) SetOnSessionStart(callback func(sessionID string)) {
	sv.onSessionStart = callback
}

// SetOnSessionStop sets the session stop callback
func (sv *SessionView) SetOnSessionStop(callback func(sessionID string)) {
	sv.onSessionStop = callback
}

// GetCurrentSession returns the currently selected session
func (sv *SessionView) GetCurrentSession() *claude.Session {
	return sv.currentSession
}

// Focus sets focus to the session view
func (sv *SessionView) Focus(delegate func(p tview.Primitive)) {
	delegate(sv.sessionList)
}

// Close cleans up the session view
func (sv *SessionView) Close() {
	if sv.refreshTicker != nil {
		sv.refreshTicker.Stop()
	}
}