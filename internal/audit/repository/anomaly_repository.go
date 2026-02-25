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

// AnomalyRepository handles anomaly detection tracking
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

// Create creates a new anomaly detection
func (r *AnomalyRepository) Create(ctx context.Context, anomaly *model.AnomalyDetection) error {
	anomaly.ID = uuid.New()
	anomaly.CreatedAt = time.Now()
	anomaly.UpdatedAt = time.Now()
	anomaly.DetectedAt = time.Now()

	// Set default status if not provided
	if anomaly.Status == "" {
		anomaly.Status = string(model.AnomalyStatusOpen)
	}

	// Set default deduplication values
	if anomaly.DuplicateCount == 0 {
		anomaly.DuplicateCount = 0
	}
	if !anomaly.IsDuplicate {
		anomaly.IsDuplicate = false
	}

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

	_, err := r.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("anomaly.Create: %w", err)
	}

	r.logger.Info().
		Str("anomaly_id", anomaly.ID.String()).
		Str("tenant_id", anomaly.TenantID.String()).
		Str("anomaly_type", anomaly.AnomalyType).
		Str("severity", anomaly.Severity).
		Float64("risk_score", anomaly.RiskScore).
		Msg("Anomaly detection created")

	return nil
}

// GetByID retrieves an anomaly by ID
func (r *AnomalyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AnomalyDetection, error) {
	var anomaly model.AnomalyDetection
	query := `SELECT * FROM anomaly_detections WHERE id = $1`

	err := r.db.GetContext(ctx, &anomaly, query, id)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetByID: %w", err)
	}

	return &anomaly, nil
}

// List retrieves anomalies with filtering and pagination
func (r *AnomalyRepository) List(ctx context.Context, tenantID uuid.UUID, filter model.AnomalyFilter, limit, offset int) ([]model.AnomalyDetection, int, error) {
	// By default, exclude duplicates unless explicitly requested
	includeDuplicates := filter.IsDuplicate != nil && *filter.IsDuplicate

	baseQuery := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM anomaly_detections WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if !includeDuplicates {
		baseQuery += " AND is_duplicate = false"
		countQuery += " AND is_duplicate = false"
	}

	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}

	if filter.AnomalyType != nil {
		baseQuery += fmt.Sprintf(" AND anomaly_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND anomaly_type = $%d", argCount)
		args = append(args, *filter.AnomalyType)
		argCount++
	}

	if filter.Severity != nil {
		baseQuery += fmt.Sprintf(" AND severity = $%d", argCount)
		countQuery += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *filter.Severity)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
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

	if filter.AssignedTo != nil {
		baseQuery += fmt.Sprintf(" AND assigned_to = $%d", argCount)
		countQuery += fmt.Sprintf(" AND assigned_to = $%d", argCount)
		args = append(args, *filter.AssignedTo)
		argCount++
	}

	if filter.CorrelationID != nil {
		baseQuery += fmt.Sprintf(" AND correlation_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND correlation_id = $%d", argCount)
		args = append(args, *filter.CorrelationID)
		argCount++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argCount += 2
	}

	// Get total count (need to use all filter args for count)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("anomaly.List.Count: %w", err)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY detected_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var anomalies []model.AnomalyDetection
	if err := r.db.SelectContext(ctx, &anomalies, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("anomaly.List: %w", err)
	}

	return anomalies, total, nil
}

