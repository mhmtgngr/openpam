package formatter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
	"github.com/signintech/gopdf"
)

// DedicatedPDFFormatter generates PDF reports using gopdf
type DedicatedPDFFormatter struct {
	storage ReportStorage
	logger  zerolog.Logger
}

// NewDedicatedPDFFormatter creates a new dedicated PDF formatter
func NewDedicatedPDFFormatter(storage ReportStorage, logger zerolog.Logger) *DedicatedPDFFormatter {
	return &DedicatedPDFFormatter{
		storage: storage,
		logger:  logger,
	}
}

// Generate generates a PDF compliance report
func (f *DedicatedPDFFormatter) Generate(ctx context.Context, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, options json.RawMessage) (string, int64, error) {
	f.logger.Info().
		Str("report_id", report.ID.String()).
		Str("snapshot_id", snapshot.ID.String()).
		Msg("Generating PDF report")

	// Parse generation options
	opts := &PDFOptions{
		IncludeCharts:     true,
		IncludeViolations: true,
		IncludeDetails:    true,
		LogoURL:          "",
	}
	if len(options) > 0 {
		if err := json.Unmarshal(options, opts); err != nil {
			f.logger.Warn().Err(err).Msg("Failed to parse PDF options, using defaults")
		}
	}

	// Create gopdf document (A4 size: 210mm x 297mm)
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	// Build PDF content
	if err := f.buildReport(&pdf, report, snapshot, opts); err != nil {
		return "", 0, fmt.Errorf("build pdf: %w", err)
	}

	// Generate PDF bytes
	var buf []byte
	if err := pdf.Write(&buf); err != nil {
		return "", 0, fmt.Errorf("generate pdf: %w", err)
	}

	// Store the report
	filename := fmt.Sprintf("%s-%s.pdf", snapshot.Framework, snapshot.ID.String())
	url, size, err := f.storage.Store(ctx, snapshot.TenantID, filename, buf, "application/pdf")
	if err != nil {
		return "", 0, fmt.Errorf("store pdf: %w", err)
	}

	f.logger.Info().
		Str("url", url).
		Int64("size", size).
		Msg("PDF report generated successfully")

	return url, size, nil
}

// PDFOptions contains options for PDF generation
type PDFOptions struct {
	IncludeCharts     bool   `json:"include_charts"`
	IncludeViolations bool   `json:"include_violations"`
	IncludeDetails    bool   `json:"include_details"`
	LogoURL           string `json:"logo_url"`
	CompanyName       string `json:"company_name"`
	ReportTitle       string `json:"report_title"`
}

// buildReport builds the PDF report content
func (f *DedicatedPDFFormatter) buildReport(pdf *gopdf.GoPdf, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, opts *PDFOptions) error {
	yPos := 30.0

	// Add header
	yPos = f.addHeader(pdf, report, snapshot, yPos)

	// Add summary section
	yPos = f.addSummary(pdf, report, snapshot, yPos)

	// Add score visualization
	yPos = f.addScoreVisualization(pdf, report, yPos)

	// Check for page break
	if yPos > 220 {
		pdf.AddPage()
		yPos = 30
	}

	// Add policy breakdown
	if opts.IncludeDetails {
		yPos = f.addPolicyBreakdown(pdf, report, yPos)
	}

	// Check for page break
	if yPos > 220 {
		pdf.AddPage()
		yPos = 30
	}

	// Add violations section
	if opts.IncludeViolations && len(report.Violations) > 0 {
		if err := f.addViolations(pdf, report, yPos); err != nil {
			return err
		}
	}

	// Add footer
	f.addFooter(pdf, snapshot)

	return nil
}

// addHeader adds the report header
func (f *DedicatedPDFFormatter) addHeader(pdf *gopdf.GoPdf, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, yPos float64) float64 {
	pdf.SetFont("Arial", "", 14)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, fmt.Sprintf("%s Compliance Report", report.Framework))
	pdf.Br(20)

	yPos += 20

	pdf.SetFont("Arial", "", 10)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, fmt.Sprintf("Generated: %s", snapshot.GeneratedAt.Format("2006-01-02 15:04:05")))
	pdf.Br(12)

	yPos += 12
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, fmt.Sprintf("Period: %s to %s",
		snapshot.PeriodStart.Format("2006-01-02"),
		snapshot.PeriodEnd.Format("2006-01-02")))
	pdf.Br(15)

	return yPos + 25
}

