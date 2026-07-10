package matchmaking

import (
	"testing"
	"time"

	"skill-arena/internal/models"
)

func TestServiceExpiresStaleMatchesAndFindsEligibleOpponent(t *testing.T) {
	now := time.Now().UTC()
	service := &Service{WaitingTimeout: time.Minute}
	matches := map[string]*models.PvPMatch{
		"expired": {
			ID:         "expired",
			PlayerAID:  "old-player",
			QueueType:  "standard",
			WalletType: "demo",
			Stake:      10,
			Status:     "waiting",
			CreatedAt:  now.Add(-2 * time.Minute),
		},
		"waiting": {
			ID:         "waiting",
			PlayerAID:  "player-a",
			QueueType:  "standard",
			WalletType: "demo",
			Stake:      10,
			Status:     "waiting",
			CreatedAt:  now,
		},
	}

	expired := service.ExpireStale(matches, now)
	if len(expired) != 1 || matches["expired"].Status != "expired" {
		t.Fatalf("expired = %#v status=%q, want expired match", expired, matches["expired"].Status)
	}

	request := JoinRequest{UserID: "player-b", QueueType: "standard", WalletType: "demo", Stake: 10, Now: now}
	match := service.FindWaitingMatch(matches, request)
	if match == nil || match.ID != "waiting" {
		t.Fatalf("match = %#v, want waiting", match)
	}
	if !service.Activate(match, "player-b", now) || match.Status != "active" || match.PlayerBID != "player-b" {
		t.Fatalf("activated match = %#v, want active with player-b", match)
	}
}

func TestServiceRejectsSelfActivation(t *testing.T) {
	now := time.Now().UTC()
	service := NewService()
	match := &models.PvPMatch{
		ID:         "self",
		PlayerAID:  "player-a",
		QueueType:  "standard",
		WalletType: "demo",
		Stake:      10,
		Status:     "waiting",
		CreatedAt:  now,
	}

	if service.Activate(match, "player-a", now) {
		t.Fatal("self activation succeeded")
	}
}
