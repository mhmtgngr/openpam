package formatter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
	"github.com/xuri/excelize/v2"
)

// DedicatedExcelFormatter generates Excel reports using excelize
type DedicatedExcelFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

// NewDedicatedExcelFormatter creates a new dedicated Excel formatter
func NewDedicatedExcelFormatter(storage ReportStorage, logger zerolog.Logger) *DedicatedExcelFormatter {
	return &DedicatedExcelFormatter{
		storage: storage,
		logger:  logger,
	}
}

// Generate generates an Excel compliance report
func (f *DedicatedExcelFormatter) Generate(ctx context.Context, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, options json.RawMessage) (string, int64, error) {
	f.logger.Info().
		Str("report_id", report.ID.String()).
		Str("snapshot_id", snapshot.ID.String()).
		Msg("Generating Excel report")

	// Parse generation options
	opts := &ExcelOptions{
		IncludeCharts:     true,
		IncludeViolations: true,
		IncludeDetails:    true,
		MultipleSheets:    true,
	}
	if len(options) > 0 {
		if err := json.Unmarshal(options, opts); err != nil {
			f.logger.Warn().Err(err).Msg("Failed to parse Excel options, using defaults")
		}
	}

	// Create Excel xlFile
	xlFile := excelize.NewFile()
	defer func() {
		if err := xlFile.Close(); err != nil {
			f.logger.Error().Err(err).Msg("Failed to close Excel xlFile")
		}
	}()

	// Set default font and style for the entire workbook
	if err := xlFile.SetDefaultFont(&excelize.Font{
		Family: "Arial",
		Size:   10,
	}); err != nil {
		return "", 0, fmt.Errorf("set default font: %w", err)
	}

	// Create summary sheet
	if err := f.createSummarySheet(xlFile, report, snapshot, opts); err != nil {
		return "", 0, fmt.Errorf("create summary sheet: %w", err)
	}

	// Create policy breakdown sheet
	if opts.IncludeDetails {
		if err := f.createPolicySheet(xlFile, report); err != nil {
			return "", 0, fmt.Errorf("create policy sheet: %w", err)
		}
	}

	// Create violations sheet
	if opts.IncludeViolations && len(report.Violations) > 0 {
		if err := f.createViolationsSheet(xlFile, report); err != nil {
			return "", 0, fmt.Errorf("create violations sheet: %w", err)
		}
	}

	// Delete default Sheet1 if it still exists
	if err := xlFile.DeleteSheet("Sheet1"); err != nil {
		// Ignore error if sheet doesn't exist
	}

	// Set the active sheet to Summary
	if err := xlFile.SetActiveSheet(0); err != nil {
		return "", 0, fmt.Errorf("set active sheet: %w", err)
	}

	// Generate bytes
	bytes, err := xlFile.WriteToBuffer()
	if err != nil {
		return "", 0, fmt.Errorf("write excel buffer: %w", err)
	}

	// Store the report
	xlFilename := fmt.Sprintf("%s-%s.xlsx", snapshot.Framework, snapshot.ID.String())
	url, size, err := f.storage.Store(ctx, snapshot.TenantID, xlFilename, bytes.Bytes(), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err != nil {
		return "", 0, fmt.Errorf("store excel: %w", err)
	}

	f.logger.Info().
		Str("url", url).
		Int64("size", size).
		Msg("Excel report generated successfully")

	return url, size, nil
}

// ExcelOptions contains options for Excel generation
type ExcelOptions struct {
	IncludeCharts     bool   `json:"include_charts"`
	IncludeViolations bool   `json:"include_violations"`
	IncludeDetails    bool   `json:"include_details"`
	MultipleSheets    bool   `json:"multiple_sheets"`
}

// createSummarySheet creates the executive summary sheet
func (f *DedicatedExcelFormatter) createSummarySheet(xlFile *excelize.File, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, opts *ExcelOptions) error {
	sheetName := "Summary"
	index, err := xlFile.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	// Set column widths
	if err := xlFile.SetColWidth(sheetName, "A", "B", 30); err != nil {
		return err
	}
	if err := xlFile.SetColWidth(sheetName, "C", "C", 20); err != nil {
		return err
	}

	// Define styles
	headerStyle, err := xlFile.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#E0E0E0"}},
	})
	if err != nil {
		return err
	}

	titleStyle, err := xlFile.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	if err != nil {
		return err
	}

	labelStyle, err := xlFile.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	if err != nil {
		return err
	}

	scoreStyle, err := xlFile.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 24},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{getScoreHexColor(report.OverallScore)}},
	})
	if err != nil {
		return err
	}

	// Title
	if _, err := xlFile.SetCellValue(sheetName, "A1", fmt.Sprintf("%s Compliance Report", report.Framework)); err != nil {
		return err
	}
	if err := xlFile.SetCellStyle(sheetName, "A1", "C1", titleStyle); err != nil {
		return err
	}
	if err := xlFile.MergeCell(sheetName, "A1", "C1"); err != nil {
		return err
	}

	// Metadata section
	row := 3
	metadataData := map[string]string{
		"Generated":    snapshot.GeneratedAt.Format("2006-01-02 15:04:05"),
		"Period Start": snapshot.PeriodStart.Format("2006-01-02"),
		"Period End":   snapshot.PeriodEnd.Format("2006-01-02"),
		"Framework":    report.Framework,
		"Report ID":    snapshot.ID.String(),
	}

	for label, value := range metadataData {
		if err := xlFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), label); err != nil {
			return err
		}
		if err := xlFile.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), labelStyle); err != nil {
			return err
		}
		if err := xlFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), value); err != nil {
			return err
		}
		if err := xlFile.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("C%d", row)); err != nil {
			return err
		}
		row++
	}

	row += 2

	// Executive Summary header
	if _, err := xlFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Executive Summary"); err != nil {
		return err
	}
	if err := xlFile.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), headerStyle); err != nil {
		return err
	}
	if err := xlFile.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row)); err != nil {
		return err
	}
	row++

	// Score metrics
	scoreData := map[string]string{
		"Overall Score":          fmt.Sprintf("%.1f%%", report.OverallScore),
		"Compliance Status":      getScoreStatus(report.OverallScore),
		"Total Controls":         fmt.Sprintf("%d", report.TotalControls),
		"Passed Controls":        fmt.Sprintf("%d", report.PassedControls),
		"Failed Controls":        fmt.Sprintf("%d", report.FailedControls),
		"Pass Rate":              fmt.Sprintf("%.1f%%", float64(report.PassedControls)/float64(report.TotalControls)*100),
		"Critical Findings":      fmt.Sprintf("%d", countViolationsBySeverity(report, "critical")),
		"High Risk Findings":     fmt.Sprintf("%d", countViolationsBySeverity(report, "high")),
		"Medium Risk Findings":   fmt.Sprintf("%d", countViolationsBySeverity(report, "medium")),
	}

	for label, value := range scoreData {
		if err := xlFile.SetCellValue(sheetName, fmt.Sprintf("A%d", row), label); err != nil {
			return err
		}
		if err := xlFile.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), labelStyle); err != nil {
			return err
		}
		if err := xlFile.SetCellValue(sheetName, fmt.Sprintf("B%d", row), value); err != nil {
			return err
		}
		if err := xlFile.MergeCell(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("C%d", row)); err != nil {
			return err
		}

		// Apply score style to Overall Score
		if label == "Overall Score" {
			if err := xlFile.SetCellStyle(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), scoreStyle); err != nil {
				return err
			}
		}

		row++
	}

	// Set active sheet
	if err := xlFile.SetActiveSheet(index); err != nil {
		return err
	}

	return nil
}

