# Session Management Guide

Vibes MCP CLI provides sophisticated session management capabilities, allowing you to maintain persistent conversations, track history, and manage multiple concurrent LLM interactions with zero memory leaks and robust process control.

## 🎯 **Overview**

Session management includes:
- **Persistent Conversations**: Maintain context across CLI invocations
- **Session Lifecycle**: Create, monitor, pause, resume, and terminate sessions
- **Resource Management**: Automatic cleanup and memory leak prevention
- **Process Control**: Robust session process monitoring and recovery
- **Multi-Session Support**: Handle multiple concurrent conversations

## 🚀 **Session Basics**

### Creating Sessions
```bash
# Create a new session
./vibes-mcp-cli session create --name "my-project" --provider openai

# Start interactive session
./vibes-mcp-cli session start --name "my-project"

# Quick session creation and chat
./vibes-mcp-cli chat --session "quick-chat" "Hello there!"
```

### Session Information
```bash
# List all sessions
./vibes-mcp-cli session list

# Show session details
./vibes-mcp-cli session info --name "my-project"

# Show session statistics
./vibes-mcp-cli session stats --name "my-project"
```

### Session Operations
```bash
# Pause a session
./vibes-mcp-cli session pause --name "my-project"

# Resume a paused session
./vibes-mcp-cli session resume --name "my-project"

# Terminate session
./vibes-mcp-cli session terminate --name "my-project"

# Delete session and history
./vibes-mcp-cli session delete --name "my-project"
```

## 📊 **Session States**

### State Lifecycle
Sessions progress through these states:

```
Creating → Active → [Paused] → Terminated → Deleted
    ↓         ↓         ↓          ↓
   Error  →  Error  →  Error  →  Error
```

### State Descriptions

#### **Creating**
- Session is being initialized
- Resources are being allocated
- Provider connection is being established

#### **Active**
- Session is running and responsive
- Can process new messages
- Background monitoring is active

#### **Paused**
- Session is temporarily stopped
- Resources are preserved
- Can be resumed quickly

#### **Terminated**
- Session has ended cleanly
- Resources have been cleaned up
- History is preserved

#### **Error**
- Session encountered an error
- May be recoverable depending on error type
- Automatic recovery attempts may be made

#### **Deleted**
- Session and all data removed
- Cannot be recovered
- Resources fully released

## 🔧 **Session Configuration**

### Default Session Settings
```yaml
# ~/.openai-cli.yaml
sessions:
  default_provider: "openai"
  default_model: "gpt-4"
  default_temperature: 0.7
  max_history: 100
  auto_save: true
  cleanup_on_exit: true
  monitoring_enabled: true
```

### Per-Session Configuration
```bash
# Create session with custom settings
./vibes-mcp-cli session create \
  --name "code-review" \
  --provider openai \
  --model gpt-4 \
  --temperature 0.3 \
  --max-history 50
```

### Session Metadata
Each session maintains:
- **Name**: User-defined identifier
- **ID**: Unique session identifier
- **Provider**: LLM provider (OpenAI, Anthropic)
- **Model**: Specific model being used
- **Created**: Session creation timestamp
- **Last Active**: Most recent activity
- **Message Count**: Number of messages exchanged
- **Token Usage**: Cumulative token statistics
- **Status**: Current session state

## 📈 **Session Monitoring**

### Real-time Status
```bash
# Monitor session in real-time
./vibes-mcp-cli session monitor --name "my-project"

# Output:
Session: my-project (ID: sess_abc123)
Status: Active
Provider: OpenAI (gpt-4)
Messages: 23
Tokens Used: 2,847 / 4,000
CPU Usage: 2.3%
Memory: 45.2 MB
Uptime: 00:15:32
Last Activity: 2 seconds ago
```

### Session Health Checks
```bash
# Check session health
./vibes-mcp-cli session health --name "my-project"

# Health check includes:
# - Process status
# - Memory usage
# - Resource leaks
# - API connectivity
# - Error rates
```

### Performance Metrics
```bash
# Session performance statistics
./vibes-mcp-cli session metrics --name "my-project"

# Metrics include:
# - Average response time
# - Token usage trends
# - Error rates
# - Memory usage patterns
# - CPU utilization
```

## 🗂️ **Session History**

### Conversation Persistence
Sessions automatically save conversation history:

```bash
# View session history
./vibes-mcp-cli session history --name "my-project"

# Export conversation
./vibes-mcp-cli session export --name "my-project" --format json

# Import conversation
./vibes-mcp-cli session import --name "restored-session" --file backup.json
```

### History Management
```bash
# Clear session history
./vibes-mcp-cli session clear --name "my-project"

# Backup all sessions
./vibes-mcp-cli session backup --output ~/sessions-backup.tar.gz

# Restore from backup
./vibes-mcp-cli session restore --input ~/sessions-backup.tar.gz
```

### Search History
```bash
# Search across session history
./vibes-mcp-cli session search --query "API integration" --session "my-project"

# Search all sessions
./vibes-mcp-cli session search --query "error handling" --all
```

## 🔄 **Session Recovery**

### Automatic Recovery
The system automatically handles:
- **Process Crashes**: Restart failed session processes
- **Memory Leaks**: Detect and cleanup leaked resources
- **Network Issues**: Retry failed API calls
- **Resource Exhaustion**: Free up resources when needed

### Manual Recovery
```bash
# Restart a failed session
./vibes-mcp-cli session restart --name "my-project"

# Force cleanup of stuck session
./vibes-mcp-cli session cleanup --name "my-project" --force

# Recover from backup
./vibes-mcp-cli session recover --name "my-project" --from-backup
```

