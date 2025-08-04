package components

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"go.uber.org/zap"

	"openai-cli/internal/telemetry"
)

// TelemetryDashboard provides a comprehensive UI for viewing telemetry data and system metrics
type TelemetryDashboard struct {
	*tview.Flex
	telemetryClient telemetry.Client
	logger          *zap.Logger

	// UI components
	metricsView     *tview.TextView
	chartsView      *tview.TextView
	systemView      *tview.TextView
	logsView        *tview.Table
	controlsView    *tview.TextView
	statusBar       *tview.TextView
	helpText        *tview.TextView

	// State
	refreshInterval time.Duration
	refreshTicker   *time.Ticker
	ctx             context.Context
	cancel          context.CancelFunc
	currentView     string
	showHelp        bool

	// Data
	metrics        *TelemetryMetrics
	systemHealth   *SystemHealth
	recentLogs     []telemetry.LogEntry
	timeRange      string
	maxLogEntries  int
}

// TelemetryMetrics holds aggregated telemetry metrics
type TelemetryMetrics struct {
	APICallsTotal    int64
	APICallsSuccess  int64
	APICallsFailed   int64
	AvgResponseTime  time.Duration
	ErrorRate        float64
	RequestsPerHour  []int
	ErrorsPerHour    []int
	TopEndpoints     map[string]int64
	TopErrors        map[string]int64
	LastUpdated      time.Time
}

// SystemHealth holds system health indicators
type SystemHealth struct {
	Status          string
	CPUUsage        float64
	MemoryUsage     float64
	DiskUsage       float64
	NetworkLatency  time.Duration
	ActiveSessions  int
	QueuedJobs      int
	LastHealthCheck time.Time
}

// TelemetryDashboardConfig holds configuration for the dashboard
type TelemetryDashboardConfig struct {
	RefreshInterval time.Duration
	MaxLogEntries   int
	DefaultTimeRange string
	ChartWidth      int
	ChartHeight     int
}

// DefaultTelemetryDashboardConfig returns default configuration
func DefaultTelemetryDashboardConfig() *TelemetryDashboardConfig {
	return &TelemetryDashboardConfig{
		RefreshInterval:  10 * time.Second,
		MaxLogEntries:    50,
		DefaultTimeRange: "1h",
		ChartWidth:       60,
		ChartHeight:      10,
	}
}

// NewTelemetryDashboard creates a new telemetry dashboard
func NewTelemetryDashboard(telemetryClient telemetry.Client, logger *zap.Logger) *TelemetryDashboard {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultTelemetryDashboardConfig()

	dashboard := &TelemetryDashboard{
		Flex:            tview.NewFlex(),
		telemetryClient: telemetryClient,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		currentView:     "overview",
		refreshInterval: config.RefreshInterval,
		timeRange:       config.DefaultTimeRange,
		maxLogEntries:   config.MaxLogEntries,
		metrics:         &TelemetryMetrics{},
		systemHealth:    &SystemHealth{},
	}

	dashboard.setupUI()
	dashboard.refreshData()
	dashboard.startAutoRefresh()

	return dashboard
}

