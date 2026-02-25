// Package cache provides Redis caching layer for audit analytics
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// Cache TTL constants for analytics data
const (
	// DefaultMetricsTTL is the TTL for metrics cache (5 minutes)
	DefaultMetricsTTL = 5 * time.Minute

	// ComplianceReportTTL is the TTL for compliance reports (1 hour)
	ComplianceReportTTL = time.Hour

	// ExceptionCacheTTL is the TTL for compliance exceptions (15 minutes)
	ExceptionCacheTTL = 15 * time.Minute

	// AnomalyCacheTTL is the TTL for anomaly data (5 minutes)
	AnomalyCacheTTL = 5 * time.Minute

	// AnomalyStatsTTL is the TTL for anomaly stats (1 minute)
	AnomalyStatsTTL = time.Minute

	// SSHKeyCacheTTL is the TTL for SSH key analytics (10 minutes)
	SSHKeyCacheTTL = 10 * time.Minute

	// BlacklistCacheTTL is the TTL for command blacklist (15 minutes)
	BlacklistCacheTTL = 15 * time.Minute
)

// AnalyticsCache provides caching functionality for analytics data
type AnalyticsCache struct {
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewAnalyticsCache creates a new analytics cache
func NewAnalyticsCache(c *cache.Cache, logger zerolog.Logger) *AnalyticsCache {
	return &AnalyticsCache{
		cache:  c,
		logger: logger,
	}
}

// =============================================================================
// Compliance Report Caching
// =============================================================================

// GetComplianceReport retrieves a compliance report from cache
func (c *AnalyticsCache) GetComplianceReport(ctx context.Context, id uuid.UUID) (*model.ComplianceReport, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("audit:compliance:report:%s", id)

	var report model.ComplianceReport
	err := c.cache.Get(ctx, key, &report)
	if err != nil {
		return nil, err
	}

	return &report, nil
}

// SetComplianceReport stores a compliance report in cache
func (c *AnalyticsCache) SetComplianceReport(ctx context.Context, report *model.ComplianceReport) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:compliance:report:%s", report.ID)
	return c.cache.Set(ctx, key, report, ComplianceReportTTL)
}

// InvalidateComplianceReport invalidates a specific compliance report cache
func (c *AnalyticsCache) InvalidateComplianceReport(ctx context.Context, id uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:compliance:report:%s", id)
	return c.cache.Delete(ctx, key)
}

// InvalidateComplianceReports invalidates all compliance report cache for a tenant
func (c *AnalyticsCache) InvalidateComplianceReports(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("audit:compliance:report:*")
	return c.cache.DeleteByPattern(ctx, pattern)
}

// =============================================================================
// Compliance Exception Caching
// =============================================================================

// GetComplianceException retrieves a compliance exception from cache
func (c *AnalyticsCache) GetComplianceException(ctx context.Context, id uuid.UUID) (*model.ComplianceException, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("audit:compliance:exception:%s", id)

	var exception model.ComplianceException
	err := c.cache.Get(ctx, key, &exception)
	if err != nil {
		return nil, err
	}

	return &exception, nil
}

// SetComplianceException stores a compliance exception in cache
func (c *AnalyticsCache) SetComplianceException(ctx context.Context, exception *model.ComplianceException) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:compliance:exception:%s", exception.ID)
	return c.cache.Set(ctx, key, exception, ExceptionCacheTTL)
}

// InvalidateComplianceException invalidates a specific compliance exception cache
func (c *AnalyticsCache) InvalidateComplianceException(ctx context.Context, id uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:compliance:exception:%s", id)
	return c.cache.Delete(ctx, key)
}

// InvalidateComplianceExceptions invalidates all compliance exception cache for a tenant
func (c *AnalyticsCache) InvalidateComplianceExceptions(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("audit:compliance:exception:*")
	return c.cache.DeleteByPattern(ctx, pattern)
}

// =============================================================================
// Anomaly Detection Caching
// =============================================================================

