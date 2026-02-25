package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/policy"
	"github.com/rs/zerolog"
)

// AnalyticsClient provides client for analytics service
type AnalyticsClient struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewAnalyticsClient creates a new analytics client
func NewAnalyticsClient(baseURL string, logger zerolog.Logger) *AnalyticsClient {
	return &AnalyticsClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// CommandBlacklist represents a command blacklist rule from analytics
type CommandBlacklistRule struct {
	ID             uuid.UUID  `json:"id"`
	CommandPattern string     `json:"command_pattern"`
	PatternType    string     `json:"pattern_type"`
	BaseCommand    *string    `json:"base_command"`
	Action         string     `json:"action"`
	Severity       string     `json:"severity"`
	AppliesToUsers []uuid.UUID `json:"applies_to_users"`
	AppliesToGroups []uuid.UUID `json:"applies_to_groups"`
	Enabled        bool       `json:"enabled"`
}

// GetCommandBlacklist retrieves command blacklist for a tenant
func (c *AnalyticsClient) GetCommandBlacklist(ctx context.Context, tenantID uuid.UUID) ([]CommandBlacklistRule, error) {
	url := fmt.Sprintf("%s/api/v1/analytics/commands/blacklist", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Tenant-ID", tenantID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("analytics client: received status %d", resp.StatusCode)
	}

	var result struct {
		Blacklist []CommandBlacklistRule `json:"blacklist"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Blacklist, nil
}

// PolicyMiddleware provides real-time policy enforcement for incoming requests
type PolicyMiddleware struct {
	service          *policy.Service
	analyticsClient *AnalyticsClient
	logger           zerolog.Logger
}

// NewPolicyMiddleware creates a new policy middleware
func NewPolicyMiddleware(service *policy.Service, analyticsClient *AnalyticsClient, logger zerolog.Logger) *PolicyMiddleware {
	return &PolicyMiddleware{
		service:          service,
		analyticsClient: analyticsClient,
		logger:           logger,
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
// Returns (allowed, action, blacklistID, error)
func (m *PolicyMiddleware) EvaluateCommand(ctx context.Context, tenantID, userID, sessionID uuid.UUID, command string, userGroups []uuid.UUID) (bool, string, *uuid.UUID, error) {
	// First check policy service
	allowed, reason, err := m.service.EvaluateCommand(ctx, tenantID, userID, command, &sessionID)
	if !allowed {
		return false, reason, nil, err
	}

	// Check analytics command blacklist if available
	if m.analyticsClient != nil {
		blacklistRules, err := m.analyticsClient.GetCommandBlacklist(ctx, tenantID)
		if err != nil {
			m.logger.Warn().Err(err).Msg("Failed to fetch command blacklist from analytics, continuing")
		} else {
			// Check command against blacklist rules
			for _, rule := range blacklistRules {
				if !rule.Enabled {
					continue
				}

				// Check if rule applies to this user
				if len(rule.AppliesToUsers) > 0 {
					userMatch := false
					for _, allowedUserID := range rule.AppliesToUsers {
						if allowedUserID == userID {
							userMatch = true
							break
						}
					}
					if !userMatch {
						continue
					}
				}

				// Check if rule applies to user groups
				if len(rule.AppliesToGroups) > 0 {
					groupMatch := false
					for _, allowedGroupID := range rule.AppliesToGroups {
						for _, userGroupID := range userGroups {
							if allowedGroupID == userGroupID {
								groupMatch = true
								break
							}
						}
						if groupMatch {
							break
						}
					}
					if !groupMatch {
						continue
					}
				}

				// Check if command matches the pattern
				if m.commandMatchesPattern(command, rule) {
					switch rule.Action {
					case "block":
						blacklistID := rule.ID
						return false, "Command blocked by blacklist policy", &blacklistID, nil
					case "warn":
						return true, "Command warned by blacklist policy", nil, nil
					case "audit":
						// Log but allow
						m.logger.Warn().
							Str("tenant_id", tenantID.String()).
							Str("user_id", userID.String()).
							Str("command", command).
							Str("blacklist_id", rule.ID.String()).
							Msg("Command matched audit blacklist rule")
						return true, "", nil, nil
					}
				}
			}
		}
	}

	// Check command blacklist for dangerous commands
	// In production, this would query the analytics service command blacklist
	// For now, implement basic dangerous command detection
	dangerousCommands := map[string]string{
		"rm -rf /":          "block",
		"rm -rf /*":         "block",
		"mkfs":              "block",
		":(){ :|:& };:":     "block",
		"dd if=/dev/zero":   "block",
		"dd if=/dev/random": "block",
		"shutdown":          "block",
		"reboot":            "warn",
		"init 0":            "block",
		"chmod 000":         "warn",
	}

	// Check for exact matches
	for dangerousCmd, action := range dangerousCommands {
		if strings.Contains(command, dangerousCmd) {
			if action == "block" {
				return false, "Command blocked by security policy", nil, nil
			}
			return true, "warned", nil, nil
		}
	}

	// Check for privilege escalation without approval
	if strings.HasPrefix(command, "sudo ") || strings.HasPrefix(command, "su ") {
		// Would check if approval is on file
		return true, "logged", nil, nil
	}

	return true, "", nil, nil
}

// EvaluateCommandAgainstBlacklist evaluates a command against the command blacklist
func (m *PolicyMiddleware) EvaluateCommandAgainstBlacklist(ctx context.Context, tenantID, userID uuid.UUID, command string, userGroups []uuid.UUID) (allowed bool, action string, reason string) {
	// Basic dangerous command patterns
	dangerousPatterns := []struct {
		pattern string
		action  string
		reason  string
	}{
		{"rm -rf", "block", "Recursive force delete is blocked"},
		{"mkfs", "block", "Filesystem creation is blocked"},
		{"dd if=", "block", "Direct disk write is blocked"},
		{":(){ :|:& };:", "block", "Fork bombs are blocked"},
		{"shutdown", "block", "System shutdown is blocked"},
		{"reboot", "warn", "System reboot requires approval"},
		{"chmod 000", "warn", "Removing all permissions is suspicious"},
	}

	cmdLower := strings.ToLower(command)

	for _, dp := range dangerousPatterns {
		if strings.Contains(cmdLower, dp.pattern) {
			if dp.action == "block" {
				return false, "blocked", dp.reason
			}
			return true, "warned", dp.reason
		}
	}

	return true, "", ""
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

// commandMatchesPattern checks if a command matches a blacklist rule pattern
func (m *PolicyMiddleware) commandMatchesPattern(command string, rule CommandBlacklistRule) bool {
	switch rule.PatternType {
	case "exact":
		return strings.TrimSpace(command) == strings.TrimSpace(rule.CommandPattern)
	case "regex":
		matched, err := m.regexMatch(command, rule.CommandPattern)
		if err != nil {
			m.logger.Warn().Err(err).
				Str("pattern", rule.CommandPattern).
				Msg("Invalid regex pattern, treating as no match")
			return false
		}
		return matched
	case "glob":
		matched, err := m.globMatch(command, rule.CommandPattern)
		if err != nil {
			m.logger.Warn().Err(err).
				Str("pattern", rule.CommandPattern).
				Msg("Invalid glob pattern, treating as no match")
			return false
		}
		return matched
	default:
		// Default to substring match
		return strings.Contains(command, rule.CommandPattern)
	}
}

// regexCache caches compiled regex patterns for performance
var (
	policyRegexCache = make(map[string]*regexp.Regexp)
	policyRegexMu    sync.RWMutex
)

// regexMatch performs regex pattern matching with caching
func (m *PolicyMiddleware) regexMatch(command, pattern string) (bool, error) {
	policyRegexMu.RLock()
	re, exists := policyRegexCache[pattern]
	policyRegexMu.RUnlock()

	if !exists {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}

		policyRegexMu.Lock()
		policyRegexCache[pattern] = re
		policyRegexMu.Unlock()
	}

	return re.MatchString(command), nil
}

// globMatch performs glob pattern matching using filepath.Match
func (m *PolicyMiddleware) globMatch(command, pattern string) (bool, error) {
	// filepath.Match requires pattern to be a valid glob pattern
	// It supports * (matches any sequence) and ? (matches single character)
	matched, err := filepath.Match(pattern, command)
	if err != nil {
		// Invalid pattern - treat as no match
		return false, nil
	}
	return matched, nil
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
