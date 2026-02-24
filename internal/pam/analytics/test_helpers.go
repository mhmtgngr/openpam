package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// mockCache is a mock cache implementation for testing
// This is shared across all test files in the analytics package
type testMockCache struct {
	data    map[string][]byte
	getErr  error
	setErr  error
	delErr  error
	healthErr error
}

func newTestMockCache() *testMockCache {
	return &testMockCache{
		data: make(map[string][]byte),
	}
}

func (m *testMockCache) Get(ctx context.Context, key string, dest interface{}) error {
	if m.getErr != nil {
		return m.getErr
	}
	data, ok := m.data[key]
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	return json.Unmarshal(data, dest)
}

func (m *testMockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.data[key] = data
	return nil
}

func (m *testMockCache) Delete(ctx context.Context, key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	delete(m.data, key)
	return nil
}

func (m *testMockCache) Exists(ctx context.Context, key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *testMockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *testMockCache) Close() error {
	return nil
}

func (m *testMockCache) Health(ctx context.Context) error {
	return m.healthErr
}

func (m *testMockCache) DeleteByPattern(ctx context.Context, pattern string) error {
	return nil
}
