# Terminal UI (TUI) Guide

The enhanced TUI provides a comprehensive interface for Claude Code session management, file navigation, and LLM interactions. This modern interface features modular components, advanced file exploration, and integrated session management.

## Overview

The new TUI architecture consists of several integrated components:

- **File Explorer**: Advanced file navigation with security features
- **Session Manager**: Multi-session Claude Code management
- **Chat Interface**: Enhanced LLM interaction with session integration
- **Status Monitoring**: Real-time system and session status
- **Configuration Panel**: Runtime configuration management

## Launching the Enhanced UI

```bash
vibes-mcp-cli ui [OPTIONS]
```

### Command Line Options

```bash
vibes-mcp-cli ui \
  --model gpt-4 \                    # LLM model to use
  --root-path /home/user/projects \  # File explorer root
  --session-storage ./sessions \     # Session storage directory
  --max-sessions 10 \                # Maximum concurrent sessions
  --theme dark \                     # UI theme
  --vim-mode                         # Enable vim-style navigation
```

### Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `--root-path` | File explorer root directory | Current directory |
| `--session-storage` | Session storage path | `./sessions` |
| `--max-sessions` | Maximum concurrent sessions | `10` |
| `--theme` | UI theme (`dark`, `light`) | `dark` |
| `--vim-mode` | Enable vim-style navigation | `false` |
| `--show-hidden` | Show hidden files by default | `false` |
| `--enable-mouse` | Enable mouse support | `true` |

## UI Components and Layout

### Main Interface Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 📍 /home/user/projects/myapp                                    [Sessions: 3] │ Breadcrumb
├─────────────────────────────────────────────────────────────────────────────┤
│ 📁 File Explorer                    │ 💬 Session: development-main           │
│ ├── 📁 src/                         │                                         │
│ │   ├── 📄 main.py                  │ > help                                  │
│ │   ├── 📄 utils.py                 │ Claude Code Help:                       │
│ │   └── 📁 components/              │ - Type 'help' for this message         │
│ ├── 📁 tests/                       │ - Type 'exit' to end session           │
│ ├── 📄 README.md                    │ - Use Ctrl+C to interrupt              │
│ ├── 📄 requirements.txt             │                                         │ Main Content
│ └── 📄 .gitignore                   │ > _                                     │
│                                     │                                         │
│ 👁️ Preview: main.py                 │ 📊 Session Status                       │
│ #!/usr/bin/env python3             │ State: Active                           │
│ import sys                          │ Uptime: 00:15:32                       │
│ import argparse                     │ CPU: 2.1%  Memory: 45MB               │
│                                     │ I/O: 1.2KB in, 3.4KB out              │
├─────────────────────────────────────────────────────────────────────────────┤
│ Status: Ready                                          F1=Help F2=Sessions  │ Status Bar
│ Enter=Open M=Claude A=Add /=Search R=Refresh H=Hidden Q=Quit               │ Help Text
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. File Explorer Component

**Location**: Left side of main interface
**Purpose**: Secure file navigation with Claude Code integration

**Features**:
- Tree-based directory navigation
- File type detection with icons
- Real-time file preview
- Advanced search capabilities
- Security-validated file access
- Direct Claude Code integration

**Key Bindings**:
- `↑↓` - Navigate files/directories
- `Enter` - Open directory or preview file
- `M` - Send file to Claude Code
- `A` - Add file to session
- `V` - Toggle preview pane
- `/` - Activate search mode
- `H` - Toggle hidden files
- `R` - Refresh directory
- `Backspace` - Navigate to parent directory

#### 2. Session Manager Component

**Location**: Right side of main interface
**Purpose**: Multi-session Claude Code management

**Features**:
- Multiple concurrent sessions
- Real-time I/O streaming
- Session state monitoring
- Resource usage tracking
- Session persistence

**Key Bindings**:
- `Ctrl+N` - Create new session
- `Ctrl+T` - Switch session tabs
- `Ctrl+W` - Close current session
- `Ctrl+S` - Save session
- `Ctrl+P` - Pause/Resume session

#### 3. Status and Monitoring

**Location**: Bottom panel and sidebar
**Purpose**: System and session status monitoring

**Information Displayed**:
- Session count and states
- Resource usage (CPU, memory)
- I/O statistics
- Error counts
- Navigation breadcrumbs

## Navigation and Keyboard Shortcuts

### Global Navigation

