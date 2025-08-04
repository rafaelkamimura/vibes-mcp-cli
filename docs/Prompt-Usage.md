# Prompt System User Guide

This guide covers how to use the vibes-mcp-cli prompt system for generating structured AI prompts, managing templates, and integrating with various AI tools.

## Table of Contents

- [Quick Start](#quick-start)
- [Command-Line Interface](#command-line-interface)
- [Interactive TUI](#interactive-tui)
- [Configuration](#configuration)
- [Common Workflows](#common-workflows)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Basic Usage

```bash
# List available templates
vibes-mcp-cli prompt list

# Generate a prompt interactively
vibes-mcp-cli prompt generate feature-development --interactive

# Generate with specific parameters
vibes-mcp-cli prompt generate bug-investigation --repo vibes-mcp-cli --severity critical --clipboard

# Show template details
vibes-mcp-cli prompt show feature-development
```

### First-Time Setup

1. **Initialize Configuration**
   ```bash
   # Set your preferred defaults
   vibes-mcp-cli prompt config set default-repository=your-project
   vibes-mcp-cli prompt config set preferred-language=go
   vibes-mcp-cli prompt config set preferred-ai-tool=claude
   ```

2. **Test AI Integration**
   ```bash
   # Verify Claude integration
   vibes-mcp-cli prompt generate hello-world --send-to-claude
   ```

3. **Explore Templates**
   ```bash
   # See all available templates
   vibes-mcp-cli prompt list

   # Check workspace context
   vibes-mcp-cli prompt workspace-status
   ```

## Command-Line Interface

### Core Commands

#### `prompt list [category]`
Lists available prompt templates, optionally filtered by category.

```bash
# List all templates
vibes-mcp-cli prompt list

# List by category
vibes-mcp-cli prompt list general
vibes-mcp-cli prompt list languages
vibes-mcp-cli prompt list workflows
vibes-mcp-cli prompt list workspace

# Show validation status
vibes-mcp-cli prompt list --validate
```

**Categories:**
- `general`: Repository-agnostic development tasks
- `languages`: Technology-specific optimizations  
- `workflows`: Multi-step orchestration patterns
- `workspace`: Vibes workspace specific optimizations

#### `prompt show <template-name>`
Displays detailed information about a specific template.

```bash
vibes-mcp-cli prompt show feature-development
```

**Output includes:**
- Template description and purpose
- Required parameters and types
- Usage examples
- Validation status

#### `prompt generate <template-name>`
Generates a prompt from the specified template.

**Basic Usage:**
```bash
# Interactive mode (recommended for first use)
vibes-mcp-cli prompt generate feature-development --interactive

# Direct parameter mode
vibes-mcp-cli prompt generate bug-investigation \
  --repo vibes-mcp-cli \
  --language go \
  --severity critical \
  --component "API client"
```

**Parameters:**
- `--category, -c`: Template category filter
- `--repo, -r`: Repository name
- `--language, -l`: Programming language
- `--framework, -f`: Framework name
- `--component`: Component/module name
- `--severity, -s`: Issue severity (critical, high, medium, low)
- `--priority, -p`: Issue priority (p0, p1, p2, p3)

**Generation Modes:**
- `--interactive, -i`: Interactive template filling
- `--auto-detect`: Auto-detect workspace context
- `--validate, -v`: Validate generated prompt (default: true)

**Output Options:**
- `--output, -o`: Save to file
- `--clipboard`: Copy to system clipboard
- `--stdout`: Print to terminal (default)

**AI Integration:**
- `--send-to-claude`: Send directly to Claude API
- `--use-context7`: Use Context7 for documentation
- `--beastmode`: Trigger Beastmode autonomous development

#### `prompt validate [template-name]`
Validates prompt templates for quality and completeness.

```bash
# Validate all templates
vibes-mcp-cli prompt validate

# Validate specific template
vibes-mcp-cli prompt validate feature-development
```

**Validation checks:**
- Template structure and required sections
- Parameter definitions and types
- Content quality and completeness
- Code example syntax and validity
- Placeholder consistency

#### `prompt workspace-status`
Shows current workspace context for prompt generation.

```bash
vibes-mcp-cli prompt workspace-status
```

**Information displayed:**
- Current directory and detected repository
- Primary programming language and framework
- Available languages and recent activity
- Template suggestions based on context

### Template Management

#### `prompt create <template-name>`
Creates a new custom template.

```bash
# Interactive creation
vibes-mcp-cli prompt create custom-optimization --interactive

# Create from existing file
vibes-mcp-cli prompt create team-pattern --from-file template.yaml
```

#### `prompt update <template-name>`
Updates an existing template.

```bash
# Interactive update
vibes-mcp-cli prompt update custom-template --interactive

# Update with validation
vibes-mcp-cli prompt update template-name --validate
```

#### `prompt delete <template-name>`
Deletes a custom template (built-in templates cannot be deleted).

```bash
# Delete with confirmation
vibes-mcp-cli prompt delete old-template

# Force delete without confirmation
vibes-mcp-cli prompt delete unused-template --force
```

### History and Configuration

#### `prompt history`
Shows prompt generation history.

```bash
# Show recent history
vibes-mcp-cli prompt history

# Limit results
vibes-mcp-cli prompt history --limit 10

# Filter by category
vibes-mcp-cli prompt history --filter workflows
```

#### `prompt config`
Manages prompt system configuration.

```bash
# Show current configuration
vibes-mcp-cli prompt config

# Set configuration values
vibes-mcp-cli prompt config set preferred-language=python
vibes-mcp-cli prompt config set auto-clipboard=true
vibes-mcp-cli prompt config set preferred-ai-tool=claude

# Get specific value
vibes-mcp-cli prompt config get default-repository
```

**Configuration Options:**
- `default-repository`: Default repository name
- `preferred-language`: Default programming language
- `preferred-framework`: Default framework
- `auto-clipboard`: Automatically copy to clipboard
- `auto-validate`: Automatically validate prompts
- `preferred-ai-tool`: Default AI tool (claude, context7, beastmode)
- `output-format`: Default output format (markdown, text, json)

## Interactive TUI

The Terminal User Interface (TUI) provides a visual way to work with prompts. Launch it with:

```bash
vibes-mcp-cli ui
```

### TUI Navigation

**Main Interface:**
- Use arrow keys or `hjkl` for navigation
- `Tab` to switch between panels
- `Enter` to select items
- `Esc` to go back or exit
- `q` to quit

**Prompt Browser:**
- Browse templates by category
- Preview template content
- Quick generate with `g`
- Show details with `i`

**Template Editor:**
- Edit template content
- Validate changes with `Ctrl+V`
- Save with `Ctrl+S`
- Cancel with `Esc`

**History View:**
- Browse generation history
- Regenerate previous prompts with `r`
- Copy to clipboard with `c`
- View details with `Enter`

### TUI Shortcuts

| Key | Action |
|-----|--------|
| `?` | Show help |
| `g` | Generate prompt |
| `l` | List templates |
| `c` | Create new template |
| `h` | Show history |
| `s` | Workspace status |
| `v` | Validate templates |
| `/` | Search templates |
| `Ctrl+C` | Exit |

## Configuration

### Configuration File

The prompt system uses `.vibes-mcp-cli.yaml` for configuration:

```yaml
prompt:
  default_repository: "vibes-mcp-cli"
  preferred_language: "go"
  preferred_framework: "cobra"
  auto_clipboard: false
  auto_validate: true
  preferred_ai_tool: "claude"
  output_format: "markdown"
  history_limit: 100
  validation_enabled: true
  backup_enabled: true
  
  # Template directories
  template_directories:
    - "~/.vibes-mcp-cli/templates"
    - "./prompt-templates"
    - "./templates"
  
  custom_templates_path: "~/.vibes-mcp-cli/custom-templates"
  
  # AI integration settings
  ai_integration:
    claude_enabled: true
    claude_model: "claude-3-sonnet-20240229"
    context7_enabled: false
    context7_url: "https://api.context7.com"
    beastmode_enabled: false
    beastmode_url: "http://localhost:8080"
```

### Environment Variables

```bash
# API Configuration
export OPENAI_CLI_API_KEY="your-api-key"
export OPENAI_CLI_BASE_URL="https://api.anthropic.com"
export OPENAI_CLI_PROVIDER="anthropic"

# AI Integration
export CONTEXT7_API_KEY="your-context7-key"
export CONTEXT7_URL="https://api.context7.com"
export BEASTMODE_TOKEN="your-beastmode-token"
export BEASTMODE_URL="http://localhost:8080"

# Prompt Configuration
export PROMPT_DEFAULT_REPO="your-default-repo"
export PROMPT_PREFERRED_LANGUAGE="go"
export PROMPT_AUTO_CLIPBOARD="true"
```

## Common Workflows

### 1. Feature Development

```bash
# Interactive feature development prompt
vibes-mcp-cli prompt generate feature-development --interactive

# Direct with parameters
vibes-mcp-cli prompt generate feature-development \
  --repo myproject \
  --language go \
  --component "authentication" \
  --clipboard \
  --send-to-claude
```

### 2. Bug Investigation

```bash
# Critical bug investigation
vibes-mcp-cli prompt generate bug-investigation \
  --severity critical \
  --repo vibes-mcp-cli \
  --component "API client" \
  --auto-detect

# With Context7 documentation
vibes-mcp-cli prompt generate bug-investigation \
  --severity high \
  --use-context7
```

### 3. Code Review

```bash
# Generate code review prompt
vibes-mcp-cli prompt generate code-review \
  --language python \
  --framework django \
  --output review-checklist.md
```

### 4. Testing Strategy

```bash
# Generate testing strategy
vibes-mcp-cli prompt generate testing-strategy \
  --language go \
  --framework "cobra cli" \
  --beastmode
```

### 5. Documentation

```bash
# API documentation prompt
vibes-mcp-cli prompt generate api-documentation \
  --repo vibes-mcp-cli \
  --component "prompt system" \
  --auto-detect \
  --clipboard
```

### 6. Performance Optimization

```bash
# Performance optimization analysis
vibes-mcp-cli prompt generate performance-optimization \
  --language go \
  --component "HTTP client" \
  --severity medium \
  --use-context7
```

### 7. Custom Template Creation

```bash
# Create team-specific template
vibes-mcp-cli prompt create team-standup --interactive

# Create from existing template file
vibes-mcp-cli prompt create project-template \
  --from-file ./templates/base-template.yaml
```

## Troubleshooting

### Common Issues

#### 1. Template Not Found
```
Error: template 'feature-development' not found
```

**Solutions:**
- Check available templates: `vibes-mcp-cli prompt list`
- Verify template name spelling
- Update template directories in configuration

#### 2. Clipboard Copy Failed
```
Error: failed to copy to clipboard: no clipboard tool found
```

**Solutions:**
- **macOS**: Should work by default
- **Linux**: Install `xclip` or `xsel`
  ```bash
  # Ubuntu/Debian
  sudo apt-get install xclip
  
  # RHEL/CentOS
  sudo yum install xsel
  ```
- **Windows**: Should work by default

#### 3. AI Integration Failed
```
Error: Claude API returned status 401
```

**Solutions:**
- Check API key: `echo $OPENAI_CLI_API_KEY`
- Verify provider configuration: `vibes-mcp-cli prompt config get preferred-ai-tool`
- Test connectivity: `vibes-mcp-cli chat "hello world"`

#### 4. Template Validation Failed
```
Template validation: FAILED
• Missing required parameter: language
• Invalid example syntax
```

**Solutions:**
- Review template structure with: `vibes-mcp-cli prompt show template-name`
- Fix missing parameters in template definition
- Validate syntax in examples section

#### 5. Workspace Context Detection Failed
```
Error: failed to detect workspace context
```

**Solutions:**
- Ensure you're in a valid project directory
- Check Git repository status: `git status`
- Run from project root directory
- Use `--repo` parameter to specify manually

### Debug Mode

Enable debug logging for troubleshooting:

```bash
export LOG_LEVEL=debug
vibes-mcp-cli prompt generate feature-development --interactive
```

### Getting Help

```bash
# Command help
vibes-mcp-cli prompt --help
vibes-mcp-cli prompt generate --help

# Template information
vibes-mcp-cli prompt show template-name

# Workspace diagnostics
vibes-mcp-cli prompt workspace-status

# Validation report
vibes-mcp-cli prompt validate
```

### Performance Tips

1. **Use Auto-Detection**
   ```bash
   vibes-mcp-cli prompt generate template-name --auto-detect
   ```

2. **Enable Validation**
   ```bash
   vibes-mcp-cli prompt config set auto-validate=true
   ```

3. **Cache Templates**
   - Templates are cached after first load
   - Clear cache by restarting the CLI

4. **Batch Operations**
   ```bash
   # Validate all templates
   vibes-mcp-cli prompt validate
   
   # Generate multiple prompts
   for template in feature-development bug-investigation testing; do
     vibes-mcp-cli prompt generate $template --auto-detect --clipboard
   done
   ```

### Support

For additional help:
- Check the [Template Management Guide](Template-Management.md)
- Review [AI Integration Guide](AI-Integration.md) 
- See [Architecture Overview](Prompt-Architecture.md) for technical details
- Report issues on the project repository