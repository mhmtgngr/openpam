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

// SSHKeyAnalyticsRepository handles SSH key usage analytics
type SSHKeyAnalyticsRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewSSHKeyAnalyticsRepository creates a new SSH key analytics repository
func NewSSHKeyAnalyticsRepository(db *sqlx.DB, logger zerolog.Logger) *SSHKeyAnalyticsRepository {
	return &SSHKeyAnalyticsRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new SSH key analytics entry
func (r *SSHKeyAnalyticsRepository) Create(ctx context.Context, analytics *model.SSHKeyAnalytics) error {
	analytics.ID = uuid.New()
	analytics.CreatedAt = time.Now()
	analytics.UpdatedAt = time.Now()

	query := `
		INSERT INTO ssh_key_analytics (
			id, tenant_id, ssh_key_id, date, usage_count, unique_users, unique_targets,
			first_use_time, last_use_time, avg_session_duration_seconds,
			off_hours_usage, unusual_source_usage, failed_attempts, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :ssh_key_id, :date, :usage_count, :unique_users, :unique_targets,
			:first_use_time, :last_use_time, :avg_session_duration_seconds,
			:off_hours_usage, :unusual_source_usage, :failed_attempts, :metadata, :created_at, :updated_at
		)
		ON CONFLICT (tenant_id, ssh_key_id, date) DO UPDATE SET
			usage_count = ssh_key_analytics.usage_count + EXCLUDED.usage_count,
			unique_users = GREATEST(ssh_key_analytics.unique_users, EXCLUDED.unique_users),
			unique_targets = GREATEST(ssh_key_analytics.unique_targets, EXCLUDED.unique_targets),
			first_use_time = LEAST(COALESCE(ssh_key_analytics.first_use_time, EXCLUDED.first_use_time), EXCLUDED.first_use_time),
			last_use_time = GREATEST(COALESCE(ssh_key_analytics.last_use_time, EXCLUDED.last_use_time), EXCLUDED.last_use_time),
			off_hours_usage = ssh_key_analytics.off_hours_usage + EXCLUDED.off_hours_usage,
			unusual_source_usage = ssh_key_analytics.unusual_source_usage + EXCLUDED.unusual_source_usage,
			failed_attempts = ssh_key_analytics.failed_attempts + EXCLUDED.failed_attempts,
			updated_at = NOW()
		RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("ssh_key_analytics.Create: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(analytics); err != nil {
			return fmt.Errorf("ssh_key_analytics.Create scan: %w", err)
		}
	}

	return nil
}

// Get retrieves SSH key analytics by tenant, key ID, and date
func (r *SSHKeyAnalyticsRepository) Get(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*model.SSHKeyAnalytics, error) {
	var analytics model.SSHKeyAnalytics
	query := `
		SELECT * FROM ssh_key_analytics
		WHERE tenant_id = $1 AND ssh_key_id = $2 AND date = $3
	`

	err := r.db.GetContext(ctx, &analytics, query, tenantID, sshKeyID, date)
	if err != nil {
		return nil, fmt.Errorf("ssh_key_analytics.Get: %w", err)
	}

	return &analytics, nil
}

// List retrieves SSH key analytics with filtering
func (r *SSHKeyAnalyticsRepository) List(ctx context.Context, filter model.SSHKeyAnalyticsFilter) ([]model.SSHKeyAnalytics, error) {
	baseQuery := `
		SELECT * FROM ssh_key_analytics
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.SSHKeyID != nil {
		baseQuery += fmt.Sprintf(" AND ssh_key_id = $%d", argCount)
		args = append(args, *filter.SSHKeyID)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND date >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND date <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	baseQuery += " ORDER BY date DESC"

	var analytics []model.SSHKeyAnalytics
	err := r.db.SelectContext(ctx, &analytics, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("ssh_key_analytics.List: %w", err)
	}

	return analytics, nil
}

// Update updates SSH key analytics
func (r *SSHKeyAnalyticsRepository) Update(ctx context.Context, analytics *model.SSHKeyAnalytics) error {
	analytics.UpdatedAt = time.Now()

	query := `
		UPDATE ssh_key_analytics SET
			usage_count = :usage_count,
			unique_users = :unique_users,
			unique_targets = :unique_targets,
			first_use_time = :first_use_time,
			last_use_time = :last_use_time,
			avg_session_duration_seconds = :avg_session_duration_seconds,
			off_hours_usage = :off_hours_usage,
			unusual_source_usage = :unusual_source_usage,
			failed_attempts = :failed_attempts,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("ssh_key_analytics.Update: %w", err)
	}

	return nil
}

// Delete deletes SSH key analytics by ID
func (r *SSHKeyAnalyticsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ssh_key_analytics WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ssh_key_analytics.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ssh_key_analytics.Delete: no rows affected")
	}

	return nil
}

// GetByKeyID retrieves analytics for a specific SSH key across all dates
func (r *SSHKeyAnalyticsRepository) GetByKeyID(ctx context.Context, tenantID, sshKeyID uuid.UUID, limit int) ([]model.SSHKeyAnalytics, error) {
	var analytics []model.SSHKeyAnalytics
	query := `
		SELECT * FROM ssh_key_analytics
		WHERE tenant_id = $1 AND ssh_key_id = $2
		ORDER BY date DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	err := r.db.SelectContext(ctx, &analytics, query, tenantID, sshKeyID)
	if err != nil {
		return nil, fmt.Errorf("ssh_key_analytics.GetByKeyID: %w", err)
	}

	return analytics, nil
}

// GetUsageSummary returns a summary of SSH key usage for a tenant
func (r *SSHKeyAnalyticsRepository) GetUsageSummary(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*SSHKeyUsageSummary, error) {
	query := `
		SELECT
			COUNT(DISTINCT ssh_key_id) as total_keys,
			SUM(usage_count) as total_usage,
			SUM(unique_users) as total_unique_users,
			SUM(unique_targets) as total_unique_targets,
			SUM(off_hours_usage) as total_off_hours,
			SUM(unusual_source_usage) as total_unusual,
			SUM(failed_attempts) as total_failed,
			COUNT(*) FILTER (WHERE usage_count = 0) as unused_keys
		FROM ssh_key_analytics
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
	`

	var summary SSHKeyUsageSummary
	err := r.db.GetContext(ctx, &summary, query, tenantID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("ssh_key_analytics.GetUsageSummary: %w", err)
	}

	return &summary, nil
}

// SSHKeyUsageSummary represents a summary of SSH key usage
type SSHKeyUsageSummary struct {
	TotalKeys         int  `db:"total_keys"`
	TotalUsage        int  `db:"total_usage"`
	TotalUniqueUsers  int  `db:"total_unique_users"`
	TotalUniqueTargets int `db:"total_unique_targets"`
	TotalOffHours     int  `db:"total_off_hours"`
	TotalUnusual      int  `db:"total_unusual"`
	TotalFailed       int  `db:"total_failed"`
	UnusedKeys        int  `db:"unused_keys"`
}

// GetMostUsedKeys retrieves the most used SSH keys for a tenant
func (r *SSHKeyAnalyticsRepository) GetMostUsedKeys(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]KeyUsageRank, error) {
	query := `
		SELECT
			ssh_key_id,
			SUM(usage_count) as total_usage,
			SUM(unique_users) as unique_users,
			SUM(unique_targets) as unique_targets
		FROM ssh_key_analytics
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
		GROUP BY ssh_key_id
		ORDER BY total_usage DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var ranks []KeyUsageRank
	err := r.db.SelectContext(ctx, &ranks, query, tenantID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("ssh_key_analytics.GetMostUsedKeys: %w", err)
	}

	return ranks, nil
}

// KeyUsageRank represents the usage rank of an SSH key
type KeyUsageRank struct {
	SSHKeyID       uuid.UUID `db:"ssh_key_id"`
	TotalUsage     int       `db:"total_usage"`
	UniqueUsers    int       `db:"unique_users"`
	UniqueTargets  int       `db:"unique_targets"`
}

// GetAnomalousKeys retrieves SSH keys with unusual usage patterns
func (r *SSHKeyAnalyticsRepository) GetAnomalousKeys(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, threshold int) ([]model.SSHKeyAnalytics, error) {
	var analytics []model.SSHKeyAnalytics
	query := `
		SELECT * FROM ssh_key_analytics
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
		  AND (unusual_source_usage > $4 OR failed_attempts > $4)
		ORDER BY unusual_source_usage DESC, failed_attempts DESC
	`

	err := r.db.SelectContext(ctx, &analytics, query, tenantID, dateFrom, dateTo, threshold)
	if err != nil {
		return nil, fmt.Errorf("ssh_key_analytics.GetAnomalousKeys: %w", err)
	}

	return analytics, nil
}
