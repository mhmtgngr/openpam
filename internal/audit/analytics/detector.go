// Package analytics provides statistical anomaly detection for security monitoring
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// Detector performs statistical anomaly detection using Z-score and IQR methods
type Detector struct {
	db           *sqlx.DB
	metricsStore *MetricsStore
	baselineMgr  *BaselineManager
	logger       zerolog.Logger
}

// NewDetector creates a new anomaly detector
func NewDetector(db *sqlx.DB, metricsStore *MetricsStore, baselineMgr *BaselineManager, logger zerolog.Logger) *Detector {
	return &Detector{
		db:           db,
		metricsStore: metricsStore,
		baselineMgr:  baselineMgr,
		logger:       logger,
	}
}

// DetectionConfig holds configuration for anomaly detection
type DetectionConfig struct {
	// Z-score threshold for flagging anomalies (default: 3.0)
	ZScoreThreshold float64
	// IQR multiplier for outlier detection (default: 1.5)
	IQRMultiplier float64
	// Minimum sample size required for detection (default: 30)
	MinSampleSize int
	// Time window for detection analysis (default: 24 hours)
	AnalysisWindow time.Duration
	// Enable real-time detection
	EnableRealtime bool
}

// DefaultDetectionConfig returns the default detection configuration
func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{
		ZScoreThreshold: 3.0,
		IQRMultiplier:   1.5,
		MinSampleSize:   30,
		AnalysisWindow:  24 * time.Hour,
		EnableRealtime:  true,
	}
}

// DetectionResult represents the result of an anomaly detection run
type DetectionResult struct {
	Anomalies      []DetectedAnomaly `json:"anomalies"`
	TenantID       uuid.UUID         `json:"tenant_id"`
	AnalysisPeriod TimeRange         `json:"analysis_period"`
	DetectionTime  time.Time         `json:"detection_time"`
	Method         string            `json:"method"`
	Config         DetectionConfig   `json:"config"`
}

