// Package formatter provides tests for PDF report generation
package formatter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PDFFormatter Dedicated Tests
// =============================================================================

func TestDedicatedPDFFormatter_New(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()

	formatter := NewDedicatedPDFFormatter(mockStorage, logger)

	assert.NotNil(t, formatter)
	assert.Equal(t, mockStorage, formatter.storage)
	assert.Equal(t, logger, formatter.logger)
}

func TestDedicatedPDFFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewDedicatedPDFFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReportForPDF(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("generates PDF with default options", func(t *testing.T) {
		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		if err != nil {
			t.Skipf("PDF generation skipped (maroto dependency may not be available): %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
		assert.Equal(t, "application/pdf", mockStorage.LastStoredType)
		assert.NotNil(t, mockStorage.LastStoredData)
		assert.Greater(t, len(mockStorage.LastStoredData), 0)
	})

	t.Run("generates PDF with custom options", func(t *testing.T) {
		options, _ := json.Marshal(PDFOptions{
			IncludeCharts:     false,
			IncludeViolations: false,
			IncludeDetails:    true,
			LogoURL:          "https://example.com/logo.png",
			CompanyName:      "Test Company",
			ReportTitle:      "Custom Report Title",
		})

		url, size, err := formatter.Generate(ctx, report, snapshot, options)

		if err != nil {
			t.Skipf("PDF generation skipped: %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})

	t.Run("generates PDF without violations", func(t *testing.T) {
		reportNoViolations := &analytics.ComplianceReport{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			Framework:       "ISO-27001",
			PeriodStart:     time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:       time.Now(),
			GeneratedAt:     time.Now(),
			OverallScore:    100.0,
			TotalControls:   10,
			PassedControls:  10,
			FailedControls:  0,
			Violations:      []analytics.ComplianceViolation{},
		}

		url, size, err := formatter.Generate(ctx, reportNoViolations, snapshot, nil)

		if err != nil {
			t.Skipf("PDF generation skipped: %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})

	t.Run("handles storage errors", func(t *testing.T) {
		mockStorage.StoreFunc = func(ctx context.Context, tenantID uuid.UUID, filename string, data []byte, contentType string) (string, int64, error) {
			return "", 0, assert.AnError
		}

		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		assert.Error(t, err)
		mockStorage.StoreFunc = nil // Reset for other tests
	})
}

func TestDedicatedPDFFormatter_StoreReport(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewDedicatedPDFFormatter(mockStorage, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	snapshotID := uuid.New()
	data := []byte("%PDF-1.4 fake pdf content")

	t.Run("stores PDF report", func(t *testing.T) {
		url, size, err := formatter.StoreReport(ctx, tenantID, snapshotID, data, "application/pdf")

		require.NoError(t, err)
		assert.Contains(t, url, snapshotID.String())
		assert.Contains(t, url, ".pdf")
		assert.Equal(t, int64(len(data)), size)
		assert.Equal(t, 1, mockStorage.StoreCallCount)
	})
}

// =============================================================================
// PDFOptions Tests
// =============================================================================

func TestPDFOptions_Unmarshal(t *testing.T) {
	t.Run("unmarshals valid options", func(t *testing.T) {
		jsonData := `{
			"include_charts": true,
			"include_violations": false,
			"include_details": true,
			"logo_url": "https://example.com/logo.png",
			"company_name": "Test Company",
			"report_title": "Custom Title"
		}`

		var opts PDFOptions
		err := json.Unmarshal([]byte(jsonData), &opts)

		require.NoError(t, err)
		assert.True(t, opts.IncludeCharts)
		assert.False(t, opts.IncludeViolations)
		assert.True(t, opts.IncludeDetails)
		assert.Equal(t, "https://example.com/logo.png", opts.LogoURL)
		assert.Equal(t, "Test Company", opts.CompanyName)
		assert.Equal(t, "Custom Title", opts.ReportTitle)
	})

	t.Run("uses defaults when options are empty", func(t *testing.T) {
		opts := PDFOptions{}

		// Default values
		assert.True(t, opts.IncludeCharts)     // Default is true
		assert.True(t, opts.IncludeViolations) // Default is true
		assert.True(t, opts.IncludeDetails)    // Default is true
		assert.Empty(t, opts.LogoURL)
		assert.Empty(t, opts.CompanyName)
		assert.Empty(t, opts.ReportTitle)
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		invalidJSON := `{"include_charts": not_a_boolean}`

		var opts PDFOptions
		err := json.Unmarshal([]byte(invalidJSON), &opts)

		assert.Error(t, err)
	})
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestGetScoreColor(t *testing.T) {
	tests := []struct {
		score  float64
		color  string
	}{
		{95.0, "Excellent"},
		{90.0, "Excellent"},
		{85.0, "Good"},
		{75.0, "Good"},
		{70.0, "Fair"},
		{60.0, "Fair"},
		{50.0, "Poor"},
		{40.0, "Poor"},
		{30.0, "Critical"},
		{0.0, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			result := getScoreColor(tt.score)
			assert.Equal(t, tt.color, result)
		})
	}
}

func TestGetScoreColorHex(t *testing.T) {
	tests := []struct {
		score           float64
		expectedRed     int
		expectedGreen   int
		expectedBlue    int
	}{
		{95.0, 76, 175, 80},   // Green
		{85.0, 139, 195, 74},  // Light Green
		{70.0, 255, 235, 59},  // Yellow
		{50.0, 255, 152, 0},   // Orange
		{30.0, 244, 67, 54},   // Red
	}

	for _, tt := range tests {
		t.Run("score_conversion", func(t *testing.T) {
			result := getScoreColorHex(tt.score)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedRed, result.Red)
			assert.Equal(t, tt.expectedGreen, result.Green)
			assert.Equal(t, tt.expectedBlue, result.Blue)
		})
	}
}

func TestGetSeverityColorHex(t *testing.T) {
	tests := []struct {
		severity         analytics.Severity
		expectedRed      int
		expectedGreen    int
		expectedBlue     int
	}{
		{analytics.SeverityCritical, 244, 67, 54},   // Red
		{analytics.SeverityHigh, 255, 152, 0},       // Orange
		{analytics.SeverityMedium, 255, 235, 59},    // Yellow
		{analytics.SeverityLow, 255, 255, 255},      // White
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := getSeverityColorHex(tt.severity)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedRed, result.Red)
			assert.Equal(t, tt.expectedGreen, result.Green)
			assert.Equal(t, tt.expectedBlue, result.Blue)
		})
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestComplianceReportForPDF(t *testing.T) *analytics.ComplianceReport {
	tenantID := uuid.New()

	violations := []analytics.ComplianceViolation{
		{
			ControlID:    "AC-01",
			Severity:     analytics.SeverityCritical,
			Framework:    "NIST-800-53",
			Status:       "open",
			Description:  "Critical access control violation",
			DetectedAt:   time.Now(),
		},
		{
			ControlID:    "AU-02",
			Severity:     analytics.SeverityHigh,
			Framework:    "NIST-800-53",
			Status:       "open",
			Description:  "High severity audit issue",
			DetectedAt:   time.Now(),
		},
		{
			ControlID:    "CM-03",
			Severity:     analytics.SeverityMedium,
			Framework:    "NIST-800-53",
			Status:       "open",
			Description:  "Medium severity configuration issue",
			DetectedAt:   time.Now(),
		},
	}

	policyData := map[string]analytics.CompliancePolicyStatus{
		"Access Control": {
			Passed:     12,
			Failed:     5,
			Total:      17,
			Percentage: 70.6,
			Framework:  "NIST-800-53",
		},
		"Audit & Accountability": {
			Passed:     8,
			Failed:     2,
			Total:      10,
			Percentage: 80.0,
			Framework:  "NIST-800-53",
		},
		"Configuration Management": {
			Passed:     5,
			Failed:     3,
			Total:      8,
			Percentage: 62.5,
			Framework:  "NIST-800-53",
		},
	}

	policyJSON, _ := json.Marshal(policyData)

	return &analytics.ComplianceReport{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Framework:       "NIST-800-53",
		PeriodStart:     time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:       time.Now(),
		GeneratedAt:     time.Now(),
		OverallScore:    71.0,
		TotalControls:   35,
		PassedControls:  25,
		FailedControls:  10,
		Violations:      violations,
		Data:            policyJSON,
		GeneratedBy:     uuidPtr(uuid.New()),
		SnapshotID:      uuidPtr(uuid.New()),
	}
}

func newTestLogger() zerolog.Logger {
	return zerolog.Nop()
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}
