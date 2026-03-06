package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/admin"
	"github.com/rs/zerolog"
)

// TenantHandler handles tenant management endpoints.
type TenantHandler struct {
	svc    *admin.Service
	logger zerolog.Logger
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(svc *admin.Service, logger zerolog.Logger) *TenantHandler {
	return &TenantHandler{svc: svc, logger: logger}
}

// RegisterRoutes registers tenant routes on the given router group.
func (h *TenantHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/tenants", h.List)
	rg.POST("/tenants", h.Create)
	rg.GET("/tenants/:id", h.Get)
	rg.PUT("/tenants/:id", h.Update)
	rg.DELETE("/tenants/:id", h.Delete)
	rg.POST("/tenants/:id/suspend", h.Suspend)
	rg.GET("/tenants/:id/config", h.GetConfig)
	rg.PUT("/tenants/:id/config", h.UpdateConfig)
	rg.GET("/tenants/:id/usage", h.GetUsage)
}

func (h *TenantHandler) List(c *gin.Context) {
	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	tenants, total, err := h.svc.ListTenants(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list tenants")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to list tenants"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"tenants": tenants, "total": total, "limit": limit, "offset": offset})
}

func (h *TenantHandler) Create(c *gin.Context) {
	var req admin.Tenant
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	if err := validateTenantRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	req.Name = sanitizeInput(req.Name)

	userTenantID, exists := c.Get("tenant_id")
	if exists {
		if userTenantIDStr, ok := userTenantID.(string); ok {
			if req.ID != uuid.Nil && req.ID.String() != userTenantIDStr {
				c.JSON(http.StatusForbidden, errorResponse("FORBIDDEN", "Cannot create tenant for different organization"))
				return
			}
		}
	}

	if err := h.svc.CreateTenant(c.Request.Context(), &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create tenant")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to create tenant"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"tenant": req})
}

func (h *TenantHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	tenant, err := h.svc.GetTenant(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get tenant")
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "Tenant not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"tenant": tenant})
}

func (h *TenantHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	var req admin.Tenant
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	req.ID = id
	if err := h.svc.UpdateTenant(c.Request.Context(), &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update tenant")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to update tenant"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"tenant": req})
}

func (h *TenantHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	if err := h.svc.DeleteTenant(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete tenant")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to delete tenant"))
		return
	}

	h.logger.Info().Str("tenant_id", id.String()).Msg("Tenant deleted")
	c.JSON(http.StatusOK, gin.H{"message": "Tenant deleted"})
}

func (h *TenantHandler) Suspend(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.svc.SuspendTenant(c.Request.Context(), id, req.Reason); err != nil {
		h.logger.Error().Err(err).Msg("Failed to suspend tenant")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to suspend tenant"))
		return
	}

	h.logger.Info().Str("tenant_id", id.String()).Str("reason", req.Reason).Msg("Tenant suspended")
	c.JSON(http.StatusOK, gin.H{"message": "Tenant suspended"})
}

func (h *TenantHandler) GetConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	config, err := h.svc.GetTenantConfig(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get tenant config")
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "Tenant not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config})
}

func (h *TenantHandler) UpdateConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	var req admin.TenantConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	if err := h.svc.UpdateTenantConfig(c.Request.Context(), id, &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update tenant config")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to update config"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

func (h *TenantHandler) GetUsage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_ID", "Invalid tenant ID"))
		return
	}

	usage, err := h.svc.GetTenantUsage(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get tenant usage")
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to get usage"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"usage": usage})
}

// SystemStats returns system-wide statistics (admin only).
func SystemStats(svc *admin.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := svc.GetSystemStats(c.Request.Context())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get system stats")
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "Failed to get stats"))
			return
		}
		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}

// --- helpers ---

func errorResponse(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	var intVal int
	if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
		return intVal
	}
	return defaultVal
}

func validateTenantRequest(req admin.Tenant) error {
	if req.Name == "" {
		return fmt.Errorf("name: required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("name: too long (max 255 characters)")
	}
	if req.Status != "" && req.Status != "active" && req.Status != "suspended" && req.Status != "deleted" {
		return fmt.Errorf("status: invalid value")
	}
	return nil
}

func sanitizeInput(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 32 && r != 127 {
			result = append(result, r)
		}
	}
	return string(result)
}

func validateUsername(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// ReadinessCheck returns a readiness handler that checks DB and cache health.
func ReadinessCheck(healthCheckers ...func(ctx context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		ready := true
		details := gin.H{}

		names := []string{"database", "cache"}
		for i, check := range healthCheckers {
			name := "unknown"
			if i < len(names) {
				name = names[i]
			}
			if err := check(ctx); err != nil {
				ready = false
				details[name] = "unhealthy"
			} else {
				details[name] = "healthy"
			}
		}

		if ready {
			c.JSON(http.StatusOK, gin.H{"ready": true, "details": details})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "details": details})
		}
	}
}
