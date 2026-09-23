package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"pyntra/internal/attackchain"
	"pyntra/internal/config"
	"pyntra/internal/database"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AttackChainHandler attack chainprocessor
type AttackChainHandler struct {
	db *database.DB
	logger *zap.Logger
	openAIConfig *config.OpenAIConfig
	mu sync.RWMutex
	generatingLocks sync.Map // map[string]*sync.Mutex
}
func NewAttackChainHandler(db *database.DB, openAIConfig *config.OpenAIConfig, logger *zap.Logger) *AttackChainHandler {
	return &AttackChainHandler{
		db: db,
		logger: logger,
		openAIConfig: openAIConfig,
	}
}

// UpdateConfig updateOpenAIconfigure
func (h *AttackChainHandler) UpdateConfig(cfg *config.OpenAIConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.openAIConfig = cfg
	h.logger.Info("AttackChainHandlerconfigurehasupdate",
		zap.String("base_url", cfg.BaseURL),
		zap.String("model", cfg.Model),
	)
}
func (h *AttackChainHandler) getOpenAIConfig() *config.OpenAIConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.openAIConfig
}
// GET /api/attack-chain/:conversationId
func (h *AttackChainHandler) GetAttackChain(c *gin.Context) {
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversationId is required"})
		return
	}
	_, err := h.db.GetConversation(conversationID)
	if err != nil {
		h.logger.Warn("conversation does not exist", zap.String("conversationId", conversationID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation does not exist"})
		return
	}
	openAIConfig := h.getOpenAIConfig()
	builder := attackchain.NewBuilder(h.db, openAIConfig, h.logger)
	chain, err := builder.LoadChainFromDatabase(conversationID)
	if err == nil && len(chain.Nodes) > 0 {
		h.logger.Info("returnalready existsofattack chain", zap.String("conversationId", conversationID))
		c.JSON(http.StatusOK, chain)
		return
	}
	lockInterface, _ := h.generatingLocks.LoadOrStore(conversationID, &sync.Mutex{})
	lock := lockInterface.(*sync.Mutex)
	acquired := lock.TryLock()
	if !acquired {
		h.logger.Info("attack chain is being generated. Please try again later", zap.String("conversationId", conversationID))
		c.JSON(http.StatusConflict, gin.H{"error": "attack chain is being generated. Please try again later"})
		return
	}
	defer lock.Unlock()
	chain, err = builder.LoadChainFromDatabase(conversationID)
	if err == nil && len(chain.Nodes) > 0 {
		h.logger.Info("returnalready existsofattack chain(hasgenerate)", zap.String("conversationId", conversationID))
		c.JSON(http.StatusOK, chain)
		return
	}

	h.logger.Info("startgenerateattack chain", zap.String("conversationId", conversationID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	chain, err = builder.BuildChainFromConversation(ctx, conversationID)
	if err != nil {
		h.logger.Error("failed to generate attack chain", zap.String("conversationId", conversationID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate attack chain: " + err.Error()})
		return
	}
	// h.generatingLocks.Delete(conversationID)

	c.JSON(http.StatusOK, chain)
}
// POST /api/attack-chain/:conversationId/regenerate
func (h *AttackChainHandler) RegenerateAttackChain(c *gin.Context) {
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversationId is required"})
		return
	}
	_, err := h.db.GetConversation(conversationID)
	if err != nil {
		h.logger.Warn("conversation does not exist", zap.String("conversationId", conversationID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation does not exist"})
		return
	}
	if err := h.db.DeleteAttackChain(conversationID); err != nil {
		h.logger.Warn("deleteattack chainfailed", zap.Error(err))
	}
	lockInterface, _ := h.generatingLocks.LoadOrStore(conversationID, &sync.Mutex{})
	lock := lockInterface.(*sync.Mutex)
	
	acquired := lock.TryLock()
	if !acquired {
		h.logger.Info("attack chain is being generated. Please try again later", zap.String("conversationId", conversationID))
		c.JSON(http.StatusConflict, gin.H{"error": "attack chain is being generated. Please try again later"})
		return
	}
	defer lock.Unlock()
	h.logger.Info("generateattack chain", zap.String("conversationId", conversationID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	openAIConfig := h.getOpenAIConfig()
	builder := attackchain.NewBuilder(h.db, openAIConfig, h.logger)
	chain, err := builder.BuildChainFromConversation(ctx, conversationID)
	if err != nil {
		h.logger.Error("failed to generate attack chain", zap.String("conversationId", conversationID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate attack chain: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, chain)
}

