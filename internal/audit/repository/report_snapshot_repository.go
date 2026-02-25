// Package repository provides data access layer for report snapshots
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// ReportSnapshotRepository handles report snapshot CRUD operations
type ReportSnapshotRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewReportSnapshotRepository creates a new report snapshot repository
func NewReportSnapshotRepository(db *sqlx.DB, logger zerolog.Logger) *ReportSnapshotRepository {
	return &ReportSnapshotRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new report snapshot
func (r *ReportSnapshotRepository) Create(ctx context.Context, snapshot *model.ReportSnapshot) error {
	snapshot.ID = uuid.New()
	snapshot.CreatedAt = time.Now()
	snapshot.UpdatedAt = time.Now()

	query := `
		INSERT INTO report_snapshots (
			id, tenant_id, report_id, snapshot_name, framework, generated_at, generated_by,
			status, file_url, file_size_bytes, file_format, storage_path,
			period_start, period_end, summary, metadata, expires_at,
			error_message, error_details, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :report_id, :snapshot_name, :framework, :generated_at, :generated_by,
			:status, :file_url, :file_size_bytes, :file_format, :storage_path,
			:period_start, :period_end, :summary, :metadata, :expires_at,
			:error_message, :error_details, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, snapshot)
	if err != nil {
		return fmt.Errorf("report_snapshot.Create: %w", err)
	}

	r.logger.Info().
		Str("snapshot_id", snapshot.ID.String()).
		Str("tenant_id", snapshot.TenantID.String()).
		Str("framework", snapshot.Framework).
		Msg("Report snapshot created")

	return nil
}

// GetByID retrieves a report snapshot by ID
func (r *ReportSnapshotRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ReportSnapshot, error) {
	var snapshot model.ReportSnapshot
	query := `SELECT * FROM report_snapshots WHERE id = $1`

	err := r.db.GetContext(ctx, &snapshot, query, id)
	if err != nil {
		return nil, fmt.Errorf("report_snapshot.GetByID: %w", err)
	}

	return &snapshot, nil
}

// List retrieves report snapshots with filtering and pagination
func (r *ReportSnapshotRepository) List(ctx context.Context, tenantID uuid.UUID, filter model.ReportSnapshotFilter, limit, offset int) ([]model.ReportSnapshot, int, error) {
	baseQuery := `
		SELECT * FROM report_snapshots
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM report_snapshots WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.ReportID != nil {
		baseQuery += fmt.Sprintf(" AND report_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND report_id = $%d", argCount)
		args = append(args, *filter.ReportID)
		argCount++
	}

	if filter.Framework != nil {
		baseQuery += fmt.Sprintf(" AND framework = $%d", argCount)
		countQuery += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, *filter.Framework)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.Format != nil {
		baseQuery += fmt.Sprintf(" AND file_format = $%d", argCount)
		countQuery += fmt.Sprintf(" AND file_format = $%d", argCount)
		args = append(args, *filter.Format)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND generated_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND generated_at >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND generated_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND generated_at <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	// Only include non-expired snapshots unless explicitly requested
	if !filter.IncludeExpired {
		baseQuery += fmt.Sprintf(" AND (status != 'expired' OR expires_at IS NULL OR expires_at > NOW())")
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:1]...); err != nil {
		return nil, 0, fmt.Errorf("report_snapshot.List.Count: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY generated_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var snapshots []model.ReportSnapshot
	if err := r.db.SelectContext(ctx, &snapshots, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report_snapshot.List: %w", err)
	}

	return snapshots, total, nil
}

// GetByReportID retrieves all snapshots for a specific report
func (r *ReportSnapshotRepository) GetByReportID(ctx context.Context, reportID uuid.UUID) ([]model.ReportSnapshot, error) {
	var snapshots []model.ReportSnapshot
	query := `
		SELECT * FROM report_snapshots
		WHERE report_id = $1
		ORDER BY generated_at DESC
	`

	err := r.db.SelectContext(ctx, &snapshots, query, reportID)
	if err != nil {
		return nil, fmt.Errorf("report_snapshot.GetByReportID: %w", err)
	}

	return snapshots, nil
}

// UpdateStatus updates the status of a report snapshot
func (r *ReportSnapshotRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, fileURL *string, fileSizeBytes *int64, errorMessage *string) error {
	query := `
		UPDATE report_snapshots SET
			status = $1,
			file_url = COALESCE($2, file_url),
			file_size_bytes = COALESCE($3, file_size_bytes),
			error_message = COALESCE($4, error_message),
			updated_at = NOW()
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query, status, fileURL, fileSizeBytes, errorMessage, id)
	if err != nil {
		return fmt.Errorf("report_snapshot.UpdateStatus: %w", err)
	}

	return nil
}

// Update updates a report snapshot
func (r *ReportSnapshotRepository) Update(ctx context.Context, snapshot *model.ReportSnapshot) error {
	snapshot.UpdatedAt = time.Now()

	query := `
		UPDATE report_snapshots SET
			snapshot_name = :snapshot_name,
			status = :status,
			file_url = :file_url,
			file_size_bytes = :file_size_bytes,
			file_format = :file_format,
			storage_path = :storage_path,
			summary = :summary,
			metadata = :metadata,
			expires_at = :expires_at,
			error_message = :error_message,
			error_details = :error_details,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, snapshot)
	if err != nil {
		return fmt.Errorf("report_snapshot.Update: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("report_snapshot.Update: no rows affected")
	}

	return nil
}

// Delete deletes a report snapshot
func (r *ReportSnapshotRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM report_snapshots WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report_snapshot.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("report_snapshot.Delete: no rows affected")
	}

	return nil
}

// ExpireOldSnapshots marks old snapshots as expired
func (r *ReportSnapshotRepository) ExpireOldSnapshots(ctx context.Context) (int, error) {
	query := `
		UPDATE report_snapshots
		SET
			status = 'expired',
			file_url = NULL,
			updated_at = NOW()
		WHERE status = 'completed'
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("report_snapshot.ExpireOldSnapshots: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// GetSnapshotStats returns statistics about report snapshots
func (r *ReportSnapshotRepository) GetSnapshotStats(ctx context.Context, tenantID uuid.UUID) (*SnapshotStats, error) {
	query := `
		SELECT
			COUNT(*) as total_snapshots,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
			COALESCE(SUM(file_size_bytes) FILTER (WHERE file_size_bytes IS NOT NULL), 0) as total_storage_bytes
		FROM report_snapshots
		WHERE tenant_id = $1
	`

	var stats SnapshotStats
	err := r.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report_snapshot.GetSnapshotStats: %w", err)
	}

	return &stats, nil
}

// SnapshotStats represents statistics about report snapshots
type SnapshotStats struct {
	TotalSnapshots    int   `db:"total_snapshots"`
	CompletedCount    int   `db:"completed_count"`
	PendingCount      int   `db:"pending_count"`
	FailedCount       int   `db:"failed_count"`
	ExpiredCount      int   `db:"expired_count"`
	TotalStorageBytes int64 `db:"total_storage_bytes"`
}

// =============================================================================
// Report Generation Job Repository Methods
// ============================================================================

// CreateJob creates a new report generation job
func (r *ReportSnapshotRepository) CreateJob(ctx context.Context, job *model.ReportGenerationJob) error {
	job.ID = uuid.New()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	job.QueuedAt = time.Now()

	query := `
		INSERT INTO report_generation_jobs (
			id, tenant_id, job_type, snapshot_id, report_id, status, progress,
			format, options, queued_at, started_at, completed_at,
			error_message, error_details, retry_count, max_retries,
			worker_id, correlation_id, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :job_type, :snapshot_id, :report_id, :status, :progress,
			:format, :options, :queued_at, :started_at, :completed_at,
			:error_message, :error_details, :retry_count, :max_retries,
			:worker_id, :correlation_id, :metadata, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, job)
	if err != nil {
		return fmt.Errorf("report_generation_job.Create: %w", err)
	}

	return nil
}

// GetJobByID retrieves a report generation job by ID
func (r *ReportSnapshotRepository) GetJobByID(ctx context.Context, id uuid.UUID) (*model.ReportGenerationJob, error) {
	var job model.ReportGenerationJob
	query := `SELECT * FROM report_generation_jobs WHERE id = $1`

	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		return nil, fmt.Errorf("report_generation_job.GetByID: %w", err)
	}

	return &job, nil
}

// UpdateJobStatus updates the status and progress of a report generation job
func (r *ReportSnapshotRepository) UpdateJobStatus(ctx context.Context, id uuid.UUID, status string, progress int, errorMessage *string) error {
	query := `
		UPDATE report_generation_jobs SET
			status = $1,
			progress = $2,
			error_message = COALESCE($3, error_message),
			updated_at = NOW()
	`

	// Set started_at when transitioning to processing
	if status == string(model.ReportJobStatusProcessing) {
		query += `, started_at = COALESCE(started_at, NOW())`
	}

	// Set completed_at when transitioning to terminal state
	if status == string(model.ReportJobStatusCompleted) || status == string(model.ReportJobStatusFailed) || status == string(model.ReportJobStatusCancelled) {
		query += `, completed_at = COALESCE(completed_at, NOW())`
	}

	query += ` WHERE id = $4`

	_, err := r.db.ExecContext(ctx, query, status, progress, errorMessage, id)
	if err != nil {
		return fmt.Errorf("report_generation_job.UpdateJobStatus: %w", err)
	}

	return nil
}

// UpdateJob updates a report generation job
func (r *ReportSnapshotRepository) UpdateJob(ctx context.Context, job *model.ReportGenerationJob) error {
	job.UpdatedAt = time.Now()

	query := `
		UPDATE report_generation_jobs SET
			status = :status,
			progress = :progress,
			started_at = :started_at,
			completed_at = :completed_at,
			error_message = :error_message,
			error_details = :error_details,
			retry_count = :retry_count,
			worker_id = :worker_id,
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, job)
	if err != nil {
		return fmt.Errorf("report_generation_job.Update: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("report_generation_job.Update: no rows affected")
	}

	return nil
}

// ListJobs retrieves report generation jobs with filtering
func (r *ReportSnapshotRepository) ListJobs(ctx context.Context, tenantID uuid.UUID, filter model.ReportGenerationJobFilter, limit, offset int) ([]model.ReportGenerationJob, int, error) {
	baseQuery := `
		SELECT * FROM report_generation_jobs
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM report_generation_jobs WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.JobType != nil {
		baseQuery += fmt.Sprintf(" AND job_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND job_type = $%d", argCount)
		args = append(args, *filter.JobType)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND queued_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND queued_at >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND queued_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND queued_at <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:1]...); err != nil {
		return nil, 0, fmt.Errorf("report_generation_job.List.Count: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY queued_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var jobs []model.ReportGenerationJob
	if err := r.db.SelectContext(ctx, &jobs, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report_generation_job.List: %w", err)
	}

	return jobs, total, nil
}

// GetQueuedJobs retrieves jobs that are queued and ready for processing
func (r *ReportSnapshotRepository) GetQueuedJobs(ctx context.Context, limit int) ([]model.ReportGenerationJob, error) {
	var jobs []model.ReportGenerationJob
	query := `
		SELECT * FROM report_generation_jobs
		WHERE status = 'queued'
		  AND (retry_count < max_retries OR max_retries = 0)
		ORDER BY queued_at ASC
		LIMIT $1
	`

	err := r.db.SelectContext(ctx, &jobs, query, limit)
	if err != nil {
		return nil, fmt.Errorf("report_generation_job.GetQueuedJobs: %w", err)
	}

	return jobs, nil
}

// DeleteJob deletes a report generation job
func (r *ReportSnapshotRepository) DeleteJob(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM report_generation_jobs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report_generation_job.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("report_generation_job.Delete: no rows affected")
	}

	return nil
}

// =============================================================================
// Report Schedule Repository Methods
// ============================================================================

// CreateSchedule creates a new report schedule
func (r *ReportSnapshotRepository) CreateSchedule(ctx context.Context, schedule *model.ReportSchedule) error {
	schedule.ID = uuid.New()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	query := `
		INSERT INTO report_schedules (
			id, tenant_id, schedule_name, framework, report_id, schedule_type,
			cron_expression, format, options, recipients, notify_on_completion,
			notify_on_failure, status, next_run_at, last_run_at, last_successful_run_at,
			total_runs, successful_runs, failed_runs, created_by, owned_by,
			retention_days, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :schedule_name, :framework, :report_id, :schedule_type,
			:cron_expression, :format, :options, :recipients, :notify_on_completion,
			:notify_on_failure, :status, :next_run_at, :last_run_at, :last_successful_run_at,
			:total_runs, :successful_runs, :failed_runs, :created_by, :owned_by,
			:retention_days, :metadata, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("report_schedule.Create: %w", err)
	}

	return nil
}

// GetScheduleByID retrieves a report schedule by ID
func (r *ReportSnapshotRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*model.ReportSchedule, error) {
	var schedule model.ReportSchedule
	query := `SELECT * FROM report_schedules WHERE id = $1`

	err := r.db.GetContext(ctx, &schedule, query, id)
	if err != nil {
		return nil, fmt.Errorf("report_schedule.GetByID: %w", err)
	}

	return &schedule, nil
}

// UpdateSchedule updates a report schedule
func (r *ReportSnapshotRepository) UpdateSchedule(ctx context.Context, schedule *model.ReportSchedule) error {
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
			updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("report_schedule.Update: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("report_schedule.Update: no rows affected")
	}

	return nil
}

// DeleteSchedule deletes a report schedule
func (r *ReportSnapshotRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM report_schedules WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report_schedule.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("report_schedule.Delete: no rows affected")
	}

	return nil
}

// ListSchedules retrieves report schedules with filtering
func (r *ReportSnapshotRepository) ListSchedules(ctx context.Context, tenantID uuid.UUID, filter model.ReportScheduleFilter, limit, offset int) ([]model.ReportSchedule, int, error) {
	baseQuery := `
		SELECT * FROM report_schedules
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM report_schedules WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.Framework != nil {
		baseQuery += fmt.Sprintf(" AND framework = $%d", argCount)
		countQuery += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, *filter.Framework)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.ScheduleType != nil {
		baseQuery += fmt.Sprintf(" AND schedule_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND schedule_type = $%d", argCount)
		args = append(args, *filter.ScheduleType)
		argCount++
	}

	if filter.OwnedBy != nil {
		baseQuery += fmt.Sprintf(" AND owned_by = $%d", argCount)
		countQuery += fmt.Sprintf(" AND owned_by = $%d", argCount)
		args = append(args, *filter.OwnedBy)
		argCount++
	}

	// Default to active schedules if requested
	if filter.IncludeActive {
		baseQuery += " AND status = 'active'"
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:1]...); err != nil {
		return nil, 0, fmt.Errorf("report_schedule.List.Count: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY next_run_at ASC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var schedules []model.ReportSchedule
	if err := r.db.SelectContext(ctx, &schedules, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report_schedule.List: %w", err)
	}

	return schedules, total, nil
}

// GetDueSchedules retrieves schedules that are due for execution
func (r *ReportSnapshotRepository) GetDueSchedules(ctx context.Context, limit int) ([]model.ReportSchedule, error) {
	var schedules []model.ReportSchedule
	query := `
		SELECT * FROM report_schedules
		WHERE status = 'active'
		  AND next_run_at <= NOW()
		ORDER BY next_run_at ASC
		LIMIT $1
	`

	err := r.db.SelectContext(ctx, &schedules, query, limit)
	if err != nil {
		return nil, fmt.Errorf("report_schedule.GetDueSchedules: %w", err)
	}

	return schedules, nil
}

// UpdateScheduleAfterRun updates a schedule after a run
func (r *ReportSnapshotRepository) UpdateScheduleAfterRun(ctx context.Context, id uuid.UUID, success bool, nextRunAt *time.Time) error {
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

	_, err := r.db.ExecContext(ctx, query, nextRunAt, success, id)
	if err != nil {
		return fmt.Errorf("report_schedule.UpdateScheduleAfterRun: %w", err)
	}

	return nil
}
