package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// EventType represents the type of audit event
type EventType string

const (
	EventTypeAuth           EventType = "authentication"
	EventTypeCredential     EventType = "credential"
	EventTypeSession        EventType = "session"
	EventTypeTarget         EventType = "target"
	EventTypeUser           EventType = "user"
	EventTypeRole           EventType = "role"
	EventTypeApproval       EventType = "approval"
	EventTypeCheckout       EventType = "checkout"
	EventTypeConfiguration  EventType = "configuration"
	EventTypeAnomaly        EventType = "anomaly"
	EventTypeBreakGlass     EventType = "break_glass"
)

// EventOutcome represents the outcome of an action
type EventOutcome string

const (
	OutcomeSuccess EventOutcome = "success"
	OutcomeFailure EventOutcome = "failure"
	OutcomeDenied  EventOutcome = "denied"
)

// Event represents an audit log entry
type Event struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	TenantID     uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	ActorID      uuid.UUID       `db:"actor_id" json:"actor_id"`
	ActorType    string          `db:"actor_type" json:"actor_type"` // user, system, api
	Action       string          `db:"action" json:"action"`         // create, read, update, delete, checkout, checkin, etc.
	ResourceType string          `db:"resource_type" json:"resource_type"`
	ResourceID   string          `db:"resource_id" json:"resource_id"`
	Outcome      EventOutcome    `db:"outcome" json:"outcome"`

	// Request info
	IP           string          `db:"ip" json:"ip"`
	UserAgent    string          `db:"user_agent" json:"user_agent"`
	RequestID    string          `db:"request_id" json:"request_id"`

	// Event details
	Details      json.RawMessage `db:"details" json:"details,omitempty"`
	ErrorCode    string          `db:"error_code" json:"error_code,omitempty"`
	ErrorMessage string          `db:"error_message" json:"error_message,omitempty"`

	// Chain integrity
	PreviousHash string          `db:"previous_hash" json:"previous_hash"`
	Hash         string          `db:"hash" json:"hash"`

	Timestamp    time.Time       `db:"created_at" json:"timestamp"`
}

// Repository handles audit data operations
type Repository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRepository creates a new audit repository
func NewRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Repository {
	repo := &Repository{db: db, cache: c, logger: logger}
	go repo.startEventProcessor()
	return repo
}

// Create creates an audit event
func (r *Repository) Create(ctx context.Context, event *Event) error {
	// Generate ID and timestamp
	event.ID = uuid.New()
	event.Timestamp = time.Now()

	// Get previous hash for chain integrity
	if err := r.setChainHash(ctx, event); err != nil {
		r.logger.Error().Err(err).Msg("Failed to set chain hash")
	}

	// Generate hash for this event
	event.Hash = r.generateHash(event)

	// Insert event
	query := `
		INSERT INTO audit_events (id, tenant_id, actor_id, actor_type, action, resource_type,
			resource_id, outcome, ip, user_agent, request_id, details, error_code, error_message,
			previous_hash, hash, created_at)
		VALUES (:id, :tenant_id, :actor_id, :actor_type, :action, :resource_type,
			:resource_id, :outcome, :ip, :user_agent, :request_id, :details, :error_code, :error_message,
			:previous_hash, :hash, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("audit.Create: %w", err)
	}

	// Publish to event bus for real-time monitoring
	go r.publishEvent(context.Background(), event)

	return nil
}

// setChainHash sets the previous hash and current hash for chain integrity
func (r *Repository) setChainHash(ctx context.Context, event *Event) error {
	// Get last event hash for this tenant
	var lastHash string
	query := `
		SELECT hash FROM audit_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &lastHash, query, event.TenantID)
	if err != nil {
		// First event, no previous hash
		lastHash = ""
	}

	event.PreviousHash = lastHash
	return nil
}