// setupUI initializes the UI components
func (td *TelemetryDashboard) setupUI() {
	td.SetDirection(tview.FlexRow)

	// Create controls view
	td.controlsView = tview.NewTextView()
	td.controlsView.SetBorder(true).SetTitle(" Controls ")
	td.controlsView.SetDynamicColors(true)
	td.updateControlsView()

	// Create metrics view
	td.metricsView = tview.NewTextView()
	td.metricsView.SetBorder(true).SetTitle(" API Metrics ")
	td.metricsView.SetScrollable(true)
	td.metricsView.SetDynamicColors(true)

	// Create charts view
	td.chartsView = tview.NewTextView()
	td.chartsView.SetBorder(true).SetTitle(" Performance Charts ")
	td.chartsView.SetScrollable(true)
	td.chartsView.SetDynamicColors(true)

	// Create system view
	td.systemView = tview.NewTextView()
	td.systemView.SetBorder(true).SetTitle(" System Health ")
	td.systemView.SetScrollable(true)
	td.systemView.SetDynamicColors(true)

	// Create logs view
	td.logsView = tview.NewTable()
	td.logsView.SetBorder(true).SetTitle(" Recent Logs ")
	td.logsView.SetSelectable(true, false)
	td.logsView.SetFixed(1, 0)

	// Set log table headers
	headers := []string{"Time", "Level", "Component", "Message", "Endpoint"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		td.logsView.SetCell(0, i, cell)
	}

	// Create help text
	td.helpText = tview.NewTextView()
	td.helpText.SetBorder(true).SetTitle(" Commands ")
	td.helpText.SetDynamicColors(true)
	td.helpText.SetText(`[yellow]Navigation:[white]
  1 - Overview metrics
  2 - Performance charts
  3 - System health
  4 - Recent logs
  Tab - Switch panels
  
[yellow]Time Range:[white]
  Shift+1 - Last hour
  Shift+2 - Last 6 hours
  Shift+3 - Last day
  Shift+4 - Last week
  
[yellow]Actions:[white]
  r - Refresh data
  e - Export metrics
  c - Clear logs
  f - Filter logs
  
[yellow]Display:[white]
  h - Toggle help
  + - Increase refresh rate
  - - Decrease refresh rate
  
[yellow]Other:[white]
  q - Quit to menu`)

	// Create status bar
	td.statusBar = tview.NewTextView()
	td.statusBar.SetDynamicColors(true)

	// Create main layout based on current view
	td.updateLayout()

	// Set up input capture
	td.SetInputCapture(td.handleInput)
}

// updateLayout updates the main layout based on current view
func (td *TelemetryDashboard) updateLayout() {
	td.Clear()

	// Add controls
	td.AddItem(td.controlsView, 3, 0, false)

	switch td.currentView {
	case "overview":
		// Two-column layout: metrics and system health
		mainContent := tview.NewFlex().SetDirection(tview.FlexColumn)
		mainContent.AddItem(td.metricsView, 0, 2, true)
		mainContent.AddItem(td.systemView, 0, 1, false)
		
		bottomPanel := tview.NewFlex().SetDirection(tview.FlexColumn)
		bottomPanel.AddItem(td.logsView, 0, 2, false)
		if td.showHelp {
			bottomPanel.AddItem(td.helpText, 35, 0, false)
		}

		content := tview.NewFlex().SetDirection(tview.FlexRow)
		content.AddItem(mainContent, 0, 2, true)
		content.AddItem(bottomPanel, 0, 1, false)
		
		td.AddItem(content, 0, 1, true)

	case "charts":
		// Full-width charts
		content := tview.NewFlex().SetDirection(tview.FlexRow)
		content.AddItem(td.chartsView, 0, 1, true)
		if td.showHelp {
			content.AddItem(td.helpText, 15, 0, false)
		}
		td.AddItem(content, 0, 1, true)

	case "system":
		// System health focus
		content := tview.NewFlex().SetDirection(tview.FlexColumn)
		content.AddItem(td.systemView, 0, 2, true)
		if td.showHelp {
			content.AddItem(td.helpText, 35, 0, false)
		}
		td.AddItem(content, 0, 1, true)

	case "logs":
		// Logs focus
		content := tview.NewFlex().SetDirection(tview.FlexRow)
		content.AddItem(td.logsView, 0, 1, true)
		if td.showHelp {
			content.AddItem(td.helpText, 15, 0, false)
		}
		td.AddItem(content, 0, 1, true)
	}

	// Add status bar
	td.AddItem(td.statusBar, 1, 0, false)
}

// updateControlsView updates the controls display
func (td *TelemetryDashboard) updateControlsView() {
	controls := fmt.Sprintf("[yellow]View:[white] %s | [yellow]Time Range:[white] %s | [yellow]Refresh:[white] %v | [yellow]Status:[white] %s",
		strings.Title(td.currentView),
		td.timeRange,
		td.refreshInterval,
		td.getConnectionStatus())

	td.controlsView.SetText(controls)
}

