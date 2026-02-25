package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// AlertManager handles alert generation and notification
type AlertManager struct {
	db        *sqlx.DB
	cache     *cache.Cache
	publisher *events.Publisher
	logger    zerolog.Logger
}

// Config holds alert manager configuration
type Config struct {
	RetryAttempts   int
	RetryDelay      time.Duration
	BatchSize       int
	MaxQueueSize    int
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
	SeverityInfo     AlertSeverity = "info"
)

// AlertType represents the type of alert
type AlertType string

const (
	TypeAnomalyDetected      AlertType = "anomaly_detected"
	TypeSessionAnomaly       AlertType = "session_anomaly"
	TypeUserAnomaly          AlertType = "user_anomaly"
	TypeCommandAnomaly       AlertType = "command_anomaly"
	TypeAccessViolation      AlertType = "access_violation"
	TypePolicyViolation      AlertType = "policy_violation"
	TypeThresholdExceeded    AlertType = "threshold_exceeded"
	TypeSecurityEvent        AlertType = "security_event"
	TypeComplianceViolation  AlertType = "compliance_violation"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	StatusOpen    AlertStatus = "open"
	StatusAcked   AlertStatus = "acknowledged"
	StatusResolved AlertStatus = "resolved"
	StatusFalsePositive AlertStatus = "false_positive"
)

// Alert represents a security alert
type Alert struct {
	ID          uuid.UUID     `db:"id" json:"id"`
	TenantID    uuid.UUID     `db:"tenant_id" json:"tenant_id"`
	Type        AlertType     `db:"type" json:"type"`
	Severity    AlertSeverity `db:"severity" json:"severity"`
	Status      AlertStatus   `db:"status" json:"status"`
	Title       string        `db:"title" json:"title"`
	Description string        `db:"description" json:"description"`
	Source      string        `db:"source" json:"source"`

	// Context
	UserID      *uuid.UUID              `db:"user_id" json:"user_id,omitempty"`
	SessionID   *uuid.UUID              `db:"session_id" json:"session_id,omitempty"`
	TargetHost  *string                 `db:"target_host" json:"target_host,omitempty"`
	Metadata    map[string]interface{}  `db:"metadata" json:"metadata"`

	// Scoring
	RiskScore   float64 `db:"risk_score" json:"risk_score"`
	Confidence  float64 `db:"confidence" json:"confidence"`

	// Escalation
	EscalationLevel int       `db:"escalation_level" json:"escalation_level"`
	EscalatedAt     *time.Time `db:"escalated_at" json:"escalated_at,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	AckedAt     *time.Time `db:"acked_at" json:"acked_at,omitempty"`
	AckedBy     *uuid.UUID `db:"acked_by" json:"acked_by,omitempty"`
	ResolvedAt  *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy  *uuid.UUID `db:"resolved_by" json:"resolved_by,omitempty"`
}

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID          uuid.UUID     `db:"id" json:"id"`
	TenantID    uuid.UUID     `db:"tenant_id" json:"tenant_id"`
	Name        string        `db:"name" json:"name"`
	Type        AlertType     `db:"type" json:"type"`
	Description string        `db:"description" json:"description"`
	Enabled     bool          `db:"enabled" json:"enabled"`
	Conditions  RuleCondition `db:"conditions" json:"conditions"`
	Actions     []RuleAction  `db:"actions" json:"actions"`
	Severity    AlertSeverity `db:"severity" json:"severity"`
	CreatedAt   time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at" json:"updated_at"`
}

// RuleCondition defines when an alert should be triggered
type RuleCondition struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"` // gt, lt, eq, ne, contains, regex
	Value     interface{} `json:"value"`
	Duration  *time.Duration `json:"duration,omitempty"` // How long condition must be true
}

// RuleAction defines what to do when an alert is triggered
type RuleAction struct {
	Type    string                 `json:"type"` // notify, block, escalate, webhook
	Config  map[string]interface{} `json:"config"`
}