// generateHash generates a hash for the event
func (r *Repository) generateHash(event *Event) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%d",
		event.ID,
		event.TenantID,
		event.ActorID,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		event.Outcome,
		event.IP,
		event.PreviousHash,
		event.Timestamp.UnixNano(),
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// List retrieves audit events with filtering and pagination
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, filter EventFilter, limit, offset int) ([]Event, int, error) {
	baseQuery := `
		SELECT * FROM audit_events
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM audit_events WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.ActorID != nil {
		baseQuery += fmt.Sprintf(" AND actor_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND actor_id = $%d", argCount)
		args = append(args, *filter.ActorID)
		argCount++
	}
	if filter.Action != nil {
		baseQuery += fmt.Sprintf(" AND action = $%d", argCount)
		countQuery += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, *filter.Action)
		argCount++
	}
	if filter.ResourceType != nil {
		baseQuery += fmt.Sprintf(" AND resource_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, *filter.ResourceType)
		argCount++
	}
	if filter.Outcome != nil {
		baseQuery += fmt.Sprintf(" AND outcome = $%d", argCount)
		countQuery += fmt.Sprintf(" AND outcome = $%d", argCount)
		args = append(args, *filter.Outcome)
		argCount++
	}
	if filter.StartTimeFrom != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartTimeFrom)
		argCount++
	}
	if filter.StartTimeTo != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.StartTimeTo)
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("audit.List.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var events []Event
	if err := r.db.SelectContext(ctx, &events, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("audit.List: %w", err)
	}

	return events, total, nil
}

// GetByID retrieves an event by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Event, error) {
	var event Event
	query := `SELECT * FROM audit_events WHERE id = $1`
	err := r.db.GetContext(ctx, &event, query, id)
	if err != nil {
		return nil, fmt.Errorf("audit.GetByID: %w", err)
	}
	return &event, nil
}

// VerifyChain verifies the integrity of the audit chain
func (r *Repository) VerifyChain(ctx context.Context, tenantID uuid.UUID) (bool, []string, error) {
	query := `
		SELECT id, previous_hash, hash FROM audit_events
		WHERE tenant_id = $1
		ORDER BY created_at ASC
	`

	var events []struct {
		ID           uuid.UUID `db:"id"`
		PreviousHash string    `db:"previous_hash"`
		Hash         string    `db:"hash"`
	}
	if err := r.db.SelectContext(ctx, &events, query, tenantID); err != nil {
		return false, nil, fmt.Errorf("audit.VerifyChain: %w", err)
	}

	errors := []string{}
	expectedPrevHash := ""

	for i, event := range events {
		if event.PreviousHash != expectedPrevHash {
			errors = append(errors, fmt.Sprintf("Event %d: previous_hash mismatch", i))
		}
		expectedPrevHash = event.Hash
	}

	return len(errors) == 0, errors, nil
}

// ExportEvents exports audit events in a format for compliance reporting
func (r *Repository) ExportEvents(ctx context.Context, tenantID uuid.UUID, filter EventFilter) ([]byte, error) {
	events, _, err := r.List(ctx, tenantID, filter, 100000, 0)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(events, "", "  ")
}

// publishEvent publishes event to Redis for real-time monitoring
func (r *Repository) publishEvent(ctx context.Context, event *Event) {
	channel := fmt.Sprintf("audit:%s", event.TenantID)
	data, _ := json.Marshal(event)
	_ = r.cache.PubSub().Publish(ctx, channel, cache.Event{
		Type: "audit_event",
		Data: map[string]interface{}{
			"event": string(data),
		},
	})
}

// startEventProcessor processes audit events for aggregation
func (r *Repository) startEventProcessor() {
	// Subscribe to audit events
	ctx := context.Background()
	channel := "audit:*"
	pubsub, err := r.cache.PubSub().Subscribe(ctx, channel)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to subscribe to audit events")
		return
	}
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var event Event
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			r.logger.Error().Err(err).Msg("Failed to unmarshal audit event")
			continue
		}
		// Aggregate event for reporting
		_ = r.aggregateEvent(ctx, event)
	}
}

// aggregateEvent aggregates event data for reporting
func (r *Repository) aggregateEvent(ctx context.Context, event Event) error {
	// Update aggregation tables
	// e.g., daily_activity, user_activity, resource_access
	return nil
}

// EventFilter filters audit event queries
type EventFilter struct {
	ActorID      *uuid.UUID
	Action       *string
	ResourceType *string
	Outcome      *EventOutcome
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
}

// Service handles audit business logic
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new audit service
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// Log creates an audit event
func (s *Service) Log(ctx context.Context, event *Event) error {
	return s.repo.Create(ctx, event)
}

// LogAction is a helper to log an action with common fields
func (s *Service) LogAction(ctx context.Context, tenantID, actorID uuid.UUID, actorType, action, resourceType, resourceID string, outcome EventOutcome, details map[string]interface{}) error {
	event := &Event{
		TenantID:     tenantID,
		ActorID:      actorID,
		ActorType:    actorType,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      outcome,
	}

	if details != nil {
		data, _ := json.Marshal(details)
		event.Details = data
	}

	return s.repo.Create(ctx, event)
}

// Query retrieves audit events
func (s *Service) Query(ctx context.Context, tenantID uuid.UUID, filter EventFilter, limit, offset int) ([]Event, int, error) {
	return s.repo.List(ctx, tenantID, filter, limit, offset)
}

// VerifyIntegrity verifies the audit log integrity
func (s *Service) VerifyIntegrity(ctx context.Context, tenantID uuid.UUID) (bool, []string, error) {
	return s.repo.VerifyChain(ctx, tenantID)
}

// GenerateComplianceReport generates a compliance report
func (s *Service) GenerateComplianceReport(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time) (*ComplianceReport, error) {
	// Get events for the period
	filter := EventFilter{
		StartTimeFrom: &startTime,
		StartTimeTo:   &endTime,
	}

	events, _, err := s.repo.List(ctx, tenantID, filter, 1000000, 0)
	if err != nil {
		return nil, err
	}

	report := &ComplianceReport{
		TenantID:  tenantID,
		StartTime: startTime,
		EndTime:   endTime,
		Generated: time.Now(),
	}

	// Aggregate events
	for _, event := range events {
		report.TotalEvents++
		switch event.Outcome {
		case OutcomeSuccess:
			report.SuccessfulEvents++
		case OutcomeFailure:
			report.FailedEvents++
		case OutcomeDenied:
			report.DeniedEvents++
		}

		report.EventTypes[event.Action]++

		switch event.ResourceType {
		case "credential":
			report.CredentialEvents++
		case "session":
			report.SessionEvents++
		case "user":
			report.UserEvents++
		}
	}

	return report, nil
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	TenantID        uuid.UUID           `json:"tenant_id"`
	StartTime       time.Time           `json:"start_time"`
	EndTime         time.Time           `json:"end_time"`
	Generated       time.Time           `json:"generated"`
	TotalEvents     int                 `json:"total_events"`
	SuccessfulEvents int                 `json:"successful_events"`
	FailedEvents    int                 `json:"failed_events"`
	DeniedEvents    int                 `json:"denied_events"`
	EventTypes      map[string]int      `json:"event_types"`
	CredentialEvents int                 `json:"credential_events"`
	SessionEvents   int                 `json:"session_events"`
	UserEvents      int                 `json:"user_events"`
	IntegrityValid  bool                `json:"integrity_valid"`
	IntegrityErrors []string            `json:"integrity_errors,omitempty"`
}
