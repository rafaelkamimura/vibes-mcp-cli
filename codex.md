<!-- codex.md: AI Assistant Instructions for Go OpenAI CLI Connector -->
# AI Assistant Instructions

## Project Overview
We are building a Go-based CLI connector to the OpenAI API, providing a type-safe, robust, and extensible command-line interface for various OpenAI endpoints (completions, chat, embeddings, etc.).

## Goals
- Maintain clear separation of concerns with professional-level architecture.
- Ensure full type safety for request/response payloads.
- Provide intuitive CLI UX with structured commands and flags.
- Facilitate easy testing and mocking of API interactions.
- Support configuration via environment variables and config files.
- Enable future extension for new endpoints and features.

## Directory Structure
```plaintext
./
  cmd/                      # entrypoint and CLI command definitions
    root.go                 # root command setup
    completion.go           # completions subcommand
    chat.go                 # chat subcommand
    embed.go                # embeddings subcommand
  internal/
    client/                 # OpenAI API client and type-safe models
      client.go             # HTTP client wrapper with auth, retries
      types.go              # Go structs for requests and responses
    config/                 # configuration loader and defaults
      config.go             # load env vars, support YAML/TOML
    service/                # business logic orchestration
      service.go            # high-level operations for each command
  pkg/                      # reusable generic utilities (optional)
    logger/                 # structured logging wrapper
    http/                   # HTTP helpers and middleware
  tests/                    # integration and unit tests
  go.mod, go.sum            # module definitions
  README.md                 # project README and usage examples
```

## Coding Guidelines
- Use `cobra` for CLI parsing; keep commands well-organized under `cmd/`.
- Define OpenAI JSON schemas as Go structs in `internal/client/types.go`.
- Implement HTTP client wrapper with context support, retries, backoff, and rate-limit handling.
- Validate input flags early and provide user-friendly errors.
- Keep CLI layer focused on parsing and output formatting; delegate business logic to `internal/service`.
- Use interfaces for the client to enable mocking in tests.
- Write unit tests for client and service layers; use `httptest` or mock RoundTripper.
- Write end-to-end tests for CLI via `os/exec` or `testing` package.
- Use `zap` or `logrus` for structured logging; allow log level configuration via flags or env.
- Document commands and examples in `README.md`.

## Workflow
1. Load configuration (API key, base URL, timeouts) via environment or config file.
2. Initialize HTTP client with authentication and middleware.
3. Parse CLI arguments and flags using cobra root and subcommands.
4. Map parsed flags to typed request structs.
5. Invoke client methods through the service layer.
6. Handle errors, context cancellations, and HTTP status codes.
7. Output responses in JSON or formatted text, based on output flag.

## Extensibility
- Add new subcommands by creating files under `cmd/` and registering in `root.go`.
- Extend client capabilities by updating `internal/client/types.go` and client methods.
- Support plugins by defining interfaces in `internal/service` and loading dynamically.

## Next Steps
- Observability: integrate Prometheus metrics into the MCP server and expose `/metrics` endpoint.
- TUI Enhancements: add a full file Explorer (F3) with MCP filesystem integration for reading and patching files, plus vim-style scrolling and UI styling.
- CI/CD: configure automated pipelines for linting, testing, and Docker builds.
- Release: automate binary releases (Homebrew, GitHub Releases).
- Plugin Architecture: enable pluggable providers for extensibility.

---
_This codex.md guides the AI assistant in generating, reviewing, and extending code for the Go OpenAI CLI connector._