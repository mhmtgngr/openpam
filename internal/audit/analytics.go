// Package audit provides analytics service for compliance, anomalies, and security monitoring
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	analyticscache "github.com/openpam/openpam/internal/audit/cache"
	corecache "github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/openpam/openpam/internal/audit/repository"
	"github.com/rs/zerolog"
)

// AnalyticsService handles analytics business logic
type AnalyticsService struct {
	db                *sqlx.DB
	cache             *analyticscache.AnalyticsCache
	logger            zerolog.Logger

	// Repositories
	complianceRepo    *repository.ComplianceReportRepository
	exceptionRepo     *repository.ComplianceExceptionRepository
	anomalyRepo       *repository.AnomalyRepository
	sshKeyRepo        *repository.SSHKeyAnalyticsRepository
	blacklistRepo     *repository.CommandBlacklistRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sqlx.DB, coreCache *corecache.Cache, logger zerolog.Logger) *AnalyticsService {
	analyticsCache := analyticscache.NewAnalyticsCache(coreCache, logger)

	return &AnalyticsService{
		db:     db,
		cache:  analyticsCache,
		logger: logger,

		// Initialize repositories
		complianceRepo: repository.NewComplianceReportRepository(db, logger),
		exceptionRepo:  repository.NewComplianceExceptionRepository(db, logger),
		anomalyRepo:    repository.NewAnomalyRepository(db, logger),
		sshKeyRepo:     repository.NewSSHKeyAnalyticsRepository(db, logger),
		blacklistRepo:  repository.NewCommandBlacklistRepository(db, logger),
	}
}

// =============================================================================
// Compliance Report Methods
// =============================================================================

// CreateComplianceReport creates a new compliance report
func (s *AnalyticsService) CreateComplianceReport(ctx context.Context, report *model.ComplianceReport) error {
	if err := s.complianceRepo.Create(ctx, report); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceReports(ctx, report.TenantID)

	return nil
}

// GetComplianceReport retrieves a compliance report by ID
func (s *AnalyticsService) GetComplianceReport(ctx context.Context, id uuid.UUID) (*model.ComplianceReport, error) {
	// Try cache first
	cached, err := s.cache.GetComplianceReport(ctx, id)
	if err == nil {
		return cached, nil
	}

	// Fetch from database
	report, err := s.complianceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache
	_ = s.cache.SetComplianceReport(ctx, report)

	return report, nil
}

// ListComplianceReports retrieves compliance reports with filtering
func (s *AnalyticsService) ListComplianceReports(ctx context.Context, tenantID uuid.UUID, filter model.ComplianceReportFilter, limit, offset int) ([]model.ComplianceReport, int, error) {
	return s.complianceRepo.List(ctx, tenantID, filter, limit, offset)
}

// GetLatestComplianceReport retrieves the latest report for a framework
func (s *AnalyticsService) GetLatestComplianceReport(ctx context.Context, tenantID uuid.UUID, framework string) (*model.ComplianceReport, error) {
	return s.complianceRepo.GetLatestByFramework(ctx, tenantID, framework)
}

// UpdateComplianceReportStatus updates the status of a compliance report
func (s *AnalyticsService) UpdateComplianceReportStatus(ctx context.Context, id uuid.UUID, status string, overallScore *float64, passedControls, failedControls, skippedControls int) error {
	err := s.complianceRepo.UpdateStatus(ctx, id, status, overallScore, passedControls, failedControls, skippedControls)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceReport(ctx, id)

	return nil
}

// DeleteComplianceReport deletes a compliance report
func (s *AnalyticsService) DeleteComplianceReport(ctx context.Context, id uuid.UUID) error {
	// Get report to determine tenant_id for cache invalidation
	report, err := s.complianceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.complianceRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceReport(ctx, id)
	_ = s.cache.InvalidateComplianceReports(ctx, report.TenantID)

	return nil
}

// GetFrameworkSummary returns a summary of all reports by framework
func (s *AnalyticsService) GetFrameworkSummary(ctx context.Context, tenantID uuid.UUID) (map[string]repository.FrameworkSummary, error) {
	return s.complianceRepo.GetFrameworkSummary(ctx, tenantID)
}

// =============================================================================
// Compliance Exception Methods
// =============================================================================

// CreateComplianceException creates a new compliance exception
func (s *AnalyticsService) CreateComplianceException(ctx context.Context, exception *model.ComplianceException) error {
	if err := s.exceptionRepo.Create(ctx, exception); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceExceptions(ctx, exception.TenantID)

	return nil
}

