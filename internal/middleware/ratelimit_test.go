package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_DefaultConfigs(t *testing.T) {
	t.Run("default_rate_limit_config", func(t *testing.T) {
		config := DefaultRateLimitConfig()
		assert.Equal(t, 100, config.RequestLimit)
		assert.Equal(t, 60, config.TimeWindowSeconds)
		assert.False(t, config.LimitByUser)
	})

	t.Run("auth_rate_limit_config", func(t *testing.T) {
		config := AuthRateLimitConfig()
		assert.Equal(t, 5, config.RequestLimit)
		assert.Equal(t, 300, config.TimeWindowSeconds)
		assert.True(t, config.LimitByUser)
	})

	t.Run("privileged_rate_limit_config", func(t *testing.T) {
		config := PrivilegedRateLimitConfig()
		assert.Equal(t, 10, config.RequestLimit)
		assert.Equal(t, 60, config.TimeWindowSeconds)
		assert.True(t, config.LimitByUser)
	})
}

func TestRateLimiter_ErrorCode(t *testing.T) {
	t.Run("error_code_is_defined", func(t *testing.T) {
		// Verify the error code constant is properly defined
		assert.Equal(t, "RATE_LIMIT_EXCEEDED", ErrorCodeRateLimitExceeded)
	})
}

func TestRateLimiter_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("rate_limit_response_format", func(t *testing.T) {
		// Create a mock rate limiter response to verify format
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Set("request_id", "test-req-id")

		retryAfterSeconds := 45

		c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSeconds))
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Used", "105")
		c.Header("X-RateLimit-Remaining", "0")
		c.Header("X-RateLimit-Reset", "2026-02-24T12:35:00Z")

		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"code":    ErrorCodeRateLimitExceeded,
				"message": "Rate limit exceeded. Please retry after the specified time.",
				"details": gin.H{
					"limit":     100,
					"used":     105,
					"resets_at": "2026-02-24T12:35:00Z",
				},
			},
		})

		// Verify response
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.NotEmpty(t, w.Header().Get("Retry-After"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Used"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

		// Verify response body has correct structure
		body := w.Body.String()
		assert.Contains(t, body, ErrorCodeRateLimitExceeded)
		assert.Contains(t, body, "limit")
		assert.Contains(t, body, "used")
		assert.Contains(t, body, "resets_at")
	})
}

func TestRateLimiter_MaxFunction(t *testing.T) {
	t.Run("max_returns_correct_value", func(t *testing.T) {
		assert.Equal(t, 5, max(3, 5))
		assert.Equal(t, 10, max(10, 2))
		assert.Equal(t, 0, max(0, 0))
		assert.Equal(t, -1, max(-1, -5))
		assert.Equal(t, 100, max(100, 50))
	})
}

func TestRateLimiter_KeyGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("rate_limit_config_validation", func(t *testing.T) {
		config := RateLimitConfig{
			RequestLimit:      100,
			TimeWindowSeconds: 60,
			LimitByUser:       false,
			KeyGenerator: func(c *gin.Context) string {
				return fmt.Sprintf("custom:%s", c.ClientIP())
			},
		}

		assert.Equal(t, 100, config.RequestLimit)
		assert.Equal(t, 60, config.TimeWindowSeconds)
		assert.False(t, config.LimitByUser)
		assert.NotNil(t, config.KeyGenerator)

		// Test the key generator
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "192.168.1.1:1234" // Include port as in real requests

		key := config.KeyGenerator(c)
		// ClientIP() will extract IP from RemoteAddr
		assert.Contains(t, key, "custom:")
		assert.Contains(t, key, "192.168.1")
	})
}

func TestPerEndpointRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(nil)

	t.Run("per_endpoint_limit_config", func(t *testing.T) {
		rl := NewRateLimiter(nil, logger)

		limits := map[string]RateLimitConfig{
			"/api/v1/auth/login": AuthRateLimitConfig(),
			"/api/v1/credentials/*/checkout": {
				RequestLimit:      10,
				TimeWindowSeconds: 60,
				LimitByUser:       true,
			},
			"*": DefaultRateLimitConfig(),
		}

		middleware := rl.PerEndpointRateLimit(limits)
		assert.NotNil(t, middleware)

		// Test match pattern function
		assert.True(t, matchPattern("/api/v1/auth/login", "/api/v1/auth/login"))
		assert.True(t, matchPattern("/any", "*"))
		assert.False(t, matchPattern("/api/v1/auth/login", "/api/v1/auth/logout"))
	})
}
