package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client interface {
	Set(context.Context, string, string, time.Duration) error
	Get(context.Context, string) (string, bool, error)
	Del(context.Context, string) error
	Lock(context.Context, string, time.Duration) (bool, error)
	Unlock(context.Context, string) error
	Health(context.Context) error
}

type MemoryClient struct {
	mu    sync.Mutex
	items map[string]memoryItem
	locks map[string]time.Time
}

type memoryItem struct {
	value     string
	expiresAt time.Time
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{items: map[string]memoryItem{}, locks: map[string]time.Time{}}
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

func (c *MemoryClient) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if ttl <= 0 {
		return false, errors.New("lock ttl must be positive")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().UTC()
	if expires, ok := c.locks[key]; ok && expires.After(now) {
		return false, nil
	}
	c.locks[key] = now.Add(ttl)
	return true, nil
}

func (c *MemoryClient) Unlock(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.locks, key)
	return nil
}

func (c *MemoryClient) Health(ctx context.Context) error {
	return ctx.Err()
}

type NetworkClient struct {
	URL string
}

func (c NetworkClient) Health(ctx context.Context) error {
	_, err := c.command(ctx, "PING")
	return err
}

func (c NetworkClient) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	args := []string{"SET", key, value}
	if ttl > 0 {
		args = append(args, "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	}
	_, err := c.command(ctx, args...)
	return err
}

func (c NetworkClient) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := c.command(ctx, "GET", key)
	if err != nil {
		if errors.Is(err, errNil) {
			return "", false, nil
		}
		return "", false, err
	}
	return value, true, nil
}

func (c NetworkClient) Del(ctx context.Context, key string) error {
	_, err := c.command(ctx, "DEL", key)
	return err
}

func (c NetworkClient) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, errors.New("lock ttl must be positive")
	}
	value, err := c.command(ctx, "SET", "lock:"+key, "1", "NX", "PX", strconv.FormatInt(ttl.Milliseconds(), 10))
	if err != nil {
		if errors.Is(err, errNil) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(value, "OK"), nil
}

func (c NetworkClient) Unlock(ctx context.Context, key string) error {
	return c.Del(ctx, "lock:"+key)
}

var errNil = errors.New("redis nil")

func (c NetworkClient) command(ctx context.Context, args ...string) (string, error) {
	if c.URL == "" {
		return "", errors.New("redis url is required")
	}
	parsed, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}
	addr := parsed.Host
	if !strings.Contains(addr, ":") {
		addr += ":6379"
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	}
	if parsed.User != nil {
		password, _ := parsed.User.Password()
		if password != "" {
			if _, err := writeRESP(conn, "AUTH", password); err != nil {
				return "", err
			}
			if _, err := readRESP(bufio.NewReader(conn)); err != nil {
				return "", err
			}
		}
	}
	if _, err := writeRESP(conn, args...); err != nil {
		return "", err
	}
	return readRESP(bufio.NewReader(conn))
}

func writeRESP(conn net.Conn, args ...string) (int, error) {
	var b strings.Builder
	b.WriteString("*" + strconv.Itoa(len(args)) + "\r\n")
	for _, arg := range args {
		b.WriteString("$" + strconv.Itoa(len(arg)) + "\r\n")
		b.WriteString(arg + "\r\n")
	}
	return conn.Write([]byte(b.String()))
}

func readRESP(r *bufio.Reader) (string, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	switch prefix {
	case '+':
		return line, nil
	case '-':
		return "", errors.New(line)
	case ':':
		return line, nil
	case '$':
		n, err := strconv.Atoi(line)
		if err != nil {
			return "", err
		}
		if n < 0 {
			return "", errNil
		}
		buf := make([]byte, n+2)
		if _, err := r.Read(buf); err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	default:
		return "", fmt.Errorf("unsupported redis response prefix %q", prefix)
	}
}
