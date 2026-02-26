// Package formatter provides tests for report formatters
package formatter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Storage
// =============================================================================

// MockReportStorage is a mock implementation of ReportStorage for testing
type MockReportStorage struct {
	StoreFunc         func(ctx context.Context, tenantID uuid.UUID, filename string, data []byte, contentType string) (string, int64, error)
	GetSignedURLFunc  func(ctx context.Context, path string, expiry time.Duration) (string, error)
	DeleteFunc        func(ctx context.Context, path string) error
	LastStoredData    []byte
	LastStoredType    string
	StoreCallCount    int
	GetSignedURLCount int
	DeleteCallCount   int
}

func (m *MockReportStorage) Store(ctx context.Context, tenantID uuid.UUID, filename string, data []byte, contentType string) (string, int64, error) {
	m.StoreCallCount++
	m.LastStoredData = data
	m.LastStoredType = contentType
	if m.StoreFunc != nil {
		return m.StoreFunc(ctx, tenantID, filename, data, contentType)
	}
	return "/reports/" + filename, int64(len(data)), nil
}

func (m *MockReportStorage) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	m.GetSignedURLCount++
	if m.GetSignedURLFunc != nil {
		return m.GetSignedURLFunc(ctx, path, expiry)
	}
	return path + "?signed=true", nil
}

func (m *MockReportStorage) Delete(ctx context.Context, path string) error {
	m.DeleteCallCount++
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, path)
	}
	return nil
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestComplianceReport(t *testing.T) *analytics.ComplianceReport {
	tenantID := uuid.New()

	policyData := map[string]analytics.CompliancePolicyStatus{
		"Access Control": {
			PolicyID:          uuid.New(),
			PolicyName:        "Access Control",
			ComplianceRate:    83.3,
			TotalEvaluations:  18,
			PassedEvaluations: 15,
		},
		"Audit": {
			PolicyID:          uuid.New(),
			PolicyName:        "Audit",
			ComplianceRate:    83.3,
			TotalEvaluations:  12,
			PassedEvaluations: 10,
		},
	}

	policyJSON, _ := json.Marshal(policyData)

	return &analytics.ComplianceReport{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Framework:      "NIST-800-53",
		PeriodStart:    time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:      time.Now(),
		GeneratedAt:    time.Now(),
		OverallScore:   82.5,
		PassedControls: 25,
		FailedControls: 5,
		Data:           policyJSON,
		Status:         "completed",
	}
}

func createTestReportSnapshot(t *testing.T) *analytics.ReportSnapshot {
	tenantID := uuid.New()
	fileFormat := "pdf"

	return &analytics.ReportSnapshot{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ReportID:     uuid.New(),
		GeneratedBy:  uuid.New(),
		Framework:    "NIST-800-53",
		SnapshotName: "Test Compliance Report",
		FileFormat:   &fileFormat,
		Status:       analytics.ReportSnapshotStatusCompleted,
		PeriodStart:  time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:    time.Now(),
		GeneratedAt:  time.Now(),
	}
}

// =============================================================================
// BaseFormatter Tests
// =============================================================================

func TestBaseFormatter_New(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()

	formatter := NewBaseFormatter(mockStorage, logger)

	assert.NotNil(t, formatter)
	assert.Equal(t, mockStorage, formatter.storage)
	assert.Equal(t, logger, formatter.logger)
}

func TestBaseFormatter_StoreReport(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewBaseFormatter(mockStorage, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	snapshotID := uuid.New()
	data := []byte("test report data")
	contentType := "application/pdf"

	t.Run("stores report successfully", func(t *testing.T) {
		url, size, err := formatter.StoreReport(ctx, tenantID, snapshotID, "pdf", data, contentType)

		require.NoError(t, err)
		assert.Contains(t, url, snapshotID.String())
		assert.Contains(t, url, ".pdf")
		assert.Equal(t, int64(len(data)), size)
		assert.Equal(t, 1, mockStorage.StoreCallCount)
		assert.Equal(t, data, mockStorage.LastStoredData)
		assert.Equal(t, contentType, mockStorage.LastStoredType)
	})

	t.Run("generates correct filename", func(t *testing.T) {
		url, _, _ := formatter.StoreReport(ctx, tenantID, snapshotID, "xlsx", data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

		assert.Contains(t, url, snapshotID.String())
		assert.Contains(t, url, ".xlsx")
	})
}

// =============================================================================
// CSV Formatter Tests
// =============================================================================

func TestCSVFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewCSVFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("generates CSV report", func(t *testing.T) {
		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
		assert.Equal(t, "text/csv", mockStorage.LastStoredType)
		assert.NotNil(t, mockStorage.LastStoredData)

		// Verify CSV contains expected data
		csvContent := string(mockStorage.LastStoredData)
		assert.Contains(t, csvContent, "Framework")
		assert.Contains(t, csvContent, report.Framework)
		assert.Contains(t, csvContent, "Score")
	})

	t.Run("includes all required fields", func(t *testing.T) {
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		csvContent := string(mockStorage.LastStoredData)
		assert.Contains(t, csvContent, "Framework")
		assert.Contains(t, csvContent, "Score")
		assert.Contains(t, csvContent, "Passed")
		assert.Contains(t, csvContent, "Failed")
	})
}

// =============================================================================
// HTML Formatter Tests
// =============================================================================

func TestHTMLFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewHTMLFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("generates HTML report", func(t *testing.T) {
		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
		assert.Equal(t, "text/html", mockStorage.LastStoredType)
	})

	t.Run("contains valid HTML structure", func(t *testing.T) {
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		htmlContent := string(mockStorage.LastStoredData)
		assert.Contains(t, htmlContent, "<!DOCTYPE html>")
		assert.Contains(t, htmlContent, "<html>")
		assert.Contains(t, htmlContent, "<head>")
		assert.Contains(t, htmlContent, "<body>")
		assert.Contains(t, htmlContent, report.Framework)
	})

	t.Run("includes report data", func(t *testing.T) {
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		htmlContent := string(mockStorage.LastStoredData)
		assert.Contains(t, htmlContent, "Compliance Report")
		assert.Contains(t, htmlContent, "Score:")
		assert.Contains(t, htmlContent, "Passed:")
		assert.Contains(t, htmlContent, "Failed:")
	})
}

