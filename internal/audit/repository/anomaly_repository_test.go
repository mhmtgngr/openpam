// Package repository provides tests for anomaly repository
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

// setupAnomalyTest creates a test database with anomaly detections table
func setupAnomalyTest(t *testing.T) (*sqlx.DB, *AnomalyRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)

	// Create anomaly_detections table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS anomaly_detections (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			anomaly_type TEXT NOT NULL,
			user_id UUID,
			session_id UUID,
			target_host TEXT,
			severity TEXT NOT NULL,
			confidence_score FLOAT NOT NULL,
			risk_score FLOAT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			indicators BYTEA,
			detection_method TEXT,
			detected_at TIMESTAMPTZ NOT NULL,
			model_version TEXT,
			status TEXT NOT NULL,
			assigned_to UUID,
			resolution_notes TEXT,
			resolved_at TIMESTAMPTZ,
			resolved_by UUID,
			auto_triggered BOOLEAN DEFAULT false,
			auto_action_taken TEXT,
			metadata BYTEA,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_tenant ON anomaly_detections(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_user ON anomaly_detections(user_id);
		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_session ON anomaly_detections(session_id);
		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_type ON anomaly_detections(anomaly_type);
		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_severity ON anomaly_detections(severity);
		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_status ON anomaly_detections(status);
		CREATE INDEX IF NOT EXISTS idx_anomaly_detections_detected ON anomaly_detections(detected_at);
	`)
	require.NoError(t, err)

	logger := opamtesting.Logger(t)
	repo := NewAnomalyRepository(db, logger)

	return db, repo
}

func TestNewAnomalyRepository(t *testing.T) {
	db, _ := setupAnomalyTest(t)
	if db == nil {
		return
	}

	logger := opamtesting.Logger(t)
	repo := NewAnomalyRepository(db, logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

// =============================================================================
// Create Tests
// =============================================================================

func TestAnomalyRepository_Create(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()
	targetHost := "prod-server-01.example.com"
	now := time.Now()
	indicators := []byte(`{"type":"behavioral","deviation":5.0}`)
	metadata := []byte(`{"model":"v2.1"}`)

	t.Run("create anomaly with all fields", func(t *testing.T) {
		title := "Unusual access pattern"
		description := "User accessed system at unusual hours"

		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			UserID:          &userID,
			SessionID:       &sessionID,
			TargetHost:      &targetHost,
			Severity:        string(model.SeverityHigh),
			ConfidenceScore: 85.5,
			RiskScore:       75.0,
			Title:           title,
			Description:     &description,
			Indicators:      indicators,
			DetectionMethod: "ml_model",
			DetectedAt:      now,
			ModelVersion:    stringPtr("v2.1.0"),
			Status:          string(model.AnomalyStatusOpen),
			AutoTriggered:   true,
			AutoActionTaken: stringPtr("session_terminated"),
			Metadata:        metadata,
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, anomaly.ID)
		assert.False(t, anomaly.CreatedAt.IsZero())
		assert.False(t, anomaly.UpdatedAt.IsZero())
		assert.Equal(t, string(model.AnomalyStatusOpen), anomaly.Status)
	})

	t.Run("create anomaly with minimal fields", func(t *testing.T) {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeTemporal),
			Severity:        string(model.SeverityMedium),
			ConfidenceScore: 60.0,
			RiskScore:       50.0,
			Title:           "Minimal anomaly",
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
		assert.NotNil(t, anomaly.DetectedAt)
	})

	t.Run("create ransomware anomaly", func(t *testing.T) {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeRansomware),
			UserID:          &userID,
			SessionID:       &sessionID,
			TargetHost:      &targetHost,
			Severity:        string(model.SeverityCritical),
			ConfidenceScore: 95.0,
			RiskScore:       100.0,
			Title:           "Ransomware activity detected",
			DetectionMethod: "signature_based",
			AutoTriggered:   true,
			AutoActionTaken: stringPtr("emergency_triggered"),
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	})
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestAnomalyRepository_GetByID(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()
	targetHost := "test-server.example.com"
	description := "Test anomaly"

	created := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeSpatial),
		UserID:          &userID,
		SessionID:       &sessionID,
		TargetHost:      &targetHost,
		Severity:        string(model.SeverityHigh),
		ConfidenceScore: 80.0,
		RiskScore:       70.0,
		Title:           "Impossible travel detected",
		Description:     &description,
		DetectionMethod: "geo_analysis",
		Status:          string(model.AnomalyStatusOpen),
	}

	err := repo.Create(ctx, created)
	require.NoError(t, err)

	t.Run("get existing anomaly", func(t *testing.T) {
		found, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)

		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, tenantID, found.TenantID)
		assert.Equal(t, string(model.AnomalyTypeSpatial), found.AnomalyType)
		assert.Equal(t, userID, *found.UserID)
		assert.Equal(t, sessionID, *found.SessionID)
		assert.Equal(t, targetHost, *found.TargetHost)
		assert.Equal(t, string(model.SeverityHigh), found.Severity)
		assert.Equal(t, 80.0, found.ConfidenceScore)
		assert.Equal(t, 70.0, found.RiskScore)
		assert.Equal(t, "Impossible travel detected", found.Title)
		assert.Equal(t, description, *found.Description)
	})

	t.Run("get non-existent anomaly", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestAnomalyRepository_List(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	// Create test anomalies with different statuses
	anomalyTypes := []string{
		string(model.AnomalyTypeBehavioral),
		string(model.AnomalyTypeTemporal),
		string(model.AnomalyTypeSpatial),
		string(model.AnomalyTypePattern),
	}
	severities := []string{
		string(model.SeverityCritical),
		string(model.SeverityHigh),
		string(model.SeverityMedium),
		string(model.SeverityLow),
	}
	statuses := []string{
		string(model.AnomalyStatusOpen),
		string(model.AnomalyStatusInvestigating),
		string(model.AnomalyStatusResolved),
	}

	for i := 0; i < 10; i++ {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     anomalyTypes[i%len(anomalyTypes)],
			UserID:          &userID,
			SessionID:       &sessionID,
			Severity:        severities[i%len(severities)],
			ConfidenceScore: float64(60 + i*3),
			RiskScore:       float64(50 + i*4),
			Title:           fmt.Sprintf("Anomaly %d", i),
			DetectedAt:      now.Add(time.Duration(-i) * time.Hour),
			Status:          statuses[i%len(statuses)],
		}
		_ = repo.Create(ctx, anomaly)
	}

	t.Run("list all anomalies", func(t *testing.T) {
		filter := model.AnomalyFilter{}
		anomalies, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(anomalies), 10)
		assert.GreaterOrEqual(t, total, 10)
	})

	t.Run("list with type filter", func(t *testing.T) {
		anomalyType := string(model.AnomalyTypeBehavioral)
		filter := model.AnomalyFilter{
			AnomalyType: &anomalyType,
		}

		anomalies, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(anomalies), 0)

		for _, a := range anomalies {
			assert.Equal(t, string(model.AnomalyTypeBehavioral), a.AnomalyType)
		}
	})

	t.Run("list with severity filter", func(t *testing.T) {
		severity := string(model.SeverityCritical)
		filter := model.AnomalyFilter{
			Severity: &severity,
		}

		anomalies, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)

		for _, a := range anomalies {
			assert.Equal(t, string(model.SeverityCritical), a.Severity)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		status := string(model.AnomalyStatusOpen)
		filter := model.AnomalyFilter{
			Status: &status,
		}

		anomalies, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)

		for _, a := range anomalies {
			assert.Equal(t, "open", a.Status)
		}
	})

	t.Run("list with user ID filter", func(t *testing.T) {
		filter := model.AnomalyFilter{
			UserID: &userID,
		}

		anomalies, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)

		for _, a := range anomalies {
			assert.Equal(t, userID, *a.UserID)
		}
	})

	t.Run("list with date range filter", func(t *testing.T) {
		dateFrom := now.Add(-5 * time.Hour)
		dateTo := now.Add(1 * time.Hour)

		filter := model.AnomalyFilter{
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
		}

		anomalies, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(anomalies), 0)
	})

	t.Run("list with pagination", func(t *testing.T) {
		filter := model.AnomalyFilter{}

		anomalies, total, err := repo.List(ctx, tenantID, filter, 3, 0)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(anomalies), 3)
		assert.GreaterOrEqual(t, total, 10)
	})

	t.Run("list with offset", func(t *testing.T) {
		filter := model.AnomalyFilter{}

		page1, _, _ := repo.List(ctx, tenantID, filter, 3, 0)
		page2, _, _ := repo.List(ctx, tenantID, filter, 3, 3)

		if len(page1) > 0 && len(page2) > 0 {
			assert.NotEqual(t, page1[0].ID, page2[0].ID)
		}
	})
}

// =============================================================================
// Update Tests
// =============================================================================

func TestAnomalyRepository_Update(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	assignedTo := uuid.New()
	resolutionNotes := "Investigated and confirmed as legitimate"

	anomaly := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeBehavioral),
		UserID:          &userID,
		Severity:        string(model.SeverityHigh),
		ConfidenceScore: 80.0,
		RiskScore:       70.0,
		Title:           "Test anomaly",
		Status:          string(model.AnomalyStatusOpen),
	}

	err := repo.Create(ctx, anomaly)
	require.NoError(t, err)

	t.Run("update anomaly", func(t *testing.T) {
		anomaly.Status = string(model.AnomalyStatusResolved)
		anomaly.AssignedTo = &assignedTo
		anomaly.ResolutionNotes = &resolutionNotes

		err := repo.Update(ctx, anomaly)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)

		assert.Equal(t, string(model.AnomalyStatusResolved), updated.Status)
		assert.Equal(t, assignedTo, *updated.AssignedTo)
		assert.Equal(t, resolutionNotes, *updated.ResolutionNotes)
	})
}

// =============================================================================
// UpdateStatus Tests
// =============================================================================

func TestAnomalyRepository_UpdateStatus(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	assignedTo := uuid.New()
	resolvedBy := uuid.New()
	resolutionNotes := "False positive - normal behavior"

	anomaly := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeBehavioral),
		UserID:          &userID,
		Severity:        string(model.SeverityMedium),
		ConfidenceScore: 65.0,
		RiskScore:       55.0,
		Title:           "Status test",
		Status:          string(model.AnomalyStatusOpen),
	}

	err := repo.Create(ctx, anomaly)
	require.NoError(t, err)

	t.Run("update to investigating", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, anomaly.ID, string(model.AnomalyStatusInvestigating), &assignedTo, nil, nil)

		require.NoError(t, err)

		updated, _ := repo.GetByID(ctx, anomaly.ID)
		assert.Equal(t, string(model.AnomalyStatusInvestigating), updated.Status)
		assert.Equal(t, assignedTo, *updated.AssignedTo)
	})

	t.Run("update to resolved", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, anomaly.ID, string(model.AnomalyStatusResolved), &assignedTo, &resolutionNotes, &resolvedBy)

		require.NoError(t, err)

		updated, _ := repo.GetByID(ctx, anomaly.ID)
		assert.Equal(t, string(model.AnomalyStatusResolved), updated.Status)
		assert.Equal(t, resolutionNotes, *updated.ResolutionNotes)
		assert.Equal(t, resolvedBy, *updated.ResolvedBy)
		assert.False(t, updated.ResolvedAt.IsZero())
	})

	t.Run("update to false positive", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, anomaly.ID, string(model.AnomalyStatusFalsePositive), nil, &resolutionNotes, nil)

		require.NoError(t, err)

		updated, _ := repo.GetByID(ctx, anomaly.ID)
		assert.Equal(t, string(model.AnomalyStatusFalsePositive), updated.Status)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestAnomalyRepository_Delete(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()

	anomaly := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeBehavioral),
		UserID:          &userID,
		Severity:        string(model.SeverityLow),
		ConfidenceScore: 50.0,
		RiskScore:       40.0,
		Title:           "To delete",
		Status:          string(model.AnomalyStatusOpen),
	}

	err := repo.Create(ctx, anomaly)
	require.NoError(t, err)

	t.Run("delete existing anomaly", func(t *testing.T) {
		err := repo.Delete(ctx, anomaly.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, anomaly.ID)
		assert.Error(t, err)
	})

	t.Run("delete non-existent anomaly", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows affected")
	})
}

// =============================================================================
// GetByUserID Tests
// =============================================================================

func TestAnomalyRepository_GetByUserID(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	// Create anomalies for the user
	for i := 0; i < 5; i++ {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			UserID:          &userID,
			SessionID:       &sessionID,
			Severity:        string(model.SeverityMedium),
			ConfidenceScore: 60.0,
			RiskScore:       50.0,
			Title:           fmt.Sprintf("User anomaly %d", i),
			DetectedAt:      now.Add(time.Duration(-i) * time.Hour),
			Status:          string(model.AnomalyStatusOpen),
		}
		_ = repo.Create(ctx, anomaly)
	}

	t.Run("get by user ID", func(t *testing.T) {
		anomalies, err := repo.GetByUserID(ctx, tenantID, userID, 0)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(anomalies), 5)

		for _, a := range anomalies {
			assert.Equal(t, userID, *a.UserID)
		}
	})

	t.Run("get by user ID with limit", func(t *testing.T) {
		anomalies, err := repo.GetByUserID(ctx, tenantID, userID, 3)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(anomalies), 3)
	})

	t.Run("get by non-existent user ID", func(t *testing.T) {
		anomalies, err := repo.GetByUserID(ctx, tenantID, uuid.New(), 0)

		require.NoError(t, err)
		assert.Empty(t, anomalies)
	})
}

// =============================================================================
// GetBySessionID Tests
// =============================================================================

func TestAnomalyRepository_GetBySessionID(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()

	// Create anomalies for the session
	for i := 0; i < 3; i++ {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			UserID:          &userID,
			SessionID:       &sessionID,
			Severity:        string(model.SeverityHigh),
			ConfidenceScore: 75.0,
			RiskScore:       65.0,
			Title:           fmt.Sprintf("Session anomaly %d", i),
			Status:          string(model.AnomalyStatusOpen),
		}
		_ = repo.Create(ctx, anomaly)
	}

	t.Run("get by session ID", func(t *testing.T) {
		anomalies, err := repo.GetBySessionID(ctx, sessionID)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(anomalies), 3)

		for _, a := range anomalies {
			assert.Equal(t, sessionID, *a.SessionID)
		}
	})

	t.Run("get by non-existent session ID", func(t *testing.T) {
		anomalies, err := repo.GetBySessionID(ctx, uuid.New())

		require.NoError(t, err)
		assert.Empty(t, anomalies)
	})
}

// =============================================================================
// GetOpen Tests
// =============================================================================

func TestAnomalyRepository_GetOpen(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()

	// Create open anomalies with different severities
	severities := []string{
		string(model.SeverityCritical),
		string(model.SeverityHigh),
		string(model.SeverityMedium),
		string(model.SeverityLow),
	}

	for _, severity := range severities {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			UserID:          &userID,
			SessionID:       &sessionID,
			Severity:        severity,
			ConfidenceScore: 70.0,
			RiskScore:       60.0,
			Title:           fmt.Sprintf("Open %s", severity),
			Status:          string(model.AnomalyStatusOpen),
		}
		_ = repo.Create(ctx, anomaly)
	}

	// Create a resolved anomaly
	resolved := &model.AnomalyDetection{
		TenantID:        tenantID,
		AnomalyType:     string(model.AnomalyTypeBehavioral),
		UserID:          &userID,
		Severity:        string(model.SeverityMedium),
		ConfidenceScore: 60.0,
		RiskScore:       50.0,
		Title:           "Resolved anomaly",
		Status:          string(model.AnomalyStatusResolved),
	}
	_ = repo.Create(ctx, resolved)

	t.Run("get all open anomalies", func(t *testing.T) {
		anomalies, err := repo.GetOpen(ctx, tenantID, nil)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(anomalies), 4)

		for _, a := range anomalies {
			assert.Equal(t, string(model.AnomalyStatusOpen), a.Status)
		}
	})

	t.Run("get open anomalies by severity", func(t *testing.T) {
		severity := string(model.SeverityCritical)
		anomalies, err := repo.GetOpen(ctx, tenantID, &severity)

		require.NoError(t, err)
		assert.Greater(t, len(anomalies), 0)

		for _, a := range anomalies {
			assert.Equal(t, string(model.SeverityCritical), a.Severity)
		}
	})
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestAnomalyRepository_GetStats(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	// Create anomalies with different statuses and severities
	statuses := []string{
		string(model.AnomalyStatusOpen),
		string(model.AnomalyStatusOpen),
		string(model.AnomalyStatusInvestigating),
		string(model.AnomalyStatusInvestigating),
		string(model.AnomalyStatusResolved),
	}
	severities := []string{
		string(model.SeverityCritical),
		string(model.SeverityHigh),
		string(model.SeverityMedium),
		string(model.SeverityMedium),
		string(model.SeverityLow),
	}

	for i := 0; i < 5; i++ {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			UserID:          &userID,
			SessionID:       &sessionID,
			Severity:        severities[i],
			ConfidenceScore: 70.0,
			RiskScore:       60.0,
			Title:           fmt.Sprintf("Stats anomaly %d", i),
			DetectedAt:      now,
			Status:          statuses[i],
		}
		_ = repo.Create(ctx, anomaly)
	}

	t.Run("get anomaly stats", func(t *testing.T) {
		stats, err := repo.GetStats(ctx, tenantID)

		require.NoError(t, err)
		assert.NotNil(t, stats)

		assert.GreaterOrEqual(t, stats.Total, 5)
		assert.GreaterOrEqual(t, stats.OpenCount, 2)
		assert.GreaterOrEqual(t, stats.InvestigatingCount, 2)
		assert.GreaterOrEqual(t, stats.ResolvedCount, 1)
		assert.GreaterOrEqual(t, stats.CriticalCount, 1)
		assert.GreaterOrEqual(t, stats.HighCount, 1)
		assert.GreaterOrEqual(t, stats.TodayCount, 5)
		assert.GreaterOrEqual(t, stats.WeekCount, 5)
	})
}

// =============================================================================
// BatchCreate Tests
// =============================================================================

func TestAnomalyRepository_BatchCreate(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()

	t.Run("batch create multiple anomalies", func(t *testing.T) {
		anomalies := make([]model.AnomalyDetection, 10)
		for i := 0; i < 10; i++ {
			anomalies[i] = model.AnomalyDetection{
				TenantID:        tenantID,
				AnomalyType:     string(model.AnomalyTypeBehavioral),
				UserID:          &userID,
				Severity:        string(model.SeverityMedium),
				ConfidenceScore: 60.0,
				RiskScore:       50.0,
				Title:           fmt.Sprintf("Batch anomaly %d", i),
			}
		}

		err := repo.BatchCreate(ctx, anomalies)
		require.NoError(t, err)

		// Verify all were created
		for _, a := range anomalies {
			found, err := repo.GetByID(ctx, a.ID)
			require.NoError(t, err)
			assert.Equal(t, a.Title, found.Title)
		}
	})

	t.Run("batch create empty slice", func(t *testing.T) {
		err := repo.BatchCreate(ctx, []model.AnomalyDetection{})
		require.NoError(t, err)
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestAnomalyRepository_EdgeCases(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())

	t.Run("create with maximum scores", func(t *testing.T) {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			Severity:        string(model.SeverityCritical),
			ConfidenceScore: 100.0,
			RiskScore:       100.0,
			Title:           "Max scores",
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	})

	t.Run("create with minimum scores", func(t *testing.T) {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			Severity:        string(model.SeverityLow),
			ConfidenceScore: 0.0,
			RiskScore:       0.0,
			Title:           "Min scores",
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	})

	t.Run("create with nil optional fields", func(t *testing.T) {
		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			Severity:        string(model.SeverityMedium),
			ConfidenceScore: 50.0,
			RiskScore:       40.0,
			Title:           "Nil fields",
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		found, _ := repo.GetByID(ctx, anomaly.ID)
		assert.Nil(t, found.UserID)
		assert.Nil(t, found.SessionID)
		assert.Nil(t, found.TargetHost)
		assert.Nil(t, found.Description)
	})

	t.Run("create with very long title", func(t *testing.T) {
		longTitle := string(make([]byte, 1000))
		for i := range longTitle {
			longTitle = longTitle[:i] + "A" + longTitle[i+1:]
		}

		anomaly := &model.AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     string(model.AnomalyTypeBehavioral),
			Severity:        string(model.SeverityLow),
			ConfidenceScore: 30.0,
			RiskScore:       20.0,
			Title:           longTitle,
		}

		err := repo.Create(ctx, anomaly)
		// Should succeed or fail gracefully
		if err != nil {
			t.Logf("Long title handled: %v", err)
		}
	})
}

// =============================================================================
// Concurrent Operations Tests
// =============================================================================

func TestAnomalyRepository_ConcurrentOperations(t *testing.T) {
	db, repo := setupAnomalyTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()

	t.Run("concurrent creates", func(t *testing.T) {
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				anomaly := &model.AnomalyDetection{
					TenantID:        tenantID,
					AnomalyType:     string(model.AnomalyTypeBehavioral),
					UserID:          &userID,
					Severity:        string(model.SeverityMedium),
					ConfidenceScore: 60.0,
					RiskScore:       50.0,
					Title:           fmt.Sprintf("Concurrent %d", index),
				}
				errChan <- repo.Create(ctx, anomaly)
			}(i)
		}

		for i := 0; i < 10; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func stringPtr(s string) *string {
	return &s
}
