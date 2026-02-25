package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
)

// ReportRepository handles report persistence operations
type ReportRepository struct {
	Db     *sqlx.DB
	logger zerolog.Logger
}

// NewReportRepository creates a new report repository
func NewReportRepository(db *sqlx.DB, logger zerolog.Logger) *ReportRepository {
	return &ReportRepository{
		Db:     db,
		logger: logger,
	}
}

// ReportSnapshot Operations

// CreateReportSnapshot creates a new report snapshot
func (r *ReportRepository) CreateReportSnapshot(ctx context.Context, snapshot *analytics.ReportSnapshot) error {
	snapshot.ID = uuid.New()
	snapshot.CreatedAt = time.Now()
	now := time.Now()
	snapshot.CreatedAt = now

	query := `
		INSERT INTO report_snapshots (
			id, tenant_id, report_id, snapshot_name, framework,
			generated_at, generated_by, status, file_url, file_size_bytes,
			file_format, storage_path, period_start, period_end, summary,
			metadata, expires_at, error_message, error_details
		) VALUES (
			:id, :tenant_id, :report_id, :snapshot_name, :framework,
			:generated_at, :generated_by, :status, :file_url, :file_size_bytes,
			:file_format, :storage_path, :period_start, :period_end, :summary,
			:metadata, :expires_at, :error_message, :error_details
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, snapshot)
	if err != nil {
		return fmt.Errorf("report.CreateReportSnapshot: %w", err)
	}
	return nil
}

// GetReportSnapshot retrieves a report snapshot by ID
func (r *ReportRepository) GetReportSnapshot(ctx context.Context, id uuid.UUID) (*analytics.ReportSnapshot, error) {
	var snapshot analytics.ReportSnapshot
	query := `SELECT * FROM report_snapshots WHERE id = $1`
	err := r.Db.GetContext(ctx, &snapshot, query, id)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportSnapshot: %w", err)
	}
	return &snapshot, nil
}

// ListReportSnapshots lists report snapshots with filters
func (r *ReportRepository) ListReportSnapshots(ctx context.Context, tenantID uuid.UUID, framework, status string, startDate, endDate *time.Time, limit, offset int) ([]analytics.ReportSnapshot, int, error) {
	var snapshots []analytics.ReportSnapshot

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if framework != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, framework)
	}

	if status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if startDate != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND generated_at >= $%d", argCount)
		args = append(args, *startDate)
	}

	if endDate != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND generated_at <= $%d", argCount)
		args = append(args, *endDate)
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM report_snapshots " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSnapshots.Count: %w", err)
	}

	// Get snapshots
	query := `
		SELECT * FROM report_snapshots
		` + whereClause + `
		ORDER BY generated_at DESC
		LIMIT $` + fmt.Sprint(argCount+1) + ` OFFSET $` + fmt.Sprint(argCount+2)
	args = append(args, limit, offset)

	err := r.Db.SelectContext(ctx, &snapshots, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSnapshots: %w", err)
	}

	return snapshots, total, nil
}

// UpdateReportSnapshotStatus updates the status of a report snapshot
func (r *ReportRepository) UpdateReportSnapshotStatus(ctx context.Context, id uuid.UUID, status string, fileURL *string, fileSize *int64, errorMsg *string) error {
	query := `
		UPDATE report_snapshots SET
			status = $1,
			file_url = COALESCE($2, file_url),
			file_size_bytes = COALESCE($3, file_size_bytes),
			error_message = COALESCE($4, error_message),
			updated_at = NOW()
		WHERE id = $5
	`
	_, err := r.Db.ExecContext(ctx, query, status, fileURL, fileSize, errorMsg, id)
	if err != nil {
		return fmt.Errorf("report.UpdateReportSnapshotStatus: %w", err)
	}
	return nil
}

// DeleteReportSnapshot deletes a report snapshot
func (r *ReportRepository) DeleteReportSnapshot(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM report_snapshots WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report.DeleteReportSnapshot: %w", err)
	}
	return nil
}

// ExpireOldSnapshots marks old snapshots as expired
func (r *ReportRepository) ExpireOldSnapshots(ctx context.Context) (int, error) {
	query := `
		UPDATE report_snapshots
		SET status = 'expired',
			file_url = NULL,
			updated_at = NOW()
		WHERE status = 'completed'
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
	`
	result, err := r.Db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("report.ExpireOldSnapshots: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// Report Generation Jobs

// CreateReportJob creates a new report generation job
func (r *ReportRepository) CreateReportJob(ctx context.Context, job *analytics.ReportGenerationJob) error {
	job.ID = uuid.New()
	job.QueuedAt = time.Now()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	query := `
		INSERT INTO report_generation_jobs (
			id, tenant_id, job_type, snapshot_id, report_id,
			status, progress, format, options, queued_at,
			error_message, error_details, retry_count, max_retries,
			correlation_id, metadata
		) VALUES (
			:id, :tenant_id, :job_type, :snapshot_id, :report_id,
			:status, :progress, :format, :options, :queued_at,
			:error_message, :error_details, :retry_count, :max_retries,
			:correlation_id, :metadata
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, job)
	if err != nil {
		return fmt.Errorf("report.CreateReportJob: %w", err)
	}
	return nil
}

