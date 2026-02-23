package proxy

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSession_Struct(t *testing.T) {
	t.Run("session with all fields", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		credentialID := uuid.New()
		now := time.Now()

		session := &Session{
			ID:           id,
			UserID:       userID,
			CredentialID: credentialID,
			TargetHost:   "server.example.com",
			TargetPort:   22,
			StartTime:    now,
			LastActivity: now,
		}

		assert.Equal(t, id, session.ID)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, credentialID, session.CredentialID)
		assert.Equal(t, "server.example.com", session.TargetHost)
		assert.Equal(t, 22, session.TargetPort)
		assert.False(t, session.StartTime.IsZero())
		assert.False(t, session.LastActivity.IsZero())
	})
}

func TestNewSSHProxy(t *testing.T) {
	t.Run("creates SSH proxy with nil dependencies", func(t *testing.T) {
		var c *cache.Cache // nil for testing
		var pub *events.Publisher
		logger := zerolog.Nop()

		proxy := NewSSHProxy(c, pub, logger)

		assert.NotNil(t, proxy)
	})
}

func TestSSHProxy_GetActiveSessions(t *testing.T) {
	t.Run("returns empty list with nil cache", func(t *testing.T) {
		var c *cache.Cache
		var pub *events.Publisher
		logger := zerolog.Nop()

		proxy := NewSSHProxy(c, pub, logger)

		sessions, err := proxy.GetActiveSessions(nil)

		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestKeystrokeEvent_Struct(t *testing.T) {
	t.Run("keystroke event with all fields", func(t *testing.T) {
		now := time.Now()

		event := KeystrokeEvent{
			Timestamp: now,
			Key:       "ls -la",
			Type:      "command",
		}

		assert.False(t, event.Timestamp.IsZero())
		assert.Equal(t, "ls -la", event.Key)
		assert.Equal(t, "command", event.Type)
	})
}

func TestSessionRecording_Struct(t *testing.T) {
	t.Run("session recording components", func(t *testing.T) {
		sessionID := uuid.New()

		recording := &SessionRecording{
			SessionID: sessionID,
		}

		assert.Equal(t, sessionID, recording.SessionID)
	})
}

func TestSSHOutputWriter_Struct(t *testing.T) {
	t.Run("SSH output writer initialization", func(t *testing.T) {
		sessionID := uuid.New()
		storage := &S3RecordingStorage{}

		writer := NewSSHOutputWriter(sessionID, storage)

		assert.NotNil(t, writer)
		assert.Equal(t, sessionID, writer.sessionID)
		assert.NotNil(t, writer.buffer)
	})
}

func TestKeystrokeRecorder_Struct(t *testing.T) {
	t.Run("keystroke recorder initialization", func(t *testing.T) {
		sessionID := uuid.New()
		storage := &S3RecordingStorage{}

		recorder := NewKeystrokeRecorder(sessionID, storage)

		assert.NotNil(t, recorder)
		assert.Equal(t, sessionID, recorder.sessionID)
		assert.NotNil(t, recorder.keystrokes)
		assert.Empty(t, recorder.keystrokes)
	})
}

func TestKeystrokeRecorder_Record(t *testing.T) {
	t.Run("record keystroke adds to list", func(t *testing.T) {
		sessionID := uuid.New()
		storage := &S3RecordingStorage{}
		recorder := NewKeystrokeRecorder(sessionID, storage)

		recorder.Record("ls")
		recorder.Record("cd /tmp")

		assert.Len(t, recorder.keystrokes, 2)
	})
}

func TestS3RecordingStorage_Struct(t *testing.T) {
	t.Run("S3 storage initialization", func(t *testing.T) {
		storage := &S3RecordingStorage{
			bucket: "test-bucket",
		}

		assert.Equal(t, "test-bucket", storage.bucket)
	})

	t.Run("S3 storage SaveOutput does not panic", func(t *testing.T) {
		storage := &S3RecordingStorage{}

		err := storage.SaveOutput(nil, uuid.New(), []byte("test"))

		assert.NoError(t, err)
	})

	t.Run("S3 storage SaveKeystroke does not panic", func(t *testing.T) {
		storage := &S3RecordingStorage{}
		event := KeystrokeEvent{
			Timestamp: time.Now(),
			Key:       "test",
			Type:      "key",
		}

		err := storage.SaveKeystroke(nil, uuid.New(), event)

		assert.NoError(t, err)
	})

	t.Run("S3 storage Finalize does not panic", func(t *testing.T) {
		storage := &S3RecordingStorage{}

		err := storage.Finalize(nil, uuid.New())

		assert.NoError(t, err)
	})
}

func TestSSHOutputWriter_Write(t *testing.T) {
	t.Run("write adds to buffer", func(t *testing.T) {
		sessionID := uuid.New()
		storage := &S3RecordingStorage{}
		writer := NewSSHOutputWriter(sessionID, storage)

		n, err := writer.Write([]byte("test data"))

		assert.NoError(t, err)
		assert.Equal(t, 9, n)
		assert.Len(t, writer.buffer, 9)
	})
}
