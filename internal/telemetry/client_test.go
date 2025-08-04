package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestTelemetryClient_Log(t *testing.T) {
	// Create a test server
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-API-Key") == "" {
			t.Error("Expected X-API-Key header")
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "batch_id": "test", "processed": 1}`))
	}))
	defer server.Close()
	
	config := &Config{
		Enabled:       true,
		APIKey:        "test-key",
		BaseURL:       server.URL,
		BatchSize:     1, // Small batch size to trigger immediate send
		FlushInterval: time.Minute,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
		BufferSize:    10,
		Timeout:       5 * time.Second,
		SessionID:     "test-session",
		UserID:        "test-user",
	}
	
	client := NewClient(config, zap.NewNop())
	defer client.Close(context.Background())
	
	entry := LogEntry{
		Level:     LogLevelInfo,
		Message:   "Test log entry",
		Component: "test",
	}
	
	err := client.Log(context.Background(), entry)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	
	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)
	
	if !called {
		t.Error("Expected HTTP request to be made")
	}
}

func TestTelemetryClient_DisabledDoesNotSend(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()
	
	config := &Config{
		Enabled:   false, // Disabled
		BaseURL:   server.URL,
		BatchSize: 1,
	}
	
	client := NewClient(config, zap.NewNop())
	defer client.Close(context.Background())
	
	entry := LogEntry{
		Level:     LogLevelInfo,
		Message:   "Test log entry",
		Component: "test",
	}
	
	err := client.Log(context.Background(), entry)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	
	// Wait a bit to ensure no async processing happens
	time.Sleep(50 * time.Millisecond)
	
	if called {
		t.Error("Expected no HTTP request when telemetry is disabled")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter()
	
	// Should start with full capacity
	if !limiter.Allow() {
		t.Error("Expected first request to be allowed")
	}
	
	// Should track remaining tokens
	remaining := limiter.Remaining()
	if remaining >= 100 {
		t.Errorf("Expected remaining tokens to decrease, got %d", remaining)
	}
}

func TestBuffer(t *testing.T) {
	buffer := NewBuffer(2)
	
	entry1 := LogEntry{Message: "entry1"}
	entry2 := LogEntry{Message: "entry2"}
	entry3 := LogEntry{Message: "entry3"}
	
	// Add entries
	if err := buffer.Add(entry1); err != nil {
		t.Fatalf("Failed to add entry1: %v", err)
	}
	
	if err := buffer.Add(entry2); err != nil {
		t.Fatalf("Failed to add entry2: %v", err)
	}
	
	// Buffer should be full
	if !buffer.IsFull() {
		t.Error("Expected buffer to be full")
	}
	
	// Adding another should fail
	if err := buffer.Add(entry3); err != ErrBufferFull {
		t.Errorf("Expected ErrBufferFull, got %v", err)
	}
	
	// Flush should return all entries
	entries := buffer.Flush()
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
	
	// Buffer should be empty after flush
	if buffer.Size() != 0 {
		t.Errorf("Expected empty buffer after flush, got size %d", buffer.Size())
	}
}

func TestTelemetryLogger(t *testing.T) {
	// Create a mock client that tracks calls
	mockClient := &mockClient{enabled: true}
	
	zapLogger := zap.NewNop()
	telemetryLogger := NewTelemetryLogger(mockClient, zapLogger, "test-component")
	
	// Test different log levels
	telemetryLogger.Info("Test info message")
	telemetryLogger.Error("Test error message")
	telemetryLogger.Warn("Test warn message")
	
	// Give some time for async processing
	time.Sleep(50 * time.Millisecond)
	
	if len(mockClient.entries) == 0 {
		t.Error("Expected telemetry entries to be logged")
	}
	
	// Check that entries have correct component
	for _, entry := range mockClient.entries {
		if entry.Component != "test-component" {
			t.Errorf("Expected component 'test-component', got '%s'", entry.Component)
		}
	}
}

// mockClient implements the Client interface for testing
type mockClient struct {
	enabled bool
	entries []LogEntry
}

func (m *mockClient) Log(ctx context.Context, entry LogEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockClient) LogBatch(ctx context.Context, entries []LogEntry) error {
	m.entries = append(m.entries, entries...)
	return nil
}

func (m *mockClient) Flush(ctx context.Context) error {
	return nil
}

func (m *mockClient) Close(ctx context.Context) error {
	return nil
}

func (m *mockClient) IsEnabled() bool {
	return m.enabled
}