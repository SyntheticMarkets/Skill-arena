package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"
)

func authTestConfig(superAdminEmail string) *config.Config {
	settings := config.LoadRuntimeSettings()
	settings.Admin.SuperAdminEmails = nil
	if superAdminEmail != "" {
		settings.Admin.SuperAdminEmails = []string{superAdminEmail}
	}
	settings.Security.AccessCookieName = "sa_access"
	settings.Security.RefreshCookieName = "sa_refresh"
	settings.Security.AccessTTL = 15 * time.Minute
	settings.Security.RefreshTTL = 24 * time.Hour
	settings.Email.BaseURL = "https://app.skillarena.test"
	settings.CORS.AllowedOrigins = []string{"https://app.skillarena.test"}
	return &config.Config{HTTPAddr: ":0", JWTSecret: strings.Repeat("x", 48), Settings: settings}
}

func authRequest(t *testing.T, handler http.Handler, method, path string, body any, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://app.skillarena.test")
	req.Header.Set("X-Device-Fingerprint", "test-device")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func emailTokenFromLatestJob(t *testing.T, store *db.Store) string {
	t.Helper()
	jobs, err := store.ListJobs(context.Background(), models.JobStatusQueued)
	if err != nil || len(jobs) == 0 {
		t.Fatalf("list email jobs: count=%d err=%v", len(jobs), err)
	}
	link, err := url.Parse(jobs[len(jobs)-1].Payload["link"])
	if err != nil {
		t.Fatal(err)
	}
	if token := link.Query().Get("token"); token != "" {
		return token
	}
	t.Fatal("email job did not contain a tokenized link")
	return ""
}

func cookieByName(t *testing.T, res *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("response did not set cookie %s", name)
	return nil
}

func totpCode(secret string, now time.Time) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(now.Unix()/30))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	return fmt.Sprintf("%06d", int(value%1000000))
}

func TestAuthenticationLifecycleAndSessionRevocation(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler
	email := "player@example.com"
	password := "StrongPassword!42"

	registered := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": email, "password": password, "country": "ZA", "dateOfBirth": "1990-01-01", "acceptTerms": true, "acceptFairPlay": true}, nil)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registered.Code, registered.Body.String())
	}
	verificationToken := emailTokenFromLatestJob(t, store)
	tamperedToken := verificationToken[:len(verificationToken)-1] + "A"
	if tamperedToken == verificationToken {
		tamperedToken = verificationToken[:len(verificationToken)-1] + "B"
	}
	tampered := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": tamperedToken}, nil)
	if tampered.Code != http.StatusBadRequest {
		t.Fatalf("tampered verification status=%d body=%s", tampered.Code, tampered.Body.String())
	}
	verified := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": verificationToken}, nil)
	if verified.Code != http.StatusNoContent {
		t.Fatalf("verify status=%d body=%s", verified.Code, verified.Body.String())
	}
	reused := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": verificationToken}, nil)
	if reused.Code != http.StatusNoContent {
		t.Fatalf("verification reuse status=%d body=%s", reused.Code, reused.Body.String())
	}

	login := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	access := cookieByName(t, login, cfg.Settings.Security.AccessCookieName)
	refresh := cookieByName(t, login, cfg.Settings.Security.RefreshCookieName)
	if !access.HttpOnly || !refresh.HttpOnly || access.SameSite != http.SameSiteStrictMode {
		t.Fatal("session cookies are not hardened")
	}
	session := authRequest(t, handler, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{access})
	if session.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", session.Code, session.Body.String())
	}
	rotated := authRequest(t, handler, http.MethodPost, "/api/v1/auth/refresh-token", nil, []*http.Cookie{refresh})
	if rotated.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", rotated.Code, rotated.Body.String())
	}
	replayRefresh := authRequest(t, handler, http.MethodPost, "/api/v1/auth/refresh-token", nil, []*http.Cookie{refresh})
	if replayRefresh.Code != http.StatusUnauthorized {
		t.Fatalf("refresh replay status=%d body=%s", replayRefresh.Code, replayRefresh.Body.String())
	}

	resetRequested := authRequest(t, handler, http.MethodPost, "/api/v1/auth/password-reset/request", map[string]string{"email": email}, nil)
	if resetRequested.Code != http.StatusAccepted {
		t.Fatalf("reset request status=%d body=%s", resetRequested.Code, resetRequested.Body.String())
	}
	resetToken := emailTokenFromLatestJob(t, store)
	newPassword := "AnotherStrong!Password43"
	reset := authRequest(t, handler, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]string{"token": resetToken, "password": newPassword, "confirmPassword": newPassword}, nil)
	if reset.Code != http.StatusNoContent {
		t.Fatalf("reset status=%d body=%s", reset.Code, reset.Body.String())
	}
	oldSession := authRequest(t, handler, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{cookieByName(t, rotated, cfg.Settings.Security.AccessCookieName)})
	if oldSession.Code != http.StatusUnauthorized {
		t.Fatalf("old session after reset status=%d body=%s", oldSession.Code, oldSession.Body.String())
	}
	newLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": newPassword}, nil)
	if newLogin.Code != http.StatusOK {
		t.Fatalf("new-password login status=%d body=%s", newLogin.Code, newLogin.Body.String())
	}
}

func TestPrivilegedAccountMustEnrollMFA(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	email := "admin@example.com"
	cfg := authTestConfig(email)
	handler := New(store, cfg).Handler
	password := "PrivilegedPassword!42"
	registered := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": email, "password": password, "country": "ZA", "dateOfBirth": "1988-02-01", "acceptTerms": true, "acceptFairPlay": true}, nil)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registered.Code, registered.Body.String())
	}
	token := emailTokenFromLatestJob(t, store)
	if got := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": token}, nil); got.Code != http.StatusNoContent {
		t.Fatalf("verify status=%d body=%s", got.Code, got.Body.String())
	}
	login := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	if login.Code != http.StatusOK || !strings.Contains(login.Body.String(), `"mfaEnrollmentRequired":true`) {
		t.Fatalf("privileged login status=%d body=%s", login.Code, login.Body.String())
	}
	access := cookieByName(t, login, cfg.Settings.Security.AccessCookieName)
	blocked := authRequest(t, handler, http.MethodGet, "/api/v1/profile", nil, []*http.Cookie{access})
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("pre-MFA profile status=%d body=%s", blocked.Code, blocked.Body.String())
	}
	setup := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/setup", nil, []*http.Cookie{access})
	if setup.Code != http.StatusOK {
		t.Fatalf("MFA setup status=%d body=%s", setup.Code, setup.Body.String())
	}
	var setupBody map[string]string
	if err := json.Unmarshal(setup.Body.Bytes(), &setupBody); err != nil {
		t.Fatal(err)
	}
	confirm := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/confirm", map[string]string{"code": totpCode(setupBody["secret"], time.Now())}, []*http.Cookie{access})
	if confirm.Code != http.StatusOK || !strings.Contains(confirm.Body.String(), "recoveryCodes") {
		t.Fatalf("MFA confirm status=%d body=%s", confirm.Code, confirm.Body.String())
	}
	if got := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/disable", map[string]string{"password": password, "code": totpCode(setupBody["secret"], time.Now())}, []*http.Cookie{access}); got.Code != http.StatusForbidden {
		t.Fatalf("privileged MFA disable status=%d body=%s", got.Code, got.Body.String())
	}
}
