package db

import "testing"

func TestGameActionShardIsStableAndBounded(t *testing.T) {
	for count := 1; count <= 16; count++ {
		first := gameActionShard("match-stable", count)
		if first < 0 || first >= count {
			t.Fatalf("count=%d shard=%d", count, first)
		}
		if second := gameActionShard("match-stable", count); second != first {
			t.Fatalf("count=%d first=%d second=%d", count, first, second)
		}
	}
}
