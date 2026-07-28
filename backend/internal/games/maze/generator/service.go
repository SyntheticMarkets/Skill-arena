package generator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"time"
)

type Service struct {
	repository Repository
	vault      *SeedVault
	now        func() time.Time
	random     io.Reader
}

type PrepareRequest struct {
	Mode           string
	ScopeType      string
	ScopeID        string
	ParticipantID  string
	DifficultyID   string
	Version        VersionKey
	IdempotencyKey string
}

type PreparedPuzzle struct {
	Metadata PuzzleMetadata
	Seed     SeedMaterial `json:"-"`
}

func NewService(repository Repository, vault *SeedVault) (*Service, error) {
	if repository == nil || vault == nil {
		return nil, errors.New("puzzle repository and seed vault are required")
	}
	return &Service{repository: repository, vault: vault, now: time.Now, random: rand.Reader}, nil
}

func (s *Service) RegisterVersion(ctx context.Context, version GeneratorVersion) error {
	return s.repository.RegisterVersion(ctx, version)
}

func (s *Service) RegisterDifficultyProfile(ctx context.Context, profile DifficultyProfile) error {
	return s.repository.SaveDifficultyProfile(ctx, profile)
}

func (s *Service) Prepare(ctx context.Context, request PrepareRequest) (PreparedPuzzle, error) {
	if err := ctx.Err(); err != nil {
		return PreparedPuzzle{}, err
	}
	if strings.TrimSpace(request.Mode) == "" || strings.TrimSpace(request.ScopeType) == "" ||
		strings.TrimSpace(request.ScopeID) == "" || strings.TrimSpace(request.DifficultyID) == "" ||
		strings.TrimSpace(request.IdempotencyKey) == "" {
		return PreparedPuzzle{}, errors.New("puzzle preparation request is incomplete")
	}
	requestHash := HashFields("skill-arena:puzzle-prepare-request:v1",
		request.Mode, request.ScopeType, request.ScopeID, request.ParticipantID,
		request.DifficultyID, request.Version.ID(), request.IdempotencyKey)
	if existing, err := s.repository.GetPuzzleByRequestHash(ctx, requestHash); err == nil {
		seed, revealErr := s.revealMetadata(existing)
		if revealErr != nil {
			return PreparedPuzzle{}, revealErr
		}
		return PreparedPuzzle{Metadata: publicPuzzleMetadata(existing), Seed: seed}, nil
	} else if !errors.Is(err, ErrNotFound) {
		return PreparedPuzzle{}, err
	}
	version, err := s.repository.GetVersion(ctx, request.Version)
	if err != nil {
		return PreparedPuzzle{}, err
	}
	if version.Status != VersionActive || !version.NewMatchAllowed {
		return PreparedPuzzle{}, ErrVersionUnavailable
	}
	profile, err := s.repository.GetDifficultyProfile(ctx, request.DifficultyID)
	if err != nil {
		return PreparedPuzzle{}, err
	}
	if profile.GameID != request.Version.GameID || profile.SchemaVersion != request.Version.DifficultySchemaVersion {
		return PreparedPuzzle{}, errors.New("difficulty profile is incompatible with the generator version")
	}
	id, err := randomID(s.random)
	if err != nil {
		return PreparedPuzzle{}, err
	}
	createdAt := s.now().UTC()
	aad := seedAAD(id, request.Version.GameID, request.Mode, profile.ID, request.Version.ID())
	material, sealed, err := s.vault.Create(SeedScope{
		Mode: request.Mode, ScopeType: request.ScopeType, ScopeID: request.ScopeID,
		ParticipantID: request.ParticipantID, DifficultyID: profile.ID,
		GeneratorVersion: request.Version.ID(),
	}, aad)
	if err != nil {
		return PreparedPuzzle{}, err
	}
	metadata := PuzzleMetadata{
		ID: id, GameID: request.Version.GameID, Mode: request.Mode, Status: PuzzlePreparing,
		Version: request.Version, DifficultyID: profile.ID, DifficultyHash: profile.ProfileHash,
		RequestHash:   requestHash,
		SeedReference: sealed.Reference, SeedKeyID: sealed.KeyID, SeedHash: sealed.Hash,
		SeedCiphertext: sealed.Ciphertext, SeedNonce: sealed.Nonce, CreatedAt: createdAt,
	}
	if err := s.repository.CreatePuzzle(ctx, metadata); err != nil {
		if errors.Is(err, ErrConflict) {
			existing, loadErr := s.repository.GetPuzzleByRequestHash(ctx, requestHash)
			if loadErr == nil {
				seed, revealErr := s.revealMetadata(existing)
				if revealErr != nil {
					return PreparedPuzzle{}, revealErr
				}
				return PreparedPuzzle{Metadata: publicPuzzleMetadata(existing), Seed: seed}, nil
			}
		}
		return PreparedPuzzle{}, err
	}
	return PreparedPuzzle{Metadata: publicPuzzleMetadata(metadata), Seed: material}, nil
}

