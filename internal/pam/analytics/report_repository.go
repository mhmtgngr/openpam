package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// ReportRepository handles report persistence operations
type ReportRepository struct {
	Db     *sqlx.DB
	logger zerolog.Logger
}

// NewReportRepository creates a new report repository
func NewReportRepository(db *sqlx.DB, logger zerolog.Logger) *ReportRepository {
	return &ReportRepository{Db: db, logger: logger}
}

// setTenantContext sets the PostgreSQL tenant context for RLS
// This must be called before any query that uses RLS policies
func (r *ReportRepository) setTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	_, err := r.Db.ExecContext(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// getTenantIDFromContext extracts tenant_id from Go context
func getTenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	tenantIDVal := ctx.Value("tenant_id")
	if tenantIDVal == nil {
		return uuid.Nil, fmt.Errorf("tenant_id not found in context")
	}

	tenantIDStr, ok := tenantIDVal.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("tenant_id is not a string")
	}

	return uuid.Parse(tenantIDStr)
}

// =============================================================================
// ComplianceReport Operations
// =============================================================================

// CreateComplianceReport creates a new compliance report
func (r *ReportRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, report.TenantID); err != nil {
		return err
	}

	report.ID = uuid.New()
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	query := `
		INSERT INTO compliance_reports (
			id, tenant_id, framework, status, generated_at,
			period_start, period_end, expires_at,
			overall_score, passed_controls, failed_controls,
			data, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :framework, :status, :generated_at,
			:period_start, :period_end, :expires_at,
			:overall_score, :passed_controls, :failed_controls,
			:data, :metadata, :created_at, :updated_at
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("report.CreateComplianceReport: %w", err)
	}
	return nil
}

// GetComplianceReportByID retrieves a compliance report by ID
func (r *ReportRepository) GetComplianceReportByID(ctx context.Context, id, tenantID uuid.UUID) (*ComplianceReport, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var report ComplianceReport
	query := `SELECT * FROM compliance_reports WHERE id = $1 AND tenant_id = $2`
	err := r.Db.GetContext(ctx, &report, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetComplianceReportByID: %w", err)
	}
	return &report, nil
}

// ListComplianceReports lists compliance reports for a tenant with filters
func (r *ReportRepository) ListComplianceReports(ctx context.Context, tenantID uuid.UUID, filter ComplianceReportFilter) ([]ComplianceReport, int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, 0, err
	}

	var reports []ComplianceReport

	// Build where clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if filter.Framework != "" {
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, filter.Framework)
		argCount++
	}

	if filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if !filter.DateFrom.IsZero() {
		whereClause += fmt.Sprintf(" AND generated_at >= $%d", argCount)
		args = append(args, filter.DateFrom)
		argCount++
	}

	if !filter.DateTo.IsZero() {
		whereClause += fmt.Sprintf(" AND generated_at <= $%d", argCount)
		args = append(args, filter.DateTo)
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM compliance_reports " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListComplianceReports.Count: %w", err)
	}

	// Get reports with pagination
	query := `
		SELECT * FROM compliance_reports
		` + whereClause + `
		ORDER BY generated_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &reports, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListComplianceReports: %w", err)
	}

	return reports, total, nil
}

// UpdateComplianceReportStatus updates the status of a compliance report
func (r *ReportRepository) UpdateComplianceReportStatus(ctx context.Context, id, tenantID uuid.UUID, status string, data *json.RawMessage) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `
		UPDATE compliance_reports SET
			status = $1,
			data = COALESCE($2, data),
			updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`
	_, err := r.Db.ExecContext(ctx, query, status, data, id, tenantID)
	if err != nil {
		return fmt.Errorf("report.UpdateComplianceReportStatus: %w", err)
	}
	return nil
}

// DeleteComplianceReport soft deletes a compliance report
func (r *ReportRepository) DeleteComplianceReport(ctx context.Context, id, tenantID uuid.UUID) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `UPDATE compliance_reports SET status = 'deleted', updated_at = NOW() WHERE id = $1 AND tenant_id = $2`
	_, err := r.Db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("report.DeleteComplianceReport: %w", err)
	}
	return nil
}

// =============================================================================
// ReportSnapshot Operations
// =============================================================================

