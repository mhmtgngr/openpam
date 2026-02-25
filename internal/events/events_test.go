package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
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

// TestNewEventBus_Security tests security validations for event bus initialization
func TestNewEventBus_Security(t *testing.T) {
	logger := zerolog.Nop()
	emptyCache := &cache.Cache{}

	t.Run("rejects empty signing key", func(t *testing.T) {
		_, err := New(EventConfig{
			Cache:      emptyCache,
			Logger:     logger,
			SigningKey: []byte(""),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signing key is required")
		assert.Contains(t, err.Error(), "at least 32 bytes")
	})

	t.Run("rejects signing key less than 32 bytes", func(t *testing.T) {
		shortKeys := []struct {
			name string
			key  []byte
		}{
			{"1 byte", []byte("a")},
			{"16 bytes", []byte("sixteen_byte_key!!")},
			{"31 bytes", []byte("thirty_one_byte_key_!!!_one")},
		}

		for _, tc := range shortKeys {
			t.Run(tc.name, func(t *testing.T) {
				_, err := New(EventConfig{
					Cache:      emptyCache,
					Logger:     logger,
					SigningKey: tc.key,
				})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "at least 32 bytes")
			})
		}
	})

	t.Run("accepts exactly 32 byte signing key", func(t *testing.T) {
		valid32ByteKey := []byte("exactly_32_byte_signing_key_for_HMAC!")
		bus, err := New(EventConfig{
			Cache:      emptyCache,
			Logger:     logger,
			SigningKey: valid32ByteKey,
		})
		require.NoError(t, err)
		assert.NotNil(t, bus)
	})

	t.Run("accepts signing key greater than 32 bytes", func(t *testing.T) {
		valid64ByteKey := []byte("64_byte_signing_key_for_HMAC_SHA256_security__exactly_double__min!")
		bus, err := New(EventConfig{
			Cache:      emptyCache,
			Logger:     logger,
			SigningKey: valid64ByteKey,
		})
		require.NoError(t, err)
		assert.NotNil(t, bus)
	})

	t.Run("NewForTest creates valid event bus for testing", func(t *testing.T) {
		bus := NewForTest(emptyCache, logger)
		assert.NotNil(t, bus)
		assert.NotNil(t, bus.signingKey)
		assert.GreaterOrEqual(t, len(bus.signingKey), 32)
	})
}
