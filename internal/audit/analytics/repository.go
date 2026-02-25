// Package analytics provides repository interface for compliance data
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Repository defines the interface for compliance data access
type Repository interface {
	// Compliance reports
	CreateComplianceReport(ctx context.Context, report *ComplianceReport) error
	GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error)
	ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]*ComplianceReport, error)
	UpdateComplianceReport(ctx context.Context, report *ComplianceReport) error

	// Control evaluations
	CreateControlEvaluation(ctx context.Context, evaluation *ControlEvaluation) error
	ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]*ControlEvaluation, error)

	// Compliance exceptions
	CreateComplianceException(ctx context.Context, exception *ComplianceException) error
	GetComplianceException(ctx context.Context, id uuid.UUID) (*ComplianceException, error)
	ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]*ComplianceException, error)
	UpdateComplianceException(ctx context.Context, exception *ComplianceException) error
}

// SQLRepository implements Repository using sqlx
type SQLRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewRepository creates a new SQL repository
func NewRepository(db *sqlx.DB, logger zerolog.Logger) Repository {
	return &SQLRepository{
		db:     db,
		logger: logger,
	}
}

// CreateComplianceReport creates a new compliance report
func (r *SQLRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	report.ID = uuid.New()
	report.CreatedAt = time.Now()

	query := `
		INSERT INTO compliance_reports (
			id, tenant_id, report_name, framework, version,
			generated_at, generated_by, status, overall_score,
			total_controls, passed_controls, failed_controls, skipped_controls,
			period_start, period_end, summary, findings, recommendations, metadata, created_at
		) VALUES (
			:id, :tenant_id, :report_name, :framework, :version,
			:generated_at, :generated_by, :status, :overall_score,
			:total_controls, :passed_controls, :failed_controls, :skipped_controls,
			:period_start, :period_end, :summary, :findings, :recommendations, :metadata, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, report)
	if err != nil {
		return err
	}

	r.logger.Info().
		Str("report_id", report.ID.String()).
		Str("framework", report.Framework).
		Msg("Compliance report created")

	return nil
}

// GetComplianceReport retrieves a compliance report by ID
func (r *SQLRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	var report ComplianceReport
	query := `SELECT * FROM compliance_reports WHERE id = $1`

	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		return nil, err
	}

	return &report, nil
}

// ListComplianceReports lists compliance reports with filtering
func (r *SQLRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]*ComplianceReport, error) {
	baseQuery := `SELECT * FROM compliance_reports WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.Framework != nil {
		baseQuery += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, *filter.Framework)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

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

	var reports []*ComplianceReport
	err := r.db.SelectContext(ctx, &reports, baseQuery, args...)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

// UpdateComplianceReport updates a compliance report
func (r *SQLRepository) UpdateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	query := `
		UPDATE compliance_reports SET
			status = :status,
			overall_score = :overall_score,
			total_controls = :total_controls,
			passed_controls = :passed_controls,
			failed_controls = :failed_controls,
			skipped_controls = :skipped_controls,
			summary = :summary,
			findings = :findings,
			recommendations = :recommendations,
			updated_at = NOW()
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, report)
	return err
}

// CreateControlEvaluation creates a new control evaluation
func (r *SQLRepository) CreateControlEvaluation(ctx context.Context, evaluation *ControlEvaluation) error {
	evaluation.ID = uuid.New()
	evaluation.EvaluatedAt = time.Now()

	query := `
		INSERT INTO compliance_control_evaluations (
			id, report_id, tenant_id, control_id, control_name, control_category,
			status, score, evidence_count, evidence_urls, findings, remediation_steps,
			metadata, evaluated_at
		) VALUES (
			:id, :report_id, :tenant_id, :control_id, :control_name, :control_category,
			:status, :score, :evidence_count, :evidence_urls, :findings, :remediation_steps,
			:metadata, :evaluated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, evaluation)
	return err
}

// ListControlEvaluations lists control evaluations for a report
func (r *SQLRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]*ControlEvaluation, error) {
	var evaluations []*ControlEvaluation
	query := `SELECT * FROM compliance_control_evaluations WHERE report_id = $1 ORDER BY control_id`

	err := r.db.SelectContext(ctx, &evaluations, query, reportID)
	if err != nil {
		return nil, err
	}

	return evaluations, nil
}

