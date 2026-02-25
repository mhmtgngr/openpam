// Package reports provides report generation functionality for compliance and analytics
package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/analytics"
	"github.com/rs/zerolog"
)

// Generator handles report generation
type Generator struct {
	db              *sqlx.DB
	complianceEngine *analytics.ComplianceEngine
	baselineMgr     *analytics.BaselineManager
	anomalyStore    *AnomalyStoreWrapper
	logger          zerolog.Logger
	config          *GeneratorConfig
}

// GeneratorConfig holds configuration for report generation
type GeneratorConfig struct {
	StoragePath   string
	BaseURL       string
	MaxFileSize   int64
	TemplatesPath string
	TempDir       string
}

// NewGenerator creates a new report generator
func NewGenerator(
	db *sqlx.DB,
	complianceEngine *analytics.ComplianceEngine,
	baselineMgr *analytics.BaselineManager,
	anomalyStore *AnomalyStoreWrapper,
	logger zerolog.Logger,
	config *GeneratorConfig,
) *Generator {
	if config == nil {
		config = DefaultGeneratorConfig()
	}

	return &Generator{
		db:               db,
		complianceEngine: complianceEngine,
		baselineMgr:      baselineMgr,
		anomalyStore:     anomalyStore,
		logger:           logger,
		config:           config,
	}
}

// DefaultGeneratorConfig returns default configuration
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		StoragePath: "/var/lib/openpam/reports",
		BaseURL:     "/api/v1/analytics/reports/download",
		MaxFileSize: 100 * 1024 * 1024, // 100MB
		TemplatesPath: "/etc/openpam/report-templates",
		TempDir:     "/tmp/openpam-reports",
	}
}

// ReportFormat represents supported report formats
type ReportFormat string

const (
	FormatPDF    ReportFormat = "pdf"
	FormatHTML   ReportFormat = "html"
	FormatCSV    ReportFormat = "csv"
	FormatJSON   ReportFormat = "json"
	FormatXLSX   ReportFormat = "xlsx"
)

