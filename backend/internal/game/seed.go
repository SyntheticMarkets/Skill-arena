package game

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"skill-arena/internal/models"
)

type SeedDerivationInput struct {
	Purpose           string                   `json:"purpose"`
	MatchID           string                   `json:"matchId"`
	PlayerID          string                   `json:"playerId"`
	Nonce             string                   `json:"nonce"`
	DifficultyProfile models.DifficultyProfile `json:"difficultyProfile"`
	PuzzleVersion     models.PuzzleVersion     `json:"puzzleVersion"`
}

type SeedDerivation struct {
	Seed           string
	GenerationHash string
	Nonce          string
}

func NewGenerationNonce() (string, error) {
	var bytes [24]byte
	if _, err := io.ReadFull(rand.Reader, bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func DerivePuzzleSeed(secret string, input SeedDerivationInput) (SeedDerivation, error) {
	if secret == "" {
		return SeedDerivation{}, errors.New("puzzle generation secret is required")
	}
	if input.Nonce == "" {
		nonce, err := NewGenerationNonce()
		if err != nil {
			return SeedDerivation{}, err
		}
		input.Nonce = nonce
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return SeedDerivation{}, err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sum := mac.Sum(nil)
	hash := hex.EncodeToString(sum)
	return SeedDerivation{
		Seed:           "hmac-sha256:" + hash,
		GenerationHash: hash,
		Nonce:          input.Nonce,
	}, nil
}
