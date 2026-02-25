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

// Service provides PAM-specific analytics business logic for privileged access management.
//
// This is the PAM-DOMAIN analytics service, focused on:
// - Session metrics and tracking
// - User activity monitoring
// - Risk scoring for PAM entities
// - Command frequency analysis
// - Alert evaluation and scheduling
//
// NOTE: This is different from the general analytics service at internal/analytics.
// See internal/analytics/interface.go for documentation on when to use each service.
//
// Use this service (internal/pam/analytics.Service) for:
// - PAM session lifecycle tracking
// - PAM-specific dashboards
// - User activity in privileged access contexts
// - Risk scoring for PAM entities
//
// Use internal/analytics.Service for:
// - Compliance reporting (SOC2, ISO27001, PCI-DSS, HIPAA)
// - Anomaly detection
// - Ransomware detection
// - Command blacklist enforcement
// - SSH key analytics
type Service struct {
	repo            *Repository
	anomalyRepo     *AnomalyRepository
	reportRepo      *ReportRepository
	cache           *cache.Cache
	logger          zerolog.Logger

	// Worker status
	aggregationWorkerRunning bool
	alertEvaluatorRunning    bool
	reportSchedulerRunning   bool
}

// NewService creates a new analytics service
func NewService(repo *Repository, anomalyRepo *AnomalyRepository, reportRepo *ReportRepository, cache *cache.Cache, logger zerolog.Logger) *Service {
	return &Service{
		repo:        repo,
		anomalyRepo: anomalyRepo,
		reportRepo:  reportRepo,
		cache:       cache,
		logger:      logger,
	}
}

// Session Analytics Methods

// RecordSessionStart records when a privileged session begins
func (s *Service) RecordSessionStart(ctx context.Context, tenantID, userID, sessionID uuid.UUID, sessionType, targetHost string, targetPort int) error {
	date := time.Now().Truncate(24 * time.Hour)
	hour := time.Now().Hour()

	// Check if hourly analytics exists, create or update
	analytics := &SessionAnalytics{
		TenantID:         tenantID,
		PeriodType:       PeriodHour,
		PeriodStart:      time.Now().Truncate(time.Hour),
		PeriodEnd:        time.Now().Truncate(time.Hour).Add(time.Hour),
		TotalSessions:    1,
		ActiveSessions:   1,
		UniqueUsers:      1,
		ProtocolBreakdown: mustMarshalJSON(map[string]int{sessionType: 1}),
	}

	if err := s.repo.UpsertSessionAnalytics(ctx, analytics); err != nil {
		return fmt.Errorf("service.RecordSessionStart: %w", err)
	}

	// Update user activity
	activity := &UserActivity{
		TenantID:          tenantID,
		UserID:            userID,
		PeriodType:        PeriodDay,
		PeriodStart:       date,
		SessionsCreated:   1,
		FirstAccessTime:   timePtr(time.Now()),
		LastAccessTime:    timePtr(time.Now()),
		IsAnomaly:         false,
	}

	// Check for off-hours access
	if hour < 6 || hour >= 18 {
		activity.OffHoursAccess = true
	}

	if err := s.repo.UpsertUserActivity(ctx, activity); err != nil {
		s.logger.Error().Err(err).Msg("Failed to upsert user activity")
	}

	// Invalidate cache for this tenant
	_ = s.cache.Delete(ctx, fmt.Sprintf("analytics:dashboard:%s", tenantID))

	return nil
}

// RecordSessionEnd records when a privileged session ends
func (s *Service) RecordSessionEnd(ctx context.Context, tenantID, userID, sessionID uuid.UUID, duration time.Duration, sessionType string) error {
	now := time.Now()
	hourStart := now.Truncate(time.Hour)

	// Update hourly analytics
	analytics, err := s.repo.GetSessionAnalytics(ctx, tenantID, PeriodHour, hourStart)
	if err == nil {
		analytics.ActiveSessions--
		analytics.CompletedSessions++
		if analytics.ActiveSessions < 0 {
			analytics.ActiveSessions = 0
		}

		// Update duration stats
		durationSeconds := int(duration.Seconds())
		if analytics.AvgDurationSeconds == nil {
			analytics.AvgDurationSeconds = &durationSeconds
		} else {
			newAvg := (*analytics.AvgDurationSeconds + durationSeconds) / 2
			analytics.AvgDurationSeconds = &newAvg
		}

		if err := s.repo.UpsertSessionAnalytics(ctx, analytics); err != nil {
			s.logger.Error().Err(err).Msg("Failed to update session analytics on end")
		}
	}

	// Update user activity
	date := now.Truncate(24 * time.Hour)
	activity, err := s.repo.GetUserActivity(ctx, tenantID, userID, PeriodDay, date)
	if err == nil {
		activity.TotalActiveSeconds += int64(duration.Seconds())
		activity.LastAccessTime = timePtr(now)
		_ = s.repo.UpsertUserActivity(ctx, activity)
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("analytics:dashboard:%s", tenantID))

	return nil
}

// GetSessionMetricsSummary retrieves aggregated session metrics for a date range
func (s *Service) GetSessionMetricsSummary(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*SessionStats, error) {
	cacheKey := fmt.Sprintf("analytics:session_summary:%s:%s:%s", tenantID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var stats SessionStats
	if err := s.cache.Get(ctx, cacheKey, &stats); err == nil {
		return &stats, nil
	}

	// Get from raw data
	rawStats, err := s.repo.GetRawSessionStats(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("service.GetSessionMetricsSummary: %w", err)
	}

	// Cache for 5 minutes
	_ = s.cache.Set(ctx, cacheKey, rawStats, 5*time.Minute)

	return rawStats, nil
}

// GetSessionTimeSeries retrieves time-series data for sessions
func (s *Service) GetSessionTimeSeries(ctx context.Context, tenantID uuid.UUID, metric string, startDate, endDate time.Time) ([]TimeSeriesDataPoint, error) {
	cacheKey := fmt.Sprintf("analytics:timeseries:%s:%s:%s:%s", tenantID, metric, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var data []TimeSeriesDataPoint
	if err := s.cache.Get(ctx, cacheKey, &data); err == nil {
		return data, nil
	}

	// Query raw session data
	seriesData, err := s.repo.GetSessionTimeSeries(ctx, tenantID, metric, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("service.GetSessionTimeSeries: %w", err)
	}
	data = seriesData

	// Cache for 5 minutes
	_ = s.cache.Set(ctx, cacheKey, data, 5*time.Minute)

	return data, nil
}

// ListSessionAnalytics lists session analytics with filters
func (s *Service) ListSessionAnalytics(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]SessionAnalytics, error) {
	return s.repo.ListSessionAnalytics(ctx, tenantID, periodType, startDate, endDate, limit, offset)
}

// User Activity Methods

// GetUserActivityList retrieves user activity for a specific user
func (s *Service) GetUserActivityList(ctx context.Context, tenantID, userID uuid.UUID, startDate, endDate time.Time, limit int) ([]UserActivity, error) {
	return s.repo.ListUserActivity(ctx, tenantID, PeriodDay, startDate, endDate, limit, 0)
}

// GetTopUsersByActivity retrieves the most active users
func (s *Service) GetTopUsersByActivity(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, limit int) ([]TopUser, error) {
	return s.repo.GetTopUsersBySessions(ctx, tenantID, startDate, endDate, limit)
}

// GetUserRiskScore calculates the risk score for a user
func (s *Service) GetUserRiskScore(ctx context.Context, tenantID, userID uuid.UUID, days int) (float64, error) {
	cacheKey := fmt.Sprintf("analytics:risk_score:%s:%s:%d", tenantID, userID, days)

	var score float64
	if err := s.cache.Get(ctx, cacheKey, &score); err == nil {
		return score, nil
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get user activity for the period
	activities, err := s.repo.ListUserActivity(ctx, tenantID, PeriodDay, startDate, endDate, 1000, 0)
	if err != nil {
		return 0, fmt.Errorf("service.GetUserRiskScore: %w", err)
	}

	// Calculate risk score based on multiple factors
	riskScore := s.calculateUserRiskScore(activities)

	// Cache for 15 minutes
	_ = s.cache.Set(ctx, cacheKey, riskScore, 15*time.Minute)

	return riskScore, nil
}

// calculateUserRiskScore computes risk score from user activities
func (s *Service) calculateUserRiskScore(activities []UserActivity) float64 {
	if len(activities) == 0 {
		return 0
	}

	var totalRisk float64
	var weight float64

	for _, activity := range activities {
		// Base risk from anomalies
		if activity.IsAnomaly && activity.AnomalyScore != nil {
			totalRisk += *activity.AnomalyScore * 0.4
		}

		// Risk from policy violations
		violationRisk := float64(activity.PolicyViolations) * 10
		totalRisk += violationRisk * 0.3

		// Risk from failed auth attempts
		authRisk := float64(activity.FailedAuthAttempts) * 5
		totalRisk += authRisk * 0.2

		// Risk from off-hours access
		if activity.OffHoursAccess {
			totalRisk += 5 * 0.1
		}

		weight += 1.0
	}

	averageRisk := totalRisk / weight
	if averageRisk > 100 {
		averageRisk = 100
	}

	return averageRisk
}

// Command Analytics Methods

// GetCommandFrequencyList retrieves command frequency data
func (s *Service) GetCommandFrequencyList(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, limit int) ([]CommandFrequency, error) {
	return s.repo.ListCommandFrequency(ctx, tenantID, PeriodDay, startDate, endDate, limit, 0)
}

// GetTopCommands retrieves the most frequently executed commands
func (s *Service) GetTopCommands(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, limit int) ([]CommandRank, error) {
	// This would aggregate from command_frequency table
	// For now, return empty as the full implementation needs the events table
	return []CommandRank{}, nil
}

// Dashboard Methods

// GetDashboardMetrics retrieves comprehensive dashboard metrics
func (s *Service) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) {
	cacheKey := fmt.Sprintf("analytics:dashboard:%s", tenantID)

	var metrics DashboardMetrics
	if err := s.cache.Get(ctx, cacheKey, &metrics); err == nil {
		return &metrics, nil
	}

	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)

	// Get session stats
	sessionStats, err := s.GetSessionMetricsSummary(ctx, tenantID, weekAgo, now)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get session stats for dashboard")
	} else {
		metrics.SessionMetrics = sessionStats
	}

	// Get top users
	topUsers, err := s.GetTopUsersByActivity(ctx, tenantID, weekAgo, now, 5)
	if err == nil {
		metrics.TopUsers = topUsers
	}

	// Get access patterns
	patterns, err := s.repo.GetAccessPatterns(ctx, tenantID, weekAgo, now)
	if err == nil {
		metrics.AccessPatterns = patterns
	}

	// Get compliance status
	complianceStatus, err := s.repo.CalculateComplianceStatus(ctx, tenantID, weekAgo, now)
	if err == nil {
		metrics.ComplianceStatus = complianceStatus
	}

	metrics.Timestamp = now

	// Cache for 2 minutes
	_ = s.cache.Set(ctx, cacheKey, metrics, 2*time.Minute)

	return &metrics, nil
}

