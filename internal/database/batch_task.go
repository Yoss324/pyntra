package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)
type BatchTaskQueueRow struct {
	ID string
	Title sql.NullString
	Role sql.NullString
	AgentMode sql.NullString
	ScheduleMode sql.NullString
	CronExpr sql.NullString
	NextRunAt sql.NullTime
	ScheduleEnabled sql.NullInt64
	LastScheduleTriggerAt sql.NullTime
	LastScheduleError sql.NullString
	LastRunError sql.NullString
	Status string
	CreatedAt time.Time
	StartedAt sql.NullTime
	CompletedAt sql.NullTime
	CurrentIndex int
}
type BatchTaskRow struct {
	ID string
	QueueID string
	Message string
	ConversationID sql.NullString
	Status string
	StartedAt sql.NullTime
	CompletedAt sql.NullTime
	Error sql.NullString
	Result sql.NullString
}
func (db *DB) CreateBatchQueue(
	queueID string,
	title string,
	role string,
	agentMode string,
	scheduleMode string,
	cronExpr string,
	nextRunAt *time.Time,
	tasks []map[string]interface{},
) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	var nextRunAtValue interface{}
	if nextRunAt != nil {
		nextRunAtValue = *nextRunAt
	}

	_, err = tx.Exec(
		"INSERT INTO batch_task_queues (id, title, role, agent_mode, schedule_mode, cron_expr, next_run_at, schedule_enabled, status, created_at, current_index) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		queueID, title, role, agentMode, scheduleMode, cronExpr, nextRunAtValue, 1, "pending", now, 0,
	)
	if err != nil {
		return fmt.Errorf("createtask queuefailed: %w", err)
	}
	for _, task := range tasks {
		taskID, ok := task["id"].(string)
		if !ok {
			continue
		}
		message, ok := task["message"].(string)
		if !ok {
			continue
		}

		_, err = tx.Exec(
			"INSERT INTO batch_tasks (id, queue_id, message, status) VALUES (?, ?, ?, ?)",
			taskID, queueID, message, "pending",
		)
		if err != nil {
			return fmt.Errorf("createTasksfailed: %w", err)
		}
	}

	return tx.Commit()
}
func (db *DB) GetBatchQueue(queueID string) (*BatchTaskQueueRow, error) {
	var row BatchTaskQueueRow
	var createdAt string
	err := db.QueryRow(
		"SELECT id, title, role, agent_mode, schedule_mode, cron_expr, next_run_at, schedule_enabled, last_schedule_trigger_at, last_schedule_error, last_run_error, status, created_at, started_at, completed_at, current_index FROM batch_task_queues WHERE id = ?",
		queueID,
	).Scan(&row.ID, &row.Title, &row.Role, &row.AgentMode, &row.ScheduleMode, &row.CronExpr, &row.NextRunAt, &row.ScheduleEnabled, &row.LastScheduleTriggerAt, &row.LastScheduleError, &row.LastRunError, &row.Status, &createdAt, &row.StartedAt, &row.CompletedAt, &row.CurrentIndex)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querytask queuefailed: %w", err)
	}

	parsedTime, parseErr := time.Parse("2006-01-02 15:04:05", createdAt)
	if parseErr != nil {
		parsedTime, parseErr = time.Parse(time.RFC3339, createdAt)
		if parseErr != nil {
			db.logger.Warn("failed to parse creation time", zap.String("createdAt", createdAt), zap.Error(parseErr))
			parsedTime = time.Now()
		}
	}
	row.CreatedAt = parsedTime
	return &row, nil
}
func (db *DB) GetAllBatchQueues() ([]*BatchTaskQueueRow, error) {
	rows, err := db.Query(
		"SELECT id, title, role, agent_mode, schedule_mode, cron_expr, next_run_at, schedule_enabled, last_schedule_trigger_at, last_schedule_error, last_run_error, status, created_at, started_at, completed_at, current_index FROM batch_task_queues ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch task queues: %w", err)
	}
	defer rows.Close()

	var queues []*BatchTaskQueueRow
	for rows.Next() {
		var row BatchTaskQueueRow
		var createdAt string
		if err := rows.Scan(&row.ID, &row.Title, &row.Role, &row.AgentMode, &row.ScheduleMode, &row.CronExpr, &row.NextRunAt, &row.ScheduleEnabled, &row.LastScheduleTriggerAt, &row.LastScheduleError, &row.LastRunError, &row.Status, &createdAt, &row.StartedAt, &row.CompletedAt, &row.CurrentIndex); err != nil {
			return nil, fmt.Errorf("failed to scan batch task queue: %w", err)
		}
		parsedTime, parseErr := time.Parse("2006-01-02 15:04:05", createdAt)
		if parseErr != nil {
			parsedTime, parseErr = time.Parse(time.RFC3339, createdAt)
			if parseErr != nil {
				db.logger.Warn("failed to parse creation time", zap.String("createdAt", createdAt), zap.Error(parseErr))
				parsedTime = time.Now()
			}
		}
		row.CreatedAt = parsedTime
		queues = append(queues, &row)
	}

	return queues, nil
}
func (db *DB) ListBatchQueues(limit, offset int, status, keyword string) ([]*BatchTaskQueueRow, error) {
	query := "SELECT id, title, role, agent_mode, schedule_mode, cron_expr, next_run_at, schedule_enabled, last_schedule_trigger_at, last_schedule_error, last_run_error, status, created_at, started_at, completed_at, current_index FROM batch_task_queues WHERE 1=1"
	args := []interface{}{}
	if status != "" && status != "all" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if keyword != "" {
		query += " AND (id LIKE ? OR title LIKE ?)"
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch task queues: %w", err)
	}
	defer rows.Close()

	var queues []*BatchTaskQueueRow
	for rows.Next() {
		var row BatchTaskQueueRow
		var createdAt string
		if err := rows.Scan(&row.ID, &row.Title, &row.Role, &row.AgentMode, &row.ScheduleMode, &row.CronExpr, &row.NextRunAt, &row.ScheduleEnabled, &row.LastScheduleTriggerAt, &row.LastScheduleError, &row.LastRunError, &row.Status, &createdAt, &row.StartedAt, &row.CompletedAt, &row.CurrentIndex); err != nil {
			return nil, fmt.Errorf("failed to scan batch task queue: %w", err)
		}
		parsedTime, parseErr := time.Parse("2006-01-02 15:04:05", createdAt)
		if parseErr != nil {
			parsedTime, parseErr = time.Parse(time.RFC3339, createdAt)
			if parseErr != nil {
				db.logger.Warn("failed to parse creation time", zap.String("createdAt", createdAt), zap.Error(parseErr))
				parsedTime = time.Now()
			}
		}
		row.CreatedAt = parsedTime
		queues = append(queues, &row)
	}

	return queues, nil
}
func (db *DB) CountBatchQueues(status, keyword string) (int, error) {
	query := "SELECT COUNT(*) FROM batch_task_queues WHERE 1=1"
	args := []interface{}{}
	if status != "" && status != "all" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if keyword != "" {
		query += " AND (id LIKE ? OR title LIKE ?)"
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}

	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("task queuefailed: %w", err)
	}

	return count, nil
}
func (db *DB) GetBatchTasks(queueID string) ([]*BatchTaskRow, error) { 
	rows, err := db.Query(
		"SELECT id, queue_id, message, conversation_id, status, started_at, completed_at, error, result FROM batch_tasks WHERE queue_id = ? ORDER BY id",
		queueID,
	)
	if err != nil {
		return nil, fmt.Errorf("queryTasksfailed: %w", err)
	}
	defer rows.Close()

	var tasks []*BatchTaskRow
	for rows.Next() {
		var task BatchTaskRow
		if err := rows.Scan(
			&task.ID, &task.QueueID, &task.Message, &task.ConversationID,
			&task.Status, &task.StartedAt, &task.CompletedAt, &task.Error, &task.Result,
		); err != nil {
			return nil, fmt.Errorf("scanTasksfailed: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}
func (db *DB) UpdateBatchQueueStatus(queueID, status string) error {
	var err error
	now := time.Now()

	if status == "running" {
		_, err = db.Exec(
			"UPDATE batch_task_queues SET status = ?, started_at = COALESCE(started_at, ?) WHERE id = ?",
			status, now, queueID,
		)
	} else if status == "completed" || status == "cancelled" {
		_, err = db.Exec(
			"UPDATE batch_task_queues SET status = ?, completed_at = COALESCE(completed_at, ?) WHERE id = ?",
			status, now, queueID,
		)
	} else {
		_, err = db.Exec(
			"UPDATE batch_task_queues SET status = ? WHERE id = ?",
			status, queueID,
		)
	}

	if err != nil {
		return fmt.Errorf("updatetask queuestatusfailed: %w", err)
	}
	return nil
}
func (db *DB) UpdateBatchTaskStatus(queueID, taskID, status string, conversationID, result, errorMsg string) error {
	var err error
	now := time.Now()
	var updates []string
	var args []interface{}

	updates = append(updates, "status = ?")
	args = append(args, status)

	if conversationID != "" {
		updates = append(updates, "conversation_id = ?")
		args = append(args, conversationID)
	}

	if result != "" {
		updates = append(updates, "result = ?")
		args = append(args, result)
	}

	if errorMsg != "" {
		updates = append(updates, "error = ?")
		args = append(args, errorMsg)
	}

	if status == "running" {
		updates = append(updates, "started_at = COALESCE(started_at, ?)")
		args = append(args, now)
	}

	if status == "completed" || status == "failed" || status == "cancelled" {
		updates = append(updates, "completed_at = COALESCE(completed_at, ?)")
		args = append(args, now)
	}

	args = append(args, queueID, taskID)
	sql := "UPDATE batch_tasks SET "
	for i, update := range updates {
		if i > 0 {
			sql += ", "
		}
		sql += update
	}
	sql += " WHERE queue_id = ? AND id = ?"

	_, err = db.Exec(sql, args...)
	if err != nil {
		return fmt.Errorf("updateTasksstatusfailed: %w", err)
	}
	return nil
}
func (db *DB) UpdateBatchQueueCurrentIndex(queueID string, currentIndex int) error {
	_, err := db.Exec(
		"UPDATE batch_task_queues SET current_index = ? WHERE id = ?",
		currentIndex, queueID,
	)
	if err != nil {
		return fmt.Errorf("updatetask queuecurrentfailed: %w", err)
	}
	return nil
}
func (db *DB) UpdateBatchQueueMetadata(queueID, title, role, agentMode string) error {
	_, err := db.Exec(
		"UPDATE batch_task_queues SET title = ?, role = ?, agent_mode = ? WHERE id = ?",
		title, role, agentMode, queueID,
	)
	if err != nil {
		return fmt.Errorf("updatetask queuemetadatafailed: %w", err)
	}
	return nil
}
func (db *DB) UpdateBatchQueueSchedule(queueID, scheduleMode, cronExpr string, nextRunAt *time.Time) error {
	var nextRunAtValue interface{}
	if nextRunAt != nil {
		nextRunAtValue = *nextRunAt
	}
	_, err := db.Exec(
		"UPDATE batch_task_queues SET schedule_mode = ?, cron_expr = ?, next_run_at = ? WHERE id = ?",
		scheduleMode, cronExpr, nextRunAtValue, queueID,
	)
	if err != nil {
		return fmt.Errorf("updateTasksconfigfailed: %w", err)
	}
	return nil
}
func (db *DB) UpdateBatchQueueScheduleEnabled(queueID string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := db.Exec(
		"UPDATE batch_task_queues SET schedule_enabled = ? WHERE id = ?",
		v, queueID,
	)
	if err != nil {
		return fmt.Errorf("updateTasksfailed: %w", err)
	}
	return nil
}
func (db *DB) RecordBatchQueueScheduledTriggerStart(queueID string, at time.Time) error {
	_, err := db.Exec(
		"UPDATE batch_task_queues SET last_schedule_trigger_at = ?, last_schedule_error = NULL WHERE id = ?",
		at, queueID,
	)
	if err != nil {
		return fmt.Errorf("recordtriggerfailed: %w", err)
	}
	return nil
}
func (db *DB) SetBatchQueueLastScheduleError(queueID, msg string) error {
	_, err := db.Exec(
		"UPDATE batch_task_queues SET last_schedule_error = ? WHERE id = ?",
		msg, queueID,
	)
	if err != nil {
		return fmt.Errorf("errorfailed: %w", err)
	}
	return nil
}
func (db *DB) SetBatchQueueLastRunError(queueID, msg string) error {
	var v interface{}
	if strings.TrimSpace(msg) == "" {
		v = nil
	} else {
		v = msg
	}
	_, err := db.Exec(
		"UPDATE batch_task_queues SET last_run_error = ? WHERE id = ?",
		v, queueID,
	)
	if err != nil {
		return fmt.Errorf("errorfailed: %w", err)
	}
	return nil
}
func (db *DB) ResetBatchQueueForRerun(queueID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE batch_task_queues SET status = ?, current_index = 0, started_at = NULL, completed_at = NULL, last_run_error = NULL, last_schedule_error = NULL WHERE id = ?",
		"pending", queueID,
	)
	if err != nil {
		return fmt.Errorf("task queuestatusfailed: %w", err)
	}

	_, err = tx.Exec(
		"UPDATE batch_tasks SET status = ?, conversation_id = NULL, started_at = NULL, completed_at = NULL, error = NULL, result = NULL WHERE queue_id = ?",
		"pending", queueID,
	)
	if err != nil {
		return fmt.Errorf("Tasksstatusfailed: %w", err)
	}

	return tx.Commit()
}
func (db *DB) UpdateBatchTaskMessage(queueID, taskID, message string) error {
	_, err := db.Exec(
		"UPDATE batch_tasks SET message = ? WHERE queue_id = ? AND id = ?",
		message, queueID, taskID,
	)
	if err != nil {
		return fmt.Errorf("updateTasksmessagefailed: %w", err)
	}
	return nil
}
func (db *DB) AddBatchTask(queueID, taskID, message string) error {
	_, err := db.Exec(
		"INSERT INTO batch_tasks (id, queue_id, message, status) VALUES (?, ?, ?, ?)",
		taskID, queueID, message, "pending",
	)
	if err != nil {
		return fmt.Errorf("Tasksfailed: %w", err)
	}
	return nil
}
func (db *DB) CancelPendingBatchTasks(queueID string, completedAt time.Time) error {
	_, err := db.Exec(
		"UPDATE batch_tasks SET status = ?, completed_at = ? WHERE queue_id = ? AND status = ?",
		"cancelled", completedAt, queueID, "pending",
	)
	if err != nil {
		return fmt.Errorf("cancel pending Tasksfailed: %w", err)
	}
	return nil
}
func (db *DB) DeleteBatchTask(queueID, taskID string) error {
	_, err := db.Exec(
		"DELETE FROM batch_tasks WHERE queue_id = ? AND id = ?",
		queueID, taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete batch task: %w", err)
	}
	return nil
}
func (db *DB) DeleteBatchQueue(queueID string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.Exec("DELETE FROM batch_tasks WHERE queue_id = ?", queueID)
	if err != nil {
		return fmt.Errorf("failed to delete batch task: %w", err)
	}
	_, err = tx.Exec("DELETE FROM batch_task_queues WHERE id = ?", queueID)
	if err != nil {
		return fmt.Errorf("Failed to delete batch task queue: %w", err)
	}

	return tx.Commit()
}
