package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)
type MonitorStorage interface {
	SaveToolExecution(exec *ToolExecution) error
	LoadToolExecutions() ([]*ToolExecution, error)
	GetToolExecution(id string) (*ToolExecution, error)
	SaveToolStats(toolName string, stats *ToolStats) error
	LoadToolStats() (map[string]*ToolStats, error)
	UpdateToolStats(toolName string, totalCalls, successCalls, failedCalls int, lastCallTime *time.Time) error
}
type Server struct {
	tools map[string]ToolHandler
	toolDefs map[string]Tool
	executions map[string]*ToolExecution
	stats map[string]*ToolStats
	prompts map[string]*Prompt
	resources map[string]*Resource
	storage MonitorStorage
	mu sync.RWMutex
	logger *zap.Logger
	maxExecutionsInMemory int
	sseClients map[string]*sseClient
}

type sseClient struct {
	id string
	send chan []byte
}
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
func NewServer(logger *zap.Logger) *Server {
	return NewServerWithStorage(logger, nil)
}
func NewServerWithStorage(logger *zap.Logger, storage MonitorStorage) *Server {
	s := &Server{
		tools: make(map[string]ToolHandler),
		toolDefs: make(map[string]Tool),
		executions: make(map[string]*ToolExecution),
		stats: make(map[string]*ToolStats),
		prompts: make(map[string]*Prompt),
		resources: make(map[string]*Resource),
		storage: storage,
		logger: logger,
		maxExecutionsInMemory: 1000,
		sseClients: make(map[string]*sseClient),
	}
	s.initDefaultPrompts()
	s.initDefaultResources()

	return s
}
func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = handler
	s.toolDefs[tool.Name] = tool
	resourceURI := fmt.Sprintf("tool://%s", tool.Name)
	s.resources[resourceURI] = &Resource{
		URI: resourceURI,
		Name: fmt.Sprintf("%stool", tool.Name),
		Description: tool.Description,
		MimeType: "text/plain",
	}
}
func (s *Server) ClearTools() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = make(map[string]ToolHandler)
	s.toolDefs = make(map[string]Tool)
	newResources := make(map[string]*Resource)
	for uri, resource := range s.resources {
		if !strings.HasPrefix(uri, "tool://") {
			newResources[uri] = resource
		}
	}
	s.resources = newResources
}
func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		s.handleSSE(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if sessionID := r.URL.Query().Get("sessionid"); sessionID != "" {
		s.serveSSESessionMessage(w, r, sessionID)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		s.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}

	response := s.handleMessage(&msg)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func (s *Server) serveSSESessionMessage(w http.ResponseWriter, r *http.Request, sessionID string) {
	s.mu.RLock()
	client, exists := s.sseClients[sessionID]
	s.mu.RUnlock()
	if !exists || client == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "failed to parse body", http.StatusBadRequest)
		return
	}

	response := s.handleMessage(&msg)
	if response == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	select {
	case client.send <- respBytes:
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "session send buffer full", http.StatusServiceUnavailable)
	}
}
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sessionID := uuid.New().String()
	client := &sseClient{
		id: sessionID,
		send: make(chan []byte, 32),
	}

	s.addSSEClient(client)
	defer s.removeSSEClient(client.id)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if r.URL.Scheme != "" {
		scheme = r.URL.Scheme
	}
	endpointURL := fmt.Sprintf("%s://%s%s?sessionid=%s", scheme, r.Host, r.URL.Path, sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
func (s *Server) addSSEClient(client *sseClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sseClients[client.id] = client
}
func (s *Server) removeSSEClient(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client, exists := s.sseClients[id]; exists {
		close(client.send)
		delete(s.sseClients, id)
	}
}
func (s *Server) handleMessage(msg *Message) *Message {
	isNotification := msg.ID.Value() == nil || msg.ID.String() == ""
	if !isNotification && msg.ID.String() == "" {
		msg.ID = MessageID{value: uuid.New().String()}
	}

	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "tools/list":
		return s.handleListTools(msg)
	case "tools/call":
		return s.handleCallTool(msg)
	case "prompts/list":
		return s.handleListPrompts(msg)
	case "prompts/get":
		return s.handleGetPrompt(msg)
	case "resources/list":
		return s.handleListResources(msg)
	case "resources/read":
		return s.handleReadResource(msg)
	case "sampling/request":
		return s.handleSamplingRequest(msg)
	case "notifications/initialized":
		s.logger.Debug(" initialized ")
		return nil
	case "":
		if isNotification {
			s.logger.Debug("message")
			return nil
		}
		fallthrough
	default:
		if isNotification {
			s.logger.Debug("Unknown", zap.String("method", msg.Method))
			return nil
		}
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32601, Message: "Method not found"},
		}
	}
}
func (s *Server) handleInitialize(msg *Message) *Message {
	var req InitializeRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32602, Message: "Invalid params"},
		}
	}

	response := InitializeResponse{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: map[string]interface{}{
				"listChanged": true,
			},
			Prompts: map[string]interface{}{
				"listChanged": true,
			},
			Resources: map[string]interface{}{
				"subscribe": true,
				"listChanged": true,
			},
			Sampling: map[string]interface{}{},
		},
		ServerInfo: ServerInfo{
			Name: "Pyntra",
			Version: "1.0.0",
		},
	}

	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) handleListTools(msg *Message) *Message {
	s.mu.RLock()
	tools := make([]Tool, 0, len(s.toolDefs))
	for _, tool := range s.toolDefs {
		tools = append(tools, tool)
	}
	s.mu.RUnlock()
	s.logger.Debug("tools/list ", zap.Int("backtool", len(tools)))

	response := ListToolsResponse{Tools: tools}
	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) handleCallTool(msg *Message) *Message {
	var req CallToolRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32602, Message: "Invalid params"},
		}
	}

	executionID := uuid.New().String()
	execution := &ToolExecution{
		ID: executionID,
		ToolName: req.Name,
		Arguments: req.Arguments,
		Status: "running",
		StartTime: time.Now(),
	}

	s.mu.Lock()
	s.executions[executionID] = execution
	s.cleanupOldExecutions()
	s.mu.Unlock()

	if s.storage != nil {
		if err := s.storage.SaveToolExecution(execution); err != nil {
			s.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
		}
	}

	s.mu.RLock()
	handler, exists := s.tools[req.Name]
	s.mu.RUnlock()

	if !exists {
		execution.Status = "failed"
		execution.Error = "Tool not found"
		now := time.Now()
		execution.EndTime = &now
		execution.Duration = now.Sub(execution.StartTime)

		if s.storage != nil {
			if err := s.storage.SaveToolExecution(execution); err != nil {
				s.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
			}
			s.mu.Lock()
			delete(s.executions, executionID)
			s.mu.Unlock()
		}

		s.updateStats(req.Name, true)

		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32601, Message: "Tool not found"},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	s.logger.Info("starttool",
		zap.String("toolName", req.Name),
		zap.Any("arguments", req.Arguments),
	)

	result, err := handler(ctx, req.Arguments)
	now := time.Now()
	var failed bool
	var finalResult *ToolResult

	s.mu.Lock()
	execution.EndTime = &now
	execution.Duration = now.Sub(execution.StartTime)

	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		failed = true
	} else if result != nil && result.IsError {
		execution.Status = "failed"
		if len(result.Content) > 0 {
			execution.Error = result.Content[0].Text
		} else {
			execution.Error = "Tool execution returned an error result"
		}
		execution.Result = result
		failed = true
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
		failed = false
	}

	finalResult = execution.Result
	s.mu.Unlock()

	if s.storage != nil {
		if err := s.storage.SaveToolExecution(execution); err != nil {
			s.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
		}
	}

	s.updateStats(req.Name, failed)

	if s.storage != nil {
		s.mu.Lock()
		delete(s.executions, executionID)
		s.mu.Unlock()
	}

	if err != nil {
		s.logger.Error("toolfailed",
			zap.String("toolName", req.Name),
			zap.Error(err),
		)

		errorResult, _ := json.Marshal(CallToolResponse{
			Content: []Content{
				{Type: "text", Text: fmt.Sprintf("toolfailed: %v", err)},
			},
			IsError: true,
		})
		return &Message{
			ID: msg.ID,
			Type: MessageTypeResponse,
			Version: "2.0",
			Result: errorResult,
		}
	}

	if finalResult != nil && finalResult.IsError {
		s.logger.Warn("Tool execution returned an error result",
			zap.String("toolName", req.Name),
		)

		errorResult, _ := json.Marshal(CallToolResponse{
			Content: finalResult.Content,
			IsError: true,
		})
		return &Message{
			ID: msg.ID,
			Type: MessageTypeResponse,
			Version: "2.0",
			Result: errorResult,
		}
	}

	if finalResult == nil {
		finalResult = &ToolResult{
			Content: []Content{
				{Type: "text", Text: "Tool execution completed but returned no result"},
			},
		}
	}

	resultJSON, _ := json.Marshal(CallToolResponse{
		Content: finalResult.Content,
		IsError: false,
	})

	s.logger.Info("toolcompleted",
		zap.String("toolName", req.Name),
		zap.Bool("isError", finalResult.IsError),
	)

	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: resultJSON,
	}
}
func (s *Server) updateStats(toolName string, failed bool) {
	now := time.Now()
	if s.storage != nil {
		totalCalls := 1
		successCalls := 0
		failedCalls := 0
		if failed {
			failedCalls = 1
		} else {
			successCalls = 1
		}
		if err := s.storage.UpdateToolStats(toolName, totalCalls, successCalls, failedCalls, &now); err != nil {
			s.logger.Warn("savedatabasefailed", zap.Error(err))
		}
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats[toolName] == nil {
		s.stats[toolName] = &ToolStats{
			ToolName: toolName,
		}
	}

	stats := s.stats[toolName]
	stats.TotalCalls++
	stats.LastCallTime = &now

	if failed {
		stats.FailedCalls++
	} else {
		stats.SuccessCalls++
	}
}
func (s *Server) GetExecution(id string) (*ToolExecution, bool) {
	s.mu.RLock()
	exec, exists := s.executions[id]
	s.mu.RUnlock()

	if exists {
		return exec, true
	}

	if s.storage != nil {
		exec, err := s.storage.GetToolExecution(id)
		if err == nil {
			return exec, true
		}
	}

	return nil, false
}
func (s *Server) loadHistoricalData() {
	if s.storage == nil {
		return
	}
	executions, err := s.storage.LoadToolExecutions()
	if err != nil {
		s.logger.Warn("loadexecution recordfailed", zap.Error(err))
	} else {
		s.mu.Lock()
		for _, exec := range executions {
			if len(s.executions) < s.maxExecutionsInMemory {
				s.executions[exec.ID] = exec
			} else {
				break
			}
		}
		s.mu.Unlock()
		s.logger.Info("loadexecution record", zap.Int("count", len(executions)))
	}
	stats, err := s.storage.LoadToolStats()
	if err != nil {
		s.logger.Warn("loadfailed", zap.Error(err))
	} else {
		s.mu.Lock()
		for k, v := range stats {
			s.stats[k] = v
		}
		s.mu.Unlock()
		s.logger.Info("load", zap.Int("count", len(stats)))
	}
}
func (s *Server) GetAllExecutions() []*ToolExecution {
	if s.storage != nil {
		dbExecutions, err := s.storage.LoadToolExecutions()
		if err == nil {
			execMap := make(map[string]*ToolExecution)
			for _, exec := range dbExecutions {
				if _, exists := execMap[exec.ID]; !exists {
					execMap[exec.ID] = exec
				}
			}

			s.mu.RLock()
			for id, exec := range s.executions {
				if _, exists := execMap[id]; !exists {
					execMap[id] = exec
				}
			}
			s.mu.RUnlock()

			result := make([]*ToolExecution, 0, len(execMap))
			for _, exec := range execMap {
				result = append(result, exec)
			}
			return result
		} else {
			s.logger.Warn("databasefailed to load execution records", zap.Error(err))
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	memExecutions := make([]*ToolExecution, 0, len(s.executions))
	for _, exec := range s.executions {
		memExecutions = append(memExecutions, exec)
	}
	return memExecutions
}
func (s *Server) GetStats() map[string]*ToolStats {
	if s.storage != nil {
		dbStats, err := s.storage.LoadToolStats()
		if err == nil {
			return dbStats
		}
		s.logger.Warn("databaseloadfailed", zap.Error(err))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	memStats := make(map[string]*ToolStats)
	for k, v := range s.stats {
		statCopy := *v
		memStats[k] = &statCopy
	}

	return memStats
}
func (s *Server) GetAllTools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.toolDefs))
	for _, tool := range s.toolDefs {
		tools = append(tools, tool)
	}
	return tools
}
func (s *Server) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, string, error) {
	s.mu.RLock()
	handler, exists := s.tools[toolName]
	s.mu.RUnlock()

	if !exists {
		return nil, "", fmt.Errorf("tool %s ", toolName)
	}
	executionID := uuid.New().String()
	execution := &ToolExecution{
		ID: executionID,
		ToolName: toolName,
		Arguments: args,
		Status: "running",
		StartTime: time.Now(),
	}

	s.mu.Lock()
	s.executions[executionID] = execution
	s.cleanupOldExecutions()
	s.mu.Unlock()

	if s.storage != nil {
		if err := s.storage.SaveToolExecution(execution); err != nil {
			s.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
		}
	}

	result, err := handler(ctx, args)

	s.mu.Lock()
	now := time.Now()
	execution.EndTime = &now
	execution.Duration = now.Sub(execution.StartTime)
	var failed bool
	var finalResult *ToolResult

	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		failed = true
	} else if result != nil && result.IsError {
		execution.Status = "failed"
		if len(result.Content) > 0 {
			execution.Error = result.Content[0].Text
		} else {
			execution.Error = "Tool execution returned an error result"
		}
		execution.Result = result
		failed = true
		finalResult = result
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
		finalResult = result
		failed = false
	}

	if finalResult == nil {
		finalResult = execution.Result
	}
	s.mu.Unlock()

	if s.storage != nil {
		if err := s.storage.SaveToolExecution(execution); err != nil {
			s.logger.Warn("saveexecution recorddatabasefailed", zap.Error(err))
		}
	}

	s.updateStats(toolName, failed)

	if s.storage != nil {
		s.mu.Lock()
		delete(s.executions, executionID)
		s.mu.Unlock()
	}

	if err != nil {
		return nil, executionID, err
	}

	return finalResult, executionID, nil
}
func (s *Server) cleanupOldExecutions() {
	if len(s.executions) <= s.maxExecutionsInMemory {
		return
	}
	type execWithTime struct {
		id string
		startTime time.Time
	}
	execs := make([]execWithTime, 0, len(s.executions))
	for id, exec := range s.executions {
		execs = append(execs, execWithTime{
			id: id,
			startTime: exec.StartTime,
		})
	}
	sort.Slice(execs, func(i, j int) bool {
		return execs[i].startTime.Before(execs[j].startTime)
	})
	toDelete := len(s.executions) - s.maxExecutionsInMemory
	for i := 0; i < toDelete; i++ {
		delete(s.executions, execs[i].id)
	}

	s.logger.Debug("execution record",
		zap.Int("before", len(execs)),
		zap.Int("after", len(s.executions)),
		zap.Int("deleted", toDelete),
	)
}
func (s *Server) initDefaultPrompts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts["security_scan"] = &Prompt{
		Name: "security_scan",
		Description: "generatescanTaskshint",
		Arguments: []PromptArgument{
			{Name: "target", Description: "scan(IP)", Required: true},
			{Name: "scan_type", Description: "scan(port, vuln, web)", Required: false},
		},
	}
	s.prompts["penetration_test"] = &Prompt{
		Name: "penetration_test",
		Description: "generatePenetration testingTaskshint",
		Arguments: []PromptArgument{
			{Name: "target", Description: "", Required: true},
			{Name: "scope", Description: "", Required: false},
		},
	}
}
func (s *Server) initDefaultResources() {
}
func (s *Server) handleListPrompts(msg *Message) *Message {
	s.mu.RLock()
	prompts := make([]Prompt, 0, len(s.prompts))
	for _, prompt := range s.prompts {
		prompts = append(prompts, *prompt)
	}
	s.mu.RUnlock()

	response := ListPromptsResponse{
		Prompts: prompts,
	}
	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) handleGetPrompt(msg *Message) *Message {
	var req GetPromptRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32602, Message: "Invalid params"},
		}
	}

	s.mu.RLock()
	prompt, exists := s.prompts[req.Name]
	s.mu.RUnlock()

	if !exists {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32601, Message: "Prompt not found"},
		}
	}
	messages := s.generatePromptMessages(prompt, req.Arguments)

	response := GetPromptResponse{
		Messages: messages,
	}
	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) generatePromptMessages(prompt *Prompt, args map[string]interface{}) []PromptMessage {
	messages := []PromptMessage{}

	switch prompt.Name {
	case "security_scan":
		target, _ := args["target"].(string)
		scanType, _ := args["scan_type"].(string)
		if scanType == "" {
			scanType = "comprehensive"
		}

		content := fmt.Sprintf(` %s %sscan.:
1. scan
2. Vulnerability detection
3. WebapplySecurity testing
4. generate`, target, scanType)

		messages = append(messages, PromptMessage{
			Role: "user",
			Content: content,
		})

	case "penetration_test":
		target, _ := args["target"].(string)
		scope, _ := args["scope"].(string)

		content := fmt.Sprintf(` %s Penetration testing.`, target)
		if scope != "" {
			content += fmt.Sprintf(":%s", scope)
		}
		content += "\nOWASP Top 10Security testing."

		messages = append(messages, PromptMessage{
			Role: "user",
			Content: content,
		})

	default:
		messages = append(messages, PromptMessage{
			Role: "user",
			Content: "Security testingTasks",
		})
	}

	return messages
}
func (s *Server) handleListResources(msg *Message) *Message {
	s.mu.RLock()
	resources := make([]Resource, 0, len(s.resources))
	for _, resource := range s.resources {
		resources = append(resources, *resource)
	}
	s.mu.RUnlock()

	response := ListResourcesResponse{
		Resources: resources,
	}
	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) handleReadResource(msg *Message) *Message {
	var req ReadResourceRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32602, Message: "Invalid params"},
		}
	}

	s.mu.RLock()
	resource, exists := s.resources[req.URI]
	s.mu.RUnlock()

	if !exists {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32601, Message: "Resource not found"},
		}
	}
	content := s.generateResourceContent(resource)

	response := ReadResourceResponse{
		Contents: []ResourceContent{content},
	}
	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) generateResourceContent(resource *Resource) ResourceContent {
	content := ResourceContent{
		URI: resource.URI,
		MimeType: resource.MimeType,
	}
	if strings.HasPrefix(resource.URI, "tool://") {
		toolName := strings.TrimPrefix(resource.URI, "tool://")
		content.Text = s.generateToolDocumentation(toolName, resource)
	} else {
		content.Text = resource.Description
	}

	return content
}
func (s *Server) generateToolDocumentation(toolName string, resource *Resource) string {
	s.mu.RLock()
	tool, hasTool := s.toolDefs[toolName]
	s.mu.RUnlock()
	if hasTool {
		doc := fmt.Sprintf("%s\n\n", resource.Description)
		if tool.InputSchema != nil {
			if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
				doc += ":\n"
				for paramName, paramInfo := range props {
					if paramMap, ok := paramInfo.(map[string]interface{}); ok {
						if desc, ok := paramMap["description"].(string); ok {
							doc += fmt.Sprintf("- %s: %s\n", paramName, desc)
						}
					}
				}
			}
		}
		return doc
	}
	return resource.Description
}
func (s *Server) handleSamplingRequest(msg *Message) *Message {
	var req SamplingRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return &Message{
			ID: msg.ID,
			Type: MessageTypeError,
			Version: "2.0",
			Error: &Error{Code: -32602, Message: "Invalid params"},
		}
	}
	s.logger.Warn("Sampling request received but not fully implemented",
		zap.Any("request", req),
	)

	response := SamplingResponse{
		Content: []SamplingContent{
			{
				Type: "text",
				Text: "configLLM.Agent Loop APIAIChat.",
			},
		},
		StopReason: "length",
	}
	result, _ := json.Marshal(response)
	return &Message{
		ID: msg.ID,
		Type: MessageTypeResponse,
		Version: "2.0",
		Result: result,
	}
}
func (s *Server) RegisterPrompt(prompt *Prompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts[prompt.Name] = prompt
}
func (s *Server) RegisterResource(resource *Resource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[resource.URI] = resource
}
func (s *Server) HandleStdio() error {
	decoder := json.NewDecoder(os.Stdin)
	stdout := bufio.NewWriter(os.Stdout)
	encoder := json.NewEncoder(stdout)

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			s.logger.Error("messagefailed", zap.Error(err))
			errorMsg := Message{
				ID: msg.ID,
				Type: MessageTypeError,
				Version: "2.0",
				Error: &Error{Code: -32700, Message: "Parse error", Data: err.Error()},
			}
			if err := encoder.Encode(errorMsg); err != nil {
				return fmt.Errorf("errorfailed: %w", err)
			}
			if err := stdout.Flush(); err != nil {
				return fmt.Errorf("failed to flush stdout: %w", err)
			}
			continue
		}
		response := s.handleMessage(&msg)
		if response == nil {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("failed: %w", err)
		}
		if err := stdout.Flush(); err != nil {
			return fmt.Errorf("failed to flush stdout: %w", err)
		}
	}

	return nil
}
func (s *Server) sendError(w http.ResponseWriter, id interface{}, code int, message, data string) {
	var msgID MessageID
	if id != nil {
		msgID = MessageID{value: id}
	}
	response := Message{
		ID: msgID,
		Type: MessageTypeError,
		Version: "2.0",
		Error: &Error{Code: code, Message: message, Data: data},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
