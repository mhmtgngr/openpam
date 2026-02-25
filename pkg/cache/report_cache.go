package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// ReportCache handles caching for report responses
type ReportCache struct {
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewReportCache creates a new report cache
func NewReportCache(cache *cache.Cache, logger zerolog.Logger) *ReportCache {
	return &ReportCache{
		cache:  cache,
		logger: logger,
	}
}

// =============================================================================
// Report Snapshot Caching
// =============================================================================

// GetReportSnapshot retrieves a cached report snapshot
func (c *ReportCache) GetReportSnapshot(ctx context.Context, id uuid.UUID) (*CachedReportSnapshot, error) {
	key := fmt.Sprintf("report:snapshot:%s", id)

	var snapshot CachedReportSnapshot
	if err := c.cache.Get(ctx, key, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// SetReportSnapshot caches a report snapshot
func (c *ReportCache) SetReportSnapshot(ctx context.Context, id uuid.UUID, snapshot *CachedReportSnapshot, ttl time.Duration) error {
	key := fmt.Sprintf("report:snapshot:%s", id)

	if err := c.cache.Set(ctx, key, snapshot, ttl); err != nil {
		return fmt.Errorf("reportCache.SetReportSnapshot: %w", err)
	}

	return nil
}

// InvalidateReportSnapshot removes a report snapshot from cache
func (c *ReportCache) InvalidateReportSnapshot(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("report:snapshot:%s", id)
	return c.cache.Delete(ctx, key)
}

// =============================================================================
// Report List Caching
// =============================================================================

// GetReportList retrieves a cached report list
func (c *ReportCache) GetReportList(ctx context.Context, tenantID uuid.UUID, params ReportListParams) (*CachedReportList, error) {
	key := c.reportListKey(tenantID, params)

	var list CachedReportList
	if err := c.cache.Get(ctx, key, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// SetReportList caches a report list
func (c *ReportCache) SetReportList(ctx context.Context, tenantID uuid.UUID, params ReportListParams, list *CachedReportList, ttl time.Duration) error {
	key := c.reportListKey(tenantID, params)

	if err := c.cache.Set(ctx, key, list, ttl); err != nil {
		return fmt.Errorf("reportCache.SetReportList: %w", err)
	}

	return nil
}

// InvalidateReportList removes report lists for a tenant from cache
func (c *ReportCache) InvalidateReportList(ctx context.Context, tenantID uuid.UUID) error {
	pattern := fmt.Sprintf("report:list:%s:*", tenantID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// reportListKey generates a cache key for report lists
func (c *ReportCache) reportListKey(tenantID uuid.UUID, params ReportListParams) string {
	return fmt.Sprintf("report:list:%s:%s:%s:%s:%s",
		tenantID,
		params.Framework,
		params.Status,
		params.StartDate,
		params.EndDate)
}

// =============================================================================
// Report Stats Caching
// =============================================================================

// GetReportStats retrieves cached report statistics
func (c *ReportCache) GetReportStats(ctx context.Context, tenantID uuid.UUID) (*CachedReportStats, error) {
	key := fmt.Sprintf("report:stats:%s", tenantID)

	var stats CachedReportStats
	if err := c.cache.Get(ctx, key, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SetReportStats caches report statistics
func (c *ReportCache) SetReportStats(ctx context.Context, tenantID uuid.UUID, stats *CachedReportStats, ttl time.Duration) error {
	key := fmt.Sprintf("report:stats:%s", tenantID)

	if err := c.cache.Set(ctx, key, stats, ttl); err != nil {
		return fmt.Errorf("reportCache.SetReportStats: %w", err)
	}

	return nil
}

// InvalidateReportStats removes report statistics from cache
func (c *ReportCache) InvalidateReportStats(ctx context.Context, tenantID uuid.UUID) error {
	key := fmt.Sprintf("report:stats:%s", tenantID)
	return c.cache.Delete(ctx, key)
}

// =============================================================================
// Compliance Status Caching
// =============================================================================

// GetComplianceStatus retrieves cached compliance status
func (c *ReportCache) GetComplianceStatus(ctx context.Context, tenantID uuid.UUID, framework string, startDate, endDate time.Time) (*CachedComplianceStatus, error) {
	key := c.complianceStatusKey(tenantID, framework, startDate, endDate)

	var status CachedComplianceStatus
	if err := c.cache.Get(ctx, key, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SetComplianceStatus caches compliance status
func (c *ReportCache) SetComplianceStatus(ctx context.Context, tenantID uuid.UUID, framework string, startDate, endDate time.Time, status *CachedComplianceStatus, ttl time.Duration) error {
	key := c.complianceStatusKey(tenantID, framework, startDate, endDate)

	if err := c.cache.Set(ctx, key, status, ttl); err != nil {
		return fmt.Errorf("reportCache.SetComplianceStatus: %w", err)
	}

	return nil
}

// complianceStatusKey generates a cache key for compliance status
func (c *ReportCache) complianceStatusKey(tenantID uuid.UUID, framework string, startDate, endDate time.Time) string {
	return fmt.Sprintf("compliance:status:%s:%s:%s:%s",
		tenantID,
		framework,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
}

// =============================================================================
// Exception Caching
// =============================================================================

// GetExceptionList retrieves cached exception list
func (c *ReportCache) GetExceptionList(ctx context.Context, tenantID uuid.UUID, params ExceptionListParams) (*CachedExceptionList, error) {
	key := c.exceptionListKey(tenantID, params)

	var list CachedExceptionList
	if err := c.cache.Get(ctx, key, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// SetExceptionList caches exception list
func (c *ReportCache) SetExceptionList(ctx context.Context, tenantID uuid.UUID, params ExceptionListParams, list *CachedExceptionList, ttl time.Duration) error {
	key := c.exceptionListKey(tenantID, params)

	if err := c.cache.Set(ctx, key, list, ttl); err != nil {
		return fmt.Errorf("reportCache.SetExceptionList: %w", err)
	}

	return nil
}

// InvalidateExceptionList removes exception lists from cache
func (c *ReportCache) InvalidateExceptionList(ctx context.Context, tenantID uuid.UUID) error {
	pattern := fmt.Sprintf("exception:list:%s:*", tenantID)
	return c.cache.DeleteByPattern(ctx, pattern)
}

// exceptionListKey generates a cache key for exception lists
func (c *ReportCache) exceptionListKey(tenantID uuid.UUID, params ExceptionListParams) string {
	return fmt.Sprintf("exception:list:%s:%s:%s:%s",
		tenantID,
		params.Status,
		params.Framework,
		params.ControlID)
}

// =============================================================================
// Cache Invalidation
// =============================================================================

// InvalidateTenant clears all cached data for a tenant
func (c *ReportCache) InvalidateTenant(ctx context.Context, tenantID uuid.UUID) error {
	// Invalidate all report-related caches for tenant
	patterns := []string{
		fmt.Sprintf("report:snapshot:%s:*", tenantID),
		fmt.Sprintf("report:list:%s:*", tenantID),
		fmt.Sprintf("report:stats:%s", tenantID),
		fmt.Sprintf("compliance:status:%s:*", tenantID),
		fmt.Sprintf("exception:list:%s:*", tenantID),
	}

	for _, pattern := range patterns {
		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			c.logger.Warn().Err(err).Str("pattern", pattern).Msg("Failed to delete cache pattern")
		}
	}

	return nil
}

// =============================================================================
// Cache Warming
// =============================================================================

// WarmReportCache warms the cache with commonly accessed reports
func (c *ReportCache) WarmReportCache(ctx context.Context, tenantID uuid.UUID, recentSnapshotIDs []uuid.UUID) error {
	c.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("count", len(recentSnapshotIDs)).
		Msg("Warming report cache")

	// In production, would load recent reports and cache them
	// For now, just log the operation

	return nil
}

// =============================================================================
// Cached Types
// =============================================================================

// CachedReportSnapshot represents a cached report snapshot
type CachedReportSnapshot struct {
	ID            uuid.UUID              `json:"id"`
	TenantID      uuid.UUID              `json:"tenant_id"`
	SnapshotName  string                 `json:"snapshot_name"`
	Framework     string                 `json:"framework"`
	Status        string                 `json:"status"`
	GeneratedAt   time.Time              `json:"generated_at"`
	PeriodStart   time.Time              `json:"period_start"`
	PeriodEnd     time.Time              `json:"period_end"`
	Summary       *string                `json:"summary,omitempty"`
	Data          json.RawMessage        `json:"data,omitempty"`
	FileURL       *string                `json:"file_url,omitempty"`
	FileSizeBytes *int64                 `json:"file_size_bytes,omitempty"`
	CachedAt      time.Time              `json:"cached_at"`
}

// CachedReportList represents a cached list of reports
type CachedReportList struct {
	Snapshots []CachedReportSnapshot `json:"snapshots"`
	Total     int                    `json:"total"`
	CachedAt  time.Time              `json:"cached_at"`
}

// ReportListParams represents parameters for report list queries
type ReportListParams struct {
	Framework string
	Status    string
	StartDate string
	EndDate   string
}

// CachedReportStats represents cached report statistics
type CachedReportStats struct {
	PendingCount      int       `json:"pending_count"`
	CompletedCount    int       `json:"completed_count"`
	FailedCount       int       `json:"failed_count"`
	ExpiredCount      int       `json:"expired_count"`
	TotalCount        int       `json:"total_count"`
	TotalStorageBytes int64     `json:"total_storage_bytes"`
	CachedAt          time.Time `json:"cached_at"`
}

// CachedComplianceStatus represents cached compliance status
type CachedComplianceStatus struct {
	Framework           string    `json:"framework"`
	OverallPercentage   float64   `json:"overall_percentage"`
	PassedControls      int       `json:"passed_controls"`
	TotalControls       int       `json:"total_controls"`
	ViolationsCount     int       `json:"violations_count"`
	CachedAt            time.Time `json:"cached_at"`
}

// CachedExceptionList represents a cached exception list
type CachedExceptionList struct {
	Exceptions []CachedException `json:"exceptions"`
	Total      int               `json:"total"`
	CachedAt   time.Time         `json:"cached_at"`
}

// CachedException represents a cached compliance exception
type CachedException struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	ControlID     string    `json:"control_id"`
	ControlName   string    `json:"control_name"`
	Framework     string    `json:"framework"`
	Status        string    `json:"status"`
	RiskLevel     string    `json:"risk_level"`
	RequestedBy   uuid.UUID `json:"requested_by"`
	RequestedAt   time.Time `json:"requested_at"`
	ApprovedBy    *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	Justification string    `json:"justification"`
}

// ExceptionListParams represents parameters for exception list queries
type ExceptionListParams struct {
	Status    string
	Framework string
	ControlID string
}

// =============================================================================
// Default TTL Values
// =============================================================================

const (
	// DefaultSnapshotCacheTTL is the default TTL for snapshot caching
	DefaultSnapshotCacheTTL = 15 * time.Minute

	// DefaultReportListCacheTTL is the default TTL for report list caching
	DefaultReportListCacheTTL = 5 * time.Minute

	// DefaultReportStatsCacheTTL is the default TTL for report stats caching
	DefaultReportStatsCacheTTL = 2 * time.Minute

	// DefaultComplianceCacheTTL is the default TTL for compliance status caching
	DefaultComplianceCacheTTL = 10 * time.Minute

	// DefaultExceptionListCacheTTL is the default TTL for exception list caching
	DefaultExceptionListCacheTTL = 5 * time.Minute
)
