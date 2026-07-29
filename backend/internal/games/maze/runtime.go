package maze

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"skill-arena/internal/games/interfaces"
	"skill-arena/internal/games/maze/engine"
	"skill-arena/internal/games/maze/generator"
	mazereplay "skill-arena/internal/games/maze/replay"
	"skill-arena/internal/games/maze/solver"
	"skill-arena/internal/storage"
)

const sharedStateSchema = "maze-shared-v1"

type Runtime struct {
	puzzles   *generator.Service
	processor *generator.ProductionProcessor
	version   generator.GeneratorVersion
	profiles  map[string]generator.DifficultyProfile
	replays   *mazereplay.ObjectRepository
	once      sync.Once
	readyErr  error
}

type SharedState struct {
	PuzzleID       string          `json:"puzzleId"`
	PuzzleHash     string          `json:"puzzleHash"`
	ValidationHash string          `json:"validationHash"`
	DifficultyHash string          `json:"difficultyHash"`
	GenerationHash string          `json:"generationHash"`
	MinimumActions int             `json:"minimumActions"`
	Board          generator.Board `json:"board"`
	StartedAtMS    int64           `json:"startedAtMs"`
	DeadlineAtMS   int64           `json:"deadlineAtMs"`
}

func NewRuntime(
	puzzles *generator.Service,
	objects storage.ObjectStore,
) (*Runtime, error) {
	if puzzles == nil || objects == nil {
		return nil, errors.New("Maze runtime Puzzle Service and object storage are required")
	}
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	if err != nil {
		return nil, err
	}
	replayRepository, err := mazereplay.NewObjectRepository(objects, "replays/maze")
	if err != nil {
		return nil, err
	}
	processor, err := generator.NewProductionProcessor(
		generator.DefaultGenerationConfig(), solverInstance, nil,
	)
	if err != nil {
		return nil, err
	}
	released := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	version := productionGeneratorVersion(released)
	profiles := make(map[string]generator.DifficultyProfile)
	for _, mode := range []string{
		"practice", "ranked", "house", "daily", "tournament", "calibration",
	} {
		profile := productionDifficultyProfile(mode, released)
		profiles[mode] = profile
	}
	profiles["pvp"] = profiles["ranked"]
	return &Runtime{
		puzzles: puzzles, processor: processor, version: version, profiles: profiles,
		replays: replayRepository,
	}, nil
}

func (r *Runtime) ensureReady(ctx context.Context) error {
	r.once.Do(func() {
		if err := r.puzzles.RegisterVersion(ctx, r.version); err != nil {
			r.readyErr = err
			return
		}
		registered := map[string]bool{}
		for _, profile := range r.profiles {
			if registered[profile.ID] {
				continue
			}
			if err := r.puzzles.RegisterDifficultyProfile(ctx, profile); err != nil {
				r.readyErr = err
				return
			}
			registered[profile.ID] = true
		}
	})
	return r.readyErr
}

func (r *Runtime) GenerateState(
	ctx context.Context,
	match interfaces.MatchContext,
	request interfaces.GenerationRequest,
) (interfaces.GeneratedState, error) {
	if err := r.ensureReady(ctx); err != nil {
		return interfaces.GeneratedState{}, err
	}
	mode := normalizeMode(request.Mode)
	if mode == "tutorial" {
		return r.generateTutorial(match, request)
	}
	profile, exists := r.profiles[mode]
	if !exists {
		return interfaces.GeneratedState{}, fmt.Errorf("Maze mode %q has no approved difficulty profile", mode)
	}
	scopeType, scopeID, participantID, reuse := assignmentPolicy(match, mode)
	assignment, err := r.puzzles.Execute(ctx, generator.WorkRequest{
		Prepare: generator.PrepareRequest{
			Mode: mode, ScopeType: scopeType, ScopeID: scopeID,
			ParticipantID: participantID, DifficultyID: profile.ID,
			Version:        r.version.Key,
			IdempotencyKey: "maze-runtime:" + match.MatchID + ":" + mode,
		},
		AssignmentMode: mode, AssignmentType: scopeType,
		AssignmentID: scopeID, ReusePolicy: reuse,
	}, r.processor)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	input, err := r.puzzles.LoadReconstructionInput(ctx, assignment.PuzzleID)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	qualified, err := r.processor.Generate(ctx, input.ProcessingInput)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	startedAt := match.ServerTime.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	shared := SharedState{
		PuzzleID: assignment.PuzzleID, PuzzleHash: input.Metadata.PuzzleHash,
		ValidationHash: input.Metadata.ValidationHash,
		DifficultyHash: input.Metadata.DifficultyHash,
		GenerationHash: input.Metadata.GenerationHash,
		MinimumActions: input.Metadata.MinimumActions,
		Board:          qualified.Candidate.Board.Clone(), StartedAtMS: startedAt.UnixMilli(),
		DeadlineAtMS: startedAt.Add(2 * time.Minute).UnixMilli(),
	}
	metadata, err := json.Marshal(shared)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	return interfaces.GeneratedState{Reference: assignment.PuzzleID, Metadata: metadata}, nil
}

