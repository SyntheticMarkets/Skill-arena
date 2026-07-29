package games

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	mazegenerator "skill-arena/internal/games/maze/generator"

	"github.com/lib/pq"
)

type PuzzleRepository struct {
	db *sql.DB
}

func NewPuzzleRepository(db *sql.DB) (*PuzzleRepository, error) {
	if db == nil {
		return nil, errors.New("PostgreSQL connection is required")
	}
	return &PuzzleRepository{db: db}, nil
}

func (r *PuzzleRepository) RegisterVersion(ctx context.Context, version mazegenerator.GeneratorVersion) error {
	if err := version.Validate(); err != nil {
		return err
	}
	record, err := json.Marshal(version)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
INSERT INTO game_generator_versions(
  version_id,game_id,generator_version,seed_format_version,random_stream_version,
  pattern_catalogue_version,pattern_selection_version,geometry_schema_version,
  candidate_scoring_version,constraint_policy_version,solver_version,validator_version,
  analyzer_version,difficulty_schema_version,canonical_encoding_version,status,
  new_match_allowed,artifact_digest,determinism_fixture_hash,record,released_at,created_at
) VALUES(
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22
) ON CONFLICT(version_id) DO UPDATE SET version_id=EXCLUDED.version_id
WHERE game_generator_versions.record=EXCLUDED.record`,
		version.Key.ID(), version.Key.GameID, version.Key.GeneratorVersion,
		version.Key.SeedFormatVersion, version.Key.RandomStreamVersion,
		version.Key.PatternCatalogueVersion, version.Key.PatternSelectionVersion,
		version.Key.GeometrySchemaVersion, version.Key.CandidateScoringVersion,
		version.Key.ConstraintPolicyVersion, version.Key.SolverVersion,
		version.Key.ValidatorVersion, version.Key.AnalyzerVersion,
		version.Key.DifficultySchemaVersion, version.Key.CanonicalEncodingVersion,
		version.Status, version.NewMatchAllowed, version.ArtifactDigest,
		version.DeterminismFixtureHash, record, version.ReleasedAt, version.CreatedAt)
	if err != nil {
		return translateError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return mazegenerator.ErrConflict
	}
	return nil
}

func (r *PuzzleRepository) GetVersion(ctx context.Context, key mazegenerator.VersionKey) (mazegenerator.GeneratorVersion, error) {
	var record []byte
	err := r.db.QueryRowContext(ctx, `SELECT record FROM game_generator_versions WHERE version_id=$1`, key.ID()).Scan(&record)
	if errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.GeneratorVersion{}, mazegenerator.ErrNotFound
	}
	if err != nil {
		return mazegenerator.GeneratorVersion{}, err
	}
	var version mazegenerator.GeneratorVersion
	if err := json.Unmarshal(record, &version); err != nil {
		return mazegenerator.GeneratorVersion{}, err
	}
	return version, nil
}

func (r *PuzzleRepository) SaveDifficultyProfile(ctx context.Context, profile mazegenerator.DifficultyProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	record, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `
INSERT INTO game_difficulty_profiles(profile_id,game_id,schema_version,source,profile_hash,record,created_at)
VALUES($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT(profile_id) DO UPDATE SET profile_id=EXCLUDED.profile_id
WHERE game_difficulty_profiles.record=EXCLUDED.record`,
		profile.ID, profile.GameID, profile.SchemaVersion, profile.Source,
		profile.ProfileHash, record, profile.CreatedAt)
	if err != nil {
		return translateError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return mazegenerator.ErrConflict
	}
	return nil
}

func (r *PuzzleRepository) GetDifficultyProfile(ctx context.Context, id string) (mazegenerator.DifficultyProfile, error) {
	var record []byte
	err := r.db.QueryRowContext(ctx, `SELECT record FROM game_difficulty_profiles WHERE profile_id=$1`, id).Scan(&record)
	if errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.DifficultyProfile{}, mazegenerator.ErrNotFound
	}
	if err != nil {
		return mazegenerator.DifficultyProfile{}, err
	}
	var profile mazegenerator.DifficultyProfile
	if err := json.Unmarshal(record, &profile); err != nil {
		return mazegenerator.DifficultyProfile{}, err
	}
	return profile, nil
}

func (r *PuzzleRepository) CreatePuzzle(ctx context.Context, puzzle mazegenerator.PuzzleMetadata) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO game_puzzles(
  puzzle_id,game_id,mode,status,generator_version_id,difficulty_profile_id,
  difficulty_profile_hash,request_hash,seed_reference,seed_key_id,seed_hash,
  seed_ciphertext,seed_nonce,created_at
) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		puzzle.ID, puzzle.GameID, puzzle.Mode, puzzle.Status, puzzle.Version.ID(),
		puzzle.DifficultyID, puzzle.DifficultyHash, puzzle.RequestHash, puzzle.SeedReference,
		puzzle.SeedKeyID, puzzle.SeedHash, puzzle.SeedCiphertext, puzzle.SeedNonce,
		puzzle.CreatedAt)
	return translateError(err)
}

func (r *PuzzleRepository) GetPuzzle(ctx context.Context, id string) (mazegenerator.PuzzleMetadata, error) {
	var puzzle mazegenerator.PuzzleMetadata
	var versionRecord []byte
	err := r.db.QueryRowContext(ctx, `
SELECT p.puzzle_id,p.game_id,p.mode,p.status,v.record,p.difficulty_profile_id,
 p.difficulty_profile_hash,p.request_hash,p.seed_reference,p.seed_key_id,p.seed_hash,
 p.seed_ciphertext,p.seed_nonce,COALESCE(p.generation_hash,''),COALESCE(p.puzzle_hash,''),
 COALESCE(p.validation_hash,''),COALESCE(p.solution_hash,''),COALESCE(p.accepted_analysis_id,''),
 COALESCE(p.minimum_actions,0),p.created_at,p.validated_at,p.assigned_at
FROM game_puzzles p
JOIN game_generator_versions v ON v.version_id=p.generator_version_id
WHERE p.puzzle_id=$1`, id).Scan(
		&puzzle.ID, &puzzle.GameID, &puzzle.Mode, &puzzle.Status, &versionRecord,
		&puzzle.DifficultyID, &puzzle.DifficultyHash, &puzzle.RequestHash, &puzzle.SeedReference,
		&puzzle.SeedKeyID, &puzzle.SeedHash, &puzzle.SeedCiphertext, &puzzle.SeedNonce,
		&puzzle.GenerationHash, &puzzle.PuzzleHash, &puzzle.ValidationHash,
		&puzzle.SolutionHash, &puzzle.AnalysisID, &puzzle.MinimumActions,
		&puzzle.CreatedAt, &puzzle.ValidatedAt, &puzzle.AssignedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.PuzzleMetadata{}, mazegenerator.ErrNotFound
	}
	if err != nil {
		return mazegenerator.PuzzleMetadata{}, err
	}
	var version mazegenerator.GeneratorVersion
	if err := json.Unmarshal(versionRecord, &version); err != nil {
		return mazegenerator.PuzzleMetadata{}, err
	}
	puzzle.Version = version.Key
	return puzzle, nil
}

func (r *PuzzleRepository) GetPuzzleByRequestHash(ctx context.Context, requestHash string) (mazegenerator.PuzzleMetadata, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT puzzle_id FROM game_puzzles WHERE request_hash=$1`, requestHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.PuzzleMetadata{}, mazegenerator.ErrNotFound
	}
	if err != nil {
		return mazegenerator.PuzzleMetadata{}, err
	}
	return r.GetPuzzle(ctx, id)
}

