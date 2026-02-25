// Package audit provides report generation functionality for compliance and analytics reports
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/openpam/openpam/internal/audit/repository"
	"github.com/rs/zerolog"
)

// ReportGenerator handles the generation of compliance and analytics reports
type ReportGenerator struct {
	db               *sqlx.DB
	snapshotRepo     *repository.ReportSnapshotRepository
	compReportRepo   *repository.ComplianceReportRepository
	logger           zerolog.Logger
	storageConfig    *StorageConfig
}

// StorageConfig holds configuration for report storage
type StorageConfig struct {
	BaseURL      string // Base URL for accessing stored files
	StoragePath  string // Local storage path or S3 bucket prefix
	MaxFileSize  int64  // Maximum file size in bytes
	RetentionDays int   // Default retention period in days
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(db *sqlx.DB, logger zerolog.Logger, storageConfig *StorageConfig) *ReportGenerator {
	return &ReportGenerator{
		db:             db,
		snapshotRepo:   repository.NewReportSnapshotRepository(db, logger),
		compReportRepo: repository.NewComplianceReportRepository(db, logger),
		logger:         logger,
		storageConfig:  storageConfig,
	}
}

// GenerateReportRequest holds parameters for report generation
type GenerateReportRequest struct {
	TenantID      uuid.UUID
	ReportID      uuid.UUID
	SnapshotName  string
	Format        string // pdf, xlsx, csv, html, json
	GeneratedBy   uuid.UUID
	Options       map[string]interface{}
	RetentionDays *int
}

// GeneratedReport holds the result of a report generation
type GeneratedReport struct {
	Snapshot      *model.ReportSnapshot
	FileURL       string
	StoragePath   string
	FileSizeBytes int64
}

// GenerateReport generates a compliance report snapshot
func (g *ReportGenerator) GenerateReport(ctx context.Context, req *GenerateReportRequest) (*GeneratedReport, error) {
	// Get the source report
	sourceReport, err := g.compReportRepo.GetByID(ctx, req.ReportID)
	if err != nil {
		return nil, fmt.Errorf("report_generator.GenerateReport: failed to get source report: %w", err)
	}

	// Verify tenant isolation
	if sourceReport.TenantID != req.TenantID {
		return nil, fmt.Errorf("report_generator.GenerateReport: tenant mismatch")
	}

	// Create the snapshot record
	snapshot := &model.ReportSnapshot{
		TenantID:     req.TenantID,
		ReportID:     req.ReportID,
		SnapshotName: req.SnapshotName,
		Framework:    sourceReport.Framework,
		GeneratedAt:  time.Now(),
		GeneratedBy:  req.GeneratedBy,
		Status:       string(model.ReportSnapshotStatusPending),
		FileFormat:   &req.Format,
		PeriodStart:  sourceReport.PeriodStart,
		PeriodEnd:    sourceReport.PeriodEnd,
	}

	// Set retention
	retentionDays := 90 // default
	if req.RetentionDays != nil {
		retentionDays = *req.RetentionDays
	}
	if g.storageConfig.RetentionDays > 0 {
		retentionDays = g.storageConfig.RetentionDays
	}
	expiresAt := time.Now().AddDate(0, 0, retentionDays)
	snapshot.ExpiresAt = &expiresAt

	// Generate the report content based on format
	var fileURL, storagePath string
	var fileSizeBytes int64
	var genErr error

	switch req.Format {
	case string(model.ReportFormatJSON):
		fileURL, storagePath, fileSizeBytes, genErr = g.generateJSONReport(ctx, sourceReport, req.Options)
	case string(model.ReportFormatHTML):
		fileURL, storagePath, fileSizeBytes, genErr = g.generateHTMLReport(ctx, sourceReport, req.Options)
	case string(model.ReportFormatCSV):
		fileURL, storagePath, fileSizeBytes, genErr = g.generateCSVReport(ctx, sourceReport, req.Options)
	case string(model.ReportFormatPDF):
		fileURL, storagePath, fileSizeBytes, genErr = g.generatePDFReport(ctx, sourceReport, req.Options)
	case string(model.ReportFormatXLSX):
		fileURL, storagePath, fileSizeBytes, genErr = g.generateExcelReport(ctx, sourceReport, req.Options)
	default:
		return nil, fmt.Errorf("report_generator.GenerateReport: unsupported format: %s", req.Format)
	}

	if genErr != nil {
		// Update snapshot with error
		snapshot.Status = string(model.ReportSnapshotStatusFailed)
		errMsg := genErr.Error()
		snapshot.ErrorMessage = &errMsg
		_ = g.snapshotRepo.Create(ctx, snapshot)
		return nil, fmt.Errorf("report_generator.GenerateReport: generation failed: %w", genErr)
	}

	// Set successful result
	snapshot.Status = string(model.ReportSnapshotStatusCompleted)
	snapshot.FileURL = &fileURL
	snapshot.StoragePath = &storagePath
	snapshot.FileSizeBytes = &fileSizeBytes

	// Save the snapshot
	if err := g.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("report_generator.GenerateReport: failed to save snapshot: %w", err)
	}

	return &GeneratedReport{
		Snapshot:      snapshot,
		FileURL:       fileURL,
		StoragePath:   storagePath,
		FileSizeBytes: fileSizeBytes,
	}, nil
}

