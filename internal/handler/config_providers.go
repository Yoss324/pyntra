package handler

import (
	"net/http"

	"pyntra/internal/config"

	"github.com/gin-gonic/gin"
)

// GetProviders returns the built-in inference-provider catalogue plus the
// currently selected provider, so the Settings UI can offer presets while
// keeping base_url/api_key/model fully custom-configurable.
func (h *ConfigHandler) GetProviders(c *gin.Context) {
	h.mu.RLock()
	current := config.NormalizeProvider(h.config.OpenAI.Provider)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"current":   current,
		"providers": config.ProviderPresets(),
	})
}

// GetMCPClients returns paste-ready connection snippets for MCP clients
// (Claude Code, Cursor, Cline, opencode, Codex, Windsurf, VS Code, generic)
// that want to consume Pyntra's MCP server.
func (h *ConfigHandler) GetMCPClients(c *gin.Context) {
	h.mu.RLock()
	mcp := h.config.MCP
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"enabled": mcp.Enabled,
		"clients": config.MCPClientConfigs(mcp),
	})
}
