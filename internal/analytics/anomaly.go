package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AnomalyDetector handles anomaly detection using statistical analysis
type AnomalyDetector struct {
	repo     Repository
	cache    *RedisCache
	threshold float64
	logger   zerolog.Logger
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(repo Repository, cache *RedisCache, threshold float64, logger zerolog.Logger) *AnomalyDetector {
	return &AnomalyDetector{
		repo:     repo,
		cache:    cache,
		threshold: threshold,
		logger:   logger,
	}
}

// AnomalyPattern represents a detected anomaly pattern
type AnomalyPattern struct {
	Type        AnomalyType
	Severity    RiskLevel
	Confidence  float64
	Title       string
	Description string
	Indicators  map[string]interface{}
}

// RunDetection runs anomaly detection for a tenant
func (d *AnomalyDetector) RunDetection(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	// Run different detection strategies
	if behavioral, err := d.detectBehavioralAnomalies(ctx, tenantID); err == nil {
		detections = append(detections, behavioral...)
	}

	if temporal, err := d.detectTemporalAnomalies(ctx, tenantID); err == nil {
		detections = append(detections, temporal...)
	}

	if spatial, err := d.detectSpatialAnomalies(ctx, tenantID); err == nil {
		detections = append(detections, spatial...)
	}

	if volumetric, err := d.detectVolumetricAnomalies(ctx, tenantID); err == nil {
		detections = append(detections, volumetric...)
	}

	if pattern, err := d.detectPatternAnomalies(ctx, tenantID); err == nil {
		detections = append(detections, pattern...)
	}

	// Check for ransomware indicators
	if ransomware, err := d.detectRansomwareIndicators(ctx, tenantID); err == nil {
		detections = append(detections, ransomware...)
	}

	// Save detections
	for _, detection := range detections {
		if detection.ConfidenceScore >= d.threshold {
			if err := d.repo.CreateAnomalyDetection(ctx, &detection); err != nil {
				d.logger.Error().Err(err).Str("anomaly_id", detection.ID.String()).Msg("Failed to save anomaly detection")
			}
		}
	}

	return detections, nil
}

// detectBehavioralAnomalies detects unusual user behavior patterns
func (d *AnomalyDetector) detectBehavioralAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	// Get users with high risk scores
	if repo, ok := d.repo.(*PostgresRepository); ok {
		// Query for high-risk users from user activity
		query := `
			SELECT user_id, AVG(risk_score) as avg_risk,
				SUM(CASE WHEN off_hours_access THEN 1 ELSE 0 END) as off_hours,
				SUM(CASE WHEN unusual_access THEN 1 ELSE 0 END) as unusual
			FROM user_activity
			WHERE tenant_id = $1 AND date >= CURRENT_DATE - INTERVAL '7 days'
			GROUP BY user_id
			HAVING AVG(risk_score) > 50 OR SUM(CASE WHEN unusual_access THEN 1 ELSE 0 END) > 3
		`

		rows, err := repo.db.QueryContext(ctx, query, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			var avgRisk float64
			var offHours, unusual int

			if err := rows.Scan(&userID, &avgRisk, &offHours, &unusual); err != nil {
				continue
			}

			// Create anomaly detection
			confidence := calculateBehavioralConfidence(avgRisk, offHours, unusual)
			if confidence < d.threshold {
				continue
			}

			detection := AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypeBehavioral),
				UserID:          &userID,
				Severity:        determineSeverity(confidence),
				ConfidenceScore: confidence,
				RiskScore:       avgRisk,
				Title:           "Unusual User Behavior Detected",
				Description:     stringPtr(fmt.Sprintf("User exhibiting risky behavior patterns (risk score: %.1f)", avgRisk)),
				DetectionMethod: "statistical_analysis",
				Status:          string(AnomalyStatusOpen),
			}

			indicators := map[string]interface{}{
				"avg_risk_score":   avgRisk,
				"off_hours_access": offHours,
				"unusual_access":   unusual,
			}
			indicatorsJSON, _ := json.Marshal(indicators)
			detection.Indicators = indicatorsJSON

			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// detectTemporalAnomalies detects anomalies in access time patterns
func (d *AnomalyDetector) detectTemporalAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	if repo, ok := d.repo.(*PostgresRepository); ok {
		// Query for users accessing during unusual times
		query := `
			SELECT user_id, COUNT(*) as count, ARRAY_AGG(DISTINCT hour) as hours
			FROM user_activity
			WHERE tenant_id = $1
				AND date >= CURRENT_DATE - INTERVAL '30 days'
				AND off_hours_access = true
			GROUP BY user_id
			HAVING COUNT(*) > 10
		`

		rows, err := repo.db.QueryContext(ctx, query, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			var count int
			var hours []int

			if err := rows.Scan(&userID, &count, &hours); err != nil {
				continue
			}

			confidence := math.Min(95.0, float64(count)*5)
			if confidence < d.threshold {
				continue
			}

			detection := AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypeTemporal),
				UserID:          &userID,
				Severity:        determineSeverity(confidence),
				ConfidenceScore: confidence,
				RiskScore:       confidence * 0.8,
				Title:           "Off-Hours Access Pattern",
				Description:     stringPtr(fmt.Sprintf("User has accessed systems %d times during off-hours", count)),
				DetectionMethod: "temporal_analysis",
				Status:          string(AnomalyStatusOpen),
			}

			indicators := map[string]interface{}{
				"off_hours_count": count,
				"hours_accessed":  hours,
			}
			indicatorsJSON, _ := json.Marshal(indicators)
			detection.Indicators = indicatorsJSON

			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// detectSpatialAnomalies detects anomalies in access location patterns
func (d *AnomalyDetector) detectSpatialAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	if repo, ok := d.repo.(*PostgresRepository); ok {
		// Query for access from unusual locations (if geo data is available)
		query := `
			SELECT user_id, country_code, COUNT(*) as count
			FROM user_activity
			WHERE tenant_id = $1
				AND date >= CURRENT_DATE - INTERVAL '30 days'
				AND country_code IS NOT NULL
				AND unusual_access = true
			GROUP BY user_id, country_code
			HAVING COUNT(*) > 5
		`

		rows, err := repo.db.QueryContext(ctx, query, tenantID)
		if err != nil {
			// If geo data not available, return empty
			return detections, nil
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			var countryCode string
			var count int

			if err := rows.Scan(&userID, &countryCode, &count); err != nil {
				continue
			}

			confidence := math.Min(90.0, float64(count)*8)
			if confidence < d.threshold {
				continue
			}

			detection := AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypeSpatial),
				UserID:          &userID,
				Severity:        determineSeverity(confidence),
				ConfidenceScore: confidence,
				RiskScore:       confidence * 0.7,
				Title:           "Unusual Access Location",
				Description:     stringPtr(fmt.Sprintf("Unusual access patterns from location: %s", countryCode)),
				DetectionMethod: "spatial_analysis",
				Status:          string(AnomalyStatusOpen),
			}

			indicators := map[string]interface{}{
				"country_code": countryCode,
				"access_count": count,
			}
			indicatorsJSON, _ := json.Marshal(indicators)
			detection.Indicators = indicatorsJSON

			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// detectVolumetricAnomalies detects unusual volumes of activity
func (d *AnomalyDetector) detectVolumetricAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	if repo, ok := d.repo.(*PostgresRepository); ok {
		// Query for unusual command execution volumes
		query := `
			SELECT user_id, date, SUM(commands_executed) as total_commands,
				AVG(commands_executed) OVER (PARTITION BY user_id) as avg_commands
			FROM user_activity
			WHERE tenant_id = $1 AND date >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY user_id, date, commands_executed
			HAVING SUM(commands_executed) > 3 * (
				SELECT AVG(commands_executed)
				FROM user_activity
				WHERE tenant_id = $1
			) OR SUM(commands_executed) > 1000
		`

		rows, err := repo.db.QueryContext(ctx, query, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			var date time.Time
			var totalCommands, avgCommands int

			if err := rows.Scan(&userID, &date, &totalCommands, &avgCommands); err != nil {
				continue
			}

			// Calculate z-score-like metric
			stdDev := float64(avgCommands) * 0.3
			if stdDev == 0 {
				stdDev = 1
			}
			zScore := math.Abs(float64(totalCommands-avgCommands) / stdDev)
			confidence := math.Min(98.0, zScore*20)

			if confidence < d.threshold {
				continue
			}

			detection := AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypeVolumetric),
				UserID:          &userID,
				Severity:        determineSeverity(confidence),
				ConfidenceScore: confidence,
				RiskScore:       confidence * 0.6,
				Title:           "Unusual Command Volume",
				Description:     stringPtr(fmt.Sprintf("Executed %d commands (avg: %d) on %s", totalCommands, avgCommands, date.Format("2006-01-02"))),
				DetectionMethod: "volumetric_analysis",
				Status:          string(AnomalyStatusOpen),
			}

			indicators := map[string]interface{}{
				"total_commands": totalCommands,
				"avg_commands":   avgCommands,
				"date":           date.Format("2006-01-02"),
				"z_score":        zScore,
			}
			indicatorsJSON, _ := json.Marshal(indicators)
			detection.Indicators = indicatorsJSON

			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// detectPatternAnomalies detects suspicious command patterns
func (d *AnomalyDetector) detectPatternAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	if repo, ok := d.repo.(*PostgresRepository); ok {
		// Query for dangerous command patterns
		query := `
			SELECT user_id, base_command, COUNT(*) as count,
				SUM(CASE WHEN is_dangerous THEN 1 ELSE 0 END) as dangerous_count
			FROM command_frequency
			WHERE tenant_id = $1
				AND date >= CURRENT_DATE - INTERVAL '7 days'
				AND (is_dangerous = true OR risk_level IN ('high', 'critical'))
			GROUP BY user_id, base_command
			HAVING COUNT(*) > 5 OR SUM(CASE WHEN is_dangerous THEN 1 ELSE 0 END) > 3
		`

		rows, err := repo.db.QueryContext(ctx, query, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var userID uuid.UUID
			var baseCommand string
			var count, dangerousCount int

			if err := rows.Scan(&userID, &baseCommand, &count, &dangerousCount); err != nil {
				continue
			}

			confidence := math.Min(99.0, float64(dangerousCount)*25)
			if confidence < d.threshold {
				continue
			}

			detection := AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypePattern),
				UserID:          &userID,
				Severity:        string(RiskLevelHigh),
				ConfidenceScore: confidence,
				RiskScore:       confidence * 0.9,
				Title:           "Suspicious Command Pattern",
				Description:     stringPtr(fmt.Sprintf("User executed dangerous command '%s' %d times", baseCommand, count)),
				DetectionMethod: "pattern_analysis",
				Status:          string(AnomalyStatusOpen),
			}

			indicators := map[string]interface{}{
				"base_command":     baseCommand,
				"execution_count":  count,
				"dangerous_count":  dangerousCount,
			}
			indicatorsJSON, _ := json.Marshal(indicators)
			detection.Indicators = indicatorsJSON

			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// detectRansomwareIndicators detects potential ransomware activity
func (d *AnomalyDetector) detectRansomwareIndicators(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	if repo, ok := d.repo.(*PostgresRepository); ok {
		// Query for ransomware-like command patterns
		ransomwareCommands := []string{
			"chmod", "chown", "find", "wget", "curl",
			"nc", "netcat", "ssh-keygen", "base64",
		}

		for _, cmd := range ransomwareCommands {
			query := `
				SELECT user_id, COUNT(*) as count,
					STRING_AGG(DISTINCT target_host, ', ') as targets
				FROM command_frequency
				WHERE tenant_id = $1
					AND date >= CURRENT_DATE - INTERVAL '1 day'
					AND base_command = $2
				GROUP BY user_id
				HAVING COUNT(*) > 50
			`

			rows, err := repo.db.QueryContext(ctx, query, tenantID, cmd)
			if err != nil {
				continue
			}

			for rows.Next() {
				var userID uuid.UUID
				var count int
				var targets string

				if err := rows.Scan(&userID, &count, &targets); err != nil {
					continue
				}

				confidence := math.Min(100.0, float64(count)*1.5)
				if confidence < d.threshold {
					continue
				}

				detection := AnomalyDetection{
					TenantID:        tenantID,
					AnomalyType:     string(AnomalyTypeRansomware),
					UserID:          &userID,
					Severity:        string(RiskLevelCritical),
					ConfidenceScore: confidence,
					RiskScore:       100.0,
					Title:           "Potential Ransomware Activity",
					Description:     stringPtr(fmt.Sprintf("High volume of '%s' commands detected - possible ransomware precursor", cmd)),
					DetectionMethod: "ransomware_heuristic",
					Status:          string(AnomalyStatusOpen),
					AutoTriggered:   true,
				}

				indicators := map[string]interface{}{
					"command":         cmd,
					"execution_count": count,
					"targets":         targets,
				}
				indicatorsJSON, _ := json.Marshal(indicators)
				detection.Indicators = indicatorsJSON

				detections = append(detections, detection)

				// Create ransomware event
				ransomwareEvent := &RansomwareEvent{
					SuspiciousProcesses: []string{cmd},
				}
				_ = d.repo.CreateRansomwareEvent(ctx, ransomwareEvent)
			}
			rows.Close()
		}
	}

	return detections, nil
}

// EvaluateUser evaluates a specific user for anomalies
func (d *AnomalyDetector) EvaluateUser(ctx context.Context, tenantID, userID uuid.UUID) ([]AnomalyDetection, error) {
	var detections []AnomalyDetection

	// Get user activity for the past 7 days
	filter := UserActivityFilter{
		TenantID: &tenantID,
		UserID:   &userID,
		DateFrom: timePtr(time.Now().AddDate(0, 0, -7)),
		DateTo:   timePtr(time.Now()),
	}

	activities, err := d.repo.ListUserActivity(ctx, filter, 0, 0)
	if err != nil {
		return nil, err
	}

	// Analyze patterns
	var totalRiskScore float64
	var offHoursCount, unusualCount int
	var commandCounts []int

	for _, activity := range activities {
		totalRiskScore += activity.RiskScore
		if activity.OffHoursAccess {
			offHoursCount++
		}
		if activity.UnusualAccess {
			unusualCount++
		}
		commandCounts = append(commandCounts, activity.CommandsExecuted)
	}

	// Calculate metrics
	if len(activities) > 0 {
		avgRisk := totalRiskScore / float64(len(activities))

		// Check for high risk
		if avgRisk > 50 {
			detection := AnomalyDetection{
				ID:              uuid.New(),
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypeBehavioral),
				UserID:          &userID,
				Severity:        determineSeverity(avgRisk),
				ConfidenceScore: avgRisk,
				RiskScore:       avgRisk,
				Title:           "High User Risk Score",
				Description:     stringPtr(fmt.Sprintf("User average risk score of %.1f exceeds threshold", avgRisk)),
				DetectionMethod: "user_evaluation",
				Status:          string(AnomalyStatusOpen),
			}
			detections = append(detections, detection)
		}

		// Check for off-hours access
		if offHoursCount > 3 {
			detection := AnomalyDetection{
				ID:              uuid.New(),
				TenantID:        tenantID,
				AnomalyType:     string(AnomalyTypeTemporal),
				UserID:          &userID,
				Severity:        string(RiskLevelMedium),
				ConfidenceScore: float64(offHoursCount) * 15,
				RiskScore:       float64(offHoursCount) * 12,
				Title:           "Excessive Off-Hours Access",
				Description:     stringPtr(fmt.Sprintf("User accessed systems %d times during off-hours period", offHoursCount)),
				DetectionMethod: "user_evaluation",
				Status:          string(AnomalyStatusOpen),
			}
			detections = append(detections, detection)
		}
	}

	return detections, nil
}