// GetComplianceException retrieves a compliance exception by ID
func (s *AnalyticsService) GetComplianceException(ctx context.Context, id uuid.UUID) (*model.ComplianceException, error) {
	// Try cache first
	cached, err := s.cache.GetComplianceException(ctx, id)
	if err == nil {
		return cached, nil
	}

	// Fetch from database
	exception, err := s.exceptionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache
	_ = s.cache.SetComplianceException(ctx, exception)

	return exception, nil
}

// ListComplianceExceptions retrieves compliance exceptions with filtering
func (s *AnalyticsService) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID, filter model.ComplianceExceptionFilter, limit, offset int) ([]model.ComplianceException, int, error) {
	return s.exceptionRepo.List(ctx, tenantID, filter, limit, offset)
}

// UpdateComplianceExceptionStatus updates the status of a compliance exception
func (s *AnalyticsService) UpdateComplianceExceptionStatus(ctx context.Context, id uuid.UUID, status string, approvedBy, riskAcceptedBy *uuid.UUID) error {
	// Get exception to determine tenant_id for cache invalidation
	exception, err := s.exceptionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.exceptionRepo.UpdateStatus(ctx, id, status, approvedBy, riskAcceptedBy)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceException(ctx, id)
	_ = s.cache.InvalidateComplianceExceptions(ctx, exception.TenantID)

	return nil
}

// UpdateComplianceException updates a compliance exception
func (s *AnalyticsService) UpdateComplianceException(ctx context.Context, exception *model.ComplianceException) error {
	err := s.exceptionRepo.Update(ctx, exception)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceException(ctx, exception.ID)

	return nil
}

// DeleteComplianceException deletes a compliance exception
func (s *AnalyticsService) DeleteComplianceException(ctx context.Context, id uuid.UUID) error {
	// Get exception to determine tenant_id for cache invalidation
	exception, err := s.exceptionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.exceptionRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateComplianceException(ctx, id)
	_ = s.cache.InvalidateComplianceExceptions(ctx, exception.TenantID)

	return nil
}

// GetExpiringExceptions retrieves exceptions that will expire soon
func (s *AnalyticsService) GetExpiringExceptions(ctx context.Context, tenantID uuid.UUID, within time.Duration) ([]model.ComplianceException, error) {
	return s.exceptionRepo.GetExpiringSoon(ctx, tenantID, within)
}

// GetExpiredExceptions retrieves expired exceptions
func (s *AnalyticsService) GetExpiredExceptions(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceException, error) {
	return s.exceptionRepo.GetExpired(ctx, tenantID)
}

// MarkExpiredExceptions marks expired exceptions as expired
func (s *AnalyticsService) MarkExpiredExceptions(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return s.exceptionRepo.MarkExpired(ctx, tenantID)
}

// =============================================================================
// Anomaly Detection Methods
// =============================================================================

// CreateAnomaly creates a new anomaly detection
func (s *AnalyticsService) CreateAnomaly(ctx context.Context, anomaly *model.AnomalyDetection) error {
	if err := s.anomalyRepo.Create(ctx, anomaly); err != nil {
		return err
	}

	// Invalidate stats cache
	_ = s.cache.InvalidateAnomalies(ctx, anomaly.TenantID)

	return nil
}

// GetAnomaly retrieves an anomaly by ID
func (s *AnalyticsService) GetAnomaly(ctx context.Context, id uuid.UUID) (*model.AnomalyDetection, error) {
	// Try cache first
	cached, err := s.cache.GetAnomaly(ctx, id)
	if err == nil {
		return cached, nil
	}

	// Fetch from database
	anomaly, err := s.anomalyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Populate cache
	_ = s.cache.SetAnomaly(ctx, anomaly)

	return anomaly, nil
}

// ListAnomalies retrieves anomalies with filtering
func (s *AnalyticsService) ListAnomalies(ctx context.Context, tenantID uuid.UUID, filter model.AnomalyFilter, limit, offset int) ([]model.AnomalyDetection, int, error) {
	return s.anomalyRepo.List(ctx, tenantID, filter, limit, offset)
}

// UpdateAnomaly updates an anomaly
func (s *AnalyticsService) UpdateAnomaly(ctx context.Context, anomaly *model.AnomalyDetection) error {
	err := s.anomalyRepo.Update(ctx, anomaly)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateAnomaly(ctx, anomaly.ID)

	return nil
}

