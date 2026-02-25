// Package anomalies provides persistence layer for anomaly detections
package anomalies

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// Store handles anomaly persistence and queries
type Store struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewStore creates a new anomaly store
func NewStore(db *sqlx.DB, logger zerolog.Logger) *Store {
	return &Store{
		db:     db,
		logger: logger,
	}
}

// Create creates a new anomaly detection
func (s *Store) Create(ctx context.Context, anomaly *model.AnomalyDetection) error {
	anomaly.ID = uuid.New()
	anomaly.CreatedAt = time.Now()
	anomaly.UpdatedAt = time.Now()
	anomaly.DetectedAt = time.Now()

	if anomaly.Status == "" {
		anomaly.Status = string(model.AnomalyStatusOpen)
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

	_, err := s.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("anomalies.Create: %w", err)
	}

	s.logger.Info().
		Str("anomaly_id", anomaly.ID.String()).
		Str("tenant_id", anomaly.TenantID.String()).
		Str("anomaly_type", anomaly.AnomalyType).
		Str("severity", anomaly.Severity).
		Float64("risk_score", anomaly.RiskScore).
		Msg("Anomaly detection created")

	return nil
}

// CreateBatch creates multiple anomalies
func (s *Store) CreateBatch(ctx context.Context, anomalies []model.AnomalyDetection) error {
	if len(anomalies) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("anomalies.CreateBatch: begin transaction: %w", err)
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

		if anomalies[i].DuplicateCount == 0 {
			anomalies[i].DuplicateCount = 0
		}
		if !anomalies[i].IsDuplicate {
			anomalies[i].IsDuplicate = false
		}

		_, err := tx.NamedExec(query, &anomalies[i])
		if err != nil {
			return fmt.Errorf("anomalies.CreateBatch: insert anomaly %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("anomalies.CreateBatch: commit: %w", err)
	}

	s.logger.Info().
		Int("count", len(anomalies)).
		Msg("Batch anomalies created")

	return nil
}

// GetByID retrieves an anomaly by ID
func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (*model.AnomalyDetection, error) {
	var anomaly model.AnomalyDetection
	query := `SELECT * FROM anomaly_detections WHERE id = $1`

	err := s.db.GetContext(ctx, &anomaly, query, id)
	if err != nil {
		return nil, fmt.Errorf("anomalies.GetByID: %w", err)
	}

	return &anomaly, nil
}

// List retrieves anomalies with filtering
func (s *Store) List(ctx context.Context, filter *Filter, limit, offset int) ([]model.AnomalyDetection, int, error) {
	baseQuery := `SELECT * FROM anomaly_detections WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM anomaly_detections WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
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

	if !filter.IncludeDuplicates {
		baseQuery += " AND is_duplicate = false"
		countQuery += " AND is_duplicate = false"
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

	// Get total count
	var total int
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("anomalies.List.Count: %w", err)
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
	if err := s.db.SelectContext(ctx, &anomalies, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("anomalies.List: %w", err)
	}

	return anomalies, total, nil
}

// Update updates an anomaly
func (s *Store) Update(ctx context.Context, anomaly *model.AnomalyDetection) error {
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

	_, err := s.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("anomalies.Update: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of an anomaly
func (s *Store) UpdateStatus(ctx context.Context, id uuid.UUID, status string, assignedTo *uuid.UUID, resolutionNotes *string, resolvedBy *uuid.UUID) error {
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

	_, err := s.db.ExecContext(ctx, query, status, assignedTo, resolutionNotes, isResolved, now, resolvedBy, now, id)
	if err != nil {
		return fmt.Errorf("anomalies.UpdateStatus: %w", err)
	}

	return nil
}

// Delete deletes an anomaly
func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM anomaly_detections WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("anomalies.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("anomalies.Delete: no rows affected")
	}

	return nil
}

// GetOpen retrieves all open anomalies for a tenant
func (s *Store) GetOpen(ctx context.Context, tenantID uuid.UUID, severity *string) ([]model.AnomalyDetection, error) {
	var anomalies []model.AnomalyDetection
	query := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND status = 'open' AND is_duplicate = false
	`

	args := []interface{}{tenantID}
	argCount := 2

	if severity != nil {
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *severity)
		argCount++
	}

	query += " ORDER BY risk_score DESC, detected_at DESC"

	err := s.db.SelectContext(ctx, &anomalies, query, args...)
	if err != nil {
		return nil, fmt.Errorf("anomalies.GetOpen: %w", err)
	}

	return anomalies, nil
}

