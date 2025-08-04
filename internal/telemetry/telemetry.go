// Package telemetry provides logging and error reporting capabilities for vibes-mcp-cli.
// It sends structured logs and errors to the vibes-agent-backend telemetry API
// with rate limiting, batching, and retry logic.
package telemetry

import (
	"context"
	"fmt"
	"time"
)

// LogUIError logs UI-specific errors with context
func LogUIError(client Client, component, message string, err error, metadata map[string]interface{}) {
	if !client.IsEnabled() {
		return
	}
	
	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// Add component info to metadata instead of deprecated Component field
	metadata["component"] = component
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelError,
		Message:   message,
		Component: component, // Keep for internal use (not serialized)
		Metadata:  metadata,
	}
	
	if err != nil {
		entry.ErrorCode = "ui_error"
		entry.StackTrace = err.Error() // Store error details in stack trace
		entry.Metadata["error"] = err.Error()
		entry.Metadata["error_type"] = "ui_error"
	}
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Log(ctx, entry)
	}()
}

// LogSessionEvent logs session-related events
func LogSessionEvent(client Client, eventType, sessionID string, metadata map[string]interface{}) {
	if !client.IsEnabled() {
		return
	}
	
	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// Add component info to metadata instead of deprecated Component field
	metadata["component"] = "session"
	metadata["event_type"] = eventType
	metadata["session_id"] = sessionID
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   "Session event: " + eventType,
		Component: "session", // Keep for internal use (not serialized)
		SessionID: sessionID, // Use dedicated session_id field
		Metadata:  metadata,
	}
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Log(ctx, entry)
	}()
}

// LogAPICall logs API call information and performance
func LogAPICall(client Client, provider, endpoint string, duration time.Duration, success bool, errorMsg string) {
	if !client.IsEnabled() {
		return
	}
	
	level := LogLevelInfo
	if !success {
		level = LogLevelError
	}
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   fmt.Sprintf("API call to %s %s", provider, endpoint),
		Component: "api_client", // Keep for internal use (not serialized)
		Endpoint:  endpoint,     // Use dedicated endpoint field
		Metadata: map[string]interface{}{
			"component":    "api_client", // Move component to metadata
			"provider":     provider,
			"endpoint":     endpoint,
			"duration_ms":  duration.Milliseconds(),
			"success":      success,
		},
	}
	
	if !success && errorMsg != "" {
		entry.ErrorCode = "api_error"
		entry.StackTrace = errorMsg
		entry.Metadata["error"] = errorMsg
		entry.Metadata["error_type"] = "api_error"
	}
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Log(ctx, entry)
	}()
}

// LogUserAction logs user interactions for analytics
func LogUserAction(client Client, action string, metadata map[string]interface{}) {
	if !client.IsEnabled() {
		return
	}
	
	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	// Add component info to metadata instead of deprecated Component field
	metadata["component"] = "ui"
	metadata["action"] = action
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   "User action: " + action,
		Component: "ui", // Keep for internal use (not serialized)
		Metadata:  metadata,
	}
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Log(ctx, entry)
	}()
}