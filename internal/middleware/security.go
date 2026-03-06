package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

// CORS returns a gin middleware for CORS handling
func CORS(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowed := false
		for _, allowedOrigin := range cfg.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			if cfg.AllowedOrigins[0] == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Header("Access-Control-Allow-Methods", join(cfg.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", join(cfg.AllowedHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", join(cfg.ExposeHeaders, ", "))
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// BodyLimit limits the size of request body
func BodyLimit(maxBody int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBody {
			c.JSON(413, gin.H{
				"error": gin.H{
					"code":    "REQUEST_TOO_LARGE",
					"message": "Request body exceeds maximum allowed size",
				},
			})
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
		c.Next()
	}
}

// ContentType validates JSON content type for write operations
func ContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "POST" || method == "PUT" || method == "PATCH" {
			ct := c.Request.Header.Get("Content-Type")
			if ct != "application/json" {
				c.JSON(415, gin.H{
					"error": gin.H{
						"code":    "UNSUPPORTED_MEDIA_TYPE",
						"message": "Content-Type must be application/json",
					},
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// PasswordComplexityConfig validates password complexity
// SECURITY: Enforces strong password policies
type PasswordComplexityConfig struct {
	MinLength           int
	RequireUppercase    bool
	RequireLowercase    bool
	RequireNumbers      bool
	RequireSpecialChars bool
	ForbiddenPasswords  []string
	ForbiddenPatterns   []string
}

// DefaultPasswordComplexityConfig returns secure defaults
func DefaultPasswordComplexityConfig() PasswordComplexityConfig {
	return PasswordComplexityConfig{
		MinLength:           12,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
		ForbiddenPasswords: []string{
			"password", "Password1!", "Admin123!", "Welcome123!",
		},
		ForbiddenPatterns: []string{
			"123456", "qwerty", "asdfgh", "zxcvbn",
		},
	}
}
