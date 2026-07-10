package puzzle

import (
	"context"
	"errors"

	"skill-arena/internal/game"
)

type Service struct {
	secret string
	repo   Repository
}

func NewService(secret string, repo Repository) *Service {
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{secret: secret, repo: repo}
}

func (s *Service) Generate(ctx context.Context, req Request) (Puzzle, error) {
	if req.Mode == "" {
		return Puzzle{}, errors.New("puzzle mode is required")
	}
	if req.Purpose == "" {
		req.Purpose = req.Mode
	}
	if req.PuzzleVersion.PuzzleEngineVersion == "" {
		req.PuzzleVersion = game.CurrentPuzzleVersion()
	}
	if req.DifficultyProfile.LineCount == 0 {
		req.DifficultyProfile = game.ProfileFromRating(10, req.Purpose)
	}
	req.DifficultyProfile = normalizeProfile(req)

	derivation, err := DeriveSeed(ctx, s.secret, req)
	if err != nil {
		return Puzzle{}, err
	}
	lines, solution := game.GenerateSolvedLinePuzzleFromProfile(derivation.Seed, req.DifficultyProfile)
	if err := Validate(ctx, lines, solution); err != nil {
		return Puzzle{}, err
	}

	puzzle := Puzzle{
		Lines:             cloneLines(lines),
		Solution:          append([]string(nil), solution...),
		DifficultyProfile: req.DifficultyProfile,
		Metadata:          MetadataFromProfile(req, derivation.GenerationHash, derivation.Nonce, derivation.Seed, solution),
	}
	if err := s.repo.Save(ctx, puzzle); err != nil {
		return Puzzle{}, err
	}
	return puzzle, nil
}

func (s *Service) Load(ctx context.Context, puzzleID string) (Puzzle, error) {
	return s.repo.Load(ctx, puzzleID)
}

func (s *Service) LoadBySeed(ctx context.Context, seed string) (Puzzle, error) {
	return s.repo.LoadBySeed(ctx, seed)
}
