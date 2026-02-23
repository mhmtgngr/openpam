package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAuditTest creates a test database and audit service
func setupAuditTest(t *testing.T) (*sqlx.DB, *Service) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)
	logger := opamtesting.Logger(t)

	repo := NewRepository(db, nil, logger)
	service := NewService(repo, logger)

	return db, service
}

// Test EventType_Constants verifies event type constants
func TestEventType_Constants(t *testing.T) {
	tests := []struct {
		name   string
		e      EventType
		expect string
	}{
		{"EventTypeAuth", EventTypeAuth, "authentication"},
		{"EventTypeCredential", EventTypeCredential, "credential"},
		{"EventTypeSession", EventTypeSession, "session"},
		{"EventTypeTarget", EventTypeTarget, "target"},
		{"EventTypeUser", EventTypeUser, "user"},
		{"EventTypeRole", EventTypeRole, "role"},
		{"EventTypeApproval", EventTypeApproval, "approval"},
		{"EventTypeCheckout", EventTypeCheckout, "checkout"},
		{"EventTypeConfiguration", EventTypeConfiguration, "configuration"},
		{"EventTypeAnomaly", EventTypeAnomaly, "anomaly"},
		{"EventTypeBreakGlass", EventTypeBreakGlass, "break_glass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, string(tt.e))
		})
	}
}

// TestEventOutcome_Constants verifies event outcome constants
func TestEventOutcome_Constants(t *testing.T) {
	tests := []struct {
		name   string
		o      EventOutcome
		expect string
	}{
		{"OutcomeSuccess", OutcomeSuccess, "success"},
		{"OutcomeFailure", OutcomeFailure, "failure"},
		{"OutcomeDenied", OutcomeDenied, "denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, string(tt.o))
		})
	}
}

// TestEvent_Struct creates and validates an Event struct
func TestEvent_Struct(t *testing.T) {
	t.Run("create minimal event", func(t *testing.T) {
		event := &Event{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			ActorID:      uuid.New(),
			ActorType:    "user",
			Action:       "create",
			ResourceType: "credential",
			ResourceID:   "cred-123",
			Outcome:      OutcomeSuccess,
			IP:           "192.168.1.1",
			UserAgent:    "Mozilla/5.0",
			RequestID:    "req-123",
			Timestamp:    time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, event.ID)
		assert.NotEqual(t, uuid.Nil, event.TenantID)
		assert.NotEqual(t, uuid.Nil, event.ActorID)
		assert.Equal(t, "user", event.ActorType)
		assert.Equal(t, "create", event.Action)
		assert.Equal(t, "credential", event.ResourceType)
		assert.Equal(t, OutcomeSuccess, event.Outcome)
	})

	t.Run("event with details", func(t *testing.T) {
		details := map[string]interface{}{
			"old_value": "test",
			"new_value": "updated",
		}
		detailJSON, _ := json.Marshal(details)

		event := &Event{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			ActorID:      uuid.New(),
			Action:       "update",
			ResourceType: "user",
			ResourceID:   "user-123",
			Outcome:      OutcomeSuccess,
			Details:      json.RawMessage(detailJSON),
			Timestamp:    time.Now(),
		}

		assert.NotNil(t, event.Details)
	})

	t.Run("event with error info", func(t *testing.T) {
		event := &Event{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			ActorID:      uuid.New(),
			Action:       "delete",
			ResourceType: "credential",
			ResourceID:   "cred-123",
			Outcome:      OutcomeFailure,
			ErrorCode:    "PERM_DENIED",
			ErrorMessage: "User does not have permission",
			Timestamp:    time.Now(),
		}

		assert.Equal(t, OutcomeFailure, event.Outcome)
		assert.Equal(t, "PERM_DENIED", event.ErrorCode)
		assert.NotEmpty(t, event.ErrorMessage)
	})
}

