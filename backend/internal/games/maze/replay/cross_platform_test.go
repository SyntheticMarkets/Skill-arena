package replay

import (
	"testing"

	"skill-arena/internal/games/maze/generator"
)

func TestPhase9CrossPlatformReplayVector(t *testing.T) {
	version := generator.VersionKey{
		GameID: generator.GameID, GeneratorVersion: 1,
		SeedFormatVersion: 1, RandomStreamVersion: 1,
		PatternCatalogueVersion: 1, PatternSelectionVersion: 1,
		GeometrySchemaVersion: 1, CandidateScoringVersion: 1,
		ConstraintPolicyVersion: 1, SolverVersion: 1, ValidatorVersion: 1,
		AnalyzerVersion: 1, DifficultySchemaVersion: 1,
		CanonicalEncodingVersion: 1,
	}
	genesis := Genesis{
		ReplayID: "phase9-replay-vector", MatchID: "phase9-match-vector",
		PuzzleID: "phase9-puzzle-vector", GameID: generator.GameID,
		Versions: Versions{
			GameVersion: "1.0.0", ProtocolVersion: 1,
			ReplayVersion: ReplayVersionEngine, RendererVersion: 1,
			StateSchemaVersion: 1, Generator: version,
		},
		SeedReference:  "phase9-seed-reference",
		SeedHash:       generator.HashFields("phase9", "seed"),
		DifficultyID:   "phase9-standard-v1",
		DifficultyHash: generator.HashFields("phase9", "difficulty"),
		AnalysisHash:   generator.HashFields("phase9", "analysis"),
		GenerationHash: generator.HashFields("phase9", "generation"),
		PuzzleHash:     generator.HashFields("phase9", "puzzle"),
		ValidationHash: generator.HashFields("phase9", "validation"),
		SolutionHash:   generator.HashFields("phase9", "solution"),
		MinimumActions: 2, CreatedAtUnixMS: 1_785_369_600_000,
	}
	genesis.GenesisHash = genesisHash(genesis)
	event := Event{
		EventDraft: EventDraft{
			Sequence: 1, ParticipantID: "player-phase9", OffsetMS: 125,
			Kind: EventArrowAccepted, ArrowID: "a0000", Code: "ACTION_ACCEPTED",
		},
		StateVersion: 1, EscapeDistance: 2,
		StateChecksum: generator.HashFields("phase9", "state"),
		PreviousHash:  genesis.GenesisHash,
	}
	event.IntegrityHash = eventHash(genesis, event)
	artifact := Artifact{
		Genesis: genesis, Events: []Event{event},
		Participants: []ParticipantResult{{
			ParticipantID: "player-phase9", StateVersion: 1,
			SuccessfulActions: 1, CurrentCombo: 1, MaximumCombo: 1,
			StateChecksum: event.StateChecksum,
		}},
		Outcome:         Outcome{Status: "active"},
		StartedAtUnixMS: genesis.CreatedAtUnixMS,
		EndedAtUnixMS:   genesis.CreatedAtUnixMS + 125,
		EventRootHash:   event.IntegrityHash,
	}
	artifact.ReplayHash = replayHash(artifact)
	const (
		expectedGenesis = "sha256:0ecda9ac310934b9eedc5165abaff87448ff0df42d0cb5fbd281cc71ac1e24b8"
		expectedEvent   = "sha256:682883a261d82656e4cdc41f0cbddc26f5d6f4b16b335b74de9429d0d31f1234"
		expectedReplay  = "sha256:c2d52609cb53291bac31bcde2d2d20a00b2b322517f9140e41e1a2f4a3c3e936"
	)
	if genesis.GenesisHash != expectedGenesis ||
		event.IntegrityHash != expectedEvent ||
		artifact.ReplayHash != expectedReplay {
		t.Fatalf(
			"cross-platform replay vector genesis=%s event=%s replay=%s",
			genesis.GenesisHash, event.IntegrityHash, artifact.ReplayHash,
		)
	}
}
