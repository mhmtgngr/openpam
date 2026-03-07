package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/auth"
	"github.com/rs/zerolog"
)

// RoleHandler handles role management endpoints.
type RoleHandler struct {
	svc    *auth.RoleService
	logger zerolog.Logger
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(svc *auth.RoleService, logger zerolog.Logger) *RoleHandler {
	return &RoleHandler{svc: svc, logger: logger}
}

// RegisterRoutes registers role management routes on the given router group.
func (h *RoleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/roles", h.List)
	rg.POST("/roles", h.Create)
	rg.GET("/roles/:id", h.Get)
	rg.PUT("/roles/:id", h.Update)
	rg.DELETE("/roles/:id", h.Delete)
	rg.GET("/roles/:id/permissions", h.GetPermissions)
	rg.POST("/roles/:id/permissions", h.GrantPermission)
	rg.DELETE("/roles/:id/permissions/:permission_id", h.RevokePermission)

	rg.GET("/users/:user_id/roles", h.GetUserRoles)
	rg.POST("/users/:user_id/roles/:role_id", h.AssignUserRole)
	rg.DELETE("/users/:user_id/roles/:role_id", h.RevokeUserRole)
}

func (h *RoleHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_TENANT", "Invalid tenant ID"))
		return
	}

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	roles, err := h.svc.GetRepo().List(c.Request.Context(), tenantIDUUID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list roles")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to list roles"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req auth.Role
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	if err := validateRoleRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	req.Name = sanitizeInput(req.Name)
	req.DisplayName = sanitizeInput(req.DisplayName)
	req.Description = sanitizeInput(req.Description)

	authCtx := extractAuthContext(c)

	if req.TenantID != authCtx.TenantID {
		c.JSON(http.StatusForbidden, errorResponse("FORBIDDEN", "Cannot create role for different tenant"))
		return
	}

	if err := h.svc.CreateRole(c.Request.Context(), &req, authCtx); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create role")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to create role"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"role": req})
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	role, err := h.svc.GetRepo().GetByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get role")
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "Role not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"role": role})
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	var req auth.Role
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	authCtx := extractAuthContext(c)
	req.ID = id

	if err := h.svc.UpdateRole(c.Request.Context(), &req, authCtx); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update role")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to update role"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"role": req})
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	authCtx := extractAuthContext(c)

	if err := h.svc.DeleteRole(c.Request.Context(), id, authCtx); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete role")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to delete role"))
		return
	}

	h.logger.Info().Str("role_id", id.String()).Msg("Role deleted")
	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}

func (h *RoleHandler) GetPermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	permissions, err := h.svc.GetRepo().GetPermissions(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get role permissions")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to get permissions"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func (h *RoleHandler) GrantPermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	var req struct {
		Resource string `json:"resource" binding:"required"`
		Action   string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	authCtx := extractAuthContext(c)

	if err := h.svc.AssignPermission(c.Request.Context(), id, req.Resource, req.Action, authCtx); err != nil {
		h.logger.Error().Err(err).Msg("Failed to grant permission")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to grant permission"))
		return
	}

	h.logger.Info().Str("role_id", id.String()).Str("resource", req.Resource).Str("action", req.Action).Msg("Permission granted")
	c.JSON(http.StatusOK, gin.H{"message": "Permission granted"})
}

func (h *RoleHandler) RevokePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	permissionID, err := uuid.Parse(c.Param("permission_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid permission ID"))
		return
	}

	if err := h.svc.GetRepo().RevokePermission(c.Request.Context(), roleID, permissionID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to revoke permission")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to revoke permission"))
		return
	}

	h.logger.Info().Str("role_id", roleID.String()).Str("permission_id", permissionID.String()).Msg("Permission revoked")
	c.JSON(http.StatusOK, gin.H{"message": "Permission revoked"})
}

func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid user ID"))
		return
	}

	roles, err := h.svc.GetRepo().GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get user roles")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to get roles"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func (h *RoleHandler) AssignUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid user ID"))
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	assignedBy, _ := c.Get("user_id")
	assignedByID, _ := uuid.Parse(assignedBy.(string))

	if err := h.svc.AssignUserToRole(c.Request.Context(), userID, roleID, assignedByID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to assign role")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to assign role"))
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Str("role_id", roleID.String()).Msg("Role assigned to user")
	c.JSON(http.StatusOK, gin.H{"message": "Role assigned"})
}

func (h *RoleHandler) RevokeUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid user ID"))
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid role ID"))
		return
	}

	if err := h.svc.RevokeRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to revoke role")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to revoke role"))
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Str("role_id", roleID.String()).Msg("Role revoked from user")
	c.JSON(http.StatusOK, gin.H{"message": "Role revoked"})
}

// ListPermissions returns all available permissions (super_admin only).
func ListPermissions(svc *auth.RoleService, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		permissions, err := svc.GetRepo().ListAllPermissions(c.Request.Context())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to list permissions")
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to list permissions"))
			return
		}
		c.JSON(http.StatusOK, gin.H{"permissions": permissions})
	}
}

func validateRoleRequest(req auth.Role) error {
	if req.Name == "" {
		return fmt.Errorf("name: required")
	}
	if !validateUsername(req.Name) {
		return fmt.Errorf("name: must be alphanumeric with underscore/hyphen, 3-64 characters")
	}
	if req.DisplayName == "" {
		return fmt.Errorf("display_name: required")
	}
	if len(req.DisplayName) > 255 {
		return fmt.Errorf("display_name: too long (max 255 characters)")
	}
	return nil
}

func extractAuthContext(c *gin.Context) auth.AuthorizationContext {
	userIDStr, _ := c.Get("user_id")
	tenantIDStr, _ := c.Get("tenant_id")
	userID, _ := uuid.Parse(userIDStr.(string))
	tenantID, _ := uuid.Parse(tenantIDStr.(string))
	return auth.AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
	}
}
