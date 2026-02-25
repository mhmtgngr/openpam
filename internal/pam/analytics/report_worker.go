package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ReportWorker handles asynchronous report generation jobs
type ReportWorker struct {
	reportRepo *ReportRepository
	formatter  FormatterFactory
	logger     zerolog.Logger

	// Job queue
	jobQueue chan *ReportGenerationJob

	// Worker pool
	workers    int
	activeJobs sync.Map
	running    bool
	stopCh     chan struct{}

	// Worker ID for tracking
	workerID string
}

// NewReportWorker creates a new report worker
func NewReportWorker(reportRepo *ReportRepository, formatter FormatterFactory, logger zerolog.Logger, workers int) *ReportWorker {
	if workers <= 0 {
		workers = 3 // Default worker count
	}

	return &ReportWorker{
		reportRepo: reportRepo,
		formatter:  formatter,
		logger:     logger,
		jobQueue:   make(chan *ReportGenerationJob, 1000),
		workers:    workers,
		stopCh:     make(chan struct{}),
		workerID:   uuid.New().String()[:8],
	}
}

// Start begins the report worker
func (w *ReportWorker) Start(ctx context.Context) {
	w.running = true
	w.logger.Info().Str("worker_id", w.workerID).Int("workers", w.workers).Msg("Starting report worker")

	// Start worker goroutines
	for i := 0; i < w.workers; i++ {
		go w.workerLoop(ctx, i)
	}

	// Start job dispatcher
	go w.dispatcher(ctx)

	// Start retry handler
	go w.retryHandler(ctx)

	w.logger.Info().Str("worker_id", w.workerID).Msg("Report worker started")
}

// Stop gracefully stops the report worker
func (w *ReportWorker) Stop() {
	if !w.running {
		return
	}

	w.logger.Info().Str("worker_id", w.workerID).Msg("Stopping report worker")
	w.running = false
	close(w.stopCh)
	close(w.jobQueue)
}

// EnqueueJob adds a job to the processing queue
func (w *ReportWorker) EnqueueJob(job *ReportGenerationJob) {
	if !w.running {
		w.logger.Warn().Str("job_id", job.ID.String()).Msg("Worker not running, job not enqueued")
		return
	}

	select {
	case w.jobQueue <- job:
		w.logger.Debug().
			Str("job_id", job.ID.String()).
			Str("tenant_id", job.TenantID.String()).
			Msg("Job enqueued")
	default:
		w.logger.Warn().
			Str("job_id", job.ID.String()).
			Msg("Job queue full, job not enqueued")
	}
}

// dispatcher pulls queued jobs from the database and enqueues them
func (w *ReportWorker) dispatcher(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.dispatchQueuedJobs(ctx)
		}
	}
}

// dispatchQueuedJobs pulls queued jobs from the database
func (w *ReportWorker) dispatchQueuedJobs(ctx context.Context) {
	jobs, err := w.reportRepo.ListQueuedJobs(ctx, 50)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to list queued jobs")
		return
	}

	for _, job := range jobs {
		// Update to processing
		_ = w.reportRepo.UpdateReportGenerationJobStatus(ctx, job.ID, job.TenantID, ReportJobStatusProcessing, 0, nil, nil)

		// Enqueue for processing
		w.EnqueueJob(&job)
	}
}

// workerLoop processes jobs from the queue
func (w *ReportWorker) workerLoop(ctx context.Context, workerNum int) {
	w.logger.Info().Str("worker_id", w.workerID).Int("worker_num", workerNum).Msg("Worker loop started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Str("worker_id", w.workerID).Int("worker_num", workerNum).Msg("Worker loop context cancelled")
			return
		case <-w.stopCh:
			w.logger.Info().Str("worker_id", w.workerID).Int("worker_num", workerNum).Msg("Worker loop stopped")
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				return
			}
			w.processJob(ctx, job, workerNum)
		}
	}
}