// getConnectionStatus returns the telemetry connection status
func (td *TelemetryDashboard) getConnectionStatus() string {
	if td.telemetryClient == nil || !td.telemetryClient.IsEnabled() {
		return "[red]Disabled[white]"
	}
	return "[green]Connected[white]"
}

// handleInput handles keyboard input
func (td *TelemetryDashboard) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case '1':
		if event.Modifiers()&tcell.ModShift != 0 {
			td.setTimeRange("1h")
		} else {
			td.setCurrentView("overview")
		}
		return nil
	case '2':
		if event.Modifiers()&tcell.ModShift != 0 {
			td.setTimeRange("6h")
		} else {
			td.setCurrentView("charts")
		}
		return nil
	case '3':
		if event.Modifiers()&tcell.ModShift != 0 {
			td.setTimeRange("24h")
		} else {
			td.setCurrentView("system")
		}
		return nil
	case '4':
		if event.Modifiers()&tcell.ModShift != 0 {
			td.setTimeRange("7d")
		} else {
			td.setCurrentView("logs")
		}
		return nil
	case 'r', 'R':
		td.refreshData()
		return nil
	case 'e', 'E':
		td.exportMetrics()
		return nil
	case 'c', 'C':
		td.clearLogs()
		return nil
	case 'f', 'F':
		td.filterLogs()
		return nil
	case 'h', 'H':
		td.toggleHelp()
		return nil
	case '+':
		td.increaseRefreshRate()
		return nil
	case '-':
		td.decreaseRefreshRate()
		return nil
	case 'q', 'Q':
		// Let parent handle quit
		return event
	}

	switch event.Key() {
	case tcell.KeyF5:
		td.refreshData()
		return nil
	case tcell.KeyTab:
		return event
	}

	return event
}

// View switching methods
func (td *TelemetryDashboard) setCurrentView(view string) {
	if td.currentView != view {
		td.currentView = view
		td.updateLayout()
		td.updateControlsView()
		td.showStatus(fmt.Sprintf("Switched to %s view", view), tcell.ColorGreen)
	}
}

func (td *TelemetryDashboard) setTimeRange(timeRange string) {
	if td.timeRange != timeRange {
		td.timeRange = timeRange
		td.updateControlsView()
		td.refreshData()
		td.showStatus(fmt.Sprintf("Time range set to %s", timeRange), tcell.ColorGreen)
	}
}

// refreshData refreshes all dashboard data with timeout protection
func (td *TelemetryDashboard) refreshData() {
	td.logger.Debug("Refreshing telemetry dashboard data")
	
	// Protect against data refresh blocking UI
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	refreshDone := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				td.logger.Error("panic in refreshData", zap.Any("panic", r))
				refreshDone <- false
				return
			}
			refreshDone <- true
		}()
		
		// Update data with individual timeouts
		td.updateMetricsSafe()
		td.updateSystemHealthSafe()
		td.updateRecentLogsSafe()
		
		// Update views with individual timeouts
		td.updateMetricsViewSafe()
		td.updateChartsViewSafe()
		td.updateSystemViewSafe()
		td.updateLogsViewSafe()
		td.updateStatusBar()
	}()
	
	select {
	case success := <-refreshDone:
		if success {
			// Log telemetry only if refresh was successful
			if td.telemetryClient != nil && td.telemetryClient.IsEnabled() {
				telemetry.LogUserAction(td.telemetryClient, "telemetry_dashboard_refresh", map[string]interface{}{
					"view": td.currentView,
					"time_range": td.timeRange,
				})
			}
		} else {
			td.showStatus("Data refresh failed", tcell.ColorRed)
		}
	case <-ctx.Done():
		td.logger.Warn("Dashboard refresh timed out")
		td.showStatus("Dashboard refresh timed out", tcell.ColorRed)
	}
}

