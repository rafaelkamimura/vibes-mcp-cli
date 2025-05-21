# API Endpoint Reference

This document provides a summary of all available HTTP and WebSocket endpoints in the Vibes Agent Backend, their methods, required authentication, request and response formats.

## Authentication & User Registration (`/auth`)

- **POST** `/auth/login`
  - Description: Exchange username & password for a JWT access token.
  - Authentication: None
  - Request (form data):

    ```
    username: <string>
    password: <string>
    scope: <space-separated list of scopes> (optional)
    ```

  - Response (200 OK):

    ```json
    {
      "access_token": "<JWT>",
      "token_type": "bearer"
    }
    ```

- **POST** `/auth/register`
  - Description: Register a new user account.
  - Authentication: None
  - Request (JSON):

    ```json
    {
      "username": "alice",
      "password": "password123",
      "full_name": "Alice Wonderland"
    }
    ```

  - Response (201 Created):

    ```json
    {
      "id": "<uuid>",
      "username": "alice",
      "full_name": "Alice Wonderland",
      "disabled": false,
      "created_at": "<timestamp>"
    }
    ```

- **POST** `/auth/{username}/roles`
  - Description: Assign a role to a user.
  - Authentication: Bearer token (any scope)
  - Request (JSON): `{ "role_id": "<uuid>" }`
  - Response: 204 No Content

- **DELETE** `/auth/{username}/roles/{role_id}`
  - Description: Revoke a role from a user.
  - Authentication: Bearer token (any scope)
  - Response: 204 No Content

- **PATCH** `/auth/{username}/disable`
  - Description: Disable (soft-delete) a user. Sets `disabled_at` automatically.
  - Authentication: Bearer token (any scope)
  - Response: 204 No Content

- **PATCH** `/auth/{username}/enable`
  - Description: Re-enable a previously disabled user (clears `disabled_at`).
  - Authentication: Bearer token (any scope)
  - Response: 204 No Content

## User Management (`/user`)

All endpoints under `/user` require a valid Bearer token.

### Tenants

- **GET** `/user/tenants`
  - Description: List all tenants.
  - Authentication: Bearer token
  - Response (200 OK): Array of tenant objects:

    ```json
    [
      { "id": "<uuid>", "name": "Acme Corp" },
      ...
    ]
    ```

- **POST** `/user/tenants`
  - Description: Create a new tenant.
  - Authentication: Bearer token
  - Request (JSON): `{ "name": "Acme Corp" }`
  - Response (201 Created): `{ "id": "<uuid>", "name": "Acme Corp" }`

### Roles

- **GET** `/user/roles`
  - Description: List all roles.
  - Authentication: Bearer token
  - Response (200 OK): Array of role objects.

- **POST** `/user/roles`
  - Description: Create a new role within a tenant.
  - Authentication: Bearer token
  - Request (JSON): `{ "name": "admin", "tenant_id": "<uuid>" }`
  - Response (201 Created): new role object.

- **DELETE** `/user/roles/{role_id}`
  - Description: Delete a role.
  - Authentication: Bearer token
  - Response: 204 No Content

- **POST** `/user/roles/{role_id}/permissions`
  - Description: Assign a permission to a role.
  - Authentication: Bearer token
  - Request (JSON): `{ "permission_name": "agent:chat" }`
  - Response: 204 No Content

- **DELETE** `/user/roles/{role_id}/permissions/{permission_name}`
  - Description: Revoke a permission from a role.
  - Authentication: Bearer token
  - Response: 204 No Content

### Permissions

- **GET** `/user/permissions`
  - Description: List all permissions.
  - Authentication: Bearer token
  - Response: Array of permission objects.

- **POST** `/user/permissions`
  - Description: Create a new permission.
  - Authentication: Bearer token
  - Request (JSON): `{ "name": "agent:stream", "description": "Stream responses" }`
  - Response (201 Created): permission object.

- **DELETE** `/user/permissions/{name}`
  - Description: Delete a permission.
  - Authentication: Bearer token
  - Response: 204 No Content

## Agent Interaction (`/agent`)

All endpoints under `/agent` require OAuth2 Bearer tokens with appropriate scopes.

- **POST** `/agent/chat`
  - Description: Send a message to the agent and get a full response.
  - Authentication: Bearer token with scope `agent:chat`
  - Request (JSON): `{ "message": "Hello, agent!" }`
  - Response (200 OK):

    ```json
    {
      "response": "<agent reply>",
      "usage": { "prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30 }
    }
    ```

- **WebSocket** `ws://<host>/agent/ws?token=<JWT>`
  - Description: Open a WebSocket for streaming agent responses.
  - Authentication: Query parameter `token=<JWT>` with `agent:stream` scope.
  - Usage: Send plain text messages; receive streamed chunks.

## JSON-RPC Tool Proxy (`/mcp`)

- **POST** `/mcp`
  - Description: JSON-RPC 2.0 proxy for `call_tool` (single-shot use).
  - Authentication: Bearer token with scope `agent:chat`
  - Request (JSON RPC):

    ```json
    {
      "jsonrpc": "2.0",
      "id": "<trace-id>",
      "method": "call_tool",
      "params": { "input": "Compute 2+2" }
    }
    ```

  - Response:
    - If `Accept: text/event-stream`, streams SSE events: `data: <result>`
    - If `Accept: application/json`, returns JSON RPC response:

      ```json
      { "jsonrpc": "2.0", "id": "<trace-id>", "result": "<output>" }
      ```

---
_All endpoints except `/auth/login` and `/auth/register` require a valid JWT Bearer token._

