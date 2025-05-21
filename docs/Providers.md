# Providers

The CLI and HTTP server support multiple LLM providers via a pluggable architecture.

## Supported Providers

- **openai**: Official OpenAI API client (`internal/client`).
- **anthropic**: Anthropic Claude API client (`internal/providers/anthropic`).
- **test**: Testing stub; use `--provider test` and override `providers.TestClient` in code or tests.

## Adding a New Provider

1. Create a new package under `internal/providers/<provider-name>` implementing the `service.APIClient` interface.
2. Implement methods:
   - `CreateCompletion(ctx, CompletionsRequest)`
   - `CreateChatCompletion(ctx, ChatCompletionsRequest)`
   - `CreateEmbedding(ctx, EmbeddingRequest)`
3. Register your provider in `internal/providers/provider.go`:
   ```go
   case "<provider-name>":
       return yourpkg.NewClient(apiKey, baseURL), nil
   ```
4. Rebuild the binary; your provider is now available via `--provider <provider-name>`.