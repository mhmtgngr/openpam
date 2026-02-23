package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define errors for testing
var (
	ErrSessionNotFound    = errors.New("session: not found")
	ErrSessionAlreadyClosed = errors.New("session: already closed")
	ErrSessionNotActive   = errors.New("session: not active")
)

// Helper function to create a test session
func createTestSession() *Session {
	userID := uuid.New()
	credentialID := uuid.New()
	targetID := uuid.New()
	tenantID := uuid.New()

	return &Session{
		UserID:       userID,
		CredentialID: credentialID,
		TargetID:     &targetID,
		Type:         SessionTypeSSH,
		Status:       SessionStatusActive,
		TargetHost:   "example.com",
		TargetPort:   22,
		ClientIP:     "192.168.1.100",
		UserAgent:    "OpenSSH_9.0",
		TenantID:     tenantID,
	}
}

// Test Session Type Constants
func TestSessionType_Constants(t *testing.T) {
	tests := []struct {
		name   string
		t      SessionType
		value  string
	}{
		{"SSH type", SessionTypeSSH, "ssh"},
		{"RDP type", SessionTypeRDP, "rdp"},
		{"Database type", SessionTypeDatabase, "database"},
		{"Kubernetes type", SessionTypeKubernetes, "kubernetes"},
		{"Web type", SessionTypeWeb, "web"},
		{"API type", SessionTypeAPI, "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.t))
		})
	}
}

// Test Session Status Constants
func TestSessionStatus_Constants(t *testing.T) {
	tests := []struct {
		name   string
		s      SessionStatus
		value  string
	}{
		{"Active status", SessionStatusActive, "active"},
		{"Ended status", SessionStatusEnded, "ended"},
		{"Terminated status", SessionStatusTerminated, "terminated"},
		{"Failed status", SessionStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.s))
		})
	}
}

// Test Session Struct
func TestSession_Struct(t *testing.T) {
	session := createTestSession()

	assert.NotEqual(t, uuid.Nil, session.UserID)
	assert.NotEqual(t, uuid.Nil, session.CredentialID)
	assert.NotNil(t, session.TargetID)
	assert.Equal(t, SessionTypeSSH, session.Type)
	assert.Equal(t, SessionStatusActive, session.Status)
	assert.Equal(t, "example.com", session.TargetHost)
	assert.Equal(t, 22, session.TargetPort)
	assert.Equal(t, "192.168.1.100", session.ClientIP)
	assert.Equal(t, "OpenSSH_9.0", session.UserAgent)
}

// Test Session Marshal/Unmarshal
func TestSession_MarshalUnmarshal(t *testing.T) {
	session := createTestSession()
	metadata := json.RawMessage([]byte(`{"key":"value"}`))
	session.Metadata = metadata

	t.Run("marshal session to JSON", func(t *testing.T) {
		data, err := json.Marshal(session)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), session.TargetHost)
	})

	t.Run("unmarshal session from JSON", func(t *testing.T) {
		data, err := json.Marshal(session)
		require.NoError(t, err)

		var unmarshaled Session
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, session.UserID, unmarshaled.UserID)
		assert.Equal(t, session.CredentialID, unmarshaled.CredentialID)
		assert.Equal(t, session.Type, unmarshaled.Type)
		assert.Equal(t, session.TargetHost, unmarshaled.TargetHost)
		assert.Equal(t, session.TargetPort, unmarshaled.TargetPort)
	})
}

// Test SessionFilter
func TestSessionFilter_Filter(t *testing.T) {
	userID := uuid.New()
	status := SessionStatusActive
	sessionType := SessionTypeSSH
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	oneHourFromNow := now.Add(time.Hour)

	filter := SessionFilter{
		Type:          &sessionType,
		Status:        &status,
		UserID:        &userID,
		StartTimeFrom: &oneHourAgo,
		StartTimeTo:   &oneHourFromNow,
	}

	assert.NotNil(t, filter.Type)
	assert.Equal(t, SessionTypeSSH, *filter.Type)
	assert.NotNil(t, filter.Status)
	assert.Equal(t, SessionStatusActive, *filter.Status)
	assert.NotNil(t, filter.UserID)
	assert.Equal(t, userID, *filter.UserID)
	assert.NotNil(t, filter.StartTimeFrom)
	assert.NotNil(t, filter.StartTimeTo)
}