// generateJSONReport generates a JSON format report
func (g *ReportGenerator) generateJSONReport(ctx context.Context, report *model.ComplianceReport, options map[string]interface{}) (string, string, int64, error) {
	// Create report data structure
	reportData := map[string]interface{}{
		"id":            report.ID,
		"report_name":   report.ReportName,
		"framework":     report.Framework,
		"version":       report.Version,
		"generated_at":  report.GeneratedAt,
		"generated_by":  report.GeneratedBy,
		"status":        report.Status,
		"overall_score": report.OverallScore,
		"total_controls": report.TotalControls,
		"passed_controls": report.PassedControls,
		"failed_controls": report.FailedControls,
		"skipped_controls": report.SkippedControls,
		"period_start":  report.PeriodStart,
		"period_end":    report.PeriodEnd,
		"summary":       report.Summary,
	}

	// Parse JSONB fields
	if report.Findings != nil {
		var findings map[string]interface{}
		if err := json.Unmarshal(report.Findings, &findings); err == nil {
			reportData["findings"] = findings
		}
	}
	if report.Recommendations != nil {
		var recommendations map[string]interface{}
		if err := json.Unmarshal(report.Recommendations, &recommendations); err == nil {
			reportData["recommendations"] = recommendations
		}
	}
	if report.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(report.Metadata, &metadata); err == nil {
			reportData["metadata"] = metadata
		}
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Generate storage path
	storagePath := g.generateStoragePath(report.TenantID, report.ID, "json")
	fileURL := g.storageConfig.BaseURL + "/" + storagePath

	// In a real implementation, this would write to storage (S3, local filesystem, etc.)
	// For now, we'll just return the path and size
	_ = jsonBytes // Would be written to storage

	return fileURL, storagePath, int64(len(jsonBytes)), nil
}

// generateHTMLReport generates an HTML format report
func (g *ReportGenerator) generateHTMLReport(ctx context.Context, report *model.ComplianceReport, options map[string]interface{}) (string, string, int64, error) {
	// Generate HTML content
	htmlContent := g.buildHTMLTemplate(report)

	// Generate storage path
	storagePath := g.generateStoragePath(report.TenantID, report.ID, "html")
	fileURL := g.storageConfig.BaseURL + "/" + storagePath

	_ = htmlContent // Would be written to storage

	return fileURL, storagePath, int64(len(htmlContent)), nil
}

// generateCSVReport generates a CSV format report
func (g *ReportGenerator) generateCSVReport(ctx context.Context, report *model.ComplianceReport, options map[string]interface{}) (string, string, int64, error) {
	// Generate CSV content (simplified - in production would use a CSV library)
	csvContent := fmt.Sprintf("Report Name,%s\nFramework,%s\nStatus,%s\nScore,%.2f\nPassed,%d\nFailed,%d\n",
		report.ReportName,
		report.Framework,
		report.Status,
		floatPtrValue(report.OverallScore),
		report.PassedControls,
		report.FailedControls,
	)

	// Generate storage path
	storagePath := g.generateStoragePath(report.TenantID, report.ID, "csv")
	fileURL := g.storageConfig.BaseURL + "/" + storagePath

	return fileURL, storagePath, int64(len(csvContent)), nil
}

// generatePDFReport generates a PDF format report
func (g *ReportGenerator) generatePDFReport(ctx context.Context, report *model.ComplianceReport, options map[string]interface{}) (string, string, int64, error) {
	// In a real implementation, this would use a PDF library like gofpdf or unidoc
	// For now, we'll return a placeholder
	storagePath := g.generateStoragePath(report.TenantID, report.ID, "pdf")
	fileURL := g.storageConfig.BaseURL + "/" + storagePath

	g.logger.Info().Str("report_id", report.ID.String()).Msg("PDF report generation (placeholder)")

	return fileURL, storagePath, 0, nil
}