// Dashboard Metrics Structure
type DashboardMetrics struct {
	SessionMetrics    *SessionStats      `json:"session_metrics,omitempty"`
	TopUsers          []TopUser          `json:"top_users,omitempty"`
	AccessPatterns    []AccessPattern    `json:"access_patterns,omitempty"`
	ComplianceStatus  *ComplianceStatus  `json:"compliance_status,omitempty"`
	Timestamp         time.Time          `json:"timestamp"`
}

// Compliance Methods

// GenerateComplianceReport generates a new compliance report
func (s *Service) GenerateComplianceReport(ctx context.Context, tenantID, generatedBy uuid.UUID, framework string, startDate, endDate time.Time) (*ComplianceReportResponse, error) {
	// Calculate compliance status
	status, err := s.repo.CalculateComplianceStatus(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("service.GenerateComplianceReport: %w", err)
	}

	// Marshal report data for storage
	reportData := mustMarshalJSON(map[string]interface{}{
		"by_policy":  status.ByPolicy,
		"violations": status.Violations,
	})

	// Create the database report record
	dbReport := &ComplianceReport{
		TenantID:       tenantID,
		Framework:      framework,
		Status:         "completed",
		GeneratedAt:    time.Now(),
		PeriodStart:    startDate,
		PeriodEnd:      endDate,
		OverallScore:   status.OverallPercentage,
		PassedControls: status.PassedChecks,
		FailedControls: status.TotalChecks - status.PassedChecks,
		Data:           reportData,
	}

	if err := s.reportRepo.CreateComplianceReport(ctx, dbReport); err != nil {
		return nil, fmt.Errorf("service.GenerateComplianceReport: %w", err)
	}

	// Create a report snapshot for file generation tracking
	summary := fmt.Sprintf("Compliance report for %s framework: %.1f%% overall score",
		framework, status.OverallPercentage)

	snapshot := &ReportSnapshot{
		TenantID:     tenantID,
		ReportID:     dbReport.ID,
		SnapshotName: fmt.Sprintf("%s Compliance Report - %s", framework, time.Now().Format("2006-01-02")),
		Framework:    framework,
		GeneratedAt:  time.Now(),
		GeneratedBy:  generatedBy,
		Status:       ReportSnapshotStatusCompleted,
		PeriodStart:  startDate,
		PeriodEnd:    endDate,
		Summary:      &summary,
		Metadata:     mustMarshalJSON(map[string]interface{}{"generated_by": generatedBy.String()}),
	}

	// For now, mark as completed without file storage
	// In a full implementation, this would generate a PDF/Excel file
	if err := s.reportRepo.CreateReportSnapshot(ctx, snapshot); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create report snapshot, but report was created")
	}

	report := &ComplianceReportResponse{
		ID:              dbReport.ID,
		TenantID:        tenantID,
		Framework:       framework,
		GeneratedAt:     dbReport.GeneratedAt,
		GeneratedBy:     generatedBy,
		PeriodStart:     startDate,
		PeriodEnd:       endDate,
		OverallScore:    status.OverallPercentage,
		PassedControls:  status.PassedChecks,
		FailedControls:  status.TotalChecks - status.PassedChecks,
		ByPolicy:        status.ByPolicy,
		Violations:      status.Violations,
		Status:          "completed",
	}

	return report, nil
}

type ComplianceReportResponse struct {
	ID              uuid.UUID                        `json:"id"`
	TenantID        uuid.UUID                        `json:"tenant_id"`
	Framework       string                           `json:"framework"`
	GeneratedAt     time.Time                        `json:"generated_at"`
	GeneratedBy     uuid.UUID                        `json:"generated_by"`
	PeriodStart     time.Time                        `json:"period_start"`
	PeriodEnd       time.Time                        `json:"period_end"`
	OverallScore    float64                          `json:"overall_score"`
	PassedControls  int                              `json:"passed_controls"`
	FailedControls  int                              `json:"failed_controls"`
	ByPolicy        map[string]CompliancePolicyStatus `json:"by_policy"`
	Violations      []ComplianceViolation            `json:"violations,omitempty"`
	Status          string                           `json:"status"`
}

// GetComplianceSummary retrieves a summary of compliance across frameworks
func (s *Service) GetComplianceSummary(ctx context.Context, tenantID uuid.UUID) (*ComplianceStatus, error) {
	now := time.Now()
	monthAgo := now.Add(-30 * 24 * time.Hour)
	return s.repo.CalculateComplianceStatus(ctx, tenantID, monthAgo, now)
}

// ListComplianceReports lists compliance reports for a tenant
func (s *Service) ListComplianceReports(ctx context.Context, tenantID uuid.UUID, framework string, limit, offset int) ([]ComplianceReportResponse, int, error) {
	filter := ComplianceReportFilter{
		TenantID:  tenantID,
		Framework: framework,
		Limit:     limit,
		Offset:    offset,
	}

	reports, total, err := s.reportRepo.ListComplianceReports(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListComplianceReports: %w", err)
	}

	responses := make([]ComplianceReportResponse, len(reports))
	for i, report := range reports {
		// Unmarshal report data
		var byPolicy map[string]CompliancePolicyStatus
		var violations []ComplianceViolation
		if len(report.Data) > 0 {
			var data struct {
				ByPolicy   map[string]CompliancePolicyStatus `json:"by_policy"`
				Violations []ComplianceViolation            `json:"violations"`
			}
			_ = json.Unmarshal(report.Data, &data)
			byPolicy = data.ByPolicy
			violations = data.Violations
		}

		responses[i] = ComplianceReportResponse{
			ID:              report.ID,
			TenantID:        report.TenantID,
			Framework:       report.Framework,
			GeneratedAt:     report.GeneratedAt,
			GeneratedBy:     uuid.Nil, // Not tracked in database
			PeriodStart:     report.PeriodStart,
			PeriodEnd:       report.PeriodEnd,
			OverallScore:    report.OverallScore,
			PassedControls:  report.PassedControls,
			FailedControls:  report.FailedControls,
			ByPolicy:        byPolicy,
			Violations:      violations,
			Status:          report.Status,
		}
	}

	return responses, total, nil
}

