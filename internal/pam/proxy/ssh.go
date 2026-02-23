package proxy

import (
	"context"
	"encoding/binary"
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
	"golang.org/x/crypto/ssh"
)

// SSHProxy handles SSH session proxying
type SSHProxy struct {
	cache         *cache.Cache
	publisher     *events.Publisher
	logger        zerolog.Logger
	dialTimeout   time.Duration
	sessionTimeout time.Duration
}

// NewSSHProxy creates a new SSH proxy
func NewSSHProxy(c *cache.Cache, publisher *events.Publisher, logger zerolog.Logger) *SSHProxy {
	return &SSHProxy{
		cache:         c,
		publisher:     publisher,
		logger:        logger,
		dialTimeout:   10 * time.Second,
		sessionTimeout: 24 * time.Hour,
	}
}

// Session represents an active SSH session
type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	CredentialID uuid.UUID `json:"credential_id"`
	TargetHost   string    `json:"target_host"`
	TargetPort   int       `json:"target_port"`
	SSHClient    *ssh.Client `json:"-"`
	WebSocket    *websocket.Conn `json:"-"`
	StartTime    time.Time `json:"start_time"`
	LastActivity time.Time `json:"last_activity"`
	Recording    *SessionRecording `json:"-"`
	mu           sync.RWMutex
	closed       bool
}

// SessionRecording handles session recording
type SessionRecording struct {
	SessionID  uuid.UUID
	Output     *SSHOutputWriter
	Keystrokes *KeystrokeRecorder
	Storage    RecordingStorage
}

// SSHOutputWriter writes SSH output to recording
type SSHOutputWriter struct {
	sessionID uuid.UUID
	storage   RecordingStorage
	buffer    []byte
	mu        sync.Mutex
}

// KeystrokeRecorder captures keystrokes for audit
type KeystrokeRecorder struct {
	sessionID uuid.UUID
	storage   RecordingStorage
	keystrokes []KeystrokeEvent
	mu        sync.Mutex
}

// KeystrokeEvent represents a single keystroke event
type KeystrokeEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Key       string    `json:"key"`
	Type      string    `json:"type"` // key, command, paste
}

// RecordingStorage defines the interface for storing recordings
type RecordingStorage interface {
	SaveOutput(ctx context.Context, sessionID uuid.UUID, data []byte) error
	SaveKeystroke(ctx context.Context, sessionID uuid.UUID, event KeystrokeEvent) error
	Finalize(ctx context.Context, sessionID uuid.UUID) error
}

// HandleWebSocketConnection handles a WebSocket connection for SSH
func (p *SSHProxy) HandleWebSocketConnection(ctx context.Context, conn *websocket.Conn, userID, credentialID uuid.UUID, targetHost string, targetPort int) (*Session, error) {
	// Generate session ID
	sessionID := uuid.New()

	// Create SSH client
	sshClient, err := p.connectSSH(ctx, credentialID, targetHost, targetPort)
	if err != nil {
		return nil, fmt.Errorf("ssh.Connect: %w", err)
	}

	// Create session
	session := &Session{
		ID:           sessionID,
		UserID:       userID,
		CredentialID: credentialID,
		TargetHost:   targetHost,
		TargetPort:   targetPort,
		SSHClient:    sshClient,
		WebSocket:    conn,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}

	// Setup recording
	recording, err := p.setupRecording(ctx, sessionID)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("ssh.SetupRecording: %w", err)
	}
	session.Recording = recording

	// Cache session
	cacheKey := fmt.Sprintf("ssh_session:%s", sessionID)
	_ = p.cache.Set(ctx, cacheKey, session, p.sessionTimeout)

	// Start SSH session
	if err := p.startSSHSession(ctx, session); err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("ssh.StartSession: %w", err)
	}

	// Publish session started event
	_ = p.publisher.PublishSessionStarted(ctx, "", userID.String(), sessionID.String(), targetHost, targetPort)

	p.logger.Info().
		Str("session_id", sessionID.String()).
		Str("user_id", userID.String()).
		Str("target", fmt.Sprintf("%s:%d", targetHost, targetPort)).
		Msg("SSH session started")

	return session, nil
}