// createPolicySheet creates the policy breakdown sheet
func (f *DedicatedExcelFormatter) createPolicySheet(xlFile *excelize.File, report *analytics.ComplianceReport) error {
	sheetName := "Policy Breakdown"
	index, err := xlFile.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	// Set column widths
	if err := xlFile.SetColWidth(sheetName, "A", "E", 20); err != nil {
		return err
	}

	// Header style
	headerStyle, err := xlFile.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{"Policy Name", "Framework", "Passed", "Failed", "Score %"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := xlFile.SetCellValue(sheetName, cell, header); err != nil {
			return err
		}
		if err := xlFile.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return err
		}
	}

	// Parse policy data
	var policyData map[string]analytics.CompliancePolicyStatus
	if len(report.Data) > 0 {
		_ = json.Unmarshal(report.Data, &policyData)
	}

	// Fill data rows
	row := 2
	for policyName, policyStatus := range policyData {
		cells := []string{
			policyName,
			policyStatus.Framework,
			fmt.Sprintf("%d", policyStatus.Passed),
			fmt.Sprintf("%d", policyStatus.Failed),
			fmt.Sprintf("%.1f", policyStatus.Percentage),
		}

		for i, value := range cells {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if err := xlFile.SetCellValue(sheetName, cell, value); err != nil {
				return err
			}

			// Add conditional formatting based on score
			if i == 4 { // Score column
				style, _ := xlFile.NewStyle(&excelize.Style{
					Font: &excelize.Font{Bold: true},
					Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{getScoreHexColor(policyStatus.Percentage)}},
				})
				if err := xlFile.SetCellStyle(sheetName, cell, cell, style); err != nil {
					return err
				}
			}
		}
		row++
	}

	// Set active sheet
	if err := xlFile.SetActiveSheet(index); err != nil {
		return err
	}

	return nil
}

