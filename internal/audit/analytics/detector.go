// Package analytics provides anomaly detection for audit service
// This file was created to implement the design spec
package analytics

import (
	"context"
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
// This is a simplified version that works with the existing model
type Detector struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewDetector creates a new anomaly detector
func NewDetector(db *sqlx.DB, logger zerolog.Logger) *Detector {
	return &Detector{
		db:     db,
		logger: logger,
	}
}

// DetectionConfig holds configuration for anomaly detection
type DetectionConfig struct {
	ZScoreThreshold float64
	IQRMultiplier   float64
	MinSampleSize   int
	AnalysisWindow  time.Duration
	EnableRealtime  bool
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

// DetectAnomaliesForUser detects anomalies for a specific user
func (d *Detector) DetectAnomaliesForUser(ctx context.Context, tenantID, userID uuid.UUID, config DetectionConfig) ([]model.AnomalyDetection, error) {
	// Query session analytics for the user
	query := `
		SELECT
			user_id,
			COUNT(*) as session_count,
			SUM(command_count) as total_commands,
			AVG(duration_seconds) as avg_duration,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_count,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_count
		FROM session_analytics
		WHERE tenant_id = $1 AND user_id = $2
			AND start_time >= NOW() - INTERVAL '1 day' * $3
		GROUP BY user_id
	`

	var metrics struct {
		UserID        uuid.UUID `db:"user_id"`
		SessionCount  int       `db:"session_count"`
		TotalCommands int       `db:"total_commands"`
		AvgDuration   float64   `db:"avg_duration"`
		OffHoursCount int       `db:"off_hours_count"`
		FailedCount   int       `db:"failed_count"`
	}

	err := d.db.GetContext(ctx, &metrics, query, tenantID, userID, int(config.AnalysisWindow.Hours()/24))
	if err != nil {
		return nil, fmt.Errorf("detector: failed to get metrics: %w", err)
	}

	anomalies := []model.AnomalyDetection{}

	// Check for off-hours access anomaly
	if metrics.SessionCount > 0 {
		offHoursRatio := float64(metrics.OffHoursCount) / float64(metrics.SessionCount)
		if offHoursRatio > 0.5 { // More than 50% off-hours
			anomalies = append(anomalies, model.AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(model.AnomalyTypeTemporal),
				UserID:          &userID,
				Severity:        string(model.SeverityMedium),
				ConfidenceScore: 75.0,
				RiskScore:       55.0,
				Title:           "Elevated Off-Hours Access",
				Description:     strPtr(fmt.Sprintf("User accessed system during off-hours %.2f%% of the time", offHoursRatio*100)),
				DetectionMethod: "temporal_analysis",
				DetectedAt:      time.Now(),
				Status:          string(model.AnomalyStatusOpen),
				AutoTriggered:   true,
			})
		}
	}

	// Check for high failure rate
	if metrics.SessionCount > 0 {
		failureRate := float64(metrics.FailedCount) / float64(metrics.SessionCount)
		if failureRate > 0.3 { // More than 30% failure
			anomalies = append(anomalies, model.AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(model.AnomalyTypeBehavioral),
				UserID:          &userID,
				Severity:        string(model.SeverityHigh),
				ConfidenceScore: 80.0,
				RiskScore:       70.0,
				Title:           "Elevated Session Failure Rate",
				Description:     strPtr(fmt.Sprintf("User had %.2f%% session failure rate", failureRate*100)),
				DetectionMethod: "behavioral_analysis",
				DetectedAt:      time.Now(),
				Status:          string(model.AnomalyStatusOpen),
				AutoTriggered:   true,
			})
		}
	}

	return anomalies, nil
}

