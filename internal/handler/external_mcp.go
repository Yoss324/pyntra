package handler

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"pyntra/internal/config"
	"pyntra/internal/mcp"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ExternalMCPHandler externalMCPprocessor
type ExternalMCPHandler struct {
	manager *mcp.ExternalMCPManager
	config *config.Config
	configPath string
	logger *zap.Logger
	mu sync.RWMutex
}

// NewExternalMCPHandler createexternalMCPprocessor
func NewExternalMCPHandler(manager *mcp.ExternalMCPManager, cfg *config.Config, configPath string, logger *zap.Logger) *ExternalMCPHandler {
	return &ExternalMCPHandler{
		manager: manager,
		config: cfg,
		configPath: configPath,
		logger: logger,
	}
}
func (h *ExternalMCPHandler) GetExternalMCPs(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	configs := h.manager.GetConfigs()
	toolCounts := h.manager.GetToolCounts()
	result := make(map[string]ExternalMCPResponse)
	for name, cfg := range configs {
		client, exists := h.manager.GetClient(name)
		status := "disconnected"
		if exists {
			status = client.GetStatus()
		} else if h.isEnabled(cfg) {
			status = "disconnected"
		} else {
			status = "disabled"
		}

		toolCount := toolCounts[name]
		errorMsg := ""
		if status == "error" {
			errorMsg = h.manager.GetError(name)
		}

		result[name] = ExternalMCPResponse{
			Config: cfg,
			Status: status,
			ToolCount: toolCount,
			Error: errorMsg,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": result,
		"stats": h.manager.GetStats(),
	})
}
func (h *ExternalMCPHandler) GetExternalMCP(c *gin.Context) {
	name := c.Param("name")

	h.mu.RLock()
	defer h.mu.RUnlock()

	configs := h.manager.GetConfigs()
	cfg, exists := configs[name]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "externalMCPconfiguredoes not exist"})
		return
	}

	client, clientExists := h.manager.GetClient(name)
	status := "disconnected"
	if clientExists {
		status = client.GetStatus()
	} else if h.isEnabled(cfg) {
		status = "disconnected"
	} else {
		status = "disabled"
	}

	// gettool count
	toolCount := 0
	if clientExists && client.IsConnected() {
		if count, err := h.manager.GetToolCount(name); err == nil {
			toolCount = count
		}
	}

	// geterror message
	errorMsg := ""
	if status == "error" {
		errorMsg = h.manager.GetError(name)
	}

	c.JSON(http.StatusOK, ExternalMCPResponse{
		Config: cfg,
		Status: status,
		ToolCount: toolCount,
		Error: errorMsg,
	})
}
func (h *ExternalMCPHandler) AddOrUpdateExternalMCP(c *gin.Context) {
	var req AddOrUpdateExternalMCPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namecannot be empty"})
		return
	}

	// validateconfigure
	if err := h.validateConfig(req.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.manager.AddOrUpdateConfig(name, req.Config); err != nil {
		h.logger.Error("orupdateexternalMCPconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "orupdateconfigurefailed: " + err.Error()})
		return
	}
	if h.config.ExternalMCP.Servers == nil {
		h.config.ExternalMCP.Servers = make(map[string]config.ExternalMCPServerConfig)
	}
	cfg := req.Config

	if req.Config.Disabled {
		cfg.ExternalMCPEnable = false
		cfg.Disabled = true
		cfg.Enabled = false
	} else if req.Config.Enabled {
		cfg.ExternalMCPEnable = true
		cfg.Enabled = true
		cfg.Disabled = false
	} else if !req.Config.ExternalMCPEnable {
		if existingCfg, exists := h.config.ExternalMCP.Servers[name]; exists {
			cfg.Enabled = existingCfg.Enabled
			cfg.Disabled = existingCfg.Disabled
		}
	} else {
		cfg.Enabled = true
		cfg.Disabled = false
	}

	h.config.ExternalMCP.Servers[name] = cfg
	if err := h.saveConfig(); err != nil {
		h.logger.Error("saveconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "saveconfigurefailed: " + err.Error()})
		return
	}

	h.logger.Info("externalMCPconfigurehasupdate", zap.String("name", name))
	c.JSON(http.StatusOK, gin.H{"message": "configurehasupdate"})
}

// DeleteExternalMCP deleteexternalMCPconfigure
func (h *ExternalMCPHandler) DeleteExternalMCP(c *gin.Context) {
	name := c.Param("name")

	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.manager.RemoveConfig(name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "configuredoes not exist"})
		return
	}
	if h.config.ExternalMCP.Servers != nil {
		delete(h.config.ExternalMCP.Servers, name)
	}
	if err := h.saveConfig(); err != nil {
		h.logger.Error("saveconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "saveconfigurefailed: " + err.Error()})
		return
	}

	h.logger.Info("externalMCPconfigurehasdelete", zap.String("name", name))
	c.JSON(http.StatusOK, gin.H{"message": "configurehasdelete"})
}

// StartExternalMCP start externalMCP
func (h *ExternalMCPHandler) StartExternalMCP(c *gin.Context) {
	name := c.Param("name")

	h.mu.Lock()
	defer h.mu.Unlock()

	// updateconfigureisenable
	if h.config.ExternalMCP.Servers == nil {
		h.config.ExternalMCP.Servers = make(map[string]config.ExternalMCPServerConfig)
	}
	cfg := h.config.ExternalMCP.Servers[name]
	cfg.ExternalMCPEnable = true
	h.config.ExternalMCP.Servers[name] = cfg
	if err := h.saveConfig(); err != nil {
		h.logger.Error("saveconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "saveconfigurefailed: " + err.Error()})
		return
	}
	h.logger.Info("startstart externalMCP", zap.String("name", name))
	if err := h.manager.StartClient(name); err != nil {
		h.logger.Error("start externalMCPfailed", zap.String("name", name), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"status": "error",
		})
		return
	}
	client, exists := h.manager.GetClient(name)
	status := "connecting"
	if exists {
		status = client.GetStatus()
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "externalMCPstartrequesthas,connectionin",
		"status": status,
	})
}

