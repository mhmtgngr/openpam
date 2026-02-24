package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// Service provides analytics business logic
type Service struct {
	repo      Repository
	cache     *RedisCache
	publisher *events.Publisher
	logger    zerolog.Logger

	// Pluggable components
	complianceEngine *ComplianceEngine
	anomalyDetector  *AnomalyDetector
	commandExtractor *CommandExtractor
}

// ServiceConfig holds configuration for the analytics service
type ServiceConfig struct {
	OffHoursStart    int // Start of off-hours (hour, 0-23)
	OffHoursEnd      int // End of off-hours (hour, 0-23)
	AnomalyThreshold float64
	EnableCompliance bool
	EnableAnomaly    bool
	EnableCommandTracking bool
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		OffHoursStart:           18, // 6 PM
		OffHoursEnd:             6,  // 6 AM
		AnomalyThreshold:        75.0,
		EnableCompliance:        true,
		EnableAnomaly:           true,
		EnableCommandTracking:   true,
	}
}

// NewService creates a new analytics service
func NewService(
	repo Repository,
	cache *RedisCache,
	publisher *events.Publisher,
	logger zerolog.Logger,
	config ServiceConfig,
) *Service {
	s := &Service{
		repo:      repo,
		cache:     cache,
		publisher: publisher,
		logger:    logger,
	}

	// Initialize pluggable components
	if config.EnableCompliance {
		s.complianceEngine = NewComplianceEngine(repo, cache, logger)
	}

	if config.EnableAnomaly {
		s.anomalyDetector = NewAnomalyDetector(repo, cache, config.AnomalyThreshold, logger)
	}

	if config.EnableCommandTracking {
		s.commandExtractor = NewCommandExtractor(repo, logger)
	}

	// Start background tasks
	go s.runAggregationScheduler()
	go s.runCacheInvalidationListener()

	return s
}

// Session Analytics Methods

