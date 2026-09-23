package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MCPClientSnippet is a ready-to-paste connection config for one MCP client that
// wants to consume Pyntra's MCP server.
type MCPClientSnippet struct {
	Client     string `json:"client"`                // canonical client id
	Label      string `json:"label"`                 // human friendly name
	Transport  string `json:"transport"`             // "http" | "sse" | "stdio"
	ConfigPath string `json:"config_path,omitempty"` // where the snippet usually goes
	Format     string `json:"format"`                // "json" | "toml"
	Snippet    string `json:"snippet"`               // the config text to paste
	Notes      string `json:"notes,omitempty"`
}

// mcpServerName is the key Pyntra uses when registering itself in a client config.
const mcpServerName = "pyntra"

// mcpHTTPURL returns the externally reachable Streamable-HTTP MCP endpoint.
func mcpHTTPURL(mcp MCPConfig) string {
	host := strings.TrimSpace(mcp.Host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d/mcp", host, mcp.Port)
}

func mcpHeaders(mcp MCPConfig) map[string]string {
	headers := map[string]string{}
	if strings.TrimSpace(mcp.AuthHeader) != "" {
		headers[mcp.AuthHeader] = mcp.AuthHeaderValue
	}
	return headers
}

func mustJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// MCPClientConfigs returns paste-ready connection snippets for the common MCP
// clients (Claude Code, Cursor, Cline, opencode, Codex, Windsurf, VS Code, and a
// generic form). Clients that support remote HTTP/SSE connect straight to
// Pyntra's server; the stdio alternative (via the pyntra-mcp-stdio binary) is
// noted for clients that prefer a local process.
func MCPClientConfigs(mcp MCPConfig) []MCPClientSnippet {
	url := mcpHTTPURL(mcp)
	headers := mcpHeaders(mcp)

	// Standard { "mcpServers": { "pyntra": { "type": "http", "url": ..., "headers": ... } } }
	httpEntry := map[string]interface{}{"type": "http", "url": url}
	if len(headers) > 0 {
		httpEntry["headers"] = headers
	}
	stdHTTP := map[string]interface{}{"mcpServers": map[string]interface{}{mcpServerName: httpEntry}}

	// Cursor / Cline flavour uses "url" (+ headers) without a "type".
	urlEntry := map[string]interface{}{"url": url}
	if len(headers) > 0 {
		urlEntry["headers"] = headers
	}
	urlHTTP := map[string]interface{}{"mcpServers": map[string]interface{}{mcpServerName: urlEntry}}

	// opencode uses a "mcp" object with a "remote" type.
	opencodeEntry := map[string]interface{}{
		"type":    "remote",
		"url":     url,
		"enabled": true,
	}
	if len(headers) > 0 {
		opencodeEntry["headers"] = headers
	}
	opencode := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"mcp":     map[string]interface{}{mcpServerName: opencodeEntry},
	}

	// Codex uses TOML (~/.codex/config.toml). It speaks stdio; point it at the
	// mcp-proxy helper or the bundled pyntra-mcp-stdio binary.
	var codexTOML strings.Builder
	codexTOML.WriteString("[mcp_servers.pyntra]\n")
	codexTOML.WriteString("command = \"pyntra-mcp-stdio\"\n")
	codexTOML.WriteString("args = [\"--config\", \"config.yaml\"]\n")

	snippets := []MCPClientSnippet{
		{
			Client:     "claude-code",
			Label:      "Claude Code",
			Transport:  "http",
			ConfigPath: ".mcp.json (project) or ~/.claude.json",
			Format:     "json",
			Snippet:    mustJSON(stdHTTP),
			Notes:      "Or run: claude mcp add --transport http " + mcpServerName + " " + url,
		},
		{
			Client:     "cursor",
			Label:      "Cursor",
			Transport:  "http",
			ConfigPath: "~/.cursor/mcp.json or .cursor/mcp.json",
			Format:     "json",
			Snippet:    mustJSON(urlHTTP),
		},
		{
			Client:     "cline",
			Label:      "Cline (VS Code)",
			Transport:  "http",
			ConfigPath: "cline_mcp_settings.json",
			Format:     "json",
			Snippet:    mustJSON(urlHTTP),
			Notes:      "Add via Cline > MCP Servers > Configure, or edit the settings file directly.",
		},
		{
			Client:     "windsurf",
			Label:      "Windsurf",
			Transport:  "http",
			ConfigPath: "~/.codeium/windsurf/mcp_config.json",
			Format:     "json",
			Snippet:    mustJSON(stdHTTP),
		},
		{
			Client:     "vscode",
			Label:      "VS Code (Copilot MCP)",
			Transport:  "http",
			ConfigPath: ".vscode/mcp.json",
			Format:     "json",
			Snippet:    mustJSON(map[string]interface{}{"servers": map[string]interface{}{mcpServerName: httpEntry}}),
		},
		{
			Client:     "opencode",
			Label:      "opencode",
			Transport:  "http",
			ConfigPath: "opencode.json or ~/.config/opencode/opencode.json",
			Format:     "json",
			Snippet:    mustJSON(opencode),
		},
		{
			Client:     "codex",
			Label:      "OpenAI Codex CLI",
			Transport:  "stdio",
			ConfigPath: "~/.codex/config.toml",
			Format:     "toml",
			Snippet:    codexTOML.String(),
			Notes:      "Codex connects over stdio; build the helper with `go build -o pyntra-mcp-stdio ./cmd/mcp-stdio` and put it on PATH.",
		},
		{
			Client:     "generic",
			Label:      "Generic (HTTP)",
			Transport:  "http",
			ConfigPath: "any MCP client that supports remote HTTP",
			Format:     "json",
			Snippet:    mustJSON(stdHTTP),
		},
	}
	return snippets
}

// PrintMCPClientConfigs prints per-client MCP connection snippets to stdout.
func PrintMCPClientConfigs(mcp MCPConfig) {
	if !mcp.Enabled {
		return
	}
	fmt.Println("[Pyntra] MCP client connection snippets (Pyntra as an MCP server):")
	fmt.Println("----------------------------------------------------------------")
	for _, s := range MCPClientConfigs(mcp) {
		fmt.Printf("### %s  [%s, %s]\n", s.Label, s.Transport, s.ConfigPath)
		fmt.Println(s.Snippet)
		if s.Notes != "" {
			fmt.Printf("(%s)\n", s.Notes)
		}
		fmt.Println()
	}
	fmt.Println("----------------------------------------------------------------")
}
