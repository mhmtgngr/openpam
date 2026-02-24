package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/policy"
	"github.com/rs/zerolog"
)

// PolicyMiddleware provides real-time policy enforcement for incoming requests
type PolicyMiddleware struct {
	service *policy.Service
	logger  zerolog.Logger
}

// NewPolicyMiddleware creates a new policy middleware
func NewPolicyMiddleware(service *policy.Service, logger zerolog.Logger) *PolicyMiddleware {
	return &PolicyMiddleware{
		service: service,
		logger:  logger,
	}
}

// Config defines middleware configuration
type Config struct {
	// SkipPaths are paths that bypass policy checks
	SkipPaths []string
	// DenyByDefault controls the behavior when no policies match
	DenyByDefault bool
	// LogAllRequests enables logging of all policy evaluations
	LogAllRequests bool
	// CacheTTL is the duration to cache evaluation results
	CacheTTL time.Duration
}

// DefaultConfig returns the default middleware configuration
func DefaultConfig() Config {
	return Config{
		SkipPaths: []string{
			"/health",
			"/ready",
			"/api/v1/auth/login",
			"/api/v1/auth/logout",
		},
		DenyByDefault: true,
		LogAllRequests: true,
		CacheTTL:       30 * time.Second,
	}
}

// EvaluateAction evaluates policies for a given action and returns the result
func (m *PolicyMiddleware) EvaluateAction(ctx context.Context, tenantID, userID uuid.UUID, userRoles, userGroups []uuid.UUID, action, resourceType string, resourceID uuid.UUID, clientIP, userAgent string) (*policy.EvaluationResponse, error) {
	req := policy.EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    userRoles,
		UserGroups:   userGroups,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		Time:         time.Now(),
		MFAVerified:  false, // Would be set from session context
		DeviceTrust:  0,     // Would be set from device context
	}

	return m.service.Evaluate(ctx, req)
}

// EvaluateSessionAccess evaluates policies for session access
func (m *PolicyMiddleware) EvaluateSessionAccess(ctx context.Context, tenantID, userID uuid.UUID, targetHost string, targetPort int, clientIP string) (*policy.EvaluationResponse, error) {
	// Get user roles from context (would be passed in)
	var userRoles []uuid.UUID

	// Create a resource ID from target (could be a target UUID lookup)
	resourceID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(fmt.Sprintf("%s:%d", targetHost, targetPort)))

	req := policy.EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    userRoles,
		Action:       "session_start",
		ResourceType: "target",
		ResourceID:   resourceID,
		ClientIP:     clientIP,
		Time:         time.Now(),
		MFAVerified:  false,
	}

	return m.service.Evaluate(ctx, req)
}

// EvaluateCredentialCheckout evaluates policies for credential checkout
func (m *PolicyMiddleware) EvaluateCredentialCheckout(ctx context.Context, tenantID, userID, credentialID uuid.UUID, clientIP string) (*policy.EvaluationResponse, error) {
	// Get user roles from context
	var userRoles []uuid.UUID

	req := policy.EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    userRoles,
		Action:       "credential_checkout",
		ResourceType: "credential",
		ResourceID:   credentialID,
		ClientIP:     clientIP,
		Time:         time.Now(),
		MFAVerified:  false,
	}

	return m.service.Evaluate(ctx, req)
}

// EvaluateCommand evaluates policies for command execution during a session
func (m *PolicyMiddleware) EvaluateCommand(ctx context.Context, tenantID, userID, sessionID uuid.UUID, command string) (bool, string, error) {
	return m.service.EvaluateCommand(ctx, tenantID, userID, command, &sessionID)
}

