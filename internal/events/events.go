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
type EventBus struct {
	cache   *cache.Cache
	logger  zerolog.Logger
	handlers map[string][]Handler
	mu      sync.RWMutex
}

// Handler processes an event
type Handler func(ctx context.Context, event Event) error

// Event represents a domain event
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
}

// New creates a new event bus
func New(c *cache.Cache, logger zerolog.Logger) *EventBus {
	bus := &EventBus{
		cache:   c,
		logger:  logger,
		handlers: make(map[string][]Handler),
	}
	go bus.startSubscriptionListener()
	return bus
}

// Publish publishes an event to the event bus
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Log event
	eb.logger.Debug().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("tenant_id", event.TenantID).
		Str("actor_id", event.ActorID).
		Msg("Publishing event")

	// Publish to Redis pub/sub (if cache is available)
	if eb.cache != nil && eb.cache.IsAvailable() {
		channel := fmt.Sprintf("events:%s", event.Type)
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

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", w.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("webhook.NewRequest: %w", err)
	}

	// Add signature header
	signature := w.sign(payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpenPAM-Event", event.Type)
	req.Header.Set("X-OpenPAM-Signature", signature)
	req.Header.Set("X-OpenPAM-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

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
func (w *Webhook) sign(payload []byte) string {
	h := hmac.New(sha256.New, []byte(w.secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
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
