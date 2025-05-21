# Testing

This project includes both unit and integration tests.

## Unit Tests

- Located alongside code in `internal/client`, `internal/service`, and `cmd/` (e.g., `client_test.go`, `service_test.go`, `cli_test.go`).
- Use Go's `testing` package and `httptest` for mocking HTTP interactions.

Run unit tests:
```bash
go test ./internal/client ./internal/service ./cmd -v
```

## Integration Tests

- Located in `tests/integration/`.
- Exercise the full stack against a live Postgres database and the HTTP server.
- Use `httpx.AsyncClient` in Go tests to call the server endpoints.

Run integration tests:
```bash
docker-compose up -d   # start Postgres + server
go test ./tests/integration -v
docker-compose down
```

## Test Coverage

Generate coverage report:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```