// updateMetrics simulates updating API metrics (in real implementation, this would query actual data)
func (td *TelemetryDashboard) updateMetrics() {
	// Simulate some metrics - in real implementation, this would query telemetry storage
	now := time.Now()
	
	td.metrics = &TelemetryMetrics{
		APICallsTotal:   1250,
		APICallsSuccess: 1190,
		APICallsFailed:  60,
		AvgResponseTime: 450 * time.Millisecond,
		ErrorRate:       4.8,
		LastUpdated:     now,
		TopEndpoints: map[string]int64{
			"/v1/chat/completions": 800,
			"/v1/completions":      250,
			"/v1/embeddings":       150,
			"/agent/chat":          50,
		},
		TopErrors: map[string]int64{
			"rate_limit_exceeded":  25,
			"invalid_api_key":      15,
			"model_not_found":      10,
			"timeout":              8,
			"server_error":         2,
		},
	}

	// Generate hourly data for charts (simulate)
	td.metrics.RequestsPerHour = make([]int, 24)
	td.metrics.ErrorsPerHour = make([]int, 24)
	for i := 0; i < 24; i++ {
		td.metrics.RequestsPerHour[i] = 30 + (i*2) + (int(now.Hour())+i)%10
		td.metrics.ErrorsPerHour[i] = int(float64(td.metrics.RequestsPerHour[i]) * td.metrics.ErrorRate / 100)
	}
}

// updateSystemHealth simulates updating system health (in real implementation, this would query actual system metrics)
func (td *TelemetryDashboard) updateSystemHealth() {
	td.systemHealth = &SystemHealth{
		Status:          "Healthy",
		CPUUsage:        23.5,
		MemoryUsage:     67.2,
		DiskUsage:       45.8,
		NetworkLatency:  12 * time.Millisecond,
		ActiveSessions:  8,
		QueuedJobs:      2,
		LastHealthCheck: time.Now(),
	}
}

// updateRecentLogs simulates updating recent logs
func (td *TelemetryDashboard) updateRecentLogs() {
	// Simulate recent log entries
	now := time.Now()
	td.recentLogs = []telemetry.LogEntry{
		{
			Level:     telemetry.LogLevelInfo,
			Message:   "Chat completion request processed successfully",
			Component: "api",
			Endpoint:  "/v1/chat/completions",
			Timestamp: now.Add(-2 * time.Minute),
		},
		{
			Level:     telemetry.LogLevelWarn,
			Message:   "Rate limit approaching for user",
			Component: "auth",
			Endpoint:  "/v1/chat/completions",
			Timestamp: now.Add(-5 * time.Minute),
		},
		{
			Level:     telemetry.LogLevelError,
			Message:   "Failed to connect to model backend",
			Component: "provider",
			Endpoint:  "/v1/completions",
			Timestamp: now.Add(-8 * time.Minute),
		},
		{
			Level:     telemetry.LogLevelInfo,
			Message:   "Session created successfully",
			Component: "session",
			Endpoint:  "/agent/chat",
			Timestamp: now.Add(-10 * time.Minute),
		},
		{
			Level:     telemetry.LogLevelDebug,
			Message:   "Telemetry batch sent successfully",
			Component: "telemetry",
			Endpoint:  "/api/telemetry/logs",
			Timestamp: now.Add(-12 * time.Minute),
		},
	}
}