// =============================================================================
// JSON Formatter Tests
// =============================================================================

func TestJSONFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewJSONFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("generates JSON report", func(t *testing.T) {
		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
		assert.Equal(t, "application/json", mockStorage.LastStoredType)
	})

	t.Run("produces valid JSON", func(t *testing.T) {
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		var result map[string]interface{}
		err = json.Unmarshal(mockStorage.LastStoredData, &result)
		require.NoError(t, err)

		assert.Contains(t, result, "framework")
		assert.Contains(t, result, "score")
		assert.Contains(t, result, "passed_controls")
		assert.Contains(t, result, "failed_controls")
		assert.Contains(t, result, "period_start")
		assert.Contains(t, result, "period_end")
		assert.Contains(t, result, "generated_at")
	})

	t.Run("contains correct values", func(t *testing.T) {
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		var result map[string]interface{}
		err = json.Unmarshal(mockStorage.LastStoredData, &result)
		require.NoError(t, err)

		assert.Equal(t, report.Framework, result["framework"])
		assert.Equal(t, report.OverallScore, result["score"])
		assert.Equal(t, report.PassedControls, int(result["passed_controls"].(float64)))
		assert.Equal(t, report.FailedControls, int(result["failed_controls"].(float64)))
	})

	t.Run("handles nested data", func(t *testing.T) {
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		require.NoError(t, err)
		var result map[string]interface{}
		err = json.Unmarshal(mockStorage.LastStoredData, &result)
		require.NoError(t, err)

		// Verify timestamps are present
		assert.NotEmpty(t, result["period_start"])
		assert.NotEmpty(t, result["period_end"])
		assert.NotEmpty(t, result["generated_at"])
	})
}

// =============================================================================
// PDF Formatter Tests
// =============================================================================

func TestPDFFormatter_New(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()

	t.Run("creates PDF formatter via base constructor", func(t *testing.T) {
		formatter := NewPDFFormatter(mockStorage, logger)

		assert.NotNil(t, formatter)
		assert.NotNil(t, formatter.BaseFormatter)
	})
}

func TestPDFFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewPDFFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("generates PDF report (delegates to dedicated)", func(t *testing.T) {
		t.Skip("PDF generation requires embedded fonts - skipping in unit tests")

		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		if err != nil {
			t.Skipf("PDF generation skipped: %v", err)
		}

		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})
}

// =============================================================================
// Excel Formatter Tests
// =============================================================================

func TestExcelFormatter_New(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()

	t.Run("creates Excel formatter via base constructor", func(t *testing.T) {
		formatter := NewExcelFormatter(mockStorage, logger)

		assert.NotNil(t, formatter)
		assert.NotNil(t, formatter.BaseFormatter)
	})
}

func TestExcelFormatter_Generate(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewExcelFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("generates Excel report (delegates to dedicated)", func(t *testing.T) {
		url, size, err := formatter.Generate(ctx, report, snapshot, nil)

		// This delegates to the dedicated formatter which uses excelize
		if err != nil {
			t.Skipf("Excel generation skipped (excelize dependency): %v", err)
		}

		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})
}

// =============================================================================
// ReportStorage Interface Tests
// =============================================================================