// processJob processes a single report generation job
func (w *ReportWorker) processJob(ctx context.Context, job *ReportGenerationJob, workerNum int) {
	startTime := time.Now()

	w.logger.Info().
		Str("job_id", job.ID.String()).
		Str("worker_id", w.workerID).
		Int("worker_num", workerNum).
		Str("format", job.Format).
		Msg("Processing report job")

	// Track active job
	w.activeJobs.Store(job.ID.String(), true)
	defer w.activeJobs.Delete(job.ID.String())

	// Update job status
	job.Status = ReportJobStatusProcessing
	job.WorkerID = stringPtr(fmt.Sprintf("%s-%d", w.workerID, workerNum))

	var err error
	var snapshot *ReportSnapshot
	var fileURL *string
	var fileSizeBytes *int64

	// Process based on job type
	switch job.JobType {
	case "compliance", "scheduled":
		snapshot, err = w.generateComplianceReport(ctx, job)
	case "analytics":
		snapshot, err = w.generateAnalyticsReport(ctx, job)
	case "custom":
		snapshot, err = w.generateCustomReport(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.JobType)
	}

	// Handle result
	if err != nil {
		w.logger.Error().
			Str("job_id", job.ID.String()).
			Err(err).
			Msg("Job failed")

		errorMsg := err.Error()
		_ = w.reportRepo.UpdateReportGenerationJobStatus(ctx, job.ID, job.TenantID, ReportJobStatusFailed, 0, &errorMsg, nil)

		if snapshot != nil {
			_ = w.reportRepo.UpdateReportSnapshotStatus(ctx, snapshot.ID, snapshot.TenantID, ReportSnapshotStatusFailed, nil, nil, &errorMsg)
		}

		// Schedule retry if available
		if job.RetryCount < job.MaxRetries {
			_ = w.reportRepo.IncrementJobRetryCount(ctx, job.ID, job.TenantID)
		}

		return
	}

	// Update snapshot with file info
	if snapshot != nil && fileURL != nil {
		_ = w.reportRepo.UpdateReportSnapshotStatus(ctx, snapshot.ID, snapshot.TenantID, ReportSnapshotStatusCompleted, fileURL, fileSizeBytes, nil)
	}

	// Mark job as completed
	_ = w.reportRepo.UpdateReportGenerationJobStatus(ctx, job.ID, job.TenantID, ReportJobStatusCompleted, 100, nil, snapshotIDPtr(snapshot))

	duration := time.Since(startTime)
	w.logger.Info().
		Str("job_id", job.ID.String()).
		Dur("duration", duration).
		Msg("Job completed successfully")
}

// generateComplianceReport generates a compliance report
func (w *ReportWorker) generateComplianceReport(ctx context.Context, job *ReportGenerationJob) (*ReportSnapshot, error) {
	// Get the snapshot
	snapshot, err := w.reportRepo.GetReportSnapshotByID(ctx, *job.SnapshotID, job.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	// Update progress
	_ = w.reportRepo.UpdateReportGenerationJobStatus(ctx, job.ID, job.TenantID, ReportJobStatusProcessing, 25, nil, nil)

	// Get the compliance report data - dereference ReportID pointer
	reportID := uuid.Nil
	if job.ReportID != nil {
		reportID = *job.ReportID
	}
	report, err := w.reportRepo.GetComplianceReportByID(ctx, reportID, job.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get compliance report: %w", err)
	}

	// Update progress
	_ = w.reportRepo.UpdateReportGenerationJobStatus(ctx, job.ID, job.TenantID, ReportJobStatusProcessing, 50, nil, nil)

	// Generate report in requested format - placeholder for now
	// TODO: Implement actual formatter integration
	var fileURL string
	var fileSizeBytes int64 = 1024

	// For now, mark as completed with a placeholder file URL
	fileURL = fmt.Sprintf("/reports/%s/%s.%s", job.TenantID.String(), snapshot.ID.String(), job.Format)

	if err != nil {
		return nil, fmt.Errorf("generate %s report: %w", job.Format, err)
	}

	// Update progress
	_ = w.reportRepo.UpdateReportGenerationJobStatus(ctx, job.ID, job.TenantID, ReportJobStatusProcessing, 90, nil, nil)

	// Update snapshot with file info
	snapshot.FileURL = &fileURL
	snapshot.FileSizeBytes = &fileSizeBytes

	// Generate summary
	summary := fmt.Sprintf("%s compliance report for period %s to %s. Score: %.1f%%",
		report.Framework,
		snapshot.PeriodStart.Format("2006-01-02"),
		snapshot.PeriodEnd.Format("2006-01-02"),
		report.OverallScore)
	snapshot.Summary = &summary

	return snapshot, nil
}

// generateAnalyticsReport generates an analytics report
func (w *ReportWorker) generateAnalyticsReport(ctx context.Context, job *ReportGenerationJob) (*ReportSnapshot, error) {
	// TODO: Implement analytics report generation
	return nil, fmt.Errorf("analytics report generation not implemented")
}

// generateCustomReport generates a custom report
func (w *ReportWorker) generateCustomReport(ctx context.Context, job *ReportGenerationJob) (*ReportSnapshot, error) {
	// TODO: Implement custom report generation
	return nil, fmt.Errorf("custom report generation not implemented")
}

// retryHandler handles retrying failed jobs
func (w *ReportWorker) retryHandler(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.retryFailedJobs(ctx)
		}
	}
}

