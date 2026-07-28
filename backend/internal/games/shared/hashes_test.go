package shared

import "testing"

func TestHashFieldsIsDeterministicAndBoundarySafe(t *testing.T) {
	first := HashFields("domain", "ab", "c")
	if first != HashFields("domain", "ab", "c") {
		t.Fatal("identical fields produced different hashes")
	}
	if first == HashFields("domain", "a", "bc") {
		t.Fatal("length boundaries were ambiguous")
	}
	if first == HashFields("other-domain", "ab", "c") {
		t.Fatal("domain separation did not affect hash")
	}
}