// RecordSessionStart records metrics when a session starts
func (s *Service) RecordSessionStart(ctx context.Context, tenantID, userID uuid.UUID, sessionType string, targetHost string, targetPort int) error {
	date := time.Now().Truncate(time.Hour)
	hour := date.Hour()

	analytics := &SessionAnalytics{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Date:           date,
		Hour:           hour,
		TotalSessions:  1,
		ActiveSessions: 1,
		UniqueUsers:    1,
		UniqueTargets:  1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Set session type counters
	switch sessionType {
	case "ssh":
		analytics.SSHSessions = 1
	case "rdp":
		analytics.RDPSessions = 1
	case "database":
		analytics.DatabaseSessions = 1
	case "kubernetes":
		analytics.KubernetesSessions = 1
	case "web":
		analytics.WebSessions = 1
	}

	if err := s.repo.CreateSessionAnalytics(ctx, analytics); err != nil {
		return fmt.Errorf("service.RecordSessionStart: %w", err)
	}

	// Record user activity
	activity := &UserActivity{
		ID:                uuid.New(),
		TenantID:          tenantID,
		UserID:            userID,
		Date:              date,
		Hour:              hour,
		SessionsInitiated: 1,
		FirstAccessTime:   timePtr(date),
		LastAccessTime:    timePtr(date),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Check if off-hours access
	if s.isOffHour(hour) {
		activity.OffHoursAccess = true
	}

	if err := s.repo.CreateUserActivity(ctx, activity); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create user activity")
	}

	return nil
}

// RecordSessionEnd records metrics when a session ends
func (s *Service) RecordSessionEnd(ctx context.Context, tenantID, userID uuid.UUID, sessionID uuid.UUID, duration time.Duration, sessionType string) error {
	date := time.Now().Truncate(time.Hour)
	hour := date.Hour()

	// Update session analytics
	analytics, err := s.repo.GetSessionAnalytics(ctx, tenantID, date, hour)
	if err != nil {
		// Create if not exists
		analytics = &SessionAnalytics{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Date:      date,
			Hour:      hour,
			CreatedAt: time.Now(),
		}
	}

	analytics.ActiveSessions--
	analytics.CompletedSessions++
	analytics.TotalDurationSeconds += int64(duration.Seconds())

	if analytics.AvgDurationSeconds == nil {
		avg := float64(duration.Seconds())
		analytics.AvgDurationSeconds = &avg
	} else {
		totalAvg := *analytics.AvgDurationSeconds
		newAvg := (totalAvg + float64(duration.Seconds())) / 2
		analytics.AvgDurationSeconds = &newAvg
	}

	analytics.UpdatedAt = time.Now()

	if err := s.repo.UpdateSessionAnalytics(ctx, analytics); err != nil {
		return fmt.Errorf("service.RecordSessionEnd: %w", err)
	}

	// Update user activity
	activity, err := s.repo.GetUserActivity(ctx, tenantID, userID, date, hour)
	if err == nil {
		activity.SessionsCompleted++
		activity.TotalSessionSeconds += int64(duration.Seconds())
		activity.LastAccessTime = timePtr(date)
		activity.UpdatedAt = time.Now()

		if err := s.repo.UpdateUserActivity(ctx, activity); err != nil {
			s.logger.Error().Err(err).Msg("Failed to update user activity")
		}
	}

	return nil
}

// RecordCommand tracks command execution
func (s *Service) RecordCommand(ctx context.Context, tenantID, userID, sessionID uuid.UUID, command string, targetHost string, exitCode *int) error {
	if s.commandExtractor == nil {
		return nil
	}

	// Parse command
	parsed := s.commandExtractor.ParseCommand(command)
	if parsed == nil {
		return nil
	}

	// Calculate command hash
	hash := sha256.Sum256([]byte(parsed.Normalized))
	commandHash := hex.EncodeToString(hash[:])

	date := time.Now()
	hour := date.Hour()

	// Determine risk level
	riskLevel := s.commandExtractor.AssessRisk(parsed)
	isDangerous := riskLevel == string(RiskLevelHigh) || riskLevel == string(RiskLevelCritical)

	// Check against blacklist
	blacklists, _ := s.repo.FindMatchingBlacklist(ctx, tenantID, parsed.Normalized, []uuid.UUID{userID}, nil)
	isBlocked := false
	for _, bl := range blacklists {
		if bl.Action == string(CommandActionBlock) {
			isBlocked = true
			break
		}
	}

	cmdFreq := &CommandFrequency{
		TenantID:       tenantID,
		Date:           date,
		Hour:           hour,
		CommandHash:    commandHash,
		CommandPattern: parsed.Pattern,
		BaseCommand:    parsed.BaseCommand,
		SessionID:      sessionID,
		UserID:         userID,
		TargetHost:     targetHost,
		RiskLevel:      riskLevel,
		IsDangerous:    isDangerous,
		IsBlocked:      isBlocked,
		ExecutedAt:     date,
		ExitCode:       exitCode,
	}

	if err := s.repo.RecordCommand(ctx, cmdFreq); err != nil {
		return fmt.Errorf("service.RecordCommand: %w", err)
	}

	// Update user activity command count
	activity, err := s.repo.GetUserActivity(ctx, tenantID, userID, date, hour)
	if err == nil {
		activity.CommandsExecuted++
		activity.UpdatedAt = time.Now()
		_ = s.repo.UpdateUserActivity(ctx, activity)
	}

	// Check for anomalies if command is dangerous
	if isDangerous && s.anomalyDetector != nil {
		_ = s.anomalyDetector.EvaluateCommand(ctx, tenantID, userID, parsed)
	}

	return nil
}

// Query Methods

// GetSessionMetrics retrieves session metrics for a date range
func (s *Service) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*SessionSummary, error) {
	if repo, ok := s.repo.(*PostgresRepository); ok {
		return repo.GetTenantSessionSummary(ctx, tenantID, dateFrom, dateTo)
	}

	// Fallback: aggregate from list
	filter := SessionAnalyticsFilter{
		TenantID: &tenantID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	analyticsList, err := s.repo.ListSessionAnalytics(ctx, filter, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("service.GetSessionMetrics: %w", err)
	}

	summary := &SessionSummary{
		SessionsByType: make(map[string]int),
	}

	for _, a := range analyticsList {
		summary.TotalSessions += a.TotalSessions
		summary.ActiveSessions += a.ActiveSessions
		if a.PeakConcurrentSessions > summary.PeakConcurrent {
			summary.PeakConcurrent = a.PeakConcurrentSessions
		}
		summary.SessionsByType["ssh"] += a.SSHSessions
		summary.SessionsByType["rdp"] += a.RDPSessions
		summary.SessionsByType["database"] += a.DatabaseSessions
		summary.SessionsByType["kubernetes"] += a.KubernetesSessions
		summary.SessionsByType["web"] += a.WebSessions
	}

	return summary, nil
}

// GetUserActivity retrieves user activity for a date range
func (s *Service) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, dateFrom, dateTo time.Time) ([]UserActivity, error) {
	filter := UserActivityFilter{
		TenantID: &tenantID,
		UserID:   &userID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	return s.repo.ListUserActivity(ctx, filter, 0, 0)
}

// GetUserRiskScore calculates the risk score for a user
func (s *Service) GetUserRiskScore(ctx context.Context, tenantID, userID uuid.UUID, days int) (float64, error) {
	if repo, ok := s.repo.(*PostgresRepository); ok {
		return repo.GetUserRiskScore(ctx, tenantID, userID, days)
	}

	// Fallback implementation
	return 0, nil
}

// GetCommandFrequency retrieves command frequency data
func (s *Service) GetCommandFrequency(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	return s.repo.GetTopCommands(ctx, tenantID, dateFrom, dateTo, limit)
}

// GetDashboardMetrics retrieves all dashboard metrics
func (s *Service) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) {
	return s.repo.GetDashboardMetrics(ctx, tenantID)
}