// GetReportJob retrieves a report generation job by ID
func (r *ReportRepository) GetReportJob(ctx context.Context, id uuid.UUID) (*analytics.ReportGenerationJob, error) {
	var job analytics.ReportGenerationJob
	query := `SELECT * FROM report_generation_jobs WHERE id = $1`
	err := r.Db.GetContext(ctx, &job, query, id)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportJob: %w", err)
	}
	return &job, nil
}

// ListReportJobs lists report generation jobs with filters
func (r *ReportRepository) ListReportJobs(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]analytics.ReportGenerationJob, int, error) {
	var jobs []analytics.ReportGenerationJob

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM report_generation_jobs " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListReportJobs.Count: %w", err)
	}

	// Get jobs
	query := `
		SELECT * FROM report_generation_jobs
		` + whereClause + `
		ORDER BY queued_at DESC
		LIMIT $` + fmt.Sprint(argCount+1) + ` OFFSET $` + fmt.Sprint(argCount+2)
	args = append(args, limit, offset)

	err := r.Db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListReportJobs: %w", err)
	}

	return jobs, total, nil
}

// UpdateReportJobStatus updates the status and progress of a report job
func (r *ReportRepository) UpdateReportJobStatus(ctx context.Context, id uuid.UUID, status string, progress int, errorMsg *string, errorDetails json.RawMessage) error {
	query := `
		UPDATE report_generation_jobs SET
			status = $1,
			progress = $2,
			error_message = COALESCE($3, error_message),
			error_details = COALESCE($4, error_details),
			updated_at = NOW()
	`
	args := []interface{}{status, progress, errorMsg, errorDetails}

	if status == "processing" {
		query += `, started_at = NOW()`
	} else if status == "completed" || status == "failed" || status == "cancelled" {
		query += `, completed_at = NOW()`
	}

	query += " WHERE id = $5"
	args = append(args, id)

	_, err := r.Db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("report.UpdateReportJobStatus: %w", err)
	}
	return nil
}

// UpdateReportJobProgress updates just the progress of a job
func (r *ReportRepository) UpdateReportJobProgress(ctx context.Context, id uuid.UUID, progress int) error {
	query := `
		UPDATE report_generation_jobs
		SET progress = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.Db.ExecContext(ctx, query, progress, id)
	if err != nil {
		return fmt.Errorf("report.UpdateReportJobProgress: %w", err)
	}
	return nil
}

// LinkJobToSnapshot links a job to its created snapshot
func (r *ReportRepository) LinkJobToSnapshot(ctx context.Context, jobID, snapshotID uuid.UUID) error {
	query := `
		UPDATE report_generation_jobs
		SET snapshot_id = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.Db.ExecContext(ctx, query, snapshotID, jobID)
	if err != nil {
		return fmt.Errorf("report.LinkJobToSnapshot: %w", err)
	}
	return nil
}

// DeleteReportJob deletes a report generation job
func (r *ReportRepository) DeleteReportJob(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM report_generation_jobs WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report.DeleteReportJob: %w", err)
	}
	return nil
}

// GetQueuedJobs retrieves jobs that are queued for processing
func (r *ReportRepository) GetQueuedJobs(ctx context.Context, limit int) ([]analytics.ReportGenerationJob, error) {
	var jobs []analytics.ReportGenerationJob
	query := `
		SELECT * FROM report_generation_jobs
		WHERE status = 'queued'
		  AND (retry_count < max_retries OR max_retries = 0)
		ORDER BY queued_at ASC
		LIMIT $1
	`
	err := r.Db.SelectContext(ctx, &jobs, query, limit)
	if err != nil {
		return nil, fmt.Errorf("report.GetQueuedJobs: %w", err)
	}
	return jobs, nil
}

// IncrementJobRetry increments the retry count for a job
func (r *ReportRepository) IncrementJobRetry(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE report_generation_jobs
		SET retry_count = retry_count + 1,
		    status = 'queued',
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report.IncrementJobRetry: %w", err)
	}
	return nil
}

// Report Schedule Operations