| Key Combination | Action |
|-----------------|--------|
| `F1` | Show help/toggle help panel |
| `F2` | Session management panel |
| `F3` | File search panel |
| `F4` | Configuration panel |
| `Ctrl+Q` | Quit application |
| `Ctrl+R` | Refresh all components |
| `Tab` | Cycle between components |
| `Shift+Tab` | Reverse cycle between components |

### File Explorer Navigation

| Key | Action |
|-----|--------|
| `↑↓` | Navigate files and directories |
| `←→` | Collapse/expand directories |
| `Enter` | Open directory or preview file |
| `Space` | Toggle file selection |
| `Ctrl+A` | Select all files |
| `Ctrl+D` | Deselect all files |
| `Home` | Go to first item |
| `End` | Go to last item |
| `Page Up/Down` | Page navigation |

### File Operations

| Key | Action |
|-----|--------|
| `M` | Send file to active Claude Code session |
| `A` | Add file to session file list |
| `V` | Toggle file preview |
| `C` | Copy file path to clipboard |
| `D` | Delete file (with confirmation) |
| `N` | Rename file |
| `P` | Create new file |
| `Shift+P` | Create new directory |

### Search Operations

| Key | Action |
|-----|--------|
| `/` | Activate search mode |
| `?` | Advanced search options |
| `Ctrl+F` | Find in current directory |
| `Ctrl+Shift+F` | Global search |
| `Esc` | Exit search mode |
| `Enter` | Execute search |
| `F3` | Find next |
| `Shift+F3` | Find previous |

### Session Management

| Key | Action |
|-----|--------|
| `Ctrl+N` | Create new session |
| `Ctrl+O` | Open/resume session |
| `Ctrl+S` | Save current session |
| `Ctrl+Shift+S` | Save all sessions |
| `Ctrl+W` | Close current session |
| `Ctrl+T` | New session tab |
| `Ctrl+Tab` | Switch between session tabs |
| `Ctrl+Shift+Tab` | Reverse tab switching |

### Session Interaction

| Key | Action |
|-----|--------|
| `Enter` | Send input to session |
| `Ctrl+Enter` | Send multi-line input |
| `Ctrl+C` | Interrupt current operation |
| `Ctrl+D` | Send EOF to session |
| `Ctrl+L` | Clear session output |
| `Ctrl+U` | Clear current input |
| `Page Up/Down` | Scroll session output |

## Advanced Features

### File Explorer Features

#### Search Functionality

**Quick Search** (`/`):
```
🔍 Search: *.py
```
- Glob pattern matching
- Real-time results
- File type filtering

**Advanced Search** (`?`):
```
┌─ Advanced Search ─────────────────────────┐
│ Pattern: [config.*\.(json|yaml)         ] │
│ □ Regex   ☑ Case sensitive              │
│ File types: [JSON, YAML               ▼] │
│ Max results: [100                      ] │
│ □ Search content   □ Include hidden      │
│ [Search] [Cancel]                        │
└─────────────────────────────────────────┘
```

#### File Type Detection

The system automatically detects file types and provides appropriate icons:

| File Type | Icon | Color |
|-----------|------|-------|
| Go files | 🐹 | Yellow |
| Python files | 🐍 | Yellow |
| JavaScript files | 📜 | Yellow |
| JSON files | 📋 | Blue |
| Markdown files | 📝 | White |
| Images | 🖼️ | Purple |
| Archives | 📦 | Red |
| Executables | ⚡ | Green |

#### Security Features

- **Path validation**: Prevents directory traversal attacks
- **Access control**: Respects allowlist/denylist configuration
- **Resource limits**: Prevents large file operations from blocking UI
- **Permission checking**: Validates file permissions before operations

### Session Management Features

#### Multi-Session Support

```
┌─ Active Sessions ─────────────────────────┐
│ ● development-main      [Active]   15:32  │
│ ⏸ test-session        [Paused]    02:15  │
│ ● background-task      [Active]    45:20  │
│ ❌ old-session        [Error]      --:--  │
└───────────────────────────────────────────┘
```

**Session States**:
- 🟢 **Active**: Running and accepting input
- ⏸️ **Paused**: Temporarily suspended
- 🔴 **Terminated**: Stopped and can be restarted  
- ❌ **Error**: Error state requiring attention

#### Session Configuration

Create sessions with custom configurations:

```go
sessionConfig := &session.SessionConfig{
    Name:        "development",
    WorkingDir:  "/workspace/myproject",
    Environment: map[string]string{
        "PYTHONPATH": "/workspace/lib",
        "DEBUG":      "true",
    },
    AutoSave:   true,
    MaxHistory: 2000,
}
```

#### Real-Time Monitoring

