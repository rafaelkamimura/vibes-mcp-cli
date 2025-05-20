# openai-cli

`openai-cli` is a Go-based Multi-Provider CLI and HTTP server for working with Large Language Model (LLM) providers such as OpenAI (and Anthropic / Claude). It provides:
- Type-safe clients for completions, chat, and embeddings
- A unified CLI (`completion`, `chat`, `embed`) with flags and REPL support
- An HTTP proxy server (`serve`) to expose a local MCP API
- Built-in support for environment variables, dotenv (`.env`), and config files (`.openai-cli.yaml`)

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

| Variable                   | Description                                        |
|----------------------------|----------------------------------------------------|
| OPENAI_CLI_API_KEY         | Default API key for the selected provider          |
| OPENAI_CLI_BASE_URL        | Base URL for API requests                          |
| OPENAI_CLI_PROVIDER        | Default provider (`openai`, `anthropic`, etc.)     |
| OPENAI_CLI_LOG_LEVEL       | Logging level (`debug`, `info`, `warn`, `error`)   |
| PROMPT_MODE_PASSWORD       | Password to unlock interactive REPL (`chat`)       |

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