package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminCRMDedicatedAuthenticationMFAAndPermissions(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	cfg.Settings.Admin.AccessCookieName = "sa_admin_access"
	cfg.Settings.Admin.RefreshCookieName = "sa_admin_refresh"
	cfg.Settings.Admin.AccessTTL = 10 * time.Minute
	cfg.Settings.Admin.RefreshTTL = 8 * time.Hour
	cfg.Settings.Admin.IdleTimeout = 30 * time.Minute
	handler := New(store, cfg).Handler

	playerHash, _ := bcrypt.GenerateFromPassword([]byte("StrongPassword!42"), bcrypt.DefaultCost)
	player := models.NewUser("", "player-crm-denied@example.com", string(playerHash))
	player.EmailVerified = true
	if err := store.CreateUser(context.Background(), player); err != nil {
		t.Fatal(err)
	}
	playerLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": player.Email, "password": "StrongPassword!42",
	}, nil)
	if playerLogin.Code != http.StatusOK {
		t.Fatalf("player login status=%d body=%s", playerLogin.Code, playerLogin.Body.String())
	}
	playerAccess := cookieByName(t, playerLogin, cfg.Settings.Security.AccessCookieName)
	legacyAdmin := authRequest(t, handler, http.MethodGet, "/api/v1/admin/users", nil, []*http.Cookie{playerAccess})
	if legacyAdmin.Code != http.StatusNotFound {
		t.Fatalf("legacy player-audience admin endpoint status=%d body=%s", legacyAdmin.Code, legacyAdmin.Body.String())
	}
	denied := authRequest(t, handler, http.MethodPost, "/api/v1/admin-crm/auth/login", map[string]string{
		"email": player.Email, "password": "StrongPassword!42",
	}, nil)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("player CRM login status=%d body=%s", denied.Code, denied.Body.String())
	}

	adminHash, _ := bcrypt.GenerateFromPassword([]byte("AdminPassword!42"), bcrypt.DefaultCost)
	admin := models.NewUser("", "support-admin@example.com", string(adminHash))
	admin.EmailVerified = true
	admin.Role = models.RoleSupport
	admin.DisplayName = "Support Operator"
	if err := store.CreateUser(context.Background(), admin); err != nil {
		t.Fatal(err)
	}
	login := authRequest(t, handler, http.MethodPost, "/api/v1/admin-crm/auth/login", map[string]string{
		"email": admin.Email, "password": "AdminPassword!42",
	}, nil)
	if login.Code != http.StatusOK || !strings.Contains(login.Body.String(), `"mfaEnrollmentRequired":true`) {
		t.Fatalf("administrator login status=%d body=%s", login.Code, login.Body.String())
	}
	adminAccess := cookieByName(t, login, cfg.Settings.Admin.AccessCookieName)
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == cfg.Settings.Security.AccessCookieName || cookie.Name == cfg.Settings.Security.RefreshCookieName {
			t.Fatalf("CRM login issued player cookie %s", cookie.Name)
		}
	}

	blocked := authRequest(t, handler, http.MethodGet, "/api/v1/admin-crm/dashboard", nil, []*http.Cookie{adminAccess})
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("pre-enrollment dashboard status=%d body=%s", blocked.Code, blocked.Body.String())
	}
	setup := authRequest(t, handler, http.MethodPost, "/api/v1/admin-crm/auth/mfa/setup", nil, []*http.Cookie{adminAccess})
	if setup.Code != http.StatusOK {
		t.Fatalf("MFA setup status=%d body=%s", setup.Code, setup.Body.String())
	}
	var setupBody map[string]string
	if err := json.Unmarshal(setup.Body.Bytes(), &setupBody); err != nil {
		t.Fatal(err)
	}
	confirmed := authRequest(t, handler, http.MethodPost, "/api/v1/admin-crm/auth/mfa/confirm", map[string]string{
		"code": totpCode(setupBody["secret"], time.Now()),
	}, []*http.Cookie{adminAccess})
	if confirmed.Code != http.StatusOK || !strings.Contains(confirmed.Body.String(), `"recoveryCodes"`) {
		t.Fatalf("MFA confirm status=%d body=%s", confirmed.Code, confirmed.Body.String())
	}

	session := authRequest(t, handler, http.MethodGet, "/api/v1/admin-crm/auth/session", nil, []*http.Cookie{adminAccess})
	if session.Code != http.StatusOK || !strings.Contains(session.Body.String(), `"support.read"`) {
		t.Fatalf("CRM session status=%d body=%s", session.Code, session.Body.String())
	}
	dashboard := authRequest(t, handler, http.MethodGet, "/api/v1/admin-crm/dashboard", nil, []*http.Cookie{adminAccess})
	if dashboard.Code != http.StatusOK {
		t.Fatalf("CRM dashboard status=%d body=%s", dashboard.Code, dashboard.Body.String())
	}
	finance := authRequest(t, handler, http.MethodGet, "/api/v1/admin-crm/finance", nil, []*http.Cookie{adminAccess})
	if finance.Code != http.StatusForbidden {
		t.Fatalf("support finance status=%d body=%s", finance.Code, finance.Body.String())
	}
}

func TestAdminPermissionMatrixDoesNotUseRoleRanking(t *testing.T) {
	if models.HasAdminPermission(models.RoleSupport, models.PermissionFinanceRead) {
		t.Fatal("support role inherited finance permission")
	}
	if !models.HasAdminPermission(models.RoleFinance, models.PermissionWithdrawalsReview) {
		t.Fatal("finance role lacks withdrawal permission")
	}
	if models.HasAdminPermission(models.RoleReadOnly, models.PermissionUsersManage) {
		t.Fatal("read-only role inherited a write permission")
	}
	if !models.HasAdminPermission(models.RoleSuperAdmin, models.PermissionAdminRolesManage) {
		t.Fatal("super administrator lacks role management")
	}
}