// createViolationsSheet creates the violations sheet
func (f *DedicatedExcelFormatter) createViolationsSheet(xlFile *excelize.File, report *analytics.ComplianceReport) error {
	sheetName := "Violations"
	index, err := xlFile.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	// Set column widths
	if err := xlFile.SetColWidth(sheetName, "A", "A", 25); err != nil {
		return err
	}
	if err := xlFile.SetColWidth(sheetName, "B", "E", 20); err != nil {
		return err
	}
	if err := xlFile.SetColWidth(sheetName, "F", "F", 40); err != nil {
		return err
	}

	// Header style
	headerStyle, err := xlFile.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#C00000"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return err
	}

	// Define headers
	headers := []string{"Control ID", "Severity", "Framework", "Status", "Detected At", "Description"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := xlFile.SetCellValue(sheetName, cell, header); err != nil {
			return err
		}
		if err := xlFile.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return err
		}
	}

	// Fill data rows
	row := 2
	for _, violation := range report.Violations {
		if row > 1002 { // Limit to 1000 violations
			break
		}

		cells := []string{
			violation.ControlID,
			string(violation.Severity),
			violation.Framework,
			string(violation.Status),
			violation.DetectedAt.Format("2006-01-02 15:04:05"),
			violation.Description,
		}

		for i, value := range cells {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if err := xlFile.SetCellValue(sheetName, cell, value); err != nil {
				return err
			}

			// Color code severity column
			if i == 1 { // Severity column
				bgColor := getSeverityHexColor(violation.Severity)
				style, _ := xlFile.NewStyle(&excelize.Style{
					Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{bgColor}},
				})
				if err := xlFile.SetCellStyle(sheetName, cell, cell, style); err != nil {
					return err
				}
			}
		}
		row++
	}

	// Set active sheet
	if err := xlFile.SetActiveSheet(index); err != nil {
		return err
	}

	return nil
}

// Helper functions

func getScoreStatus(score float64) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 75:
		return "Good"
	case score >= 60:
		return "Fair"
	case score >= 40:
		return "Poor"
	default:
		return "Critical"
	}
}

func getScoreHexColor(score float64) string {
	switch {
	case score >= 90:
		return "#70AD47" // Green
	case score >= 75:
		return "#A9D08E" // Light Green
	case score >= 60:
		return "#FFEB9C" // Yellow
	case score >= 40:
		return "#FFC000" // Orange
	default:
		return "#C00000" // Red
	}
}

func getSeverityHexColor(severity analytics.Severity) string {
	switch severity {
	case analytics.SeverityCritical:
		return "#C00000" // Red
	case analytics.SeverityHigh:
		return "#FFC000" // Orange
	case analytics.SeverityMedium:
		return "#FFEB9C" // Yellow
	default:
		return "#FFFFFF" // White
	}
}

func countViolationsBySeverity(report *analytics.ComplianceReport, severity string) int {
	count := 0
	for _, v := range report.Violations {
		if string(v.Severity) == severity {
			count++
		}
	}
	return count
}

// StoreReport stores the Excel report data
func (f *DedicatedExcelFormatter) StoreReport(ctx context.Context, tenantID uuid.UUID, snapshotID uuid.UUID, data []byte, contentType string) (string, int64, error) {
	xlFilename := fmt.Sprintf("%s-%s.xlsx", snapshotID.String(), time.Now().Format("20060102-150405"))
	return f.storage.Store(ctx, tenantID, xlFilename, data, contentType)
}
