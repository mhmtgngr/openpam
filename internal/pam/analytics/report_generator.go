package analytics

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ReportGenerator handles report generation in various formats
type ReportGenerator struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(repo *Repository, logger zerolog.Logger) *ReportGenerator {
	return &ReportGenerator{
		repo:   repo,
		logger: logger,
	}
}

// GenerateSessionReport generates a session activity report
func (g *ReportGenerator) GenerateSessionReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, format string) (*GeneratedReport, error) {
	g.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("start", startDate.Format("2006-01-02")).
		Str("end", endDate.Format("2006-01-02")).
		Str("format", format).
		Msg("Generating session report")

	// Get session data
	stats, err := g.repo.GetRawSessionStats(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("generator.GenerateSessionReport: %w", err)
	}

	// Get top users
	topUsers, err := g.repo.GetTopUsersBySessions(ctx, tenantID, startDate, endDate, 20)
	if err != nil {
		g.logger.Error().Err(err).Msg("Failed to get top users for report")
		topUsers = []TopUser{}
	}

	// Get access patterns
	patterns, err := g.repo.GetAccessPatterns(ctx, tenantID, startDate, endDate)
	if err != nil {
		g.logger.Error().Err(err).Msg("Failed to get access patterns for report")
		patterns = []AccessPattern{}
	}

	// Build report data
	reportData := map[string]interface{}{
		"period": map[string]string{
			"start": startDate.Format(time.RFC3339),
			"end":   endDate.Format(time.RFC3339),
		},
		"summary": stats,
		"top_users": topUsers,
		"access_patterns": g.aggregateAccessPatternsByDay(patterns),
		"generated_at": time.Now().Format(time.RFC3339),
	}

	// Generate output based on format
	var content []byte
	var contentType string
	var fileExtension string

	switch format {
	case "json":
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"

	case "csv":
		content, contentType = g.generateCSVReport(reportData)
		fileExtension = "csv"

	case "pdf":
		// PDF generation would require a library like gofpdf
		// For now, return JSON with note
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"
		// In production, use a proper PDF library

	default:
		// Default to JSON
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"
	}

	return &GeneratedReport{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReportType:   "session_activity",
		StartDate:    startDate,
		EndDate:      endDate,
		Format:       format,
		ContentType:  contentType,
		FileExtension: fileExtension,
		ContentSize:  len(content),
		Content:      content,
		Status:       "completed",
		GeneratedAt:   time.Now(),
	}, nil
}

