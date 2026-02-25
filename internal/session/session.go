package session

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// SessionType represents the type of session
type SessionType string

const (
	SessionTypeSSH       SessionType = "ssh"
	SessionTypeRDP       SessionType = "rdp"
	SessionTypeDatabase  SessionType = "database"
	SessionTypeKubernetes SessionType = "kubernetes"
	SessionTypeWeb       SessionType = "web"
	SessionTypeAPI       SessionType = "api"
)

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusEnded      SessionStatus = "ended"
	SessionStatusTerminated SessionStatus = "terminated"
	SessionStatusFailed     SessionStatus = "failed"
)

// Session represents a privileged access session
// SECURITY: Sensitive fields are exposed via JSON for API responses, but ForLog() method
// should be used for any logging to prevent credential/target information leakage
type Session struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	UserID       uuid.UUID       `db:"user_id" json:"user_id"`
	CredentialID uuid.UUID       `db:"credential_id" json:"credential_id"` // Sensitive: should be redacted in logs
	TargetID     *uuid.UUID      `db:"target_id" json:"target_id,omitempty"`
	Type         SessionType     `db:"type" json:"type"`
	Status       SessionStatus   `db:"status" json:"status"`

	// Connection details - SENSITIVE: should be redacted in logs
	TargetHost   string          `db:"target_host" json:"target_host"`
	TargetPort   int             `db:"target_port" json:"target_port"`
	ClientIP     string          `db:"client_ip" json:"client_ip"`
	UserAgent    string          `db:"user_agent" json:"user_agent"`

	// Recording
	RecordingID  *uuid.UUID      `db:"recording_id" json:"recording_id,omitempty"`
	RecordingURL string          `db:"recording_url" json:"recording_url"`

	// Timing
	StartedAt    time.Time       `db:"started_at" json:"started_at"`
	EndedAt      *time.Time      `db:"ended_at" json:"ended_at,omitempty"`
	TerminatedBy *uuid.UUID      `db:"terminated_by" json:"terminated_by,omitempty"`
	TerminateReason string       `db:"terminate_reason" json:"terminate_reason,omitempty"`

	// Metadata
	Metadata     json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	TenantID     uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// SessionLoggable represents a sanitized version of Session safe for logging
