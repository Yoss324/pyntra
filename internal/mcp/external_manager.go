package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"pyntra/internal/config"

	"github.com/google/uuid"

	"go.uber.org/zap"
)
type ExternalMCPManager struct {
	clients map[string]ExternalMCPClient
	configs map[string]config.ExternalMCPServerConfig
	logger *zap.Logger
	storage MonitorStorage
	executions map[string]*ToolExecution
	stats map[string]*ToolStats
	errors map[string]string
	toolCounts map[string]int
	toolCountsMu sync.RWMutex
	toolCache map[string][]Tool
	toolCacheMu sync.RWMutex
	stopRefresh chan struct{}
	refreshWg sync.WaitGroup
	mu sync.RWMutex
}
func NewExternalMCPManager(logger *zap.Logger) *ExternalMCPManager {
	return NewExternalMCPManagerWithStorage(logger, nil)
}
func NewExternalMCPManagerWithStorage(logger *zap.Logger, storage MonitorStorage) *ExternalMCPManager {
	manager := &ExternalMCPManager{
		clients: make(map[string]ExternalMCPClient),
		configs: make(map[string]config.ExternalMCPServerConfig),
		logger: logger,
		storage: storage,
		executions: make(map[string]*ToolExecution),
		stats: make(map[string]*ToolStats),
		errors: make(map[string]string),
		toolCounts: make(map[string]int),
		toolCache: make(map[string][]Tool),
		stopRefresh: make(chan struct{}),
	}
	manager.startToolCountRefresh()
	return manager
}
func (m *ExternalMCPManager) LoadConfigs(cfg *config.ExternalMCPConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil || cfg.Servers == nil {
		return
	}

	m.configs = make(map[string]config.ExternalMCPServerConfig)
	for name, serverCfg := range cfg.Servers {
		m.configs[name] = serverCfg
	}
}
func (m *ExternalMCPManager) GetConfigs() map[string]config.ExternalMCPServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]config.ExternalMCPServerConfig)
	for k, v := range m.configs {
		result[k] = v
	}
	return result
}
func (m *ExternalMCPManager) AddOrUpdateConfig(name string, serverCfg config.ExternalMCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, exists := m.clients[name]; exists {
		client.Close()
		delete(m.clients, name)
	}

	m.configs[name] = serverCfg
	if m.isEnabled(serverCfg) {
		go m.connectClient(name, serverCfg)
	}

	return nil
}
func (m *ExternalMCPManager) RemoveConfig(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, exists := m.clients[name]; exists {
		client.Close()
		delete(m.clients, name)
	}

	delete(m.configs, name)
	m.toolCountsMu.Lock()
	delete(m.toolCounts, name)
	m.toolCountsMu.Unlock()
	m.toolCacheMu.Lock()
	delete(m.toolCache, name)
	m.toolCacheMu.Unlock()

	return nil
}
func (m *ExternalMCPManager) StartClient(name string) error {
	m.mu.Lock()
	serverCfg, exists := m.configs[name]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("configuration does not exist: %s", name)
	}
	m.mu.RLock()
	existingClient, hasClient := m.clients[name]
	m.mu.RUnlock()

	if hasClient {
		if existingClient.IsConnected() {
			m.mu.Lock()
			serverCfg.ExternalMCPEnable = true
			m.configs[name] = serverCfg
			m.mu.Unlock()
			return nil
		}
		existingClient.Close()
		m.mu.Lock()
		delete(m.clients, name)
		m.mu.Unlock()
	}
	m.mu.Lock()
	serverCfg.ExternalMCPEnable = true
	m.configs[name] = serverCfg
	delete(m.errors, name)
	m.mu.Unlock()
	client := m.createClient(serverCfg)
	if client == nil {
		return fmt.Errorf("cannot create client: unsupported transport mode")
	}
	m.setClientStatus(client, "connecting")
	m.mu.Lock()
	m.clients[name] = client
	m.mu.Unlock()
	go func() {
		if err := m.doConnect(name, serverCfg, client); err != nil {
			m.logger.Error("connectionMCPfailed",
				zap.String("name", name),
				zap.Error(err),
			)
			m.setClientStatus(client, "error")
			m.mu.Lock()
			m.errors[name] = err.Error()
			m.mu.Unlock()
			m.triggerToolCountRefresh()
		} else {
			m.mu.Lock()
			delete(m.errors, name)
			m.mu.Unlock()
			m.triggerToolCountRefresh()
			m.refreshToolCache(name, client)
			go func() {
				time.Sleep(2 * time.Second)
				m.triggerToolCountRefresh()
				m.refreshToolCache(name, client)
			}()
		}
	}()

	return nil
}
func (m *ExternalMCPManager) StopClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	serverCfg, exists := m.configs[name]
	if !exists {
		return fmt.Errorf("configuration does not exist: %s", name)
	}
	if client, exists := m.clients[name]; exists {
		client.Close()
		delete(m.clients, name)
	}
	delete(m.errors, name)
	m.toolCountsMu.Lock()
	m.toolCounts[name] = 0
	m.toolCountsMu.Unlock()
	serverCfg.ExternalMCPEnable = false
	m.configs[name] = serverCfg

	return nil
}
func (m *ExternalMCPManager) GetClient(name string) (ExternalMCPClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[name]
	return client, exists
}
func (m *ExternalMCPManager) GetError(name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.errors[name]
}
func (m *ExternalMCPManager) GetAllTools(ctx context.Context) ([]Tool, error) {
	m.mu.RLock()
	clients := make(map[string]ExternalMCPClient)
	for k, v := range m.clients {
		clients[k] = v
	}
	m.mu.RUnlock()

	var allTools []Tool
	var hasError bool
	var lastError error
	quickCtx, quickCancel := context.WithTimeout(ctx, 3*time.Second)
	defer quickCancel()

	for name, client := range clients {
		tools, err := m.getToolsForClient(name, client, quickCtx)
		if err != nil {
			hasError = true
			if lastError == nil {
				lastError = err
			}
			continue
		}
		for _, tool := range tools {
			tool.Name = fmt.Sprintf("%s::%s", name, tool.Name)
			allTools = append(allTools, tool)
		}
	}
	if hasError && len(allTools) == 0 {
		return nil, fmt.Errorf("fetchMCPtoolfailed: %w", lastError)
	}

	return allTools, nil
}
func (m *ExternalMCPManager) getToolsForClient(name string, client ExternalMCPClient, ctx context.Context) ([]Tool, error) {
	status := client.GetStatus()
	if status == "error" {
		m.logger.Debug("connectionfailedMCP()",
			zap.String("name", name),
			zap.String("status", status),
		)
		return nil, fmt.Errorf("MCPconnectionfailed: %s", name)
	}
	if client.IsConnected() {
		tools, err := client.ListTools(ctx)
		if err != nil {
			return m.getCachedTools(name, "connectionfetchfailed", err)
		}
		m.updateToolCache(name, tools)
		return tools, nil
	}
	if status == "disconnected" || status == "connecting" {
		return m.getCachedTools(name, fmt.Sprintf("(status: %s)", status), nil)
	}
	m.logger.Debug("MCP(Unknownstatus)",
		zap.String("name", name),
		zap.String("status", status),
	)
	return nil, fmt.Errorf("MCPstatusUnknown: %s (status: %s)", name, status)
}
func (m *ExternalMCPManager) getCachedTools(name, reason string, originalErr error) ([]Tool, error) {
	m.toolCacheMu.RLock()
	cachedTools, hasCache := m.toolCache[name]
	m.toolCacheMu.RUnlock()

	if hasCache && len(cachedTools) > 0 {
		m.logger.Debug("tool",
			zap.String("name", name),
			zap.String("reason", reason),
			zap.Int("count", len(cachedTools)),
			zap.Error(originalErr),
		)
		return cachedTools, nil
	}
	if originalErr != nil {
		return nil, fmt.Errorf("fetchMCPtoolfailed: %w", originalErr)
	}
	return nil, fmt.Errorf("MCPtool: %s", name)
}
func (m *ExternalMCPManager) updateToolCache(name string, tools []Tool) {
	m.toolCacheMu.Lock()
	m.toolCache[name] = tools
	m.toolCacheMu.Unlock()
	if len(tools) == 0 {
		m.logger.Warn("MCPbacktool",
			zap.String("name", name),
			zap.String("hint", ",toolis empty"),
		)
	} else {
		m.logger.Debug("toolupdate",
			zap.String("name", name),
			zap.Int("count", len(tools)),
		)
	}
}
func (m *ExternalMCPManager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, string, error) {
	var mcpName, actualToolName string
	if idx := findSubstring(toolName, "::"); idx > 0 {
		mcpName = toolName[:idx]
		actualToolName = toolName[idx+2:]
	} else {
		return nil, "", fmt.Errorf("invalidtoolnameFormat: %s", toolName)
	}

	client, exists := m.GetClient(mcpName)
	if !exists {
		return nil, "", fmt.Errorf("MCPdoes not exist: %s", mcpName)
	}
	if !client.IsConnected() {
		status := client.GetStatus()
		if status == "error" {
			errorMsg := m.GetError(mcpName)
			if errorMsg != "" {
				return nil, "", fmt.Errorf("MCPconnectionfailed: %s (error: %s)", mcpName, errorMsg)
			}
			return nil, "", fmt.Errorf("MCPconnectionfailed: %s", mcpName)
		}
		return nil, "", fmt.Errorf("MCPnot connected: %s (status: %s)", mcpName, status)
	}
	executionID := uuid.New().String()
	execution := &ToolExecution{
		ID: executionID,
		ToolName: toolName,
		Arguments: args,
		Status: "running",
		StartTime: time.Now(),
	}

	m.mu.Lock()
	m.executions[executionID] = execution
	m.cleanupOldExecutions()
	m.mu.Unlock()

	if m.storage != nil {
		if err := m.storage.SaveToolExecution(execution); err != nil {
			m.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
		}
	}
	result, err := client.CallTool(ctx, actualToolName, args)
	m.mu.Lock()
	now := time.Now()
	execution.EndTime = &now
	execution.Duration = now.Sub(execution.StartTime)

	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
	} else if result != nil && result.IsError {
		execution.Status = "failed"
		if len(result.Content) > 0 {
			execution.Error = result.Content[0].Text
		} else {
			execution.Error = "Tool execution returned an error result"
		}
		execution.Result = result
	} else {
		execution.Status = "completed"
		if result == nil {
			result = &ToolResult{
				Content: []Content{
					{Type: "text", Text: "Tool execution completed but returned no result"},
				},
			}
		}
		execution.Result = result
	}
	m.mu.Unlock()

	if m.storage != nil {
		if err := m.storage.SaveToolExecution(execution); err != nil {
			m.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
		}
	}
	failed := err != nil || (result != nil && result.IsError)
	m.updateStats(toolName, failed)
	if m.storage != nil {
		m.mu.Lock()
		delete(m.executions, executionID)
		m.mu.Unlock()
	}

	if err != nil {
		return nil, executionID, err
	}

	return result, executionID, nil
}
func (m *ExternalMCPManager) cleanupOldExecutions() {
	const maxExecutionsInMemory = 1000
	if len(m.executions) <= maxExecutionsInMemory {
		return
	}
	type execTime struct {
		id string
		startTime time.Time
	}
	var execs []execTime
	for id, exec := range m.executions {
		execs = append(execs, execTime{id: id, startTime: exec.StartTime})
	}
	for i := 0; i < len(execs)-1; i++ {
		for j := i + 1; j < len(execs); j++ {
			if execs[i].startTime.After(execs[j].startTime) {
				execs[i], execs[j] = execs[j], execs[i]
			}
		}
	}
	toDelete := len(m.executions) - maxExecutionsInMemory
	for i := 0; i < toDelete && i < len(execs); i++ {
		delete(m.executions, execs[i].id)
	}
}
func (m *ExternalMCPManager) GetExecution(id string) (*ToolExecution, bool) {
	m.mu.RLock()
	exec, exists := m.executions[id]
	m.mu.RUnlock()

	if exists {
		return exec, true
	}

	if m.storage != nil {
		exec, err := m.storage.GetToolExecution(id)
		if err == nil {
			return exec, true
		}
	}

	return nil, false
}
func (m *ExternalMCPManager) updateStats(toolName string, failed bool) {
	now := time.Now()
	if m.storage != nil {
		totalCalls := 1
		successCalls := 0
		failedCalls := 0
		if failed {
			failedCalls = 1
		} else {
			successCalls = 1
		}
		if err := m.storage.UpdateToolStats(toolName, totalCalls, successCalls, failedCalls, &now); err != nil {
			m.logger.Warn("savedatabasefailed", zap.Error(err))
		}
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stats[toolName] == nil {
		m.stats[toolName] = &ToolStats{
			ToolName: toolName,
		}
	}

	stats := m.stats[toolName]
	stats.TotalCalls++
	stats.LastCallTime = &now

	if failed {
		stats.FailedCalls++
	} else {
		stats.SuccessCalls++
	}
}
func (m *ExternalMCPManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := len(m.configs)
	enabled := 0
	disabled := 0
	connected := 0

	for name, cfg := range m.configs {
		if m.isEnabled(cfg) {
			enabled++
			if client, exists := m.clients[name]; exists && client.IsConnected() {
				connected++
			}
		} else {
			disabled++
		}
	}

	return map[string]interface{}{
		"total": total,
		"enabled": enabled,
		"disabled": disabled,
		"connected": connected,
	}
}
func (m *ExternalMCPManager) GetToolStats() map[string]*ToolStats {
	result := make(map[string]*ToolStats)
	if m.storage != nil {
		dbStats, err := m.storage.LoadToolStats()
		if err == nil {
			for k, v := range dbStats {
				if findSubstring(k, "::") > 0 {
					result[k] = v
				}
			}
		} else {
			m.logger.Warn("databaseloadfailed", zap.Error(err))
		}
	}
	m.mu.RLock()
	for k, v := range m.stats {
		if existing, exists := result[k]; exists {
			merged := &ToolStats{
				ToolName: k,
				TotalCalls: existing.TotalCalls + v.TotalCalls,
				SuccessCalls: existing.SuccessCalls + v.SuccessCalls,
				FailedCalls: existing.FailedCalls + v.FailedCalls,
			}
			if v.LastCallTime != nil && (existing.LastCallTime == nil || v.LastCallTime.After(*existing.LastCallTime)) {
				merged.LastCallTime = v.LastCallTime
			} else if existing.LastCallTime != nil {
				timeCopy := *existing.LastCallTime
				merged.LastCallTime = &timeCopy
			}
			result[k] = merged
		} else {
			statCopy := *v
			result[k] = &statCopy
		}
	}
	m.mu.RUnlock()

	return result
}
func (m *ExternalMCPManager) GetToolCount(name string) (int, error) {
	m.toolCountsMu.RLock()
	if count, exists := m.toolCounts[name]; exists {
		m.toolCountsMu.RUnlock()
		return count, nil
	}
	m.toolCountsMu.RUnlock()
	client, exists := m.GetClient(name)
	if !exists {
		return 0, fmt.Errorf("does not exist: %s", name)
	}

	if !client.IsConnected() {
		m.toolCountsMu.Lock()
		m.toolCounts[name] = 0
		m.toolCountsMu.Unlock()
		return 0, nil
	}
	m.triggerToolCountRefresh()
	return 0, nil
}
func (m *ExternalMCPManager) GetToolCounts() map[string]int {
	m.toolCountsMu.RLock()
	defer m.toolCountsMu.RUnlock()
	result := make(map[string]int)
	for k, v := range m.toolCounts {
		result[k] = v
	}
	return result
}
func (m *ExternalMCPManager) refreshToolCounts() {
	m.mu.RLock()
	clients := make(map[string]ExternalMCPClient)
	for k, v := range m.clients {
		clients[k] = v
	}
	m.mu.RUnlock()

	newCounts := make(map[string]int)
	type countResult struct {
		name string
		count int
	}
	resultChan := make(chan countResult, len(clients))

	for name, client := range clients {
		go func(n string, c ExternalMCPClient) {
			if !c.IsConnected() {
				resultChan <- countResult{name: n, count: 0}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			tools, err := c.ListTools(ctx)
			cancel()

			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "EOF") || strings.Contains(errStr, "client is closing") {
					m.logger.Warn("fetchMCPtoolfailed(SSE back tools/list )",
						zap.String("name", n),
						zap.String("hint", " SSE connection, GET MCP event: message JSON-RPC "),
						zap.Error(err),
					)
				} else {
					m.logger.Warn("fetchMCPtoolfailed,connection tools/list",
						zap.String("name", n),
						zap.Error(err),
					)
				}
				resultChan <- countResult{name: n, count: -1}
				return
			}

			resultChan <- countResult{name: n, count: len(tools)}
		}(name, client)
	}
	m.toolCountsMu.RLock()
	oldCounts := make(map[string]int)
	for k, v := range m.toolCounts {
		oldCounts[k] = v
	}
	m.toolCountsMu.RUnlock()

	for i := 0; i < len(clients); i++ {
		result := <-resultChan
		if result.count >= 0 {
			newCounts[result.name] = result.count
		} else {
			if oldCount, exists := oldCounts[result.name]; exists {
				newCounts[result.name] = oldCount
			} else {
				newCounts[result.name] = 0
			}
		}
	}
	m.toolCountsMu.Lock()
	for name, count := range newCounts {
		m.toolCounts[name] = count
	}
	for name, client := range clients {
		if !client.IsConnected() {
			m.toolCounts[name] = 0
		}
	}
	m.toolCountsMu.Unlock()
}
func (m *ExternalMCPManager) refreshToolCache(name string, client ExternalMCPClient) {
	if !client.IsConnected() {
		return
	}
	status := client.GetStatus()
	if status == "error" {
		m.logger.Debug("refreshtool(connectionfailed)",
			zap.String("name", name),
			zap.String("status", status),
		)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tools, err := client.ListTools(ctx)
	if err != nil {
		m.logger.Debug("refreshtoolfailed",
			zap.String("name", name),
			zap.Error(err),
		)
		return
	}
	m.updateToolCache(name, tools)
}
func (m *ExternalMCPManager) startToolCountRefresh() {
	m.refreshWg.Add(1)
	go func() {
		defer m.refreshWg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		m.refreshToolCounts()

		for {
			select {
			case <-ticker.C:
				m.refreshToolCounts()
			case <-m.stopRefresh:
				return
			}
		}
	}()
}
func (m *ExternalMCPManager) triggerToolCountRefresh() {
	go m.refreshToolCounts()
}
func (m *ExternalMCPManager) createClient(serverCfg config.ExternalMCPServerConfig) ExternalMCPClient {
	transport := serverCfg.Transport
	if transport == "" {
		if serverCfg.Command != "" {
			transport = "stdio"
		} else if serverCfg.URL != "" {
			transport = "http"
		} else {
			return nil
		}
	}

	switch transport {
	case "http":
		if serverCfg.URL == "" {
			return nil
		}
		return newLazySDKClient(serverCfg, m.logger)
	case "simple_http":
		if serverCfg.URL == "" {
			return nil
		}
		return newLazySDKClient(serverCfg, m.logger)
	case "stdio":
		if serverCfg.Command == "" {
			return nil
		}
		return newLazySDKClient(serverCfg, m.logger)
	case "sse":
		if serverCfg.URL == "" {
			return nil
		}
		return newLazySDKClient(serverCfg, m.logger)
	default:
		return nil
	}
}
func (m *ExternalMCPManager) doConnect(name string, serverCfg config.ExternalMCPServerConfig, client ExternalMCPClient) error {
	timeout := time.Duration(serverCfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		return err
	}

	m.logger.Info("External MCP client connected",
		zap.String("name", name),
	)

	return nil
}
func (m *ExternalMCPManager) setClientStatus(client ExternalMCPClient, status string) {
	if c, ok := client.(*lazySDKClient); ok {
		c.setStatus(status)
	}
}
func (m *ExternalMCPManager) connectClient(name string, serverCfg config.ExternalMCPServerConfig) error {
	client := m.createClient(serverCfg)
	if client == nil {
		return fmt.Errorf("cannot create client: unsupported transport mode")
	}
	m.setClientStatus(client, "connecting")
	timeout := time.Duration(serverCfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		m.logger.Error("initializeMCPfailed",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}
	m.mu.Lock()
	m.clients[name] = client
	m.mu.Unlock()

	m.logger.Info("External MCP client connected",
		zap.String("name", name),
	)
	m.triggerToolCountRefresh()
	m.mu.RLock()
	if client, exists := m.clients[name]; exists {
		m.refreshToolCache(name, client)
	}
	m.mu.RUnlock()

	return nil
}
func (m *ExternalMCPManager) isEnabled(cfg config.ExternalMCPServerConfig) bool {
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
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
func (m *ExternalMCPManager) StartAllEnabled() {
	m.mu.RLock()
	configs := make(map[string]config.ExternalMCPServerConfig)
	for k, v := range m.configs {
		configs[k] = v
	}
	m.mu.RUnlock()

	for name, cfg := range configs {
		if m.isEnabled(cfg) {
			go func(n string, c config.ExternalMCPServerConfig) {
				if err := m.connectClient(n, c); err != nil {
					errStr := strings.ToLower(err.Error())
					isConnectionRefused := strings.Contains(errStr, "connection refused") ||
						strings.Contains(errStr, "dial tcp") ||
						strings.Contains(errStr, "connect: connection refused")

					if isConnectionRefused {
						fields := []zap.Field{
							zap.String("name", n),
							zap.String("message", ",.connection,retry"),
							zap.Error(err),
						}
						transport := c.Transport
						if transport == "" {
							if c.Command != "" {
								transport = "stdio"
							} else if c.URL != "" {
								transport = "http"
							}
						}

						if transport == "http" && c.URL != "" {
							fields = append(fields, zap.String("url", c.URL))
						} else if transport == "stdio" && c.Command != "" {
							fields = append(fields, zap.String("command", c.Command))
						}

						m.logger.Warn("MCP", fields...)
					} else {
						m.logger.Error("MCPfailed",
							zap.String("name", n),
							zap.Error(err),
						)
					}
				}
			}(name, cfg)
		}
	}
}
func (m *ExternalMCPManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		client.Close()
		delete(m.clients, name)
	}
	m.toolCountsMu.Lock()
	m.toolCounts = make(map[string]int)
	m.toolCountsMu.Unlock()
	m.toolCacheMu.Lock()
	m.toolCache = make(map[string][]Tool)
	m.toolCacheMu.Unlock()
	select {
	case <-m.stopRefresh:
	default:
		close(m.stopRefresh)
		m.refreshWg.Wait()
	}
}
