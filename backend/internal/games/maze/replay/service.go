package replay

import (
	"context"
	"errors"
	"strings"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/generator"
	"skill-arena/internal/games/maze/solver"
)

type Service struct {
	puzzles   *generator.Service
	processor *generator.ProductionProcessor
	solver    *solver.Solver
	integrity interfaces.ReplayIntegrityService
}

func NewService(
	puzzles *generator.Service,
	processor *generator.ProductionProcessor,
	solverInstance *solver.Solver,
	integrity interfaces.ReplayIntegrityService,
) (*Service, error) {
	if puzzles == nil || processor == nil || solverInstance == nil || integrity == nil {
		return nil, errors.New("replay service dependencies are required")
	}
	return &Service{
		puzzles: puzzles, processor: processor,
		solver: solverInstance, integrity: integrity,
	}, nil
}

func (s *Service) BuildGenesis(ctx context.Context, request GenesisRequest) (Genesis, error) {
	if err := validateGenesisRequest(request); err != nil {
		return Genesis{}, err
	}
	input, qualified, err := s.reconstruct(ctx, request.PuzzleID)
	if err != nil {
		return Genesis{}, err
	}
	metadata := input.Metadata
	if qualified.Result.GenerationHash != metadata.GenerationHash ||
		qualified.Result.PuzzleHash != metadata.PuzzleHash ||
		qualified.Result.ValidationHash != metadata.ValidationHash ||
		qualified.Result.SolutionHash != metadata.SolutionHash ||
		qualified.Result.Analysis.AnalysisHash != input.Analysis.AnalysisHash {
		return Genesis{}, errors.New("persisted puzzle integrity does not match reconstruction")
	}
	genesis := Genesis{
		ReplayID: request.ReplayID, MatchID: request.MatchID,
		PuzzleID: metadata.ID, GameID: metadata.GameID, Versions: request.Versions,
		SeedReference: metadata.SeedReference, SeedHash: metadata.SeedHash,
		DifficultyID: metadata.DifficultyID, DifficultyHash: metadata.DifficultyHash,
		AnalysisHash:   input.Analysis.AnalysisHash,
		GenerationHash: metadata.GenerationHash, PuzzleHash: metadata.PuzzleHash,
		ValidationHash: metadata.ValidationHash, SolutionHash: metadata.SolutionHash,
		MinimumActions: metadata.MinimumActions, CreatedAtUnixMS: request.CreatedAtUnixMS,
	}
	genesis.GenesisHash = genesisHash(genesis)
	if err := compareGenesis(genesis, input, qualified); err != nil {
		return Genesis{}, err
	}
	return genesis, nil
}

func (s *Service) Seal(ctx context.Context, request SealRequest) (Artifact, error) {
	if err := validateSealRequest(request); err != nil {
		return Artifact{}, err
	}
	input, qualified, err := s.reconstruct(ctx, request.Genesis.PuzzleID)
	if err != nil {
		return Artifact{}, err
	}
	if err := compareGenesis(request.Genesis, input, qualified); err != nil {
		return Artifact{}, err
	}
	events, participants, eventRoot, err := s.reconstructEvents(
		ctx, request.Genesis, qualified.Candidate.Board,
		request.ParticipantIDs, request.Events,
	)
	if err != nil {
		return Artifact{}, err
	}
	artifact := Artifact{
		Genesis: request.Genesis, Events: events, Participants: participants,
		Outcome:         normalizeOutcome(request.Outcome),
		StartedAtUnixMS: request.StartedAtUnixMS, EndedAtUnixMS: request.EndedAtUnixMS,
		EventRootHash: eventRoot,
	}
	if err := validateOutcome(artifact.Outcome, artifact.Participants); err != nil {
		return Artifact{}, err
	}
	artifact.ReplayHash = replayHash(artifact)
	proof, err := s.integrity.SignReplayIntegrity(ctx, integrityRequest(artifact))
	if err != nil {
		return Artifact{}, err
	}
	artifact.Proof = proof
	return artifact, nil
}