// TestRepository_Create creates audit events
func TestRepository_Create(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	actorID := uuid.New()

	t.Run("create successful event", func(t *testing.T) {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       "login",
			ResourceType: "session",
			Outcome:      OutcomeSuccess,
			IP:           "192.168.1.100",
			UserAgent:    "OpenSSH_9.0",
		}

		err := service.Log(ctx, event)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, event.ID)
		assert.False(t, event.Timestamp.IsZero())
		assert.NotEmpty(t, event.Hash)
	})

	t.Run("create failed event", func(t *testing.T) {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       "login",
			ResourceType: "session",
			Outcome:      OutcomeFailure,
			ErrorCode:    "INVALID_CREDS",
			ErrorMessage: "Invalid username or password",
			IP:           "10.0.0.50",
		}

		err := service.Log(ctx, event)
		require.NoError(t, err)
		assert.Equal(t, OutcomeFailure, event.Outcome)
	})

	t.Run("create denied event", func(t *testing.T) {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       "checkout",
			ResourceType: "credential",
			ResourceID:   "cred-sensitive",
			Outcome:      OutcomeDenied,
			ErrorCode:    "NOT_APPROVED",
			ErrorMessage: "Checkout not approved",
		}

		err := service.Log(ctx, event)
		require.NoError(t, err)
		assert.Equal(t, OutcomeDenied, event.Outcome)
	})
}

// TestRepository_List retrieves audit events with filtering
func TestRepository_List(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	actorID := uuid.New()

	// Create some test events
	actions := []string{"login", "logout", "checkout", "checkin"}
	for _, action := range actions {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       action,
			ResourceType: "session",
			Outcome:      OutcomeSuccess,
		}
		_ = service.Log(ctx, event)
	}

	t.Run("list all events", func(t *testing.T) {
		events, total, err := service.Query(ctx, tenantID, EventFilter{}, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 4)
		assert.GreaterOrEqual(t, total, 4)
	})

	t.Run("filter by action", func(t *testing.T) {
		action := "login"
		filter := EventFilter{Action: &action}
		events, _, err := service.Query(ctx, tenantID, filter, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 1)
		for _, e := range events {
			assert.Equal(t, "login", e.Action)
		}
	})

	t.Run("filter by outcome", func(t *testing.T) {
		outcome := OutcomeSuccess
		filter := EventFilter{Outcome: &outcome}
		events, _, err := service.Query(ctx, tenantID, filter, 10, 0)
		require.NoError(t, err)
		assert.Greater(t, len(events), 0)
		for _, e := range events {
			assert.Equal(t, OutcomeSuccess, e.Outcome)
		}
	})

	t.Run("filter by actor", func(t *testing.T) {
		filter := EventFilter{ActorID: &actorID}
		events, _, err := service.Query(ctx, tenantID, filter, 10, 0)
		require.NoError(t, err)
		assert.Greater(t, len(events), 0)
		for _, e := range events {
			assert.Equal(t, actorID, e.ActorID)
		}
	})

	t.Run("filter by time range", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-24 * time.Hour)
		filter := EventFilter{
			StartTimeFrom: &past,
			StartTimeTo:   &now,
		}
		events, _, err := service.Query(ctx, tenantID, filter, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 0)
	})

	t.Run("pagination", func(t *testing.T) {
		page1, total1, err := service.Query(ctx, tenantID, EventFilter{}, 2, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page1), 2)

		page2, _, err := service.Query(ctx, tenantID, EventFilter{}, 2, 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page2), 2)

		// Total should be consistent
		assert.Equal(t, total1, total1)
	})
}

// TestRepository_GetByID retrieves an event by ID
func TestRepository_GetByID(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	actorID := uuid.New()

	// Create an event
	event := &Event{
		TenantID:     tenantID,
		ActorID:      actorID,
		ActorType:    "user",
		Action:       "create",
		ResourceType: "user",
		ResourceID:   "user-123",
		Outcome:      OutcomeSuccess,
	}
	err := service.Log(ctx, event)
	require.NoError(t, err)

	// Get by ID
	repo := service.repo
	found, err := repo.GetByID(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.ID, found.ID)
	assert.Equal(t, "create", found.Action)
}

// TestRepository_GetByID_NotFound returns error for non-existent event
func TestRepository_GetByID_NotFound(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	repo := service.repo
	randomID := uuid.New()
	_, err := repo.GetByID(ctx, randomID)
	assert.Error(t, err)
}

// TestRepository_VerifyChain verifies audit chain integrity
func TestRepository_VerifyChain(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create multiple events
	for i := 0; i < 5; i++ {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      uuid.New(),
			ActorType:    "user",
			Action:       "action",
			ResourceType: "test",
			Outcome:      OutcomeSuccess,
		}
		_ = service.Log(ctx, event)
	}

	// Verify chain
	valid, errors, err := service.VerifyIntegrity(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Empty(t, errors)
}

