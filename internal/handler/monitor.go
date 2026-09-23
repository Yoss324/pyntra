package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"pyntra/internal/database"
	"pyntra/internal/mcp"
	"pyntra/internal/security"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MonitorHandler monitorprocessor
type MonitorHandler struct {
	mcpServer *mcp.Server
	externalMCPMgr *mcp.ExternalMCPManager
	executor *security.Executor
	db *database.DB
	logger *zap.Logger
}
func NewMonitorHandler(mcpServer *mcp.Server, executor *security.Executor, db *database.DB, logger *zap.Logger) *MonitorHandler {
	return &MonitorHandler{
		mcpServer: mcpServer,
		externalMCPMgr: nil,
		executor: executor,
		db: db,
		logger: logger,
	}
}
func (h *MonitorHandler) SetExternalMCPManager(mgr *mcp.ExternalMCPManager) {
	h.externalMCPMgr = mgr
}

// MonitorResponse monitorresponse
type MonitorResponse struct {
	Executions []*mcp.ToolExecution `json:"executions"`
	Stats map[string]*mcp.ToolStats `json:"stats"`
	Timestamp time.Time `json:"timestamp"`
	Total int `json:"total,omitempty"`
	Page int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Monitor getmonitorinformation
func (h *MonitorHandler) Monitor(c *gin.Context) {
	// parsepaginationparameter
	page := 1
	pageSize := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// parsestatusfilterparameter
	status := c.Query("status")
	// parsetoolfilterparameter
	toolName := c.Query("tool")

	executions, total := h.loadExecutionsWithPagination(page, pageSize, status, toolName)
	stats := h.loadStats()

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, MonitorResponse{
		Executions: executions,
		Stats: stats,
		Timestamp: time.Now(),
		Total: total,
		Page: page,
		PageSize: pageSize,
		TotalPages: totalPages,
	})
}

func (h *MonitorHandler) loadExecutions() []*mcp.ToolExecution {
	executions, _ := h.loadExecutionsWithPagination(1, 1000, "", "")
	return executions
}

func (h *MonitorHandler) loadExecutionsWithPagination(page, pageSize int, status, toolName string) ([]*mcp.ToolExecution, int) {
	if h.db == nil {
		allExecutions := h.mcpServer.GetAllExecutions()
		if status != "" || toolName != "" {
			filtered := make([]*mcp.ToolExecution, 0)
			for _, exec := range allExecutions {
				matchStatus := status == "" || exec.Status == status
				matchTool := toolName == "" || strings.Contains(strings.ToLower(exec.ToolName), strings.ToLower(toolName))
				if matchStatus && matchTool {
					filtered = append(filtered, exec)
				}
			}
			allExecutions = filtered
		}
		total := len(allExecutions)
		offset := (page - 1) * pageSize
		end := offset + pageSize
		if end > total {
			end = total
		}
		if offset >= total {
			return []*mcp.ToolExecution{}, total
		}
		return allExecutions[offset:end], total
	}

	offset := (page - 1) * pageSize
	executions, err := h.db.LoadToolExecutionsWithPagination(offset, pageSize, status, toolName)
	if err != nil {
		h.logger.Warn("fromdataloadexecution recordfailed,data", zap.Error(err))
		allExecutions := h.mcpServer.GetAllExecutions()
		if status != "" || toolName != "" {
			filtered := make([]*mcp.ToolExecution, 0)
			for _, exec := range allExecutions {
				matchStatus := status == "" || exec.Status == status
				matchTool := toolName == "" || strings.Contains(strings.ToLower(exec.ToolName), strings.ToLower(toolName))
				if matchStatus && matchTool {
					filtered = append(filtered, exec)
				}
			}
			allExecutions = filtered
		}
		total := len(allExecutions)
		offset := (page - 1) * pageSize
		end := offset + pageSize
		if end > total {
			end = total
		}
		if offset >= total {
			return []*mcp.ToolExecution{}, total
		}
		return allExecutions[offset:end], total
	}
	total, err := h.db.CountToolExecutions(status, toolName)
	if err != nil {
		h.logger.Warn("getexecution recordtotalfailed", zap.Error(err))
		total = offset + len(executions)
		if len(executions) == pageSize {
			total = offset + len(executions) + 1
		}
	}

	return executions, total
}

