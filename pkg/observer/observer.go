package observer

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

// DomainEvent represents a typed domain event
type DomainEvent interface {
	EventName() string
	TenantID() string
}

// EventHandler handles a specific domain event type
type EventHandler[T DomainEvent] func(ctx context.Context, event T) error

// handler wraps a typed handler into an untyped one
type handler struct {
	name    string
	handle  func(ctx context.Context, event DomainEvent) error
	eventFn func() string
}

// EventDispatcher dispatches domain events to registered handlers
type EventDispatcher struct {
	handlers map[string][]handler
	mu       sync.RWMutex
	logger   zerolog.Logger
	async    bool
}

// NewEventDispatcher creates a new event dispatcher
func NewEventDispatcher(logger zerolog.Logger) *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string][]handler),
		logger:   logger,
	}
}

// NewAsyncEventDispatcher creates an event dispatcher that runs handlers asynchronously
func NewAsyncEventDispatcher(logger zerolog.Logger) *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string][]handler),
		logger:   logger,
		async:    true,
	}
}

// Subscribe registers a typed handler for a domain event
func Subscribe[T DomainEvent](d *EventDispatcher, handlerName string, fn EventHandler[T]) {
	var zero T
	eventName := zero.EventName()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers[eventName] = append(d.handlers[eventName], handler{
		name: handlerName,
		handle: func(ctx context.Context, event DomainEvent) error {
			typed, ok := event.(T)
			if !ok {
				return fmt.Errorf("observer: expected %T, got %T", zero, event)
			}
			return fn(ctx, typed)
		},
	})

	d.logger.Debug().
		Str("event", eventName).
		Str("handler", handlerName).
		Msg("Event handler registered")
}

// Dispatch sends a domain event to all registered handlers
func (d *EventDispatcher) Dispatch(ctx context.Context, event DomainEvent) error {
	d.mu.RLock()
	handlers := d.handlers[event.EventName()]
	d.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	if d.async {
		for _, h := range handlers {
			go func(h handler) {
				if err := h.handle(ctx, event); err != nil {
					d.logger.Error().
						Str("event", event.EventName()).
						Str("handler", h.name).
						Err(err).
						Msg("Async event handler failed")
				}
			}(h)
		}
		return nil
	}

	var firstErr error
	for _, h := range handlers {
		if err := h.handle(ctx, event); err != nil {
			d.logger.Error().
				Str("event", event.EventName()).
				Str("handler", h.name).
				Err(err).
				Msg("Event handler failed")
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// --- Concrete PAM Domain Events ---

// SessionStartedEvent is emitted when a privileged session begins
type SessionStartedEvent struct {
	Tenant    string
	UserID    string
	SessionID string
	TargetID  string
	Type      string
}

func (e SessionStartedEvent) EventName() string { return "session.started" }
func (e SessionStartedEvent) TenantID() string  { return e.Tenant }

// SessionEndedEvent is emitted when a privileged session ends
type SessionEndedEvent struct {
	Tenant    string
	UserID    string
	SessionID string
	Duration  string
}

func (e SessionEndedEvent) EventName() string { return "session.ended" }
func (e SessionEndedEvent) TenantID() string  { return e.Tenant }

// CredentialCheckedOutEvent is emitted when a credential is checked out
type CredentialCheckedOutEvent struct {
	Tenant       string
	UserID       string
	CredentialID string
	CheckoutID   string
	Duration     int
	IsBreakGlass bool
}

func (e CredentialCheckedOutEvent) EventName() string { return "credential.checked_out" }
func (e CredentialCheckedOutEvent) TenantID() string  { return e.Tenant }

// CredentialCheckedInEvent is emitted when a credential is returned
type CredentialCheckedInEvent struct {
	Tenant       string
	UserID       string
	CredentialID string
	CheckoutID   string
}

func (e CredentialCheckedInEvent) EventName() string { return "credential.checked_in" }
func (e CredentialCheckedInEvent) TenantID() string  { return e.Tenant }

// CredentialRotatedEvent is emitted after credential rotation
type CredentialRotatedEvent struct {
	Tenant       string
	CredentialID string
	Strategy     string
	NextRotation string
}

func (e CredentialRotatedEvent) EventName() string { return "credential.rotated" }
func (e CredentialRotatedEvent) TenantID() string  { return e.Tenant }

// AnomalyDetectedEvent is emitted when suspicious activity is detected
type AnomalyDetectedEvent struct {
	Tenant    string
	UserID    string
	SessionID string
	Type      string
	Severity  string
	Score     float64
	Details   map[string]interface{}
}

func (e AnomalyDetectedEvent) EventName() string { return "anomaly.detected" }
func (e AnomalyDetectedEvent) TenantID() string  { return e.Tenant }

// PolicyViolationEvent is emitted when a policy violation occurs
type PolicyViolationEvent struct {
	Tenant    string
	UserID    string
	PolicyID  string
	Action    string
	Resource  string
	Violation string
}

func (e PolicyViolationEvent) EventName() string { return "policy.violation" }
func (e PolicyViolationEvent) TenantID() string  { return e.Tenant }