// connectSSH establishes an SSH connection
func (p *SSHProxy) connectSSH(ctx context.Context, credentialID uuid.UUID, host string, port int) (*ssh.Client, error) {
	// In production, you would:
	// 1. Retrieve the credential from vault
	// 2. Parse the SSH key or use password
	// 3. Establish SSH connection with timeout

	// For now, return a mock client (to be implemented with actual vault integration)
	// This is a placeholder for the actual SSH connection logic

	config := &ssh.ClientConfig{
		User: "root", // Would come from credential
		Auth: []ssh.AuthMethod{
			// ssh.Password(password) or ssh.PublicKeys(signer)
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         p.dialTimeout,
	}

	address := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("ssh.Dial: %w", err)
	}

	return client, nil
}

// startSSHSession starts an interactive SSH session
func (p *SSHProxy) startSSHSession(ctx context.Context, session *Session) error {
	// Create SSH session
	sshSession, err := session.SSHClient.NewSession()
	if err != nil {
		return fmt.Errorf("ssh.NewSession: %w", err)
	}

	// Request PTY
	// sshSession.RequestPty("xterm", 80, 40, ssh.TerminalModes{})

	// Setup pipes
	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("ssh.StdinPipe: %w", err)
	}

	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ssh.StdoutPipe: %w", err)
	}

	stderrPipe, err := sshSession.StderrPipe()
	if err != nil {
		return fmt.Errorf("ssh.StderrPipe: %w", err)
	}

	// Start shell
	if err := sshSession.Shell(); err != nil {
		return fmt.Errorf("ssh.Shell: %w", err)
	}

	// Start bidirectional forwarding
	go p.forwardToSSH(session, stdinPipe)
	go p.forwardFromSSH(session, stdoutPipe)
	go p.forwardFromSSH(session, stderrPipe)

	return nil
}

// forwardToSSH forwards WebSocket messages to SSH stdin
func (p *SSHProxy) forwardToSSH(session *Session, stdin io.Writer) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error().Interface("panic", r).Msg("Panic in forwardToSSH")
		}
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

		// Parse message (could be structured JSON with type)
		// For now, assume raw data to forward

		if _, err := stdin.Write(message); err != nil {
			p.logger.Error().Err(err).Msg("SSH write error")
			break
		}

		// Record keystrokes
		if session.Recording != nil && session.Recording.Keystrokes != nil {
			session.Recording.Keystrokes.Record(string(message))
		}
	}
}

// forwardFromSSH forwards SSH stdout/stderr to WebSocket
func (p *SSHProxy) forwardFromSSH(session *Session, output io.Reader) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error().Interface("panic", r).Msg("Panic in forwardFromSSH")
		}
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := output.Read(buf)
		if err != nil {
			if err != io.EOF {
				p.logger.Error().Err(err).Msg("SSH read error")
			}
			break
		}

		if n > 0 {
			// Send to WebSocket
			if err := session.WebSocket.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				p.logger.Error().Err(err).Msg("WebSocket write error")
				break
			}

			// Record output
			if session.Recording != nil && session.Recording.Output != nil {
				session.Recording.Output.Write(buf[:n])
			}
		}
	}
}

// BroadcastKeystrokes broadcasts keystrokes to monitoring sessions
func (p *SSHProxy) BroadcastKeystrokes(ctx context.Context, sessionID uuid.UUID, keystroke string) {
	// Send to monitoring subscribers via Redis pub/sub
	channel := fmt.Sprintf("session:%s:keystrokes", sessionID)
	_ = p.cache.PubSub().Publish(ctx, channel, events.Event{
		Type:     "keystroke",
		Data:     map[string]interface{}{"keystroke": keystroke},
	})
}