func (r *Runtime) generateTutorial(
	match interfaces.MatchContext,
	request interfaces.GenerationRequest,
) (interfaces.GeneratedState, error) {
	level := 1
	if len(request.DifficultyProfile) > 0 {
		var value struct {
			Level int `json:"level"`
		}
		if err := json.Unmarshal(request.DifficultyProfile, &value); err != nil {
			return interfaces.GeneratedState{}, errors.New("tutorial difficulty request is invalid")
		}
		level = value.Level
	}
	board, err := engine.TutorialBoard(level)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	boardBytes, err := generator.CanonicalBoard(board)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	startedAt := match.ServerTime.UTC()
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	puzzleHash := generator.HashBytes(
		"skill-arena:maze-tutorial-puzzle:v1", boardBytes, []byte(strconv.Itoa(level)),
	)
	shared := SharedState{
		PuzzleID: "tutorial-v1-" + strconv.Itoa(level), PuzzleHash: puzzleHash,
		ValidationHash: generator.HashFields("skill-arena:maze-tutorial-validation:v1", puzzleHash),
		DifficultyHash: generator.HashFields("skill-arena:maze-tutorial-difficulty:v1", strconv.Itoa(level)),
		GenerationHash: generator.HashFields("skill-arena:maze-tutorial-generation:v1", puzzleHash),
		MinimumActions: len(board.Arrows), Board: board,
		StartedAtMS:  startedAt.UnixMilli(),
		DeadlineAtMS: startedAt.Add(5 * time.Minute).UnixMilli(),
	}
	metadata, err := json.Marshal(shared)
	if err != nil {
		return interfaces.GeneratedState{}, err
	}
	return interfaces.GeneratedState{Reference: shared.PuzzleID, Metadata: metadata}, nil
}

func (r *Runtime) InitializeMatch(
	ctx context.Context,
	match interfaces.MatchContext,
	request interfaces.MatchRequest,
) (interfaces.GameState, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.GameState{}, err
	}
	var shared SharedState
	decoder := json.NewDecoder(strings.NewReader(string(request.Configuration)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&shared); err != nil {
		return interfaces.GameState{}, errors.New("Maze shared state is invalid")
	}
	if shared.PuzzleID == "" || !generator.ValidHash(shared.PuzzleHash) ||
		!generator.ValidHash(shared.ValidationHash) ||
		!generator.ValidHash(shared.DifficultyHash) ||
		!generator.ValidHash(shared.GenerationHash) ||
		shared.MinimumActions != len(shared.Board.Arrows) {
		return interfaces.GameState{}, errors.New("Maze shared state integrity is incomplete")
	}
	payload, err := json.Marshal(shared)
	if err != nil {
		return interfaces.GameState{}, err
	}
	return interfaces.GameState{
		SchemaVersion: sharedStateSchema, Payload: payload,
		Checksum: generator.HashFields(
			"skill-arena:maze-shared-state:v1", match.MatchID,
			shared.PuzzleHash, shared.ValidationHash, string(payload),
		),
	}, nil
}

func (r *Runtime) InitializeParticipant(
	ctx context.Context,
	participant interfaces.ParticipantContext,
	sharedState interfaces.GameState,
) (interfaces.GameState, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.GameState{}, err
	}
	if sharedState.SchemaVersion != sharedStateSchema {
		return interfaces.GameState{}, errors.New("Maze shared-state schema is unsupported")
	}
	var shared SharedState
	if err := json.Unmarshal(sharedState.Payload, &shared); err != nil {
		return interfaces.GameState{}, err
	}
	state, err := engine.NewState(engine.StartRequest{
		MatchID: participant.MatchID, ParticipantID: participant.UserID,
		PuzzleID: shared.PuzzleID, PuzzleHash: shared.PuzzleHash,
		DifficultyHash: shared.DifficultyHash, Board: shared.Board,
		MinimumActions: shared.MinimumActions, StartedAtMS: shared.StartedAtMS,
		DeadlineAtMS: shared.DeadlineAtMS,
	})
	if err != nil {
		return interfaces.GameState{}, err
	}
	return state.Generic()
}

