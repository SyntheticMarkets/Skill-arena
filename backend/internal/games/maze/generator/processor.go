package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
)

const (
	RejectionCanceled             = "generation_canceled"
	RejectionRandomStream         = "random_stream_failed"
	RejectionProfileInvalid       = "profile_invalid"
	RejectionProfileHash          = "profile_hash_mismatch"
	RejectionPatternUnavailable   = "pattern_unavailable"
	RejectionPatternMismatch      = "pattern_mismatch"
	RejectionPlacementExhausted   = "placement_exhausted"
	RejectionGeometryInvalid      = "geometry_invalid"
	RejectionDependencyInvalid    = "dependency_invalid"
	RejectionDependencyOrder      = "dependency_order_invalid"
	RejectionCycle                = "dependency_cycle"
	RejectionIsolatedArrow        = "isolated_arrow"
	RejectionVerifierUnavailable  = "verifier_unavailable"
	RejectionVerifierRejected     = "verifier_rejected"
	RejectionVerifierInvalid      = "verifier_output_invalid"
	RejectionComplexityMismatch   = "complexity_mismatch"
	RejectionLineCountMismatch    = "line_count_mismatch"
	RejectionDepthMismatch        = "dependency_depth_mismatch"
	RejectionBranchingMismatch    = "branching_mismatch"
	RejectionFalseRoutesMismatch  = "false_routes_mismatch"
	RejectionDensityMismatch      = "density_mismatch"
	RejectionSolveTimeMismatch    = "solve_time_mismatch"
	RejectionVisualMismatch       = "visual_complexity_mismatch"
	RejectionCanonicalEncoding    = "canonical_encoding_failed"
	RejectionNoAcceptedCandidates = "no_accepted_candidates"
)

type IndependentVerifier interface {
	Verify(context.Context, VerificationInput) (Verification, error)
}

type VerificationInput struct {
	Board         Board
	SolverVersion int
	AllowIsolated bool
}

type VerificationMetrics struct {
	ArrowCount        int
	InitiallyOpen     int
	DependencyEdges   int
	DependencyDepth   int
	Branching         int
	CrossDependencies int
	IsolatedArrows    int
}

type Verification struct {
	Accepted       bool
	SolverVersion  int
	DependencyHash string
	SolutionHash   string
	MinimumActions int
	Classification string
	FinalChecksum  string
	Metrics        VerificationMetrics
}

type CandidateObservation struct {
	CandidateIndex int
	PatternID      string
	Accepted       bool
	RejectionCode  string
}

type Observer interface {
	ObserveCandidate(context.Context, CandidateObservation)
}

type GenerationReport struct {
	Attempted       int            `json:"attempted"`
	Accepted        int            `json:"accepted"`
	SelectedIndex   int            `json:"selectedIndex"`
	RejectionCounts map[string]int `json:"rejectionCounts"`
}

type QualifiedCandidate struct {
	Candidate Candidate
	Measured  MeasuredDifficulty
	Rank      [5]int64
	Result    ProcessingResult
	Report    GenerationReport
}

type ProductionProcessor struct {
	config   GenerationConfig
	verifier IndependentVerifier
	observer Observer
}

func NewProductionProcessor(config GenerationConfig, verifier IndependentVerifier, observer Observer) (*ProductionProcessor, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if verifier == nil {
		return nil, errors.New(RejectionVerifierUnavailable)
	}
	return &ProductionProcessor{config: config, verifier: verifier, observer: observer}, nil
}

func (p *ProductionProcessor) Process(ctx context.Context, input ProcessingInput) (ProcessingResult, error) {
	qualified, err := p.Generate(ctx, input)
	if err != nil {
		return ProcessingResult{}, err
	}
	return qualified.Result, nil
}