// TerminateSession terminates an active session
func (p *SSHProxy) TerminateSession(ctx context.Context, sessionID uuid.UUID, terminatedBy uuid.UUID, reason string) error {
	// Get session from cache
	cacheKey := fmt.Sprintf("ssh_session:%s", sessionID)
	var session Session
	if err := p.cache.Get(ctx, cacheKey, &session); err != nil {
		return fmt.Errorf("ssh.SessionNotFound: %w", err)
	}

	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return fmt.Errorf("ssh: session already closed")
	}
	session.closed = true
	session.mu.Unlock()

	// Close SSH connection
	if session.SSHClient != nil {
		_ = session.SSHClient.Close()
	}

	// Close WebSocket
	if session.WebSocket != nil {
		_ = session.WebSocket.Close()
	}

	// Finalize recording
	if session.Recording != nil {
		_ = session.Recording.Storage.Finalize(ctx, sessionID)
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
		Msg("SSH session terminated")

	return nil
}

// GetActiveSessions retrieves all active SSH sessions
func (p *SSHProxy) GetActiveSessions(ctx context.Context) ([]Session, error) {
	// In production, this would query Redis or a database
	// For now, return empty
	return []Session{}, nil
}

// setupRecording sets up session recording
func (p *SSHProxy) setupRecording(ctx context.Context, sessionID uuid.UUID) (*SessionRecording, error) {
	storage := &S3RecordingStorage{ // Would be configured based on settings
		bucket: "session-recordings",
	}

	return &SessionRecording{
		SessionID:  sessionID,
		Output:     NewSSHOutputWriter(sessionID, storage),
		Keystrokes: NewKeystrokeRecorder(sessionID, storage),
		Storage:    storage,
	}, nil
}

// S3RecordingStorage stores recordings in S3/MinIO
type S3RecordingStorage struct {
	bucket string
	client interface{} // S3 client
}

// SaveOutput saves session output
func (s *S3RecordingStorage) SaveOutput(ctx context.Context, sessionID uuid.UUID, data []byte) error {
	// Implement S3 upload
	return nil
}

// SaveKeystroke saves a keystroke event
func (s *S3RecordingStorage) SaveKeystroke(ctx context.Context, sessionID uuid.UUID, event KeystrokeEvent) error {
	// Implement S3 upload for keystroke log
	return nil
}

// Finalize finalizes the recording
func (s *S3RecordingStorage) Finalize(ctx context.Context, sessionID uuid.UUID) error {
	// Implement finalization (e.g., upload remaining buffer, create index)
	return nil
}

// NewSSHOutputWriter creates a new output writer
func NewSSHOutputWriter(sessionID uuid.UUID, storage RecordingStorage) *SSHOutputWriter {
	return &SSHOutputWriter{
		sessionID: sessionID,
		storage:   storage,
		buffer:    make([]byte, 0),
	}
}

// Write implements io.Writer
func (w *SSHOutputWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, p...)

	// Flush buffer periodically (e.g., every 1MB or 10 seconds)
	if len(w.buffer) > 1024*1024 {
		go w.flush(context.Background())
	}

	return len(p), nil
}

func (w *SSHOutputWriter) flush(ctx context.Context) {
	w.mu.Lock()
	data := make([]byte, len(w.buffer))
	copy(data, w.buffer)
	w.buffer = w.buffer[:0]
	w.mu.Unlock()

	if len(data) > 0 {
		_ = w.storage.SaveOutput(ctx, w.sessionID, data)
	}
}

// NewKeystrokeRecorder creates a new keystroke recorder
func NewKeystrokeRecorder(sessionID uuid.UUID, storage RecordingStorage) *KeystrokeRecorder {
	return &KeystrokeRecorder{
		sessionID:  sessionID,
		storage:    storage,
		keystrokes: make([]KeystrokeEvent, 0),
	}
}

// Record records a keystroke
func (r *KeystrokeRecorder) Record(keystroke string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	event := KeystrokeEvent{
		Timestamp: time.Now(),
		Key:       keystroke,
		Type:      "key",
	}
	r.keystrokes = append(r.keystrokes, event)

	// Flush periodically
	if len(r.keystrokes) > 100 {
		go r.flush(context.Background())
	}
}

func (r *KeystrokeRecorder) flush(ctx context.Context) {
	r.mu.Lock()
	events := make([]KeystrokeEvent, len(r.keystrokes))
	copy(events, r.keystrokes)
	r.keystrokes = r.keystrokes[:0]
	r.mu.Unlock()

	for _, event := range events {
		_ = r.storage.SaveKeystroke(ctx, r.sessionID, event)
	}
}