// UpdateAnomalyStatus updates the status of an anomaly
func (s *AnalyticsService) UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, status string, assignedTo *uuid.UUID, resolutionNotes *string, resolvedBy *uuid.UUID) error {
	// Get anomaly to determine tenant_id for cache invalidation
	anomaly, err := s.anomalyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.anomalyRepo.UpdateStatus(ctx, id, status, assignedTo, resolutionNotes, resolvedBy)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateAnomaly(ctx, id)
	_ = s.cache.InvalidateAnomalies(ctx, anomaly.TenantID)

	return nil
}

// DeleteAnomaly deletes an anomaly
func (s *AnalyticsService) DeleteAnomaly(ctx context.Context, id uuid.UUID) error {
	// Get anomaly to determine tenant_id for cache invalidation
	anomaly, err := s.anomalyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.anomalyRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateAnomaly(ctx, id)
	_ = s.cache.InvalidateAnomalies(ctx, anomaly.TenantID)

	return nil
}

// GetAnomalyStats retrieves statistics about anomalies
func (s *AnalyticsService) GetAnomalyStats(ctx context.Context, tenantID uuid.UUID) (*repository.AnomalyStats, error) {
	// Try cache first
	cached, err := s.cache.GetAnomalyStats(ctx, tenantID)
	if err == nil && cached != nil {
		return &repository.AnomalyStats{
			Total:              cached["total"],
			OpenCount:          cached["open_count"],
			InvestigatingCount: cached["investigating_count"],
			ResolvedCount:      cached["resolved_count"],
			CriticalCount:      cached["critical_count"],
			HighCount:          cached["high_count"],
			TodayCount:         cached["today_count"],
			WeekCount:          cached["week_count"],
		}, nil
	}

	// Fetch from database
	stats, err := s.anomalyRepo.GetStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Populate cache
	statsMap := map[string]int{
		"total":              stats.Total,
		"open_count":         stats.OpenCount,
		"investigating_count": stats.InvestigatingCount,
		"resolved_count":     stats.ResolvedCount,
		"critical_count":     stats.CriticalCount,
		"high_count":         stats.HighCount,
		"today_count":        stats.TodayCount,
		"week_count":         stats.WeekCount,
	}
	_ = s.cache.SetAnomalyStats(ctx, tenantID, statsMap)

	return stats, nil
}

// GetOpenAnomalies retrieves all open anomalies
func (s *AnalyticsService) GetOpenAnomalies(ctx context.Context, tenantID uuid.UUID, severity *string) ([]model.AnomalyDetection, error) {
	return s.anomalyRepo.GetOpen(ctx, tenantID, severity)
}

// BatchCreateAnomalies creates multiple anomalies
func (s *AnalyticsService) BatchCreateAnomalies(ctx context.Context, anomalies []model.AnomalyDetection) error {
	if err := s.anomalyRepo.BatchCreate(ctx, anomalies); err != nil {
		return err
	}

	// Invalidate stats cache for the tenant
	if len(anomalies) > 0 {
		_ = s.cache.InvalidateAnomalies(ctx, anomalies[0].TenantID)
	}

	return nil
}

// DetectBehavioralAnomaly creates a behavioral anomaly detection
func (s *AnalyticsService) DetectBehavioralAnomaly(ctx context.Context, tenantID, userID uuid.UUID, title, description string, riskScore, confidenceScore float64, indicators map[string]interface{}) (*model.AnomalyDetection, error) {
	indicatorData, _ := json.Marshal(indicators)

	anomaly := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeBehavioral),
		UserID:          &userID,
		Severity:        s.getSeverityFromRisk(riskScore),
		ConfidenceScore: confidenceScore,
		RiskScore:       riskScore,
		Title:           title,
		Description:     &description,
		Indicators:      indicatorData,
		DetectionMethod: "behavioral_analysis",
		Status:          string(model.AnomalyStatusOpen),
	}

	return s.createAndReturnAnomaly(ctx, anomaly)
}

