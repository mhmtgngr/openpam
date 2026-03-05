package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

const (
	// ErrorCodeRateLimitExceeded is the error code for rate limiting (1308)
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Requests per time window
	RequestLimit int
	// Time window in seconds
	TimeWindowSeconds int
	// Whether to limit per-IP or per-user
	LimitByUser bool
	// Custom key generator function
	KeyGenerator func(*gin.Context) string
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestLimit:      100,
		TimeWindowSeconds: 60,
		LimitByUser:       false, // Default to IP-based limiting
	}
}

// AuthRateLimitConfig returns stricter rate limit for auth endpoints
func AuthRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestLimit:      5, // 5 attempts
		TimeWindowSeconds: 300, // per 5 minutes
		LimitByUser:       true,
	}
}

// PrivilegedRateLimitConfig returns rate limit for privileged operations
func PrivilegedRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestLimit:      10,
		TimeWindowSeconds: 60,
		LimitByUser:       true,
	}
}

// ProtectedRateLimitConfig returns rate limit for authenticated/protected endpoints
// SECURITY: Prevents authenticated users from abusing API endpoints
func ProtectedRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestLimit:      60, // 60 requests
		TimeWindowSeconds: 60, // per minute per user
		LimitByUser:       true,
	}
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cache *cache.Cache, logger zerolog.Logger) *RateLimiter {
	return &RateLimiter{
		cache:  cache,
		logger: logger,
	}
}

// RateLimit returns a rate limiting middleware
func (rl *RateLimiter) RateLimit(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := rl.getKey(c, config)

		// Get current count and window info
		count, _, windowEnd, err := rl.getRequestCount(ctx, key, config.TimeWindowSeconds)
		if err != nil {
			rl.logger.Error().Err(err).Str("key", key).Msg("Rate limit check failed")
			// Fail open - allow request if rate limit check fails
			c.Next()
			return
		}

		// Check if limit exceeded
		if count > config.RequestLimit {
			requestID, _ := c.Get("request_id")
			rl.logger.Warn().
				Str("request_id", requestID.(string)).
				Str("key", key).
				Int("count", count).
				Int("limit", config.RequestLimit).
				Str("path", c.Request.URL.Path).
				Msg("Rate limit exceeded")

			// Calculate seconds until window resets
			retryAfterSeconds := int(time.Until(windowEnd).Seconds())
			if retryAfterSeconds < 1 {
				retryAfterSeconds = 1
			}

			// Set standard HTTP 429 Retry-After header
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))

			// Add rate limit info headers even when exceeded
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestLimit))
			c.Header("X-RateLimit-Used", strconv.Itoa(count))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", windowEnd.Format(time.RFC3339))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    ErrorCodeRateLimitExceeded,
					"message": "Rate limit exceeded. Please retry after the specified time.",
					"details": gin.H{
						"limit":     config.RequestLimit,
						"used":     count,
						"resets_at": windowEnd.Format(time.RFC3339),
					},
				},
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestLimit))
		c.Header("X-RateLimit-Used", strconv.Itoa(count))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, config.RequestLimit-count)))
		c.Header("X-RateLimit-Reset", windowEnd.Format(time.RFC3339))

		c.Next()
	}
}

// getKey generates the rate limit key based on configuration
func (rl *RateLimiter) getKey(c *gin.Context, config RateLimitConfig) string {
	if config.KeyGenerator != nil {
		return config.KeyGenerator(c)
	}

	if config.LimitByUser {
		// Use user ID if available, otherwise fall back to IP
		if userID, exists := c.Get("user_id"); exists {
			return fmt.Sprintf("ratelimit:user:%s", userID)
		}
		if tenantID, exists := c.Get("tenant_id"); exists {
			return fmt.Sprintf("ratelimit:tenant:%s", tenantID)
		}
	}

	// Default to IP-based limiting
	return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
}

