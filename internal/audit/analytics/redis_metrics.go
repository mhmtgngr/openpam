// Package analytics provides Redis-backed caching for analytics metrics
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// RedisMetricsCache provides Redis-backed caching for analytics data
type RedisMetricsCache struct {
	client    *redis.Client
	logger    zerolog.Logger
	keyPrefix string
}

// NewRedisMetricsCache creates a new Redis metrics cache
func NewRedisMetricsCache(client *redis.Client, logger zerolog.Logger) *RedisMetricsCache {
	return &RedisMetricsCache{
		client:    client,
		logger:    logger,
		keyPrefix: "analytics:",
	}
}

// SessionMetricsCacheDuration is how long session metrics are cached
const SessionMetricsCacheDuration = 24 * time.Hour

// BaselineCacheDuration is how long baselines are cached
const BaselineCacheDuration = 1 * time.Hour

// StoreSessionMetrics stores session metrics in Redis
func (r *RedisMetricsCache) StoreSessionMetrics(ctx context.Context, metrics *SessionMetricsFromStore) error {
	key := r.sessionMetricsKey(metrics.SessionID)

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("redis_metrics.StoreSessionMetrics: marshal: %w", err)
	}

	if err := r.client.Set(ctx, key, data, SessionMetricsCacheDuration).Err(); err != nil {
		return fmt.Errorf("redis_metrics.StoreSessionMetrics: set: %w", err)
	}

	return nil
}

// GetSessionMetrics retrieves session metrics from Redis
func (r *RedisMetricsCache) GetSessionMetrics(ctx context.Context, sessionID uuid.UUID) (*SessionMetricsFromStore, error) {
	key := r.sessionMetricsKey(sessionID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("redis_metrics.GetSessionMetrics: get: %w", err)
	}

	var metrics SessionMetricsFromStore
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("redis_metrics.GetSessionMetrics: unmarshal: %w", err)
	}

	return &metrics, nil
}

// StoreUserBaseline stores a user baseline in Redis
func (r *RedisMetricsCache) StoreUserBaseline(ctx context.Context, baseline *UserBaseline) error {
	key := r.userBaselineKey(baseline.TenantID, baseline.UserID)

	data, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("redis_metrics.StoreUserBaseline: marshal: %w", err)
	}

	if err := r.client.Set(ctx, key, data, BaselineCacheDuration).Err(); err != nil {
		return fmt.Errorf("redis_metrics.StoreUserBaseline: set: %w", err)
	}

	return nil
}

// GetUserBaseline retrieves a user baseline from Redis
func (r *RedisMetricsCache) GetUserBaseline(ctx context.Context, tenantID, userID uuid.UUID) (*UserBaseline, error) {
	key := r.userBaselineKey(tenantID, userID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("redis_metrics.GetUserBaseline: get: %w", err)
	}

	var baseline UserBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("redis_metrics.GetUserBaseline: unmarshal: %w", err)
	}

	return &baseline, nil
}

// InvalidateUserBaseline removes a baseline from cache
func (r *RedisMetricsCache) InvalidateUserBaseline(ctx context.Context, tenantID, userID uuid.UUID) error {
	key := r.userBaselineKey(tenantID, userID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis_metrics.InvalidateUserBaseline: del: %w", err)
	}

	return nil
}

// StoreMetricsWindow stores a sliding window of metrics
func (r *RedisMetricsCache) StoreMetricsWindow(ctx context.Context, tenantID uuid.UUID, windowType string, metrics interface{}) error {
	key := r.metricsWindowKey(tenantID, windowType)

	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("redis_metrics.StoreMetricsWindow: marshal: %w", err)
	}

	// Store with shorter TTL for windowed data
	if err := r.client.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("redis_metrics.StoreMetricsWindow: set: %w", err)
	}

	return nil
}

// GetMetricsWindow retrieves a metrics window from Redis
func (r *RedisMetricsCache) GetMetricsWindow(ctx context.Context, tenantID uuid.UUID, windowType string, dest interface{}) error {
	key := r.metricsWindowKey(tenantID, windowType)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil // Not found
		}
		return fmt.Errorf("redis_metrics.GetMetricsWindow: get: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("redis_metrics.GetMetricsWindow: unmarshal: %w", err)
	}

	return nil
}

// UpdateSlidingWindowCounter updates a counter in a sliding time window
func (r *RedisMetricsCache) UpdateSlidingWindowCounter(ctx context.Context, key string, increment int64, window time.Duration) (int64, error) {
	// Use a sorted set for sliding window counter
	now := float64(time.Now().UnixMicro())
	member := fmt.Sprintf("%f", now)

	pipe := r.client.Pipeline()

	// Add current value
	pipe.ZAdd(ctx, key, redis.Z{Score: now, Member: member})

	// Remove old values outside the window
	cutoff := now - float64(window.Microseconds())
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", cutoff))

	// Count remaining values
	countCmd := pipe.ZCard(ctx, key)

	// Set expiration
	pipe.Expire(ctx, key, window+time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("redis_metrics.UpdateSlidingWindowCounter: pipeline: %w", err)
	}

	return countCmd.Val(), nil
}