// DetectVolumetricAnomalies detects volume-based anomalies across all users
func (d *Detector) DetectVolumetricAnomalies(ctx context.Context, tenantID uuid.UUID, config DetectionConfig) ([]model.AnomalyDetection, error) {
	query := `
		SELECT
			user_id,
			COUNT(*) as session_count,
			SUM(command_count) as total_commands
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= NOW() - INTERVAL '1 day'
		GROUP BY user_id
		HAVING COUNT(*) > 0
	`

	rows, err := d.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("detector: failed to query volumetric metrics: %w", err)
	}
	defer rows.Close()

	var userMetrics []struct {
		UserID        uuid.UUID `db:"user_id"`
		SessionCount  int       `db:"session_count"`
		TotalCommands int       `db:"total_commands"`
	}

	for rows.Next() {
		var m struct {
			UserID        uuid.UUID `db:"user_id"`
			SessionCount  int       `db:"session_count"`
			TotalCommands int       `db:"total_commands"`
		}
		if err := rows.Scan(&m.UserID, &m.SessionCount, &m.TotalCommands); err != nil {
			continue
		}
		userMetrics = append(userMetrics, m)
	}

	// Calculate averages
	if len(userMetrics) == 0 {
		return []model.AnomalyDetection{}, nil
	}

	totalSessions := 0
	totalCommands := 0
	for _, m := range userMetrics {
		totalSessions += m.SessionCount
		totalCommands += m.TotalCommands
	}

	avgSessions := float64(totalSessions) / float64(len(userMetrics))
	avgCommands := float64(totalCommands) / float64(len(userMetrics))

	// Calculate standard deviation
	var sessionVariance, commandVariance float64
	for _, m := range userMetrics {
		sessionDiff := float64(m.SessionCount) - avgSessions
		sessionVariance += sessionDiff * sessionDiff
		commandDiff := float64(m.TotalCommands) - avgCommands
		commandVariance += commandDiff * commandDiff
	}

	sessionStdDev := math.Sqrt(sessionVariance / float64(len(userMetrics)))
	commandStdDev := math.Sqrt(commandVariance / float64(len(userMetrics)))

	anomalies := []model.AnomalyDetection{}

	// Find outliers
	for _, m := range userMetrics {
		// Check session count anomaly
		if sessionStdDev > 0 {
			zScore := (float64(m.SessionCount) - avgSessions) / sessionStdDev
			if math.Abs(zScore) > config.ZScoreThreshold {
				severity := string(model.SeverityMedium)
				if math.Abs(zScore) > 4 {
					severity = string(model.SeverityHigh)
				}

				anomalies = append(anomalies, model.AnomalyDetection{
					TenantID:        tenantID,
					AnomalyType:     string(model.AnomalyTypeVolumetric),
					UserID:          &m.UserID,
					Severity:        severity,
					ConfidenceScore: confidenceFromZScore(zScore),
					RiskScore:       riskScoreFromZScore(zScore),
					Title:           "Unusual Session Volume",
					Description:     strPtr(fmt.Sprintf("User initiated %d sessions, deviating from mean of %.2f (z-score: %.2f)", m.SessionCount, avgSessions, zScore)),
					DetectionMethod: "volumetric_analysis",
					DetectedAt:      time.Now(),
					Status:          string(model.AnomalyStatusOpen),
					AutoTriggered:   true,
				})
			}
		}

		// Check command count anomaly
		if commandStdDev > 0 {
			zScore := (float64(m.TotalCommands) - avgCommands) / commandStdDev
			if math.Abs(zScore) > config.ZScoreThreshold {
				anomalies = append(anomalies, model.AnomalyDetection{
					TenantID:        tenantID,
					AnomalyType:     string(model.AnomalyTypeVolumetric),
					UserID:          &m.UserID,
					Severity:        string(model.SeverityMedium),
					ConfidenceScore: confidenceFromZScore(zScore),
					RiskScore:       riskScoreFromZScore(zScore),
					Title:           "Unusual Command Volume",
					Description:     strPtr(fmt.Sprintf("User executed %d commands, deviating from mean of %.2f (z-score: %.2f)", m.TotalCommands, avgCommands, zScore)),
					DetectionMethod: "volumetric_analysis",
					DetectedAt:      time.Now(),
					Status:          string(model.AnomalyStatusOpen),
					AutoTriggered:   true,
				})
			}
		}
	}

	return anomalies, nil
}

// Helper functions

func confidenceFromZScore(zScore float64) float64 {
	absZ := math.Abs(zScore)
	return math.Min(99.9, 50 + (absZ * 10))
}

func riskScoreFromZScore(zScore float64) float64 {
	absZ := math.Abs(zScore)
	return math.Min(100, 40 + (absZ * 12))
}

func strPtr(s string) *string {
	return &s
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
