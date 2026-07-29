package replay

import (
	"errors"
	"sort"
	"strings"

	"skill-arena/internal/games/maze/generator"
)

const (
	maxReplayEvents = 1_000_000
	maxIdentitySize = 256
	maxReasonSize   = 2_048
)

func validateSealRequest(request SealRequest) error {
	if err := validateGenesis(request.Genesis); err != nil {
		return err
	}
	if request.StartedAtUnixMS <= 0 || request.EndedAtUnixMS < request.StartedAtUnixMS {
		return errors.New("replay timing is invalid")
	}
	if _, err := canonicalParticipants(request.ParticipantIDs); err != nil {
		return err
	}
	if len(request.Events) > maxReplayEvents {
		return errors.New("replay event count exceeds production bounds")
	}
	duration := request.EndedAtUnixMS - request.StartedAtUnixMS
	for _, event := range request.Events {
		if event.OffsetMS > duration {
			return errors.New("replay event occurs after the replay ended")
		}
	}
	return nil
}

func validateArtifactEnvelope(artifact Artifact) error {
	if err := validateGenesis(artifact.Genesis); err != nil {
		return err
	}
	if artifact.StartedAtUnixMS <= 0 || artifact.EndedAtUnixMS < artifact.StartedAtUnixMS ||
		len(artifact.Events) > maxReplayEvents || !generator.ValidHash(artifact.EventRootHash) ||
		!generator.ValidHash(artifact.ReplayHash) ||
		strings.TrimSpace(artifact.Proof.Algorithm) == "" ||
		strings.TrimSpace(artifact.Proof.KeyID) == "" ||
		strings.TrimSpace(artifact.Proof.Signature) == "" {
		return errors.New("replay artifact envelope is invalid")
	}
	participantIDs := make([]string, len(artifact.Participants))
	duration := artifact.EndedAtUnixMS - artifact.StartedAtUnixMS
	for _, event := range artifact.Events {
		if event.OffsetMS > duration {
			return errors.New("replay event occurs after the replay ended")
		}
	}
	for index, participant := range artifact.Participants {
		participantIDs[index] = participant.ParticipantID
		if participant.CompletedAtMS < -1 || participant.CompletedAtMS > duration ||
			participant.CurrentCombo < 0 || participant.MaximumCombo < participant.CurrentCombo ||
			!generator.ValidHash(participant.StateChecksum) {
			return errors.New("replay participant result is invalid")
		}
	}
	_, err := canonicalParticipants(participantIDs)
	return err
}

func validateGenesis(genesis Genesis) error {
	if strings.TrimSpace(genesis.ReplayID) == "" || strings.TrimSpace(genesis.MatchID) == "" ||
		strings.TrimSpace(genesis.PuzzleID) == "" || strings.TrimSpace(genesis.GameID) == "" ||
		strings.TrimSpace(genesis.SeedReference) == "" ||
		strings.TrimSpace(genesis.DifficultyID) == "" || genesis.MinimumActions <= 0 ||
		genesis.CreatedAtUnixMS <= 0 || genesis.GenesisHash != genesisHash(genesis) {
		return errors.New("replay genesis is invalid")
	}
	for _, value := range []string{
		genesis.ReplayID, genesis.MatchID, genesis.PuzzleID, genesis.GameID,
		genesis.SeedReference, genesis.DifficultyID, genesis.Versions.GameVersion,
	} {
		if len(value) > maxIdentitySize {
			return errors.New("replay genesis identifier exceeds production bounds")
		}
	}
	if err := genesis.Versions.Validate(); err != nil {
		return err
	}
	hashes := []string{
		genesis.SeedHash, genesis.DifficultyHash, genesis.AnalysisHash,
		genesis.GenerationHash, genesis.PuzzleHash, genesis.ValidationHash,
		genesis.SolutionHash, genesis.GenesisHash,
	}
	for _, hash := range hashes {
		if !generator.ValidHash(hash) {
			return errors.New("replay genesis hash is invalid")
		}
	}
	return nil
}

func validateOutcome(outcome Outcome, participants []ParticipantResult) error {
	allowed := map[string]bool{
		"completed": true, "draw": true, "invalid": true, "canceled": true,
	}
	if !allowed[outcome.Status] {
		return errors.New("replay outcome status is invalid")
	}
	if len(outcome.Reason) > maxReasonSize {
		return errors.New("replay outcome reason exceeds production bounds")
	}
	known := make(map[string]ParticipantResult, len(participants))
	for _, participant := range participants {
		known[participant.ParticipantID] = participant
	}
	seen := map[string]bool{}
	for _, id := range append(append([]string(nil), outcome.WinnerIDs...), outcome.LoserIDs...) {
		if _, exists := known[id]; !exists || seen[id] {
			return errors.New("replay outcome participant is invalid")
		}
		seen[id] = true
	}
	if outcome.Status == "completed" {
		if len(outcome.WinnerIDs) == 0 {
			return errors.New("completed replay requires a winner")
		}
		for _, id := range outcome.WinnerIDs {
			if !known[id].Completed {
				return errors.New("replay winner did not complete the puzzle")
			}
		}
	} else if len(outcome.WinnerIDs) != 0 {
		return errors.New("non-completed replay cannot declare a winner")
	}
	return nil
}

func normalizeOutcome(outcome Outcome) Outcome {
	outcome.Status = strings.ToLower(strings.TrimSpace(outcome.Status))
	outcome.Reason = strings.TrimSpace(outcome.Reason)
	outcome.WinnerIDs = append([]string(nil), outcome.WinnerIDs...)
	outcome.LoserIDs = append([]string(nil), outcome.LoserIDs...)
	sort.Strings(outcome.WinnerIDs)
	sort.Strings(outcome.LoserIDs)
	return outcome
}