// addSummary adds the executive summary
func (f *DedicatedPDFFormatter) addSummary(pdf *gopdf.GoPdf, report *analytics.ComplianceReport, snapshot *analytics.ReportSnapshot, yPos float64) float64 {
	pdf.SetFont("Arial", "B", 14)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, "Executive Summary")
	pdf.Br(15)

	yPos += 15

	// Overall score
	pdf.SetFont("Arial", "B", 28)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, fmt.Sprintf("%.1f%%", report.OverallScore))
	pdf.Br(18)

	yPos += 18

	// Stats row
	pdf.SetFont("Arial", "", 11)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, fmt.Sprintf("Passed: %d   Failed: %d   Total: %d",
		report.PassedControls, report.FailedControls, report.TotalControls))
	pdf.Br(12)

	return yPos + 20
}

// addScoreVisualization adds a visual representation of the score
func (f *DedicatedPDFFormatter) addScoreVisualization(pdf *gopdf.GoPdf, report *analytics.ComplianceReport, yPos float64) float64 {
	pdf.SetFont("Arial", "B", 12)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, "Compliance Score Visualization")
	pdf.Br(12)

	yPos += 12

	// Draw score bar
	barWidth := 170.0
	barHeight := 12.0
	fillWidth := (report.OverallScore / 100) * barWidth

	// Background bar (gray)
	pdf.SetFillColor(200, 200, 200)
	pdf.RectFromX(20, yPos, barWidth, barHeight, "F", 0, 0)

	// Fill bar (colored)
	fillColor := getPDFScoreColor(report.OverallScore)
	pdf.SetFillColor(fillColor.R, fillColor.G, fillColor.B)
	pdf.RectFromX(20, yPos, fillWidth, barHeight, "F", 0, 0)

	// Reset fill
	pdf.SetFillColor(0, 0, 0)

	// Score text
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(20 + barWidth/2 - 20)
	pdf.SetY(yPos + 2)
	pdf.Cell(nil, fmt.Sprintf("%.1f%%", report.OverallScore))
	pdf.Br(15)

	return yPos + 20
}

// addPolicyBreakdown adds detailed policy breakdown
func (f *DedicatedPDFFormatter) addPolicyBreakdown(pdf *gopdf.GoPdf, report *analytics.ComplianceReport, yPos float64) float64 {
	pdf.SetFont("Arial", "B", 14)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, "Policy Breakdown")
	pdf.Br(15)

	yPos += 15

	// Table header
	pdf.SetFillColor(200, 200, 200)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(20)
	pdf.SetY(yPos)

	// Header row
	headerY := yPos
	pdf.CellWithOption(80, 10, "Policy", gopdf.CellOption{Align: gopdf.AlignLeft, Fill: true})
	pdf.SetX(100)
	pdf.SetY(headerY)
	pdf.CellWithOption(30, 10, "Passed", gopdf.CellOption{Align: gopdf.AlignCenter, Fill: true})
	pdf.SetX(130)
	pdf.SetY(headerY)
	pdf.CellWithOption(30, 10, "Failed", gopdf.CellOption{Align: gopdf.AlignCenter, Fill: true})
	pdf.SetX(160)
	pdf.SetY(headerY)
	pdf.CellWithOption(30, 10, "Score", gopdf.CellOption{Align: gopdf.AlignCenter, Fill: true})
	pdf.Br(12)

	yPos += 12
	pdf.SetFillColor(255, 255, 255)

	// Parse policy breakdown from report data
	var policyData map[string]analytics.CompliancePolicyStatus
	if len(report.Data) > 0 {
		_ = json.Unmarshal(report.Data, &policyData)
	}

	// Data rows
	pdf.SetFont("Arial", "", 9)
	isAlt := false

	for policyName, policyStatus := range policyData {
		if isAlt {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.SetX(20)
		pdf.SetY(yPos)
		pdf.CellWithOption(80, 9, policyName, gopdf.CellOption{Align: gopdf.AlignLeft, Fill: true})
		pdf.SetX(100)
		pdf.SetY(yPos)
		pdf.CellWithOption(30, 9, fmt.Sprintf("%d", policyStatus.Passed), gopdf.CellOption{Align: gopdf.AlignCenter, Fill: true})
		pdf.SetX(130)
		pdf.SetY(yPos)
		pdf.CellWithOption(30, 9, fmt.Sprintf("%d", policyStatus.Failed), gopdf.CellOption{Align: gopdf.AlignCenter, Fill: true})
		pdf.SetX(160)
		pdf.SetY(yPos)
		pdf.CellWithOption(30, 9, fmt.Sprintf("%.1f%%", policyStatus.Percentage), gopdf.CellOption{Align: gopdf.AlignCenter, Fill: true})
		pdf.Br(9)

		yPos += 9
		isAlt = !isAlt

		// Check if we need a new page
		if yPos > 260 {
			pdf.AddPage()
			yPos = 30
		}
	}

	pdf.SetFillColor(0, 0, 0)
	return yPos + 10
}

