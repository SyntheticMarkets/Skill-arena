package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

type startGameRequest struct {
	GameType string  `json:"gameType"`
	Stake    float64 `json:"stake"`
}

type startGameResponse struct {
	SessionID         string                    `json:"sessionId"`
	GameType          string                    `json:"gameType"`
	Stake             float64                   `json:"stake"`
	DifficultyRating  int                       `json:"difficultyRating,omitempty"`
	DifficultyProfile *models.DifficultyProfile `json:"difficultyProfile,omitempty"`
	GenerationHash    string                    `json:"generationHash,omitempty"`
	PuzzleVersion     models.PuzzleVersion      `json:"puzzleVersion,omitempty"`
	State             string                    `json:"state,omitempty"`
	Maze              []string                  `json:"maze"`
	Width             int                       `json:"width"`
	Height            int                       `json:"height"`
	StartX            int                       `json:"startX"`
	StartY            int                       `json:"startY"`
	EndX              int                       `json:"endX"`
	EndY              int                       `json:"endY"`
	Lines             []models.ArrowLine        `json:"lines,omitempty"`
}

type finishGameRequest struct {
	SessionID      string                    `json:"sessionId"`
	Moves          []string                  `json:"moves"`
	ClickedLineIDs []string                  `json:"clickedLineIds"`
	Telemetry      *models.GameplayTelemetry `json:"telemetry,omitempty"`
}

type finishGameResponse struct {
	Session *models.GameSession `json:"session"`
}

func StartGameHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req startGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.GameType == "" || req.Stake <= 0 {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "gameType and stake are required")
			return
		}
		if !config.Runtime().FeatureEnabled("maze_arena") {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "Maze Arena is disabled")
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		session := &models.GameSession{
			UserID:   userID,
			GameType: req.GameType,
			Stake:    req.Stake,
		}

		if err := store.StartGameSession(r.Context(), session); err != nil {
			http.Error(w, fmt.Sprintf("cannot start game: %v", err), http.StatusBadRequest)
			return
		}
		_ = store.AppendAuditLog(r.Context(), userID, "game.session.started", session.ID, clientIP(r), map[string]string{"gameType": session.GameType})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(startGameResponse{
			SessionID:         session.ID,
			GameType:          session.GameType,
			Stake:             session.Stake,
			DifficultyRating:  session.DifficultyRating,
			DifficultyProfile: session.DifficultyProfile,
			GenerationHash:    session.GenerationHash,
			PuzzleVersion:     session.PuzzleVersion,
			State:             session.State,
			Maze:              session.MazeCells,
			Width:             session.Width,
			Height:            session.Height,
			StartX:            session.StartX,
			StartY:            session.StartY,
			EndX:              session.EndX,
			EndY:              session.EndY,
			Lines:             session.Lines,
		})
	}
}

func FinishGameHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req finishGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		lineIDs := req.ClickedLineIDs
		if len(lineIDs) == 0 {
			lineIDs = req.Moves
		}
		if req.SessionID == "" || len(lineIDs) == 0 {
			http.Error(w, "sessionId and clickedLineIds are required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		moveHistory := make([]models.MazeMove, 0, len(lineIDs))
		for _, direction := range lineIDs {
			moveHistory = append(moveHistory, models.MazeMove{
				Direction: direction,
				Timestamp: time.Now().UTC(),
			})
		}

		session, err := store.SubmitMazeMoves(r.Context(), userID, req.SessionID, moveHistory)
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot complete game: %v", err), http.StatusBadRequest)
			return
		}
		if req.Telemetry != nil {
			req.Telemetry.UserID = userID
			req.Telemetry.Scope = "game_session"
			req.Telemetry.ScopeID = session.ID
			req.Telemetry.UserAgent = r.UserAgent()
			_ = store.RecordGameplayTelemetry(r.Context(), req.Telemetry)
		}
		action := "game.session.finished"
		if session.Calibration {
			action = "calibration.finished"
		}
		_ = store.AppendAuditLog(r.Context(), userID, action, session.ID, clientIP(r), map[string]string{"outcome": session.Outcome})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finishGameResponse{Session: session})
	}
}

func GetGameSessionHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/games/")
		if sessionID == "" || sessionID == "history" {
			http.Error(w, "session id required", http.StatusBadRequest)
			return
		}

		session, err := store.GetSessionByID(r.Context(), sessionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("game session not found: %v", err), http.StatusNotFound)
			return
		}
		if session.UserID != userID {
			http.Error(w, "session does not belong to user", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
	}
}

func GameHistoryHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessions, err := store.GetSessionsByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load game history: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}
}