func (p *ProductionProcessor) Generate(ctx context.Context, input ProcessingInput) (QualifiedCandidate, error) {
	if err := ctx.Err(); err != nil {
		return QualifiedCandidate{}, err
	}
	if err := input.Metadata.Version.Validate(); err != nil {
		return QualifiedCandidate{}, fmt.Errorf("%s: %w", RejectionProfileInvalid, err)
	}
	if err := input.Profile.Validate(); err != nil {
		return QualifiedCandidate{}, fmt.Errorf("%s: %w", RejectionProfileInvalid, err)
	}
	profileHash, err := CanonicalProfileHash(input.Profile)
	if err != nil {
		return QualifiedCandidate{}, fmt.Errorf("%s: %w", RejectionProfileInvalid, err)
	}
	if profileHash != input.Profile.ProfileHash || input.Metadata.DifficultyHash != profileHash {
		return QualifiedCandidate{}, errors.New(RejectionProfileHash)
	}
	if input.Profile.GameID != input.Metadata.GameID ||
		input.Profile.SchemaVersion != input.Metadata.Version.DifficultySchemaVersion {
		return QualifiedCandidate{}, errors.New(RejectionProfileInvalid)
	}

	type candidateResult struct {
		qualified   QualifiedCandidate
		observation CandidateObservation
		err         error
	}
	results := make([]candidateResult, p.config.CandidateBatch)
	semaphore := make(chan struct{}, p.config.MaximumWorkers)
	var wg sync.WaitGroup
	for index := 0; index < p.config.CandidateBatch; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[index] = candidateResult{
					observation: CandidateObservation{CandidateIndex: index, RejectionCode: RejectionCanceled},
					err:         ctx.Err(),
				}
				return
			}
			qualified, code, candidateErr := p.qualifyCandidate(ctx, input, index)
			results[index] = candidateResult{
				qualified: qualified,
				observation: CandidateObservation{
					CandidateIndex: index, PatternID: qualified.Candidate.Pattern.ID,
					Accepted: candidateErr == nil && code == "", RejectionCode: code,
				},
				err: candidateErr,
			}
		}(index)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return QualifiedCandidate{}, err
	}

	report := GenerationReport{
		Attempted: p.config.CandidateBatch, SelectedIndex: -1,
		RejectionCounts: map[string]int{},
	}
	accepted := make([]QualifiedCandidate, 0, p.config.CandidateBatch)
	for _, result := range results {
		if p.observer != nil {
			p.observer.ObserveCandidate(ctx, result.observation)
		}
		if result.err != nil && errors.Is(result.err, context.Canceled) {
			return QualifiedCandidate{}, result.err
		}
		if result.observation.Accepted {
			accepted = append(accepted, result.qualified)
			continue
		}
		code := result.observation.RejectionCode
		if code == "" {
			code = RejectionNoAcceptedCandidates
		}
		report.RejectionCounts[code]++
	}
	report.Accepted = len(accepted)
	if len(accepted) == 0 {
		return QualifiedCandidate{}, &GenerationError{Report: report}
	}
	sort.Slice(accepted, func(i, j int) bool {
		return rankLess(accepted[i].Rank, accepted[j].Rank)
	})
	selected := accepted[0]
	report.SelectedIndex = selected.Candidate.Index
	selected.Report = report
	return selected, nil
}

