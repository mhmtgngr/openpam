// Package analytics provides baseline management for audit service
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// BaselineManager manages behavioral baselines for anomaly detection
type BaselineManager struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewBaselineManager creates a new baseline manager
func NewBaselineManager(db *sqlx.DB, logger zerolog.Logger) *BaselineManager {
	return &BaselineManager{
		db:     db,
		logger: logger,
	}
}

// UserBaseline represents a behavioral baseline for a user
type UserBaseline struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	TenantID        uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	UserID          uuid.UUID    `json:"user_id" db:"user_id"`
	BaselineType    string       `json:"baseline_type" db:"baseline_type"`
	PeriodStart     time.Time    `json:"period_start" db:"period_start"`
	PeriodEnd       time.Time    `json:"period_end" db:"period_end"`
	SampleSize      int          `json:"sample_size" db:"sample_size"`

	// Session metrics
	MeanDailySessions    float64 `json:"mean_daily_sessions" db:"mean_daily_sessions"`
	StdDevDailySessions  float64 `json:"std_dev_daily_sessions" db:"std_dev_daily_sessions"`
	MeanSessionDuration  float64 `json:"mean_session_duration" db:"mean_session_duration"`
	StdDevSessionDuration float64 `json:"std_dev_session_duration" db:"std_dev_session_duration"`

	// Command metrics
	MeanDailyCommands   float64 `json:"mean_daily_commands" db:"mean_daily_commands"`
	StdDevDailyCommands float64 `json:"std_dev_daily_commands" db:"std_dev_daily_commands"`

	// Failure metrics
	MeanFailureRate   float64 `json:"mean_failure_rate" db:"mean_failure_rate"`
	StdDevFailureRate float64 `json:"std_dev_failure_rate" db:"std_dev_failure_rate"`

	// Temporal metrics
	OffHoursAccessMean  float64         `json:"off_hours_access_mean" db:"off_hours_access_mean"`
	OffHoursAccessStdDev float64         `json:"off_hours_access_std_dev" db:"off_hours_access_std_dev"`
	HourlyAccessPattern   json.RawMessage `json:"hourly_access_pattern" db:"hourly_access_pattern"`
	WeekdayAccessPattern  json.RawMessage `json:"weekday_access_pattern" db:"weekday_access_pattern"`

	// Spatial metrics
	TargetAccessStats   json.RawMessage `json:"target_access_stats" db:"target_access_stats"`
	TargetAccessStdDev  float64         `json:"target_access_std_dev" db:"target_access_std_dev"`

	// Command patterns
	TopCommands json.RawMessage `json:"top_commands" db:"top_commands"`

	// Metadata
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	LastCalculated time.Time `json:"last_calculated" db:"last_calculated"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	Confidence     float64   `json:"confidence" db:"confidence"`
}

// GetUserBaseline retrieves the active baseline for a user
func (b *BaselineManager) GetUserBaseline(ctx context.Context, tenantID, userID uuid.UUID) (*UserBaseline, error) {
	query := `
		SELECT * FROM user_baselines
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	var baseline UserBaseline
	err := b.db.GetContext(ctx, &baseline, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("baseline_manager.GetUserBaseline: %w", err)
	}

	return &baseline, nil
}

