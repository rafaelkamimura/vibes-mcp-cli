# Installation Guide

This guide covers all installation methods for Vibes MCP CLI, from quick setup to advanced deployment scenarios.

## 🚀 **Quick Start**

### Prerequisites
- **Go 1.23+** with toolchain go1.24.0
- **Git** for cloning the repository
- **Make** for build automation (optional but recommended)

### 1. Clone the Repository
```bash
git clone https://github.com/your-org/vibes-mcp-cli.git
cd vibes-mcp-cli
```

### 2. Initial Setup
```bash
# Copy environment configuration and install dependencies
make init

# This runs:
# - cp .env_example .env (if .env doesn't exist)
# - go mod tidy
```

### 3. Configure Environment
Edit the `.env` file with your API credentials:
```bash
# Required: API key for your chosen provider
OPENAI_CLI_API_KEY=your_api_key_here

# Provider selection (openai, anthropic)
OPENAI_CLI_PROVIDER=openai

# Optional: Custom API base URL
OPENAI_CLI_BASE_URL=https://api.openai.com/v1

# Optional: Interactive mode password
PROMPT_MODE_PASSWORD=your_secure_password
```

### 4. Build and Test
```bash
# Build the binary
make build

# Test your installation
./vibes-mcp-cli --version
./vibes-mcp-cli --help
```

## 📦 **Installation Methods**

### Method 1: Source Build (Recommended)
```bash
# Clone and build from source
git clone https://github.com/your-org/vibes-mcp-cli.git
cd vibes-mcp-cli
make init
make build

# Binary will be created as ./vibes-mcp-cli
```

### Method 2: Go Install
```bash
# Install directly via Go (requires Go 1.23+)
go install github.com/your-org/vibes-mcp-cli@latest

# Note: Binary will be installed as 'openai-cli' due to go.mod module name
```

### Method 3: Docker
```bash
# Build Docker image
make docker-build

# Run via Docker Compose
make docker-up

# Or run directly
docker run -it --rm vibes-mcp-cli:latest --help
```

