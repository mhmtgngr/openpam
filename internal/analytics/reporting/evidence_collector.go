package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/internal/analytics"
	"github.com/rs/zerolog"
)

// EvidenceCollector handles automatic evidence gathering for compliance reports
type EvidenceCollector struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewEvidenceCollector creates a new evidence collector
func NewEvidenceCollector(db *sqlx.DB, logger zerolog.Logger) *EvidenceCollector {
	return &EvidenceCollector{
		db:     db,
		logger: logger,
	}
}

// Evidence represents collected compliance evidence
type Evidence struct {
	ID               uuid.UUID              `db:"id" json:"id"`
	TenantID         uuid.UUID              `db:"tenant_id" json:"tenant_id"`
	ReportID         *uuid.UUID             `db:"report_id" json:"report_id,omitempty"`

	// Evidence metadata
	Type             string                 `db:"type" json:"type"`
	Category         string                 `db:"category" json:"category"`
	Title            string                 `db:"title" json:"title"`
	Description      string                 `db:"description" json:"description"`

	// Evidence data
	EvidenceData     map[string]interface{} `db:"evidence_data" json:"evidence_data"`
	FilePath         string                 `db:"file_path" json:"file_path,omitempty"`
	FileHash         string                 `db:"file_hash" json:"file_hash,omitempty"`

	// Period validity
	ApplicableFrom   time.Time              `db:"applicable_from" json:"applicable_from"`
	ApplicableUntil   time.Time              `db:"applicable_until" json:"applicable_until"`

	// Source
	SourceType       string                 `db:"source_type" json:"source_type"`
	SourceID         *uuid.UUID             `db:"source_id" json:"source_id,omitempty"`

	// Status
	Status           string                 `db:"status" json:"status"`
	VerifiedBy       *uuid.UUID             `db:"verified_by" json:"verified_by,omitempty"`
	VerifiedAt       *time.Time             `db:"verified_at" json:"verified_at,omitempty"`

	// Tags and metadata
	Tags             []string               `db:"tags" json:"tags,omitempty"`
	Metadata         map[string]interface{} `db:"metadata" json:"metadata,omitempty"`

	CreatedAt        time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time              `db:"updated_at" json:"updated_at"`
	DeletedAt        *time.Time             `db:"deleted_at" json:"deleted_at,omitempty"`
}

// CollectPeriodEvidence collects evidence for a compliance period
func (c *EvidenceCollector) CollectPeriodEvidence(ctx context.Context, tenantID uuid.UUID, framework analytics.ComplianceFramework, periodStart, periodEnd time.Time, controls []ControlMapping) ([]Evidence, error) {
	evidenceList := make([]Evidence, 0)

	// Collect evidence for each control
	for _, control := range controls {
		// Query OpenPAM data sources for this control
		controlEvidence, err := c.collectControlEvidence(ctx, tenantID, control, periodStart, periodEnd)
		if err != nil {
			c.logger.Warn().Err(err).
				Str("control_id", control.ControlID).
				Msg("Failed to collect evidence for control")
			continue
		}

		evidenceList = append(evidenceList, controlEvidence...)
	}

	return evidenceList, nil
}

// collectControlEvidence collects evidence for a single control
func (c *EvidenceCollector) collectControlEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) ([]Evidence, error) {
	evidenceList := make([]Evidence, 0)

	// Determine which data sources to query based on OpenPAM feature
	for _, source := range control.EvidenceSources {
		var evidence []Evidence

		switch source {
		case "access_requests":
			evidence = c.getAccessRequestEvidence(ctx, tenantID, control, periodStart, periodEnd)
		case "approvals":
			evidence = c.getApprovalEvidence(ctx, tenantID, control, periodStart, periodEnd)
		case "audit_logs":
			evidence = c.getAuditLogEvidence(ctx, tenantID, control, periodStart, periodEnd)
		case "session_analytics":
			evidence = c.getSessionAnalyticsEvidence(ctx, tenantID, control, periodStart, periodEnd)
		case "session_recordings":
			evidence = c.getSessionRecordingEvidence(ctx, tenantID, control, periodStart, periodEnd)
		case "anomaly_detections":
			evidence = c.getAnomalyDetectionEvidence(ctx, tenantID, control, periodStart, periodEnd)
		case "mfa_logs":
			evidence = c.getMFALogsEvidence(ctx, tenantID, control, periodStart, periodEnd)
		}

		evidenceList = append(evidenceList, evidence...)
	}

	return evidenceList, nil
}

