// Package analytics provides tests for compliance exception persistence
package analytics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup
// =============================================================================

func setupComplianceExceptionTestDB(t *testing.T) *sqlx.DB {
	dsn := "host=localhost port=5432 user=openpam_test password=openpam_test dbname=openpam_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
		return nil
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	schema := `
		CREATE TABLE IF NOT EXISTS compliance_exceptions (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			control_id TEXT NOT NULL,
			control_name TEXT NOT NULL,
			framework TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			risk_level TEXT NOT NULL,
			requested_by UUID NOT NULL,
			requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
			approved_by UUID,
			approved_at TIMESTAMP,
			expires_at TIMESTAMP,
			justification TEXT,
			business_reason TEXT,
			compensating_controls TEXT,
			risk_accepted_by UUID,
			risk_accepted_at TIMESTAMP,
			review_date TIMESTAMP,
			review_notes TEXT,
			metadata BYTEA,
			last_notification_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_exception_tenant ON compliance_exceptions(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_exception_status ON compliance_exceptions(status);
		CREATE INDEX IF NOT EXISTS idx_exception_framework ON compliance_exceptions(framework);
		CREATE INDEX IF NOT EXISTS idx_exception_control ON compliance_exceptions(control_id);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	_, _ = db.Exec("DELETE FROM compliance_exceptions")

	t.Cleanup(func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS compliance_exceptions")
		_ = db.Close()
	})

	return db
}

func newTestComplianceExceptionRepository(t *testing.T) *ComplianceExceptionRepository {
	db := setupComplianceExceptionTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
	}
	logger := zerolog.Nop()
	return NewComplianceExceptionRepository(db, logger)
}

func createTestComplianceException(t *testing.T) *ComplianceException {
	tenantID := uuid.New()
	requestedBy := uuid.New()

	metadata := map[string]interface{}{
		"test": true,
		"source": "unit_test",
	}
	metadataJSON, _ := json.Marshal(metadata)

	return &ComplianceException{
		TenantID:      tenantID,
		ControlID:     "AC-01",
		ControlName:   "Access Control Policy",
		Framework:     "NIST-800-53",
		Status:        ExceptionStatusPending,
		RiskLevel:     "medium",
		RequestedBy:   requestedBy,
		RequestedAt:   time.Now(),
		Justification: strPtr("Test justification for exception"),
		BusinessReason: strPtr("Business requirement testing"),
		CompensatingControls: strPtr("Alternative control mechanism"),
		Metadata:      metadataJSON,
	}
}

// =============================================================================
// Create Tests
// =============================================================================

func TestComplianceExceptionRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	t.Run("creates exception successfully", func(t *testing.T) {
		exception := createTestComplianceException(t)

		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, exception.ID)
		assert.False(t, exception.CreatedAt.IsZero())
		assert.False(t, exception.UpdatedAt.IsZero())
	})

	t.Run("creates exception with all optional fields", func(t *testing.T) {
		tenantID := uuid.New()
		requestedBy := uuid.New()
		approvedBy := uuid.New()
		riskAcceptedBy := uuid.New()

		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		reviewDate := time.Now().Add(7 * 24 * time.Hour)

		exception := &ComplianceException{
			TenantID:       tenantID,
			ControlID:      "AU-02",
			ControlName:    "Audit Events",
			Framework:      "NIST-800-53",
			Status:         ExceptionStatusApproved,
			RiskLevel:      "high",
			RequestedBy:    requestedBy,
			RequestedAt:    time.Now(),
			ApprovedBy:     &approvedBy,
			ApprovedAt:     timePtr(time.Now()),
			ExpiresAt:      &expiresAt,
			Justification:  strPtr("Need exception for audit"),
			BusinessReason: strPtr("Legacy system compatibility"),
			CompensatingControls: strPtr("Manual audit review"),
			RiskAcceptedBy: &riskAcceptedBy,
			RiskAcceptedAt: timePtr(time.Now()),
			ReviewDate:     &reviewDate,
			ReviewNotes:    strPtr("Review pending validation"),
		}

		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, exception.ID)
		assert.NotNil(t, exception.ApprovedBy)
		assert.NotNil(t, exception.ExpiresAt)
	})
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestComplianceExceptionRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	t.Run("retrieves existing exception", func(t *testing.T) {
		exception := createTestComplianceException(t)
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, exception.ID, exception.TenantID)
		require.NoError(t, err)

		assert.Equal(t, exception.ID, retrieved.ID)
		assert.Equal(t, exception.TenantID, retrieved.TenantID)
		assert.Equal(t, exception.ControlID, retrieved.ControlID)
		assert.Equal(t, exception.ControlName, retrieved.ControlName)
		assert.Equal(t, exception.Framework, retrieved.Framework)
		assert.Equal(t, exception.Status, retrieved.Status)
		assert.Equal(t, exception.RiskLevel, retrieved.RiskLevel)
		assert.Equal(t, exception.Justification, retrieved.Justification)
	})

	t.Run("returns error for non-existent exception", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New(), uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestComplianceExceptionRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create test exceptions
	statusPending := ExceptionStatusPending
	statusApproved := ExceptionStatusApproved
	statusDenied := ExceptionStatusDenied

	exceptions := []ComplianceException{
		{
			TenantID:    tenantID,
			ControlID:   "AC-01",
			ControlName: "Access Control",
			Framework:   "NIST-800-53",
			Status:      statusPending,
			RiskLevel:   "high",
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
		},
		{
			TenantID:    tenantID,
			ControlID:   "AU-02",
			ControlName: "Audit Events",
			Framework:   "NIST-800-53",
			Status:      statusApproved,
			RiskLevel:   "medium",
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
		},
		{
			TenantID:    tenantID,
			ControlID:   "CM-03",
			ControlName: "Configuration",
			Framework:   "ISO-27001",
			Status:      statusDenied,
			RiskLevel:   "critical",
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
		},
	}

	for i := range exceptions {
		err := repo.Create(ctx, &exceptions[i])
		require.NoError(t, err)
	}

	t.Run("lists all exceptions for tenant", func(t *testing.T) {
		filter := ExceptionFilter{
			Limit:  50,
			Offset: 0,
		}
		results, total, err := repo.List(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, results, 3)
	})

	t.Run("filters by status", func(t *testing.T) {
		filter := ExceptionFilter{
			Status: &statusPending,
			Limit:  50,
			Offset: 0,
		}
		results, total, err := repo.List(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, ExceptionStatusPending, results[0].Status)
	})

	t.Run("filters by framework", func(t *testing.T) {
		filter := ExceptionFilter{
			Framework: "NIST-800-53",
			Limit:     50,
			Offset:    0,
		}
		results, total, err := repo.List(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
	})

	t.Run("filters by risk level", func(t *testing.T) {
		filter := ExceptionFilter{
			RiskLevel: "high",
			Limit:     50,
			Offset:    0,
		}
		results, total, err := repo.List(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("filters by control ID", func(t *testing.T) {
		filter := ExceptionFilter{
			ControlID: "AC-01",
			Limit:     50,
			Offset:    0,
		}
		results, total, err := repo.List(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
	})

	t.Run("applies pagination", func(t *testing.T) {
		filter := ExceptionFilter{
			Limit:  2,
			Offset: 0,
		}
		results, total, err := repo.List(ctx, tenantID, filter)

		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, results, 2)

		results2, _, err := repo.List(ctx, tenantID, ExceptionFilter{Limit: 2, Offset: 2})
		require.NoError(t, err)
		assert.Len(t, results2, 1)
	})
}

// =============================================================================
// Update Tests
// =============================================================================

func TestComplianceExceptionRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	t.Run("updates exception fields", func(t *testing.T) {
		exception := createTestComplianceException(t)
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		// Update fields
		newJustification := "Updated justification"
		newRiskLevel := "high"
		exception.Justification = &newJustification
		exception.RiskLevel = newRiskLevel

		err = repo.Update(ctx, exception)
		require.NoError(t, err)

		// Verify update
		retrieved, err := repo.GetByID(ctx, exception.ID, exception.TenantID)
		require.NoError(t, err)
		assert.Equal(t, &newJustification, retrieved.Justification)
		assert.Equal(t, newRiskLevel, retrieved.RiskLevel)
	})
}

// =============================================================================
// UpdateStatus Tests
// =============================================================================

func TestComplianceExceptionRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	t.Run("updates status to approved", func(t *testing.T) {
		exception := createTestComplianceException(t)
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		approvedBy := uuid.New()
		expiresAt := time.Now().Add(90 * 24 * time.Hour)

		err = repo.UpdateStatus(ctx, exception.ID, exception.TenantID, ExceptionStatusApproved, &approvedBy, &expiresAt)
		require.NoError(t, err)

		// Verify update
		retrieved, err := repo.GetByID(ctx, exception.ID, exception.TenantID)
		require.NoError(t, err)
		assert.Equal(t, ExceptionStatusApproved, retrieved.Status)
		assert.Equal(t, &approvedBy, retrieved.ApprovedBy)
		assert.NotNil(t, retrieved.ApprovedAt)
		assert.NotNil(t, retrieved.ExpiresAt)
	})

	t.Run("updates status to denied", func(t *testing.T) {
		exception := createTestComplianceException(t)
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, exception.ID, exception.TenantID, ExceptionStatusDenied, nil, nil)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, exception.ID, exception.TenantID)
		require.NoError(t, err)
		assert.Equal(t, ExceptionStatusDenied, retrieved.Status)
	})

	t.Run("updates status to expired", func(t *testing.T) {
		exception := createTestComplianceException(t)
		exception.Status = ExceptionStatusApproved
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, exception.ID, exception.TenantID, ExceptionStatusExpired, nil, nil)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, exception.ID, exception.TenantID)
		require.NoError(t, err)
		assert.Equal(t, ExceptionStatusExpired, retrieved.Status)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestComplianceExceptionRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	t.Run("deletes exception", func(t *testing.T) {
		exception := createTestComplianceException(t)
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		err = repo.Delete(ctx, exception.ID, exception.TenantID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetByID(ctx, exception.ID, exception.TenantID)
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent exception", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New(), uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestComplianceExceptionRepository_GetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create exceptions with different statuses
	exceptions := []struct {
		status    ExceptionStatus
		riskLevel string
	}{
		{ExceptionStatusPending, "critical"},
		{ExceptionStatusPending, "high"},
		{ExceptionStatusApproved, "medium"},
		{ExceptionStatusApproved, "low"},
		{ExceptionStatusDenied, "high"},
		{ExceptionStatusExpired, "critical"},
		{ExceptionStatusRevoked, "medium"},
	}

	for i, e := range exceptions {
		expirationTime := time.Now().Add(3 * 24 * time.Hour)
		exception := &ComplianceException{
			TenantID:    tenantID,
			ControlID:   "TEST-01",
			ControlName: "Test Control",
			Framework:   "NIST-800-53",
			Status:      e.status,
			RiskLevel:   e.riskLevel,
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
			ExpiresAt:   &expirationTime,
		}
		if e.status == ExceptionStatusApproved {
			exception.ApprovedAt = timePtr(time.Now())
		}
		err := repo.Create(ctx, exception)
		require.NoError(t, err)
		_ = i
	}

	t.Run("returns accurate statistics", func(t *testing.T) {
		stats, err := repo.GetStats(ctx, tenantID)
		require.NoError(t, err)

		assert.Equal(t, 7, stats.TotalCount)
		assert.Equal(t, 2, stats.PendingCount)
		assert.Equal(t, 2, stats.ApprovedCount)
		assert.Equal(t, 1, stats.DeniedCount)
		assert.Equal(t, 1, stats.ExpiredCount)
		assert.Equal(t, 1, stats.RevokedCount)
		assert.Equal(t, 2, stats.CriticalCount)
		assert.Equal(t, 2, stats.HighCount)
	})
}

// =============================================================================
// GetByControlID Tests
// =============================================================================

func TestComplianceExceptionRepository_GetByControlID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	controlID := "AC-01"
	framework := "NIST-800-53"

	// Create exceptions for same control
	for i := 0; i < 3; i++ {
		exception := &ComplianceException{
			TenantID:    tenantID,
			ControlID:   controlID,
			ControlName: "Access Control",
			Framework:   framework,
			Status:      ExceptionStatusApproved,
			RiskLevel:   "medium",
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
		}
		err := repo.Create(ctx, exception)
		require.NoError(t, err)
	}

	t.Run("retrieves exceptions by control", func(t *testing.T) {
		results, err := repo.GetByControlID(ctx, tenantID, controlID, framework)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("returns empty for non-existent control", func(t *testing.T) {
		results, err := repo.GetByControlID(ctx, tenantID, "NON-EXISTENT", framework)
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

// =============================================================================
// GetPendingApprovals Tests
// =============================================================================

func TestComplianceExceptionRepository_GetPendingApprovals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()
	tenantID := uuid.New()

	// Create pending exceptions
	for i := 0; i < 3; i++ {
		exception := &ComplianceException{
			TenantID:    tenantID,
			ControlID:   "TEST-01",
			ControlName: "Test Control",
			Framework:   "NIST-800-53",
			Status:      ExceptionStatusPending,
			RiskLevel:   "high",
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
		}
		err := repo.Create(ctx, exception)
		require.NoError(t, err)
	}

	// Create approved exception
	approved := &ComplianceException{
		TenantID:    tenantID,
		ControlID:   "TEST-02",
		ControlName: "Test Control 2",
		Framework:   "NIST-800-53",
		Status:      ExceptionStatusApproved,
		RiskLevel:   "medium",
		RequestedBy: uuid.New(),
		RequestedAt: time.Now(),
	}
	err := repo.Create(ctx, approved)
	require.NoError(t, err)

	t.Run("retrieves pending exceptions", func(t *testing.T) {
		results, err := repo.GetPendingApprovals(ctx, tenantID, 10)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("applies limit", func(t *testing.T) {
		results, err := repo.GetPendingApprovals(ctx, tenantID, 2)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

// =============================================================================
// GetExpiringExceptions Tests
// =============================================================================

func TestComplianceExceptionRepository_GetExpiringExceptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	// Create exceptions expiring soon
	for i := 0; i < 3; i++ {
		expiresAt := time.Now().Add(24 * time.Hour)
		exception := &ComplianceException{
			TenantID:    uuid.New(),
			ControlID:   "TEST-01",
			ControlName: "Test Control",
			Framework:   "NIST-800-53",
			Status:      ExceptionStatusApproved,
			RiskLevel:   "medium",
			RequestedBy: uuid.New(),
			RequestedAt: time.Now(),
			ExpiresAt:   &expiresAt,
		}
		err := repo.Create(ctx, exception)
		require.NoError(t, err)
	}

	t.Run("retrieves expiring exceptions", func(t *testing.T) {
		results, err := repo.GetExpiringExceptions(ctx, 7, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 3)
	})
}

// =============================================================================
// UpdateLastNotification Tests
// =============================================================================

func TestComplianceExceptionRepository_UpdateLastNotification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	repo := newTestComplianceExceptionRepository(t)
	ctx := context.Background()

	t.Run("updates last notification timestamp", func(t *testing.T) {
		exception := createTestComplianceException(t)
		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		err = repo.UpdateLastNotification(ctx, exception.ID)
		require.NoError(t, err)

		// Verify updated_at changed
		// Note: We can't directly verify last_notification_at without
		// retrieving the record, which would require adding that field to the struct
	})
}

// =============================================================================
// ExceptionStatus Enum Tests
// =============================================================================

func TestExceptionStatus_Enum(t *testing.T) {
	t.Run("has correct status values", func(t *testing.T) {
		assert.Equal(t, ExceptionStatus("pending"), ExceptionStatusPending)
		assert.Equal(t, ExceptionStatus("approved"), ExceptionStatusApproved)
		assert.Equal(t, ExceptionStatus("denied"), ExceptionStatusDenied)
		assert.Equal(t, ExceptionStatus("expired"), ExceptionStatusExpired)
		assert.Equal(t, ExceptionStatus("revoked"), ExceptionStatusRevoked)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func timePtr(t time.Time) *time.Time {
	return &t
}
