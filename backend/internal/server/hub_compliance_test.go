package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

func TestSprint2HubProfileNotificationAndSupportContracts(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler
	access, _ := registerVerifyLogin(
		t, handler, store,
		[2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName},
		"hub-player@example.com", "StrongPassword!42",
	)
	user, err := store.GetUserByEmail(context.Background(), "hub-player@example.com")
	if err != nil {
		t.Fatal(err)
	}

	catalog := authRequest(t, handler, http.MethodGet, "/api/v1/catalog/games", nil, nil)
	if catalog.Code != http.StatusOK || !strings.Contains(catalog.Body.String(), `"id":"maze_arena"`) ||
		strings.Contains(catalog.Body.String(), "test_arena") {
		t.Fatalf("catalog status=%d body=%s", catalog.Code, catalog.Body.String())
	}
	rules := authRequest(t, handler, http.MethodGet, "/api/v1/catalog/games/maze_arena", nil, nil)
	if rules.Code != http.StatusOK || !strings.Contains(rules.Body.String(), "server validates every action") {
		t.Fatalf("rules status=%d body=%s", rules.Code, rules.Body.String())
	}

	hub := authRequest(t, handler, http.MethodGet, "/api/v1/hub", nil, []*http.Cookie{access})
	if hub.Code != http.StatusOK {
		t.Fatalf("hub status=%d body=%s", hub.Code, hub.Body.String())
	}
	var snapshot models.HubSnapshot
	if err := json.Unmarshal(hub.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Profile.UserID != user.ID || snapshot.Wallet.AvailableBalance != 0 ||
		len(snapshot.Games) != 1 || len(snapshot.Objectives) != 3 ||
		snapshot.RecommendedAction.ActionURL == "" || len(snapshot.Tournaments) != 0 {
		t.Fatalf("unexpected hub snapshot: %+v", snapshot)
	}

	updated := authRequest(t, handler, http.MethodPost, "/api/v1/profile", map[string]string{
		"username": "hub_player", "displayName": "Hub Player", "country": "ZA",
		"language": "en", "avatarUrl": "strategist",
	}, []*http.Cookie{access})
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"displayName":"Hub Player"`) {
		t.Fatalf("profile update status=%d body=%s", updated.Code, updated.Body.String())
	}
	invalidAvatar := authRequest(t, handler, http.MethodPost, "/api/v1/profile", map[string]string{
		"username": "hub_player", "displayName": "Hub Player", "country": "ZA",
		"language": "en", "avatarUrl": "https://unapproved.example/avatar.png",
	}, []*http.Cookie{access})
	if invalidAvatar.Code != http.StatusBadRequest {
		t.Fatalf("invalid avatar status=%d body=%s", invalidAvatar.Code, invalidAvatar.Body.String())
	}
	invalidIdentity := authRequest(t, handler, http.MethodPost, "/api/v1/profile", map[string]string{
		"username": "not allowed!", "displayName": "H", "country": "Z1",
		"language": "invalid", "avatarUrl": "",
	}, []*http.Cookie{access})
	if invalidIdentity.Code != http.StatusBadRequest {
		t.Fatalf("invalid identity status=%d body=%s", invalidIdentity.Code, invalidIdentity.Body.String())
	}

	notification := &models.Notification{
		UserID: user.ID, Category: "progression", Title: "Profile complete",
		Message: "Your competitor identity is ready.", ActionURL: "/profile",
	}
	if err := store.CreateNotification(context.Background(), notification); err != nil {
		t.Fatal(err)
	}
	listed := authRequest(t, handler, http.MethodGet, "/api/v1/notifications?status=unread", nil, []*http.Cookie{access})
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), notification.ID) {
		t.Fatalf("notification list status=%d body=%s", listed.Code, listed.Body.String())
	}
	read := authRequest(t, handler, http.MethodPost, "/api/v1/notifications/read", map[string]string{
		"notificationId": notification.ID,
	}, []*http.Cookie{access})
	if read.Code != http.StatusNoContent {
		t.Fatalf("notification read status=%d body=%s", read.Code, read.Body.String())
	}
	archived := authRequest(t, handler, http.MethodPost, "/api/v1/notifications/archive", map[string]string{
		"notificationId": notification.ID,
	}, []*http.Cookie{access})
	if archived.Code != http.StatusNoContent {
		t.Fatalf("notification archive status=%d body=%s", archived.Code, archived.Body.String())
	}

	content := authRequest(t, handler, http.MethodGet, "/api/v1/support/content", nil, nil)
	if content.Code != http.StatusOK || !strings.Contains(content.Body.String(), "Responsible Gaming") {
		t.Fatalf("support content status=%d body=%s", content.Code, content.Body.String())
	}
	createdTicket := authRequest(t, handler, http.MethodPost, "/api/v1/support/tickets", map[string]string{
		"category": "account", "subject": "Session question",
		"message": "Please explain the session shown on my account.",
	}, []*http.Cookie{access})
	if createdTicket.Code != http.StatusCreated || !strings.Contains(createdTicket.Body.String(), `"status":"received"`) {
		t.Fatalf("ticket create status=%d body=%s", createdTicket.Code, createdTicket.Body.String())
	}
	tickets := authRequest(t, handler, http.MethodGet, "/api/v1/support/tickets", nil, []*http.Cookie{access})
	if tickets.Code != http.StatusOK || !strings.Contains(tickets.Body.String(), "Session question") {
		t.Fatalf("ticket list status=%d body=%s", tickets.Code, tickets.Body.String())
	}
	supportNotification := authRequest(t, handler, http.MethodGet, "/api/v1/notifications?status=unread", nil, []*http.Cookie{access})
	if supportNotification.Code != http.StatusOK || !strings.Contains(supportNotification.Body.String(), "Support ticket received") {
		t.Fatalf("support notification status=%d body=%s", supportNotification.Code, supportNotification.Body.String())
	}
}

func TestSprint2HubAuthorizationAndOwnershipContracts(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler
	access, _ := registerVerifyLogin(
		t, handler, store,
		[2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName},
		"hub-owner@example.com", "StrongPassword!42",
	)
	other := models.NewUser("", "other-owner@example.com", "hash")
	other.EmailVerified = true
	if err := store.CreateUser(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	otherNotification := &models.Notification{
		UserID: other.ID, Category: "security", Title: "Private",
		Message: "This notification belongs to another player.",
	}
	if err := store.CreateNotification(context.Background(), otherNotification); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/api/v1/hub", "/api/v1/profile", "/api/v1/notifications", "/api/v1/support/tickets",
	} {
		response := authRequest(t, handler, http.MethodGet, path, nil, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s without session status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	crossAccount := authRequest(t, handler, http.MethodPost, "/api/v1/notifications/read", map[string]string{
		"notificationId": otherNotification.ID,
	}, []*http.Cookie{access})
	if crossAccount.Code != http.StatusNotFound {
		t.Fatalf("cross-account notification status=%d body=%s", crossAccount.Code, crossAccount.Body.String())
	}
	invalidTicket := authRequest(t, handler, http.MethodPost, "/api/v1/support/tickets", map[string]string{
		"category": "account", "subject": "x", "message": "short",
	}, []*http.Cookie{access})
	if invalidTicket.Code != http.StatusBadRequest {
		t.Fatalf("invalid ticket status=%d body=%s", invalidTicket.Code, invalidTicket.Body.String())
	}
	unsupportedTicket := authRequest(t, handler, http.MethodPost, "/api/v1/support/tickets", map[string]string{
		"category": "admin", "subject": "Unsupported category",
		"message": "This category must not be accepted by the player API.",
	}, []*http.Cookie{access})
	if unsupportedTicket.Code != http.StatusBadRequest {
		t.Fatalf("unsupported ticket status=%d body=%s", unsupportedTicket.Code, unsupportedTicket.Body.String())
	}
}
