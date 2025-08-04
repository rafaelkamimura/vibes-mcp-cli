# AI Integration Guide

This guide covers how to integrate the vibes-mcp-cli prompt system with various AI tools and services, including Claude, Context7, Beastmode, and custom integrations.

## Table of Contents

- [Overview](#overview)
- [Claude Integration](#claude-integration)
- [Context7 Integration](#context7-integration)
- [Beastmode Integration](#beastmode-integration)
- [Custom Integrations](#custom-integrations)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

## Overview

The prompt system supports multiple AI integrations that enhance prompt generation and provide additional functionality:

- **Claude**: Direct API integration for prompt processing
- **Context7**: Real-time documentation and context enhancement
- **Beastmode**: Autonomous development workflow automation
- **Custom**: Extensible integration framework for third-party tools

### Integration Types

| Integration | Purpose | Status |
|-------------|---------|--------|
| **Claude** | AI-powered prompt processing and enhancement | ✅ Active |
| **Context7** | Documentation enrichment and real-time context | 🚧 Beta |
| **Beastmode** | Autonomous development and code generation | 🔬 Experimental |
| **Webhooks** | Custom notification and workflow triggers | ✅ Active |
| **Slack** | Team notifications and collaboration | ✅ Active |

## Claude Integration

### Setup

1. **API Key Configuration**
   ```bash
   export OPENAI_CLI_API_KEY="your-anthropic-api-key"
   export OPENAI_CLI_BASE_URL="https://api.anthropic.com"
   export OPENAI_CLI_PROVIDER="anthropic"
   ```

2. **Test Connection**
   ```bash
   vibes-mcp-cli chat "Hello, Claude!"
   ```

3. **Configure Prompt Integration**
   ```bash
   vibes-mcp-cli prompt config set preferred-ai-tool=claude
   ```

### Usage

#### Direct Prompt Sending
```bash
# Send generated prompt directly to Claude
vibes-mcp-cli prompt generate feature-development \
  --repo myproject \
  --language go \
  --send-to-claude
```

#### Interactive Integration
```bash
# Generate prompt and get Claude's response
vibes-mcp-cli prompt generate bug-investigation \
  --severity critical \
  --interactive \
  --send-to-claude
```

#### Batch Processing
```bash
# Process multiple prompts with Claude
for template in feature-development code-review testing-strategy; do
  vibes-mcp-cli prompt generate $template \
    --auto-detect \
    --send-to-claude \
    --output "${template}-response.md"
done
```

### Claude-Specific Features

#### 1. Prompt Enhancement
```bash
# Use Claude to enhance generated prompts
vibes-mcp-cli prompt enhance feature-development \
  --repo myproject \
  --language python
```

#### 2. Quality Feedback
```bash
# Get quality feedback from Claude
vibes-mcp-cli prompt feedback my-template
```

#### 3. Template Optimization
```bash
# Optimize template for specific context
vibes-mcp-cli prompt optimize feature-development \
  --context workspace \
  --language go
```

### Configuration Options

```yaml
ai_integration:
  claude:
    enabled: true
    model: "claude-3-sonnet-20240229"
    max_tokens: 4000
    temperature: 0.7
    timeout: 60
    retry_attempts: 3
    
    # Response processing
    extract_suggestions: true
    extract_improvements: true
    format_output: true
    
    # Integration features
    auto_enhance: false
    quality_scoring: true
    context_optimization: true
```

## Context7 Integration

Context7 provides real-time documentation enrichment and contextual information for prompts.

### Setup

1. **API Key Configuration**
   ```bash
   export CONTEXT7_API_KEY="your-context7-api-key"
   export CONTEXT7_URL="https://api.context7.com"
   ```

2. **Enable Integration**
   ```bash
   vibes-mcp-cli prompt config set context7-enabled=true
   ```

3. **Test Connection**
   ```bash
   vibes-mcp-cli prompt test-integration context7
   ```

### Usage

#### Automatic Documentation Enhancement
```bash
# Generate prompt with Context7 documentation
vibes-mcp-cli prompt generate api-development \
  --language python \
  --framework fastapi \
  --use-context7
```

#### Manual Documentation Queries
```bash
# Query Context7 for specific documentation
vibes-mcp-cli prompt context7-query "FastAPI authentication middleware"
```

#### Context-Aware Generation
```bash
# Generate prompt with real-time context
vibes-mcp-cli prompt generate feature-development \
  --auto-detect \
  --use-context7 \
  --include-docs
```

### Context7 Features

#### 1. Documentation Enrichment
- Automatically fetches relevant documentation
- Includes API references and examples
- Provides framework-specific guidance
- Updates with latest documentation versions

#### 2. Code Example Integration
- Inserts relevant code examples
- Shows best practices and patterns
- Includes error handling examples
- Provides testing strategies

#### 3. Dependency Analysis
- Analyzes project dependencies
- Suggests compatible libraries
- Identifies potential conflicts
- Recommends updates

### Configuration

```yaml
ai_integration:
  context7:
    enabled: true
    url: "https://api.context7.com"
    timeout: 30
    cache_duration: "1h"
    
    # Documentation settings
    include_api_docs: true
    include_examples: true
    include_best_practices: true
    max_documentation_size: 5000
    
    # Query optimization
    auto_query: true
    query_threshold: 0.7
    max_queries_per_prompt: 5
```

### Context7 Workflow

```mermaid
graph TD
    A[Generate Prompt] --> B[Analyze Content]
    B --> C[Extract Documentation Queries]
    C --> D[Query Context7 API]
    D --> E[Retrieve Documentation]
    E --> F[Enrich Prompt Content]
    F --> G[Return Enhanced Prompt]
```

## Beastmode Integration

Beastmode enables autonomous development workflows based on generated prompts.

### Setup

1. **Token Configuration**
   ```bash
   export BEASTMODE_TOKEN="your-beastmode-token"
   export BEASTMODE_URL="http://localhost:8080"
   ```

2. **Enable Integration**
   ```bash
   vibes-mcp-cli prompt config set beastmode-enabled=true
   ```

3. **Test Connection**
   ```bash
   vibes-mcp-cli prompt test-integration beastmode
   ```

### Usage

#### Autonomous Development
```bash
# Trigger autonomous development workflow
vibes-mcp-cli prompt generate feature-development \
  --repo myproject \
  --language go \
  --component authentication \
  --beastmode
```

#### Task Automation
```bash
# Automate testing strategy implementation
vibes-mcp-cli prompt generate testing-strategy \
  --auto-detect \
  --beastmode \
  --execute-workflow
```

#### Multi-Step Workflows
```bash
# Execute complex development workflow
vibes-mcp-cli prompt workflow create-feature \
  --steps "design,implement,test,document" \
  --beastmode
```

### Beastmode Features

#### 1. Workflow Creation
- Analyzes prompts for actionable tasks
- Creates structured development workflows
- Manages task dependencies
- Tracks execution progress

#### 2. Code Generation
- Generates boilerplate code
- Implements basic functionality
- Creates test scaffolds
- Generates documentation

#### 3. Automation Engine
- Executes development tasks
- Runs tests and validations
- Manages file operations
- Handles Git operations

### Configuration

```yaml
ai_integration:
  beastmode:
    enabled: true
    url: "http://localhost:8080"
    timeout: 300
    
    # Workflow settings
    auto_execute: false
    confirm_actions: true
    max_workflow_steps: 10
    
    # Code generation
    generate_tests: true
    generate_docs: true
    follow_conventions: true
    
    # Safety settings
    dry_run_mode: false
    backup_before_changes: true
    require_confirmation: true
```

### Beastmode Workflow Types

#### 1. Feature Development
```yaml
workflow_type: "feature_development"
steps:
  - analyze_requirements
  - design_architecture
  - generate_boilerplate
  - implement_logic
  - create_tests
  - generate_documentation
  - validate_implementation
```

#### 2. Bug Investigation
```yaml
workflow_type: "bug_investigation"
steps:
  - analyze_symptoms
  - reproduce_issue
  - identify_root_cause
  - develop_fix
  - create_regression_tests
  - validate_fix
  - update_documentation
```

#### 3. Performance Optimization
```yaml
workflow_type: "performance_optimization"
steps:
  - profile_application
  - identify_bottlenecks
  - design_optimizations
  - implement_changes
  - benchmark_improvements
  - create_monitoring
  - document_changes
```

## Custom Integrations

The system supports custom integrations through webhooks, APIs, and plugins.

### Webhook Integration

#### Setup
```bash
# Configure webhook endpoints
vibes-mcp-cli prompt config set webhook-url="https://your-webhook.com/prompt-generated"
vibes-mcp-cli prompt config set webhook-events="prompt_generated,template_created"
```

#### Usage
```bash
# Generate prompt with webhook notification
vibes-mcp-cli prompt generate feature-development \
  --repo myproject \
  --notify-webhook
```

#### Webhook Payload
```json
{
  "event": "prompt_generated",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "template": "feature-development",
    "repository": "myproject",
    "language": "go",
    "word_count": 250,
    "parameters": {
      "repo": "myproject",
      "language": "go",
      "component": "authentication"
    }
  },
  "metadata": {
    "source": "vibes-mcp-cli",
    "version": "1.0.0",
    "user": "developer@example.com"
  }
}
```

### Slack Integration

#### Setup
```bash
# Configure Slack webhook
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
vibes-mcp-cli prompt config set slack-notifications=true
```

#### Usage
```bash
# Generate prompt with Slack notification
vibes-mcp-cli prompt generate bug-investigation \
  --severity critical \
  --notify-slack
```

#### Slack Message Format
```
🎯 *Prompt Generated*
Template: `feature-development`
Repository: `myproject`
Language: `go`
Word Count: 250

Generated by vibes-mcp-cli
```

### Custom AI Providers

#### API Integration
```go
// Custom AI provider implementation
type CustomAIProvider struct {
    apiKey    string
    baseURL   string
    model     string
}

func (p *CustomAIProvider) ProcessPrompt(ctx context.Context, prompt string) (*AIResponse, error) {
    // Implementation for custom AI service
}
```

#### Configuration
```yaml
ai_integration:
  custom_providers:
    - name: "openai"
      enabled: true
      api_key_env: "OPENAI_API_KEY"
      base_url: "https://api.openai.com/v1"
      model: "gpt-4"
      
    - name: "cohere"
      enabled: false
      api_key_env: "COHERE_API_KEY"
      base_url: "https://api.cohere.ai/v1"
      model: "command"
```

### Plugin System

#### Creating Custom Plugins
```go
// Plugin interface
type PromptPlugin interface {
    Name() string
    Version() string
    Process(ctx context.Context, prompt *GenerationResult) error
    Configure(config map[string]interface{}) error
}

// Example plugin implementation
type EmailNotificationPlugin struct {
    smtpServer string
    recipients []string
}

func (p *EmailNotificationPlugin) Process(ctx context.Context, prompt *GenerationResult) error {
    return p.sendEmailNotification(prompt)
}
```

#### Plugin Configuration
```yaml
plugins:
  - name: "email-notification"
    enabled: true
    config:
      smtp_server: "smtp.example.com"
      recipients: ["team@example.com"]
      
  - name: "jira-integration"
    enabled: true
    config:
      jira_url: "https://company.atlassian.net"
      project_key: "DEV"
```

## Configuration

### Global Configuration File

```yaml
# ~/.vibes-mcp-cli/config.yaml
ai_integration:
  # Default AI tool
  default_provider: "claude"
  
  # Claude configuration
  claude:
    enabled: true
    model: "claude-3-sonnet-20240229"
    max_tokens: 4000
    temperature: 0.7
    timeout: 60
    
  # Context7 configuration
  context7:
    enabled: false
    url: "https://api.context7.com"
    cache_duration: "1h"
    
  # Beastmode configuration
  beastmode:
    enabled: false
    url: "http://localhost:8080"
    auto_execute: false
    
  # Webhook configuration
  webhooks:
    enabled: false
    endpoints:
      - url: "https://webhook.example.com"
        events: ["prompt_generated"]
        
  # Slack configuration
  slack:
    enabled: false
    webhook_url: ""
    channel: "#development"
```

### Environment Variables

```bash
# Core API configuration
export OPENAI_CLI_API_KEY="your-api-key"
export OPENAI_CLI_BASE_URL="https://api.anthropic.com"
export OPENAI_CLI_PROVIDER="anthropic"

# Context7 integration
export CONTEXT7_API_KEY="your-context7-key"
export CONTEXT7_URL="https://api.context7.com"

# Beastmode integration
export BEASTMODE_TOKEN="your-beastmode-token"
export BEASTMODE_URL="http://localhost:8080"

# Webhook integration
export WEBHOOK_URL="https://your-webhook.com/endpoint"
export WEBHOOK_SECRET="your-webhook-secret"

# Slack integration
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
export SLACK_BOT_TOKEN="xoxb-your-slack-bot-token"

# Custom integrations
export CUSTOM_AI_API_KEY="your-custom-ai-key"
export CUSTOM_AI_URL="https://api.custom-ai.com"
```

### Runtime Configuration

```bash
# Set AI tool preference
vibes-mcp-cli prompt config set preferred-ai-tool=claude

# Enable specific integrations
vibes-mcp-cli prompt config set context7-enabled=true
vibes-mcp-cli prompt config set beastmode-enabled=false

# Configure timeouts and limits
vibes-mcp-cli prompt config set ai-timeout=60
vibes-mcp-cli prompt config set max-tokens=4000
```

## Troubleshooting

### Common Issues

#### 1. Authentication Failures
```
Error: Claude API returned status 401
```

**Solutions:**
- Verify API key: `echo $OPENAI_CLI_API_KEY`
- Check provider setting: `vibes-mcp-cli prompt config get preferred-ai-tool`
- Test with basic chat: `vibes-mcp-cli chat "test"`

#### 2. Network Connectivity
```
Error: failed to connect to Context7: connection timeout
```

**Solutions:**
- Check network connectivity: `curl -I https://api.context7.com`
- Verify URL configuration: `echo $CONTEXT7_URL`
- Check firewall settings
- Test with reduced timeout

#### 3. Integration Configuration
```
Error: Beastmode token not configured
```

**Solutions:**
- Set environment variable: `export BEASTMODE_TOKEN="your-token"`
- Check configuration: `vibes-mcp-cli prompt config list`
- Verify integration is enabled: `vibes-mcp-cli prompt test-integration beastmode`

#### 4. Webhook Delivery Failures
```
Error: webhook returned status 500
```

**Solutions:**
- Check webhook endpoint health
- Verify webhook URL configuration
- Review webhook payload format
- Check webhook secret configuration

### Debug Mode

Enable debug logging for integration troubleshooting:

```bash
export LOG_LEVEL=debug
vibes-mcp-cli prompt generate feature-development --send-to-claude --debug
```

### Integration Testing

#### Test All Integrations
```bash
# Test Claude integration
vibes-mcp-cli prompt test-integration claude

# Test Context7 integration
vibes-mcp-cli prompt test-integration context7

# Test Beastmode integration
vibes-mcp-cli prompt test-integration beastmode

# Test webhook integration
vibes-mcp-cli prompt test-integration webhook

# Test all integrations
vibes-mcp-cli prompt test-integrations --all
```

#### Integration Health Check
```bash
#!/bin/bash
# integration-health-check.sh

echo "Checking AI integrations..."

# Claude
echo "Testing Claude..."
if vibes-mcp-cli prompt test-integration claude >/dev/null 2>&1; then
    echo "✅ Claude: OK"
else
    echo "❌ Claude: Failed"
fi

# Context7
echo "Testing Context7..."
if vibes-mcp-cli prompt test-integration context7 >/dev/null 2>&1; then
    echo "✅ Context7: OK"
else
    echo "❌ Context7: Failed"
fi

# Beastmode
echo "Testing Beastmode..."
if vibes-mcp-cli prompt test-integration beastmode >/dev/null 2>&1; then
    echo "✅ Beastmode: OK"
else
    echo "❌ Beastmode: Failed"
fi

echo "Integration health check complete."
```

### Performance Optimization

#### 1. Caching
```yaml
ai_integration:
  caching:
    enabled: true
    ttl: "1h"
    max_size: "100MB"
    storage: "memory"  # or "file"
```

#### 2. Request Batching
```bash
# Batch multiple prompts for efficiency
vibes-mcp-cli prompt batch-generate \
  --templates "feature-development,code-review,testing" \
  --send-to-claude \
  --batch-size 3
```

#### 3. Async Processing
```yaml
ai_integration:
  async_processing:
    enabled: true
    max_concurrent: 5
    queue_size: 100
    timeout: "5m"
```

### Monitoring and Metrics

#### Integration Metrics
```bash
# View integration usage statistics
vibes-mcp-cli prompt metrics --integrations

# Export metrics to file
vibes-mcp-cli prompt metrics --output metrics.json
```

#### Health Monitoring
```bash
# Monitor integration health
vibes-mcp-cli prompt monitor --interval 60s --alerts-webhook https://alerts.example.com
```

### Security Considerations

#### 1. API Key Management
- Store API keys in environment variables
- Use secret management systems in production
- Rotate keys regularly
- Monitor API key usage

#### 2. Network Security
- Use HTTPS for all API calls
- Implement proper certificate validation
- Consider VPN or private network access
- Monitor network traffic

#### 3. Data Privacy
- Review AI provider data policies
- Implement data sanitization
- Consider on-premises AI solutions
- Monitor data transmission

For more technical details, see the [Architecture Overview](Prompt-Architecture.md) and [API Reference](Prompt-API.md).