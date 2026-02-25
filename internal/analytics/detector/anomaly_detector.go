package detector

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	analyticsmodels "github.com/openpam/openpam/internal/analytics/models"
)

var (
	// Anomaly severity levels
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// AnomalyType represents the type of anomaly detected
type AnomalyType string

const (
	AnomalyTypeTime           AnomalyType = "time_anomaly"           // Unusual time of access
	AnomalyTypeLocation       AnomalyType = "location_anomaly"       // Unusual location/IP
	AnomalyTypeDuration       AnomalyType = "duration_anomaly"       // Unusual session duration
	AnomalyTypeResource       AnomalyType = "resource_anomaly"       // Unusual resource access
	AnomalyTypeCommand        AnomalyType = "command_anomaly"        // Unusual commands
	AnomalyTypeVolume         AnomalyType = "volume_anomaly"         // Unusual access volume
	AnomalyTypeUserAgent      AnomalyType = "user_agent_anomaly"     // Unusual user agent
	AnomalyTypeVelocity       AnomalyType = "velocity_anomaly"       // Rapid successive access
	AnomalyTypeImpossible     AnomalyType = "impossible_travel"      // Impossible travel
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	TenantID        uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	UserID          uuid.UUID              `json:"user_id" db:"user_id"`
	SessionID       uuid.UUID              `json:"session_id" db:"session_id"`
	Type            AnomalyType            `json:"type" db:"type"`
	Severity        string                 `json:"severity" db:"severity"`
	Score           float64                `json:"score" db:"score"`           // 0-1
	Confidence      float64                `json:"confidence" db:"confidence"` // 0-1
	Description     string                 `json:"description" db:"description"`
	DetectionMethod string                 `json:"detection_method" db:"detection_method"`
	BaselineValue   float64                `json:"baseline_value" db:"baseline_value"`
	ObservedValue   float64                `json:"observed_value" db:"observed_value"`
	Context         map[string]string      `json:"context" db:"context"`
	AlertSent       bool                   `json:"alert_sent" db:"alert_sent"`
	Resolved        bool                   `json:"resolved" db:"resolved"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy      *uuid.UUID             `json:"resolved_by,omitempty" db:"resolved_by"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// AnomalyDetectorConfig holds configuration for the anomaly detector
type AnomalyDetectorConfig struct {
	ThresholdMedium float64 // Threshold for medium severity (default 0.5)
	ThresholdHigh   float64 // Threshold for high severity (default 0.7)
	ThresholdCritical float64 // Threshold for critical severity (default 0.9)

	EnableTimeDetection      bool
	EnableLocationDetection  bool
	EnableDurationDetection  bool
	EnableResourceDetection  bool
	EnableCommandDetection   bool
	EnableVelocityDetection  bool
	EnableImpossibleTravel   bool

	MaxTravelSpeedKmh float64 // Max realistic travel speed (default 900)
	MinTimeBetweenSessions int // Min minutes between sessions for velocity check
}

// DefaultAnomalyDetectorConfig returns default configuration
func DefaultAnomalyDetectorConfig() AnomalyDetectorConfig {
	return AnomalyDetectorConfig{
		ThresholdMedium:     0.5,
		ThresholdHigh:       0.7,
		ThresholdCritical:   0.9,
		EnableTimeDetection: true,
		EnableLocationDetection: true,
		EnableDurationDetection: true,
		EnableResourceDetection: true,
		EnableCommandDetection: true,
		EnableVelocityDetection: true,
		EnableImpossibleTravel: true,
		MaxTravelSpeedKmh: 900, // About speed of commercial aircraft
		MinTimeBetweenSessions: 1,
	}
}

// AnomalyDetector detects anomalous behavior in session activity
type AnomalyDetector struct {
	db                *sqlx.DB
	profileService    *analyticsmodels.BehaviorProfileService
	config            AnomalyDetectorConfig
	logger            *zerolog.Logger
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(
	db *sqlx.DB,
	profileService *analyticsmodels.BehaviorProfileService,
	logger *zerolog.Logger,
) *AnomalyDetector {
	return &AnomalyDetector{
		db:             db,
		profileService: profileService,
		config:         DefaultAnomalyDetectorConfig(),
		logger:         logger,
	}
}

// AnalyzeSession analyzes a session for anomalies
func (d *AnomalyDetector) AnalyzeSession(ctx context.Context, sample *analyticsmodels.BehaviorSample) ([]Anomaly, error) {
	var anomalies []Anomaly

	// Get or create behavior profile
	profile, err := d.profileService.GetProfile(ctx, sample.TenantID, sample.UserID, "session")
	if err != nil {
		// No profile exists yet, cannot detect anomalies
		return anomalies, nil
	}

	if profile.Status == "stale" || profile.Confidence < 0.5 {
		// Profile not reliable enough
		return anomalies, nil
	}

	// Time anomaly
	if d.config.EnableTimeDetection {
		if anomaly := d.detectTimeAnomaly(sample, profile); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}
	}

	// Duration anomaly
	if d.config.EnableDurationDetection {
		if anomaly := d.detectDurationAnomaly(sample, profile); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}
	}

	// Location anomaly
	if d.config.EnableLocationDetection {
		if anomaly := d.detectLocationAnomaly(sample, profile); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}
	}

	// User agent anomaly
	if anomaly := d.detectUserAgentAnomaly(sample, profile); anomaly != nil {
		anomalies = append(anomalies, *anomaly)
	}

	// Save anomalies
	for _, a := range anomalies {
		if err := d.saveAnomaly(ctx, &a); err != nil {
			d.logger.Error().Err(err).Str("anomaly_id", a.ID.String()).Msg("Failed to save anomaly")
		}
	}

	return anomalies, nil
}