// GetAnomaly retrieves an anomaly from cache
func (c *AnalyticsCache) GetAnomaly(ctx context.Context, id uuid.UUID) (*model.AnomalyDetection, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("audit:anomaly:%s", id)

	var anomaly model.AnomalyDetection
	err := c.cache.Get(ctx, key, &anomaly)
	if err != nil {
		return nil, err
	}

	return &anomaly, nil
}

// SetAnomaly stores an anomaly in cache
func (c *AnalyticsCache) SetAnomaly(ctx context.Context, anomaly *model.AnomalyDetection) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:anomaly:%s", anomaly.ID)
	return c.cache.Set(ctx, key, anomaly, AnomalyCacheTTL)
}

// InvalidateAnomaly invalidates a specific anomaly cache
func (c *AnalyticsCache) InvalidateAnomaly(ctx context.Context, id uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:anomaly:%s", id)
	return c.cache.Delete(ctx, key)
}

// GetAnomalyStats retrieves anomaly stats from cache
func (c *AnalyticsCache) GetAnomalyStats(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("audit:anomaly:stats:%s", tenantID)

	var stats map[string]int
	err := c.cache.Get(ctx, key, &stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// SetAnomalyStats stores anomaly stats in cache
func (c *AnalyticsCache) SetAnomalyStats(ctx context.Context, tenantID uuid.UUID, stats map[string]int) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("audit:anomaly:stats:%s", tenantID)
	return c.cache.Set(ctx, key, stats, AnomalyStatsTTL)
}

// InvalidateAnomalies invalidates all anomaly cache for a tenant
func (c *AnalyticsCache) InvalidateAnomalies(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("audit:anomaly:*")
	return c.cache.DeleteByPattern(ctx, pattern)
}

// =============================================================================
// SSH Key Analytics Caching
// =============================================================================

// GetSSHKeyAnalytics retrieves SSH key analytics from cache
func (c *AnalyticsCache) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date string) (*model.SSHKeyAnalytics, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("audit:ssh_key:%s:%s:%s", tenantID, sshKeyID, date)

	var analytics model.SSHKeyAnalytics
	err := c.cache.Get(ctx, key, &analytics)
	if err != nil {
		return nil, err
	}

	return &analytics, nil
}

// SetSSHKeyAnalytics stores SSH key analytics in cache
func (c *AnalyticsCache) SetSSHKeyAnalytics(ctx context.Context, analytics *model.SSHKeyAnalytics) error {
	if c == nil || c.cache == nil {
		return nil
	}
	date := analytics.Date.Format("2006-01-02")
	key := fmt.Sprintf("audit:ssh_key:%s:%s:%s", analytics.TenantID, analytics.SSHKeyID, date)
	return c.cache.Set(ctx, key, analytics, SSHKeyCacheTTL)
}

// InvalidateSSHKeyAnalytics invalidates SSH key analytics cache for a key
func (c *AnalyticsCache) InvalidateSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("audit:ssh_key:%s:%s:*", tenantID, sshKeyID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// =============================================================================
// Command Blacklist Caching
// =============================================================================

// GetCommandBlacklist retrieves command blacklist from cache
func (c *AnalyticsCache) GetCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]model.CommandBlacklist, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	var key string
	if tenantID != nil {
		key = fmt.Sprintf("audit:blacklist:%s", *tenantID)
	} else {
		key = "audit:blacklist:global"
	}

	var blacklist []model.CommandBlacklist
	err := c.cache.Get(ctx, key, &blacklist)
	if err != nil {
		return nil, err
	}

	return blacklist, nil
}

// SetCommandBlacklist stores command blacklist in cache
func (c *AnalyticsCache) SetCommandBlacklist(ctx context.Context, tenantID *uuid.UUID, blacklist []model.CommandBlacklist) error {
	if c == nil || c.cache == nil {
		return nil
	}

	var key string
	if tenantID != nil {
		key = fmt.Sprintf("audit:blacklist:%s", *tenantID)
	} else {
		key = "audit:blacklist:global"
	}

	return c.cache.Set(ctx, key, blacklist, BlacklistCacheTTL)
}