// ReportRequest represents a request to generate a report
type ReportRequest struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ReportType     string    `json:"report_type"` // compliance, anomalies, baseline, summary
	Framework      string    `json:"framework,omitempty"`
	Format         ReportFormat `json:"format"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	GeneratedBy    uuid.UUID `json:"generated_by"`
	Options        map[string]interface{} `json:"options,omitempty"`
}

// ReportResult represents the result of report generation
type ReportResult struct {
	ReportID       uuid.UUID `json:"report_id"`
	FileURL        string    `json:"file_url"`
	FilePath       string    `json:"file_path"`
	FileSize       int64     `json:"file_size"`
	Format         ReportFormat `json:"format"`
	GeneratedAt    time.Time `json:"generated_at"`
	ExpirationTime time.Time `json:"expiration_time"`
}

// GenerateReport generates a report based on the request
func (g *Generator) GenerateReport(ctx context.Context, req *ReportRequest) (*ReportResult, error) {
	startTime := time.Now()

	// Ensure storage directory exists
	tenantDir := filepath.Join(g.config.StoragePath, req.TenantID.String())
	if err := os.MkdirAll(tenantDir, 0755); err != nil {
		return nil, fmt.Errorf("reports.GenerateReport: create directory: %w", err)
	}

	// Generate report content based on type
	var content []byte
	var filename string
	var err error

	switch req.ReportType {
	case "compliance":
		content, filename, err = g.generateComplianceReport(ctx, req)
	case "anomalies":
		content, filename, err = g.generateAnomalyReport(ctx, req)
	case "baseline":
		content, filename, err = g.generateBaselineReport(ctx, req)
	case "summary":
		content, filename, err = g.generateSummaryReport(ctx, req)
	default:
		return nil, fmt.Errorf("reports.GenerateReport: unknown report type: %s", req.ReportType)
	}

	if err != nil {
		return nil, fmt.Errorf("reports.GenerateReport: generation failed: %w", err)
	}

	// Check file size
	if int64(len(content)) > g.config.MaxFileSize {
		return nil, fmt.Errorf("reports.GenerateReport: report exceeds maximum size of %d bytes", g.config.MaxFileSize)
	}

	// Write file
	filePath := filepath.Join(tenantDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return nil, fmt.Errorf("reports.GenerateReport: write file: %w", err)
	}

	// Calculate expiration (default 90 days)
	expirationTime := time.Now().AddDate(0, 0, 90)

	result := &ReportResult{
		ReportID:       req.ID,
		FileURL:        fmt.Sprintf("%s/%s/%s", g.config.BaseURL, req.TenantID.String(), filename),
		FilePath:       filePath,
		FileSize:       int64(len(content)),
		Format:         req.Format,
		GeneratedAt:    startTime,
		ExpirationTime: expirationTime,
	}

	g.logger.Info().
		Str("report_id", req.ID.String()).
		Str("report_type", req.ReportType).
		Str("format", string(req.Format)).
		Int64("file_size", result.FileSize).
		Str("file_path", filePath).
		Dur("duration", time.Since(startTime)).
		Msg("Report generated successfully")

	return result, nil
}

// generateComplianceReport generates a compliance report
func (g *Generator) generateComplianceReport(ctx context.Context, req *ReportRequest) ([]byte, string, error) {
	// Generate compliance report using the engine's GenerateReport method
	framework := analytics.ComplianceFramework(req.Framework)
	generatedBy := req.GeneratedBy

	report, err := g.complianceEngine.GenerateReport(
		ctx,
		req.TenantID,
		generatedBy,
		framework,
		req.PeriodStart,
		req.PeriodEnd,
	)
	if err != nil {
		return nil, "", err
	}

	// Convert ComplianceReport to ComplianceScore for templates
	score := convertReportToScore(report)

	// Generate filename
	filename := fmt.Sprintf("compliance_%s_%s_%s.%s",
		req.Framework,
		req.PeriodStart.Format("2006-01-02"),
		req.PeriodEnd.Format("2006-01-02"),
		req.Format,
	)

	// Generate based on format
	switch req.Format {
	case FormatJSON:
		return g.generateJSONCompliance(score, filename)
	case FormatHTML:
		return g.generateHTMLCompliance(score, req)
	case FormatCSV:
		return g.generateCSVCompliance(score, filename)
	case FormatPDF:
		return g.generatePDFCompliance(score, req, filename)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", req.Format)
	}
}

// generateAnomalyReport generates an anomaly report
func (g *Generator) generateAnomalyReport(ctx context.Context, req *ReportRequest) ([]byte, string, error) {
	// Get anomalies for the period
	filter := &AnomalyFilterWrapper{
		TenantID:   &req.TenantID,
		DateFrom:   &req.PeriodStart,
		DateTo:     &req.PeriodEnd,
		IncludeDuplicates: false,
	}

	anomalies, _, err := g.anomalyStore.List(ctx, filter, 10000, 0)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("anomalies_%s_%s.%s",
		req.PeriodStart.Format("2006-01-02"),
		req.PeriodEnd.Format("2006-01-02"),
		req.Format,
	)

	switch req.Format {
	case FormatJSON:
		return g.generateJSONAnomalies(anomalies, filename)
	case FormatHTML:
		return g.generateHTMLAnomalies(anomalies, req, filename)
	case FormatCSV:
		return g.generateCSVAnomalies(anomalies, filename)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", req.Format)
	}
}

// generateBaselineReport generates a baseline report
func (g *Generator) generateBaselineReport(ctx context.Context, req *ReportRequest) ([]byte, string, error) {
	// Get baseline statistics for the tenant
	stats, err := g.baselineMgr.GetBaselineStatistics(ctx, req.TenantID)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("baseline_%s.%s",
		time.Now().Format("2006-01-02"),
		req.Format,
	)

	switch req.Format {
	case FormatJSON:
		return g.generateJSONBaselines(stats, filename)
	case FormatHTML:
		return g.generateHTMLBaselines(stats, req, filename)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", req.Format)
	}
}

// generateSummaryReport generates a summary report
func (g *Generator) generateSummaryReport(ctx context.Context, req *ReportRequest) ([]byte, string, error) {
	// Gather all summary data
	summary, err := g.gatherSummaryData(ctx, req.TenantID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("summary_%s_%s.%s",
		req.PeriodStart.Format("2006-01-02"),
		req.PeriodEnd.Format("2006-01-02"),
		req.Format,
	)

	switch req.Format {
	case FormatJSON:
		return g.generateJSONSummary(summary, filename)
	case FormatHTML:
		return g.generateHTMLSummary(summary, req, filename)
	case FormatPDF:
		return g.generatePDFSummary(summary, req, filename)
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", req.Format)
	}
}

// Format-specific generators

func (g *Generator) generateJSONCompliance(score *ComplianceScore, filename string) ([]byte, string, error) {
	data, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, filename, nil
}

func (g *Generator) generateHTMLCompliance(score *ComplianceScore, req *ReportRequest) ([]byte, string, error) {
	tmpl := GetComplianceReportTemplate()

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"Score":        score,
		"GeneratedAt":  time.Now(),
		"TenantID":     req.TenantID,
		"ReportDate":   time.Now().Format("2006-01-02"),
	}); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), fmt.Sprintf("compliance_%s.html", time.Now().Format("20060102")), nil
}

func (g *Generator) generateCSVCompliance(score *ComplianceScore, filename string) ([]byte, string, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// Write header
	writer.Write([]string{"Category", "Score", "Weight", "Weighted Score", "Status", "Findings Count"})

	// Write control scores
	for _, cs := range score.ControlScores {
		writer.Write([]string{
			cs.Category,
			fmt.Sprintf("%.2f", cs.Score),
			fmt.Sprintf("%.2f", cs.Weight),
			fmt.Sprintf("%.2f", cs.WeightedScore),
			cs.Status,
			fmt.Sprintf("%d", cs.FindingsCount),
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), filename, nil
}

func (g *Generator) generatePDFCompliance(score *ComplianceScore, req *ReportRequest, filename string) ([]byte, string, error) {
	// PDF generation would use a library like gofpdf or unidoc
	// For now, return HTML and let the frontend handle PDF rendering
	html, _, err := g.generateHTMLCompliance(score, req)
	if err != nil {
		return nil, "", err
	}

	// In production, this would convert HTML to PDF
	return html, filename + ".html", nil
}

func (g *Generator) generateJSONAnomalies(anomalies []Anomaly, filename string) ([]byte, string, error) {
	data, err := json.MarshalIndent(anomalies, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, filename, nil
}

func (g *Generator) generateHTMLAnomalies(anomalies []Anomaly, req *ReportRequest, filename string) ([]byte, string, error) {
	tmpl := GetAnomalyReportTemplate()

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"Anomalies":    anomalies,
		"Count":        len(anomalies),
		"GeneratedAt":  time.Now(),
		"TenantID":     req.TenantID,
		"PeriodStart":  req.PeriodStart,
		"PeriodEnd":    req.PeriodEnd,
	}); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), filename, nil
}

func (g *Generator) generateCSVAnomalies(anomalies []Anomaly, filename string) ([]byte, string, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	// Write header
	writer.Write([]string{"ID", "Type", "Severity", "Risk Score", "Title", "Detected At", "Status"})

	// Write anomalies
	for _, a := range anomalies {
		writer.Write([]string{
			a.ID,
			a.AnomalyType,
			a.Severity,
			fmt.Sprintf("%.2f", a.RiskScore),
			a.Title,
			a.DetectedAt.Format("2006-01-02 15:04:05"),
			a.Status,
		})
	}

	writer.Flush()
	return buf.Bytes(), filename, nil
}

func (g *Generator) generateJSONBaselines(stats *analytics.BaselineStatistics, filename string) ([]byte, string, error) {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, filename, nil
}

func (g *Generator) generateHTMLBaselines(stats *analytics.BaselineStatistics, req *ReportRequest, filename string) ([]byte, string, error) {
	tmpl := GetBaselineReportTemplate()

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"Stats":       stats,
		"GeneratedAt": time.Now(),
		"TenantID":    req.TenantID,
	}); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), filename, nil
}

func (g *Generator) generateJSONSummary(summary *SummaryData, filename string) ([]byte, string, error) {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, filename, nil
}

func (g *Generator) generateHTMLSummary(summary *SummaryData, req *ReportRequest, filename string) ([]byte, string, error) {
	tmpl := GetSummaryReportTemplate()

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"Summary":     summary,
		"GeneratedAt": time.Now(),
		"TenantID":    req.TenantID,
		"PeriodStart": req.PeriodStart,
		"PeriodEnd":   req.PeriodEnd,
	}); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), filename, nil
}

func (g *Generator) generatePDFSummary(summary *SummaryData, req *ReportRequest, filename string) ([]byte, string, error) {
	// Similar to PDF compliance, would use PDF library
	html, _, err := g.generateHTMLSummary(summary, req, filename)
	if err != nil {
		return nil, "", err
	}
	return html, filename + ".html", nil
}

// gatherSummaryData collects summary data from all sources
func (g *Generator) gatherSummaryData(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*SummaryData, error) {
	summary := &SummaryData{
		TenantID:    tenantID,
		PeriodStart: start,
		PeriodEnd:   end,
		GeneratedAt: time.Now(),
	}

	// Get anomaly stats
	// (In production, would query actual data)

	return summary, nil
}

// CleanupOldReports removes reports older than retention period
func (g *Generator) CleanupOldReports(ctx context.Context, retentionDays int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	removedCount := 0

	// Walk through storage directory
	err := filepath.Walk(g.config.StoragePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				g.logger.Warn().Err(err).
					Str("path", path).
					Msg("Failed to remove old report file")
			} else {
				removedCount++
			}
		}

		return nil
	})

	if err != nil {
		return removedCount, fmt.Errorf("reports.CleanupOldReports: walk: %w", err)
	}

	g.logger.Info().
		Int("removed_count", removedCount).
		Int("retention_days", retentionDays).
		Msg("Old reports cleaned up")

	return removedCount, nil
}

// Data structures for report generation

// AnomalyStoreWrapper wraps the anomaly store for report generation
type AnomalyStoreWrapper struct {
	// store *anomalies.Store
}

// List retrieves anomalies (simplified)
func (w *AnomalyStoreWrapper) List(ctx context.Context, filter *AnomalyFilterWrapper, limit, offset int) ([]Anomaly, int, error) {
	// Implementation would query the actual store
	return []Anomaly{}, 0, nil
}

// AnomalyFilterWrapper wraps filter parameters
type AnomalyFilterWrapper struct {
	TenantID          *uuid.UUID
	UserID            *uuid.UUID
	AnomalyType       *string
	Severity          *string
	Status            *string
	IncludeDuplicates bool
	DateFrom          *time.Time
	DateTo            *time.Time
}

// Anomaly represents a simplified anomaly for reporting
type Anomaly struct {
	ID          string    `json:"id"`
	AnomalyType string    `json:"anomaly_type"`
	Severity    string    `json:"severity"`
	RiskScore   float64   `json:"risk_score"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
	Status      string    `json:"status"`
}

