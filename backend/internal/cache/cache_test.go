package cache

import (
	"testing"
	"time"
)

func TestCacheExpiresItems(t *testing.T) {
	c := New()
	c.Set("key", "value", time.Nanosecond)
	time.Sleep(time.Millisecond)
	if _, ok := c.Get("key"); ok {
		t.Fatal("expected cached item to expire")
	}
}

func TestCacheReturnsStoredItem(t *testing.T) {
	c := New()
	c.Set("key", "value", time.Minute)
	value, ok := c.Get("key")
	if !ok || value.(string) != "value" {
		t.Fatalf("cached value = %#v/%v, want value/true", value, ok)
	}
}
