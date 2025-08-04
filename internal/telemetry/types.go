package telemetry

import (
	"context"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry represents a single log entry to be sent to the telemetry API
// This struct aligns with the vibes-agent-backend TelemetryLogEntryCreate schema
type LogEntry struct {
	ID            string                 `json:"id,omitempty"`
	ClientName    string                 `json:"client_name"`
	ClientVersion string                 `json:"client_version"`
	SessionID     string                 `json:"session_id,omitempty"`
	Level         LogLevel               `json:"log_level"`
	Message       string                 `json:"message"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	StackTrace    string                 `json:"stack_trace,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	
	// DEPRECATED: Fields below are kept for internal use only and not serialized to JSON
	// They should be moved to Metadata field for backend compatibility
	Component string `json:"-"` // No longer sent to backend, use metadata instead
	UserID    string `json:"-"` // No longer sent to backend, use metadata instead
	Error     *ErrorContext `json:"-"` // No longer sent to backend, use error_code/stack_trace instead
}

// ErrorContext contains additional error information
type ErrorContext struct {
	Type       string `json:"type"`
	StackTrace string `json:"stack_trace,omitempty"`
	Code       string `json:"code,omitempty"`
}

// BatchRequest represents a batch of log entries to send to the API
// This aligns with the vibes-agent-backend TelemetryLogsBatchCreate schema
type BatchRequest struct {
	Logs []LogEntry `json:"logs"`
}

// BatchResponse represents the API response for a batch submission
// This aligns with the vibes-agent-backend TelemetryLogsBatchResponse schema
type BatchResponse struct {
	CreatedLogs []LogEntryResponse `json:"created_logs"`
	Count       int                `json:"count"`
}

// LogEntryResponse represents a telemetry log entry response from the API
type LogEntryResponse struct {
	ID            string                 `json:"id"`
	ClientName    string                 `json:"client_name"`
	ClientVersion string                 `json:"client_version"`
	SessionID     string                 `json:"session_id,omitempty"`
	Level         LogLevel               `json:"log_level"`
	Message       string                 `json:"message"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	StackTrace    string                 `json:"stack_trace,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	CreatedAt     time.Time              `json:"created_at"`
}

// Config holds telemetry configuration
type Config struct {
	// Enabled controls whether telemetry is active
	Enabled bool
	// APIKey for authenticating with the telemetry API
	APIKey string
	// BaseURL is the base URL of the vibes-agent-backend
	BaseURL string
	// ClientName identifies the client application
	ClientName string
	// ClientVersion is the version of the client application
	ClientVersion string
	// BatchSize is the maximum number of logs per batch
	BatchSize int
	// FlushInterval is how often to flush logs
	FlushInterval time.Duration
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// RetryBackoff is the initial backoff duration for retries
	RetryBackoff time.Duration
	// BufferSize is the maximum number of logs to buffer
	BufferSize int
	// Timeout is the HTTP request timeout
	Timeout time.Duration
	// SessionID identifies the current session
	SessionID string
	// UserID identifies the current user (optional)
	UserID string
	// AuthToken is the JWT token for authentication (alternative to API key)
	AuthToken string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Enabled:       false, // Disabled by default
		ClientName:    "vibes-mcp-cli",
		ClientVersion: "1.0.0",
		BatchSize:     50,
		FlushInterval: 30 * time.Second,
		MaxRetries:    3,
		RetryBackoff:  time.Second,
		BufferSize:    1000,
		Timeout:       10 * time.Second,
	}
}

// Client interface defines the telemetry client contract
type Client interface {
	// Log sends a log entry to the telemetry system
	Log(ctx context.Context, entry LogEntry) error
	// LogBatch sends multiple log entries in a single request
	LogBatch(ctx context.Context, entries []LogEntry) error
	// Flush forces all buffered logs to be sent immediately
	Flush(ctx context.Context) error
	// Close gracefully shuts down the client, flushing remaining logs
	Close(ctx context.Context) error
	// IsEnabled returns whether telemetry is currently enabled
	IsEnabled() bool
}

// Buffer interface defines the log buffer contract
type Buffer interface {
	// Add adds a log entry to the buffer
	Add(entry LogEntry) error
	// Flush returns all buffered entries and clears the buffer
	Flush() []LogEntry
	// Size returns the current number of buffered entries
	Size() int
	// IsFull returns whether the buffer is at capacity
	IsFull() bool
	// Clear removes all entries from the buffer
	Clear()
}

// RateLimiter interface defines rate limiting contract
type RateLimiter interface {
	// Allow returns whether a request is allowed under the rate limit
	Allow() bool
	// Wait blocks until a request can be made
	Wait(ctx context.Context) error
	// Remaining returns the number of requests remaining in the current window
	Remaining() int
}