package replay

import (
	"bytes"
	"encoding/binary"
	"sort"

	"skill-arena/internal/games/maze/generator"
)

func genesisHash(genesis Genesis) string {
	buffer := &bytes.Buffer{}
	writeString(buffer, genesis.ReplayID)
	writeString(buffer, genesis.MatchID)
	writeString(buffer, genesis.PuzzleID)
	writeString(buffer, genesis.GameID)
	writeVersions(buffer, genesis.Versions)
	writeString(buffer, genesis.SeedReference)
	writeString(buffer, genesis.SeedHash)
	writeString(buffer, genesis.DifficultyID)
	writeString(buffer, genesis.DifficultyHash)
	writeString(buffer, genesis.AnalysisHash)
	writeString(buffer, genesis.GenerationHash)
	writeString(buffer, genesis.PuzzleHash)
	writeString(buffer, genesis.ValidationHash)
	writeString(buffer, genesis.SolutionHash)
	writeUint64(buffer, uint64(genesis.MinimumActions))
	writeInt64(buffer, genesis.CreatedAtUnixMS)
	return generator.HashBytes("skill-arena:maze-replay-genesis:v1", buffer.Bytes())
}

func eventHash(genesis Genesis, event Event) string {
	buffer := &bytes.Buffer{}
	writeString(buffer, genesis.GenesisHash)
	writeUint64(buffer, event.Sequence)
	writeString(buffer, event.ParticipantID)
	writeInt64(buffer, event.OffsetMS)
	writeString(buffer, event.Kind)
	writeString(buffer, event.ArrowID)
	writeString(buffer, event.Code)
	writeUint64(buffer, event.StateVersion)
	writeString(buffer, event.BlockerID)
	writeInt64(buffer, int64(event.CollisionCell.Column))
	writeInt64(buffer, int64(event.CollisionCell.Row))
	writeUint64(buffer, uint64(event.CollisionDistance))
	writeUint64(buffer, uint64(event.EscapeDistance))
	writeString(buffer, event.StateChecksum)
	writeString(buffer, event.PreviousHash)
	return generator.HashBytes("skill-arena:maze-replay-event:v1", buffer.Bytes())
}

func legacyStateChecksum(
	genesis Genesis,
	participantID string,
	stateVersion uint64,
	sequence uint64,
	removedIDs []string,
	successful int,
	blocked int,
	complete bool,
) string {
	buffer := &bytes.Buffer{}
	writeString(buffer, genesis.PuzzleHash)
	writeString(buffer, participantID)
	writeUint64(buffer, uint64(genesis.Versions.StateSchemaVersion))
	writeUint64(buffer, stateVersion)
	writeUint64(buffer, sequence)
	ids := append([]string(nil), removedIDs...)
	sort.Strings(ids)
	writeUint64(buffer, uint64(len(ids)))
	for _, id := range ids {
		writeString(buffer, id)
	}
	writeUint64(buffer, uint64(successful))
	writeUint64(buffer, uint64(blocked))
	writeBool(buffer, complete)
	return generator.HashBytes("skill-arena:maze-replay-state:v1", buffer.Bytes())
}

func replayHash(artifact Artifact) string {
	buffer := &bytes.Buffer{}
	writeString(buffer, artifact.Genesis.GenesisHash)
	writeString(buffer, artifact.EventRootHash)
	writeUint64(buffer, uint64(len(artifact.Events)))
	writeInt64(buffer, artifact.StartedAtUnixMS)
	writeInt64(buffer, artifact.EndedAtUnixMS)
	participants := append([]ParticipantResult(nil), artifact.Participants...)
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].ParticipantID < participants[j].ParticipantID
	})
	writeUint64(buffer, uint64(len(participants)))
	for _, participant := range participants {
		writeString(buffer, participant.ParticipantID)
		writeUint64(buffer, participant.StateVersion)
		writeUint64(buffer, uint64(participant.SuccessfulActions))
		writeUint64(buffer, uint64(participant.BlockedActions))
		if artifact.Genesis.Versions.ReplayVersion >= ReplayVersionEngine {
			writeUint64(buffer, uint64(participant.CurrentCombo))
			writeUint64(buffer, uint64(participant.MaximumCombo))
		}
		writeBool(buffer, participant.Completed)
		writeInt64(buffer, participant.CompletedAtMS)
		writeString(buffer, participant.StateChecksum)
	}
	writeOutcome(buffer, artifact.Outcome)
	return generator.HashBytes("skill-arena:maze-replay:v1", buffer.Bytes())
}

func writeVersions(buffer *bytes.Buffer, versions Versions) {
	writeString(buffer, versions.GameVersion)
	writeUint64(buffer, uint64(versions.ProtocolVersion))
	writeUint64(buffer, uint64(versions.ReplayVersion))
	writeUint64(buffer, uint64(versions.RendererVersion))
	writeUint64(buffer, uint64(versions.StateSchemaVersion))
	writeString(buffer, versions.Generator.ID())
}

func writeOutcome(buffer *bytes.Buffer, outcome Outcome) {
	writeString(buffer, outcome.Status)
	winners := append([]string(nil), outcome.WinnerIDs...)
	losers := append([]string(nil), outcome.LoserIDs...)
	sort.Strings(winners)
	sort.Strings(losers)
	writeUint64(buffer, uint64(len(winners)))
	for _, id := range winners {
		writeString(buffer, id)
	}
	writeUint64(buffer, uint64(len(losers)))
	for _, id := range losers {
		writeString(buffer, id)
	}
	writeString(buffer, outcome.Reason)
}

func writeString(buffer *bytes.Buffer, value string) {
	writeUint64(buffer, uint64(len(value)))
	buffer.WriteString(value)
}

func writeUint64(buffer *bytes.Buffer, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	buffer.Write(encoded[:])
}

func writeInt64(buffer *bytes.Buffer, value int64) {
	writeUint64(buffer, uint64(value))
}

func writeBool(buffer *bytes.Buffer, value bool) {
	if value {
		buffer.WriteByte(1)
		return
	}
	buffer.WriteByte(0)
}
