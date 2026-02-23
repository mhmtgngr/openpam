package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_MarshalUnmarshal(t *testing.T) {
	event := Event{
		ID:       uuid.New().String(),
		Type:     EventTypeSessionStarted,
		TenantID: "tenant-123",
		ActorID:  "user-456",
		Action:   "start",
		Resource: "session",
		Data: map[string]interface{}{
			"session_id":  "sess-789",
			"target_host": "example.com",
			"target_port": 22,
		},
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"ip": "192.168.1.1",
		},
	}

	t.Run("marshal event to JSON", func(t *testing.T) {
		data, err := json.Marshal(event)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("unmarshal event from JSON", func(t *testing.T) {
		data, err := json.Marshal(event)
		require.NoError(t, err)

		var unmarshaled Event
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, event.ID, unmarshaled.ID)
		assert.Equal(t, event.Type, unmarshaled.Type)
		assert.Equal(t, event.TenantID, unmarshaled.TenantID)
	})
}

func TestEvent_Constants(t *testing.T) {
	t.Run("session event types", func(t *testing.T) {
		assert.Equal(t, "session.started", EventTypeSessionStarted)
		assert.Equal(t, "session.ended", EventTypeSessionEnded)
		assert.Equal(t, "session.terminated", EventTypeSessionTerminated)
	})

	t.Run("credential event types", func(t *testing.T) {
		assert.Equal(t, "credential.created", EventTypeCredentialCreated)
		assert.Equal(t, "credential.updated", EventTypeCredentialUpdated)
		assert.Equal(t, "credential.deleted", EventTypeCredentialDeleted)
	})

	t.Run("checkout event types", func(t *testing.T) {
		assert.Equal(t, "checkout.requested", EventTypeCheckoutRequested)
		assert.Equal(t, "checkout.approved", EventTypeCheckoutApproved)
		assert.Equal(t, "checkout.denied", EventTypeCheckoutDenied)
		assert.Equal(t, "checkout.checked_out", EventTypeCheckoutCheckedOut)
		assert.Equal(t, "checkout.checked_in", EventTypeCheckoutCheckedIn)
	})

	t.Run("user event types", func(t *testing.T) {
		assert.Equal(t, "user.created", EventTypeUserCreated)
		assert.Equal(t, "user.updated", EventTypeUserUpdated)
		assert.Equal(t, "user.deleted", EventTypeUserDeleted)
	})

	t.Run("alert event types", func(t *testing.T) {
		assert.Equal(t, "alert.triggered", EventTypeAlertTriggered)
		assert.Equal(t, "anomaly.detected", EventTypeAnomalyDetected)
	})
}

func TestEvent_Struct(t *testing.T) {
	event := Event{
		ID:        uuid.New().String(),
		Type:      EventTypeSessionStarted,
		TenantID:  "tenant-123",
		ActorID:   "user-456",
		Action:    "start",
		Resource:  "session",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, EventTypeSessionStarted, event.Type)
	assert.Equal(t, "tenant-123", event.TenantID)
}

func BenchmarkEvent_Marshal(b *testing.B) {
	event := Event{
		ID:        uuid.New().String(),
		Type:      EventTypeSessionStarted,
		TenantID:  "tenant-123",
		ActorID:   "user-456",
		Action:    "start",
		Resource:  "session",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(event)
	}
}
