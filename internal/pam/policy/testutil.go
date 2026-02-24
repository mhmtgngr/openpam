package policy

import (
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// NewTestService creates a new policy service for testing with injectable dependencies
// This function is intended for testing purposes only
func NewTestService(repo RepositoryInterface, cache CacheInterface, eventBus *events.EventBus, logger zerolog.Logger) *Service {
	return &Service{
		repo:     repo,
		cache:    cache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}
}

// NewServiceWithCache creates a new policy service with a specific cache implementation
func NewServiceWithCache(repo RepositoryInterface, c CacheInterface, eventBus *events.EventBus, logger zerolog.Logger) *Service {
	return &Service{
		repo:     repo,
		cache:    c,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
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