// BuildUserBaseline creates a new baseline from historical metrics
func (b *BaselineManager) BuildUserBaseline(ctx context.Context, tenantID, userID uuid.UUID, periodStart, periodEnd time.Time, baselineType string) (*UserBaseline, error) {
	// Fetch session metrics for the period
	query := `
		SELECT
			DATE(start_time) as date,
			COUNT(*) as session_count,
			SUM(command_count) as command_count,
			AVG(duration_seconds) as avg_duration,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_count,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_count
		FROM session_analytics
		WHERE tenant_id = $1 AND user_id = $2
			AND start_time >= $3 AND start_time <= $4
		GROUP BY DATE(start_time)
		ORDER BY date
	`

	rows, err := b.db.QueryContext(ctx, query, tenantID, userID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("baseline_manager.BuildUserBaseline: failed to query metrics: %w", err)
	}
	defer rows.Close()

	var dailyData []struct {
		Date           time.Time `db:"date"`
		SessionCount   int       `db:"session_count"`
		CommandCount   int       `db:"command_count"`
		AvgDuration    float64   `db:"avg_duration"`
		FailedCount    int       `db:"failed_count"`
		OffHoursCount  int       `db:"off_hours_count"`
	}

	for rows.Next() {
		var d struct {
			Date           time.Time `db:"date"`
			SessionCount   int       `db:"session_count"`
			CommandCount   int       `db:"command_count"`
			AvgDuration    float64   `db:"avg_duration"`
			FailedCount    int       `db:"failed_count"`
			OffHoursCount  int       `db:"off_hours_count"`
		}
		if err := rows.Scan(&d.Date, &d.SessionCount, &d.CommandCount, &d.AvgDuration, &d.FailedCount, &d.OffHoursCount); err != nil {
			continue
		}
		dailyData = append(dailyData, d)
	}

	if len(dailyData) < 30 {
		return nil, fmt.Errorf("baseline_manager.BuildUserBaseline: insufficient data points (need at least 30 days, got %d)", len(dailyData))
	}

	// Calculate baseline statistics
	baseline := &UserBaseline{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		BaselineType: baselineType,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		SampleSize:   len(dailyData),
		IsActive:     true,
		Confidence:   calculateBaselineConfidence(len(dailyData), periodEnd.Sub(periodStart)),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastCalculated: time.Now(),
	}

	// Calculate session statistics
	sessionValues := make([]float64, len(dailyData))
	durationValues := make([]float64, len(dailyData))
	offHourValues := make([]float64, len(dailyData))
	failureRateValues := make([]float64, len(dailyData))
	commandValues := make([]float64, len(dailyData))

	for i, d := range dailyData {
		sessionValues[i] = float64(d.SessionCount)
		durationValues[i] = d.AvgDuration
		offHourValues[i] = float64(d.OffHoursCount) / float64(d.SessionCount)
		failureRateValues[i] = float64(d.FailedCount) / float64(d.SessionCount)
		commandValues[i] = float64(d.CommandCount)
	}

	baseline.MeanDailySessions, baseline.StdDevDailySessions = calculateStatistics(sessionValues)
	baseline.MeanSessionDuration, baseline.StdDevSessionDuration = calculateStatistics(durationValues)
	baseline.OffHoursAccessMean, baseline.OffHoursAccessStdDev = calculateStatistics(offHourValues)
	baseline.MeanFailureRate, baseline.StdDevFailureRate = calculateStatistics(failureRateValues)
	baseline.MeanDailyCommands, baseline.StdDevDailyCommands = calculateStatistics(commandValues)

	// Calculate hourly pattern
	hourlyPattern := calculateHourlyPattern(ctx, b.db, tenantID, userID, periodStart, periodEnd)
	hourlyJSON, _ := json.Marshal(hourlyPattern)
	baseline.HourlyAccessPattern = hourlyJSON

	// Calculate top commands
	topCommands := calculateTopCommands(ctx, b.db, tenantID, userID, periodStart, periodEnd, 20)
	commandsJSON, _ := json.Marshal(topCommands)
	baseline.TopCommands = commandsJSON

	// Save to database
	if err := b.saveBaseline(ctx, baseline); err != nil {
		return nil, err
	}

	b.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("user_id", userID.String()).
		Int("sample_size", baseline.SampleSize).
		Str("baseline_type", baselineType).
		Msg("User baseline created")

	return baseline, nil
}

