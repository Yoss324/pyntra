//go:build windows

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *TerminalHandler) RunCommandWS(c *gin.Context) {
	// WebSocket terminal not supported on Windows in this build
	c.AbortWithStatus(http.StatusNotImplemented)
}
