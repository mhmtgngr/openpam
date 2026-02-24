package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/target"
	"github.com/rs/zerolog"
)

// TargetHandler handles target endpoints
type TargetHandler struct {
	service *target.TargetService
	logger  zerolog.Logger
}

// NewTargetHandler creates a new target handler
func NewTargetHandler(service *target.TargetService, logger zerolog.Logger) *TargetHandler {
	return &TargetHandler{
		service: service,
		logger:  logger,
	}
}

// CreateTargetRequest represents a target creation request
type CreateTargetRequest struct {
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Type             string                 `json:"type" binding:"required,oneof=ssh rdp database web kubernetes api"`
	Environment      string                 `json:"environment" binding:"required,oneof=production staging development test"`
	Sensitivity      string                 `json:"sensitivity" binding:"required,oneof=critical high medium low"`
	Host             string                 `json:"host" binding:"required"`
	Port             int                    `json:"port" binding:"required,min=1,max=65535"`
	ConnectionString string                 `json:"connection_string"`
	Platform         string                 `json:"platform"`
	Tags             []string               `json:"tags"`
	Metadata         map[string]interface{} `json:"metadata"`
	RequireApproval  *bool                  `json:"require_approval"`
	RequireMFA       *bool                  `json:"require_mfa"`
	MaxDuration      int                    `json:"max_duration_minutes"`
	ApprovalGroupID  string                 `json:"approval_group_id"`
}

// UpdateTargetRequest represents a target update request
type UpdateTargetRequest struct {
	Name             *string                 `json:"name"`
	Description      *string                 `json:"description"`
	Type             *string                 `json:"type"`
	Environment      *string                 `json:"environment"`
	Sensitivity      *string                 `json:"sensitivity"`
	Host             *string                 `json:"host"`
	Port             *int                    `json:"port"`
	ConnectionString *string                 `json:"connection_string"`
	Platform         *string                 `json:"platform"`
	Tags             []string                `json:"tags"`
	Metadata         map[string]interface{}  `json:"metadata"`
	RequireApproval  *bool                   `json:"require_approval"`
	RequireMFA       *bool                   `json:"require_mfa"`
	MaxDuration      *int                    `json:"max_duration_minutes"`
	ApprovalGroupID  *string                 `json:"approval_group_id"`
	Status           *string                 `json:"status"`
}

// List returns a paginated list of targets
func (h *TargetHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	// Parse query parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build filter
	filter := target.TargetFilter{}
	if t := c.Query("type"); t != "" {
		tType := target.TargetType(t)
		filter.Type = &tType
	}
	if e := c.Query("environment"); e != "" {
		env := target.TargetEnvironment(e)
		filter.Environment = &env
	}
	if s := c.Query("sensitivity"); s != "" {
		sens := target.TargetSensitivity(s)
		filter.Sensitivity = &sens
	}
	if st := c.Query("status"); st != "" {
		filter.Status = &st
	}
	if ra := c.Query("require_approval"); ra != "" {
		if ra == "true" {
			val := true
			filter.RequireApproval = &val
		} else if ra == "false" {
			val := false
			filter.RequireApproval = &val
		}
	}
	if search := c.Query("search"); search != "" {
		filter.SearchTerm = search
	}

	targets, total, err := h.service.GetTargetsForUser(c.Request.Context(), uuid.Nil, tenantIDUUID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list targets")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_TARGETS_FAILED",
				"message": "Failed to list targets",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"targets": targets,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// Get retrieves a single target by ID
func (h *TargetHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TARGET_ID",
				"message": "Invalid target ID",
			},
		})
		return
	}

	t, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "TARGET_NOT_FOUND",
				"message": "Target not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"target": t})
}