// EvaluateCommand evaluates a single command for anomalies
func (d *AnomalyDetector) EvaluateCommand(ctx context.Context, tenantID, userID uuid.UUID, command *ParsedCommand) error {
	// Check if command is highly dangerous
	if command.RiskLevel == string(RiskLevelCritical) {
		detection := AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(AnomalyTypePattern),
			UserID:          &userID,
			Severity:        string(RiskLevelCritical),
			ConfidenceScore: 95.0,
			RiskScore:       100.0,
			Title:           "Critical Risk Command Executed",
			Description:     stringPtr(fmt.Sprintf("Executed critical risk command: %s", command.Pattern)),
			DetectionMethod: "command_evaluation",
			Status:          string(AnomalyStatusOpen),
		}

		indicators := map[string]interface{}{
			"command":      command.Normalized,
			"base_command": command.BaseCommand,
		}
		indicatorsJSON, _ := json.Marshal(indicators)
		detection.Indicators = indicatorsJSON

		return d.repo.CreateAnomalyDetection(ctx, &detection)
	}

	return nil
}

// Helper functions

func calculateBehavioralConfidence(avgRisk float64, offHours, unusual int) float64 {
	confidence := avgRisk * 0.6
	confidence += float64(offHours) * 5
	confidence += float64(unusual) * 10
	return math.Min(100.0, confidence)
}

