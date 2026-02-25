// Package analytics provides baseline management for behavioral analysis
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/analytics"
	"github.com/rs/zerolog"
)

// BaselineManager manages behavioral baselines for anomaly detection
type BaselineManager struct {
	db           *sqlx.DB
	metricsStore *MetricsStore
	redis        *analytics.RedisMetricsCache
	logger       zerolog.Logger
}

// NewBaselineManager creates a new baseline manager
func NewBaselineManager(db *sqlx.DB, metricsStore *MetricsStore, redis *analytics.RedisMetricsCache, logger zerolog.Logger) *BaselineManager {
	return &BaselineManager{
		db:           db,
		metricsStore: metricsStore,
		redis:        redis,
		logger:       logger,
	}
}

// UserBaseline represents a behavioral baseline for a user
type UserBaseline struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	TenantID        uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	UserID          uuid.UUID    `json:"user_id" db:"user_id"`
	BaselineType    string       `json:"baseline_type" db:"baseline_type"` // daily, weekly, monthly
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
	OffHoursAccessMean  float64            `json:"off_hours_access_mean" db:"off_hours_access_mean"`
	OffHoursAccessStdDev float64           `json:"off_hours_access_std_dev" db:"off_hours_access_std_dev"`
	HourlyAccessPattern map[int]float64    `json:"hourly_access_pattern" db:"hourly_access_pattern"`
	WeekdayAccessPattern map[time.Weekday]float64 `json:"weekday_access_pattern" db:"weekday_access_pattern"`

	// Spatial metrics
	TargetAccessStats   map[string]TargetStats `json:"target_access_stats" db:"target_access_stats"`
	TargetAccessStdDev  float64                `json:"target_access_std_dev" db:"target_access_std_dev"`

	// Command patterns (top commands)
	TopCommands         map[string]int `json:"top_commands" db:"top_commands"`

	// Metadata
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" db:"updated_at"`
	LastCalculated time.Time          `json:"last_calculated" db:"last_calculated"`
	IsActive       bool               `json:"is_active" db:"is_active"`
	Confidence     float64            `json:"confidence" db:"confidence"` // 0-100
}

// TargetStats represents statistics for a specific target
type TargetStats struct {
	TargetHost       string  `json:"target_host"`
	MeanAccessCount  float64 `json:"mean_access_count"`
	StdDevAccessCount float64 `json:"std_dev_access_count"`
	FirstAccess      time.Time `json:"first_access"`
	LastAccess       time.Time `json:"last_access"`
}