// Create creates a new target
func (h *TargetHandler) Create(c *gin.Context) {
	var req CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	targetObj := &target.Target{
		Name:             req.Name,
		Description:      req.Description,
		Type:             target.TargetType(req.Type),
		Environment:      target.TargetEnvironment(req.Environment),
		Sensitivity:      target.TargetSensitivity(req.Sensitivity),
		Host:             req.Host,
		Port:             req.Port,
		ConnectionString: req.ConnectionString,
		Platform:         req.Platform,
		Tags:             req.Tags,
		Metadata:         req.Metadata,
		TenantID:         tenantIDUUID,
	}

	if req.RequireApproval != nil {
		targetObj.RequireApproval = *req.RequireApproval
	}
	if req.RequireMFA != nil {
		targetObj.RequireMFA = *req.RequireMFA
	}
	if req.MaxDuration > 0 {
		targetObj.MaxDuration = req.MaxDuration
	}
	if req.ApprovalGroupID != "" {
		if agID, err := uuid.Parse(req.ApprovalGroupID); err == nil {
			targetObj.ApprovalGroupID = &agID
		}
	}

	if err := h.service.CreateTarget(c.Request.Context(), targetObj); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create target")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CREATE_TARGET_FAILED",
				"message": "Failed to create target",
			},
		})
		return
	}

	h.logger.Info().
		Str("target_id", targetObj.ID.String()).
		Str("name", targetObj.Name).
		Msg("Target created")

	c.JSON(http.StatusCreated, gin.H{"target": targetObj})
}

// Update updates an existing target
func (h *TargetHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TARGET_ID",
				"message": "Invalid target ID",
			},
		})
		return
	}

	var req UpdateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	// Get existing target
	t, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "TARGET_NOT_FOUND",
				"message": "Target not found",
			},
		})
		return
	}

	// Update fields
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Type != nil {
		t.Type = target.TargetType(*req.Type)
	}
	if req.Environment != nil {
		t.Environment = target.TargetEnvironment(*req.Environment)
	}
	if req.Sensitivity != nil {
		t.Sensitivity = target.TargetSensitivity(*req.Sensitivity)
	}
	if req.Host != nil {
		t.Host = *req.Host
	}
	if req.Port != nil {
		t.Port = *req.Port
	}
	if req.ConnectionString != nil {
		t.ConnectionString = *req.ConnectionString
	}
	if req.Platform != nil {
		t.Platform = *req.Platform
	}
	if req.Tags != nil {
		t.Tags = req.Tags
	}
	if req.Metadata != nil {
		t.Metadata = req.Metadata
	}
	if req.RequireApproval != nil {
		t.RequireApproval = *req.RequireApproval
	}
	if req.RequireMFA != nil {
		t.RequireMFA = *req.RequireMFA
	}
	if req.MaxDuration != nil {
		t.MaxDuration = *req.MaxDuration
	}
	if req.ApprovalGroupID != nil {
		if agID, err := uuid.Parse(*req.ApprovalGroupID); err == nil {
			t.ApprovalGroupID = &agID
		} else {
			t.ApprovalGroupID = nil
		}
	}
	if req.Status != nil {
		t.Status = *req.Status
	}

	if err := h.service.UpdateTarget(c.Request.Context(), t); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update target")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "UPDATE_TARGET_FAILED",
				"message": "Failed to update target",
			},
		})
		return
	}

	h.logger.Info().
		Str("target_id", t.ID.String()).
		Msg("Target updated")

	c.JSON(http.StatusOK, gin.H{"target": t})
}

// Delete soft deletes a target
func (h *TargetHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TARGET_ID",
				"message": "Invalid target ID",
			},
		})
		return
	}

	if err := h.service.DeleteTarget(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete target")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DELETE_TARGET_FAILED",
				"message": "Failed to delete target",
			},
		})
		return
	}

	h.logger.Info().
		Str("target_id", id.String()).
		Msg("Target deleted")

	c.JSON(http.StatusOK, gin.H{"message": "Target deleted successfully"})
}

// Verify verifies connectivity to a target
func (h *TargetHandler) Verify(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_TARGET_ID",
				"message": "Invalid target ID",
			},
		})
		return
	}

	// Verify in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = h.service.VerifyTarget(ctx, id)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Target verification initiated",
	})
}