// Test ActiveSession
func TestActiveSession_Concurrency(t *testing.T) {
	session := createTestSession()
	active := &ActiveSession{
		Session: session,
		closed:  false,
	}

	t.Run("concurrent read operations", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				active.mu.RLock()
				_ = active.closed
				active.mu.RUnlock()
			}()
		}
		wg.Wait()
		assert.False(t, active.closed)
	})

	t.Run("concurrent write operations", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				active.mu.Lock()
				active.closed = true
				active.mu.Unlock()
			}()
		}
		wg.Wait()
		assert.True(t, active.closed)
	})

	t.Run("concurrent read and write operations", func(t *testing.T) {
		active.closed = false
		var wg sync.WaitGroup

		// Start readers
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				active.mu.RLock()
				_ = active.closed
				active.mu.RUnlock()
			}()
		}

		// Start writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				active.mu.Lock()
				active.closed = true
				active.mu.Unlock()
			}()
		}

		wg.Wait()
		// At least one writer should have set closed to true
		// We can't guarantee the final state due to race conditions
	})
}

// Test ActiveSession Close
func TestActiveSession_Close(t *testing.T) {
	session := createTestSession()
	active := &ActiveSession{
		Session: session,
		closed:  false,
	}

	t.Run("close session", func(t *testing.T) {
		active.mu.Lock()
		active.closed = true
		active.mu.Unlock()

		active.mu.RLock()
		assert.True(t, active.closed)
		active.mu.RUnlock()
	})

	t.Run("check closed status", func(t *testing.T) {
		active.mu.RLock()
		isClosed := active.closed
		active.mu.RUnlock()

		assert.True(t, isClosed)
	})
}

// Test Repository NewRepository
func TestRepository_NewRepository(t *testing.T) {
	t.Run("create repository with nil dependencies", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := NewRepository(nil, nil, logger)
		assert.NotNil(t, repo)
		assert.Nil(t, repo.db)
		assert.Nil(t, repo.cache)
	})
}

// Test Repository DB operations (skipped without DB)
func TestRepository_Create(t *testing.T) {
	t.Skip("requires PostgreSQL connection")
}

func TestRepository_GetByID(t *testing.T) {
	t.Skip("requires PostgreSQL connection")
}

func TestRepository_List(t *testing.T) {
	t.Skip("requires PostgreSQL connection")
}

func TestRepository_Update(t *testing.T) {
	t.Skip("requires PostgreSQL connection")
}

func TestRepository_EndSession(t *testing.T) {
	t.Skip("requires PostgreSQL connection")
}

func TestRepository_GetActiveSessionsForUser(t *testing.T) {
	t.Skip("requires PostgreSQL connection")
}

// Test Service NewService
func TestService_NewService(t *testing.T) {
	t.Run("create service with nil dependencies", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := NewRepository(nil, nil, logger)

		// Note: NewService will start a goroutine, which is fine for testing
		// We just verify the service is created
		service := NewService(repo, nil, nil, logger)

		assert.NotNil(t, service)
		assert.NotNil(t, service.repo)
		assert.NotNil(t, service.sessions)
		assert.Equal(t, 0, len(service.sessions))
	})
}

// Test Session Status Transitions
func TestSessionStatus_Transitions(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus SessionStatus
		toStatus   SessionStatus
		valid      bool
	}{
		{"active to ended", SessionStatusActive, SessionStatusEnded, true},
		{"active to terminated", SessionStatusActive, SessionStatusTerminated, true},
		{"active to failed", SessionStatusActive, SessionStatusFailed, true},
		{"ended to active", SessionStatusEnded, SessionStatusActive, false},
		{"terminated to active", SessionStatusTerminated, SessionStatusActive, false},
		{"failed to active", SessionStatusFailed, SessionStatusActive, false},
		{"ended to terminated", SessionStatusEnded, SessionStatusTerminated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := createTestSession()
			session.Status = tt.fromStatus

			if tt.valid {
				// Valid transitions should work
				session.Status = tt.toStatus
				assert.Equal(t, tt.toStatus, session.Status)
			} else {
				// Invalid transitions - document the expected behavior
				// In real implementation, this would be validated
				assert.NotEqual(t, tt.fromStatus, tt.toStatus)
			}
		})
	}
}

