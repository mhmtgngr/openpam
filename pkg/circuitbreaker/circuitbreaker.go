package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed   State = iota // Normal operation, requests flow through
	StateOpen                  // Requests are blocked
	StateHalfOpen              // Limited requests allowed to test recovery
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures   int           // Failures before opening circuit
	Timeout       time.Duration // How long to stay open before half-open
	HalfOpenMax   int           // Max requests allowed in half-open state
	ResetInterval time.Duration // Interval to reset failure count in closed state
}

// DefaultConfig returns sensible defaults for PAM services
func DefaultConfig() Config {
	return Config{
		MaxFailures:   5,
		Timeout:       30 * time.Second,
		HalfOpenMax:   2,
		ResetInterval: 60 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern for service calls
type CircuitBreaker struct {
	name           string
	config         Config
	state          State
	failures       int
	successes      int
	lastFailure    time.Time
	lastStateChange time.Time
	mu             sync.RWMutex
	logger         zerolog.Logger
}

// New creates a new circuit breaker
func New(name string, cfg Config, logger zerolog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		config:          cfg,
		state:           StateClosed,
		lastStateChange: time.Now(),
		logger:          logger,
	}
}

// Execute runs a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.allowRequest() {
		cb.logger.Warn().
			Str("circuit", cb.name).
			Str("state", cb.State().String()).
			Msg("Circuit breaker rejected request")
		return fmt.Errorf("circuitbreaker: %s is open", cb.name)
	}

	err := fn(ctx)

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// ExecuteWithFallback runs a function with a fallback when the circuit is open
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func(ctx context.Context) error, fallback func(ctx context.Context, err error) error) error {
	err := cb.Execute(ctx, fn)
	if err != nil {
		return fallback(ctx, err)
	}
	return nil
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateOpen && time.Since(cb.lastStateChange) >= cb.config.Timeout {
		return StateHalfOpen
	}
	return cb.state
}

// Failures returns the current failure count
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) >= cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			cb.lastStateChange = time.Now()
			cb.logger.Info().Str("circuit", cb.name).Msg("Circuit breaker transitioning to half-open")
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successes < cb.config.HalfOpenMax
	}
	return false
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		cb.logger.Warn().
			Str("circuit", cb.name).
			Msg("Circuit breaker reopened from half-open")
		return
	}

	if cb.failures >= cb.config.MaxFailures {
		cb.state = StateOpen
		cb.lastStateChange = time.Now()
		cb.logger.Warn().
			Str("circuit", cb.name).
			Int("failures", cb.failures).
			Msg("Circuit breaker opened")
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.HalfOpenMax {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
			cb.lastStateChange = time.Now()
			cb.logger.Info().Str("circuit", cb.name).Msg("Circuit breaker closed (recovered)")
		}
		return
	}

	// Reset failures on success in closed state
	if cb.state == StateClosed && time.Since(cb.lastFailure) >= cb.config.ResetInterval {
		cb.failures = 0
	}
}

// BreakerManager manages multiple circuit breakers
type BreakerManager struct {
	breakers map[string]*CircuitBreaker
	config   Config
	logger   zerolog.Logger
	mu       sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager(cfg Config, logger zerolog.Logger) *BreakerManager {
	return &BreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   cfg,
		logger:   logger,
	}
}

// Get returns (or creates) a circuit breaker for the given service
func (m *BreakerManager) Get(name string) *CircuitBreaker {
	m.mu.RLock()
	if cb, ok := m.breakers[name]; ok {
		m.mu.RUnlock()
		return cb
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := m.breakers[name]; ok {
		return cb
	}

	cb := New(name, m.config, m.logger)
	m.breakers[name] = cb
	return cb
}

// Summary returns state of all circuit breakers
func (m *BreakerManager) Summary() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string, len(m.breakers))
	for name, cb := range m.breakers {
		result[name] = cb.State().String()
	}
	return result
}