Monitor session resources in real-time:
- CPU usage percentage
- Memory consumption
- I/O statistics (bytes in/out)
- Process count
- Error count
- Uptime

### Integration Features

#### File-to-Session Integration

**Send File to Claude Code** (`M` key):
1. Select file in explorer
2. Press `M` to send to active session
3. File content is automatically formatted with context
4. Syntax highlighting information included

**Add to Session** (`A` key):
1. Select multiple files
2. Press `A` to add to session file list
3. Files can be batch-processed
4. Session maintains file context

#### Context-Aware Operations

The system maintains context between file operations and session interactions:

```
> You've selected main.py (Python file, 245 lines)
> 
> Content:
> #!/usr/bin/env python3
> """
> Main application entry point
> """
> import sys
> import argparse
> ...
```

## Themes and Customization

### Theme Configuration

**Dark Theme** (default):
- Dark background with light text
- Syntax highlighting optimized for dark backgrounds
- Reduced eye strain for long sessions

**Light Theme**:
- Light background with dark text  
- High contrast for better readability
- Suitable for bright environments

### Color Scheme

| Element | Dark Theme | Light Theme |
|---------|------------|-------------|
| Background | `#1a1a1a` | `#ffffff` |
| Text | `#ffffff` | `#000000` |
| Selected | `#404040` | `#e0e0e0` |
| Borders | `#606060` | `#a0a0a0` |
| Error | `#ff6b6b` | `#d63031` |
| Success | `#51cf66` | `#00b894` |
| Warning | `#ffd43b` | `#fdcb6e` |

### Customization Options

Configure appearance in `~/.vibes-mcp-cli.yaml`:

```yaml
ui:
  theme: "dark"              # dark, light
  show_line_numbers: true    # Show line numbers in preview
  tab_size: 4                # Tab size for code display
  enable_mouse: true         # Enable mouse support
  vim_mode: false            # Enable vim-style navigation
  font_size: 12              # Terminal font size hint
  
file_explorer:
  show_hidden: false         # Show hidden files by default
  tree_indent: 2             # Tree indentation spaces
  preview_max_lines: 1000    # Maximum lines in preview
  
session_manager:
  max_history_display: 500   # Maximum history lines to display
  auto_scroll: true          # Auto-scroll to new output
  timestamp_format: "15:04"  # Timestamp format
```

## Error Handling and Recovery

### Common Error Scenarios

#### File Access Errors

**Security Violation**:
```
❌ Error: Access denied to /etc/passwd
Security policy prevents access to system files
```

**File Not Found**:
```
❌ Error: File not found: /nonexistent/file.txt
Check the file path and permissions
```

#### Session Errors

**Session Unresponsive**:
```
⚠️ Warning: Session 'development' is not responding
Options: [Restart] [Terminate] [Wait]
```

**Resource Limit Exceeded**:
```
❌ Error: Maximum sessions reached (10/10)
Close inactive sessions or increase the limit
```

### Recovery Actions

The UI provides automated recovery for common issues:

1. **Automatic Retry**: Failed operations are automatically retried
2. **Session Recovery**: Crashed sessions can be automatically restarted
3. **State Persistence**: Session state is preserved across restarts
4. **Graceful Degradation**: UI remains functional even with component failures

## Performance Optimization

### Lazy Loading

- Directory contents loaded on-demand
- File previews generated only when needed
- Session output streamed efficiently

### Memory Management

- Automatic cleanup of old session data
- Efficient tree data structures
- Garbage collection of unused resources

### Responsive Design

- Non-blocking operations prevent UI freezing
- Progressive loading for large directories
- Efficient rendering for smooth scrolling

## Troubleshooting

### Common Issues

1. **UI Not Responsive**:
   - Check terminal size (minimum 80x24)
   - Verify no background processes blocking
   - Restart with `--debug` flag

2. **File Explorer Empty**:
   - Check root path permissions
   - Verify security configuration
   - Check for hidden files with `H` key

3. **Session Won't Start**:
   - Verify Claude executable path
   - Check session storage permissions
   - Review resource limits

4. **High Memory Usage**:
   - Clear session history
   - Reduce max history settings
   - Close unused sessions

### Debug Mode

Enable debug mode for troubleshooting:

```bash
vibes-mcp-cli ui --debug --log-level debug
```

This provides detailed logging of:
- Component initialization
- File system operations
- Session management events
- Error conditions and recovery

---

This comprehensive TUI guide covers all aspects of the enhanced Terminal UI in vibes-mcp-cli. For more technical details, see the [API Reference](API-Reference.md) and [Architecture](Architecture.md) documentation.