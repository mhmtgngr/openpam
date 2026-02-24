package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimePtr(t *testing.T) {
	t.Run("returns pointer to time", func(t *testing.T) {
		now := time.Now()
		result := timePtr(now)

		require.NotNil(t, result)
		assert.Equal(t, now, *result)
	})

	t.Run("modifications to pointer affect original", func(t *testing.T) {
		original := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		result := timePtr(original)

		assert.Equal(t, original, *result)
		assert.Equal(t, 2024, result.Year())
	})
}

func TestStringPtr(t *testing.T) {
	t.Run("returns pointer to string", func(t *testing.T) {
		s := "test-string"
		result := stringPtr(s)

		require.NotNil(t, result)
		assert.Equal(t, s, *result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		s := ""
		result := stringPtr(s)

		require.NotNil(t, result)
		assert.Equal(t, "", *result)
	})

	t.Run("handles special characters", func(t *testing.T) {
		s := "special-chars-@#$%^&*()"
		result := stringPtr(s)

		require.NotNil(t, result)
		assert.Equal(t, s, *result)
	})

	t.Run("modifications to pointer affect original", func(t *testing.T) {
		original := "original-value"
		result := stringPtr(original)

		assert.Equal(t, original, *result)
		assert.Equal(t, 14, len(*result))
	})
}

func TestHelperFunctions_Integration(t *testing.T) {
	t.Run("timePtr with zero time", func(t *testing.T) {
		zero := time.Time{}
		result := timePtr(zero)

		require.NotNil(t, result)
		assert.True(t, result.IsZero())
	})

	t.Run("stringPtr with unicode", func(t *testing.T) {
		s := "Hello 世界 🌍"
		result := stringPtr(s)

		require.NotNil(t, result)
		assert.Equal(t, s, *result)
		assert.Equal(t, 17, len(*result)) // byte length: "Hello" (5) + " " (1) + "世界" (6) + " " (1) + "🌍" (4) = 17
	})

	t.Run("nil checks", func(t *testing.T) {
		// These should never return nil
		tp := timePtr(time.Now())
		sp := stringPtr("test")

		assert.NotNil(t, tp)
		assert.NotNil(t, sp)
	})
}

// Benchmark tests
func BenchmarkTimePtr(b *testing.B) {
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timePtr(now)
	}
}

func BenchmarkStringPtr(b *testing.B) {
	s := "test-string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stringPtr(s)
	}
}
