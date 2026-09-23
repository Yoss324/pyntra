package handler

import (
	"net/http"
	"time"

	"pyntra/internal/database"
	"pyntra/internal/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OpenAPIHandler OpenAPI processor
type OpenAPIHandler struct {
	db *database.DB
	logger *zap.Logger
	resultStorage storage.ResultStorage
	conversationHdlr *ConversationHandler
	agentHdlr *AgentHandler
}

// NewOpenAPIHandler create new OpenAPI processor
func NewOpenAPIHandler(db *database.DB, logger *zap.Logger, resultStorage storage.ResultStorage, conversationHdlr *ConversationHandler, agentHdlr *AgentHandler) *OpenAPIHandler {
	return &OpenAPIHandler{
		db: db,
		logger: logger,
		resultStorage: resultStorage,
		conversationHdlr: conversationHdlr,
		agentHdlr: agentHdlr,
	}
}

// GetOpenAPISpec get OpenAPI specification
func (h *OpenAPIHandler) GetOpenAPISpec(c *gin.Context) {
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "Pyntra API",
			"description": "AI-driven automated security testing platform API documentation",
			"version": "1.0.0",
			"contact": map[string]interface{}{
				"name": "Pyntra",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": scheme + "://" + host,
				"description": "current server",
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
					"description": "use Bearer Token for authentication. Token can be obtained through /api/auth/login interface.",
				},
			},
			"schemas": map[string]interface{}{
				"CreateConversationRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type": "string",
							"description": "conversation title",
							"example": "web application security testing",
						},
					},
				},
				"Conversation": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
							"example": "550e8400-e29b-41d4-a716-446655440000",
						},
						"title": map[string]interface{}{
							"type": "string",
							"description": "conversation title",
							"example": "web application security testing",
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
						"updatedAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "update time",
						},
					},
				},
				"ConversationDetail": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
						"title": map[string]interface{}{
							"type": "string",
							"description": "conversation title",
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "conversation status: active (in progress), completed (completed), failed (failed)",
							"enum": []string{"active", "completed", "failed"},
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
						"updatedAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "update time",
						},
						"messages": map[string]interface{}{
							"type": "array",
							"description": "message list",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/Message",
							},
						},
						"messageCount": map[string]interface{}{
							"type": "integer",
							"description": "message count",
						},
					},
				},
				"Message": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "message ID",
						},
						"conversationId": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
						"role": map[string]interface{}{
							"type": "string",
							"description": "message role: user (user), assistant (assistant)",
							"enum": []string{"user", "assistant"},
						},
						"content": map[string]interface{}{
							"type": "string",
							"description": "message content",
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
					},
				},
				"ConversationResults": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"conversationId": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
						"messages": map[string]interface{}{
							"type": "array",
							"description": "message list",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/Message",
							},
						},
						"vulnerabilities": map[string]interface{}{
							"type": "array",
							"description": "discovered vulnerabilities list",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/Vulnerability",
							},
						},
						"executionResults": map[string]interface{}{
							"type": "array",
							"description": "execution results list",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ExecutionResult",
							},
						},
					},
				},
				"Vulnerability": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "vulnerability ID",
						},
						"title": map[string]interface{}{
							"type": "string",
							"description": "vulnerability title",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "vulnerability description",
						},
						"severity": map[string]interface{}{
							"type": "string",
							"description": "severity level",
							"enum": []string{"critical", "high", "medium", "low", "info"},
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "status",
							"enum": []string{"open", "closed", "fixed"},
						},
						"target": map[string]interface{}{
							"type": "string",
							"description": "affected target",
						},
					},
				},
				"ExecutionResult": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "execution ID",
						},
						"toolName": map[string]interface{}{
							"type": "string",
							"description": "tool name",
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "execution status",
							"enum": []string{"success", "failed", "running"},
						},
						"result": map[string]interface{}{
							"type": "string",
							"description": "execution result",
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
					},
				},
				"Error": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error": map[string]interface{}{
							"type": "string",
							"description": "error message",
						},
					},
				},
				"LoginRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"password"},
					"properties": map[string]interface{}{
						"password": map[string]interface{}{
							"type": "string",
							"description": "login password",
						},
					},
				},
				"LoginResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"token": map[string]interface{}{
							"type": "string",
							"description": "authentication token",
						},
						"expires_at": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "token expiration time",
						},
						"session_duration_hr": map[string]interface{}{
							"type": "integer",
							"description": "session duration (hours)",
						},
					},
				},
				"ChangePasswordRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"oldPassword", "newPassword"},
					"properties": map[string]interface{}{
						"oldPassword": map[string]interface{}{
							"type": "string",
							"description": "current password",
						},
						"newPassword": map[string]interface{}{
							"type": "string",
							"description": "new password (at least 8 characters)",
						},
					},
				},
				"UpdateConversationRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"title"},
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type": "string",
							"description": "conversation title",
						},
					},
				},
				"Group": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "group ID",
						},
						"name": map[string]interface{}{
							"type": "string",
							"description": "group name",
						},
						"icon": map[string]interface{}{
							"type": "string",
							"description": "group icon",
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
						"updatedAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "update time",
						},
					},
				},
				"CreateGroupRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"name"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "group name",
						},
						"icon": map[string]interface{}{
							"type": "string",
							"description": "group icon(optional)",
						},
					},
				},
				"UpdateGroupRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"name"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "group name",
						},
						"icon": map[string]interface{}{
							"type": "string",
							"description": "group icon",
						},
					},
				},
				"AddConversationToGroupRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"conversationId", "groupId"},
					"properties": map[string]interface{}{
						"conversationId": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
						"groupId": map[string]interface{}{
							"type": "string",
							"description": "group ID",
						},
					},
				},
				"BatchTaskRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"tasks"},
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type": "string",
							"description": "task title (optional)",
						},
						"tasks": map[string]interface{}{
							"type": "array",
							"description": "task list, one task per row",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"role": map[string]interface{}{
							"type": "string",
							"description": "role name (optional)",
						},
						"agentMode": map[string]interface{}{
							"type": "string",
							"description": "agent mode: single (native ReAct) | eino_single (Eino ADK single agent) | deep | plan_execute | supervisor; react same as single; old value multi treated as deep",
							"enum": []string{"single", "eino_single", "deep", "plan_execute", "supervisor", "multi", "react"},
						},
						"scheduleMode": map[string]interface{}{
							"type": "string",
							"description": "scheduling method (manual | cron)",
							"enum": []string{"manual", "cron"},
						},
						"cronExpr": map[string]interface{}{
							"type": "string",
							"description": "Cron expression (required when scheduleMode=cron)",
						},
						"executeNow": map[string]interface{}{
							"type": "boolean",
							"description": "whether to execute immediately after creation (default false)",
						},
					},
				},
				"BatchQueue": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "queueID",
						},
						"title": map[string]interface{}{
							"type": "string",
							"description": "queue title",
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "queue status",
							"enum": []string{"pending", "running", "paused", "completed", "failed"},
						},
						"tasks": map[string]interface{}{
							"type": "array",
							"description": "task list",
							"items": map[string]interface{}{
								"type": "object",
							},
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
					},
				},
				"CancelAgentLoopRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"conversationId"},
					"properties": map[string]interface{}{
						"conversationId": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
					},
				},
				"AgentTask": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"conversationId": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "task status",
							"enum": []string{"running", "completed", "failed", "cancelled", "timeout"},
						},
						"startedAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "start time",
						},
					},
				},
				"CreateVulnerabilityRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"conversation_id", "title", "severity"},
					"properties": map[string]interface{}{
						"conversation_id": map[string]interface{}{
							"type": "string",
							"description": "conversation ID",
						},
						"title": map[string]interface{}{
							"type": "string",
							"description": "vulnerability title",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "vulnerability description",
						},
						"severity": map[string]interface{}{
							"type": "string",
							"description": "severity level",
							"enum": []string{"critical", "high", "medium", "low", "info"},
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "status",
							"enum": []string{"open", "closed", "fixed"},
						},
						"type": map[string]interface{}{
							"type": "string",
							"description": "vulnerability type",
						},
						"target": map[string]interface{}{
							"type": "string",
							"description": "affected target",
						},
						"proof": map[string]interface{}{
							"type": "string",
							"description": "vulnerability proof",
						},
						"impact": map[string]interface{}{
							"type": "string",
							"description": "impact",
						},
						"recommendation": map[string]interface{}{
							"type": "string",
							"description": "remediation recommendation",
						},
					},
				},
				"UpdateVulnerabilityRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type": "string",
							"description": "vulnerability title",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "vulnerability description",
						},
						"severity": map[string]interface{}{
							"type": "string",
							"description": "severity level",
							"enum": []string{"critical", "high", "medium", "low", "info"},
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "status",
							"enum": []string{"open", "closed", "fixed"},
						},
						"type": map[string]interface{}{
							"type": "string",
							"description": "vulnerability type",
						},
						"target": map[string]interface{}{
							"type": "string",
							"description": "affected target",
						},
						"proof": map[string]interface{}{
							"type": "string",
							"description": "vulnerability proof",
						},
						"impact": map[string]interface{}{
							"type": "string",
							"description": "impact",
						},
						"recommendation": map[string]interface{}{
							"type": "string",
							"description": "remediation recommendation",
						},
					},
				},
				"ListVulnerabilitiesResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"vulnerabilities": map[string]interface{}{
							"type": "array",
							"description": "vulnerabilitylist",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/Vulnerability",
							},
						},
						"total": map[string]interface{}{
							"type": "integer",
							"description": "total",
						},
						"page": map[string]interface{}{
							"type": "integer",
							"description": "current page",
						},
						"page_size": map[string]interface{}{
							"type": "integer",
							"description": "items per page",
						},
						"total_pages": map[string]interface{}{
							"type": "integer",
							"description": "total pages",
						},
					},
				},
				"VulnerabilityStats": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"total": map[string]interface{}{
							"type": "integer",
							"description": "vulnerability",
						},
						"by_severity": map[string]interface{}{
							"type": "object",
							"description": "severity levelstatistics",
						},
						"by_status": map[string]interface{}{
							"type": "object",
							"description": "statusstatistics",
						},
					},
				},
				"RoleConfig": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "role name",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "role",
						},
						"enabled": map[string]interface{}{
							"type": "boolean",
							"description": "enable",
						},
						"systemPrompt": map[string]interface{}{
							"type": "string",
							"description": "hint",
						},
						"userPrompt": map[string]interface{}{
							"type": "string",
							"description": "hint",
						},
						"tools": map[string]interface{}{
							"type": "array",
							"description": "toollist",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"skills": map[string]interface{}{
							"type": "array",
							"description": "Skillslist",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
				"Skill": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "Skillname",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "Skill description",
						},
						"path": map[string]interface{}{
							"type": "string",
							"description": "Skillpath",
						},
					},
				},
				"CreateSkillRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"name", "description"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "Skillname",
						},
						"description": map[string]interface{}{
							"type": "string",
							"description": "Skill description",
						},
					},
				},
				"UpdateSkillRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"description": map[string]interface{}{
							"type": "string",
							"description": "Skill description",
						},
					},
				},
				"ToolExecution": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
							"description": "execution ID",
						},
						"toolName": map[string]interface{}{
							"type": "string",
							"description": "tool name",
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "execution status",
							"enum": []string{"success", "failed", "running"},
						},
						"createdAt": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "creation time",
						},
					},
				},
				"MonitorResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"executions": map[string]interface{}{
							"type": "array",
							"description": "execution recordlist",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ToolExecution",
							},
						},
						"stats": map[string]interface{}{
							"type": "object",
							"description": "statistics",
						},
						"timestamp": map[string]interface{}{
							"type": "string",
							"format": "date-time",
							"description": "",
						},
						"total": map[string]interface{}{
							"type": "integer",
							"description": "total",
						},
						"page": map[string]interface{}{
							"type": "integer",
							"description": "current page",
						},
						"page_size": map[string]interface{}{
							"type": "integer",
							"description": "items per page",
						},
						"total_pages": map[string]interface{}{
							"type": "integer",
							"description": "total pages",
						},
					},
				},
				"ConfigResponse": map[string]interface{}{
					"type": "object",
					"description": "configureinformation",
				},
				"UpdateConfigRequest": map[string]interface{}{
					"type": "object",
					"description": "updateconfigurerequest",
				},
				"ExternalMCPConfig": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"enabled": map[string]interface{}{
							"type": "boolean",
							"description": "enable",
						},
						"command": map[string]interface{}{
							"type": "string",
							"description": "command",
						},
						"args": map[string]interface{}{
							"type": "array",
							"description": "parameterlist",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
				"ExternalMCPResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"config": map[string]interface{}{
							"$ref": "#/components/schemas/ExternalMCPConfig",
						},
						"status": map[string]interface{}{
							"type": "string",
							"description": "status",
							"enum": []string{"connected", "disconnected", "error", "disabled"},
						},
						"toolCount": map[string]interface{}{
							"type": "integer",
							"description": "tool count",
						},
						"error": map[string]interface{}{
							"type": "string",
							"description": "error message",
						},
					},
				},
				"AddOrUpdateExternalMCPRequest": map[string]interface{}{
					"type": "object",
					"required": []string{"config"},
					"properties": map[string]interface{}{
						"config": map[string]interface{}{
							"$ref": "#/components/schemas/ExternalMCPConfig",
						},
					},
				},
				"AttackChain": map[string]interface{}{
					"type": "object",
					"description": "attack chaindata",
				},
				"MCPMessage": map[string]interface{}{
					"type": "object",
					"description": "MCPmessage(JSON-RPC 2.0)",
					"required": []string{"jsonrpc"},
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"description": "message ID,/ornull.request,;,",
							"oneOf": []map[string]interface{}{
								{"type": "string"},
								{"type": "number"},
								{"type": "null"},
							},
							"example": "550e8400-e29b-41d4-a716-446655440000",
						},
						"method": map[string]interface{}{
							"type": "string",
							"description": ".supportof:\n- `initialize`: initializeMCPconnection\n- `tools/list`: listtool\n- `tools/call`: calltool\n- `prompts/list`: listhint\n- `prompts/get`: gethint\n- `resources/list`: list\n- `resources/read`: \n- `sampling/request`: request",
							"enum": []string{
								"initialize",
								"tools/list",
								"tools/call",
								"prompts/list",
								"prompts/get",
								"resources/list",
								"resources/read",
								"sampling/request",
							},
							"example": "tools/list",
						},
						"params": map[string]interface{}{
							"description": "parameter(JSON),ofmethodof",
							"type": "object",
						},
						"jsonrpc": map[string]interface{}{
							"type": "string",
							"description": "JSON-RPCversion,is\"2.0\"",
							"enum": []string{"2.0"},
							"example": "2.0",
						},
					},
				},
				"MCPInitializeParams": map[string]interface{}{
					"type": "object",
					"required": []string{"protocolVersion", "capabilities", "clientInfo"},
					"properties": map[string]interface{}{
						"protocolVersion": map[string]interface{}{
							"type": "string",
							"description": "protocolversion",
							"example": "2024-11-05",
						},
						"capabilities": map[string]interface{}{
							"type": "object",
							"description": "",
						},
						"clientInfo": map[string]interface{}{
							"type": "object",
							"required": []string{"name", "version"},
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type": "string",
									"description": "name",
									"example": "MyClient",
								},
								"version": map[string]interface{}{
									"type": "string",
									"description": "version",
									"example": "1.0.0",
								},
							},
						},
					},
				},
				"MCPCallToolParams": map[string]interface{}{
					"type": "object",
					"required": []string{"name", "arguments"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
							"description": "tool name",
							"example": "nmap",
						},
						"arguments": map[string]interface{}{
							"type": "object",
							"description": "toolparameter(),parametertool",
							"example": map[string]interface{}{
								"target": "192.168.1.1",
								"ports": "80,443",
							},
						},
					},
				},
				"MCPResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"description": "message ID(andrequestinofid)",
							"oneOf": []map[string]interface{}{
								{"type": "string"},
								{"type": "number"},
								{"type": "null"},
							},
						},
						"result": map[string]interface{}{
							"description": "execution result(JSON),callof",
							"type": "object",
						},
						"error": map[string]interface{}{
							"type": "object",
							"description": "error message(ifexecution failed)",
							"properties": map[string]interface{}{
								"code": map[string]interface{}{
									"type": "integer",
									"description": "error",
									"example": -32600,
								},
								"message": map[string]interface{}{
									"type": "string",
									"description": "errormessage",
									"example": "Invalid Request",
								},
								"data": map[string]interface{}{
									"description": "errordetails(optional)",
								},
							},
						},
						"jsonrpc": map[string]interface{}{
							"type": "string",
							"description": "JSON-RPCversion",
							"example": "2.0",
						},
					},
				},
			},
		},
		"security": []map[string]interface{}{
			{
				"bearerAuth": []string{},
			},
		},
		"paths": map[string]interface{}{
			"/api/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"authentication"},
					"summary": "login",
					"description": "usepasswordlogingetauthentication token",
					"operationId": "login",
					"security": []map[string]interface{}{},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/LoginRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "loginsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/LoginResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "passworderror",
						},
					},
				},
			},
			"/api/auth/logout": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"authentication"},
					"summary": "",
					"description": "current,makeToken",
					"operationId": "logout",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"message": map[string]interface{}{
												"type": "string",
												"example": "hasSign out",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/auth/change-password": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"authentication"},
					"summary": "modifypassword",
					"description": "modifylogin password,modify",
					"operationId": "changePassword",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ChangePasswordRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "passwordmodifysuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"message": map[string]interface{}{
												"type": "string",
												"example": "passwordhasupdate,usepasswordlogin",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/auth/validate": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"authentication"},
					"summary": "validateToken",
					"description": "validatecurrentToken",
					"operationId": "validateToken",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Token",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
												"description": "Token",
											},
											"expires_at": map[string]interface{}{
												"type": "string",
												"format": "date-time",
												"description": "",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "Tokeninvalidorhas",
						},
					},
				},
			},
			"/api/conversations": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "createconversation",
					"description": "createitemsofSecurity testingconversation.\n****:\n- ✅ createofconversation**savedata**\n- ✅ page**refresh**new conversation\n- ✅ andcreateofconversation****\n**createconversationof**:\n**1():** use `/api/agent-loop` message,**** `conversationId` parameter,createnew conversationmessage.of,completedcreateand.\n**2:** callcreateconversation,usereturnof `conversationId` call `/api/agent-loop` message.used forneedcreateconversation,messageof.\n****:\n```json\n{\n \"title\": \"web application security testing\"\n}\n```",
					"operationId": "createConversation",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/CreateConversationRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "conversationcreatesuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Conversation",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
						"500": map[string]interface{}{
							"description": "internal server error",
						},
					},
				},
				"get": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "listconversation",
					"description": "getconversation list,supportpaginationandsearch",
					"operationId": "listConversations",
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"required": false,
							"description": "return",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 50,
								"minimum": 1,
								"maximum": 100,
							},
						},
						{
							"name": "offset",
							"in": "query",
							"required": false,
							"description": "offset",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 0,
								"minimum": 0,
							},
						},
						{
							"name": "search",
							"in": "query",
							"required": false,
							"description": "search keyword",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"$ref": "#/components/schemas/Conversation",
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
					},
				},
			},
			"/api/conversations/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "conversationdetails",
					"description": "getspecifyconversationofinformation,conversationinformationandmessage list",
					"operationId": "getConversation",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ConversationDetail",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "conversation does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "morenew conversation",
					"description": "morenew conversationtitle",
					"operationId": "updateConversation",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UpdateConversationRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Conversation",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"404": map[string]interface{}{
							"description": "conversation does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "deleteconversation",
					"description": "deletespecifyofconversationdata(message/vulnerability).**restore**.",
					"operationId": "deleteConversation",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"message": map[string]interface{}{
												"type": "string",
												"description": "successfulmessage",
												"example": "deletesuccessful",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "conversation does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
						"500": map[string]interface{}{
							"description": "internal server error",
						},
					},
				},
			},
			"/api/conversations/{id}/results": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "getconversationresult",
					"description": "getspecifyconversationofexecution result,message/vulnerabilityinformationandexecution result",
					"operationId": "getConversationResults",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ConversationResults",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "conversation does not existorresultdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
					},
				},
			},
			"/api/agent-loop": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "send message andgetAIreply()",
					"description": "AIsend message andgetreply(response).**andAIof**,and.\n****:\n- ✅ APIcreate/ofmessage**savedata**\n- ✅ page**refresh**createofconversationandmessage\n- ✅ **of**,\n- ✅ supportroleconfigure,specifyuseitemsrole\n**use**:\n1. **createconversation**:call `POST /api/conversations` createnew conversation,get `conversationId`\n2. **message**:usereturnof `conversationId` callmessage\n**use**:\n**1 - createconversation:**\n```json\nPOST /api/conversations\n{\n \"title\": \"web application security testing\"\n}\n```\n**2 - message:**\n```json\nPOST /api/agent-loop\n{\n \"conversationId\": \"returnofconversation ID\",\n \"message\": \"scan http://example.com ofSQLinjection vulnerability\",\n \"role\": \"Penetration testing\"\n}\n```\n****:\nif `conversationId`,createnew conversationmessage.**createconversation**,moremanageconversation list.\n**response**:returnAIofreply/conversation IDandMCPexecution IDlist.refreshmessage.",
					"operationId": "sendMessage",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"message": map[string]interface{}{
											"type": "string",
											"description": "message to send (required)",
											"example": "scan http://example.com ofSQLinjection vulnerability",
										},
										"conversationId": map[string]interface{}{
											"type": "string",
											"description": "conversation ID(optional).\n- ****:createnew conversationmessage()\n- ****:messagespecifyconversationin(conversation)",
											"example": "550e8400-e29b-41d4-a716-446655440000",
										},
										"role": map[string]interface{}{
											"type": "string",
											"description": "role name (optional),such as:default/Penetration testing/Webapplyscan",
											"example": "default",
										},
									},
									"required": []string{"message"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "messagesuccessful,returnAIreply",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"response": map[string]interface{}{
												"type": "string",
												"description": "AIofreply",
											},
											"conversationId": map[string]interface{}{
												"type": "string",
												"description": "conversation ID",
											},
											"mcpExecutionIds": map[string]interface{}{
												"type": "array",
												"description": "MCPexecution IDlist",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
											"time": map[string]interface{}{
												"type": "string",
												"format": "date-time",
												"description": "response",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
						"500": map[string]interface{}{
							"description": "internal server error",
						},
					},
				},
			},
			"/api/agent-loop/stream": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "send message andgetAIreply()",
					"description": "AIsend message andgetreply(Server-Sent Events).**andAIof**,and.\n****:\n- ✅ APIcreate/ofmessage**savedata**\n- ✅ page**refresh**createofconversationandmessage\n- ✅ **of**,\n- ✅ supportroleconfigure,specifyuseitemsrole\n- ✅ returnresponse,AIreply\n**use**:\n1. **createconversation**:call `POST /api/conversations` createnew conversation,get `conversationId`\n2. **message**:usereturnof `conversationId` callmessage\n**use**:\n**1 - createconversation:**\n```json\nPOST /api/conversations\n{\n \"title\": \"web application security testing\"\n}\n```\n**2 - message():**\n```json\nPOST /api/agent-loop/stream\n{\n \"conversationId\": \"returnofconversation ID\",\n \"message\": \"scan http://example.com ofSQLinjection vulnerability\",\n \"role\": \"Penetration testing\"\n}\n```\n**responseFormat**:Server-Sent Events (SSE),:\n- `message`: message\n- `response`: AIreply\n- `progress`: update\n- `done`: completed\n- `error`: error\n- `cancelled`: hascancel",
					"operationId": "sendMessageStream",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"message": map[string]interface{}{
											"type": "string",
											"description": "message to send (required)",
											"example": "scan http://example.com ofSQLinjection vulnerability",
										},
										"conversationId": map[string]interface{}{
											"type": "string",
											"description": "conversation ID(optional).\n- ****:createnew conversationmessage()\n- ****:messagespecifyconversationin(conversation)",
											"example": "550e8400-e29b-41d4-a716-446655440000",
										},
										"role": map[string]interface{}{
											"type": "string",
											"description": "role name (optional),such as:default/Penetration testing/Webapplyscan",
											"example": "default",
										},
									},
									"required": []string{"message"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "response(Server-Sent Events)",
							"content": map[string]interface{}{
								"text/event-stream": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "string",
										"description": "SSEdata",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
						"500": map[string]interface{}{
							"description": "internal server error",
						},
					},
				},
			},
			"/api/eino-agent": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "send message andget AI reply(Eino ADK single agent,)",
					"description": "and `POST /api/agent-loop` request, **CloudWeGo Eino** `adk.NewChatModelAgent` + `adk.NewRunner.Run` execute(single agent MCP tool).**** `multi_agent.enabled`;`multi_agent.eino_skills` / `eino_middleware` andmulti-agent.support `webshellConnectionId`.",
					"operationId": "sendMessageEinoSingleAgent",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"message": map[string]interface{}{"type": "string"},
										"conversationId": map[string]interface{}{"type": "string"},
										"role": map[string]interface{}{"type": "string"},
										"webshellConnectionId": map[string]interface{}{"type": "string"},
									},
									"required": []string{"message"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{"description": "successful,responseFormat /api/agent-loop"},
						"400": map[string]interface{}{"description": "parametererror"},
						"401": map[string]interface{}{"description": "unauthorized"},
						"500": map[string]interface{}{"description": "execution failed"},
					},
				},
			},
			"/api/eino-agent/stream": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "send message andget AI reply(Eino ADK single agent,SSE)",
					"description": "and `POST /api/agent-loop/stream` ; Eino **single agent** ADK execute.andmulti-agent( `tool_call` / `response_delta` ).**** `multi_agent.enabled`.",
					"operationId": "sendMessageEinoSingleAgentStream",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"message": map[string]interface{}{"type": "string"},
										"conversationId": map[string]interface{}{"type": "string"},
										"role": map[string]interface{}{"type": "string"},
										"webshellConnectionId": map[string]interface{}{"type": "string"},
									},
									"required": []string{"message"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "text/event-stream(SSE)",
							"content": map[string]interface{}{
								"text/event-stream": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "string",
										"description": "SSE ",
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "unauthorized"},
					},
				},
			},
			"/api/multi-agent": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "send message andget AI reply(Eino multi-agent,)",
					"description": "and `POST /api/agent-loop` request, **CloudWeGo Eino** multi-agentexecute.request `orchestration`(`deep` | `plan_execute` | `supervisor`)specify,is `deep`.****:`multi_agent.enabled: true`;enablereturn 404 JSON.support `webshellConnectionId`.",
					"operationId": "sendMessageMultiAgent",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"message": map[string]interface{}{
											"type": "string",
											"description": "message to send (required)",
										},
										"conversationId": map[string]interface{}{
											"type": "string",
											"description": "conversation ID(optional,)",
										},
										"role": map[string]interface{}{
											"type": "string",
											"description": "role name (optional)",
										},
										"webshellConnectionId": map[string]interface{}{
											"type": "string",
											"description": "WebShell connection ID(optional,and agent-loop is)",
										},
										"orchestration": map[string]interface{}{
											"type": "string",
											"description": "Eino :deep | plan_execute | supervisor; deep",
											"enum": []string{"deep", "plan_execute", "supervisor"},
										},
									},
									"required": []string{"message"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful,responseFormat /api/agent-loop",
						},
						"400": map[string]interface{}{"description": "parametererror"},
						"401": map[string]interface{}{"description": "unauthorized"},
						"404": map[string]interface{}{"description": "multi-agentenableorconversation does not exist"},
						"500": map[string]interface{}{"description": "execution failed"},
					},
				},
			},
			"/api/multi-agent/stream": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "send message andget AI reply(Eino multi-agent,SSE)",
					"description": "and `POST /api/agent-loop/stream` ; Eino multi-agentexecute.`orchestration` specify deep / plan_execute / supervisor, deep.****:`multi_agent.enabled: true`;enable SSE is `type: error` `done`.support `webshellConnectionId`.",
					"operationId": "sendMessageMultiAgentStream",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"message": map[string]interface{}{"type": "string"},
										"conversationId": map[string]interface{}{"type": "string"},
										"role": map[string]interface{}{"type": "string"},
										"webshellConnectionId": map[string]interface{}{"type": "string"},
										"orchestration": map[string]interface{}{
											"type": "string",
											"description": "deep | plan_execute | supervisor; deep",
											"enum": []string{"deep", "plan_execute", "supervisor"},
										},
									},
									"required": []string{"message"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "text/event-stream(SSE)",
							"content": map[string]interface{}{
								"text/event-stream": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "string",
										"description": "SSE ",
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "unauthorized"},
					},
				},
			},
			"/api/agent-loop/cancel": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "canceltask",
					"description": "cancelexecuteofAgent Looptask",
					"operationId": "cancelAgentLoop",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/CancelAgentLoopRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "cancelrequesthas",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
												"example": "cancelling",
											},
											"conversationId": map[string]interface{}{
												"type": "string",
												"description": "conversation ID",
											},
											"message": map[string]interface{}{
												"type": "string",
												"example": "hascancelrequest,taskcurrentcompletedstop.",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "not foundexecuteoftask",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/agent-loop/tasks": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "listinoftask",
					"description": "getofAgent Looptask",
					"operationId": "listAgentTasks",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"tasks": map[string]interface{}{
												"type": "array",
												"description": "task list",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/AgentTask",
												},
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/agent-loop/tasks/completed": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"conversation interaction"},
					"summary": "listcompletedoftask",
					"description": "getcompletedofAgent Looptask",
					"operationId": "listCompletedTasks",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"tasks": map[string]interface{}{
												"type": "array",
												"description": "completedtask list",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/AgentTask",
												},
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/batch-tasks": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "createbatch task queue",
					"description": "createitemsbatch task queue,itemstask",
					"operationId": "createBatchQueue",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/BatchTaskRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "createsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"queueId": map[string]interface{}{
												"type": "string",
												"description": "queueID",
											},
											"queue": map[string]interface{}{
												"$ref": "#/components/schemas/BatchQueue",
											},
											"started": map[string]interface{}{
												"type": "boolean",
												"description": "hasstartexecute",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"get": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "listbatch task queue",
					"description": "getbatch task queue",
					"operationId": "listBatchQueues",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"queues": map[string]interface{}{
												"type": "array",
												"description": "queuelist",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/BatchQueue",
												},
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/batch-tasks/{queueId}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "getbatch task queue",
					"description": "getspecifybatch task queueofinformation",
					"operationId": "getBatchQueue",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/BatchQueue",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "queue does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "deletebatch task queue",
					"description": "deletespecifyofbatch task queue",
					"operationId": "deleteBatchQueue",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "queue does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/batch-tasks/{queueId}/start": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "startbatch task queue",
					"description": "startexecutebatch task queueinoftask",
					"operationId": "startBatchQueue",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "startsuccessful",
						},
						"404": map[string]interface{}{
							"description": "queue does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/batch-tasks/{queueId}/pause": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "batch task queue",
					"description": "executeofbatch task queue",
					"operationId": "pauseBatchQueue",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful",
						},
						"404": map[string]interface{}{
							"description": "queue does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/batch-tasks/{queueId}/tasks": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "taskqueue",
					"description": "batch task queuetask.taskqueue,queueexecute.each/peritemstaskcreateitemsofconversation,supportofstatustrace.\n**taskFormat**:\ntaskitems,executeofSecurity testingtask.such as:\n- \"scan http://example.com ofSQLinjection vulnerability\"\n- \" 192.168.1.1 portscan\"\n- \" https://target.com ofXSSvulnerability\"\n**use**:\n```json\n{\n \"task\": \"scan http://example.com ofSQLinjection vulnerability\"\n}\n```",
					"operationId": "addBatchTask",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"task"},
									"properties": map[string]interface{}{
										"task": map[string]interface{}{
											"type": "string",
											"description": "task,executeofSecurity testingtask(need)",
											"example": "scan http://example.com ofSQLinjection vulnerability",
										},
									},
								},
								"examples": map[string]interface{}{
									"sqlInjection": map[string]interface{}{
										"summary": "SQLinjectscan",
										"description": "scantargetofSQLinjection vulnerability",
										"value": map[string]interface{}{
											"task": "scan http://example.com ofSQLinjection vulnerability",
										},
									},
									"portScan": map[string]interface{}{
										"summary": "portscan",
										"description": "targetIPportscan",
										"value": map[string]interface{}{
											"task": " 192.168.1.1 portscan",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"taskId": map[string]interface{}{
												"type": "string",
												"description": "oftask ID",
											},
											"message": map[string]interface{}{
												"type": "string",
												"description": "successfulmessage",
												"example": "taskhasqueue",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error(such astaskis)",
						},
						"404": map[string]interface{}{
							"description": "queue does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/batch-tasks/{queueId}/tasks/{taskId}": map[string]interface{}{
				"put": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "updatebatch task",
					"description": "updatebatch task queueinofspecifytask",
					"operationId": "updateBatchTask",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name": "taskId",
							"in": "path",
							"required": true,
							"description": "task ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"task": map[string]interface{}{
											"type": "string",
											"description": "task",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"404": map[string]interface{}{
							"description": "taskdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"batch task"},
					"summary": "deletebatch task",
					"description": "frombatch task queueindeletespecifytask",
					"operationId": "deleteBatchTask",
					"parameters": []map[string]interface{}{
						{
							"name": "queueId",
							"in": "path",
							"required": true,
							"description": "queueID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name": "taskId",
							"in": "path",
							"required": true,
							"description": "task ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "taskdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "creategroup",
					"description": "createitemsofconversation group",
					"operationId": "createGroup",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/CreateGroupRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "createsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Group",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter errororgroup namealready exists",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"get": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "listgroup",
					"description": "getconversation group",
					"operationId": "listGroups",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"$ref": "#/components/schemas/Group",
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "getgroup",
					"description": "getspecifygroupofinformation",
					"operationId": "getGroup",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Group",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "group does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "updategroup",
					"description": "updategroupinformation",
					"operationId": "updateGroup",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UpdateGroupRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Group",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter errororgroup namealready exists",
						},
						"404": map[string]interface{}{
							"description": "group does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "deletegroup",
					"description": "deletespecifygroup",
					"operationId": "deleteGroup",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "group does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups/{id}/conversations": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "getgroupinofconversation",
					"description": "getspecifygroupinofconversation",
					"operationId": "getGroupConversations",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"$ref": "#/components/schemas/Conversation",
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "group does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups/conversations": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "conversationgroup",
					"description": "conversationspecifygroup",
					"operationId": "addConversationToGroup",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/AddConversationToGroupRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"404": map[string]interface{}{
							"description": "conversationorgroup does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups/{id}/conversations/{conversationId}": map[string]interface{}{
				"delete": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "fromgroupconversation",
					"description": "fromspecifygroupinconversation",
					"operationId": "removeConversationFromGroup",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name": "conversationId",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful",
						},
						"404": map[string]interface{}{
							"description": "conversationorgroup does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/vulnerabilities": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"vulnerabilitymanage"},
					"summary": "listvulnerability",
					"description": "getvulnerabilitylist,supportpaginationandfilter",
					"operationId": "listVulnerabilities",
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"required": false,
							"description": "items per page",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 20,
								"minimum": 1,
								"maximum": 100,
							},
						},
						{
							"name": "offset",
							"in": "query",
							"required": false,
							"description": "offset",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 0,
								"minimum": 0,
							},
						},
						{
							"name": "page",
							"in": "query",
							"required": false,
							"description": "page(andoffset)",
							"schema": map[string]interface{}{
								"type": "integer",
								"minimum": 1,
							},
						},
						{
							"name": "id",
							"in": "query",
							"required": false,
							"description": "vulnerability ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name": "conversation_id",
							"in": "query",
							"required": false,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name": "severity",
							"in": "query",
							"required": false,
							"description": "severity level",
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"critical", "high", "medium", "low", "info"},
							},
						},
						{
							"name": "status",
							"in": "query",
							"required": false,
							"description": "status",
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"open", "closed", "fixed"},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ListVulnerabilitiesResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"post": map[string]interface{}{
					"tags": []string{"vulnerabilitymanage"},
					"summary": "createvulnerability",
					"description": "createitemsofvulnerabilityrecord",
					"operationId": "createVulnerability",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/CreateVulnerabilityRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "createsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Vulnerability",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/vulnerabilities/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"vulnerabilitymanage"},
					"summary": "getvulnerabilitystatistics",
					"description": "getvulnerabilitystatistics",
					"operationId": "getVulnerabilityStats",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/VulnerabilityStats",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/vulnerabilities/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"vulnerabilitymanage"},
					"summary": "getvulnerability",
					"description": "getspecifyvulnerabilityofinformation",
					"operationId": "getVulnerability",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "vulnerability ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Vulnerability",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "vulnerabilitydoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"vulnerabilitymanage"},
					"summary": "updatevulnerability",
					"description": "updatevulnerabilityinformation",
					"operationId": "updateVulnerability",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "vulnerability ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UpdateVulnerabilityRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Vulnerability",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"404": map[string]interface{}{
							"description": "vulnerabilitydoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"vulnerabilitymanage"},
					"summary": "deletevulnerability",
					"description": "deletespecifyvulnerability",
					"operationId": "deleteVulnerability",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "vulnerability ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "vulnerabilitydoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/roles": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"rolemanage"},
					"summary": "listrole",
					"description": "getSecurity testingrole",
					"operationId": "getRoles",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"roles": map[string]interface{}{
												"type": "array",
												"description": "role list",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/RoleConfig",
												},
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"post": map[string]interface{}{
					"tags": []string{"rolemanage"},
					"summary": "createrole",
					"description": "createitemsofSecurity testingrole",
					"operationId": "createRole",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/RoleConfig",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "createsuccessful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/roles/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"rolemanage"},
					"summary": "getrole",
					"description": "getspecifyroleofinformation",
					"operationId": "getRole",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "role name",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"role": map[string]interface{}{
												"$ref": "#/components/schemas/RoleConfig",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "roledoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"rolemanage"},
					"summary": "updaterole",
					"description": "updatespecifyroleofconfigure",
					"operationId": "updateRole",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "role name",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/RoleConfig",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"404": map[string]interface{}{
							"description": "roledoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"rolemanage"},
					"summary": "deleterole",
					"description": "deletespecifyrole",
					"operationId": "deleteRole",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "role name",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "roledoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/roles/skills/list": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"rolemanage"},
					"summary": "getSkillslist",
					"description": "getofSkillslist,used forroleconfigure",
					"operationId": "getSkills",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"skills": map[string]interface{}{
												"type": "array",
												"description": "Skillslist",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/skills": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "listSkills",
					"description": "getSkillslist,supportpaginationandsearch",
					"operationId": "getSkills",
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"required": false,
							"description": "items per page",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 20,
							},
						},
						{
							"name": "offset",
							"in": "query",
							"required": false,
							"description": "offset",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 0,
							},
						},
						{
							"name": "search",
							"in": "query",
							"required": false,
							"description": "search keyword",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"skills": map[string]interface{}{
												"type": "array",
												"description": "Skillslist",
												"items": map[string]interface{}{
													"$ref": "#/components/schemas/Skill",
												},
											},
											"total": map[string]interface{}{
												"type": "integer",
												"description": "total",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"post": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "createSkill",
					"description": "createitemsofSkill",
					"operationId": "createSkill",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/CreateSkillRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "createsuccessful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/skills/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "getSkillstatistics",
					"description": "getSkillcallstatistics",
					"operationId": "getSkillStats",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"description": "statistics",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "clearSkillstatistics",
					"description": "clearSkillofcallstatistics",
					"operationId": "clearSkillStats",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "clearsuccessful",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/skills/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "getSkill",
					"description": "getspecifySkillofinformation",
					"operationId": "getSkill",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "Skillname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Skill",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Skilldoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "updateSkill",
					"description": "updatespecifySkillofinformation",
					"operationId": "updateSkill",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "Skillname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UpdateSkillRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"404": map[string]interface{}{
							"description": "Skilldoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "deleteSkill",
					"description": "deletespecifySkill",
					"operationId": "deleteSkill",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "Skillname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "Skilldoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/skills/{name}/bound-roles": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "getbindrole",
					"description": "getusespecifySkillofrole",
					"operationId": "getSkillBoundRoles",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "Skillname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"roles": map[string]interface{}{
												"type": "array",
												"description": "role list",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Skilldoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/skills/{name}/stats": map[string]interface{}{
				"delete": map[string]interface{}{
					"tags": []string{"Skillsmanage"},
					"summary": "clearSkillstatistics",
					"description": "clearspecifySkillofcallstatistics",
					"operationId": "clearSkillStatsByName",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "Skillname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "clearsuccessful",
						},
						"404": map[string]interface{}{
							"description": "Skilldoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/monitor": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"monitor"},
					"summary": "getmonitorinformation",
					"description": "gettoolexecutemonitorinformation,supportpaginationandfilter",
					"operationId": "monitor",
					"parameters": []map[string]interface{}{
						{
							"name": "page",
							"in": "query",
							"required": false,
							"description": "page",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 1,
								"minimum": 1,
							},
						},
						{
							"name": "page_size",
							"in": "query",
							"required": false,
							"description": "items per page",
							"schema": map[string]interface{}{
								"type": "integer",
								"default": 20,
								"minimum": 1,
								"maximum": 100,
							},
						},
						{
							"name": "status",
							"in": "query",
							"required": false,
							"description": "statusfilter",
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"success", "failed", "running"},
							},
						},
						{
							"name": "tool",
							"in": "query",
							"required": false,
							"description": "tool namefilter(support)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/MonitorResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/monitor/execution/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"monitor"},
					"summary": "getexecution record",
					"description": "getspecifyexecution recordofinformation",
					"operationId": "getExecution",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "execution ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ToolExecution",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "execution recorddoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"monitor"},
					"summary": "deleteexecution record",
					"description": "deletespecifyofexecution record",
					"operationId": "deleteExecution",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "execution ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "execution recorddoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/monitor/executions": map[string]interface{}{
				"delete": map[string]interface{}{
					"tags": []string{"monitor"},
					"summary": "batchdeleteexecution record",
					"description": "batchdeleteexecution record",
					"operationId": "deleteExecutions",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/monitor/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"monitor"},
					"summary": "getstatistics",
					"description": "gettoolexecutestatistics",
					"operationId": "getStats",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"description": "statistics",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/config": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"configuremanage"},
					"summary": "getconfigure",
					"description": "getconfigureinformation",
					"operationId": "getConfig",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ConfigResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"configuremanage"},
					"summary": "updateconfigure",
					"description": "updateconfigure",
					"operationId": "updateConfig",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/UpdateConfigRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/config/tools": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"configuremanage"},
					"summary": "gettoolconfigure",
					"description": "gettoolofconfigureinformation",
					"operationId": "getTools",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"description": "toolconfigurelist",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/config/apply": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"configuremanage"},
					"summary": "applyconfigure",
					"description": "applyconfiguremore",
					"operationId": "applyConfig",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "applysuccessful",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/external-mcp": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "listexternalMCP",
					"description": "getexternalMCPconfigureandstatus",
					"operationId": "getExternalMCPs",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"servers": map[string]interface{}{
												"type": "object",
												"description": "MCPserverconfigure",
												"additionalProperties": map[string]interface{}{
													"$ref": "#/components/schemas/ExternalMCPResponse",
												},
											},
											"stats": map[string]interface{}{
												"type": "object",
												"description": "statistics",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/external-mcp/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "getexternalMCPstatistics",
					"description": "getexternalMCPstatistics",
					"operationId": "getExternalMCPStats",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"description": "statistics",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/external-mcp/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "getexternalMCP",
					"description": "getspecifyexternalMCPofconfigureandstatus",
					"operationId": "getExternalMCP",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "MCPname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ExternalMCPResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "MCPdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "orupdateexternalMCP",
					"description": "ofexternalMCPconfigureorupdateconfigure.\n****:\nsupport:\n**1. stdio(inputoutput)**:\n```json\n{\n \"config\": {\n \"enabled\": true,\n \"command\": \"node\",\n \"args\": [\"/path/to/mcp-server.js\"],\n \"env\": {}\n }\n}\n```\n**2. sse(Server-Sent Events)**:\n```json\n{\n \"config\": {\n \"enabled\": true,\n \"transport\": \"sse\",\n \"url\": \"http://127.0.0.1:8082/sse\",\n \"timeout\": 30\n }\n}\n```\n**configureparameter**:\n- `enabled`: enable(boolean,need)\n- `command`: command(stdiomodeneed,such as:\"node\", \"python\")\n- `args`: commandparameter(stdiomodeneed)\n- `env`: (object,optional)\n- `transport`: (\"stdio\" or \"sse\",ssemodeneed)\n- `url`: SSEURL(ssemodeneed)\n- `timeout`: timeout( seconds,optional,default30)\n- `description`: (optional)",
					"operationId": "addOrUpdateExternalMCP",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "MCPname()",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/AddOrUpdateExternalMCPRequest",
								},
								"examples": map[string]interface{}{
									"stdio": map[string]interface{}{
										"summary": "stdiomodeconfigure",
										"description": "useinputoutputconnectionexternalMCPserver",
										"value": map[string]interface{}{
											"config": map[string]interface{}{
												"enabled": true,
												"command": "node",
												"args": []string{"/path/to/mcp-server.js"},
												"env": map[string]interface{}{},
												"timeout": 30,
												"description": "Node.js MCPserver",
											},
										},
									},
									"sse": map[string]interface{}{
										"summary": "SSEmodeconfigure",
										"description": "useServer-Sent EventsconnectionexternalMCPserver",
										"value": map[string]interface{}{
											"config": map[string]interface{}{
												"enabled": true,
												"transport": "sse",
												"url": "http://127.0.0.1:8082/sse",
												"timeout": 30,
												"description": "SSE MCPserver",
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"message": map[string]interface{}{
												"type": "string",
												"example": "externalMCPconfigurehassave",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error(such asconfigureFormat/needfield)",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Error",
									},
									"example": map[string]interface{}{
										"error": "stdiomodeneedcommandandargsparameter",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "deleteexternalMCP",
					"description": "deletespecifyofexternalMCPconfigure",
					"operationId": "deleteExternalMCP",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "MCPname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "MCPdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/external-mcp/{name}/start": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "start externalMCP",
					"description": "startspecifyofexternalMCPserver",
					"operationId": "startExternalMCP",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "MCPname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "startsuccessful",
						},
						"404": map[string]interface{}{
							"description": "MCPdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/external-mcp/{name}/stop": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"externalMCPmanage"},
					"summary": "stopexternalMCP",
					"description": "stopspecifyofexternalMCPserver",
					"operationId": "stopExternalMCP",
					"parameters": []map[string]interface{}{
						{
							"name": "name",
							"in": "path",
							"required": true,
							"description": "MCPname",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "stopsuccessful",
						},
						"404": map[string]interface{}{
							"description": "MCPdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/attack-chain/{conversationId}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"attack chain"},
					"summary": "getattack chain",
					"description": "getspecifyconversationofattack chaindata",
					"operationId": "getAttackChain",
					"parameters": []map[string]interface{}{
						{
							"name": "conversationId",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AttackChain",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "conversation does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/attack-chain/{conversationId}/regenerate": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"attack chain"},
					"summary": "generateattack chain",
					"description": "generatespecifyconversationofattack chaindata",
					"operationId": "regenerateAttackChain",
					"parameters": []map[string]interface{}{
						{
							"name": "conversationId",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "generatesuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AttackChain",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "conversation does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/conversations/{id}/pinned": map[string]interface{}{
				"put": map[string]interface{}{
					"tags": []string{"conversationmanage"},
					"summary": "conversationpin",
					"description": "orcancelconversationofpinstatus",
					"operationId": "updateConversationPinned",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"pinned"},
									"properties": map[string]interface{}{
										"pinned": map[string]interface{}{
											"type": "boolean",
											"description": "whether pinned",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"404": map[string]interface{}{
							"description": "conversation does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups/{id}/pinned": map[string]interface{}{
				"put": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "grouppin",
					"description": "orcancelgroupofpinstatus",
					"operationId": "updateGroupPinned",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"pinned"},
									"properties": map[string]interface{}{
										"pinned": map[string]interface{}{
											"type": "boolean",
											"description": "whether pinned",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"404": map[string]interface{}{
							"description": "group does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/groups/{id}/conversations/{conversationId}/pinned": map[string]interface{}{
				"put": map[string]interface{}{
					"tags": []string{"conversation group"},
					"summary": "groupinconversationofpin",
					"description": "orcancelgroupinconversationofpinstatus",
					"operationId": "updateConversationPinnedInGroup",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "group ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name": "conversationId",
							"in": "path",
							"required": true,
							"description": "conversation ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"pinned"},
									"properties": map[string]interface{}{
										"pinned": map[string]interface{}{
											"type": "boolean",
											"description": "whether pinned",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"404": map[string]interface{}{
							"description": "conversationorgroup does not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/categories": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "getcategory",
					"description": "getknowledge baseofcategory",
					"operationId": "getKnowledgeCategories",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"categories": map[string]interface{}{
												"type": "array",
												"description": "categorylist",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
											"enabled": map[string]interface{}{
												"type": "boolean",
												"description": "whether the knowledge base is enabled",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/items": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "listknowledge item",
					"description": "getknowledge baseinofknowledge item",
					"operationId": "getKnowledgeItems",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"items": map[string]interface{}{
												"type": "array",
												"description": "knowledge itemlist",
											},
											"enabled": map[string]interface{}{
												"type": "boolean",
												"description": "whether the knowledge base is enabled",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"post": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "createknowledge item",
					"description": "createofknowledge item",
					"operationId": "createKnowledgeItem",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"description": "knowledge itemdata",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "createsuccessful",
						},
						"400": map[string]interface{}{
							"description": "request parameter error",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/items/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "getknowledge item",
					"description": "getspecifyknowledge itemofinformation",
					"operationId": "getKnowledgeItem",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "knowledge itemID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
						},
						"404": map[string]interface{}{
							"description": "knowledge itemdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"put": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "updateknowledge item",
					"description": "updatespecifyknowledge item",
					"operationId": "updateKnowledgeItem",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "knowledge itemID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"description": "knowledge itemdata",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "updatesuccessful",
						},
						"404": map[string]interface{}{
							"description": "knowledge itemdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
				"delete": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "deleteknowledge item",
					"description": "deletespecifyknowledge item",
					"operationId": "deleteKnowledgeItem",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "knowledge itemID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "knowledge itemdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/index-status": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "getstatus",
					"description": "getknowledge baseofbuildstatus",
					"operationId": "getIndexStatus",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"enabled": map[string]interface{}{
												"type": "boolean",
												"description": "whether the knowledge base is enabled",
											},
											"total_items": map[string]interface{}{
												"type": "integer",
												"description": "knowledge item",
											},
											"indexed_items": map[string]interface{}{
												"type": "integer",
												"description": "hasknowledge item",
											},
											"progress_percent": map[string]interface{}{
												"type": "number",
												"description": "",
											},
											"is_complete": map[string]interface{}{
												"type": "boolean",
												"description": "completed",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/index": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "",
					"description": "buildknowledge base",
					"operationId": "rebuildIndex",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "taskhasstart",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/scan": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "scanknowledge base",
					"description": "scanknowledge basedirectory,importofKnowledgefile",
					"operationId": "scanKnowledgeBase",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "scantaskhasstart",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/search": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "searchknowledge base",
					"description": "knowledge baseinsearch.retrieval,queryandKnowledgeof()returnresult.\n**search**:\n- search: + ,configureand TopK\n- risk typedata(such as:SQLinject/XSS/fileupload)\n- call `/api/knowledge/categories` getofrisk typelist\n**use**:\n```json\n{\n \"query\": \"SQLinjection vulnerabilityof\",\n \"riskType\": \"SQLinject\",\n \"topK\": 5,\n \"threshold\": 0.7\n}\n```",
					"operationId": "searchKnowledge",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"required": []string{"query"},
									"properties": map[string]interface{}{
										"query": map[string]interface{}{
											"type": "string",
											"description": "searchquery,ofKnowledge(need)",
											"example": "SQLinjection vulnerabilityof",
										},
										"riskType": map[string]interface{}{
											"type": "string",
											"description": "optional:specifyrisk type(such as:SQLinject/XSS/fileupload).call `/api/knowledge/categories` getofrisk typelist,useofrisk typesearch,retrieval.ifspecifysearch.",
											"example": "SQLinject",
										},
										"topK": map[string]interface{}{
											"type": "integer",
											"description": "optional:returnTop-Kresult,default5",
											"default": 5,
											"minimum": 1,
											"maximum": 50,
											"example": 5,
										},
										"threshold": map[string]interface{}{
											"type": "number",
											"format": "float",
											"description": "optional:(0-1),default0.7.ofresultreturn",
											"default": 0.7,
											"minimum": 0,
											"maximum": 1,
											"example": 0.7,
										},
									},
								},
								"examples": map[string]interface{}{
									"basic": map[string]interface{}{
										"summary": "search",
										"description": "ofsearch,query",
										"value": map[string]interface{}{
											"query": "SQLinjection vulnerabilityof",
										},
									},
									"withRiskType": map[string]interface{}{
										"summary": "risk typesearch",
										"description": "specifyrisk typesearch",
										"value": map[string]interface{}{
											"query": "SQLinjection vulnerabilityof",
											"riskType": "SQLinject",
											"topK": 5,
											"threshold": 0.7,
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "searchsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"results": map[string]interface{}{
												"type": "array",
												"description": "searchresultlist,each/peritemsresult:item(knowledge iteminformation)/chunks(ofKnowledge)/score()",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"item": map[string]interface{}{
															"type": "object",
															"description": "knowledge iteminformation",
														},
														"chunks": map[string]interface{}{
															"type": "array",
															"description": "ofKnowledgelist",
														},
														"score": map[string]interface{}{
															"type": "number",
															"description": "(0-1)",
														},
													},
												},
											},
											"enabled": map[string]interface{}{
												"type": "boolean",
												"description": "whether the knowledge base is enabled",
											},
										},
									},
									"example": map[string]interface{}{
										"results": []map[string]interface{}{
											{
												"item": map[string]interface{}{
													"id": "item-1",
													"title": "SQLinjection vulnerability",
													"category": "SQLinject",
												},
												"chunks": []map[string]interface{}{
													{
														"text": "SQLinjection vulnerabilityof...",
													},
												},
												"score": 0.85,
											},
										},
										"enabled": true,
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "request parameter error(such asqueryis)",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Error",
									},
									"example": map[string]interface{}{
										"error": "querycannot be empty",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
						"500": map[string]interface{}{
							"description": "internal server error(such asknowledge baseenableorretrievalfailed)",
						},
					},
				},
			},
			"/api/knowledge/retrieval-logs": map[string]interface{}{
				"get": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "getretrieval log",
					"description": "getknowledge baseretrieval log",
					"operationId": "getRetrievalLogs",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "getsuccessful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"logs": map[string]interface{}{
												"type": "array",
												"description": "retrieval loglist",
											},
											"enabled": map[string]interface{}{
												"type": "boolean",
												"description": "whether the knowledge base is enabled",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/knowledge/retrieval-logs/{id}": map[string]interface{}{
				"delete": map[string]interface{}{
					"tags": []string{"knowledge base"},
					"summary": "deleteretrieval log",
					"description": "deletespecifyofretrieval log",
					"operationId": "deleteRetrievalLog",
					"parameters": []map[string]interface{}{
						{
							"name": "id",
							"in": "path",
							"required": true,
							"description": "logID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "deletesuccessful",
						},
						"404": map[string]interface{}{
							"description": "logdoes not exist",
						},
						"401": map[string]interface{}{
							"description": "unauthorized",
						},
					},
				},
			},
			"/api/mcp": map[string]interface{}{
				"post": map[string]interface{}{
					"tags": []string{"MCP"},
					"summary": "MCP",
					"description": "MCP (Model Context Protocol) ,used forprocessMCPprotocolrequest.\n**protocol**:\n JSON-RPC 2.0 ,support:\n**1. initialize** - initializeMCPconnection\n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"init-1\",\n \"method\": \"initialize\",\n \"params\": {\n \"protocolVersion\": \"2024-11-05\",\n \"capabilities\": {},\n \"clientInfo\": {\n \"name\": \"MyClient\",\n \"version\": \"1.0.0\"\n }\n }\n}\n```\n**2. tools/list** - listtool\n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"list-1\",\n \"method\": \"tools/list\",\n \"params\": {}\n}\n```\n**3. tools/call** - calltool\n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"call-1\",\n \"method\": \"tools/call\",\n \"params\": {\n \"name\": \"nmap\",\n \"arguments\": {\n \"target\": \"192.168.1.1\",\n \"ports\": \"80,443\"\n }\n }\n}\n```\n**4. prompts/list** - listhint\n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"prompts-list-1\",\n \"method\": \"prompts/list\",\n \"params\": {}\n}\n```\n**5. prompts/get** - gethint\n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"prompt-get-1\",\n \"method\": \"prompts/get\",\n \"params\": {\n \"name\": \"prompt-name\",\n \"arguments\": {}\n }\n}\n```\n**6. resources/list** - list\n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"resources-list-1\",\n \"method\": \"resources/list\",\n \"params\": {}\n}\n```\n**7. resources/read** - \n```json\n{\n \"jsonrpc\": \"2.0\",\n \"id\": \"resource-read-1\",\n \"method\": \"resources/read\",\n \"params\": {\n \"uri\": \"resource://example\"\n }\n}\n```\n**error**:\n- `-32700`: Parse error - JSONparseerror\n- `-32600`: Invalid Request - invalidrequest\n- `-32601`: Method not found - does not exist\n- `-32602`: Invalid params - parameterinvalid\n- `-32603`: Internal error - error",
					"operationId": "mcpEndpoint",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/MCPMessage",
								},
								"examples": map[string]interface{}{
									"listTools": map[string]interface{}{
										"summary": "listtool",
										"description": "getinofMCPtoollist",
										"value": map[string]interface{}{
											"jsonrpc": "2.0",
											"id": "list-tools-1",
											"method": "tools/list",
											"params": map[string]interface{}{},
										},
									},
									"callTool": map[string]interface{}{
										"summary": "calltool",
										"description": "callspecifyofMCPtool",
										"value": map[string]interface{}{
											"jsonrpc": "2.0",
											"id": "call-tool-1",
											"method": "tools/call",
											"params": map[string]interface{}{
												"name": "nmap",
												"arguments": map[string]interface{}{
													"target": "192.168.1.1",
													"ports": "80,443",
												},
											},
										},
									},
									"initialize": map[string]interface{}{
										"summary": "initializeconnection",
										"description": "initializeMCPconnection,getserver",
										"value": map[string]interface{}{
											"jsonrpc": "2.0",
											"id": "init-1",
											"method": "initialize",
											"params": map[string]interface{}{
												"protocolVersion": "2024-11-05",
												"capabilities": map[string]interface{}{},
												"clientInfo": map[string]interface{}{
													"name": "MyClient",
													"version": "1.0.0",
												},
											},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "MCPresponse(JSON-RPC 2.0Format)",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/MCPResponse",
									},
									"examples": map[string]interface{}{
										"success": map[string]interface{}{
											"summary": "successfulresponse",
											"description": "toolcallsuccessfulofresponse",
											"value": map[string]interface{}{
												"jsonrpc": "2.0",
												"id": "call-tool-1",
												"result": map[string]interface{}{
													"content": []map[string]interface{}{
														{
															"type": "text",
															"text": "toolexecution result...",
														},
													},
													"isError": false,
												},
											},
										},
										"error": map[string]interface{}{
											"summary": "errorresponse",
											"description": "toolcallfailedofresponse",
											"value": map[string]interface{}{
												"jsonrpc": "2.0",
												"id": "call-tool-1",
												"error": map[string]interface{}{
													"code": -32601,
													"message": "Tool not found",
													"data": "tool 'unknown-tool' does not exist",
												},
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "requestFormaterror(JSONparsefailed)",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/MCPResponse",
									},
									"example": map[string]interface{}{
										"id": nil,
										"error": map[string]interface{}{
											"code": -32700,
											"message": "Parse error",
											"data": "unexpected end of JSON input",
										},
										"jsonrpc": "2.0",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "unauthorized,need validToken",
						},
						"405": map[string]interface{}{
							"description": "(supportPOSTrequest)",
						},
					},
				},
			},
		},
	}

	enrichSpecWithI18nKeys(spec)
	c.JSON(http.StatusOK, spec)
}
func (h *OpenAPIHandler) GetConversationResults(c *gin.Context) {
	conversationID := c.Param("id")

	// verify conversation exists
	conv, err := h.db.GetConversation(conversationID)
	if err != nil {
		h.logger.Error("getconversationfailed", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation does not exist"})
		return
	}

	// getmessage list
	messages, err := h.db.GetMessages(conversationID)
	if err != nil {
		h.logger.Error("getmessagefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// getvulnerabilitylist
	vulnList, err := h.db.ListVulnerabilities(1000, 0, "", conversationID, "", "")
	if err != nil {
		h.logger.Warn("getvulnerabilitylistfailed", zap.Error(err))
		vulnList = []*database.Vulnerability{}
	}
	vulnerabilities := make([]database.Vulnerability, len(vulnList))
	for i, v := range vulnList {
		vulnerabilities[i] = *v
	}

	// getexecution result(fromMCPexecution recordinget)
	executionResults := []map[string]interface{}{}
	for _, msg := range messages {
		if len(msg.MCPExecutionIDs) > 0 {
			for _, execID := range msg.MCPExecutionIDs {
				if h.resultStorage != nil {
					result, err := h.resultStorage.GetResult(execID)
					if err == nil && result != "" {
						metadata, err := h.resultStorage.GetResultMetadata(execID)
						toolName := "unknown"
						createdAt := time.Now()
						if err == nil && metadata != nil {
							toolName = metadata.ToolName
							createdAt = metadata.CreatedAt
						}
						executionResults = append(executionResults, map[string]interface{}{
							"id": execID,
							"toolName": toolName,
							"status": "success",
							"result": result,
							"createdAt": createdAt.Format(time.RFC3339),
						})
					}
				}
			}
		}
	}

	response := map[string]interface{}{
		"conversationId": conv.ID,
		"messages": messages,
		"vulnerabilities": vulnerabilities,
		"executionResults": executionResults,
	}

	c.JSON(http.StatusOK, response)
}