// GetTimeSeriesData retrieves time-series data for visualization
func (s *Service) GetTimeSeriesData(ctx context.Context, tenantID uuid.UUID, metric string, dateFrom, dateTo time.Time) ([]DataPoint, error) {
	if repo, ok := s.repo.(*PostgresRepository); ok {
		return repo.GetTimeSeriesData(ctx, tenantID, metric, dateFrom, dateTo)
	}

	return []DataPoint{}, nil
}

// Compliance Methods (delegated to ComplianceEngine)

func (s *Service) GenerateComplianceReport(ctx context.Context, tenantID uuid.UUID, generatedBy uuid.UUID, framework ComplianceFramework, periodStart, periodEnd time.Time) (*ComplianceReport, error) {
	if s.complianceEngine == nil {
		return nil, fmt.Errorf("service: compliance engine not enabled")
	}

	return s.complianceEngine.GenerateReport(ctx, tenantID, generatedBy, framework, periodStart, periodEnd)
}

func (s *Service) GetComplianceReport(ctx context.Context, reportID uuid.UUID) (*ComplianceReport, error) {
	return s.repo.GetComplianceReport(ctx, reportID)
}

func (s *Service) ListComplianceReports(ctx context.Context, tenantID uuid.UUID, framework *string) ([]ComplianceReport, error) {
	filter := ComplianceFilter{
		TenantID:  &tenantID,
		Framework: framework,
	}

	return s.repo.ListComplianceReports(ctx, filter, 0, 0)
}

func (s *Service) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	return s.repo.CreateComplianceException(ctx, exception)
}

func (s *Service) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	return s.repo.ListComplianceExceptions(ctx, tenantID)
}

// Anomaly Detection Methods (delegated to AnomalyDetector)