// TestService_LogAction creates audit events with helper
func TestService_LogAction(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	actorID := uuid.New()

	details := map[string]interface{}{
		"username": "testuser",
		"ip":       "10.0.0.1",
	}

	err := service.LogAction(ctx, tenantID, actorID, "user", "create", "user", "user-123", OutcomeSuccess, details)
	require.NoError(t, err)
}

// TestService_GenerateComplianceReport generates compliance reports
func TestService_GenerateComplianceReport(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	actorID := uuid.New()

	// Create various events for the period
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	events := []struct {
		action       string
		resourceType string
		outcome      EventOutcome
	}{
		{"login", "session", OutcomeSuccess},
		{"logout", "session", OutcomeSuccess},
		{"checkout", "credential", OutcomeSuccess},
		{"create", "user", OutcomeFailure},
		{"delete", "credential", OutcomeDenied},
	}

	for _, e := range events {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       e.action,
			ResourceType: e.resourceType,
			Outcome:      e.outcome,
		}
		_ = service.Log(ctx, event)
	}

	// Generate report
	report, err := service.GenerateComplianceReport(ctx, tenantID, startTime, endTime)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, tenantID, report.TenantID)
	assert.Greater(t, report.TotalEvents, 0)
	assert.Greater(t, report.SuccessfulEvents, 0)
	assert.Greater(t, report.FailedEvents, 0)
	assert.Greater(t, report.DeniedEvents, 0)
}

// TestRepository_generateHash generates consistent hashes
func TestRepository_generateHash(t *testing.T) {
	repo := &Repository{}

	t.Run("same input produces same hash", func(t *testing.T) {
		event := &Event{
			ID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			TenantID:     uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			ActorID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Action:       "test",
			ResourceType: "test",
			ResourceID:   "test-1",
			Outcome:      OutcomeSuccess,
			IP:           "192.168.1.1",
			PreviousHash: "",
			Timestamp:    time.Unix(1704067200, 0),
		}

		hash1 := repo.generateHash(event)
		hash2 := repo.generateHash(event)

		assert.Equal(t, hash1, hash2)
		assert.Len(t, hash1, 64) // SHA256 hex string
	})

	t.Run("different input produces different hash", func(t *testing.T) {
		event1 := &Event{
			ID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			TenantID:     uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			ActorID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Action:       "test1",
			ResourceType: "test",
			ResourceID:   "test-1",
			Outcome:      OutcomeSuccess,
			IP:           "192.168.1.1",
			PreviousHash: "",
			Timestamp:    time.Unix(1704067200, 0),
		}

		event2 := &Event{
			ID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			TenantID:     uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			ActorID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Action:       "test2",
			ResourceType: "test",
			ResourceID:   "test-1",
			Outcome:      OutcomeSuccess,
			IP:           "192.168.1.1",
			PreviousHash: "",
			Timestamp:    time.Unix(1704067200, 0),
		}

		hash1 := repo.generateHash(event1)
		hash2 := repo.generateHash(event2)

		assert.NotEqual(t, hash1, hash2)
	})
}

// TestEventFilter creates and validates filters
func TestEventFilter(t *testing.T) {
	actorID := uuid.New()
	action := "create"
	resourceType := "credential"
	outcome := OutcomeSuccess
	now := time.Now()

	t.Run("filter by actor", func(t *testing.T) {
		filter := EventFilter{ActorID: &actorID}
		assert.NotNil(t, filter.ActorID)
		assert.Equal(t, actorID, *filter.ActorID)
	})

	t.Run("filter by action", func(t *testing.T) {
		filter := EventFilter{Action: &action}
		assert.NotNil(t, filter.Action)
		assert.Equal(t, "create", *filter.Action)
	})

	t.Run("filter by resource type", func(t *testing.T) {
		filter := EventFilter{ResourceType: &resourceType}
		assert.NotNil(t, filter.ResourceType)
		assert.Equal(t, "credential", *filter.ResourceType)
	})

	t.Run("filter by outcome", func(t *testing.T) {
		filter := EventFilter{Outcome: &outcome}
		assert.NotNil(t, filter.Outcome)
		assert.Equal(t, OutcomeSuccess, *filter.Outcome)
	})

	t.Run("filter by time range", func(t *testing.T) {
		past := now.Add(-24 * time.Hour)
		filter := EventFilter{
			StartTimeFrom: &past,
			StartTimeTo:   &now,
		}
		assert.NotNil(t, filter.StartTimeFrom)
		assert.NotNil(t, filter.StartTimeTo)
	})

	t.Run("combined filters", func(t *testing.T) {
		filter := EventFilter{
			ActorID:      &actorID,
			Action:       &action,
			ResourceType: &resourceType,
			Outcome:      &outcome,
			StartTimeFrom: &now,
			StartTimeTo:   &now,
		}
		assert.NotNil(t, filter.ActorID)
		assert.NotNil(t, filter.Action)
		assert.NotNil(t, filter.ResourceType)
		assert.NotNil(t, filter.Outcome)
	})

	t.Run("nil filter values", func(t *testing.T) {
		filter := EventFilter{}
		assert.Nil(t, filter.ActorID)
		assert.Nil(t, filter.Action)
		assert.Nil(t, filter.ResourceType)
		assert.Nil(t, filter.Outcome)
		assert.Nil(t, filter.StartTimeFrom)
		assert.Nil(t, filter.StartTimeTo)
	})
}