func (p *ProductionProcessor) qualifyCandidate(ctx context.Context, input ProcessingInput, index int) (QualifiedCandidate, string, error) {
	candidate, code, err := generateCandidate(ctx, input, p.config, index)
	if err != nil || code != "" {
		return QualifiedCandidate{Candidate: candidate}, code, err
	}
	allowIsolated := input.Profile.Source == "tutorial"
	if err := ValidateDependencyGraph(candidate.Board, candidate.Graph, allowIsolated); err != nil {
		code := RejectionDependencyInvalid
		if err.Error() == "dependency graph contains a cycle" {
			code = RejectionCycle
		} else if err.Error() == "competitive puzzle contains an isolated arrow" {
			code = RejectionIsolatedArrow
		}
		return QualifiedCandidate{Candidate: candidate}, code, err
	}
	measured, err := MeasureDifficulty(candidate.Board, candidate.Graph)
	if err != nil {
		return QualifiedCandidate{Candidate: candidate}, RejectionDependencyInvalid, err
	}
	if accepted, mismatch := CompareDifficulty(input.Profile, measured, candidate.Pattern); !accepted {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, mismatch, nil
	}
	verification, err := p.verifier.Verify(ctx, VerificationInput{
		Board: candidate.Board.Clone(), SolverVersion: input.Metadata.Version.SolverVersion,
		AllowIsolated: allowIsolated,
	})
	if err != nil {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionVerifierRejected, err
	}
	if !verification.Accepted {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionVerifierRejected, nil
	}
	graphBytes, err := CanonicalGraph(candidate.Graph)
	if err != nil {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionCanonicalEncoding, err
	}
	expectedDependencyHash := HashBytes(
		"skill-arena:maze-dependency-graph:v1", graphBytes,
		[]byte(intString(input.Metadata.Version.SolverVersion)),
	)
	if verification.SolverVersion != input.Metadata.Version.SolverVersion ||
		verification.DependencyHash != expectedDependencyHash ||
		!ValidHash(verification.SolutionHash) || !ValidHash(verification.FinalChecksum) ||
		verification.MinimumActions != len(candidate.Board.Arrows) ||
		(verification.Classification != "unique" && verification.Classification != "multiple") ||
		!verificationMetricsMatch(verification.Metrics, measured) {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionVerifierInvalid, nil
	}
	boardBytes, err := CanonicalBoard(candidate.Board)
	if err != nil {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionCanonicalEncoding, err
	}
	measuredBytes, err := measured.Canonical()
	if err != nil {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionCanonicalEncoding, err
	}
	patternBytes, err := json.Marshal(candidate.Pattern)
	if err != nil {
		return QualifiedCandidate{Candidate: candidate, Measured: measured}, RejectionCanonicalEncoding, err
	}
	puzzleHash := HashBytes(
		"skill-arena:maze-puzzle:v1", boardBytes,
		[]byte(input.Metadata.Version.ID()),
	)
	analysisHash := HashBytes(
		"skill-arena:maze-difficulty-analysis:v1", measuredBytes,
		[]byte(input.Metadata.Version.ID()), []byte(input.Profile.ProfileHash),
	)
	generationHash := HashBytes(
		"skill-arena:maze-generation:v1",
		[]byte(input.Metadata.GameID), []byte(input.Metadata.Version.ID()),
		[]byte(input.Metadata.SeedHash), []byte(input.Profile.ProfileHash),
		patternBytes, []byte(intString(candidate.Index)), boardBytes,
	)
	validationHash := HashBytes(
		"skill-arena:maze-validation:v1",
		[]byte(puzzleHash), graphBytes, []byte(verification.SolutionHash),
		[]byte(analysisHash), []byte(verification.FinalChecksum),
		[]byte(intString(input.Metadata.Version.ValidatorVersion)),
		[]byte(intString(input.Metadata.Version.AnalyzerVersion)),
	)
	result := ProcessingResult{
		GenerationHash: generationHash, PuzzleHash: puzzleHash,
		ValidationHash: validationHash, SolutionHash: verification.SolutionHash,
		MinimumActions: verification.MinimumActions,
		Analysis: DifficultyAnalysis{
			AnalyzerVersion: input.Metadata.Version.AnalyzerVersion,
			Accepted:        true, Classification: verification.Classification,
			MeasuredFields: measuredBytes, AnalysisHash: analysisHash,
		},
	}
	return QualifiedCandidate{
		Candidate: candidate, Measured: measured,
		Rank:   candidateRank(input.Profile, measured, candidate.Pattern, candidate.Index),
		Result: result,
	}, "", nil
}

func verificationMetricsMatch(metrics VerificationMetrics, measured MeasuredDifficulty) bool {
	return metrics.ArrowCount == measured.ArrowCount &&
		metrics.InitiallyOpen == measured.InitiallyOpen &&
		metrics.DependencyEdges == measured.DependencyEdges &&
		metrics.DependencyDepth == measured.DependencyDepth &&
		metrics.Branching == measured.Branching &&
		metrics.CrossDependencies == measured.CrossDependencies &&
		metrics.IsolatedArrows == measured.IsolatedArrows
}

type GenerationError struct {
	Report GenerationReport
}

func (e *GenerationError) Error() string {
	return RejectionNoAcceptedCandidates
}
