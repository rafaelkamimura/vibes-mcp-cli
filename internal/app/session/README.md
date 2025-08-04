# Enhanced Session Manager

This package provides comprehensive session management capabilities for the vibes-mcp-cli with advanced features for interactive control, persistence, search, and detailed tracking.

## Architecture Overview

The enhanced session manager is built as a layered architecture with the following components:

### Core Components

1. **EnhancedManager** (`enhanced_manager.go`) - Main orchestrator that integrates all capabilities
2. **Manager** (`manager.go`) - Base session manager for basic operations
3. **Registry** (`registry.go`) - Session metadata storage and indexing

### Enhanced Features

4. **ConversationHistory** (`history.go`) - Conversation logging and history management
5. **InteractiveController** (`interactive.go`) - Real-time session interaction
6. **MetadataTracker** (`metadata.go`) - Detailed session metrics and tracking
7. **SessionSearcher** (`search.go`) - Advanced search and filtering
8. **PersistenceManager** (`persistence.go`) - Session state persistence and backups

## Key Features

### 1. Interactive Session Control

- **Real-time Communication**: Send commands and receive responses asynchronously
- **Streaming Responses**: Get partial responses as they're generated
- **Request Management**: Track and cancel active requests
- **Response Filtering**: Configure response handling and filtering

```go
// Send interactive input with real-time response handling
ctx := context.Background()
interactionCtx, err := manager.SendInteractiveInput(ctx, sessionID, "Hello Claude", &InteractionOptions{
    Timeout: time.Minute * 5,
    StreamResponse: true,
})

// Stream responses
for response := range interactionCtx.ResponseCh {
    fmt.Printf("Response: %s\n", response.Content)
    if response.Type == ResponseTypeComplete {
        break
    }
}
```

### 2. Session History & Persistence

- **Conversation Logging**: Complete conversation history with metadata
- **Persistent Storage**: Sessions survive application restarts
- **Compression**: Efficient storage with optional compression
- **Backup System**: Automated backups with retention policies
- **Restoration**: Restore sessions from snapshots

```go
// Create backup of all sessions
err := manager.CreateBackup()

// Load session from storage
snapshot, err := manager.LoadSession(sessionID)

// Get conversation history
history, err := manager.GetSessionHistory(sessionID, 100) // Last 100 entries
```

### 3. Session Metadata Tracking

- **Detailed Metrics**: Track inputs, outputs, tokens, response times
- **Resource Usage**: Monitor memory, CPU, disk usage
- **Performance Metrics**: Response time percentiles, throughput
- **Custom Metadata**: Store arbitrary session data
- **Lifecycle Tracking**: Complete session state transitions

```go
// Get enhanced metadata
metadata, err := manager.GetSessionMetadata(sessionID)
fmt.Printf("Session has %d inputs, %d outputs, %d tokens used\n", 
    metadata.Stats.InputCount, 
    metadata.Stats.OutputCount, 
    metadata.Stats.TotalTokensUsed)

// Update session tags and labels
manager.UpdateSessionTags(sessionID, []string{"important", "development"})
manager.UpdateSessionLabels(sessionID, map[string]string{
    "project": "vibes-mcp-cli",
    "environment": "development",
})
```

### 4. Advanced Search & Filtering

- **Text Search**: Search across session names, descriptions, and conversation content
- **Regex Support**: Use regular expressions for complex queries
- **Multi-criteria Filtering**: Filter by state, tags, dates, metrics
- **Sorting**: Sort by various fields with multiple criteria
- **Pagination**: Handle large result sets efficiently

```go
// Quick search
results, err := manager.QuickSearchSessions("error handling", 10)

// Advanced search
criteria := &SearchCriteria{
    Query: "debugging",
    QueryFields: []string{"name", "content"},
    States: []claude.SessionState{claude.SessionStateActive},
    Tags: []string{"development"},
    CreatedAfter: &time.Now().Add(-24 * time.Hour), // Last 24 hours
    MinInputCount: &10, // At least 10 interactions
}

options := &SearchOptions{
    Limit: 50,
    Sort: &SortCriteria{
        Field: SortFieldLastActiveAt,
        Direction: SortDirectionDesc,
    },
}

results, err := manager.SearchSessions(criteria, options)
```

