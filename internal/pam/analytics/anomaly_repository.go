// Package analytics provides anomaly persistence integration with audit service
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// AnomalyRepository bridges PAM analytics with audit anomaly persistence
// This repository integrates with the audit service's anomaly_detections table
type AnomalyRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewAnomalyRepository creates a new anomaly repository
func NewAnomalyRepository(db *sqlx.DB, logger zerolog.Logger) *AnomalyRepository {
	return &AnomalyRepository{
		db:     db,
		logger: logger,
	}
}

// Create persists a new anomaly detection
func (r *AnomalyRepository) Create(ctx context.Context, anomaly *Anomaly) error {
	now := time.Now()
	anomaly.ID = uuid.New()
	anomaly.CreatedAt = now
	anomaly.UpdatedAt = now
	anomaly.DetectedAt = now

	// Set default status if not provided
	if anomaly.Status == "" {
		anomaly.Status = AnomalyStatusOpen
	}

	// Ensure correlation ID is set
	if anomaly.CorrelationID == nil {
		correlationID := anomaly.ID
		anomaly.CorrelationID = &correlationID
	}

	// Convert to audit model
	auditAnomaly := r.toAuditModel(anomaly)

	query := `
		INSERT INTO anomaly_detections (
			id, tenant_id, anomaly_type, user_id, session_id, target_host,
			severity, confidence_score, risk_score, title, description,
			indicators, detection_method, detected_at, model_version, status,
			assigned_to, resolution_notes, resolved_at, resolved_by,
			auto_triggered, auto_action_taken,
			correlation_id, correlation_key, duplicate_count, is_duplicate,
			first_detection_id, merged_into_id,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :anomaly_type, :user_id, :session_id, :target_host,
			:severity, :confidence_score, :risk_score, :title, :description,
			:indicators, :detection_method, :detected_at, :model_version, :status,
			:assigned_to, :resolution_notes, :resolved_at, :resolved_by,
			:auto_triggered, :auto_action_taken,
			:correlation_id, :correlation_key, :duplicate_count, :is_duplicate,
			:first_detection_id, :merged_into_id,
			:metadata, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, auditAnomaly)
	if err != nil {
		return fmt.Errorf("anomaly.Create: %w", err)
	}

	r.logger.Info().
		Str("anomaly_id", anomaly.ID.String()).
		Str("tenant_id", anomaly.TenantID.String()).
		Str("anomaly_type", string(anomaly.AnomalyType)).
		Str("severity", string(anomaly.Severity)).
		Float64("risk_score", anomaly.RiskScore).
		Msg("Anomaly detection created")

	return nil
}

// GetByID retrieves an anomaly by ID
func (r *AnomalyRepository) GetByID(ctx context.Context, id uuid.UUID) (*Anomaly, error) {
	var auditAnomaly model.AnomalyDetection
	query := `SELECT * FROM anomaly_detections WHERE id = $1`

	err := r.db.GetContext(ctx, &auditAnomaly, query, id)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetByID: %w", err)
	}

	return r.fromAuditModel(&auditAnomaly), nil
}

// List retrieves anomalies with filtering and pagination
func (r *AnomalyRepository) List(ctx context.Context, tenantID uuid.UUID, filter AnomalyFilter, limit, offset int) ([]Anomaly, int, error) {
	baseQuery := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND is_duplicate = false
	`
	countQuery := `
		SELECT COUNT(*) FROM anomaly_detections
		WHERE tenant_id = $1 AND is_duplicate = false
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}

	if filter.AnomalyType != nil {
		baseQuery += fmt.Sprintf(" AND anomaly_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND anomaly_type = $%d", argCount)
		args = append(args, string(*filter.AnomalyType))
		argCount++
	}

	if filter.Severity != nil {
		baseQuery += fmt.Sprintf(" AND severity = $%d", argCount)
		countQuery += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, string(*filter.Severity))
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, string(*filter.Status))
		argCount++
	}

	if filter.AssignedTo != nil {
		baseQuery += fmt.Sprintf(" AND assigned_to = $%d", argCount)
		countQuery += fmt.Sprintf(" AND assigned_to = $%d", argCount)
		args = append(args, *filter.AssignedTo)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND detected_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND detected_at >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND detected_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND detected_at <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argCount += 2
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args[:1]...); err != nil {
		return nil, 0, fmt.Errorf("anomaly.List.Count: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY detected_at DESC, risk_score DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var auditAnomalies []model.AnomalyDetection
	if err := r.db.SelectContext(ctx, &auditAnomalies, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("anomaly.List: %w", err)
	}

	anomalies := make([]Anomaly, len(auditAnomalies))
	for i, aa := range auditAnomalies {
		anomalies[i] = *r.fromAuditModel(&aa)
	}

	return anomalies, total, nil
}

// Update updates an existing anomaly
func (r *AnomalyRepository) Update(ctx context.Context, anomaly *Anomaly) error {
	anomaly.UpdatedAt = time.Now()

	auditAnomaly := r.toAuditModel(anomaly)

	query := `
		UPDATE anomaly_detections SET
			status = :status,
			assigned_to = :assigned_to,
			resolution_notes = :resolution_notes,
			resolved_at = :resolved_at,
			resolved_by = :resolved_by,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, auditAnomaly)
	if err != nil {
		return fmt.Errorf("anomaly.Update: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of an anomaly
func (r *AnomalyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status AnomalyStatus, assignedTo, resolvedBy *uuid.UUID, resolutionNotes *string) error {
	now := time.Now()
	query := `
		UPDATE anomaly_detections SET
			status = $1,
			assigned_to = $2,
			resolution_notes = $3,
			resolved_at = CASE WHEN $4 THEN $5 ELSE resolved_at END,
			resolved_by = CASE WHEN $4 THEN $6 ELSE resolved_by END,
			updated_at = $7
		WHERE id = $8
	`

	isResolved := status == AnomalyStatusResolved

	_, err := r.db.ExecContext(ctx, query, string(status), assignedTo, resolutionNotes, isResolved, now, resolvedBy, now, id)
	if err != nil {
		return fmt.Errorf("anomaly.UpdateStatus: %w", err)
	}

	return nil
}

// Delete soft-deletes an anomaly by marking as ignored
func (r *AnomalyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE anomaly_detections SET status = 'ignored', updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("anomaly.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("anomaly.Delete: no rows affected")
	}

	return nil
}

// GetByUserID retrieves anomalies for a specific user
func (r *AnomalyRepository) GetByUserID(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]Anomaly, error) {
	query := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND user_id = $2 AND is_duplicate = false
		ORDER BY detected_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var auditAnomalies []model.AnomalyDetection
	err := r.db.SelectContext(ctx, &auditAnomalies, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetByUserID: %w", err)
	}

	anomalies := make([]Anomaly, len(auditAnomalies))
	for i, aa := range auditAnomalies {
		anomalies[i] = *r.fromAuditModel(&aa)
	}

	return anomalies, nil
}

// GetBySessionID retrieves anomalies for a specific session
func (r *AnomalyRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]Anomaly, error) {
	query := `
		SELECT * FROM anomaly_detections
		WHERE session_id = $1 AND is_duplicate = false
		ORDER BY detected_at DESC
	`

	var auditAnomalies []model.AnomalyDetection
	err := r.db.SelectContext(ctx, &auditAnomalies, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetBySessionID: %w", err)
	}

	anomalies := make([]Anomaly, len(auditAnomalies))
	for i, aa := range auditAnomalies {
		anomalies[i] = *r.fromAuditModel(&aa)
	}

	return anomalies, nil
}

