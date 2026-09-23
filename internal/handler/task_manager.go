package handler

import (
	"context"
	"errors"
	"sync"
	"time"
)
var ErrTaskCancelled = errors.New("agent task cancelled by user")
var ErrTaskAlreadyRunning = errors.New("agent task already running for conversation")
type AgentTask struct {
	ConversationID string    `json:"conversationId"`
	Message        string    `json:"message,omitempty"`
	StartedAt      time.Time `json:"startedAt"`
	Status         string    `json:"status"`
	CancellingAt   time.Time `json:"-"`

	cancel func(error)
}
type CompletedTask struct {
	ConversationID string    `json:"conversationId"`
	Message        string    `json:"message,omitempty"`
	StartedAt      time.Time `json:"startedAt"`
	CompletedAt    time.Time `json:"completedAt"`
	Status         string    `json:"status"`
}
type AgentTaskManager struct {
	mu             sync.RWMutex
	tasks          map[string]*AgentTask
	completedTasks []*CompletedTask
	maxHistorySize int
	historyRetention time.Duration
}

const (
	cancellingStuckThreshold = 45 * time.Second
	cancellingStuckThresholdLegacy = 2 * time.Minute
	cleanupInterval                = 15 * time.Second
)

// NewAgentTaskManager createtaskmanagedevice/processor
func NewAgentTaskManager() *AgentTaskManager {
	m := &AgentTaskManager{
		tasks:            make(map[string]*AgentTask),
		completedTasks:   make([]*CompletedTask, 0),
		maxHistorySize:   50,
		historyRetention: 24 * time.Hour,
	}
	go m.runStuckCancellingCleanup()
	return m
}
func (m *AgentTaskManager) runStuckCancellingCleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		m.cleanupStuckCancelling()
	}
}

func (m *AgentTaskManager) cleanupStuckCancelling() {
	m.mu.Lock()
	var toFinish []string
	now := time.Now()
	for id, task := range m.tasks {
		if task.Status != "cancelling" {
			continue
		}
		var elapsed time.Duration
		if !task.CancellingAt.IsZero() {
			elapsed = now.Sub(task.CancellingAt)
			if elapsed < cancellingStuckThreshold {
				continue
			}
		} else {
			elapsed = now.Sub(task.StartedAt)
			if elapsed < cancellingStuckThresholdLegacy {
				continue
			}
		}
		toFinish = append(toFinish, id)
	}
	m.mu.Unlock()
	for _, id := range toFinish {
		m.FinishTask(id, "cancelled")
	}
}
func (m *AgentTaskManager) StartTask(conversationID, message string, cancel context.CancelCauseFunc) (*AgentTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[conversationID]; exists {
		return nil, ErrTaskAlreadyRunning
	}

	task := &AgentTask{
		ConversationID: conversationID,
		Message:        message,
		StartedAt:      time.Now(),
		Status:         "running",
		cancel: func(err error) {
			if cancel != nil {
				cancel(err)
			}
		},
	}

	m.tasks[conversationID] = task
	return task, nil
}
func (m *AgentTaskManager) CancelTask(conversationID string, cause error) (bool, error) {
	m.mu.Lock()
	task, exists := m.tasks[conversationID]
	if !exists {
		m.mu.Unlock()
		return false, nil
	}
	if task.Status == "cancelling" {
		m.mu.Unlock()
		return true, nil
	}

	task.Status = "cancelling"
	task.CancellingAt = time.Now()
	cancel := task.cancel
	m.mu.Unlock()

	if cause == nil {
		cause = ErrTaskCancelled
	}
	if cancel != nil {
		cancel(cause)
	}
	return true, nil
}
func (m *AgentTaskManager) UpdateTaskStatus(conversationID string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[conversationID]
	if !exists {
		return
	}

	if status != "" {
		task.Status = status
	}
}
func (m *AgentTaskManager) FinishTask(conversationID string, finalStatus string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[conversationID]
	if !exists {
		return
	}

	if finalStatus != "" {
		task.Status = finalStatus
	}
	completedTask := &CompletedTask{
		ConversationID: task.ConversationID,
		Message:        task.Message,
		StartedAt:       task.StartedAt,
		CompletedAt:     time.Now(),
		Status:          finalStatus,
	}
	m.completedTasks = append(m.completedTasks, completedTask)
	m.cleanupHistory()
	delete(m.tasks, conversationID)
}
func (m *AgentTaskManager) cleanupHistory() {
	now := time.Now()
	cutoffTime := now.Add(-m.historyRetention)
	validTasks := make([]*CompletedTask, 0, len(m.completedTasks))
	for _, task := range m.completedTasks {
		if task.CompletedAt.After(cutoffTime) {
			validTasks = append(validTasks, task)
		}
	}
	if len(validTasks) > m.maxHistorySize {
		start := len(validTasks) - m.maxHistorySize
		validTasks = validTasks[start:]
	}
	
	m.completedTasks = validTasks
}
func (m *AgentTaskManager) GetActiveTasks() []*AgentTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*AgentTask, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, &AgentTask{
			ConversationID: task.ConversationID,
			Message:        task.Message,
			StartedAt:      task.StartedAt,
			Status:         task.Status,
		})
	}
	return result
}
func (m *AgentTaskManager) GetCompletedTasks() []*CompletedTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now()
	cutoffTime := now.Add(-m.historyRetention)
	
	result := make([]*CompletedTask, 0, len(m.completedTasks))
	for _, task := range m.completedTasks {
		if task.CompletedAt.After(cutoffTime) {
			result = append(result, task)
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) > m.maxHistorySize {
		result = result[:m.maxHistorySize]
	}
	
	return result
}
