package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/discovery"
	"github.com/openpam/openpam/pkg/validation"
	"github.com/rs/zerolog"
)

// DiscoveryHandler handles asset discovery endpoints
type DiscoveryHandler struct {
	service *discovery.Service
	logger  zerolog.Logger
}

// NewDiscoveryHandler creates a new discovery handler
func NewDiscoveryHandler(service *discovery.Service, logger zerolog.Logger) *DiscoveryHandler {
	return &DiscoveryHandler{service: service, logger: logger}
}

// ListScans returns discovery scan history
func (h *DiscoveryHandler) ListScans(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	limit := getIntParam(c, "limit", 20)
	offset := getIntParam(c, "offset", 0)

	scans, total, err := h.service.ListScans(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list scans")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list scans"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": scans, "total": total, "limit": limit, "offset": offset})
}

// CreateScan creates a new discovery scan
func (h *DiscoveryHandler) CreateScan(c *gin.Context) {
	var req struct {
		Name      string   `json:"name" binding:"required"`
		Subnets   []string `json:"subnets" binding:"required"`
		PortRange string   `json:"port_range"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Validate using Chain of Responsibility pattern
	chain := validation.NewChain().
		Add(validation.RequiredString{Field: "name", Value: req.Name}).
		Add(validation.MaxLength{Field: "name", Value: req.Name, Max: 128})

	if errs := chain.Run(); errs.HasErrors() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": errs.Error(), "details": errs}})
		return
	}

	if len(req.Subnets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "At least one subnet is required"}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetString("tenant_id"))
	portRange := req.PortRange
	if portRange == "" {
		portRange = "22,80,443,3389,5432,3306"
	}

	scan := &discovery.Scan{
		TenantID:  tenantID,
		Name:      req.Name,
		Subnets:   req.Subnets,
		PortRange: portRange,
	}

	if err := h.service.CreateScan(c.Request.Context(), scan); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create scan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create scan"}})
		return
	}

	c.JSON(http.StatusCreated, scan)
}

// RunScan triggers a scan execution
func (h *DiscoveryHandler) RunScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid scan ID"}})
		return
	}

	if err := h.service.RunScan(c.Request.Context(), scanID); err != nil {
		h.logger.Error().Err(err).Str("scan_id", scanID.String()).Msg("Failed to run scan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "SCAN_ERROR", "message": err.Error()}})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Scan started", "scan_id": scanID})
}

// ListAssets returns discovered assets
func (h *DiscoveryHandler) ListAssets(c *gin.Context) {
	tenantID, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	limit := getIntParam(c, "limit", 20)
	offset := getIntParam(c, "offset", 0)

	var filter discovery.AssetFilter
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if assetType := c.Query("asset_type"); assetType != "" {
		filter.AssetType = &assetType
	}

	assets, total, err := h.service.ListAssets(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list assets")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list assets"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": assets, "total": total, "limit": limit, "offset": offset})
}
