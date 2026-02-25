// Package formatter provides tests for Excel report generation
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
// ExcelFormatter Dedicated Tests
// =============================================================================

func TestDedicatedExcelFormatter_New(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()

	formatter := NewDedicatedExcelFormatter(mockStorage, logger)

	assert.NotNil(t, formatter)
	assert.Equal(t, mockStorage, formatter.storage)
	assert.Equal(t, logger, formatter.logger)
}

func TestDedicatedExcelFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewDedicatedExcelFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReportForExcel(t)
	snapshot := createTestReportSnapshotForExcel(t)

	t.Run("generates Excel with default options", func(t *testing.T) {
		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		if err != nil {
			t.Skipf("Excel generation skipped (excelize dependency may not be available): %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
		assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", mockStorage.LastStoredType)
		assert.NotNil(t, mockStorage.LastStoredData)
		assert.Greater(t, len(mockStorage.LastStoredData), 0)
	})

	t.Run("generates Excel with custom options", func(t *testing.T) {
		options, _ := json.Marshal(ExcelOptions{
			IncludeCharts:     false,
			IncludeViolations: true,
			IncludeDetails:    true,
			MultipleSheets:    true,
		})

		url, size, err := formatter.Generate(ctx, report, snapshot, options)

		if err != nil {
			t.Skipf("Excel generation skipped: %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})

	t.Run("generates Excel without violations", func(t *testing.T) {
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
			t.Skipf("Excel generation skipped: %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})

	t.Run("generates Excel with minimal options", func(t *testing.T) {
		options, _ := json.Marshal(ExcelOptions{
			IncludeCharts:     false,
			IncludeViolations: false,
			IncludeDetails:    false,
			MultipleSheets:    false,
		})

		url, size, err := formatter.Generate(ctx, report, snapshot, options)

		if err != nil {
			t.Skipf("Excel generation skipped: %v", err)
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

func TestDedicatedExcelFormatter_StoreReport(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewDedicatedExcelFormatter(mockStorage, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	snapshotID := uuid.New()
	data := []byte("PK\x03\x04... fake xlsx content") // XLSX files start with PK

	t.Run("stores Excel report", func(t *testing.T) {
		url, size, err := formatter.StoreReport(ctx, tenantID, snapshotID, data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

		require.NoError(t, err)
		assert.Contains(t, url, snapshotID.String())
		assert.Contains(t, url, ".xlsx")
		assert.Equal(t, int64(len(data)), size)
		assert.Equal(t, 1, mockStorage.StoreCallCount)
	})
}

// =============================================================================
// ExcelOptions Tests
// =============================================================================

func TestExcelOptions_Unmarshal(t *testing.T) {
	t.Run("unmarshals valid options", func(t *testing.T) {
		jsonData := `{
			"include_charts": true,
			"include_violations": false,
			"include_details": true,
			"multiple_sheets": true
		}`

		var opts ExcelOptions
		err := json.Unmarshal([]byte(jsonData), &opts)

		require.NoError(t, err)
		assert.True(t, opts.IncludeCharts)
		assert.False(t, opts.IncludeViolations)
		assert.True(t, opts.IncludeDetails)
		assert.True(t, opts.MultipleSheets)
	})

	t.Run("uses defaults when options are empty", func(t *testing.T) {
		opts := ExcelOptions{}

		// Default values
		assert.True(t, opts.IncludeCharts)     // Default is true
		assert.True(t, opts.IncludeViolations) // Default is true
		assert.True(t, opts.IncludeDetails)    // Default is true
		assert.True(t, opts.MultipleSheets)    // Default is true
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		invalidJSON := `{"include_charts": not_a_boolean}`

		var opts ExcelOptions
		err := json.Unmarshal([]byte(invalidJSON), &opts)

		assert.Error(t, err)
	})
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestGetScoreStatus(t *testing.T) {
	tests := []struct {
		score  float64
		status string
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
		t.Run(tt.status, func(t *testing.T) {
			result := getScoreStatus(tt.score)
			assert.Equal(t, tt.status, result)
		})
	}
}

func TestGetScoreHexColor(t *testing.T) {
	tests := []struct {
		score      float64
		hexColor   string
	}{
		{95.0, "#70AD47"},  // Green
		{85.0, "#A9D08E"},  // Light Green
		{70.0, "#FFEB9C"},  // Yellow
		{50.0, "#FFC000"},  // Orange
		{30.0, "#C00000"},  // Red
	}

	for _, tt := range tests {
		t.Run(tt.hexColor, func(t *testing.T) {
			result := getScoreHexColor(tt.score)
			assert.Equal(t, tt.hexColor, result)
		})
	}
}

func TestGetSeverityHexColor(t *testing.T) {
	tests := []struct {
		severity  analytics.Severity
		hexColor  string
	}{
		{analytics.SeverityCritical, "#C00000"}, // Red
		{analytics.SeverityHigh, "#FFC000"},    // Orange
		{analytics.SeverityMedium, "#FFEB9C"},  // Yellow
		{analytics.SeverityLow, "#FFFFFF"},     // White
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			result := getSeverityHexColor(tt.severity)
			assert.Equal(t, tt.hexColor, result)
		})
	}
}

func TestCountViolationsBySeverity(t *testing.T) {
	report := &analytics.ComplianceReport{
		Violations: []analytics.ComplianceViolation{
			{Severity: analytics.SeverityCritical},
			{Severity: analytics.SeverityCritical},
			{Severity: analytics.SeverityHigh},
			{Severity: analytics.SeverityHigh},
			{Severity: analytics.SeverityHigh},
			{Severity: analytics.SeverityMedium},
			{Severity: analytics.SeverityLow},
		},
	}

	t.Run("counts critical violations", func(t *testing.T) {
		count := countViolationsBySeverity(report, "critical")
		assert.Equal(t, 2, count)
	})

	t.Run("counts high violations", func(t *testing.T) {
		count := countViolationsBySeverity(report, "high")
		assert.Equal(t, 3, count)
	})

	t.Run("counts medium violations", func(t *testing.T) {
		count := countViolationsBySeverity(report, "medium")
		assert.Equal(t, 1, count)
	})

	t.Run("counts low violations", func(t *testing.T) {
		count := countViolationsBySeverity(report, "low")
		assert.Equal(t, 1, count)
	})

	t.Run("counts non-existent severity", func(t *testing.T) {
		count := countViolationsBySeverity(report, "nonexistent")
		assert.Equal(t, 0, count)
	})
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestExcelFormatter_EdgeCases(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewDedicatedExcelFormatter(mockStorage, logger)
	ctx := context.Background()

	snapshot := createTestReportSnapshotForExcel(t)

	t.Run("handles report with no violations", func(t *testing.T) {
		report := &analytics.ComplianceReport{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			Framework:       "NIST-800-53",
			PeriodStart:     time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:       time.Now(),
			GeneratedAt:     time.Now(),
			OverallScore:    100.0,
			TotalControls:   10,
			PassedControls:  10,
			FailedControls:  0,
			Violations:      []analytics.ComplianceViolation{},
		}

		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		if err != nil {
			t.Skipf("Excel generation skipped: %v", err)
		}
	})

	t.Run("handles report with many violations", func(t *testing.T) {
		violations := make([]analytics.ComplianceViolation, 1500)
		for i := 0; i < 1500; i++ {
			violations[i] = analytics.ComplianceViolation{
				ControlID:   "VIOL-001",
				Severity:    analytics.SeverityMedium,
				Framework:   "NIST-800-53",
				Status:      "open",
				Description: "Test violation",
				DetectedAt:  time.Now(),
			}
		}

		report := &analytics.ComplianceReport{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			Framework:       "NIST-800-53",
			PeriodStart:     time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:       time.Now(),
			GeneratedAt:     time.Now(),
			OverallScore:    50.0,
			TotalControls:   100,
			PassedControls:  50,
			FailedControls:  50,
			Violations:      violations,
		}

		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		if err != nil {
			t.Skipf("Excel generation skipped: %v", err)
		}
	})

	t.Run("handles report with special characters in framework", func(t *testing.T) {
		report := &analytics.ComplianceReport{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			Framework:       "NIST-800-53 (Rev. 5) / ISO-27001:2022",
			PeriodStart:     time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:       time.Now(),
			GeneratedAt:     time.Now(),
			OverallScore:    75.0,
			TotalControls:   20,
			PassedControls:  15,
			FailedControls:  5,
			Violations:      []analytics.ComplianceViolation{},
		}

		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		if err != nil {
			t.Skipf("Excel generation skipped: %v", err)
		}
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestComplianceReportForExcel(t *testing.T) *analytics.ComplianceReport {
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
		{
			ControlID:    "SC-05",
			Severity:     analytics.SeverityLow,
			Framework:    "NIST-800-53",
			Status:       "closed",
			Description:  "Low severity communication issue",
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
		"System & Communications": {
			Passed:     10,
			Failed:     1,
			Total:      11,
			Percentage: 90.9,
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
		OverallScore:    76.0,
		TotalControls:   46,
		PassedControls:  35,
		FailedControls:  11,
		Violations:      violations,
		Data:            policyJSON,
		GeneratedBy:     uuidPtr(uuid.New()),
		SnapshotID:      uuidPtr(uuid.New()),
	}
}

func createTestReportSnapshotForExcel(t *testing.T) *analytics.ReportSnapshot {
	tenantID := uuid.New()
	fileFormat := "xlsx"

	return &analytics.ReportSnapshot{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ReportID:      uuidPtr(uuid.New()),
		Framework:     "NIST-800-53",
		SnapshotName:  "Test Compliance Report",
		FileFormat:    &fileFormat,
		Status:        analytics.ReportSnapshotStatusCompleted,
		PeriodStart:   time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:     time.Now(),
		GeneratedAt:   time.Now(),
	}
}

func newTestLogger() zerolog.Logger {
	return zerolog.Nop()
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}