func determineSeverity(confidence float64) string {
	if confidence >= 90 {
		return string(RiskLevelCritical)
	}
	if confidence >= 75 {
		return string(RiskLevelHigh)
	}
	if confidence >= 50 {
		return string(RiskLevelMedium)
	}
	return string(RiskLevelLow)
}

// Ransomware Detection Methods

// CheckRansomwareIndicators checks for specific ransomware indicators
func (d *AnomalyDetector) CheckRansomwareIndicators(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (*RansomwareEvent, error) {
	event := &RansomwareEvent{
		TenantID:        tenantID,
		EncryptionActivity: false,
		MassFileModification: false,
	}

	// Check for mass file modification commands
	if repo, ok := d.repo.(*PostgresRepository); ok {
		query := `
			SELECT base_command, COUNT(*) as count
			FROM command_frequency
			WHERE tenant_id = $1 AND user_id = $2
				AND date >= CURRENT_DATE - INTERVAL '1 hour'
				AND base_command IN ('chmod', 'chown', 'rm', 'find', 'mv', 'cp')
			GROUP BY base_command
			HAVING COUNT(*) > 100
		`

		rows, err := repo.db.QueryContext(ctx, query, tenantID, userID)
		if err == nil {
			var processes []string
			totalCount := 0

			for rows.Next() {
				var cmd string
				var count int
				if rows.Scan(&cmd, &count) == nil {
					processes = append(processes, cmd)
					totalCount += count
				}
			}
			rows.Close()

			if totalCount > 500 {
				event.MassFileModification = true
				event.SuspiciousProcesses = processes
				event.FilesAffected = totalCount
			}
		}
	}

	// If significant indicators found, create detection
	if event.MassFileModification || event.EncryptionActivity {
		detection := AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(AnomalyTypeRansomware),
			UserID:          &userID,
			Severity:        string(RiskLevelCritical),
			ConfidenceScore: 95.0,
			RiskScore:       100.0,
			Title:           "Ransomware Indicators Detected",
			Description:     stringPtr("Detected multiple ransomware precursor activities"),
			DetectionMethod: "ransomware_detection",
			Status:          string(AnomalyStatusOpen),
			AutoTriggered:   true,
		}

		if err := d.repo.CreateAnomalyDetection(ctx, &detection); err != nil {
			return nil, err
		}

		event.DetectionID = detection.ID
		_ = d.repo.CreateRansomwareEvent(ctx, event)

		return event, nil
	}

	return nil, nil
}

