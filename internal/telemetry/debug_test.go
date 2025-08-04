package telemetry

import (
	"encoding/json"
	"fmt"
	"time"
)

// DebugJSONSerialization shows exactly what JSON the CLI produces
func DebugJSONSerialization() {
	// Create a log entry similar to what the CLI would create
	entry := LogEntry{
		ClientName:    "vibes-mcp-cli",
		ClientVersion: "1.0.0",
		SessionID:     "session-123",
		Level:         LogLevelInfo,
		Message:       "Test log message",
		Metadata: map[string]interface{}{
			"user_id":   "test-user",
			"component": "test",
		},
		ErrorCode:  "", // Empty string
		StackTrace: "", // Empty string
		Endpoint:   "", // Empty string
		UserAgent:  "vibes-mcp-cli-telemetry/1.0",
		IPAddress:  "", // Empty string
		Timestamp:  time.Now().UTC(),
	}

	// Marshal to JSON to see the exact format
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("JSON marshal error: %v\n", err)
		return
	}

	fmt.Printf("Single LogEntry JSON:\n%s\n\n", string(jsonData))

	// Create a batch request
	batch := BatchRequest{
		Logs: []LogEntry{entry},
	}

	batchJSON, err := json.Marshal(batch)
	if err != nil {
		fmt.Printf("Batch JSON marshal error: %v\n", err)
		return
	}

	fmt.Printf("BatchRequest JSON:\n%s\n\n", string(batchJSON))

	// Test with nil values instead of empty strings
	entryWithNils := entry
	entryWithNils.ErrorCode = ""
	entryWithNils.StackTrace = ""
	entryWithNils.Endpoint = ""
	entryWithNils.IPAddress = ""

	nilJSON, err := json.Marshal(entryWithNils)
	if err != nil {
		fmt.Printf("Nil JSON marshal error: %v\n", err)
		return
	}

	fmt.Printf("LogEntry with empty strings JSON:\n%s\n\n", string(nilJSON))
}