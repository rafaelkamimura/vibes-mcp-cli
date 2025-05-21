# Architecture Overview

This section describes the high-level architecture and code organization of the Vibes MCP CLI.

```
vibes-mcp-cli/
├── cmd/                # Cobra-based CLI commands and TUI
│   ├── root.go         # Entrypoint and global flags
│   ├── completion.go   # 'completion' subcommand
│   ├── chat.go         # 'chat' subcommand
│   ├── embed.go        # 'embed' subcommand
│   ├── models.go       # 'models' subcommand
│   ├── serve.go        # 'serve' subcommand (HTTP server)
│   └── ui.go           # 'ui' subcommand (terminal UI)

├── internal/
│   ├── config/         # Application configuration loader (dotenv, Viper)
│   ├── client/         # Type-safe HTTP client for OpenAI API
│   ├── providers/      # Provider factory and specific provider implementations
│   └── service/        # Service layer wrapping APIClient interface

├── dist/               # Prebuilt binaries for various platforms
├── docs/               # Project documentation and wiki pages
├── Makefile            # Common development commands
├── docker-compose.yml  # Docker Compose configuration
├── Dockerfile          # Container image for the CLI and server
├── LICENSE
└── README.md
```

## Key Components

- **Cobra Commands (`cmd/`):** Define the CLI surface and TUI behavior.
- **Config (`internal/config`):** Loads environment variables, `.env`, and config files; persists auth tokens.
- **HTTP Client (`internal/client`):** Implements retries, backoff, and low-level HTTP calls to LLM endpoints.
- **Providers (`internal/providers`):** Factory method to select provider implementations at runtime.
- **Service Layer (`internal/service`):** Defines the `APIClient` interface and coordinates calls from commands.

## Data Flow

1. **CLI/TUI** parses user input and flags.
2. **Config** provides API credentials and endpoints.
3. **Providers** creates an `APIClient` implementation.
4. **Service** calls the appropriate client method (`CreateCompletion`, etc.).
5. **Client** sends HTTP requests to the provider's REST API.
6. Results are returned to the CLI/TUI or HTTP server response.