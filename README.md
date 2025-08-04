# vibes-mcp-cli

`vibes-mcp-cli` is a comprehensive Go-based Multi-Provider CLI and HTTP server for working with Large Language Model (LLM) providers, featuring enterprise-grade session management and interactive terminal UI. It provides:

## 🚀 **Key Features**

### **Multi-Provider LLM Support**
- Type-safe clients for completions, chat, and embeddings
- Support for OpenAI, Anthropic Claude, and other providers
- Unified CLI interface with consistent command structure

### **Advanced Session Management** ⭐
- **Interactive Session Control**: Real-time communication with Claude CLI processes
- **Session Persistence**: Complete conversation history with backup and restoration
- **Advanced Search & Filtering**: Find sessions by content, date, status, or metadata
- **Metadata Tracking**: Detailed tracking of tokens, response times, and resource usage

### **Interactive Terminal UI** ⭐
- **Full-Screen TUI**: Comprehensive terminal interface with multiple pages
- **Session Logs Viewer**: Browse, search, and manage session history
- **Telemetry Dashboard**: Real-time system monitoring with ASCII charts
- **File Explorer**: Browse and interact with project files

### **Production-Ready Stability** ⭐
- **Memory Leak Protection**: Robust resource management and cleanup
- **TTY Detection**: Works in containers, CI/CD, and headless environments
- **Error Boundaries**: Comprehensive error handling with graceful recovery
- **Timeout Protection**: All operations have configurable timeouts

### **Enterprise Integration**
- HTTP proxy server (`serve`) to expose MCP-compatible API
- Telemetry integration with vibes-agent-backend
- Authentication support with JWT tokens
- Built-in support for environment variables, dotenv (`.env`), and config files

## Installation