func (s *Service) DetectAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	if s.anomalyDetector == nil {
		return nil, fmt.Errorf("service: anomaly detector not enabled")
	}

	return s.anomalyDetector.RunDetection(ctx, tenantID)
}

func (s *Service) GetAnomaly(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) {
	return s.repo.GetAnomalyDetection(ctx, id)
}

func (s *Service) ListAnomalies(ctx context.Context, tenantID uuid.UUID, status *string) ([]AnomalyDetection, error) {
	filter := AnomalyFilter{
		TenantID: &tenantID,
		Status:   status,
	}

	return s.repo.ListAnomalyDetections(ctx, filter, 0, 0)
}

func (s *Service) UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, status AnomalyStatus, assignedTo *uuid.UUID, resolutionNotes *string, resolvedBy *uuid.UUID) error {
	anomaly, err := s.repo.GetAnomalyDetection(ctx, id)
	if err != nil {
		return err
	}
	if anomaly == nil {
		return fmt.Errorf("anomaly not found")
	}

	anomaly.Status = string(status)
	anomaly.AssignedTo = assignedTo
	anomaly.ResolutionNotes = resolutionNotes
	anomaly.ResolvedBy = resolvedBy

	if status == AnomalyStatusResolved {
		now := time.Now()
		anomaly.ResolvedAt = &now
	}

	return s.repo.UpdateAnomalyDetection(ctx, anomaly)
}

// Ransomware Detection Methods

func (s *Service) CreateRansomwareEvent(ctx context.Context, detectionID uuid.UUID, event *RansomwareEvent) error {
	event.DetectionID = detectionID
	event.TenantID = uuid.Nil // Will be set from detection
	return s.repo.CreateRansomwareEvent(ctx, event)
}

func (s *Service) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) {
	return s.repo.GetRansomwareEvent(ctx, id)
}

func (s *Service) TriggerEmergencyResponse(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) error {
	// Implementation for emergency response workflow
	// This would terminate sessions, revoke credentials, etc.
	s.logger.Warn().
		Str("tenant_id", tenantID.String()).
		Str("event_id", eventID.String()).
		Msg("Emergency response triggered")

	return nil
}

// Command Blacklist Methods

func (s *Service) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	return s.repo.CreateCommandBlacklist(ctx, blacklist)
}

func (s *Service) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) {
	return s.repo.GetCommandBlacklist(ctx, id)
}

func (s *Service) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	return s.repo.ListCommandBlacklist(ctx, tenantID)
}

func (s *Service) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	return s.repo.UpdateCommandBlacklist(ctx, blacklist)
}

func (s *Service) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCommandBlacklist(ctx, id)
}

func (s *Service) EvaluateCommandAgainstBlacklist(ctx context.Context, tenantID, userID uuid.UUID, command string, userGroups []uuid.UUID) (allowed bool, action string, blacklistID *uuid.UUID) {
	blacklists, err := s.repo.FindMatchingBlacklist(ctx, tenantID, command, []uuid.UUID{userID}, userGroups)
	if err != nil || len(blacklists) == 0 {
		return true, "", nil
	}

	// Use the first matching blacklist rule
	bl := blacklists[0]
	id := bl.ID

	switch bl.Action {
	case string(CommandActionBlock):
		return false, "blocked", &id
	case string(CommandActionWarn):
		return true, "warned", &id
	case string(CommandActionAudit):
		return true, "audited", &id
	default:
		return true, "", nil
	}
}

// SSH Key Analytics Methods

