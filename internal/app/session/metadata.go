package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"openai-cli/internal/app/claude"
)

// EnhancedSessionMetadata provides comprehensive session tracking
type EnhancedSessionMetadata struct {
	// Basic metadata
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	State       claude.SessionState    `json:"state"`
	Config      *claude.SessionConfig  `json:"config"`
	
	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	TerminatedAt *time.Time `json:"terminated_at,omitempty"`
	
	// Classification and organization
	Tags       []string               `json:"tags"`
	Category   string                 `json:"category,omitempty"`
	Priority   SessionPriority        `json:"priority"`
	Labels     map[string]string      `json:"labels,omitempty"`
	
	// Usage statistics
	Stats      *EnhancedSessionStats  `json:"stats"`
	
	// Resource tracking
	ResourceUsage *SessionResourceUsage `json:"resource_usage,omitempty"`
	
	// Custom metadata
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
	
	// Versioning
	Version    int    `json:"version"`
	SchemaVersion string `json:"schema_version"`
}

// SessionPriority represents the priority level of a session
type SessionPriority string

const (
	SessionPriorityLow    SessionPriority = "low"
	SessionPriorityNormal SessionPriority = "normal"
	SessionPriorityHigh   SessionPriority = "high"
	SessionPriorityCritical SessionPriority = "critical"
)

