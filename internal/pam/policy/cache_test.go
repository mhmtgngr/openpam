package policy

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPolicyCache(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("creates policy cache with nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		assert.NotNil(t, pc)
		assert.Nil(t, pc.cache)
	})
}

func TestPolicyCache_GetSetDeletePolicy(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("returns error when cache is not available", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)
		policyID := uuid.New()

		policy, err := pc.GetPolicy(ctx, policyID)

		assert.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "not available")
	})

	t.Run("deletes policy from cache does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)
		policyID := uuid.New()

		err := pc.DeletePolicy(ctx, policyID)
		assert.NoError(t, err) // Should silently succeed
	})

	t.Run("sets policy does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		policy := &Policy{
			ID:   uuid.New(),
			Name: "test-policy",
		}

		err := pc.SetPolicy(ctx, policy, time.Minute)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_GetSetPolicyList(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns error when cache is not available", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		policies, total, err := pc.GetPolicyList(ctx, tenantID, "test-filter")

		assert.Error(t, err)
		assert.Nil(t, policies)
		assert.Zero(t, total)
	})

	t.Run("sets policy list does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		policies := []Policy{
			{ID: uuid.New(), Name: "policy-1"},
			{ID: uuid.New(), Name: "policy-2"},
		}
		total := 2

		err := pc.SetPolicyList(ctx, tenantID, "test-filter", policies, total, time.Minute)
		assert.NoError(t, err) // Should silently succeed
	})

	t.Run("invalidates policy list does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		err := pc.InvalidPolicyList(ctx, tenantID)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_GetSetApplicablePolicies(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	resourceID := uuid.New()

	t.Run("sets and gets applicable policies does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		policies := []Policy{
			{ID: uuid.New(), Name: "applicable-policy-1"},
			{ID: uuid.New(), Name: "applicable-policy-2"},
		}

		err := pc.SetApplicablePolicies(ctx, tenantID, userID, resourceID, policies, time.Minute)
		assert.NoError(t, err) // Should silently succeed
	})

	t.Run("invalidates applicable policies does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		err := pc.InvalidateApplicablePolicies(ctx, tenantID)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_GetSetEvalResult(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("sets eval result does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		result := &EvaluationResponse{
			Allowed: true,
			Reason:  "Test evaluation",
		}

		err := pc.SetEvalResult(ctx, "test-key", result, 30*time.Second)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_InvalidateTenant(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("invalidates all tenant cache entries does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		err := pc.InvalidateTenant(ctx, tenantID)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_Warmup(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("warms up cache does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		policies := []Policy{
			{ID: uuid.New(), Name: "policy-1"},
			{ID: uuid.New(), Name: "policy-2"},
			{ID: uuid.New(), Name: "policy-3"},
		}

		err := pc.Warmup(ctx, tenantID, policies)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_InvalidateOnPolicyChange(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()
	policyID := uuid.New()

	t.Run("invalidates on policy change does nothing when nil cache", func(t *testing.T) {
		pc := NewPolicyCache(nil, logger)

		err := pc.InvalidateOnPolicyChange(ctx, tenantID, policyID)
		assert.NoError(t, err) // Should silently succeed
	})
}

func TestPolicyCache_KeyGeneration(t *testing.T) {
	logger := zerolog.Nop()
	pc := NewPolicyCache(nil, logger)

	t.Run("generates correct policy key", func(t *testing.T) {
		policyID := uuid.New()
		key := pc.policyKey(policyID)

		assert.Contains(t, key, "policy:")
		assert.Contains(t, key, policyID.String())
	})

	t.Run("generates correct policy list key", func(t *testing.T) {
		tenantID := uuid.New()
		filterKey := "test-filter"
		key := pc.policyListKey(tenantID, filterKey)

		assert.Contains(t, key, "policy:list:")
		assert.Contains(t, key, tenantID.String())
		assert.Contains(t, key, filterKey)
	})

	t.Run("generates correct applicable policies key with resource", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		resourceID := uuid.New()
		key := pc.applicablePoliciesKey(tenantID, userID, resourceID)

		assert.Contains(t, key, "policy:applicable:")
		assert.Contains(t, key, tenantID.String())
		assert.Contains(t, key, userID.String())
		assert.Contains(t, key, resourceID.String())
		assert.NotContains(t, key, "*")
	})

	t.Run("generates correct applicable policies key without resource", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		key := pc.applicablePoliciesKey(tenantID, userID, uuid.Nil)

		assert.Contains(t, key, "policy:applicable:")
		assert.Contains(t, key, tenantID.String())
		assert.Contains(t, key, userID.String())
		assert.Contains(t, key, "*")
	})

	t.Run("generates correct eval key", func(t *testing.T) {
		cacheKey := "test-cache-key"
		key := pc.evalKey(cacheKey)

		assert.Contains(t, key, "policy:eval:")
		assert.Contains(t, key, cacheKey)
	})
}

func TestGenerateEvalCacheKey(t *testing.T) {
	t.Run("generates consistent cache key for request", func(t *testing.T) {
		req := EvaluationRequest{
			TenantID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			UserID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
			Action:       "checkout",
			ResourceType: "credential",
			ResourceID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
		}

		key := GenerateEvalCacheKey(req)

		assert.Contains(t, key, req.TenantID.String())
		assert.Contains(t, key, req.UserID.String())
		assert.Contains(t, key, req.Action)
		assert.Contains(t, key, req.ResourceType)
		assert.Contains(t, key, req.ResourceID.String())
	})

	t.Run("generates different keys for different requests", func(t *testing.T) {
		req1 := EvaluationRequest{
			TenantID:     uuid.New(),
			UserID:       uuid.New(),
			Action:       "checkout",
			ResourceType: "credential",
			ResourceID:   uuid.New(),
		}

		req2 := EvaluationRequest{
			TenantID:     req1.TenantID,
			UserID:       req1.UserID,
			Action:       "checkin", // Different action
			ResourceType: req1.ResourceType,
			ResourceID:   req1.ResourceID,
		}

		key1 := GenerateEvalCacheKey(req1)
		key2 := GenerateEvalCacheKey(req2)

		assert.NotEqual(t, key1, key2)
	})
}

func TestPolicyCache_Constants(t *testing.T) {
	t.Run("has correct default TTL values", func(t *testing.T) {
		assert.Equal(t, 5*time.Minute, defaultPolicyTTL)
		assert.Equal(t, 2*time.Minute, defaultPolicyListTTL)
		assert.Equal(t, 30*time.Second, defaultEvalTTL)
		assert.Equal(t, 3*time.Minute, defaultApplicableTTL)
	})

	t.Run("has correct cache prefixes", func(t *testing.T) {
		assert.Equal(t, "policy:", policyCachePrefix)
		assert.Equal(t, "policy:list:", policyListCachePrefix)
		assert.Equal(t, "policy:eval:", policyEvalCachePrefix)
		assert.Equal(t, "policy:applicable:", applicablePoliciesPrefix)
	})

	t.Run("has correct invalidation channel", func(t *testing.T) {
		assert.Equal(t, "policy:invalidate", policyInvalidationChannel)
	})
}