// ComplianceScore represents compliance scoring data
type ComplianceScore struct {
	TenantID        uuid.UUID              `json:"tenant_id"`
	Framework       string                 `json:"framework"`
	OverallScore    float64                `json:"overall_score"`
	ControlScores   map[string]ControlScore `json:"control_scores"`
	PeriodStart     time.Time              `json:"period_start"`
	PeriodEnd       time.Time              `json:"period_end"`
	Findings        []ComplianceFinding    `json:"findings"`
	TotalControls   int                    `json:"total_controls"`
	PassedControls  int                    `json:"passed_controls"`
	FailedControls  int                    `json:"failed_controls"`
	SkippedControls int                    `json:"skipped_controls"`
	Recommendations []string               `json:"recommendations,omitempty"`
}

// ControlScore represents a control category score
type ControlScore struct {
	Category      string  `json:"category"`
	Score         float64 `json:"score"`
	Weight        float64 `json:"weight"`
	WeightedScore float64 `json:"weighted_score"`
	Status        string  `json:"status"`
	FindingsCount int     `json:"findings_count"`
}

// ComplianceFinding represents a compliance issue
type ComplianceFinding struct {
	ID          string    `json:"id"`
	ControlID   string    `json:"control_id"`
	ControlName string    `json:"control_name"`
	Category    string    `json:"category"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation"`
	DetectedAt  time.Time `json:"detected_at"`
}

