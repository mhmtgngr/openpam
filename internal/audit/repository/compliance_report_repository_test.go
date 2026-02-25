// Package repository provides tests for compliance report repository
package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupComplianceReportTest creates a test database with compliance reports table
func setupComplianceReportTest(t *testing.T) (*sqlx.DB, *ComplianceReportRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)

	// Create compliance_reports table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS compliance_reports (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			report_name TEXT NOT NULL,
			framework TEXT NOT NULL,
			version TEXT,
			generated_at TIMESTAMPTZ NOT NULL,
			generated_by UUID NOT NULL,
			status TEXT NOT NULL,
			overall_score FLOAT,
			total_controls INTEGER NOT NULL DEFAULT 0,
			passed_controls INTEGER NOT NULL DEFAULT 0,
			failed_controls INTEGER NOT NULL DEFAULT 0,
			skipped_controls INTEGER NOT NULL DEFAULT 0,
			period_start TIMESTAMPTZ,
			period_end TIMESTAMPTZ,
			summary TEXT,
			findings BYTEA,
			recommendations BYTEA,
			metadata BYTEA,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_compliance_reports_tenant ON compliance_reports(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_compliance_reports_framework ON compliance_reports(framework);
		CREATE INDEX IF NOT EXISTS idx_compliance_reports_status ON compliance_reports(status);
		CREATE INDEX IF NOT EXISTS idx_compliance_reports_generated ON compliance_reports(generated_at);
	`)
	require.NoError(t, err)

	logger := opamtesting.Logger(t)
	repo := NewComplianceReportRepository(db, logger)

	return db, repo
}

func TestNewComplianceReportRepository(t *testing.T) {
	db, _ := setupComplianceReportTest(t)
	if db == nil {
		return
	}

	logger := opamtesting.Logger(t)
	repo := NewComplianceReportRepository(db, logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

// =============================================================================
// Create Tests
// =============================================================================

func TestComplianceReportRepository_Create(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now
	score := 87.5
	summary := "Compliance evaluation completed"

	t.Run("create new compliance report", func(t *testing.T) {
		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     "Q4 2024 SOC2 Report",
			Framework:      model.FrameworkSOC2,
			Version:        "2022",
			GeneratedAt:    now,
			GeneratedBy:    generatedBy,
			Status:         string(model.ComplianceStatusPassed),
			OverallScore:   &score,
			TotalControls:  50,
			PassedControls: 45,
			FailedControls: 3,
			SkippedControls: 2,
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
			Summary:        &summary,
			Findings:       []byte(`[{"severity":"high"}]`),
			Recommendations: []byte(`[{"action":"review"}]`),
			Metadata:       []byte(`{"assessor":"john.doe"}`),
		}

		err := repo.Create(ctx, report)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, report.ID)
		assert.False(t, report.CreatedAt.IsZero())
	})

	t.Run("create report with minimal fields", func(t *testing.T) {
		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     "Minimal Report",
			Framework:      model.FrameworkISO27001,
			GeneratedAt:    now,
			GeneratedBy:    generatedBy,
			Status:         string(model.ComplianceStatusPending),
			TotalControls:  10,
			PassedControls: 0,
			FailedControls: 0,
			SkippedControls: 10,
		}

		err := repo.Create(ctx, report)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, report.ID)
	})

	t.Run("create report with nil score", func(t *testing.T) {
		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     "Pending Report",
			Framework:      model.FrameworkPCIDSS,
			GeneratedAt:    now,
			GeneratedBy:    generatedBy,
			Status:         string(model.ComplianceStatusPending),
			OverallScore:   nil,
			TotalControls:  100,
			PassedControls: 0,
			FailedControls: 0,
			SkippedControls: 100,
		}

		err := repo.Create(ctx, report)
		require.NoError(t, err)
	})
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestComplianceReportRepository_GetByID(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()
	score := 85.0

	// Create a test report
	created := &model.ComplianceReport{
		TenantID:       tenantID,
		ReportName:     "Test Report",
		Framework:      model.FrameworkSOC2,
		GeneratedAt:    now,
		GeneratedBy:    generatedBy,
		Status:         string(model.ComplianceStatusPassed),
		OverallScore:   &score,
		TotalControls:  20,
		PassedControls: 18,
		FailedControls: 2,
		SkippedControls: 0,
	}

	err := repo.Create(ctx, created)
	require.NoError(t, err)

	t.Run("get existing report", func(t *testing.T) {
		found, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)

		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, tenantID, found.TenantID)
		assert.Equal(t, "Test Report", found.ReportName)
		assert.Equal(t, model.FrameworkSOC2, found.Framework)
		assert.Equal(t, string(model.ComplianceStatusPassed), found.Status)
		assert.Equal(t, 85.0, *found.OverallScore)
		assert.Equal(t, 20, found.TotalControls)
		assert.Equal(t, 18, found.PassedControls)
	})

	t.Run("get non-existent report", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestComplianceReportRepository_List(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()

	// Create test reports
	for i := 0; i < 5; i++ {
		score := float64(70 + i*5)
		status := string(model.ComplianceStatusPassed)
		if i%2 == 0 {
			status = string(model.ComplianceStatusFailed)
		}

		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     fmt.Sprintf("Report %d", i),
			Framework:      model.FrameworkSOC2,
			GeneratedAt:    now.Add(time.Duration(-i) * time.Hour),
			GeneratedBy:    generatedBy,
			Status:         status,
			OverallScore:   &score,
			TotalControls:  50,
			PassedControls: 40 + i,
			FailedControls: 10 - i,
			SkippedControls: 0,
		}
		_ = repo.Create(ctx, report)
	}

	t.Run("list all reports for tenant", func(t *testing.T) {
		filter := model.ComplianceReportFilter{}
		reports, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(reports), 5)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("list with framework filter", func(t *testing.T) {
		framework := model.FrameworkSOC2
		filter := model.ComplianceReportFilter{
			Framework: &framework,
		}

		reports, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(reports), 5)
		assert.GreaterOrEqual(t, total, 5)

		// Verify all results have SOC2 framework
		for _, r := range reports {
			assert.Equal(t, model.FrameworkSOC2, r.Framework)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		status := string(model.ComplianceStatusPassed)
		filter := model.ComplianceReportFilter{
			Status: &status,
		}

		reports, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(reports), 0)

		for _, r := range reports {
			assert.Equal(t, "passed", r.Status)
		}
	})

	t.Run("list with date range filter", func(t *testing.T) {
		dateFrom := now.Add(-2 * time.Hour)
		dateTo := now.Add(2 * time.Hour)

		filter := model.ComplianceReportFilter{
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
		}

		reports, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(reports), 0)
	})

	t.Run("list with pagination", func(t *testing.T) {
		filter := model.ComplianceReportFilter{}

		reports, total, err := repo.List(ctx, tenantID, filter, 2, 0)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(reports), 2)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("list with offset", func(t *testing.T) {
		filter := model.ComplianceReportFilter{}

		// Get first page
		page1, _, _ := repo.List(ctx, tenantID, filter, 2, 0)

		// Get second page
		page2, _, _ := repo.List(ctx, tenantID, filter, 2, 2)

		// Pages should have different content
		if len(page1) > 0 && len(page2) > 0 {
			assert.NotEqual(t, page1[0].ID, page2[0].ID)
		}
	})
}

// =============================================================================
// GetLatestByFramework Tests
// =============================================================================

func TestComplianceReportRepository_GetLatestByFramework(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()
	score := 90.0

	// Create multiple SOC2 reports
	for i := 0; i < 3; i++ {
		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     fmt.Sprintf("SOC2 Report %d", i),
			Framework:      model.FrameworkSOC2,
			GeneratedAt:    now.Add(time.Duration(-i) * 24 * time.Hour),
			GeneratedBy:    generatedBy,
			Status:         string(model.ComplianceStatusPassed),
			OverallScore:   &score,
			TotalControls:  50,
			PassedControls: 45,
			FailedControls: 5,
			SkippedControls: 0,
		}
		_ = repo.Create(ctx, report)
	}

	t.Run("get latest by framework", func(t *testing.T) {
		latest, err := repo.GetLatestByFramework(ctx, tenantID, model.FrameworkSOC2)

		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.Equal(t, model.FrameworkSOC2, latest.Framework)
		assert.Equal(t, "SOC2 Report 0", latest.ReportName) // Most recent
	})

	t.Run("get latest for non-existent framework", func(t *testing.T) {
		_, err := repo.GetLatestByFramework(ctx, tenantID, model.FrameworkHIPAA)

		assert.Error(t, err)
	})
}

// =============================================================================
// UpdateStatus Tests
// =============================================================================

func TestComplianceReportRepository_UpdateStatus(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()
	score := 50.0

	report := &model.ComplianceReport{
		TenantID:       tenantID,
		ReportName:     "Pending Report",
		Framework:      model.FrameworkSOC2,
		GeneratedAt:    now,
		GeneratedBy:    generatedBy,
		Status:         string(model.ComplianceStatusPending),
		OverallScore:   &score,
		TotalControls:  50,
		PassedControls: 0,
		FailedControls: 0,
		SkippedControls: 50,
	}

	err := repo.Create(ctx, report)
	require.NoError(t, err)

	t.Run("update status to passed", func(t *testing.T) {
		newScore := 95.0
		err := repo.UpdateStatus(ctx, report.ID, string(model.ComplianceStatusPassed), &newScore, 45, 3, 2)

		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, report.ID)
		require.NoError(t, err)

		assert.Equal(t, string(model.ComplianceStatusPassed), updated.Status)
		assert.Equal(t, 95.0, *updated.OverallScore)
		assert.Equal(t, 45, updated.PassedControls)
		assert.Equal(t, 3, updated.FailedControls)
		assert.Equal(t, 2, updated.SkippedControls)
	})

	t.Run("update status with nil score", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, report.ID, string(model.ComplianceStatusFailed), nil, 20, 25, 5)

		require.NoError(t, err)

		updated, _ := repo.GetByID(ctx, report.ID)
		assert.Equal(t, string(model.ComplianceStatusFailed), updated.Status)
		assert.Nil(t, updated.OverallScore)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestComplianceReportRepository_Delete(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()
	score := 80.0

	report := &model.ComplianceReport{
		TenantID:       tenantID,
		ReportName:     "To Delete",
		Framework:      model.FrameworkSOC2,
		GeneratedAt:    now,
		GeneratedBy:    generatedBy,
		Status:         string(model.ComplianceStatusPassed),
		OverallScore:   &score,
		TotalControls:  50,
		PassedControls: 40,
		FailedControls: 10,
		SkippedControls: 0,
	}

	err := repo.Create(ctx, report)
	require.NoError(t, err)

	t.Run("delete existing report", func(t *testing.T) {
		err := repo.Delete(ctx, report.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetByID(ctx, report.ID)
		assert.Error(t, err)
	})

	t.Run("delete non-existent report", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows affected")
	})
}

// =============================================================================
// GetByPeriod Tests
// =============================================================================

func TestComplianceReportRepository_GetByPeriod(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()
	score := 85.0

	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now

	// Create reports with different periods
	report1 := &model.ComplianceReport{
		TenantID:       tenantID,
		ReportName:     "Q1 Report",
		Framework:      model.FrameworkSOC2,
		GeneratedAt:    now.Add(-90 * 24 * time.Hour),
		GeneratedBy:    generatedBy,
		Status:         string(model.ComplianceStatusPassed),
		OverallScore:   &score,
		TotalControls:  50,
		PassedControls: 45,
		FailedControls: 5,
		SkippedControls: 0,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	}

	report2 := &model.ComplianceReport{
		TenantID:       tenantID,
		ReportName:     "Q2 Report",
		Framework:      model.FrameworkSOC2,
		GeneratedAt:    now.Add(-60 * 24 * time.Hour),
		GeneratedBy:    generatedBy,
		Status:         string(model.ComplianceStatusPassed),
		OverallScore:   &score,
		TotalControls:  50,
		PassedControls: 47,
		FailedControls: 3,
		SkippedControls: 0,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	}

	_ = repo.Create(ctx, report1)
	_ = repo.Create(ctx, report2)

	t.Run("get by period", func(t *testing.T) {
		reports, err := repo.GetByPeriod(ctx, tenantID, periodStart.Add(-24*time.Hour), periodEnd.Add(24*time.Hour))

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(reports), 2)
	})

	t.Run("get by period with no results", func(t *testing.T) {
		futureStart := now.Add(30 * 24 * time.Hour)
		futureEnd := now.Add(60 * 24 * time.Hour)

		reports, err := repo.GetByPeriod(ctx, tenantID, futureStart, futureEnd)

		require.NoError(t, err)
		assert.Empty(t, reports)
	})
}

// =============================================================================
// GetFrameworkSummary Tests
// =============================================================================

func TestComplianceReportRepository_GetFrameworkSummary(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	generatedBy := uuid.New()
	now := time.Now()

	// Create reports for different frameworks
	frameworks := []string{
		model.FrameworkSOC2,
		model.FrameworkISO27001,
		model.FrameworkPCIDSS,
	}

	for _, fw := range frameworks {
		for i := 0; i < 3; i++ {
			score := float64(70 + i*5)
			status := string(model.ComplianceStatusPassed)
			if i == 0 {
				status = string(model.ComplianceStatusFailed)
			}

			report := &model.ComplianceReport{
				TenantID:       tenantID,
				ReportName:     fmt.Sprintf("%s Report %d", fw, i),
				Framework:      fw,
				GeneratedAt:    now.Add(time.Duration(-i) * time.Hour),
				GeneratedBy:    generatedBy,
				Status:         status,
				OverallScore:   &score,
				TotalControls:  50,
				PassedControls: 40 + i,
				FailedControls: 10 - i,
				SkippedControls: 0,
			}
			_ = repo.Create(ctx, report)
		}
	}

	t.Run("get framework summary", func(t *testing.T) {
		summary, err := repo.GetFrameworkSummary(ctx, tenantID)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(summary), 3)

		// Check SOC2 summary
		if soc2Summary, ok := summary[model.FrameworkSOC2]; ok {
			assert.Equal(t, 3, soc2Summary.TotalReports)
			assert.Greater(t, soc2Summary.AvgScore, 0.0)
			assert.Greater(t, soc2Summary.PassedCount, 0)
			assert.Greater(t, soc2Summary.FailedCount, 0)
			assert.False(t, soc2Summary.LastGenerated.IsZero())
		}
	})

	t.Run("get summary for tenant with no reports", func(t *testing.T) {
		emptyTenantID := uuid.New()
		summary, err := repo.GetFrameworkSummary(ctx, emptyTenantID)

		require.NoError(t, err)
		assert.Empty(t, summary)
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestComplianceReportRepository_EdgeCases(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())

	t.Run("create with maximum values", func(t *testing.T) {
		maxScore := 100.0
		maxInt := 2147483647

		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     "Max Values Report",
			Framework:      model.FrameworkSOC2,
			GeneratedAt:    time.Now(),
			GeneratedBy:    uuid.New(),
			Status:         string(model.ComplianceStatusPassed),
			OverallScore:   &maxScore,
			TotalControls:  maxInt,
			PassedControls: maxInt,
			FailedControls: 0,
			SkippedControls: 0,
		}

		err := repo.Create(ctx, report)
		require.NoError(t, err)
	})

	t.Run("create with large metadata", func(t *testing.T) {
		largeMetadata := make([]byte, 1024*100) // 100KB
		score := 80.0

		report := &model.ComplianceReport{
			TenantID:       tenantID,
			ReportName:     "Large Metadata Report",
			Framework:      model.FrameworkSOC2,
			GeneratedAt:    time.Now(),
			GeneratedBy:    uuid.New(),
			Status:         string(model.ComplianceStatusPassed),
			OverallScore:   &score,
			TotalControls:  10,
			PassedControls: 8,
			FailedControls: 2,
			SkippedControls: 0,
			Metadata:       largeMetadata,
		}

		err := repo.Create(ctx, report)
		// May fail due to size limits
		if err != nil {
			t.Logf("Large metadata rejected (expected): %v", err)
		}
	})

	t.Run("handle nil tenant ID in list", func(t *testing.T) {
		filter := model.ComplianceReportFilter{}
		_, _, err := repo.List(ctx, uuid.Nil, filter, 10, 0)

		// Should return empty results, not error
		require.NoError(t, err)
	})
}

// =============================================================================
// Concurrent Operations Tests
// =============================================================================

func TestComplianceReportRepository_ConcurrentOperations(t *testing.T) {
	db, repo := setupComplianceReportTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())

	t.Run("concurrent creates", func(t *testing.T) {
		errChan := make(chan error, 10)
		generatedBy := uuid.New()

		for i := 0; i < 10; i++ {
			go func(index int) {
				score := float64(70 + index)
				report := &model.ComplianceReport{
					TenantID:       tenantID,
					ReportName:     fmt.Sprintf("Concurrent Report %d", index),
					Framework:      model.FrameworkSOC2,
					GeneratedAt:    time.Now(),
					GeneratedBy:    generatedBy,
					Status:         string(model.ComplianceStatusPending),
					OverallScore:   &score,
					TotalControls:  50,
					PassedControls: 0,
					FailedControls: 0,
					SkippedControls: 50,
				}
				errChan <- repo.Create(ctx, report)
			}(i)
		}

		// Collect results
		for i := 0; i < 10; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
	})
}
