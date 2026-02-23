package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// RDPProxy handles RDP session proxying
type RDPProxy struct {
	cache         *cache.Cache
	publisher     *events.Publisher
	logger        zerolog.Logger
	dialTimeout   time.Duration
	sessionTimeout time.Duration
}

// NewRDPProxy creates a new RDP proxy
func NewRDPProxy(c *cache.Cache, publisher *events.Publisher, logger zerolog.Logger) *RDPProxy {
	return &RDPProxy{
		cache:         c,
		publisher:     publisher,
		logger:        logger,
		dialTimeout:   10 * time.Second,
		sessionTimeout: 24 * time.Hour,
	}
}

// RDPSession represents an active RDP session
type RDPSession struct {
	ID           uuid.UUID      `json:"id"`
	UserID       uuid.UUID      `json:"user_id"`
	CredentialID uuid.UUID      `json:"credential_id"`
	TargetHost   string         `json:"target_host"`
	TargetPort   int            `json:"target_port"`
	RDPConn      net.Conn       `json:"-"`
	WebSocket    *websocket.Conn `json:"-"`
	StartTime    time.Time      `json:"start_time"`
	LastActivity time.Time      `json:"last_activity"`
	Recording    *RDPRecording  `json:"-"`
	mu           sync.RWMutex
	closed       bool
}

// RDPRecording handles RDP session recording (video capture)
type RDPRecording struct {
	SessionID   uuid.UUID
	FrameBuffer *RDPFrameBuffer
	Storage     RecordingStorage
	Encoder     *VideoEncoder
}

// RDPFrameBuffer captures RDP frames
type RDPFrameBuffer struct {
	sessionID uuid.UUID
	frames    []RDPFrame
	mu        sync.Mutex
}

// RDPFrame represents a single RDP frame
type RDPFrame struct {
	Timestamp time.Time
	Width     int
	Height    int
	Data      []byte // Encoded frame data
	Changes   []RDPChange
}

// RDPChange represents a screen change region
type RDPChange struct {
	X      int
	Y      int
	Width  int
	Height int
	Data   []byte
}

// VideoEncoder encodes frames to video
type VideoEncoder struct {
	sessionID uuid.UUID
	fps       int
	bitrate   int
}

// HandleWebSocketConnection handles a WebSocket connection for RDP
func (p *RDPProxy) HandleWebSocketConnection(ctx context.Context, conn *websocket.Conn, userID, credentialID uuid.UUID, targetHost string, targetPort int) (*RDPSession, error) {
	// Generate session ID
	sessionID := uuid.New()

	// Create RDP connection
	rdpConn, err := p.connectRDP(ctx, credentialID, targetHost, targetPort)
	if err != nil {
		return nil, fmt.Errorf("rdp.Connect: %w", err)
	}

	// Create session
	session := &RDPSession{
		ID:           sessionID,
		UserID:       userID,
		CredentialID: credentialID,
		TargetHost:   targetHost,
		TargetPort:   targetPort,
		RDPConn:      rdpConn,
		WebSocket:    conn,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}

	// Setup recording
	recording, err := p.setupRecording(ctx, sessionID)
	if err != nil {
		rdpConn.Close()
		return nil, fmt.Errorf("rdp.SetupRecording: %w", err)
	}
	session.Recording = recording

	// Cache session
	cacheKey := fmt.Sprintf("rdp_session:%s", sessionID)
	_ = p.cache.Set(ctx, cacheKey, session, p.sessionTimeout)

	// Start bidirectional forwarding
	go p.forwardToRDP(ctx, session)
	go p.forwardFromRDP(ctx, session)

	// Publish session started event
	_ = p.publisher.PublishSessionStarted(ctx, "", userID.String(), sessionID.String(), targetHost, targetPort)

	p.logger.Info().
		Str("session_id", sessionID.String()).
		Str("user_id", userID.String()).
		Str("target", fmt.Sprintf("%s:%d", targetHost, targetPort)).
		Msg("RDP session started")

	return session, nil
}

// connectRDP establishes an RDP connection
func (p *RDPProxy) connectRDP(ctx context.Context, credentialID uuid.UUID, host string, port int) (net.Conn, error) {
	// RDP connection setup
	// In production, this would:
	// 1. Retrieve credentials from vault
	// 2. Perform RDP handshake (X.224, MCS, etc.)
	// 3. Establish TLS tunnel for secure RDP

	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, p.dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("rdp.Dial: %w", err)
	}

	// Send RDP initial connection request
	// This is simplified; actual RDP protocol is much more complex
	// You'd typically use a library like go-rdp

	return conn, nil
}

// forwardToRDP forwards WebSocket messages to RDP connection
func (p *RDPProxy) forwardToRDP(ctx context.Context, session *RDPSession) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error().Interface("panic", r).Msg("Panic in forwardToRDP")
		}
		p.TerminateSession(ctx, session.ID, session.UserID, "client disconnect")
	}()

	for {
		_, message, err := session.WebSocket.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				p.logger.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		session.mu.Lock()
		session.LastActivity = time.Now()
		session.mu.Unlock()

		// Parse RDP message (could be input events, clipboard, etc.)
		// For now, forward raw data

		if _, err := session.RDPConn.Write(message); err != nil {
			p.logger.Error().Err(err).Msg("RDP write error")
			break
		}
	}
}

