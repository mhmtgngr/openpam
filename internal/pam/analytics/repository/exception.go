package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
)

// ExceptionRepository handles compliance exception operations
type ExceptionRepository struct {
	Db     *sqlx.DB
	logger zerolog.Logger
}

// NewExceptionRepository creates a new exception repository
func NewExceptionRepository(db *sqlx.DB, logger zerolog.Logger) *ExceptionRepository {
	return &ExceptionRepository{
		Db:     db,
		logger: logger,
	}
}

// CreateComplianceException creates a new compliance exception
func (r *ExceptionRepository) CreateComplianceException(ctx context.Context, exception *analytics.ComplianceException) error {
	exception.ID = uuid.New()
	exception.RequestedAt = time.Now()
	exception.Status = analytics.ExceptionStatusPending
	exception.CreatedAt = time.Now()
	exception.UpdatedAt = time.Now()

	query := `
		INSERT INTO compliance_exceptions (
			id, tenant_id, control_id, control_name, framework,
			status, risk_level, requested_by, requested_at,
			justification, business_reason, compensating_controls,
			review_date, review_notes, metadata
		) VALUES (
			:id, :tenant_id, :control_id, :control_name, :framework,
			:status, :risk_level, :requested_by, :requested_at,
			:justification, :business_reason, :compensating_controls,
			:review_date, :review_notes, :metadata
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("exception.CreateComplianceException: %w", err)
	}
	return nil
}

// GetComplianceException retrieves a compliance exception by ID
func (r *ExceptionRepository) GetComplianceException(ctx context.Context, id uuid.UUID) (*analytics.ComplianceException, error) {
	var exception analytics.ComplianceException
	query := `SELECT * FROM compliance_exceptions WHERE id = $1`
	err := r.Db.GetContext(ctx, &exception, query, id)
	if err != nil {
		return nil, fmt.Errorf("exception.GetComplianceException: %w", err)
	}
	return &exception, nil
}

// ListComplianceExceptions lists compliance exceptions with filters
func (r *ExceptionRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID, status *analytics.ExceptionStatus, framework, controlID string, limit, offset int) ([]analytics.ComplianceException, int, error) {
	var exceptions []analytics.ComplianceException

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if status != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *status)
	}

	if framework != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, framework)
	}

	if controlID != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND control_id = $%d", argCount)
		args = append(args, controlID)
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM compliance_exceptions " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("exception.ListComplianceExceptions.Count: %w", err)
	}

	// Get exceptions
	query := `
		SELECT * FROM compliance_exceptions
		` + whereClause + `
		ORDER BY requested_at DESC
		LIMIT $` + fmt.Sprint(argCount+1) + ` OFFSET $` + fmt.Sprint(argCount+2)
	args = append(args, limit, offset)

	err := r.Db.SelectContext(ctx, &exceptions, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("exception.ListComplianceExceptions: %w", err)
	}

	return exceptions, total, nil
}

// UpdateExceptionStatus updates the status of a compliance exception
func (r *ExceptionRepository) UpdateExceptionStatus(ctx context.Context, id uuid.UUID, status analytics.ExceptionStatus, approvedBy *uuid.UUID, expiresAt *time.Time, riskAcceptedBy *uuid.UUID, reviewNotes *string) error {
	query := `
		UPDATE compliance_exceptions SET
			status = $1,
			approved_by = COALESCE($2, approved_by),
			approved_at = CASE WHEN $2 IS NOT NULL THEN NOW() ELSE approved_at END,
			expires_at = COALESCE($3, expires_at),
			risk_accepted_by = COALESCE($4, risk_accepted_by),
			risk_accepted_at = CASE WHEN $4 IS NOT NULL THEN NOW() ELSE risk_accepted_at END,
			review_notes = COALESCE($5, review_notes),
			updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.Db.ExecContext(ctx, query, status, approvedBy, expiresAt, riskAcceptedBy, reviewNotes, id)
	if err != nil {
		return fmt.Errorf("exception.UpdateExceptionStatus: %w", err)
	}
	return nil
}

// UpdateException updates a compliance exception
func (r *ExceptionRepository) UpdateException(ctx context.Context, exception *analytics.ComplianceException) error {
	exception.UpdatedAt = time.Now()

	query := `
		UPDATE compliance_exceptions SET
			control_id = :control_id,
			control_name = :control_name,
			framework = :framework,
			risk_level = :risk_level,
			justification = :justification,
			business_reason = :business_reason,
			compensating_controls = :compensating_controls,
			review_date = :review_date,
			review_notes = :review_notes,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.Db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("exception.UpdateException: %w", err)
	}
	return nil
}

// DeleteComplianceException deletes a compliance exception
func (r *ExceptionRepository) DeleteComplianceException(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM compliance_exceptions WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("exception.DeleteComplianceException: %w", err)
	}
	return nil
}

// GetExpiringExceptions retrieves exceptions that will expire soon
func (r *ExceptionRepository) GetExpiringExceptions(ctx context.Context, tenantID uuid.UUID, days int) ([]analytics.ComplianceException, error) {
	var exceptions []analytics.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1
		  AND status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at BETWEEN NOW() AND NOW() + INTERVAL '1 day' * $2
		ORDER BY expires_at ASC
	`
	err := r.Db.SelectContext(ctx, &exceptions, query, tenantID, days)
	if err != nil {
		return nil, fmt.Errorf("exception.GetExpiringExceptions: %w", err)
	}
	return exceptions, nil
}