// CreateComplianceException creates a new compliance exception
func (r *SQLRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	exception.ID = uuid.New()
	exception.CreatedAt = time.Now()
	exception.UpdatedAt = time.Now()
	exception.RequestedAt = time.Now()

	if exception.Status == "" {
		exception.Status = "pending"
	}

	query := `
		INSERT INTO compliance_exceptions (
			id, tenant_id, control_id, control_name, framework, status, risk_level,
			requested_by, requested_at, approved_by, approved_at, expires_at,
			justification, business_reason, compensating_controls,
			risk_accepted_by, risk_accepted_at, review_date, review_notes,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :control_id, :control_name, :framework, :status, :risk_level,
			:requested_by, :requested_at, :approved_by, :approved_at, :expires_at,
			:justification, :business_reason, :compensating_controls,
			:risk_accepted_by, :risk_accepted_at, :review_date, :review_notes,
			:metadata, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return err
	}

	r.logger.Info().
		Str("exception_id", exception.ID.String()).
		Str("control_id", exception.ControlID).
		Str("framework", exception.Framework).
		Msg("Compliance exception created")

	return nil
}

// GetComplianceException retrieves a compliance exception by ID
func (r *SQLRepository) GetComplianceException(ctx context.Context, id uuid.UUID) (*ComplianceException, error) {
	var exception ComplianceException
	query := `SELECT * FROM compliance_exceptions WHERE id = $1`

	err := r.db.GetContext(ctx, &exception, query, id)
	if err != nil {
		return nil, err
	}

	return &exception, nil
}

// ListComplianceExceptions lists compliance exceptions for a tenant
func (r *SQLRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]*ComplianceException, error) {
	var exceptions []*ComplianceException
	query := `SELECT * FROM compliance_exceptions WHERE tenant_id = $1 ORDER BY requested_at DESC`

	err := r.db.SelectContext(ctx, &exceptions, query, tenantID)
	if err != nil {
		return nil, err
	}

	return exceptions, nil
}

// UpdateComplianceException updates a compliance exception
func (r *SQLRepository) UpdateComplianceException(ctx context.Context, exception *ComplianceException) error {
	exception.UpdatedAt = time.Now()

	query := `
		UPDATE compliance_exceptions SET
			status = :status,
			approved_by = :approved_by,
			approved_at = :approved_at,
			expires_at = :expires_at,
			review_date = :review_date,
			review_notes = :review_notes,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, exception)
	return err
}

// =============================================================================
// Data Types
// =============================================================================

// ComplianceFramework represents a compliance framework
type ComplianceFramework string

const (
	FrameworkSOC2     ComplianceFramework = "SOC2"
	FrameworkISO27001 ComplianceFramework = "ISO27001"
	FrameworkPCIDSS   ComplianceFramework = "PCI-DSS"
	FrameworkHIPAA    ComplianceFramework = "HIPAA"
	FrameworkNIST     ComplianceFramework = "NIST-800-53"
	FrameworkGDPR     ComplianceFramework = "GDPR"
)

// ComplianceStatus represents the status of a compliance evaluation
type ComplianceStatus string

const (
	ComplianceStatusPending ComplianceStatus = "pending"
	ComplianceStatusPassed  ComplianceStatus = "passed"
	ComplianceStatusFailed  ComplianceStatus = "failed"
	ComplianceStatusPartial ComplianceStatus = "partial"
)

