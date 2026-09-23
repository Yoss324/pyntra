package handler

import (
	"fmt"
	"strings"

	"pyntra/internal/agent"
	"pyntra/internal/database"
	"pyntra/internal/mcp/builtin"

	"go.uber.org/zap"
)
type multiAgentPrepared struct {
	ConversationID string
	CreatedNew bool
	History []agent.ChatMessage
	FinalMessage string
	RoleTools []string
	RoleSkills []string
	AssistantMessageID string
	UserMessageID string
}

func (h *AgentHandler) prepareMultiAgentSession(req *ChatRequest) (*multiAgentPrepared, error) {
	if len(req.Attachments) > maxAttachments {
		return nil, fmt.Errorf("maximum attachments %d items", maxAttachments)
	}

	conversationID := strings.TrimSpace(req.ConversationID)
	createdNew := false
	if conversationID == "" {
		title := safeTruncateString(req.Message, 50)
		var conv *database.Conversation
		var err error
		if strings.TrimSpace(req.WebShellConnectionID) != "" {
			conv, err = h.db.CreateConversationWithWebshell(strings.TrimSpace(req.WebShellConnectionID), title)
		} else {
			conv, err = h.db.CreateConversation(title)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}
		conversationID = conv.ID
		createdNew = true
	} else {
		if _, err := h.db.GetConversation(conversationID); err != nil {
			return nil, fmt.Errorf("conversation does not exist")
		}
	}

	agentHistoryMessages, err := h.loadHistoryFromReActData(conversationID)
	if err != nil {
		historyMessages, getErr := h.db.GetMessages(conversationID)
		if getErr != nil {
			agentHistoryMessages = []agent.ChatMessage{}
		} else {
			agentHistoryMessages = make([]agent.ChatMessage, 0, len(historyMessages))
			for _, msg := range historyMessages {
				agentHistoryMessages = append(agentHistoryMessages, agent.ChatMessage{
					Role: msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	finalMessage := req.Message
	var roleTools []string
	var roleSkills []string
	if req.WebShellConnectionID != "" {
		conn, errConn := h.db.GetWebshellConnection(strings.TrimSpace(req.WebShellConnectionID))
		if errConn != nil || conn == nil {
			h.logger.Warn("WebShell AI assistant:not foundconnection", zap.String("id", req.WebShellConnectionID), zap.Error(errConn))
			return nil, fmt.Errorf("WebShell connection not found")
		}
		remark := conn.Remark
		if remark == "" {
			remark = conn.URL
		}
		finalMessage = fmt.Sprintf("[WebShell assistant context] current connection ID:%s,remark:%s.available tools (only use when operating on this connection, connection_id fill \"%s\"):webshell_exec/webshell_file_list/webshell_file_read/webshell_file_write/record_vulnerability/list_knowledge_risk_types/search_knowledge_base.Skills use Eino multi-agent `skill` tool.\n\nuser request:%s",
			conn.ID, remark, conn.ID, req.Message)
		roleTools = []string{
			builtin.ToolWebshellExec,
			builtin.ToolWebshellFileList,
			builtin.ToolWebshellFileRead,
			builtin.ToolWebshellFileWrite,
			builtin.ToolRecordVulnerability,
			builtin.ToolListKnowledgeRiskTypes,
			builtin.ToolSearchKnowledgeBase,
		}
	} else if req.Role != "" && req.Role != "default" && h.config != nil && h.config.Roles != nil {
		if role, exists := h.config.Roles[req.Role]; exists && role.Enabled {
			if role.UserPrompt != "" {
				finalMessage = role.UserPrompt + "\n\n" + req.Message
			}
			roleTools = role.Tools
			roleSkills = role.Skills
		}
	}

	var savedPaths []string
	if len(req.Attachments) > 0 {
		var aerr error
		savedPaths, aerr = saveAttachmentsToDateAndConversationDir(req.Attachments, conversationID, h.logger)
		if aerr != nil {
			return nil, fmt.Errorf("failed to save uploaded file: %w", aerr)
		}
	}
	finalMessage = appendAttachmentsToMessage(finalMessage, req.Attachments, savedPaths)

	userContent := userMessageContentForStorage(req.Message, req.Attachments, savedPaths)
	userMsgRow, uerr := h.db.AddMessage(conversationID, "user", userContent, nil)
	if uerr != nil {
		h.logger.Error("failed to save user message", zap.Error(uerr))
		return nil, fmt.Errorf("failed to save user message: %w", uerr)
	}
	userMessageID := ""
	if userMsgRow != nil {
		userMessageID = userMsgRow.ID
	}

	assistantMsg, aerr := h.db.AddMessage(conversationID, "assistant", "processing...", nil)
	var assistantMessageID string
	if aerr != nil {
		h.logger.Warn("createassistantmessagefailed", zap.Error(aerr))
	} else if assistantMsg != nil {
		assistantMessageID = assistantMsg.ID
	}

	return &multiAgentPrepared{
		ConversationID: conversationID,
		CreatedNew: createdNew,
		History: agentHistoryMessages,
		FinalMessage: finalMessage,
		RoleTools: roleTools,
		RoleSkills: roleSkills,
		AssistantMessageID: assistantMessageID,
		UserMessageID: userMessageID,
	}, nil
}
