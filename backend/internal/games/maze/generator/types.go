package generator

import (
	"errors"
	"strings"
	"time"
)

const (
	GameID = "maze_arena"

	VersionQualification = "qualification"
	VersionActive        = "active"
	VersionReplayOnly    = "replay_only"
	VersionRetired       = "retired"
	VersionRevoked       = "revoked"

	PuzzlePreparing = "preparing"
	PuzzleValidated = "validated"
	PuzzleRejected  = "rejected"
	PuzzleAssigned  = "assigned"
	PuzzleConsumed  = "consumed"
	PuzzleRetired   = "retired"

	ReuseOneUse         = "one_use"
	ReuseTutorial       = "tutorial_fixture"
	ReuseDailyWindow    = "daily_window"
	SeedFormatVersion   = 1
	RandomStreamVersion = 1
)

var (
	ErrNotFound           = errors.New("puzzle metadata not found")
	ErrConflict           = errors.New("puzzle metadata conflict")
	ErrDuplicatePuzzle    = errors.New("puzzle uniqueness claim already exists")
	ErrInvalidTransition  = errors.New("invalid puzzle state transition")
	ErrVersionUnavailable = errors.New("generator version is unavailable")
)

type VersionKey struct {
	GameID                   string `json:"gameId"`
	GeneratorVersion         int    `json:"generatorVersion"`
	SeedFormatVersion        int    `json:"seedFormatVersion"`
	RandomStreamVersion      int    `json:"randomStreamVersion"`
	PatternCatalogueVersion  int    `json:"patternCatalogueVersion"`
	PatternSelectionVersion  int    `json:"patternSelectionVersion"`
	GeometrySchemaVersion    int    `json:"geometrySchemaVersion"`
	CandidateScoringVersion  int    `json:"candidateScoringVersion"`
	ConstraintPolicyVersion  int    `json:"constraintPolicyVersion"`
	SolverVersion            int    `json:"solverVersion"`
	ValidatorVersion         int    `json:"validatorVersion"`
	AnalyzerVersion          int    `json:"analyzerVersion"`
	DifficultySchemaVersion  int    `json:"difficultySchemaVersion"`
	CanonicalEncodingVersion int    `json:"canonicalEncodingVersion"`
}

func (v VersionKey) Validate() error {
	if strings.TrimSpace(v.GameID) == "" {
		return errors.New("game id is required")
	}
	versions := []int{
		v.GeneratorVersion, v.SeedFormatVersion, v.RandomStreamVersion,
		v.PatternCatalogueVersion, v.PatternSelectionVersion, v.GeometrySchemaVersion,
		v.CandidateScoringVersion, v.ConstraintPolicyVersion, v.SolverVersion,
		v.ValidatorVersion, v.AnalyzerVersion, v.DifficultySchemaVersion,
		v.CanonicalEncodingVersion,
	}
	for _, version := range versions {
		if version <= 0 {
			return errors.New("all generator version components must be positive")
		}
	}
	return nil
}

func (v VersionKey) ID() string {
	return HashFields("skill-arena:generator-version:v1",
		v.GameID,
		intString(v.GeneratorVersion),
		intString(v.SeedFormatVersion),
		intString(v.RandomStreamVersion),
		intString(v.PatternCatalogueVersion),
		intString(v.PatternSelectionVersion),
		intString(v.GeometrySchemaVersion),
		intString(v.CandidateScoringVersion),
		intString(v.ConstraintPolicyVersion),
		intString(v.SolverVersion),
		intString(v.ValidatorVersion),
		intString(v.AnalyzerVersion),
		intString(v.DifficultySchemaVersion),
		intString(v.CanonicalEncodingVersion),
	)
}

type GeneratorVersion struct {
	Key                    VersionKey `json:"key"`
	Status                 string     `json:"status"`
	NewMatchAllowed        bool       `json:"newMatchAllowed"`
	ArtifactDigest         string     `json:"artifactDigest"`
	DeterminismFixtureHash string     `json:"determinismFixtureHash"`
	ReleasedAt             time.Time  `json:"releasedAt"`
	CreatedAt              time.Time  `json:"createdAt"`
}

func (v GeneratorVersion) Validate() error {
	if err := v.Key.Validate(); err != nil {
		return err
	}
	if !oneOf(v.Status, VersionQualification, VersionActive, VersionReplayOnly, VersionRetired, VersionRevoked) {
		return errors.New("invalid generator version status")
	}
	if v.NewMatchAllowed && v.Status != VersionActive {
		return errors.New("new matches require an active generator version")
	}
	if !ValidHash(v.ArtifactDigest) || !ValidHash(v.DeterminismFixtureHash) {
		return errors.New("artifact and determinism fixture hashes are required")
	}
	if v.ReleasedAt.IsZero() || v.CreatedAt.IsZero() {
		return errors.New("generator version timestamps are required")
	}
	return nil
}