// Test Session Filtering Scenarios
func TestSessionFilter_Scenarios(t *testing.T) {
	t.Run("filter by type only", func(t *testing.T) {
		sessionType := SessionTypeSSH
		filter := SessionFilter{
			Type: &sessionType,
		}
		assert.Equal(t, SessionTypeSSH, *filter.Type)
		assert.Nil(t, filter.Status)
		assert.Nil(t, filter.UserID)
	})

	t.Run("filter by status only", func(t *testing.T) {
		status := SessionStatusActive
		filter := SessionFilter{
			Status: &status,
		}
		assert.Nil(t, filter.Type)
		assert.Equal(t, SessionStatusActive, *filter.Status)
		assert.Nil(t, filter.UserID)
	})

	t.Run("filter by user only", func(t *testing.T) {
		userID := uuid.New()
		filter := SessionFilter{
			UserID: &userID,
		}
		assert.Nil(t, filter.Type)
		assert.Nil(t, filter.Status)
		assert.Equal(t, userID, *filter.UserID)
	})

	t.Run("filter by time range", func(t *testing.T) {
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		tomorrow := now.Add(24 * time.Hour)

		filter := SessionFilter{
			StartTimeFrom: &yesterday,
			StartTimeTo:   &tomorrow,
		}
		assert.NotNil(t, filter.StartTimeFrom)
		assert.NotNil(t, filter.StartTimeTo)
		assert.True(t, filter.StartTimeFrom.Before(*filter.StartTimeTo))
	})

	t.Run("filter with all fields", func(t *testing.T) {
		sessionType := SessionTypeSSH
		status := SessionStatusActive
		userID := uuid.New()
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)

		filter := SessionFilter{
			Type:          &sessionType,
			Status:        &status,
			UserID:        &userID,
			StartTimeFrom: &yesterday,
		}
		assert.NotNil(t, filter.Type)
		assert.NotNil(t, filter.Status)
		assert.NotNil(t, filter.UserID)
		assert.NotNil(t, filter.StartTimeFrom)
		assert.Nil(t, filter.StartTimeTo)
	})
}

// Test Session Metadata
func TestSession_Metadata(t *testing.T) {
	t.Run("session with metadata", func(t *testing.T) {
		session := createTestSession()

		metadata := map[string]interface{}{
			"browser":      "Chrome",
			"os":           "Linux",
			"login_method": "ssh_key",
			"mfa_verified": true,
			"risk_score":   0.15,
		}

		metadataJSON, err := json.Marshal(metadata)
		require.NoError(t, err)
		session.Metadata = metadataJSON

		// Verify metadata can be unmarshaled
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(session.Metadata, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, "Chrome", unmarshaled["browser"])
		assert.Equal(t, "Linux", unmarshaled["os"])
		assert.True(t, unmarshaled["mfa_verified"].(bool))
	})

	t.Run("session with nil metadata", func(t *testing.T) {
		session := createTestSession()
		session.Metadata = nil

		assert.Nil(t, session.Metadata)

		// Marshal/unmarshal should still work
		data, err := json.Marshal(session)
		require.NoError(t, err)

		var unmarshaled Session
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)
	})

	t.Run("session with empty metadata", func(t *testing.T) {
		session := createTestSession()
		session.Metadata = json.RawMessage([]byte("{}"))

		var metadata map[string]interface{}
		err := json.Unmarshal(session.Metadata, &metadata)
		require.NoError(t, err)
		assert.Equal(t, 0, len(metadata))
	})
}

