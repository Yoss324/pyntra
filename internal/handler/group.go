package handler

import (
	"net/http"
	"time"

	"pyntra/internal/database"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GroupHandler groupprocessor
type GroupHandler struct {
	db *database.DB
	logger *zap.Logger
}
func NewGroupHandler(db *database.DB, logger *zap.Logger) *GroupHandler {
	return &GroupHandler{
		db: db,
		logger: logger,
	}
}

// CreateGroupRequest creategrouprequest
type CreateGroupRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// CreateGroup creategroup
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group namecannot be empty"})
		return
	}

	group, err := h.db.CreateGroup(req.Name, req.Icon)
	if err != nil {
		h.logger.Error("creategroupfailed", zap.Error(err))
		if err.Error() == "group namealready exists" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group namealready exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}
func (h *GroupHandler) ListGroups(c *gin.Context) {
	groups, err := h.db.ListGroups()
	if err != nil {
		h.logger.Error("getgrouplistfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// GetGroup getgroup
func (h *GroupHandler) GetGroup(c *gin.Context) {
	id := c.Param("id")

	group, err := h.db.GetGroup(id)
	if err != nil {
		h.logger.Error("getgroupfailed", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "group does not exist"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// UpdateGroupRequest updategrouprequest
type UpdateGroupRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// UpdateGroup updategroup
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group namecannot be empty"})
		return
	}

	if err := h.db.UpdateGroup(id, req.Name, req.Icon); err != nil {
		h.logger.Error("updategroupfailed", zap.Error(err))
		if err.Error() == "group namealready exists" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group namealready exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	group, err := h.db.GetGroup(id)
	if err != nil {
		h.logger.Error("getupdateofgroupfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteGroup deletegroup
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteGroup(id); err != nil {
		h.logger.Error("deletegroupfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deletesuccessful"})
}
type AddConversationToGroupRequest struct {
	ConversationID string `json:"conversationId"`
	GroupID string `json:"groupId"`
}
func (h *GroupHandler) AddConversationToGroup(c *gin.Context) {
	var req AddConversationToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.AddConversationToGroup(req.ConversationID, req.GroupID); err != nil {
		h.logger.Error("conversationgroupfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successful"})
}
func (h *GroupHandler) RemoveConversationFromGroup(c *gin.Context) {
	conversationID := c.Param("conversationId")
	groupID := c.Param("id")

	if err := h.db.RemoveConversationFromGroup(conversationID, groupID); err != nil {
		h.logger.Error("fromgroupinconversationfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successful"})
}
type GroupConversation struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Pinned bool `json:"pinned"`
	GroupPinned bool `json:"groupPinned"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
func (h *GroupHandler) GetGroupConversations(c *gin.Context) {
	groupID := c.Param("id")
	searchQuery := c.Query("search") // getsearchparameter

	var conversations []*database.Conversation
	var err error
	if searchQuery != "" {
		conversations, err = h.db.SearchConversationsByGroup(groupID, searchQuery)
	} else {
		conversations, err = h.db.GetConversationsByGroup(groupID)
	}

	if err != nil {
		h.logger.Error("getgroupconversationfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	groupConvs := make([]GroupConversation, 0, len(conversations))
	for _, conv := range conversations {
		var groupPinned int
		err := h.db.QueryRow(
			"SELECT COALESCE(pinned, 0) FROM conversation_group_mappings WHERE conversation_id = ? AND group_id = ?",
			conv.ID, groupID,
		).Scan(&groupPinned)
		if err != nil {
			h.logger.Warn("querygrouppinstatusfailed", zap.String("conversationId", conv.ID), zap.Error(err))
			groupPinned = 0
		}

		groupConvs = append(groupConvs, GroupConversation{
			ID: conv.ID,
			Title: conv.Title,
			Pinned: conv.Pinned,
			GroupPinned: groupPinned != 0,
			CreatedAt: conv.CreatedAt,
			UpdatedAt: conv.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, groupConvs)
}
func (h *GroupHandler) GetAllMappings(c *gin.Context) {
	mappings, err := h.db.GetAllGroupMappings()
	if err != nil {
		h.logger.Error("getgroupfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mappings)
}
type UpdateConversationPinnedRequest struct {
	Pinned bool `json:"pinned"`
}
func (h *GroupHandler) UpdateConversationPinned(c *gin.Context) {
	conversationID := c.Param("id")

	var req UpdateConversationPinnedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.UpdateConversationPinned(conversationID, req.Pinned); err != nil {
		h.logger.Error("morenew conversationpinstatusfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updatesuccessful"})
}
type UpdateGroupPinnedRequest struct {
	Pinned bool `json:"pinned"`
}
func (h *GroupHandler) UpdateGroupPinned(c *gin.Context) {
	groupID := c.Param("id")

	var req UpdateGroupPinnedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.UpdateGroupPinned(groupID, req.Pinned); err != nil {
		h.logger.Error("updategrouppinstatusfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updatesuccessful"})
}
type UpdateConversationPinnedInGroupRequest struct {
	Pinned bool `json:"pinned"`
}
func (h *GroupHandler) UpdateConversationPinnedInGroup(c *gin.Context) {
	groupID := c.Param("id")
	conversationID := c.Param("conversationId")

	var req UpdateConversationPinnedInGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.UpdateConversationPinnedInGroup(conversationID, groupID, req.Pinned); err != nil {
		h.logger.Error("updategroupconversationpinstatusfailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updatesuccessful"})
}