func (r *Runtime) ValidateAction(
	ctx context.Context,
	actionContext interfaces.ActionContext,
	state interfaces.GameState,
	action interfaces.ActionEnvelope,
) (interfaces.ValidatedAction, error) {
	decoded, err := engine.DecodeState(state)
	if err != nil {
		return interfaces.ValidatedAction{}, err
	}
	validated, err := engine.ValidateAction(ctx, actionContext, decoded, action)
	if err != nil {
		return interfaces.ValidatedAction{}, err
	}
	payload, err := json.Marshal(engine.ArrowClick{ArrowID: validated.ArrowID})
	if err != nil {
		return interfaces.ValidatedAction{}, err
	}
	return interfaces.ValidatedAction{
		ActionID: validated.ActionID, Kind: engine.ActionArrowClick, Payload: payload,
	}, nil
}

func (r *Runtime) ApplyAction(
	ctx context.Context,
	actionContext interfaces.ActionContext,
	state interfaces.GameState,
	action interfaces.ValidatedAction,
) (interfaces.Transition, error) {
	decoded, err := engine.DecodeState(state)
	if err != nil {
		return interfaces.Transition{}, err
	}
	var click engine.ArrowClick
	if err := json.Unmarshal(action.Payload, &click); err != nil {
		return interfaces.Transition{}, errors.New("validated Maze action payload is invalid")
	}
	result, err := engine.ApplyAction(ctx, actionContext, decoded, engine.ValidatedAction{
		ActionID: action.ActionID, ArrowID: click.ArrowID,
	})
	if err != nil {
		return interfaces.Transition{}, err
	}
	return result.Transition, nil
}

func (r *Runtime) Snapshot(
	ctx context.Context,
	viewer interfaces.ViewerContext,
	state interfaces.GameState,
) (interfaces.RendererSnapshot, error) {
	decoded, err := engine.DecodeState(state)
	if err != nil {
		return interfaces.RendererSnapshot{}, err
	}
	return engine.Snapshot(ctx, viewer, decoded)
}

func (r *Runtime) Completion(
	ctx context.Context,
	_ interfaces.MatchContext,
	state interfaces.GameState,
) (interfaces.CompletionResult, error) {
	decoded, err := engine.DecodeState(state)
	if err != nil {
		return interfaces.CompletionResult{}, err
	}
	return engine.Completion(ctx, decoded)
}

func (r *Runtime) Expire(
	ctx context.Context,
	_ interfaces.MatchContext,
	state interfaces.GameState,
	serverTime time.Time,
) (interfaces.Transition, error) {
	decoded, err := engine.DecodeState(state)
	if err != nil {
		return interfaces.Transition{}, err
	}
	if serverTime.UTC().UnixMilli() < decoded.DeadlineAtMS {
		return interfaces.Transition{}, nil
	}
	result, err := engine.Expire(ctx, decoded, serverTime.UTC().UnixMilli())
	if err != nil {
		return interfaces.Transition{}, err
	}
	return result.Transition, nil
}

func (r *Runtime) DetermineWinner(
	ctx context.Context,
	_ interfaces.MatchContext,
	states []interfaces.GameState,
) (interfaces.MatchOutcome, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.MatchOutcome{}, err
	}
	type completed struct {
		id string
		at int64
	}
	completedStates := make([]completed, 0, len(states))
	allTerminal := len(states) > 0
	participants := make([]string, 0, len(states))
	for _, generic := range states {
		state, err := engine.DecodeState(generic)
		if err != nil {
			return interfaces.MatchOutcome{}, err
		}
		participants = append(participants, state.ParticipantID)
		if state.Status == engine.StatusCompleted {
			completedStates = append(completedStates, completed{id: state.ParticipantID, at: state.CompletedAtMS})
		} else if state.Status == engine.StatusActive {
			allTerminal = false
		}
	}
	if len(completedStates) == 0 && !allTerminal {
		return interfaces.MatchOutcome{Status: "incomplete"}, nil
	}
	if len(completedStates) == 0 {
		return interfaces.MatchOutcome{Status: "draw", Reason: "all_participants_timed_out"}, nil
	}
	winner := completedStates[0]
	for _, candidate := range completedStates[1:] {
		if candidate.at < winner.at || (candidate.at == winner.at && candidate.id < winner.id) {
			winner = candidate
		}
	}
	losers := make([]string, 0, len(participants)-1)
	for _, id := range participants {
		if id != winner.id {
			losers = append(losers, id)
		}
	}
	return interfaces.MatchOutcome{
		Status: "complete", WinnerIDs: []string{winner.id},
		LoserIDs: losers, Reason: "first_verified_completion",
	}, nil
}