// GetOpen retrieves all open anomalies for a tenant
func (r *AnomalyRepository) GetOpen(ctx context.Context, tenantID uuid.UUID, severity *Severity) ([]Anomaly, error) {
	query := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND status = 'open' AND is_duplicate = false
	`

	args := []interface{}{tenantID}
	argCount := 2

	if severity != nil {
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, string(*severity))
		argCount++
	}

	query += " ORDER BY risk_score DESC, detected_at DESC"

	var auditAnomalies []model.AnomalyDetection
	err := r.db.SelectContext(ctx, &auditAnomalies, query, args...)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetOpen: %w", err)
	}

	anomalies := make([]Anomaly, len(auditAnomalies))
	for i, aa := range auditAnomalies {
		anomalies[i] = *r.fromAuditModel(&aa)
	}

	return anomalies, nil
}

// GetStats retrieves statistics about anomalies for a tenant
func (r *AnomalyRepository) GetStats(ctx context.Context, tenantID uuid.UUID) (*AnomalyStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE is_duplicate = false) as total,
			COUNT(*) FILTER (WHERE status = 'open' AND is_duplicate = false) as open_count,
			COUNT(*) FILTER (WHERE status = 'investigating' AND is_duplicate = false) as investigating_count,
			COUNT(*) FILTER (WHERE status = 'resolved' AND is_duplicate = false) as resolved_count,
			COUNT(*) FILTER (WHERE severity = 'critical' AND is_duplicate = false) as critical_count,
			COUNT(*) FILTER (WHERE severity = 'high' AND is_duplicate = false) as high_count,
			COUNT(*) FILTER (WHERE severity = 'medium' AND is_duplicate = false) as medium_count,
			COUNT(*) FILTER (WHERE severity = 'low' AND is_duplicate = false) as low_count,
			COUNT(*) FILTER (WHERE detected_at >= CURRENT_DATE AND is_duplicate = false) as today_count,
			COUNT(*) FILTER (WHERE detected_at >= CURRENT_DATE - INTERVAL '7 days' AND is_duplicate = false) as week_count,
			COUNT(DISTINCT correlation_id) FILTER (WHERE is_duplicate = false) as unique_correlations,
			COALESCE(SUM(duplicate_count), 0) as total_duplicates,
			COALESCE(AVG(risk_score) FILTER (WHERE is_duplicate = false), 0) as avg_risk_score
		FROM anomaly_detections
		WHERE tenant_id = $1
	`

	var stats AnomalyStats
	err := r.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetStats: %w", err)
	}

	return &stats, nil
}