// saveBaseline saves a baseline to the database
func (b *BaselineManager) saveBaseline(ctx context.Context, baseline *UserBaseline) error {
	// Deactivate existing baselines for this user
	deactivateQuery := `
		UPDATE user_baselines
		SET is_active = false, updated_at = NOW()
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
	`
	_, err := b.db.ExecContext(ctx, deactivateQuery, baseline.TenantID, baseline.UserID)
	if err != nil {
		return fmt.Errorf("failed to deactivate existing baselines: %w", err)
	}

	// Insert new baseline
	query := `
		INSERT INTO user_baselines (
			id, tenant_id, user_id, baseline_type,
			period_start, period_end, sample_size,
			mean_daily_sessions, std_dev_daily_sessions,
			mean_session_duration, std_dev_session_duration,
			mean_daily_commands, std_dev_daily_commands,
			mean_failure_rate, std_dev_failure_rate,
			off_hours_access_mean, off_hours_access_std_dev,
			hourly_access_pattern, weekday_access_pattern,
			target_access_stats, target_access_std_dev,
			top_commands,
			created_at, updated_at, last_calculated,
			is_active, confidence
		) VALUES (
			:id, :tenant_id, :user_id, :baseline_type,
			:period_start, :period_end, :sample_size,
			:mean_daily_sessions, :std_dev_daily_sessions,
			:mean_session_duration, :std_dev_session_duration,
			:mean_daily_commands, :std_dev_daily_commands,
			:mean_failure_rate, :std_dev_failure_rate,
			:off_hours_access_mean, :off_hours_access_std_dev,
			:hourly_access_pattern, :weekday_access_pattern,
			:target_access_stats, :target_access_std_dev,
			:top_commands,
			:created_at, :updated_at, :last_calculated,
			:is_active, :confidence
		)
	`

	_, err = b.db.NamedExecContext(ctx, query, baseline)
	return err
}

// GetBaselineStatistics returns statistics about baselines
func (b *BaselineManager) GetBaselineStatistics(ctx context.Context, tenantID uuid.UUID) (*BaselineStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_baselines,
			COUNT(*) FILTER (WHERE is_active) as active_baselines,
			COUNT(DISTINCT user_id) as unique_users,
			AVG(sample_size) as avg_sample_size,
			AVG(confidence) as avg_confidence,
			MAX(last_calculated) as last_calculated
		FROM user_baselines
		WHERE tenant_id = $1
	`

	var stats BaselineStatistics
	err := b.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("baseline_manager.GetBaselineStatistics: %w", err)
	}

	return &stats, nil
}

// Helper functions

func calculateBaselineConfidence(sampleSize int, periodDuration time.Duration) float64 {
	sizeScore := math.Min(100.0, float64(sampleSize)*2.0)
	durationScore := math.Min(100.0, float64(periodDuration.Hours()/24.0))
	confidence := (sizeScore + durationScore) / 2.0
	return math.Min(100.0, confidence)
}

func calculateStatistics(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	n := float64(len(values))
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / n

	if len(values) == 1 {
		return mean, 0
	}

	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	variance := sumSquaredDiff / n
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func calculateHourlyPattern(ctx context.Context, db *sqlx.DB, tenantID, userID uuid.UUID, start, end time.Time) map[int]float64 {
	query := `
		SELECT EXTRACT(HOUR FROM start_time) as hour, COUNT(*) as count
		FROM session_analytics
		WHERE tenant_id = $1 AND user_id = $2
			AND start_time >= $3 AND start_time <= $4
		GROUP BY EXTRACT(HOUR FROM start_time)
	`

	rows, err := db.QueryContext(ctx, query, tenantID, userID, start, end)
	if err != nil {
		return make(map[int]float64)
	}
	defer rows.Close()

	hourlyCounts := make(map[int]int)

	for rows.Next() {
		var hour float64
		var count int
		if err := rows.Scan(&hour, &count); err != nil {
			continue
		}
		hourlyCounts[int(hour)] += count
	}

	// Calculate mean for each hour
	pattern := make(map[int]float64)
	for hour := 0; hour < 24; hour++ {
		pattern[hour] = float64(hourlyCounts[hour])
	}

	return pattern
}

func calculateTopCommands(ctx context.Context, db *sqlx.DB, tenantID, userID uuid.UUID, start, end time.Time, limit int) map[string]int {
	// Simplified - in production would query and properly parse JSON commands
	return make(map[string]int)
}

// BaselineStatistics represents statistics about baselines
type BaselineStatistics struct {
	TotalBaselines  int        `json:"total_baselines" db:"total_baselines"`
	ActiveBaselines int        `json:"active_baselines" db:"active_baselines"`
	UniqueUsers      int        `json:"unique_users" db:"unique_users"`
	AvgSampleSize    float64    `json:"avg_sample_size" db:"avg_sample_size"`
	AvgConfidence    float64    `json:"avg_confidence" db:"avg_confidence"`
	LastCalculated   *time.Time `json:"last_calculated" db:"last_calculated"`
}