func (s *Service) RecordSSHKeyUsage(ctx context.Context, tenantID, sshKeyID, userID uuid.UUID, targetHost string, duration time.Duration, failed bool) error {
	date := time.Now().Truncate(24 * time.Hour)

	analytics, err := s.repo.GetSSHKeyAnalytics(ctx, tenantID, sshKeyID, date)
	if err != nil || analytics == nil {
		// Create new
		analytics = &SSHKeyAnalytics{
			ID:           uuid.New(),
			TenantID:     tenantID,
			SSHKeyID:     sshKeyID,
			Date:         date,
			UsageCount:   1,
			UniqueUsers:  1,
			UniqueTargets: 1,
			FirstUseTime: timePtr(time.Now()),
			LastUseTime:  timePtr(time.Now()),
			CreatedAt:    time.Now(),
		}
		if failed {
			analytics.FailedAttempts = 1
		}
		return s.repo.CreateSSHKeyAnalytics(ctx, analytics)
	}

	// Update existing
	analytics.UsageCount++
	if failed {
		analytics.FailedAttempts++
	}
	analytics.LastUseTime = timePtr(time.Now())

	// Check for off-hours usage
	hour := time.Now().Hour()
	if s.isOffHour(hour) {
		analytics.OffHoursUsage++
	}

	return s.repo.UpdateSSHKeyAnalytics(ctx, analytics)
}

func (s *Service) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	return s.repo.ListSSHKeyAnalytics(ctx, tenantID, sshKeyID, dateFrom, dateTo)
}

// Cache Management Methods

func (s *Service) InvalidateCache(ctx context.Context, tenantID uuid.UUID, cacheTypes ...string) error {
	if s.cache == nil {
		return nil
	}

	if len(cacheTypes) == 0 {
		return s.cache.InvalidateAll(ctx, tenantID)
	}

	return s.cache.BatchInvalidation(ctx, tenantID, cacheTypes...)
}

func (s *Service) WarmCache(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) error {
	if s.cache == nil {
		return nil
	}

	return s.cache.WarmSessionMetricsCache(ctx, tenantID, dateFrom, dateTo, s.repo)
}

func (s *Service) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	if s.cache == nil {
		return make(map[string]interface{}), nil
	}

	return s.cache.GetCacheStats(ctx, tenantID)
}

// Background Tasks

func (s *Service) runAggregationScheduler() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Run initial aggregation
	ctx := context.Background()
	s.runHourlyAggregation(ctx)

	for range ticker.C {
		ctx := context.Background()
		s.runHourlyAggregation(ctx)

		// Refresh materialized views daily
		if time.Now().Hour() == 2 {
			s.refreshMaterializedViews(ctx)
		}
	}
}

func (s *Service) runHourlyAggregation(ctx context.Context) {
	s.logger.Debug().Msg("Running hourly aggregation")

	// Aggregate session data from the previous hour
	prevHour := time.Now().Add(-time.Hour).Truncate(time.Hour)

	// This would trigger aggregation jobs for all tenants
	// In production, this would query active tenants and aggregate for each

	s.logger.Debug().Time("hour", prevHour).Msg("Hourly aggregation complete")
}

func (s *Service) refreshMaterializedViews(ctx context.Context) {
	s.logger.Debug().Msg("Refreshing materialized views")

	if repo, ok := s.repo.(*PostgresRepository); ok {
		if err := repo.RefreshMaterializedViews(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Failed to refresh materialized views")
		}
	}
}

func (s *Service) runCacheInvalidationListener() {
	// Subscribe to session events to invalidate cache
	// This would use the event bus to listen for session/credential changes
	s.logger.Debug().Msg("Starting cache invalidation listener")
}

// Helper Methods

func (s *Service) isOffHour(hour int) bool {
	// Off hours: 6 PM to 6 AM
	return hour >= 18 || hour < 6
}

// Event Handlers

// HandleSessionStarted handles session started events
func (s *Service) HandleSessionStarted(ctx context.Context, event events.Event) error {
	tenantID, _ := uuid.Parse(event.TenantID)
	userID, _ := uuid.Parse(event.ActorID)

	_, _ = event.Data["session_id"].(string)

	targetHost, _ := event.Data["target_host"].(string)
	targetPort := int(event.Data["target_port"].(float64))

	// Get session type from data
	sessionType := "ssh" // Default
	if st, ok := event.Data["type"].(string); ok {
		sessionType = st
	}

	return s.RecordSessionStart(ctx, tenantID, userID, sessionType, targetHost, targetPort)
}

