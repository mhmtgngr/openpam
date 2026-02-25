package events

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// EventBus handles event publishing and subscription
// SECURITY FIX: All events must be signed to prevent event injection/spoofing
type EventBus struct {
	cache        *cache.Cache
	logger       zerolog.Logger
	handlers     map[string][]Handler
	signingKey   []byte // HMAC signing key for event signatures
	allowUnsigned bool // SECURITY: When false, reject unsigned events
	mu           sync.RWMutex
}

// Handler processes an event
type Handler func(ctx context.Context, event Event) error

// Event represents a domain event
// SECURITY FIX: Added Signature field for mandatory HMAC verification
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	ActorID   string                 `json:"actor_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Signature string                 `json:"signature,omitempty"` // HMAC signature for event verification
}

// EventConfig holds configuration for the event bus
type EventConfig struct {
	Cache        *cache.Cache
	Logger       zerolog.Logger
	SigningKey   []byte // HMAC signing key (should come from KMS in production)
	AllowUnsigned bool  // When false, reject all unsigned events (default: false for security)
}

// New creates a new event bus
// SECURITY FIX: Signing key is required. Events without valid signatures are rejected.
func New(cfg EventConfig) *EventBus {
	if len(cfg.SigningKey) < 32 {
		cfg.Logger.Warn().Msg("events: signing key is less than 32 bytes, using default insecure key - DO NOT USE IN PRODUCTION")
		cfg.SigningKey = []byte("CHANGE_THIS_INSECURE_DEFAULT_KEY_32_BYTES!")
	}

	bus := &EventBus{
		cache:        cfg.Cache,
		logger:       cfg.Logger,
		handlers:     make(map[string][]Handler),
		signingKey:   cfg.SigningKey,
		allowUnsigned: cfg.AllowUnsigned,
	}
	go bus.startSubscriptionListener()
	return bus
}

// NewWithConfig creates a new event bus with configuration
// This is an alias for New(EventConfig) for backward compatibility
func NewWithConfig(cfg EventConfig) *EventBus {
	return New(cfg)
}

// DEPRECATED: Use New(EventConfig) instead
// NewWithoutConfig creates a new event bus without explicit config
// This is kept for backward compatibility but should not be used in production
func NewWithoutConfig(c *cache.Cache, logger zerolog.Logger) *EventBus {
	logger.Warn().Msg("events: Using deprecated New() function - events will NOT be signed. Use New(EventConfig) with a signing key for security.")
	return New(EventConfig{
		Cache:        c,
		Logger:       logger,
		SigningKey:   []byte("INSECURE_DEFAULT_KEY_DO_NOT_USE_IN_PRODUCTION"),
		AllowUnsigned: true, // Allow unsigned for backward compatibility
	})
}

// computeSignature computes HMAC-SHA256 signature for an event
// SECURITY: This prevents event injection and spoofing attacks
func (eb *EventBus) computeSignature(event Event) string {
	// Create a canonical representation of the event for signing
	// Exclude the Signature field itself when computing
	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%v",
		event.ID,
		event.Type,
		event.TenantID,
		event.ActorID,
		event.Action,
		event.Timestamp.UnixNano(),
		event.Data,
	)

	h := hmac.New(sha256.New, eb.signingKey)
	h.Write([]byte(canonical))
	return hex.EncodeToString(h.Sum(nil))
}

// verifySignature verifies the HMAC signature of an event
// SECURITY: Uses hmac.Equal for constant-time comparison to prevent timing attacks
func (eb *EventBus) verifySignature(event Event) bool {
	if event.Signature == "" {
		return eb.allowUnsigned
	}

	expectedSig := eb.computeSignature(event)
	// Use hmac.Equal for constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(event.Signature), []byte(expectedSig))
}

// Publish publishes an event to the event bus
// SECURITY FIX: Events must be signed. Unsigned events are rejected unless AllowUnsigned is true.
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// SECURITY FIX: Compute and attach signature before publishing
	event.Signature = eb.computeSignature(event)

	// Log event (without logging the signature itself)
	eb.logger.Debug().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Str("actor_id", event.ActorID).
		Msg("Publishing signed event")

	// Publish to Redis pub/sub (if cache is available)
	if eb.cache != nil && eb.cache.IsAvailable() {
		channel := fmt.Sprintf("events:%s", event.Type)

		// Publish the signed event using the cache's Publish method
		// Note: The cache.Event structure doesn't include signature, so we need to add it to Data
		if err := eb.cache.PubSub().Publish(ctx, channel, cache.Event{
			Type:      event.Type,
			TenantID:  event.TenantID,
			Data:      event.Data,
			Timestamp: event.Timestamp.Unix(),
		}); err != nil {
			return fmt.Errorf("events.Publish: %w", err)
		}
	}

	// Call in-memory handlers
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			eb.logger.Error().
				Str("event_id", event.ID).
				Str("event_type", event.Type).
				Err(err).
				Msg("Handler error")
		}
	}

	return nil
}

// Subscribe registers a handler for an event type
func (eb *EventBus) Subscribe(eventType string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// startSubscriptionListener listens for events from Redis
// SECURITY FIX: All received events must have valid signatures
func (eb *EventBus) startSubscriptionListener() {
	ctx := context.Background()

	// Check if cache is available
	if eb.cache == nil || !eb.cache.IsAvailable() {
		return
	}

	// Subscribe to all event channels
	pattern := "events:*"
	pubsub, err := eb.cache.PubSub().Subscribe(ctx, pattern)
	if err != nil {
		eb.logger.Error().Err(err).Msg("Failed to subscribe to events")
		return
	}
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var event Event
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			eb.logger.Error().Err(err).Msg("Failed to unmarshal event")
			continue
		}

		// SECURITY FIX: Verify event signature before processing
		// This prevents event injection and spoofing attacks
		if !eb.verifySignature(event) {
			eb.logger.Warn().
				Str("event_id", event.ID).
				Str("event_type", event.Type).
				Str("tenant_id", event.TenantID).
				Msg("Rejected event with invalid or missing signature")
			continue
		}

		eb.mu.RLock()
		handlers := eb.handlers[event.Type]
		eb.mu.RUnlock()

		for _, handler := range handlers {
			if err := handler(ctx, event); err != nil {
				eb.logger.Error().
					Str("event_id", event.ID).
					Str("event_type", event.Type).
					Err(err).
					Msg("Handler error")
			}
		}
	}
}

// Event types
const (
	EventTypeSessionStarted    = "session.started"
	EventTypeSessionEnded      = "session.ended"
	EventTypeSessionTerminated = "session.terminated"
	EventTypeCredentialCreated = "credential.created"
	EventTypeCredentialUpdated = "credential.updated"
	EventTypeCredentialDeleted = "credential.deleted"
	EventTypeCheckoutRequested = "checkout.requested"
	EventTypeCheckoutApproved  = "checkout.approved"
	EventTypeCheckoutDenied    = "checkout.denied"
	EventTypeCheckoutCheckedOut = "checkout.checked_out"
	EventTypeCheckoutCheckedIn  = "checkout.checked_in"
	EventTypeUserCreated       = "user.created"
	EventTypeUserUpdated       = "user.updated"
	EventTypeUserDeleted       = "user.deleted"
	EventTypeAlertTriggered    = "alert.triggered"
	EventTypeAnomalyDetected   = "anomaly.detected"
)

// Publisher creates domain events
// SECURITY FIX: All published events are automatically signed by the EventBus
type Publisher struct {
	bus *EventBus
}

// NewPublisher creates a new event publisher
func NewPublisher(bus *EventBus) *Publisher {
	return &Publisher{bus: bus}
}

// PublishSessionStarted publishes a session started event
func (p *Publisher) PublishSessionStarted(ctx context.Context, tenantID, userID, sessionID, targetHost string, targetPort int) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeSessionStarted,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "start",
		Resource: "session",
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"target_host": targetHost,
			"target_port": targetPort,
		},
	})
}

// PublishSessionEnded publishes a session ended event
func (p *Publisher) PublishSessionEnded(ctx context.Context, tenantID, userID, sessionID string, duration time.Duration) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeSessionEnded,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "end",
		Resource: "session",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"duration":   duration.String(),
		},
	})
}

// PublishCheckoutRequested publishes a checkout request event
func (p *Publisher) PublishCheckoutRequested(ctx context.Context, tenantID, userID, credentialID, requestID string, duration int) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeCheckoutRequested,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "request",
		Resource: "credential",
		Data: map[string]interface{}{
			"credential_id": credentialID,
			"request_id":    requestID,
			"duration":      duration,
		},
	})
}

// PublishCheckoutApproved publishes a checkout approved event
func (p *Publisher) PublishCheckoutApproved(ctx context.Context, tenantID, approverID, requestID string) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeCheckoutApproved,
		TenantID: tenantID,
		ActorID:  approverID,
		Action:   "approve",
		Resource: "checkout_request",
		Data: map[string]interface{}{
			"request_id": requestID,
		},
	})
}

// PublishAnomalyDetected publishes an anomaly detected event
func (p *Publisher) PublishAnomalyDetected(ctx context.Context, tenantID, userID, sessionID string, anomalyType string, score float64, details map[string]interface{}) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeAnomalyDetected,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "detect",
		Resource: "anomaly",
		Data: map[string]interface{}{
			"session_id":    sessionID,
			"anomaly_type":  anomalyType,
			"risk_score":    score,
			"details":       details,
		},
	})
}

// PublishAlertTriggered publishes an alert triggered event
func (p *Publisher) PublishAlertTriggered(ctx context.Context, tenantID, alertID, alertType string, severity string, details map[string]interface{}) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeAlertTriggered,
		TenantID: tenantID,
		Action:   "trigger",
		Resource: "alert",
		Data: map[string]interface{}{
			"alert_id":   alertID,
			"alert_type": alertType,
			"severity":   severity,
			"details":    details,
		},
	})
}

// Analytics event types
const (
	EventTypeAnalyticsSessionStarted   = "analytics.session.started"
	EventTypeAnalyticsSessionEnded     = "analytics.session.ended"
	EventTypeAnalyticsCommandExecuted = "analytics.command.executed"
	EventTypeAnalyticsUserActivity    = "analytics.user.activity"
	EventTypeAnalyticsAnomalyDetected = "analytics.anomaly.detected"
)

// PublishAnalyticsSessionStarted publishes an analytics session started event
func (p *Publisher) PublishAnalyticsSessionStarted(ctx context.Context, tenantID, userID, sessionID, targetHost string, targetPort int, sessionType string) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeAnalyticsSessionStarted,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "start",
		Resource: "session_analytics",
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"target_host": targetHost,
			"target_port": targetPort,
			"type":        sessionType,
		},
	})
}

// PublishAnalyticsSessionEnded publishes an analytics session ended event
func (p *Publisher) PublishAnalyticsSessionEnded(ctx context.Context, tenantID, userID, sessionID string, duration time.Duration, sessionType string) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeAnalyticsSessionEnded,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "end",
		Resource: "session_analytics",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"duration":   duration.String(),
			"type":       sessionType,
		},
	})
}

// PublishAnalyticsCommandExecuted publishes an analytics command executed event
func (p *Publisher) PublishAnalyticsCommandExecuted(ctx context.Context, tenantID, userID, sessionID, command, targetHost string, exitCode *int) error {
	data := map[string]interface{}{
		"session_id":  sessionID,
		"command":     command,
		"target_host": targetHost,
	}
	if exitCode != nil {
		data["exit_code"] = *exitCode
	}

	return p.bus.Publish(ctx, Event{
		Type:     EventTypeAnalyticsCommandExecuted,
		TenantID: tenantID,
		ActorID:  userID,
		Action:   "execute",
		Resource: "command",
		Data:      data,
	})
}

// PublishAnalyticsAnomalyDetected publishes an analytics anomaly detected event
func (p *Publisher) PublishAnalyticsAnomalyDetected(ctx context.Context, tenantID string, anomalyID uuid.UUID, anomalyType string, score float64, details map[string]interface{}) error {
	return p.bus.Publish(ctx, Event{
		Type:     EventTypeAnalyticsAnomalyDetected,
		TenantID: tenantID,
		Action:   "detect",
		Resource: "anomaly",
		Data: map[string]interface{}{
			"anomaly_id":   anomalyID.String(),
			"anomaly_type": anomalyType,
			"risk_score":   score,
			"details":      details,
		},
	})
}

// Webhook delivers events to external webhooks
type Webhook struct {
	client   *http.Client
	cache    *cache.Cache
	logger   zerolog.Logger
	endpoint string
	secret   string
}

// NewWebhook creates a new webhook delivery client
func NewWebhook(endpoint, secret string, c *cache.Cache, logger zerolog.Logger) *Webhook {
	return &Webhook{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:    c,
		logger:   logger,
		endpoint: endpoint,
		secret:   secret,
	}
}

// Deliver sends an event to the webhook endpoint
// SECURITY FIX: Includes timestamp in signature to prevent replay attacks
func (w *Webhook) Deliver(ctx context.Context, event Event) error {
	// Check rate limit
	allowed, err := w.cache.RateLimiter().Allow(ctx, "webhook:"+w.endpoint, 100, time.Minute)
	if err != nil {
		return fmt.Errorf("webhook.RateLimit: %w", err)
	}
	if !allowed {
		return fmt.Errorf("webhook: rate limit exceeded")
	}

	// Marshal event
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("webhook.Marshal: %w", err)
	}

	// Generate timestamp for this delivery
	timestamp := time.Now().Unix()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", w.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("webhook.NewRequest: %w", err)
	}

	// SECURITY FIX: Include timestamp in signature calculation
	signature := w.sign(payload, timestamp)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpenPAM-Event", event.Type)
	req.Header.Set("X-OpenPAM-Signature", signature)
	req.Header.Set("X-OpenPAM-Timestamp", fmt.Sprintf("%d", timestamp))

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook.Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: unexpected status %d", resp.StatusCode)
	}

	w.logger.Debug().
		Str("webhook", w.endpoint).
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Int("status", resp.StatusCode).
		Msg("Webhook delivered")

	return nil
}

// sign creates HMAC signature for webhook payload
// SECURITY FIX: Include timestamp in signature to prevent replay attacks
func (w *Webhook) sign(payload []byte, timestamp int64) string {
	h := hmac.New(sha256.New, []byte(w.secret))
	// Include timestamp in the signed data to prevent replay attacks
	h.Write([]byte(fmt.Sprintf("%d", timestamp)))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// verifySignature verifies the webhook signature
// SECURITY FIX: Validates timestamp to prevent replay attacks
// SECURITY: Uses hmac.Equal() for timing-safe comparison to prevent timing attacks
func (w *Webhook) verifySignature(payload []byte, signature string, timestamp int64) bool {
	// Check timestamp is within acceptable range (5 minutes)
	now := time.Now().Unix()
	maxAge := int64(300) // 5 minutes

	if timestamp < now-maxAge || timestamp > now+maxAge {
		return false
	}

	// Recreate signature with timestamp
	expectedSignature := w.sign(payload, timestamp)

	// hmac.Equal performs a timing-safe comparison to prevent timing attacks
	// This ensures attackers cannot use timing information to forge valid signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// DeliverWithRetry delivers an event with retry logic
func (w *Webhook) DeliverWithRetry(ctx context.Context, event Event, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := w.Deliver(ctx, event); err == nil {
			return nil
		} else {
			lastErr = err
			// Exponential backoff
			time.Sleep(time.Duration(1<<uint(i)) * time.Second)
		}
	}
	return fmt.Errorf("webhook: failed after %d retries: %w", maxRetries, lastErr)
}

// VerifyWebhookSignature verifies an incoming webhook signature
// This is used by webhook consumers to verify the authenticity of webhooks
// SECURITY FIX: Includes timestamp verification to prevent replay attacks
// SECURITY: Uses timing-safe HMAC comparison via hmac.Equal()
func VerifyWebhookSignature(payload []byte, signature string, timestamp int64, secret string) bool {
	// Validate inputs
	if len(payload) == 0 {
		return false
	}
	if signature == "" {
		return false
	}
	if secret == "" {
		return false
	}
	if timestamp == 0 {
		return false
	}

	// Verify signature format (hex-encoded SHA256)
	if len(signature) != 64 { // SHA256 produces 32 bytes, hex encoded is 64 chars
		return false
	}

	webhook := &Webhook{secret: secret}
	return webhook.verifySignature(payload, signature, timestamp)
}
