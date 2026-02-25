package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics/repository"
	"github.com/rs/zerolog"
)

// ReportService provides report persistence and generation operations
type ReportService struct {
	reportRepo    *repository.ReportRepository
	exceptionRepo *repository.ExceptionRepository
	cache         Cache
	logger        zerolog.Logger
}

// Cache interface for report caching
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewReportService creates a new report service
func NewReportService(
	reportRepo *repository.ReportRepository,
	exceptionRepo *repository.ExceptionRepository,
	cache Cache,
	logger zerolog.Logger,
) *ReportService {
	return &ReportService{
		reportRepo:    reportRepo,
		exceptionRepo: exceptionRepo,
		cache:         cache,
		logger:        logger,
	}
}

// =============================================================================
// Report Snapshot Operations
// =============================================================================

// GenerateComplianceReport generates a new compliance report and persists it
func (s *ReportService) GenerateComplianceReport(
	ctx context.Context,
	tenantID, generatedBy uuid.UUID,
	framework string,
	startDate, endDate time.Time,
) (*ComplianceReportResponse, error) {
	// Calculate compliance status
	status, err := s.calculateComplianceStatus(ctx, tenantID, framework, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("report.GenerateComplianceReport: %w", err)
	}

	report := &ComplianceReportResponse{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Framework:       framework,
		GeneratedAt:     time.Now(),
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

	// Cache the report
	cacheKey := fmt.Sprintf("report:compliance:%s:%s:%s:%s",
		tenantID, framework, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	_ = s.cache.Set(ctx, cacheKey, report, 15*time.Minute)

	return report, nil
}

// calculateComplianceStatus calculates compliance status for a framework and period
func (s *ReportService) calculateComplianceStatus(
	ctx context.Context,
	tenantID uuid.UUID,
	framework string,
	startDate, endDate time.Time,
) (*ComplianceStatus, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("compliance:status:%s:%s:%s:%s",
		tenantID, framework, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	var status ComplianceStatus
	if err := s.cache.Get(ctx, cacheKey, &status); err == nil {
		return &status, nil
	}

	// Calculate compliance based on framework
	// This is a simplified implementation - in production would query actual data
	status = ComplianceStatus{
		OverallPercentage: 85.0,
		PassedChecks:      17,
		TotalChecks:       20,
		Violations: []ComplianceViolation{
			{
				PolicyID:      uuid.New(),
				PolicyName:    "Access Review",
				ViolationType: "missing_review",
				Severity:      "medium",
				Description:   " Quarterly access review not completed",
				Count:         5,
				FirstSeen:     startDate,
				LastSeen:      endDate,
			},
		},
		ByPolicy: map[string]CompliancePolicyStatus{
			"access_control": {
				PolicyID:          uuid.New(),
				PolicyName:        "Access Control",
				ComplianceRate:    90.0,
				TotalEvaluations:  10,
				PassedEvaluations: 9,
			},
		},
	}

	// Cache for 10 minutes
	_ = s.cache.Set(ctx, cacheKey, status, 10*time.Minute)

	return &status, nil
}

// GetComplianceReport retrieves a specific compliance report by ID
func (s *ReportService) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReportResponse, error) {
	snapshot, err := s.reportRepo.GetReportSnapshot(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("report.GetComplianceReport: %w", err)
	}

	return s.snapshotToResponse(snapshot), nil
}

// ListComplianceReports lists compliance reports with filters
func (s *ReportService) ListComplianceReports(
	ctx context.Context,
	tenantID uuid.UUID,
	framework, status string,
	startDate, endDate *time.Time,
	limit, offset int,
) ([]ComplianceReportResponse, int, error) {
	snapshots, total, err := s.reportRepo.ListReportSnapshots(
		ctx, tenantID, framework, status, startDate, endDate, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListComplianceReports: %w", err)
	}

	responses := make([]ComplianceReportResponse, len(snapshots))
	for i, snapshot := range snapshots {
		responses[i] = *s.snapshotToResponse(&snapshot)
	}

	return responses, total, nil
}

// GenerateReportSnapshot generates and persists a report snapshot
func (s *ReportService) GenerateReportSnapshot(
	ctx context.Context,
	tenantID, generatedBy uuid.UUID,
	reportID uuid.UUID,
	framework string,
	startDate, endDate time.Time,
	format ReportFormat,
) (*ReportSnapshot, error) {
	// Calculate compliance status first
	status, err := s.calculateComplianceStatus(ctx, tenantID, framework, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Create snapshot
	snapshot := &ReportSnapshot{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReportID:     reportID,
		SnapshotName: fmt.Sprintf("%s Report - %s to %s", framework,
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		Framework:    framework,
		GeneratedAt:  time.Now(),
		GeneratedBy:  generatedBy,
		Status:       ReportSnapshotStatusPending,
		PeriodStart:  startDate,
		PeriodEnd:    endDate,
	}

	// Marshal report data
	reportData := map[string]interface{}{
		"overall_score":   status.OverallPercentage,
		"passed_controls": status.PassedChecks,
		"failed_controls": status.TotalChecks - status.PassedChecks,
		"violations":      status.Violations,
		"by_policy":       status.ByPolicy,
	}
	snapshot.Data = mustMarshalJSON(reportData)

	// Create summary
	summary := fmt.Sprintf("Compliance Score: %.1f%%, %d/%d controls passed",
		status.OverallPercentage, status.PassedChecks, status.TotalChecks)
	snapshot.Summary = &summary

	// Set format
	formatStr := string(format)
	snapshot.FileFormat = &formatStr

	// Set expiration (default 90 days)
	expiresAt := time.Now().AddDate(0, 0, 90)
	snapshot.ExpiresAt = &expiresAt

	// Persist snapshot
	if err := s.reportRepo.CreateReportSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("report.GenerateReportSnapshot: %w", err)
	}

	// Queue async generation job
	job := &ReportGenerationJob{
		TenantID:  tenantID,
		JobType:   "compliance",
		ReportID:  &reportID,
		Status:    ReportJobStatusQueued,
		Format:    string(format),
		Progress:  0,
		MaxRetries: 3,
	}

	if err := s.reportRepo.CreateReportJob(ctx, job); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create report generation job")
	}

	s.logger.Info().
		Str("snapshot_id", snapshot.ID.String()).
		Str("tenant_id", tenantID.String()).
		Str("framework", framework).
		Msg("Report snapshot generated")

	return snapshot, nil
}

// GetReportSnapshot retrieves a report snapshot
func (s *ReportService) GetReportSnapshot(ctx context.Context, id uuid.UUID) (*ReportSnapshot, error) {
	snapshot, err := s.reportRepo.GetReportSnapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// DeleteReportSnapshot deletes a report snapshot
func (s *ReportService) DeleteReportSnapshot(ctx context.Context, id uuid.UUID) error {
	return s.reportRepo.DeleteReportSnapshot(ctx, id)
}

// GetReportSnapshotStats retrieves statistics about report snapshots
func (s *ReportService) GetReportSnapshotStats(ctx context.Context, tenantID uuid.UUID) (*ReportSnapshotStats, error) {
	return s.reportRepo.GetReportSnapshotStats(ctx, tenantID)
}

// =============================================================================
// Report Generation Job Operations
// =============================================================================

// GetReportJob retrieves a report generation job
func (s *ReportService) GetReportJob(ctx context.Context, id uuid.UUID) (*ReportGenerationJob, error) {
	return s.reportRepo.GetReportJob(ctx, id)
}

// ListReportJobs lists report generation jobs
func (s *ReportService) ListReportJobs(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]ReportGenerationJob, int, error) {
	return s.reportRepo.ListReportJobs(ctx, tenantID, status, limit, offset)
}

// DeleteReportJob deletes a report generation job
func (s *ReportService) DeleteReportJob(ctx context.Context, id uuid.UUID) error {
	return s.reportRepo.DeleteReportJob(ctx, id)
}

// UpdateReportJobProgress updates job progress
func (s *ReportService) UpdateReportJobProgress(ctx context.Context, id uuid.UUID, progress int) error {
	return s.reportRepo.UpdateReportJobProgress(ctx, id, progress)
}

// =============================================================================
// Report Schedule Operations
// =============================================================================

// CreateReportSchedule creates a new report schedule
func (s *ReportService) CreateReportSchedule(
	ctx context.Context,
	tenantID, createdBy uuid.UUID,
	req *ReportScheduleRequest,
) (*ReportSchedule, error) {
	schedule := &ReportSchedule{
		TenantID:            tenantID,
		ScheduleName:        req.ScheduleName,
		Framework:           req.Framework,
		ReportID:            req.ReportID,
		ScheduleType:        req.ScheduleType,
		CronExpression:      req.CronExpression,
		Format:              string(req.Format),
		Recipients:          req.Recipients,
		NotifyOnCompletion:  req.NotifyOnCompletion,
		NotifyOnFailure:     req.NotifyOnFailure,
		Status:              ReportScheduleStatusActive,
		CreatedBy:            createdBy,
		OwnedBy:             createdBy,
		RetentionDays:       req.RetentionDays,
	}

	// Calculate next run time
	nextRunAt, err := s.calculateNextRunTime(req.ScheduleType, req.CronExpression)
	if err != nil {
		return nil, fmt.Errorf("report.CreateReportSchedule: %w", err)
	}
	schedule.NextRunAt = nextRunAt

	// Marshal options
	if req.Options != nil {
		schedule.Options = mustMarshalJSON(req.Options)
	}

	if err := s.reportRepo.CreateReportSchedule(ctx, schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// GetReportSchedule retrieves a report schedule
func (s *ReportService) GetReportSchedule(ctx context.Context, id uuid.UUID) (*ReportSchedule, error) {
	return s.reportRepo.GetReportSchedule(ctx, id)
}

// ListReportSchedules lists report schedules
func (s *ReportService) ListReportSchedules(
	ctx context.Context,
	tenantID uuid.UUID,
	status string,
	framework string,
	limit, offset int,
) ([]ReportSchedule, int, error) {
	return s.reportRepo.ListReportSchedules(ctx, tenantID, status, framework, limit, offset)
}

// UpdateReportSchedule updates a report schedule
func (s *ReportService) UpdateReportSchedule(
	ctx context.Context,
	id uuid.UUID,
	req *ReportScheduleRequest,
) error {
	schedule, err := s.reportRepo.GetReportSchedule(ctx, id)
	if err != nil {
		return err
	}

	schedule.ScheduleName = req.ScheduleName
	schedule.ScheduleType = req.ScheduleType
	schedule.CronExpression = req.CronExpression
	schedule.Format = string(req.Format)
	schedule.Recipients = req.Recipients
	schedule.NotifyOnCompletion = req.NotifyOnCompletion
	schedule.NotifyOnFailure = req.NotifyOnFailure
	schedule.RetentionDays = req.RetentionDays

	if req.Options != nil {
		schedule.Options = mustMarshalJSON(req.Options)
	}

	return s.reportRepo.UpdateReportSchedule(ctx, schedule)
}

// DeleteReportSchedule deletes a report schedule
func (s *ReportService) DeleteReportSchedule(ctx context.Context, id uuid.UUID) error {
	return s.reportRepo.DeleteReportSchedule(ctx, id)
}

// =============================================================================
// Compliance Exception Operations
// =============================================================================

// CreateComplianceException creates a new compliance exception
func (s *ReportService) CreateComplianceException(
	ctx context.Context,
	tenantID, requestedBy uuid.UUID,
	req *CreateComplianceExceptionRequest,
) (*ComplianceException, error) {
	exception := &ComplianceException{
		TenantID:            tenantID,
		ControlID:           req.ControlID,
		ControlName:         req.ControlName,
		Framework:           req.Framework,
		RiskLevel:           req.RiskLevel,
		RequestedBy:         requestedBy,
		Justification:       req.Justification,
		BusinessReason:      req.BusinessReason,
		CompensatingControls: req.CompensatingControls,
		ExpiresAt:           req.ExpiresAt,
	}

	if req.Metadata != nil {
		exception.Metadata = mustMarshalJSON(req.Metadata)
	}

	if err := s.exceptionRepo.CreateComplianceException(ctx, exception); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("exception_id", exception.ID.String()).
		Str("tenant_id", tenantID.String()).
		Str("control_id", req.ControlID).
		Str("framework", req.Framework).
		Msg("Compliance exception created")

	return exception, nil
}

// GetComplianceException retrieves a compliance exception
func (s *ReportService) GetComplianceException(ctx context.Context, id uuid.UUID) (*ComplianceException, error) {
	return s.exceptionRepo.GetComplianceException(ctx, id)
}

// ListComplianceExceptions lists compliance exceptions
func (s *ReportService) ListComplianceExceptions(
	ctx context.Context,
	tenantID uuid.UUID,
	status *ExceptionStatus,
	framework, controlID string,
	limit, offset int,
) ([]ComplianceException, int, error) {
	return s.exceptionRepo.ListComplianceExceptions(ctx, tenantID, status, framework, controlID, limit, offset)
}

// UpdateComplianceException updates a compliance exception
func (s *ReportService) UpdateComplianceException(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateComplianceExceptionRequest,
) error {
	exception, err := s.exceptionRepo.GetComplianceException(ctx, id)
	if err != nil {
		return err
	}

	if req.Justification != nil {
		exception.Justification = *req.Justification
	}
	if req.BusinessReason != nil {
		exception.BusinessReason = *req.BusinessReason
	}
	if req.CompensatingControls != nil {
		exception.CompensatingControls = *req.CompensatingControls
	}
	if req.ReviewDate != nil {
		exception.ReviewDate = req.ReviewDate
	}
	if req.ReviewNotes != nil {
		exception.ReviewNotes = req.ReviewNotes
	}

	return s.exceptionRepo.UpdateException(ctx, exception)
}

// ApproveComplianceException approves a compliance exception
func (s *ReportService) ApproveComplianceException(
	ctx context.Context,
	id, approvedBy uuid.UUID,
	expiresAt *time.Time,
	notes string,
) error {
	return s.exceptionRepo.UpdateExceptionStatus(ctx, id, ExceptionStatusApproved, &approvedBy, expiresAt, &approvedBy, &notes)
}

// DenyComplianceException denies a compliance exception
func (s *ReportService) DenyComplianceException(
	ctx context.Context,
	id, deniedBy uuid.UUID,
	notes string,
) error {
	return s.exceptionRepo.UpdateExceptionStatus(ctx, id, ExceptionStatusDenied, &deniedBy, nil, nil, &notes)
}

// DeleteComplianceException deletes a compliance exception
func (s *ReportService) DeleteComplianceException(ctx context.Context, id uuid.UUID) error {
	return s.exceptionRepo.DeleteComplianceException(ctx, id)
}

// GetExpiringExceptions retrieves exceptions expiring soon
func (s *ReportService) GetExpiringExceptions(ctx context.Context, tenantID uuid.UUID, days int) ([]ComplianceException, error) {
	return s.exceptionRepo.GetExpiringExceptions(ctx, tenantID, days)
}

// GetExceptionStats retrieves exception statistics
func (s *ReportService) GetExceptionStats(ctx context.Context, tenantID uuid.UUID) (*ExceptionStats, error) {
	return s.exceptionRepo.GetExceptionStats(ctx, tenantID)
}

// =============================================================================
// Helper Methods
// =============================================================================

// snapshotToResponse converts a snapshot to a compliance report response
func (s *ReportService) snapshotToResponse(snapshot *ReportSnapshot) *ComplianceReportResponse {
	response := &ComplianceReportResponse{
		ID:          snapshot.ID,
		TenantID:    snapshot.TenantID,
		Framework:   snapshot.Framework,
		GeneratedAt: snapshot.GeneratedAt,
		GeneratedBy: snapshot.GeneratedBy,
		PeriodStart: snapshot.PeriodStart,
		PeriodEnd:   snapshot.PeriodEnd,
		Status:      string(snapshot.Status),
	}

	// Unmarshal data if available
	if len(snapshot.Data) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(snapshot.Data, &data); err == nil {
			if score, ok := data["overall_score"].(float64); ok {
				response.OverallScore = score
			}
			if passed, ok := data["passed_controls"].(float64); ok {
				response.PassedControls = int(passed)
			}
			if total, ok := data["total_controls"].(float64); ok {
				response.FailedControls = int(total) - response.PassedControls
			}
			if violations, ok := data["violations"].([]interface{}); ok {
				// Convert violations
				vBytes, _ := json.Marshal(violations)
				json.Unmarshal(vBytes, &response.Violations)
			}
			if byPolicy, ok := data["by_policy"].(map[string]interface{}); ok {
				bpBytes, _ := json.Marshal(byPolicy)
				json.Unmarshal(bpBytes, &response.ByPolicy)
			}
		}
	}

	return response
}

// calculateNextRunTime calculates the next run time based on schedule type
func (s *ReportService) calculateNextRunTime(scheduleType ReportScheduleType, cronExpr *string) (time.Time, error) {
	now := time.Now()

	// If custom cron expression provided, use it
	if cronExpr != nil && scheduleType == ReportScheduleCustom {
		// In production, would parse cron expression
		// For now, return tomorrow at same time
		return now.Add(24 * time.Hour), nil
	}

	switch scheduleType {
	case ReportScheduleDaily:
		// Next run at midnight tomorrow
		next := now.Add(24 * time.Hour)
		return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location()), nil

	case ReportScheduleWeekly:
		// Next run at next Sunday midnight
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		next := now.AddDate(0, 0, daysUntilSunday)
		return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location()), nil

	case ReportScheduleMonthly:
		// Next run at first of next month midnight
		next := now.AddDate(0, 1, 0)
		return time.Date(next.Year(), next.Month(), 1, 0, 0, 0, 0, next.Location()), nil

	case ReportScheduleQuarterly:
		// Next run at first of next quarter
		nextMonth := now.Month() + 1
		if nextMonth > 12 {
			nextMonth = 1
		}
		quarterStartMonth := ((nextMonth-1)/3)*3 + 1
		if quarterStartMonth <= nextMonth {
			// Move to next quarter
			quarterStartMonth += 3
			if quarterStartMonth > 12 {
				quarterStartMonth = 1
			}
		}
		next := time.Date(now.Year(), quarterStartMonth, 1, 0, 0, 0, 0, now.Location())
		return next, nil

	case ReportScheduleYearly:
		// Next run at January 1 of next year
		next := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
		return next, nil

	default:
		return now.Add(24 * time.Hour), nil
	}
}

// mustMarshalJSON marshals data to JSON, panicking on error
func mustMarshalJSON(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return bytes
}

// timePtr returns a pointer to a time value
func timePtr(t time.Time) *time.Time {
	return &t
}
