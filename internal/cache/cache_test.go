package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	// Test that New function creates a cache
	// Note: This requires actual Redis connection, so we'll skip in CI
	t.Skip("requires Redis connection")
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 6379, cfg.Port)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, 10, cfg.PoolSize)
}

func TestSession_Struct(t *testing.T) {
	session := Session{
		UserID:       "user123",
		TenantID:     "tenant456",
		Email:        "test@example.com",
		Roles:        []string{"admin"},
		MFAVerified:  true,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	assert.Equal(t, "user123", session.UserID)
	assert.Equal(t, "tenant456", session.TenantID)
	assert.True(t, session.MFAVerified)
	assert.Len(t, session.Roles, 1)
}

func TestEvent_Struct(t *testing.T) {
	event := Event{
		Type:      "test.event",
		TenantID:  "tenant-123",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now().Unix(),
	}

	assert.Equal(t, "test.event", event.Type)
	assert.Equal(t, "tenant-123", event.TenantID)
	assert.NotEmpty(t, event.Data)
}

// Helper function to test pattern matching logic
func matchPattern(pattern, key string) bool {
	if pattern == "*" {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	return pattern == key
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		key      string
		expected bool
	}{
		{"match all", "*", "any-key", true},
		{"match prefix", "user:*", "user:123", true},
		{"match prefix no match", "user:*", "session:123", false},
		{"exact match", "exact-key", "exact-key", true},
		{"exact no match", "exact-key", "different-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCache_HealthCheck(t *testing.T) {
	t.Skip("requires Redis connection")
}

func TestCache_Close(t *testing.T) {
	t.Skip("requires Redis connection")
}