// BatchCreate creates multiple anomalies in a single transaction
func (r *AnomalyRepository) BatchCreate(ctx context.Context, anomalies []Anomaly) error {
	if len(anomalies) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("anomaly.BatchCreate: begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	query := `
		INSERT INTO anomaly_detections (
			id, tenant_id, anomaly_type, user_id, session_id, target_host,
			severity, confidence_score, risk_score, title, description,
			indicators, detection_method, detected_at, model_version, status,
			assigned_to, resolution_notes, resolved_at, resolved_by,
			auto_triggered, auto_action_taken,
			correlation_id, correlation_key, duplicate_count, is_duplicate,
			first_detection_id, merged_into_id,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :anomaly_type, :user_id, :session_id, :target_host,
			:severity, :confidence_score, :risk_score, :title, :description,
			:indicators, :detection_method, :detected_at, :model_version, :status,
			:assigned_to, :resolution_notes, :resolved_at, :resolved_by,
			:auto_triggered, :auto_action_taken,
			:correlation_id, :correlation_key, :duplicate_count, :is_duplicate,
			:first_detection_id, :merged_into_id,
			:metadata, :created_at, :updated_at
		)
	`

	for i := range anomalies {
		anomalies[i].ID = uuid.New()
		anomalies[i].CreatedAt = now
		anomalies[i].UpdatedAt = now
		anomalies[i].DetectedAt = now

		if anomalies[i].Status == "" {
			anomalies[i].Status = AnomalyStatusOpen
		}

		if anomalies[i].CorrelationID == nil {
			correlationID := anomalies[i].ID
			anomalies[i].CorrelationID = &correlationID
		}

		auditAnomaly := r.toAuditModel(&anomalies[i])

		_, err := tx.NamedExec(query, auditAnomaly)
		if err != nil {
			return fmt.Errorf("anomaly.BatchCreate: insert anomaly %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("anomaly.BatchCreate: commit: %w", err)
	}

	r.logger.Info().
		Int("count", len(anomalies)).
		Msg("Batch anomalies created")

	return nil
}

// GetByCorrelationID retrieves all anomalies in a correlation group
func (r *AnomalyRepository) GetByCorrelationID(ctx context.Context, correlationID uuid.UUID) ([]Anomaly, error) {
	query := `
		SELECT * FROM anomaly_detections
		WHERE correlation_id = $1
		ORDER BY detected_at ASC
	`

	var auditAnomalies []model.AnomalyDetection
	err := r.db.SelectContext(ctx, &auditAnomalies, query, correlationID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetByCorrelationID: %w", err)
	}

	anomalies := make([]Anomaly, len(auditAnomalies))
	for i, aa := range auditAnomalies {
		anomalies[i] = *r.fromAuditModel(&aa)
	}

	return anomalies, nil
}

// toAuditModel converts PAM analytics Anomaly to audit model AnomalyDetection
func (r *AnomalyRepository) toAuditModel(anomaly *Anomaly) *model.AnomalyDetection {
	var indicators []byte
	if anomaly.Indicators != nil && len(anomaly.Indicators) > 0 {
		indicators = anomaly.Indicators
	} else {
		indicators = []byte("{}")
	}

	var metadata []byte
	if anomaly.Metadata != nil && len(anomaly.Metadata) > 0 {
		metadata = anomaly.Metadata
	} else {
		metadata = []byte("{}")
	}

	return &model.AnomalyDetection{
		ID:              anomaly.ID,
		TenantID:        anomaly.TenantID,
		AnomalyType:     string(anomaly.AnomalyType),
		UserID:          anomaly.UserID,
		SessionID:       anomaly.SessionID,
		TargetHost:      anomaly.TargetHost,
		Severity:        string(anomaly.Severity),
		ConfidenceScore: anomaly.ConfidenceScore,
		RiskScore:       anomaly.RiskScore,
		Title:           anomaly.Title,
		Description:     anomaly.Description,
		Indicators:      indicators,
		DetectionMethod: anomaly.DetectionMethod,
		DetectedAt:      anomaly.DetectedAt,
		ModelVersion:    anomaly.ModelVersion,
		Status:          string(anomaly.Status),
		AssignedTo:      anomaly.AssignedTo,
		ResolutionNotes: anomaly.ResolutionNotes,
		ResolvedAt:      anomaly.ResolvedAt,
		ResolvedBy:      anomaly.ResolvedBy,
		AutoTriggered:   anomaly.AutoTriggered,
		AutoActionTaken: anomaly.AutoActionTaken,
		CorrelationID:   anomaly.CorrelationID,
		CorrelationKey:  anomaly.CorrelationKey,
		DuplicateCount:  anomaly.DuplicateCount,
		IsDuplicate:     anomaly.IsDuplicate,
		FirstDetectionID: anomaly.FirstDetectionID,
		MergedIntoID:    anomaly.MergedIntoID,
		Metadata:        metadata,
		CreatedAt:       anomaly.CreatedAt,
		UpdatedAt:       anomaly.UpdatedAt,
	}
}

// fromAuditModel converts audit model AnomalyDetection to PAM analytics Anomaly
func (r *AnomalyRepository) fromAuditModel(auditAnomaly *model.AnomalyDetection) *Anomaly {
	return &Anomaly{
		ID:               auditAnomaly.ID,
		TenantID:         auditAnomaly.TenantID,
		AnomalyType:      AnomalyType(auditAnomaly.AnomalyType),
		UserID:           auditAnomaly.UserID,
		SessionID:        auditAnomaly.SessionID,
		TargetHost:       auditAnomaly.TargetHost,
		Severity:         Severity(auditAnomaly.Severity),
		ConfidenceScore:  auditAnomaly.ConfidenceScore,
		RiskScore:        auditAnomaly.RiskScore,
		Title:            auditAnomaly.Title,
		Description:      auditAnomaly.Description,
		Indicators:       auditAnomaly.Indicators,
		DetectionMethod:  auditAnomaly.DetectionMethod,
		DetectedAt:       auditAnomaly.DetectedAt,
		ModelVersion:     auditAnomaly.ModelVersion,
		Status:           AnomalyStatus(auditAnomaly.Status),
		AssignedTo:       auditAnomaly.AssignedTo,
		ResolutionNotes:  auditAnomaly.ResolutionNotes,
		ResolvedAt:       auditAnomaly.ResolvedAt,
		ResolvedBy:       auditAnomaly.ResolvedBy,
		AutoTriggered:    auditAnomaly.AutoTriggered,
		AutoActionTaken:  auditAnomaly.AutoActionTaken,
		CorrelationID:    auditAnomaly.CorrelationID,
		CorrelationKey:   auditAnomaly.CorrelationKey,
		DuplicateCount:   auditAnomaly.DuplicateCount,
		IsDuplicate:      auditAnomaly.IsDuplicate,
		FirstDetectionID: auditAnomaly.FirstDetectionID,
		MergedIntoID:     auditAnomaly.MergedIntoID,
		Metadata:         auditAnomaly.Metadata,
		CreatedAt:        auditAnomaly.CreatedAt,
		UpdatedAt:        auditAnomaly.UpdatedAt,
	}
}

// GetAnomalyTypes retrieves unique anomaly types for a tenant
func (r *AnomalyRepository) GetAnomalyTypes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT anomaly_type
		FROM anomaly_detections
		WHERE tenant_id = $1 AND is_duplicate = false
		ORDER BY anomaly_type
	`

	var types []string
	err := r.db.SelectContext(ctx, &types, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetAnomalyTypes: %w", err)
	}

	return types, nil
}

// GetTopUsersByAnomalyCount retrieves users with the most anomalies
func (r *AnomalyRepository) GetTopUsersByAnomalyCount(ctx context.Context, tenantID uuid.UUID, limit int, dateFrom, dateTo *time.Time) ([]UserAnomalyCount, error) {
	query := `
		SELECT user_id, COUNT(*) as anomaly_count, MAX(risk_score) as max_risk_score
		FROM anomaly_detections
		WHERE tenant_id = $1 AND user_id IS NOT NULL AND is_duplicate = false
	`

	args := []interface{}{tenantID}
	argCount := 2

	if dateFrom != nil {
		query += fmt.Sprintf(" AND detected_at >= $%d", argCount)
		args = append(args, *dateFrom)
		argCount++
	}

	if dateTo != nil {
		query += fmt.Sprintf(" AND detected_at <= $%d", argCount)
		args = append(args, *dateTo)
		argCount++
	}

	query += " GROUP BY user_id ORDER BY anomaly_count DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var results []UserAnomalyCount
	err := r.db.SelectContext(ctx, &results, query, args...)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetTopUsersByAnomalyCount: %w", err)
	}

	return results, nil
}