// GetUserBaseline retrieves the active baseline for a user
func (b *BaselineManager) GetUserBaseline(ctx context.Context, tenantID, userID uuid.UUID) (*UserBaseline, error) {
	// Try Redis cache first
	cached, err := b.redis.GetUserBaseline(ctx, tenantID, userID)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from database
	query := `
		SELECT * FROM user_baselines
		WHERE tenant_id = $1 AND user_id = $2 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	var baseline UserBaseline
	err = b.db.GetContext(ctx, &baseline, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("baseline_manager.GetUserBaseline: %w", err)
	}

	// Parse JSON fields
	if baseline.HourlyAccessPatternStr != nil {
		json.Unmarshal(baseline.HourlyAccessPatternStr, &baseline.HourlyAccessPattern)
	}
	if baseline.WeekdayAccessPatternStr != nil {
		json.Unmarshal(baseline.WeekdayAccessPatternStr, &baseline.WeekdayAccessPattern)
	}
	if baseline.TargetAccessStatsStr != nil {
		json.Unmarshal(baseline.TargetAccessStatsStr, &baseline.TargetAccessStats)
	}
	if baseline.TopCommandsStr != nil {
		json.Unmarshal(baseline.TopCommandsStr, &baseline.TopCommands)
	}

	// Cache in Redis
	_ = b.redis.StoreUserBaseline(ctx, &baseline)

	return &baseline, nil
}

// BuildUserBaseline creates a new baseline from historical metrics
func (b *BaselineManager) BuildUserBaseline(ctx context.Context, tenantID, userID uuid.UUID, periodStart, periodEnd time.Time, baselineType string) (*UserBaseline, error) {
	// Fetch metrics for the period
	metrics, err := b.metricsStore.GetUserBehaviorMetrics(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("baseline_manager.BuildUserBaseline: failed to get metrics: %w", err)
	}

	if len(metrics) < 30 {
		return nil, fmt.Errorf("baseline_manager.BuildUserBaseline: insufficient data points (need at least 30, got %d)", len(metrics))
	}

	baseline := &UserBaseline{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		BaselineType: baselineType,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		SampleSize:   len(metrics),
		IsActive:     true,
		Confidence:   calculateBaselineConfidence(len(metrics), periodEnd.Sub(periodStart)),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastCalculated: time.Now(),
	}

	// Calculate session statistics
	sessionCounts := groupMetricsByDay(metrics)
	dailyValues := make([]float64, 0, len(sessionCounts))
	for _, count := range sessionCounts {
		dailyValues = append(dailyValues, float64(count))
	}
	baseline.MeanDailySessions, baseline.StdDevDailySessions = calculateStatistics(dailyValues)

	// Calculate duration statistics
	durations := make([]float64, len(metrics))
	for i, m := range metrics {
		durations[i] = m.DurationSeconds
	}
	baseline.MeanSessionDuration, baseline.StdDevSessionDuration = calculateStatistics(durations)

	// Calculate command statistics
	commandCounts := groupCommandsByDay(metrics)
	dailyCommands := make([]float64, 0, len(commandCounts))
	for _, count := range commandCounts {
		dailyCommands = append(dailyCommands, float64(count))
	}
	baseline.MeanDailyCommands, baseline.StdDevDailyCommands = calculateStatistics(dailyCommands)

	// Calculate failure rate statistics
	failureRates := make([]float64, 0, len(sessionCounts))
	for _, dayMetrics := range sessionCounts {
		failedCount := 0
		for _, m := range dayMetrics {
			if m.Failed {
				failedCount++
			}
		}
		failureRate := float64(failedCount) / float64(len(dayMetrics))
		failureRates = append(failureRates, failureRate)
	}
	baseline.MeanFailureRate, baseline.StdDevFailureRate = calculateStatistics(failureRates)

	// Calculate off-hours access statistics
	offHourCounts := groupOffHoursByDay(metrics)
	offHourValues := make([]float64, 0, len(offHourCounts))
	for _, dayMetrics := range offHourCounts {
		offHourCount := 0
		for _, m := range dayMetrics {
			if m.OffHoursAccess {
				offHourCount++
			}
		}
		offHourRatio := float64(offHourCount) / float64(len(dayMetrics))
		offHourValues = append(offHourValues, offHourRatio)
	}
	baseline.OffHoursAccessMean, baseline.OffHoursAccessStdDev = calculateStatistics(offHourValues)

	// Calculate hourly pattern
	baseline.HourlyAccessPattern = calculateHourlyPattern(metrics)

	// Calculate weekday pattern
	baseline.WeekdayAccessPattern = calculateWeekdayPattern(metrics)

	// Calculate target access statistics
	baseline.TargetAccessStats, baseline.TargetAccessStdDev = calculateTargetAccessStats(metrics)

	// Calculate top commands
	baseline.TopCommands = calculateTopCommands(metrics, 20)

	// Save to database
	if err := b.saveBaseline(ctx, baseline); err != nil {
		return nil, err
	}

	// Cache in Redis
	_ = b.redis.StoreUserBaseline(ctx, baseline)

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
	// Marshal JSON fields
	hourlyJSON, _ := json.Marshal(baseline.HourlyAccessPattern)
	weekdayJSON, _ := json.Marshal(baseline.WeekdayAccessPattern)
	targetJSON, _ := json.Marshal(baseline.TargetAccessStats)
	commandsJSON, _ := json.Marshal(baseline.TopCommands)

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

	_, err := b.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":                         baseline.ID,
		"tenant_id":                  baseline.TenantID,
		"user_id":                    baseline.UserID,
		"baseline_type":              baseline.BaselineType,
		"period_start":               baseline.PeriodStart,
		"period_end":                 baseline.PeriodEnd,
		"sample_size":                baseline.SampleSize,
		"mean_daily_sessions":        baseline.MeanDailySessions,
		"std_dev_daily_sessions":     baseline.StdDevDailySessions,
		"mean_session_duration":      baseline.MeanSessionDuration,
		"std_dev_session_duration":   baseline.StdDevSessionDuration,
		"mean_daily_commands":        baseline.MeanDailyCommands,
		"std_dev_daily_commands":     baseline.StdDevDailyCommands,
		"mean_failure_rate":          baseline.MeanFailureRate,
		"std_dev_failure_rate":       baseline.StdDevFailureRate,
		"off_hours_access_mean":      baseline.OffHoursAccessMean,
		"off_hours_access_std_dev":   baseline.OffHoursAccessStdDev,
		"hourly_access_pattern":      hourlyJSON,
		"weekday_access_pattern":     weekdayJSON,
		"target_access_stats":        targetJSON,
		"target_access_std_dev":      baseline.TargetAccessStdDev,
		"top_commands":               commandsJSON,
		"created_at":                 baseline.CreatedAt,
		"updated_at":                 baseline.UpdatedAt,
		"last_calculated":            baseline.LastCalculated,
		"is_active":                  baseline.IsActive,
		"confidence":                 baseline.Confidence,
	})

	return err
}

// UpdateBaseline updates an existing baseline
func (b *BaselineManager) UpdateBaseline(ctx context.Context, baseline *UserBaseline) error {
	baseline.UpdatedAt = time.Now()

	hourlyJSON, _ := json.Marshal(baseline.HourlyAccessPattern)
	weekdayJSON, _ := json.Marshal(baseline.WeekdayAccessPattern)
	targetJSON, _ := json.Marshal(baseline.TargetAccessStats)
	commandsJSON, _ := json.Marshal(baseline.TopCommands)

	query := `
		UPDATE user_baselines SET
			period_start = :period_start,
			period_end = :period_end,
			sample_size = :sample_size,
			mean_daily_sessions = :mean_daily_sessions,
			std_dev_daily_sessions = :std_dev_daily_sessions,
			mean_session_duration = :mean_session_duration,
			std_dev_session_duration = :std_dev_session_duration,
			mean_daily_commands = :mean_daily_commands,
			std_dev_daily_commands = :std_dev_daily_commands,
			mean_failure_rate = :mean_failure_rate,
			std_dev_failure_rate = :std_dev_failure_rate,
			off_hours_access_mean = :off_hours_access_mean,
			off_hours_access_std_dev = :off_hours_access_std_dev,
			hourly_access_pattern = :hourly_access_pattern,
			weekday_access_pattern = :weekday_access_pattern,
			target_access_stats = :target_access_stats,
			target_access_std_dev = :target_access_std_dev,
			top_commands = :top_commands,
			updated_at = :updated_at,
			last_calculated = :last_calculated,
			is_active = :is_active,
			confidence = :confidence
		WHERE id = :id
	`

	_, err := b.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":                         baseline.ID,
		"period_start":               baseline.PeriodStart,
		"period_end":                 baseline.PeriodEnd,
		"sample_size":                baseline.SampleSize,
		"mean_daily_sessions":        baseline.MeanDailySessions,
		"std_dev_daily_sessions":     baseline.StdDevDailySessions,
		"mean_session_duration":      baseline.MeanSessionDuration,
		"std_dev_session_duration":   baseline.StdDevSessionDuration,
		"mean_daily_commands":        baseline.MeanDailyCommands,
		"std_dev_daily_commands":     baseline.StdDevDailyCommands,
		"mean_failure_rate":          baseline.MeanFailureRate,
		"std_dev_failure_rate":       baseline.StdDevFailureRate,
		"off_hours_access_mean":      baseline.OffHoursAccessMean,
		"off_hours_access_std_dev":   baseline.OffHoursAccessStdDev,
		"hourly_access_pattern":      hourlyJSON,
		"weekday_access_pattern":     weekdayJSON,
		"target_access_stats":        targetJSON,
		"target_access_std_dev":      baseline.TargetAccessStdDev,
		"top_commands":               commandsJSON,
		"updated_at":                 baseline.UpdatedAt,
		"last_calculated":            baseline.LastCalculated,
		"is_active":                  baseline.IsActive,
		"confidence":                 baseline.Confidence,
	})

	if err != nil {
		return fmt.Errorf("baseline_manager.UpdateBaseline: %w", err)
	}

	// Invalidate cache
	_ = b.redis.InvalidateUserBaseline(ctx, baseline.TenantID, baseline.UserID)

	return nil
}

// DeactivateBaseline deactivates a baseline
func (b *BaselineManager) DeactivateBaseline(ctx context.Context, baselineID uuid.UUID) error {
	query := `
		UPDATE user_baselines
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	_, err := b.db.ExecContext(ctx, query, baselineID)
	if err != nil {
		return fmt.Errorf("baseline_manager.DeactivateBaseline: %w", err)
	}

	return nil
}