// CreateReportSnapshot creates a new report snapshot
func (r *ReportRepository) CreateReportSnapshot(ctx context.Context, snapshot *ReportSnapshot) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, snapshot.TenantID); err != nil {
		return err
	}

	snapshot.ID = uuid.New()
	snapshot.CreatedAt = time.Now()
	snapshot.UpdatedAt = time.Now()

	query := `
		INSERT INTO report_snapshots (
			id, tenant_id, report_id, snapshot_name, framework,
			generated_at, generated_by, status,
			file_url, file_size_bytes, file_format, storage_path,
			period_start, period_end, summary, metadata,
			expires_at, error_message, error_details, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :report_id, :snapshot_name, :framework,
			:generated_at, :generated_by, :status,
			:file_url, :file_size_bytes, :file_format, :storage_path,
			:period_start, :period_end, :summary, :metadata,
			:expires_at, :error_message, :error_details, :created_at, :updated_at
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, snapshot)
	if err != nil {
		return fmt.Errorf("report.CreateReportSnapshot: %w", err)
	}
	return nil
}

// GetReportSnapshotByID retrieves a report snapshot by ID
func (r *ReportRepository) GetReportSnapshotByID(ctx context.Context, id, tenantID uuid.UUID) (*ReportSnapshot, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var snapshot ReportSnapshot
	query := `SELECT * FROM report_snapshots WHERE id = $1 AND tenant_id = $2`
	err := r.Db.GetContext(ctx, &snapshot, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportSnapshotByID: %w", err)
	}
	return &snapshot, nil
}

// ListReportSnapshots lists report snapshots with filters
func (r *ReportRepository) ListReportSnapshots(ctx context.Context, tenantID uuid.UUID, filter ReportSnapshotFilter) ([]ReportSnapshot, int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, 0, err
	}

	var snapshots []ReportSnapshot

	// Build where clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if filter.ReportID != nil {
		whereClause += fmt.Sprintf(" AND report_id = $%d", argCount)
		args = append(args, *filter.ReportID)
		argCount++
	}

	if filter.Framework != "" {
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, filter.Framework)
		argCount++
	}

	if filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if !filter.DateFrom.IsZero() {
		whereClause += fmt.Sprintf(" AND generated_at >= $%d", argCount)
		args = append(args, filter.DateFrom)
		argCount++
	}

	if !filter.DateTo.IsZero() {
		whereClause += fmt.Sprintf(" AND generated_at <= $%d", argCount)
		args = append(args, filter.DateTo)
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM report_snapshots " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSnapshots.Count: %w", err)
	}

	// Get snapshots with pagination
	query := `
		SELECT * FROM report_snapshots
		` + whereClause + `
		ORDER BY generated_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &snapshots, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSnapshots: %w", err)
	}

	return snapshots, total, nil
}

// UpdateReportSnapshotStatus updates the status and related fields of a report snapshot
func (r *ReportRepository) UpdateReportSnapshotStatus(ctx context.Context, id, tenantID uuid.UUID, status ReportSnapshotStatus, fileURL *string, fileSizeBytes *int64, errorMsg *string) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `
		UPDATE report_snapshots SET
			status = $1,
			file_url = COALESCE($2, file_url),
			file_size_bytes = COALESCE($3, file_size_bytes),
			error_message = COALESCE($4, error_message),
			updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6
	`
	_, err := r.Db.ExecContext(ctx, query, status, fileURL, fileSizeBytes, errorMsg, id, tenantID)
	if err != nil {
		return fmt.Errorf("report.UpdateReportSnapshotStatus: %w", err)
	}
	return nil
}

// GetReportSnapshotStats retrieves statistics for report snapshots
func (r *ReportRepository) GetReportSnapshotStats(ctx context.Context, tenantID uuid.UUID) (*ReportSnapshotStats, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var stats ReportSnapshotStats
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
			COUNT(*) as total_count,
			COALESCE(SUM(file_size_bytes) FILTER (WHERE status = 'completed'), 0) as total_storage_bytes
		FROM report_snapshots
		WHERE tenant_id = $1
	`
	err := r.Db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportSnapshotStats: %w", err)
	}
	return &stats, nil
}

// =============================================================================
// ReportGenerationJob Operations
// =============================================================================

// CreateReportGenerationJob creates a new report generation job
func (r *ReportRepository) CreateReportGenerationJob(ctx context.Context, job *ReportGenerationJob) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, job.TenantID); err != nil {
		return err
	}

	job.ID = uuid.New()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.QueuedAt = time.Now()
	if job.Status == "" {
		job.Status = ReportJobStatusQueued
	}
	if job.Progress == 0 {
		job.Progress = 0
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	query := `
		INSERT INTO report_generation_jobs (
			id, tenant_id, job_type, snapshot_id, report_id,
			status, progress, format, options,
			queued_at, started_at, completed_at,
			error_message, error_details, retry_count, max_retries,
			worker_id, correlation_id, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :job_type, :snapshot_id, :report_id,
			:status, :progress, :format, :options,
			:queued_at, :started_at, :completed_at,
			:error_message, :error_details, :retry_count, :max_retries,
			:worker_id, :correlation_id, :metadata, :created_at, :updated_at
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, job)
	if err != nil {
		return fmt.Errorf("report.CreateReportGenerationJob: %w", err)
	}
	return nil
}