// generateExcelReport generates an Excel format report
func (g *ReportGenerator) generateExcelReport(ctx context.Context, report *model.ComplianceReport, options map[string]interface{}) (string, string, int64, error) {
	// In a real implementation, this would use excelize or similar library
	storagePath := g.generateStoragePath(report.TenantID, report.ID, "xlsx")
	fileURL := g.storageConfig.BaseURL + "/" + storagePath

	g.logger.Info().Str("report_id", report.ID.String()).Msg("Excel report generation (placeholder)")

	return fileURL, storagePath, 0, nil
}

// generateStoragePath generates a unique storage path for a report file
func (g *ReportGenerator) generateStoragePath(tenantID, reportID uuid.UUID, format string) string {
	timestamp := time.Now().Format("20060102")
	filename := uuid.New().String()
	return fmt.Sprintf("reports/%s/%s/%s.%s", tenantID, timestamp, filename, format)
}

// buildHTMLTemplate builds an HTML template for the report
func (g *ReportGenerator) buildHTMLTemplate(report *model.ComplianceReport) string {
	score := "N/A"
	if report.OverallScore != nil {
		score = fmt.Sprintf("%.2f", *report.OverallScore)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - %s Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { border-bottom: 2px solid #333; padding-bottom: 20px; }
        .section { margin: 30px 0; }
        .score { font-size: 48px; font-weight: bold; color: %s; }
        .passed { color: green; }
        .failed { color: red; }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s</h1>
        <p>Framework: %s | Period: %s to %s</p>
        <p>Generated: %s</p>
    </div>
    <div class="section">
        <h2>Compliance Score</h2>
        <div class="score">%s</div>
    </div>
    <div class="section">
        <h2>Control Results</h2>
        <p>Total Controls: %d</p>
        <p class="passed">Passed: %d</p>
        <p class="failed">Failed: %d</p>
        <p>Skipped: %d</p>
    </div>
</body>
</html>`,
		report.ReportName,
		report.Framework,
		getScoreColor(report.OverallScore),
		report.ReportName,
		report.Framework,
		report.PeriodStart.Format("2006-01-02"),
		report.PeriodEnd.Format("2006-01-02"),
		report.GeneratedAt.Format("2006-01-02 15:04:05"),
		score,
		report.TotalControls,
		report.PassedControls,
		report.FailedControls,
		report.SkippedControls,
	)
}

// getScoreColor returns a color based on the score
func getScoreColor(score *float64) string {
	if score == nil {
		return "#666"
	}
	if *score >= 80 {
		return "green"
	}
	if *score >= 60 {
		return "orange"
	}
	return "red"
}

// floatPtrValue safely dereferences a float pointer
func floatPtrValue(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// =============================================================================
// Async Report Generation
// ============================================================================

// QueueReportGeneration queues a report for async generation
func (g *ReportGenerator) QueueReportGeneration(ctx context.Context, req *GenerateReportRequest) (*model.ReportGenerationJob, error) {
	// Create the job
	job := &model.ReportGenerationJob{
		TenantID:   req.TenantID,
		JobType:    string(model.ReportJobTypeCompliance),
		ReportID:   &req.ReportID,
		Status:     string(model.ReportJobStatusQueued),
		Progress:   0,
		Format:     req.Format,
		MaxRetries: 3,
	}

	// Marshal options
	if req.Options != nil {
		optionsBytes, err := json.Marshal(req.Options)
		if err == nil {
			job.Options = optionsBytes
		}
	}

	// Create the job record
	if err := g.snapshotRepo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("report_generator.QueueReportGeneration: %w", err)
	}

	return job, nil
}

// ProcessJob processes a queued report generation job
func (g *ReportGenerator) ProcessJob(ctx context.Context, jobID uuid.UUID) error {
	// Get the job
	job, err := g.snapshotRepo.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("report_generator.ProcessJob: failed to get job: %w", err)
	}

	// Update to processing status
	if err := g.snapshotRepo.UpdateJobStatus(ctx, jobID, string(model.ReportJobStatusProcessing), 0, nil); err != nil {
		return fmt.Errorf("report_generator.ProcessJob: failed to update status: %w", err)
	}

	// Parse options
	var options map[string]interface{}
	if job.Options != nil {
		_ = json.Unmarshal(job.Options, &options)
	}

	// Generate the report
	genReq := &GenerateReportRequest{
		TenantID:     job.TenantID,
		ReportID:     *job.ReportID,
		SnapshotName: fmt.Sprintf("Scheduled Report %s", time.Now().Format("2006-01-02")),
		Format:       job.Format,
		GeneratedBy:  uuid.Nil, // System-generated
		Options:      options,
	}

	result, err := g.GenerateReport(ctx, genReq)
	if err != nil {
		// Update job as failed
		_ = g.snapshotRepo.UpdateJobStatus(ctx, jobID, string(model.ReportJobStatusFailed), 0, ptr(err.Error()))
		return fmt.Errorf("report_generator.ProcessJob: generation failed: %w", err)
	}

	// Link snapshot to job
	job.SnapshotID = &result.Snapshot.ID

	// Update job as completed
	if err := g.snapshotRepo.UpdateJob(ctx, job); err != nil {
		return fmt.Errorf("report_generator.ProcessJob: failed to update job: %w", err)
	}

	_ = g.snapshotRepo.UpdateJobStatus(ctx, jobID, string(model.ReportJobStatusCompleted), 100, nil)

	return nil
}

// GetNextQueuedJob retrieves the next job to process
func (g *ReportGenerator) GetNextQueuedJob(ctx context.Context) (*model.ReportGenerationJob, error) {
	jobs, err := g.snapshotRepo.GetQueuedJobs(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("report_generator.GetNextQueuedJob: %w", err)
	}
	if len(jobs) == 0 {
		return nil, nil // No jobs available
	}
	return &jobs[0], nil
}

// ptr returns a pointer to a string
func ptr(s string) *string {
	return &s
}

// =============================================================================
// Scheduled Report Generation
// ============================================================================

// ProcessDueSchedules processes all schedules that are due for execution
func (g *ReportGenerator) ProcessDueSchedules(ctx context.Context) (int, error) {
	schedules, err := g.snapshotRepo.GetDueSchedules(ctx, 100)
	if err != nil {
		return 0, fmt.Errorf("report_generator.ProcessDueSchedules: failed to get due schedules: %w", err)
	}

	processed := 0
	for _, schedule := range schedules {
		if err := g.processSchedule(ctx, &schedule); err != nil {
			g.logger.Error().Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("Failed to process schedule")
			// Update schedule as failed run
			_ = g.snapshotRepo.UpdateScheduleAfterRun(ctx, schedule.ID, false, nil)
		} else {
			processed++
			// Calculate next run time
			nextRun := g.calculateNextRunTime(&schedule)
			_ = g.snapshotRepo.UpdateScheduleAfterRun(ctx, schedule.ID, true, &nextRun)
		}
	}

	return processed, nil
}

// processSchedule processes a single schedule
func (g *ReportGenerator) processSchedule(ctx context.Context, schedule *model.ReportSchedule) error {
	// Parse options
	var options map[string]interface{}
	if schedule.Options != nil {
		_ = json.Unmarshal(schedule.Options, &options)
	}

	// Generate the report
	genReq := &GenerateReportRequest{
		TenantID:      schedule.TenantID,
		ReportID:      schedule.ReportID,
		SnapshotName:  schedule.ScheduleName,
		Format:        schedule.Format,
		GeneratedBy:   schedule.CreatedBy,
		Options:       options,
		RetentionDays: &schedule.RetentionDays,
	}

	_, err := g.GenerateReport(ctx, genReq)
	return err
}

// calculateNextRunTime calculates the next run time for a schedule
func (g *ReportGenerator) calculateNextRunTime(schedule *model.ReportSchedule) time.Time {
	now := time.Now()

	switch schedule.ScheduleType {
	case string(ScheduleTypeDaily):
		return now.Add(24 * time.Hour)
	case string(ScheduleTypeWeekly):
		return now.Add(7 * 24 * time.Hour)
	case string(ScheduleTypeMonthly):
		return now.AddDate(0, 1, 0)
	case string(ScheduleTypeQuarterly):
		return now.AddDate(0, 3, 0)
	case string(ScheduleTypeYearly):
		return now.AddDate(1, 0, 0)
	default:
		// Default to daily
		return now.Add(24 * time.Hour)
	}
}

// ScheduleType aliases for use in report generator
const (
	ScheduleTypeDaily     ScheduleTypeEnum = "daily"
	ScheduleTypeWeekly    ScheduleTypeEnum = "weekly"
	ScheduleTypeMonthly   ScheduleTypeEnum = "monthly"
	ScheduleTypeQuarterly ScheduleTypeEnum = "quarterly"
	ScheduleTypeYearly    ScheduleTypeEnum = "yearly"
)

type ScheduleTypeEnum string