// DetectTemporalAnomaly creates a temporal anomaly detection
func (s *AnalyticsService) DetectTemporalAnomaly(ctx context.Context, tenantID, userID uuid.UUID, title, description string, riskScore, confidenceScore float64, indicators map[string]interface{}) (*model.AnomalyDetection, error) {
	indicatorData, _ := json.Marshal(indicators)

	anomaly := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeTemporal),
		UserID:          &userID,
		Severity:        s.getSeverityFromRisk(riskScore),
		ConfidenceScore: confidenceScore,
		RiskScore:       riskScore,
		Title:           title,
		Description:     &description,
		Indicators:      indicatorData,
		DetectionMethod: "temporal_analysis",
		Status:          string(model.AnomalyStatusOpen),
	}

	return s.createAndReturnAnomaly(ctx, anomaly)
}

// DetectSpatialAnomaly creates a spatial anomaly detection
func (s *AnalyticsService) DetectSpatialAnomaly(ctx context.Context, tenantID, userID uuid.UUID, targetHost string, title, description string, riskScore, confidenceScore float64, indicators map[string]interface{}) (*model.AnomalyDetection, error) {
	indicatorData, _ := json.Marshal(indicators)

	anomaly := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeSpatial),
		UserID:          &userID,
		TargetHost:      &targetHost,
		Severity:        s.getSeverityFromRisk(riskScore),
		ConfidenceScore: confidenceScore,
		RiskScore:       riskScore,
		Title:           title,
		Description:     &description,
		Indicators:      indicatorData,
		DetectionMethod: "spatial_analysis",
		Status:          string(model.AnomalyStatusOpen),
	}

	return s.createAndReturnAnomaly(ctx, anomaly)
}

// createAndReturnAnomaly is a helper to create and return the anomaly
func (s *AnalyticsService) createAndReturnAnomaly(ctx context.Context, anomaly *model.AnomalyDetection) (*model.AnomalyDetection, error) {
	if err := s.anomalyRepo.Create(ctx, anomaly); err != nil {
		return nil, err
	}

	_ = s.cache.InvalidateAnomalies(ctx, anomaly.TenantID)
	return anomaly, nil
}

// getSeverityFromRisk converts a risk score to a severity level
func (s *AnalyticsService) getSeverityFromRisk(risk float64) string {
	switch {
	case risk >= 90:
		return string(model.SeverityCritical)
	case risk >= 70:
		return string(model.SeverityHigh)
	case risk >= 40:
		return string(model.SeverityMedium)
	default:
		return string(model.SeverityLow)
	}
}

// =============================================================================
// Anomaly Correlation and Deduplication Methods
// =============================================================================

// GetAnomaliesByCorrelationID retrieves all anomalies in a correlation group
func (s *AnalyticsService) GetAnomaliesByCorrelationID(ctx context.Context, correlationID uuid.UUID) ([]model.AnomalyDetection, error) {
	return s.anomalyRepo.GetByCorrelationID(ctx, correlationID)
}

// GetAnomalyTypes retrieves unique anomaly types for a tenant
func (s *AnalyticsService) GetAnomalyTypes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return s.anomalyRepo.GetAnomalyTypes(ctx, tenantID)
}

// GetTopUsersByAnomalyCount retrieves users with the most anomalies
func (s *AnalyticsService) GetTopUsersByAnomalyCount(ctx context.Context, tenantID uuid.UUID, limit int, dateFrom, dateTo *time.Time) ([]repository.UserAnomalyCount, error) {
	return s.anomalyRepo.GetTopUsersByAnomalyCount(ctx, tenantID, limit, dateFrom, dateTo)
}

// MergeDuplicateAnomalies marks all anomalies with the same correlation key as duplicates
func (s *AnalyticsService) MergeDuplicateAnomalies(ctx context.Context, anomalyID uuid.UUID) (int, error) {
	count, err := s.anomalyRepo.MergeDuplicateAnomalies(ctx, anomalyID)
	if err != nil {
		return 0, err
	}

	// Get anomaly to invalidate cache
	if anomaly, err := s.anomalyRepo.GetByID(ctx, anomalyID); err == nil {
		_ = s.cache.InvalidateAnomalies(ctx, anomaly.TenantID)
	}

	return count, nil
}

// =============================================================================
// SSH Key Analytics Methods
// =============================================================================

