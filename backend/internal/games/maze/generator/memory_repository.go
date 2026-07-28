package generator

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	versions    map[string]GeneratorVersion
	profiles    map[string]DifficultyProfile
	puzzles     map[string]PuzzleMetadata
	analyses    map[string]DifficultyAnalysis
	assignments map[string]Assignment
	seedClaims  map[string]string
	hashClaims  map[string]string
	requests    map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		versions:    map[string]GeneratorVersion{},
		profiles:    map[string]DifficultyProfile{},
		puzzles:     map[string]PuzzleMetadata{},
		analyses:    map[string]DifficultyAnalysis{},
		assignments: map[string]Assignment{},
		seedClaims:  map[string]string{},
		hashClaims:  map[string]string{},
		requests:    map[string]string{},
	}
}

func (r *MemoryRepository) RegisterVersion(ctx context.Context, version GeneratorVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := version.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := version.Key.ID()
	if current, exists := r.versions[key]; exists {
		if current != version {
			return ErrConflict
		}
		return nil
	}
	r.versions[key] = version
	return nil
}

func (r *MemoryRepository) GetVersion(ctx context.Context, key VersionKey) (GeneratorVersion, error) {
	if err := ctx.Err(); err != nil {
		return GeneratorVersion{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	version, ok := r.versions[key.ID()]
	if !ok {
		return GeneratorVersion{}, ErrNotFound
	}
	return version, nil
}

func (r *MemoryRepository) SaveDifficultyProfile(ctx context.Context, profile DifficultyProfile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := profile.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, exists := r.profiles[profile.ID]; exists {
		if current != profile {
			return ErrConflict
		}
		return nil
	}
	for _, current := range r.profiles {
		if current.GameID == profile.GameID && current.SchemaVersion == profile.SchemaVersion && current.ProfileHash == profile.ProfileHash {
			return ErrConflict
		}
	}
	r.profiles[profile.ID] = profile
	return nil
}

func (r *MemoryRepository) GetDifficultyProfile(ctx context.Context, id string) (DifficultyProfile, error) {
	if err := ctx.Err(); err != nil {
		return DifficultyProfile{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, ok := r.profiles[id]
	if !ok {
		return DifficultyProfile{}, ErrNotFound
	}
	return profile, nil
}

func (r *MemoryRepository) CreatePuzzle(ctx context.Context, puzzle PuzzleMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePreparedPuzzle(puzzle); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.puzzles[puzzle.ID]; exists {
		return ErrConflict
	}
	if _, exists := r.requests[puzzle.RequestHash]; exists {
		return ErrConflict
	}
	if _, exists := r.seedClaims[puzzle.SeedHash]; exists {
		return ErrDuplicatePuzzle
	}
	r.puzzles[puzzle.ID] = clonePuzzle(puzzle)
	r.requests[puzzle.RequestHash] = puzzle.ID
	return nil
}

func (r *MemoryRepository) GetPuzzle(ctx context.Context, id string) (PuzzleMetadata, error) {
	if err := ctx.Err(); err != nil {
		return PuzzleMetadata{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	puzzle, ok := r.puzzles[id]
	if !ok {
		return PuzzleMetadata{}, ErrNotFound
	}
	return clonePuzzle(puzzle), nil
}

func (r *MemoryRepository) GetPuzzleByRequestHash(ctx context.Context, requestHash string) (PuzzleMetadata, error) {
	if err := ctx.Err(); err != nil {
		return PuzzleMetadata{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.requests[requestHash]
	if !ok {
		return PuzzleMetadata{}, ErrNotFound
	}
	return clonePuzzle(r.puzzles[id]), nil
}

func (r *MemoryRepository) FinalizeAndAssign(ctx context.Context, final Finalization) (Assignment, error) {
	if err := ctx.Err(); err != nil {
		return Assignment{}, err
	}
	if err := validateFinalization(final); err != nil {
		return Assignment{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	puzzle, exists := r.puzzles[final.PuzzleID]
	if !exists {
		return Assignment{}, ErrNotFound
	}
	if puzzle.Status != PuzzlePreparing {
		return Assignment{}, ErrInvalidTransition
	}
	scopeKey := final.Assignment.ScopeType + "\x1f" + final.Assignment.ScopeID
	if _, exists := r.assignments[scopeKey]; exists {
		return Assignment{}, ErrConflict
	}
	if _, exists := r.seedClaims[puzzle.SeedHash]; exists {
		return Assignment{}, ErrDuplicatePuzzle
	}
	if _, exists := r.hashClaims[final.PuzzleHash]; exists {
		return Assignment{}, ErrDuplicatePuzzle
	}

	puzzle.Status = PuzzleAssigned
	puzzle.GenerationHash = final.GenerationHash
	puzzle.PuzzleHash = final.PuzzleHash
	puzzle.ValidationHash = final.ValidationHash
	puzzle.SolutionHash = final.SolutionHash
	puzzle.MinimumActions = final.MinimumActions
	puzzle.AnalysisID = final.Analysis.ID
	validatedAt := final.Analysis.CreatedAt
	assignedAt := final.Assignment.AssignedAt
	puzzle.ValidatedAt = &validatedAt
	puzzle.AssignedAt = &assignedAt
	r.puzzles[puzzle.ID] = clonePuzzle(puzzle)
	r.analyses[final.Analysis.ID] = final.Analysis
	r.seedClaims[puzzle.SeedHash] = puzzle.ID
	r.hashClaims[final.PuzzleHash] = puzzle.ID
	r.assignments[scopeKey] = final.Assignment
	return final.Assignment, nil
}

func (r *MemoryRepository) GetAssignment(ctx context.Context, scopeType, scopeID string) (Assignment, error) {
	if err := ctx.Err(); err != nil {
		return Assignment{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	assignment, ok := r.assignments[scopeType+"\x1f"+scopeID]
	if !ok {
		return Assignment{}, ErrNotFound
	}
	return assignment, nil
}

func clonePuzzle(puzzle PuzzleMetadata) PuzzleMetadata {
	puzzle.SeedCiphertext = append([]byte(nil), puzzle.SeedCiphertext...)
	puzzle.SeedNonce = append([]byte(nil), puzzle.SeedNonce...)
	return puzzle
}
