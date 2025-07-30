# User Guide

This comprehensive user guide will help you get started with the vibes-mcp-cli Claude Code session manager, from initial setup through advanced workflows. Whether you're new to the system or looking to master advanced features, this guide has you covered.

## Table of Contents

- [Getting Started](#getting-started)
- [First Steps](#first-steps)
- [File Navigation Workflows](#file-navigation-workflows)
- [Session Management Workflows](#session-management-workflows)
- [Integration Workflows](#integration-workflows)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Tips and Best Practices](#tips-and-best-practices)

## Getting Started

### System Requirements

**Minimum Requirements:**
- Go 1.20 or later
- Terminal with 80x24 minimum size
- 512MB RAM available
- 100MB disk space for sessions

**Recommended:**
- Go 1.21 or later
- Terminal with 120x40 size
- 2GB RAM available
- 1GB disk space for sessions
- Claude Code executable installed

**Supported Platforms:**
- Linux (Ubuntu 20.04+, CentOS 8+)
- macOS (10.15+)
- Windows (WSL2 recommended)

### Installation

#### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/your-org/vibes-mcp-cli.git
cd vibes-mcp-cli

# Copy and configure environment
cp .env_example .env
# Edit .env with your API keys and configuration

# Build the application
make build

# Optional: Install globally
sudo cp vibes-mcp-cli /usr/local/bin/
```

#### Option 2: Download Pre-built Binary

```bash
# Download for your platform
curl -L https://github.com/your-org/vibes-mcp-cli/releases/latest/download/vibes-mcp-cli-linux-amd64 -o vibes-mcp-cli
chmod +x vibes-mcp-cli

# Move to PATH
sudo mv vibes-mcp-cli /usr/local/bin/
```

#### Option 3: Docker

```bash
# Pull the image
docker pull your-org/vibes-mcp-cli:latest

# Run with local directory mounted
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.vibes-mcp-cli:/root/.vibes-mcp-cli \
  your-org/vibes-mcp-cli:latest ui
```

### Initial Configuration

#### 1. Environment Configuration

Create your configuration file at `~/.vibes-mcp-cli.yaml`:

```yaml
# API Configuration
api_key: "your-openai-or-claude-api-key"
base_url: "https://api.openai.com"
provider: "openai"  # or "anthropic"

# Session Management
session_manager:
  storage_path: "~/.vibes-sessions"
  max_sessions: 10
  auto_cleanup: true
  cleanup_interval: "2h"
  claude_path: "claude"  # Path to Claude Code executable

# Security Settings
security:
  allowed_paths:
    - "~/projects"
    - "~/workspace"
    - "/tmp"
  forbidden_paths:
    - "/etc"
    - "/root"
    - "~/.ssh"
  max_file_size: 10485760  # 10MB
  allow_hidden: false

# UI Configuration
ui:
  theme: "dark"
  show_line_numbers: true
  enable_mouse: true
  vim_mode: false

# Logging
log_level: "info"
```

#### 2. Claude Code Setup

Ensure Claude Code is available in your system:

```bash
# Check if Claude Code is installed
which claude

# If not installed, download from Anthropic
# Or set custom path in configuration
```

#### 3. Verify Installation

```bash
# Test basic functionality
vibes-mcp-cli --version
vibes-mcp-cli --help

# Test configuration
vibes-mcp-cli ui --dry-run
```

## First Steps

### 1. Launch the UI

```bash
# Basic launch
vibes-mcp-cli ui

# With custom options
vibes-mcp-cli ui \
  --root-path ~/projects \
  --max-sessions 5 \
  --theme dark
```

### 2. Understanding the Interface

When you first launch the UI, you'll see:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 📍 /home/user/projects                                       [Sessions: 0/10] │
├─────────────────────────────────────────────────────────────────────────────┤
│ 📁 File Explorer                    │ 💬 No Active Session                   │
│                                     │                                         │
│ Welcome to vibes-mcp-cli!           │ To get started:                        │
│                                     │ 1. Navigate files on the left          │
│ Press F1 for help                   │ 2. Press Ctrl+N for new session       │
│ Press H to toggle hidden files      │ 3. Press M to send files to Claude    │
│                                     │                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│ Status: Ready - Welcome!                               F1=Help F2=Sessions   │
│ Enter=Open M=Claude A=Add /=Search R=Refresh H=Hidden Q=Quit               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3. Create Your First Session

1. **Press `Ctrl+N`** to create a new session
2. **Enter session name**: e.g., "my-first-session"
3. **Configure working directory**: Use current directory or specify another
4. **Start the session**: Session will appear in the right panel

```
┌─ New Session ──────────────────────────┐
│ Name: [my-first-session              ] │
│ Working Dir: [/home/user/projects    ] │  
│ Environment Variables:                 │
│ ┌────────────────────────────────────┐ │
│ │ PATH=/usr/local/bin:/usr/bin:/bin  │ │
│ │                                    │ │
│ └────────────────────────────────────┘ │
│ [ ] Auto-save session                  │
│ [Create] [Cancel]                      │
└────────────────────────────────────────┘
```

### 4. Your First File Operation

1. **Navigate to a file** using arrow keys
2. **Preview the file** by pressing `Enter`
3. **Send to Claude Code** by pressing `M`
4. **See the result** in the session panel

## File Navigation Workflows

### Basic Navigation

#### Directory Traversal

```bash
# Navigation keys
↑↓        # Move up/down in file list
←→        # Collapse/expand directories  
Enter     # Enter directory or preview file
Backspace # Go to parent directory
Home/End  # Jump to first/last item
```

#### File Operations

```bash
# File actions
M         # Send file to active Claude session
A         # Add file to session file list
V         # Toggle file preview pane
C         # Copy file path to clipboard
R         # Refresh current directory
H         # Toggle hidden files visibility
```

### Advanced File Navigation

#### Search Operations

**Quick Search** (press `/`):
```
🔍 Search: *.py
Results: 15 files found
├── 📄 main.py
├── 📄 utils.py  
├── 📄 config.py
└── 📁 tests/
    ├── 📄 test_main.py
    └── 📄 test_utils.py
```

**Advanced Search** (press `?`):
```
┌─ Advanced Search ─────────────────────────┐
│ Pattern: [api.*\.(py|json)             ] │
│ ☑ Regex   □ Case sensitive              │
│ File types: [Python, JSON             ▼] │
│ Max results: [50                       ] │
│ ☑ Search content   □ Include hidden      │
│ Search in: [Current directory        ▼] │
│ [Search] [Cancel]                        │
└─────────────────────────────────────────┘
```

#### File Type Recognition

The system automatically recognizes file types and displays appropriate icons:

| Pattern | File Type | Icon | Actions Available |
|---------|-----------|------|-------------------|
| `*.py` | Python | 🐍 | Send to Claude, Preview, Edit |
| `*.go` | Go | 🐹 | Send to Claude, Preview, Edit |
| `*.js`, `*.ts` | JavaScript | 📜 | Send to Claude, Preview, Edit |
| `*.json` | JSON | 📋 | Send to Claude, Preview, Validate |
| `*.md` | Markdown | 📝 | Send to Claude, Preview, Render |
| `*.jpg`, `*.png` | Image | 🖼️ | Preview (metadata only) |
| `*.zip`, `*.tar.gz` | Archive | 📦 | Extract, List contents |

### Working with Large Directories

#### Performance Tips

```bash
# For directories with many files
1. Use search instead of browsing all files
2. Filter by file type using advanced search
3. Navigate directly using breadcrumb paths
4. Use pagination for very large results
```

#### Batch Operations

```bash
# Select multiple files
Space     # Toggle selection on current file
Ctrl+A    # Select all files in directory
Ctrl+D    # Deselect all files

# Batch operations on selected files
M         # Send all selected files to Claude
A         # Add all selected files to session
```

## Session Management Workflows

### Creating Sessions

#### Basic Session Creation

```bash
# Quick session with defaults
Ctrl+N → Enter name → Enter

# Custom session configuration
Ctrl+N → Configure → Create
```

#### Session Templates

Create session templates for common workflows:

**Development Session**:
```yaml
name: "development"
working_dir: "~/projects/current"
environment:
  NODE_ENV: "development"
  DEBUG: "true"
auto_save: true
max_history: 2000
```

**Testing Session**:
```yaml
name: "testing"
working_dir: "~/projects/current"
environment:
  NODE_ENV: "test"
  CI: "true"
auto_save: true
max_history: 1000
```

### Managing Multiple Sessions

#### Session Switching

```bash
# Session management keys  
Ctrl+T       # New session tab
Ctrl+Tab     # Switch to next session
Ctrl+Shift+Tab # Switch to previous session
Ctrl+W       # Close current session
```

#### Session States

Monitor session states in the status panel:

```
┌─ Active Sessions ─────────────────────────┐
│ ● development      [Active]    01:23:45   │
│ ⏸ testing         [Paused]    00:15:30   │ 
│ ● background       [Active]    02:45:12   │
│ ❌ old-session    [Error]     --:--:--   │
└───────────────────────────────────────────┘
```

**State Actions**:
- **Active** → Pause (`Ctrl+P`), Terminate (`Ctrl+T`), Send input
- **Paused** → Resume (`Ctrl+P`), Terminate (`Ctrl+T`)
- **Error** → Restart, Terminate, View logs
- **Terminated** → Delete, Archive

### Session I/O Operations

#### Sending Input

```bash
# Basic input
Type your message → Enter

# Multi-line input  
Type first line → Ctrl+Enter → Continue typing → Enter

# File content input
Select file → M (automatically formats and sends)

# Command shortcuts
help        # Show Claude Code help
exit        # Terminate session gracefully  
clear       # Clear session output
status      # Show session status
```

#### Managing Output

```bash
# Output navigation
Page Up/Down  # Scroll through output
Ctrl+L        # Clear output display
Ctrl+S        # Save output to file
Ctrl+F        # Search in output
```

### Session Persistence

#### Auto-Save Features

Sessions automatically save:
- Input history
- Output logs
- Session state
- Working directory
- Environment variables

#### Manual Save Operations

```bash
Ctrl+S        # Save current session
Ctrl+Shift+S  # Save all sessions
```

#### Session Recovery

When restarting the application:
1. Previously active sessions are restored
2. Session history is preserved
3. Working directories are maintained
4. Environment variables are restored

## Integration Workflows

### File-to-Session Integration

#### Sending Files to Claude

**Single File**:
1. Navigate to file
2. Press `M` to send to active session
3. File content is formatted with context

**Multiple Files**:
1. Select files using `Space` or `Ctrl+A`
2. Press `M` to send all selected files
3. Files are sent with relationship context

**File Context Example**:
```
> Sending file: main.py (Python, 156 lines)
> Project context: /home/user/projects/myapp
> 
> File content:
> #!/usr/bin/env python3
> """
> Main application entry point
> """
> ...
```

#### Building Session Context

**Add Files to Session** (`A` key):
1. Files are added to session context
2. Context is maintained across interactions
3. Files can be referenced by name in conversation

**Session Context Panel**:
```
┌─ Session Context ─────────────────────────┐
│ Files in context:                         │
│ 📄 main.py (156 lines)                   │
│ 📄 config.json (23 lines)                │
│ 📄 README.md (45 lines)                  │
│                                           │
│ [Clear Context] [Save Context]            │
└───────────────────────────────────────────┘
```

### Project Workflow Examples

#### Code Review Workflow

```bash
1. Navigate to project directory
2. Create session: "code-review"
3. Add relevant files to context (A key)
4. Send specific file for review (M key)
5. Discuss changes with Claude
6. Save session for later reference
```

#### Debugging Workflow

```bash
1. Create session: "debug-session"
2. Send error logs to Claude (M key)
3. Send relevant source files (M key)
4. Work through debugging steps
5. Document solution in session
```

#### Learning Workflow

```bash
1. Create session: "learning-python"
2. Send example files (M key)
3. Ask questions about code patterns
4. Request explanations and improvements
5. Save session as learning reference
```

## Advanced Usage

### Custom Keyboard Shortcuts

Configure custom shortcuts in `~/.vibes-mcp-cli.yaml`:

```yaml
ui:
  keybindings:
    global:
      quit: "Ctrl+Q"
      help: "F1"
      refresh: "Ctrl+R"
    file_explorer:
      send_to_claude: "M"
      add_to_session: "A"
      quick_search: "/"
      advanced_search: "?"
    session:
      new_session: "Ctrl+N"
      close_session: "Ctrl+W"
      save_session: "Ctrl+S"
```

### Security Configuration

#### Path Access Control

```yaml
security:
  # Allowed base paths
  allowed_paths:
    - "~/projects"
    - "~/workspace" 
    - "~/Documents/code"
    - "/tmp"
  
  # Explicitly forbidden paths
  forbidden_paths:
    - "/etc"
    - "/root"
    - "~/.ssh"
    - "~/.gnupg"
  
  # File size limits (bytes)
  max_file_size: 10485760  # 10MB
  
  # Directory depth limits
  max_depth: 20
  
  # Hidden file access
  allow_hidden: false
```

#### Resource Limits

```yaml
session_manager:
  # Maximum concurrent sessions
  max_sessions: 15
  
  # Session timeout (inactive sessions)
  session_timeout: "4h"
  
  # Automatic cleanup interval
  cleanup_interval: "2h"
  
  # Maximum session history entries
  max_history_per_session: 5000
  
  # Maximum session storage per session (MB)
  max_session_storage: 100
```

### Automation and Scripting

#### Batch Operations

Create scripts for common operations:

```bash
#!/bin/bash
# send-project-to-claude.sh

# Start vibes-mcp-cli in batch mode
vibes-mcp-cli session create "batch-review" \
  --working-dir "$(pwd)" \
  --auto-start

# Send all Python files
find . -name "*.py" -type f | while read file; do
  vibes-mcp-cli session send "batch-review" --file "$file"
done

# Send project overview
vibes-mcp-cli session send "batch-review" \
  --message "Please review this Python project structure and provide feedback."
```

#### Session Templates

Define reusable session templates:

```yaml
# ~/.vibes-mcp-cli/templates/development.yaml
name: "development-${PROJECT_NAME}"
working_dir: "${PROJECT_ROOT}"
environment:
  NODE_ENV: "development"
  DEBUG: "true"
  PROJECT_ROOT: "${PROJECT_ROOT}"
auto_save: true
max_history: 3000
initial_files:
  - "README.md"
  - "package.json"
  - "src/main.*"
```

### Performance Optimization

#### Large Project Handling

```bash
# For large projects (>1000 files)
1. Use .vibeignore file to exclude unnecessary files
2. Configure appropriate max_file_size limits
3. Use specific searches instead of browsing
4. Limit session history to reasonable sizes
```

#### Memory Management

```yaml
# Optimize for lower memory usage
session_manager:
  max_sessions: 5
  max_history_per_session: 1000
  cleanup_interval: "30m"

file_explorer:
  preview_max_lines: 500
  cache_previews: false
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Application Won't Start

**Symptoms**: 
- Application exits immediately
- "Permission denied" errors
- "File not found" errors

**Solutions**:

```bash
# Check permissions
ls -la $(which vibes-mcp-cli)
chmod +x vibes-mcp-cli

# Check configuration
vibes-mcp-cli --config-check

# Check dependencies
which claude  # Verify Claude Code is available
```

#### 2. File Explorer Shows No Files

**Symptoms**:
- Empty file explorer
- "No files found" message
- Permission errors in status

**Solutions**:

```bash
# Check directory permissions
ls -la /path/to/directory

# Check security configuration
vibes-mcp-cli --show-config | grep -A 10 security

# Try different root path
vibes-mcp-cli ui --root-path ~/
```

#### 3. Sessions Won't Start

**Symptoms**:
- Session creation succeeds but won't start
- "Executor error" messages
- Sessions stuck in "Created" state

**Solutions**:

```bash
# Check Claude Code installation
which claude
claude --version

# Check session storage permissions
ls -la ~/.vibes-sessions/
mkdir -p ~/.vibes-sessions
chmod 755 ~/.vibes-sessions

# Check resource limits
ulimit -a
```

#### 4. High Memory Usage

**Symptoms**:
- Application consumes excessive memory
- System becomes slow
- Out of memory errors

**Solutions**:

```bash
# Reduce session limits
vibes-mcp-cli ui --max-sessions 3

# Clear session history
rm -rf ~/.vibes-sessions/*/history/

# Optimize configuration
```

```yaml
session_manager:
  max_sessions: 5
  max_history_per_session: 500
  cleanup_interval: "15m"
```

#### 5. UI Rendering Issues

**Symptoms**:
- Garbled text display
- Incorrect colors
- Layout problems

**Solutions**:

```bash
# Check terminal compatibility
echo $TERM
export TERM=xterm-256color

# Reset terminal
reset

# Try different theme
vibes-mcp-cli ui --theme light

# Check terminal size
tput cols
tput lines
```

### Debug Mode

Enable debug mode for detailed troubleshooting:

```bash
# Enable debug logging
vibes-mcp-cli ui --debug --log-level debug

# Save debug output
vibes-mcp-cli ui --debug --log-file debug.log

# Verbose session information
vibes-mcp-cli session list --verbose
```

### Getting Help

#### Built-in Help

```bash
# General help
vibes-mcp-cli --help

# Subcommand help  
vibes-mcp-cli ui --help
vibes-mcp-cli session --help

# In-UI help
Press F1 for context-sensitive help
```

#### Log Analysis

```bash
# View application logs
tail -f ~/.vibes-mcp-cli/logs/app.log

# View session logs
tail -f ~/.vibes-sessions/*/logs/session.log

# Search for errors
grep -i error ~/.vibes-mcp-cli/logs/app.log
```

## Tips and Best Practices

### Workflow Optimization

#### Efficient File Navigation

1. **Use Search Extensively**: Don't browse large directories manually
2. **Set Up Project Roots**: Configure common project directories
3. **Use File Type Filters**: Narrow down search results by file type
4. **Leverage Breadcrumbs**: Quick navigation to parent directories

#### Session Management

1. **Use Descriptive Names**: Make session purpose clear
2. **Organize by Project**: Create separate sessions for different projects
3. **Regular Cleanup**: Remove old terminated sessions
4. **Save Important Sessions**: Preserve valuable conversations

#### Performance Tips

1. **Limit History**: Don't accumulate unlimited session history
2. **Close Unused Sessions**: Free up system resources
3. **Use Appropriate File Limits**: Balance functionality with performance
4. **Regular Maintenance**: Clean up old session data

### Security Best Practices

#### File Access Security

1. **Configure Allowed Paths**: Restrict access to necessary directories only
2. **Exclude Sensitive Directories**: Never allow access to system directories
3. **Review File Permissions**: Ensure proper file system permissions
4. **Monitor Access Logs**: Regular review of file access patterns

#### Session Security

1. **Don't Store Secrets**: Avoid sending passwords or API keys to Claude
2. **Review Session Content**: Be mindful of sensitive information in sessions
3. **Clean Up Sessions**: Remove sessions containing sensitive data
4. **Use Environment Variables**: For configuration, not secrets

### Productivity Enhancements

#### Keyboard Mastery

Focus on learning these essential shortcuts:
- `Ctrl+N`: New session (most used)
- `M`: Send file to Claude (most used)
- `/`: Quick search (most used)
- `Ctrl+Tab`: Switch sessions
- `F1`: Help when stuck

#### Configuration Optimization

Customize your setup for maximum productivity:

```yaml
# ~/.vibes-mcp-cli.yaml
ui:
  theme: "dark"  # Easier on eyes for long sessions
  vim_mode: true  # If you're a vim user
  enable_mouse: true  # Helpful for beginners

file_explorer:
  show_hidden: false  # Cleaner view
  preview_max_lines: 1000  # Good balance

session_manager:
  auto_cleanup: true  # Reduces maintenance
  max_sessions: 8  # Good for most users
```

#### Workflow Templates

Create standardized workflows for common tasks:

**Code Review Template**:
1. Create session: "review-YYYYMMDD"
2. Add project context files
3. Send specific files for review
4. Document decisions in session
5. Save session for future reference

**Learning Template**:
1. Create session: "learn-TOPIC"
2. Send example files
3. Ask structured questions
4. Request exercises or examples
5. Build knowledge base over time

---

This comprehensive user guide covers everything from initial setup to advanced workflows. As you become more comfortable with the system, you'll develop your own efficient patterns and discover new ways to leverage the powerful integration between file navigation and Claude Code sessions.

For technical details, see the [API Reference](API-Reference.md), and for implementation details, see the [Architecture](Architecture.md) documentation.