func (h *MonitorHandler) loadStats() map[string]*mcp.ToolStats {
	stats := make(map[string]*mcp.ToolStats)
	if h.db == nil {
		internalStats := h.mcpServer.GetStats()
		for k, v := range internalStats {
			stats[k] = v
		}
	} else {
		dbStats, err := h.db.LoadToolStats()
		if err != nil {
			h.logger.Warn("fromdataloadstatisticsfailed,data", zap.Error(err))
			internalStats := h.mcpServer.GetStats()
			for k, v := range internalStats {
				stats[k] = v
			}
		} else {
			for k, v := range dbStats {
				stats[k] = v
			}
		}
	}

	// mergeexternalMCPmanagedevice/processorofstatistics
	if h.externalMCPMgr != nil {
		externalStats := h.externalMCPMgr.GetToolStats()
		for k, v := range externalStats {
			// ifalready exists,mergestatistics
			if existing, exists := stats[k]; exists {
				existing.TotalCalls += v.TotalCalls
				existing.SuccessCalls += v.SuccessCalls
				existing.FailedCalls += v.FailedCalls
				if v.LastCallTime != nil && (existing.LastCallTime == nil || v.LastCallTime.After(*existing.LastCallTime)) {
					existing.LastCallTime = v.LastCallTime
				}
			} else {
				stats[k] = v
			}
		}
	}

	return stats
}
func (h *MonitorHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	exec, exists := h.mcpServer.GetExecution(id)
	if exists {
		c.JSON(http.StatusOK, exec)
		return
	}
	if h.externalMCPMgr != nil {
		exec, exists = h.externalMCPMgr.GetExecution(id)
		if exists {
			c.JSON(http.StatusOK, exec)
			return
		}
	}
	if h.db != nil {
		exec, err := h.db.GetToolExecution(id)
		if err == nil && exec != nil {
			c.JSON(http.StatusOK, exec)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "execution recordnot found"})
}
func (h *MonitorHandler) BatchGetToolNames(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string, len(req.IDs))
	for _, id := range req.IDs {
		if exec, exists := h.mcpServer.GetExecution(id); exists {
			result[id] = exec.ToolName
			continue
		}
		if h.externalMCPMgr != nil {
			if exec, exists := h.externalMCPMgr.GetExecution(id); exists {
				result[id] = exec.ToolName
				continue
			}
		}
		if h.db != nil {
			if exec, err := h.db.GetToolExecution(id); err == nil && exec != nil {
				result[id] = exec.ToolName
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetStats getstatistics
func (h *MonitorHandler) GetStats(c *gin.Context) {
	stats := h.loadStats()
	c.JSON(http.StatusOK, stats)
}

// DeleteExecution deleteexecution record
func (h *MonitorHandler) DeleteExecution(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution recordIDcannot be empty"})
		return
	}
	if h.db != nil {
		exec, err := h.db.GetToolExecution(id)
		if err != nil {
			h.logger.Warn("execution recorddoes not exist,hasdelete", zap.String("executionId", id), zap.Error(err))
			c.JSON(http.StatusOK, gin.H{"message": "execution recorddoes not existorhasdelete"})
			return
		}

		// deleteexecution record
		err = h.db.DeleteToolExecution(id)
		if err != nil {
			h.logger.Error("deleteexecution recordfailed", zap.Error(err), zap.String("executionId", id))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "deleteexecution recordfailed: " + err.Error()})
			return
		}
		totalCalls := 1
		successCalls := 0
		failedCalls := 0
		if exec.Status == "failed" {
			failedCalls = 1
		} else if exec.Status == "completed" {
			successCalls = 1
		}

		if exec.ToolName != "" {
			if err := h.db.DecreaseToolStats(exec.ToolName, totalCalls, successCalls, failedCalls); err != nil {
				h.logger.Warn("updatestatisticsfailed", zap.Error(err), zap.String("toolName", exec.ToolName))
			}
		}

		h.logger.Info("execution recordhasfromdatadelete", zap.String("executionId", id), zap.String("toolName", exec.ToolName))
		c.JSON(http.StatusOK, gin.H{"message": "execution recordhasdelete"})
		return
	}
	h.logger.Info("deleteinofexecution record", zap.String("executionId", id))
	c.JSON(http.StatusOK, gin.H{"message": "execution recordhasdelete(if exists)"})
}

// DeleteExecutions batchdeleteexecution record
func (h *MonitorHandler) DeleteExecutions(c *gin.Context) {
	var request struct {
		IDs []string `json:"ids"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "requestparameterinvalid: " + err.Error()})
		return
	}

	if len(request.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution recordIDlistcannot be empty"})
		return
	}
	if h.db != nil {
		executions, err := h.db.GetToolExecutionsByIds(request.IDs)
		if err != nil {
			h.logger.Error("getexecution recordfailed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "getexecution recordfailed: " + err.Error()})
			return
		}
		toolStats := make(map[string]struct {
			totalCalls int
			successCalls int
			failedCalls int
		})

		for _, exec := range executions {
			if exec.ToolName == "" {
				continue
			}

			stats := toolStats[exec.ToolName]
			stats.totalCalls++
			if exec.Status == "failed" {
				stats.failedCalls++
			} else if exec.Status == "completed" {
				stats.successCalls++
			}
			toolStats[exec.ToolName] = stats
		}

		// batchdeleteexecution record
		err = h.db.DeleteToolExecutions(request.IDs)
		if err != nil {
			h.logger.Error("batchdeleteexecution recordfailed", zap.Error(err), zap.Int("count", len(request.IDs)))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "batchdeleteexecution recordfailed: " + err.Error()})
			return
		}
		for toolName, stats := range toolStats {
			if err := h.db.DecreaseToolStats(toolName, stats.totalCalls, stats.successCalls, stats.failedCalls); err != nil {
				h.logger.Warn("updatestatisticsfailed", zap.Error(err), zap.String("toolName", toolName))
			}
		}

		h.logger.Info("batchdeleteexecution recordsuccessful", zap.Int("count", len(request.IDs)))
		c.JSON(http.StatusOK, gin.H{"message": "successfuldeleteexecution record", "deleted": len(executions)})
		return
	}
	h.logger.Info("batchdeleteinofexecution record", zap.Int("count", len(request.IDs)))
	c.JSON(http.StatusOK, gin.H{"message": "execution recordhasdelete(if exists)"})
}