// addViolations adds the violations section
func (f *DedicatedPDFFormatter) addViolations(pdf *gopdf.GoPdf, report *analytics.ComplianceReport, yPos float64) error {
	pdf.SetFont("Arial", "B", 14)
	pdf.SetX(20)
	pdf.SetY(yPos)
	pdf.Cell(nil, "Compliance Violations")
	pdf.Br(15)

	yPos += 15

	if len(report.Violations) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.SetX(20)
		pdf.SetY(yPos)
		pdf.Cell(nil, "No violations found. All controls passed.")
		pdf.Br(12)
		return nil
	}

	// Limit to top 20 violations
	maxViolations := 20
	if len(report.Violations) < maxViolations {
		maxViolations = len(report.Violations)
	}

	for i := 0; i < maxViolations; i++ {
		violation := report.Violations[i]

		// Color code by severity
		severityColor := getPDFSeverityColor(violation.Severity)
		pdf.SetFillColor(severityColor.R, severityColor.G, severityColor.B)

		pdf.SetFont("Arial", "B", 11)
		pdf.SetX(20)
		pdf.SetY(yPos)
		pdf.CellWithOption(170, 10, fmt.Sprintf("%d. %s", i+1, violation.ControlID), gopdf.CellOption{Fill: true})
		pdf.Br(12)

		yPos += 12
		pdf.SetFillColor(255, 255, 255)

		pdf.SetFont("Arial", "", 9)
		pdf.SetX(20)
		pdf.SetY(yPos)
		pdf.Cell(nil, fmt.Sprintf("Severity: %s | Framework: %s", violation.Severity, violation.Framework))
		pdf.Br(9)
		yPos += 9

		pdf.SetX(20)
		pdf.SetY(yPos)
		description := violation.Description
		if len(description) > 100 {
			description = description[:100] + "..."
		}
		pdf.MultiCell(170, 8, description, "", gopdf.AlignLeft, false)
		pdf.Br(5)
		yPos += 5

		// Check if we need a new page
		if yPos > 250 {
			pdf.AddPage()
			yPos = 30
		}
	}

	return nil
}

// addFooter adds footer information
func (f *DedicatedPDFFormatter) addFooter(pdf *gopdf.GoPdf, snapshot *analytics.ReportSnapshot) {
	// Add page number at bottom
	pdf.SetFont("Arial", "I", 8)
	pdf.SetX(20)
	pdf.SetY(280)
	pdf.Cell(nil, fmt.Sprintf("Report ID: %s | Generated by OpenPAM on %s",
		snapshot.ID.String(),
		time.Now().Format("2006-01-02 15:04:05")))
}

// Helper functions for colors

// PDFColor represents RGB color values (0-255)
type PDFColor struct {
	R, G, B int
}

func getPDFScoreColor(score float64) PDFColor {
	switch {
	case score >= 90:
		return PDFColor{76, 175, 80}  // Green
	case score >= 75:
		return PDFColor{139, 195, 74} // Light Green
	case score >= 60:
		return PDFColor{255, 235, 59} // Yellow
	case score >= 40:
		return PDFColor{255, 152, 0}  // Orange
	default:
		return PDFColor{244, 67, 54}  // Red
	}
}

func getPDFSeverityColor(severity analytics.Severity) PDFColor {
	switch severity {
	case analytics.SeverityCritical:
		return PDFColor{244, 67, 54} // Red
	case analytics.SeverityHigh:
		return PDFColor{255, 152, 0} // Orange
	case analytics.SeverityMedium:
		return PDFColor{255, 235, 59} // Yellow
	default:
		return PDFColor{200, 200, 200} // Gray
	}
}

// StoreReport stores the PDF report data
func (f *DedicatedPDFFormatter) StoreReport(ctx context.Context, tenantID uuid.UUID, snapshotID uuid.UUID, data []byte, contentType string) (string, int64, error) {
	filename := fmt.Sprintf("%s-%s.pdf", snapshotID.String(), time.Now().Format("20060102-150405"))
	return f.storage.Store(ctx, tenantID, filename, data, contentType)
}
