package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// Cache TTL constants
const (
	// DefaultMetricsTTL is the TTL for metrics cache (5 minutes)
	DefaultMetricsTTL = 5 * time.Minute

	// DailyAggregationTTL is the TTL for daily aggregations (1 hour)
	DailyAggregationTTL = 1 * time.Hour

	// ComplianceReportTTL is the TTL for compliance reports (until next evaluation)
	ComplianceReportTTL = 24 * time.Hour

	// UserActivityTTL is the TTL for user activity (10 minutes)
	UserActivityTTL = 10 * time.Minute

	// AnomalyStatsTTL is the TTL for anomaly stats (1 minute)
	AnomalyStatsTTL = time.Minute

	// BlacklistCacheTTL is the TTL for command blacklist (15 minutes)
	BlacklistCacheTTL = 15 * time.Minute
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRedisCache creates a new Redis-backed cache
func NewRedisCache(c *cache.Cache, logger zerolog.Logger) *RedisCache {
	return &RedisCache{
		cache:  c,
		logger: logger,
	}
}

// Session Metrics Caching

func (c *RedisCache) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time) (*SessionAnalytics, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("analytics:session:%s:%s", tenantID, date.Format("2006-01-02"))

	var analytics SessionAnalytics
	err := c.cache.Get(ctx, key, &analytics)
	if err != nil {
		return nil, err
	}

	return &analytics, nil
}

func (c *RedisCache) SetSessionMetrics(ctx context.Context, analytics *SessionAnalytics, ttl time.Duration) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("analytics:session:%s:%s", analytics.TenantID, analytics.Date.Format("2006-01-02"))

	if ttl == 0 {
		ttl = DefaultMetricsTTL
	}

	return c.cache.Set(ctx, key, analytics, ttl)
}

func (c *RedisCache) InvalidateSessionMetrics(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("analytics:session:%s:*", tenantID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// User Activity Caching

func (c *RedisCache) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time) (*UserActivity, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("analytics:activity:%s:%s:%s", tenantID, userID, date.Format("2006-01-02"))

	var activity UserActivity
	err := c.cache.Get(ctx, key, &activity)
	if err != nil {
		return nil, err
	}

	return &activity, nil
}

func (c *RedisCache) SetUserActivity(ctx context.Context, activity *UserActivity, ttl time.Duration) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("analytics:activity:%s:%s:%s", activity.TenantID, activity.UserID, activity.Date.Format("2006-01-02"))

	if ttl == 0 {
		ttl = UserActivityTTL
	}

	return c.cache.Set(ctx, key, activity, ttl)
}

func (c *RedisCache) InvalidateUserActivity(ctx context.Context, tenantID, userID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("analytics:activity:%s:%s:*", tenantID, userID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// Compliance Report Caching

func (c *RedisCache) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("analytics:compliance:%s", id)

	var report ComplianceReport
	err := c.cache.Get(ctx, key, &report)
	if err != nil {
		return nil, err
	}

	return &report, nil
}

func (c *RedisCache) SetComplianceReport(ctx context.Context, report *ComplianceReport, ttl time.Duration) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("analytics:compliance:%s", report.ID)

	if ttl == 0 {
		ttl = ComplianceReportTTL
	}

	return c.cache.Set(ctx, key, report, ttl)
}

func (c *RedisCache) InvalidateComplianceCache(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("analytics:compliance:*")
	// In production, might want to track tenant-specific compliance reports
	return c.cache.DeleteByPattern(ctx, pattern)
}

// Anomaly Stats Caching