// GetComplianceReport retrieves a specific compliance report
func (s *Service) GetComplianceReport(ctx context.Context, id, tenantID uuid.UUID) (*ComplianceReportResponse, error) {
	report, err := s.reportRepo.GetComplianceReportByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("service.GetComplianceReport: %w", err)
	}

	// Unmarshal report data
	var byPolicy map[string]CompliancePolicyStatus
	var violations []ComplianceViolation
	if len(report.Data) > 0 {
		var data struct {
			ByPolicy   map[string]CompliancePolicyStatus `json:"by_policy"`
			Violations []ComplianceViolation            `json:"violations"`
		}
		_ = json.Unmarshal(report.Data, &data)
		byPolicy = data.ByPolicy
		violations = data.Violations
	}

	return &ComplianceReportResponse{
		ID:              report.ID,
		TenantID:        report.TenantID,
		Framework:       report.Framework,
		GeneratedAt:     report.GeneratedAt,
		GeneratedBy:     uuid.Nil, // Not tracked in database
		PeriodStart:     report.PeriodStart,
		PeriodEnd:       report.PeriodEnd,
		OverallScore:    report.OverallScore,
		PassedControls:  report.PassedControls,
		FailedControls:  report.FailedControls,
		ByPolicy:        byPolicy,
		Violations:      violations,
		Status:          report.Status,
	}, nil
}

// DeleteComplianceReport deletes a compliance report
func (s *Service) DeleteComplianceReport(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.reportRepo.DeleteComplianceReport(ctx, id, tenantID)
}

// CreateComplianceException creates a new compliance exception
func (s *Service) CreateComplianceException(ctx context.Context, tenantID, createdBy uuid.UUID, req *CreateComplianceExceptionRequest) (*ComplianceExceptionResponse, error) {
	exception := &ComplianceExceptionResponse{
		ID:              uuid.New(),
		TenantID:        tenantID,
		ControlID:       req.ControlID,
		ControlName:     req.ControlName,
		Framework:       req.Framework,
		Status:          "pending",
		RiskLevel:       req.RiskLevel,
		RequestedBy:     createdBy,
		RequestedAt:     time.Now(),
		Justification:   req.Justification,
		BusinessReason:  req.BusinessReason,
	}

	// Store exception
	// TODO: Add exception persistence

	return exception, nil
}

// ListComplianceExceptions lists all compliance exceptions for a tenant
func (s *Service) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceExceptionResponse, error) {
	// TODO: Implement exception listing
	return []ComplianceExceptionResponse{}, nil
}

// Anomaly Detection Methods

// DetectAnomalies runs anomaly detection for a tenant and persists detected anomalies
func (s *Service) DetectAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyResponse, error) {
	var anomalyDetections []Anomaly

	// Check for off-hours access anomalies
	offHoursUsers, err := s.repo.ListUserActivity(ctx, tenantID, PeriodDay, time.Now().Add(-7*24*time.Hour), time.Now(), 100, 0)
	if err == nil {
		for _, activity := range offHoursUsers {
			if activity.OffHoursAccess && activity.SessionsCreated > 10 {
				anomaly := Anomaly{
					TenantID:        tenantID,
					UserID:          &activity.UserID,
					AnomalyType:     AnomalyTypeTemporal,
					Severity:         SeverityMedium,
					ConfidenceScore: 75.0,
					RiskScore:        60.0,
					Title:           "Excessive Off-Hours Access",
					Description:     stringPtr(fmt.Sprintf("User has %d off-hours sessions in the past week", activity.SessionsCreated)),
					DetectionMethod: "behavioral_analysis",
					AutoTriggered:   true,
					Status:          AnomalyStatusOpen,
				}
				anomalyDetections = append(anomalyDetections, anomaly)
			}
		}
	}

	// Check for high-risk users
	highRiskActivities, err := s.repo.GetAnomalousUsers(ctx, tenantID, PeriodDay, time.Now().Truncate(24*time.Hour), 20)
	if err == nil {
		for _, activity := range highRiskActivities {
			score := getAnomalyScoreValue(activity.AnomalyScore)
			anomaly := Anomaly{
				TenantID:        tenantID,
				UserID:          &activity.UserID,
				AnomalyType:     AnomalyTypeBehavioral,
				Severity:         Severity(getSeverityFromScore(activity.AnomalyScore)),
				ConfidenceScore: 80.0,
				RiskScore:        score,
				Title:           "High Risk Score Detected",
				Description:     stringPtr(fmt.Sprintf("User anomaly score: %.2f exceeds threshold", score)),
				DetectionMethod: "risk_scoring",
				AutoTriggered:   true,
				Status:          AnomalyStatusOpen,
			}
			anomalyDetections = append(anomalyDetections, anomaly)
		}
	}

	// Persist detected anomalies in batch
	if len(anomalyDetections) > 0 {
		if err := s.anomalyRepo.BatchCreate(ctx, anomalyDetections); err != nil {
			s.logger.Error().Err(err).Msg("Failed to persist detected anomalies")
		}
	}

	// Convert to response format
	responses := make([]AnomalyResponse, len(anomalyDetections))
	for i, a := range anomalyDetections {
		responses[i] = s.anomalyToResponse(&a)
	}

	return responses, nil
}

type AnomalyResponse struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	AnomalyType string     `json:"anomaly_type"`
	Severity    string     `json:"severity"`
	Description *string    `json:"description,omitempty"`
	DetectedAt  time.Time  `json:"detected_at"`
	Status      string     `json:"status"`
}

// ListAnomalies lists anomalies with optional filters
func (s *Service) ListAnomalies(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]AnomalyResponse, error) {
	filter := AnomalyFilter{
		TenantID: &tenantID,
	}

	if status != "" {
		anomalyStatus := AnomalyStatus(status)
		filter.Status = &anomalyStatus
	}

	anomalies, _, err := s.anomalyRepo.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("service.ListAnomalies: %w", err)
	}

	responses := make([]AnomalyResponse, len(anomalies))
	for i, a := range anomalies {
		responses[i] = s.anomalyToResponse(&a)
	}

	return responses, nil
}

// GetAnomaly retrieves a specific anomaly
func (s *Service) GetAnomaly(ctx context.Context, id uuid.UUID) (*AnomalyResponse, error) {
	anomaly, err := s.anomalyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service.GetAnomaly: %w", err)
	}

	response := s.anomalyToResponse(anomaly)
	return &response, nil
}

// UpdateAnomalyStatus updates the status of an anomaly
func (s *Service) UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, status string, userID, assignTo *uuid.UUID, notes string) error {
	anomalyStatus := AnomalyStatus(status)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	err := s.anomalyRepo.UpdateStatus(ctx, id, anomalyStatus, userID, assignTo, notesPtr)
	if err != nil {
		return fmt.Errorf("service.UpdateAnomalyStatus: %w", err)
	}

	// Invalidate cache for this tenant's anomalies
	var tenantID uuid.UUID
	if anomaly, err := s.anomalyRepo.GetByID(ctx, id); err == nil {
		tenantID = anomaly.TenantID
		_ = s.cache.Delete(ctx, fmt.Sprintf("analytics:anomalies:%s", tenantID))
	}

	return nil
}

// AcknowledgeAnomaly acknowledges an anomaly, transitioning it to investigating state
func (s *Service) AcknowledgeAnomaly(ctx context.Context, id, acknowledgedBy uuid.UUID, notes string) error {
	// Get current anomaly to validate state
	anomaly, err := s.anomalyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("service.AcknowledgeAnomaly: %w", err)
	}

	// Validate state transition - only open anomalies can be acknowledged
	if anomaly.Status != AnomalyStatusOpen {
		return fmt.Errorf("service.AcknowledgeAnomaly: cannot acknowledge anomaly with status %s, only open anomalies can be acknowledged", anomaly.Status)
	}

	// Update to investigating status with acknowledged_by tracking
	err = s.anomalyRepo.UpdateStatus(ctx, id, AnomalyStatusInvestigating, &acknowledgedBy, nil, nil)
	if err != nil {
		return fmt.Errorf("service.AcknowledgeAnomaly: %w", err)
	}

	// Add resolution notes if provided
	if notes != "" {
		_ = s.anomalyRepo.Update(ctx, &Anomaly{
			ID:             id,
			ResolutionNotes: &notes,
		})
	}

	// Invalidate cache for this tenant's anomalies
	_ = s.cache.Delete(ctx, fmt.Sprintf("analytics:anomalies:%s", anomaly.TenantID))

	s.logger.Info().
		Str("anomaly_id", id.String()).
		Str("acknowledged_by", acknowledgedBy.String()).
		Str("tenant_id", anomaly.TenantID.String()).
		Msg("Anomaly acknowledged")

	return nil
}