// GetReportGenerationJobByID retrieves a report generation job by ID
func (r *ReportRepository) GetReportGenerationJobByID(ctx context.Context, id, tenantID uuid.UUID) (*ReportGenerationJob, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var job ReportGenerationJob
	query := `SELECT * FROM report_generation_jobs WHERE id = $1 AND tenant_id = $2`
	err := r.Db.GetContext(ctx, &job, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportGenerationJobByID: %w", err)
	}
	return &job, nil
}

// ListReportGenerationJobs lists report generation jobs with filters
func (r *ReportRepository) ListReportGenerationJobs(ctx context.Context, tenantID uuid.UUID, filter ReportJobFilter) ([]ReportGenerationJob, int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, 0, err
	}

	var jobs []ReportGenerationJob

	// Build where clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.JobType != "" {
		whereClause += fmt.Sprintf(" AND job_type = $%d", argCount)
		args = append(args, filter.JobType)
		argCount++
	}

	if !filter.DateFrom.IsZero() {
		whereClause += fmt.Sprintf(" AND queued_at >= $%d", argCount)
		args = append(args, filter.DateFrom)
		argCount++
	}

	if !filter.DateTo.IsZero() {
		whereClause += fmt.Sprintf(" AND queued_at <= $%d", argCount)
		args = append(args, filter.DateTo)
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM report_generation_jobs " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListReportGenerationJobs.Count: %w", err)
	}

	// Get jobs with pagination
	query := `
		SELECT * FROM report_generation_jobs
		` + whereClause + `
		ORDER BY queued_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListReportGenerationJobs: %w", err)
	}

	return jobs, total, nil
}

// UpdateReportGenerationJobStatus updates the status and progress of a job
func (r *ReportRepository) UpdateReportGenerationJobStatus(ctx context.Context, id, tenantID uuid.UUID, status ReportJobStatus, progress int, errorMsg *string, snapshotID *uuid.UUID) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `
		UPDATE report_generation_jobs SET
			status = $1,
			progress = $2,
			error_message = COALESCE($3, error_message),
			snapshot_id = COALESCE($4, snapshot_id),
			started_at = CASE WHEN $5 = 'processing' AND started_at IS NULL THEN NOW() ELSE started_at END,
			completed_at = CASE WHEN $5 IN ('completed', 'failed', 'cancelled') THEN NOW() ELSE completed_at END,
			updated_at = NOW()
		WHERE id = $6 AND tenant_id = $7
	`
	_, err := r.Db.ExecContext(ctx, query, status, progress, errorMsg, snapshotID, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("report.UpdateReportGenerationJobStatus: %w", err)
	}
	return nil
}

// IncrementJobRetryCount increments the retry count for a failed job
func (r *ReportRepository) IncrementJobRetryCount(ctx context.Context, id, tenantID uuid.UUID) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `
		UPDATE report_generation_jobs SET
			retry_count = retry_count + 1,
			status = CASE WHEN retry_count + 1 >= max_retries THEN 'failed' ELSE 'queued' END,
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := r.Db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("report.IncrementJobRetryCount: %w", err)
	}
	return nil
}

// ListQueuedJobs retrieves queued jobs for processing
func (r *ReportRepository) ListQueuedJobs(ctx context.Context, limit int) ([]ReportGenerationJob, error) {
	var jobs []ReportGenerationJob
	query := `
		SELECT * FROM report_generation_jobs
		WHERE status = 'queued'
		ORDER BY queued_at ASC
		LIMIT $1
	`
	err := r.Db.SelectContext(ctx, &jobs, query, limit)
	if err != nil {
		return nil, fmt.Errorf("report.ListQueuedJobs: %w", err)
	}
	return jobs, nil
}

// =============================================================================
// ReportSchedule Operations
// =============================================================================