func (r *Runtime) SerializeReplay(
	context.Context,
	interfaces.ReplaySource,
) (interfaces.ReplayMetadata, error) {
	return interfaces.ReplayMetadata{}, errors.New("Maze replay sealing is owned by the Phase 5 replay service")
}

func (r *Runtime) RestoreReplay(
	context.Context,
	interfaces.ReplayMetadata,
	[]interfaces.ReplayEvent,
) (interfaces.GameState, error) {
	return interfaces.GameState{}, errors.New("Maze replay restoration is owned by the Phase 5 replay service")
}

func (r *Runtime) FinalizeAuthoritativeReplay(
	ctx context.Context,
	match interfaces.MatchContext,
	source interfaces.ReplaySource,
	integrity interfaces.ReplayIntegrityService,
) (interfaces.FinalizedReplay, error) {
	if source.ReplayID == "" || source.MatchID != match.MatchID ||
		source.StartedAtUnixMS <= 0 || source.EndedAtUnixMS < source.StartedAtUnixMS {
		return interfaces.FinalizedReplay{}, errors.New("Maze replay source is incomplete")
	}
	solverInstance, err := solver.New(solver.Config{Version: 1, MaxArrows: 2048})
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	replayService, err := mazereplay.NewService(
		r.puzzles, r.processor, solverInstance, integrity,
	)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	participantIDs := make([]string, 0, len(source.States))
	puzzleID := ""
	for _, generic := range source.States {
		state, decodeErr := engine.DecodeState(generic)
		if decodeErr != nil {
			return interfaces.FinalizedReplay{}, decodeErr
		}
		if puzzleID == "" {
			puzzleID = state.PuzzleID
		} else if puzzleID != state.PuzzleID {
			return interfaces.FinalizedReplay{}, errors.New("replay participant puzzles diverge")
		}
		participantIDs = append(participantIDs, state.ParticipantID)
	}
	input, err := r.puzzles.LoadReconstructionInput(ctx, puzzleID)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	genesis, err := replayService.BuildGenesis(ctx, mazereplay.GenesisRequest{
		ReplayID: source.ReplayID, MatchID: source.MatchID, PuzzleID: puzzleID,
		Versions: mazereplay.Versions{
			GameVersion:        match.Versions.Game,
			ProtocolVersion:    numericVersion(match.Versions.Protocol),
			ReplayVersion:      mazereplay.ReplayVersionEngine,
			RendererVersion:    engine.RendererVersion,
			StateSchemaVersion: engine.StateSchemaVersion,
			Generator:          input.Metadata.Version,
		},
		CreatedAtUnixMS: source.StartedAtUnixMS,
	})
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	drafts := make([]mazereplay.EventDraft, 0)
	for _, event := range source.Events {
		if event.Kind != "game.action.processed" {
			continue
		}
		var payload struct {
			Payload struct {
				ArrowID string `json:"arrowId"`
			} `json:"payload"`
			Accepted     bool   `json:"accepted"`
			Code         string `json:"code"`
			OccurredAtMS int64  `json:"occurredAtMs"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return interfaces.FinalizedReplay{}, err
		}
		kind := mazereplay.EventArrowBlocked
		if payload.Accepted {
			kind = mazereplay.EventArrowAccepted
		}
		drafts = append(drafts, mazereplay.EventDraft{
			Sequence: uint64(len(drafts) + 1), ParticipantID: event.ParticipantID,
			OffsetMS: payload.OccurredAtMS - source.StartedAtUnixMS,
			Kind:     kind, ArrowID: payload.Payload.ArrowID, Code: payload.Code,
		})
	}
	outcome := mazereplay.Outcome{
		Status: source.Outcome.Status, WinnerIDs: source.Outcome.WinnerIDs,
		LoserIDs: source.Outcome.LoserIDs, Reason: source.Outcome.Reason,
	}
	if outcome.Status == "complete" {
		outcome.Status = "completed"
	}
	artifact, err := replayService.Seal(ctx, mazereplay.SealRequest{
		Genesis: genesis, ParticipantIDs: participantIDs, Events: drafts,
		Outcome: outcome, StartedAtUnixMS: source.StartedAtUnixMS,
		EndedAtUnixMS: source.EndedAtUnixMS,
	})
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	report, err := replayService.Verify(ctx, artifact)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	if !report.Verified || report.Status != mazereplay.StatusVerified {
		return interfaces.FinalizedReplay{}, errors.New("Maze replay verification entered review")
	}
	if err := r.replays.Save(ctx, artifact); err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	storageKey, err := r.replays.StorageKey(source.ReplayID)
	if err != nil {
		return interfaces.FinalizedReplay{}, err
	}
	return interfaces.FinalizedReplay{
		ReplayID: source.ReplayID, ReplayHash: artifact.ReplayHash,
		EventRootHash: artifact.EventRootHash, EventCount: len(artifact.Events),
		Proof: artifact.Proof, StorageKey: storageKey, Status: report.Status,
	}, nil
}

func (r *Runtime) Cleanup(
	ctx context.Context,
	_ interfaces.MatchContext,
) (interfaces.CleanupInstructions, error) {
	if err := ctx.Err(); err != nil {
		return interfaces.CleanupInstructions{}, err
	}
	return interfaces.CleanupInstructions{RetainReplay: true}, nil
}

func normalizeMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "house_challenge":
		return "house"
	case "daily_challenge":
		return "daily"
	default:
		return mode
	}
}

func assignmentPolicy(match interfaces.MatchContext, mode string) (string, string, string, string) {
	switch mode {
	case "practice":
		participant := ""
		if len(match.ParticipantIDs) == 1 {
			participant = match.ParticipantIDs[0]
		}
		return "practice_session", match.MatchID, participant, generator.ReuseOneUse
	case "house":
		return "house_attempt", match.MatchID, firstParticipant(match), generator.ReuseOneUse
	case "daily":
		window := match.ServerTime.UTC().Format("2006-01-02")
		return "daily_challenge", "maze-daily-" + window, "", generator.ReuseDailyWindow
	case "tournament":
		return "tournament_match", match.MatchID, "", generator.ReuseOneUse
	default:
		return "match", match.MatchID, "", generator.ReuseOneUse
	}
}

func firstParticipant(match interfaces.MatchContext) string {
	if len(match.ParticipantIDs) == 0 {
		return ""
	}
	return match.ParticipantIDs[0]
}

func productionGeneratorVersion(released time.Time) generator.GeneratorVersion {
	key := generator.VersionKey{
		GameID: generator.GameID, GeneratorVersion: 1,
		SeedFormatVersion:       generator.SeedFormatVersion,
		RandomStreamVersion:     generator.RandomStreamVersion,
		PatternCatalogueVersion: 1, PatternSelectionVersion: 1,
		GeometrySchemaVersion: 1, CandidateScoringVersion: 1,
		ConstraintPolicyVersion: 1, SolverVersion: 1, ValidatorVersion: 1,
		AnalyzerVersion: 1, DifficultySchemaVersion: 1,
		CanonicalEncodingVersion: 1,
	}
	return generator.GeneratorVersion{
		Key: key, Status: generator.VersionActive, NewMatchAllowed: true,
		ArtifactDigest:         generator.HashFields("skill-arena:maze-generator-artifact:v1", key.ID()),
		DeterminismFixtureHash: generator.HashFields("skill-arena:maze-generator-fixtures:v1", key.ID()),
		ReleasedAt:             released, CreatedAt: released,
	}
}

func productionDifficultyProfile(mode string, created time.Time) generator.DifficultyProfile {
	source := mode
	if source == "pvp" {
		source = "ranked"
	}
	profile := generator.DifficultyProfile{
		ID: "maze-" + mode + "-standard-v1", GameID: generator.GameID,
		SchemaVersion: 1, Source: source,
		ComplexityMin: 0, ComplexityMax: 10_000_000,
		LineCountMin: 12, LineCountMax: 20,
		DependencyDepthMin: 1, DependencyDepthMax: 64,
		BranchingMin: 0, BranchingMax: 64,
		FalseRoutesMin: 0, FalseRoutesMax: 128,
		DensityMinBPS: 0, DensityMaxBPS: 10_000,
		PatternBias: "any", ExpectedSolveTimeMinMS: 0,
		ExpectedSolveTimeMaxMS: 3_600_000,
		VisualComplexityMin:    0, VisualComplexityMax: 100,
		CreatedAt: created,
	}
	hash, err := generator.CanonicalProfileHash(profile)
	if err != nil {
		panic(err)
	}
	profile.ProfileHash = hash
	return profile
}

func numericVersion(value string) int {
	value = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "v"))
	parts := strings.Split(value, ".")
	version, err := strconv.Atoi(parts[0])
	if err != nil || version <= 0 {
		return 1
	}
	return version
}

var _ interfaces.Runtime = (*Runtime)(nil)
var _ interfaces.DeadlineRuntime = (*Runtime)(nil)
var _ interfaces.AuthoritativeReplayRuntime = (*Runtime)(nil)
