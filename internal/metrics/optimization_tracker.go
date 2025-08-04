package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"openai-cli/internal/mcp"
	"openai-cli/internal/telemetry"
	"go.uber.org/zap"
)

// OptimizationMetrics tracks detailed optimization performance
type OptimizationMetrics struct {
	// Token metrics
	TotalOriginalTokens   int64   `json:"total_original_tokens"`
	TotalOptimizedTokens  int64   `json:"total_optimized_tokens"`
	TotalTokensSaved      int64   `json:"total_tokens_saved"`
	AverageCompressionRatio float64 `json:"average_compression_ratio"`
	
	// Performance metrics
	TotalOptimizations    int64         `json:"total_optimizations"`
	SuccessfulOptimizations int64       `json:"successful_optimizations"`
	FailedOptimizations   int64         `json:"failed_optimizations"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	
	// Subagent metrics
	SubagentUsage         map[string]int64 `json:"subagent_usage"`
	SubagentSuccessRates  map[string]float64 `json:"subagent_success_rates"`
	SubagentAvgTimes      map[string]time.Duration `json:"subagent_avg_times"`
	
	// Cost efficiency metrics
	EstimatedCostSavings  float64 `json:"estimated_cost_savings"`
	TokenCostPerK         float64 `json:"token_cost_per_k"`
	
	// Time tracking
	FirstOptimization     time.Time `json:"first_optimization"`
	LastOptimization      time.Time `json:"last_optimization"`
	
	// User experience metrics
	UserSatisfactionScore float64 `json:"user_satisfaction_score"`
	OptimizationAcceptanceRate float64 `json:"optimization_acceptance_rate"`
}

// OptimizationTracker handles metrics collection and reporting
type OptimizationTracker struct {
	metrics         OptimizationMetrics
	telemetryClient telemetry.Client
	logger          *telemetry.TelemetryLogger
	mu              sync.RWMutex
	
	// Configuration
	reportInterval    time.Duration
	batchSize         int
	costPerToken      float64
	
	// Internal state
	pendingEvents     []OptimizationEvent
	eventBuffer       chan OptimizationEvent
	ctx               context.Context
	cancel            context.CancelFunc
}

// OptimizationEvent represents a single optimization operation
type OptimizationEvent struct {
	Timestamp        time.Time              `json:"timestamp"`
	EventType        string                 `json:"event_type"` // "optimization", "subagent_call", "error"
	OriginalTokens   int                    `json:"original_tokens"`
	OptimizedTokens  int                    `json:"optimized_tokens"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	SubagentID       string                 `json:"subagent_id,omitempty"`
	Success          bool                   `json:"success"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	CompressionRatio float64                `json:"compression_ratio"`
	TaskType         string                 `json:"task_type,omitempty"`
	UserID           string                 `json:"user_id,omitempty"`
	SessionID        string                 `json:"session_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NewOptimizationTracker creates a new optimization metrics tracker
func NewOptimizationTracker(telemetryClient telemetry.Client, logger *telemetry.TelemetryLogger) *OptimizationTracker {
	ctx, cancel := context.WithCancel(context.Background())
	
	tracker := &OptimizationTracker{
		metrics: OptimizationMetrics{
			SubagentUsage:        make(map[string]int64),
			SubagentSuccessRates: make(map[string]float64),
			SubagentAvgTimes:     make(map[string]time.Duration),
		},
		telemetryClient: telemetryClient,
		logger:          logger,
		reportInterval:  5 * time.Minute,
		batchSize:       100,
		costPerToken:    0.0015, // Default cost per 1K tokens (GPT-3.5-turbo pricing)
		eventBuffer:     make(chan OptimizationEvent, 1000),
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Start background processing
	go tracker.processEvents()
	go tracker.periodicReporting()
	
	return tracker
}

// TrackOptimization records an optimization operation
func (ot *OptimizationTracker) TrackOptimization(result *mcp.OptimizationResult, processingTime time.Duration, userID, sessionID string) {
	event := OptimizationEvent{
		Timestamp:        time.Now(),
		EventType:        "optimization",
		OriginalTokens:   result.OriginalTokens,
		OptimizedTokens:  result.OptimizedTokens,
		ProcessingTime:   processingTime,
		Success:          result.Success,
		CompressionRatio: result.CompressionRatio,
		UserID:           userID,
		SessionID:        sessionID,
		Metadata: map[string]interface{}{
			"tokens_saved":    result.TokensSaved,
			"subagent_usage":  result.SubagentUsage,
			"strategy":        result.Strategy,
		},
	}
	
	if !result.Success {
		event.ErrorMessage = result.Error
	}
	
	select {
	case ot.eventBuffer <- event:
	default:
		if ot.logger != nil {
			ot.logger.Warn("Optimization event buffer full, dropping event")
		}
	}
}

// TrackSubagentCall records a subagent operation
func (ot *OptimizationTracker) TrackSubagentCall(subagentID, taskType string, response *mcp.SubagentResponse, userID, sessionID string) {
	event := OptimizationEvent{
		Timestamp:      time.Now(),
		EventType:      "subagent_call",
		SubagentID:     subagentID,
		TaskType:       taskType,
		ProcessingTime: response.ProcessingTime,
		Success:        response.Success,
		UserID:         userID,
		SessionID:      sessionID,
		Metadata: map[string]interface{}{
			"tokens_processed": response.TokensProcessed,
			"tokens_saved":     response.TokensSaved,
			"confidence":       response.Confidence,
		},
	}
	
	if !response.Success {
		event.ErrorMessage = response.Error
	}
	
	select {
	case ot.eventBuffer <- event:
	default:
		if ot.logger != nil {
			ot.logger.Warn("Subagent event buffer full, dropping event")
		}
	}
}

// TrackError records an optimization error
func (ot *OptimizationTracker) TrackError(errorType, errorMessage string, userID, sessionID string) {
	event := OptimizationEvent{
		Timestamp:    time.Now(),
		EventType:    "error",
		Success:      false,
		ErrorMessage: errorMessage,
		UserID:       userID,
		SessionID:    sessionID,
		Metadata: map[string]interface{}{
			"error_type": errorType,
		},
	}
	
	select {
	case ot.eventBuffer <- event:
	default:
		if ot.logger != nil {
			ot.logger.Warn("Error event buffer full, dropping event")
		}
	}
}

// GetMetrics returns current optimization metrics
func (ot *OptimizationTracker) GetMetrics() OptimizationMetrics {
	ot.mu.RLock()
	defer ot.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := ot.metrics
	
	// Copy maps
	metrics.SubagentUsage = make(map[string]int64)
	metrics.SubagentSuccessRates = make(map[string]float64)
	metrics.SubagentAvgTimes = make(map[string]time.Duration)
	
	for k, v := range ot.metrics.SubagentUsage {
		metrics.SubagentUsage[k] = v
	}
	for k, v := range ot.metrics.SubagentSuccessRates {
		metrics.SubagentSuccessRates[k] = v
	}
	for k, v := range ot.metrics.SubagentAvgTimes {
		metrics.SubagentAvgTimes[k] = v
	}
	
	return metrics
}

// GetSummaryReport generates a human-readable summary report
func (ot *OptimizationTracker) GetSummaryReport() string {
	metrics := ot.GetMetrics()
	
	report := "# Optimization Summary Report\n\n"
	
	// Token efficiency
	report += "## Token Efficiency\n"
	if metrics.TotalOriginalTokens > 0 {
		report += fmt.Sprintf("- Total tokens processed: %d\n", metrics.TotalOriginalTokens)
		report += fmt.Sprintf("- Total tokens after optimization: %d\n", metrics.TotalOptimizedTokens)
		report += fmt.Sprintf("- Total tokens saved: %d (%.1f%%)\n", 
			metrics.TotalTokensSaved,
			float64(metrics.TotalTokensSaved)/float64(metrics.TotalOriginalTokens)*100)
		report += fmt.Sprintf("- Average compression ratio: %.1f%%\n", metrics.AverageCompressionRatio*100)
	}
	
	// Performance
	report += "\n## Performance\n"
	report += fmt.Sprintf("- Total optimizations: %d\n", metrics.TotalOptimizations)
	report += fmt.Sprintf("- Success rate: %.1f%%\n", 
		float64(metrics.SuccessfulOptimizations)/float64(metrics.TotalOptimizations)*100)
	report += fmt.Sprintf("- Average processing time: %v\n", metrics.AverageProcessingTime.Truncate(time.Millisecond))
	
	// Cost savings
	if metrics.EstimatedCostSavings > 0 {
		report += "\n## Cost Efficiency\n"
		report += fmt.Sprintf("- Estimated cost savings: $%.4f\n", metrics.EstimatedCostSavings)
		report += fmt.Sprintf("- Cost per 1K tokens: $%.4f\n", metrics.TokenCostPerK)
	}
	
	// Subagent usage
	if len(metrics.SubagentUsage) > 0 {
		report += "\n## Subagent Usage\n"
		for subagent, count := range metrics.SubagentUsage {
			successRate := metrics.SubagentSuccessRates[subagent] * 100
			avgTime := metrics.SubagentAvgTimes[subagent].Truncate(time.Millisecond)
			report += fmt.Sprintf("- %s: %d calls (%.1f%% success, %v avg)\n", 
				subagent, count, successRate, avgTime)
		}
	}
	
	return report
}

// processEvents handles incoming optimization events
func (ot *OptimizationTracker) processEvents() {
	for {
		select {
		case <-ot.ctx.Done():
			return
		case event := <-ot.eventBuffer:
			ot.processEvent(event)
		}
	}
}

// processEvent updates metrics based on a single event
func (ot *OptimizationTracker) processEvent(event OptimizationEvent) {
	ot.mu.Lock()
	defer ot.mu.Unlock()
	
	// Update timestamps
	if ot.metrics.FirstOptimization.IsZero() {
		ot.metrics.FirstOptimization = event.Timestamp
	}
	ot.metrics.LastOptimization = event.Timestamp
	
	switch event.EventType {
	case "optimization":
		ot.metrics.TotalOptimizations++
		if event.Success {
			ot.metrics.SuccessfulOptimizations++
			ot.metrics.TotalOriginalTokens += int64(event.OriginalTokens)
			ot.metrics.TotalOptimizedTokens += int64(event.OptimizedTokens)
			ot.metrics.TotalTokensSaved += int64(event.OriginalTokens - event.OptimizedTokens)
			
			// Update compression ratio
			if ot.metrics.TotalOriginalTokens > 0 {
				ot.metrics.AverageCompressionRatio = float64(ot.metrics.TotalOptimizedTokens) / 
					float64(ot.metrics.TotalOriginalTokens)
			}
			
			// Update cost savings
			tokensSaved := float64(event.OriginalTokens - event.OptimizedTokens)
			ot.metrics.EstimatedCostSavings += (tokensSaved / 1000.0) * ot.costPerToken
		} else {
			ot.metrics.FailedOptimizations++
		}
		
		// Update processing time
		ot.updateAverageProcessingTime(event.ProcessingTime)
		
	case "subagent_call":
		if event.SubagentID != "" {
			ot.metrics.SubagentUsage[event.SubagentID]++
			ot.updateSubagentMetrics(event.SubagentID, event.Success, event.ProcessingTime)
		}
	}
	
	// Send to telemetry (async)
	go ot.sendEventToTelemetry(event)
}

// updateAverageProcessingTime updates the running average of processing times
func (ot *OptimizationTracker) updateAverageProcessingTime(duration time.Duration) {
	if ot.metrics.TotalOptimizations > 0 {
		totalDuration := int64(ot.metrics.AverageProcessingTime) * int64(ot.metrics.TotalOptimizations-1)
		ot.metrics.AverageProcessingTime = time.Duration((totalDuration + int64(duration)) / int64(ot.metrics.TotalOptimizations))
	} else {
		ot.metrics.AverageProcessingTime = duration
	}
}

// updateSubagentMetrics updates subagent-specific metrics
func (ot *OptimizationTracker) updateSubagentMetrics(subagentID string, success bool, processingTime time.Duration) {
	count := ot.metrics.SubagentUsage[subagentID]
	
	// Update success rate
	currentSuccessRate := ot.metrics.SubagentSuccessRates[subagentID]
	if success {
		ot.metrics.SubagentSuccessRates[subagentID] = (currentSuccessRate*float64(count-1) + 1.0) / float64(count)
	} else {
		ot.metrics.SubagentSuccessRates[subagentID] = (currentSuccessRate * float64(count-1)) / float64(count)
	}
	
	// Update average processing time
	currentAvgTime := ot.metrics.SubagentAvgTimes[subagentID]
	ot.metrics.SubagentAvgTimes[subagentID] = time.Duration(
		(int64(currentAvgTime)*int64(count-1) + int64(processingTime)) / int64(count))
}

// sendEventToTelemetry sends an event to the telemetry system
func (ot *OptimizationTracker) sendEventToTelemetry(event OptimizationEvent) {
	if ot.telemetryClient == nil {
		return
	}
	
	// Convert event to telemetry format
	eventData := map[string]interface{}{
		"event_type":        event.EventType,
		"timestamp":         event.Timestamp,
		"original_tokens":   event.OriginalTokens,
		"optimized_tokens":  event.OptimizedTokens,
		"processing_time_ms": event.ProcessingTime.Milliseconds(),
		"success":           event.Success,
		"compression_ratio": event.CompressionRatio,
	}
	
	if event.SubagentID != "" {
		eventData["subagent_id"] = event.SubagentID
	}
	if event.TaskType != "" {
		eventData["task_type"] = event.TaskType
	}
	if event.ErrorMessage != "" {
		eventData["error_message"] = event.ErrorMessage
	}
	if event.UserID != "" {
		eventData["user_id"] = event.UserID
	}
	if event.SessionID != "" {
		eventData["session_id"] = event.SessionID
	}
	
	// Add metadata
	for k, v := range event.Metadata {
		eventData[k] = v
	}
	
	// Send to telemetry
	telemetry.LogUserAction(ot.telemetryClient, "optimization_event", eventData)
}

// periodicReporting sends periodic summary reports
func (ot *OptimizationTracker) periodicReporting() {
	ticker := time.NewTicker(ot.reportInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ot.ctx.Done():
			return
		case <-ticker.C:
			ot.sendPeriodicReport()
		}
	}
}

// sendPeriodicReport sends a summary report to telemetry
func (ot *OptimizationTracker) sendPeriodicReport() {
	metrics := ot.GetMetrics()
	
	reportData := map[string]interface{}{
		"report_type":               "optimization_summary",
		"timestamp":                 time.Now(),
		"total_optimizations":       metrics.TotalOptimizations,
		"successful_optimizations":  metrics.SuccessfulOptimizations,
		"failed_optimizations":      metrics.FailedOptimizations,
		"total_original_tokens":     metrics.TotalOriginalTokens,
		"total_optimized_tokens":    metrics.TotalOptimizedTokens,
		"total_tokens_saved":        metrics.TotalTokensSaved,
		"average_compression_ratio": metrics.AverageCompressionRatio,
		"average_processing_time_ms": metrics.AverageProcessingTime.Milliseconds(),
		"estimated_cost_savings":    metrics.EstimatedCostSavings,
		"subagent_usage":            metrics.SubagentUsage,
		"subagent_success_rates":    metrics.SubagentSuccessRates,
	}
	
	if ot.telemetryClient != nil {
		telemetry.LogUserAction(ot.telemetryClient, "optimization_summary_report", reportData)
	}
	
	if ot.logger != nil {
		ot.logger.Info("Sent periodic optimization report", zap.Any("data", reportData))
	}
}

// Close shuts down the optimization tracker
func (ot *OptimizationTracker) Close() {
	ot.cancel()
	
	// Send final report
	ot.sendPeriodicReport()
	
	// Close the event buffer channel
	close(ot.eventBuffer)
}

// SetCostPerToken updates the cost per token for cost calculations
func (ot *OptimizationTracker) SetCostPerToken(cost float64) {
	ot.mu.Lock()
	defer ot.mu.Unlock()
	ot.costPerToken = cost
}