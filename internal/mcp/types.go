package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)
type ExternalMCPClient interface {
	Initialize(ctx context.Context) error
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
	Close() error
	IsConnected() bool
	GetStatus() string
}
const (
	MessageTypeRequest  = "request"
	MessageTypeResponse = "response"
	MessageTypeError    = "error"
	MessageTypeNotify   = "notify"
)
const ProtocolVersion = "2024-11-05"
type MessageID struct {
	value interface{}
}
func (m *MessageID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		m.value = nil
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		m.value = str
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		m.value = num
		return nil
	}

	return fmt.Errorf("invalid id type")
}
func (m MessageID) MarshalJSON() ([]byte, error) {
	if m.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(m.value)
}
func (m MessageID) String() string {
	if m.value == nil {
		return ""
	}
	return fmt.Sprintf("%v", m.value)
}
func (m MessageID) Value() interface{} {
	return m.value
}
type Message struct {
	ID      MessageID       `json:"id,omitempty"`
	Type    string          `json:"-"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	Version string          `json:"jsonrpc,omitempty"`
}
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
type Tool struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ShortDescription string                 `json:"shortDescription,omitempty"`
	InputSchema      map[string]interface{} `json:"inputSchema"`
}
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type InitializeResponse struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}
type ServerCapabilities struct {
	Tools     map[string]interface{} `json:"tools,omitempty"`
	Prompts   map[string]interface{} `json:"prompts,omitempty"`
	Resources map[string]interface{} `json:"resources,omitempty"`
	Sampling  map[string]interface{} `json:"sampling,omitempty"`
}
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type ListToolsRequest struct{}
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}
type ListPromptsResponse struct {
	Prompts []Prompt `json:"prompts"`
}
type ListResourcesResponse struct {
	Resources []Resource `json:"resources"`
}
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}
type CallToolResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}
type ToolExecution struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"toolName"`
	Arguments map[string]interface{} `json:"arguments"`
	Status    string                 `json:"status"` // pending, running, completed, failed
	Result    *ToolResult            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	StartTime time.Time              `json:"startTime"`
	EndTime   *time.Time             `json:"endTime,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
}
type ToolStats struct {
	ToolName     string     `json:"toolName"`
	TotalCalls   int        `json:"totalCalls"`
	SuccessCalls int        `json:"successCalls"`
	FailedCalls  int        `json:"failedCalls"`
	LastCallTime *time.Time `json:"lastCallTime,omitempty"`
}
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}
type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}
type GetPromptResponse struct {
	Messages []PromptMessage `json:"messages"`
}
type PromptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}
type ReadResourceRequest struct {
	URI string `json:"uri"`
}
type ReadResourceResponse struct {
	Contents []ResourceContent `json:"contents"`
}
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}
type SamplingRequest struct {
	Messages    []SamplingMessage `json:"messages"`
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"maxTokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"topP,omitempty"`
}
type SamplingMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type SamplingResponse struct {
	Content    []SamplingContent `json:"content"`
	Model      string            `json:"model,omitempty"`
	StopReason string            `json:"stopReason,omitempty"`
}
type SamplingContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