// RecordSSHKeyUsage records SSH key usage for analytics
func (s *AnalyticsService) RecordSSHKeyUsage(ctx context.Context, tenantID, sshKeyID, userID uuid.UUID, targetHost string, duration time.Duration, failed bool) error {
	date := time.Now().Truncate(24 * time.Hour)

	// Get existing analytics
	analytics, err := s.sshKeyRepo.Get(ctx, tenantID, sshKeyID, date)
	if err != nil || analytics == nil {
		// Create new
		analytics = &model.SSHKeyAnalytics{
			TenantID:       tenantID,
			SSHKeyID:       sshKeyID,
			Date:           date,
			UsageCount:     1,
			UniqueUsers:    1,
			UniqueTargets:  1,
			FirstUseTime:   timePtr(time.Now()),
			LastUseTime:    timePtr(time.Now()),
			FailedAttempts: boolToInt(failed),
		}
		return s.sshKeyRepo.Create(ctx, analytics)
	}

	// Update existing
	analytics.UsageCount++
	if failed {
		analytics.FailedAttempts++
	}
	analytics.LastUseTime = timePtr(time.Now())

	// Update session duration average
	if duration > 0 {
		if analytics.AvgSessionDurationSeconds == nil {
			avg := float64(duration.Seconds())
			analytics.AvgSessionDurationSeconds = &avg
		} else {
			newAvg := (*analytics.AvgSessionDurationSeconds + float64(duration.Seconds())) / 2
			analytics.AvgSessionDurationSeconds = &newAvg
		}
	}

	// Check for off-hours usage (outside 6 AM - 6 PM)
	hour := time.Now().Hour()
	if hour < 6 || hour >= 18 {
		analytics.OffHoursUsage++
	}

	return s.sshKeyRepo.Update(ctx, analytics)
}

// GetSSHKeyAnalytics retrieves SSH key analytics for a key
func (s *AnalyticsService) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*model.SSHKeyAnalytics, error) {
	return s.sshKeyRepo.Get(ctx, tenantID, sshKeyID, date)
}

// ListSSHKeyAnalytics retrieves SSH key analytics with filtering
func (s *AnalyticsService) ListSSHKeyAnalytics(ctx context.Context, filter model.SSHKeyAnalyticsFilter) ([]model.SSHKeyAnalytics, error) {
	return s.sshKeyRepo.List(ctx, filter)
}

// GetSSHKeyUsageSummary returns a summary of SSH key usage
func (s *AnalyticsService) GetSSHKeyUsageSummary(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*repository.SSHKeyUsageSummary, error) {
	return s.sshKeyRepo.GetUsageSummary(ctx, tenantID, dateFrom, dateTo)
}

// GetMostUsedSSHKeys retrieves the most used SSH keys
func (s *AnalyticsService) GetMostUsedSSHKeys(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]repository.KeyUsageRank, error) {
	return s.sshKeyRepo.GetMostUsedKeys(ctx, tenantID, dateFrom, dateTo, limit)
}

// GetAnomalousSSHKeys retrieves SSH keys with unusual usage patterns
func (s *AnalyticsService) GetAnomalousSSHKeys(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, threshold int) ([]model.SSHKeyAnalytics, error) {
	return s.sshKeyRepo.GetAnomalousKeys(ctx, tenantID, dateFrom, dateTo, threshold)
}

// =============================================================================
// Command Blacklist Methods
// =============================================================================

// CreateCommandBlacklist creates a new command blacklist entry
func (s *AnalyticsService) CreateCommandBlacklist(ctx context.Context, blacklist *model.CommandBlacklist) error {
	if err := s.blacklistRepo.Create(ctx, blacklist); err != nil {
		return err
	}

	// Invalidate cache
	tenantID := blacklist.TenantID
	_ = s.cache.InvalidateCommandBlacklist(ctx, tenantID)

	return nil
}

// GetCommandBlacklist retrieves a command blacklist entry by ID
func (s *AnalyticsService) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*model.CommandBlacklist, error) {
	return s.blacklistRepo.GetByID(ctx, id)
}

// ListCommandBlacklist retrieves command blacklist entries with filtering
func (s *AnalyticsService) ListCommandBlacklist(ctx context.Context, filter model.CommandBlacklistFilter) ([]model.CommandBlacklist, error) {
	return s.blacklistRepo.List(ctx, filter)
}

// UpdateCommandBlacklist updates a command blacklist entry
func (s *AnalyticsService) UpdateCommandBlacklist(ctx context.Context, blacklist *model.CommandBlacklist) error {
	err := s.blacklistRepo.Update(ctx, blacklist)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateCommandBlacklist(ctx, blacklist.TenantID)

	return nil
}

// DeleteCommandBlacklist deletes (disables) a command blacklist entry
func (s *AnalyticsService) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	blacklist, err := s.blacklistRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.blacklistRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateCommandBlacklist(ctx, blacklist.TenantID)

	return nil
}