type DifficultyProfile struct {
	ID                     string    `json:"id"`
	GameID                 string    `json:"gameId"`
	SchemaVersion          int       `json:"schemaVersion"`
	Source                 string    `json:"source"`
	ComplexityMin          int64     `json:"complexityMin"`
	ComplexityMax          int64     `json:"complexityMax"`
	LineCountMin           int       `json:"lineCountMin"`
	LineCountMax           int       `json:"lineCountMax"`
	DependencyDepthMin     int       `json:"dependencyDepthMin"`
	DependencyDepthMax     int       `json:"dependencyDepthMax"`
	BranchingMin           int       `json:"branchingMin"`
	BranchingMax           int       `json:"branchingMax"`
	FalseRoutesMin         int       `json:"falseRoutesMin"`
	FalseRoutesMax         int       `json:"falseRoutesMax"`
	DensityMinBPS          int       `json:"densityMinBps"`
	DensityMaxBPS          int       `json:"densityMaxBps"`
	PatternBias            string    `json:"patternBias"`
	ExpectedSolveTimeMinMS int64     `json:"expectedSolveTimeMinMs"`
	ExpectedSolveTimeMaxMS int64     `json:"expectedSolveTimeMaxMs"`
	VisualComplexityMin    int       `json:"visualComplexityMin"`
	VisualComplexityMax    int       `json:"visualComplexityMax"`
	ProfileHash            string    `json:"profileHash"`
	CreatedAt              time.Time `json:"createdAt"`
}

func (p DifficultyProfile) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.GameID) == "" || p.SchemaVersion <= 0 {
		return errors.New("difficulty profile identity is incomplete")
	}
	if !oneOf(p.Source, "practice", "ranked", "house", "daily", "tournament", "tutorial", "calibration") {
		return errors.New("invalid difficulty profile source")
	}
	if p.ComplexityMin < 0 || p.ComplexityMax < p.ComplexityMin ||
		p.LineCountMin < 0 || p.LineCountMax < p.LineCountMin ||
		p.DependencyDepthMin < 0 || p.DependencyDepthMax < p.DependencyDepthMin ||
		p.BranchingMin < 0 || p.BranchingMax < p.BranchingMin ||
		p.FalseRoutesMin < 0 || p.FalseRoutesMax < p.FalseRoutesMin ||
		p.DensityMinBPS < 0 || p.DensityMaxBPS > 10_000 || p.DensityMaxBPS < p.DensityMinBPS ||
		p.ExpectedSolveTimeMinMS < 0 || p.ExpectedSolveTimeMaxMS < p.ExpectedSolveTimeMinMS ||
		p.VisualComplexityMin < 0 || p.VisualComplexityMax < p.VisualComplexityMin {
		return errors.New("difficulty profile ranges are invalid")
	}
	if !ValidHash(p.ProfileHash) || p.CreatedAt.IsZero() {
		return errors.New("difficulty profile hash and creation time are required")
	}
	return nil
}

type PuzzleMetadata struct {
	ID             string     `json:"id"`
	GameID         string     `json:"gameId"`
	Mode           string     `json:"mode"`
	Status         string     `json:"status"`
	Version        VersionKey `json:"version"`
	DifficultyID   string     `json:"difficultyProfileId"`
	DifficultyHash string     `json:"difficultyProfileHash"`
	RequestHash    string     `json:"requestHash"`
	SeedReference  string     `json:"seedReference"`
	SeedKeyID      string     `json:"seedKeyId"`
	SeedHash       string     `json:"seedHash"`
	SeedCiphertext []byte     `json:"-"`
	SeedNonce      []byte     `json:"-"`
	GenerationHash string     `json:"generationHash,omitempty"`
	PuzzleHash     string     `json:"puzzleHash,omitempty"`
	ValidationHash string     `json:"validationHash,omitempty"`
	SolutionHash   string     `json:"solutionHash,omitempty"`
	AnalysisID     string     `json:"analysisId,omitempty"`
	MinimumActions int        `json:"minimumActions,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	ValidatedAt    *time.Time `json:"validatedAt,omitempty"`
	AssignedAt     *time.Time `json:"assignedAt,omitempty"`
}

type DifficultyAnalysis struct {
	ID              string    `json:"id"`
	PuzzleID        string    `json:"puzzleId"`
	AnalyzerVersion int       `json:"analyzerVersion"`
	Accepted        bool      `json:"accepted"`
	Classification  string    `json:"classification"`
	MeasuredFields  []byte    `json:"measuredFields"`
	AnalysisHash    string    `json:"analysisHash"`
	RejectionCode   string    `json:"rejectionCode,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Assignment struct {
	ID          string     `json:"id"`
	PuzzleID    string     `json:"puzzleId"`
	Mode        string     `json:"mode"`
	ScopeType   string     `json:"scopeType"`
	ScopeID     string     `json:"scopeId"`
	ReusePolicy string     `json:"reusePolicy"`
	AssignedAt  time.Time  `json:"assignedAt"`
	ConsumedAt  *time.Time `json:"consumedAt,omitempty"`
}

type Finalization struct {
	PuzzleID       string
	GenerationHash string
	PuzzleHash     string
	ValidationHash string
	SolutionHash   string
	MinimumActions int
	Analysis       DifficultyAnalysis
	Assignment     Assignment
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
