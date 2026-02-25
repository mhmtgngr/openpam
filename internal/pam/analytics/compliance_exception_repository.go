package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// ComplianceExceptionRepository handles compliance exception persistence operations
type ComplianceExceptionRepository struct {
	Db     *sqlx.DB
	Logger zerolog.Logger
}

// NewComplianceExceptionRepository creates a new compliance exception repository
func NewComplianceExceptionRepository(db *sqlx.DB, logger zerolog.Logger) *ComplianceExceptionRepository {
	return &ComplianceExceptionRepository{
		Db:     db,
		Logger: logger,
	}
}

// setTenantContext sets the PostgreSQL tenant context for RLS
func (r *ComplianceExceptionRepository) setTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	_, err := r.Db.ExecContext(ctx, "SET LOCAL app.tenant_id = $1", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// Create creates a new compliance exception
func (r *ComplianceExceptionRepository) Create(ctx context.Context, exception *ComplianceException) error {
	if err := r.setTenantContext(ctx, exception.TenantID); err != nil {
		return err
	}

	exception.ID = uuid.New()
	exception.CreatedAt = time.Now()
	exception.UpdatedAt = time.Now()

	query := `
		INSERT INTO compliance_exceptions (
			id, tenant_id, control_id, control_name, framework,
			status, risk_level,
			requested_by, requested_at, approved_by, approved_at, expires_at,
			justification, business_reason, compensating_controls,
			risk_accepted_by, risk_accepted_at,
			review_date, review_notes, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :control_id, :control_name, :framework,
			:status, :risk_level,
			:requested_by, :requested_at, :approved_by, :approved_at, :expires_at,
			:justification, :business_reason, :compensating_controls,
			:risk_accepted_by, :risk_accepted_at,
			:review_date, :review_notes, :metadata, :created_at, :updated_at
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("exception.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a compliance exception by ID
func (r *ComplianceExceptionRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*ComplianceException, error) {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var exception ComplianceException
	query := `SELECT * FROM compliance_exceptions WHERE id = $1 AND tenant_id = $2`
	err := r.Db.GetContext(ctx, &exception, query, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exception.GetByID: %w", err)
	}

	return &exception, nil
}

// List retrieves compliance exceptions with filters
func (r *ComplianceExceptionRepository) List(ctx context.Context, tenantID uuid.UUID, filter ExceptionFilter) ([]ComplianceException, int, error) {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, 0, err
	}

	var exceptions []ComplianceException

	// Build where clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 2

	if filter.Status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.Framework != "" {
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, filter.Framework)
		argCount++
	}

	if filter.RiskLevel != "" {
		whereClause += fmt.Sprintf(" AND risk_level = $%d", argCount)
		args = append(args, filter.RiskLevel)
		argCount++
	}

	if filter.ControlID != "" {
		whereClause += fmt.Sprintf(" AND control_id = $%d", argCount)
		args = append(args, filter.ControlID)
		argCount++
	}

	if !filter.DateFrom.IsZero() {
		whereClause += fmt.Sprintf(" AND requested_at >= $%d", argCount)
		args = append(args, filter.DateFrom)
		argCount++
	}

	if !filter.DateTo.IsZero() {
		whereClause += fmt.Sprintf(" AND requested_at <= $%d", argCount)
		args = append(args, filter.DateTo)
		argCount++
	}

	if filter.ExpiringSoon {
		// Exceptions expiring within 7 days
		whereClause += fmt.Sprintf(" AND expires_at <= NOW() + INTERVAL '7 days' AND expires_at > NOW()")
	}

	if !filter.IncludeExpired {
		whereClause += " AND (expires_at IS NULL OR expires_at > NOW())"
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM compliance_exceptions " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("exception.List.Count: %w", err)
	}

	// Get exceptions with pagination
	query := `
		SELECT * FROM compliance_exceptions
		` + whereClause + `
		ORDER BY requested_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &exceptions, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("exception.List: %w", err)
	}

	return exceptions, total, nil
}