// Update updates an anomaly
func (r *AnomalyRepository) Update(ctx context.Context, anomaly *model.AnomalyDetection) error {
	anomaly.UpdatedAt = time.Now()

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

	_, err := r.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("anomaly.Update: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of an anomaly
func (r *AnomalyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, assignedTo *uuid.UUID, resolutionNotes *string, resolvedBy *uuid.UUID) error {
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

	isResolved := status == string(model.AnomalyStatusResolved)

	_, err := r.db.ExecContext(ctx, query, status, assignedTo, resolutionNotes, isResolved, now, resolvedBy, now, id)
	if err != nil {
		return fmt.Errorf("anomaly.UpdateStatus: %w", err)
	}

	return nil
}

// Delete deletes an anomaly
func (r *AnomalyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM anomaly_detections WHERE id = $1`

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
func (r *AnomalyRepository) GetByUserID(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]model.AnomalyDetection, error) {
	var anomalies []model.AnomalyDetection
	query := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY detected_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	err := r.db.SelectContext(ctx, &anomalies, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetByUserID: %w", err)
	}

	return anomalies, nil
}

// GetBySessionID retrieves anomalies for a specific session
func (r *AnomalyRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.AnomalyDetection, error) {
	var anomalies []model.AnomalyDetection
	query := `
		SELECT * FROM anomaly_detections
		WHERE session_id = $1
		ORDER BY detected_at DESC
	`

	err := r.db.SelectContext(ctx, &anomalies, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetBySessionID: %w", err)
	}

	return anomalies, nil
}

// GetOpen retrieves all open anomalies for a tenant
func (r *AnomalyRepository) GetOpen(ctx context.Context, tenantID uuid.UUID, severity *string) ([]model.AnomalyDetection, error) {
	var anomalies []model.AnomalyDetection
	query := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND status = 'open'
	`

	args := []interface{}{tenantID}
	argCount := 2

	if severity != nil {
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *severity)
		argCount++
	}

	query += " ORDER BY risk_score DESC, detected_at DESC"

	err := r.db.SelectContext(ctx, &anomalies, query, args...)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetOpen: %w", err)
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

// AnomalyStats represents statistics about anomalies
type AnomalyStats struct {
	Total              int     `db:"total"`
	OpenCount          int     `db:"open_count"`
	InvestigatingCount int     `db:"investigating_count"`
	ResolvedCount      int     `db:"resolved_count"`
	CriticalCount      int     `db:"critical_count"`
	HighCount          int     `db:"high_count"`
	MediumCount        int     `db:"medium_count"`
	LowCount           int     `db:"low_count"`
	TodayCount         int     `db:"today_count"`
	WeekCount          int     `db:"week_count"`
	UniqueCorrelations int     `db:"unique_correlations"`
	TotalDuplicates    int     `db:"total_duplicates"`
	AvgRiskScore       float64 `db:"avg_risk_score"`
}

// BatchCreate creates multiple anomalies in a single transaction
func (r *AnomalyRepository) BatchCreate(ctx context.Context, anomalies []model.AnomalyDetection) error {
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
			anomalies[i].Status = string(model.AnomalyStatusOpen)
		}

		// Set default deduplication values
		if anomalies[i].DuplicateCount == 0 {
			anomalies[i].DuplicateCount = 0
		}
		if !anomalies[i].IsDuplicate {
			anomalies[i].IsDuplicate = false
		}

		_, err := tx.NamedExec(query, &anomalies[i])
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
func (r *AnomalyRepository) GetByCorrelationID(ctx context.Context, correlationID uuid.UUID) ([]model.AnomalyDetection, error) {
	var anomalies []model.AnomalyDetection
	query := `
		SELECT * FROM anomaly_detections
		WHERE correlation_id = $1
		ORDER BY detected_at ASC
	`

	err := r.db.SelectContext(ctx, &anomalies, query, correlationID)
	if err != nil {
		return nil, fmt.Errorf("anomaly.GetByCorrelationID: %w", err)
	}

	return anomalies, nil
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

// MergeDuplicateAnomalies marks all anomalies with the same correlation key as duplicates
func (r *AnomalyRepository) MergeDuplicateAnomalies(ctx context.Context, anomalyID uuid.UUID) (int, error) {
	// This relies on the database trigger function merge_duplicate_anomalies
	// defined in migration 003
	query := `SELECT merge_duplicate_anomalies($1)`

	var result int
	err := r.db.GetContext(ctx, &result, query, anomalyID)
	if err != nil {
		return 0, fmt.Errorf("anomaly.MergeDuplicateAnomalies: %w", err)
	}

	return result, nil
}
