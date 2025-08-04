# Telemetry Package

This package provides telemetry functionality for vibes-mcp-cli, enabling the collection and transmission of usage data, errors, and performance metrics to the vibes-agent-backend.

## Features

- **Buffered Logging**: Logs are buffered and sent in batches to reduce network overhead
- **Rate Limiting**: Respects API rate limits (100 logs per 5 minutes)
- **Retry Logic**: Implements exponential backoff for failed requests
- **Async Processing**: Non-blocking log transmission to avoid impacting UI performance
- **Graceful Shutdown**: Ensures all buffered logs are sent before application exit
- **Zap Integration**: Works seamlessly with existing zap loggers
- **Session Tracking**: Includes session information for better analytics

## Configuration

Telemetry is configured via environment variables:

```bash
# Enable/disable telemetry
OPENAI_CLI_TELEMETRY_ENABLED=true

# API key for authentication with vibes-agent-backend
OPENAI_CLI_TELEMETRY_API_KEY=your-api-key-here

# Batch size (number of logs per batch)
OPENAI_CLI_TELEMETRY_BATCH_SIZE=50

# How often to flush buffered logs
OPENAI_CLI_TELEMETRY_FLUSH_INTERVAL=30s
```

## Usage

### Basic Setup

```go
import "openai-cli/internal/telemetry"

// Setup telemetry client
client, err := telemetry.SetupTelemetry(
    cfg.TelemetryEnabled,
    cfg.TelemetryAPIKey,
    cfg.AgentURL,
    cfg.TelemetryBatchSize,
    cfg.TelemetryFlushInterval,
    nil,
)

// Setup telemetry logger
logger, err := telemetry.SetupTelemetryLogger(client, "component-name", "info")

// Use the logger
logger.Info("Application started", zap.String("version", "1.0.0"))
logger.Error("Something went wrong", zap.Error(err))

// Cleanup on shutdown
defer client.Close(context.Background())
```

### Logging Specific Events

```go
// Log UI errors
telemetry.LogUIError(client, "ui", "Failed to render component", err, map[string]interface{}{
    "component": "chat-input",
    "user_id": "123",
})

// Log API calls
telemetry.LogAPICall(client, "openai", "/v1/chat/completions", duration, success, errorMsg)

// Log user actions
telemetry.LogUserAction(client, "button_click", map[string]interface{}{
    "button": "send-message",
    "page": "chat",
})

// Log session events
telemetry.LogSessionEvent(client, "session_start", sessionID, map[string]interface{}{
    "user_id": "123",
})
```

## Log Levels

The telemetry system supports four log levels:

- `DEBUG`: Detailed debugging information
- `INFO`: General information about application operation
- `WARN`: Warning messages about potential issues
- `ERROR`: Error messages about actual problems

## Rate Limiting

The telemetry client implements a token bucket rate limiter that respects the vibes-agent-backend API limits:

- **Capacity**: 100 logs
- **Refill Rate**: 100 logs per 5 minutes
- **Behavior**: Requests are blocked when the limit is exceeded, with automatic retry

## Error Handling

The telemetry system is designed to be fault-tolerant:

- **Network Failures**: Automatic retry with exponential backoff
- **Rate Limiting**: Waits for rate limit reset
- **Buffer Overflow**: Drops oldest logs when buffer is full
- **API Errors**: Logs errors locally without disrupting application flow

## Data Structure

Each log entry contains:

```go
type LogEntry struct {
    ID        string                 `json:"id"`
    Timestamp time.Time              `json:"timestamp"`
    Level     LogLevel               `json:"level"`
    Message   string                 `json:"message"`
    SessionID string                 `json:"session_id"`
    Component string                 `json:"component"`
    UserID    string                 `json:"user_id,omitempty"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
    Error     *ErrorContext          `json:"error,omitempty"`
}
```

## Security Considerations

- **API Keys**: Stored securely and transmitted via HTTPS
- **Data Privacy**: No sensitive user data is logged by default
- **Opt-in**: Telemetry is disabled by default and must be explicitly enabled
- **Local Fallback**: Application continues to function even if telemetry fails

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/telemetry/...
```

Test coverage includes:
- Client functionality
- Rate limiting
- Buffer management
- Logger integration
- Error scenarios

## Performance

The telemetry system is designed for minimal performance impact:

- **Async Processing**: All network requests are asynchronous
- **Batching**: Reduces network overhead by sending logs in batches
- **Connection Pooling**: Reuses HTTP connections for efficiency
- **Buffer Management**: Ring buffer with configurable capacity

## Troubleshooting

### Telemetry Not Working

1. Check that `OPENAI_CLI_TELEMETRY_ENABLED=true`
2. Verify the API key is correctly set
3. Ensure the agent backend URL is reachable
4. Check application logs for telemetry errors

### High Memory Usage

1. Reduce batch size: `OPENAI_CLI_TELEMETRY_BATCH_SIZE=25`
2. Increase flush interval: `OPENAI_CLI_TELEMETRY_FLUSH_INTERVAL=60s`
3. Check for network connectivity issues causing buffer buildup

### Rate Limiting Issues

The client automatically handles rate limiting, but you can:

1. Reduce log frequency in your application
2. Use appropriate log levels (avoid excessive DEBUG logs)
3. Monitor the remaining token count via the rate limiter

## Architecture

```
Application Code
       ↓
TelemetryLogger (zap integration)
       ↓
TelemetryClient (batching, rate limiting)
       ↓
Buffer (ring buffer)
       ↓
HTTP Client (connection pooling)
       ↓
vibes-agent-backend (/api/telemetry/logs)
```

The system uses a multi-layered approach to ensure reliability, performance, and ease of use.