// updateMetricsView updates the metrics display
func (td *TelemetryDashboard) updateMetricsView() {
	if td.metrics == nil {
		return
	}

	successRate := float64(td.metrics.APICallsSuccess) / float64(td.metrics.APICallsTotal) * 100

	content := fmt.Sprintf(`[yellow]API Performance[white]

[yellow]Total Requests:[white] %d
[yellow]Successful:[white] %d (%.1f%%)
[yellow]Failed:[white] %d (%.1f%%)
[yellow]Average Response Time:[white] %v
[yellow]Error Rate:[white] %.1f%%

[yellow]Top Endpoints[white]`,
		td.metrics.APICallsTotal,
		td.metrics.APICallsSuccess, successRate,
		td.metrics.APICallsFailed, td.metrics.ErrorRate,
		td.metrics.AvgResponseTime,
		td.metrics.ErrorRate)

	// Sort endpoints by count
	type endpointCount struct {
		endpoint string
		count    int64
	}
	var endpoints []endpointCount
	for endpoint, count := range td.metrics.TopEndpoints {
		endpoints = append(endpoints, endpointCount{endpoint, count})
	}
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].count > endpoints[j].count
	})

	for _, ep := range endpoints {
		percentage := float64(ep.count) / float64(td.metrics.APICallsTotal) * 100
		content += fmt.Sprintf("\n  %s: %d (%.1f%%)", ep.endpoint, ep.count, percentage)
	}

	content += "\n\n[yellow]Top Errors[white]"
	
	// Sort errors by count
	type errorCount struct {
		error string
		count int64
	}
	var errors []errorCount
	for error, count := range td.metrics.TopErrors {
		errors = append(errors, errorCount{error, count})
	}
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})

	for _, err := range errors {
		content += fmt.Sprintf("\n  %s: %d", err.error, err.count)
	}

	content += fmt.Sprintf("\n\n[gray]Last Updated: %s[white]", td.metrics.LastUpdated.Format("15:04:05"))

	td.metricsView.SetText(content)
}

// updateChartsView updates the charts display with ASCII charts
func (td *TelemetryDashboard) updateChartsView() {
	if td.metrics == nil {
		return
	}

	content := "[yellow]Request Volume (Last 24 Hours)[white]\n\n"
	content += td.generateASCIIChart("Requests", td.metrics.RequestsPerHour, 50, 10)
	
	content += "\n\n[yellow]Error Count (Last 24 Hours)[white]\n\n"
	content += td.generateASCIIChart("Errors", td.metrics.ErrorsPerHour, 10, 8)

	// Response time trend (simulated)
	responseTimes := make([]int, 24)
	base := 400
	for i := 0; i < 24; i++ {
		variation := int(math.Sin(float64(i)*0.3) * 100)
		responseTimes[i] = base + variation + (i%6)*20
	}
	
	content += "\n\n[yellow]Response Time Trend (ms)[white]\n\n"
	content += td.generateASCIIChart("Response Time", responseTimes, 600, 10)

	td.chartsView.SetText(content)
}

// generateASCIIChart generates a simple ASCII bar chart
func (td *TelemetryDashboard) generateASCIIChart(title string, data []int, maxValue int, height int) string {
	if len(data) == 0 {
		return "No data available"
	}

	// Find actual max value in data
	actualMax := 0
	for _, v := range data {
		if v > actualMax {
			actualMax = v
		}
	}
	if actualMax == 0 {
		actualMax = 1
	}

	var chart strings.Builder
	
	// Generate chart from top to bottom
	for row := height; row > 0; row-- {
		threshold := float64(actualMax) * float64(row) / float64(height)
		
		for _, value := range data {
			if float64(value) >= threshold {
				chart.WriteString("▓")
			} else if float64(value) >= threshold-float64(actualMax)/float64(height*2) {
				chart.WriteString("▒")
			} else {
				chart.WriteString(" ")
			}
		}
		chart.WriteString(fmt.Sprintf(" %d\n", int(threshold)))
	}

	// Add X-axis
	chart.WriteString(strings.Repeat("▁", len(data)))
	chart.WriteString(" 0\n")

	// Add time labels (every 6 hours)
	labels := ""
	for i := 0; i < len(data); i++ {
		if i%6 == 0 {
			labels += fmt.Sprintf("%-6s", fmt.Sprintf("%02d:00", i))
		}
	}
	chart.WriteString(labels + "\n")

	return chart.String()
}

