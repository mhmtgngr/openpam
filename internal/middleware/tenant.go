package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Tenant extracts tenant from context and adds it to gin context.
// SECURITY: This is a basic tenant extractor. For production isolation,
// use RequireTenantIsolation() which enforces tenant presence.
func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if exists {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	}
}

// RequireTenantIsolation enforces tenant isolation to prevent A01:2021 access control bypasses.
// This middleware should be applied globally to all authenticated routes.
func RequireTenantIsolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/metrics" || path == "/api/v1/auth/login" || path == "/api/v1/auth/register" {
			c.Next()
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists || tenantID == nil || tenantID.(string) == "" {
			if _, hasUser := c.Get("user_id"); hasUser {
				c.JSON(403, gin.H{
					"error": gin.H{
						"code":    "TENANT_ISOLATION_VIOLATION",
						"message": "Tenant context is required for this request",
					},
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		tenantStr, ok := tenantID.(string)
		if !ok {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "INVALID_TENANT_CONTEXT",
					"message": "Invalid tenant context type",
				},
			})
			c.Abort()
			return
		}

		if _, err := uuid.Parse(tenantStr); err != nil {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "INVALID_TENANT_ID",
					"message": "Tenant ID must be a valid UUID",
				},
			})
			c.Abort()
			return
		}

		// SECURITY: Prevent tenant_id spoofing via query parameters
		if c.Query("tenant_id") != "" || c.Query("tenantId") != "" {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "TENANT_ISOLATION_VIOLATION",
					"message": "Tenant cannot be specified via query parameters",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