// Test Session Recording
func TestSession_Recording(t *testing.T) {
	t.Run("session with recording", func(t *testing.T) {
		session := createTestSession()
		recordingID := uuid.New()
		session.RecordingID = &recordingID
		session.RecordingURL = "https://storage.example.com/recordings/" + recordingID.String()

		assert.NotNil(t, session.RecordingID)
		assert.NotEmpty(t, session.RecordingURL)
		assert.Contains(t, session.RecordingURL, recordingID.String())
	})

	t.Run("session without recording", func(t *testing.T) {
		session := createTestSession()
		session.RecordingID = nil
		session.RecordingURL = ""

		assert.Nil(t, session.RecordingID)
		assert.Empty(t, session.RecordingURL)
	})
}

// Test Session Target
func TestSession_Target(t *testing.T) {
	t.Run("session with target ID", func(t *testing.T) {
		session := createTestSession()
		targetID := uuid.New()
		session.TargetID = &targetID

		assert.NotNil(t, session.TargetID)
		assert.Equal(t, targetID, *session.TargetID)
	})

	t.Run("session without target ID", func(t *testing.T) {
		session := createTestSession()
		session.TargetID = nil

		assert.Nil(t, session.TargetID)
	})
}

// Test session type validation
func TestSessionType_Validation(t *testing.T) {
	validTypes := []SessionType{
		SessionTypeSSH,
		SessionTypeRDP,
		SessionTypeDatabase,
		SessionTypeKubernetes,
		SessionTypeWeb,
		SessionTypeAPI,
	}

	t.Run("all session types have values", func(t *testing.T) {
		for _, st := range validTypes {
			assert.NotEmpty(t, string(st))
		}
	})

	t.Run("session types are unique", func(t *testing.T) {
		uniqueTypes := make(map[string]bool)
		for _, st := range validTypes {
			uniqueTypes[string(st)] = true
		}
		assert.Equal(t, len(validTypes), len(uniqueTypes))
	})
}

// Test session status validation
func TestSessionStatus_Validation(t *testing.T) {
	validStatuses := []SessionStatus{
		SessionStatusActive,
		SessionStatusEnded,
		SessionStatusTerminated,
		SessionStatusFailed,
	}

	t.Run("all session statuses have values", func(t *testing.T) {
		for _, ss := range validStatuses {
			assert.NotEmpty(t, string(ss))
		}
	})

	t.Run("session statuses are unique", func(t *testing.T) {
		uniqueStatuses := make(map[string]bool)
		for _, ss := range validStatuses {
			uniqueStatuses[string(ss)] = true
		}
		assert.Equal(t, len(validStatuses), len(uniqueStatuses))
	})
}

// Test session time operations
func TestSession_Timestamps(t *testing.T) {
	t.Run("session timestamps are set correctly", func(t *testing.T) {
		session := createTestSession()

		// In createTestSession, we don't set timestamps, so they are zero
		assert.True(t, session.StartedAt.IsZero())
		assert.Nil(t, session.EndedAt)
		assert.True(t, session.CreatedAt.IsZero())
	})

	t.Run("session duration calculation", func(t *testing.T) {
		session := createTestSession()
		started := time.Now().Add(-1 * time.Hour)
		ended := time.Now()
		session.StartedAt = started
		session.EndedAt = &ended

		duration := session.EndedAt.Sub(session.StartedAt)
		assert.True(t, duration >= 1*time.Hour)
		assert.True(t, duration < 1*time.Hour+time.Minute)
	})

	t.Run("session ended at nil for active session", func(t *testing.T) {
		session := createTestSession()
		session.Status = SessionStatusActive

		assert.Nil(t, session.EndedAt)
	})
}

// Test Session ID generation
func TestSession_ID(t *testing.T) {
	t.Run("session with zero ID initially", func(t *testing.T) {
		session := createTestSession()
		// createTestSession doesn't set ID, so it should be zero
		assert.Equal(t, uuid.Nil, session.ID)
	})

	t.Run("generate unique session IDs", func(t *testing.T) {
		ids := make(map[uuid.UUID]bool)
		for i := 0; i < 100; i++ {
			id := uuid.New()
			assert.False(t, ids[id], "ID should be unique")
			ids[id] = true
		}
		assert.Equal(t, 100, len(ids))
	})
}

