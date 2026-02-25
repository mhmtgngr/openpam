package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

const (
	// Cache key prefixes
	policyCachePrefix        = "policy:"
	policyListCachePrefix    = "policy:list:"
	policyEvalCachePrefix    = "policy:eval:"
	applicablePoliciesPrefix = "policy:applicable:"

	// Default TTL values
	defaultPolicyTTL       = 5 * time.Minute
	defaultPolicyListTTL   = 2 * time.Minute
	defaultEvalTTL         = 30 * time.Second
	defaultApplicableTTL   = 3 * time.Minute

	// Cache invalidation channel
	policyInvalidationChannel = "policy:invalidate"
)

// CacheInterface defines the policy cache interface
type CacheInterface interface {
	GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error)
	SetPolicy(ctx context.Context, policy *Policy, ttl time.Duration) error
	DeletePolicy(ctx context.Context, id uuid.UUID) error

	GetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string) ([]Policy, int, error)
	SetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string, policies []Policy, total int, ttl time.Duration) error
	InvalidPolicyList(ctx context.Context, tenantID uuid.UUID) error

	GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, resourceID uuid.UUID) ([]Policy, error)
	SetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, resourceID uuid.UUID, policies []Policy, ttl time.Duration) error
	InvalidateApplicablePolicies(ctx context.Context, tenantID uuid.UUID) error

	GetEvalResult(ctx context.Context, cacheKey string) (*EvaluationResponse, error)
	SetEvalResult(ctx context.Context, cacheKey string, result *EvaluationResponse, ttl time.Duration) error

	InvalidateTenant(ctx context.Context, tenantID uuid.UUID) error
	InvalidateOnPolicyChange(ctx context.Context, tenantID uuid.UUID, policyID uuid.UUID) error
	Warmup(ctx context.Context, tenantID uuid.UUID, policies []Policy) error
}

// PolicyCache implements the policy cache with Redis
type PolicyCache struct {
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewPolicyCache creates a new policy cache
func NewPolicyCache(c *cache.Cache, logger zerolog.Logger) *PolicyCache {
	pc := &PolicyCache{
		cache:  c,
		logger: logger,
	}

	// Start invalidation listener only if cache is available
	if c != nil {
		go pc.listenForInvalidations()
	}

	return pc
}

// GetPolicy retrieves a policy from cache
func (c *PolicyCache) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	if c.cache == nil {
		return nil, errors.New("cache: not available")
	}
	key := c.policyKey(id)

	var policy Policy
	err := c.cache.Get(ctx, key, &policy)
	if err != nil {
		return nil, err // Key not found or error
	}

	return &policy, nil
}

// SetPolicy stores a policy in cache
func (c *PolicyCache) SetPolicy(ctx context.Context, policy *Policy, ttl time.Duration) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	key := c.policyKey(policy.ID)

	if ttl == 0 {
		ttl = defaultPolicyTTL
	}

	return c.cache.Set(ctx, key, policy, ttl)
}

// DeletePolicy removes a policy from cache
func (c *PolicyCache) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	key := c.policyKey(id)
	return c.cache.Delete(ctx, key)
}

// GetPolicyList retrieves a policy list from cache
func (c *PolicyCache) GetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string) ([]Policy, int, error) {
	if c.cache == nil {
		return nil, 0, errors.New("cache: not available")
	}
	key := c.policyListKey(tenantID, filterKey)

	var cached struct {
		Policies []Policy `json:"policies"`
		Total    int      `json:"total"`
	}

	err := c.cache.Get(ctx, key, &cached)
	if err != nil {
		return nil, 0, err
	}

	return cached.Policies, cached.Total, nil
}

// SetPolicyList stores a policy list in cache
func (c *PolicyCache) SetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string, policies []Policy, total int, ttl time.Duration) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	key := c.policyListKey(tenantID, filterKey)

	if ttl == 0 {
		ttl = defaultPolicyListTTL
	}

	cached := struct {
		Policies []Policy `json:"policies"`
		Total    int      `json:"total"`
	}{
		Policies: policies,
		Total:    total,
	}

	return c.cache.Set(ctx, key, cached, ttl)
}

// InvalidatePolicyList removes policy lists for a tenant from cache
func (c *PolicyCache) InvalidPolicyList(ctx context.Context, tenantID uuid.UUID) error {
	if c.cache == nil {
		return nil
	}
	pattern := c.policyListKey(tenantID, "*")
	return c.cache.DeleteByPattern(ctx, pattern)
}

// GetApplicablePolicies retrieves applicable policies from cache
func (c *PolicyCache) GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, resourceID uuid.UUID) ([]Policy, error) {
	if c.cache == nil {
		return nil, errors.New("cache: not available")
	}
	key := c.applicablePoliciesKey(tenantID, userID, resourceID)

	var policies []Policy
	err := c.cache.Get(ctx, key, &policies)
	if err != nil {
		return nil, err
	}

	return policies, nil
}

// SetApplicablePolicies stores applicable policies in cache
func (c *PolicyCache) SetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, resourceID uuid.UUID, policies []Policy, ttl time.Duration) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	key := c.applicablePoliciesKey(tenantID, userID, resourceID)

	if ttl == 0 {
		ttl = defaultApplicableTTL
	}

	return c.cache.Set(ctx, key, policies, ttl)
}