// AnalyzeBehaviorChange analyzes behavior changes over time
func (d *AnomalyDetector) AnalyzeBehaviorChange(ctx context.Context, tenantID, userID uuid.UUID) ([]Anomaly, error) {
	var anomalies []Anomaly

	// Get recent samples
	samples, err := d.profileService.GetSamples(ctx, tenantID, userID, 50)
	if err != nil || len(samples) < 10 {
		return anomalies, nil
	}

	// Check for velocity anomalies - rapid successive access
	if d.config.EnableVelocityDetection {
		if anomaly := d.detectVelocityAnomaly(samples, tenantID, userID); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}
	}

	// Check for impossible travel
	if d.config.EnableImpossibleTravel {
		if anomaly := d.detectImpossibleTravel(samples, tenantID, userID); anomaly != nil {
			anomalies = append(anomalies, *anomaly)
		}
	}

	return anomalies, nil
}

// detectTimeAnomaly detects unusual time of access
func (d *AnomalyDetector) detectTimeAnomaly(sample *analyticsmodels.BehaviorSample, profile *analyticsmodels.BehaviorProfile) *Anomaly {
	hour := sample.HourOfDay
	day := sample.DayOfWeek

	// Check if hour is in peak hours
	isPeakHour := false
	for _, ph := range profile.PeakHours {
		if ph == hour {
			isPeakHour = true
			break
		}
	}

	// Check if day is in peak days
	isPeakDay := false
	for _, pd := range profile.PeakDays {
		if pd == day {
			isPeakDay = true
			break
		}
	}

	// Calculate anomaly score
	// High anomaly if both time and day are unusual
	var score float64
	if !isPeakHour && !isPeakDay {
		score = 0.8
	} else if !isPeakHour || !isPeakDay {
		score = 0.5
	} else {
		return nil // Normal time
	}

	return &Anomaly{
		ID:              uuid.New(),
		TenantID:        sample.TenantID,
		UserID:          sample.UserID,
		SessionID:       sample.SessionID,
		Type:            AnomalyTypeTime,
		Severity:        d.calculateSeverity(score),
		Score:           score,
		Confidence:      profile.Confidence,
		Description:     fmt.Sprintf("Access at unusual time: %02d:00 on day %d", hour, day),
		DetectionMethod: "statistical",
		BaselineValue:   float64(profile.PeakHours[0]), // First peak hour as baseline
		ObservedValue:   float64(hour),
		Context: map[string]string{
			"hour":       fmt.Sprintf("%d", hour),
			"day_of_week": fmt.Sprintf("%d", day),
			"is_peak_hour": fmt.Sprintf("%t", isPeakHour),
			"is_peak_day":  fmt.Sprintf("%t", isPeakDay),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// detectDurationAnomaly detects unusual session duration
func (d *AnomalyDetector) detectDurationAnomaly(sample *analyticsmodels.BehaviorSample, profile *analyticsmodels.BehaviorProfile) *Anomaly {
	if profile.AverageSessionDuration == 0 {
		return nil
	}

	// Calculate z-score
	avg := float64(profile.AverageSessionDuration)
	stddev := profile.SessionDurationStdDev
	if stddev == 0 {
		stddev = 1 // Avoid division by zero
	}

	zScore := math.Abs(float64(sample.Duration)-avg) / stddev

	// Anomaly if duration is more than 2 standard deviations away
	if zScore < 2 {
		return nil
	}

	score := math.Min(0.95, zScore/4) // Cap at 0.95, scale z-score

	return &Anomaly{
		ID:              uuid.New(),
		TenantID:        sample.TenantID,
		UserID:          sample.UserID,
		SessionID:       sample.SessionID,
		Type:            AnomalyTypeDuration,
		Severity:        d.calculateSeverity(score),
		Score:           score,
		Confidence:      profile.Confidence,
		Description:     fmt.Sprintf("Unusual session duration: %d minutes (avg: %d)", sample.Duration, profile.AverageSessionDuration),
		DetectionMethod: "zscore",
		BaselineValue:   avg,
		ObservedValue:   float64(sample.Duration),
		Context: map[string]string{
			"z_score":       fmt.Sprintf("%.2f", zScore),
			"avg_duration":  fmt.Sprintf("%d", profile.AverageSessionDuration),
			"stddev":        fmt.Sprintf("%.2f", stddev),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// detectLocationAnomaly detects unusual location/IP access
func (d *AnomalyDetector) detectLocationAnomaly(sample *analyticsmodels.BehaviorSample, profile *analyticsmodels.BehaviorProfile) *Anomaly {
	// Check if IP is in typical IPs
	isTypicalIP := false
	for _, ip := range profile.TypicalIPs {
		if ip == sample.IPAddress {
			isTypicalIP = true
			break
		}
	}

	if isTypicalIP {
		return nil
	}

	// High severity for new IP
	score := 0.7

	return &Anomaly{
		ID:              uuid.New(),
		TenantID:        sample.TenantID,
		UserID:          sample.UserID,
		SessionID:       sample.SessionID,
		Type:            AnomalyTypeLocation,
		Severity:        d.calculateSeverity(score),
		Score:           score,
		Confidence:      profile.Confidence,
		Description:     fmt.Sprintf("Access from new IP address: %s", sample.IPAddress),
		DetectionMethod: "whitelist",
		BaselineValue:   0,
		ObservedValue:   1,
		Context: map[string]string{
			"ip_address": sample.IPAddress,
			"location":   sample.Location,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// detectUserAgentAnomaly detects unusual user agent
func (d *AnomalyDetector) detectUserAgentAnomaly(sample *analyticsmodels.BehaviorSample, profile *analyticsmodels.BehaviorProfile) *Anomaly {
	// Check if user agent is in typical user agents
	isTypicalUA := false
	for _, ua := range profile.TypicalUserAgents {
		if ua == sample.UserAgent {
			isTypicalUA = true
			break
		}
	}

	if isTypicalUA {
		return nil
	}

	score := 0.6

	return &Anomaly{
		ID:              uuid.New(),
		TenantID:        sample.TenantID,
		UserID:          sample.UserID,
		SessionID:       sample.SessionID,
		Type:            AnomalyTypeUserAgent,
		Severity:        SeverityMedium,
		Score:           score,
		Confidence:      profile.Confidence,
		Description:     fmt.Sprintf("Access from new user agent"),
		DetectionMethod: "whitelist",
		Context: map[string]string{
			"user_agent": sample.UserAgent,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// detectVelocityAnomaly detects rapid successive access (possible automated attack)
func (d *AnomalyDetector) detectVelocityAnomaly(samples []analyticsmodels.BehaviorSample, tenantID, userID uuid.UUID) *Anomaly {
	if len(samples) < 5 {
		return nil
	}

	// Count sessions in the last hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentCount := 0
	for _, s := range samples {
		if s.Timestamp.After(oneHourAgo) {
			recentCount++
		}
	}

	// More than 10 sessions in an hour is suspicious
	if recentCount <= 10 {
		return nil
	}

	score := math.Min(0.95, float64(recentCount)/50) // Scale up to 50 sessions/hour

	return &Anomaly{
		ID:              uuid.New(),
		TenantID:        tenantID,
		UserID:          userID,
		Type:            AnomalyTypeVelocity,
		Severity:        d.calculateSeverity(score),
		Score:           score,
		Confidence:      0.8,
		Description:     fmt.Sprintf("High access velocity: %d sessions in the last hour", recentCount),
		DetectionMethod: "velocity",
		BaselineValue:   5,
		ObservedValue:   float64(recentCount),
		Context: map[string]string{
			"recent_count": fmt.Sprintf("%d", recentCount),
			"window":       "1 hour",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// detectImpossibleTravel detects impossible travel between locations
func (d *AnomalyDetector) detectImpossibleTravel(samples []analyticsmodels.BehaviorSample, tenantID, userID uuid.UUID) *Anomaly {
	if len(samples) < 2 {
		return nil
	}

	// Sort samples by timestamp and check for impossible travel
	for i := 0; i < len(samples)-1; i++ {
		s1 := samples[i]
		s2 := samples[i+1]

		// Time difference in hours
		timeDiff := s1.Timestamp.Sub(s2.Timestamp).Hours()
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff < 0.5 { // Less than 30 minutes - skip
			continue
		}

		// Simple check: if locations differ significantly within short time
		// In production, use actual geolocation distance calculation
		if s1.Location != s2.Location && s1.Location != "" && s2.Location != "" {
			// If time difference is less than 2 hours and locations differ
			if timeDiff < 2 {
				score := 0.9

				return &Anomaly{
					ID:              uuid.New(),
					TenantID:        tenantID,
					UserID:          userID,
					Type:            AnomalyTypeImpossible,
					Severity:        SeverityCritical,
					Score:           score,
					Confidence:      0.9,
					Description:     fmt.Sprintf("Impossible travel detected: %s to %s in %.1f hours", s1.Location, s2.Location, timeDiff),
					DetectionMethod: "geospatial",
					Context: map[string]string{
						"location1":     s1.Location,
						"location2":     s2.Location,
						"time_diff":     fmt.Sprintf("%.1f hours", timeDiff),
						"session1_id":   s1.SessionID.String(),
						"session2_id":   s2.SessionID.String(),
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
			}
		}
	}

	return nil
}

// calculateSeverity converts a score to a severity level
func (d *AnomalyDetector) calculateSeverity(score float64) string {
	if score >= d.config.ThresholdCritical {
		return SeverityCritical
	}
	if score >= d.config.ThresholdHigh {
		return SeverityHigh
	}
	if score >= d.config.ThresholdMedium {
		return SeverityMedium
	}
	return SeverityLow
}

// saveAnomaly saves an anomaly to the database
func (d *AnomalyDetector) saveAnomaly(ctx context.Context, anomaly *Anomaly) error {
	anomaly.CreatedAt = time.Now()
	anomaly.UpdatedAt = time.Now()

	query := `
		INSERT INTO anomalies (
			id, tenant_id, user_id, session_id, type, severity,
			score, confidence, description, detection_method,
			baseline_value, observed_value, context,
			alert_sent, resolved, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :user_id, :session_id, :type, :severity,
			:score, :confidence, :description, :detection_method,
			:baseline_value, :observed_value, :context,
			:alert_sent, :resolved, :created_at, :updated_at
		)
	`

	_, err := d.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("save anomaly: %w", err)
	}

	return nil
}

// GetAnomalies retrieves anomalies for a user
func (d *AnomalyDetector) GetAnomalies(ctx context.Context, tenantID, userID uuid.UUID, includeResolved bool, limit int) ([]Anomaly, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, type, severity,
			score, confidence, description, detection_method,
			baseline_value, observed_value, context,
			alert_sent, resolved, resolved_at, resolved_by,
			created_at, updated_at
		FROM anomalies
		WHERE tenant_id = $1 AND user_id = $2
	`

	if !includeResolved {
		query += " AND resolved = false"
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	} else {
		query += " LIMIT 100"
	}

	var anomalies []Anomaly
	err := d.db.SelectContext(ctx, &anomalies, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("get anomalies: %w", err)
	}

	return anomalies, nil
}

// ResolveAnomaly marks an anomaly as resolved
func (d *AnomalyDetector) ResolveAnomaly(ctx context.Context, anomalyID uuid.UUID, resolvedBy uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE anomalies
		SET resolved = true, resolved_at = $1, resolved_by = $2, updated_at = $1
		WHERE id = $3
	`

	_, err := d.db.ExecContext(ctx, query, now, resolvedBy, anomalyID)
	if err != nil {
		return fmt.Errorf("resolve anomaly: %w", err)
	}

	return nil
}
