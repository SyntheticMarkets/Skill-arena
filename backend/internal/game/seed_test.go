package game

import "testing"

func TestDerivePuzzleSeedIsDeterministicWithSameNonce(t *testing.T) {
	profile := ProfileFromComplexity(420, 42, "test")
	version := CurrentPuzzleVersion()
	input := SeedDerivationInput{
		Purpose:           "pvp_player_a",
		MatchID:           "match-1",
		PlayerID:          "player-1",
		Nonce:             "nonce-1",
		DifficultyProfile: profile,
		PuzzleVersion:     version,
	}

	first, err := DerivePuzzleSeed("server-secret", input)
	if err != nil {
		t.Fatalf("derive first seed: %v", err)
	}
	second, err := DerivePuzzleSeed("server-secret", input)
	if err != nil {
		t.Fatalf("derive second seed: %v", err)
	}

	if first.Seed != second.Seed || first.GenerationHash != second.GenerationHash {
		t.Fatalf("expected deterministic derivation, got %#v and %#v", first, second)
	}
	if first.Seed == "" || first.GenerationHash == "" || first.Nonce != "nonce-1" {
		t.Fatalf("unexpected derivation metadata: %#v", first)
	}
}

func TestDerivePuzzleSeedChangesByPlayerAndNonce(t *testing.T) {
	profile := ProfileFromComplexity(420, 42, "test")
	version := CurrentPuzzleVersion()
	base := SeedDerivationInput{
		Purpose:           "pvp_player",
		MatchID:           "match-1",
		PlayerID:          "player-1",
		Nonce:             "nonce-1",
		DifficultyProfile: profile,
		PuzzleVersion:     version,
	}
	first, err := DerivePuzzleSeed("server-secret", base)
	if err != nil {
		t.Fatalf("derive first seed: %v", err)
	}

	otherPlayer := base
	otherPlayer.PlayerID = "player-2"
	second, err := DerivePuzzleSeed("server-secret", otherPlayer)
	if err != nil {
		t.Fatalf("derive second seed: %v", err)
	}

	otherNonce := base
	otherNonce.Nonce = "nonce-2"
	third, err := DerivePuzzleSeed("server-secret", otherNonce)
	if err != nil {
		t.Fatalf("derive third seed: %v", err)
	}

	if first.Seed == second.Seed || first.GenerationHash == second.GenerationHash {
		t.Fatal("expected different players to produce different puzzle derivations")
	}
	if first.Seed == third.Seed || first.GenerationHash == third.GenerationHash {
		t.Fatal("expected different nonces to produce different puzzle derivations")
	}
}