// ListBaselinesForUser lists all baselines for a user
func (b *BaselineManager) ListBaselinesForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]UserBaseline, error) {
	query := `
		SELECT * FROM user_baselines
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`

	var baselines []UserBaseline
	err := b.db.SelectContext(ctx, &baselines, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("baseline_manager.ListBaselinesForUser: %w", err)
	}

	return baselines, nil
}

// RebuildAllBaselinesForTenant rebuilds baselines for all users in a tenant
func (b *BaselineManager) RebuildAllBaselinesForTenant(ctx context.Context, tenantID uuid.UUID, baselineType string) (int, error) {
	// Get all users with recent activity
	query := `
		SELECT DISTINCT user_id
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= NOW() - INTERVAL '90 days'
	`

	rows, err := b.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("baseline_manager.RebuildAllBaselinesForTenant: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		userIDs = append(userIDs, userID)
	}

	builtCount := 0
	now := time.Now()
	periodStart := now.AddDate(0, 0, -90) // 90 day lookback

	for _, userID := range userIDs {
		_, err := b.BuildUserBaseline(ctx, tenantID, userID, periodStart, now, baselineType)
		if err != nil {
			b.logger.Warn().Err(err).
				Str("user_id", userID.String()).
				Msg("Failed to build baseline for user")
			continue
		}
		builtCount++
	}

	b.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("built", builtCount).
		Int("total_users", len(userIDs)).
		Msg("Baselines rebuilt for tenant")

	return builtCount, nil
}

