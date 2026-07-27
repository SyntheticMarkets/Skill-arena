package realtime

import (
	"errors"

	"skill-arena/internal/models"
)

var ErrInvalidTransition = errors.New("invalid match lifecycle transition")

var transitions = map[string]map[string]bool{
	models.MatchCreated:      {models.MatchWaiting: true, models.MatchReady: true, models.MatchCancelled: true},
	models.MatchWaiting:      {models.MatchReady: true, models.MatchCancelled: true, models.MatchAbandoned: true},
	models.MatchReady:        {models.MatchStarting: true, models.MatchCancelled: true},
	models.MatchStarting:     {models.MatchLive: true, models.MatchCancelled: true},
	models.MatchLive:         {models.MatchPaused: true, models.MatchReconnecting: true, models.MatchCompleted: true, models.MatchAbandoned: true},
	models.MatchPaused:       {models.MatchLive: true, models.MatchReconnecting: true, models.MatchCompleted: true, models.MatchAbandoned: true},
	models.MatchReconnecting: {models.MatchLive: true, models.MatchCompleted: true, models.MatchAbandoned: true},
}

func CanTransition(from, to string) bool {
	return transitions[from][to]
}

func ValidateTransition(from, to string) error {
	if !CanTransition(from, to) {
		return ErrInvalidTransition
	}
	return nil
}

func terminal(status string) bool {
	return status == models.MatchCompleted || status == models.MatchCancelled || status == models.MatchAbandoned
}