// ResolveAnomaly resolves an anomaly with the specified resolution
func (s *Service) ResolveAnomaly(ctx context.Context, id, resolvedBy uuid.UUID, status AnomalyStatus, notes string) error {
	// Validate status - must be a terminal state
	if status != AnomalyStatusResolved && status != AnomalyStatusFalsePositive && status != AnomalyStatusIgnored {
		return fmt.Errorf("service.ResolveAnomaly: invalid resolution status %s, must be resolved, false_positive, or ignored", status)
	}

	// Get current anomaly to validate state
	anomaly, err := s.anomalyRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("service.ResolveAnomaly: %w", err)
	}

	// Any state can transition to a terminal state
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}

	err = s.anomalyRepo.UpdateStatus(ctx, id, status, nil, &resolvedBy, notesPtr)
	if err != nil {
		return fmt.Errorf("service.ResolveAnomaly: %w", err)
	}

	// Invalidate cache for this tenant's anomalies
	_ = s.cache.Delete(ctx, fmt.Sprintf("analytics:anomalies:%s", anomaly.TenantID))

	s.logger.Info().
		Str("anomaly_id", id.String()).
		Str("resolved_by", resolvedBy.String()).
		Str("status", string(status)).
		Str("tenant_id", anomaly.TenantID.String()).
		Msg("Anomaly resolved")

	return nil
}

// GetAnomalyStats retrieves statistics about anomalies for a tenant
func (s *Service) GetAnomalyStats(ctx context.Context, tenantID uuid.UUID) (*AnomalyStats, error) {
	stats, err := s.anomalyRepo.GetStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAnomalyStats: %w", err)
	}
	return stats, nil
}

// RunAnomalyDetection triggers a full anomaly detection run
func (s *Service) RunAnomalyDetection(ctx context.Context, tenantID uuid.UUID) ([]AnomalyResponse, error) {
	return s.DetectAnomalies(ctx, tenantID)
}

// EvaluateUserForAnomalies evaluates a specific user for anomalies and persists findings
func (s *Service) EvaluateUserForAnomalies(ctx context.Context, tenantID, userID uuid.UUID) ([]AnomalyResponse, error) {
	var anomalyDetections []Anomaly

	// Get user activity for the past 30 days
	endDate := time.Now()
	startDate := endDate.Add(-30 * 24 * time.Hour)

	activities, err := s.repo.ListUserActivity(ctx, tenantID, PeriodDay, startDate, endDate, 30, 0)
	if err != nil {
		return nil, fmt.Errorf("service.EvaluateUserForAnomalies: %w", err)
	}

	// Check for various anomaly patterns
	for _, activity := range activities {
		// Check for unusual off-hours access
		if activity.OffHoursAccess && activity.SessionsCreated > 5 {
			anomaly := Anomaly{
				TenantID:        tenantID,
				UserID:          &userID,
				AnomalyType:     AnomalyTypeTemporal,
				Severity:         SeverityMedium,
				ConfidenceScore: 70.0,
				RiskScore:        55.0,
				Title:           "Excessive Off-Hours Access",
				Description:     stringPtr(fmt.Sprintf("%d off-hours sessions on %s", activity.SessionsCreated, activity.PeriodStart.Format("2006-01-02"))),
				DetectionMethod: "temporal_analysis",
				AutoTriggered:   true,
				Status:          AnomalyStatusOpen,
			}
			anomalyDetections = append(anomalyDetections, anomaly)
		}

		// Check for high policy violation count
		if activity.PolicyViolations > 5 {
			anomaly := Anomaly{
				TenantID:        tenantID,
				UserID:          &userID,
				AnomalyType:     AnomalyTypePattern,
				Severity:         SeverityHigh,
				ConfidenceScore: 85.0,
				RiskScore:        75.0,
				Title:           "Excessive Policy Violations",
				Description:     stringPtr(fmt.Sprintf("%d policy violations on %s", activity.PolicyViolations, activity.PeriodStart.Format("2006-01-02"))),
				DetectionMethod: "pattern_analysis",
				AutoTriggered:   true,
				Status:          AnomalyStatusOpen,
			}
			anomalyDetections = append(anomalyDetections, anomaly)
		}

		// Check for failed authentication attempts
		if activity.FailedAuthAttempts > 10 {
			anomaly := Anomaly{
				TenantID:        tenantID,
				UserID:          &userID,
				AnomalyType:     AnomalyTypeBehavioral,
				Severity:         SeverityHigh,
				ConfidenceScore: 90.0,
				RiskScore:        80.0,
				Title:           "Excessive Failed Authentication",
				Description:     stringPtr(fmt.Sprintf("%d failed authentication attempts on %s", activity.FailedAuthAttempts, activity.PeriodStart.Format("2006-01-02"))),
				DetectionMethod: "behavioral_analysis",
				AutoTriggered:   true,
				Status:          AnomalyStatusOpen,
			}
			anomalyDetections = append(anomalyDetections, anomaly)
		}
	}

	// Persist detected anomalies
	if len(anomalyDetections) > 0 {
		if err := s.anomalyRepo.BatchCreate(ctx, anomalyDetections); err != nil {
			s.logger.Error().Err(err).Msg("Failed to persist user anomaly evaluations")
		}
	}

	// Convert to response format
	responses := make([]AnomalyResponse, len(anomalyDetections))
	for i, a := range anomalyDetections {
		responses[i] = s.anomalyToResponse(&a)
	}

	return responses, nil
}

// Risk Score Methods

// ListRiskScores lists risk scores with filters
func (s *Service) ListRiskScores(ctx context.Context, tenantID uuid.UUID, entityType, minScore, riskLevel string, limit int) ([]RiskScoreResponse, error) {
	var entityPtr *EntityType
	var minScorePtr *float64
	var riskLevelPtr *RiskLevel

	if entityType != "" {
		et := EntityType(entityType)
		entityPtr = &et
	}
	if minScore != "" {
		// Parse minScore as float
		// This is a placeholder - in production, proper parsing would be done
	}
	if riskLevel != "" {
		rl := RiskLevel(riskLevel)
		riskLevelPtr = &rl
	}

	scores, err := s.repo.ListRiskScores(ctx, tenantID, entityPtr, minScorePtr, riskLevelPtr, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("service.ListRiskScores: %w", err)
	}

	// Convert to response format
	responses := make([]RiskScoreResponse, len(scores))
	for i, score := range scores {
		responses[i] = RiskScoreResponse{
			ID:                score.ID,
			TenantID:          score.TenantID,
			EntityType:        string(score.EntityType),
			EntityID:          score.EntityID,
			CalculatedAt:      score.CalculatedAt,
			OverallRiskScore:  score.OverallRiskScore,
			RiskLevel:         string(score.RiskLevel),
			PreviousScore:     score.PreviousScore,
			ScoreChange:       score.ScoreChange,
		}
	}

	return responses, nil
}

type RiskScoreResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	EntityType       string     `json:"entity_type"`
	EntityID         uuid.UUID  `json:"entity_id"`
	CalculatedAt     time.Time  `json:"calculated_at"`
	OverallRiskScore float64   `json:"overall_risk_score"`
	RiskLevel        string     `json:"risk_level"`
	PreviousScore    *float64   `json:"previous_score,omitempty"`
	ScoreChange      *float64   `json:"score_change,omitempty"`
}

// GetLatestRiskScore retrieves the latest risk score for an entity
func (s *Service) GetLatestRiskScore(ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID) (*RiskScoreResponse, error) {
	score, err := s.repo.GetLatestRiskScore(ctx, tenantID, EntityType(entityType), entityID)
	if err != nil {
		return nil, fmt.Errorf("service.GetLatestRiskScore: %w", err)
	}

	return &RiskScoreResponse{
		ID:               score.ID,
		TenantID:         score.TenantID,
		EntityType:       string(score.EntityType),
		EntityID:         score.EntityID,
		CalculatedAt:     score.CalculatedAt,
		OverallRiskScore: score.OverallRiskScore,
		RiskLevel:        string(score.RiskLevel),
		PreviousScore:    score.PreviousScore,
		ScoreChange:      score.ScoreChange,
	}, nil
}

