# Template Management Guide

This guide covers everything you need to know about creating, organizing, and managing prompt templates in the vibes-mcp-cli prompt system.

## Table of Contents

- [Template Structure](#template-structure)
- [Creating Templates](#creating-templates)
- [Template Categories](#template-categories)
- [Best Practices](#best-practices)
- [Quality Guidelines](#quality-guidelines)
- [Template Organization](#template-organization)
- [Advanced Features](#advanced-features)

## Template Structure

### Basic Template Format

Templates are defined in YAML format with a specific structure:

```yaml
name: "feature-development"
category: "general"
language: "go"
framework: "cobra"
description: "Generate comprehensive feature development prompts with context"
content: |
  # Feature Development Request

  ## Context
  Repository: {{repo}}
  Language: {{language}}
  Framework: {{framework}}
  Component: {{component}}

  ## Feature Requirements
  Please develop a new feature with the following specifications:
  
  {{feature_description}}

  ## Implementation Details
  - Follow {{language}} best practices
  - Use {{framework}} conventions
  - Include comprehensive tests
  - Add documentation
  
  ## Success Criteria
  1. Feature works as specified
  2. All tests pass
  3. Code follows style guidelines
  4. Documentation is complete

parameters:
  - name: "repo"
    description: "Repository name"
    type: "string"
    required: true
    placeholder: "project-name"
    
  - name: "language"
    description: "Programming language"
    type: "select"
    required: true
    options: ["go", "python", "javascript", "typescript", "java"]
    default: "go"
    
  - name: "framework"
    description: "Framework or library"
    type: "string"
    required: false
    placeholder: "cobra, gin, echo"
    
  - name: "component"
    description: "Component or module name"
    type: "string"
    required: true
    placeholder: "user-authentication"
    
  - name: "feature_description"
    description: "Detailed feature description"
    type: "text"
    required: true
    placeholder: "Describe the feature requirements..."

examples:
  - "vibes-mcp-cli prompt generate feature-development --repo myproject --language go --component auth"
  - "vibes-mcp-cli prompt generate feature-development --interactive"

tags: ["development", "feature", "planning"]
author: "vibes-team"
version: "1.0"
created_at: "2024-01-15T10:00:00Z"
updated_at: "2024-01-15T10:00:00Z"
```

### Required Fields

Every template must include:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique template identifier |
| `category` | string | Template category (general, languages, workflows, workspace) |
| `description` | string | Brief description of the template's purpose |
| `content` | string | Template body with placeholders |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `language` | string | Primary programming language |
| `framework` | string | Framework or technology |
| `parameters` | array | Parameter definitions |
| `examples` | array | Usage examples |
| `tags` | array | Searchable tags |
| `author` | string | Template author |
| `version` | string | Template version |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## Creating Templates

### Interactive Creation

Use the interactive mode for guided template creation:

```bash
vibes-mcp-cli prompt create my-template --interactive
```

The interactive mode will prompt you for:
1. Template name and description
2. Category selection
3. Target language and framework
4. Parameter definitions
5. Content creation with placeholder guidance
6. Usage examples

### File-Based Creation

Create a template from an existing YAML file:

```bash
vibes-mcp-cli prompt create my-template --from-file template.yaml
```

### Manual Creation

1. **Create the template file:**
   ```bash
   mkdir -p ~/.vibes-mcp-cli/custom-templates
   touch ~/.vibes-mcp-cli/custom-templates/my-template.yaml
   ```

2. **Edit the template:**
   ```yaml
   name: "my-template"
   category: "custom"
   description: "My custom template description"
   content: |
     # My Template
     
     Repository: {{repo}}
     Task: {{task}}
     
     Please help me with {{task}} in the {{repo}} repository.
   
   parameters:
     - name: "repo"
       description: "Repository name"
       type: "string"
       required: true
     - name: "task"
       description: "Task description"
       type: "string"
       required: true
   ```

3. **Validate the template:**
   ```bash
   vibes-mcp-cli prompt validate my-template
   ```

## Template Categories

### General Templates
Location: `templates/general/`
Purpose: Repository-agnostic development tasks

Examples:
- `feature-development` - New feature implementation
- `bug-investigation` - Bug analysis and fixing
- `code-review` - Code review checklists
- `documentation` - Documentation generation
- `refactoring` - Code refactoring guidance

### Language Templates
Location: `templates/languages/`
Purpose: Language-specific optimization and patterns

Examples:
- `go/service-implementation` - Go service patterns
- `python/api-development` - Python API best practices
- `javascript/react-component` - React component development
- `typescript/type-definitions` - TypeScript type modeling

### Workflow Templates
Location: `templates/workflows/`
Purpose: Multi-step development processes

Examples:
- `testing-strategy` - Comprehensive testing approach
- `deployment-pipeline` - CI/CD pipeline setup
- `performance-optimization` - Performance analysis workflow
- `security-audit` - Security review process

### Workspace Templates
Location: `templates/workspace/`
Purpose: Vibes workspace specific optimizations

Examples:
- `mcp-integration` - MCP protocol implementation
- `cli-enhancement` - CLI tool improvements
- `tui-development` - Terminal UI components
- `prompt-system` - Prompt system extensions

## Best Practices

### 1. Template Naming

**Good naming conventions:**
```yaml
# Use kebab-case
name: "feature-development"
name: "bug-investigation"
name: "api-documentation"

# Be descriptive but concise
name: "performance-optimization"  # Good
name: "perf-opt"                 # Too abbreviated
name: "performance-optimization-and-analysis-workflow"  # Too long
```

**Category-specific naming:**
```yaml
# Language templates include language prefix
name: "go/service-implementation"
name: "python/api-development"

# Workflow templates describe the process
name: "testing-strategy"
name: "deployment-pipeline"
```

### 2. Parameter Design

**Well-defined parameters:**
```yaml
parameters:
  - name: "repo"
    description: "Repository name (e.g., vibes-mcp-cli)"
    type: "string"
    required: true
    placeholder: "project-name"
    validation: "^[a-zA-Z0-9-_]+$"
    
  - name: "severity"
    description: "Issue severity level"
    type: "select"
    required: true
    options: ["critical", "high", "medium", "low"]
    default: "medium"
    
  - name: "description"
    description: "Detailed problem description"
    type: "text"
    required: true
    placeholder: "Describe the issue in detail..."
```

**Parameter types:**
- `string`: Single-line text input
- `text`: Multi-line text input
- `select`: Dropdown with predefined options
- `int`: Integer number
- `bool`: Boolean true/false
- `file`: File path input

### 3. Content Structure

**Effective template content:**
```yaml
content: |
  # {{title}}
  
  ## Context
  - Repository: {{repo}}
  - Language: {{language}}
  - Component: {{component}}
  
  ## Problem Statement
  {{problem_description}}
  
  ## Requirements
  1. {{requirement_1}}
  2. {{requirement_2}}
  3. {{requirement_3}}
  
  ## Implementation Guidelines
  - Follow {{language}} best practices
  - Use {{framework}} conventions
  - Include comprehensive tests
  - Add proper documentation
  
  ## Success Criteria
  - [ ] Requirement 1 implemented
  - [ ] All tests pass
  - [ ] Code reviewed
  - [ ] Documentation updated
  
  ## Additional Context
  {{additional_context}}
```

**Content guidelines:**
- Use clear headings and structure
- Include actionable requirements
- Provide success criteria
- Add context placeholders
- Use markdown formatting
- Include checklists where appropriate

### 4. Examples and Documentation

**Comprehensive examples:**
```yaml
examples:
  - "vibes-mcp-cli prompt generate feature-development --repo myproject --language go"
  - "vibes-mcp-cli prompt generate feature-development --interactive"
  - "vibes-mcp-cli prompt generate feature-development --auto-detect --clipboard"

# Also include usage scenarios
usage_scenarios:
  - scenario: "New API endpoint"
    description: "Creating a new REST API endpoint"
    command: "vibes-mcp-cli prompt generate api-endpoint --language go --framework gin"
    
  - scenario: "Database integration"
    description: "Adding database operations"
    command: "vibes-mcp-cli prompt generate database-integration --language python --framework sqlalchemy"
```

## Quality Guidelines

### Template Validation

The system automatically validates templates for:

1. **Structure validation:**
   - Required fields present
   - Valid parameter definitions
   - Proper YAML syntax

2. **Content quality:**
   - Clear descriptions
   - Actionable requirements
   - Proper placeholder usage

3. **Parameter validation:**
   - Required parameters defined
   - Valid parameter types
   - Consistent placeholder names

### Quality Checklist

Before submitting a template, ensure:

- [ ] **Clear Purpose**: Template has a specific, well-defined purpose
- [ ] **Good Documentation**: Description explains when and how to use
- [ ] **Proper Parameters**: All placeholders have corresponding parameters
- [ ] **Actionable Content**: Content provides clear, actionable guidance
- [ ] **Good Examples**: Usage examples are comprehensive and realistic
- [ ] **Consistent Style**: Follows established formatting conventions
- [ ] **Validation Passes**: Template passes all validation checks

### Quality Scoring

Templates are scored on:

| Criterion | Weight | Description |
|-----------|--------|-------------|
| **Clarity** | 25% | How clear and understandable is the template |
| **Completeness** | 25% | Are all necessary elements included |
| **Specificity** | 25% | How specific and detailed are the requirements |
| **Actionability** | 25% | How easy is it to act on the generated prompt |

**Score ranges:**
- 90-100: Excellent template, ready for production
- 80-89: Good template, minor improvements needed
- 70-79: Acceptable template, some issues to address
- Below 70: Template needs significant improvements

## Template Organization

### Directory Structure

```
templates/
├── general/
│   ├── feature-development.yaml
│   ├── bug-investigation.yaml
│   ├── code-review.yaml
│   └── documentation.yaml
├── languages/
│   ├── go/
│   │   ├── service-implementation.yaml
│   │   ├── cli-development.yaml
│   │   └── testing-patterns.yaml
│   ├── python/
│   │   ├── api-development.yaml
│   │   ├── data-processing.yaml
│   │   └── web-scraping.yaml
│   └── javascript/
│       ├── react-component.yaml
│       ├── node-api.yaml
│       └── frontend-optimization.yaml
├── workflows/
│   ├── testing-strategy.yaml
│   ├── deployment-pipeline.yaml
│   ├── performance-optimization.yaml
│   └── security-audit.yaml
└── workspace/
    ├── mcp-integration.yaml
    ├── cli-enhancement.yaml
    ├── tui-development.yaml
    └── prompt-system.yaml
```

### Custom Templates

User custom templates are stored in:
```
~/.vibes-mcp-cli/
├── custom-templates/
│   ├── team-standup.yaml
│   ├── project-review.yaml
│   └── deployment-checklist.yaml
└── config.yaml
```

### Template Discovery

The system searches for templates in this order:
1. User custom templates (`~/.vibes-mcp-cli/custom-templates/`)
2. Project templates (`./prompt-templates/`)
3. Local templates (`./templates/`)
4. Built-in templates (embedded in binary)

## Advanced Features

### Template Inheritance

Templates can extend other templates:

```yaml
name: "go-service-development"
extends: "feature-development"
category: "languages"
language: "go"
framework: "gin"

# Override specific parameters
parameters:
  - name: "service_type"
    description: "Type of service (REST API, gRPC, Worker)"
    type: "select"
    options: ["rest-api", "grpc", "worker"]
    required: true

# Extend content
content: |
  {{parent.content}}
  
  ## Go-Specific Requirements
  - Use Go modules for dependency management
  - Follow effective Go guidelines
  - Implement proper error handling
  - Add context support for cancellation
```

### Conditional Content

Use conditional blocks for dynamic content:

```yaml
content: |
  # {{title}}
  
  {{#if language == "go"}}
  ## Go Implementation
  - Use Go modules
  - Follow effective Go patterns
  {{/if}}
  
  {{#if language == "python"}}
  ## Python Implementation
  - Use virtual environments
  - Follow PEP 8 style guide
  {{/if}}
  
  {{#if testing_required}}
  ## Testing Requirements
  - Write unit tests
  - Achieve {{test_coverage}}% coverage
  {{/if}}
```

### Template Macros

Define reusable content blocks:

```yaml
macros:
  go_best_practices: |
    - Use Go modules for dependencies
    - Follow effective Go guidelines
    - Implement proper error handling
    - Use context for cancellation
    - Write comprehensive tests
    
  api_requirements: |
    - RESTful endpoint design
    - Proper HTTP status codes
    - Request/response validation
    - Authentication/authorization
    - Rate limiting

content: |
  # API Development
  
  ## Requirements
  {{>api_requirements}}
  
  {{#if language == "go"}}
  ## Go Best Practices
  {{>go_best_practices}}
  {{/if}}
```

### Template Variables

Use variables for configuration:

```yaml
variables:
  default_test_coverage: 80
  api_version: "v1"
  supported_methods: ["GET", "POST", "PUT", "DELETE"]

content: |
  # API Implementation
  
  API Version: {{api_version}}
  Required Coverage: {{default_test_coverage}}%
  
  Supported HTTP Methods:
  {{#each supported_methods}}
  - {{this}}
  {{/each}}
```

### Template Hooks

Execute actions during template processing:

```yaml
hooks:
  before_generate:
    - validate_workspace
    - check_dependencies
    
  after_generate:
    - copy_to_clipboard
    - notify_team
    
  on_parameter_change:
    - refresh_suggestions
    - validate_combination
```

### Multi-File Templates

Create templates that generate multiple files:

```yaml
name: "full-feature-scaffold"
category: "workflows"
description: "Generate complete feature scaffold with multiple files"

files:
  - path: "{{component}}/handler.go"
    content: |
      package {{component}}
      
      // Handler implementation
      
  - path: "{{component}}/service.go"
    content: |
      package {{component}}
      
      // Service implementation
      
  - path: "{{component}}/repository.go"
    content: |
      package {{component}}
      
      // Repository implementation
      
  - path: "tests/{{component}}_test.go"
    content: |
      package {{component}}_test
      
      // Test implementation
```

## Template Testing

### Validation Testing

```bash
# Validate specific template
vibes-mcp-cli prompt validate my-template

# Validate all templates
vibes-mcp-cli prompt validate

# Validate with detailed output
vibes-mcp-cli prompt validate --verbose
```

### Generation Testing

```bash
# Test generation with minimal parameters
vibes-mcp-cli prompt generate my-template \
  --repo test-repo \
  --language go \
  --validate

# Test interactive mode
vibes-mcp-cli prompt generate my-template --interactive

# Test auto-detection
vibes-mcp-cli prompt generate my-template --auto-detect
```

### Integration Testing

Create test scripts for template validation:

```bash
#!/bin/bash
# test-template.sh

TEMPLATE_NAME="feature-development"

echo "Testing template: $TEMPLATE_NAME"

# Test validation
echo "1. Validating template..."
vibes-mcp-cli prompt validate $TEMPLATE_NAME || exit 1

# Test generation
echo "2. Testing generation..."
vibes-mcp-cli prompt generate $TEMPLATE_NAME \
  --repo test-repo \
  --language go \
  --component test-component \
  --validate || exit 1

# Test interactive mode (with timeout)
echo "3. Testing interactive mode..."
timeout 30s vibes-mcp-cli prompt generate $TEMPLATE_NAME --interactive <<EOF
test-repo
go
cobra
test-component
Test feature description
Additional context
EOF

echo "Template test completed successfully!"
```

## Contributing Templates

### Submission Process

1. **Create the template** following the guidelines
2. **Validate the template** using the CLI tools
3. **Test thoroughly** with various scenarios
4. **Document usage** with clear examples
5. **Submit for review** through the standard process

### Review Criteria

Templates are reviewed for:
- **Quality**: Meets quality guidelines and scoring criteria
- **Uniqueness**: Provides value not covered by existing templates
- **Clarity**: Clear documentation and examples
- **Testing**: Comprehensive testing completed
- **Standards**: Follows naming and organization conventions

### Template Lifecycle

1. **Draft**: Initial creation and testing
2. **Review**: Community and maintainer review
3. **Accepted**: Approved for inclusion
4. **Published**: Available in the system
5. **Maintained**: Regular updates and improvements
6. **Deprecated**: Marked for removal if obsolete

For more information on contributing, see the [Development Guide](Prompt-Development.md).