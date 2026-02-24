package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/session"
	"github.com/rs/zerolog"
)

// SessionHandler handles session endpoints
type SessionHandler struct {
	service *session.Service
	logger  zerolog.Logger
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(service *session.Service, logger zerolog.Logger) *SessionHandler {
	return &SessionHandler{
		service: service,
		logger:  logger,
	}
}

// CreateSessionRequest represents a session creation request
type CreateSessionRequest struct {
	CredentialID string `json:"credential_id" binding:"required"`
	TargetID     string `json:"target_id"`
	Type         string `json:"type" binding:"required,oneof=ssh rdp database kubernetes web api"`
	TargetHost   string `json:"target_host" binding:"required"`
	TargetPort   int    `json:"target_port" binding:"required,min=1,max=65535"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// List returns a paginated list of sessions
func (h *SessionHandler) List(c *gin.Context) {
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
	filter := session.SessionFilter{}
	if t := c.Query("type"); t != "" {
		sessionType := session.SessionType(t)
		filter.Type = &sessionType
	}
	if st := c.Query("status"); st != "" {
		status := session.SessionStatus(st)
		filter.Status = &status
	}
	if uid := c.Query("user_id"); uid != "" {
		if userID, err := uuid.Parse(uid); err == nil {
			filter.UserID = &userID
		}
	}
	if from := c.Query("from"); from != "" {
		if startTime, err := time.Parse(time.RFC3339, from); err == nil {
			filter.StartTimeFrom = &startTime
		}
	}
	if to := c.Query("to"); to != "" {
		if endTime, err := time.Parse(time.RFC3339, to); err == nil {
			filter.StartTimeTo = &endTime
		}
	}

	sessions, total, err := h.service.ListSessions(c.Request.Context(), tenantIDUUID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_SESSIONS_FAILED",
				"message": "Failed to list sessions",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// Get retrieves a single session by ID
func (h *SessionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_SESSION_ID",
				"message": "Invalid session ID",
			},
		})
		return
	}

	s, err := h.service.GetSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "SESSION_NOT_FOUND",
				"message": "Session not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": s})
}

// Create creates a new session
func (h *SessionHandler) Create(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	credentialID, err := uuid.Parse(req.CredentialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CREDENTIAL_ID",
				"message": "Invalid credential ID",
			},
		})
		return
	}

	// Parse target ID if provided
	var targetID *uuid.UUID
	if req.TargetID != "" {
		if tid, err := uuid.Parse(req.TargetID); err == nil {
			targetID = &tid
		}
	}

	sess := &session.Session{
		UserID:       userIDUUID,
		CredentialID: credentialID,
		TargetID:     targetID,
		Type:         session.SessionType(req.Type),
		TargetHost:   req.TargetHost,
		TargetPort:   req.TargetPort,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		TenantID:     tenantIDUUID,
	}

	if err := h.service.StartSession(c.Request.Context(), sess); err != nil {
		h.logger.Error().Err(err).Msg("Failed to start session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "START_SESSION_FAILED",
				"message": "Failed to start session",
			},
		})
		return
	}

	h.logger.Info().
		Str("session_id", sess.ID.String()).
		Str("credential_id", credentialID.String()).
		Msg("Session started")

	c.JSON(http.StatusCreated, gin.H{"session": sess})
}

// Terminate terminates an active session
func (h *SessionHandler) Terminate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_SESSION_ID",
				"message": "Invalid session ID",
			},
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	reason := req.Reason
	if reason == "" {
		reason = "terminated by user"
	}

	if err := h.service.TerminateSession(c.Request.Context(), id, userIDUUID, reason); err != nil {
		h.logger.Error().Err(err).Msg("Failed to terminate session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "TERMINATE_SESSION_FAILED",
				"message": "Failed to terminate session: " + err.Error(),
			},
		})
		return
	}

	h.logger.Info().
		Str("session_id", id.String()).
		Str("terminated_by", userIDUUID.String()).
		Msg("Session terminated")

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated successfully"})
}

// GetStats returns session statistics
func (h *SessionHandler) GetStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	stats, err := h.service.GetSessionStats(c.Request.Context(), tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get session stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "GET_STATS_FAILED",
				"message": "Failed to get session statistics",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// GetActive returns active sessions for the current user
func (h *SessionHandler) GetActive(c *gin.Context) {
	userID, _ := c.Get("user_id")
	_ = userID // Will be used when implementing active sessions query

	// Use repository to get active sessions for user
	// For now, return empty list as this requires direct repo access
	c.JSON(http.StatusOK, gin.H{
		"sessions": []interface{}{},
		"total":    0,
	})
}
