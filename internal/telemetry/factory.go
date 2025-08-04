package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupTelemetry configures telemetry based on application config
func SetupTelemetry(enabled bool, apiKey, baseURL string, batchSize int, flushInterval time.Duration, logger *zap.Logger) (Client, error) {
	config := &Config{
		Enabled:       enabled,
		APIKey:        apiKey,
		BaseURL:       baseURL,
		ClientName:    "vibes-mcp-cli",
		ClientVersion: getVersion(),
		BatchSize:     batchSize,
		FlushInterval: flushInterval,
		MaxRetries:    3,
		RetryBackoff:  time.Second,
		BufferSize:    1000,
		Timeout:       10 * time.Second,
		SessionID:     generateSessionID(),
		UserID:        getCurrentUser(),
	}
	
	return NewClient(config, logger), nil
}

// SetupTelemetryForAgentBackend sets up telemetry specifically for vibes-agent-backend integration
// This auto-enables telemetry when connecting to the agent backend for better debugging
func SetupTelemetryForAgentBackend(agentURL, authToken, telemetryAPIKey string, logger *zap.Logger) (Client, error) {
	// Enable telemetry by default when using agent backend
	enabled := true
	
	config := &Config{
		Enabled:       enabled,
		APIKey:        telemetryAPIKey, // Use dedicated telemetry API key if provided
		AuthToken:     authToken,       // Use JWT token if API key is not available
		BaseURL:       agentURL,
		ClientName:    "vibes-mcp-cli",
		ClientVersion: getVersion(),
		BatchSize:     50,
		FlushInterval: 30 * time.Second,
		MaxRetries:    3,
		RetryBackoff:  time.Second,
		BufferSize:    1000,
		Timeout:       10 * time.Second,
		SessionID:     generateSessionID(),
		UserID:        getCurrentUser(),
	}
	
	return NewClient(config, logger), nil
}

// SetupTelemetryLogger creates a logger that sends logs to both console and telemetry
func SetupTelemetryLogger(client Client, component string, logLevel string) (*TelemetryLogger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}
	
	// Create console encoder config
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	// Create console core
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleConfig),
		zapcore.AddSync(os.Stderr),
		level,
	)
	
	var cores []zapcore.Core
	cores = append(cores, consoleCore)
	
	// Add telemetry core if enabled
	if client.IsEnabled() {
		telemetryCore := NewTelemetryCore(client, component, level)
		cores = append(cores, telemetryCore)
	}
	
	// Create combined core
	core := zapcore.NewTee(cores...)
	
	// Create zap logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	
	// Create telemetry logger wrapper
	telemetryLogger := NewTelemetryLogger(client, zapLogger, component)
	
	return telemetryLogger, nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("session_%s_%d", hex.EncodeToString(bytes), time.Now().Unix())
}

// getCurrentUser returns the current system user name
func getCurrentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	
	// Fallback to environment variables
	if username := os.Getenv("USER"); username != "" {
		return username
	}
	if username := os.Getenv("USERNAME"); username != "" {
		return username
	}
	
	return "unknown"
}

// getVersion returns the application version
func getVersion() string {
	// In production, this could read from a version file or build info
	// For now, return a default version
	return "1.0.0"
}