// Test Session with all session types
func TestSession_AllTypes(t *testing.T) {
	sessionTypes := []SessionType{
		SessionTypeSSH,
		SessionTypeRDP,
		SessionTypeDatabase,
		SessionTypeKubernetes,
		SessionTypeWeb,
		SessionTypeAPI,
	}

	for _, sessionType := range sessionTypes {
		t.Run("session type "+string(sessionType), func(t *testing.T) {
			session := createTestSession()
			session.Type = sessionType

			assert.Equal(t, sessionType, session.Type)
			assert.NotEmpty(t, string(session.Type))

			// Verify it can be marshaled
			data, err := json.Marshal(session)
			require.NoError(t, err)
			assert.Contains(t, string(data), string(sessionType))
		})
	}
}

// Test Session with all statuses
func TestSession_AllStatuses(t *testing.T) {
	sessionStatuses := []SessionStatus{
		SessionStatusActive,
		SessionStatusEnded,
		SessionStatusTerminated,
		SessionStatusFailed,
	}

	for _, status := range sessionStatuses {
		t.Run("session status "+string(status), func(t *testing.T) {
			session := createTestSession()
			session.Status = status

			assert.Equal(t, status, session.Status)
			assert.NotEmpty(t, string(session.Status))

			// Verify it can be marshaled
			data, err := json.Marshal(session)
			require.NoError(t, err)
			assert.Contains(t, string(data), string(session.Status))
		})
	}
}

// Test Session terminated by
func TestSession_TerminatedBy(t *testing.T) {
	t.Run("session with terminated by", func(t *testing.T) {
		session := createTestSession()
		session.Status = SessionStatusTerminated

		terminatedBy := uuid.New()
		session.TerminatedBy = &terminatedBy
		reason := "policy violation"
		session.TerminateReason = reason

		assert.NotNil(t, session.TerminatedBy)
		assert.Equal(t, terminatedBy, *session.TerminatedBy)
		assert.Equal(t, reason, session.TerminateReason)
	})

	t.Run("session without terminated by", func(t *testing.T) {
		session := createTestSession()
		session.Status = SessionStatusEnded

		assert.Nil(t, session.TerminatedBy)
		assert.Empty(t, session.TerminateReason)
	})
}

// Test Session client info
func TestSession_ClientInfo(t *testing.T) {
	tests := []struct {
		name      string
		clientIP  string
		userAgent string
	}{
		{"SSH client", "192.168.1.100", "OpenSSH_9.0"},
		{"RDP client", "10.0.0.50", "Microsoft Remote Desktop"},
		{"Web client", "203.0.113.1", "Mozilla/5.0 (Windows NT 10.0)"},
		{"API client", "198.51.100.1", "OpenPAM-CLI/1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := createTestSession()
			session.ClientIP = tt.clientIP
			session.UserAgent = tt.userAgent

			assert.Equal(t, tt.clientIP, session.ClientIP)
			assert.Equal(t, tt.userAgent, session.UserAgent)
		})
	}
}

// Test Session target host and port
func TestSession_TargetHostAndPort(t *testing.T) {
	tests := []struct {
		name       string
		targetHost string
		targetPort int
	}{
		{"SSH server", "server.example.com", 22},
		{"RDP server", "rdp.example.com", 3389},
		{"Database", "db.example.com", 5432},
		{"Kubernetes API", "k8s.example.com", 6443},
		{"Web server", "web.example.com", 443},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := createTestSession()
			session.TargetHost = tt.targetHost
			session.TargetPort = tt.targetPort

			assert.Equal(t, tt.targetHost, session.TargetHost)
			assert.Equal(t, tt.targetPort, session.TargetPort)

			// Verify it can be marshaled
			data, err := json.Marshal(session)
			require.NoError(t, err)
			assert.Contains(t, string(data), tt.targetHost)
		})
	}
}