func (s *Service) LoadMetadata(ctx context.Context, puzzleID string) (PuzzleMetadata, error) {
	metadata, err := s.repository.GetPuzzle(ctx, puzzleID)
	if err != nil {
		return PuzzleMetadata{}, err
	}
	return publicPuzzleMetadata(metadata), nil
}

func (s *Service) RevealSeed(ctx context.Context, puzzleID string) (SeedMaterial, error) {
	metadata, err := s.repository.GetPuzzle(ctx, puzzleID)
	if err != nil {
		return SeedMaterial{}, err
	}
	return s.revealMetadata(metadata)
}

func (s *Service) revealMetadata(metadata PuzzleMetadata) (SeedMaterial, error) {
	return s.vault.Open(SealedSeed{
		Reference: metadata.SeedReference, KeyID: metadata.SeedKeyID, Hash: metadata.SeedHash,
		Ciphertext: metadata.SeedCiphertext, Nonce: metadata.SeedNonce,
	}, seedAAD(metadata.ID, metadata.GameID, metadata.Mode, metadata.DifficultyID, metadata.Version.ID()))
}

func (s *Service) FinalizeAndAssign(ctx context.Context, final Finalization) (Assignment, error) {
	if err := ctx.Err(); err != nil {
		return Assignment{}, err
	}
	assignment, err := s.repository.FinalizeAndAssign(ctx, final)
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrInvalidTransition) {
		existing, loadErr := s.repository.GetAssignment(ctx, final.Assignment.ScopeType, final.Assignment.ScopeID)
		if loadErr == nil && existing.PuzzleID == final.PuzzleID {
			return existing, nil
		}
	}
	return assignment, err
}

func randomID(random io.Reader) (string, error) {
	var data [16]byte
	if _, err := io.ReadFull(random, data[:]); err != nil {
		return "", err
	}
	return "puz_" + hex.EncodeToString(data[:]), nil
}

func seedAAD(id, gameID, mode, difficultyID, versionID string) string {
	return HashFields("skill-arena:puzzle-seed-aad:v1", id, gameID, mode, difficultyID, versionID)
}

func publicPuzzleMetadata(metadata PuzzleMetadata) PuzzleMetadata {
	metadata.SeedCiphertext = nil
	metadata.SeedNonce = nil
	return metadata
}

func validatePreparedPuzzle(puzzle PuzzleMetadata) error {
	if strings.TrimSpace(puzzle.ID) == "" || strings.TrimSpace(puzzle.GameID) == "" ||
		strings.TrimSpace(puzzle.Mode) == "" || puzzle.Status != PuzzlePreparing ||
		strings.TrimSpace(puzzle.DifficultyID) == "" || !ValidHash(puzzle.DifficultyHash) ||
		!ValidHash(puzzle.RequestHash) ||
		strings.TrimSpace(puzzle.SeedReference) == "" || strings.TrimSpace(puzzle.SeedKeyID) == "" ||
		!ValidHash(puzzle.SeedHash) || len(puzzle.SeedCiphertext) == 0 || len(puzzle.SeedNonce) == 0 ||
		puzzle.CreatedAt.IsZero() {
		return errors.New("prepared puzzle metadata is incomplete")
	}
	return puzzle.Version.Validate()
}

func validateFinalization(final Finalization) error {
	if strings.TrimSpace(final.PuzzleID) == "" || final.MinimumActions <= 0 ||
		!ValidHash(final.GenerationHash) || !ValidHash(final.PuzzleHash) ||
		!ValidHash(final.ValidationHash) || !ValidHash(final.SolutionHash) {
		return errors.New("puzzle finalization is incomplete")
	}
	if strings.TrimSpace(final.Analysis.ID) == "" || final.Analysis.PuzzleID != final.PuzzleID ||
		final.Analysis.AnalyzerVersion <= 0 || !final.Analysis.Accepted ||
		!ValidHash(final.Analysis.AnalysisHash) || final.Analysis.CreatedAt.IsZero() {
		return errors.New("accepted difficulty analysis is required")
	}
	if strings.TrimSpace(final.Assignment.ID) == "" || final.Assignment.PuzzleID != final.PuzzleID ||
		strings.TrimSpace(final.Assignment.ScopeType) == "" || strings.TrimSpace(final.Assignment.ScopeID) == "" ||
		!oneOf(final.Assignment.ReusePolicy, ReuseOneUse, ReuseTutorial, ReuseDailyWindow) ||
		final.Assignment.AssignedAt.IsZero() {
		return errors.New("puzzle assignment is incomplete")
	}
	return nil
}
