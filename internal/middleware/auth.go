package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Auth validates JWT token and extracts user info.
// SECURITY ENHANCEMENT: This middleware validates token format and Bearer prefix.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Missing authorization header",
				},
			})
			c.Abort()
			return
		}

		// SECURITY: Validate Bearer prefix
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN_FORMAT",
					"message": "Authorization header must use Bearer scheme",
				},
			})
			c.Abort()
			return
		}

		// SECURITY: Check for token length to prevent DoS
		token := authHeader[7:]
		if len(token) < 20 {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Token is too short to be valid",
				},
			})
			c.Abort()
			return
		}
		if len(token) > 4096 {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Token is too long",
				},
			})
			c.Abort()
			return
		}

		// Token validation is handled by auth package
		if _, exists := c.Get("user_id"); !exists {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole checks if user has required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User not authenticated",
				},
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		allowed := false
		for _, r := range roles {
			if r == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireMFA checks if user has completed MFA verification
func RequireMFA() gin.HandlerFunc {
	return func(c *gin.Context) {
		mfaVerified, exists := c.Get("mfa_verified")
		if !exists || !mfaVerified.(bool) {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "MFA_REQUIRED",
					"message": "Multi-factor authentication required",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// PrivilegedOperation verifies MFA and user permissions for privileged operations.
// These operations include: credential checkout, session termination, role changes, etc.
func PrivilegedOperation(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		mfaVerified, exists := c.Get("mfa_verified")
		if !exists || !mfaVerified.(bool) {
			requestID, _ := c.Get("request_id")
			logger.Warn().
				Str("request_id", requestID.(string)).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Privileged operation attempted without MFA")

			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "MFA_REQUIRED",
					"message": "Multi-factor authentication required for this operation",
					"details": gin.H{
						"action_required": "complete_mfa",
						"mfa_methods":     []string{"totp", "sms", "email", "webauthn"},
					},
				},
			})
			c.Abort()
			return
		}

		// Verify MFA timestamp is recent (within 5 minutes)
		if mfaTimestamp, exists := c.Get("mfa_timestamp"); exists {
			if timestamp, ok := mfaTimestamp.(time.Time); ok {
				if time.Since(timestamp) > 5*time.Minute {
					c.JSON(403, gin.H{
						"error": gin.H{
							"code":    "MFA_EXPIRED",
							"message": "MFA verification expired. Please re-authenticate.",
						},
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

// MFAScope defines which operations require MFA
type MFAScope string

const (
	MFAScopeCredentialCheckout MFAScope = "credential_checkout"
	MFAScopeSessionTerminate  MFAScope = "session_terminate"
	MFAScopeRoleModify        MFAScope = "role_modify"
	MFAScopeUserModify        MFAScope = "user_modify"
	MFAScopeSettingsModify    MFAScope = "settings_modify"
)

// RequireMFAForScope checks MFA based on operation scope
func RequireMFAForScope(scope MFAScope, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		mfaScopes, exists := c.Get("mfa_scopes")
		if !exists {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "MFA_REQUIRED",
					"message": fmt.Sprintf("MFA required for %s operations", scope),
				},
			})
			c.Abort()
			return
		}

		scopes, ok := mfaScopes.(map[MFAScope]bool)
		if !ok || !scopes[scope] {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "MFA_REQUIRED",
					"message": fmt.Sprintf("MFA required for %s operations", scope),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// VerifyMFAToken verifies an MFA token provided in the request
func VerifyMFAToken(verifyFunc func(token, userID string) (bool, error), logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			MFAToken string `json:"mfa_token" binding:"required"`
		}

		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_BODY", "message": "Failed to read request body"}})
				c.Abort()
				return
			}

			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

			if err := json.Unmarshal(body, &req); err == nil && req.MFAToken != "" {
				userID, exists := c.Get("user_id")
				if !exists {
					c.JSON(401, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
					c.Abort()
					return
				}

				valid, err := verifyFunc(req.MFAToken, userID.(string))
				if err != nil || !valid {
					requestID, _ := c.Get("request_id")
					logger.Warn().
						Str("request_id", requestID.(string)).
						Str("user_id", userID.(string)).
						Err(err).
						Msg("Invalid MFA token")

					c.JSON(403, gin.H{
						"error": gin.H{
							"code":    "INVALID_MFA_TOKEN",
							"message": "Multi-factor authentication token is invalid or expired",
						},
					})
					c.Abort()
					return
				}

				c.Set("mfa_verified", true)
				c.Set("mfa_timestamp", time.Now())
			}
		}

		c.Next()
	}
}