func (r *PuzzleRepository) GetDifficultyAnalysis(ctx context.Context, id string) (mazegenerator.DifficultyAnalysis, error) {
	var analysis mazegenerator.DifficultyAnalysis
	err := r.db.QueryRowContext(ctx, `
SELECT analysis_id,puzzle_id,analyzer_version,accepted,classification,
 measured_fields,analysis_hash,COALESCE(rejection_code,''),created_at
FROM game_difficulty_analyses WHERE analysis_id=$1`, id).Scan(
		&analysis.ID, &analysis.PuzzleID, &analysis.AnalyzerVersion,
		&analysis.Accepted, &analysis.Classification, &analysis.MeasuredFields,
		&analysis.AnalysisHash, &analysis.RejectionCode, &analysis.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.DifficultyAnalysis{}, mazegenerator.ErrNotFound
	}
	return analysis, err
}

func (r *PuzzleRepository) FinalizeAndAssign(ctx context.Context, final mazegenerator.Finalization) (mazegenerator.Assignment, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return mazegenerator.Assignment{}, err
	}
	defer tx.Rollback()

	var status, seedHash string
	if err := tx.QueryRowContext(ctx,
		`SELECT status,seed_hash FROM game_puzzles WHERE puzzle_id=$1 FOR UPDATE`,
		final.PuzzleID).Scan(&status, &seedHash); errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.Assignment{}, mazegenerator.ErrNotFound
	} else if err != nil {
		return mazegenerator.Assignment{}, err
	}
	if status != mazegenerator.PuzzlePreparing {
		return mazegenerator.Assignment{}, mazegenerator.ErrInvalidTransition
	}
	measured := final.Analysis.MeasuredFields
	if len(measured) == 0 {
		measured = []byte(`{}`)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO game_difficulty_analyses(
 analysis_id,puzzle_id,analyzer_version,accepted,classification,measured_fields,
 analysis_hash,rejection_code,created_at
) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9)`,
		final.Analysis.ID, final.PuzzleID, final.Analysis.AnalyzerVersion,
		final.Analysis.Accepted, final.Analysis.Classification, measured,
		final.Analysis.AnalysisHash, final.Analysis.RejectionCode,
		final.Analysis.CreatedAt); err != nil {
		return mazegenerator.Assignment{}, translateError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO game_puzzle_uniqueness_claims(
 claim_id,puzzle_id,seed_hash,puzzle_hash,reuse_policy,scope_type,scope_id,claimed_at
) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		"claim_"+final.Assignment.ID, final.PuzzleID, seedHash, final.PuzzleHash,
		final.Assignment.ReusePolicy, final.Assignment.ScopeType,
		final.Assignment.ScopeID, final.Assignment.AssignedAt); err != nil {
		return mazegenerator.Assignment{}, translateError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO game_puzzle_assignments(
 assignment_id,puzzle_id,mode,scope_type,scope_id,reuse_policy,assigned_at,consumed_at
) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		final.Assignment.ID, final.PuzzleID, final.Assignment.Mode,
		final.Assignment.ScopeType, final.Assignment.ScopeID,
		final.Assignment.ReusePolicy, final.Assignment.AssignedAt,
		final.Assignment.ConsumedAt); err != nil {
		return mazegenerator.Assignment{}, translateError(err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE game_puzzles SET
 status='assigned',generation_hash=$2,puzzle_hash=$3,validation_hash=$4,
 solution_hash=$5,accepted_analysis_id=$6,minimum_actions=$7,
 validated_at=$8,assigned_at=$9
WHERE puzzle_id=$1 AND status='preparing'`,
		final.PuzzleID, final.GenerationHash, final.PuzzleHash, final.ValidationHash,
		final.SolutionHash, final.Analysis.ID, final.MinimumActions,
		final.Analysis.CreatedAt, final.Assignment.AssignedAt)
	if err != nil {
		return mazegenerator.Assignment{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return mazegenerator.Assignment{}, err
	}
	if affected != 1 {
		return mazegenerator.Assignment{}, mazegenerator.ErrInvalidTransition
	}
	if err := tx.Commit(); err != nil {
		return mazegenerator.Assignment{}, translateError(err)
	}
	return final.Assignment, nil
}

func (r *PuzzleRepository) GetAssignment(ctx context.Context, scopeType, scopeID string) (mazegenerator.Assignment, error) {
	var assignment mazegenerator.Assignment
	err := r.db.QueryRowContext(ctx, `
SELECT assignment_id,puzzle_id,mode,scope_type,scope_id,reuse_policy,assigned_at,consumed_at
FROM game_puzzle_assignments WHERE scope_type=$1 AND scope_id=$2`,
		scopeType, scopeID).Scan(
		&assignment.ID, &assignment.PuzzleID, &assignment.Mode,
		&assignment.ScopeType, &assignment.ScopeID, &assignment.ReusePolicy,
		&assignment.AssignedAt, &assignment.ConsumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return mazegenerator.Assignment{}, mazegenerator.ErrNotFound
	}
	return assignment, err
}

func translateError(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		if pqErr.Constraint == "uq_game_puzzle_claims_one_use_seed" ||
			pqErr.Constraint == "uq_game_puzzle_claims_one_use_hash" {
			return fmt.Errorf("%w: %s", mazegenerator.ErrDuplicatePuzzle, pqErr.Constraint)
		}
		return fmt.Errorf("%w: %s", mazegenerator.ErrConflict, pqErr.Constraint)
	}
	return err
}