func (c *RedisCache) GetAnomalyStats(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	if c == nil || c.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := fmt.Sprintf("analytics:anomaly:stats:%s", tenantID)

	var stats map[string]int
	err := c.cache.Get(ctx, key, &stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (c *RedisCache) SetAnomalyStats(ctx context.Context, tenantID uuid.UUID, stats map[string]int, ttl time.Duration) error {
	if c == nil || c.cache == nil {
		return nil
	}
	key := fmt.Sprintf("analytics:anomaly:stats:%s", tenantID)

	if ttl == 0 {
		ttl = AnomalyStatsTTL
	}

	return c.cache.Set(ctx, key, stats, ttl)
}

// Invalidate All

func (c *RedisCache) InvalidateAll(ctx context.Context, tenantID uuid.UUID) error {
	if c == nil || c.cache == nil {
		return nil
	}
	pattern := fmt.Sprintf("analytics:*:%s*", tenantID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// Additional cache methods for multi-layer caching

// CachedRepository wraps a Repository with caching
type CachedRepository struct {
	repo   Repository
	cache  *RedisCache
	logger zerolog.Logger
}

// NewCachedRepository creates a new repository with caching
func NewCachedRepository(repo Repository, cache *RedisCache, logger zerolog.Logger) *CachedRepository {
	return &CachedRepository{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// Session Analytics with caching

func (r *CachedRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	err := r.repo.CreateSessionAnalytics(ctx, analytics)
	if err != nil {
		return err
	}

	// Update cache
	_ = r.cache.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)

	return nil
}

func (r *CachedRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	err := r.repo.UpdateSessionAnalytics(ctx, analytics)
	if err != nil {
		return err
	}

	// Update cache
	_ = r.cache.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)

	return nil
}

func (r *CachedRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	// Try cache first
	cached, err := r.cache.GetSessionMetrics(ctx, tenantID, date)
	if err == nil {
		r.logger.Debug().Str("tenant_id", tenantID.String()).Msg("Cache hit for session analytics")
		return cached, nil
	}

	// Cache miss, fetch from DB
	analytics, err := r.repo.GetSessionAnalytics(ctx, tenantID, date, hour)
	if err != nil {
		return nil, err
	}

	// Populate cache
	_ = r.cache.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)

	return analytics, nil
}

func (r *CachedRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	return r.repo.ListSessionAnalytics(ctx, filter, limit, offset)
}

// User Activity with caching

func (r *CachedRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error {
	err := r.repo.CreateUserActivity(ctx, activity)
	if err != nil {
		return err
	}

	// Update cache
	_ = r.cache.SetUserActivity(ctx, activity, UserActivityTTL)

	return nil
}

func (r *CachedRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error {
	err := r.repo.UpdateUserActivity(ctx, activity)
	if err != nil {
		return err
	}

	// Update cache
	_ = r.cache.SetUserActivity(ctx, activity, UserActivityTTL)

	return nil
}

func (r *CachedRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) {
	// Try cache first
	cached, err := r.cache.GetUserActivity(ctx, tenantID, userID, date)
	if err == nil {
		return cached, nil
	}

	// Cache miss, fetch from DB
	activity, err := r.repo.GetUserActivity(ctx, tenantID, userID, date, hour)
	if err != nil {
		return nil, err
	}

	// Populate cache
	_ = r.cache.SetUserActivity(ctx, activity, UserActivityTTL)

	return activity, nil
}

func (r *CachedRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) {
	return r.repo.ListUserActivity(ctx, filter, limit, offset)
}

// Command Frequency (no caching for inserts)

func (r *CachedRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error {
	return r.repo.RecordCommand(ctx, cmd)
}

func (r *CachedRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) {
	return r.repo.ListCommandFrequency(ctx, filter, limit, offset)
}

func (r *CachedRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	return r.repo.GetTopCommands(ctx, tenantID, dateFrom, dateTo, limit)
}

// Compliance with caching

func (r *CachedRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	err := r.repo.CreateComplianceReport(ctx, report)
	if err != nil {
		return err
	}

	// Update cache
	_ = r.cache.SetComplianceReport(ctx, report, ComplianceReportTTL)

	return nil
}

func (r *CachedRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	// Try cache first
	cached, err := r.cache.GetComplianceReport(ctx, id)
	if err == nil {
		return cached, nil
	}

	// Cache miss, fetch from DB
	report, err := r.repo.GetComplianceReport(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache
	_ = r.cache.SetComplianceReport(ctx, report, ComplianceReportTTL)

	return report, nil
}

func (r *CachedRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) {
	return r.repo.ListComplianceReports(ctx, filter, limit, offset)
}

func (r *CachedRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error {
	return r.repo.CreateControlEvaluation(ctx, evaluation)
}

func (r *CachedRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) {
	return r.repo.ListControlEvaluations(ctx, reportID)
}

func (r *CachedRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	return r.repo.CreateComplianceException(ctx, exception)
}

func (r *CachedRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	return r.repo.ListComplianceExceptions(ctx, tenantID)
}

// Anomaly Detection

func (r *CachedRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	err := r.repo.CreateAnomalyDetection(ctx, anomaly)
	if err != nil {
		return err
	}

	// Invalidate stats cache
	tenantID := anomaly.TenantID
	_ = r.cache.InvalidateAll(ctx, tenantID)

	return nil
}

func (r *CachedRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) {
	return r.repo.GetAnomalyDetection(ctx, id)
}

func (r *CachedRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) {
	return r.repo.ListAnomalyDetections(ctx, filter, limit, offset)
}

func (r *CachedRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	err := r.repo.UpdateAnomalyDetection(ctx, anomaly)
	if err != nil {
		return err
	}

	// Invalidate stats cache
	_ = r.cache.InvalidateAll(ctx, anomaly.TenantID)

	return nil
}

// Ransomware Events

func (r *CachedRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	err := r.repo.CreateRansomwareEvent(ctx, event)
	if err != nil {
		return err
	}

	// Invalidate all caches for tenant
	_ = r.cache.InvalidateAll(ctx, event.TenantID)

	return nil
}

func (r *CachedRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) {
	return r.repo.GetRansomwareEvent(ctx, id)
}

func (r *CachedRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) {
	return r.repo.ListRansomwareEvents(ctx, tenantID, limit, offset)
}

// Command Blacklist (cached for fast lookups)

func (r *CachedRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	err := r.repo.CreateCommandBlacklist(ctx, blacklist)
	if err != nil {
		return err
	}

	// Invalidate blacklist cache
	if blacklist.TenantID != nil {
		_ = r.invalidateBlacklistCache(ctx, *blacklist.TenantID)
	} else {
		// Global blacklist changed - invalidate all
		_ = r.invalidateAllBlacklistCache(ctx)
	}

	return nil
}

func (r *CachedRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) {
	return r.repo.GetCommandBlacklist(ctx, id)
}

func (r *CachedRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	// Try cache first
	if tenantID != nil && r.cache != nil && r.cache.cache != nil {
		key := fmt.Sprintf("analytics:blacklist:%s", *tenantID)
		var cached []CommandBlacklist
		if err := r.cache.cache.Get(ctx, key, &cached); err == nil {
			return cached, nil
		}

		// Cache miss - fetch and cache
		blacklists, err := r.repo.ListCommandBlacklist(ctx, tenantID)
		if err != nil {
			return nil, err
		}

		_ = r.cache.cache.Set(ctx, key, blacklists, BlacklistCacheTTL)
		return blacklists, nil
	}

	return r.repo.ListCommandBlacklist(ctx, tenantID)
}

func (r *CachedRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	err := r.repo.UpdateCommandBlacklist(ctx, blacklist)
	if err != nil {
		return err
	}

	// Invalidate blacklist cache
	if blacklist.TenantID != nil {
		_ = r.invalidateBlacklistCache(ctx, *blacklist.TenantID)
	} else {
		_ = r.invalidateAllBlacklistCache(ctx)
	}

	return nil
}

func (r *CachedRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	blacklist, err := r.repo.GetCommandBlacklist(ctx, id)
	if err != nil {
		return err
	}

	err = r.repo.DeleteCommandBlacklist(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate blacklist cache
	if blacklist.TenantID != nil {
		_ = r.invalidateBlacklistCache(ctx, *blacklist.TenantID)
	} else {
		_ = r.invalidateAllBlacklistCache(ctx)
	}

	return nil
}

func (r *CachedRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) {
	return r.repo.FindMatchingBlacklist(ctx, tenantID, command, userIDs, groupIDs)
}

func (r *CachedRepository) invalidateBlacklistCache(ctx context.Context, tenantID uuid.UUID) error {
	if r.cache == nil || r.cache.cache == nil {
		return nil
	}
	key := fmt.Sprintf("analytics:blacklist:%s", tenantID)
	return r.cache.cache.Delete(ctx, key)
}

func (r *CachedRepository) invalidateAllBlacklistCache(ctx context.Context) error {
	if r.cache == nil || r.cache.cache == nil {
		return nil
	}
	pattern := "analytics:blacklist:*"
	return r.cache.cache.DeleteByPattern(ctx, pattern)
}

// SSH Key Analytics

func (r *CachedRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	return r.repo.CreateSSHKeyAnalytics(ctx, analytics)
}

func (r *CachedRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	return r.repo.UpdateSSHKeyAnalytics(ctx, analytics)
}

func (r *CachedRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) {
	return r.repo.GetSSHKeyAnalytics(ctx, tenantID, sshKeyID, date)
}

func (r *CachedRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	return r.repo.ListSSHKeyAnalytics(ctx, tenantID, sshKeyID, dateFrom, dateTo)
}

// Dashboard

func (r *CachedRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) {
	// Try cache first for dashboard metrics (short TTL)
	if r.cache != nil && r.cache.cache != nil {
		key := fmt.Sprintf("analytics:dashboard:%s", tenantID)
		var cached DashboardMetrics
		if err := r.cache.cache.Get(ctx, key, &cached); err == nil {
			return &cached, nil
		}
	}

	// Cache miss, fetch from DB
	metrics, err := r.repo.GetDashboardMetrics(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Populate cache with short TTL (30 seconds)
	if r.cache != nil && r.cache.cache != nil {
		key := fmt.Sprintf("analytics:dashboard:%s", tenantID)
		_ = r.cache.cache.Set(ctx, key, metrics, 30*time.Second)
	}

	return metrics, nil
}

// GetSessionMetrics retrieves session metrics (delegates to wrapped repo)
func (r *CachedRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return r.repo.GetSessionMetrics(ctx, tenantID, date, hour)
}

// UpdateRansomwareEvent updates a ransomware event
func (r *CachedRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	err := r.repo.UpdateRansomwareEvent(ctx, event)
	if err != nil {
		return err
	}

	// Invalidate all caches for tenant
	_ = r.cache.InvalidateAll(ctx, event.TenantID)

	return nil
}

// Cache warming methods

// WarmSessionMetricsCache preloads session metrics cache
func (c *RedisCache) WarmSessionMetricsCache(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, repo Repository) error {
	if c == nil || c.cache == nil {
		return nil
	}
	current := dateFrom
	for current.Before(dateTo) || current.Equal(dateTo) {
		for hour := 0; hour < 24; hour++ {
			analytics, err := repo.GetSessionMetrics(ctx, tenantID, current, hour)
			if err == nil && analytics != nil {
				_ = c.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)
			}
		}
		current = current.AddDate(0, 0, 1)
	}

	return nil
}

// WarmComplianceReportCache preloads compliance report cache
func (c *RedisCache) WarmComplianceReportCache(ctx context.Context, reportIDs []uuid.UUID, repo Repository) error {
	if c == nil || c.cache == nil {
		return nil
	}
	for _, id := range reportIDs {
		report, err := repo.GetComplianceReport(ctx, id)
		if err == nil && report != nil {
			_ = c.SetComplianceReport(ctx, report, ComplianceReportTTL)
		}
	}

	return nil
}

// BatchInvalidation invalidates multiple cache entries at once
func (c *RedisCache) BatchInvalidation(ctx context.Context, tenantID uuid.UUID, cacheTypes ...string) error {
	if c == nil || c.cache == nil {
		return nil
	}
	for _, cacheType := range cacheTypes {
		var pattern string
		switch cacheType {
		case "session":
			pattern = fmt.Sprintf("analytics:session:%s:*", tenantID)
		case "activity":
			pattern = fmt.Sprintf("analytics:activity:%s:*", tenantID)
		case "compliance":
			pattern = fmt.Sprintf("analytics:compliance:*")
		case "anomaly":
			pattern = fmt.Sprintf("analytics:anomaly:%s:*", tenantID)
		case "blacklist":
			pattern = fmt.Sprintf("analytics:blacklist:%s", tenantID)
		case "dashboard":
			pattern = fmt.Sprintf("analytics:dashboard:%s", tenantID)
		default:
			pattern = fmt.Sprintf("analytics:*:%s*", tenantID)
		}

		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			c.logger.Warn().Err(err).Str("pattern", pattern).Msg("Failed to invalidate cache pattern")
		}
	}

	return nil
}

// GetCacheStats returns statistics about cache utilization
func (c *RedisCache) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	if c == nil || c.cache == nil {
		return make(map[string]interface{}), nil
	}
	stats := make(map[string]interface{})

	// Count keys by pattern
	patterns := map[string]string{
		"session":    fmt.Sprintf("analytics:session:%s:*", tenantID),
		"activity":   fmt.Sprintf("analytics:activity:%s:*", tenantID),
		"compliance": "analytics:compliance:*",
		"anomaly":    fmt.Sprintf("analytics:anomaly:*"),
		"dashboard":  fmt.Sprintf("analytics:dashboard:%s", tenantID),
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

// Export cache data for backup/migration
func (c *RedisCache) ExportCacheData(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	if c == nil || c.cache == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	pattern := fmt.Sprintf("analytics:*:%s*", tenantID)

	data := make(map[string]interface{})
	iter := c.cache.Client().Scan(ctx, 0, pattern, 1000).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		var value interface{}
		if err := c.cache.Get(ctx, key, &value); err == nil {
			data[key] = value
		}
	}

	return json.Marshal(data)
}

// Import cache data from backup
func (c *RedisCache) ImportCacheData(ctx context.Context, data []byte) error {
	if c == nil || c.cache == nil {
		return nil
	}
	var cacheData map[string]interface{}
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return err
	}

	for key, value := range cacheData {
		// Determine appropriate TTL based on key type
		ttl := DefaultMetricsTTL
		if contains(key, "compliance") {
			ttl = ComplianceReportTTL
		} else if contains(key, "activity") {
			ttl = UserActivityTTL
		}

		if err := c.cache.Set(ctx, key, value, ttl); err != nil {
			c.logger.Warn().Err(err).Str("key", key).Msg("Failed to restore cache entry")
		}
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
