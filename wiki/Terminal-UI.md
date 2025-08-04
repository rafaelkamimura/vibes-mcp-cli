# Interactive Terminal UI Guide

The Vibes MCP CLI features a sophisticated Terminal User Interface (TUI) built with TView, providing a rich interactive experience for managing LLM conversations and monitoring system performance.

## 🎯 **Overview**

The Terminal UI mode offers:
- **Rich Interactive Interface**: Full-featured TUI with multiple panels and views
- **Session Management**: Create, switch, and manage conversation sessions
- **Real-time Monitoring**: Live telemetry data, system metrics, and session logs
- **Keyboard Navigation**: Complete keyboard-driven interface with shortcuts
- **Environment Adaptation**: Smart detection of terminal capabilities and fallbacks

## 🚀 **Launching the TUI**

### Basic Usage
```bash
# Launch TUI with default settings
./vibes-mcp-cli ui

# Launch with debug information
./vibes-mcp-cli ui --debug

# Force fallback mode (for headless environments)
./vibes-mcp-cli ui --fallback-mode
```

### Environment Detection
The TUI automatically detects your environment:

```bash
# ✅ Native terminal (full features)
./vibes-mcp-cli ui

# ✅ SSH session (full features with detection)
ssh user@server './vibes-mcp-cli ui'

# ✅ Docker container (automatic fallback)
docker run -it vibes-mcp-cli ui

# ✅ CI/CD environment (text-only fallback)
TERM="" ./vibes-mcp-cli ui
```

## 🎨 **Interface Components**

### Main Dashboard
The main dashboard provides an overview of:
- **Active Sessions**: List of conversation sessions with status
- **System Metrics**: CPU, memory, and network usage
- **Provider Status**: API connectivity and rate limits
- **Recent Activity**: Latest interactions and events

### Session Management Panel
- **Session List**: All conversation sessions with metadata
- **Session Details**: Individual session information and statistics
- **Session Controls**: Start, stop, delete, and rename sessions
- **Session History**: Previous conversations and context

### Telemetry Dashboard
Real-time monitoring with:
- **ASCII Charts**: Performance metrics visualization
- **System Health**: Resource usage and alerts
- **API Metrics**: Request/response statistics
- **Error Tracking**: Failed requests and recovery

### Logs Viewer
Comprehensive logging interface:
- **Real-time Logs**: Live log streaming with filtering
- **Log Levels**: Debug, info, warn, error filtering
- **Search Function**: Text search across log entries
- **Export Options**: Save logs to file

## ⌨️ **Keyboard Shortcuts**

### Global Navigation
| Key | Action |
|-----|--------|
| `Tab` | Next panel/field |
| `Shift+Tab` | Previous panel/field |
| `Ctrl+C` | Exit application |
| `Ctrl+R` | Refresh current view |
| `F1` | Help/About |
| `F5` | Force refresh |
| `Esc` | Cancel/Back |

### Session Management
| Key | Action |
|-----|--------|
| `n` | New session |
| `d` | Delete session |
| `r` | Rename session |
| `Enter` | Open/Switch to session |
| `Space` | Toggle session selection |
| `↑/↓` | Navigate session list |
| `Ctrl+N` | Quick new session |

### Chat Interface
| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | New line in message |
| `Ctrl+L` | Clear chat history |
| `Ctrl+S` | Save conversation |
| `Ctrl+O` | Open conversation |
| `PageUp/PageDown` | Scroll chat history |

### Telemetry & Monitoring
| Key | Action |
|-----|--------|
| `m` | Toggle metrics view |
| `t` | Toggle telemetry dashboard |
| `l` | Toggle logs viewer |
| `c` | Clear metrics/logs |
| `e` | Export data |
| `f` | Filter/search |

### Advanced Features
| Key | Action |
|-----|--------|
| `Ctrl+T` | Toggle theme |
| `Ctrl+P` | Provider settings |
| `Ctrl+Settings` | Configuration |
| `F9` | Debug mode toggle |
| `F10` | Performance profiler |

## 🎛️ **Interface Modes**

### Full TUI Mode (Default)
Rich interactive interface with all features:
- Multiple panels and views
- Mouse support (if available)
- Color themes and styling
- Real-time updates
- Full keyboard navigation

```bash
# Automatically selected on capable terminals
./vibes-mcp-cli ui
```

### Simplified Mode
Reduced interface for limited terminals:
- Single panel focus
- Essential features only
- Reduced color usage
- Keyboard-only navigation

```bash
# Automatically triggered on basic terminals
TERM=vt100 ./vibes-mcp-cli ui
```

### Text-Only Fallback
Command-line style interface for headless environments:
- Menu-driven interaction
- Text-based prompts
- No special terminal features
- Full functionality preserved

```bash
# For CI/CD, containers without TTY
TERM="" ./vibes-mcp-cli ui --fallback-mode
```

## 📊 **Session Management**

### Creating Sessions
```bash
# From TUI: Press 'n' or Ctrl+N
# Session creation dialog will appear:
Session Name: [my-session]
Provider: [openai] ↑↓ to select
Model: [gpt-4] ↑↓ to select
Temperature: [0.7]
```

### Session Information
Each session displays:
- **Name**: User-defined session identifier
- **Status**: Active, idle, terminated, error
- **Provider**: OpenAI, Anthropic, etc.
- **Model**: Specific model being used
- **Messages**: Conversation length
- **Tokens**: Token usage statistics
- **Duration**: Session runtime
- **Last Activity**: Timestamp of last interaction