### Method 4: Pre-built Binaries
Download from the [Releases page](https://github.com/your-org/vibes-mcp-cli/releases):

```bash
# Linux (x86_64)
wget https://github.com/your-org/vibes-mcp-cli/releases/download/v0.0.10/vibes-mcp-cli-linux-amd64
chmod +x vibes-mcp-cli-linux-amd64
sudo mv vibes-mcp-cli-linux-amd64 /usr/local/bin/vibes-mcp-cli

# macOS (Apple Silicon)
wget https://github.com/your-org/vibes-mcp-cli/releases/download/v0.0.10/vibes-mcp-cli-darwin-arm64
chmod +x vibes-mcp-cli-darwin-arm64
sudo mv vibes-mcp-cli-darwin-arm64 /usr/local/bin/vibes-mcp-cli

# Windows
# Download vibes-mcp-cli-windows-amd64.exe from releases
```

## 🔧 **Environment Configuration**

### Required Environment Variables
```bash
# API Key (required)
OPENAI_CLI_API_KEY=sk-...                    # OpenAI API key
# OR
OPENAI_CLI_API_KEY=claude-...                # Anthropic API key

# Provider Selection (required)
OPENAI_CLI_PROVIDER=openai                   # or 'anthropic'
```

### Optional Environment Variables
```bash
# API Configuration
OPENAI_CLI_BASE_URL=https://api.openai.com/v1    # Custom API endpoint
OPENAI_CLI_MODEL=gpt-4                           # Default model
OPENAI_CLI_TEMPERATURE=0.7                       # Response creativity

# Security
PROMPT_MODE_PASSWORD=secure_password              # Interactive mode protection

# Logging & Debug
OPENAI_CLI_LOG_LEVEL=info                        # debug, info, warn, error
OPENAI_CLI_DEBUG=false                           # Enable debug mode
```

### Configuration File
Alternatively, create `.openai-cli.yaml` in your home directory or project root:

```yaml
# ~/.openai-cli.yaml
api_key: "your_api_key_here"
provider: "openai"
base_url: "https://api.openai.com/v1"
model: "gpt-4"
temperature: 0.7
prompt_mode_password: "secure_password"
log_level: "info"
```

## 🐳 **Docker Installation**

### Using Docker Compose (Recommended)
```bash
# Clone repository
git clone https://github.com/your-org/vibes-mcp-cli.git
cd vibes-mcp-cli

# Setup environment
cp .env_example .env
# Edit .env with your configuration

# Build and run
make docker-build
make docker-up
```

### Manual Docker Build
```bash
# Build image
docker build -t vibes-mcp-cli .

# Run interactively
docker run -it --rm \
  -e OPENAI_CLI_API_KEY=your_key \
  -e OPENAI_CLI_PROVIDER=openai \
  vibes-mcp-cli chat "Hello world"

# Run HTTP server
docker run -d -p 8080:8080 \
  -e OPENAI_CLI_API_KEY=your_key \
  -e OPENAI_CLI_PROVIDER=openai \
  vibes-mcp-cli serve --host 0.0.0.0 --port 8080
```

## 🔍 **Verification**

### Test Basic Functionality
```bash
# Check version
./vibes-mcp-cli --version

# Test chat functionality
./vibes-mcp-cli chat "Hello, can you help me?"

# Test TUI (interactive mode)
./vibes-mcp-cli ui

# Test HTTP server
./vibes-mcp-cli serve --port 8080 &
curl http://localhost:8080/v1/models
```

### Test Environment Detection
```bash
# Test TTY detection
./vibes-mcp-cli ui --debug

# In container
docker run -it vibes-mcp-cli ui

# In CI/CD (headless)
TERM="" ./vibes-mcp-cli ui --fallback-mode
```

## 🚨 **Troubleshooting Installation**

### Common Issues

#### 1. Go Version Too Old
```bash
# Error: "go: requires Go 1.23 or later"
# Solution: Update Go
go version  # Check current version
# Visit https://golang.org/dl/ to download Go 1.23+
```

#### 2. Module Name Confusion
```bash
# Error: Binary not found after go install
# The module name in go.mod is 'openai-cli', not 'vibes-mcp-cli'
# Check your $GOPATH/bin for 'openai-cli' binary
ls $GOPATH/bin | grep cli
```

#### 3. Missing Dependencies
```bash
# Error: "make: command not found"
# Install make on your system:

# Ubuntu/Debian
sudo apt-get install build-essential

# macOS
xcode-select --install

# Alternative: Use Go directly
go mod tidy
go build -o vibes-mcp-cli .
```

#### 4. Permission Issues
```bash
# Error: Permission denied
chmod +x vibes-mcp-cli

# For system-wide installation
sudo cp vibes-mcp-cli /usr/local/bin/
```

#### 5. Docker Issues
```bash
# Error: Docker daemon not running
sudo systemctl start docker  # Linux
# Or start Docker Desktop on macOS/Windows

# Error: Permission denied
sudo usermod -aG docker $USER  # Add user to docker group
# Then log out and back in
```

## 📈 **Performance Optimization**

### Build Optimizations
```bash
# Build with optimizations
go build -ldflags="-s -w" -o vibes-mcp-cli .

# Cross-compile for production
make release  # Builds for linux/amd64, darwin/amd64, windows/amd64
```

### Resource Limits
```bash
# For containers, set appropriate limits
docker run --memory=512m --cpus=0.5 vibes-mcp-cli
```

## 🔄 **Updating**

### Update from Source
```bash
cd vibes-mcp-cli
git pull origin main
make build
```

### Update Docker Images
```bash
make docker-build  # Rebuild with latest changes
```

### Version Management
```bash
# Check current version
./vibes-mcp-cli --version

# List available versions
git tag -l

# Switch to specific version
git checkout v0.0.10
make build
```

---

## 📞 **Need Help?**

- **Installation Issues**: [GitHub Issues](https://github.com/your-org/vibes-mcp-cli/issues/new?template=installation.md)
- **Configuration Help**: [Configuration Guide](Configuration)
- **Docker Problems**: [Docker Setup Guide](Docker)

---

*Next: [Configuration Guide](Configuration) →*