// getAccessRequestEvidence retrieves access request evidence
func (c *EvidenceCollector) getAccessRequestEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	query := `
		SELECT
			id,
			tenant_id,
			'text' AS type,
			'Access Request' AS category,
			CONCAT('Access Request: ', justification) AS title,
			justification AS description,
			jsonb_build_object(
				'control_id', $1::text,
				'control_name', $2::text,
				'request_id', id::text,
				'status', status,
				'requested_at', requested_at,
				'user_id', requester_id
			) AS evidence_data,
			'access_requests' AS source_type,
			requested_at AS applicable_from,
			DATE_ADD(requested_at, INTERVAL '1 year') AS applicable_until,
			'active' AS status
		FROM access_requests
		WHERE tenant_id = $3
			AND requested_at >= $4
			AND requested_at <= $5
			AND deleted_at IS NULL
		LIMIT 100
	`

	var evidence []Evidence
	err := c.db.SelectContext(ctx, &evidence, query,
		control.ControlID, control.ControlName, tenantID, periodStart, periodEnd,
	)

	if err != nil {
		c.logger.Error().Err(err).
			Str("control_id", control.ControlID).
			Msg("Failed to get access request evidence")
	}

	// Unmarshal JSON
	for i := range evidence {
		json.Unmarshal([]byte(evidence[i].EvidenceData), &evidence[i].EvidenceData)
	}

	return evidence
}

// getApprovalEvidence retrieves approval evidence
func (c *EvidenceCollector) getApprovalEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	// Query approval chain from access_requests
	query := `
		SELECT
			id,
			tenant_id,
			'text' AS type,
			'Approval' AS category,
			CONCAT('Approval: ', jsonb_array_elements(approval_chain)->>'approver_name') AS title,
			CONCAT('Approved at: ', approved_at::text) AS description,
			jsonb_build_object(
				'control_id', $1::text,
				'control_name', $2::text,
				'request_id', id::text,
				'approver', jsonb_array_elements(approval_chain)->>'approver_name',
				'approved_at', approved_at
			) AS evidence_data,
			'access_requests' AS source_type,
			approved_at AS applicable_from,
			DATE_ADD(approved_at, INTERVAL '1 year') AS applicable_until,
			'active' AS status
		FROM access_requests
		WHERE tenant_id = $3
			AND status = 'approved'
			AND approved_at >= $4
			AND approved_at <= $5
			AND deleted_at IS NULL
		LIMIT 100
	`

	var evidence []Evidence
	err := c.db.SelectContext(ctx, &evidence, query,
		control.ControlID, control.ControlName, tenantID, periodStart, periodEnd,
	)

	if err != nil {
		c.logger.Error().Err(err).
			Str("control_id", control.ControlID).
			Msg("Failed to get approval evidence")
	}

	// Unmarshal JSON
	for i := range evidence {
		json.Unmarshal([]byte(evidence[i].EvidenceData), &evidence[i].EvidenceData)
	}

	return evidence
}

// getAuditLogEvidence retrieves audit log evidence
func (c *EvidenceCollector) getAuditLogEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	// This would query the audit service's logs
	return []Evidence{}
}

