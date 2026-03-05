package circuitbreaker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestCircuitBreakerClosedState(t *testing.T) {
	logger := zerolog.Nop()
	cb := New("test", DefaultConfig(), logger)

	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.State())
	}

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCircuitBreakerOpensAfterMaxFailures(t *testing.T) {
	logger := zerolog.Nop()
	cfg := Config{
		MaxFailures:   3,
		Timeout:       100 * time.Millisecond,
		HalfOpenMax:   1,
		ResetInterval: time.Minute,
	}
	cb := New("test", cfg, logger)

	// Cause failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure %d", i)
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen after %d failures, got %v", 3, cb.State())
	}

	// Should be rejected
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error from open circuit breaker")
	}
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	logger := zerolog.Nop()
	cfg := Config{
		MaxFailures:   2,
		Timeout:       50 * time.Millisecond,
		HalfOpenMax:   1,
		ResetInterval: time.Minute,
	}
	cb := New("test", cfg, logger)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("fail")
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen after timeout, got %v", cb.State())
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	logger := zerolog.Nop()
	cfg := Config{
		MaxFailures:   2,
		Timeout:       50 * time.Millisecond,
		HalfOpenMax:   2,
		ResetInterval: time.Minute,
	}
	cb := New("test", cfg, logger)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("fail")
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Succeed in half-open to recover
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatalf("expected success in half-open, got %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed after recovery, got %v", cb.State())
	}
}

func TestBreakerManager(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(DefaultConfig(), logger)

	cb1 := manager.Get("service-a")
	cb2 := manager.Get("service-b")
	cb3 := manager.Get("service-a") // Should return same breaker

	if cb1 != cb3 {
		t.Fatal("expected same circuit breaker for same name")
	}
	if cb1 == cb2 {
		t.Fatal("expected different circuit breakers for different names")
	}

	summary := manager.Summary()
	if len(summary) != 2 {
		t.Fatalf("expected 2 breakers in summary, got %d", len(summary))
	}
}

func TestExecuteWithFallback(t *testing.T) {
	logger := zerolog.Nop()
	cfg := Config{
		MaxFailures:   1,
		Timeout:       100 * time.Millisecond,
		HalfOpenMax:   1,
		ResetInterval: time.Minute,
	}
	cb := New("test", cfg, logger)

	// Open the circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return fmt.Errorf("fail")
	})

	// Execute with fallback
	fallbackCalled := false
	err := cb.ExecuteWithFallback(context.Background(),
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context, err error) error {
			fallbackCalled = true
			return nil
		},
	)

	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if !fallbackCalled {
		t.Fatal("expected fallback to be called")
	}
}
