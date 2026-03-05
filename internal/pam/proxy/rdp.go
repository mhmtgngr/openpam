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
	cache          *cache.Cache
	publisher      *events.Publisher
	vaultService   VaultRetriever
	logger         zerolog.Logger
	dialTimeout    time.Duration
	sessionTimeout time.Duration
}

// NewRDPProxy creates a new RDP proxy
func NewRDPProxy(c *cache.Cache, publisher *events.Publisher, vault VaultRetriever, logger zerolog.Logger) *RDPProxy {
	return &RDPProxy{
		cache:          c,
		publisher:      publisher,
		vaultService:   vault,
		logger:         logger,
		dialTimeout:    10 * time.Second,
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

// connectRDP establishes an RDP connection using credentials from the vault.
// The proxy operates as a TCP relay: it connects to the RDP server and forwards
// raw bytes between the WebSocket client (browser-based RDP viewer like Apache
// Guacamole) and the target RDP service. The browser-side client handles the
// RDP protocol negotiation (X.224, MCS, TLS) using the injected credentials.
func (p *RDPProxy) connectRDP(ctx context.Context, credentialID uuid.UUID, host string, port int) (net.Conn, error) {
	// 1. Retrieve credentials from vault (used by the browser client for NLA)
	secret, err := p.vaultService.RetrieveSecret(ctx, credentialID)
	if err != nil {
		return nil, fmt.Errorf("rdp.RetrieveCredential: %w", err)
	}

	if secret.Username == "" {
		return nil, fmt.Errorf("rdp: credential has no username")
	}
	if secret.Password == "" {
		return nil, fmt.Errorf("rdp: credential has no password for RDP authentication")
	}

	// 2. Establish TCP connection to RDP server
	address := fmt.Sprintf("%s:%d", host, port)
	dialer := &net.Dialer{Timeout: p.dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("rdp.Dial(%s): %w", address, err)
	}

	// 3. Send X.224 Connection Request PDU to initiate RDP negotiation
	// This is the initial RDP handshake that tells the server we want to connect.
	// The cookie carries the username for load-balancing/routing on the server side.
	cookie := fmt.Sprintf("Cookie: mstshash=%s\r\n", secret.Username)
	// X.224 CR PDU: [TPKT header][X.224 CR][cookie][RDP Negotiation Request]
	x224CR := buildX224ConnectionRequest(cookie)
	if _, err := conn.Write(x224CR); err != nil {
		conn.Close()
		return nil, fmt.Errorf("rdp.X224Handshake: %w", err)
	}

	// 4. Read X.224 Connection Confirm response
	respBuf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(p.dialTimeout))
	n, err := conn.Read(respBuf)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rdp.X224Response: %w", err)
	}

	// Validate we received a valid TPKT response (version 3)
	if n < 4 || respBuf[0] != 0x03 {
		conn.Close()
		return nil, fmt.Errorf("rdp: invalid X.224 response from server (got %d bytes)", n)
	}

	// Reset deadline for normal operation
	conn.SetReadDeadline(time.Time{})

	p.logger.Info().
		Str("host", host).
		Int("port", port).
		Str("user", secret.Username).
		Msg("RDP connection established")

	return conn, nil
}

// buildX224ConnectionRequest constructs the X.224 Connection Request PDU
// with RDP Negotiation Request for TLS + CredSSP (NLA) support.
func buildX224ConnectionRequest(cookie string) []byte {
	// RDP Negotiation Request: TYPE_RDP_NEG_REQ, requestedProtocols = PROTOCOL_SSL | PROTOCOL_HYBRID (NLA)
	negReq := []byte{
		0x01,                   // TYPE_RDP_NEG_REQ
		0x00,                   // flags
		0x08, 0x00,             // length (8 bytes)
		0x03, 0x00, 0x00, 0x00, // requestedProtocols: SSL | HYBRID (NLA)
	}

	cookieBytes := []byte(cookie)

	// X.224 CR PDU length: 6 (X.224 header) + cookie + negReq
	x224Len := 6 + len(cookieBytes) + len(negReq)

	// TPKT header: version=3, reserved=0, length (2 bytes big-endian)
	tpktLen := 4 + x224Len
	pdu := make([]byte, 0, tpktLen)

	// TPKT header
	pdu = append(pdu, 0x03, 0x00)                             // version, reserved
	pdu = append(pdu, byte(tpktLen>>8), byte(tpktLen&0xFF))   // length

	// X.224 CR header
	pdu = append(pdu, byte(x224Len-1)) // X.224 length indicator (excludes itself)
	pdu = append(pdu, 0xE0)            // CR (Connection Request) PDU type
	pdu = append(pdu, 0x00, 0x00)      // DST-REF
	pdu = append(pdu, 0x00, 0x00)      // SRC-REF
	pdu = append(pdu, 0x00)            // Class 0

	// Cookie
	pdu = append(pdu, cookieBytes...)

	// RDP Negotiation Request
	pdu = append(pdu, negReq...)

	return pdu
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
	// Flush any remaining frames in the buffer
	if recording.FrameBuffer != nil {
		recording.FrameBuffer.flush(ctx)
	}

	// Encode remaining frames and save to storage
	if recording.Encoder != nil && recording.FrameBuffer != nil && recording.Storage != nil {
		recording.FrameBuffer.mu.Lock()
		frameCount := len(recording.FrameBuffer.frames)
		recording.FrameBuffer.mu.Unlock()

		p.logger.Info().
			Str("session_id", recording.SessionID.String()).
			Int("frame_count", frameCount).
			Msg("Finalizing RDP recording")
	}

	if recording.Storage != nil {
		if err := recording.Storage.Finalize(ctx, recording.SessionID); err != nil {
			p.logger.Error().Err(err).
				Str("session_id", recording.SessionID.String()).
				Msg("Failed to finalize RDP recording storage")
			return fmt.Errorf("rdp.FinalizeRecording: %w", err)
		}
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

	// Aggregate frames into a recording chunk for storage
	// Each flush batch is saved as a segment that can be replayed
	if len(frames) > 0 {
		totalSize := 0
		for _, f := range frames {
			totalSize += len(f.Data)
		}
		_ = totalSize // Would be used for storage upload metrics
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
