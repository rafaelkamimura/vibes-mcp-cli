# Vibes MCP CLI Wiki

Welcome to the comprehensive documentation for **Vibes MCP CLI** - a production-ready, multi-provider command-line interface for LLM APIs with advanced session management and interactive terminal UI.

## 🚀 **What is Vibes MCP CLI?**

Vibes MCP CLI is a sophisticated Go-based CLI tool that provides:
- **Multi-Provider Support**: OpenAI, Anthropic, and more
- **Interactive Terminal UI**: Beautiful terminal interface with session management
- **HTTP Server Mode**: MCP-compatible API endpoints
- **Session Management**: Persistent conversation history and state
- **Resource Monitoring**: Real-time system metrics and telemetry
- **Environment Detection**: Smart TTY detection for containers and headless environments

## 📖 **Quick Navigation**

### 🏗️ **Setup & Installation**
- [Installation Guide](Installation) - Get started in minutes
- [Configuration](Configuration) - Environment setup and provider configuration
- [Docker Setup](Docker) - Container deployment options

### 💻 **Usage Guides**
- [CLI Commands](CLI-Commands) - Complete command reference
- [Interactive Terminal UI](Terminal-UI) - TUI features and keyboard shortcuts
- [HTTP Server Mode](HTTP-Server) - API endpoints and integration
- [Session Management](Session-Management) - Conversation persistence and control

### 🔧 **Development**
- [Development Setup](Development) - Local development environment
- [Architecture](Architecture) - Code structure and design patterns
- [Testing](Testing) - Test coverage and quality assurance
- [Contributing](Contributing) - How to contribute to the project

### 📊 **Features**
- [Telemetry & Monitoring](Telemetry) - Performance metrics and observability
- [Environment Support](Environment-Support) - Cross-platform compatibility
- [Security](Security) - Authentication and secure API handling

### 🚨 **Operations**
- [Troubleshooting](Troubleshooting) - Common issues and solutions
- [Performance Tuning](Performance) - Optimization tips and tricks
- [Deployment](Deployment) - Production deployment strategies

## 🎯 **Key Features Highlights**

### **Production-Ready Session Management** ⭐
- **Zero Memory Leaks**: Eliminated resource leaks and freezing issues
- **Robust Process Control**: Proper session termination and cleanup
- **Real-time Monitoring**: Session status tracking and performance metrics
- **TTY Detection**: Graceful fallbacks for containers and CI/CD environments

### **Advanced Terminal Interface** ⭐
- **Rich TUI**: Interactive terminal UI with session logs viewer
- **Telemetry Dashboard**: Real-time system metrics and ASCII charts
- **Keyboard Navigation**: Full keyboard support with shortcuts
- **Responsive Design**: Adapts to terminal size and capabilities

### **Multi-Provider Architecture** ⭐
- **Unified API**: Consistent interface across different LLM providers
- **Provider Switching**: Runtime provider selection via headers
- **Custom Endpoints**: Support for custom API base URLs
- **Retry Logic**: Intelligent retry mechanisms with exponential backoff

## 📚 **Getting Started**

```bash
# Quick installation
git clone https://github.com/your-org/vibes-mcp-cli.git
cd vibes-mcp-cli
make init    # Setup environment
make build   # Build binary

# Start using
./vibes-mcp-cli chat "Hello, world!"
./vibes-mcp-cli ui                    # Launch TUI
./vibes-mcp-cli serve                 # Start HTTP server
```

## 🏆 **Why Choose Vibes MCP CLI?**

1. **Enterprise-Grade Reliability**: Production-ready with comprehensive error handling
2. **Developer Experience**: Intuitive CLI with rich terminal interface
3. **Extensible Architecture**: Clean codebase with modular design
4. **Performance Optimized**: Efficient resource usage and monitoring
5. **Cross-Platform**: Works on Linux, macOS, Windows, and containers
6. **Well-Documented**: Comprehensive documentation and examples

## 📞 **Support & Community**

- **Issues**: [GitHub Issues](https://github.com/your-org/vibes-mcp-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/vibes-mcp-cli/discussions)
- **Contributing**: See our [Contributing Guide](Contributing)

---

*Last Updated: August 2025 | Version: v0.0.10*