func (s *Service) Verify(ctx context.Context, artifact Artifact) (VerificationReport, error) {
	if err := ctx.Err(); err != nil {
		return VerificationReport{}, err
	}
	report := VerificationReport{
		Status: StatusUnderReview, ReplayHash: artifact.ReplayHash,
		EventRootHash: artifact.EventRootHash,
	}
	if err := validateArtifactEnvelope(artifact); err != nil {
		report.Issues = []string{"artifact_schema_invalid"}
		return report, nil
	}
	input, qualified, err := s.reconstruct(ctx, artifact.Genesis.PuzzleID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return VerificationReport{}, err
		}
		report.Issues = []string{"historical_reconstruction_failed"}
		return report, nil
	}
	if err := compareGenesis(artifact.Genesis, input, qualified); err != nil {
		report.Issues = []string{"genesis_integrity_mismatch"}
		return report, nil
	}
	drafts := make([]EventDraft, len(artifact.Events))
	for index, event := range artifact.Events {
		drafts[index] = event.EventDraft
	}
	participantIDs := make([]string, len(artifact.Participants))
	for index, participant := range artifact.Participants {
		participantIDs[index] = participant.ParticipantID
	}
	events, participants, eventRoot, err := s.reconstructEvents(
		ctx, artifact.Genesis, qualified.Candidate.Board, participantIDs, drafts,
	)
	if err != nil || !eventsEqual(events, artifact.Events) ||
		!participantsEqual(participants, artifact.Participants) ||
		eventRoot != artifact.EventRootHash {
		report.Issues = []string{"event_or_state_integrity_mismatch"}
		return report, nil
	}
	if err := validateOutcome(artifact.Outcome, participants); err != nil {
		report.Issues = []string{"outcome_invalid"}
		return report, nil
	}
	expectedReplayHash := replayHash(artifact)
	if expectedReplayHash != artifact.ReplayHash {
		report.Issues = []string{"replay_hash_mismatch"}
		return report, nil
	}
	if err := s.integrity.VerifyReplayIntegrity(ctx, integrityRequest(artifact), artifact.Proof); err != nil {
		report.Issues = []string{"platform_signature_invalid"}
		return report, nil
	}
	report.Status = StatusVerified
	report.Verified = true
	report.Participants = participants
	report.Issues = nil
	return report, nil
}

func (s *Service) reconstruct(
	ctx context.Context,
	puzzleID string,
) (generator.ReconstructionInput, generator.QualifiedCandidate, error) {
	input, err := s.puzzles.LoadReconstructionInput(ctx, puzzleID)
	if err != nil {
		return generator.ReconstructionInput{}, generator.QualifiedCandidate{}, err
	}
	qualified, err := s.processor.Generate(ctx, input.ProcessingInput)
	if err != nil {
		return generator.ReconstructionInput{}, generator.QualifiedCandidate{}, err
	}
	return input, qualified, nil
}

func integrityRequest(artifact Artifact) interfaces.ReplayIntegrityRequest {
	return interfaces.ReplayIntegrityRequest{
		MatchID: artifact.Genesis.MatchID, GameID: artifact.Genesis.GameID,
		ReplayHash: artifact.ReplayHash, EventRootHash: artifact.EventRootHash,
		EventCount: len(artifact.Events),
	}
}

func compareGenesis(
	genesis Genesis,
	input generator.ReconstructionInput,
	qualified generator.QualifiedCandidate,
) error {
	metadata := input.Metadata
	if genesis.GenesisHash != genesisHash(genesis) ||
		genesis.PuzzleID != metadata.ID || genesis.GameID != metadata.GameID ||
		genesis.Versions.Generator != metadata.Version ||
		genesis.SeedReference != metadata.SeedReference || genesis.SeedHash != metadata.SeedHash ||
		genesis.DifficultyID != metadata.DifficultyID ||
		genesis.DifficultyHash != metadata.DifficultyHash ||
		genesis.AnalysisHash != input.Analysis.AnalysisHash ||
		genesis.GenerationHash != metadata.GenerationHash ||
		genesis.PuzzleHash != metadata.PuzzleHash ||
		genesis.ValidationHash != metadata.ValidationHash ||
		genesis.SolutionHash != metadata.SolutionHash ||
		genesis.MinimumActions != metadata.MinimumActions ||
		qualified.Result.GenerationHash != metadata.GenerationHash ||
		qualified.Result.PuzzleHash != metadata.PuzzleHash ||
		qualified.Result.ValidationHash != metadata.ValidationHash ||
		qualified.Result.SolutionHash != metadata.SolutionHash ||
		qualified.Result.Analysis.AnalysisHash != input.Analysis.AnalysisHash {
		return errors.New("replay genesis does not match reconstructed puzzle")
	}
	return nil
}

func validateGenesisRequest(request GenesisRequest) error {
	if strings.TrimSpace(request.ReplayID) == "" || strings.TrimSpace(request.MatchID) == "" ||
		strings.TrimSpace(request.PuzzleID) == "" || request.CreatedAtUnixMS <= 0 {
		return errors.New("replay genesis request is incomplete")
	}
	return request.Versions.Validate()
}
