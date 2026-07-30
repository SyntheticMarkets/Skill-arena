package redis

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryRateLimitIsAtomic(t *testing.T) {
	client := NewMemoryClient()
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := client.Allow(context.Background(), "login:test", 12, time.Minute)
			if err != nil {
				t.Errorf("allow: %v", err)
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := allowed.Load(); got != 12 {
		t.Fatalf("allowed=%d, want 12", got)
	}
}

func TestMemoryLockRequiresOwnerToken(t *testing.T) {
	client := NewMemoryClient()
	token, acquired, err := client.Lock(context.Background(), "matchmaking", time.Minute)
	if err != nil || !acquired || token == "" {
		t.Fatalf("lock token=%q acquired=%v err=%v", token, acquired, err)
	}
	if err := client.Unlock(context.Background(), "matchmaking", "wrong-token"); err == nil {
		t.Fatal("non-owner unlocked distributed lock")
	}
	if _, acquired, err := client.Lock(context.Background(), "matchmaking", time.Minute); err != nil || acquired {
		t.Fatalf("lock changed owner after rejected unlock: acquired=%v err=%v", acquired, err)
	}
	if err := client.Unlock(context.Background(), "matchmaking", token); err != nil {
		t.Fatal(err)
	}
	if _, acquired, err := client.Lock(context.Background(), "matchmaking", time.Minute); err != nil || !acquired {
		t.Fatalf("released lock was not acquirable: acquired=%v err=%v", acquired, err)
	}
}

func TestNetworkClientIntegration(t *testing.T) {
	redisURL := os.Getenv("SKILL_ARENA_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("SKILL_ARENA_TEST_REDIS_URL is not configured")
	}
	ctx := t.Context()
	client := NetworkClient{URL: redisURL}
	if err := client.Health(ctx); err != nil {
		t.Fatal(err)
	}

	key := "phase9:integration:value"
	t.Cleanup(func() { _ = client.Del(context.Background(), key) })
	if err := client.Set(ctx, key, "ready", 150*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if value, ok, err := client.Get(ctx, key); err != nil || !ok || value != "ready" {
		t.Fatalf("value=%q ok=%v err=%v", value, ok, err)
	}
	time.Sleep(200 * time.Millisecond)
	if value, ok, err := client.Get(ctx, key); err != nil || ok {
		t.Fatalf("expired value=%q ok=%v err=%v", value, ok, err)
	}

	lockKey := "phase9:integration:lock"
	token, acquired, err := client.Lock(ctx, lockKey, time.Second)
	if err != nil || !acquired {
		t.Fatalf("token=%q acquired=%v err=%v", token, acquired, err)
	}
	if _, acquired, err := client.Lock(ctx, lockKey, time.Second); err != nil || acquired {
		t.Fatalf("contended lock acquired=%v err=%v", acquired, err)
	}
	if err := client.Unlock(ctx, lockKey, "wrong-owner"); err == nil {
		t.Fatal("non-owner unlocked network lock")
	}
	if err := client.Unlock(ctx, lockKey, token); err != nil {
		t.Fatal(err)
	}

	rateKey := "phase9:integration:rate"
	t.Cleanup(func() { _ = client.Del(context.Background(), rateKey) })
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, allowErr := client.Allow(ctx, rateKey, 17, time.Minute)
			if allowErr != nil {
				t.Errorf("allow: %v", allowErr)
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := allowed.Load(); got != 17 {
		t.Fatalf("allowed=%d, want 17", got)
	}
}