// GenerateComplianceReport generates a compliance report
func (g *ReportGenerator) GenerateComplianceReport(ctx context.Context, tenantID uuid.UUID, framework string, startDate, endDate time.Time, format string) (*GeneratedReport, error) {
	g.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("framework", framework).
		Str("start", startDate.Format("2006-01-02")).
	Str("end", endDate.Format("2006-01-02")).
		Msg("Generating compliance report")

	// Get compliance status
	status, err := g.repo.CalculateComplianceStatus(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("generator.GenerateComplianceReport: %w", err)
	}

	// Build report data
	reportData := map[string]interface{}{
		"framework": framework,
		"period": map[string]string{
			"start": startDate.Format(time.RFC3339),
			"end":   endDate.Format(time.RFC3339),
		},
		"summary": status,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	// Generate output based on format
	var content []byte
	var contentType string
	var fileExtension string

	switch format {
	case "json":
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"

	case "csv":
		content, contentType = g.generateComplianceCSV(reportData, framework)
		fileExtension = "csv"

	case "pdf":
		// PDF generation would require a library
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"

	default:
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"
	}

	return &GeneratedReport{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReportType:   "compliance",
		StartDate:    startDate,
		EndDate:      endDate,
		Format:       format,
		ContentType:  contentType,
		FileExtension: fileExtension,
		ContentSize:  len(content),
		Content:      content,
		Status:       "completed",
		GeneratedAt:   time.Now(),
	}, nil
}

// GenerateUserActivityReport generates a user activity report
func (g *ReportGenerator) GenerateUserActivityReport(ctx context.Context, tenantID, userID uuid.UUID, startDate, endDate time.Time, format string) (*GeneratedReport, error) {
	g.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("user_id", userID.String()).
		Str("start", startDate.Format("2006-01-02")).
		Str("end", endDate.Format("2006-01-02")).
		Msg("Generating user activity report")

	// Get user activity
	activities, err := g.repo.ListUserActivity(ctx, tenantID, PeriodDay, startDate, endDate, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("generator.GenerateUserActivityReport: %w", err)
	}

	// Get user risk score
	riskScore, _ := g.repo.GetLatestRiskScore(ctx, tenantID, EntityType("user"), userID)

	// Build report data
	reportData := map[string]interface{}{
		"user_id":     userID.String(),
		"period": map[string]string{
			"start": startDate.Format(time.RFC3339),
			"end":   endDate.Format(time.RFC3339),
		},
		"activities": activities,
		"risk_score": riskScore,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	// Generate output based on format
	var content []byte
	var contentType string
	var fileExtension string

	switch format {
	case "json":
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"

	case "csv":
		content, contentType = g.generateUserActivityCSV(reportData)
		fileExtension = "csv"

	default:
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"
	}

	return &GeneratedReport{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       &userID,
		ReportType:   "user_activity",
		StartDate:    startDate,
		EndDate:      endDate,
		Format:       format,
		ContentType:  contentType,
		FileExtension: fileExtension,
		ContentSize:  len(content),
		Content:      content,
		Status:       "completed",
		GeneratedAt:   time.Now(),
	}, nil
}

// GenerateRiskAnalysisReport generates a risk analysis report
func (g *ReportGenerator) GenerateRiskAnalysisReport(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, format string) (*GeneratedReport, error) {
	g.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("start", startDate.Format("2006-01-02")).
		Str("end", endDate.Format("2006-01-02")).
		Msg("Generating risk analysis report")

	// Get risk scores
	scores, err := g.repo.ListRiskScores(ctx, tenantID, nil, nil, nil, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("generator.GenerateRiskAnalysisReport: %w", err)
	}

	// Get anomalies
	anomalies, err := g.repo.GetAnomalousUsers(ctx, tenantID, PeriodDay, time.Now().Truncate(24*time.Hour), 50)
	if err != nil {
		g.logger.Error().Err(err).Msg("Failed to get anomalous users")
		anomalies = []UserActivity{}
	}

	// Build report data
	reportData := map[string]interface{}{
		"period": map[string]string{
			"start": startDate.Format(time.RFC3339),
			"end":   endDate.Format(time.RFC3339),
		},
		"risk_scores": scores,
		"anomalies":  anomalies,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	// Generate output based on format
	var content []byte
	var contentType string
	var fileExtension string

	switch format {
	case "json":
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"

	default:
		content, _ = json.MarshalIndent(reportData, "", "  ")
		contentType = "application/json"
		fileExtension = "json"
	}

	return &GeneratedReport{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReportType:   "risk_analysis",
		StartDate:    startDate,
		EndDate:      endDate,
		Format:       format,
		ContentType:  contentType,
		FileExtension: fileExtension,
		ContentSize:  len(content),
		Content:      content,
		Status:       "completed",
		GeneratedAt:   time.Now(),
	}, nil
}

// GeneratedReport represents a generated report
type GeneratedReport struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	ReportType     string    `json:"report_type"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	Format         string    `json:"format"`
	ContentType    string    `json:"content_type"`
	FileExtension  string    `json:"file_extension"`
	ContentSize    int       `json:"content_size"`
	Content        []byte    `json:"content"`
	FileURL        string    `json:"file_url,omitempty"`
	Status         string    `json:"status"` // pending, generating, completed, failed
	Error          string    `json:"error,omitempty"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// CSV generation helpers

func (g *ReportGenerator) generateCSVReport(data map[string]interface{}) ([]byte, string) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write headers based on data type
	if sessionStats, ok := data["summary"].(*SessionStats); ok {
		writer.Write([]string{"Metric", "Value"})

		writer.Write([]string{"Total Sessions", fmt.Sprintf("%d", sessionStats.Total)})
		writer.Write([]string{"Active Sessions", fmt.Sprintf("%d", sessionStats.Active)})
		writer.Write([]string{"Completed Sessions", fmt.Sprintf("%d", sessionStats.Completed)})
		writer.Write([]string{"Failed Sessions", fmt.Sprintf("%d", sessionStats.Failed)})
	}

	writer.Flush()
	return buf.Bytes(), "text/csv"
}

func (g *ReportGenerator) generateComplianceCSV(data map[string]interface{}, framework string) ([]byte, string) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write headers
	writer.Write([]string{"Control ID", "Control Name", "Status", "Score", "Total", "Passed", "Failed"})

	if status, ok := data["summary"].(*ComplianceStatus); ok {
		for policyID, policyStatus := range status.ByPolicy {
			statusStr := "passed"
			if policyStatus.ComplianceRate < 100 {
				statusStr = "failed"
			}
			writer.Write([]string{
				policyID,
				policyStatus.PolicyName,
				statusStr,
				fmt.Sprintf("%.1f", policyStatus.ComplianceRate),
				fmt.Sprintf("%d", policyStatus.TotalEvaluations),
				fmt.Sprintf("%d", policyStatus.PassedEvaluations),
				fmt.Sprintf("%d", policyStatus.TotalEvaluations-policyStatus.PassedEvaluations),
			})
		}
	}

	writer.Flush()
	return buf.Bytes(), "text/csv"
}

func (g *ReportGenerator) generateUserActivityCSV(data map[string]interface{}) ([]byte, string) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write headers
	writer.Write([]string{"Date", "Sessions Created", "Credentials Accessed", "Approvals Requested", "Total Active Seconds", "Off Hours Access", "Policy Violations", "Failed Auth"})

	if activities, ok := data["activities"].([]UserActivity); ok {
		for _, activity := range activities {
			writer.Write([]string{
				activity.PeriodStart.Format("2006-01-02"),
				fmt.Sprintf("%d", activity.SessionsCreated),
				fmt.Sprintf("%d", activity.CredentialsAccessed),
				fmt.Sprintf("%d", activity.ApprovalsRequested),
				fmt.Sprintf("%d", activity.TotalActiveSeconds),
				fmt.Sprintf("%t", activity.OffHoursAccess),
				fmt.Sprintf("%d", activity.PolicyViolations),
				fmt.Sprintf("%d", activity.FailedAuthAttempts),
			})
		}
	}

	writer.Flush()
	return buf.Bytes(), "text/csv"
}

// Helper functions

func (g *ReportGenerator) aggregateAccessPatternsByDay(patterns []AccessPattern) map[string][]int {
	byDay := make(map[string][]int)

	for _, pattern := range patterns {
		// Create a key from hour and day of week
		dayKey := fmt.Sprintf("dow:%d-hod:%d", pattern.DayOfWeek, pattern.HourOfDay)
		byDay[dayKey] = append(byDay[dayKey], pattern.SessionCount)
	}

	return byDay
}