// Update updates a compliance exception
func (r *ComplianceExceptionRepository) Update(ctx context.Context, exception *ComplianceException) error {
	if err := r.setTenantContext(ctx, exception.TenantID); err != nil {
		return err
	}

	exception.UpdatedAt = time.Now()

	query := `
		UPDATE compliance_exceptions SET
			status = :status,
			risk_level = :risk_level,
			approved_by = :approved_by,
			approved_at = :approved_at,
			expires_at = :expires_at,
			justification = :justification,
			business_reason = :business_reason,
			compensating_controls = :compensating_controls,
			risk_accepted_by = :risk_accepted_by,
			risk_accepted_at = :risk_accepted_at,
			review_date = :review_date,
			review_notes = :review_notes,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id
	`
	_, err := r.Db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("exception.Update: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of a compliance exception
func (r *ComplianceExceptionRepository) UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, status ExceptionStatus, approvedBy *uuid.UUID, expiresAt *time.Time) error {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `
		UPDATE compliance_exceptions SET
			status = $1,
			approved_by = $2,
			approved_at = CASE WHEN $2 IS NOT NULL THEN NOW() ELSE approved_at END,
			expires_at = COALESCE($3, expires_at),
			updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5
	`

	_, err := r.Db.ExecContext(ctx, query, status, approvedBy, expiresAt, id, tenantID)
	if err != nil {
		return fmt.Errorf("exception.UpdateStatus: %w", err)
	}

	return nil
}

// Delete soft deletes a compliance exception
func (r *ComplianceExceptionRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return err
	}

	query := `DELETE FROM compliance_exceptions WHERE id = $1 AND tenant_id = $2`
	_, err := r.Db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("exception.Delete: %w", err)
	}

	return nil
}

// GetStats retrieves statistics about compliance exceptions
func (r *ComplianceExceptionRepository) GetStats(ctx context.Context, tenantID uuid.UUID) (*ExceptionStats, error) {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var stats ExceptionStats
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'approved') as approved_count,
			COUNT(*) FILTER (WHERE status = 'denied') as denied_count,
			COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
			COUNT(*) FILTER (WHERE status = 'revoked') as revoked_count,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE risk_level = 'critical') as critical_count,
			COUNT(*) FILTER (WHERE risk_level = 'high') as high_count,
			COUNT(*) FILTER (WHERE expires_at <= NOW() + INTERVAL '7 days' AND expires_at > NOW() AND status = 'approved') as expiring_soon_count
		FROM compliance_exceptions
		WHERE tenant_id = $1
	`

	err := r.Db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("exception.GetStats: %w", err)
	}

	return &stats, nil
}

// GetByControlID retrieves exceptions for a specific control
func (r *ComplianceExceptionRepository) GetByControlID(ctx context.Context, tenantID uuid.UUID, controlID string, framework string) ([]ComplianceException, error) {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var exceptions []ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND control_id = $2 AND framework = $3
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY requested_at DESC
	`

	err := r.Db.SelectContext(ctx, &exceptions, query, tenantID, controlID, framework)
	if err != nil {
		return nil, fmt.Errorf("exception.GetByControlID: %w", err)
	}

	return exceptions, nil
}

// GetPendingApprovals retrieves exceptions pending approval
func (r *ComplianceExceptionRepository) GetPendingApprovals(ctx context.Context, tenantID uuid.UUID, limit int) ([]ComplianceException, error) {
	if err := r.setTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	var exceptions []ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY requested_at ASC
		LIMIT $2
	`

	err := r.Db.SelectContext(ctx, &exceptions, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("exception.GetPendingApprovals: %w", err)
	}

	return exceptions, nil
}

// GetExpiringExceptions retrieves exceptions expiring soon
func (r *ComplianceExceptionRepository) GetExpiringExceptions(ctx context.Context, days int, limit int) ([]ComplianceException, error) {
	var exceptions []ComplianceException
	query := `
		SELECT ce.* FROM compliance_exceptions ce
		WHERE ce.status = 'approved'
		  AND ce.expires_at <= NOW() + INTERVAL '1 day' * $1
		  AND ce.expires_at > NOW()
		  AND (ce.last_notification_at IS NULL OR ce.last_notification_at < NOW() - INTERVAL '1 day')
		ORDER BY ce.expires_at ASC
		LIMIT $2
	`

	err := r.Db.SelectContext(ctx, &exceptions, query, days, limit)
	if err != nil {
		return nil, fmt.Errorf("exception.GetExpiringExceptions: %w", err)
	}

	return exceptions, nil
}

// UpdateLastNotification updates the last notification timestamp
func (r *ComplianceExceptionRepository) UpdateLastNotification(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE compliance_exceptions
		SET last_notification_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("exception.UpdateLastNotification: %w", err)
	}

	return nil
}
