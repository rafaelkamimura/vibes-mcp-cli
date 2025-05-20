# Configuration

The application can be configured using environment variables and a configuration file.

## Environment Variables

| Name                   | Description                                                   |
|------------------------|---------------------------------------------------------------|
| OPENAI_CLI_API_KEY     | API key for the LLM provider                                  |
| OPENAI_CLI_BASE_URL    | Base URL for the LLM API (default: https://api.openai.com)    |
| OPENAI_CLI_PROVIDER    | Default provider name (e.g., openai, anthropic)               |
| OPENAI_CLI_LOG_LEVEL   | Logging level (`debug`, `info`, `warn`, `error`)              |
| OPENAI_CLI_AGENT_URL   | URL of the Vibes Agent backend (default: http://localhost:8000) |
| OPENAI_CLI_AUTH_TOKEN  | JWT token for Vibes Agent backend                             |
| PROMPT_MODE_PASSWORD   | Password for interactive chat REPL mode                       |

## Configuration File

You can also use a config file (`.openai-cli.yaml`, JSON, or TOML) in your home directory or working directory. Example YAML:

```yaml
api_key: "your-openai-api-key"
base_url: "https://api.openai.com"
provider: "openai"
log_level: "info"
agent_url: "http://localhost:8000"
auth_token: "your-jwt-token"
templates:
  - "Hello!"
  - "What is the weather?"
```

CLI flags override config file values.