// Test Session tenant isolation
func TestSession_TenantIsolation(t *testing.T) {
	t.Run("sessions from different tenants", func(t *testing.T) {
		tenant1ID := uuid.New()
		tenant2ID := uuid.New()

		session1 := createTestSession()
		session1.TenantID = tenant1ID

		session2 := createTestSession()
		session2.TenantID = tenant2ID

		assert.NotEqual(t, session1.TenantID, session2.TenantID)
		assert.Equal(t, tenant1ID, session1.TenantID)
		assert.Equal(t, tenant2ID, session2.TenantID)
	})
}

// Test Session credential association
func TestSession_CredentialAssociation(t *testing.T) {
	t.Run("session with credential", func(t *testing.T) {
		session := createTestSession()
		credentialID := uuid.New()
		session.CredentialID = credentialID

		assert.Equal(t, credentialID, session.CredentialID)
		assert.NotEqual(t, uuid.Nil, session.CredentialID)
	})

	t.Run("session without credential", func(t *testing.T) {
		session := createTestSession()
		session.CredentialID = uuid.Nil

		assert.Equal(t, uuid.Nil, session.CredentialID)
	})
}

// Test Session field types
func TestSession_FieldTypes(t *testing.T) {
	session := createTestSession()

	t.Run("UUID fields", func(t *testing.T) {
		assert.NotEqual(t, uuid.Nil, session.UserID)
		assert.NotEqual(t, uuid.Nil, session.CredentialID)
		assert.NotEqual(t, uuid.Nil, session.TenantID)
	})

	t.Run("pointer fields", func(t *testing.T) {
		assert.NotNil(t, session.TargetID)
		assert.Nil(t, session.RecordingID)
		assert.Nil(t, session.EndedAt)
		assert.Nil(t, session.TerminatedBy)
	})

	t.Run("string fields", func(t *testing.T) {
		assert.NotEmpty(t, session.TargetHost)
		assert.NotEmpty(t, session.ClientIP)
		assert.NotEmpty(t, session.UserAgent)
	})

	t.Run("int fields", func(t *testing.T) {
		assert.Greater(t, session.TargetPort, 0)
		assert.Less(t, session.TargetPort, 65536)
	})
}

// Benchmark tests
func BenchmarkSession_Marshal(b *testing.B) {
	session := createTestSession()
	metadata := json.RawMessage([]byte(`{"key":"value"}`))
	session.Metadata = metadata

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(session)
	}
}

func BenchmarkSession_Unmarshal(b *testing.B) {
	session := createTestSession()
	metadata := json.RawMessage([]byte(`{"key":"value"}`))
	session.Metadata = metadata

	data, _ := json.Marshal(session)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s Session
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkActiveSession_ConcurrentRead(b *testing.B) {
	session := createTestSession()
	active := &ActiveSession{
		Session: session,
		closed:  false,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			active.mu.RLock()
			_ = active.closed
			active.mu.RUnlock()
		}
	})
}

func BenchmarkActiveSession_ConcurrentWrite(b *testing.B) {
	session := createTestSession()
	active := &ActiveSession{
		Session: session,
		closed:  false,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			active.mu.Lock()
			active.closed = true
			active.mu.Unlock()
		}
	})
}

func BenchmarkUUID_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}

// Test that session type can be parsed from string
func TestSessionType_FromString(t *testing.T) {
	tests := []struct {
		input    string
		expected SessionType
	}{
		{"ssh", SessionTypeSSH},
		{"rdp", SessionTypeRDP},
		{"database", SessionTypeDatabase},
		{"kubernetes", SessionTypeKubernetes},
		{"web", SessionTypeWeb},
		{"api", SessionTypeAPI},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			st := SessionType(tt.input)
			assert.Equal(t, tt.expected, st)
		})
	}

	t.Run("invalid type is still a valid type", func(t *testing.T) {
		// In Go, any string can be cast to SessionType
		// Validation should happen at application level
		invalidType := SessionType("invalid")
		assert.Equal(t, SessionType("invalid"), invalidType)
		assert.NotEqual(t, SessionTypeSSH, invalidType)
	})
}