// GetBaselineStatistics returns statistics about baselines
func (b *BaselineManager) GetBaselineStatistics(ctx context.Context, tenantID uuid.UUID) (*BaselineStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_baselines,
			COUNT(*) FILTER (WHERE is_active) as active_baselines,
			COUNT(*) FILTER (WHERE baseline_type = 'daily') as daily_baselines,
			COUNT(*) FILTER (WHERE baseline_type = 'weekly') as weekly_baselines,
			COUNT(*) FILTER (WHERE baseline_type = 'monthly') as monthly_baselines,
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

// BaselineStatistics represents statistics about baselines
type BaselineStatistics struct {
	TotalBaselines   int        `json:"total_baselines" db:"total_baselines"`
	ActiveBaselines  int        `json:"active_baselines" db:"active_baselines"`
	DailyBaselines   int        `json:"daily_baselines" db:"daily_baselines"`
	WeeklyBaselines  int        `json:"weekly_baselines" db:"weekly_baselines"`
	MonthlyBaselines int        `json:"monthly_baselines" db:"monthly_baselines"`
	UniqueUsers      int        `json:"unique_users" db:"unique_users"`
	AvgSampleSize    float64    `json:"avg_sample_size" db:"avg_sample_size"`
	AvgConfidence    float64    `json:"avg_confidence" db:"avg_confidence"`
	LastCalculated   *time.Time `json:"last_calculated" db:"last_calculated"`
}

// Helper functions for baseline calculation

func calculateBaselineConfidence(sampleSize int, periodDuration time.Duration) float64 {
	// Confidence based on sample size and period length
	sizeScore := math.Min(100.0, float64(sampleSize)*2.0)
	durationScore := math.Min(100.0, float64(periodDuration.Hours()/24.0)) // 1 point per day
	confidence := (sizeScore + durationScore) / 2.0
	return math.Min(100.0, confidence)
}

func groupMetricsByDay(metrics []SessionMetric) map[string][]SessionMetric {
	result := make(map[string][]SessionMetric)
	for _, m := range metrics {
		dayKey := m.Timestamp.Format("2006-01-02")
		result[dayKey] = append(result[dayKey], m)
	}
	return result
}

func groupCommandsByDay(metrics []SessionMetric) map[string]int {
	result := make(map[string]int)
	dayMetrics := groupMetricsByDay(metrics)
	for day, dayMs := range dayMetrics {
		totalCommands := 0
		for _, m := range dayMs {
			totalCommands += len(m.Commands)
		}
		result[day] = totalCommands
	}
	return result
}

func groupOffHoursByDay(metrics []SessionMetric) map[string][]SessionMetric {
	return groupMetricsByDay(metrics)
}

func calculateHourlyPattern(metrics []SessionMetric) map[int]float64 {
	hourlyCounts := make(map[int]int)
	hourlyTotal := make(map[int]int)

	for _, m := range metrics {
		hour := m.Timestamp.Hour()
		hourlyCounts[hour]++
	}

	// Calculate mean for each hour (0-23)
	// For each hour, calculate the average sessions across all days
	hourlyPattern := make(map[int]float64)
	daysWithData := make(map[int]map[string]bool) // hour -> set of dates

	for _, m := range metrics {
		hour := m.Timestamp.Hour()
		date := m.Timestamp.Format("2006-01-02")
		if daysWithData[hour] == nil {
			daysWithData[hour] = make(map[string]bool)
		}
		daysWithData[hour][date] = true
	}

	for hour := 0; hour < 24; hour++ {
		daysCount := len(daysWithData[hour])
		if daysCount > 0 {
			hourlyPattern[hour] = float64(hourlyCounts[hour]) / float64(daysCount)
		}
	}

	return hourlyPattern
}

func calculateWeekdayPattern(metrics []SessionMetric) map[time.Weekday]float64 {
	weekdayCounts := make(map[time.Weekday]int)
	daysCount := make(map[time.Weekday]map[string]bool)

	for _, m := range metrics {
		weekday := m.Timestamp.Weekday()
		weekdayCounts[weekday]++
		date := m.Timestamp.Format("2006-01-02")
		if daysCount[weekday] == nil {
			daysCount[weekday] = make(map[string]bool)
		}
		daysCount[weekday][date] = true
	}

	pattern := make(map[time.Weekday]float64)
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		totalDays := len(daysCount[wd])
		if totalDays > 0 {
			pattern[wd] = float64(weekdayCounts[wd]) / float64(totalDays)
		}
	}

	return pattern
}