// DetectedAnomaly represents a single detected anomaly
type DetectedAnomaly struct {
	Type           model.AnomalyType `json:"type"`
	UserID         *uuid.UUID        `json:"user_id,omitempty"`
	SessionID      *uuid.UUID        `json:"session_id,omitempty"`
	TargetHost     *string           `json:"target_host,omitempty"`
	Severity       model.Severity    `json:"severity"`
	Confidence     float64           `json:"confidence"`
	RiskScore      float64           `json:"risk_score"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Indicators     map[string]interface{} `json:"indicators"`
	DetectedAt     time.Time         `json:"detected_at"`
	CorrelationKey string            `json:"correlation_key"`
}

// TimeRange represents a time period
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// DetectAnomalies runs anomaly detection for a tenant
func (d *Detector) DetectAnomalies(ctx context.Context, tenantID uuid.UUID, config DetectionConfig) (*DetectionResult, error) {
	startTime := time.Now()
	to := time.Now()
	from := to.Add(-config.AnalysisWindow)

	result := &DetectionResult{
		TenantID: tenantID,
		AnalysisPeriod: TimeRange{
			From: from,
			To:   to,
		},
		DetectionTime: startTime,
		Method:        "statistical_composite",
		Config:        config,
		Anomalies:     []DetectedAnomaly{},
	}

	// Run all detection types
	behavioralAnomalies, err := d.detectBehavioralAnomalies(ctx, tenantID, from, to, config)
	if err != nil {
		d.logger.Error().Err(err).
			Str("tenant_id", tenantID.String()).
			Msg("Failed to detect behavioral anomalies")
	} else {
		result.Anomalies = append(result.Anomalies, behavioralAnomalies...)
	}

	temporalAnomalies, err := d.detectTemporalAnomalies(ctx, tenantID, from, to, config)
	if err != nil {
		d.logger.Error().Err(err).
			Str("tenant_id", tenantID.String()).
			Msg("Failed to detect temporal anomalies")
	} else {
		result.Anomalies = append(result.Anomalies, temporalAnomalies...)
	}

	spatialAnomalies, err := d.detectSpatialAnomalies(ctx, tenantID, from, to, config)
	if err != nil {
		d.logger.Error().Err(err).
			Str("tenant_id", tenantID.String()).
			Msg("Failed to detect spatial anomalies")
	} else {
		result.Anomalies = append(result.Anomalies, spatialAnomalies...)
	}

	volumetricAnomalies, err := d.detectVolumetricAnomalies(ctx, tenantID, from, to, config)
	if err != nil {
		d.logger.Error().Err(err).
			Str("tenant_id", tenantID.String()).
			Msg("Failed to detect volumetric anomalies")
	} else {
		result.Anomalies = append(result.Anomalies, volumetricAnomalies...)
	}

	d.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("anomalies_detected", len(result.Anomalies)).
		Dur("detection_duration", time.Since(startTime)).
		Msg("Anomaly detection completed")

	return result, nil
}

// detectBehavioralAnomalies detects unusual user behavior patterns
func (d *Detector) detectBehavioralAnomalies(ctx context.Context, tenantID uuid.UUID, from, to time.Time, config DetectionConfig) ([]DetectedAnomaly, error) {
	anomalies := []DetectedAnomaly{}

	// Get behavioral metrics for the period
	metrics, err := d.metricsStore.GetUserBehaviorMetrics(ctx, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("detector.behavioral: failed to get metrics: %w", err)
	}

	// Group by user for individual analysis
	userMetrics := groupByUser(metrics)

	for userID, userMetricList := range userMetrics {
		if len(userMetricList) < config.MinSampleSize {
			continue
		}

		// Get baseline for this user
		baseline, err := d.baselineMgr.GetUserBaseline(ctx, tenantID, userID)
		if err != nil {
			d.logger.Debug().Err(err).
				Str("user_id", userID.String()).
				Msg("No baseline found for user, skipping")
			continue
		}

		// Check for command pattern anomalies
		cmdAnomalies := d.detectCommandPatternAnomalies(userID, userMetricList, baseline, config)
		anomalies = append(anomalies, cmdAnomalies...)

		// Check for failure rate anomalies
		failAnomalies := d.detectFailureRateAnomalies(userID, userMetricList, baseline, config)
		anomalies = append(anomalies, failAnomalies...)

		// Check for session duration anomalies
		durationAnomalies := d.detectSessionDurationAnomalies(userID, userMetricList, baseline, config)
		anomalies = append(anomalies, durationAnomalies...)
	}

	return anomalies, nil
}

// detectTemporalAnomalies detects time-based anomalies (off-hours access, unusual times)
func (d *Detector) detectTemporalAnomalies(ctx context.Context, tenantID uuid.UUID, from, to time.Time, config DetectionConfig) ([]DetectedAnomaly, error) {
	anomalies := []DetectedAnomaly{}

	// Get metrics for time analysis
	metrics, err := d.metricsStore.GetTemporalMetrics(ctx, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("detector.temporal: failed to get metrics: %w", err)
	}

	// Group by user
	userMetrics := groupByUser(metrics)

	for userID, userMetricList := range userMetrics {
		// Get baseline
		baseline, err := d.baselineMgr.GetUserBaseline(ctx, tenantID, userID)
		if err != nil {
			continue
		}

		// Calculate off-hours access deviation
		offHoursCount := 0
		for _, m := range userMetricList {
			if m.OffHoursAccess {
				offHoursCount++
			}
		}
		offHoursRatio := float64(offHoursCount) / float64(len(userMetricList))

		// Check if off-hours access is unusual for this user
		if baseline.OffHoursAccessMean > 0 {
			zScore := calculateZScore(offHoursRatio, baseline.OffHoursAccessMean, baseline.OffHoursAccessStdDev)
			if math.Abs(zScore) > config.ZScoreThreshold {
				anomalies = append(anomalies, DetectedAnomaly{
					Type:        model.AnomalyTypeTemporal,
					UserID:      &userID,
					Severity:    severityFromZScore(zScore),
					Confidence:  confidenceFromZScore(zScore),
					RiskScore:   riskScoreFromZScore(zScore),
					Title:       "Unusual Off-Hours Access",
					Description: fmt.Sprintf("User accessed system during off-hours %.2f%% of the time, deviating significantly from baseline (%.2f%%)", offHoursRatio*100, baseline.OffHoursAccessMean*100),
					Indicators: map[string]interface{}{
						"off_hours_ratio":    offHoursRatio,
						"baseline_mean":      baseline.OffHoursAccessMean,
						"z_score":            zScore,
						"off_hours_sessions": offHoursCount,
						"total_sessions":     len(userMetricList),
					},
					DetectedAt:     time.Now(),
					CorrelationKey: generateCorrelationKey("temporal_offhours", userID.String(), ""),
				})
			}
		}

		// Check for unusual access patterns by hour of day
		hourAnomalies := d.detectHourlyPatternAnomalies(userID, userMetricList, baseline, config)
		anomalies = append(anomalies, hourAnomalies...)
	}

	return anomalies, nil
}

// detectSpatialAnomalies detects location/target-based anomalies
func (d *Detector) detectSpatialAnomalies(ctx context.Context, tenantID uuid.UUID, from, to time.Time, config DetectionConfig) ([]DetectedAnomaly, error) {
	anomalies := []DetectedAnomaly{}

	// Get metrics for spatial analysis
	metrics, err := d.metricsStore.GetSpatialMetrics(ctx, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("detector.spatial: failed to get metrics: %w", err)
	}

	// Group by user and target
	userTargetMetrics := groupByUserAndTarget(metrics)

	for userID, targetMetrics := range userTargetMetrics {
		baseline, err := d.baselineMgr.GetUserBaseline(ctx, tenantID, userID)
		if err != nil {
			continue
		}

		// Check for access to unusual targets
		for targetHost, targetMetricList := range targetMetrics {
			accessCount := len(targetMetricList)
			isKnownTarget := false
			knownAccessMean := 0.0

			// Check if this is a known target
			if baseline.TargetAccessStats != nil {
				if stats, ok := baseline.TargetAccessStats[targetHost]; ok {
					isKnownTarget = true
					knownAccessMean = stats.MeanAccessCount
				}
			}

			// If new target or unusual access frequency
			if !isKnownTarget && accessCount > 3 {
				anomalies = append(anomalies, DetectedAnomaly{
					Type:       model.AnomalyTypeSpatial,
					UserID:     &userID,
					TargetHost: &targetHost,
					Severity:   model.SeverityMedium,
					Confidence: 70.0,
					RiskScore:  50.0,
					Title:      "Access to Previously Unseen Target",
					Description: fmt.Sprintf("User accessed target %s %d times, which is not in their historical baseline", targetHost, accessCount),
					Indicators: map[string]interface{}{
						"target_host":      targetHost,
						"access_count":     accessCount,
						"is_known_target":  isKnownTarget,
						"first_access":     targetMetricList[0].Timestamp,
					},
					DetectedAt:     time.Now(),
					CorrelationKey: generateCorrelationKey("spatial_new_target", userID.String(), targetHost),
				})
			} else if isKnownTarget {
				// Check for unusual frequency to known target
				zScore := (float64(accessCount) - knownAccessMean) / (baseline.TargetAccessStdDev + 0.001)
				if math.Abs(zScore) > config.ZScoreThreshold {
					anomalies = append(anomalies, DetectedAnomaly{
						Type:       model.AnomalyTypeSpatial,
						UserID:     &userID,
						TargetHost: &targetHost,
						Severity:   severityFromZScore(zScore),
						Confidence: confidenceFromZScore(zScore),
						RiskScore:  riskScoreFromZScore(zScore),
						Title:      "Unusual Access Frequency to Known Target",
						Description: fmt.Sprintf("User accessed %s %d times, deviating from baseline mean of %.2f", targetHost, accessCount, knownAccessMean),
						Indicators: map[string]interface{}{
							"target_host":     targetHost,
							"access_count":    accessCount,
							"baseline_mean":   knownAccessMean,
							"z_score":         zScore,
						},
						DetectedAt:     time.Now(),
						CorrelationKey: generateCorrelationKey("spatial_frequency", userID.String(), targetHost),
					})
				}
			}
		}
	}

	return anomalies, nil
}

// detectVolumetricAnomalies detects volume-based anomalies (command count, session count)
func (d *Detector) detectVolumetricAnomalies(ctx context.Context, tenantID uuid.UUID, from, to time.Time, config DetectionConfig) ([]DetectedAnomaly, error) {
	anomalies := []DetectedAnomaly{}

	// Get aggregate metrics
	metrics, err := d.metricsStore.GetVolumetricMetrics(ctx, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("detector.volumetric: failed to get metrics: %w", err)
	}

	// Analyze per-user volume metrics
	for _, userMetric := range metrics {
		baseline, err := d.baselineMgr.GetUserBaseline(ctx, tenantID, userMetric.UserID)
		if err != nil {
			continue
		}

		// Check command volume anomaly
		if baseline.MeanDailyCommands > 0 {
			zScore := (float64(userMetric.TotalCommands) - baseline.MeanDailyCommands) / (baseline.StdDevDailyCommands + 0.001)
			if math.Abs(zScore) > config.ZScoreThreshold {
				severity := severityFromZScore(zScore)
				title := "Unusual Command Volume"
				if zScore > 0 {
					title = "Elevated Command Volume"
				} else {
					title = "Unusually Low Command Volume"
				}

				anomalies = append(anomalies, DetectedAnomaly{
					Type:        model.AnomalyTypeVolumetric,
					UserID:      &userMetric.UserID,
					Severity:    severity,
					Confidence:  confidenceFromZScore(zScore),
					RiskScore:   riskScoreFromZScore(zScore),
					Title:       title,
					Description: fmt.Sprintf("User executed %d commands, deviating from baseline mean of %.2f (z-score: %.2f)", userMetric.TotalCommands, baseline.MeanDailyCommands, zScore),
					Indicators: map[string]interface{}{
						"command_count":   userMetric.TotalCommands,
						"baseline_mean":   baseline.MeanDailyCommands,
						"z_score":         zScore,
						"session_count":   userMetric.TotalSessions,
					},
					DetectedAt:     time.Now(),
					CorrelationKey: generateCorrelationKey("volumetric_commands", userMetric.UserID.String(), ""),
				})
			}
		}

		// Check session count anomaly
		if baseline.MeanDailySessions > 0 {
			zScore := (float64(userMetric.TotalSessions) - baseline.MeanDailySessions) / (baseline.StdDevDailySessions + 0.001)
			if math.Abs(zScore) > config.ZScoreThreshold {
				anomalies = append(anomalies, DetectedAnomaly{
					Type:        model.AnomalyTypeVolumetric,
					UserID:      &userMetric.UserID,
					Severity:    severityFromZScore(zScore),
					Confidence:  confidenceFromZScore(zScore),
					RiskScore:   riskScoreFromZScore(zScore),
					Title:       "Unusual Session Volume",
					Description: fmt.Sprintf("User initiated %d sessions, deviating from baseline mean of %.2f (z-score: %.2f)", userMetric.TotalSessions, baseline.MeanDailySessions, zScore),
					Indicators: map[string]interface{}{
						"session_count":   userMetric.TotalSessions,
						"baseline_mean":   baseline.MeanDailySessions,
						"z_score":         zScore,
					},
					DetectedAt:     time.Now(),
					CorrelationKey: generateCorrelationKey("volumetric_sessions", userMetric.UserID.String(), ""),
				})
			}
		}
	}

	return anomalies, nil
}

// Helper functions for anomaly detection

// detectCommandPatternAnomalies detects unusual command patterns
func (d *Detector) detectCommandPatternAnomalies(userID uuid.UUID, metrics []SessionMetric, baseline *UserBaseline, config DetectionConfig) []DetectedAnomaly {
	anomalies := []DetectedAnomaly{}

	// Aggregate command patterns
	commandCounts := make(map[string]int)
	for _, m := range metrics {
		for _, cmd := range m.Commands {
			commandCounts[cmd]++
		}
	}

	// Check for高危 commands (dangerous commands)
	dangerousCommands := []string{"rm -rf", "dd if=", "mkfs", "chmod 000", "chown -R"}
	for _, cmd := range dangerousCommands {
		if count, exists := commandCounts[cmd]; exists && count > 0 {
			anomalies = append(anomalies, DetectedAnomaly{
				Type:        model.AnomalyTypeBehavioral,
				UserID:      &userID,
				Severity:    model.SeverityCritical,
				Confidence:  95.0,
				RiskScore:   95.0,
				Title:       "Dangerous Command Execution",
				Description: fmt.Sprintf("User executed dangerous command '%s' %d times", cmd, count),
				Indicators: map[string]interface{}{
					"command":       cmd,
					"execution_count": count,
					"risk_category":  "data_destruction",
				},
				DetectedAt:     time.Now(),
				CorrelationKey: generateCorrelationKey("behavioral_dangerous_cmd", userID.String(), cmd),
			})
		}
	}

	return anomalies
}

// detectFailureRateAnomalies detects unusual failure rates
func (d *Detector) detectFailureRateAnomalies(userID uuid.UUID, metrics []SessionMetric, baseline *UserBaseline, config DetectionConfig) []DetectedAnomaly {
	anomalies := []DetectedAnomaly{}

	totalSessions := len(metrics)
	failedSessions := 0
	for _, m := range metrics {
		if m.Failed {
			failedSessions++
		}
	}

	if totalSessions < config.MinSampleSize {
		return anomalies
	}

	failureRate := float64(failedSessions) / float64(totalSessions)

	if baseline.MeanFailureRate > 0 {
		zScore := (failureRate - baseline.MeanFailureRate) / (baseline.StdDevFailureRate + 0.001)
		if zScore > config.ZScoreThreshold {
			anomalies = append(anomalies, DetectedAnomaly{
				Type:        model.AnomalyTypeBehavioral,
				UserID:      &userID,
				Severity:    severityFromZScore(zScore),
				Confidence:  confidenceFromZScore(zScore),
				RiskScore:   riskScoreFromZScore(zScore),
				Title:       "Elevated Session Failure Rate",
				Description: fmt.Sprintf("User had %.2f%% session failure rate, significantly above baseline of %.2f%%", failureRate*100, baseline.MeanFailureRate*100),
				Indicators: map[string]interface{}{
					"failure_rate":    failureRate,
					"baseline_mean":   baseline.MeanFailureRate,
					"failed_sessions": failedSessions,
					"total_sessions":  totalSessions,
				},
				DetectedAt:     time.Now(),
				CorrelationKey: generateCorrelationKey("behavioral_failure_rate", userID.String(), ""),
			})
		}
	}

	return anomalies
}

// detectSessionDurationAnomalies detects unusual session durations
func (d *Detector) detectSessionDurationAnomalies(userID uuid.UUID, metrics []SessionMetric, baseline *UserBaseline, config DetectionConfig) []DetectedAnomaly {
	anomalies := []DetectedAnomaly{}

	if len(metrics) < config.MinSampleSize {
		return anomalies
	}

	// Calculate mean duration for this period
	totalDuration := 0.0
	for _, m := range metrics {
		totalDuration += m.DurationSeconds
	}
	meanDuration := totalDuration / float64(len(metrics))

	if baseline.MeanSessionDuration > 0 {
		zScore := (meanDuration - baseline.MeanSessionDuration) / (baseline.StdDevSessionDuration + 0.001)
		if math.Abs(zScore) > config.ZScoreThreshold {
			title := "Unusual Session Duration"
			if zScore > 0 {
				title = "Extended Session Duration"
			} else {
				title = "Shortened Session Duration"
			}

			anomalies = append(anomalies, DetectedAnomaly{
				Type:        model.AnomalyTypeBehavioral,
				UserID:      &userID,
				Severity:    severityFromZScore(zScore),
				Confidence:  confidenceFromZScore(zScore),
				RiskScore:   riskScoreFromZScore(zScore),
				Title:       title,
				Description: fmt.Sprintf("User session duration averaged %.2f minutes, deviating from baseline of %.2f minutes", meanDuration/60, baseline.MeanSessionDuration/60),
				Indicators: map[string]interface{}{
					"mean_duration_minutes": meanDuration / 60,
					"baseline_mean_minutes": baseline.MeanSessionDuration / 60,
					"z_score":               zScore,
					"session_count":         len(metrics),
				},
				DetectedAt:     time.Now(),
				CorrelationKey: generateCorrelationKey("behavioral_session_duration", userID.String(), ""),
			})
		}
	}

	return anomalies
}

// detectHourlyPatternAnomalies detects unusual hourly access patterns
func (d *Detector) detectHourlyPatternAnomalies(userID uuid.UUID, metrics []SessionMetric, baseline *UserBaseline, config DetectionConfig) []DetectedAnomaly {
	anomalies := []DetectedAnomaly{}

	// Build hourly distribution
	hourlyCounts := make(map[int]int)
	for _, m := range metrics {
		hour := m.Timestamp.Hour()
		hourlyCounts[hour]++
	}

	// Compare with baseline
	if baseline.HourlyAccessPattern != nil {
		for hour, count := range hourlyCounts {
			expectedMean := 0.0
			if val, ok := baseline.HourlyAccessPattern[hour]; ok {
				expectedMean = val
			}

			if expectedMean > 0 {
				zScore := (float64(count) - expectedMean) / (expectedMean * 0.5 + 0.001) // Simplified std dev
				if zScore > config.ZScoreThreshold {
					anomalies = append(anomalies, DetectedAnomaly{
						Type:        model.AnomalyTypeTemporal,
						UserID:      &userID,
						Severity:    severityFromZScore(zScore),
						Confidence:  confidenceFromZScore(zScore),
						RiskScore:   riskScoreFromZScore(zScore),
						Title:       "Unusual Hourly Access Pattern",
						Description: fmt.Sprintf("User accessed system %d times at hour %d, deviating from baseline mean of %.2f", count, hour, expectedMean),
						Indicators: map[string]interface{}{
							"hour":           hour,
							"access_count":   count,
							"baseline_mean":  expectedMean,
							"z_score":        zScore,
						},
						DetectedAt:     time.Now(),
						CorrelationKey: generateCorrelationKey("temporal_hourly", userID.String(), fmt.Sprintf("hour_%d", hour)),
					})
				}
			}
		}
	}

	return anomalies
}

// Statistical utility functions

// calculateZScore computes the Z-score for a value
func calculateZScore(value, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (value - mean) / stdDev
}

// calculateIQRBounds computes the lower and upper bounds using IQR method
func calculateIQRBounds(values []float64, multiplier float64) (lower, upper float64) {
	if len(values) == 0 {
		return 0, 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	q1Index := len(sorted) / 4
	q3Index := (3 * len(sorted)) / 4

	q1 := sorted[q1Index]
	q3 := sorted[q3Index]
	iqr := q3 - q1

	lower = q1 - (iqr * multiplier)
	upper = q3 + (iqr * multiplier)

	return lower, upper
}

// calculateStatistics computes mean and standard deviation
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

// severityFromZScore converts a z-score to a severity level
func severityFromZScore(zScore float64) model.Severity {
	absZ := math.Abs(zScore)
	switch {
	case absZ >= 5:
		return model.SeverityCritical
	case absZ >= 4:
		return model.SeverityHigh
	case absZ >= 3:
		return model.SeverityMedium
	default:
		return model.SeverityLow
	}
}

// confidenceFromZScore converts a z-score to a confidence percentage
func confidenceFromZScore(zScore float64) float64 {
	absZ := math.Abs(zScore)
	confidence := math.Min(99.9, 50 + (absZ * 10))
	return confidence
}

// riskScoreFromZScore calculates a risk score from z-score
func riskScoreFromZScore(zScore float64) float64 {
	absZ := math.Abs(zScore)
	risk := math.Min(100, 40 + (absZ * 12))
	return risk
}

// generateCorrelationKey creates a deterministic correlation key
func generateCorrelationKey(anomalyType, userID, extra string) string {
	key := fmt.Sprintf("%s:%s", anomalyType, userID)
	if extra != "" {
		key += ":" + extra
	}
	return key
}

// Data structures for metrics processing

// SessionMetric represents a single session's metrics
type SessionMetric struct {
	SessionID       uuid.UUID
	UserID          uuid.UUID
	Timestamp       time.Time
	TargetHost      string
	DurationSeconds float64
	Commands        []string
	Failed          bool
	OffHoursAccess  bool
	IPAddress       string
}

// VolumetricMetric represents aggregated volume metrics
type VolumetricMetric struct {
	UserID         uuid.UUID
	TotalSessions  int
	TotalCommands  int
	FailedSessions int
	TotalDuration  float64
}

// Grouping helpers

func groupByUser(metrics []SessionMetric) map[uuid.UUID][]SessionMetric {
	result := make(map[uuid.UUID][]SessionMetric)
	for _, m := range metrics {
		result[m.UserID] = append(result[m.UserID], m)
	}
	return result
}

func groupByUserAndTarget(metrics []SessionMetric) map[uuid.UUID]map[string][]SessionMetric {
	result := make(map[uuid.UUID]map[string][]SessionMetric)
	for _, m := range metrics {
		if _, ok := result[m.UserID]; !ok {
			result[m.UserID] = make(map[string][]SessionMetric)
		}
		result[m.UserID][m.TargetHost] = append(result[m.UserID][m.TargetHost], m)
	}
	return result
}

// ConvertToModelAnomalies converts detected anomalies to model.AnomalyDetection
func (d *Detector) ConvertToModelAnomalies(result *DetectionResult) []model.AnomalyDetection {
	anomalies := make([]model.AnomalyDetection, 0, len(result.Anomalies))

	for _, detected := range result.Anomalies {
		indicatorsJSON, _ := json.Marshal(detected.Indicators)

		anomaly := model.AnomalyDetection{
			TenantID:        result.TenantID,
			AnomalyType:     string(detected.Type),
			UserID:          detected.UserID,
			TargetHost:      detected.TargetHost,
			Severity:        string(detected.Severity),
			ConfidenceScore: detected.Confidence,
			RiskScore:       detected.RiskScore,
			Title:           detected.Title,
			Description:     &detected.Description,
			Indicators:      indicatorsJSON,
			DetectionMethod: "statistical_composite",
			DetectedAt:      detected.DetectedAt,
			Status:          string(model.AnomalyStatusOpen),
			CorrelationKey:  &detected.CorrelationKey,
			AutoTriggered:   true,
		}

		anomalies = append(anomalies, anomaly)
	}

	return anomalies
}
