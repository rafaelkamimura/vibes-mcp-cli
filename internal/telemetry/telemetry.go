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
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelError,
		Message:   message,
		Component: component, // Keep for backward compatibility
		Metadata:  metadata,
	}
	
	if err != nil {
		entry.ErrorCode = "ui_error"
		entry.StackTrace = err.Error() // Store error details in stack trace
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]interface{})
		}
		entry.Metadata["error"] = err.Error()
		entry.Metadata["component"] = component
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
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   "Session event: " + eventType,
		Component: "session",
		Metadata:  metadata,
	}
	
	if entry.Metadata == nil {
		entry.Metadata = make(map[string]interface{})
	}
	entry.Metadata["event_type"] = eventType
	entry.Metadata["session_id"] = sessionID
	
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
		Component: "api_client", // Keep for backward compatibility
		Endpoint:  endpoint,     // Use dedicated endpoint field
		Metadata: map[string]interface{}{
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
	
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     LogLevelInfo,
		Message:   "User action: " + action,
		Component: "ui",
		Metadata:  metadata,
	}
	
	if entry.Metadata == nil {
		entry.Metadata = make(map[string]interface{})
	}
	entry.Metadata["action"] = action
	
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Log(ctx, entry)
	}()
}