package realtime

import (
	"context"
	"testing"

	"skill-arena/internal/db"
	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
)

func TestReplayIntegritySignsAndRejectsTampering(t *testing.T) {
	store, err := db.New(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })
	service := NewService(store)
	request := interfaces.ReplayIntegrityRequest{
		MatchID: "match-1", GameID: "maze",
		ReplayHash:    generator.HashFields("test:replay", "one"),
		EventRootHash: generator.HashFields("test:events", "one"),
		EventCount:    3,
	}
	proof, err := service.SignReplayIntegrity(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyReplayIntegrity(t.Context(), request, proof); err != nil {
		t.Fatal(err)
	}
	settings := store.RuntimeSettings()
	oldKey := settings.Security.ReplaySigningKey
	oldKeyID := settings.Security.ReplaySigningKeyID
	settings.Security.ReplayVerificationKeys = map[string]string{oldKeyID: oldKey}
	settings.Security.ReplaySigningKey = "rotated-replay-signing-key-material-for-tests"
	settings.Security.ReplaySigningKeyID = "rotated-test-key-v2"
	if err := service.VerifyReplayIntegrity(t.Context(), request, proof); err != nil {
		t.Fatalf("historical replay proof did not survive key rotation: %v", err)
	}
	rotatedProof, err := service.SignReplayIntegrity(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if rotatedProof.KeyID != "rotated-test-key-v2" {
		t.Fatalf("new replay used key %q", rotatedProof.KeyID)
	}
	tampered := request
	tampered.EventCount++
	if err := service.VerifyReplayIntegrity(t.Context(), tampered, proof); err == nil {
		t.Fatal("event-count tampering passed replay signature verification")
	}
	tampered = request
	tampered.ReplayHash = generator.HashFields("test:replay", "two")
	if err := service.VerifyReplayIntegrity(t.Context(), tampered, proof); err == nil {
		t.Fatal("replay-hash tampering passed replay signature verification")
	}
	proof.Signature = proof.Signature[:len(proof.Signature)-2] + "00"
	if err := service.VerifyReplayIntegrity(t.Context(), request, proof); err == nil {
		t.Fatal("signature tampering passed verification")
	}
}