## Usage Examples

### Basic Session Creation and Management

```go
// Create enhanced manager
config := DefaultEnhancedManagerConfig()
config.EnablePersistence = true
config.EnableInteractive = true

manager, err := NewEnhancedManager(config, logger)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Create a new session with metadata
session, err := manager.CreateEnhancedSession(
    "Debug Session",                    // name
    "Debugging authentication issues", // description
    nil,                               // use default config
    []string{"debug", "auth"},         // tags
    SessionPriorityHigh,               // priority
)

// Start the session
err = manager.StartEnhancedSession(session.GetID())

// Execute a command and wait for completion
response, err := manager.ExecuteCommand(
    context.Background(),
    session.GetID(),
    "help auth",
    &InteractionOptions{Timeout: time.Minute * 2},
)

fmt.Printf("Command output: %s\n", response.Content)
```

### Advanced Session Operations

```go
// Search for recent active sessions
recentSessions, err := manager.SearchRecentSessions(time.Hour*24, 20)

// Get comprehensive statistics
stats := manager.GetEnhancedStats()
fmt.Printf("Total sessions: %d, Active: %d\n", 
    stats.TotalSessions, stats.ActiveSessions)

// Clean up old data
cleaned, err := manager.CleanupOldSessions(time.Hour * 24 * 30) // 30 days
fmt.Printf("Cleaned up %d old sessions\n", cleaned)

// Create a backup
err = manager.CreateBackup()
```

## Configuration

### Enhanced Manager Configuration

```go
config := &EnhancedManagerConfig{
    ManagerConfig: &ManagerConfig{
        StoragePath: "./sessions",
        MaxSessions: 50,
        DefaultTimeout: time.Hour * 2,
        CleanupInterval: time.Hour,
        AutoCleanup: true,
        ClaudePath: "claude",
    },
    PersistenceConfig: &PersistenceConfig{
        StoragePath: "./sessions",
        CompressData: true,
        BackupEnabled: true,
        BackupInterval: time.Hour * 6,
        BackupRetention: 24,
        AutoSaveInterval: time.Minute * 5,
        AutoSaveEnabled: true,
        SyncMode: SyncModeBatched,
    },
    MaxHistoryEntries: 10000,
    EnableInteractive: true,
    EnableMetadataTracking: true,
    EnableSearching: true,
    EnablePersistence: true,
    TelemetryEnabled: true,
}
```

## File Structure

```
internal/app/session/
├── enhanced_manager.go    # Main enhanced manager
├── manager.go            # Base session manager
├── registry.go           # Session registry
├── history.go            # Conversation history
├── interactive.go        # Interactive control
├── metadata.go           # Metadata tracking
├── search.go             # Search and filtering
├── persistence.go        # Session persistence
└── README.md            # This documentation
```

## Integration with Telemetry

The enhanced session manager integrates with the existing telemetry system to provide:

- Session lifecycle events
- Performance metrics
- Error tracking
- Resource usage monitoring
- User interaction patterns

## Thread Safety

All components are designed to be thread-safe with proper synchronization:

- Read-Write mutexes for high-concurrency scenarios
- Atomic operations for counters and flags
- Channel-based communication for async operations
- Context-based cancellation for long-running operations

## Error Handling

Comprehensive error handling with:

- Wrapped errors with context
- Structured logging with zap
- Graceful degradation when components are disabled
- Recovery mechanisms for transient failures

## Performance Considerations

- Efficient indexing for fast searches
- Pagination for large result sets
- Compression for storage efficiency
- Background processing for non-critical operations
- Resource monitoring and limiting

This enhanced session manager provides a robust foundation for sophisticated Claude CLI session management with enterprise-grade features for persistence, search, and monitoring.