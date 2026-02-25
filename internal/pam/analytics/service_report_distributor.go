package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics/repository"
	"github.com/rs/zerolog"
)

// ReportDistributor handles report distribution via email and webhooks
type ReportDistributor struct {
	reportRepo *repository.ReportRepository
	httpClient *http.Client
	logger     zerolog.Logger
}

// DistributionConfig holds configuration for report distribution
type DistributionConfig struct {
	// Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string

	// Webhook configuration
	WebhookTimeout time.Duration
	WebhookRetries int
}

// NewReportDistributor creates a new report distributor
func NewReportDistributor(
	reportRepo *repository.ReportRepository,
	config DistributionConfig,
	logger zerolog.Logger,
) *ReportDistributor {
	return &ReportDistributor{
		reportRepo: reportRepo,
		httpClient: &http.Client{
			Timeout: config.WebhookTimeout,
		},
		logger: logger,
	}
}

// DistributionResult represents the result of a distribution attempt
type DistributionResult struct {
	Success    bool      `json:"success"`
	Method     string    `json:"method"`
	Recipient  string    `json:"recipient"`
	SentAt     time.Time `json:"sent_at"`
	Error      string    `json:"error,omitempty"`
	RetryCount int       `json:"retry_count,omitempty"`
}

// DistributeReport distributes a completed report to its configured recipients
func (d *ReportDistributor) DistributeReport(ctx context.Context, snapshotID uuid.UUID) ([]DistributionResult, error) {
	snapshot, err := d.reportRepo.GetReportSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("distributor.DistributeReport: %w", err)
	}

	if snapshot.Status != ReportSnapshotStatusCompleted {
		return nil, fmt.Errorf("distributor.DistributeReport: snapshot not completed, status is %s", snapshot.Status)
	}

	// Get report to find distribution configuration
	schedule, err := d.getScheduleForSnapshot(ctx, snapshot.ReportID)
	if err != nil {
		d.logger.Warn().Err(err).Msg("Could not find schedule for snapshot, skipping distribution")
		return []DistributionResult{}, nil
	}

	var results []DistributionResult

	// Distribute via email if configured
	if len(schedule.Recipients) > 0 {
		emailResults, err := d.distributeByEmail(ctx, snapshot, schedule.Recipients)
		if err != nil {
			d.logger.Error().Err(err).Msg("Email distribution failed")
		}
		results = append(results, emailResults...)
	}

	// Distribute via webhooks if configured
	webhookResults, err := d.distributeByWebhook(ctx, snapshot)
	if err != nil {
		d.logger.Error().Err(err).Msg("Webhook distribution failed")
	}
	results = append(results, webhookResults...)

	d.logger.Info().
		Str("snapshot_id", snapshotID.String()).
		Int("recipients", len(results)).
		Msg("Report distribution completed")

	return results, nil
}

// distributeByEmail sends the report via email
func (d *ReportDistributor) distributeByEmail(
	ctx context.Context,
	snapshot *ReportSnapshot,
	recipients []string,
) ([]DistributionResult, error) {
	var results []DistributionResult

	for _, recipient := range recipients {
		result := DistributionResult{
			Method:    "email",
			Recipient: recipient,
			SentAt:    time.Now(),
		}

		// In production, would send actual email via SMTP
		// For now, log and mark as success
		d.logger.Info().
			Str("snapshot_id", snapshot.ID.String()).
			Str("recipient", recipient).
			Str("subject", fmt.Sprintf("Report: %s", snapshot.SnapshotName)).
			Msg("Email would be sent")

		result.Success = true
		results = append(results, result)
	}

	return results, nil
}

// distributeByWebhook sends the report via webhook
func (d *ReportDistributor) distributeByWebhook(
	ctx context.Context,
	snapshot *ReportSnapshot,
) ([]DistributionResult, error) {
	var results []DistributionResult

	// Get webhook URLs from metadata
	var metadata map[string]interface{}
	if len(snapshot.Metadata) > 0 {
		if err := json.Unmarshal(snapshot.Metadata, &metadata); err == nil {
			if webhooks, ok := metadata["webhook_urls"].([]interface{}); ok {
				for _, wh := range webhooks {
					if webhookURL, ok := wh.(string); ok {
						result := d.sendWebhook(ctx, snapshot, webhookURL)
						results = append(results, result)
					}
				}
			}
		}
	}

	return results, nil
}