// InvalidateApplicablePolicies removes applicable policy caches for a tenant
func (c *PolicyCache) InvalidateApplicablePolicies(ctx context.Context, tenantID uuid.UUID) error {
	if c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("%s%s:*", applicablePoliciesPrefix, tenantID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// GetEvalResult retrieves an evaluation result from cache
func (c *PolicyCache) GetEvalResult(ctx context.Context, cacheKey string) (*EvaluationResponse, error) {
	if c.cache == nil {
		return nil, errors.New("cache: not available")
	}
	key := c.evalKey(cacheKey)

	var result EvaluationResponse
	err := c.cache.Get(ctx, key, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// SetEvalResult stores an evaluation result in cache
func (c *PolicyCache) SetEvalResult(ctx context.Context, cacheKey string, result *EvaluationResponse, ttl time.Duration) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	key := c.evalKey(cacheKey)

	if ttl == 0 {
		ttl = defaultEvalTTL
	}

	return c.cache.Set(ctx, key, result, ttl)
}

// InvalidateTenant removes all policy-related cache entries for a tenant
func (c *PolicyCache) InvalidateTenant(ctx context.Context, tenantID uuid.UUID) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	// Invalidate all cache keys for this tenant
	patterns := []string{
		c.policyListKey(tenantID, "*"),
		fmt.Sprintf("%s%s:*", applicablePoliciesPrefix, tenantID),
	}

	for _, pattern := range patterns {
		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			c.logger.Error().Err(err).Str("pattern", pattern).Msg("Failed to delete cache pattern")
		}
	}

	// Publish invalidation event
	c.publishInvalidation(ctx, tenantID)

	return nil
}

// Warmup preloads policies for a tenant into cache
func (c *PolicyCache) Warmup(ctx context.Context, tenantID uuid.UUID, policies []Policy) error {
	for _, policy := range policies {
		if err := c.SetPolicy(ctx, &policy, defaultPolicyTTL); err != nil {
			c.logger.Error().
				Err(err).
				Str("policy_id", policy.ID.String()).
				Msg("Failed to warm up policy cache")
		}
	}

	c.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("count", len(policies)).
		Msg("Policy cache warmed up")

	return nil
}

// publishInvalidation publishes a cache invalidation event
func (c *PolicyCache) publishInvalidation(ctx context.Context, tenantID uuid.UUID) {
	event := cache.Event{
		Type:      "policy.invalidated",
		TenantID:  tenantID.String(),
		Data:      map[string]interface{}{"tenant_id": tenantID.String()},
		Timestamp: time.Now().Unix(),
	}

	channel := fmt.Sprintf("policies:%s", tenantID)
	if err := c.cache.PubSub().Publish(ctx, channel, event); err != nil {
		c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("Failed to publish invalidation")
	}
}

// listenForInvalidations listens for cache invalidation events
func (c *PolicyCache) listenForInvalidations() {
	ctx := context.Background()
	pattern := "policies:*"

	// Check if cache is available
	if c.cache == nil || !c.cache.IsAvailable() {
		return
	}

	pubsub, err := c.cache.PubSub().Subscribe(ctx, pattern)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to subscribe to policy invalidations")
		return
	}
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var event cache.Event
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			c.logger.Error().Err(err).Msg("Failed to unmarshal invalidation event")
			continue
		}

		c.logger.Debug().
			Str("type", event.Type).
			Str("tenant_id", event.TenantID).
			Msg("Received policy invalidation")

		// Invalidate local cache for this tenant
		if tenantIDStr, ok := event.Data["tenant_id"].(string); ok {
			if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
				ctx := context.Background()
				_ = c.InvalidateApplicablePolicies(ctx, tenantID)
				_ = c.InvalidPolicyList(ctx, tenantID)
			}
		}
	}
}

// Key generation helpers

func (c *PolicyCache) policyKey(id uuid.UUID) string {
	return fmt.Sprintf("%s%s", policyCachePrefix, id.String())
}

func (c *PolicyCache) policyListKey(tenantID uuid.UUID, filterKey string) string {
	return fmt.Sprintf("%s%s:%s", policyListCachePrefix, tenantID.String(), filterKey)
}

func (c *PolicyCache) applicablePoliciesKey(tenantID, userID, resourceID uuid.UUID) string {
	if resourceID == uuid.Nil {
		return fmt.Sprintf("%s%s:%s:*", applicablePoliciesPrefix, tenantID.String(), userID.String())
	}
	return fmt.Sprintf("%s%s:%s:%s", applicablePoliciesPrefix, tenantID.String(), userID.String(), resourceID.String())
}

func (c *PolicyCache) evalKey(cacheKey string) string {
	return fmt.Sprintf("%s%s", policyEvalCachePrefix, cacheKey)
}

// GenerateEvalCacheKey generates a cache key for evaluation results
func GenerateEvalCacheKey(req EvaluationRequest) string {
	// Create a hashable key from the request
	// In production, use proper hashing
	return fmt.Sprintf("%s:%s:%s:%s:%s",
		req.TenantID.String(),
		req.UserID.String(),
		req.Action,
		req.ResourceType,
		req.ResourceID.String())
}

// InvalidateOnPolicyChange handles cache invalidation when a policy changes
func (c *PolicyCache) InvalidateOnPolicyChange(ctx context.Context, tenantID uuid.UUID, policyID uuid.UUID) error {
	if c.cache == nil {
		return nil // Silently fail if cache not available
	}
	// Delete the specific policy
	if err := c.DeletePolicy(ctx, policyID); err != nil {
		c.logger.Error().Err(err).Str("policy_id", policyID.String()).Msg("Failed to delete policy from cache")
	}

	// Invalidate all lists and applicable caches
	if err := c.InvalidPolicyList(ctx, tenantID); err != nil {
		c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("Failed to invalidate policy lists")
	}

	if err := c.InvalidateApplicablePolicies(ctx, tenantID); err != nil {
		c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("Failed to invalidate applicable policies")
	}

	// Publish invalidation
	c.publishInvalidation(ctx, tenantID)

	return nil
}