// updateSystemView updates the system health display
func (td *TelemetryDashboard) updateSystemView() {
	if td.systemHealth == nil {
		return
	}

	statusColor := "[green]"
	if td.systemHealth.Status != "Healthy" {
		statusColor = "[red]"
	}

	content := fmt.Sprintf(`[yellow]System Status[white]

[yellow]Overall Status:[white] %s%s[white]
[yellow]Last Check:[white] %s

[yellow]Resource Usage[white]

CPU Usage: %s
%s

Memory Usage: %s
%s

Disk Usage: %s
%s

[yellow]Network & Performance[white]

[yellow]Network Latency:[white] %v
[yellow]Active Sessions:[white] %d
[yellow]Queued Jobs:[white] %d`,
		statusColor, td.systemHealth.Status,
		td.systemHealth.LastHealthCheck.Format("15:04:05"),
		td.formatPercentage(td.systemHealth.CPUUsage),
		td.generateProgressBar(td.systemHealth.CPUUsage, 100),
		td.formatPercentage(td.systemHealth.MemoryUsage),
		td.generateProgressBar(td.systemHealth.MemoryUsage, 100),
		td.formatPercentage(td.systemHealth.DiskUsage),
		td.generateProgressBar(td.systemHealth.DiskUsage, 100),
		td.systemHealth.NetworkLatency,
		td.systemHealth.ActiveSessions,
		td.systemHealth.QueuedJobs)

	// Add alerts if any
	alerts := td.generateSystemAlerts()
	if len(alerts) > 0 {
		content += "\n\n[yellow]Alerts[white]\n"
		for _, alert := range alerts {
			content += fmt.Sprintf("[%s]● %s[white]\n", alert.Color, alert.Message)
		}
	}

	td.systemView.SetText(content)
}

// SystemAlert represents a system alert
type SystemAlert struct {
	Message string
	Color   string
}

// generateSystemAlerts generates system alerts based on current health
func (td *TelemetryDashboard) generateSystemAlerts() []SystemAlert {
	var alerts []SystemAlert

	if td.systemHealth.CPUUsage > 80 {
		alerts = append(alerts, SystemAlert{
			Message: fmt.Sprintf("High CPU usage: %.1f%%", td.systemHealth.CPUUsage),
			Color:   "red",
		})
	}

	if td.systemHealth.MemoryUsage > 85 {
		alerts = append(alerts, SystemAlert{
			Message: fmt.Sprintf("High memory usage: %.1f%%", td.systemHealth.MemoryUsage),
			Color:   "red",
		})
	}

	if td.systemHealth.DiskUsage > 90 {
		alerts = append(alerts, SystemAlert{
			Message: fmt.Sprintf("High disk usage: %.1f%%", td.systemHealth.DiskUsage),
			Color:   "red",
		})
	}

	if td.systemHealth.NetworkLatency > 100*time.Millisecond {
		alerts = append(alerts, SystemAlert{
			Message: fmt.Sprintf("High network latency: %v", td.systemHealth.NetworkLatency),
			Color:   "yellow",
		})
	}

	if td.metrics != nil && td.metrics.ErrorRate > 10 {
		alerts = append(alerts, SystemAlert{
			Message: fmt.Sprintf("High error rate: %.1f%%", td.metrics.ErrorRate),
			Color:   "red",
		})
	}

	return alerts
}

// formatPercentage formats a percentage value
func (td *TelemetryDashboard) formatPercentage(value float64) string {
	color := "[green]"
	if value > 80 {
		color = "[red]"
	} else if value > 60 {
		color = "[yellow]"
	}
	return fmt.Sprintf("%s%.1f%%[white]", color, value)
}

// generateProgressBar generates a text-based progress bar
func (td *TelemetryDashboard) generateProgressBar(value, max float64) string {
	width := 30
	filled := int((value / max) * float64(width))
	
	bar := "["
	for i := 0; i < width; i++ {
		if i < filled {
			if value > 80 {
				bar += "[red]█"
			} else if value > 60 {
				bar += "[yellow]█"
			} else {
				bar += "[green]█"
			}
		} else {
			bar += "[gray]░"
		}
	}
	bar += "[white]]"
	
	return bar
}