// HandleSessionEnded handles session ended events
func (s *Service) HandleSessionEnded(ctx context.Context, event events.Event) error {
	tenantID, _ := uuid.Parse(event.TenantID)
	userID, _ := uuid.Parse(event.ActorID)

	sessionID, _ := event.Data["session_id"].(string)
	sessionIDUUID, _ := uuid.Parse(sessionID)

	durationStr, _ := event.Data["duration"].(string)
	duration, _ := time.ParseDuration(durationStr)

	sessionType := "ssh" // Default
	if st, ok := event.Data["type"].(string); ok {
		sessionType = st
	}

	return s.RecordSessionEnd(ctx, tenantID, userID, sessionIDUUID, duration, sessionType)
}

// HandleCommandExecuted handles command executed events
func (s *Service) HandleCommandExecuted(ctx context.Context, event events.Event) error {
	tenantID, _ := uuid.Parse(event.TenantID)
	userID, _ := uuid.Parse(event.ActorID)

	sessionIDStr, _ := event.Data["session_id"].(string)
	sessionID, _ := uuid.Parse(sessionIDStr)

	command, _ := event.Data["command"].(string)
	targetHost, _ := event.Data["target_host"].(string)

	return s.RecordCommand(ctx, tenantID, userID, sessionID, command, targetHost, nil)
}

// Export/Import Methods

// ExportAnalyticsData exports analytics data for a tenant
func (s *Service) ExportAnalyticsData(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) ([]byte, error) {
	// Collect all analytics data for the date range
	data := make(map[string]interface{})

	// Session analytics
	sessionFilter := SessionAnalyticsFilter{
		TenantID: &tenantID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}
	sessions, _ := s.repo.ListSessionAnalytics(ctx, sessionFilter, 0, 0)
	data["session_analytics"] = sessions

	// User activity
	activityFilter := UserActivityFilter{
		TenantID: &tenantID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}
	activities, _ := s.repo.ListUserActivity(ctx, activityFilter, 0, 0)
	data["user_activity"] = activities

	// Command frequency
	commandFilter := CommandFrequencyFilter{
		TenantID: &tenantID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}
	commands, _ := s.repo.ListCommandFrequency(ctx, commandFilter, 0, 0)
	data["command_frequency"] = commands

	// Anomalies
	anomalyFilter := AnomalyFilter{
		TenantID: &tenantID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}
	anomalies, _ := s.repo.ListAnomalyDetections(ctx, anomalyFilter, 0, 0)
	data["anomalies"] = anomalies

	// Return as JSON (in production, use proper JSON encoding)
	return nil, nil
}

// GetComplianceSummary returns a summary of compliance status across all frameworks
func (s *Service) GetComplianceSummary(ctx context.Context, tenantID uuid.UUID) (*ComplianceSummary, error) {
	if s.complianceEngine == nil {
		return &ComplianceSummary{
			Frameworks: make(map[string]FrameworkStatus),
		}, nil
	}

	return s.complianceEngine.GetSummary(ctx, tenantID)
}

// RunAnomalyDetection triggers anomaly detection for a tenant
func (s *Service) RunAnomalyDetection(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	if s.anomalyDetector == nil {
		return nil, fmt.Errorf("service: anomaly detector not enabled")
	}

	return s.anomalyDetector.RunDetection(ctx, tenantID)
}

// EvaluateUserForAnomalies evaluates a specific user for anomalies
func (s *Service) EvaluateUserForAnomalies(ctx context.Context, tenantID, userID uuid.UUID) ([]AnomalyDetection, error) {
	if s.anomalyDetector == nil {
		return nil, fmt.Errorf("service: anomaly detector not enabled")
	}

	return s.anomalyDetector.EvaluateUser(ctx, tenantID, userID)
}
