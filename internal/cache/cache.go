package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// Cache wraps Redis client with PAM-specific operations
type Cache struct {
	client *redis.Client
	logger zerolog.Logger
}

// New creates a new Redis cache client
func New(cfg Config, logger zerolog.Logger) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache.Ping: %w", err)
	}

	logger.Info().Str("addr", client.Options().Addr).Msg("Redis connection established")

	return &Cache{client: client, logger: logger}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// Client returns the underlying Redis client
func (c *Cache) Client() *redis.Client {
	return c.client
}

// Health checks Redis health
func (c *Cache) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Get retrieves a value
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("cache.Get: %w", err)
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value with expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache.Marshal: %w", err)
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

// Delete removes a key
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// DeleteByPattern removes all keys matching a pattern
func (c *Cache) DeleteByPattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("cache.Del: %w", err)
		}
	}
	return iter.Err()
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) bool {
	n, _ := c.client.Exists(ctx, key).Result()
	return n > 0
}

// TTL returns the remaining time to live
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// SessionStore handles session-related cache operations
type SessionStore struct {
	cache *Cache
}

// Sessions returns a session store
func (c *Cache) Sessions() *SessionStore {
	return &SessionStore{cache: c}
}

// Session represents a cached user session
type Session struct {
	UserID       string    `json:"user_id"`
	TenantID     string    `json:"tenant_id"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
	MFAVerified  bool      `json:"mfa_verified"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}

// SetSession stores a session
func (s *SessionStore) SetSession(ctx context.Context, sessionID string, session Session, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.cache.Set(ctx, key, session, expiration)
}

// GetSession retrieves a session
func (s *SessionStore) GetSession(ctx context.Context, sessionID string) (Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	var session Session
	err := s.cache.Get(ctx, key, &session)
	return session, err
}

// DeleteSession removes a session
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.cache.Delete(ctx, key)
}

// DeleteUserSessions removes all sessions for a user
func (s *SessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("session:*")
	iter := s.cache.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		var session Session
		if err := s.cache.Get(ctx, iter.Val(), &session); err == nil && session.UserID == userID {
			_ = s.cache.Delete(ctx, iter.Val())
		}
	}
	return iter.Err()
}

// RateLimiter handles rate limiting via Redis
type RateLimiter struct {
	cache *Cache
}

// RateLimiter returns a rate limiter
func (c *Cache) RateLimiter() *RateLimiter {
	return &RateLimiter{cache: c}
}

// Allow checks if a request is allowed using sliding window
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	pipe := r.cache.client.Pipeline()
	keyCurrent := fmt.Sprintf("ratelimit:%s:current", key)
	keyHistory := fmt.Sprintf("ratelimit:%s:history", key)

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, keyHistory, "0", fmt.Sprintf("%d", windowStart))

	// Count current requests
	incrCmd := pipe.Incr(ctx, keyCurrent)
	pipe.Expire(ctx, keyCurrent, window)

	// Add to history
	pipe.ZAdd(ctx, keyHistory, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, keyHistory, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("ratelimit.Pipeline: %w", err)
	}

	count := incrCmd.Val()
	return count <= int64(limit), nil
}

// Reset resets the rate limit for a key
func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	keyCurrent := fmt.Sprintf("ratelimit:%s:current", key)
	keyHistory := fmt.Sprintf("ratelimit:%s:history", key)
	return r.cache.client.Del(ctx, keyCurrent, keyHistory).Err()
}

// PubSub handles Redis pub/sub for events
type PubSub struct {
	cache *Cache
}

// PubSub returns a pub/sub handler
func (c *Cache) PubSub() *PubSub {
	return &PubSub{cache: c}
}

// Event represents a real-time event
type Event struct {
	Type      string                 `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// Publish publishes an event to a channel
func (p *PubSub) Publish(ctx context.Context, channel string, event Event) error {
	event.Timestamp = time.Now().Unix()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("pubsub.Marshal: %w", err)
	}
	return p.cache.client.Publish(ctx, channel, data).Err()
}

// Subscribe subscribes to a channel
func (p *PubSub) Subscribe(ctx context.Context, channel string) (*redis.PubSub, error) {
	pubsub := p.cache.client.Subscribe(ctx, channel)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("pubsub.Subscribe: %w", err)
	}
	return pubsub, nil
}

// TokenStore handles token blacklist for logout
type TokenStore struct {
	cache *Cache
}

// Tokens returns a token store
func (c *Cache) Tokens() *TokenStore {
	return &TokenStore{cache: c}
}

// RevokeToken adds a token to the blacklist
func (t *TokenStore) RevokeToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	key := fmt.Sprintf("revoked:%s", tokenID)
	return t.cache.Set(ctx, key, true, expiration)
}

// IsRevoked checks if a token is revoked
func (t *TokenStore) IsRevoked(ctx context.Context, tokenID string) bool {
	key := fmt.Sprintf("revoked:%s", tokenID)
	return t.cache.Exists(ctx, key)
}