// updateLogsView updates the logs table
func (td *TelemetryDashboard) updateLogsView() {
	// Clear existing rows (except header)
	rowCount := td.logsView.GetRowCount()
	for row := 1; row < rowCount; row++ {
		td.logsView.RemoveRow(row)
	}

	// Add recent logs
	for i, logEntry := range td.recentLogs {
		if i >= td.maxLogEntries {
			break
		}
		
		row := i + 1
		
		// Time
		td.logsView.SetCell(row, 0, tview.NewTableCell(logEntry.Timestamp.Format("15:04:05")))
		
		// Level with color
		levelColor := td.getLogLevelColor(logEntry.Level)
		levelCell := tview.NewTableCell(string(logEntry.Level)).SetTextColor(levelColor)
		td.logsView.SetCell(row, 1, levelCell)
		
		// Component
		td.logsView.SetCell(row, 2, tview.NewTableCell(logEntry.Component))
		
		// Message (truncated)
		message := logEntry.Message
		if len(message) > 50 {
			message = message[:47] + "..."
		}
		td.logsView.SetCell(row, 3, tview.NewTableCell(message))
		
		// Endpoint
		endpoint := logEntry.Endpoint
		if len(endpoint) > 25 {
			endpoint = endpoint[:22] + "..."
		}
		td.logsView.SetCell(row, 4, tview.NewTableCell(endpoint))
	}
}

// getLogLevelColor returns appropriate color for log level
func (td *TelemetryDashboard) getLogLevelColor(level telemetry.LogLevel) tcell.Color {
	switch level {
	case telemetry.LogLevelError:
		return tcell.ColorRed
	case telemetry.LogLevelWarn:
		return tcell.ColorYellow
	case telemetry.LogLevelInfo:
		return tcell.ColorGreen
	case telemetry.LogLevelDebug:
		return tcell.ColorGray
	default:
		return tcell.ColorWhite
	}
}

// Action methods
func (td *TelemetryDashboard) exportMetrics() {
	td.showStatus("Preparing export...", tcell.ColorYellow)
	
	// Simulate export with timeout protection
	exportDone := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				td.logger.Error("panic in exportMetrics", zap.Any("panic", r))
				exportDone <- fmt.Errorf("export failed: %v", r)
			}
		}()
		
		// Simulate export preparation
		time.Sleep(1 * time.Second)
		exportDone <- nil
	}()
	
	go func() {
		select {
		case err := <-exportDone:
			if err != nil {
				td.showStatus(fmt.Sprintf("Export failed: %v", err), tcell.ColorRed)
			} else {
				td.showStatus("Export feature coming soon - metrics data prepared", tcell.ColorGreen)
			}
		case <-time.After(10 * time.Second):
			td.showStatus("Export operation timed out", tcell.ColorRed)
		}
	}()
}

func (td *TelemetryDashboard) clearLogs() {
	td.recentLogs = []telemetry.LogEntry{}
	td.updateLogsView()
	td.showStatus("Logs cleared", tcell.ColorGreen)
}

func (td *TelemetryDashboard) filterLogs() {
	td.showStatus("Preparing log filter...", tcell.ColorYellow)
	
	// Simulate filter with timeout protection
	filterDone := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				td.logger.Error("panic in filterLogs", zap.Any("panic", r))
				filterDone <- false
			}
		}()
		
		// Simulate filter preparation
		time.Sleep(500 * time.Millisecond)
		filterDone <- true
	}()
	
	go func() {
		select {
		case success := <-filterDone:
			if success {
				td.showStatus("Log filtering feature coming soon", tcell.ColorGreen)
			} else {
				td.showStatus("Filter preparation failed", tcell.ColorRed)
			}
		case <-time.After(5 * time.Second):
			td.showStatus("Filter operation timed out", tcell.ColorRed)
		}
	}()
}