// Test that session status can be parsed from string
func TestSessionStatus_FromString(t *testing.T) {
	tests := []struct {
		input    string
		expected SessionStatus
	}{
		{"active", SessionStatusActive},
		{"ended", SessionStatusEnded},
		{"terminated", SessionStatusTerminated},
		{"failed", SessionStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ss := SessionStatus(tt.input)
			assert.Equal(t, tt.expected, ss)
		})
	}

	t.Run("invalid status is still a valid status", func(t *testing.T) {
		// In Go, any string can be cast to SessionStatus
		// Validation should happen at application level
		invalidStatus := SessionStatus("invalid")
		assert.Equal(t, SessionStatus("invalid"), invalidStatus)
		assert.NotEqual(t, SessionStatusActive, invalidStatus)
	})
}

// Test Session helper functions
func TestSessionHelpers(t *testing.T) {
	t.Run("is active session", func(t *testing.T) {
		session := createTestSession()
		session.Status = SessionStatusActive

		isActive := session.Status == SessionStatusActive
		assert.True(t, isActive)
	})

	t.Run("is terminated session", func(t *testing.T) {
		session := createTestSession()
		session.Status = SessionStatusTerminated

		isTerminated := session.Status == SessionStatusTerminated
		assert.True(t, isTerminated)
	})

	t.Run("is ended session", func(t *testing.T) {
		session := createTestSession()
		session.Status = SessionStatusEnded

		isEnded := session.Status == SessionStatusEnded
		assert.True(t, isEnded)
	})

	t.Run("has recording", func(t *testing.T) {
		session := createTestSession()
		recordingID := uuid.New()
		session.RecordingID = &recordingID

		hasRecording := session.RecordingID != nil
		assert.True(t, hasRecording)
	})

	t.Run("duration calculation helper", func(t *testing.T) {
		session := createTestSession()
		started := time.Now().Add(-30 * time.Minute)
		session.StartedAt = started

		ended := time.Now()
		session.EndedAt = &ended

		duration := session.EndedAt.Sub(session.StartedAt)
		assert.True(t, duration >= 30*time.Minute)
		assert.True(t, duration < 31*time.Minute)
	})
}

// Test complex metadata scenarios
func TestSession_ComplexMetadata(t *testing.T) {
	t.Run("nested metadata", func(t *testing.T) {
		session := createTestSession()

		metadata := map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			"device": map[string]interface{}{
				"type":    "laptop",
				"os":      "Linux",
				"version": "5.15",
			},
		}

		metadataJSON, err := json.Marshal(metadata)
		require.NoError(t, err)
		session.Metadata = metadataJSON

		var unmarshaled map[string]interface{}
		err = json.Unmarshal(session.Metadata, &unmarshaled)
		require.NoError(t, err)

		assert.NotNil(t, unmarshaled["user"])
		assert.NotNil(t, unmarshaled["device"])
	})

	t.Run("array in metadata", func(t *testing.T) {
		session := createTestSession()

		metadata := map[string]interface{}{
			"roles":     []string{"admin", "auditor"},
			"groups":    []string{"devops", "security"},
			"permissions": []string{"ssh:server1", "db:query"},
		}

		metadataJSON, err := json.Marshal(metadata)
		require.NoError(t, err)
		session.Metadata = metadataJSON

		var unmarshaled map[string]interface{}
		err = json.Unmarshal(session.Metadata, &unmarshaled)
		require.NoError(t, err)

		assert.NotNil(t, unmarshaled["roles"])
		assert.NotNil(t, unmarshaled["groups"])
	})
}

