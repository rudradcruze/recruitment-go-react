package infra

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

var _ app.FailedLogins = (*RedisFailedLogins)(nil)

// RedisFailedLogins counts failed logins in Redis. The key has the form admin:failed-logins:<email address>.
type RedisFailedLogins struct {
	client *redis.Client
}

// NewRedisFailedLogins builds a counter on a Redis client. Give it a client for the cache database.
func NewRedisFailedLogins(client *redis.Client) *RedisFailedLogins {
	return &RedisFailedLogins{client: client}
}

func failedLoginsKey(emailAddress domain.EmailAddress) string {
	return "admin:failed-logins:" + string(emailAddress)
}

// Count gives the failed logins in the window and the time until the count expires.
func (f *RedisFailedLogins) Count(ctx context.Context, emailAddress domain.EmailAddress) (int, time.Duration, error) {
	key := failedLoginsKey(emailAddress)
	var count *redis.StringCmd
	var ttl *redis.DurationCmd
	_, err := f.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		count = p.Get(ctx, key)
		ttl = p.PTTL(ctx, key)
		return nil
	})
	if errors.Is(err, redis.Nil) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	n, err := count.Int()
	if err != nil {
		return 0, 0, err
	}
	return n, max(ttl.Val(), 0), nil
}

// Add counts one failed login. The first failure sets the expiry, and the next failures keep it.
// The 2 commands run in one transaction, so a crash cannot leave a count without an expiry.
func (f *RedisFailedLogins) Add(ctx context.Context, emailAddress domain.EmailAddress, window time.Duration) error {
	key := failedLoginsKey(emailAddress)
	_, err := f.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.SetNX(ctx, key, 0, window)
		p.Incr(ctx, key)
		return nil
	})
	return err
}
