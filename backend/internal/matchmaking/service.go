package matchmaking

import (
	"time"

	"skill-arena/internal/models"
)

type JoinRequest struct {
	UserID     string
	QueueType  string
	WalletType string
	Stake      float64
	Now        time.Time
}

type Service struct {
	WaitingTimeout time.Duration
}

func NewService() *Service {
	return &Service{WaitingTimeout: 10 * time.Minute}
}

func (s *Service) ExpireStale(matches map[string]*models.PvPMatch, now time.Time) []string {
	if s.WaitingTimeout <= 0 {
		s.WaitingTimeout = 10 * time.Minute
	}
	expired := []string{}
	for _, match := range matches {
		if match.Status == "waiting" && now.Sub(match.CreatedAt) > s.WaitingTimeout {
			match.Status = "expired"
			expired = append(expired, match.ID)
		}
	}
	return expired
}

func (s *Service) ActiveOrWaitingForUser(matches map[string]*models.PvPMatch, userID string) *models.PvPMatch {
	for _, match := range matches {
		if match.PlayerAID != "" && match.PlayerAID == match.PlayerBID {
			match.Status = "aborted"
			continue
		}
		if (match.PlayerAID == userID || match.PlayerBID == userID) && (match.Status == "waiting" || match.Status == "active") {
			return match
		}
	}
	return nil
}

func (s *Service) FindWaitingMatch(matches map[string]*models.PvPMatch, request JoinRequest) *models.PvPMatch {
	for _, match := range matches {
		if match.PlayerAID != "" && match.PlayerAID == match.PlayerBID {
			match.Status = "aborted"
			continue
		}
		if match.Status == "waiting" &&
			match.PlayerAID != request.UserID &&
			match.QueueType == request.QueueType &&
			match.WalletType == request.WalletType &&
			match.Stake == request.Stake {
			return match
		}
	}
	return nil
}

func (s *Service) Activate(match *models.PvPMatch, userID string, now time.Time) bool {
	if match == nil || match.Status != "waiting" || match.PlayerAID == userID {
		return false
	}
	match.PlayerBID = userID
	if match.PlayerAID == match.PlayerBID {
		match.PlayerBID = ""
		match.Status = "aborted"
		return false
	}
	match.Status = "active"
	match.StartedAt = &now
	return true
}
