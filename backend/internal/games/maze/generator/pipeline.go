package generator

import (
	"context"
	"errors"
	"strings"
)

// Processor is the CPU-work boundary implemented by later generator, solver,
// validator, and difficulty-analysis phases. Repository transactions never wrap
// this call.
type Processor interface {
	Process(context.Context, ProcessingInput) (ProcessingResult, error)
}

type ProcessingInput struct {
	Metadata PuzzleMetadata
	Profile  DifficultyProfile
	Seed     SeedMaterial `json:"-"`
}

type ProcessingResult struct {
	GenerationHash string
	PuzzleHash     string
	ValidationHash string
	SolutionHash   string
	MinimumActions int
	Analysis       DifficultyAnalysis
}

type WorkRequest struct {
	Prepare        PrepareRequest
	AssignmentMode string
	AssignmentType string
	AssignmentID   string
	ReusePolicy    string
}

// Execute prepares durable generation metadata, runs CPU work outside any
// repository transaction, then atomically claims and assigns the result.
func (s *Service) Execute(ctx context.Context, request WorkRequest, processor Processor) (Assignment, error) {
	if processor == nil {
		return Assignment{}, errors.New("puzzle processor is required")
	}
	if strings.TrimSpace(request.AssignmentMode) == "" ||
		strings.TrimSpace(request.AssignmentType) == "" ||
		strings.TrimSpace(request.AssignmentID) == "" {
		return Assignment{}, errors.New("puzzle work assignment is incomplete")
	}
	prepared, err := s.Prepare(ctx, request.Prepare)
	if err != nil {
		return Assignment{}, err
	}
	if existing, lookupErr := s.repository.GetAssignment(ctx, request.AssignmentType, request.AssignmentID); lookupErr == nil {
		if existing.PuzzleID != prepared.Metadata.ID {
			return Assignment{}, ErrConflict
		}
		return existing, nil
	} else if !errors.Is(lookupErr, ErrNotFound) {
		return Assignment{}, lookupErr
	}

	profile, err := s.repository.GetDifficultyProfile(ctx, prepared.Metadata.DifficultyID)
	if err != nil {
		return Assignment{}, err
	}
	result, err := processor.Process(ctx, ProcessingInput{
		Metadata: prepared.Metadata, Profile: profile, Seed: prepared.Seed,
	})
	if err != nil {
		return Assignment{}, err
	}
	if err := ctx.Err(); err != nil {
		return Assignment{}, err
	}
	now := s.now().UTC()
	analysisID, err := randomID(s.random)
	if err != nil {
		return Assignment{}, err
	}
	assignmentID, err := randomID(s.random)
	if err != nil {
		return Assignment{}, err
	}
	result.Analysis.ID = "analysis_" + analysisID
	result.Analysis.PuzzleID = prepared.Metadata.ID
	if result.Analysis.CreatedAt.IsZero() {
		result.Analysis.CreatedAt = now
	}
	return s.FinalizeAndAssign(ctx, Finalization{
		PuzzleID: prepared.Metadata.ID, GenerationHash: result.GenerationHash,
		PuzzleHash: result.PuzzleHash, ValidationHash: result.ValidationHash,
		SolutionHash: result.SolutionHash, MinimumActions: result.MinimumActions,
		Analysis: result.Analysis,
		Assignment: Assignment{
			ID: "assignment_" + assignmentID, PuzzleID: prepared.Metadata.ID,
			Mode: request.AssignmentMode, ScopeType: request.AssignmentType,
			ScopeID: request.AssignmentID, ReusePolicy: request.ReusePolicy,
			AssignedAt: now,
		},
	})
}