// getSessionAnalyticsEvidence retrieves session analytics evidence
func (c *EvidenceCollector) getSessionAnalyticsEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	query := `
		SELECT
			gen_random_uuid() as id,
			tenant_id,
			'text' as type,
			'Session Analytics' as category,
			CONCAT('Session Analytics for ', DATE_TRUNC('day', date)::text) as title,
			CONCAT('Total sessions: ', total_sessions) as description,
			jsonb_build_object(
				'control_id', $1::text,
				'control_name', $2::text,
				'date', date,
				'total_sessions', total_sessions,
				'unique_users', unique_users,
				'ssh_sessions', ssh_sessions,
				'rdp_sessions', rdp_sessions
			) as evidence_data,
			'session_analytics' as source_type,
			date as applicable_from,
			DATE_ADD(date, INTERVAL '1 day') as applicable_until,
			'active' as status
		FROM session_analytics
		WHERE tenant_id = $3
			AND date >= $4
			AND date <= $5
		LIMIT 50
	`

	var evidence []Evidence
	err := c.db.SelectContext(ctx, &evidence, query,
		control.ControlID, control.ControlName, tenantID, periodStart, periodEnd,
	)

	if err != nil {
		c.logger.Error().Err(err).
			Str("control_id", control.ControlID).
			Msg("Failed to get session analytics evidence")
	}

	// Unmarshal JSON
	for i := range evidence {
		json.Unmarshal([]byte(evidence[i].EvidenceData), &evidence[i].EvidenceData)
	}

	return evidence
}

// getSessionRecordingEvidence retrieves session recording evidence
func (c *EvidenceCollector) getSessionRecordingEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	// This would reference stored session recordings
	return []Evidence{}
}

// getAnomalyDetectionEvidence retrieves anomaly detection evidence
func (c *EvidenceCollector) getAnomalyDetectionEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	query := `
		SELECT
			id,
			tenant_id,
			'text' as type,
			'Anomaly Detection' as category,
			title,
			description,
			jsonb_build_object(
				'control_id', $1::text,
				'control_name', $2::text,
				'anomaly_type', anomaly_type,
				'severity', severity,
				'confidence_score', confidence_score,
				'detected_at', detected_at
			) as evidence_data,
			'anomaly_detections' as source_type,
			detected_at as applicable_from,
			DATE_ADD(detected_at, INTERVAL '1 year') as applicable_until,
			status
		FROM anomaly_detections
		WHERE tenant_id = $3
			AND detected_at >= $4
			AND detected_at <= $5
			AND deleted_at IS NULL
		LIMIT 50
	`

	var evidence []Evidence
	err := c.db.SelectContext(ctx, &evidence, query,
		control.ControlID, control.ControlName, tenantID, periodStart, periodEnd,
	)

	if err != nil {
		c.logger.Error().Err(err).
			Str("control_id", control.ControlID).
			Msg("Failed to get anomaly detection evidence")
	}

	// Unmarshal JSON
	for i := range evidence {
		var rawData []byte
		if evidence[i].EvidenceData != nil {
			rawData, _ = json.Marshal(evidence[i].EvidenceData)
			evidence[i].EvidenceData = make(map[string]interface{})
			json.Unmarshal(rawData, &evidence[i].EvidenceData)
		}
	}

	return evidence
}

// getMFALogsEvidence retrieves MFA logs evidence
func (c *EvidenceCollector) getMFALogsEvidence(ctx context.Context, tenantID uuid.UUID, control ControlMapping, periodStart, periodEnd time.Time) []Evidence {
	// This would query MFA verification logs
	return []Evidence{}
}

// SaveEvidence saves evidence to the database
func (c *EvidenceCollector) SaveEvidence(ctx context.Context, evidence *Evidence) error {
	evidence.ID = uuid.New()
	evidence.CreatedAt = time.Now()
	evidence.UpdatedAt = time.Now()

	evidenceData, _ := json.Marshal(evidence.EvidenceData)
	tagsJSON, _ := json.Marshal(evidence.Tags)
	metadataJSON, _ := json.Marshal(evidence.Metadata)

	query := `
		INSERT INTO evidence (
			id, tenant_id, report_id, type, category, title, description,
			evidence_data, file_path, file_hash,
			applicable_from, applicable_until,
			source_type, source_id,
			status, verified_by, verified_at,
			tags, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12,
			$13, $14, $15,
			$16, $17, $18, $19
		)
	`

	_, err := c.db.ExecContext(ctx, query,
		evidence.ID, evidence.TenantID, evidence.ReportID, evidence.Type, evidence.Category,
		evidence.Title, evidence.Description, evidenceData, evidence.FilePath,
		evidence.FileHash, evidence.ApplicableFrom, evidence.ApplicableUntil,
		evidence.SourceType, evidence.SourceID, evidence.Status,
		evidence.VerifiedBy, evidence.VerifiedAt, tagsJSON, metadataJSON,
		evidence.CreatedAt, evidence.UpdatedAt,
	)

	return err
}