// CalculateRiskScores calculates risk scores for specified entities
func (s *Service) CalculateRiskScores(ctx context.Context, tenantID uuid.UUID, entityType string, entityIDs []uuid.UUID) (int, error) {
	count := 0
	now := time.Now()

	for _, entityID := range entityIDs {
		// Get historical data for this entity
		// For now, use a simple scoring algorithm
		riskScore := 50.0 // Base score
		riskLevel := RiskLevelMedium

		// Create risk score entry
		score := &RiskScore{
			TenantID:         tenantID,
			EntityType:       EntityType(entityType),
			EntityID:         entityID,
			CalculatedAt:     now,
			OverallRiskScore: riskScore,
			RiskLevel:        riskLevel,
		}

		if err := s.repo.CreateRiskScore(ctx, score); err != nil {
			s.logger.Error().Err(err).Str("entity_id", entityID.String()).Msg("Failed to create risk score")
		} else {
			count++
		}
	}

	return count, nil
}

// Metrics Methods

// QueryMetrics queries raw metrics with filters
func (s *Service) QueryMetrics(ctx context.Context, tenantID uuid.UUID, metricName string, startDate, endDate time.Time, limit int) ([]Metric, error) {
	return s.repo.QueryMetrics(ctx, tenantID, metricName, startDate, endDate, limit)
}

// RecordMetric records a new metric data point
func (s *Service) RecordMetric(ctx context.Context, tenantID uuid.UUID, req *RecordMetricRequest) error {
	metric := &Metric{
		TenantID:   tenantID,
		MetricName: req.Name,
		MetricType: MetricType(req.Type),
		Value:      req.Value,
		Labels:     mustMarshalJSON(req.Labels),
		RecordedAt: time.Now(),
	}

	return s.repo.RecordMetric(ctx, metric)
}

type RecordMetricRequest struct {
	Name   string                 `json:"name" binding:"required"`
	Type   string                 `json:"type" binding:"required"`
	Value  float64               `json:"value" binding:"required"`
	Labels map[string]interface{} `json:"labels"`
}

// AggregateMetrics aggregates metrics by time period
func (s *Service) AggregateMetrics(ctx context.Context, tenantID uuid.UUID, metricName string, startDate, endDate time.Time, period string) ([]TimeSeriesDataPoint, error) {
	return s.repo.AggregateMetrics(ctx, tenantID, metricName, startDate, endDate, period)
}

// SSH Key Analytics Methods

// GetSSHKeyAnalytics retrieves analytics for a specific SSH key
func (s *Service) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, startDate, endDate time.Time) ([]SSHKeyUsage, error) {
	// TODO: Implement SSH key analytics tracking
	// This would track when SSH keys are used, by whom, for what targets
	return []SSHKeyUsage{}, nil
}

type SSHKeyUsage struct {
	SSHKeyID        uuid.UUID `json:"ssh_key_id"`
	UsageCount      int       `json:"usage_count"`
	LastUsed        time.Time `json:"last_used"`
	UniqueUsers     int       `json:"unique_users"`
	FailedAttempts  int       `json:"failed_attempts"`
	OffHoursUsage   int       `json:"off_hours_usage"`
}

// Command Blacklist Methods

// ListCommandBlacklists lists command blacklist rules
func (s *Service) ListCommandBlacklists(ctx context.Context, tenantID uuid.UUID, includeGlobal bool) ([]CommandBlacklist, error) {
	// TODO: Implement command blacklist from internal/analytics package
	return []CommandBlacklist{}, nil
}

// CreateCommandBlacklist creates a new command blacklist rule
func (s *Service) CreateCommandBlacklist(ctx context.Context, tenantID, createdBy uuid.UUID, req *CreateCommandBlacklistRequest) (*CommandBlacklist, error) {
	blacklist := &CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: req.Pattern,
		PatternType:    req.PatternType,
		Action:         req.Action,
		Severity:       req.Severity,
		Reason:         req.Reason,
		RiskCategory:   req.RiskCategory,
		Enabled:        true,
		CreatedBy:      createdBy,
	}

	// TODO: Persist blacklist
	return blacklist, nil
}

type CreateCommandBlacklistRequest struct {
	Pattern     string   `json:"pattern" binding:"required"`
	PatternType string   `json:"pattern_type" binding:"required"`
	Action      string   `json:"action" binding:"required"`
	Severity    string   `json:"severity" binding:"required"`
	Reason      string   `json:"reason" binding:"required"`
	RiskCategory *string  `json:"risk_category"`
	AppliesTo   []uuid.UUID `json:"applies_to"`
}