// TriggerEmergencyWorkflow triggers emergency response for ransomware
func (d *AnomalyDetector) TriggerEmergencyWorkflow(ctx context.Context, eventID uuid.UUID) error {
	event, err := d.repo.GetRansomwareEvent(ctx, eventID)
	if err != nil {
		return err
	}

	event.EmergencyTriggered = true

	// Update containment status
	status := "isolating"
	event.ContainmentStatus = &status

	return d.repo.UpdateRansomwareEvent(ctx, event)
}

// GetAnomalyTrends returns trends in anomaly detections
func (d *AnomalyDetector) GetAnomalyTrends(ctx context.Context, tenantID uuid.UUID, days int) (map[string]int, error) {
	dateFrom := time.Now().AddDate(0, 0, -days)

	filter := AnomalyFilter{
		TenantID: &tenantID,
		DateFrom: &dateFrom,
	}

	detections, err := d.repo.ListAnomalyDetections(ctx, filter, 0, 0)
	if err != nil {
		return nil, err
	}

	trends := make(map[string]int)
	for _, detection := range detections {
		trends[detection.AnomalyType]++
	}

	return trends, nil
}

// GetUserAnomalyHistory returns anomaly history for a user
func (d *AnomalyDetector) GetUserAnomalyHistory(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]AnomalyDetection, error) {
	filter := AnomalyFilter{
		TenantID: &tenantID,
		UserID:   &userID,
	}

	return d.repo.ListAnomalyDetections(ctx, filter, limit, 0)
}
