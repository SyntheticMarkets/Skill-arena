package puzzle

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("puzzle not found")

type Repository interface {
	Save(ctx context.Context, puzzle Puzzle) error
	Load(ctx context.Context, puzzleID string) (Puzzle, error)
	LoadBySeed(ctx context.Context, seed string) (Puzzle, error)
}

type MemoryRepository struct {
	mu     sync.RWMutex
	byID   map[string]Puzzle
	bySeed map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:   map[string]Puzzle{},
		bySeed: map[string]string{},
	}
}

func (r *MemoryRepository) Save(ctx context.Context, puzzle Puzzle) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	puzzle.Lines = cloneLines(puzzle.Lines)
	puzzle.Solution = append([]string(nil), puzzle.Solution...)
	r.byID[puzzle.Metadata.PuzzleID] = puzzle
	r.bySeed[puzzle.Metadata.Seed] = puzzle.Metadata.PuzzleID
	return nil
}

func (r *MemoryRepository) Load(ctx context.Context, puzzleID string) (Puzzle, error) {
	if err := ctx.Err(); err != nil {
		return Puzzle{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	puzzle, ok := r.byID[puzzleID]
	if !ok {
		return Puzzle{}, ErrNotFound
	}
	puzzle.Lines = cloneLines(puzzle.Lines)
	puzzle.Solution = append([]string(nil), puzzle.Solution...)
	return puzzle, nil
}

func (r *MemoryRepository) LoadBySeed(ctx context.Context, seed string) (Puzzle, error) {
	if err := ctx.Err(); err != nil {
		return Puzzle{}, err
	}
	r.mu.RLock()
	puzzleID, ok := r.bySeed[seed]
	r.mu.RUnlock()
	if !ok {
		return Puzzle{}, ErrNotFound
	}
	return r.Load(ctx, puzzleID)
}