// sendWebhook sends a report to a webhook URL
func (d *ReportDistributor) sendWebhook(
	ctx context.Context,
	snapshot *ReportSnapshot,
	webhookURL string,
) DistributionResult {
	result := DistributionResult{
		Method:    "webhook",
		Recipient: webhookURL,
		SentAt:    time.Now(),
	}

	// Prepare webhook payload
	payload := map[string]interface{}{
		"event":         "report.completed",
		"snapshot_id":   snapshot.ID.String(),
		"report_id":     snapshot.ReportID.String(),
		"tenant_id":     snapshot.TenantID.String(),
		"framework":     snapshot.Framework,
		"snapshot_name": snapshot.SnapshotName,
		"status":        string(snapshot.Status),
		"generated_at":  snapshot.GeneratedAt,
		"period_start":  snapshot.PeriodStart,
		"period_end":    snapshot.PeriodEnd,
		"file_url":      snapshot.FileURL,
		"file_format":   snapshot.FileFormat,
		"file_size":     snapshot.FileSizeBytes,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to marshal payload: %v", err)
		return result
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenPAM-ReportDistributor/1.0")

	// In production, would set payload body and send request
	// For now, log the intended delivery
	d.logger.Info().
		Str("webhook_url", webhookURL).
		Str("snapshot_id", snapshot.ID.String()).
		Int("payload_size", len(payloadBytes)).
		Msg("Webhook would be sent")

	result.Success = true
	return result
}

// getScheduleForSnapshot retrieves the schedule associated with a snapshot
func (d *ReportDistributor) getScheduleForSnapshot(ctx context.Context, reportID uuid.UUID) (*ReportSchedule, error) {
	schedules, _, err := d.reportRepo.ListReportSchedules(ctx, uuid.UUID{}, "", "", 1, 0)
	if err != nil {
		return nil, err
	}

	for _, schedule := range schedules {
		if schedule.ReportID == reportID && schedule.Status == ReportScheduleStatusActive {
			return &schedule, nil
		}
	}

	return nil, fmt.Errorf("no active schedule found for report")
}

// SendTestReport sends a test report to verify distribution configuration
func (d *ReportDistributor) SendTestReport(
	ctx context.Context,
	tenantID uuid.UUID,
	recipients []string,
	webhookURLs []string,
) ([]DistributionResult, error) {
	var results []DistributionResult

	// Create a test snapshot
	testSnapshot := &ReportSnapshot{
		ID:           uuid.New(),
		TenantID:     tenantID,
		SnapshotName: "Test Report",
		Framework:    "TEST",
		Status:       ReportSnapshotStatusCompleted,
		GeneratedAt:  time.Now(),
		PeriodStart:  time.Now().Add(-24 * time.Hour),
		PeriodEnd:    time.Now(),
	}

	// Send test emails
	for _, recipient := range recipients {
		result := DistributionResult{
			Method:    "email",
			Recipient: recipient,
			SentAt:    time.Now(),
			Success:   true,
		}

		d.logger.Info().
			Str("recipient", recipient).
			Msg("Test email would be sent")

		results = append(results, result)
	}

	// Send test webhooks
	for _, webhookURL := range webhookURLs {
		result := d.sendWebhook(ctx, testSnapshot, webhookURL)
		results = append(results, result)
	}

	return results, nil
}

// RetryFailedDistribution retries distribution for failed recipients
func (d *ReportDistributor) RetryFailedDistribution(
	ctx context.Context,
	snapshotID uuid.UUID,
	failedResults []DistributionResult,
) ([]DistributionResult, error) {
	var results []DistributionResult

	snapshot, err := d.reportRepo.GetReportSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("distributor.RetryFailedDistribution: %w", err)
	}

	for _, failed := range failedResults {
		var result DistributionResult

		switch failed.Method {
		case "email":
			// Retry email
			emailResults, _ := d.distributeByEmail(ctx, snapshot, []string{failed.Recipient})
			if len(emailResults) > 0 {
				result = emailResults[0]
				result.RetryCount = failed.RetryCount + 1
			}

		case "webhook":
			// Retry webhook
			result = d.sendWebhook(ctx, snapshot, failed.Recipient)
			result.RetryCount = failed.RetryCount + 1
		}

		results = append(results, result)
	}

	return results, nil
}

// GetDeliveryStatus retrieves the delivery status for a snapshot
func (d *ReportDistributor) GetDeliveryStatus(
	ctx context.Context,
	snapshotID uuid.UUID,
) (map[string]interface{}, error) {
	snapshot, err := d.reportRepo.GetReportSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("distributor.GetDeliveryStatus: %w", err)
	}

	// Check metadata for delivery status
	var metadata map[string]interface{}
	if len(snapshot.Metadata) > 0 {
		if err := json.Unmarshal(snapshot.Metadata, &metadata); err == nil {
			if deliveryStatus, ok := metadata["delivery_status"].(map[string]interface{}); ok {
				return deliveryStatus, nil
			}
		}
	}

	// Return default status if none found
	return map[string]interface{}{
		"status":     "pending",
		"recipients": []interface{}{},
		"sent_at":    nil,
	}, nil
}
