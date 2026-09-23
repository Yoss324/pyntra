package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pyntra/internal/mcp"
	"pyntra/internal/mcp/builtin"

	"go.uber.org/zap"
)
func RegisterBatchTaskMCPTools(mcpServer *mcp.Server, h *AgentHandler, logger *zap.Logger) {
	if mcpServer == nil || h == nil || logger == nil {
		return
	}

	reg := func(tool mcp.Tool, fn func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error)) {
		mcpServer.RegisterTool(tool, fn)
	}

	// --- list ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskList,
		Description: "listbatch task queue(,).queuedata/task id/status/of message/status.task( result/error/conversationId/) batch_task_get(queue_id).",
		ShortDescription: "listbatch task queue",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type": "string",
					"description": "filterstatus:all(default)/pending/running/paused/completed/cancelled",
					"enum": []string{"all", "pending", "running", "paused", "completed", "cancelled"},
				},
				"keyword": map[string]interface{}{
					"type": "string",
					"description": "queue ID ortitlesearch",
				},
				"page": map[string]interface{}{
					"type": "integer",
					"description": "page,from 1 start,default 1",
				},
				"page_size": map[string]interface{}{
					"type": "integer",
					"description": "each/per,default 20, 100",
				},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		status := mcpArgString(args, "status")
		if status == "" {
			status = "all"
		}
		keyword := mcpArgString(args, "keyword")
		page := int(mcpArgFloat(args, "page"))
		if page <= 0 {
			page = 1
		}
		pageSize := int(mcpArgFloat(args, "page_size"))
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}
		offset := (page - 1) * pageSize
		if offset > 100000 {
			offset = 100000
		}
		queues, total, err := h.batchTaskManager.ListQueues(pageSize, offset, status, keyword)
		if err != nil {
			return batchMCPTextResult(fmt.Sprintf("listqueuefailed: %v", err), true), nil
		}
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		slim := make([]batchTaskQueueMCPListItem, 0, len(queues))
		for _, q := range queues {
			if q == nil {
				continue
			}
			slim = append(slim, toBatchTaskQueueMCPListItem(q))
		}
		payload := map[string]interface{}{
			"queues": slim,
			"total": total,
			"page": page,
			"page_size": pageSize,
			"total_pages": totalPages,
		}
		logger.Info("MCP batch_task_list", zap.String("status", status), zap.Int("total", total))
		return batchMCPJSONResult(payload)
	})

	// --- get ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskGet,
		Description: " queue_id getitemsbatch task queuedetails(task list/Cron/anderror message).",
		ShortDescription: "getbatch task queuedetails",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
			},
			"required": []string{"queue_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		queue, ok := h.batchTaskManager.GetBatchQueue(qid)
		if !ok {
			return batchMCPTextResult("queue does not exist: "+qid, true), nil
		}
		return batchMCPJSONResult(queue)
	})

	// --- create ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskCreate,
		Description: `【】apply「taskmanage / batch task queue」:ofqueue,//continue/.queuedataand,items""current.

【】batchexecute/Cron /orneedandtaskmanagepagecall.need/currentconversationof/,conversationcompleted,is""createqueue.

【parameter】tasks()or tasks_text(,each/per);each/perqueueexecuteof.agent_mode:single( ReAct,default)/eino_single(Eino ADK single agent)/deep / plan_execute / supervisor(needenablemulti-agent); multi(is deep)."conversation".schedule_mode:manual(default)or cron;cron cron_expr(5 ,such as "0 */6 * * *").

【execute】defaultcreateis pending,.execute_now=true create;call batch_task_start.Cron need schedule_enabled is true( batch_task_schedule_enabled).`,
		ShortDescription: "taskmanage:createbatch task queue(,optionalor Cron)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type": "string",
					"description": "optionalqueue title,taskmanagein",
				},
				"role": map[string]interface{}{
					"type": "string",
					"description": "queueuseofrole,default",
				},
				"tasks": map[string]interface{}{
					"type": "array",
					"description": "queueinoftask,each/perexecute(and tasks_text )",
					"items": map[string]interface{}{"type": "string"},
				},
				"tasks_text": map[string]interface{}{
					"type": "string",
					"description": ",each/pertask(and tasks )",
				},
				"agent_mode": map[string]interface{}{
					"type": "string",
					"description": "executemode:single( ReAct)/eino_single(Eino ADK)/deep/plan_execute/supervisor(Eino ,needenablemulti-agent);multi is deep",
					"enum": []string{"single", "eino_single", "deep", "plan_execute", "supervisor", "multi"},
				},
				"schedule_mode": map[string]interface{}{
					"type": "string",
					"description": "manual(/start)or cron(expressiontrigger)",
					"enum": []string{"manual", "cron"},
				},
				"cron_expr": map[string]interface{}{
					"type": "string",
					"description": "schedule_mode is cron . 5 : minutes hours ,such as \"0 */6 * * *\"/\"30 2 * * 1-5\"",
				},
				"execute_now": map[string]interface{}{
					"type": "boolean",
					"description": "createstartexecutequeue,default false(pending,need batch_task_start)",
				},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		tasks, errMsg := batchMCPTasksFromArgs(args)
		if errMsg != "" {
			return batchMCPTextResult(errMsg, true), nil
		}
		title := mcpArgString(args, "title")
		role := mcpArgString(args, "role")
		agentMode := normalizeBatchQueueAgentMode(mcpArgString(args, "agent_mode"))
		scheduleMode := normalizeBatchQueueScheduleMode(mcpArgString(args, "schedule_mode"))
		cronExpr := strings.TrimSpace(mcpArgString(args, "cron_expr"))
		var nextRunAt *time.Time
		if scheduleMode == "cron" {
			if cronExpr == "" {
				return batchMCPTextResult("Cron mode cron_expr cannot be empty", true), nil
			}
			sch, err := h.batchCronParser.Parse(cronExpr)
			if err != nil {
				return batchMCPTextResult("invalid cron expression: "+err.Error(), true), nil
			}
			n := sch.Next(time.Now())
			nextRunAt = &n
		}
		executeNow, ok := mcpArgBool(args, "execute_now")
		if !ok {
			executeNow = false
		}
		queue, createErr := h.batchTaskManager.CreateBatchQueue(title, role, agentMode, scheduleMode, cronExpr, nextRunAt, tasks)
		if createErr != nil {
			return batchMCPTextResult("createqueuefailed: "+createErr.Error(), true), nil
		}
		started := false
		if executeNow {
			ok, err := h.startBatchQueueExecution(queue.ID, false)
			if !ok {
				return batchMCPTextResult("queue does not exist: "+queue.ID, true), nil
			}
			if err != nil {
				return batchMCPTextResult("createsuccessfulstartfailed: "+err.Error(), true), nil
			}
			started = true
			if refreshed, exists := h.batchTaskManager.GetBatchQueue(queue.ID); exists {
				queue = refreshed
			}
		}
		logger.Info("MCP batch_task_create", zap.String("queueId", queue.ID), zap.Int("taskCount", len(tasks)))
		return batchMCPJSONResult(map[string]interface{}{
			"queue_id": queue.ID,
			"queue": queue,
			"started": started,
			"execute_now": executeNow,
			"reminder": func() string {
				if started {
					return "queuehascreatestart."
				}
				return "queuehascreate,currentis pending.needstartexecutecall MCP tool batch_task_start(queue_id ).Cron need schedule_enabled is true, batch_task_schedule_enabled."
			}(),
		})
	})

	// --- start ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskStart,
		Description: `startorcontinueexecutebatch task queue(pending / paused).
and batch_task_create use:createqueueexecute,needcalltoolstarttask.`,
		ShortDescription: "start/continuebatch task queue(createneedcallexecute)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
			},
			"required": []string{"queue_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		ok, err := h.startBatchQueueExecution(qid, false)
		if !ok {
			return batchMCPTextResult("queue does not exist: "+qid, true), nil
		}
		if err != nil {
			return batchMCPTextResult("startfailed: "+err.Error(), true), nil
		}
		logger.Info("MCP batch_task_start", zap.String("queueId", qid))
		return batchMCPTextResult("hasstart,queuestartexecute.", false), nil
	})

	// --- rerun (reset + start for completed/cancelled queues) ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskRerun,
		Description: "completedorhascancelofbatch task queue.resettask statusexecute.",
		ShortDescription: "batch task queue",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
			},
			"required": []string{"queue_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		queue, exists := h.batchTaskManager.GetBatchQueue(qid)
		if !exists {
			return batchMCPTextResult("queue does not exist: "+qid, true), nil
		}
		if queue.Status != "completed" && queue.Status != "cancelled" {
			return batchMCPTextResult("completedorhascancelofqueue,currentstatus: "+queue.Status, true), nil
		}
		if !h.batchTaskManager.ResetQueueForRerun(qid) {
			return batchMCPTextResult("resetqueuefailed", true), nil
		}
		ok, err := h.startBatchQueueExecution(qid, false)
		if !ok {
			return batchMCPTextResult("startfailed", true), nil
		}
		if err != nil {
			return batchMCPTextResult("startfailed: "+err.Error(), true), nil
		}
		logger.Info("MCP batch_task_rerun", zap.String("queueId", qid))
		return batchMCPTextResult("hasresetstartqueue.", false), nil
	})

	// --- pause ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskPause,
		Description: "ofbatch task queue(currenttaskcancel).",
		ShortDescription: "batch task queue",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
			},
			"required": []string{"queue_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		if !h.batchTaskManager.PauseQueue(qid) {
			return batchMCPTextResult(":queue does not existorcurrent running status", true), nil
		}
		logger.Info("MCP batch_task_pause", zap.String("queueId", qid))
		return batchMCPTextResult("queuehas.", false), nil
	})

	// --- delete queue ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskDelete,
		Description: "deletebatch task queuetaskrecord.",
		ShortDescription: "deletebatch task queue",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
			},
			"required": []string{"queue_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		if !h.batchTaskManager.DeleteQueue(qid) {
			return batchMCPTextResult("deletefailed:queue does not exist", true), nil
		}
		logger.Info("MCP batch_task_delete", zap.String("queueId", qid))
		return batchMCPTextResult("queuehasdelete.", false), nil
	})

	// --- update metadata (title/role/agentMode) ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskUpdateMetadata,
		Description: "modifybatch task queueoftitle/roleandmode.queue running statusmodify.",
		ShortDescription: "modifybatch task queue title/role/mode",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
				"title": map[string]interface{}{
					"type": "string",
					"description": "title(cleartitle)",
				},
				"role": map[string]interface{}{
					"type": "string",
					"description": "role(usedefaultrole)",
				},
				"agent_mode": map[string]interface{}{
					"type": "string",
					"description": "mode:single/eino_single/deep/plan_execute/supervisor;multi is deep",
					"enum": []string{"single", "eino_single", "deep", "plan_execute", "supervisor", "multi"},
				},
			},
			"required": []string{"queue_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		title := mcpArgString(args, "title")
		role := mcpArgString(args, "role")
		agentMode := mcpArgString(args, "agent_mode")
		if err := h.batchTaskManager.UpdateQueueMetadata(qid, title, role, agentMode); err != nil {
			return batchMCPTextResult(err.Error(), true), nil
		}
		updated, _ := h.batchTaskManager.GetBatchQueue(qid)
		logger.Info("MCP batch_task_update_metadata", zap.String("queueId", qid))
		return batchMCPJSONResult(updated)
	})

	// --- update schedule ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskUpdateSchedule,
		Description: `modifybatch task queueofand Cron expression.queue running statusmodify.
schedule_mode is cron cron_expr;is manual clear Cron configure.`,
		ShortDescription: "modifybatch taskconfigure(Cron expression)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
				"schedule_mode": map[string]interface{}{
					"type": "string",
					"description": "manual or cron",
					"enum": []string{"manual", "cron"},
				},
				"cron_expr": map[string]interface{}{
					"type": "string",
					"description": "Cron expression(schedule_mode is cron ). 5 Format: minutes hours ,such as \"0 */6 * * *\"(each/per6 hours)/\"30 2 * * 1-5\"(2:30)",
				},
			},
			"required": []string{"queue_id", "schedule_mode"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		queue, exists := h.batchTaskManager.GetBatchQueue(qid)
		if !exists {
			return batchMCPTextResult("queue does not exist: "+qid, true), nil
		}
		if queue.Status == "running" {
			return batchMCPTextResult("queuein,modifyconfigure", true), nil
		}
		scheduleMode := normalizeBatchQueueScheduleMode(mcpArgString(args, "schedule_mode"))
		cronExpr := strings.TrimSpace(mcpArgString(args, "cron_expr"))
		var nextRunAt *time.Time
		if scheduleMode == "cron" {
			if cronExpr == "" {
				return batchMCPTextResult("Cron mode cron_expr cannot be empty", true), nil
			}
			sch, err := h.batchCronParser.Parse(cronExpr)
			if err != nil {
				return batchMCPTextResult("invalid cron expression: "+err.Error(), true), nil
			}
			n := sch.Next(time.Now())
			nextRunAt = &n
		}
		h.batchTaskManager.UpdateQueueSchedule(qid, scheduleMode, cronExpr, nextRunAt)
		updated, _ := h.batchTaskManager.GetBatchQueue(qid)
		logger.Info("MCP batch_task_update_schedule", zap.String("queueId", qid), zap.String("scheduleMode", scheduleMode), zap.String("cronExpr", cronExpr))
		return batchMCPJSONResult(updated)
	})

	// --- schedule enabled ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskScheduleEnabled,
		Description: ` Cron triggerqueue. Cron expression,stop;「start」execute.
 schedule_mode is cron ofqueue.`,
		ShortDescription: "batch task Cron ",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
				"schedule_enabled": map[string]interface{}{
					"type": "boolean",
					"description": "true trigger,false execute",
				},
			},
			"required": []string{"queue_id", "schedule_enabled"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		if qid == "" {
			return batchMCPTextResult("queue_id cannot be empty", true), nil
		}
		en, ok := mcpArgBool(args, "schedule_enabled")
		if !ok {
			return batchMCPTextResult("schedule_enabled is", true), nil
		}
		if _, exists := h.batchTaskManager.GetBatchQueue(qid); !exists {
			return batchMCPTextResult("queue does not exist", true), nil
		}
		if !h.batchTaskManager.SetScheduleEnabled(qid, en) {
			return batchMCPTextResult("updatefailed", true), nil
		}
		queue, _ := h.batchTaskManager.GetBatchQueue(qid)
		logger.Info("MCP batch_task_schedule_enabled", zap.String("queueId", qid), zap.Bool("enabled", en))
		return batchMCPJSONResult(queue)
	})

	// --- add task ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskAdd,
		Description: " pending statusofqueuetask.",
		ShortDescription: "batchqueuetask",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
				"message": map[string]interface{}{
					"type": "string",
					"description": "task",
				},
			},
			"required": []string{"queue_id", "message"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		msg := strings.TrimSpace(mcpArgString(args, "message"))
		if qid == "" || msg == "" {
			return batchMCPTextResult("queue_id and message cannot be empty", true), nil
		}
		task, err := h.batchTaskManager.AddTaskToQueue(qid, msg)
		if err != nil {
			return batchMCPTextResult(err.Error(), true), nil
		}
		queue, _ := h.batchTaskManager.GetBatchQueue(qid)
		logger.Info("MCP batch_task_add_task", zap.String("queueId", qid), zap.String("taskId", task.ID))
		return batchMCPJSONResult(map[string]interface{}{"task": task, "queue": queue})
	})

	// --- update task ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskUpdate,
		Description: "modify pending queueinis pending oftask.",
		ShortDescription: "updatebatchtask",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
				"task_id": map[string]interface{}{
					"type": "string",
					"description": "subtask ID",
				},
				"message": map[string]interface{}{
					"type": "string",
					"description": "oftask",
				},
			},
			"required": []string{"queue_id", "task_id", "message"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		tid := mcpArgString(args, "task_id")
		msg := strings.TrimSpace(mcpArgString(args, "message"))
		if qid == "" || tid == "" || msg == "" {
			return batchMCPTextResult("queue_id/task_id/message cannot be empty", true), nil
		}
		if err := h.batchTaskManager.UpdateTaskMessage(qid, tid, msg); err != nil {
			return batchMCPTextResult(err.Error(), true), nil
		}
		queue, _ := h.batchTaskManager.GetBatchQueue(qid)
		logger.Info("MCP batch_task_update_task", zap.String("queueId", qid), zap.String("taskId", tid))
		return batchMCPJSONResult(queue)
	})

	// --- remove task ---
	reg(mcp.Tool{
		Name: builtin.ToolBatchTaskRemove,
		Description: "from pending queueindeleteis pending oftask.",
		ShortDescription: "deletebatchtask",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"queue_id": map[string]interface{}{
					"type": "string",
					"description": "queue ID",
				},
				"task_id": map[string]interface{}{
					"type": "string",
					"description": "subtask ID",
				},
			},
			"required": []string{"queue_id", "task_id"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		qid := mcpArgString(args, "queue_id")
		tid := mcpArgString(args, "task_id")
		if qid == "" || tid == "" {
			return batchMCPTextResult("queue_id and task_id cannot be empty", true), nil
		}
		if err := h.batchTaskManager.DeleteTask(qid, tid); err != nil {
			return batchMCPTextResult(err.Error(), true), nil
		}
		queue, _ := h.batchTaskManager.GetBatchQueue(qid)
		logger.Info("MCP batch_task_remove_task", zap.String("queueId", qid), zap.String("taskId", tid))
		return batchMCPJSONResult(queue)
	})

	logger.Info("batch task MCP toolhasregister", zap.Int("count", 12))
}