// retryFailedJobs retries jobs that have remaining retries
func (w *ReportWorker) retryFailedJobs(ctx context.Context) {
	// Get failed jobs with remaining retries
	jobs, err := w.reportRepo.ListQueuedJobs(ctx, 50)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to list jobs for retry")
		return
	}

	for _, job := range jobs {
		if job.Status == ReportJobStatusQueued && job.RetryCount > 0 && job.RetryCount < job.MaxRetries {
			w.logger.Info().
				Str("job_id", job.ID.String()).
				Int("retry_count", job.RetryCount).
				Msg("Retrying failed job")
			w.EnqueueJob(&job)
		}
	}
}

// GetStatus returns the current status of the report worker
func (w *ReportWorker) GetStatus() ReportWorkerStatus {
	activeCount := 0
	w.activeJobs.Range(func(_, _ interface{}) bool {
		activeCount++
		return true
	})

	return ReportWorkerStatus{
		Running:       w.running,
		WorkerID:      w.workerID,
		Workers:       w.workers,
		ActiveJobs:    activeCount,
		QueuedJobs:    len(w.jobQueue),
		LastActivity:  time.Now(),
	}
}

// ReportWorkerStatus represents the status of the report worker
type ReportWorkerStatus struct {
	Running      bool      `json:"running"`
	WorkerID     string    `json:"worker_id"`
	Workers      int       `json:"workers"`
	ActiveJobs   int       `json:"active_jobs"`
	QueuedJobs   int       `json:"queued_jobs"`
	LastActivity time.Time `json:"last_activity"`
}

// FormatterFactory creates formatters for different report formats
type FormatterFactory interface {
	GeneratePDF(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error)
	GenerateExcel(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error)
	GenerateCSV(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error)
	GenerateHTML(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error)
	GenerateJSON(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error)
}

// DefaultFormatterFactory provides default formatters
type DefaultFormatterFactory struct {
	reportRepo *ReportRepository
	storage    ReportStorage
	logger     zerolog.Logger
}

// NewDefaultFormatterFactory creates a new default formatter factory
func NewDefaultFormatterFactory(reportRepo *ReportRepository, storage ReportStorage, logger zerolog.Logger) *DefaultFormatterFactory {
	return &DefaultFormatterFactory{
		reportRepo: reportRepo,
		storage:    storage,
		logger:     logger,
	}
}

// GeneratePDF generates a PDF report
func (f *DefaultFormatterFactory) GeneratePDF(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Generate PDF using PDF formatter
	form := NewPDFFormatter(f.storage, f.logger)
	return form.Generate(ctx, report, snapshot, options)
}

// GenerateExcel generates an Excel report
func (f *DefaultFormatterFactory) GenerateExcel(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Generate Excel using Excel formatter
	form := NewExcelFormatter(f.storage, f.logger)
	return form.Generate(ctx, report, snapshot, options)
}

// GenerateCSV generates a CSV report
func (f *DefaultFormatterFactory) GenerateCSV(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Generate CSV using CSV formatter
	form := NewCSVFormatter(f.storage, f.logger)
	return form.Generate(ctx, report, snapshot, options)
}