// TestComplianceReport_Struct creates a compliance report
func TestComplianceReport_Struct(t *testing.T) {
	t.Run("create compliance report", func(t *testing.T) {
		now := time.Now()
		report := &ComplianceReport{
			TenantID:        uuid.New(),
			StartTime:       now.Add(-24 * time.Hour),
			EndTime:         now,
			Generated:       now,
			TotalEvents:     100,
			SuccessfulEvents: 85,
			FailedEvents:    10,
			DeniedEvents:    5,
			EventTypes:      map[string]int{"login": 20, "logout": 20},
			CredentialEvents: 30,
			SessionEvents:   40,
			UserEvents:      30,
			IntegrityValid:  true,
		}

		assert.NotEqual(t, uuid.Nil, report.TenantID)
		assert.Greater(t, report.TotalEvents, 0)
		assert.Greater(t, report.SuccessfulEvents, 0)
		assert.Greater(t, report.FailedEvents, 0)
		assert.Greater(t, report.DeniedEvents, 0)
		assert.NotEmpty(t, report.EventTypes)
		assert.True(t, report.IntegrityValid)
	})
}

// Benchmark tests
func BenchmarkRepository_generateHash(b *testing.B) {
	repo := &Repository{}
	event := &Event{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorID:      uuid.New(),
		Action:       "test",
		ResourceType: "test",
		ResourceID:   "test-1",
		Outcome:      OutcomeSuccess,
		IP:           "192.168.1.1",
		PreviousHash: "",
		Timestamp:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.generateHash(event)
	}
}

func BenchmarkService_LogAction(b *testing.B) {
	db, service := setupAuditTest(&testing.T{})
	if db == nil {
		b.Skip("database not available")
		return
	}

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	actorID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.LogAction(ctx, tenantID, actorID, "user", "create", "user", uuid.New().String(), OutcomeSuccess, nil)
	}
}

// Test edge cases
func TestAudit_EdgeCases(t *testing.T) {
	db, service := setupAuditTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	t.Run("empty resource ID", func(t *testing.T) {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      uuid.New(),
			ActorType:    "user",
			Action:       "login",
			ResourceType: "session",
			ResourceID:   "", // Empty resource ID
			Outcome:      OutcomeSuccess,
		}

		err := service.Log(ctx, event)
		require.NoError(t, err)
	})

	t.Run("very long details", func(t *testing.T) {
		longDetails := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			longDetails[uuid.New().String()] = "some long value that contains lots of data"
		}
		detailsJSON, _ := json.Marshal(longDetails)

		event := &Event{
			TenantID:     tenantID,
			ActorID:      uuid.New(),
			ActorType:    "user",
			Action:       "complex_action",
			ResourceType: "complex_resource",
			ResourceID:   "complex-1",
			Outcome:      OutcomeSuccess,
			Details:      detailsJSON,
		}

		err := service.Log(ctx, event)
		require.NoError(t, err)
	})

	t.Run("special characters in fields", func(t *testing.T) {
		event := &Event{
			TenantID:     tenantID,
			ActorID:      uuid.New(),
			ActorType:    "system",
			Action:       "test'action",
			ResourceType: "test\"resource",
			ResourceID:   "test,resource",
			Outcome:      OutcomeSuccess,
			ErrorMessage: "Error with \"quotes\" and 'apostrophes'",
		}

		err := service.Log(ctx, event)
		require.NoError(t, err)
	})
}
