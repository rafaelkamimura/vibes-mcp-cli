# Terminal UI (TUI)

The TUI provides an interactive text-based interface for chat sessions and browsing your local filesystem.

## Launching the UI

```bash
openai-cli ui [--model MODEL] [--explorer-root PATH]
```

### Flags

- `--model`: Chat model to use (default: `gpt-3.5-turbo`).
- `--explorer-root`: Root path for the file explorer (default: current working directory).

## UI Modes

Use the function keys or menu to switch between modes:

- **Home**: Main menu for navigation.
- **Chat**: Interactive chat view (send messages to LLM or MCP agent).
- **Agent**: Chat view against the Vibes Agent backend (`/agent/chat`).
- **Explorer**: Browse and view files in your local filesystem. Press Enter to view, or press 'm' on a file to switch to MCP mode with the file path prefilled.
- **MCP**: Invoke a JSON-RPC tool call via the `/mcp` endpoint. Use the Tools dropdown to pick a tool, which pre-fills the input box.

## Controls

- **Enter**: Submit input in chat or Postman view.
- **Esc**: Return focus to previous pane or cancel selection.
- **Tab**: Cycle focus between input, templates, and model dropdown.
- **j/k**: Scroll down/up in chat view (vim-style).
- **F1/F2**: Toggle between Home and menu (requires authentication for menu).
- **C/E/A/M**: In menu, press C (Chat), E (Explorer), A (Agent), M (MCP) to switch modes.
- **Up/Down / Enter**: In MCP mode, use arrow keys to select a tool from the Tools dropdown and Enter to populate the input field. Press Tab or Esc to return focus to the input box.
 
### File Explorer Keys
- **Enter**: Toggle directory or view selected file.
- **m**: On a file, switch to MCP mode with the file path prefilled.

## Templates & Models

- Use the **Templates** dropdown to insert predefined prompts.
- Use the **Models** dropdown to switch chat models on the fly.

## Authentication

On launch, you will be prompted to Login or Register (if not authenticated):

- **Login** sends credentials to `/auth/login` and stores the JWT.
- **Register** sends a new user request to `/auth/register`.

Successful login unlocks the Chat and Agent views.