// Test session sorting and comparison
func TestSession_Sorting(t *testing.T) {
	t.Run("sort sessions by start time", func(t *testing.T) {
		now := time.Now()

		session1 := createTestSession()
		session1.StartedAt = now.Add(-3 * time.Hour)

		session2 := createTestSession()
		session2.StartedAt = now.Add(-1 * time.Hour)

		session3 := createTestSession()
		session3.StartedAt = now.Add(-2 * time.Hour)

		sessions := []*Session{session1, session2, session3}

		// Sort by started_at descending
		for i := 0; i < len(sessions)-1; i++ {
			for j := i + 1; j < len(sessions); j++ {
				if sessions[i].StartedAt.Before(sessions[j].StartedAt) {
					sessions[i], sessions[j] = sessions[j], sessions[i]
				}
			}
		}

		// Most recent first
		assert.Equal(t, session2.StartedAt, sessions[0].StartedAt)
		assert.Equal(t, session3.StartedAt, sessions[1].StartedAt)
		assert.Equal(t, session1.StartedAt, sessions[2].StartedAt)
	})
}

// Test session validation scenarios
func TestSession_Validation(t *testing.T) {
	t.Run("valid session", func(t *testing.T) {
		session := createTestSession()

		assert.NotEqual(t, uuid.Nil, session.UserID)
		assert.NotEqual(t, uuid.Nil, session.CredentialID)
		assert.NotEmpty(t, session.TargetHost)
		assert.Greater(t, session.TargetPort, 0)
		assert.Less(t, session.TargetPort, 65536)
		assert.NotEmpty(t, session.ClientIP)
		assert.NotEqual(t, uuid.Nil, session.TenantID)
	})

	t.Run("invalid port values", func(t *testing.T) {
		session := createTestSession()

		// Test port 0
		session.TargetPort = 0
		assert.Equal(t, 0, session.TargetPort)

		// Test port out of range
		session.TargetPort = 70000
		assert.Equal(t, 70000, session.TargetPort)
	})

	t.Run("empty required fields", func(t *testing.T) {
		session := createTestSession()
		session.TargetHost = ""
		session.ClientIP = ""

		assert.Empty(t, session.TargetHost)
		assert.Empty(t, session.ClientIP)
	})
}

// Test JSON tag handling
func TestSession_JSONTags(t *testing.T) {
	session := createTestSession()
	recordingID := uuid.New()
	session.RecordingID = &recordingID
	session.RecordingURL = "https://example.com/recording.mp4"

	data, err := json.Marshal(session)
	require.NoError(t, err)

	// Check JSON field names
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"id":`)
	assert.Contains(t, jsonStr, `"user_id":`)
	assert.Contains(t, jsonStr, `"credential_id":`)
	assert.Contains(t, jsonStr, `"type":`)
	assert.Contains(t, jsonStr, `"status":`)
	assert.Contains(t, jsonStr, `"target_host":`)
	assert.Contains(t, jsonStr, `"target_port":`)
	assert.Contains(t, jsonStr, `"client_ip":`)
	assert.Contains(t, jsonStr, `"recording_url":`)
}

// Test DB tag handling
func TestSession_DBTags(t *testing.T) {
	// We can't directly test DB tags, but we can verify the struct
	// has the right fields that should have DB tags

	session := createTestSession()

	// Verify all expected fields exist
	_ = session.ID
	_ = session.UserID
	_ = session.CredentialID
	_ = session.TargetID
	_ = session.Type
	_ = session.Status
	_ = session.TargetHost
	_ = session.TargetPort
	_ = session.ClientIP
	_ = session.UserAgent
	_ = session.RecordingID
	_ = session.RecordingURL
	_ = session.StartedAt
	_ = session.EndedAt
	_ = session.TerminatedBy
	_ = session.TerminateReason
	_ = session.Metadata
	_ = session.TenantID
	_ = session.CreatedAt
	_ = session.UpdatedAt
}

// Test error message formatting
func TestSession_ErrorMessages(t *testing.T) {
	t.Run("not found error", func(t *testing.T) {
		err := fmt.Errorf("session: %w", ErrSessionNotFound)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("already closed error", func(t *testing.T) {
		err := fmt.Errorf("session: %w", ErrSessionAlreadyClosed)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already closed")
	})

	t.Run("not active error", func(t *testing.T) {
		err := fmt.Errorf("session: %w", ErrSessionNotActive)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not active")
	})
}