// BaselineStatistics represents baseline statistics
type BaselineStatistics struct {
	TotalBaselines   int     `json:"total_baselines"`
	ActiveBaselines  int     `json:"active_baselines"`
	UniqueUsers      int     `json:"unique_users"`
	AvgConfidence    float64 `json:"avg_confidence"`
	LastCalculated   *time.Time `json:"last_calculated"`
}

// SummaryData represents summary report data
type SummaryData struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	GeneratedAt time.Time `json:"generated_at"`
	// Add summary fields as needed
}

// convertReportToScore converts a ComplianceReport to a ComplianceScore for templates
func convertReportToScore(report *analytics.ComplianceReport) *ComplianceScore {
	score := &ComplianceScore{
		TenantID:      report.TenantID,
		Framework:     report.Framework,
		OverallScore:  0,
		ControlScores: make(map[string]ControlScore),
		PeriodStart:   report.PeriodStart,
		PeriodEnd:     report.PeriodEnd,
		Findings:      []ComplianceFinding{},
	}

	if report.OverallScore != nil {
		score.OverallScore = *report.OverallScore
	}

	// Unmarshal findings and recommendations
	if report.Findings != nil {
		var findings []map[string]interface{}
		_ = json.Unmarshal(report.Findings, &findings)
		for _, f := range findings {
			finding := ComplianceFinding{
				ID:          uuid.New().String(),
				ControlID:   getStringValue(f, "control_id"),
				ControlName: getStringValue(f, "control_name"),
				Category:    getStringValue(f, "category"),
				Severity:    getStringValue(f, "severity"),
				Description: getStringValue(f, "description"),
				Remediation: getStringValue(f, "remediation"),
				DetectedAt:  report.GeneratedAt,
			}
			score.Findings = append(score.Findings, finding)
		}
	}

	// Unmarshal recommendations
	if report.Recommendations != nil {
		var recs []string
		_ = json.Unmarshal(report.Recommendations, &recs)
		// Store recommendations in a field that templates can access
	}

	// Set additional fields for template rendering
	score.TotalControls = report.TotalControls
	score.PassedControls = report.PassedControls
	score.FailedControls = report.FailedControls
	score.SkippedControls = report.SkippedControls

	return score
}

func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getStringSlice extracts a string slice from an interface
func getStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if arr, ok := v.([]string); ok {
		return arr
	}
	if arr, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
