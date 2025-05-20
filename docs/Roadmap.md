# Roadmap

Key features and improvements planned for upcoming releases:

## Short-term
- Integrate remaining API endpoints in the TUI (see [API_ENDPOINTS.md](API_ENDPOINTS.md)).
  - Auth role management (`/auth/{username}/roles`).
  - User enable/disable endpoints (`/auth/{username}/disable`, `/auth/{username}/enable`).
  - Tenant, role, and permission CRUD under `/user`.
  - WebSocket streaming via `/agent/ws`.
  - JSON-RPC `/mcp` proxy for tools.
- Improve error handling and user feedback in TUI.

## Mid-term
- Support additional providers (Azure OpenAI, AWS Bedrock).
- Plugin architecture for custom tools and chains.
- Metrics and observability integration (Prometheus, OpenTelemetry).

## Long-term
- GUI application for desktop platforms.
- End-to-end example workflows and tutorials.
- Automated release pipeline and Homebrew formula.