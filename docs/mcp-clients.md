# Connecting MCP clients to Pyntra

Pyntra exposes its security tools as an **MCP server**, so AI coding agents and
IDEs can call them directly. This covers **Claude Code, Cursor, Cline, opencode,
OpenAI Codex CLI, Windsurf, VS Code**, and any generic MCP client.

Enable the server first (`mcp.enabled: true` in `config.yaml`). Pyntra prints
ready-to-paste snippets at startup, and `GET /api/config/mcp-clients` returns
them at runtime.

## Endpoints

- **HTTP (Streamable):** `http://<host>:<port>/mcp` (default `:8081`). If
  `mcp.auth_header` is set, clients must send that header with
  `auth_header_value`. Best for Claude Code, Cursor, Cline, Windsurf, VS Code.
- **stdio:** build the helper and put it on PATH — best for Codex/opencode or
  anything that prefers a local process:

  ```bash
  go build -o pyntra-mcp-stdio ./cmd/mcp-stdio
  ```

## Per-client snippets

Replace the URL/headers with the values Pyntra prints for your config.

### Claude Code
`.mcp.json` (project) or `~/.claude.json`:
```json
{ "mcpServers": { "pyntra": { "type": "http", "url": "http://localhost:8081/mcp", "headers": { "X-MCP-Token": "<value>" } } } }
```
Or: `claude mcp add --transport http pyntra http://localhost:8081/mcp`

### Cursor
`~/.cursor/mcp.json` or `.cursor/mcp.json`:
```json
{ "mcpServers": { "pyntra": { "url": "http://localhost:8081/mcp", "headers": { "X-MCP-Token": "<value>" } } } }
```

### Cline (VS Code)
`cline_mcp_settings.json` (via Cline → MCP Servers → Configure):
```json
{ "mcpServers": { "pyntra": { "url": "http://localhost:8081/mcp", "headers": { "X-MCP-Token": "<value>" } } } }
```

### opencode
`opencode.json` or `~/.config/opencode/opencode.json`:
```json
{ "$schema": "https://opencode.ai/config.json", "mcp": { "pyntra": { "type": "remote", "url": "http://localhost:8081/mcp", "enabled": true } } }
```

### OpenAI Codex CLI
`~/.codex/config.toml` (Codex speaks stdio):
```toml
[mcp_servers.pyntra]
command = "pyntra-mcp-stdio"
args = ["--config", "config.yaml"]
```

### Windsurf
`~/.codeium/windsurf/mcp_config.json`:
```json
{ "mcpServers": { "pyntra": { "type": "http", "url": "http://localhost:8081/mcp", "headers": { "X-MCP-Token": "<value>" } } } }
```

### VS Code (Copilot MCP)
`.vscode/mcp.json`:
```json
{ "servers": { "pyntra": { "type": "http", "url": "http://localhost:8081/mcp", "headers": { "X-MCP-Token": "<value>" } } } }
```