func (td *TelemetryDashboard) toggleHelp() {
	td.showHelp = !td.showHelp
	td.updateLayout()
	if td.showHelp {
		td.showStatus("Help shown", tcell.ColorGreen)
	} else {
		td.showStatus("Help hidden", tcell.ColorGray)
	}
}

func (td *TelemetryDashboard) increaseRefreshRate() {
	if td.refreshInterval > 5*time.Second {
		td.refreshInterval -= 5*time.Second
		td.restartAutoRefresh()
		td.updateControlsView()
		td.showStatus(fmt.Sprintf("Refresh rate increased to %v", td.refreshInterval), tcell.ColorGreen)
	}
}

func (td *TelemetryDashboard) decreaseRefreshRate() {
	if td.refreshInterval < 60*time.Second {
		td.refreshInterval += 5*time.Second
		td.restartAutoRefresh()
		td.updateControlsView()
		td.showStatus(fmt.Sprintf("Refresh rate decreased to %v", td.refreshInterval), tcell.ColorYellow)
	}
}

// updateStatusBar updates the status bar
func (td *TelemetryDashboard) updateStatusBar() {
	status := fmt.Sprintf(" View: %s | Time Range: %s | Refresh: %v | Last Update: %s",
		strings.Title(td.currentView),
		td.timeRange,
		td.refreshInterval,
		time.Now().Format("15:04:05"))

	if td.telemetryClient != nil && td.telemetryClient.IsEnabled() {
		status += " | [green]Connected[white]"
	} else {
		status += " | [red]Disconnected[white]"
	}

	td.statusBar.SetText(status)
}

// showStatus shows a temporary status message
func (td *TelemetryDashboard) showStatus(message string, color tcell.Color) {
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

	originalText := td.statusBar.GetText(false)
	td.statusBar.SetText(fmt.Sprintf("[%s]%s[white]", colorName, message))

	// Reset after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		td.statusBar.SetText(originalText)
	}()
}

// startAutoRefresh starts automatic refresh timer
func (td *TelemetryDashboard) startAutoRefresh() {
	td.refreshTicker = time.NewTicker(td.refreshInterval)
	
	go func() {
		for {
			select {
			case <-td.ctx.Done():
				return
			case <-td.refreshTicker.C:
				td.refreshData()
			}
		}
	}()
}

// restartAutoRefresh restarts the refresh timer with new interval
func (td *TelemetryDashboard) restartAutoRefresh() {
	if td.refreshTicker != nil {
		td.refreshTicker.Stop()
	}
	td.startAutoRefresh()
}

// Safe update methods with timeout protection
func (td *TelemetryDashboard) updateMetricsSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		td.updateMetrics()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateMetrics timed out")
	}
}

func (td *TelemetryDashboard) updateSystemHealthSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		td.updateSystemHealth()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateSystemHealth timed out")
	}
}

func (td *TelemetryDashboard) updateRecentLogsSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		td.updateRecentLogs()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateRecentLogs timed out")
	}
}

func (td *TelemetryDashboard) updateMetricsViewSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		td.updateMetricsView()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateMetricsView timed out")
	}
}

func (td *TelemetryDashboard) updateChartsViewSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				td.logger.Error("panic in updateChartsView", zap.Any("panic", r))
			}
			done <- true
		}()
		td.updateChartsView()
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateChartsView timed out")
		// Set fallback content
		td.chartsView.SetText("[yellow]Charts temporarily unavailable[white]\n\nChart rendering timed out. Please try refreshing.")
	}
}

func (td *TelemetryDashboard) updateSystemViewSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		td.updateSystemView()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateSystemView timed out")
	}
}

func (td *TelemetryDashboard) updateLogsViewSafe() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		td.updateLogsView()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-ctx.Done():
		td.logger.Warn("updateLogsView timed out")
	}
}

// Close cleans up resources
func (td *TelemetryDashboard) Close() {
	if td.refreshTicker != nil {
		td.refreshTicker.Stop()
	}
	td.cancel()
}