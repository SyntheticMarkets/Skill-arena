package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Client interface {
	Set(context.Context, string, string, time.Duration) error
	Get(context.Context, string) (string, bool, error)
	Del(context.Context, string) error
	Lock(context.Context, string, time.Duration) (string, bool, error)
	Unlock(context.Context, string, string) error
	Allow(context.Context, string, int, time.Duration) (bool, error)
	Health(context.Context) error
}

type MemoryClient struct {
	mu    sync.Mutex
	items map[string]memoryItem
	locks map[string]memoryLock
}

type memoryItem struct {
	value     string
	expiresAt time.Time
}

type memoryLock struct {
	token     string
	expiresAt time.Time
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{items: map[string]memoryItem{}, locks: map[string]memoryLock{}}
}

func (c *MemoryClient) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item := memoryItem{value: value}
	if ttl > 0 {
		item.expiresAt = time.Now().UTC().Add(ttl)
	}
	c.items[key] = item
	return nil
}

func (c *MemoryClient) Get(ctx context.Context, key string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok {
		return "", false, nil
	}
	if !item.expiresAt.IsZero() && item.expiresAt.Before(time.Now().UTC()) {
		delete(c.items, key)
		return "", false, nil
	}
	return item.value, true, nil
}

func (c *MemoryClient) Del(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

func (c *MemoryClient) Lock(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if ttl <= 0 {
		return "", false, errors.New("lock ttl must be positive")
	}
	token, err := lockToken()
	if err != nil {
		return "", false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().UTC()
	if lock, ok := c.locks[key]; ok && lock.expiresAt.After(now) {
		return "", false, nil
	}
	c.locks[key] = memoryLock{token: token, expiresAt: now.Add(ttl)}
	return token, true, nil
}

func (c *MemoryClient) Unlock(ctx context.Context, key, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	lock, ok := c.locks[key]
	if !ok || lock.token != token {
		return errors.New("lock is not owned by token")
	}
	delete(c.locks, key)
	return nil
}

func (c *MemoryClient) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if limit <= 0 || window <= 0 {
		return false, errors.New("rate limit and window must be positive")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().UTC()
	item, ok := c.items[key]
	if !ok || (!item.expiresAt.IsZero() && !item.expiresAt.After(now)) {
		c.items[key] = memoryItem{value: "1", expiresAt: now.Add(window)}
		return true, nil
	}
	count, err := strconv.Atoi(item.value)
	if err != nil {
		return false, err
	}
	count++
	item.value = strconv.Itoa(count)
	c.items[key] = item
	return count <= limit, nil
}

func (c *MemoryClient) Health(ctx context.Context) error {
	return ctx.Err()
}

type NetworkClient struct {
	URL string
}

var networkClients sync.Map

func (c NetworkClient) client() (*goredis.Client, error) {
	if strings.TrimSpace(c.URL) == "" {
		return nil, errors.New("redis url is required")
	}
	if existing, ok := networkClients.Load(c.URL); ok {
		return existing.(*goredis.Client), nil
	}
	options, err := goredis.ParseURL(c.URL)
	if err != nil {
		return nil, err
	}
	// Coordination commands do not consume RESP3 push notifications. RESP2
	// avoids push-processing reads on every lock and rate-limit operation.
	options.Protocol = 2
	options.DisableIdentity = true
	options.PoolSize = 256
	options.MinIdleConns = 64
	options.MaxConcurrentDials = 64
	client := goredis.NewClient(options)
	existing, loaded := networkClients.LoadOrStore(c.URL, client)
	if loaded {
		_ = client.Close()
		return existing.(*goredis.Client), nil
	}
	return client, nil
}

func (c NetworkClient) Health(ctx context.Context) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	return client.Ping(ctx).Err()
}

func (c NetworkClient) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	return client.Set(ctx, key, value, ttl).Err()
}

func (c NetworkClient) Get(ctx context.Context, key string) (string, bool, error) {
	client, err := c.client()
	if err != nil {
		return "", false, err
	}
	value, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return "", false, nil
		}
		return "", false, err
	}
	return value, true, nil
}

func (c NetworkClient) Del(ctx context.Context, key string) error {
	client, err := c.client()
	if err != nil {
		return err
	}
	return client.Del(ctx, key).Err()
}

func (c NetworkClient) Lock(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if ttl <= 0 {
		return "", false, errors.New("lock ttl must be positive")
	}
	token, err := lockToken()
	if err != nil {
		return "", false, err
	}
	client, err := c.client()
	if err != nil {
		return "", false, err
	}
	acquired, err := client.SetNX(ctx, "lock:"+key, token, ttl).Result()
	return token, acquired, err
}

func (c NetworkClient) Unlock(ctx context.Context, key, token string) error {
	const script = `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) else return 0 end`
	client, err := c.client()
	if err != nil {
		return err
	}
	value, err := client.Eval(ctx, script, []string{"lock:" + key}, token).Int64()
	if err != nil {
		return err
	}
	if value != 1 {
		return errors.New("lock is not owned by token")
	}
	return nil
}

func lockToken() (string, error) {
	var value [24]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func (c NetworkClient) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 || window <= 0 {
		return false, errors.New("rate limit and window must be positive")
	}
	const script = `local count = redis.call('INCR', KEYS[1]); if count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]); end; return count`
	client, err := c.client()
	if err != nil {
		return false, err
	}
	count, err := client.Eval(
		ctx, script, []string{key}, strconv.FormatInt(window.Milliseconds(), 10),
	).Int64()
	if err != nil {
		return false, err
	}
	return count <= int64(limit), nil
}