### Session Operations
- **Start**: Initialize new conversation
- **Resume**: Continue existing conversation
- **Pause**: Temporarily stop session
- **Terminate**: End session and cleanup
- **Delete**: Remove session and history
- **Export**: Save conversation to file
- **Duplicate**: Clone session configuration

## 📈 **Telemetry Dashboard**

### Real-time Metrics
The telemetry dashboard shows:

```
┌─ System Performance ─────────────────────────────┐
│ CPU Usage    ████████░░ 80%                      │
│ Memory       ██████░░░░ 60%                      │
│ Network I/O  ███░░░░░░░ 30%                      │
│                                                  │
│ Sessions     Active: 3  Idle: 1  Total: 5       │
│ API Calls    Success: 142  Failed: 2            │
│ Tokens       Used: 15.2K  Remaining: 84.8K      │
└──────────────────────────────────────────────────┘
```

### Performance Charts
ASCII-based charts showing:
- Response time trends
- Token usage over time
- API success rates
- Resource utilization

### Health Monitoring
- **Connection Status**: API connectivity
- **Rate Limits**: Current usage vs limits
- **Error Rates**: Failed request tracking
- **Resource Alerts**: High usage warnings

## 🔍 **Logs Viewer**

### Log Display
```
[2025-08-04 04:49:23] INFO  Session created: chat-session-1
[2025-08-04 04:49:24] DEBUG API request to openai: /chat/completions
[2025-08-04 04:49:25] INFO  Response received (200ms, 156 tokens)
[2025-08-04 04:49:26] WARN  Rate limit approaching: 80% used
[2025-08-04 04:49:30] ERROR Session timeout: chat-session-3
```

### Log Filtering
- **Level Filter**: Show only specific log levels
- **Text Search**: Search for specific terms
- **Time Range**: Filter by timestamp
- **Session Filter**: Show logs for specific session
- **Component Filter**: Filter by system component

### Log Export
Export logs in various formats:
- **Plain Text**: Simple text file
- **JSON**: Structured log data
- **CSV**: Spreadsheet-compatible format

## 🎨 **Themes and Customization**

### Available Themes
- **Default**: Standard blue/white theme
- **Dark**: Dark mode with reduced eye strain
- **High Contrast**: Accessibility-focused theme
- **Minimal**: Reduced colors for basic terminals

### Theme Switching
```bash
# Toggle theme in TUI
Ctrl+T

# Set theme via environment
OPENAI_CLI_THEME=dark ./vibes-mcp-cli ui

# Set in config file
theme: "dark"
```

### Custom Styling
Customize colors and styles in `.openai-cli.yaml`:
```yaml
ui:
  theme: "custom"
  colors:
    primary: "#0066cc"
    secondary: "#333333"
    success: "#00cc66"
    warning: "#ffcc00"
    error: "#cc0066"
```

## 🚨 **Troubleshooting TUI Issues**

### Common Issues

#### 1. Display Corruption
```bash
# Terminal not properly detected
TERM=xterm-256color ./vibes-mcp-cli ui

# Force fallback mode
./vibes-mcp-cli ui --fallback-mode

# Reset terminal
reset && ./vibes-mcp-cli ui
```

#### 2. Keyboard Not Working
```bash
# Check terminal capabilities
echo $TERM
tput keys

# Try different terminal
TERM=screen ./vibes-mcp-cli ui
```

#### 3. No Colors/Formatting
```bash
# Enable color support
export COLORTERM=truecolor
./vibes-mcp-cli ui

# Check color capability
tput colors
```

#### 4. Performance Issues
```bash
# Reduce update frequency
./vibes-mcp-cli ui --refresh-rate=1000  # 1 second

# Disable animations
./vibes-mcp-cli ui --no-animations
```

### Debug Mode
Enable debug mode for troubleshooting:
```bash
./vibes-mcp-cli ui --debug

# Debug output will show:
# - Terminal detection results
# - Capability detection
# - Rendering performance
# - Event handling
```

## 📱 **Responsive Design**

### Terminal Size Adaptation
The TUI automatically adapts to different terminal sizes:

#### Large Screens (≥120 columns)
- Full dashboard with all panels
- Side-by-side session and telemetry views
- Detailed metrics and charts

#### Medium Screens (80-119 columns)
- Tabbed interface
- Essential panels only
- Condensed information display

#### Small Screens (<80 columns)
- Single panel mode
- Menu navigation
- Essential features only

### Mobile Terminals
For mobile SSH clients:
- Touch-friendly navigation
- Simplified keyboard shortcuts
- Reduced information density

## 🔗 **Integration with CLI Commands**

### Launching Specific Views
```bash
# Start with specific session
./vibes-mcp-cli ui --session=my-session

# Start with telemetry dashboard
./vibes-mcp-cli ui --view=telemetry

# Start with logs viewer
./vibes-mcp-cli ui --view=logs
```

### Passing Configuration
```bash
# Override provider
./vibes-mcp-cli ui --provider=anthropic

# Set debug level
./vibes-mcp-cli ui --log-level=debug

# Load specific config
./vibes-mcp-cli ui --config=/path/to/config.yaml
```

---

## 📞 **Need Help?**

- **TUI Issues**: [GitHub Issues](https://github.com/your-org/vibes-mcp-cli/issues/new?template=tui.md)
- **Keyboard Shortcuts**: Press `F1` in the TUI
- **Terminal Compatibility**: [Environment Support Guide](Environment-Support)

---

*Next: [Session Management Guide](Session-Management) →*