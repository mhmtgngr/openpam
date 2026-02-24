package policy

import (
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// NewTestService creates a new policy service for testing with injectable dependencies
// This function is intended for testing purposes only and uses allow-by-default for tests
func NewTestService(repo RepositoryInterface, cache CacheInterface, eventBus *events.EventBus, logger zerolog.Logger) *Service {
	return &Service{
		repo:          repo,
		cache:         cache,
		evaluator:     NewEvaluatorWithConfig(logger, false), // Allow by default for tests
		eventBus:      eventBus,
		logger:        logger,
		denyByDefault: false,
	}
}

// NewTestServiceWithDenyByDefault creates a new policy service for testing with specified default behavior
func NewTestServiceWithDenyByDefault(repo RepositoryInterface, cache CacheInterface, eventBus *events.EventBus, logger zerolog.Logger, denyByDefault bool) *Service {
	return &Service{
		repo:          repo,
		cache:         cache,
		evaluator:     NewEvaluatorWithConfig(logger, denyByDefault),
		eventBus:      eventBus,
		logger:        logger,
		denyByDefault: denyByDefault,
	}
}

// NewServiceWithCache creates a new policy service with a specific cache implementation
func NewServiceWithCache(repo RepositoryInterface, c CacheInterface, eventBus *events.EventBus, logger zerolog.Logger) *Service {
	return &Service{
		repo:          repo,
		cache:         c,
		evaluator:     NewEvaluator(logger), // Uses default deny-by-default for security
		eventBus:      eventBus,
		logger:        logger,
		denyByDefault: true,
	}
}

// GetUnderlyingCache returns the underlying cache from a PolicyCache
// This is useful for testing when you need to access the wrapped cache
func GetUnderlyingCache(pc *PolicyCache) *cache.Cache {
	if pc != nil {
		return pc.cache
	}
	return nil
}
