# Vibes MCP CLI Documentation

Welcome to the comprehensive documentation for the vibes-mcp-cli Claude Code session manager. This enhanced system transforms the original MCP CLI into a powerful, secure, and scalable session management platform with advanced file navigation and Claude Code integration.

## 🚀 Quick Start

New to the system? Start here:

1. **[Getting Started](User-Guide.md#getting-started)** - Installation and initial setup
2. **[First Steps](User-Guide.md#first-steps)** - Your first session and file operations
3. **[TUI Guide](TUI-Usage.md)** - Master the terminal interface

## 📚 Documentation Structure

### For Users

**[User Guide](User-Guide.md)**
- Installation and configuration
- File navigation workflows
- Session management
- Troubleshooting and tips

**[TUI Usage Guide](TUI-Usage.md)**
- Terminal interface overview
- Keyboard shortcuts and navigation
- Advanced features and customization
- Themes and performance optimization

### For Developers

**[Developer Guide](Developer-Guide.md)**
- Development environment setup
- Extending core components
- Adding UI components
- Testing patterns and best practices

**[API Reference](API-Reference.md)**
- Complete API documentation
- Interface definitions
- Configuration options
- Usage examples

### For System Administrators

**[Architecture Overview](Architecture.md)**
- System architecture and design patterns
- Component interactions and data flow
- Migration from legacy architecture
- Performance considerations

**[Security Guide](Security-Guide.md)**
- Multi-layer security architecture
- File system security and validation
- Process isolation and resource limits
- Security best practices and incident response

**[Deployment Guide](Deployment-Guide.md)**
- Production deployment strategies
- Container and Kubernetes deployment
- Configuration management
- Monitoring and maintenance

### Specialized Documentation

**[Session Management](Session-Management.md)**
- Session lifecycle and operations
- Multi-session orchestration
- I/O streaming and capture
- Resource monitoring and cleanup

**[File Navigation System](File-Navigation-System.md)**
- Enhanced file navigation architecture
- Security features and validation
- Search capabilities and file type detection
- Claude Code integration workflows

### Legacy Documentation

**[Getting Started](Getting-Started.md)** - Legacy setup guide
**[Configuration](Configuration.md)** - Legacy configuration
**[CLI Usage](CLI-Usage.md)** - Legacy CLI commands
**[HTTP Server](HTTP-Server.md)** - Legacy HTTP server
**[Providers](Providers.md)** - Legacy provider system
**[Testing](Testing.md)** - Legacy testing guide
**[Contributing](Contributing.md)** - Legacy contribution guide
**[Roadmap](Roadmap.md)** - Legacy roadmap

## 🎯 Key Features

### 🔐 Enterprise Security
- **Path traversal prevention** with multiple validation layers
- **Access control** with allowlist/denylist configuration
- **Process isolation** with resource limits and sandboxing
- **Audit logging** for compliance and monitoring

### 📁 Advanced File Navigation
- **Tree-based navigation** with lazy loading for performance
- **Advanced search** with regex support and file type filtering
- **File type detection** with 20+ supported languages
- **Security validation** for all file operations

### 💬 Claude Code Integration
- **Multi-session support** with up to 50 concurrent sessions
- **Real-time I/O streaming** with capture and replay
- **Session persistence** across application restarts
- **Resource monitoring** with CPU, memory, and I/O tracking

### 🎨 Modern TUI
- **Component-based architecture** with reusable widgets
- **Vim-style navigation** with customizable key bindings
- **Theme support** with dark/light modes
- **Mouse support** for hybrid interaction

## 🏗️ Architecture Highlights

### Layered Design
```
┌─── Presentation Layer (TUI/CLI) ───┐
├─── Application Layer (Core Logic) ─┤
├─── Service Layer (API Coordination) ┤
└─── Infrastructure Layer (External) ─┘
```

### Core Components
- **Session Manager**: Multi-session orchestration and lifecycle management
- **File Navigator**: Secure file system operations with enterprise security
- **Claude Executor**: Process management with resource monitoring
- **UI Components**: Modern, reusable TUI widgets

### Security Architecture
- **Input Validation Layer**: XSS prevention and parameter validation
- **Access Control Layer**: Path validation and permission enforcement
- **Process Isolation Layer**: Resource limits and sandboxing
- **Audit Layer**: Comprehensive logging and monitoring

## 🔧 Configuration

### Quick Configuration
```yaml
# ~/.vibes-mcp-cli.yaml
api_key: "your-openai-or-claude-api-key"
provider: "openai"

security:
  allowed_paths:
    - "~/projects"
    - "~/workspace"
  max_file_size: 10485760  # 10MB

session_manager:
  max_sessions: 10
  storage_path: "~/.vibes-sessions"
  auto_cleanup: true

ui:
  theme: "dark"
  vim_mode: false
  enable_mouse: true
```

### Environment Variables
```bash
export OPENAI_API_KEY="your-api-key"
export VIBES_MAX_SESSIONS="10"
export VIBES_LOG_LEVEL="info"
```

## 🚦 System Status

### Current Version: 2.0.0

#### ✅ Completed Features
- Multi-session Claude Code management
- Enhanced file navigation with security
- Modern TUI with component architecture
- Comprehensive security framework
- Session persistence and recovery
- Real-time I/O streaming
- Advanced search and file type detection

#### 🔄 In Development
- Plugin system for extensibility
- Web-based management interface
- Advanced analytics dashboard
- Integration with additional LLM providers

#### 🎯 Roadmap
- Collaborative session features
- Git integration and version control
- Advanced AI-powered code analysis
- Mobile companion app

## 🤝 Getting Help

### Documentation
- **User Issues**: See [User Guide](User-Guide.md#troubleshooting)
- **Development Questions**: See [Developer Guide](Developer-Guide.md)
- **Security Concerns**: See [Security Guide](Security-Guide.md)
- **Deployment Issues**: See [Deployment Guide](Deployment-Guide.md)

### Support Channels
- **GitHub Issues**: Bug reports and feature requests
- **Discussions**: Community Q&A and general discussion
- **Security Issues**: Private security disclosure

## 📋 Migration Guide

### From Legacy vibes-mcp-cli

The new system maintains backward compatibility with existing CLI commands while adding powerful new features:

#### What's New
- **Multi-session management** replaces single-session model
- **Enhanced security** with comprehensive validation
- **Modern TUI** replaces basic terminal interface
- **File navigation** adds secure file system integration

#### Migration Steps
1. **Backup existing configuration** and data
2. **Install new version** alongside existing
3. **Update configuration** to use new format
4. **Test functionality** with existing workflows
5. **Switch to new version** and remove old

#### Compatibility
- **CLI commands**: All existing commands supported
- **Configuration**: Automatic migration available
- **Data**: Sessions and history preserved
- **APIs**: Extended with new endpoints

## 🏆 Key Benefits

### For Individual Developers
- **Enhanced productivity** with advanced file navigation
- **Better code context** with multi-file session support
- **Improved security** with safe file access controls
- **Modern interface** with intuitive keyboard navigation

### For Teams
- **Standardized workflows** with consistent session management
- **Security compliance** with audit logging and access controls
- **Scalable deployment** with container and Kubernetes support
- **Monitoring capabilities** with comprehensive metrics

### For Organizations
- **Enterprise security** with multi-layer validation
- **Compliance features** with audit trails and access logs
- **Resource management** with limits and monitoring
- **Deployment flexibility** with multiple deployment patterns

---

## 🔗 Quick Links

| Topic | Link | Description |
|-------|------|-------------|
| **Getting Started** | [User Guide](User-Guide.md) | Installation and first steps |
| **TUI Interface** | [TUI Usage](TUI-Usage.md) | Terminal interface guide |
| **API Reference** | [API Docs](API-Reference.md) | Complete API documentation |
| **Architecture** | [Architecture](Architecture.md) | System design and patterns |
| **Security** | [Security Guide](Security-Guide.md) | Security features and best practices |
| **Development** | [Developer Guide](Developer-Guide.md) | Extending and contributing |
| **Deployment** | [Deployment Guide](Deployment-Guide.md) | Production deployment |
| **Sessions** | [Session Management](Session-Management.md) | Session workflows and patterns |
| **File Navigation** | [File Navigation](File-Navigation-System.md) | File system integration |

Welcome to the future of Claude Code session management! 🎉