// SECURITY: This type ensures sensitive information is never logged
type SessionLoggable struct {
	ID           uuid.UUID     `json:"id"`
	UserID       uuid.UUID     `json:"user_id"`
	TargetID     *uuid.UUID    `json:"target_id,omitempty"`
	Type         SessionType   `json:"type"`
	Status       SessionStatus `json:"status"`
	// Sensitive fields are redacted
	TargetHost   string        `json:"target_host,omitempty"`   // Redacted in ForLog()
	TargetPort   int           `json:"target_port,omitempty"`   // Redacted in ForLog()
	ClientIP     string        `json:"client_ip,omitempty"`     // Redacted in ForLog()
	UserAgent    string        `json:"user_agent,omitempty"`
	RecordingID  *uuid.UUID    `json:"recording_id,omitempty"`
	RecordingURL string        `json:"recording_url,omitempty"`
	StartedAt    time.Time     `json:"started_at"`
	EndedAt      *time.Time    `json:"ended_at,omitempty"`
	TerminatedBy *uuid.UUID    `json:"terminated_by,omitempty"`
	TerminateReason string     `json:"terminate_reason,omitempty"`
	TenantID     uuid.UUID     `json:"tenant_id"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// ForLog returns a sanitized version of the session safe for logging
// SECURITY: This method must be used when logging sessions to prevent leakage of:
// - CredentialID (which credential was used)
// - TargetHost (which system was accessed)
// - TargetPort (which port on the target)
// - ClientIP (the accessing IP address)
func (s *Session) ForLog() SessionLoggable {
	return SessionLoggable{
		ID:             s.ID,
		UserID:         s.UserID,
		TargetID:       s.TargetID,
		Type:           s.Type,
		Status:         s.Status,
		TargetHost:     "[REDACTED]",
		TargetPort:     0,
		ClientIP:       "[REDACTED]",
		UserAgent:      s.UserAgent,
		RecordingID:    s.RecordingID,
		RecordingURL:   s.RecordingURL,
		StartedAt:      s.StartedAt,
		EndedAt:        s.EndedAt,
		TerminatedBy:   s.TerminatedBy,
		TerminateReason: s.TerminateReason,
		TenantID:       s.TenantID,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// MarshalJSON implements custom JSON marshaling with optional redaction
// SECURITY: By default, this marshals the full session including sensitive fields
// For logging purposes, use ForLog() method instead
func (s *Session) MarshalJSON() ([]byte, error) {
	// Define a local type to avoid recursive MarshalJSON
	type Alias Session
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	})
}

// ActiveSession holds runtime session data
type ActiveSession struct {
	Session     *Session
	WebSocket   interface{} // *websocket.Conn
	Recording   interface{} // *SessionRecording
	mu          sync.RWMutex
	closed      bool
}

// Repository handles session data operations
type Repository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRepository creates a new session repository
func NewRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Repository {
	return &Repository{db: db, cache: c, logger: logger}
}

// Create creates a new session
func (r *Repository) Create(ctx context.Context, session *Session) error {
	session.ID = uuid.New()
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	session.StartedAt = time.Now()
	session.Status = SessionStatusActive

	query := `
		INSERT INTO sessions (id, user_id, credential_id, target_id, type, status,
			target_host, target_port, client_ip, user_agent, recording_id, recording_url,
			started_at, ended_at, terminated_by, terminate_reason, metadata, tenant_id,
			created_at, updated_at)
		VALUES (:id, :user_id, :credential_id, :target_id, :type, :status,
			:target_host, :target_port, :client_ip, :user_agent, :recording_id, :recording_url,
			:started_at, :ended_at, :terminated_by, :terminate_reason, :metadata, :tenant_id,
			:created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, session)
	if err != nil {
		return fmt.Errorf("session.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a session by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	var session Session
	query := `SELECT * FROM sessions WHERE id = $1`
	err := r.db.GetContext(ctx, &session, query, id)
	if err != nil {
		return nil, fmt.Errorf("session.GetByID: %w", err)
	}
	return &session, nil
}

// List retrieves sessions with filtering and pagination
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, filter SessionFilter, limit, offset int) ([]Session, int, error) {
	baseQuery := `
		SELECT * FROM sessions
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM sessions WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.Type != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
		argCount++
	}
	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}
	if filter.StartTimeFrom != nil {
		baseQuery += fmt.Sprintf(" AND started_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND started_at >= $%d", argCount)
		args = append(args, *filter.StartTimeFrom)
		argCount++
	}
	if filter.StartTimeTo != nil {
		baseQuery += fmt.Sprintf(" AND started_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND started_at <= $%d", argCount)
		args = append(args, *filter.StartTimeTo)
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("session.List.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY started_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var sessions []Session
	if err := r.db.SelectContext(ctx, &sessions, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("session.List: %w", err)
	}

	return sessions, total, nil
}

// Update updates a session
func (r *Repository) Update(ctx context.Context, session *Session) error {
	session.UpdatedAt = time.Now()

	query := `
		UPDATE sessions SET
			status = :status,
			recording_id = :recording_id,
			recording_url = :recording_url,
			ended_at = :ended_at,
			terminated_by = :terminated_by,
			terminate_reason = :terminate_reason,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, session)
	if err != nil {
		return fmt.Errorf("session.Update: %w", err)
	}

	return nil
}