const mcpBatchListTaskMessageMaxRunes = 160
type batchTaskMCPListSummary struct {
	ID string `json:"id"`
	Status string `json:"status"`
	Message string `json:"message,omitempty"`
}
type batchTaskQueueMCPListItem struct {
	ID string `json:"id"`
	Title string `json:"title,omitempty"`
	Role string `json:"role,omitempty"`
	AgentMode string `json:"agentMode"`
	ScheduleMode string `json:"scheduleMode"`
	CronExpr string `json:"cronExpr,omitempty"`
	NextRunAt *time.Time `json:"nextRunAt,omitempty"`
	ScheduleEnabled bool `json:"scheduleEnabled"`
	LastScheduleTriggerAt *time.Time `json:"lastScheduleTriggerAt,omitempty"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	CurrentIndex int `json:"currentIndex"`
	TaskTotal int `json:"task_total"`
	TaskCounts map[string]int `json:"task_counts"`
	Tasks []batchTaskMCPListSummary `json:"tasks"`
}

func truncateStringRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	n := 0
	for i := range s {
		if n == maxRunes {
			out := strings.TrimSpace(s[:i])
			if out == "" {
				return "…"
			}
			return out + "…"
		}
		n++
	}
	return s
}

const mcpBatchListMaxTasksPerQueue = 200

func toBatchTaskQueueMCPListItem(q *BatchTaskQueue) batchTaskQueueMCPListItem {
	counts := map[string]int{
		"pending": 0,
		"running": 0,
		"completed": 0,
		"failed": 0,
		"cancelled": 0,
	}
	tasks := make([]batchTaskMCPListSummary, 0, len(q.Tasks))
	for _, t := range q.Tasks {
		if t == nil {
			continue
		}
		counts[t.Status]++
		if len(tasks) < mcpBatchListMaxTasksPerQueue {
			tasks = append(tasks, batchTaskMCPListSummary{
				ID: t.ID,
				Status: t.Status,
				Message: truncateStringRunes(t.Message, mcpBatchListTaskMessageMaxRunes),
			})
		}
	}
	return batchTaskQueueMCPListItem{
		ID: q.ID,
		Title: q.Title,
		Role: q.Role,
		AgentMode: q.AgentMode,
		ScheduleMode: q.ScheduleMode,
		CronExpr: q.CronExpr,
		NextRunAt: q.NextRunAt,
		ScheduleEnabled: q.ScheduleEnabled,
		LastScheduleTriggerAt: q.LastScheduleTriggerAt,
		Status: q.Status,
		CreatedAt: q.CreatedAt,
		StartedAt: q.StartedAt,
		CompletedAt: q.CompletedAt,
		CurrentIndex: q.CurrentIndex,
		TaskTotal: len(tasks),
		TaskCounts: counts,
		Tasks: tasks,
	}
}

func batchMCPTextResult(text string, isErr bool) *mcp.ToolResult {
	return &mcp.ToolResult{
		Content: []mcp.Content{{Type: "text", Text: text}},
		IsError: isErr,
	}
}

func batchMCPJSONResult(v interface{}) (*mcp.ToolResult, error) {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return batchMCPTextResult(fmt.Sprintf("JSON failed: %v", err), true), nil
	}
	return &mcp.ToolResult{Content: []mcp.Content{{Type: "text", Text: string(b)}}}, nil
}

func batchMCPTasksFromArgs(args map[string]interface{}) ([]string, string) {
	if raw, ok := args["tasks"]; ok && raw != nil {
		switch t := raw.(type) {
		case []interface{}:
			out := make([]string, 0, len(t))
			for _, x := range t {
				if s, ok := x.(string); ok {
					if tr := strings.TrimSpace(s); tr != "" {
						out = append(out, tr)
					}
				}
			}
			if len(out) > 0 {
				return out, ""
			}
		}
	}
	if txt := mcpArgString(args, "tasks_text"); txt != "" {
		lines := strings.Split(txt, "\n")
		out := make([]string, 0, len(lines))
		for _, line := range lines {
			if tr := strings.TrimSpace(line); tr != "" {
				out = append(out, tr)
			}
		}
		if len(out) > 0 {
			return out, ""
		}
	}
	return nil, "need tasks()or tasks_text(,each/pertask)"
}

func mcpArgString(args map[string]interface{}, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(t, 'f', -1, 64))
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func mcpArgFloat(args map[string]interface{}, key string) float64 {
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}

func mcpArgBool(args map[string]interface{}, key string) (val bool, ok bool) {
	v, exists := args[key]
	if !exists {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "true" || s == "1" || s == "yes" {
			return true, true
		}
		if s == "false" || s == "0" || s == "no" {
			return false, true
		}
	case float64:
		return t != 0, true
	}
	return false, false
}
