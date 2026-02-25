// Package repository provides data access layer for audit analytics
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

// ComplianceReportRepository handles compliance report CRUD operations
type ComplianceReportRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewComplianceReportRepository creates a new compliance report repository
func NewComplianceReportRepository(db *sqlx.DB, logger zerolog.Logger) *ComplianceReportRepository {
	return &ComplianceReportRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new compliance report
func (r *ComplianceReportRepository) Create(ctx context.Context, report *model.ComplianceReport) error {
	report.ID = uuid.New()
	report.CreatedAt = time.Now()

	query := `
		INSERT INTO compliance_reports (
			id, tenant_id, report_name, framework, version, generated_at, generated_by,
			status, overall_score, total_controls, passed_controls, failed_controls,
			skipped_controls, period_start, period_end, summary, findings,
			recommendations, metadata, created_at
		) VALUES (
			:id, :tenant_id, :report_name, :framework, :version, :generated_at, :generated_by,
			:status, :overall_score, :total_controls, :passed_controls, :failed_controls,
			:skipped_controls, :period_start, :period_end, :summary, :findings,
			:recommendations, :metadata, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("compliance_report.Create: %w", err)
	}

	r.logger.Info().
		Str("report_id", report.ID.String()).
		Str("tenant_id", report.TenantID.String()).
		Str("framework", report.Framework).
		Msg("Compliance report created")

	return nil
}

// GetByID retrieves a compliance report by ID
func (r *ComplianceReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ComplianceReport, error) {
	var report model.ComplianceReport
	query := `SELECT * FROM compliance_reports WHERE id = $1`

	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		return nil, fmt.Errorf("compliance_report.GetByID: %w", err)
	}

	return &report, nil
}

// List retrieves compliance reports with filtering and pagination
func (r *ComplianceReportRepository) List(ctx context.Context, tenantID uuid.UUID, filter model.ComplianceReportFilter, limit, offset int) ([]model.ComplianceReport, int, error) {
	baseQuery := `
		SELECT * FROM compliance_reports
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM compliance_reports WHERE tenant_id = $1
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

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:1]...); err != nil {
		return nil, 0, fmt.Errorf("compliance_report.List.Count: %w", err)
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

	var reports []model.ComplianceReport
	if err := r.db.SelectContext(ctx, &reports, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("compliance_report.List: %w", err)
	}

	return reports, total, nil
}

// GetLatestByFramework retrieves the latest report for a framework and tenant
func (r *ComplianceReportRepository) GetLatestByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*model.ComplianceReport, error) {
	var report model.ComplianceReport
	query := `
		SELECT * FROM compliance_reports
		WHERE tenant_id = $1 AND framework = $2
		ORDER BY generated_at DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &report, query, tenantID, framework)
	if err != nil {
		return nil, fmt.Errorf("compliance_report.GetLatestByFramework: %w", err)
	}

	return &report, nil
}

// UpdateStatus updates the status of a compliance report
func (r *ComplianceReportRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, overallScore *float64, passedControls, failedControls, skippedControls int) error {
	query := `
		UPDATE compliance_reports SET
			status = $1,
			overall_score = $2,
			passed_controls = $3,
			failed_controls = $4,
			skipped_controls = $5
		WHERE id = $6
	`

	_, err := r.db.ExecContext(ctx, query, status, overallScore, passedControls, failedControls, skippedControls, id)
	if err != nil {
		return fmt.Errorf("compliance_report.UpdateStatus: %w", err)
	}

	return nil
}

// Delete deletes a compliance report
func (r *ComplianceReportRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM compliance_reports WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("compliance_report.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("compliance_report.Delete: no rows affected")
	}

	return nil
}

// GetByPeriod retrieves reports for a specific time period
func (r *ComplianceReportRepository) GetByPeriod(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) ([]model.ComplianceReport, error) {
	var reports []model.ComplianceReport
	query := `
		SELECT * FROM compliance_reports
		WHERE tenant_id = $1
		  AND period_start >= $2
		  AND period_end <= $3
		ORDER BY generated_at DESC
	`

	err := r.db.SelectContext(ctx, &reports, query, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("compliance_report.GetByPeriod: %w", err)
	}

	return reports, nil
}

// GetFrameworkSummary returns a summary of all reports by framework for a tenant
func (r *ComplianceReportRepository) GetFrameworkSummary(ctx context.Context, tenantID uuid.UUID) (map[string]FrameworkSummary, error) {
	query := `
		SELECT
			framework,
			COUNT(*) as total_reports,
			COALESCE(AVG(overall_score), 0) as avg_score,
			SUM(CASE WHEN status = 'passed' THEN 1 ELSE 0 END) as passed_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			MAX(generated_at) as last_generated
		FROM compliance_reports
		WHERE tenant_id = $1
		GROUP BY framework
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("compliance_report.GetFrameworkSummary: %w", err)
	}
	defer rows.Close()

	summary := make(map[string]FrameworkSummary)
	for rows.Next() {
		var fs FrameworkSummary
		var lastGenerated time.Time
		if err := rows.Scan(&fs.Framework, &fs.TotalReports, &fs.AvgScore, &fs.PassedCount, &fs.FailedCount, &lastGenerated); err != nil {
			return nil, err
		}
		fs.LastGenerated = lastGenerated
		summary[fs.Framework] = fs
	}

	return summary, nil
}

// FrameworkSummary represents a summary of reports for a framework
type FrameworkSummary struct {
	Framework      string    `db:"framework"`
	TotalReports   int       `db:"total_reports"`
	AvgScore       float64   `db:"avg_score"`
	PassedCount    int       `db:"passed_count"`
	FailedCount    int       `db:"failed_count"`
	LastGenerated  time.Time `db:"last_generated"`
}
