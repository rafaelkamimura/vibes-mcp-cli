package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"openai-cli/internal/telemetry"
	"go.uber.org/zap"
)

func main() {
	// This is a demonstration of the telemetry system
	fmt.Println("Telemetry Demo - vibes-mcp-cli")
	fmt.Println("=====================================")
	
	// Setup telemetry client (disabled by default for demo)
	client, err := telemetry.SetupTelemetry(
		false, // Set to true to actually send logs
		"demo-api-key",
		"http://localhost:8000",
		10,    // Small batch size for demo
		5*time.Second, // Short flush interval for demo
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to setup telemetry: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client.Close(ctx)
	}()
	
	// Setup telemetry logger
	logger, err := telemetry.SetupTelemetryLogger(client, "demo", "debug")
	if err != nil {
		log.Fatalf("Failed to setup telemetry logger: %v", err)
	}
	defer logger.Sync()
	
	fmt.Printf("Telemetry enabled: %v\n", client.IsEnabled())
	fmt.Println()
	
	// Demonstrate different types of logging
	fmt.Println("1. Basic logging with different levels:")
	logger.Debug("This is a debug message", zap.String("component", "demo"))
	logger.Info("Application started", zap.String("version", "1.0.0"))
	logger.Warn("This is a warning", zap.Int("warning_code", 404))
	logger.Error("This is an error", zap.String("error_type", "network"))
	
	fmt.Println("2. Specialized telemetry functions:")
	
	// Log a UI error
	telemetry.LogUIError(client, "demo-ui", "Button click failed", 
		fmt.Errorf("network timeout"), map[string]interface{}{
			"button_id": "submit-btn",
			"user_id": "demo-user",
		})
	
	// Log an API call
	telemetry.LogAPICall(client, "openai", "/v1/chat/completions", 
		250*time.Millisecond, true, "")
	
	// Log a user action
	telemetry.LogUserAction(client, "demo_action", map[string]interface{}{
		"page": "home",
		"feature": "telemetry_demo",
	})
	
	// Log a session event
	telemetry.LogSessionEvent(client, "demo_session_start", "demo-session-123", 
		map[string]interface{}{
			"user": "demo-user",
			"timestamp": time.Now().Unix(),
		})
	
	fmt.Println("3. Demonstrating structured logging:")
	logger.Info("User performed action",
		zap.String("action", "file_upload"),
		zap.String("user_id", "user123"),
		zap.String("file_name", "document.pdf"),
		zap.Int64("file_size", 1024*1024),
		zap.Duration("processing_time", 2*time.Second),
		zap.Bool("success", true),
	)
	
	fmt.Println()
	fmt.Println("Demo completed!")
	fmt.Println("Note: Telemetry is disabled in this demo.")
	fmt.Println("To enable, set OPENAI_CLI_TELEMETRY_ENABLED=true and provide a valid API key.")
	
	// Wait a moment for any async processing
	time.Sleep(100 * time.Millisecond)
}