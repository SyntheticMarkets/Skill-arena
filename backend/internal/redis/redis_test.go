package redis

import (
	"context"
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