// getRequestCount gets the current request count and window boundaries for a key
func (rl *RateLimiter) getRequestCount(ctx context.Context, key string, windowSeconds int) (count int, windowStart, windowEnd time.Time, err error) {
	now := time.Now()
	windowStart = now.Truncate(time.Duration(windowSeconds) * time.Second)
	windowEnd = windowStart.Add(time.Duration(windowSeconds) * time.Second)

	cacheKey := fmt.Sprintf("%s:%d", key, windowStart.Unix())

	// Try to increment existing counter
	currentVal, err := rl.cache.Increment(ctx, cacheKey, 1)
	if err != nil {
		// Key doesn't exist, create it
		if err := rl.cache.Set(ctx, cacheKey, 1, time.Duration(windowSeconds)*time.Second); err != nil {
			return 0, windowStart, windowEnd, err
		}
		return 1, windowStart, windowEnd, nil
	}

	return int(currentVal), windowStart, windowEnd, nil
}

// incrementRequest increments the request counter for a key
// Deprecated: Use getRequestCount for better window tracking
func (rl *RateLimiter) incrementRequest(ctx context.Context, key string, windowSeconds int) (int, error) {
	cacheKey := fmt.Sprintf("%s:%d", key, time.Now().Unix()/int64(windowSeconds))

	// Try to increment existing counter
	currentVal, err := rl.cache.Increment(ctx, cacheKey, 1)
	if err != nil {
		// Key doesn't exist, create it
		if err := rl.cache.Set(ctx, cacheKey, 1, time.Duration(windowSeconds)*time.Second); err != nil {
			return 0, err
		}
		return 1, nil
	}

	return int(currentVal), nil
}

// TenantRateLimit creates a rate limiter that limits per-tenant
func (rl *RateLimiter) TenantRateLimit(requestsPerMinute int) gin.HandlerFunc {
	config := RateLimitConfig{
		RequestLimit:      requestsPerMinute,
		TimeWindowSeconds: 60,
		LimitByUser:       false,
		KeyGenerator: func(c *gin.Context) string {
			if tenantID, exists := c.Get("tenant_id"); exists {
				return fmt.Sprintf("ratelimit:tenant:%s", tenantID)
			}
			return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
		},
	}
	return rl.RateLimit(config)
}

// PerEndpointRateLimit creates different rate limits for different endpoints
func (rl *RateLimiter) PerEndpointRateLimit(limits map[string]RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Find matching limit config
		var config RateLimitConfig
		for pattern, cfg := range limits {
			if matchPattern(path, pattern) {
				config = cfg
				break
			}
		}

		// If no match, use default
		if config.RequestLimit == 0 {
			config = DefaultRateLimitConfig()
		}

		// Apply the limit
		handler := rl.RateLimit(config)
		handler(c)
	}
}

// matchPattern checks if a path matches a pattern (supports wildcards)
func matchPattern(path, pattern string) bool {
	if pattern == path {
		return true
	}
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	// TODO: Add more sophisticated pattern matching
	return false
}

// SlidingWindowRateLimit implements sliding window rate limiting
// This is more accurate than fixed window but requires more storage
func (rl *RateLimiter) SlidingWindowRateLimit(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := rl.getKey(c, config)

		// In a real implementation, you'd store timestamps in Redis sorted set
		// For simplicity, we'll use a basic counter approach here

		count, _, windowEnd, err := rl.getRequestCount(ctx, key, config.TimeWindowSeconds)
		if err != nil {
			rl.logger.Error().Err(err).Str("key", key).Msg("Sliding window rate limit check failed")
			c.Next()
			return
		}

		if count > config.RequestLimit {
			// Calculate seconds until window resets
			retryAfterSeconds := int(time.Until(windowEnd).Seconds())
			if retryAfterSeconds < 1 {
				retryAfterSeconds = 1
			}

			// Set standard HTTP 429 Retry-After header
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))

			// Add rate limit info headers even when exceeded
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestLimit))
			c.Header("X-RateLimit-Used", strconv.Itoa(count))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", windowEnd.Format(time.RFC3339))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    ErrorCodeRateLimitExceeded,
					"message": "Rate limit exceeded. Please retry after the specified time.",
					"details": gin.H{
						"limit":     config.RequestLimit,
						"used":     count,
						"resets_at": windowEnd.Format(time.RFC3339),
					},
				},
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestLimit))
		c.Header("X-RateLimit-Used", strconv.Itoa(count))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, config.RequestLimit-count)))
		c.Header("X-RateLimit-Reset", windowEnd.Format(time.RFC3339))

		c.Next()
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