1. Install Go 1.20+ (https://golang.org/dl)
2. Clone this repo:
   ```bash
   git clone <repo-url>
   cd openai-cli
   ```
3. Copy environment example and edit your keys:
   ```bash
   cp .env_example .env
   # Edit .env: set OPENAI_CLI_API_KEY, PROMPT_MODE_PASSWORD, etc.
   ```
4. (Optional) Create a config file in `$HOME/.openai-cli.yaml` or `./.openai-cli.yaml`:
   ```yaml
   api_key: "your-openai-api-key"
   base_url: "https://api.openai.com"
   provider: "openai"
   log_level: "info"
   # Optional: JWT auth token for Vibes Agent backend to persist login
   auth_token: "your-agent-auth-token"
   templates:
     - "Hey, what's up!"
     - "Hows the weather in Brasilia - DF right now?"
   ```
5. Build:
   ```bash
   go mod tidy
   go build -o openai-cli
   ```

## Makefile

This project includes a `Makefile` to simplify common tasks:

```bash
make init         # Initialize environment: copy .env, install deps
make build        # Build the CLI binary
make test         # Run all tests (client, service, cmd)
make lint         # Format code (go fmt) and run vet
make docker-build # Build the Docker image
make docker-up    # Start the server via docker-compose
make release      # Cross-compile binaries for multiple platforms into dist/
make clean        # Remove built binaries
```

## Environment Variables

| Variable                         | Description                                        |
|----------------------------------|----------------------------------------------------|
| OPENAI_CLI_API_KEY               | Default API key for the selected provider          |
| OPENAI_CLI_BASE_URL              | Base URL for API requests                          |
| OPENAI_CLI_PROVIDER              | Default provider (`openai`, `anthropic`, etc.)     |
| OPENAI_CLI_LOG_LEVEL             | Logging level (`debug`, `info`, `warn`, `error`)   |
| OPENAI_CLI_AGENT_URL             | Vibes Agent backend URL (default: http://localhost:8000) |
| OPENAI_CLI_AUTH_TOKEN            | JWT token for backend authentication              |
| OPENAI_CLI_TELEMETRY_ENABLED     | Enable telemetry data collection (`true`/`false`) |
| OPENAI_CLI_TELEMETRY_API_KEY     | API key for telemetry service                     |
| PROMPT_MODE_PASSWORD             | Password to unlock interactive REPL (`chat`)       |

Environment variables can be set in a `.env` file (via `github.com/joho/godotenv`) or directly in your shell.

## CLI Usage

Run `./openai-cli --help` for global flags and available commands.

### Common Global Flags

- `--config string`    : path to config file (default `$HOME/.openai-cli.yaml`)
- `--provider string`  : provider to use (overrides config / env)
- `--api-key string`   : API key (overrides config / env)
- `--base-url string`  : API base URL (overrides config / env)
- `--server-url string`: MCP server URL to proxy CLI calls
- `--print-curl`       : print equivalent `curl` command and exit
- `--log-level string` : set log level (`debug`, `info`, `warn`, `error`)

### Completion

Generate a one-shot text completion:
```bash
./openai-cli completion \
  --prompt "Once upon a time" \
  --model text-davinci-003
```

### Chat

#### Single request
Send a single chat message:
```bash
./openai-cli chat \
  --message "Hello, how are you?" \
  --model gpt-3.5-turbo
```

#### Interactive REPL
Keep context across messages:
```bash
export PROMPT_MODE_PASSWORD=your-password
./openai-cli chat --prompt-mode
``` 
Type your message at the `>>> ` prompt. Enter `exit` or `quit` to end.

### UI (Terminal TUI)

Launch an interactive terminal UI for chat and Postman collections:
```bash
./openai-cli ui [--model MODEL] [--collection PATH]
```
Use F1 to switch to Chat mode and F2 to switch to Postman mode. In Postman mode, navigate and select a `.json` collection, then press **Ctrl+S** to send a request.

### Embeddings

Compute embeddings for one or more inputs:
```bash
./openai-cli embed \
  --input "The quick brown fox" \
  --input "jumps over the lazy dog" \
  --model text-embedding-ada-002
```

### Models

List available models you can use with the `--model` flag:

```bash
./openai-cli models
```

Output:

```
o4-mini
gpt-3.5-turbo
codex-cli
```

### print-curl and server-url

To see the raw `curl` you can run:
```bash
./openai-cli completion --prompt "Hello" --print-curl
```

To proxy commands through a running MCP server:
```bash
./openai-cli completion \
  --prompt "Hello" \
  --server-url http://localhost:8080
```

## 🖥️ **Interactive Terminal UI**

Launch the comprehensive terminal user interface:

```bash
./vibes-mcp-cli ui
```

### **UI Features**

#### **Main Pages**
- **Chat**: Interactive chat interface with conversation context
- **Session Logs**: Browse and manage Claude CLI session history
- **Telemetry**: Real-time system monitoring and performance metrics
- **File Explorer**: Browse project files with MCP integration
- **Settings**: Configuration and tenant management

#### **Session Management**
- **Create/Terminate Sessions**: Full lifecycle management
- **Search & Filter**: Find sessions by name, content, date, or status
- **Conversation History**: View complete session interactions
- **Real-time Updates**: Live session status and monitoring

#### **Telemetry Dashboard**
- **System Health**: CPU, memory, disk usage with progress bars
- **API Metrics**: Request counts, success rates, response times
- **ASCII Charts**: Visual trends and performance graphs
- **Log Viewer**: Real-time log streaming with severity filtering

#### **Keyboard Shortcuts**
- `F1`: Home menu
- `F2`: Main navigation menu
- `F3`: File explorer
- `G`: Session logs viewer
- `T`: Telemetry dashboard
- `Q`: Quit application
- `/`: Search functionality
- `Esc`: Return to previous page

### **Environment Support**

The UI automatically detects your environment and provides appropriate alternatives:

#### **Interactive Terminals**
- Full TUI functionality with all features enabled

#### **Containers/Headless Systems**
```bash
# Automatic fallback suggestions
./vibes-mcp-cli ui --fallback-server  # Auto-start server mode
./vibes-mcp-cli serve --port 8080     # Manual server mode
./vibes-mcp-cli chat "message"        # CLI mode
```

#### **CI/CD Environments**
- Graceful degradation with helpful error messages
- Alternative command suggestions
- No hanging or freezing issues

## 📊 **Session Management**

### **Advanced Session Control**

```bash
# Create a new Claude CLI session
./vibes-mcp-cli ui  # Use session management UI

# Sessions are automatically persisted in ./claude-sessions/
# Session history includes:
# - Complete conversation logs
# - Metadata (tokens, response times, resource usage)
# - Search indexes for fast filtering
# - Backup files with retention policies
```

### **Session Features**
- **Interactive Communication**: Real-time streaming with Claude CLI
- **Persistent History**: All conversations saved with metadata
- **Advanced Search**: Text search, regex support, multi-criteria filtering
- **Resource Monitoring**: Track memory, CPU, and token usage
- **Backup & Restore**: Automated backups with configurable retention
- **Concurrent Management**: Handle multiple sessions safely

## HTTP MCP Server

Start the built-in HTTP proxy:
```bash
./openai-cli serve --host 0.0.0.0 --port 8080
```
Available endpoints:
- `POST /v1/completions`
- `POST /v1/chat/completions`
- `POST /v1/embeddings`

Use the `X-Provider` header to switch providers per request:
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Provider: anthropic" \
  -d '{"model":"claude-v1","messages":[{"role":"user","content":"Hi"}]}'
```

## Configuration File

Config files (`.openai-cli.yaml`, JSON, TOML) are supported via Viper in your home or working directory.

## Extending

- Add new subcommands under `cmd/`
- Update models in `internal/client/types.go`
- Implement additional providers under `internal/providers/`

---
## Docker & Deployment

Build the Docker image locally:
```bash
docker build -t openai-cli:latest .
```

Run the server in a container (using `.env` for config):
```bash
docker run --rm -it \
  --env-file .env \
  -p 8080:8080 \
  openai-cli:latest serve --host 0.0.0.0 --port 8080
```

Alternatively, use Docker Compose:
```bash
docker-compose up --build
```

Now your MCP server is listening on `http://localhost:8080`.

---
_Generated README by the openai-cli scaffolding agent._

## Next Steps

- Integrate remaining API endpoints into the UI client (per API_ENDPOINTS.md):
  - Role management and user enable/disable endpoints under `/auth`
  - Tenant, role, and permission management under `/user`
  - WebSocket streaming via `/agent/ws`
  - JSON-RPC tool proxy endpoint `/mcp`
  - Any other endpoints outlined in API_ENDPOINTS.md not yet supported in the TUI