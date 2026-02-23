package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRDPSession_Struct(t *testing.T) {
	t.Run("RDP session with all fields", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		credentialID := uuid.New()
		now := time.Now()

		session := &RDPSession{
			ID:           id,
			UserID:       userID,
			CredentialID: credentialID,
			TargetHost:   "windows.example.com",
			TargetPort:   3389,
			StartTime:    now,
			LastActivity: now,
			closed:       false,
		}

		assert.Equal(t, id, session.ID)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, credentialID, session.CredentialID)
		assert.Equal(t, "windows.example.com", session.TargetHost)
		assert.Equal(t, 3389, session.TargetPort)
		assert.False(t, session.closed)
		assert.False(t, session.StartTime.IsZero())
		assert.False(t, session.LastActivity.IsZero())
	})
}

func TestRDPFrame_Struct(t *testing.T) {
	t.Run("RDP frame with all fields", func(t *testing.T) {
		now := time.Now()

		frame := RDPFrame{
			Timestamp: now,
			Width:     1920,
			Height:    1080,
			Data:      []byte("test-frame-data"),
			Changes: []RDPChange{
				{
					X:      0,
					Y:      0,
					Width:  100,
					Height: 100,
					Data:   []byte("change-data"),
				},
			},
		}

		assert.Equal(t, 1920, frame.Width)
		assert.Equal(t, 1080, frame.Height)
		assert.NotEmpty(t, frame.Data)
		assert.Len(t, frame.Changes, 1)
		assert.False(t, frame.Timestamp.IsZero())
	})
}

func TestRDPChange_Struct(t *testing.T) {
	t.Run("RDP change region", func(t *testing.T) {
		change := RDPChange{
			X:      100,
			Y:      200,
			Width:  300,
			Height: 400,
			Data:   []byte("changed-region"),
		}

		assert.Equal(t, 100, change.X)
		assert.Equal(t, 200, change.Y)
		assert.Equal(t, 300, change.Width)
		assert.Equal(t, 400, change.Height)
		assert.NotEmpty(t, change.Data)
	})
}

func TestRDPFrameBuffer_Struct(t *testing.T) {
	t.Run("frame buffer initialization", func(t *testing.T) {
		sessionID := uuid.New()
		fb := NewRDPFrameBuffer(sessionID)

		assert.NotNil(t, fb)
		assert.Equal(t, sessionID, fb.sessionID)
		assert.Empty(t, fb.frames)
	})

	t.Run("add frames to buffer", func(t *testing.T) {
		sessionID := uuid.New()
		fb := NewRDPFrameBuffer(sessionID)

		frame1 := RDPFrame{
			Timestamp: time.Now(),
			Width:     1920,
			Height:    1080,
			Data:      []byte("frame1"),
		}

		frame2 := RDPFrame{
			Timestamp: time.Now(),
			Width:     1920,
			Height:    1080,
			Data:      []byte("frame2"),
		}

		fb.AddFrame(frame1)
		fb.AddFrame(frame2)

		assert.Len(t, fb.frames, 2)
	})
}

func TestRDPSession_RecordFrame(t *testing.T) {
	t.Run("record frame without recording", func(t *testing.T) {
		session := &RDPSession{
			ID:        uuid.New(),
			Recording: nil,
		}

		// Should not panic
		data := []byte("frame-data")
		session.RecordFrame(data)

		assert.Nil(t, session.Recording)
	})
}

func TestNewRDPProxy(t *testing.T) {
	t.Run("creates proxy with nil dependencies", func(t *testing.T) {
		var c *cache.Cache // nil for testing
		var pub *events.Publisher
		logger := zerolog.Nop()

		proxy := NewRDPProxy(c, pub, logger)

		assert.NotNil(t, proxy)
	})
}

func TestRDPProxy_GetActiveSessions(t *testing.T) {
	t.Run("returns empty list", func(t *testing.T) {
		var c *cache.Cache
		var pub *events.Publisher
		logger := zerolog.Nop()
		proxy := NewRDPProxy(c, pub, logger)

		ctx := context.Background()
		sessions, err := proxy.GetActiveSessions(ctx)

		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestVideoEncoder_Struct(t *testing.T) {
	t.Run("video encoder initialization", func(t *testing.T) {
		sessionID := uuid.New()
		encoder := &VideoEncoder{
			sessionID: sessionID,
			fps:       15,
			bitrate:   1000,
		}

		assert.Equal(t, sessionID, encoder.sessionID)
		assert.Equal(t, 15, encoder.fps)
		assert.Equal(t, 1000, encoder.bitrate)
	})
}

func TestRDPRecording_Struct(t *testing.T) {
	t.Run("RDP recording components", func(t *testing.T) {
		sessionID := uuid.New()

		recording := &RDPRecording{
			SessionID: sessionID,
		}

		assert.Equal(t, sessionID, recording.SessionID)
	})
}