// EvaluateCommandAgainstBlacklist evaluates a command against the blacklist
func (s *AnalyticsService) EvaluateCommandAgainstBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) (allowed bool, action string, blacklistEntry *model.CommandBlacklist) {
	return s.blacklistRepo.CheckCommandAgainstBlacklist(ctx, tenantID, command, userIDs, groupIDs)
}

// GetBlacklistStats returns statistics about command blacklist
func (s *AnalyticsService) GetBlacklistStats(ctx context.Context, tenantID uuid.UUID) (*repository.BlacklistStats, error) {
	return s.blacklistRepo.GetStats(ctx, tenantID)
}

// EnableCommandBlacklist enables a command blacklist entry
func (s *AnalyticsService) EnableCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	blacklist, err := s.blacklistRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.blacklistRepo.Enable(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateCommandBlacklist(ctx, blacklist.TenantID)

	return nil
}

// DisableCommandBlacklist disables a command blacklist entry
func (s *AnalyticsService) DisableCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	blacklist, err := s.blacklistRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.blacklistRepo.Disable(ctx, id)
	if err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.InvalidateCommandBlacklist(ctx, blacklist.TenantID)

	return nil
}

// =============================================================================
// Cache Management
// =============================================================================

// InvalidateCache invalidates analytics cache for a tenant
func (s *AnalyticsService) InvalidateCache(ctx context.Context, tenantID uuid.UUID, cacheTypes ...string) error {
	if len(cacheTypes) == 0 {
		return s.cache.InvalidateAll(ctx, tenantID)
	}
	return s.cache.BatchInvalidation(ctx, tenantID, cacheTypes...)
}

// GetCacheStats returns statistics about cache utilization
func (s *AnalyticsService) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	return s.cache.GetCacheStats(ctx, tenantID)
}

// =============================================================================
// Dashboard Methods
// =============================================================================

// GetDashboardSummary returns a summary of analytics for the dashboard
func (s *AnalyticsService) GetDashboardSummary(ctx context.Context, tenantID uuid.UUID) (*DashboardSummary, error) {
	summary := &DashboardSummary{
		TenantID:  tenantID,
		GeneratedAt: time.Now(),
	}

	// Get compliance summary
	frameworkSummaries, err := s.complianceRepo.GetFrameworkSummary(ctx, tenantID)
	if err == nil {
		summary.ComplianceSummary = frameworkSummaries
	}

	// Get anomaly stats
	anomalyStats, err := s.anomalyRepo.GetStats(ctx, tenantID)
	if err == nil {
		summary.AnomalyStats = anomalyStats
	}

	// Get recent open anomalies
	openAnomalies, err := s.anomalyRepo.GetOpen(ctx, tenantID, nil)
	if err == nil {
		summary.OpenAnomaliesCount = len(openAnomalies)
		if len(openAnomalies) > 0 {
			summary.RecentAnomalies = openAnomalies[:min(len(openAnomalies), 10)]
		}
	}

	// Get SSH key usage summary (last 30 days)
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -30)
	sshSummary, err := s.sshKeyRepo.GetUsageSummary(ctx, tenantID, dateFrom, dateTo)
	if err == nil {
		summary.SSHKeySummary = sshSummary
	}

	// Get blacklist stats
	blacklistStats, err := s.blacklistRepo.GetStats(ctx, tenantID)
	if err == nil {
		summary.BlacklistStats = blacklistStats
	}

	return summary, nil
}

// DashboardSummary represents a summary of analytics for the dashboard
type DashboardSummary struct {
	TenantID           uuid.UUID                             `json:"tenant_id"`
	GeneratedAt        time.Time                             `json:"generated_at"`
	ComplianceSummary  map[string]repository.FrameworkSummary `json:"compliance_summary,omitempty"`
	AnomalyStats       *repository.AnomalyStats              `json:"anomaly_stats,omitempty"`
	OpenAnomaliesCount int                                   `json:"open_anomalies_count"`
	RecentAnomalies    []model.AnomalyDetection              `json:"recent_anomalies,omitempty"`
	SSHKeySummary      *repository.SSHKeyUsageSummary         `json:"ssh_key_summary,omitempty"`
	BlacklistStats     *repository.BlacklistStats             `json:"blacklist_stats,omitempty"`
}

// =============================================================================
// Helper Functions
// =============================================================================

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