// CreateReportSchedule creates a new report schedule
func (r *ReportRepository) CreateReportSchedule(ctx context.Context, schedule *ReportSchedule) error {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, schedule.TenantID); err != nil {
		return err
	}

	schedule.ID = uuid.New()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()
	if schedule.Status == "" {
		schedule.Status = ReportScheduleStatusActive
	}

	query := `
		INSERT INTO report_schedules (
			id, tenant_id, schedule_name, framework, report_id,
			schedule_type, cron_expression, format, options,
			recipients, notify_on_completion, notify_on_failure,
			status, next_run_at, last_run_at, last_successful_run_at,
			total_runs, successful_runs, failed_runs,
			created_by, owned_by, retention_days, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :schedule_name, :framework, :report_id,
			:schedule_type, :cron_expression, :format, :options,
			:recipients, :notify_on_completion, :notify_on_failure,
			:status, :next_run_at, :last_run_at, :last_successful_run_at,
			:total_runs, :successful_runs, :failed_runs,
			:created_by, :owned_by, :retention_days, :metadata, :created_at, :updated_at
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("report.CreateReportSchedule: %w", err)
	}
	return nil
}

// GetReportScheduleByID retrieves a report schedule by ID
func (r *ReportRepository) GetReportScheduleByID(ctx context.Context, id, tenantID uuid.UUID) (*ReportSchedule, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var schedule ReportSchedule
	query := `SELECT * FROM report_schedules WHERE id = $1 AND tenant_id = $2`
	err := r.Db.GetContext(ctx, &schedule, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportScheduleByID: %w", err)
	}
	return &schedule, nil
}

// ListReportSchedules lists report schedules with filters
func (r *ReportRepository) ListReportSchedules(ctx context.Context, tenantID uuid.UUID, filter ReportScheduleFilter) ([]ReportSchedule, int, error) {
	// Set tenant context for RLS
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, 0, err
	}

	var schedules []ReportSchedule

	// Build where clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if filter.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.Framework != "" {
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, filter.Framework)
		argCount++
	}

	if filter.OwnedBy != nil {
		whereClause += fmt.Sprintf(" AND owned_by = $%d", argCount)
		args = append(args, *filter.OwnedBy)
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM report_schedules " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSchedules.Count: %w", err)
	}

	// Get schedules with pagination
	query := `
		SELECT * FROM report_schedules
		` + whereClause + `
		ORDER BY next_run_at ASC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &schedules, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSchedules: %w", err)
	}

	return schedules, total, nil
}

// UpdateReportSchedule updates a report schedule
func (r *ReportRepository) UpdateReportSchedule(ctx context.Context, schedule *ReportSchedule) error {
	schedule.UpdatedAt = time.Now()

	query := `
		UPDATE report_schedules SET
			schedule_name = :schedule_name,
			schedule_type = :schedule_type,
			cron_expression = :cron_expression,
			format = :format,
			options = :options,
			recipients = :recipients,
			notify_on_completion = :notify_on_completion,
			notify_on_failure = :notify_on_failure,
			status = :status,
			next_run_at = :next_run_at,
			retention_days = :retention_days,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id
	`
	_, err := r.Db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("report.UpdateReportSchedule: %w", err)
	}
	return nil
}

// UpdateScheduleAfterRun updates schedule statistics after a run
func (r *ReportRepository) UpdateScheduleAfterRun(ctx context.Context, id uuid.UUID, success bool, nextRunAt *time.Time) error {
	query := `
		UPDATE report_schedules SET
			last_run_at = NOW(),
			next_run_at = COALESCE($1, next_run_at),
			total_runs = total_runs + 1,
			successful_runs = successful_runs + CASE WHEN $2 THEN 1 ELSE 0 END,
			failed_runs = failed_runs + CASE WHEN NOT $2 THEN 1 ELSE 0 END,
			last_successful_run_at = CASE WHEN $2 THEN NOW() ELSE last_successful_run_at END,
			updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.Db.ExecContext(ctx, query, nextRunAt, success, id)
	if err != nil {
		return fmt.Errorf("report.UpdateScheduleAfterRun: %w", err)
	}
	return nil
}

// DeleteReportSchedule deletes a report schedule
func (r *ReportRepository) DeleteReportSchedule(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM report_schedules WHERE id = $1 AND tenant_id = $2`
	_, err := r.Db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("report.DeleteReportSchedule: %w", err)
	}
	return nil
}

// GetDueSchedules retrieves schedules that are due to run
func (r *ReportRepository) GetDueSchedules(ctx context.Context, limit int) ([]ReportSchedule, error) {
	var schedules []ReportSchedule
	query := `
		SELECT * FROM report_schedules
		WHERE status = 'active' AND next_run_at <= NOW()
		ORDER BY next_run_at ASC
		LIMIT $1
	`
	err := r.Db.SelectContext(ctx, &schedules, query, limit)
	if err != nil {
		return nil, fmt.Errorf("report.GetDueSchedules: %w", err)
	}
	return schedules, nil
}

// SetTenantIDContext sets the tenant_id context for RLS
func (r *ReportRepository) SetTenantIDContext(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, "tenant_id", tenantID.String())
}
