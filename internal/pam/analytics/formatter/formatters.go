package formatter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
)

// BaseFormatter provides common functionality for all formatters
type BaseFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

// NewBaseFormatter creates a new base formatter
func NewBaseFormatter(storage ReportStorage, logger zerolog.Logger) *BaseFormatter {
	return &BaseFormatter{
		storage: storage,
		logger:  logger,
	}
}

// StoreReport stores the report data and returns the URL and size
func (f *BaseFormatter) StoreReport(ctx context.Context, tenantID uuid.UUID, snapshotID uuid.UUID, format string, data []byte, contentType string) (string, int64, error) {
	filename := fmt.Sprintf("%s-%s.%s", snapshotID.String(), time.Now().Format("20060102-150405"), format)
	return f.storage.Store(ctx, tenantID, filename, data, contentType)
}

// ReportStorage handles storing generated reports
type ReportStorage interface {
	Store(ctx context.Context, tenantID uuid.UUID, filename string, data []byte, contentType string) (string, int64, error)
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, path string) error
}

// =============================================================================
// PDF Formatter
// =============================================================================

// PDFFormatter generates PDF reports
type PDFFormatter struct {
	*BaseFormatter
}

// NewPDFFormatter creates a new PDF formatter
func NewPDFFormatter(storage ReportStorage, logger zerolog.Logger) *PDFFormatter {
	return &PDFFormatter{
		BaseFormatter: NewBaseFormatter(storage, logger),
	}
}

// Generate generates a PDF report
func (f *PDFFormatter) Generate(ctx context.Context, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Delegate to the dedicated PDF formatter
	dedicatedFormatter := NewDedicatedPDFFormatter(f.storage, f.logger)
	return dedicatedFormatter.Generate(ctx, report, snapshot, options)
}

// =============================================================================
// Excel Formatter
// =============================================================================

// ExcelFormatter generates Excel reports
type ExcelFormatter struct {
	*BaseFormatter
}

// NewExcelFormatter creates a new Excel formatter
func NewExcelFormatter(storage ReportStorage, logger zerolog.Logger) *ExcelFormatter {
	return &ExcelFormatter{
		BaseFormatter: NewBaseFormatter(storage, logger),
	}
}

// Generate generates an Excel report
func (f *ExcelFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	// Delegate to the dedicated Excel formatter
	dedicatedFormatter := NewDedicatedExcelFormatter(f.storage, f.logger)
	return dedicatedFormatter.Generate(ctx, report, snapshot, options)
}

// =============================================================================
// CSV Formatter
// =============================================================================

// CSVFormatter generates CSV reports
type CSVFormatter struct {
	*BaseFormatter
}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter(storage ReportStorage, logger zerolog.Logger) *CSVFormatter {
	return &CSVFormatter{
		BaseFormatter: NewBaseFormatter(storage, logger),
	}
}

// Generate generates a CSV report
func (f *CSVFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	data := []byte(fmt.Sprintf("Framework,Score,Passed,Failed\n%s,%.1f,%d,%d",
		report.Framework, report.OverallScore, report.PassedControls, report.FailedControls))

	url, size, err := f.StoreReport(ctx, snapshot.TenantID, snapshot.ID, "csv", data, "text/csv")
	if err != nil {
		return "", 0, fmt.Errorf("store csv: %w", err)
	}

	return url, size, nil
}

// =============================================================================
// HTML Formatter
// =============================================================================

// HTMLFormatter generates HTML reports
type HTMLFormatter struct {
	*BaseFormatter
}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter(storage ReportStorage, logger zerolog.Logger) *HTMLFormatter {
	return &HTMLFormatter{
		BaseFormatter: NewBaseFormatter(storage, logger),
	}
}

// Generate generates an HTML report
func (f *HTMLFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>%s Report</title></head><body>
<h1>%s Compliance Report</h1>
<p>Score: %.1f%%</p>
<p>Passed: %d | Failed: %d</p>
</body></html>`, report.Framework, report.Framework, report.OverallScore, report.PassedControls, report.FailedControls)

	data := []byte(html)

	url, size, err := f.StoreReport(ctx, snapshot.TenantID, snapshot.ID, "html", data, "text/html")
	if err != nil {
		return "", 0, fmt.Errorf("store html: %w", err)
	}

	return url, size, nil
}

// =============================================================================
// JSON Formatter
// =============================================================================

// JSONFormatter generates JSON reports
type JSONFormatter struct {
	*BaseFormatter
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(storage ReportStorage, logger zerolog.Logger) *JSONFormatter {
	return &JSONFormatter{
		BaseFormatter: NewBaseFormatter(storage, logger),
	}
}

// Generate generates a JSON report
func (f *JSONFormatter) Generate(ctx context.Context, report *ComplianceReport, snapshot *ReportSnapshot, options json.RawMessage) (string, int64, error) {
	reportData := map[string]interface{}{
		"framework":        report.Framework,
		"score":           report.OverallScore,
		"passed_controls": report.PassedControls,
		"failed_controls": report.FailedControls,
		"period_start":    report.PeriodStart,
		"period_end":      report.PeriodEnd,
		"generated_at":    report.GeneratedAt,
	}

	data, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("marshal json: %w", err)
	}

	url, size, err := f.StoreReport(ctx, snapshot.TenantID, snapshot.ID, "json", data, "application/json")
	if err != nil {
		return "", 0, fmt.Errorf("store json: %w", err)
	}

	return url, size, nil
}
