package infra

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
)

var _ app.SessionStore = (*RedisSessionStore)(nil)

// RedisSessionStore keeps sessions in Redis. The key has the form admin:session:<hash of the token>.
// Redis has the hash of the token, not the token, so a person who reads Redis cannot use a session.
type RedisSessionStore struct {
	client *redis.Client
}

// NewRedisSessionStore builds a store on a Redis client. Give it a client for the cache database.
func NewRedisSessionStore(client *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client}
}

func sessionKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "admin:session:" + hex.EncodeToString(sum[:])
}

// Create starts a session for the Admin. The token has 32 random bytes.
func (s *RedisSessionStore) Create(ctx context.Context, adminID uuid.UUID, lifetime time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if err := s.client.Set(ctx, sessionKey(token), adminID.String(), lifetime).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// Get gives the Admin of the session. found is false when the token is not known or the session ended.
func (s *RedisSessionStore) Get(ctx context.Context, token string) (uuid.UUID, bool, error) {
	value, err := s.client.Get(ctx, sessionKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	adminID, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, false, nil
	}
	return adminID, true, nil
}

// Delete ends the session.
func (s *RedisSessionStore) Delete(ctx context.Context, token string) error {
	return s.client.Del(ctx, sessionKey(token)).Err()
}