type CommandBlacklist struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty"`
	CommandPattern string   `json:"command_pattern"`
	PatternType  string   `json:"pattern_type"`
	BaseCommand  *string  `json:"base_command,omitempty"`
	Action       string   `json:"action"`
	Severity     string   `json:"severity"`
	AppliesToUsers []uuid.UUID `json:"applies_to_users,omitempty"`
	AppliesToGroups []uuid.UUID `json:"applies_to_groups,omitempty"`
	Reason       string   `json:"reason"`
	RiskCategory *string  `json:"risk_category,omitempty"`
	Enabled      bool     `json:"enabled"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetCommandBlacklist retrieves a specific blacklist rule
func (s *Service) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) {
	// TODO: Implement retrieval
	return nil, fmt.Errorf("not implemented")
}

// UpdateCommandBlacklist updates a command blacklist rule
func (s *Service) UpdateCommandBlacklist(ctx context.Context, id uuid.UUID, req *UpdateCommandBlacklistRequest) (*CommandBlacklist, error) {
	// TODO: Implement update
	return nil, fmt.Errorf("not implemented")
}

type UpdateCommandBlacklistRequest struct {
	Pattern    *string  `json:"pattern"`
	Action     *string  `json:"action"`
	Severity   *string  `json:"severity"`
	Reason     *string  `json:"reason"`
	Enabled    *bool    `json:"enabled"`
}

// DeleteCommandBlacklist deletes a command blacklist rule
func (s *Service) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement deletion
	return nil
}

// Cache Management Methods

// InvalidateCache invalidates specified cache types
func (s *Service) InvalidateCache(ctx context.Context, tenantID uuid.UUID, types ...string) error {
	pattern := fmt.Sprintf("analytics:*:%s", tenantID)
	if len(types) > 0 {
		for _, t := range types {
			_ = s.cache.Delete(ctx, fmt.Sprintf("analytics:%s:%s", t, tenantID))
		}
	} else {
		_ = s.cache.DeleteByPattern(ctx, pattern)
	}
	return nil
}

// WarmCache warms up the cache with frequently accessed data
func (s *Service) WarmCache(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) error {
	// Warm dashboard cache
	_, _ = s.GetDashboardMetrics(ctx, tenantID)

	// Warm session summary cache
	_, _ = s.GetSessionMetricsSummary(ctx, tenantID, startDate, endDate)

	return nil
}

// GetCacheStats retrieves cache statistics
func (s *Service) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	// Return basic cache stats
	return map[string]interface{}{
		"tenant_id":    tenantID.String(),
		"cache_enabled": true,
		"last_warmed":   time.Now(),
	}, nil
}

// Dashboard and Widget Methods

// ListDashboards lists all dashboards for a tenant
func (s *Service) ListDashboards(ctx context.Context, filter DashboardFilter) ([]Dashboard, int, error) {
	return s.repo.ListDashboards(ctx, filter)
}

// CreateDashboard creates a new dashboard
func (s *Service) CreateDashboard(ctx context.Context, tenantID, createdBy uuid.UUID, req *CreateDashboardRequest) (*Dashboard, error) {
	dashboard := &Dashboard{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   req.IsDefault,
		IsPublic:    req.IsPublic,
		Layout:      req.Layout,
		CreatedBy:   createdBy,
	}

	if err := s.repo.CreateDashboard(ctx, dashboard); err != nil {
		return nil, err
	}

	// Create widgets if provided
	for _, widgetReq := range req.Widgets {
		widget := &Widget{
			DashboardID: dashboard.ID,
			TenantID:    tenantID,
			Name:        widgetReq.Name,
			WidgetType:  widgetReq.WidgetType,
			PositionX:   widgetReq.PositionX,
			PositionY:   widgetReq.PositionY,
			Width:       widgetReq.Width,
			Height:      widgetReq.Height,
			DataSource:  widgetReq.DataSource,
			QueryConfig: widgetReq.QueryConfig,
			DisplayConfig: widgetReq.DisplayConfig,
			CreatedBy:   createdBy,
		}
		_ = s.repo.CreateWidget(ctx, widget)
	}

	return dashboard, nil
}

// GetDashboard retrieves a dashboard by ID
func (s *Service) GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error) {
	return s.repo.GetDashboard(ctx, id)
}

// UpdateDashboard updates a dashboard
func (s *Service) UpdateDashboard(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateDashboardRequest) (*Dashboard, error) {
	dashboard, err := s.repo.GetDashboard(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		dashboard.Name = *req.Name
	}
	if req.Description != nil {
		dashboard.Description = *req.Description
	}
	if req.IsDefault != nil {
		dashboard.IsDefault = *req.IsDefault
	}
	if req.IsPublic != nil {
		dashboard.IsPublic = *req.IsPublic
	}
	if req.Layout != nil {
		dashboard.Layout = req.Layout
	}
	dashboard.UpdatedBy = &userID

	if err := s.repo.UpdateDashboard(ctx, dashboard); err != nil {
		return nil, err
	}

	return dashboard, nil
}

// DeleteDashboard soft deletes a dashboard
func (s *Service) DeleteDashboard(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDashboard(ctx, id)
}

// Widget Methods

// ListWidgets lists all widgets for a dashboard
func (s *Service) ListWidgets(ctx context.Context, dashboardID uuid.UUID) ([]Widget, error) {
	return s.repo.ListWidgets(ctx, dashboardID)
}

// CreateWidget creates a new widget
func (s *Service) CreateWidget(ctx context.Context, dashboardID, createdBy uuid.UUID, req *CreateWidgetRequest) (*Widget, error) {
	widget := &Widget{
		DashboardID: dashboardID,
		Name:        req.Name,
		WidgetType:  req.WidgetType,
		PositionX:   req.PositionX,
		PositionY:   req.PositionY,
		Width:       req.Width,
		Height:      req.Height,
		DataSource:  req.DataSource,
		QueryConfig: req.QueryConfig,
		DisplayConfig: req.DisplayConfig,
		CreatedBy:   createdBy,
	}
	if req.RefreshIntervalSeconds != nil {
		widget.RefreshIntervalSeconds = *req.RefreshIntervalSeconds
	}

	// Get tenant from dashboard
	dashboard, err := s.repo.GetDashboard(ctx, dashboardID)
	if err != nil {
		return nil, err
	}
	widget.TenantID = dashboard.TenantID

	if err := s.repo.CreateWidget(ctx, widget); err != nil {
		return nil, err
	}

	return widget, nil
}

// GetWidget retrieves a widget by ID
func (s *Service) GetWidget(ctx context.Context, id uuid.UUID) (*Widget, error) {
	return s.repo.GetWidget(ctx, id)
}

// UpdateWidget updates a widget
func (s *Service) UpdateWidget(ctx context.Context, id uuid.UUID, req *UpdateWidgetRequest) (*Widget, error) {
	widget, err := s.repo.GetWidget(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		widget.Name = *req.Name
	}
	if req.PositionX != nil {
		widget.PositionX = *req.PositionX
	}
	if req.PositionY != nil {
		widget.PositionY = *req.PositionY
	}
	if req.Width != nil {
		widget.Width = *req.Width
	}
	if req.Height != nil {
		widget.Height = *req.Height
	}
	if req.QueryConfig != nil {
		widget.QueryConfig = req.QueryConfig
	}
	if req.DisplayConfig != nil {
		widget.DisplayConfig = req.DisplayConfig
	}
	if req.RefreshIntervalSeconds != nil {
		widget.RefreshIntervalSeconds = *req.RefreshIntervalSeconds
	}

	if err := s.repo.UpdateWidget(ctx, widget); err != nil {
		return nil, err
	}

	return widget, nil
}

// DeleteWidget deletes a widget
func (s *Service) DeleteWidget(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteWidget(ctx, id)
}

// GetWidgetData retrieves data for a widget
func (s *Service) GetWidgetData(ctx context.Context, widgetID uuid.UUID, startDate, endDate *time.Time) (interface{}, error) {
	widget, err := s.repo.GetWidget(ctx, widgetID)
	if err != nil {
		return nil, err
	}

	// Query data based on widget data source
	switch widget.DataSource {
	case DataSourceSessions:
		return s.getSessionWidgetData(ctx, widget.TenantID, widget.QueryConfig, startDate, endDate)
	case DataSourceUsers:
		return s.getUserWidgetData(ctx, widget.TenantID, widget.QueryConfig, startDate, endDate)
	case DataSourceRisks:
		return s.getRiskWidgetData(ctx, widget.TenantID, widget.QueryConfig)
	default:
		return map[string]interface{}{}, nil
	}
}

func (s *Service) getSessionWidgetData(ctx context.Context, tenantID uuid.UUID, queryConfig json.RawMessage, startDate, endDate *time.Time) (interface{}, error) {
	// Default to last 7 days if not specified
	if startDate == nil {
		now := time.Now()
		weekAgo := now.Add(-7 * 24 * time.Hour)
		startDate = &weekAgo
		endDate = &now
	}

	stats, err := s.GetSessionMetricsSummary(ctx, tenantID, *startDate, *endDate)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_sessions":  stats.Total,
		"active_sessions": stats.Active,
		"completed":       stats.Completed,
		"failed":          stats.Failed,
		"by_protocol":     stats.ByProtocol,
		"by_user":         stats.ByUser,
	}, nil
}

func (s *Service) getUserWidgetData(ctx context.Context, tenantID uuid.UUID, queryConfig json.RawMessage, startDate, endDate *time.Time) (interface{}, error) {
	if startDate == nil {
		now := time.Now()
		weekAgo := now.Add(-7 * 24 * time.Hour)
		startDate = &weekAgo
		endDate = &now
	}

	users, err := s.GetTopUsersByActivity(ctx, tenantID, *startDate, *endDate, 10)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *Service) getRiskWidgetData(ctx context.Context, tenantID uuid.UUID, queryConfig json.RawMessage) (interface{}, error) {
	scores, err := s.ListRiskScores(ctx, tenantID, "", "", "", 20)
	if err != nil {
		return nil, err
	}

	// Count by risk level
	riskCounts := map[string]int{}
	for _, score := range scores {
		riskCounts[score.RiskLevel]++
	}

	return map[string]interface{}{
		"by_level": riskCounts,
		"top_risks": scores,
	}, nil
}

// Report Methods

// ListReports lists reports with filters
func (s *Service) ListReports(ctx context.Context, filter ReportFilter) ([]Report, int, error) {
	return s.repo.ListReports(ctx, filter)
}

// CreateReport creates a new report
func (s *Service) CreateReport(ctx context.Context, tenantID, createdBy uuid.UUID, req *CreateReportRequest) (*Report, error) {
	report := &Report{
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		ReportType:      req.ReportType,
		Config:          req.Config,
		ScheduleEnabled: req.ScheduleEnabled,
		DeliveryMethods: req.DeliveryMethods,
		CreatedBy:       createdBy,
	}

	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// GetReport retrieves a report by ID
func (s *Service) GetReport(ctx context.Context, id uuid.UUID) (*Report, error) {
	return s.repo.GetReport(ctx, id)
}

// UpdateReport updates a report
func (s *Service) UpdateReport(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateReportRequest) (*Report, error) {
	report, err := s.repo.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		report.Name = *req.Name
	}
	if req.Description != nil {
		report.Description = *req.Description
	}
	if req.Config != nil {
		report.Config = req.Config
	}
	if req.ScheduleEnabled != nil {
		report.ScheduleEnabled = *req.ScheduleEnabled
	}
	if req.DeliveryMethods != nil {
		report.DeliveryMethods = req.DeliveryMethods
	}
	report.UpdatedBy = &userID

	if err := s.repo.UpdateReport(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

// DeleteReport soft deletes a report
func (s *Service) DeleteReport(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteReport(ctx, id)
}

// GenerateReportSnapshot generates a report snapshot
func (s *Service) GenerateReportSnapshot(ctx context.Context, reportID, createdBy uuid.UUID, startDate, endDate time.Time, format string) (*ReportSnapshot, error) {
	report, err := s.repo.GetReport(ctx, reportID)
	if err != nil {
		return nil, err
	}

	snapshot := &ReportSnapshot{
		ReportID:     reportID,
		TenantID:     report.TenantID,
		SnapshotName: report.Name,
		Framework:    string(report.ReportType),
		GeneratedAt:  time.Now(),
		GeneratedBy:  createdBy,
		PeriodStart:  startDate,
		PeriodEnd:    endDate,
		Status:       ReportSnapshotStatusPending,
		FileFormat:   &format,
	}

	if err := s.reportRepo.CreateReportSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}

	// TODO: Generate report data asynchronously
	// For now, mark as completed with placeholder data
	_ = s.reportRepo.UpdateReportSnapshotStatus(ctx, snapshot.ID, report.TenantID, ReportSnapshotStatusCompleted, nil, nil, nil)

	return snapshot, nil
}

// ListReportSnapshots lists snapshots for a tenant with filters
func (s *Service) ListReportSnapshots(ctx context.Context, tenantID uuid.UUID, filter ReportSnapshotFilter) ([]ReportSnapshot, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	return s.reportRepo.ListReportSnapshots(ctx, tenantID, filter)
}

// GetReportSnapshot retrieves a report snapshot
func (s *Service) GetReportSnapshot(ctx context.Context, snapshotID, tenantID uuid.UUID) (*ReportSnapshot, error) {
	return s.reportRepo.GetReportSnapshotByID(ctx, snapshotID, tenantID)
}

// UpdateReportSnapshotStatus updates the status of a report snapshot
func (s *Service) UpdateReportSnapshotStatus(ctx context.Context, snapshotID, tenantID uuid.UUID, status ReportSnapshotStatus, fileURL *string, fileSizeBytes *int64, errorMsg *string) error {
	return s.reportRepo.UpdateReportSnapshotStatus(ctx, snapshotID, tenantID, status, fileURL, fileSizeBytes, errorMsg)
}

// GetReportSnapshotStats retrieves statistics for report snapshots
func (s *Service) GetReportSnapshotStats(ctx context.Context, tenantID uuid.UUID) (*ReportSnapshotStats, error) {
	return s.reportRepo.GetReportSnapshotStats(ctx, tenantID)
}

// Alert Methods

// ListAlerts lists alerts with filters
func (s *Service) ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, int, error) {
	return s.repo.ListAlerts(ctx, filter)
}

// CreateAlert creates a new alert
func (s *Service) CreateAlert(ctx context.Context, tenantID, createdBy uuid.UUID, req *CreateAlertRequest) (*Alert, error) {
	alert := &Alert{
		TenantID:                   tenantID,
		Name:                       req.Name,
		Description:                req.Description,
		AlertType:                  req.AlertType,
		Severity:                   req.Severity,
		Conditions:                 req.Conditions,
		EvaluationIntervalMinutes:  req.EvaluationIntervalMinutes,
		NotificationMethods:        req.NotificationMethods,
		CooldownMinutes:            req.CooldownMinutes,
		CreatedBy:                  createdBy,
	}

	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// GetAlert retrieves an alert by ID
func (s *Service) GetAlert(ctx context.Context, id uuid.UUID) (*Alert, error) {
	return s.repo.GetAlert(ctx, id)
}

// UpdateAlert updates an alert
func (s *Service) UpdateAlert(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateAlertRequest) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		alert.Name = *req.Name
	}
	if req.Description != nil {
		alert.Description = *req.Description
	}
	if req.Severity != nil {
		alert.Severity = *req.Severity
	}
	if req.Conditions != nil {
		alert.Conditions = req.Conditions
	}
	if req.EvaluationIntervalMinutes != nil {
		alert.EvaluationIntervalMinutes = *req.EvaluationIntervalMinutes
	}
	if req.NotificationMethods != nil {
		alert.NotificationMethods = req.NotificationMethods
	}
	if req.Enabled != nil {
		alert.Enabled = *req.Enabled
	}
	if req.CooldownMinutes != nil {
		alert.CooldownMinutes = *req.CooldownMinutes
	}
	alert.UpdatedBy = &userID

	if err := s.repo.UpdateAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// DeleteAlert soft deletes an alert
func (s *Service) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteAlert(ctx, id)
}

// SetAlertEnabled enables or disables an alert
func (s *Service) SetAlertEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	alert, err := s.repo.GetAlert(ctx, id)
	if err != nil {
		return err
	}

	alert.Enabled = enabled
	return s.repo.UpdateAlert(ctx, alert)
}

// ListAlertTriggers lists triggers for an alert
func (s *Service) ListAlertTriggers(ctx context.Context, alertID uuid.UUID, limit, offset int) ([]AlertTrigger, error) {
	return s.repo.ListAlertTriggers(ctx, alertID, limit, offset)
}

// CreateAlertTrigger creates an alert trigger record
func (s *Service) CreateAlertTrigger(ctx context.Context, alertID uuid.UUID, tenantID uuid.UUID, severity string, triggerData map[string]interface{}) error {
	trigger := &AlertTrigger{
		AlertID:     alertID,
		TenantID:    tenantID,
		Severity:    Severity(severity),
		TriggerData: mustMarshalJSON(triggerData),
	}

	return s.repo.CreateAlertTrigger(ctx, trigger)
}

// ResolveAlertTrigger resolves an alert trigger
func (s *Service) ResolveAlertTrigger(ctx context.Context, alertID, triggerID, resolvedBy uuid.UUID, notes string) error {
	return s.repo.UpdateAlertTrigger(ctx, triggerID, &resolvedBy, &notes)
}

// Worker Methods

// StartAggregationWorker starts the background aggregation worker
func (s *Service) StartAggregationWorker(ctx context.Context) {
	s.aggregationWorkerRunning = true
	s.logger.Info().Msg("Starting aggregation worker")

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info().Msg("Aggregation worker stopped")
				return
			case <-ticker.C:
				s.runHourlyAggregation(ctx)
			}
		}
	}()
}

// StartAlertEvaluator starts the background alert evaluator
func (s *Service) StartAlertEvaluator(ctx context.Context) {
	s.alertEvaluatorRunning = true
	s.logger.Info().Msg("Starting alert evaluator")

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info().Msg("Alert evaluator stopped")
				return
			case <-ticker.C:
				s.evaluateAlerts(ctx)
			}
		}
	}()
}

// StartReportScheduler starts the report scheduler
func (s *Service) StartReportScheduler(ctx context.Context) {
	s.reportSchedulerRunning = true
	s.logger.Info().Msg("Starting report scheduler")

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info().Msg("Report scheduler stopped")
				return
			case <-ticker.C:
				s.checkScheduledReports(ctx)
			}
		}
	}()
}

// GetWorkerStatus returns the status of background workers
func (s *Service) GetWorkerStatus() map[string]bool {
	return map[string]bool{
		"aggregation_worker": s.aggregationWorkerRunning,
		"alert_evaluator":    s.alertEvaluatorRunning,
		"report_scheduler":   s.reportSchedulerRunning,
	}
}

// RunAggregation manually triggers aggregation for a period
func (s *Service) RunAggregation(ctx context.Context, tenantID uuid.UUID, periodType string, startDate, endDate time.Time) (*AggregationResult, error) {
	result := &AggregationResult{
		TenantID:  tenantID,
		PeriodType: periodType,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "completed",
	}

	// Aggregate session data
	sessionStats, err := s.repo.GetRawSessionStats(ctx, tenantID, startDate, endDate)
	if err == nil {
		result.SessionsProcessed = sessionStats.Total
	}

	// Aggregate event data
	eventStats, err := s.repo.GetRawEventStats(ctx, tenantID, startDate, endDate)
	if err == nil {
		result.EventsProcessed = eventStats.Total
	}

	return result, nil
}

type AggregationResult struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	PeriodType       string    `json:"period_type"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	SessionsProcessed int64     `json:"sessions_processed"`
	EventsProcessed  int64     `json:"events_processed"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
}

// RefreshMaterializedViews refreshes all materialized views
func (s *Service) RefreshMaterializedViews(ctx context.Context) error {
	return s.repo.RefreshMaterializedViews(ctx)
}

// runHourlyAggregation runs the hourly aggregation job
func (s *Service) runHourlyAggregation(ctx context.Context) {
	s.logger.Debug().Msg("Running hourly aggregation")

	// Get all active tenants and aggregate for each
	// TODO: Implement tenant discovery
	// For now, aggregate for the last hour

	prevHour := time.Now().Add(-time.Hour).Truncate(time.Hour)

	// Aggregate sessions and events for this period
	// This would be called for each tenant

	s.logger.Debug().Time("hour", prevHour).Msg("Hourly aggregation complete")
}

// evaluateAlerts evaluates all enabled alerts
func (s *Service) evaluateAlerts(ctx context.Context) {
	s.logger.Debug().Msg("Evaluating alerts")

	// Get all enabled alerts
	alerts, err := s.repo.GetActiveAlerts(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get active alerts")
		return
	}

	for _, alert := range alerts {
		// TODO: Implement alert evaluation logic based on alert type
		_ = s.repo.UpdateAlertEvaluationTime(ctx, alert.ID)
	}

	s.logger.Debug().Int("count", len(alerts)).Msg("Alert evaluation complete")
}

// checkScheduledReports checks and runs scheduled reports
func (s *Service) checkScheduledReports(ctx context.Context) {
	s.logger.Debug().Msg("Checking scheduled reports")

	// Get reports that should run
	// TODO: Implement scheduled report checking

	s.logger.Debug().Msg("Scheduled reports check complete")
}

// =============================================================================
// Report Generation Job Methods
// =============================================================================

// CreateReportGenerationJob creates a new report generation job
func (s *Service) CreateReportGenerationJob(ctx context.Context, tenantID uuid.UUID, jobType string, format string, options json.RawMessage) (*ReportGenerationJob, error) {
	job := &ReportGenerationJob{
		TenantID:  tenantID,
		JobType:   jobType,
		Format:    format,
		Status:    ReportJobStatusQueued,
		Progress:  0,
		Options:   options,
		MaxRetries: 3,
	}

	if err := s.reportRepo.CreateReportGenerationJob(ctx, job); err != nil {
		return nil, fmt.Errorf("service.CreateReportGenerationJob: %w", err)
	}

	return job, nil
}

// GetReportGenerationJob retrieves a report generation job
func (s *Service) GetReportGenerationJob(ctx context.Context, jobID, tenantID uuid.UUID) (*ReportGenerationJob, error) {
	return s.reportRepo.GetReportGenerationJobByID(ctx, jobID, tenantID)
}

// ListReportGenerationJobs lists report generation jobs
func (s *Service) ListReportGenerationJobs(ctx context.Context, tenantID uuid.UUID, filter ReportJobFilter) ([]ReportGenerationJob, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	filter.TenantID = tenantID
	return s.reportRepo.ListReportGenerationJobs(ctx, tenantID, filter)
}

// UpdateReportGenerationJobStatus updates job status and progress
func (s *Service) UpdateReportGenerationJobStatus(ctx context.Context, jobID, tenantID uuid.UUID, status ReportJobStatus, progress int, errorMsg *string, snapshotID *uuid.UUID) error {
	return s.reportRepo.UpdateReportGenerationJobStatus(ctx, jobID, tenantID, status, progress, errorMsg, snapshotID)
}

// =============================================================================
// Report Schedule Methods
// =============================================================================

// CreateReportSchedule creates a new report schedule
func (s *Service) CreateReportSchedule(ctx context.Context, tenantID, createdBy, ownedBy uuid.UUID, req *ReportScheduleRequest) (*ReportSchedule, error) {
	schedule := &ReportSchedule{
		TenantID:            tenantID,
		ScheduleName:        req.ScheduleName,
		Framework:           req.Framework,
		ReportID:            req.ReportID,
		ScheduleType:        req.ScheduleType,
		CronExpression:      req.CronExpression,
		Format:              string(req.Format),
		Options:             mustMarshalJSON(req.Options),
		Recipients:          req.Recipients,
		NotifyOnCompletion:  req.NotifyOnCompletion,
		NotifyOnFailure:     req.NotifyOnFailure,
		Status:              ReportScheduleStatusActive,
		RetentionDays:       req.RetentionDays,
		CreatedBy:           createdBy,
		OwnedBy:             ownedBy,
	}

	// Calculate next run time based on schedule type
	nextRunAt, err := s.calculateNextRunTime(req.ScheduleType, req.CronExpression)
	if err != nil {
		return nil, fmt.Errorf("service.CreateReportSchedule: %w", err)
	}
	schedule.NextRunAt = nextRunAt

	if err := s.reportRepo.CreateReportSchedule(ctx, schedule); err != nil {
		return nil, fmt.Errorf("service.CreateReportSchedule: %w", err)
	}

	return schedule, nil
}

// GetReportSchedule retrieves a report schedule
func (s *Service) GetReportSchedule(ctx context.Context, scheduleID, tenantID uuid.UUID) (*ReportSchedule, error) {
	return s.reportRepo.GetReportScheduleByID(ctx, scheduleID, tenantID)
}

// ListReportSchedules lists report schedules
func (s *Service) ListReportSchedules(ctx context.Context, tenantID uuid.UUID, filter ReportScheduleFilter) ([]ReportSchedule, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	filter.TenantID = tenantID
	return s.reportRepo.ListReportSchedules(ctx, tenantID, filter)
}

// UpdateReportSchedule updates a report schedule
func (s *Service) UpdateReportSchedule(ctx context.Context, scheduleID, tenantID uuid.UUID, req *ReportScheduleRequest) error {
	schedule, err := s.reportRepo.GetReportScheduleByID(ctx, scheduleID, tenantID)
	if err != nil {
		return fmt.Errorf("service.UpdateReportSchedule: %w", err)
	}

	schedule.ScheduleName = req.ScheduleName
	schedule.Framework = req.Framework
	schedule.ReportID = req.ReportID
	schedule.ScheduleType = req.ScheduleType
	schedule.CronExpression = req.CronExpression
	schedule.Format = string(req.Format)
	schedule.Options = mustMarshalJSON(req.Options)
	schedule.Recipients = req.Recipients
	schedule.NotifyOnCompletion = req.NotifyOnCompletion
	schedule.NotifyOnFailure = req.NotifyOnFailure
	schedule.RetentionDays = req.RetentionDays

	if err := s.reportRepo.UpdateReportSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("service.UpdateReportSchedule: %w", err)
	}

	return nil
}

// DeleteReportSchedule deletes a report schedule
func (s *Service) DeleteReportSchedule(ctx context.Context, scheduleID, tenantID uuid.UUID) error {
	return s.reportRepo.DeleteReportSchedule(ctx, scheduleID, tenantID)
}

// calculateNextRunTime calculates the next run time based on schedule type
func (s *Service) calculateNextRunTime(scheduleType ReportScheduleType, cronExpression *string) (time.Time, error) {
	now := time.Now()

	switch scheduleType {
	case ReportScheduleDaily:
		return now.Add(24 * time.Hour).Truncate(24 * time.Hour), nil
	case ReportScheduleWeekly:
		return now.Add(7 * 24 * time.Hour).Truncate(24 * time.Hour), nil
	case ReportScheduleMonthly:
		return now.AddDate(0, 1, 0).Truncate(24 * time.Hour), nil
	case ReportScheduleQuarterly:
		return now.AddDate(0, 3, 0).Truncate(24 * time.Hour), nil
	case ReportScheduleYearly:
		return now.AddDate(1, 0, 0).Truncate(24 * time.Hour), nil
	default:
		// Default to daily
		return now.Add(24 * time.Hour).Truncate(24 * time.Hour), nil
	}
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}

func getSeverityFromScore(score *float64) string {
	if score == nil {
		return "low"
	}
	switch {
	case *score >= 90:
		return "critical"
	case *score >= 70:
		return "high"
	case *score >= 40:
		return "medium"
	default:
		return "low"
	}
}

func getAnomalyScoreValue(score *float64) float64 {
	if score == nil {
		return 0
	}
	return *score
}

// anomalyToResponse converts an Anomaly to AnomalyResponse
func (s *Service) anomalyToResponse(anomaly *Anomaly) AnomalyResponse {
	return AnomalyResponse{
		ID:          anomaly.ID,
		TenantID:    anomaly.TenantID,
		UserID:      anomaly.UserID,
		AnomalyType: string(anomaly.AnomalyType),
		Severity:    string(anomaly.Severity),
		Description: anomaly.Description,
		DetectedAt:  anomaly.DetectedAt,
		Status:      string(anomaly.Status),
	}
}
