# CLI Usage

The `openai-cli` binary provides several subcommands for interacting with LLM providers and the built-in MCP server.

## Available Commands

- `completion`: Generate one-shot text completions.
- `chat`: Send chat completion requests (single-shot or interactive REPL).
- `embed`: Compute embeddings for input texts.
- `models`: List available LLM models.
- `serve`: Run the HTTP proxy server (MCP).
- `ui`: Launch the interactive terminal UI.

## Global Flags

| Flag             | Description                                                       |
|------------------|-------------------------------------------------------------------|
| `--config`       | Path to config file (default `$HOME/.openai-cli.yaml`)            |
| `--provider`     | Provider to use (overrides config/env; e.g., openai, anthropic)   |
| `--api-key`      | API key (overrides config/env)                                    |
| `--base-url`     | API base URL (overrides config/env)                               |
| `--server-url`   | MCP server URL to proxy CLI calls                                 |
| `--agent-url`    | Vibes Agent backend URL (for TUI auth and agent chat)             |
| `--print-curl`   | Print the equivalent curl command instead of executing            |
| `--log-level`    | Logging level (`debug`, `info`, `warn`, `error`)                  |

Use `--help` after any command for detailed usage information. For example:

```bash
openai-cli chat --help
```