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

// ComplianceExceptionRepository handles compliance exception management
type ComplianceExceptionRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewComplianceExceptionRepository creates a new compliance exception repository
func NewComplianceExceptionRepository(db *sqlx.DB, logger zerolog.Logger) *ComplianceExceptionRepository {
	return &ComplianceExceptionRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new compliance exception
func (r *ComplianceExceptionRepository) Create(ctx context.Context, exception *model.ComplianceException) error {
	exception.ID = uuid.New()
	exception.CreatedAt = time.Now()
	exception.UpdatedAt = time.Now()
	exception.RequestedAt = time.Now()
	exception.Status = string(model.ExceptionStatusPending)

	query := `
		INSERT INTO compliance_exceptions (
			id, tenant_id, control_id, control_name, framework, status, risk_level,
			requested_by, requested_at, justification, business_reason,
			compensating_controls, expires_at, review_date, review_notes, metadata,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :control_id, :control_name, :framework, :status, :risk_level,
			:requested_by, :requested_at, :justification, :business_reason,
			:compensating_controls, :expires_at, :review_date, :review_notes, :metadata,
			:created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("compliance_exception.Create: %w", err)
	}

	r.logger.Info().
		Str("exception_id", exception.ID.String()).
		Str("tenant_id", exception.TenantID.String()).
		Str("control_id", exception.ControlID).
		Str("framework", exception.Framework).
		Msg("Compliance exception created")

	return nil
}

// GetByID retrieves a compliance exception by ID
func (r *ComplianceExceptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ComplianceException, error) {
	var exception model.ComplianceException
	query := `SELECT * FROM compliance_exceptions WHERE id = $1`

	err := r.db.GetContext(ctx, &exception, query, id)
	if err != nil {
		return nil, fmt.Errorf("compliance_exception.GetByID: %w", err)
	}

	return &exception, nil
}

// List retrieves compliance exceptions with filtering
func (r *ComplianceExceptionRepository) List(ctx context.Context, tenantID uuid.UUID, filter model.ComplianceExceptionFilter, limit, offset int) ([]model.ComplianceException, int, error) {
	baseQuery := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM compliance_exceptions WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.ControlID != nil {
		baseQuery += fmt.Sprintf(" AND control_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND control_id = $%d", argCount)
		args = append(args, *filter.ControlID)
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

	if filter.RiskLevel != nil {
		baseQuery += fmt.Sprintf(" AND risk_level = $%d", argCount)
		countQuery += fmt.Sprintf(" AND risk_level = $%d", argCount)
		args = append(args, *filter.RiskLevel)
		argCount++
	}

	if !filter.IncludeExpired {
		baseQuery += fmt.Sprintf(" AND (expires_at IS NULL OR expires_at > $%d)", argCount)
		countQuery += fmt.Sprintf(" AND (expires_at IS NULL OR expires_at > $%d)", argCount)
		args = append(args, time.Now())
		argCount++
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:1]...); err != nil {
		return nil, 0, fmt.Errorf("compliance_exception.List.Count: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY created_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var exceptions []model.ComplianceException
	if err := r.db.SelectContext(ctx, &exceptions, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("compliance_exception.List: %w", err)
	}

	return exceptions, total, nil
}

// UpdateStatus updates the status of a compliance exception
func (r *ComplianceExceptionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, approvedBy *uuid.UUID, riskAcceptedBy *uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE compliance_exceptions SET
			status = $1,
			approved_by = $2,
			approved_at = CASE WHEN $3 THEN $4 ELSE approved_at END,
			risk_accepted_by = $5,
			risk_accepted_at = CASE WHEN $6 THEN $7 ELSE risk_accepted_at END,
			updated_at = $8
		WHERE id = $9
	`

	isApproval := status == string(model.ExceptionStatusApproved)
	isRiskAcceptance := status == string(model.ExceptionStatusApproved)

	_, err := r.db.ExecContext(ctx, query,
		status, approvedBy, isApproval, now, riskAcceptedBy, isRiskAcceptance, now, now, id)
	if err != nil {
		return fmt.Errorf("compliance_exception.UpdateStatus: %w", err)
	}

	return nil
}

// Update updates a compliance exception
func (r *ComplianceExceptionRepository) Update(ctx context.Context, exception *model.ComplianceException) error {
	exception.UpdatedAt = time.Now()

	query := `
		UPDATE compliance_exceptions SET
			control_id = :control_id,
			control_name = :control_name,
			framework = :framework,
			status = :status,
			risk_level = :risk_level,
			justification = :justification,
			business_reason = :business_reason,
			compensating_controls = :compensating_controls,
			expires_at = :expires_at,
			review_date = :review_date,
			review_notes = :review_notes,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("compliance_exception.Update: %w", err)
	}

	return nil
}

// Delete deletes a compliance exception
func (r *ComplianceExceptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM compliance_exceptions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("compliance_exception.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("compliance_exception.Delete: no rows affected")
	}

	return nil
}

// GetByControlID retrieves exceptions for a specific control
func (r *ComplianceExceptionRepository) GetByControlID(ctx context.Context, tenantID uuid.UUID, controlID string) ([]model.ComplianceException, error) {
	var exceptions []model.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND control_id = $2
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &exceptions, query, tenantID, controlID)
	if err != nil {
		return nil, fmt.Errorf("compliance_exception.GetByControlID: %w", err)
	}

	return exceptions, nil
}

// GetExpiringSoon retrieves exceptions that will expire within the specified duration
func (r *ComplianceExceptionRepository) GetExpiringSoon(ctx context.Context, tenantID uuid.UUID, within time.Duration) ([]model.ComplianceException, error) {
	var exceptions []model.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1
		  AND status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at <= $2
		  AND expires_at > NOW()
		ORDER BY expires_at ASC
	`

	err := r.db.SelectContext(ctx, &exceptions, query, tenantID, time.Now().Add(within))
	if err != nil {
		return nil, fmt.Errorf("compliance_exception.GetExpiringSoon: %w", err)
	}

	return exceptions, nil
}

// GetExpired retrieves expired exceptions that are still marked as approved
func (r *ComplianceExceptionRepository) GetExpired(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceException, error) {
	var exceptions []model.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1
		  AND status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at <= NOW()
		ORDER BY expires_at DESC
	`

	err := r.db.SelectContext(ctx, &exceptions, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("compliance_exception.GetExpired: %w", err)
	}

	return exceptions, nil
}

// MarkExpired marks expired exceptions as expired
func (r *ComplianceExceptionRepository) MarkExpired(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `
		UPDATE compliance_exceptions
		SET status = 'expired', updated_at = NOW()
		WHERE tenant_id = $1
		  AND status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at <= NOW()
	`

	result, err := r.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("compliance_exception.MarkExpired: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// GetPending retrieves pending exceptions for approval
func (r *ComplianceExceptionRepository) GetPending(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceException, error) {
	var exceptions []model.ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY requested_at ASC
	`

	err := r.db.SelectContext(ctx, &exceptions, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("compliance_exception.GetPending: %w", err)
	}

	return exceptions, nil
}