// GetExpiredExceptions retrieves exceptions that have expired
func (r *ExceptionRepository) GetExpiredExceptions(ctx context.Context) ([]analytics.ComplianceException, error) {
	var exceptions []analytics.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
		ORDER BY expires_at ASC
	`
	err := r.Db.SelectContext(ctx, &exceptions, query)
	if err != nil {
		return nil, fmt.Errorf("exception.GetExpiredExceptions: %w", err)
	}
	return exceptions, nil
}

// ExpireExceptions marks expired exceptions as expired
func (r *ExceptionRepository) ExpireExceptions(ctx context.Context) (int, error) {
	query := `
		UPDATE compliance_exceptions
		SET status = 'expired',
		    updated_at = NOW()
		WHERE status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at < NOW()
	`
	result, err := r.Db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("exception.ExpireExceptions: %w", err)
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// GetPendingExceptions retrieves exceptions pending approval
func (r *ExceptionRepository) GetPendingExceptions(ctx context.Context, tenantID uuid.UUID, limit int) ([]analytics.ComplianceException, error) {
	var exceptions []analytics.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY requested_at ASC
		LIMIT $2
	`
	err := r.Db.SelectContext(ctx, &exceptions, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("exception.GetPendingExceptions: %w", err)
	}
	return exceptions, nil
}

// GetExceptionStats retrieves statistics about compliance exceptions
func (r *ExceptionRepository) GetExceptionStats(ctx context.Context, tenantID uuid.UUID) (*analytics.ExceptionStats, error) {
	var stats analytics.ExceptionStats

	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'approved') as approved_count,
			COUNT(*) FILTER (WHERE status = 'denied') as denied_count,
			COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
			COUNT(*) FILTER (WHERE status = 'revoked') as revoked_count,
			COUNT(*) as total_count,
			COUNT(*) FILTER (WHERE status = 'approved' AND risk_level = 'critical') as critical_count,
			COUNT(*) FILTER (WHERE status = 'approved' AND risk_level = 'high') as high_count,
			COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at < NOW() + INTERVAL '7 days' AND status = 'approved') as expiring_soon_count
		FROM compliance_exceptions
		WHERE tenant_id = $1
	`
	err := r.Db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("report.GetExceptionStats: %w", err)
	}

	return &stats, nil
}

// GetExceptionsByFramework retrieves exceptions grouped by framework
func (r *ExceptionRepository) GetExceptionsByFramework(ctx context.Context, tenantID uuid.UUID, framework string) ([]analytics.ComplianceException, error) {
	var exceptions []analytics.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND framework = $2
		ORDER BY requested_at DESC
	`
	err := r.Db.SelectContext(ctx, &exceptions, query, tenantID, framework)
	if err != nil {
		return nil, fmt.Errorf("exception.GetExceptionsByFramework: %w", err)
	}
	return exceptions, nil
}

// GetExceptionsByControl retrieves exceptions for a specific control
func (r *ExceptionRepository) GetExceptionsByControl(ctx context.Context, tenantID uuid.UUID, controlID string) ([]analytics.ComplianceException, error) {
	var exceptions []analytics.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND control_id = $2
		ORDER BY requested_at DESC
	`
	err := r.Db.SelectContext(ctx, &exceptions, query, tenantID, controlID)
	if err != nil {
		return nil, fmt.Errorf("exception.GetExceptionsByControl: %w", err)
	}
	return exceptions, nil
}

// RevokeException revokes an approved exception
func (r *ExceptionRepository) RevokeException(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID, reason string) error {
	query := `
		UPDATE compliance_exceptions
		SET status = 'revoked',
		    updated_at = NOW(),
		    review_notes = COALESCE(review_notes || '; Revoked: ' || $2, 'Revoked: ' || $2)
		WHERE id = $1
	`
	_, err := r.Db.ExecContext(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("exception.RevokeException: %w", err)
	}

	// Log the revocation
	r.logger.Info().
		Str("exception_id", id.String()).
		Str("revoked_by", revokedBy.String()).
		Str("reason", reason).
		Msg("Compliance exception revoked")

	return nil
}