// NotificationChannel represents a channel for sending alert notifications
type NotificationChannel struct {
	ID          uuid.UUID             `db:"id" json:"id"`
	TenantID    uuid.UUID             `db:"tenant_id" json:"tenant_id"`
	Name        string                `db:"name" json:"name"`
	Type        string                `db:"type" json:"type"` // email, slack, webhook, pagerduty
	Config      map[string]interface{} `db:"config" json:"config"`
	Enabled     bool                  `db:"enabled" json:"enabled"`
	CreatedAt   time.Time             `db:"created_at" json:"created_at"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(db *sqlx.DB, c *cache.Cache, publisher *events.Publisher, logger zerolog.Logger) *AlertManager {
	return &AlertManager{
		db:        db,
		cache:     c,
		publisher: publisher,
		logger:    logger,
	}
}

// CreateAlert creates a new alert
func (m *AlertManager) CreateAlert(ctx context.Context, alert *Alert) error {
	alert.ID = uuid.New()
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	if alert.Status == "" {
		alert.Status = StatusOpen
	}
	if alert.EscalationLevel == 0 {
		alert.EscalationLevel = 1
	}

	metadataJSON, _ := json.Marshal(alert.Metadata)

	query := `
		INSERT INTO alerts (
			id, tenant_id, type, severity, status, title, description, source,
			user_id, session_id, target_host, metadata, risk_score, confidence,
			escalation_level, created_at, updated_at
		) VALUES (
			$id, $tenant_id, $type, $severity, $status, $title, $description, $source,
			$user_id, $session_id, $target_host, $metadata, $risk_score, $confidence,
			$escalation_level, $created_at, $updated_at
		)
	`

	_, err := m.db.ExecContext(ctx, query,
		alert.ID, alert.TenantID, alert.Type, alert.Severity, alert.Status,
		alert.Title, alert.Description, alert.Source,
		alert.UserID, alert.SessionID, alert.TargetHost, metadataJSON,
		alert.RiskScore, alert.Confidence, alert.EscalationLevel,
		alert.CreatedAt, alert.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("alert_manager.CreateAlert: %w", err)
	}

	// Send notifications
	if err := m.sendNotifications(ctx, alert); err != nil {
		m.logger.Error().Err(err).Msg("Failed to send alert notifications")
	}

	// Publish event
	if m.publisher != nil {
		_ = m.publisher.Publish(ctx, events.Event{
			Type:     "alert.created",
			TenantID: alert.TenantID.String(),
			ActorID:  "system",
			Action:   "create",
			Resource: "alert",
			Data: map[string]interface{}{
				"alert_id":  alert.ID.String(),
				"type":      string(alert.Type),
				"severity":  string(alert.Severity),
				"title":     alert.Title,
			},
		})
	}

	m.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("type", string(alert.Type)).
		Str("severity", string(alert.Severity)).
		Str("title", alert.Title).
		Msg("Alert created")

	return nil
}

// AcknowledgeAlert acknowledges an alert
func (m *AlertManager) AcknowledgeAlert(ctx context.Context, alertID, userID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE alerts
		SET status = 'acknowledged', acked_at = $2, acked_by = $3, updated_at = $2
		WHERE id = $1
	`
	_, err := m.db.ExecContext(ctx, query, alertID, now, userID)
	if err != nil {
		return fmt.Errorf("alert_manager.AcknowledgeAlert: %w", err)
	}

	m.logger.Info().
		Str("alert_id", alertID.String()).
		Str("user_id", userID.String()).
		Msg("Alert acknowledged")

	return nil
}

// ResolveAlert resolves an alert
func (m *AlertManager) ResolveAlert(ctx context.Context, alertID, userID uuid.UUID, resolution string) error {
	now := time.Now()
	query := `
		UPDATE alerts
		SET status = 'resolved', resolved_at = $2, resolved_by = $3, updated_at = $2
		WHERE id = $1
	`
	_, err := m.db.ExecContext(ctx, query, alertID, now, userID)
	if err != nil {
		return fmt.Errorf("alert_manager.ResolveAlert: %w", err)
	}

	m.logger.Info().
		Str("alert_id", alertID.String()).
		Str("user_id", userID.String()).
		Str("resolution", resolution).
		Msg("Alert resolved")

	return nil
}

// GetAlert retrieves an alert by ID
func (m *AlertManager) GetAlert(ctx context.Context, alertID uuid.UUID) (*Alert, error) {
	query := `SELECT * FROM alerts WHERE id = $1`
	var alert Alert
	if err := m.db.GetContext(ctx, &alert, query, alertID); err != nil {
		return nil, fmt.Errorf("alert_manager.GetAlert: %w", err)
	}

	// Parse metadata JSON
	var metadata map[string]interface{}
	if err := json.Unmarshal(alert.Metadata, &metadata); err == nil {
		// Note: In real implementation, you'd have a separate field for parsed metadata
	}

	return &alert, nil
}

