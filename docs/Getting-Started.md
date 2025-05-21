# Getting Started

This guide will help you get up and running with the Vibes MCP CLI.

## Prerequisites

- Go 1.20 or higher
- Git
- (Optional) Docker & Docker Compose for containerized server

## Clone the repository

```bash
git clone https://github.com/rafaelkamimura/vibes-mcp-cli.git
cd vibes-mcp-cli
```

## Installation

### Using Go directly

```bash
go mod tidy
go build -o openai-cli .
```

### Using Makefile

```bash
make init       # Copy .env and install dependencies
make build      # Build the CLI binary
```

## Initial setup

1. Copy the example environment file:
   ```bash
   cp .env_example .env
   ```
2. Edit `.env` to set your variables:
   - `OPENAI_CLI_API_KEY`
   - `OPENAI_CLI_BASE_URL` (optional)
   - `OPENAI_CLI_PROVIDER` (optional)
   - `PROMPT_MODE_PASSWORD` (for interactive chat)

## Verify installation

```bash
./openai-cli --help
```

You should see the list of available commands and flags.