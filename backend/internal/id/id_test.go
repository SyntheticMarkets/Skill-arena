package id

import (
	"strings"
	"testing"
)

func TestNewUsesPrefixAndIsUnique(t *testing.T) {
	first := New("ses")
	second := New("ses")
	if first == second {
		t.Fatal("expected generated IDs to be unique")
	}
	if !strings.HasPrefix(first, "ses_") || !strings.HasPrefix(second, "ses_") {
		t.Fatalf("ids = %q/%q, want ses_ prefix", first, second)
	}
}
