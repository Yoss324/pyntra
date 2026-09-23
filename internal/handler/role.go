package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"pyntra/internal/config"

	"gopkg.in/yaml.v3"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RoleHandler roleprocessor
type RoleHandler struct {
	config *config.Config
	configPath string
	logger *zap.Logger
	skillsManager SkillsManager // Skillsmanagedevice/processorinterface(optional)
}

// SkillsManager Skillsmanagedevice/processorinterface
type SkillsManager interface {
	ListSkills() ([]string, error)
}
func NewRoleHandler(cfg *config.Config, configPath string, logger *zap.Logger) *RoleHandler {
	return &RoleHandler{
		config: cfg,
		configPath: configPath,
		logger: logger,
	}
}
func (h *RoleHandler) SetSkillsManager(manager SkillsManager) {
	h.skillsManager = manager
}
func (h *RoleHandler) GetSkills(c *gin.Context) {
	if h.skillsManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"skills": []string{},
		})
		return
	}

	skills, err := h.skillsManager.ListSkills()
	if err != nil {
		h.logger.Warn("getskillslistfailed", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"skills": []string{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"skills": skills,
	})
}
func (h *RoleHandler) GetRoles(c *gin.Context) {
	if h.config.Roles == nil {
		h.config.Roles = make(map[string]config.RoleConfig)
	}

	roles := make([]config.RoleConfig, 0, len(h.config.Roles))
	for key, role := range h.config.Roles {
		if role.Name == "" {
			role.Name = key
		}
		roles = append(roles, role)
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
	})
}
func (h *RoleHandler) GetRole(c *gin.Context) {
	roleName := c.Param("name")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role namecannot be empty"})
		return
	}

	if h.config.Roles == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "roledoes not exist"})
		return
	}

	role, exists := h.config.Roles[roleName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "roledoes not exist"})
		return
	}
	if role.Name == "" {
		role.Name = roleName
	}

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// UpdateRole updaterole
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	roleName := c.Param("name")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role namecannot be empty"})
		return
	}

	var req config.RoleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}
	if req.Name == "" {
		req.Name = roleName
	}
	if h.config.Roles == nil {
		h.config.Roles = make(map[string]config.RoleConfig)
	}
	finalKey := req.Name
	keysToDelete := make([]string, 0)
	for key := range h.config.Roles {
		if key != finalKey {
			role := h.config.Roles[key]
			if role.Name == "" {
				role.Name = key
			}
			if role.Name == req.Name {
				keysToDelete = append(keysToDelete, key)
			}
		}
	}
	for _, key := range keysToDelete {
		delete(h.config.Roles, key)
		h.logger.Info("deleteofrole", zap.String("oldKey", key), zap.String("name", req.Name))
	}
	if roleName != finalKey {
		delete(h.config.Roles, roleName)
	}
	if roleName != finalKey {
		configDir := filepath.Dir(h.configPath)
		rolesDir := h.config.RolesDir
		if rolesDir == "" {
			rolesDir = "roles" // defaultdirectory
		}
		if !filepath.IsAbs(rolesDir) {
			rolesDir = filepath.Join(configDir, rolesDir)
		}
		oldSafeFileName := sanitizeFileName(roleName)
		oldRoleFileYaml := filepath.Join(rolesDir, oldSafeFileName+".yaml")
		oldRoleFileYml := filepath.Join(rolesDir, oldSafeFileName+".yml")

		if _, err := os.Stat(oldRoleFileYaml); err == nil {
			if err := os.Remove(oldRoleFileYaml); err != nil {
				h.logger.Warn("deleteroleconfigurefilefailed", zap.String("file", oldRoleFileYaml), zap.Error(err))
			}
		}
		if _, err := os.Stat(oldRoleFileYml); err == nil {
			if err := os.Remove(oldRoleFileYml); err != nil {
				h.logger.Warn("deleteroleconfigurefilefailed", zap.String("file", oldRoleFileYml), zap.Error(err))
			}
		}
	}
	h.config.Roles[finalKey] = req

	// saveconfigureto file
	if err := h.saveConfig(); err != nil {
		h.logger.Error("saveconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "saveconfigurefailed: " + err.Error()})
		return
	}

	h.logger.Info("updaterole", zap.String("oldKey", roleName), zap.String("newKey", finalKey), zap.String("name", req.Name))
	c.JSON(http.StatusOK, gin.H{
		"message": "rolehasupdate",
		"role": req,
	})
}
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req config.RoleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role namecannot be empty"})
		return
	}
	if h.config.Roles == nil {
		h.config.Roles = make(map[string]config.RoleConfig)
	}
	if _, exists := h.config.Roles[req.Name]; exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rolealready exists"})
		return
	}

	// createrole(defaultenable)
	if !req.Enabled {
		req.Enabled = true
	}

	h.config.Roles[req.Name] = req

	// saveconfigureto file
	if err := h.saveConfig(); err != nil {
		h.logger.Error("saveconfigurefailed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "saveconfigurefailed: " + err.Error()})
		return
	}

	h.logger.Info("createrole", zap.String("roleName", req.Name))
	c.JSON(http.StatusOK, gin.H{
		"message": "rolehascreate",
		"role": req,
	})
}