// StopExternalMCP stopexternalMCP
func (h *ExternalMCPHandler) StopExternalMCP(c *gin.Context) {
	name := c.Param("name")

	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.manager.StopClient(name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// updateconfigure
	if h.config.ExternalMCP.Servers == nil {
		h.config.ExternalMCP.Servers = make(map[string]config.ExternalMCPServerConfig)
	}
	cfg := h.config.ExternalMCP.Servers[name]
	cfg.ExternalMCPEnable = false
	h.config.ExternalMCP.Servers[name] = cfg
	if err := h.saveConfig(); err != nil {
		h.logger.Error("saveconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "saveconfigurefailed: " + err.Error()})
		return
	}

	h.logger.Info("externalMCPhasstop", zap.String("name", name))
	c.JSON(http.StatusOK, gin.H{"message": "externalMCPhasstop"})
}

// GetExternalMCPStats getstatistics
func (h *ExternalMCPHandler) GetExternalMCPStats(c *gin.Context) {
	stats := h.manager.GetStats()
	c.JSON(http.StatusOK, stats)
}

// validateConfig validateconfigure
func (h *ExternalMCPHandler) validateConfig(cfg config.ExternalMCPServerConfig) error {
	transport := cfg.Transport
	if transport == "" {
		if cfg.Command != "" {
			transport = "stdio"
		} else if cfg.URL != "" {
			transport = "http"
		} else {
			return fmt.Errorf("needspecifycommand(stdiomode)orurl(http/ssemode)")
		}
	}

	switch transport {
	case "http":
		if cfg.URL == "" {
			return fmt.Errorf("HTTPmodeneedURL")
		}
	case "stdio":
		if cfg.Command == "" {
			return fmt.Errorf("stdiomodeneedcommand")
		}
	case "sse":
		if cfg.URL == "" {
			return fmt.Errorf("SSEmodeneedURL")
		}
	default:
		return fmt.Errorf("supportofmode: %s,supportofmode: http, stdio, sse", transport)
	}

	return nil
}
func (h *ExternalMCPHandler) isEnabled(cfg config.ExternalMCPServerConfig) bool {
	// prefer to use ExternalMCPEnable field
	if cfg.ExternalMCPEnable {
		return true
	}
	if cfg.Disabled {
		return false
	}
	if cfg.Enabled {
		return true
	}
	return true
}

// saveConfig saveconfigureto file
func (h *ExternalMCPHandler) saveConfig() error {
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := os.WriteFile(h.configPath+".backup", data, 0644); err != nil {
		h.logger.Warn("createconfigurebackupfailed", zap.Error(err))
	}

	root, err := loadYAMLDocument(h.configPath)
	if err != nil {
		return fmt.Errorf("parseconfigurefilefailed: %w", err)
	}
	originalConfigs := make(map[string]map[string]bool)
	externalMCPNode := findMapValue(root.Content[0], "external_mcp")
	if externalMCPNode != nil && externalMCPNode.Kind == yaml.MappingNode {
		serversNode := findMapValue(externalMCPNode, "servers")
		if serversNode != nil && serversNode.Kind == yaml.MappingNode {
			for i := 0; i < len(serversNode.Content); i += 2 {
				if i+1 >= len(serversNode.Content) {
					break
				}
				nameNode := serversNode.Content[i]
				serverNode := serversNode.Content[i+1]
				if nameNode.Kind == yaml.ScalarNode && serverNode.Kind == yaml.MappingNode {
					serverName := nameNode.Value
					originalConfigs[serverName] = make(map[string]bool)
					if enabledVal := findBoolInMap(serverNode, "enabled"); enabledVal != nil {
						originalConfigs[serverName]["enabled"] = *enabledVal
					}
					if disabledVal := findBoolInMap(serverNode, "disabled"); disabledVal != nil {
						originalConfigs[serverName]["disabled"] = *disabledVal
					}
				}
			}
		}
	}

	// updateexternalMCPconfigure
	updateExternalMCPConfig(root, h.config.ExternalMCP, originalConfigs)

	if err := writeYAMLDocument(h.configPath, root); err != nil {
		return fmt.Errorf("saveconfigurefilefailed: %w", err)
	}

	h.logger.Info("configurehassave", zap.String("path", h.configPath))
	return nil
}

// updateExternalMCPConfig updateexternalMCPconfigure
func updateExternalMCPConfig(doc *yaml.Node, cfg config.ExternalMCPConfig, originalConfigs map[string]map[string]bool) {
	root := doc.Content[0]
	externalMCPNode := ensureMap(root, "external_mcp")
	serversNode := ensureMap(externalMCPNode, "servers")
	serversNode.Content = nil
	for name, serverCfg := range cfg.Servers {
		nameNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}
		serverNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		serversNode.Content = append(serversNode.Content, nameNode, serverNode)
		if serverCfg.Command != "" {
			setStringInMap(serverNode, "command", serverCfg.Command)
		}
		if len(serverCfg.Args) > 0 {
			setStringArrayInMap(serverNode, "args", serverCfg.Args)
		}
		if serverCfg.Env != nil && len(serverCfg.Env) > 0 {
			envNode := ensureMap(serverNode, "env")
			for envKey, envValue := range serverCfg.Env {
				setStringInMap(envNode, envKey, envValue)
			}
		}
		if serverCfg.Transport != "" {
			setStringInMap(serverNode, "transport", serverCfg.Transport)
		}
		if serverCfg.URL != "" {
			setStringInMap(serverNode, "url", serverCfg.URL)
		}
		if serverCfg.Headers != nil && len(serverCfg.Headers) > 0 {
			headersNode := ensureMap(serverNode, "headers")
			for k, v := range serverCfg.Headers {
				setStringInMap(headersNode, k, v)
			}
		}
		if serverCfg.Description != "" {
			setStringInMap(serverNode, "description", serverCfg.Description)
		}
		if serverCfg.Timeout > 0 {
			setIntInMap(serverNode, "timeout", serverCfg.Timeout)
		}
		setBoolInMap(serverNode, "external_mcp_enable", serverCfg.ExternalMCPEnable)
		// save tool_enabled field(each/peritemstoolofenablestatus)
		if serverCfg.ToolEnabled != nil && len(serverCfg.ToolEnabled) > 0 {
			toolEnabledNode := ensureMap(serverNode, "tool_enabled")
			for toolName, enabled := range serverCfg.ToolEnabled {
				setBoolInMap(toolEnabledNode, toolName, enabled)
			}
		}
		originalFields, hasOriginal := originalConfigs[name]
		if hasOriginal {
			if enabledVal, hasEnabled := originalFields["enabled"]; hasEnabled {
				setBoolInMap(serverNode, "enabled", enabledVal)
			}
			if disabledVal, hasDisabled := originalFields["disabled"]; hasDisabled {
				if disabledVal {
					setBoolInMap(serverNode, "disabled", disabledVal)
				} else {
					setBoolInMap(serverNode, "enabled", true)
				}
			}
		}
		if serverCfg.Enabled {
			setBoolInMap(serverNode, "enabled", serverCfg.Enabled)
		}
		if serverCfg.Disabled {
			setBoolInMap(serverNode, "disabled", serverCfg.Disabled)
		} else if !hasOriginal && serverCfg.ExternalMCPEnable {
			setBoolInMap(serverNode, "enabled", true)
		}
	}
}
func setStringArrayInMap(mapNode *yaml.Node, key string, values []string) {
	_, valueNode := ensureKeyValue(mapNode, key)
	valueNode.Kind = yaml.SequenceNode
	valueNode.Tag = "!!seq"
	valueNode.Content = nil
	for _, v := range values {
		itemNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
		valueNode.Content = append(valueNode.Content, itemNode)
	}
}
type AddOrUpdateExternalMCPRequest struct {
	Config config.ExternalMCPServerConfig `json:"config"`
}

// ExternalMCPResponse externalMCPresponse
type ExternalMCPResponse struct {
	Config config.ExternalMCPServerConfig `json:"config"`
	Status string `json:"status"` // "connected", "disconnected", "disabled", "error", "connecting"
	ToolCount int `json:"tool_count"` // tool count
	Error string `json:"error,omitempty"`
}