// Middleware returns a Gin middleware for policy enforcement
func (m *PolicyMiddleware) Middleware(config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path should be skipped
		if m.shouldSkipPath(c.Request.URL.Path, config.SkipPaths) {
			c.Next()
			return
		}

		// Extract context from request
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			m.abortWithPolicyError(c, http.StatusUnauthorized, "TENANT_REQUIRED", "Tenant ID required")
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			m.abortWithPolicyError(c, http.StatusUnauthorized, "USER_REQUIRED", "User ID required")
			return
		}

		// Get user roles
		userRoles := m.getUserRoles(c)

		// Get user groups
		userGroups := m.getUserGroups(c)

		// Build evaluation request from HTTP request
		req := m.buildEvaluationRequest(c, tenantID.(string), userID.(string), userRoles, userGroups)

		// Evaluate policies
		result, err := m.service.Evaluate(c.Request.Context(), req)
		if err != nil {
			m.logger.Error().Err(err).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Policy evaluation failed")
			m.abortWithPolicyError(c, http.StatusInternalServerError, "EVALUATION_ERROR", "Policy evaluation failed")
			return
		}

		// Store result in context for downstream handlers
		c.Set("policy_result", result)

		// Log evaluation if configured
		if config.LogAllRequests {
			m.logEvaluation(c, req, result)
		}

		// Check if access is allowed
		if result.Effect == policy.EvaluationResultDeny {
			reason := "Access denied by policy"
			if len(result.DenialReasons) > 0 {
				reason = strings.Join(result.DenialReasons, "; ")
			}
			m.abortWithPolicyError(c, http.StatusForbidden, "ACCESS_DENIED", reason)
			return
		}

		// Check if approval is required
		if result.Effect == policy.EvaluationResultApprovalRequired {
			m.abortWithPolicyError(c, http.StatusForbidden, "APPROVAL_REQUIRED", "Approval required for this action")
			return
		}

		// Check for required actions
		if len(result.RequiredActions) > 0 {
			c.Set("required_actions", result.RequiredActions)
		}

		c.Next()
	}
}

// SessionMiddleware returns middleware for session-specific policy checks
func (m *PolicyMiddleware) SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to session-related endpoints
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/sessions") {
			c.Next()
			return
		}

		// Extract session context
		sessionID, exists := c.Get("session_id")
		if !exists {
			// For new session requests, evaluate creation policy
			m.evaluateSessionCreation(c)
			return
		}

		// For existing sessions, validate continued access
		m.evaluateSessionAccess(c, sessionID.(uuid.UUID))

		c.Next()
	}
}

// CredentialMiddleware returns middleware for credential-specific policy checks
func (m *PolicyMiddleware) CredentialMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to credential-related endpoints
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/credentials") {
			c.Next()
			return
		}

		// Check credential access policies
		credentialID := c.Param("id")
		if credentialID != "" {
			m.evaluateCredentialAccess(c, uuid.MustParse(credentialID))
		}

		c.Next()
	}
}

// Helper methods

func (m *PolicyMiddleware) shouldSkipPath(path string, skipPaths []string) bool {
	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

func (m *PolicyMiddleware) getUserRoles(c *gin.Context) []uuid.UUID {
	if roles, exists := c.Get("user_roles"); exists {
		if roleSlice, ok := roles.([]uuid.UUID); ok {
			return roleSlice
		}
		if roleSlice, ok := roles.([]string); ok {
			var roles []uuid.UUID
			for _, r := range roleSlice {
				if id, err := uuid.Parse(r); err == nil {
					roles = append(roles, id)
				}
			}
			return roles
		}
	}
	return []uuid.UUID{}
}

func (m *PolicyMiddleware) getUserGroups(c *gin.Context) []uuid.UUID {
	if groups, exists := c.Get("user_groups"); exists {
		if groupSlice, ok := groups.([]uuid.UUID); ok {
			return groupSlice
		}
		if groupSlice, ok := groups.([]string); ok {
			var groups []uuid.UUID
			for _, g := range groupSlice {
				if id, err := uuid.Parse(g); err == nil {
					groups = append(groups, id)
				}
			}
			return groups
		}
	}
	return []uuid.UUID{}
}

func (m *PolicyMiddleware) buildEvaluationRequest(c *gin.Context, tenantID, userID string, userRoles, userGroups []uuid.UUID) policy.EvaluationRequest {
	// Map HTTP method and path to action and resource type
	action, resourceType := m.mapRequestToAction(c)

	// Generate resource ID from path params or body
	resourceID := m.getResourceID(c, resourceType)

	return policy.EvaluationRequest{
		TenantID:     uuid.MustParse(tenantID),
		UserID:       uuid.MustParse(userID),
		UserRoles:    userRoles,
		UserGroups:   userGroups,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Time:         time.Now(),
		MFAVerified:  m.isMFAVerified(c),
		DeviceTrust:  m.getDeviceTrust(c),
	}
}

func (m *PolicyMiddleware) mapRequestToAction(c *gin.Context) (action, resourceType string) {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Map request to action
	switch method {
	case "GET":
		action = "read"
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = "access"
	}

	// Determine resource type from path
	switch {
	case strings.Contains(path, "/sessions"):
		resourceType = "session"
	case strings.Contains(path, "/credentials"):
		resourceType = "credential"
	case strings.Contains(path, "/targets"):
		resourceType = "target"
	case strings.Contains(path, "/users"):
		resourceType = "user"
	case strings.Contains(path, "/roles"):
		resourceType = "role"
	case strings.Contains(path, "/policies"):
		resourceType = "policy"
	default:
		resourceType = "system"
	}

	return action, resourceType
}

func (m *PolicyMiddleware) getResourceID(c *gin.Context, resourceType string) uuid.UUID {
	// Try to get ID from path parameter
	if idParam := c.Param("id"); idParam != "" {
		if id, err := uuid.Parse(idParam); err == nil {
			return id
		}
	}

	// For create operations, might need to extract from body
	// This would require more complex handling

	return uuid.Nil
}

func (m *PolicyMiddleware) isMFAVerified(c *gin.Context) bool {
	if verified, exists := c.Get("mfa_verified"); exists {
		if v, ok := verified.(bool); ok {
			return v
		}
	}
	return false
}

func (m *PolicyMiddleware) getDeviceTrust(c *gin.Context) int {
	if trust, exists := c.Get("device_trust"); exists {
		if t, ok := trust.(int); ok {
			return t
		}
		if t, ok := trust.(float64); ok {
			return int(t)
		}
	}
	return 0
}

func (m *PolicyMiddleware) abortWithPolicyError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
	c.Abort()
}