// InvalidateCommandBlacklist invalidates command blacklist cache
func (c *AnalyticsCache) InvalidateCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}

	var pattern string
	if tenantID != nil {
		pattern = fmt.Sprintf("audit:blacklist:%s", *tenantID)
	} else {
		pattern = "audit:blacklist:*"
	}

	return c.cache.DeleteByPattern(ctx, pattern)
}

// =============================================================================
// General Cache Management
// =============================================================================

// InvalidateAll invalidates all analytics cache for a tenant
func (c *AnalyticsCache) InvalidateAll(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}

	patterns := []string{
		fmt.Sprintf("audit:compliance:*:%s*", tenantID),
		fmt.Sprintf("audit:anomaly:*:%s*", tenantID),
		fmt.Sprintf("audit:ssh_key:%s:*", tenantID),
		fmt.Sprintf("audit:blacklist:%s", tenantID),
	}

	for _, pattern := range patterns {
		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			c.logger.Warn().Err(err).Str("pattern", pattern).Msg("Failed to invalidate cache pattern")
		}
	}

	return nil
}

// BatchInvalidation invalidates multiple cache entry types
func (c *AnalyticsCache) BatchInvalidation(ctx context.Context, tenantID uuid.UUID, cacheTypes ...string) error {
	if c == nil || c.cache == nil {
		return nil
	}

	for _, cacheType := range cacheTypes {
		var pattern string
		switch cacheType {
		case "compliance":
			pattern = fmt.Sprintf("audit:compliance:*")
		case "compliance_report":
			pattern = "audit:compliance:report:*"
		case "compliance_exception":
			pattern = "audit:compliance:exception:*"
		case "anomaly":
			pattern = fmt.Sprintf("audit:anomaly:*")
		case "ssh_key":
			pattern = fmt.Sprintf("audit:ssh_key:%s:*", tenantID)
		case "blacklist":
			pattern = fmt.Sprintf("audit:blacklist:*")
		default:
			pattern = fmt.Sprintf("audit:*:%s*", tenantID)
		}

		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			c.logger.Warn().Err(err).Str("pattern", pattern).Msg("Failed to invalidate cache pattern")
		}
	}

	return nil
}

// GetCacheStats returns statistics about cache utilization for a tenant
func (c *AnalyticsCache) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	if c == nil || c.cache == nil {
		return make(map[string]interface{}), nil
	}

	stats := make(map[string]interface{})

	// Count keys by pattern
	patterns := map[string]string{
		"compliance_reports":  "audit:compliance:report:*",
		"compliance_exceptions": "audit:compliance:exception:*",
		"anomalies":           "audit:anomaly:*",
		"ssh_keys":            fmt.Sprintf("audit:ssh_key:%s:*", tenantID),
		"blacklist":           "audit:blacklist:*",
	}

	for name, pattern := range patterns {
		count := 0
		iter := c.cache.Client().Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			count++
		}
		stats[name] = count
	}

	return stats, nil
}

// WarmCache preloads frequently accessed data into cache
func (c *AnalyticsCache) WarmCache(ctx context.Context, tenantID uuid.UUID, data map[string]interface{}) error {
	if c == nil || c.cache == nil {
		return nil
	}

	// Preload data based on type
	for key, value := range data {
		var ttl time.Duration

		switch value.(type) {
		case *model.ComplianceReport:
			ttl = ComplianceReportTTL
		case *model.ComplianceException:
			ttl = ExceptionCacheTTL
		case *model.AnomalyDetection:
			ttl = AnomalyCacheTTL
		case *model.SSHKeyAnalytics:
			ttl = SSHKeyCacheTTL
		case []model.CommandBlacklist:
			ttl = BlacklistCacheTTL
		default:
			ttl = DefaultMetricsTTL
		}

		if err := c.cache.Set(ctx, key, value, ttl); err != nil {
			c.logger.Warn().Err(err).Str("key", key).Msg("Failed to warm cache entry")
		}
	}

	return nil
}

// Ping checks if cache is available
func (c *AnalyticsCache) Ping(ctx context.Context) error {
	if c == nil || c.cache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return c.cache.Client().Ping(ctx).Err()
}

// Health checks cache health
func (c *AnalyticsCache) Health(ctx context.Context) error {
	return c.Ping(ctx)
}