// DeleteRole deleterole
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	roleName := c.Param("name")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role namecannot be empty"})
		return
	}

	if h.config.Roles == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "roledoes not exist"})
		return
	}

	if _, exists := h.config.Roles[roleName]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "roledoes not exist"})
		return
	}
	if roleName == "default" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deletedefaultrole"})
		return
	}

	delete(h.config.Roles, roleName)
	configDir := filepath.Dir(h.configPath)
	rolesDir := h.config.RolesDir
	if rolesDir == "" {
		rolesDir = "roles" // defaultdirectory
	}
	if !filepath.IsAbs(rolesDir) {
		rolesDir = filepath.Join(configDir, rolesDir)
	}
	safeFileName := sanitizeFileName(roleName)
	roleFileYaml := filepath.Join(rolesDir, safeFileName+".yaml")
	roleFileYml := filepath.Join(rolesDir, safeFileName+".yml")

	// delete .yaml file(if exists)
	if _, err := os.Stat(roleFileYaml); err == nil {
		if err := os.Remove(roleFileYaml); err != nil {
			h.logger.Warn("deleteroleconfigurefilefailed", zap.String("file", roleFileYaml), zap.Error(err))
		} else {
			h.logger.Info("hasdeleteroleconfigurefile", zap.String("file", roleFileYaml))
		}
	}

	// delete .yml file(if exists)
	if _, err := os.Stat(roleFileYml); err == nil {
		if err := os.Remove(roleFileYml); err != nil {
			h.logger.Warn("deleteroleconfigurefilefailed", zap.String("file", roleFileYml), zap.Error(err))
		} else {
			h.logger.Info("hasdeleteroleconfigurefile", zap.String("file", roleFileYml))
		}
	}

	h.logger.Info("deleterole", zap.String("roleName", roleName))
	c.JSON(http.StatusOK, gin.H{
		"message": "rolehasdelete",
	})
}
func (h *RoleHandler) saveConfig() error {
	configDir := filepath.Dir(h.configPath)
	rolesDir := h.config.RolesDir
	if rolesDir == "" {
		rolesDir = "roles" // defaultdirectory
	}
	if !filepath.IsAbs(rolesDir) {
		rolesDir = filepath.Join(configDir, rolesDir)
	}
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		return fmt.Errorf("createroledirectoryfailed: %w", err)
	}
	if h.config.Roles != nil {
		for roleName, role := range h.config.Roles {
			if role.Name == "" {
				role.Name = roleName
			}
			safeFileName := sanitizeFileName(role.Name)
			roleFile := filepath.Join(rolesDir, safeFileName+".yaml")
			roleData, err := yaml.Marshal(&role)
			if err != nil {
				h.logger.Error("failed to serialize role configuration", zap.String("role", roleName), zap.Error(err))
				continue
			}
			roleDataStr := string(roleData)
			if role.Icon != "" && strings.HasPrefix(role.Icon, "\\U") {
				re := regexp.MustCompile(`(?m)^(icon:\s+)(\\U[0-9A-F]{8})(\s*)$`)
				roleDataStr = re.ReplaceAllString(roleDataStr, `${1}"${2}"${3}`)
				roleData = []byte(roleDataStr)
			}

			// write file
			if err := os.WriteFile(roleFile, roleData, 0644); err != nil {
				h.logger.Error("saveroleconfigurefilefailed", zap.String("role", roleName), zap.String("file", roleFile), zap.Error(err))
				continue
			}

			h.logger.Info("roleconfigurehassaveto file", zap.String("role", roleName), zap.String("file", roleFile))
		}
	}

	return nil
}
func sanitizeFileName(name string) string {
	replacer := map[rune]string{
		'/': "_",
		'\\': "_",
		':': "_",
		'*': "_",
		'?': "_",
		'"': "_",
		'<': "_",
		'>': "_",
		'|': "_",
		' ': "_",
	}

	var result []rune
	for _, r := range name {
		if replacement, ok := replacer[r]; ok {
			result = append(result, []rune(replacement)...)
		} else {
			result = append(result, r)
		}
	}

	fileName := string(result)
	if fileName == "" {
		fileName = "role"
	}

	return fileName
}

// updateRolesConfig updateroleconfigure
func updateRolesConfig(doc *yaml.Node, cfg config.RolesConfig) {
	root := doc.Content[0]
	rolesNode := ensureMap(root, "roles")
	if rolesNode.Kind == yaml.MappingNode {
		rolesNode.Content = nil
	}
	if cfg.Roles != nil {
		rolesByName := make(map[string]config.RoleConfig)
		for roleKey, role := range cfg.Roles {
			if role.Name == "" {
				role.Name = roleKey
			}
			rolesByName[role.Name] = role
		}
		for roleName, role := range rolesByName {
			roleNode := ensureMap(rolesNode, roleName)
			setStringInMap(roleNode, "name", role.Name)
			setStringInMap(roleNode, "description", role.Description)
			setStringInMap(roleNode, "user_prompt", role.UserPrompt)
			if role.Icon != "" {
				setStringInMap(roleNode, "icon", role.Icon)
			}
			setBoolInMap(roleNode, "enabled", role.Enabled)
			if len(role.Tools) > 0 {
				toolsNode := ensureArray(roleNode, "tools")
				toolsNode.Content = nil
				for _, toolKey := range role.Tools {
					toolNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: toolKey}
					toolsNode.Content = append(toolsNode.Content, toolNode)
				}
			} else if len(role.MCPs) > 0 {
				mcpsNode := ensureArray(roleNode, "mcps")
				mcpsNode.Content = nil
				for _, mcpName := range role.MCPs {
					mcpNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: mcpName}
					mcpsNode.Content = append(mcpsNode.Content, mcpNode)
				}
			}
		}
	}
}
func ensureArray(parent *yaml.Node, key string) *yaml.Node {
	_, valueNode := ensureKeyValue(parent, key)
	if valueNode.Kind != yaml.SequenceNode {
		valueNode.Kind = yaml.SequenceNode
		valueNode.Tag = "!!seq"
		valueNode.Content = nil
	}
	return valueNode
}