func (m *PolicyMiddleware) logEvaluation(c *gin.Context, req policy.EvaluationRequest, result *policy.EvaluationResponse) {
	m.logger.Info().
		Str("tenant_id", req.TenantID.String()).
		Str("user_id", req.UserID.String()).
		Str("action", req.Action).
		Str("resource_type", req.ResourceType).
		Str("result", string(result.Effect)).
		Str("path", c.Request.URL.Path).
		Str("method", c.Request.Method).
		Int64("duration_ms", result.DurationMs).
		Msg("Policy evaluation")
}

func (m *PolicyMiddleware) evaluateSessionCreation(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")

	req := policy.EvaluationRequest{
		TenantID:     uuid.MustParse(tenantID.(string)),
		UserID:       uuid.MustParse(userID.(string)),
		UserRoles:    m.getUserRoles(c),
		Action:       "create",
		ResourceType: "session",
		ResourceID:   uuid.Nil,
		ClientIP:     c.ClientIP(),
		Time:         time.Now(),
	}

	result, err := m.service.Evaluate(c.Request.Context(), req)
	if err != nil || result.Effect == policy.EvaluationResultDeny {
		m.abortWithPolicyError(c, http.StatusForbidden, "SESSION_CREATE_DENIED", "Not authorized to create session")
		return
	}

	c.Set("policy_result", result)
}

func (m *PolicyMiddleware) evaluateSessionAccess(c *gin.Context, sessionID uuid.UUID) {
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")

	req := policy.EvaluationRequest{
		TenantID:     uuid.MustParse(tenantID.(string)),
		UserID:       uuid.MustParse(userID.(string)),
		UserRoles:    m.getUserRoles(c),
		Action:       "access",
		ResourceType: "session",
		ResourceID:   sessionID,
		ClientIP:     c.ClientIP(),
		Time:         time.Now(),
		SessionID:    &sessionID,
	}

	result, err := m.service.Evaluate(c.Request.Context(), req)
	if err != nil || result.Effect == policy.EvaluationResultDeny {
		m.abortWithPolicyError(c, http.StatusForbidden, "SESSION_ACCESS_DENIED", "Not authorized to access this session")
		return
	}

	c.Set("policy_result", result)
}

func (m *PolicyMiddleware) evaluateCredentialAccess(c *gin.Context, credentialID uuid.UUID) {
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")

	req := policy.EvaluationRequest{
		TenantID:     uuid.MustParse(tenantID.(string)),
		UserID:       uuid.MustParse(userID.(string)),
		UserRoles:    m.getUserRoles(c),
		Action:       "access",
		ResourceType: "credential",
		ResourceID:   credentialID,
		ClientIP:     c.ClientIP(),
		Time:         time.Now(),
	}

	result, err := m.service.Evaluate(c.Request.Context(), req)
	if err != nil || result.Effect == policy.EvaluationResultDeny {
		m.abortWithPolicyError(c, http.StatusForbidden, "CREDENTIAL_ACCESS_DENIED", "Not authorized to access this credential")
		return
	}

	c.Set("policy_result", result)
}