// GetSlidingWindowCount returns the count in a sliding window
func (r *RedisMetricsCache) GetSlidingWindowCount(ctx context.Context, key string, window time.Duration) (int64, error) {
	// Clean old values first
	now := float64(time.Now().UnixMicro())
	cutoff := now - float64(window.Microseconds())
	r.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", cutoff))

	// Count remaining
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis_metrics.GetSlidingWindowCount: zcard: %w", err)
	}

	return count, nil
}

// StoreComplianceScore stores a compliance score in cache
func (r *RedisMetricsCache) StoreComplianceScore(ctx context.Context, tenantID uuid.UUID, framework string, score *ComplianceScore) error {
	key := r.complianceScoreKey(tenantID, framework)

	data, err := json.Marshal(score)
	if err != nil {
		return fmt.Errorf("redis_metrics.StoreComplianceScore: marshal: %w", err)
	}

	// Cache for 1 hour
	if err := r.client.Set(ctx, key, data, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis_metrics.StoreComplianceScore: set: %w", err)
	}

	return nil
}

// GetComplianceScore retrieves a compliance score from cache
func (r *RedisMetricsCache) GetComplianceScore(ctx context.Context, tenantID uuid.UUID, framework string) (*ComplianceScore, error) {
	key := r.complianceScoreKey(tenantID, framework)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("redis_metrics.GetComplianceScore: get: %w", err)
	}

	var score ComplianceScore
	if err := json.Unmarshal(data, &score); err != nil {
		return nil, fmt.Errorf("redis_metrics.GetComplianceScore: unmarshal: %w", err)
	}

	return &score, nil
}

// InvalidateTenantCache clears all cached data for a tenant
func (r *RedisMetricsCache) InvalidateTenantCache(ctx context.Context, tenantID uuid.UUID) error {
	pattern := r.keyPrefix + "tenant:" + tenantID.String() + ":*"

	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
	keys := []string{}

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis_metrics.InvalidateTenantCache: scan: %w", err)
	}

	if len(keys) > 0 {
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("redis_metrics.InvalidateTenantCache: del: %w", err)
		}
	}

	return nil
}

// GetOrSetWithLock implements a cache-aside pattern with distributed locking
func (r *RedisMetricsCache) GetOrSetWithLock(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	cached, err := r.client.Get(ctx, key).Result()
	if err == nil {
		var result interface{}
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Not in cache, acquire lock
	lockKey := key + ":lock"
	lockValue := uuid.New().String()

	// Try to acquire lock with 10 second timeout
	acquired, err := r.client.SetNX(ctx, lockKey, lockValue, 10*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("redis_metrics.GetOrSetWithLock: setnx: %w", err)
	}

	if acquired {
		defer r.releaseLock(ctx, lockKey, lockValue)

		// Generate the value
		result, err := fn()
		if err != nil {
			return nil, err
		}

		// Cache the result
		data, err := json.Marshal(result)
		if err == nil {
			r.client.Set(ctx, key, data, ttl)
		}

		return result, nil
	}

	// Wait for lock to be released and retry
	time.Sleep(100 * time.Millisecond)

	cached, err = r.client.Get(ctx, key).Result()
	if err == nil {
		var result interface{}
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("redis_metrics.GetOrSetWithLock: failed to acquire lock and retrieve value")
}

// releaseLock releases a distributed lock
func (r *RedisMetricsCache) releaseLock(ctx context.Context, key, value string) error {
	// Use Lua script to ensure atomic release
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	return r.client.Eval(ctx, script, []string{key}, value).Err()
}

// Key generation helpers

func (r *RedisMetricsCache) sessionMetricsKey(sessionID uuid.UUID) string {
	return fmt.Sprintf("%ssession:%s", r.keyPrefix, sessionID.String())
}

func (r *RedisMetricsCache) userBaselineKey(tenantID, userID uuid.UUID) string {
	return fmt.Sprintf("%stenant:%s:baseline:user:%s", r.keyPrefix, tenantID.String(), userID.String())
}

func (r *RedisMetricsCache) metricsWindowKey(tenantID uuid.UUID, windowType string) string {
	return fmt.Sprintf("%stenant:%s:window:%s", r.keyPrefix, tenantID.String(), windowType)
}

func (r *RedisMetricsCache) complianceScoreKey(tenantID uuid.UUID, framework string) string {
	return fmt.Sprintf("%stenant:%s:compliance:%s", r.keyPrefix, tenantID.String(), framework)
}

// SessionMetricsFromStore is used for Redis caching (simplified)
type SessionMetricsFromStore struct {
	SessionID        uuid.UUID `json:"session_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	UserID           uuid.UUID `json:"user_id"`
	TargetHost       string    `json:"target_host"`
	StartTime        time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	DurationSeconds  float64   `json:"duration_seconds"`
	CommandCount     int       `json:"command_count"`
	FailedCommands   int       `json:"failed_commands"`
	OffHoursAccess   bool      `json:"off_hours_access"`
	FailureRate      float64   `json:"failure_rate"`
}

// MetricsWindow represents a sliding window of metrics
type MetricsWindow struct {
	TenantID     uuid.UUID    `json:"tenant_id"`
	WindowStart  time.Time    `json:"window_start"`
	WindowEnd    time.Time    `json:"window_end"`
	MetricType   string       `json:"metric_type"`
	Data         json.RawMessage `json:"data"`
	LastUpdated  time.Time    `json:"last_updated"`
}