// UserAnomalyCount represents anomaly count per user
type UserAnomalyCount struct {
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	AnomalyCount int       `db:"anomaly_count" json:"anomaly_count"`
	MaxRiskScore float64   `db:"max_risk_score" json:"max_risk_score"`
}

// CreateFromDetectionRequest creates an anomaly from a detection request
func (r *AnomalyRepository) CreateFromDetectionRequest(ctx context.Context, req *CreateAnomalyRequest, detectedBy uuid.UUID) (*Anomaly, error) {
	anomaly := &Anomaly{
		TenantID:        req.TenantID,
		AnomalyType:     req.AnomalyType,
		UserID:          req.UserID,
		SessionID:       req.SessionID,
		TargetHost:      req.TargetHost,
		Title:           req.Title,
		Description:     req.Description,
		Severity:        req.Severity,
		ConfidenceScore: req.ConfidenceScore,
		RiskScore:       req.RiskScore,
		DetectionMethod: req.DetectionMethod,
		ModelVersion:    req.ModelVersion,
		AutoTriggered:   req.AutoTriggered,
		AutoActionTaken: req.AutoActionTaken,
		Status:          AnomalyStatusOpen,
		DuplicateCount:  0,
		IsDuplicate:     false,
	}

	// Marshal indicators
	if req.Indicators != nil {
		indicatorsJSON, err := json.Marshal(req.Indicators)
		if err == nil {
			anomaly.Indicators = indicatorsJSON
		}
	}

	// Marshal metadata
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err == nil {
			anomaly.Metadata = metadataJSON
		}
	}

	if err := r.Create(ctx, anomaly); err != nil {
		return nil, err
	}

	return anomaly, nil
}