// GetEvidence retrieves evidence by ID
func (c *EvidenceCollector) GetEvidence(ctx context.Context, id uuid.UUID) (*Evidence, error) {
	query := `
		SELECT * FROM evidence
		WHERE id = $1 AND deleted_at IS NULL
	`

	var evidence Evidence
	err := c.db.GetContext(ctx, &evidence, query, id)
	if err != nil {
		return nil, fmt.Errorf("evidence_collector.GetEvidence: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(evidence.EvidenceData), &evidence.EvidenceData)
	json.Unmarshal([]byte(evidence.Metadata), &evidence.Metadata)

	return &evidence, nil
}

// GetEvidenceForControl retrieves all evidence for a control
func (c *EvidenceCollector) GetEvidenceForControl(ctx context.Context, tenantID uuid.UUID, controlID string) ([]Evidence, error) {
	query := `
		SELECT * FROM evidence
		WHERE tenant_id = $1
		AND evidence_data->>'control_id' = $2
		AND deleted_at IS NULL
		AND applicable_from <= CURRENT_DATE
		AND (applicable_until IS NULL OR applicable_until > CURRENT_DATE)
		ORDER BY applicable_from DESC
	`

	var evidence []Evidence
	err := c.db.SelectContext(ctx, &evidence, query, tenantID, controlID)
	if err != nil {
		return nil, fmt.Errorf("evidence_collector.GetEvidenceForControl: %w", err)
	}

	// Unmarshal JSON fields
	for i := range evidence {
		json.Unmarshal([]byte(evidence.EvidenceData), &evidence[i].EvidenceData)
		json.Unmarshal([]byte(evidence[i].Metadata), &evidence[i].Metadata)
	}

	return evidence, nil
}

// GenerateEvidenceReport generates a summary evidence report for controls
func (c *EvidenceCollector) GenerateEvidenceReport(ctx context.Context, tenantID uuid.UUID, framework analytics.ComplianceFramework, periodStart, periodEnd time.Time) (*EvidenceReport, error) {
	report := &EvidenceReport{
		TenantID:    tenantID,
		Framework:   framework,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		GeneratedAt: time.Now(),
	}

	// Get controls for framework
	templateManager := NewTemplateManager(c.db, c.logger)
	template, err := templateManager.GetFrameworkTemplate(ctx, framework)
	if err != nil {
		return nil, err
	}

	controls := template.TemplateData["controls"].([]ControlMapping)

	// Collect evidence for each control
	controlEvidence := make(map[string][]Evidence)
	for _, control := range controls {
		evidence, err := c.CollectPeriodEvidence(ctx, tenantID, framework, periodStart, periodEnd, []ControlMapping{control})
		if err != nil {
			c.logger.Warn().Err(err).
				Str("control_id", control.ControlID).
				Msg("Failed to collect evidence for control")
			continue
		}
		controlEvidence[control.ControlID] = evidence
	}

	report.ControlEvidence = controlEvidence
	report.TotalControls = len(controls)
	report.ControlsWithEvidence = len(controlEvidence)

	return report, nil
}

// EvidenceReport represents a summary of collected evidence
type EvidenceReport struct {
	TenantID            uuid.UUID                    `json:"tenant_id"`
	Framework           analytics.ComplianceFramework `json:"framework"`
	PeriodStart         time.Time                    `json:"period_start"`
	PeriodEnd           time.Time                    `json:"period_end"`
	GeneratedAt         time.Time                    `json:"generated_at"`
	TotalControls       int                          `json:"total_controls"`
	ControlsWithEvidence int                          `json:"controls_with_evidence"`
	ControlEvidence     map[string][]Evidence        `json:"control_evidence"`
}
