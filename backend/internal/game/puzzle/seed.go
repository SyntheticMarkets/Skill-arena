package puzzle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"skill-arena/internal/game"
)

func DeriveSeed(ctx context.Context, secret string, req Request) (game.SeedDerivation, error) {
	if err := ctx.Err(); err != nil {
		return game.SeedDerivation{}, err
	}
	return game.DerivePuzzleSeed(secret, game.SeedDerivationInput{
		Purpose:           req.Purpose,
		MatchID:           req.MatchID,
		PlayerID:          seedPlayer(req),
		Nonce:             req.Nonce,
		DifficultyProfile: req.DifficultyProfile,
		PuzzleVersion:     req.PuzzleVersion,
	})
}

func seedPlayer(req Request) string {
	if req.Shared {
		return "shared"
	}
	return req.PlayerID
}

func puzzleID(mode string, generationHash string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", mode, generationHash)))
	return "puzzle_" + hex.EncodeToString(sum[:])[:24]
}