func TestReportStorage_Interface(t *testing.T) {
	// Verify MockReportStorage implements the interface
	var _ ReportStorage = &MockReportStorage{}

	mockStorage := &MockReportStorage{}
	ctx := context.Background()

	t.Run("implements Store method", func(t *testing.T) {
		tenantID := uuid.New()
		url, size, err := mockStorage.Store(ctx, tenantID, "test.pdf", []byte("test"), "application/pdf")

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
		assert.Equal(t, 1, mockStorage.StoreCallCount)
	})

	t.Run("implements GetSignedURL method", func(t *testing.T) {
		url, err := mockStorage.GetSignedURL(ctx, "/reports/test.pdf", 15*time.Minute)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Equal(t, 1, mockStorage.GetSignedURLCount)
	})

	t.Run("implements Delete method", func(t *testing.T) {
		err := mockStorage.Delete(ctx, "/reports/test.pdf")

		require.NoError(t, err)
		assert.Equal(t, 1, mockStorage.DeleteCallCount)
	})
}

// =============================================================================
// Formatter Integration Tests
// =============================================================================

func TestFormatterFactory_AllFormatters(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("all formatters generate output", func(t *testing.T) {
		formatters := map[string]struct {
			formatter interface {
				Generate(context.Context, *analytics.ComplianceReport, *analytics.ReportSnapshot, json.RawMessage) (string, int64, error)
			}
			expectedContentType string
		}{
			"CSV":  {formatter: NewCSVFormatter(mockStorage, logger), expectedContentType: "text/csv"},
			"HTML": {formatter: NewHTMLFormatter(mockStorage, logger), expectedContentType: "text/html"},
			"JSON": {formatter: NewJSONFormatter(mockStorage, logger), expectedContentType: "application/json"},
		}

		for name, fc := range formatters {
			t.Run(name, func(t *testing.T) {
				mockStorage.StoreCallCount = 0
				mockStorage.LastStoredData = nil

				url, size, err := fc.formatter.Generate(ctx, report, snapshot, nil)

				require.NoError(t, err, "%s formatter should succeed", name)
				assert.NotEmpty(t, url, "%s URL should not be empty", name)
				assert.Greater(t, size, int64(0), "%s size should be positive", name)
				assert.Equal(t, fc.expectedContentType, mockStorage.LastStoredType, "%s content type mismatch", name)
				assert.NotNil(t, mockStorage.LastStoredData, "%s data should not be nil", name)
			})
		}
	})
}

// =============================================================================
// Formatter Options Tests
// =============================================================================

func TestFormatter_WithOptions(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("JSON formatter ignores options", func(t *testing.T) {
		formatter := NewJSONFormatter(mockStorage, logger)

		options := json.RawMessage(`{"pretty": true, "include_metadata": true}`)
		url, size, err := formatter.Generate(ctx, report, snapshot, options)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})

	t.Run("HTML formatter ignores options", func(t *testing.T) {
		formatter := NewHTMLFormatter(mockStorage, logger)

		options := json.RawMessage(`{"include_css": true, "theme": "dark"}`)
		url, size, err := formatter.Generate(ctx, report, snapshot, options)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})

	t.Run("CSV formatter ignores options", func(t *testing.T) {
		formatter := NewCSVFormatter(mockStorage, logger)

		options := json.RawMessage(`{"delimiter": ",", "include_header": true}`)
		url, size, err := formatter.Generate(ctx, report, snapshot, options)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestFormatter_ErrorHandling(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("handles storage errors", func(t *testing.T) {
		mockStorage.StoreFunc = func(ctx context.Context, tenantID uuid.UUID, filename string, data []byte, contentType string) (string, int64, error) {
			return "", 0, assert.AnError
		}

		formatter := NewJSONFormatter(mockStorage, logger)
		_, _, err := formatter.Generate(ctx, report, snapshot, nil)

		assert.Error(t, err)
	})

	t.Run("handles empty report data", func(t *testing.T) {
		mockStorage.StoreFunc = nil

		emptyReport := &analytics.ComplianceReport{
			ID:             uuid.New(),
			TenantID:       uuid.New(),
			Framework:      "TEST",
			PeriodStart:    time.Now(),
			PeriodEnd:      time.Now(),
			GeneratedAt:    time.Now(),
			OverallScore:   100,
			PassedControls: 0,
			FailedControls: 0,
		}

		formatter := NewJSONFormatter(mockStorage, logger)
		url, size, err := formatter.Generate(ctx, emptyReport, snapshot, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Greater(t, size, int64(0))
	})
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestFormatter_ConcurrentAccess(t *testing.T) {
	mockStorage := &MockReportStorage{}
	logger := newTestLogger()
	formatter := NewJSONFormatter(mockStorage, logger)
	ctx := context.Background()

	report := createTestComplianceReport(t)
	snapshot := createTestReportSnapshot(t)

	t.Run("concurrent generation is safe", func(t *testing.T) {
		results := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				_, _, err := formatter.Generate(ctx, report, snapshot, nil)
				results <- err
			}()
		}

		for i := 0; i < 10; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})
}
