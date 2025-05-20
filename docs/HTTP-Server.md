# HTTP MCP Server

The built-in HTTP server proxies requests to supported LLM providers and exposes a JSON-RPC tool proxy.

## Starting the Server

```bash
openai-cli serve --host 0.0.0.0 --port 8080
```

Use `--help` to see additional flags for host, port, and logging.

## Available Endpoints

- `POST /v1/completions`
  - Single-shot text completions proxy
- `POST /v1/chat/completions`
  - Single-shot chat completions proxy
- `POST /v1/embeddings`
  - Embedding generation proxy
- `POST /mcp`
  - JSON-RPC 2.0 proxy for tool calls (method `call_tool`)
  - Supports SSE streaming or JSON response based on `Accept` header

### Common Headers

- `Content-Type: application/json`
- `Authorization: Bearer <API_KEY>` or set `X-Provider: <provider>` to select provider

## Switching Providers

Use the `X-Provider` header to specify which provider to use for a given request. If omitted, the default provider from configuration is used.

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Provider: anthropic" \
  -d '{"model":"claude-v1","messages":[{"role":"user","content":"Hi"}]}'
```