// GetStats retrieves statistics about anomalies
func (s *Store) GetStats(ctx context.Context, tenantID uuid.UUID) (*AnomalyStats, error) {
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
			COALESCE(AVG(risk_score) FILTER (WHERE is_duplicate = false), 0) as avg_risk_score
		FROM anomaly_detections
		WHERE tenant_id = $1
	`

	var stats AnomalyStats
	err := s.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("anomalies.GetStats: %w", err)
	}

	return &stats, nil
}

// GetByUserID retrieves anomalies for a specific user
func (s *Store) GetByUserID(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]model.AnomalyDetection, error) {
	var anomalies []model.AnomalyDetection
	query := `
		SELECT * FROM anomaly_detections
		WHERE tenant_id = $1 AND user_id = $2 AND is_duplicate = false
		ORDER BY detected_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	err := s.db.SelectContext(ctx, &anomalies, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("anomalies.GetByUserID: %w", err)
	}

	return anomalies, nil
}

// MergeDuplicateAnomalies marks all anomalies with the same correlation key as duplicates
func (s *Store) MergeDuplicateAnomalies(ctx context.Context, anomalyID uuid.UUID) (int, error) {
	query := `SELECT merge_duplicate_anomalies($1)`

	var result int
	err := s.db.GetContext(ctx, &result, query, anomalyID)
	if err != nil {
		return 0, fmt.Errorf("anomalies.MergeDuplicateAnomalies: %w", err)
	}

	return result, nil
}

// GetAnomalyTypes retrieves unique anomaly types
func (s *Store) GetAnomalyTypes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT anomaly_type
		FROM anomaly_detections
		WHERE tenant_id = $1 AND is_duplicate = false
		ORDER BY anomaly_type
	`

	var types []string
	err := s.db.SelectContext(ctx, &types, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("anomalies.GetAnomalyTypes: %w", err)
	}

	return types, nil
}

// GetTopUsersByAnomalyCount retrieves users with the most anomalies
func (s *Store) GetTopUsersByAnomalyCount(ctx context.Context, tenantID uuid.UUID, limit int, dateFrom, dateTo *time.Time) ([]UserAnomalyCount, error) {
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
	err := s.db.SelectContext(ctx, &results, query, args...)
	if err != nil {
		return nil, fmt.Errorf("anomalies.GetTopUsersByAnomalyCount: %w", err)
	}

	return results, nil
}

// Filter represents filtering options
type Filter struct {
	TenantID          *uuid.UUID
	UserID            *uuid.UUID
	AnomalyType       *string
	Severity          *string
	Status            *string
	IncludeDuplicates bool
	DateFrom          *time.Time
	DateTo            *time.Time
	AssignedTo        *uuid.UUID
	CorrelationID     *uuid.UUID
	Search            string
}

// AnomalyStats represents statistics
type AnomalyStats struct {
	Total              int     `db:"total" json:"total"`
	OpenCount          int     `db:"open_count" json:"open_count"`
	InvestigatingCount int     `db:"investigating_count" json:"investigating_count"`
	ResolvedCount      int     `db:"resolved_count" json:"resolved_count"`
	CriticalCount      int     `db:"critical_count" json:"critical_count"`
	HighCount          int     `db:"high_count" json:"high_count"`
	MediumCount        int     `db:"medium_count" json:"medium_count"`
	LowCount           int     `db:"low_count" json:"low_count"`
	TodayCount         int     `db:"today_count" json:"today_count"`
	WeekCount          int     `db:"week_count" json:"week_count"`
	AvgRiskScore       float64 `db:"avg_risk_score" json:"avg_risk_score"`
}

// UserAnomalyCount represents anomaly count per user
type UserAnomalyCount struct {
	UserID       uuid.UUID `db:"user_id" json:"user_id"`
	AnomalyCount int       `db:"anomaly_count" json:"anomaly_count"`
	MaxRiskScore float64   `db:"max_risk_score" json:"max_risk_score"`
}