### Error Handling
```bash
# View session errors
./vibes-mcp-cli session errors --name "my-project"

# Debug session issues
./vibes-mcp-cli session debug --name "my-project"

# Generate diagnostic report
./vibes-mcp-cli session diagnose --name "my-project" --output report.json
```

## 🚧 **Resource Management**

### Memory Leak Prevention
The enhanced session manager prevents memory leaks through:

- **Proper Cleanup**: Automatic resource deallocation
- **Timeout Protection**: Prevent hanging operations
- **Mutex Management**: Avoid deadlocks and race conditions
- **Process Monitoring**: Track resource usage

### Resource Limits
```yaml
# Session resource limits
sessions:
  limits:
    max_memory: "512MB"
    max_cpu_percent: 50
    max_sessions: 10
    idle_timeout: "30m"
    max_message_history: 1000
```

### Cleanup Operations
```bash
# Clean up idle sessions
./vibes-mcp-cli session cleanup --idle --older-than "1h"

# Force cleanup all sessions
./vibes-mcp-cli session cleanup --all --force

# Clean up orphaned resources
./vibes-mcp-cli session cleanup --orphaned
```

## 🔐 **Session Security**

### Session Isolation
- Each session runs in its own process space
- No data sharing between sessions
- Independent API key management
- Isolated error boundaries

### Data Protection
```bash
# Encrypt session data
./vibes-mcp-cli session encrypt --name "sensitive-project"

# Set session password
./vibes-mcp-cli session password --name "my-project" --set

# Lock session
./vibes-mcp-cli session lock --name "my-project"
```

### Audit Trail
```bash
# View session audit log
./vibes-mcp-cli session audit --name "my-project"

# Export audit trail
./vibes-mcp-cli session audit --export --format json
```

## 🎛️ **Advanced Features**

### Session Templates
```bash
# Create session template
./vibes-mcp-cli template create --name "code-review" \
  --provider openai \
  --model gpt-4 \
  --temperature 0.3 \
  --system-prompt "You are a code reviewer..."

# Create session from template
./vibes-mcp-cli session create --template "code-review" --name "pr-123"
```

### Session Sharing
```bash
# Export session for sharing
./vibes-mcp-cli session share --name "my-project" --output shareable.json

# Import shared session
./vibes-mcp-cli session import --shared shareable.json --name "imported-session"
```

### Batch Operations
```bash
# Start multiple sessions
./vibes-mcp-cli session batch-start --pattern "project-*"

# Terminate multiple sessions
./vibes-mcp-cli session batch-terminate --tag "development"

# Export multiple sessions
./vibes-mcp-cli session batch-export --provider openai --output backup/
```

## 📊 **Session Analytics**

### Usage Statistics
```bash
# Session usage report
./vibes-mcp-cli session analytics --name "my-project" --period "7d"

# Token usage breakdown
./vibes-mcp-cli session tokens --name "my-project" --detailed

# Performance analysis
./vibes-mcp-cli session performance --name "my-project" --chart
```

### Cost Tracking
```bash
# Session cost analysis
./vibes-mcp-cli session costs --name "my-project" --currency USD

# Monthly cost report
./vibes-mcp-cli session costs --all --period "month" --summary
```

## 🚨 **Troubleshooting Sessions**

### Common Issues

#### 1. Session Won't Start
```bash
# Check session status
./vibes-mcp-cli session status --name "my-project"

# Verify configuration
./vibes-mcp-cli session config --name "my-project" --validate

# Check provider connectivity
./vibes-mcp-cli provider test --provider openai
```

#### 2. Memory Leaks/High Usage
```bash
# Monitor resource usage
./vibes-mcp-cli session monitor --name "my-project" --memory

# Force cleanup
./vibes-mcp-cli session cleanup --name "my-project" --force

# Restart session
./vibes-mcp-cli session restart --name "my-project"
```

#### 3. Session Freezing
```bash
# Kill frozen session
./vibes-mcp-cli session kill --name "my-project"

# Debug session state
./vibes-mcp-cli session debug --name "my-project" --verbose

# Check for deadlocks
./vibes-mcp-cli session diagnose --name "my-project" --deadlock-check
```

#### 4. History Corruption
```bash
# Verify history integrity
./vibes-mcp-cli session verify --name "my-project"

# Repair corrupted history
./vibes-mcp-cli session repair --name "my-project"

# Restore from backup
./vibes-mcp-cli session restore --name "my-project" --latest-backup
```

### Debug Commands
```bash
# Enable session debugging
./vibes-mcp-cli session debug --name "my-project" --enable

# View debug logs
./vibes-mcp-cli session logs --name "my-project" --level debug

# Generate diagnostic dump
./vibes-mcp-cli session dump --name "my-project" --output debug.tar.gz
```

## 📋 **Best Practices**

### Session Naming
- Use descriptive, project-specific names
- Avoid special characters
- Consider using prefixes for organization
- Keep names under 50 characters

### Resource Management
- Regularly clean up unused sessions
- Monitor memory usage in long-running sessions
- Set appropriate resource limits
- Use session templates for consistency

### Data Safety
- Regularly backup important sessions
- Use encrypted storage for sensitive data
- Set up automated cleanup policies
- Monitor audit trails

### Performance
- Limit message history for large conversations
- Use appropriate models for your use case
- Monitor token usage to control costs
- Clean up idle sessions regularly

---

## 📞 **Need Help?**

- **Session Issues**: [GitHub Issues](https://github.com/your-org/vibes-mcp-cli/issues/new?template=session.md)
- **Performance Problems**: [Performance Guide](Performance)
- **Memory Leaks**: [Troubleshooting Guide](Troubleshooting)

---

*Next: [HTTP Server Mode](HTTP-Server) →*