package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)
type Conversation struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Pinned bool `json:"pinned"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Messages []Message `json:"messages,omitempty"`
}
type Message struct {
	ID string `json:"id"`
	ConversationID string `json:"conversationId"`
	Role string `json:"role"`
	Content string `json:"content"`
	MCPExecutionIDs []string `json:"mcpExecutionIds,omitempty"`
	ProcessDetails []map[string]interface{} `json:"processDetails,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
func (db *DB) CreateConversation(title string) (*Conversation, error) {
	return db.CreateConversationWithWebshell("", title)
}
func (db *DB) CreateConversationWithWebshell(webshellConnectionID, title string) (*Conversation, error) {
	id := uuid.New().String()
	now := time.Now()

	var err error
	if webshellConnectionID != "" {
		_, err = db.Exec(
			"INSERT INTO conversations (id, title, created_at, updated_at, webshell_connection_id) VALUES (?, ?, ?, ?, ?)",
			id, title, now, now, webshellConnectionID,
		)
	} else {
		_, err = db.Exec(
			"INSERT INTO conversations (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)",
			id, title, now, now,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("createChatfailed: %w", err)
	}

	return &Conversation{
		ID: id,
		Title: title,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
func (db *DB) GetConversationByWebshellConnectionID(connectionID string) (*Conversation, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("connectionID is empty")
	}
	var conv Conversation
	var createdAt, updatedAt string
	var pinned int
	err := db.QueryRow(
		"SELECT id, title, pinned, created_at, updated_at FROM conversations WHERE webshell_connection_id = ? ORDER BY updated_at DESC LIMIT 1",
		connectionID,
	).Scan(&conv.ID, &conv.Title, &pinned, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query conversation: %w", err)
	}
	conv.Pinned = pinned != 0
	if t, e := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt); e == nil {
		conv.CreatedAt = t
	} else if t, e := time.Parse("2006-01-02 15:04:05", createdAt); e == nil {
		conv.CreatedAt = t
	} else {
		conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}
	if t, e := time.Parse("2006-01-02 15:04:05.999999999-07:00", updatedAt); e == nil {
		conv.UpdatedAt = t
	} else if t, e := time.Parse("2006-01-02 15:04:05", updatedAt); e == nil {
		conv.UpdatedAt = t
	} else {
		conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	}
	messages, err := db.GetMessages(conv.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}
	conv.Messages = messages
	processDetailsMap, err := db.GetProcessDetailsByConversation(conv.ID)
	if err != nil {
		db.logger.Warn("loadprocess detailsfailed", zap.Error(err))
		processDetailsMap = make(map[string][]ProcessDetail)
	}
	for i := range conv.Messages {
		if details, ok := processDetailsMap[conv.Messages[i].ID]; ok {
			detailsJSON := make([]map[string]interface{}, len(details))
			for j, detail := range details {
				var data interface{}
				if detail.Data != "" {
					if err := json.Unmarshal([]byte(detail.Data), &data); err != nil {
						db.logger.Warn("parseprocess detailsfailed", zap.Error(err))
					}
				}
				detailsJSON[j] = map[string]interface{}{
					"id": detail.ID,
					"messageId": detail.MessageID,
					"conversationId": detail.ConversationID,
					"eventType": detail.EventType,
					"message": detail.Message,
					"data": data,
					"createdAt": detail.CreatedAt,
				}
			}
			conv.Messages[i].ProcessDetails = detailsJSON
		}
	}

	return &conv, nil
}
type WebShellConversationItem struct {
	ID string `json:"id"`
	Title string `json:"title"`
	UpdatedAt time.Time `json:"updatedAt"`
}
func (db *DB) ListConversationsByWebshellConnectionID(connectionID string) ([]WebShellConversationItem, error) {
	if connectionID == "" {
		return nil, nil
	}
	rows, err := db.Query(
		"SELECT id, title, updated_at FROM conversations WHERE webshell_connection_id = ? ORDER BY updated_at DESC",
		connectionID,
	)
	if err != nil {
		return nil, fmt.Errorf("queryChatfailed: %w", err)
	}
	defer rows.Close()
	var list []WebShellConversationItem
	for rows.Next() {
		var item WebShellConversationItem
		var updatedAt string
		if err := rows.Scan(&item.ID, &item.Title, &updatedAt); err != nil {
			continue
		}
		if t, e := time.Parse("2006-01-02 15:04:05.999999999-07:00", updatedAt); e == nil {
			item.UpdatedAt = t
		} else if t, e := time.Parse("2006-01-02 15:04:05", updatedAt); e == nil {
			item.UpdatedAt = t
		} else {
			item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}
func (db *DB) GetConversation(id string) (*Conversation, error) {
	var conv Conversation
	var createdAt, updatedAt string
	var pinned int

	err := db.QueryRow(
		"SELECT id, title, pinned, created_at, updated_at FROM conversations WHERE id = ?",
		id,
	).Scan(&conv.ID, &conv.Title, &pinned, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation does not exist")
		}
		return nil, fmt.Errorf("failed to query conversation: %w", err)
	}
	var err1, err2 error
	conv.CreatedAt, err1 = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt)
	if err1 != nil {
		conv.CreatedAt, err1 = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if err1 != nil {
		conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	conv.UpdatedAt, err2 = time.Parse("2006-01-02 15:04:05.999999999-07:00", updatedAt)
	if err2 != nil {
		conv.UpdatedAt, err2 = time.Parse("2006-01-02 15:04:05", updatedAt)
	}
	if err2 != nil {
		conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	}

	conv.Pinned = pinned != 0
	messages, err := db.GetMessages(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}
	conv.Messages = messages
	processDetailsMap, err := db.GetProcessDetailsByConversation(id)
	if err != nil {
		db.logger.Warn("loadprocess detailsfailed", zap.Error(err))
		processDetailsMap = make(map[string][]ProcessDetail)
	}
	for i := range conv.Messages {
		if details, ok := processDetailsMap[conv.Messages[i].ID]; ok {
			detailsJSON := make([]map[string]interface{}, len(details))
			for j, detail := range details {
				var data interface{}
				if detail.Data != "" {
					if err := json.Unmarshal([]byte(detail.Data), &data); err != nil {
						db.logger.Warn("parseprocess detailsfailed", zap.Error(err))
					}
				}
				detailsJSON[j] = map[string]interface{}{
					"id": detail.ID,
					"messageId": detail.MessageID,
					"conversationId": detail.ConversationID,
					"eventType": detail.EventType,
					"message": detail.Message,
					"data": data,
					"createdAt": detail.CreatedAt,
				}
			}
			conv.Messages[i].ProcessDetails = detailsJSON
		}
	}

	return &conv, nil
}
func (db *DB) GetConversationLite(id string) (*Conversation, error) {
	var conv Conversation
	var createdAt, updatedAt string
	var pinned int

	err := db.QueryRow(
		"SELECT id, title, pinned, created_at, updated_at FROM conversations WHERE id = ?",
		id,
	).Scan(&conv.ID, &conv.Title, &pinned, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation does not exist")
		}
		return nil, fmt.Errorf("failed to query conversation: %w", err)
	}
	var err1, err2 error
	conv.CreatedAt, err1 = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt)
	if err1 != nil {
		conv.CreatedAt, err1 = time.Parse("2006-01-02 15:04:05", createdAt)
	}
	if err1 != nil {
		conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	conv.UpdatedAt, err2 = time.Parse("2006-01-02 15:04:05.999999999-07:00", updatedAt)
	if err2 != nil {
		conv.UpdatedAt, err2 = time.Parse("2006-01-02 15:04:05", updatedAt)
	}
	if err2 != nil {
		conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	}

	conv.Pinned = pinned != 0
	messages, err := db.GetMessages(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}
	conv.Messages = messages
	return &conv, nil
}
func (db *DB) ListConversations(limit, offset int, search string) ([]*Conversation, error) {
	var rows *sql.Rows
	var err error
	
	if search != "" {
		searchPattern := "%" + search + "%"
		rows, err = db.Query(
			`SELECT c.id, c.title, COALESCE(c.pinned, 0), c.created_at, c.updated_at
			 FROM conversations c
			 WHERE c.title LIKE ?
			 OR EXISTS (SELECT 1 FROM messages m WHERE m.conversation_id = c.id AND m.content LIKE ?)
			 ORDER BY c.updated_at DESC
			 LIMIT ? OFFSET ?`,
			searchPattern, searchPattern, limit, offset,
		)
	} else {
		rows, err = db.Query(
			"SELECT id, title, COALESCE(pinned, 0), created_at, updated_at FROM conversations ORDER BY updated_at DESC LIMIT ? OFFSET ?",
			limit, offset,
		)
	}
	
	if err != nil {
		return nil, fmt.Errorf("queryChatfailed: %w", err)
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		var conv Conversation
		var createdAt, updatedAt string
		var pinned int

		if err := rows.Scan(&conv.ID, &conv.Title, &pinned, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		var err1, err2 error
		conv.CreatedAt, err1 = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt)
		if err1 != nil {
			conv.CreatedAt, err1 = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		if err1 != nil {
			conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}

		conv.UpdatedAt, err2 = time.Parse("2006-01-02 15:04:05.999999999-07:00", updatedAt)
		if err2 != nil {
			conv.UpdatedAt, err2 = time.Parse("2006-01-02 15:04:05", updatedAt)
		}
		if err2 != nil {
			conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		}

		conv.Pinned = pinned != 0

		conversations = append(conversations, &conv)
	}

	return conversations, nil
}
func (db *DB) UpdateConversationTitle(id, title string) error {
	_, err := db.Exec(
		"UPDATE conversations SET title = ? WHERE id = ?",
		title, id,
	)
	if err != nil {
		return fmt.Errorf("updateChattitlefailed: %w", err)
	}
	return nil
}
func (db *DB) UpdateConversationTime(id string) error {
	_, err := db.Exec(
		"UPDATE conversations SET updated_at = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updateChatfailed: %w", err)
	}
	return nil
}
func (db *DB) DeleteConversation(id string) error {
	_, err := db.Exec("DELETE FROM knowledge_retrieval_logs WHERE conversation_id = ?", id)
	if err != nil {
		db.logger.Warn("deleteKnowledgeretrievallogsfailed", zap.String("conversationId", id), zap.Error(err))
	}
	_, err = db.Exec("DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleteChatfailed: %w", err)
	}

	db.logger.Info("Chatdelete", zap.String("conversationId", id))
	return nil
}
func (db *DB) SaveReActData(conversationID, reactInput, reactOutput string) error {
	_, err := db.Exec(
		"UPDATE conversations SET last_react_input = ?, last_react_output = ?, updated_at = ? WHERE id = ?",
		reactInput, reactOutput, time.Now(), conversationID,
	)
	if err != nil {
		return fmt.Errorf("saveReActfailed: %w", err)
	}
	return nil
}
func (db *DB) GetReActData(conversationID string) (reactInput, reactOutput string, err error) {
	var input, output sql.NullString
	err = db.QueryRow(
		"SELECT last_react_input, last_react_output FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&input, &output)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("conversation does not exist")
		}
		return "", "", fmt.Errorf("fetchReActfailed: %w", err)
	}

	if input.Valid {
		reactInput = input.String
	}
	if output.Valid {
		reactOutput = output.String
	}

	return reactInput, reactOutput, nil
}
func (db *DB) ConversationHasToolProcessDetails(conversationID string) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM process_details WHERE conversation_id = ? AND event_type IN ('tool_call', 'tool_result')`,
		conversationID,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("failed to query process details: %w", err)
	}
	return n > 0, nil
}
func (db *DB) AddMessage(conversationID, role, content string, mcpExecutionIDs []string) (*Message, error) {
	id := uuid.New().String()

	var mcpIDsJSON string
	if len(mcpExecutionIDs) > 0 {
		jsonData, err := json.Marshal(mcpExecutionIDs)
		if err != nil {
			db.logger.Warn("MCPIDfailed", zap.Error(err))
		} else {
			mcpIDsJSON = string(jsonData)
		}
	}

	_, err := db.Exec(
		"INSERT INTO messages (id, conversation_id, role, content, mcp_execution_ids, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, conversationID, role, content, mcpIDsJSON, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("messagefailed: %w", err)
	}
	if err := db.UpdateConversationTime(conversationID); err != nil {
		db.logger.Warn("updateChatfailed", zap.Error(err))
	}

	message := &Message{
		ID: id,
		ConversationID: conversationID,
		Role: role,
		Content: content,
		MCPExecutionIDs: mcpExecutionIDs,
		CreatedAt: time.Now(),
	}

	return message, nil
}
func (db *DB) GetMessages(conversationID string) ([]Message, error) {
	rows, err := db.Query(
		"SELECT id, conversation_id, role, content, mcp_execution_ids, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("querymessagefailed: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var mcpIDsJSON sql.NullString
		var createdAt string

		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &mcpIDsJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scanmessagefailed: %w", err)
		}
		var err error
		msg.CreatedAt, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt)
		if err != nil {
			msg.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		if err != nil {
			msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}
		if mcpIDsJSON.Valid && mcpIDsJSON.String != "" {
			if err := json.Unmarshal([]byte(mcpIDsJSON.String), &msg.MCPExecutionIDs); err != nil {
				db.logger.Warn("parseMCPIDfailed", zap.Error(err))
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
func turnSliceRange(msgs []Message, anchorID string) (start, end int, err error) {
	idx := -1
	for i := range msgs {
		if msgs[i].ID == anchorID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0, 0, fmt.Errorf("message not found")
	}
	start = idx
	for start > 0 && msgs[start].Role != "user" {
		start--
	}
	if start < len(msgs) && msgs[start].Role != "user" {
		start = 0
	}
	end = len(msgs)
	for i := start + 1; i < len(msgs); i++ {
		if msgs[i].Role == "user" {
			end = i
			break
		}
	}
	return start, end, nil
}
func (db *DB) DeleteConversationTurn(conversationID, anchorMessageID string) (deletedIDs []string, err error) {
	msgs, err := db.GetMessages(conversationID)
	if err != nil {
		return nil, err
	}
	start, end, err := turnSliceRange(msgs, anchorMessageID)
	if err != nil {
		return nil, err
	}
	if start >= end {
		return nil, fmt.Errorf("empty turn range")
	}
	deletedIDs = make([]string, 0, end-start)
	for i := start; i < end; i++ {
		deletedIDs = append(deletedIDs, msgs[i].ID)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ph := strings.Repeat("?,", len(deletedIDs))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, 0, 1+len(deletedIDs))
	args = append(args, conversationID)
	for _, id := range deletedIDs {
		args = append(args, id)
	}
	res, err := tx.Exec(
		"DELETE FROM messages WHERE conversation_id = ? AND id IN ("+ph+")",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("delete messages: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if int(n) != len(deletedIDs) {
		return nil, fmt.Errorf("deleted count mismatch")
	}

	_, err = tx.Exec(
		`UPDATE conversations SET last_react_input = NULL, last_react_output = NULL, updated_at = ? WHERE id = ?`,
		time.Now(), conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("clear react data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	db.logger.Info("conversation turn deleted",
		zap.String("conversationId", conversationID),
		zap.Strings("deletedMessageIds", deletedIDs),
		zap.Int("count", len(deletedIDs)),
	)
	return deletedIDs, nil
}
type ProcessDetail struct {
	ID string `json:"id"`
	MessageID string `json:"messageId"`
	ConversationID string `json:"conversationId"`
	EventType string `json:"eventType"` // iteration, thinking, tool_calls_detected, tool_call, tool_result, progress, error
	Message string `json:"message"`
	Data string `json:"data"`
	CreatedAt time.Time `json:"createdAt"`
}
func (db *DB) AddProcessDetail(messageID, conversationID, eventType, message string, data interface{}) error {
	id := uuid.New().String()

	var dataJSON string
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			db.logger.Warn("process detailsfailed", zap.Error(err))
		} else {
			dataJSON = string(jsonData)
		}
	}

	_, err := db.Exec(
		"INSERT INTO process_details (id, message_id, conversation_id, event_type, message, data, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, messageID, conversationID, eventType, message, dataJSON, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("process detailsfailed: %w", err)
	}

	return nil
}
func (db *DB) GetProcessDetails(messageID string) ([]ProcessDetail, error) {
	rows, err := db.Query(
		"SELECT id, message_id, conversation_id, event_type, message, data, created_at FROM process_details WHERE message_id = ? ORDER BY created_at ASC",
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query process details: %w", err)
	}
	defer rows.Close()

	var details []ProcessDetail
	for rows.Next() {
		var detail ProcessDetail
		var createdAt string

		if err := rows.Scan(&detail.ID, &detail.MessageID, &detail.ConversationID, &detail.EventType, &detail.Message, &detail.Data, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan process details: %w", err)
		}
		var err error
		detail.CreatedAt, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt)
		if err != nil {
			detail.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		if err != nil {
			detail.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}

		details = append(details, detail)
	}

	return details, nil
}
func (db *DB) GetProcessDetailsByConversation(conversationID string) (map[string][]ProcessDetail, error) {
	rows, err := db.Query(
		"SELECT id, message_id, conversation_id, event_type, message, data, created_at FROM process_details WHERE conversation_id = ? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query process details: %w", err)
	}
	defer rows.Close()

	detailsMap := make(map[string][]ProcessDetail)
	for rows.Next() {
		var detail ProcessDetail
		var createdAt string

		if err := rows.Scan(&detail.ID, &detail.MessageID, &detail.ConversationID, &detail.EventType, &detail.Message, &detail.Data, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan process details: %w", err)
		}
		var err error
		detail.CreatedAt, err = time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt)
		if err != nil {
			detail.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		}
		if err != nil {
			detail.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}

		detailsMap[detail.MessageID] = append(detailsMap[detail.MessageID], detail)
	}

	return detailsMap, nil
}