// EndSession ends a session
func (r *Repository) EndSession(ctx context.Context, id uuid.UUID, status SessionStatus, endedAt *time.Time) error {
	query := `
		UPDATE sessions
		SET status = $2, ended_at = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, endedAt)
	if err != nil {
		return fmt.Errorf("session.EndSession: %w", err)
	}
	return nil
}

// GetActiveSessionsForUser retrieves active sessions for a user
func (r *Repository) GetActiveSessionsForUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	var sessions []Session
	query := `
		SELECT * FROM sessions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY started_at DESC
	`
	err := r.db.SelectContext(ctx, &sessions, query, userID)
	if err != nil {
		return nil, fmt.Errorf("session.GetActiveSessionsForUser: %w", err)
	}
	return sessions, nil
}

// SessionFilter filters session queries
type SessionFilter struct {
	Type          *SessionType
	Status        *SessionStatus
	UserID        *uuid.UUID
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
}

// Service handles session business logic
type Service struct {
	repo       *Repository
	cache      *cache.Cache
	publisher  *events.Publisher
	logger     zerolog.Logger
	sessions   map[uuid.UUID]*ActiveSession
	mu         sync.RWMutex
}

// NewService creates a new session service
func NewService(repo *Repository, c *cache.Cache, publisher *events.Publisher, logger zerolog.Logger) *Service {
	s := &Service{
		repo:      repo,
		cache:     c,
		publisher: publisher,
		logger:    logger,
		sessions:  make(map[uuid.UUID]*ActiveSession),
	}
	// Start cleanup goroutine
	go s.cleanupExpiredSessions()
	return s
}

// StartSession starts a new session
func (s *Service) StartSession(ctx context.Context, session *Session) error {
	// Create session record
	if err := s.repo.Create(ctx, session); err != nil {
		return err
	}

	// Create active session tracker
	active := &ActiveSession{
		Session: session,
		closed:  false,
	}

	s.mu.Lock()
	s.sessions[session.ID] = active
	s.mu.Unlock()

	// Cache session
	cacheKey := fmt.Sprintf("session:%s", session.ID)
	_ = s.cache.Set(ctx, cacheKey, session, 24*time.Hour)

	// Publish event
	_ = s.publisher.PublishSessionStarted(ctx, session.TenantID.String(),
		session.UserID.String(), session.ID.String(),
		session.TargetHost, session.TargetPort)

	// Publish analytics event with session type
	_ = s.publisher.PublishAnalyticsSessionStarted(ctx, session.TenantID.String(),
		session.UserID.String(), session.ID.String(),
		session.TargetHost, session.TargetPort, string(session.Type))

	s.logger.Info().
		Str("session_id", session.ID.String()).
		Str("user_id", session.UserID.String()).
		Str("target_host", "[REDACTED]").
		Int("target_port", 0).
		Str("type", string(session.Type)).
		Msg("Session started")

	return nil
}

// EndSession ends a session
func (s *Service) EndSession(ctx context.Context, sessionID uuid.UUID, terminatedBy *uuid.UUID, reason string) error {
	s.mu.RLock()
	active, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session: not found")
	}

	active.mu.Lock()
	if active.closed {
		active.mu.Unlock()
		return fmt.Errorf("session: already closed")
	}
	active.closed = true
	active.mu.Unlock()

	now := time.Now()
	status := SessionStatusEnded
	if terminatedBy != nil {
		status = SessionStatusTerminated
		active.Session.TerminatedBy = terminatedBy
		active.Session.TerminateReason = reason
	}
	active.Session.Status = status
	active.Session.EndedAt = &now

	// Update database
	if err := s.repo.Update(ctx, active.Session); err != nil {
		return err
	}

	// Remove from active sessions
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()

	// Invalidate cache
	cacheKey := fmt.Sprintf("session:%s", sessionID)
	_ = s.cache.Delete(ctx, cacheKey)

	duration := now.Sub(active.Session.StartedAt)
	_ = s.publisher.PublishSessionEnded(ctx, active.Session.TenantID.String(),
		active.Session.UserID.String(), sessionID.String(), duration)

	// Publish analytics event
	_ = s.publisher.PublishAnalyticsSessionEnded(ctx, active.Session.TenantID.String(),
		active.Session.UserID.String(), sessionID.String(), duration, string(active.Session.Type))

	s.logger.Info().
		Str("session_id", sessionID.String()).
		Str("status", string(status)).
		Dur("duration", duration).
		Msg("Session ended")

	return nil
}

// GetSession retrieves a session
func (s *Service) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	return s.repo.GetByID(ctx, sessionID)
}

// GetActiveSession retrieves an active session tracker
func (s *Service) GetActiveSession(sessionID uuid.UUID) (*ActiveSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session: not found or not active")
	}

	return active, nil
}

// ListSessions lists sessions with filtering
func (s *Service) ListSessions(ctx context.Context, tenantID uuid.UUID, filter SessionFilter, limit, offset int) ([]Session, int, error) {
	return s.repo.List(ctx, tenantID, filter, limit, offset)
}

// TerminateSession terminates an active session (admin operation)
func (s *Service) TerminateSession(ctx context.Context, sessionID, terminatedBy uuid.UUID, reason string) error {
	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != SessionStatusActive {
		return fmt.Errorf("session: not active")
	}

	return s.EndSession(ctx, sessionID, &terminatedBy, reason)
}

// TerminateUserSessions terminates all active sessions for a user
func (s *Service) TerminateUserSessions(ctx context.Context, userID, terminatedBy uuid.UUID, reason string) error {
	activeSessions, err := s.repo.GetActiveSessionsForUser(ctx, userID)
	if err != nil {
		return err
	}

	for _, session := range activeSessions {
		_ = s.TerminateSession(ctx, session.ID, terminatedBy, reason)
	}

	return nil
}

// UpdateSessionActivity updates the last activity time for a session
func (s *Service) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	cacheKey := fmt.Sprintf("session:%s:activity", sessionID)
	return s.cache.Set(ctx, cacheKey, time.Now(), 10*time.Minute)
}

// IsSessionActive checks if a session is still active
func (s *Service) IsSessionActive(ctx context.Context, sessionID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active, exists := s.sessions[sessionID]
	if !exists {
		return false
	}

	active.mu.RLock()
	defer active.mu.RUnlock()
	return !active.closed
}

// cleanupExpiredSessions cleans up inactive sessions
func (s *Service) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		s.mu.Lock()
		for id, active := range s.sessions {
			// Check activity
			cacheKey := fmt.Sprintf("session:%s:activity", id)
			var lastActivity time.Time
			if err := s.cache.Get(ctx, cacheKey, &lastActivity); err != nil {
				// No recent activity, check session start time
				lastActivity = active.Session.StartedAt
			}

			// Terminate if inactive for more than 30 minutes
			if time.Since(lastActivity) > 30*time.Minute {
				s.logger.Info().
					Str("session_id", id.String()).
					Msg("Terminating inactive session")
				_ = s.EndSession(ctx, id, nil, "inactive timeout")
			}
		}
		s.mu.Unlock()
	}
}

// GetSessionStats returns statistics about sessions
func (s *Service) GetSessionStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	// Get active count
	s.mu.RLock()
	activeCount := len(s.sessions)
	s.mu.RUnlock()

	// Get today's session count
	today := time.Now().Truncate(24 * time.Hour)
	todayQuery := `
		SELECT COUNT(*) FROM sessions
		WHERE tenant_id = $1 AND started_at >= $2
	`
	var todayCount int
	if err := s.repo.db.GetContext(ctx, &todayCount, todayQuery, tenantID, today); err != nil {
		return nil, err
	}

	// Get total count
	totalQuery := `
		SELECT COUNT(*) FROM sessions
		WHERE tenant_id = $1
	`
	var totalCount int
	if err := s.repo.db.GetContext(ctx, &totalCount, totalQuery, tenantID); err != nil {
		return nil, err
	}

	// Get average duration
	durationQuery := `
		SELECT AVG(EXTRACT(EPOCH FROM (ended_at - started_at))) as avg_duration
		FROM sessions
		WHERE tenant_id = $1 AND ended_at IS NOT NULL
	`
	var avgDuration float64
	if err := s.repo.db.GetContext(ctx, &avgDuration, durationQuery, tenantID); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"active_count":  activeCount,
		"today_count":   todayCount,
		"total_count":   totalCount,
		"avg_duration":  avgDuration, // in seconds
	}, nil
}
