package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

func HubHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		userID := UserIDFromContext(r.Context())
		snapshot, err := store.BuildHubSnapshot(r.Context(), userID)
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	}
}

func GameCatalogHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		games, err := store.ListGameCatalog(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"games": games})
	}
}

func GameRulesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		gameID := strings.TrimPrefix(r.URL.Path, "/api/v1/catalog/games/")
		if gameID == "" || strings.Contains(gameID, "/") {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "game id is required")
			return
		}
		games, err := store.ListGameCatalog(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		for _, gameEntry := range games {
			if gameEntry.ID == gameID {
				writeJSON(w, http.StatusOK, gameEntry)
				return
			}
		}
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "game was not found")
	}
}

func NotificationsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
		if status != "" && status != models.NotificationStatusUnread &&
			status != models.NotificationStatusRead && status != models.NotificationStatusArchived {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "notification status is invalid")
			return
		}
		notifications, err := store.ListNotifications(r.Context(), UserIDFromContext(r.Context()), status)
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"notifications": notifications})
	}
}

type notificationStatusRequest struct {
	NotificationID string `json:"notificationId"`
}

func NotificationStatusHandler(store *db.Store, status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		var request notificationStatusRequest
		if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.NotificationID) == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "notificationId is required")
			return
		}
		if err := store.UpdateNotificationStatus(
			r.Context(), UserIDFromContext(r.Context()), request.NotificationID, status,
		); err != nil {
			WriteMappedError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func SupportContentHandler(contactEmail string) http.HandlerFunc {
	content := models.SupportContent{
		ContactEmail: contactEmail,
		Articles: []models.SupportArticle{
			{
				ID: "fair-play", Category: "rules", Title: "Fair Play",
				Body: "Skill Arena validates game actions on the server and retains replay evidence for competitive review.",
			},
			{
				ID: "responsible-gaming", Category: "responsible_gaming", Title: "Responsible Gaming",
				Body: "Choose limits before live competition, never chase a loss, and use Practice when competition stops feeling constructive.",
			},
			{
				ID: "account-security", Category: "faq", Title: "How do I protect my account?",
				Body: "Verify your email, enable MFA, save recovery codes securely, and revoke any session or device you do not recognize.",
			},
			{
				ID: "withdrawal-review", Category: "faq", Title: "Why is a withdrawal reviewed?",
				Body: "Withdrawals move through verification, risk, and treasury checks before provider settlement. The player can follow each status but cannot approve it.",
			},
		},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		writeJSON(w, http.StatusOK, content)
	}
}

type supportTicketRequest struct {
	Category string `json:"category"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
}

var supportTicketCategories = map[string]struct{}{
	"account": {}, "security": {}, "gameplay": {}, "wallet": {}, "responsible_gaming": {},
}

func SupportTicketsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		switch r.Method {
		case http.MethodGet:
			tickets, err := store.ListSupportTickets(r.Context(), userID)
			if err != nil {
				WriteMappedError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"tickets": tickets})
		case http.MethodPost:
			var request supportTicketRequest
			if err := decodeJSON(r, &request); err != nil {
				WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "request body is invalid")
				return
			}
			request.Category = strings.ToLower(strings.TrimSpace(request.Category))
			request.Subject = strings.TrimSpace(request.Subject)
			request.Message = strings.TrimSpace(request.Message)
			_, validCategory := supportTicketCategories[request.Category]
			if !validCategory || len(request.Subject) < 4 || len(request.Subject) > 120 ||
				len(request.Message) < 10 || len(request.Message) > 4000 {
				WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "a supported category, subject, and a message between 10 and 4000 characters are required")
				return
			}
			ticket := &models.SupportTicket{
				UserID: userID, Category: request.Category,
				Subject: request.Subject, Message: request.Message,
			}
			if err := store.CreateSupportTicket(r.Context(), ticket); err != nil {
				WriteMappedError(w, http.StatusInternalServerError, err)
				return
			}
			notification := &models.Notification{
				UserID: ticket.UserID, Category: "support", Title: "Support ticket received",
				Message:   "Your support request is recorded and available in your support history.",
				ActionURL: "/support", Metadata: map[string]string{"ticketId": ticket.ID},
			}
			if err := store.CreateNotification(r.Context(), notification); err != nil {
				WriteMappedError(w, http.StatusInternalServerError, err)
				return
			}
			_ = store.AppendAuditLog(r.Context(), userID, "support.ticket.created", ticket.ID, clientIP(r), map[string]string{"category": request.Category})
			writeJSON(w, http.StatusCreated, ticket)
		default:
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		}
	}
}

func validAvatarKey(value string) bool {
	switch value {
	case "", "vanguard", "strategist", "pathfinder", "accelerator":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}