// EnhancedSessionStats provides detailed session statistics
type EnhancedSessionStats struct {
	// Interaction counts
	InputCount          int   `json:"input_count"`
	OutputCount         int   `json:"output_count"`
	CommandCount        int   `json:"command_count"`
	ErrorCount          int   `json:"error_count"`
	
	// Size metrics
	TotalInputBytes     int64 `json:"total_input_bytes"`
	TotalOutputBytes    int64 `json:"total_output_bytes"`
	AverageInputSize    int   `json:"average_input_size"`
	AverageOutputSize   int   `json:"average_output_size"`
	
	// Timing metrics
	TotalDuration       time.Duration `json:"total_duration"`
	ActiveDuration      time.Duration `json:"active_duration"`
	IdleDuration        time.Duration `json:"idle_duration"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastActiveTime      time.Time     `json:"last_active_time"`
	
	// Process metrics
	ProcessCount        int   `json:"process_count"`
	ProcessRestarts     int   `json:"process_restarts"`
	ProcessCrashes      int   `json:"process_crashes"`
	
	// Token metrics (if applicable)
	TotalTokensUsed     int     `json:"total_tokens_used"`
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	TokensPerMinute     float64 `json:"tokens_per_minute"`
	
	// Session lifecycle
	StartCount          int       `json:"start_count"`
	PauseCount          int       `json:"pause_count"`
	ResumeCount         int       `json:"resume_count"`
	SaveCount           int       `json:"save_count"`
	LastSaveTime        time.Time `json:"last_save_time"`
	
	// Performance metrics
	PerformanceMetrics  *PerformanceMetrics `json:"performance_metrics,omitempty"`
}

// PerformanceMetrics tracks session performance
type PerformanceMetrics struct {
	MinResponseTime    time.Duration `json:"min_response_time"`
	MaxResponseTime    time.Duration `json:"max_response_time"`
	P50ResponseTime    time.Duration `json:"p50_response_time"`
	P95ResponseTime    time.Duration `json:"p95_response_time"`
	P99ResponseTime    time.Duration `json:"p99_response_time"`
	ThroughputRPM      float64       `json:"throughput_rpm"` // Requests per minute
	ErrorRate          float64       `json:"error_rate"`     // Error percentage
	AvailabilityPercent float64      `json:"availability_percent"`
}

// SessionResourceUsage tracks resource consumption
type SessionResourceUsage struct {
	// Memory usage
	PeakMemoryMB       int64   `json:"peak_memory_mb"`
	CurrentMemoryMB    int64   `json:"current_memory_mb"`
	AverageMemoryMB    int64   `json:"average_memory_mb"`
	MemoryEfficiency   float64 `json:"memory_efficiency"`
	
	// CPU usage
	PeakCPUPercent     float64 `json:"peak_cpu_percent"`
	CurrentCPUPercent  float64 `json:"current_cpu_percent"`
	AverageCPUPercent  float64 `json:"average_cpu_percent"`
	CPUTime            time.Duration `json:"cpu_time"`
	
	// Disk usage
	DiskReadBytes      int64 `json:"disk_read_bytes"`
	DiskWriteBytes     int64 `json:"disk_write_bytes"`
	DiskOperations     int64 `json:"disk_operations"`
	
	// Network usage
	NetworkBytesIn     int64 `json:"network_bytes_in"`
	NetworkBytesOut    int64 `json:"network_bytes_out"`
	NetworkConnections int   `json:"network_connections"`
	
	// File system
	FilesCreated       int   `json:"files_created"`
	FilesModified      int   `json:"files_modified"`
	FilesDeleted       int   `json:"files_deleted"`
	
	// Last updated
	LastUpdated        time.Time `json:"last_updated"`
}

// NewEnhancedSessionMetadata creates new enhanced session metadata
func NewEnhancedSessionMetadata(id, name string, config *claude.SessionConfig) *EnhancedSessionMetadata {
	now := time.Now()
	
	return &EnhancedSessionMetadata{
		ID:          id,
		Name:        name,
		State:       claude.SessionStateCreated,
		Config:      config,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        make([]string, 0),
		Priority:    SessionPriorityNormal,
		Labels:      make(map[string]string),
		CustomData:  make(map[string]interface{}),
		Version:     1,
		SchemaVersion: "1.0",
		Stats:       &EnhancedSessionStats{
			LastActiveTime: now,
			LastSaveTime:   now,
		},
		ResourceUsage: &SessionResourceUsage{
			LastUpdated: now,
		},
	}
}

// MetadataTracker tracks and updates session metadata
type MetadataTracker struct {
	mu          sync.RWMutex
	metadata    map[string]*EnhancedSessionMetadata // SessionID -> Metadata
	responseTimes []time.Duration // For calculating percentiles
	lastCleanup time.Time
}

// NewMetadataTracker creates a new metadata tracker
func NewMetadataTracker() *MetadataTracker {
	return &MetadataTracker{
		metadata:    make(map[string]*EnhancedSessionMetadata),
		responseTimes: make([]time.Duration, 0, 1000), // Keep last 1000 response times
		lastCleanup: time.Now(),
	}
}

// TrackSessionCreated tracks session creation
func (mt *MetadataTracker) TrackSessionCreated(sessionID, name string, config *claude.SessionConfig) *EnhancedSessionMetadata {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata := NewEnhancedSessionMetadata(sessionID, name, config)
	mt.metadata[sessionID] = metadata
	
	return metadata
}

// TrackSessionStarted tracks session start
func (mt *MetadataTracker) TrackSessionStarted(sessionID string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	metadata.State = claude.SessionStateActive
	metadata.StartedAt = &now
	metadata.LastActiveAt = &now
	metadata.UpdatedAt = now
	metadata.Stats.StartCount++
	metadata.Version++
	
	return nil
}

// TrackSessionTerminated tracks session termination
func (mt *MetadataTracker) TrackSessionTerminated(sessionID string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	metadata.State = claude.SessionStateTerminated
	metadata.TerminatedAt = &now
	metadata.UpdatedAt = now
	metadata.Version++
	
	// Calculate total duration
	if metadata.StartedAt != nil {
		metadata.Stats.TotalDuration = now.Sub(*metadata.StartedAt)
	}
	
	return nil
}

// TrackSessionPaused tracks session pause
func (mt *MetadataTracker) TrackSessionPaused(sessionID string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	metadata.State = claude.SessionStatePaused
	metadata.UpdatedAt = now
	metadata.Stats.PauseCount++
	metadata.Version++
	
	return nil
}

// TrackSessionResumed tracks session resume
func (mt *MetadataTracker) TrackSessionResumed(sessionID string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	metadata.State = claude.SessionStateActive
	metadata.LastActiveAt = &now
	metadata.UpdatedAt = now
	metadata.Stats.ResumeCount++
	metadata.Version++
	
	return nil
}

// TrackInput tracks input sent to session
func (mt *MetadataTracker) TrackInput(sessionID, input string, metadata map[string]interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	sessionMeta, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	inputSize := len(input)
	
	// Update basic stats
	sessionMeta.Stats.InputCount++
	sessionMeta.Stats.TotalInputBytes += int64(inputSize)
	sessionMeta.LastActiveAt = &now
	sessionMeta.UpdatedAt = now
	sessionMeta.Stats.LastActiveTime = now
	sessionMeta.Version++
	
	// Update average input size
	if sessionMeta.Stats.InputCount > 0 {
		sessionMeta.Stats.AverageInputSize = int(sessionMeta.Stats.TotalInputBytes / int64(sessionMeta.Stats.InputCount))
	}
	
	// Track tokens if provided in metadata
	if tokens, ok := metadata["tokens"].(int); ok {
		sessionMeta.Stats.InputTokens += tokens
		sessionMeta.Stats.TotalTokensUsed += tokens
	}
	
	return nil
}

// TrackOutput tracks output received from session
func (mt *MetadataTracker) TrackOutput(sessionID, output string, responseTime time.Duration, metadata map[string]interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	sessionMeta, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	outputSize := len(output)
	
	// Update basic stats
	sessionMeta.Stats.OutputCount++
	sessionMeta.Stats.TotalOutputBytes += int64(outputSize)
	sessionMeta.LastActiveAt = &now
	sessionMeta.UpdatedAt = now
	sessionMeta.Stats.LastActiveTime = now
	sessionMeta.Version++
	
	// Update average output size
	if sessionMeta.Stats.OutputCount > 0 {
		sessionMeta.Stats.AverageOutputSize = int(sessionMeta.Stats.TotalOutputBytes / int64(sessionMeta.Stats.OutputCount))
	}
	
	// Track response time
	if responseTime > 0 {
		mt.responseTimes = append(mt.responseTimes, responseTime)
		// Keep only last 1000 response times
		if len(mt.responseTimes) > 1000 {
			mt.responseTimes = mt.responseTimes[len(mt.responseTimes)-1000:]
		}
		
		// Update average response time
		totalResponseTime := time.Duration(0)
		for _, rt := range mt.responseTimes {
			totalResponseTime += rt
		}
		sessionMeta.Stats.AverageResponseTime = totalResponseTime / time.Duration(len(mt.responseTimes))
		
		// Update performance metrics
		mt.updatePerformanceMetrics(sessionMeta, responseTime)
	}
	
	// Track tokens if provided in metadata
	if tokens, ok := metadata["tokens"].(int); ok {
		sessionMeta.Stats.OutputTokens += tokens
		sessionMeta.Stats.TotalTokensUsed += tokens
	}
	
	return nil
}

// TrackError tracks an error in the session
func (mt *MetadataTracker) TrackError(sessionID string, err error, metadata map[string]interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	sessionMeta, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	sessionMeta.Stats.ErrorCount++
	sessionMeta.UpdatedAt = now
	sessionMeta.Version++
	
	// Update error rate
	totalOperations := sessionMeta.Stats.InputCount + sessionMeta.Stats.OutputCount
	if totalOperations > 0 {
		if sessionMeta.Stats.PerformanceMetrics == nil {
			sessionMeta.Stats.PerformanceMetrics = &PerformanceMetrics{}
		}
		sessionMeta.Stats.PerformanceMetrics.ErrorRate = float64(sessionMeta.Stats.ErrorCount) / float64(totalOperations) * 100
	}
	
	// Track process crashes if it's a process-related error
	if processError, ok := metadata["process_error"].(bool); ok && processError {
		sessionMeta.Stats.ProcessCrashes++
	}
	
	return nil
}

// TrackResourceUsage updates resource usage statistics
func (mt *MetadataTracker) TrackResourceUsage(sessionID string, usage *SessionResourceUsage) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	now := time.Now()
	metadata.ResourceUsage = usage
	metadata.ResourceUsage.LastUpdated = now
	metadata.UpdatedAt = now
	metadata.Version++
	
	return nil
}

// updatePerformanceMetrics updates performance metrics with new response time
func (mt *MetadataTracker) updatePerformanceMetrics(metadata *EnhancedSessionMetadata, responseTime time.Duration) {
	if metadata.Stats.PerformanceMetrics == nil {
		metadata.Stats.PerformanceMetrics = &PerformanceMetrics{
			MinResponseTime: responseTime,
			MaxResponseTime: responseTime,
		}
	}
	
	pm := metadata.Stats.PerformanceMetrics
	
	// Update min/max
	if responseTime < pm.MinResponseTime || pm.MinResponseTime == 0 {
		pm.MinResponseTime = responseTime
	}
	if responseTime > pm.MaxResponseTime {
		pm.MaxResponseTime = responseTime
	}
	
	// Calculate percentiles from recent response times
	if len(mt.responseTimes) > 0 {
		sortedTimes := make([]time.Duration, len(mt.responseTimes))
		copy(sortedTimes, mt.responseTimes)
		
		// Simple sorting (could use sort.Slice for better performance)
		for i := 0; i < len(sortedTimes)-1; i++ {
			for j := i + 1; j < len(sortedTimes); j++ {
				if sortedTimes[i] > sortedTimes[j] {
					sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
				}
			}
		}
		
		length := len(sortedTimes)
		pm.P50ResponseTime = sortedTimes[length*50/100]
		pm.P95ResponseTime = sortedTimes[length*95/100]
		pm.P99ResponseTime = sortedTimes[length*99/100]
	}
	
	// Calculate throughput (requests per minute)
	if metadata.StartedAt != nil {
		duration := time.Since(*metadata.StartedAt)
		if duration > 0 {
			totalOperations := metadata.Stats.InputCount + metadata.Stats.OutputCount
			pm.ThroughputRPM = float64(totalOperations) / duration.Minutes()
		}
	}
}

// GetMetadata returns session metadata
func (mt *MetadataTracker) GetMetadata(sessionID string) (*EnhancedSessionMetadata, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return nil, fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	// Return a deep copy
	return mt.copyMetadata(metadata), nil
}

// ListMetadata returns all session metadata
func (mt *MetadataTracker) ListMetadata() []*EnhancedSessionMetadata {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	result := make([]*EnhancedSessionMetadata, 0, len(mt.metadata))
	for _, metadata := range mt.metadata {
		result = append(result, mt.copyMetadata(metadata))
	}
	
	return result
}

// UpdateTags updates session tags
func (mt *MetadataTracker) UpdateTags(sessionID string, tags []string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	metadata.Tags = make([]string, len(tags))
	copy(metadata.Tags, tags)
	metadata.UpdatedAt = time.Now()
	metadata.Version++
	
	return nil
}

// UpdateLabels updates session labels
func (mt *MetadataTracker) UpdateLabels(sessionID string, labels map[string]string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	metadata.Labels = make(map[string]string)
	for k, v := range labels {
		metadata.Labels[k] = v
	}
	metadata.UpdatedAt = time.Now()
	metadata.Version++
	
	return nil
}

// UpdateCustomData updates custom metadata
func (mt *MetadataTracker) UpdateCustomData(sessionID string, customData map[string]interface{}) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	metadata, exists := mt.metadata[sessionID]
	if !exists {
		return fmt.Errorf("session metadata not found: %s", sessionID)
	}
	
	if metadata.CustomData == nil {
		metadata.CustomData = make(map[string]interface{})
	}
	
	for k, v := range customData {
		metadata.CustomData[k] = v
	}
	metadata.UpdatedAt = time.Now()
	metadata.Version++
	
	return nil
}

// RemoveMetadata removes session metadata
func (mt *MetadataTracker) RemoveMetadata(sessionID string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	delete(mt.metadata, sessionID)
	return nil
}

// copyMetadata creates a deep copy of metadata
func (mt *MetadataTracker) copyMetadata(original *EnhancedSessionMetadata) *EnhancedSessionMetadata {
	// Use JSON marshaling/unmarshaling for deep copy (simple but not most efficient)
	data, _ := json.Marshal(original)
	var copy EnhancedSessionMetadata
	json.Unmarshal(data, &copy)
	return &copy
}

// CleanupOldMetadata removes metadata for old sessions
func (mt *MetadataTracker) CleanupOldMetadata(maxAge time.Duration) int {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	var toDelete []string
	
	for sessionID, metadata := range mt.metadata {
		if metadata.UpdatedAt.Before(cutoff) {
			toDelete = append(toDelete, sessionID)
		}
	}
	
	for _, sessionID := range toDelete {
		delete(mt.metadata, sessionID)
	}
	
	mt.lastCleanup = time.Now()
	return len(toDelete)
}