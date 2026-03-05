package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/admin"
	"github.com/openpam/openpam/pkg/validation"
	"github.com/rs/zerolog"
)

// AdminHandler handles admin-level operations (tenants, system stats)
type AdminHandler struct {
	service *admin.Service
	logger  zerolog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(service *admin.Service, logger zerolog.Logger) *AdminHandler {
	return &AdminHandler{service: service, logger: logger}
}

// ListTenants returns all tenants
func (h *AdminHandler) ListTenants(c *gin.Context) {
	limit := getIntParam(c, "limit", 20)
	offset := getIntParam(c, "offset", 0)

	tenants, total, err := h.service.ListTenants(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list tenants")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list tenants"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": tenants, "total": total, "limit": limit, "offset": offset})
}

// CreateTenant creates a new tenant
func (h *AdminHandler) CreateTenant(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Domain string `json:"domain"`
		Plan   string `json:"plan"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Validate using Chain of Responsibility pattern
	chain := validation.NewChain().
		Add(validation.RequiredString{Field: "name", Value: req.Name}).
		Add(validation.MaxLength{Field: "name", Value: req.Name, Max: 128}).
		Add(validation.NoSQLInjection{Field: "name", Value: req.Name})

	if req.Plan != "" {
		chain.Add(validation.OneOf{Field: "plan", Value: req.Plan, Allowed: []string{"free", "pro", "enterprise"}})
	}

	if errs := chain.Run(); errs.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": errs.Error(), "details": errs}})
		return
	}

	plan := req.Plan
	if plan == "" {
		plan = "free"
	}

	tenant := &admin.Tenant{
		Name:   req.Name,
		Domain: req.Domain,
		Plan:   plan,
	}

	if err := h.service.CreateTenant(c.Request.Context(), tenant); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create tenant"}})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// UpdateTenant updates a tenant
func (h *AdminHandler) UpdateTenant(c *gin.Context) {
	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid tenant ID"}})
		return
	}

	tenant, err := h.service.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Tenant not found"}})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Domain != "" {
		tenant.Domain = req.Domain
	}

	if err := h.service.UpdateTenant(c.Request.Context(), tenant); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update tenant"}})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// SystemStats returns system-wide statistics
func (h *AdminHandler) SystemStats(c *gin.Context) {
	stats, err := h.service.GetSystemStats(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get system stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get system stats"}})
		return
	}

	c.JSON(http.StatusOK, stats)
}