// GenerateHTML generates an HTML report
func (f *DefaultFormatterFactory) GenerateHTML(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Generate HTML using HTML formatter
	form := NewHTMLFormatter(f.storage, f.logger)
	return form.Generate(ctx, report, snapshot, options)
}

// GenerateJSON generates a JSON report
func (f *DefaultFormatterFactory) GenerateJSON(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Generate JSON using JSON formatter
	form := NewJSONFormatter(f.storage, f.logger)
	return form.Generate(ctx, report, snapshot, options)
}

// =============================================================================
// Formatter constructors (reference to formatter package)
// =============================================================================

// NewPDFFormatter creates a new PDF formatter
func NewPDFFormatter(storage ReportStorage, logger zerolog.Logger) PDFFormatter {
	return PDFFormatter{storage: storage, logger: logger}
}

// NewExcelFormatter creates a new Excel formatter
func NewExcelFormatter(storage ReportStorage, logger zerolog.Logger) ExcelFormatter {
	return ExcelFormatter{storage: storage, logger: logger}
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter(storage ReportStorage, logger zerolog.Logger) CSVFormatter {
	return CSVFormatter{storage: storage, logger: logger}
}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter(storage ReportStorage, logger zerolog.Logger) HTMLFormatter {
	return HTMLFormatter{storage: storage, logger: logger}
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(storage ReportStorage, logger zerolog.Logger) JSONFormatter {
	return JSONFormatter{storage: storage, logger: logger}
}

// =============================================================================
// Formatter types
// =============================================================================

// PDFFormatter generates PDF reports
type PDFFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

func (f PDFFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	data := []byte(fmt.Sprintf("PDF Report: %s Compliance Report\nScore: %.1f%%", report.Framework, report.OverallScore))
	return f.storage.Store(ctx, snapshot.TenantID, "report.pdf", data, "application/pdf")
}

// ExcelFormatter generates Excel reports
type ExcelFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

func (f ExcelFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	data := []byte(fmt.Sprintf("Framework,Score,Passed,Failed\n%s,%.1f,%d,%d",
		report.Framework, report.OverallScore, report.PassedControls, report.FailedControls))
	return f.storage.Store(ctx, snapshot.TenantID, "report.xlsx", data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
}

// CSVFormatter generates CSV reports
type CSVFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

func (f CSVFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	data := []byte(fmt.Sprintf("Framework,Score,Passed,Failed\n%s,%.1f,%d,%d",
		report.Framework, report.OverallScore, report.PassedControls, report.FailedControls))
	return f.storage.Store(ctx, snapshot.TenantID, "report.csv", data, "text/csv")
}

// HTMLFormatter generates HTML reports
type HTMLFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

func (f HTMLFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>%s Report</title></head><body>
<h1>%s Compliance Report</h1>
<p>Score: %.1f%%</p>
<p>Passed: %d | Failed: %d</p>
</body></html>`, report.Framework, report.Framework, report.OverallScore, report.PassedControls, report.FailedControls)
	return f.storage.Store(ctx, snapshot.TenantID, "report.html", []byte(html), "text/html")
}

// JSONFormatter generates JSON reports
type JSONFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

func (f JSONFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	reportData := map[string]interface{}{
		"framework":        report.Framework,
		"score":           report.OverallScore,
		"passed_controls": report.PassedControls,
		"failed_controls": report.FailedControls,
		"period_start":    report.PeriodStart,
		"period_end":      report.PeriodEnd,
		"generated_at":    report.GeneratedAt,
	}
	data, _ := json.MarshalIndent(reportData, "", "  ")
	return f.storage.Store(ctx, snapshot.TenantID, "report.json", data, "application/json")
}

// ReportStorage handles storing generated reports
type ReportStorage interface {
	Store(ctx context.Context, tenantID uuid.UUID, filename string, data []byte, contentType string) (string, int64, error)
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, path string) error
}

func snapshotIDPtr(s *ReportSnapshot) *uuid.UUID {
	if s == nil {
		return nil
	}
	return &s.ID
}