// RiskLevel represents the risk level of an exception
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	TenantID      uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ReportName    string     `db:"report_name" json:"report_name"`
	Framework     string     `db:"framework" json:"framework"`
	Version       string     `db:"version" json:"version"`
	GeneratedAt   time.Time  `db:"generated_at" json:"generated_at"`
	GeneratedBy   uuid.UUID  `db:"generated_by" json:"generated_by"`
	Status        string     `db:"status" json:"status"`
	OverallScore  *float64   `db:"overall_score" json:"overall_score"`
	TotalControls int        `db:"total_controls" json:"total_controls"`
	PassedControls int       `db:"passed_controls" json:"passed_controls"`
	FailedControls int       `db:"failed_controls" json:"failed_controls"`
	SkippedControls int      `db:"skipped_controls" json:"skipped_controls"`
	PeriodStart    time.Time `db:"period_start" json:"period_start"`
	PeriodEnd      time.Time `db:"period_end" json:"period_end"`
	Summary        *string   `db:"summary" json:"summary"`
	Findings       []byte    `db:"findings" json:"findings"`
	Recommendations []byte   `db:"recommendations" json:"recommendations"`
	Metadata       []byte    `db:"metadata" json:"metadata"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// ControlEvaluation represents a control evaluation
type ControlEvaluation struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	ReportID         uuid.UUID  `db:"report_id" json:"report_id"`
	TenantID         uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ControlID        string     `db:"control_id" json:"control_id"`
	ControlName      string     `db:"control_name" json:"control_name"`
	ControlCategory  *string    `db:"control_category" json:"control_category"`
	Status           string     `db:"status" json:"status"`
	Score            *float64   `db:"score" json:"score"`
	EvidenceCount    int        `db:"evidence_count" json:"evidence_count"`
	EvidenceURLs     []string   `db:"evidence_urls" json:"evidence_urls"`
	Findings         *string    `db:"findings" json:"findings"`
	RemediationSteps []string   `db:"remediation_steps" json:"remediation_steps"`
	Metadata         []byte     `db:"metadata" json:"metadata"`
	EvaluatedAt      time.Time  `db:"evaluated_at" json:"evaluated_at"`
}

// ComplianceException represents a compliance exception
type ComplianceException struct {
	ID                   uuid.UUID  `db:"id" json:"id"`
	TenantID             uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ControlID            string     `db:"control_id" json:"control_id"`
	ControlName          string     `db:"control_name" json:"control_name"`
	Framework            string     `db:"framework" json:"framework"`
	Status               string     `db:"status" json:"status"`
	RiskLevel            string     `db:"risk_level" json:"risk_level"`
	RequestedBy          uuid.UUID  `db:"requested_by" json:"requested_by"`
	RequestedAt          time.Time  `db:"requested_at" json:"requested_at"`
	ApprovedBy           *uuid.UUID  `db:"approved_by" json:"approved_by"`
	ApprovedAt           *time.Time `db:"approved_at" json:"approved_at"`
	ExpiresAt            *time.Time `db:"expires_at" json:"expires_at"`
	Justification        string     `db:"justification" json:"justification"`
	BusinessReason       *string    `db:"business_reason" json:"business_reason"`
	CompensatingControls []string   `db:"compensating_controls" json:"compensating_controls"`
	RiskAcceptedBy       *uuid.UUID  `db:"risk_accepted_by" json:"risk_accepted_by"`
	RiskAcceptedAt       *time.Time `db:"risk_accepted_at" json:"risk_accepted_at"`
	ReviewDate           *time.Time `db:"review_date" json:"review_date"`
	ReviewNotes          *string    `db:"review_notes" json:"review_notes"`
	Metadata             []byte     `db:"metadata" json:"metadata"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
}

// ComplianceFilter filters compliance report queries
type ComplianceFilter struct {
	TenantID  *uuid.UUID
	Framework *string
	Status    *string
	DateFrom  *time.Time
	DateTo    *time.Time
}

// ComplianceSummary represents a summary of compliance across frameworks
type ComplianceSummary struct {
	OverallScore    float64                      `json:"overall_score"`
	PassedControls  int                          `json:"passed_controls"`
	FailedControls  int                          `json:"failed_controls"`
	OpenExceptions  int                          `json:"open_exceptions"`
	Frameworks      map[string]FrameworkStatus   `json:"frameworks"`
}

// FrameworkStatus represents status for a single framework
type FrameworkStatus struct {
	Score         float64   `json:"score"`
	Status        string    `json:"status"`
	LastEvaluated time.Time `json:"last_evaluated"`
}