// ListAlerts retrieves alerts with filtering
func (m *AlertManager) ListAlerts(ctx context.Context, tenantID uuid.UUID, filter *AlertFilter) ([]*Alert, error) {
	baseQuery := `
		SELECT * FROM alerts
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argCount := 2

	if filter != nil {
		if filter.Type != nil {
			baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
			args = append(args, *filter.Type)
			argCount++
		}
		if filter.Severity != nil {
			baseQuery += fmt.Sprintf(" AND severity = $%d", argCount)
			args = append(args, *filter.Severity)
			argCount++
		}
		if filter.Status != nil {
			baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, *filter.Status)
			argCount++
		}
		if filter.UserID != nil {
			baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
			args = append(args, *filter.UserID)
			argCount++
		}
	}

	baseQuery += " ORDER BY created_at DESC"
	if filter != nil && filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			baseQuery += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	var alerts []*Alert
	if err := m.db.SelectContext(ctx, &alerts, baseQuery, args...); err != nil {
		return nil, fmt.Errorf("alert_manager.ListAlerts: %w", err)
	}

	return alerts, nil
}

// EscalateAlert escalates an alert to a higher level
func (m *AlertManager) EscalateAlert(ctx context.Context, alertID uuid.UUID) error {
	query := `
		UPDATE alerts
		SET escalation_level = escalation_level + 1, escalated_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := m.db.ExecContext(ctx, query, alertID)
	if err != nil {
		return fmt.Errorf("alert_manager.EscalateAlert: %w", err)
	}

	m.logger.Info().
		Str("alert_id", alertID.String()).
		Msg("Alert escalated")

	return nil
}

// sendNotifications sends notifications for an alert
func (m *AlertManager) sendNotifications(ctx context.Context, alert *Alert) error {
	// Get active notification channels for the tenant
	channels, err := m.getNotificationChannels(ctx, alert.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get notification channels: %w", err)
	}

	// Send to each channel
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		switch channel.Type {
		case "webhook":
			go m.sendWebhookNotification(ctx, alert, channel)
		case "slack":
			go m.sendSlackNotification(ctx, alert, channel)
		case "email":
			go m.sendEmailNotification(ctx, alert, channel)
		}
	}

	return nil
}

// getNotificationChannels retrieves notification channels for a tenant
func (m *AlertManager) getNotificationChannels(ctx context.Context, tenantID uuid.UUID) ([]*NotificationChannel, error) {
	query := `SELECT * FROM notification_channels WHERE tenant_id = $1 AND enabled = true`
	var channels []*NotificationChannel
	if err := m.db.SelectContext(ctx, &channels, query, tenantID); err != nil {
		return nil, err
	}
	return channels, nil
}

// sendWebhookNotification sends a webhook notification
func (m *AlertManager) sendWebhookNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	m.logger.Debug().
		Str("alert_id", alert.ID.String()).
		Str("channel_id", channel.ID.String()).
		Msg("Sending webhook notification")
	// Implementation would send HTTP POST to webhook URL
}

// sendSlackNotification sends a Slack notification
func (m *AlertManager) sendSlackNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	m.logger.Debug().
		Str("alert_id", alert.ID.String()).
		Str("channel_id", channel.ID.String()).
		Msg("Sending Slack notification")
	// Implementation would send to Slack webhook
}

// sendEmailNotification sends an email notification
func (m *AlertManager) sendEmailNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	m.logger.Debug().
		Str("alert_id", alert.ID.String()).
		Str("channel_id", channel.ID.String()).
		Msg("Sending email notification")
	// Implementation would send email
}

// CreateAlertRule creates a new alert rule
func (m *AlertManager) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	conditionsJSON, _ := json.Marshal(rule.Conditions)
	actionsJSON, _ := json.Marshal(rule.Actions)

	query := `
		INSERT INTO alert_rules (
			id, tenant_id, name, type, description, enabled, conditions, actions, severity, created_at, updated_at
		) VALUES (
			$id, $tenant_id, $name, $type, $description, $enabled, $conditions, $actions, $severity, $created_at, $updated_at
		)
	`

	_, err := m.db.ExecContext(ctx, query,
		rule.ID, rule.TenantID, rule.Name, rule.Type, rule.Description,
		rule.Enabled, conditionsJSON, actionsJSON, rule.Severity,
		rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("alert_manager.CreateAlertRule: %w", err)
	}

	return nil
}

// EvaluateRule evaluates an alert rule against data
func (m *AlertManager) EvaluateRule(ctx context.Context, rule *AlertRule, data map[string]interface{}) bool {
	if !rule.Enabled {
		return false
	}

	// Evaluate conditions
	for _, condition := range rule.Conditions {
		if !m.evaluateCondition(condition, data) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single condition
func (m *AlertManager) evaluateCondition(condition RuleCondition, data map[string]interface{}) bool {
	value, exists := data[condition.Field]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", condition.Value)
	case "ne":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", condition.Value)
	case "gt":
		if numValue, ok := value.(float64); ok {
			if numCondition, ok := condition.Value.(float64); ok {
				return numValue > numCondition
			}
		}
	case "lt":
		if numValue, ok := value.(float64); ok {
			if numCondition, ok := condition.Value.(float64); ok {
				return numValue < numCondition
			}
		}
	}

	return false
}

// AlertFilter filters alert queries
type AlertFilter struct {
	Type     *AlertType
	Severity *AlertSeverity
	Status   *AlertStatus
	UserID   *uuid.UUID
	Limit    int
	Offset   int
}
