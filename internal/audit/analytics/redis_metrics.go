// Package analytics provides Redis caching for audit service
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

// StoreUserBaseline stores a user baseline in Redis
func (r *RedisMetricsCache) StoreUserBaseline(ctx context.Context, baseline *UserBaseline) error {
	key := r.userBaselineKey(baseline.TenantID, baseline.UserID)

	data, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("redis_cache.StoreUserBaseline: marshal: %w", err)
	}

	if err := r.client.Set(ctx, key, data, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis_cache.StoreUserBaseline: set: %w", err)
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
		return nil, fmt.Errorf("redis_cache.GetUserBaseline: get: %w", err)
	}

	var baseline UserBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("redis_cache.GetUserBaseline: unmarshal: %w", err)
	}

	return &baseline, nil
}

// InvalidateUserBaseline removes a baseline from cache
func (r *RedisMetricsCache) InvalidateUserBaseline(ctx context.Context, tenantID, userID uuid.UUID) error {
	key := r.userBaselineKey(tenantID, userID)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis_cache.InvalidateUserBaseline: del: %w", err)
	}

	return nil
}

// StoreComplianceScore stores a compliance score in cache
func (r *RedisMetricsCache) StoreComplianceScore(ctx context.Context, tenantID uuid.UUID, framework string, score interface{}) error {
	key := r.complianceScoreKey(tenantID, framework)

	data, err := json.Marshal(score)
	if err != nil {
		return fmt.Errorf("redis_cache.StoreComplianceScore: marshal: %w", err)
	}

	if err := r.client.Set(ctx, key, data, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis_cache.StoreComplianceScore: set: %w", err)
	}

	return nil
}

// GetComplianceScore retrieves a compliance score from cache
func (r *RedisMetricsCache) GetComplianceScore(ctx context.Context, tenantID uuid.UUID, framework string) (interface{}, error) {
	key := r.complianceScoreKey(tenantID, framework)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("redis_cache.GetComplianceScore: get: %w", err)
	}

	var score interface{}
	if err := json.Unmarshal(data, &score); err != nil {
		return nil, fmt.Errorf("redis_cache.GetComplianceScore: unmarshal: %w", err)
	}

	return score, nil
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
		return fmt.Errorf("redis_cache.InvalidateTenantCache: scan: %w", err)
	}

	if len(keys) > 0 {
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("redis_cache.InvalidateTenantCache: del: %w", err)
		}
	}

	return nil
}

// UpdateSlidingWindowCounter updates a counter in a sliding time window
func (r *RedisMetricsCache) UpdateSlidingWindowCounter(ctx context.Context, key string, increment int64, window time.Duration) (int64, error) {
	now := float64(time.Now().UnixMicro())
	member := fmt.Sprintf("%f", now)

	pipe := r.client.Pipeline()

	pipe.ZAdd(ctx, key, redis.Z{Score: now, Member: member})

	cutoff := now - float64(window.Microseconds())
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", cutoff))

	countCmd := pipe.ZCard(ctx, key)

	pipe.Expire(ctx, key, window+time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("redis_cache.UpdateSlidingWindowCounter: pipeline: %w", err)
	}

	return countCmd.Val(), nil
}

// InvalidateComplianceCache clears compliance-related cache for a tenant
func (r *RedisMetricsCache) InvalidateComplianceCache(ctx context.Context, tenantID uuid.UUID) error {
	pattern := r.keyPrefix + "compliance:*"
	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
	keys := []string{}

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis_cache.InvalidateComplianceCache: scan: %w", err)
	}

	if len(keys) > 0 {
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("redis_cache.InvalidateComplianceCache: del: %w", err)
		}
	}

	return nil
}

// Key generation helpers
func (r *RedisMetricsCache) userBaselineKey(tenantID, userID uuid.UUID) string {
	return fmt.Sprintf("%stenant:%s:baseline:user:%s", r.keyPrefix, tenantID.String(), userID.String())
}

func (r *RedisMetricsCache) complianceScoreKey(tenantID uuid.UUID, framework string) string {
	return fmt.Sprintf("%stenant:%s:compliance:%s", r.keyPrefix, tenantID.String(), framework)
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
