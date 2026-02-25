// Package analytics provides tests for anomaly persistence
package analytics

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup
// =============================================================================

func setupAnomalyTestDB(t *testing.T) *sqlx.DB {
	// Create in-memory SQLite for testing or connect to test PostgreSQL
	dsn := "host=localhost port=5432 user=openpam_test password=openpam_test dbname=openpam_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
		return nil
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Create test schema
	schema := `
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
			detection_method TEXT NOT NULL,
			detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
			model_version TEXT,
			status TEXT NOT NULL DEFAULT 'open',
			assigned_to UUID,
			resolution_notes TEXT,
			resolved_at TIMESTAMP,
			resolved_by UUID,
			auto_triggered BOOLEAN NOT NULL DEFAULT FALSE,
			auto_action_taken TEXT,
			correlation_id UUID,
			correlation_key TEXT,
			duplicate_count INTEGER NOT NULL DEFAULT 0,
			is_duplicate BOOLEAN NOT NULL DEFAULT FALSE,
			first_detection_id UUID,
			merged_into_id UUID,
			metadata BYTEA,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_anomaly_tenant ON anomaly_detections(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_anomaly_user ON anomaly_detections(user_id);
		CREATE INDEX IF NOT EXISTS idx_anomaly_status ON anomaly_detections(status);
		CREATE INDEX IF NOT EXISTS idx_anomaly_severity ON anomaly_detections(severity);
		CREATE INDEX IF NOT EXISTS idx_anomaly_correlation ON anomaly_detections(correlation_id);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	// Clean up before test
	_, _ = db.Exec("DELETE FROM anomaly_detections")

	t.Cleanup(func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS anomaly_detections")
		_ = db.Close()
	})

	return db
}

func newTestAnomalyRepository(t *testing.T) *AnomalyRepository {
	db := setupAnomalyTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
	}
	logger := zerolog.Nop()
	return NewAnomalyRepository(db, logger)
}

// Helper to create a test anomaly
func createTestAnomaly(t *testing.T) *Anomaly {
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	indicators := map[string]interface{}{
		"failed_attempts": 5,
		"source_ip":       "192.168.1.100",
	}
	indicatorsJSON, _ := json.Marshal(indicators)

	metadata := map[string]interface{}{
		"test": true,
	}
	metadataJSON, _ := json.Marshal(metadata)

	return &Anomaly{
		TenantID:        tenantID,
		AnomalyType:     AnomalyTypeBehavioral,
		UserID:          &userID,
		SessionID:       &sessionID,
		TargetHost:      strPtr("test-host.example.com"),
		Severity:        SeverityHigh,
		ConfidenceScore: 0.85,
		RiskScore:       75.5,
		Title:           "Test Anomaly",
		Description:     strPtr("Test anomaly description"),
		Indicators:      indicatorsJSON,
		DetectionMethod: "test",
		ModelVersion:    strPtr("1.0.0"),
		Status:          AnomalyStatusOpen,
		AutoTriggered:   true,
		AutoActionTaken: strPtr("blocked"),
		Metadata:        metadataJSON,
	}
}

func strPtr(s string) *string {
	return &s
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

// =============================================================================
// Create Tests
// =============================================================================

func TestAnomalyRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()

	t.Run("creates anomaly successfully", func(t *testing.T) {
		anomaly := createTestAnomaly(t)

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		// Verify ID was set
		assert.NotEqual(t, uuid.Nil, anomaly.ID)

		// Verify timestamps were set
		assert.False(t, anomaly.CreatedAt.IsZero())
		assert.False(t, anomaly.UpdatedAt.IsZero())
		assert.False(t, anomaly.DetectedAt.IsZero())

		// Verify default status
		assert.Equal(t, AnomalyStatusOpen, anomaly.Status)

		// Verify correlation ID was set
		assert.NotNil(t, anomaly.CorrelationID)
	})

	t.Run("creates anomaly with default values", func(t *testing.T) {
		anomaly := &Anomaly{
			TenantID:        uuid.New(),
			AnomalyType:     AnomalyTypeTemporal,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.5,
			RiskScore:       50.0,
			Title:           "Default Test Anomaly",
			DetectionMethod: "test",
		}

		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		assert.Equal(t, AnomalyStatusOpen, anomaly.Status)
		assert.NotNil(t, anomaly.CorrelationID)
		assert.Equal(t, 0, anomaly.DuplicateCount)
		assert.False(t, anomaly.IsDuplicate)
	})
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestAnomalyRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()

	t.Run("retrieves existing anomaly", func(t *testing.T) {
		anomaly := createTestAnomaly(t)
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)

		assert.Equal(t, anomaly.ID, retrieved.ID)
		assert.Equal(t, anomaly.TenantID, retrieved.TenantID)
		assert.Equal(t, anomaly.AnomalyType, retrieved.AnomalyType)
		assert.Equal(t, anomaly.Severity, retrieved.Severity)
		assert.Equal(t, anomaly.Title, retrieved.Title)
		assert.Equal(t, anomaly.Description, retrieved.Description)
	})

	t.Run("returns error for non-existent anomaly", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestAnomalyRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create test anomalies
	severityHigh := SeverityHigh
	severityCritical := SeverityCritical
	statusOpen := AnomalyStatusOpen
	statusResolved := AnomalyStatusResolved

	anomalies := []Anomaly{
		{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			Severity:        severityHigh,
			ConfidenceScore: 0.7,
			RiskScore:       60,
			Title:           "High Severity Anomaly",
			DetectionMethod: "test",
			Status:          statusOpen,
		},
		{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypePattern,
			Severity:        severityCritical,
			ConfidenceScore: 0.9,
			RiskScore:       85,
			Title:           "Critical Severity Anomaly",
			DetectionMethod: "test",
			Status:          statusOpen,
		},
		{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeSpatial,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.6,
			RiskScore:       45,
			Title:           "Resolved Anomaly",
			DetectionMethod: "test",
			Status:          statusResolved,
		},
	}

	for i := range anomalies {
		err := repo.Create(ctx, &anomalies[i])
		require.NoError(t, err)
	}

	t.Run("lists all anomalies for tenant", func(t *testing.T) {
		filter := AnomalyFilter{}
		results, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, results, 3)
	})

	t.Run("filters by severity", func(t *testing.T) {
		filter := AnomalyFilter{Severity: &severityHigh}
		results, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, results, 1)
		assert.Equal(t, SeverityHigh, results[0].Severity)
	})

	t.Run("filters by status", func(t *testing.T) {
		filter := AnomalyFilter{Status: &statusOpen}
		results, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, results, 2)
	})

	t.Run("filters by anomaly type", func(t *testing.T) {
		anomalyType := AnomalyTypeBehavioral
		filter := AnomalyFilter{AnomalyType: &anomalyType}
		results, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, AnomalyTypeBehavioral, results[0].AnomalyType)
	})

	t.Run("filters by search", func(t *testing.T) {
		filter := AnomalyFilter{Search: "Critical"}
		results, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Contains(t, results[0].Title, "Critical")
	})

	t.Run("applies pagination", func(t *testing.T) {
		filter := AnomalyFilter{}
		results, total, err := repo.List(ctx, tenantID, filter, 2, 0)

		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, results, 2)

		// Second page
		results2, _, err := repo.List(ctx, tenantID, filter, 2, 2)
		require.NoError(t, err)
		assert.Len(t, results2, 1)
	})

	t.Run("excludes duplicates", func(t *testing.T) {
		// Create a duplicate
		duplicateAnomaly := &Anomaly{
			TenantID:      tenantID,
			AnomalyType:   AnomalyTypeBehavioral,
			Severity:      SeverityLow,
			RiskScore:     30,
			ConfidenceScore: 0.4,
			Title:         "Duplicate Anomaly",
			DetectionMethod: "test",
			Status:        AnomalyStatusOpen,
			IsDuplicate:   true,
		}
		err := repo.Create(ctx, duplicateAnomaly)
		require.NoError(t, err)

		filter := AnomalyFilter{}
		results, total, err := repo.List(ctx, tenantID, filter, 100, 0)

		require.NoError(t, err)
		// Should not include duplicate
		assert.Equal(t, 3, total) // Still 3 non-duplicates
		for _, r := range results {
			assert.False(t, r.IsDuplicate)
		}
	})
}

// =============================================================================
// Update Tests
// =============================================================================

func TestAnomalyRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()

	t.Run("updates anomaly status", func(t *testing.T) {
		anomaly := createTestAnomaly(t)
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		// Update status
		anomaly.Status = AnomalyStatusInvestigating
		notes := "Under investigation"
		anomaly.ResolutionNotes = &notes

		err = repo.Update(ctx, anomaly)
		require.NoError(t, err)

		// Verify update
		retrieved, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)
		assert.Equal(t, AnomalyStatusInvestigating, retrieved.Status)
		assert.Equal(t, &notes, retrieved.ResolutionNotes)
	})

	t.Run("updates assignment", func(t *testing.T) {
		anomaly := createTestAnomaly(t)
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		assignedTo := uuid.New()
		anomaly.AssignedTo = &assignedTo

		err = repo.Update(ctx, anomaly)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)
		assert.Equal(t, &assignedTo, retrieved.AssignedTo)
	})
}

// =============================================================================
// UpdateStatus Tests
// =============================================================================

func TestAnomalyRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()

	t.Run("updates status to resolved", func(t *testing.T) {
		anomaly := createTestAnomaly(t)
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		resolvedBy := uuid.New()
		notes := "Issue resolved"

		err = repo.UpdateStatus(ctx, anomaly.ID, AnomalyStatusResolved, nil, &resolvedBy, &notes)
		require.NoError(t, err)

		// Verify update
		retrieved, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)
		assert.Equal(t, AnomalyStatusResolved, retrieved.Status)
		assert.Equal(t, &resolvedBy, retrieved.ResolvedBy)
		assert.Equal(t, &notes, retrieved.ResolutionNotes)
		assert.NotNil(t, retrieved.ResolvedAt)
	})

	t.Run("updates to investigating", func(t *testing.T) {
		anomaly := createTestAnomaly(t)
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		assignedTo := uuid.New()

		err = repo.UpdateStatus(ctx, anomaly.ID, AnomalyStatusInvestigating, &assignedTo, nil, nil)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)
		assert.Equal(t, AnomalyStatusInvestigating, retrieved.Status)
		assert.Equal(t, &assignedTo, retrieved.AssignedTo)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestAnomalyRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()

	t.Run("soft deletes anomaly", func(t *testing.T) {
		anomaly := createTestAnomaly(t)
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)

		err = repo.Delete(ctx, anomaly.ID)
		require.NoError(t, err)

		// Verify status changed to ignored
		retrieved, err := repo.GetByID(ctx, anomaly.ID)
		require.NoError(t, err)
		assert.Equal(t, AnomalyStatusIgnored, retrieved.Status)
	})

	t.Run("returns error for non-existent anomaly", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// GetByUserID Tests
// =============================================================================

func TestAnomalyRepository_GetByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Create anomalies for user
	for i := 0; i < 3; i++ {
		anomaly := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			UserID:          &userID,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.5,
			RiskScore:       50,
			Title:           "User Anomaly",
			DetectionMethod: "test",
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	}

	// Create anomaly for different user
	otherUserID := uuid.New()
	otherAnomaly := &Anomaly{
		TenantID:        tenantID,
		AnomalyType:     AnomalyTypeBehavioral,
		UserID:          &otherUserID,
		Severity:        SeverityMedium,
		ConfidenceScore: 0.5,
		RiskScore:       50,
		Title:           "Other User Anomaly",
		DetectionMethod: "test",
	}
	err := repo.Create(ctx, otherAnomaly)
	require.NoError(t, err)

	t.Run("retrieves anomalies for user", func(t *testing.T) {
		results, err := repo.GetByUserID(ctx, tenantID, userID, 10)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("applies limit", func(t *testing.T) {
		results, err := repo.GetByUserID(ctx, tenantID, userID, 2)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty for user with no anomalies", func(t *testing.T) {
		emptyUserID := uuid.New()
		results, err := repo.GetByUserID(ctx, tenantID, emptyUserID, 10)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

// =============================================================================
// GetBySessionID Tests
// =============================================================================

func TestAnomalyRepository_GetBySessionID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	sessionID := uuid.New()

	// Create anomalies for session
	for i := 0; i < 2; i++ {
		anomaly := &Anomaly{
			TenantID:        uuid.New(),
			AnomalyType:     AnomalyTypeBehavioral,
			SessionID:       &sessionID,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.5,
			RiskScore:       50,
			Title:           "Session Anomaly",
			DetectionMethod: "test",
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	}

	t.Run("retrieves anomalies for session", func(t *testing.T) {
		results, err := repo.GetBySessionID(ctx, sessionID)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

// =============================================================================
// GetOpen Tests
// =============================================================================

func TestAnomalyRepository_GetOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create open anomalies
	for i := 0; i < 3; i++ {
		anomaly := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			Severity:        SeverityHigh,
			ConfidenceScore: 0.7,
			RiskScore:       70,
			Title:           "Open Anomaly",
			DetectionMethod: "test",
			Status:          AnomalyStatusOpen,
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	}

	// Create resolved anomaly
	resolved := &Anomaly{
		TenantID:        tenantID,
		AnomalyType:     AnomalyTypeBehavioral,
		Severity:        SeverityHigh,
		ConfidenceScore: 0.7,
		RiskScore:       70,
		Title:           "Resolved Anomaly",
		DetectionMethod: "test",
		Status:          AnomalyStatusResolved,
	}
	err := repo.Create(ctx, resolved)
	require.NoError(t, err)

	t.Run("retrieves all open anomalies", func(t *testing.T) {
		results, err := repo.GetOpen(ctx, tenantID, nil)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("filters open anomalies by severity", func(t *testing.T) {
		highSeverity := SeverityHigh
		results, err := repo.GetOpen(ctx, tenantID, &highSeverity)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestAnomalyRepository_GetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create anomalies with different statuses and severities
	anomalies := []struct {
		severity Severity
		status   AnomalyStatus
	}{
		{SeverityCritical, AnomalyStatusOpen},
		{SeverityHigh, AnomalyStatusOpen},
		{SeverityHigh, AnomalyStatusInvestigating},
		{SeverityMedium, AnomalyStatusResolved},
		{SeverityLow, AnomalyStatusResolved},
		{SeverityCritical, AnomalyStatusResolved},
	}

	for i, a := range anomalies {
		anomaly := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			Severity:        a.severity,
			ConfidenceScore: 0.5,
			RiskScore:       50,
			Title:           "Stats Test Anomaly",
			DetectionMethod: "test",
			Status:          a.status,
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
		_ = i // Use index
	}

	t.Run("returns accurate statistics", func(t *testing.T) {
		stats, err := repo.GetStats(ctx, tenantID)
		require.NoError(t, err)

		assert.Equal(t, 6, stats.Total)
		assert.Equal(t, 2, stats.OpenCount)
		assert.Equal(t, 1, stats.InvestigatingCount)
		assert.Equal(t, 3, stats.ResolvedCount)
		assert.Equal(t, 2, stats.CriticalCount)
		assert.Equal(t, 2, stats.HighCount)
		assert.Equal(t, 1, stats.MediumCount)
		assert.Equal(t, 1, stats.LowCount)
	})
}

// =============================================================================
// BatchCreate Tests
// =============================================================================

func TestAnomalyRepository_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("creates multiple anomalies", func(t *testing.T) {
		anomalies := make([]Anomaly, 5)
		for i := range anomalies {
			anomalies[i] = Anomaly{
				TenantID:        tenantID,
				AnomalyType:     AnomalyTypeBehavioral,
				Severity:        SeverityMedium,
				ConfidenceScore: 0.5,
				RiskScore:       50,
				Title:           "Batch Anomaly",
				DetectionMethod: "test",
			}
		}

		err := repo.BatchCreate(ctx, anomalies)
		require.NoError(t, err)

		// Verify all were created
		for _, a := range anomalies {
			assert.NotEqual(t, uuid.Nil, a.ID)
			assert.False(t, a.CreatedAt.IsZero())
		}
	})

	t.Run("handles empty batch", func(t *testing.T) {
		err := repo.BatchCreate(ctx, []Anomaly{})
		require.NoError(t, err)
	})
}

// =============================================================================
// GetByCorrelationID Tests
// =============================================================================

func TestAnomalyRepository_GetByCorrelationID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	correlationID := uuid.New()

	// Create original anomaly
	original := &Anomaly{
		TenantID:        tenantID,
		AnomalyType:     AnomalyTypeBehavioral,
		Severity:        SeverityHigh,
		ConfidenceScore: 0.7,
		RiskScore:       70,
		Title:           "Original Anomaly",
		DetectionMethod: "test",
		Status:          AnomalyStatusOpen,
		CorrelationID:   &correlationID,
		CorrelationKey:  strPtr("test-key"),
	}
	err := repo.Create(ctx, original)
	require.NoError(t, err)

	// Create duplicates
	for i := 0; i < 2; i++ {
		duplicate := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			Severity:        SeverityHigh,
			ConfidenceScore: 0.7,
			RiskScore:       70,
			Title:           "Duplicate Anomaly",
			DetectionMethod: "test",
			Status:          AnomalyStatusOpen,
			CorrelationID:   &correlationID,
			CorrelationKey:  strPtr("test-key"),
			IsDuplicate:     true,
			FirstDetectionID: &original.ID,
		}
		err = repo.Create(ctx, duplicate)
		require.NoError(t, err)
		_ = i // Use index
	}

	t.Run("retrieves all anomalies in correlation group", func(t *testing.T) {
		results, err := repo.GetByCorrelationID(ctx, correlationID)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})
}

// =============================================================================
// GetAnomalyTypes Tests
// =============================================================================

func TestAnomalyRepository_GetAnomalyTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create anomalies of different types
	anomalyTypes := []AnomalyType{
		AnomalyTypeBehavioral,
		AnomalyTypeTemporal,
		AnomalyTypeSpatial,
	}

	for _, at := range anomalyTypes {
		anomaly := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     at,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.5,
			RiskScore:       50,
			Title:           "Type Test Anomaly",
			DetectionMethod: "test",
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	}

	t.Run("returns unique anomaly types", func(t *testing.T) {
		types, err := repo.GetAnomalyTypes(ctx, tenantID)
		require.NoError(t, err)
		assert.Len(t, types, 3)
		assert.Contains(t, types, string(AnomalyTypeBehavioral))
		assert.Contains(t, types, string(AnomalyTypeTemporal))
		assert.Contains(t, types, string(AnomalyTypeSpatial))
	})
}

// =============================================================================
// GetTopUsersByAnomalyCount Tests
// =============================================================================

func TestAnomalyRepository_GetTopUsersByAnomalyCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	user1ID := uuid.New()
	user2ID := uuid.New()

	// Create anomalies for user1
	for i := 0; i < 5; i++ {
		anomaly := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			UserID:          &user1ID,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.5,
			RiskScore:       float64(50 + i*10),
			Title:           "User1 Anomaly",
			DetectionMethod: "test",
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
	}

	// Create anomalies for user2
	for i := 0; i < 2; i++ {
		anomaly := &Anomaly{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeBehavioral,
			UserID:          &user2ID,
			Severity:        SeverityMedium,
			ConfidenceScore: 0.5,
			RiskScore:       60,
			Title:           "User2 Anomaly",
			DetectionMethod: "test",
		}
		err := repo.Create(ctx, anomaly)
		require.NoError(t, err)
		_ = i // Use index
	}

	t.Run("returns top users by anomaly count", func(t *testing.T) {
		results, err := repo.GetTopUsersByAnomalyCount(ctx, tenantID, 10, nil, nil)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// User1 should be first
		assert.Equal(t, user1ID, results[0].UserID)
		assert.Equal(t, 5, results[0].AnomalyCount)
		assert.Equal(t, 90.0, results[0].MaxRiskScore)

		// User2 should be second
		assert.Equal(t, user2ID, results[1].UserID)
		assert.Equal(t, 2, results[1].AnomalyCount)
	})

	t.Run("applies limit", func(t *testing.T) {
		results, err := repo.GetTopUsersByAnomalyCount(ctx, tenantID, 1, nil, nil)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

// =============================================================================
// CreateFromDetectionRequest Tests
// =============================================================================

func TestAnomalyRepository_CreateFromDetectionRequest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()
	detectedBy := uuid.New()

	t.Run("creates anomaly from detection request", func(t *testing.T) {
		indicators := map[string]interface{}{
			"test_indicator": "value",
		}
		metadata := map[string]interface{}{
			"source": "test",
		}

		req := &CreateAnomalyRequest{
			TenantID:        tenantID,
			AnomalyType:     AnomalyTypeRansomware,
			Title:           "Detection Test Anomaly",
			Description:     strPtr("Created from detection request"),
			Indicators:      indicators,
			Severity:        SeverityCritical,
			ConfidenceScore: 0.95,
			RiskScore:       90,
			DetectionMethod: "ml_model",
			ModelVersion:    strPtr("2.0.0"),
			AutoTriggered:   true,
			AutoActionTaken: strPtr("quarantined"),
			Metadata:        metadata,
		}

		anomaly, err := repo.CreateFromDetectionRequest(ctx, req, detectedBy)
		require.NoError(t, err)
		require.NotNil(t, anomaly)

		assert.NotEqual(t, uuid.Nil, anomaly.ID)
		assert.Equal(t, tenantID, anomaly.TenantID)
		assert.Equal(t, AnomalyTypeRansomware, anomaly.AnomalyType)
		assert.Equal(t, SeverityCritical, anomaly.Severity)
		assert.Equal(t, AnomalyStatusOpen, anomaly.Status)
		assert.Equal(t, 0, anomaly.DuplicateCount)
		assert.False(t, anomaly.IsDuplicate)
		assert.NotNil(t, anomaly.Indicators)
		assert.NotNil(t, anomaly.Metadata)
	})
}

// =============================================================================
// Model Conversion Tests
// =============================================================================

func TestAnomalyRepository_ModelConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestAnomalyRepository(t)
	ctx := context.Background()

	t.Run("round trip conversion preserves data", func(t *testing.T) {
		original := createTestAnomaly(t)
		err := repo.Create(ctx, original)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, original.ID)
		require.NoError(t, err)

		// Verify all fields are preserved
		assert.Equal(t, original.ID, retrieved.ID)
		assert.Equal(t, original.TenantID, retrieved.TenantID)
		assert.Equal(t, original.AnomalyType, retrieved.AnomalyType)
		assert.Equal(t, original.Severity, retrieved.Severity)
		assert.Equal(t, original.ConfidenceScore, retrieved.ConfidenceScore)
		assert.Equal(t, original.RiskScore, retrieved.RiskScore)
		assert.Equal(t, original.Title, retrieved.Title)
		assert.Equal(t, original.Description, retrieved.Description)
		assert.Equal(t, original.DetectionMethod, retrieved.DetectionMethod)
		assert.Equal(t, original.Status, retrieved.Status)
		assert.Equal(t, original.AutoTriggered, retrieved.AutoTriggered)
		assert.Equal(t, original.IsDuplicate, retrieved.IsDuplicate)
		assert.Equal(t, original.DuplicateCount, retrieved.DuplicateCount)

		// Verify JSON fields
		assert.NotNil(t, retrieved.Indicators)
		assert.NotNil(t, retrieved.Metadata)
	})
}
