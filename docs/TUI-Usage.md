# Terminal UI (TUI)

The TUI provides an interactive text-based interface for chat sessions and browsing Postman collections.

## Launching the UI

```bash
openai-cli ui [--model MODEL] [--collection PATH]
```

### Flags

- `--model`: Chat model to use (default: `gpt-3.5-turbo`).
- `--collection`: Path to a Postman collection JSON file to load.

## UI Modes

Use the function keys or menu to switch between modes:

- **Home**: Main menu for navigation.
- **Chat**: Interactive chat view (send messages to LLM or MCP agent).
- **Agent**: Chat view against the Vibes Agent backend (`/agent/chat`).
- **Postman**: Browse and load a Postman collection for issuing HTTP requests.

## Controls

- **Enter**: Submit input in chat or Postman view.
- **Esc**: Return focus to previous pane or cancel selection.
- **Tab**: Cycle focus between input, templates, and model dropdown.
- **j/k**: Scroll down/up in chat view (vim-style).
- **F1/F2**: Toggle between Chat and Postman modes.

## Templates & Models

- Use the **Templates** dropdown to insert predefined prompts.
- Use the **Models** dropdown to switch chat models on the fly.

## Authentication

On launch, you will be prompted to Login or Register (if not authenticated):

- **Login** sends credentials to `/auth/login` and stores the JWT.
- **Register** sends a new user request to `/auth/register`.

Successful login unlocks the Chat and Agent views.