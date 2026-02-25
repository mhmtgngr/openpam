// Package repository provides tests for compliance exception repository
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

// setupComplianceExceptionTest creates a test database with compliance exceptions table
func setupComplianceExceptionTest(t *testing.T) (*sqlx.DB, *ComplianceExceptionRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)

	// Create compliance_exceptions table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS compliance_exceptions (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			control_id TEXT NOT NULL,
			control_name TEXT NOT NULL,
			framework TEXT NOT NULL,
			status TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			requested_by UUID NOT NULL,
			requested_at TIMESTAMPTZ NOT NULL,
			justification TEXT NOT NULL,
			business_reason TEXT,
			compensating_controls TEXT[],
			expires_at TIMESTAMPTZ,
			review_date TIMESTAMPTZ,
			review_notes TEXT,
			approved_by UUID,
			approved_at TIMESTAMPTZ,
			risk_accepted_by UUID,
			risk_accepted_at TIMESTAMPTZ,
			metadata BYTEA,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_compliance_exceptions_tenant ON compliance_exceptions(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_compliance_exceptions_control ON compliance_exceptions(control_id);
		CREATE INDEX IF NOT EXISTS idx_compliance_exceptions_framework ON compliance_exceptions(framework);
		CREATE INDEX IF NOT EXISTS idx_compliance_exceptions_status ON compliance_exceptions(status);
		CREATE INDEX IF NOT EXISTS idx_compliance_exceptions_expires ON compliance_exceptions(expires_at);
	`)
	require.NoError(t, err)

	logger := opamtesting.Logger(t)
	repo := NewComplianceExceptionRepository(db, logger)

	return db, repo
}

func TestNewComplianceExceptionRepository(t *testing.T) {
	db, _ := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}

	logger := opamtesting.Logger(t)
	repo := NewComplianceExceptionRepository(db, logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

// =============================================================================
// Create Tests
// =============================================================================

func TestComplianceExceptionRepository_Create(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)
	reviewDate := now.AddDate(0, 6, 0)

	t.Run("create new exception with all fields", func(t *testing.T) {
		businessReason := "Legacy system compatibility"
		reviewNotes := "Review in 6 months"

		exception := &model.ComplianceException{
			TenantID:            tenantID,
			ControlID:           "CC1.1",
			ControlName:         "Access Control",
			Framework:           model.FrameworkSOC2,
			RequestedBy:         requestedBy,
			Justification:       "Third-party provider does not support MFA",
			BusinessReason:      &businessReason,
			CompensatingControls: []string{"IP whitelist", "Certificate-based auth"},
			ExpiresAt:           &expiresAt,
			ReviewDate:          &reviewDate,
			ReviewNotes:         &reviewNotes,
			Metadata:            []byte(`{"approved_by_committee":true}`),
		}

		err := repo.Create(ctx, exception)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, exception.ID)
		assert.Equal(t, string(model.ExceptionStatusPending), exception.Status)
		assert.False(t, exception.CreatedAt.IsZero())
		assert.False(t, exception.UpdatedAt.IsZero())
		assert.False(t, exception.RequestedAt.IsZero())
	})

	t.Run("create exception with minimal fields", func(t *testing.T) {
		exception := &model.ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC3.2",
			ControlName:   "Encryption",
			Framework:     model.FrameworkISO27001,
			RequestedBy:   requestedBy,
			Justification: "Technical limitation",
		}

		err := repo.Create(ctx, exception)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, exception.ID)
		assert.Nil(t, exception.ExpiresAt)
		assert.Nil(t, exception.ReviewDate)
	})
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestComplianceExceptionRepository_GetByID(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)

	exception := &model.ComplianceException{
		TenantID:            tenantID,
		ControlID:           "CC1.1",
		ControlName:         "Access Control",
		Framework:           model.FrameworkSOC2,
		RequestedBy:         requestedBy,
		Justification:       "Testing justification",
		CompensatingControls: []string{"Manual review"},
		ExpiresAt:           &expiresAt,
	}

	err := repo.Create(ctx, exception)
	require.NoError(t, err)

	t.Run("get existing exception", func(t *testing.T) {
		found, err := repo.GetByID(ctx, exception.ID)
		require.NoError(t, err)

		assert.Equal(t, exception.ID, found.ID)
		assert.Equal(t, tenantID, found.TenantID)
		assert.Equal(t, "CC1.1", found.ControlID)
		assert.Equal(t, "Access Control", found.ControlName)
		assert.Equal(t, model.FrameworkSOC2, found.Framework)
		assert.Equal(t, string(model.ExceptionStatusPending), found.Status)
		assert.NotNil(t, found.ExpiresAt)
		assert.NotEmpty(t, found.CompensatingControls)
	})

	t.Run("get non-existent exception", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestComplianceExceptionRepository_List(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)
	expiredAt := now.Add(-1 * time.Hour) // Already expired

	// Create test exceptions
	statuses := []string{
		string(model.ExceptionStatusPending),
		string(model.ExceptionStatusApproved),
		string(model.ExceptionStatusDenied),
		string(model.ExceptionStatusApproved),
	}

	for i, status := range statuses {
		exception := &model.ComplianceException{
			TenantID:            tenantID,
			ControlID:           fmt.Sprintf("CC1.%d", i+1),
			ControlName:         fmt.Sprintf("Control %d", i+1),
			Framework:           model.FrameworkSOC2,
			RequestedBy:         requestedBy,
			Justification:       fmt.Sprintf("Justification %d", i),
			CompensatingControls: []string{fmt.Sprintf("Control %d", i)},
			ExpiresAt:           &expiresAt,
		}

		err := repo.Create(ctx, exception)
		require.NoError(t, err)

		// Update status for testing
		if status != string(model.ExceptionStatusPending) {
			_ = repo.UpdateStatus(ctx, exception.ID, status, nil, nil)
		}
	}

	// Create expired exception
	expiredException := &model.ComplianceException{
		TenantID:            tenantID,
		ControlID:           "CC2.1",
		ControlName:         "Expired Control",
		Framework:           model.FrameworkSOC2,
		RequestedBy:         requestedBy,
		Justification:       "Expired test",
		ExpiresAt:           &expiredAt,
	}
	_ = repo.Create(ctx, expiredException)

	t.Run("list all exceptions", func(t *testing.T) {
		filter := model.ComplianceExceptionFilter{
			IncludeExpired: true,
		}

		exceptions, total, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(exceptions), 5)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("list with status filter", func(t *testing.T) {
		status := string(model.ExceptionStatusApproved)
		filter := model.ComplianceExceptionFilter{
			Status:        &status,
			IncludeExpired: true,
		}

		exceptions, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(exceptions), 0)

		for _, e := range exceptions {
			assert.Equal(t, "approved", e.Status)
		}
	})

	t.Run("list with framework filter", func(t *testing.T) {
		framework := model.FrameworkSOC2
		filter := model.ComplianceExceptionFilter{
			Framework:      &framework,
			IncludeExpired: true,
		}

		exceptions, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(exceptions), 0)

		for _, e := range exceptions {
			assert.Equal(t, model.FrameworkSOC2, e.Framework)
		}
	})

	t.Run("list excluding expired", func(t *testing.T) {
		filter := model.ComplianceExceptionFilter{
			IncludeExpired: false,
		}

		exceptions, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)

		// Should not include expired exception
		for _, e := range exceptions {
			if e.ExpiresAt != nil {
				assert.True(t, e.ExpiresAt.After(now))
			}
		}
	})

	t.Run("list with control ID filter", func(t *testing.T) {
		controlID := "CC1.1"
		filter := model.ComplianceExceptionFilter{
			ControlID:     &controlID,
			IncludeExpired: true,
		}

		exceptions, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
		assert.Greater(t, len(exceptions), 0)

		for _, e := range exceptions {
			assert.Equal(t, "CC1.1", e.ControlID)
		}
	})

	t.Run("list with risk level filter", func(t *testing.T) {
		riskLevel := string(model.SeverityMedium)
		filter := model.ComplianceExceptionFilter{
			RiskLevel:     &riskLevel,
			IncludeExpired: true,
		}

		_, _, err := repo.List(ctx, tenantID, filter, 10, 0)

		require.NoError(t, err)
	})

	t.Run("list with pagination", func(t *testing.T) {
		filter := model.ComplianceExceptionFilter{
			IncludeExpired: true,
		}

		exceptions, total, err := repo.List(ctx, tenantID, filter, 2, 0)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(exceptions), 2)
		assert.GreaterOrEqual(t, total, 5)
	})
}

// =============================================================================
// UpdateStatus Tests
// =============================================================================

func TestComplianceExceptionRepository_UpdateStatus(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	approvedBy := uuid.New()
	riskAcceptedBy := uuid.New()

	exception := &model.ComplianceException{
		TenantID:            tenantID,
		ControlID:           "CC1.1",
		ControlName:         "Access Control",
		Framework:           model.FrameworkSOC2,
		RequestedBy:         requestedBy,
		Justification:       "Testing approval",
		CompensatingControls: []string{"Manual review"},
	}

	err := repo.Create(ctx, exception)
	require.NoError(t, err)

	t.Run("approve exception", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, exception.ID, string(model.ExceptionStatusApproved), &approvedBy, &riskAcceptedBy)

		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, exception.ID)
		require.NoError(t, err)

		assert.Equal(t, string(model.ExceptionStatusApproved), updated.Status)
		assert.Equal(t, approvedBy, *updated.ApprovedBy)
		assert.False(t, updated.ApprovedAt.IsZero())
		assert.Equal(t, riskAcceptedBy, *updated.RiskAcceptedBy)
		assert.False(t, updated.RiskAcceptedAt.IsZero())
	})

	t.Run("deny exception", func(t *testing.T) {
		deniedException := &model.ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC1.2",
			ControlName:   "Denied Control",
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: "Will be denied",
		}

		_ = repo.Create(ctx, deniedException)

		err := repo.UpdateStatus(ctx, deniedException.ID, string(model.ExceptionStatusDenied), nil, nil)

		require.NoError(t, err)

		updated, _ := repo.GetByID(ctx, deniedException.ID)
		assert.Equal(t, string(model.ExceptionStatusDenied), updated.Status)
	})
}

// =============================================================================
// Update Tests
// =============================================================================

func TestComplianceExceptionRepository_Update(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	originalExpiresAt := time.Now().AddDate(0, 6, 0)

	exception := &model.ComplianceException{
		TenantID:            tenantID,
		ControlID:           "CC1.1",
		ControlName:         "Access Control",
		Framework:           model.FrameworkSOC2,
		RequestedBy:         requestedBy,
		Justification:       "Original justification",
		CompensatingControls: []string{"Manual review"},
		ExpiresAt:           &originalExpiresAt,
	}

	err := repo.Create(ctx, exception)
	require.NoError(t, err)

	t.Run("update exception fields", func(t *testing.T) {
		newExpiresAt := time.Now().AddDate(1, 0, 0)
		newReviewDate := time.Now().AddDate(0, 9, 0)
		newBusinessReason := "Updated business reason"
		newReviewNotes := "Updated review notes"

		exception.Justification = "Updated justification"
		exception.BusinessReason = &newBusinessReason
		exception.CompensatingControls = []string{"Manual review", "Enhanced monitoring"}
		exception.ExpiresAt = &newExpiresAt
		exception.ReviewDate = &newReviewDate
		exception.ReviewNotes = &newReviewNotes

		err := repo.Update(ctx, exception)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, exception.ID)
		require.NoError(t, err)

		assert.Equal(t, "Updated justification", updated.Justification)
		assert.Equal(t, "Updated business reason", *updated.BusinessReason)
		assert.Len(t, updated.CompensatingControls, 2)
		assert.Equal(t, newExpiresAt.Truncate(time.Second), updated.ExpiresAt.Truncate(time.Second))
		assert.Equal(t, "Updated review notes", *updated.ReviewNotes)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestComplianceExceptionRepository_Delete(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()

	exception := &model.ComplianceException{
		TenantID:      tenantID,
		ControlID:     "CC1.1",
		ControlName:   "To Delete",
		Framework:     model.FrameworkSOC2,
		RequestedBy:   requestedBy,
		Justification: "Will be deleted",
	}

	err := repo.Create(ctx, exception)
	require.NoError(t, err)

	t.Run("delete existing exception", func(t *testing.T) {
		err := repo.Delete(ctx, exception.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetByID(ctx, exception.ID)
		assert.Error(t, err)
	})

	t.Run("delete non-existent exception", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows affected")
	})
}

// =============================================================================
// GetByControlID Tests
// =============================================================================

func TestComplianceExceptionRepository_GetByControlID(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()

	controlID := "CC1.1"

	// Create multiple exceptions for the same control
	for i := 0; i < 3; i++ {
		exception := &model.ComplianceException{
			TenantID:            tenantID,
			ControlID:           controlID,
			ControlName:         fmt.Sprintf("Exception %d", i),
			Framework:           model.FrameworkSOC2,
			RequestedBy:         requestedBy,
			Justification:       fmt.Sprintf("Justification %d", i),
			CompensatingControls: []string{fmt.Sprintf("Control %d", i)},
		}
		_ = repo.Create(ctx, exception)
	}

	t.Run("get by control ID", func(t *testing.T) {
		exceptions, err := repo.GetByControlID(ctx, tenantID, controlID)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(exceptions), 3)

		for _, e := range exceptions {
			assert.Equal(t, controlID, e.ControlID)
		}
	})

	t.Run("get by non-existent control ID", func(t *testing.T) {
		exceptions, err := repo.GetByControlID(ctx, tenantID, "CC999")

		require.NoError(t, err)
		assert.Empty(t, exceptions)
	})
}

// =============================================================================
// GetExpiringSoon Tests
// =============================================================================

func TestComplianceExceptionRepository_GetExpiringSoon(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	approvedBy := uuid.New()
	now := time.Now()

	// Create exceptions with different expiration times
	expiresSoon := now.Add(15 * 24 * time.Hour)     // Expires in 15 days
	expiresLater := now.Add(60 * 24 * time.Hour)    // Expires in 60 days
	expiresFar := now.Add(365 * 24 * time.Hour)    // Expires in 1 year

	exceptions := []*model.ComplianceException{
		{
			TenantID:      tenantID,
			ControlID:     "CC1.1",
			ControlName:   "Expiring Soon",
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: "Test",
			ExpiresAt:     &expiresSoon,
		},
		{
			TenantID:      tenantID,
			ControlID:     "CC1.2",
			ControlName:   "Expires Later",
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: "Test",
			ExpiresAt:     &expiresLater,
		},
		{
			TenantID:      tenantID,
			ControlID:     "CC1.3",
			ControlName:   "Expires Far",
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: "Test",
			ExpiresAt:     &expiresFar,
		},
	}

	for _, e := range exceptions {
		_ = repo.Create(ctx, e)
		_ = repo.UpdateStatus(ctx, e.ID, string(model.ExceptionStatusApproved), &approvedBy, nil)
	}

	t.Run("get exceptions expiring within 30 days", func(t *testing.T) {
		expiring, err := repo.GetExpiringSoon(ctx, tenantID, 30*24*time.Hour)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(expiring), 1)

		for _, e := range expiring {
			assert.Equal(t, string(model.ExceptionStatusApproved), e.Status)
			assert.NotNil(t, e.ExpiresAt)
			assert.True(t, e.ExpiresAt.Before(now.Add(30*24*time.Hour)))
		}
	})

	t.Run("get exceptions expiring within 7 days", func(t *testing.T) {
		expiring, err := repo.GetExpiringSoon(ctx, tenantID, 7*24*time.Hour)

		require.NoError(t, err)
		// May be empty or have results depending on test timing
		assert.NotNil(t, expiring)
	})
}

// =============================================================================
// GetExpired Tests
// =============================================================================

func TestComplianceExceptionRepository_GetExpired(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	approvedBy := uuid.New()
	now := time.Now()

	expiredTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(30 * 24 * time.Hour)

	// Create expired exception
	expiredException := &model.ComplianceException{
		TenantID:      tenantID,
		ControlID:     "CC1.1",
		ControlName:   "Expired Exception",
		Framework:     model.FrameworkSOC2,
		RequestedBy:   requestedBy,
		Justification: "Test expired",
		ExpiresAt:     &expiredTime,
	}
	_ = repo.Create(ctx, expiredException)
	_ = repo.UpdateStatus(ctx, expiredException.ID, string(model.ExceptionStatusApproved), &approvedBy, nil)

	// Create active exception
	activeException := &model.ComplianceException{
		TenantID:      tenantID,
		ControlID:     "CC1.2",
		ControlName:   "Active Exception",
		Framework:     model.FrameworkSOC2,
		RequestedBy:   requestedBy,
		Justification: "Test active",
		ExpiresAt:     &futureTime,
	}
	_ = repo.Create(ctx, activeException)
	_ = repo.UpdateStatus(ctx, activeException.ID, string(model.ExceptionStatusApproved), &approvedBy, nil)

	t.Run("get expired exceptions", func(t *testing.T) {
		expired, err := repo.GetExpired(ctx, tenantID)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(expired), 1)

		for _, e := range expired {
			assert.Equal(t, string(model.ExceptionStatusApproved), e.Status)
			assert.NotNil(t, e.ExpiresAt)
			assert.True(t, e.ExpiresAt.Before(now))
		}
	})
}

// =============================================================================
// MarkExpired Tests
// =============================================================================

func TestComplianceExceptionRepository_MarkExpired(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()
	approvedBy := uuid.New()
	now := time.Now()

	expiredTime := now.Add(-1 * time.Hour)

	// Create expired but approved exceptions
	for i := 0; i < 3; i++ {
		exception := &model.ComplianceException{
			TenantID:      tenantID,
			ControlID:     fmt.Sprintf("CC1.%d", i+1),
			ControlName:   fmt.Sprintf("Expired %d", i+1),
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: "Test",
			ExpiresAt:     &expiredTime,
		}
		_ = repo.Create(ctx, exception)
		_ = repo.UpdateStatus(ctx, exception.ID, string(model.ExceptionStatusApproved), &approvedBy, nil)
	}

	t.Run("mark expired exceptions", func(t *testing.T) {
		count, err := repo.MarkExpired(ctx, tenantID)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 3)

		// Verify they are marked as expired
		expired, _ := repo.GetExpired(ctx, tenantID)
		for _, e := range expired {
			assert.Equal(t, string(model.ExceptionStatusExpired), e.Status)
		}
	})
}

// =============================================================================
// GetPending Tests
// =============================================================================

func TestComplianceExceptionRepository_GetPending(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()

	// Create pending exceptions
	for i := 0; i < 3; i++ {
		exception := &model.ComplianceException{
			TenantID:      tenantID,
			ControlID:     fmt.Sprintf("CC2.%d", i+1),
			ControlName:   fmt.Sprintf("Pending %d", i+1),
			Framework:     model.FrameworkISO27001,
			RequestedBy:   requestedBy,
			Justification: fmt.Sprintf("Pending request %d", i),
		}
		_ = repo.Create(ctx, exception)
	}

	t.Run("get pending exceptions", func(t *testing.T) {
		pending, err := repo.GetPending(ctx, tenantID)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(pending), 3)

		for _, e := range pending {
			assert.Equal(t, string(model.ExceptionStatusPending), e.Status)
		}
	})

	t.Run("get pending when none exist", func(t *testing.T) {
		emptyTenantID := uuid.New()
		pending, err := repo.GetPending(ctx, emptyTenantID)

		require.NoError(t, err)
		assert.Empty(t, pending)
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestComplianceExceptionRepository_EdgeCases(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	requestedBy := uuid.New()

	t.Run("create with empty compensating controls", func(t *testing.T) {
		exception := &model.ComplianceException{
			TenantID:            tenantID,
			ControlID:           "CC1.1",
			ControlName:         "Test",
			Framework:           model.FrameworkSOC2,
			RequestedBy:         requestedBy,
			Justification:       "Test",
			CompensatingControls: []string{},
		}

		err := repo.Create(ctx, exception)
		require.NoError(t, err)
	})

	t.Run("create with nil compensating controls", func(t *testing.T) {
		exception := &model.ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC1.2",
			ControlName:   "Test",
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: "Test",
		}

		err := repo.Create(ctx, exception)
		require.NoError(t, err)
	})

	t.Run("handle very long justification", func(t *testing.T) {
		longJustification := string(make([]byte, 10000))
		for i := range longJustification {
			longJustification = longJustification[:i] + "A" + longJustification[i+1:]
		}

		exception := &model.ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC1.3",
			ControlName:   "Long Justification",
			Framework:     model.FrameworkSOC2,
			RequestedBy:   requestedBy,
			Justification: longJustification,
		}

		err := repo.Create(ctx, exception)
		// Should succeed or fail gracefully
		if err != nil {
			t.Logf("Long justification handled: %v", err)
		}
	})
}

// =============================================================================
// Concurrent Operations Tests
// =============================================================================

func TestComplianceExceptionRepository_ConcurrentOperations(t *testing.T) {
	db, repo := setupComplianceExceptionTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())

	t.Run("concurrent creates", func(t *testing.T) {
		errChan := make(chan error, 10)
		requestedBy := uuid.New()

		for i := 0; i < 10; i++ {
			go func(index int) {
				exception := &model.ComplianceException{
					TenantID:      tenantID,
					ControlID:     fmt.Sprintf("CC3.%d", index),
					ControlName:   fmt.Sprintf("Concurrent %d", index),
					Framework:     model.FrameworkSOC2,
					RequestedBy:   requestedBy,
					Justification: fmt.Sprintf("Concurrent test %d", index),
				}
				errChan <- repo.Create(ctx, exception)
			}(i)
		}

		for i := 0; i < 10; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}
	})
}
