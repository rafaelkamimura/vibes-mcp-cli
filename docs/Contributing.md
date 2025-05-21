# Contributing

Thank you for your interest in contributing to the Vibes MCP CLI!

## Code Style

- Follow Go formatting conventions (`go fmt`).
- Keep line length ≤120 characters.
- Use descriptive names and modular design.

## Git Workflow

1. Fork the repository.
2. Create a feature branch with a descriptive name (`feat/...`, `fix/...`, `docs/...`).
3. Commit changes with semantic messages:
   - `feat(...)`: new feature
   - `fix(...)`: bug fix
   - `refactor(...)`: code refactoring
   - `docs(...)`: documentation changes
4. Ensure tests pass locally:
   ```bash
   make test
   ```
5. Push your branch and open a Pull Request.

## Pull Request Guidelines

- Provide a clear description of the changes and motivation.
- Include relevant test coverage for new functionality.
- Update or add documentation as needed.
- Ensure continuous integration checks (lint, tests) pass.