func calculateTargetAccessStats(metrics []SessionMetric) (map[string]TargetStats, float64) {
	targetCounts := make(map[string]int)
	targetFirstSeen := make(map[string]time.Time)
	targetLastSeen := make(map[string]time.Time)

	for _, m := range metrics {
		targetCounts[m.TargetHost]++
		if first, ok := targetFirstSeen[m.TargetHost]; !ok || m.Timestamp.Before(first) {
			targetFirstSeen[m.TargetHost] = m.Timestamp
		}
		if last, ok := targetLastSeen[m.TargetHost]; !ok || m.Timestamp.After(last) {
			targetLastSeen[m.TargetHost] = m.Timestamp
		}
	}

	stats := make(map[string]TargetStats)
	accessCounts := make([]float64, 0, len(targetCounts))

	for target, count := range targetCounts {
		stats[target] = TargetStats{
			TargetHost:      target,
			MeanAccessCount: float64(count),
			FirstAccess:     targetFirstSeen[target],
			LastAccess:      targetLastSeen[target],
		}
		accessCounts = append(accessCounts, float64(count))
	}

	_, stdDev := calculateStatistics(accessCounts)

	// Update std dev for each target
	for target := range stats {
		stats[target].StdDevAccessCount = stdDev
	}

	return stats, stdDev
}

func calculateTopCommands(metrics []SessionMetric, topN int) map[string]int {
	commandCounts := make(map[string]int)

	for _, m := range metrics {
		for _, cmd := range m.Commands {
			commandCounts[cmd]++
		}
	}

	// Sort by count and take top N
	type cmdCount struct {
		cmd   string
		count int
	}
	sorted := make([]cmdCount, 0, len(commandCounts))
	for cmd, count := range commandCounts {
		sorted = append(sorted, cmdCount{cmd, count})
	}

	// Simple sort (could use sort.Slice)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	topCommands := make(map[string]int)
	limit := topN
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		topCommands[sorted[i].cmd] = sorted[i].count
	}

	return topCommands
}

// UserBaseline extension for JSON scanning
type UserBaselineExtended struct {
	UserBaseline
	HourlyAccessPatternStr    []byte `db:"hourly_access_pattern"`
	WeekdayAccessPatternStr   []byte `db:"weekday_access_pattern"`
	TargetAccessStatsStr      []byte `db:"target_access_stats"`
	TopCommandsStr            []byte `db:"top_commands"`
}