// CreateReportSchedule creates a new report schedule
func (r *ReportRepository) CreateReportSchedule(ctx context.Context, schedule *analytics.ReportSchedule) error {
	schedule.ID = uuid.New()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	query := `
		INSERT INTO report_schedules (
			id, tenant_id, schedule_name, framework, report_id,
			schedule_type, cron_expression, format, options,
			recipients, notify_on_completion, notify_on_failure,
			status, next_run_at, retention_days,
			created_by, owned_by, metadata
		) VALUES (
			:id, :tenant_id, :schedule_name, :framework, :report_id,
			:schedule_type, :cron_expression, :format, :options,
			:recipients, :notify_on_completion, :notify_on_failure,
			:status, :next_run_at, :retention_days,
			:created_by, :owned_by, :metadata
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("report.CreateReportSchedule: %w", err)
	}
	return nil
}

// GetReportSchedule retrieves a report schedule by ID
func (r *ReportRepository) GetReportSchedule(ctx context.Context, id uuid.UUID) (*analytics.ReportSchedule, error) {
	var schedule analytics.ReportSchedule
	query := `SELECT * FROM report_schedules WHERE id = $1`
	err := r.Db.GetContext(ctx, &schedule, query, id)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportSchedule: %w", err)
	}
	return &schedule, nil
}

// ListReportSchedules lists report schedules with filters
func (r *ReportRepository) ListReportSchedules(ctx context.Context, tenantID uuid.UUID, status string, framework string, limit, offset int) ([]analytics.ReportSchedule, int, error) {
	var schedules []analytics.ReportSchedule

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if framework != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, framework)
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM report_schedules " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSchedules.Count: %w", err)
	}

	// Get schedules
	query := `
		SELECT * FROM report_schedules
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argCount+1) + ` OFFSET $` + fmt.Sprint(argCount+2)
	args = append(args, limit, offset)

	err := r.Db.SelectContext(ctx, &schedules, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("report.ListReportSchedules: %w", err)
	}

	return schedules, total, nil
}

// UpdateReportSchedule updates a report schedule
func (r *ReportRepository) UpdateReportSchedule(ctx context.Context, schedule *analytics.ReportSchedule) error {
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
		WHERE id = :id
	`
	_, err := r.Db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("report.UpdateReportSchedule: %w", err)
	}
	return nil
}

// DeleteReportSchedule deletes a report schedule
func (r *ReportRepository) DeleteReportSchedule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM report_schedules WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("report.DeleteReportSchedule: %w", err)
	}
	return nil
}

// GetDueSchedules retrieves schedules that are due for execution
func (r *ReportRepository) GetDueSchedules(ctx context.Context, limit int) ([]analytics.ReportSchedule, error) {
	var schedules []analytics.ReportSchedule
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

// GetReportSnapshotStats retrieves statistics about report snapshots
func (r *ReportRepository) GetReportSnapshotStats(ctx context.Context, tenantID uuid.UUID) (*analytics.ReportSnapshotStats, error) {
	var stats analytics.ReportSnapshotStats

	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
			COUNT(*) as total_count,
			COALESCE(SUM(file_size_bytes) FILTER (WHERE status = 'completed' AND file_size_bytes IS NOT NULL), 0) as total_storage_bytes
		FROM report_snapshots
		WHERE tenant_id = $1
	`
	err := r.Db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetReportSnapshotStats: %w", err)
	}

	return &stats, nil
}

// GetJobsBySnapshotID retrieves all jobs for a specific snapshot
func (r *ReportRepository) GetJobsBySnapshotID(ctx context.Context, snapshotID uuid.UUID) ([]analytics.ReportGenerationJob, error) {
	var jobs []analytics.ReportGenerationJob
	query := `
		SELECT * FROM report_generation_jobs
		WHERE snapshot_id = $1
		ORDER BY queued_at DESC
	`
	err := r.Db.SelectContext(ctx, &jobs, query, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("report.GetJobsBySnapshotID: %w", err)
	}
	return jobs, nil
}

// GetSnapshotsByReportID retrieves all snapshots for a specific report
func (r *ReportRepository) GetSnapshotsByReportID(ctx context.Context, reportID uuid.UUID, limit int) ([]analytics.ReportSnapshot, error) {
	var snapshots []analytics.ReportSnapshot
	query := `
		SELECT * FROM report_snapshots
		WHERE report_id = $1
		ORDER BY generated_at DESC
		LIMIT $2
	`
	err := r.Db.SelectContext(ctx, &snapshots, query, reportID, limit)
	if err != nil {
		return nil, fmt.Errorf("report.GetSnapshotsByReportID: %w", err)
	}
	return snapshots, nil
}