// forwardFromRDP forwards RDP connection data to WebSocket
func (p *RDPProxy) forwardFromRDP(ctx context.Context, session *RDPSession) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error().Interface("panic", r).Msg("Panic in forwardFromRDP")
		}
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := session.RDPConn.Read(buf)
		if err != nil {
			if err != io.EOF {
				p.logger.Error().Err(err).Msg("RDP read error")
			}
			break
		}

		if n > 0 {
			// Send to WebSocket
			if err := session.WebSocket.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				p.logger.Error().Err(err).Msg("WebSocket write error")
				break
			}

			// Record frame if this is graphics data
			if session.Recording != nil {
				session.RecordFrame(buf[:n])
			}
		}
	}
}

// RecordFrame records a video frame
func (s *RDPSession) RecordFrame(data []byte) {
	if s.Recording != nil && s.Recording.FrameBuffer != nil {
		s.Recording.FrameBuffer.AddFrame(RDPFrame{
			Timestamp: time.Now(),
			Data:      data,
		})
	}
}

// BroadcastFrame broadcasts a frame to monitoring sessions
func (p *RDPProxy) BroadcastFrame(ctx context.Context, sessionID uuid.UUID, frame []byte) {
	channel := fmt.Sprintf("session:%s:frames", sessionID)
	_ = p.cache.PubSub().Publish(ctx, channel, cache.Event{
		Type: "frame",
		Data: map[string]interface{}{
			"frame": frame,
		},
	})
}

// TerminateSession terminates an active RDP session
func (p *RDPProxy) TerminateSession(ctx context.Context, sessionID uuid.UUID, terminatedBy uuid.UUID, reason string) error {
	// Get session from cache
	cacheKey := fmt.Sprintf("rdp_session:%s", sessionID)
	var session RDPSession
	if err := p.cache.Get(ctx, cacheKey, &session); err != nil {
		return fmt.Errorf("rdp.SessionNotFound: %w", err)
	}

	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return fmt.Errorf("rdp: session already closed")
	}
	session.closed = true
	session.mu.Unlock()

	// Close RDP connection
	if session.RDPConn != nil {
		_ = session.RDPConn.Close()
	}

	// Close WebSocket
	if session.WebSocket != nil {
		_ = session.WebSocket.Close()
	}

	// Finalize recording
	if session.Recording != nil {
		_ = p.finalizeRecording(ctx, session.Recording)
	}

	// Remove from cache
	_ = p.cache.Delete(ctx, cacheKey)

	duration := time.Since(session.StartTime)
	_ = p.publisher.PublishSessionEnded(ctx, "", terminatedBy.String(), sessionID.String(), duration)

	p.logger.Info().
		Str("session_id", sessionID.String()).
		Str("terminated_by", terminatedBy.String()).
		Str("reason", reason).
		Dur("duration", duration).
		Msg("RDP session terminated")

	return nil
}

// setupRecording sets up RDP session recording
func (p *RDPProxy) setupRecording(ctx context.Context, sessionID uuid.UUID) (*RDPRecording, error) {
	storage := &S3RecordingStorage{
		bucket: "session-recordings",
	}

	return &RDPRecording{
		SessionID:   sessionID,
		FrameBuffer: NewRDPFrameBuffer(sessionID),
		Storage:     storage,
		Encoder:     &VideoEncoder{sessionID: sessionID, fps: 15, bitrate: 1000},
	}, nil
}

// finalizeRecording finalizes and uploads the recording
func (p *RDPProxy) finalizeRecording(ctx context.Context, recording *RDPRecording) error {
	if recording.Storage != nil {
		_ = recording.Storage.Finalize(ctx, recording.SessionID)
	}

	// Encode frames to video
	if recording.Encoder != nil && recording.FrameBuffer != nil {
		// Encode and upload
	}

	return nil
}

// NewRDPFrameBuffer creates a new RDP frame buffer
func NewRDPFrameBuffer(sessionID uuid.UUID) *RDPFrameBuffer {
	return &RDPFrameBuffer{
		sessionID: sessionID,
		frames:    make([]RDPFrame, 0),
	}
}

// AddFrame adds a frame to the buffer
func (fb *RDPFrameBuffer) AddFrame(frame RDPFrame) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	fb.frames = append(fb.frames, frame)

	// Flush buffer periodically
	if len(fb.frames) > 900 { // 60 seconds at 15 fps
		go fb.flush(context.Background())
	}
}

func (fb *RDPFrameBuffer) flush(ctx context.Context) {
	fb.mu.Lock()
	frames := make([]RDPFrame, len(fb.frames))
	copy(frames, fb.frames)
	fb.frames = fb.frames[:0]
	fb.mu.Unlock()

	// Process frames for recording
	for range frames {
		// Upload to storage or add to encoder
		// TODO: implement frame processing
	}
}

// GetActiveSessions retrieves all active RDP sessions
func (p *RDPProxy) GetActiveSessions(ctx context.Context) ([]RDPSession, error) {
	// In production, this would query Redis or a database
	return []RDPSession{}, nil
}

// MonitorSession subscribes to session updates for monitoring
func (p *RDPProxy) MonitorSession(ctx context.Context, sessionID uuid.UUID) (<-chan []byte, error) {
	channel := fmt.Sprintf("session:%s:frames", sessionID)
	pubsub, err := p.cache.PubSub().Subscribe(ctx, channel)
	if err != nil {
		return nil, err
	}

	output := make(chan []byte, 100)

	go func() {
		defer close(output)
		defer pubsub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-pubsub.Channel():
				if msg != nil {
					output <- []byte(msg.Payload)
				}
			}
		}
	}()

	return output, nil
}
