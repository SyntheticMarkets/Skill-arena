package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func expiredSignedToken(secret, purpose string) string {
	nonce := "expired-validation-token"
	expires := strconv.FormatInt(time.Now().UTC().Add(-time.Minute).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(purpose + "\x00" + nonce + "\x00" + expires))
	return nonce + "." + expires + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func registerVerifyLogin(t *testing.T, handler http.Handler, store *db.Store, cfgCookieNames [2]string, email, password string) (*http.Cookie, *http.Cookie) {
	t.Helper()
	registered := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": email, "password": password, "country": "ZA", "dateOfBirth": "1990-01-01", "acceptTerms": true, "acceptFairPlay": true}, nil)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registered.Code, registered.Body.String())
	}
	token := emailTokenFromLatestJob(t, store)
	if verified := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": token}, nil); verified.Code != http.StatusNoContent {
		t.Fatalf("verify status=%d body=%s", verified.Code, verified.Body.String())
	}
	login := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	return cookieByName(t, login, cfgCookieNames[0]), cookieByName(t, login, cfgCookieNames[1])
}

func TestSprint1SessionDeviceLogoutAndCSRFContracts(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	cfg.Settings.Security.CookieSecure = true
	handler := New(store, cfg).Handler
	password := "StrongPassword!42"
	accessOne, refreshOne := registerVerifyLogin(t, handler, store, [2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName}, "sessions@example.com", password)

	user, err := store.GetUserByEmail(context.Background(), "sessions@example.com")
	if err != nil || user.PasswordHash == password || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		t.Fatal("password was not stored as a bcrypt hash")
	}
	if !accessOne.Secure || !refreshOne.Secure || !accessOne.HttpOnly || !refreshOne.HttpOnly || accessOne.SameSite != http.SameSiteStrictMode || refreshOne.SameSite != http.SameSiteStrictMode {
		t.Fatal("production cookie flags are incomplete")
	}

	csrfBody := bytes.NewReader([]byte(`{}`))
	csrfRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", csrfBody)
	csrfRequest.Header.Set("Content-Type", "application/json")
	csrfRequest.AddCookie(accessOne)
	csrfRequest.AddCookie(refreshOne)
	csrfResponse := httptest.NewRecorder()
	handler.ServeHTTP(csrfResponse, csrfRequest)
	if csrfResponse.Code != http.StatusForbidden || !strings.Contains(csrfResponse.Body.String(), `"code":"FORBIDDEN"`) {
		t.Fatalf("CSRF request status=%d body=%s", csrfResponse.Code, csrfResponse.Body.String())
	}

	sessions := authRequest(t, handler, http.MethodGet, "/api/v1/auth/sessions", nil, []*http.Cookie{accessOne})
	if sessions.Code != http.StatusOK || !strings.Contains(sessions.Body.String(), `"current":true`) {
		t.Fatalf("sessions status=%d body=%s", sessions.Code, sessions.Body.String())
	}
	devices := authRequest(t, handler, http.MethodGet, "/api/v1/auth/devices", nil, []*http.Cookie{accessOne})
	if devices.Code != http.StatusOK || !strings.Contains(devices.Body.String(), "test-device") {
		t.Fatalf("devices status=%d body=%s", devices.Code, devices.Body.String())
	}
	registeredDevice := authRequest(t, handler, http.MethodPost, "/api/v1/devices/fingerprint", map[string]string{"fingerprint": "manual-device", "deviceName": "Validation browser", "os": "Windows", "browser": "Chromium"}, []*http.Cookie{accessOne})
	if registeredDevice.Code != http.StatusOK || !strings.Contains(registeredDevice.Body.String(), "manual-device") {
		t.Fatalf("device registration status=%d body=%s", registeredDevice.Code, registeredDevice.Body.String())
	}

	secondLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "sessions@example.com", "password": password}, nil)
	accessTwo := cookieByName(t, secondLogin, cfg.Settings.Security.AccessCookieName)
	var sessionList struct {
		Sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"sessions"`
	}
	listed := authRequest(t, handler, http.MethodGet, "/api/v1/auth/sessions", nil, []*http.Cookie{accessTwo})
	if err := json.Unmarshal(listed.Body.Bytes(), &sessionList); err != nil {
		t.Fatal(err)
	}
	oldSessionID := ""
	for _, session := range sessionList.Sessions {
		if !session.Current {
			oldSessionID = session.ID
			break
		}
	}
	if oldSessionID == "" {
		t.Fatal("second login did not expose a revocable prior session")
	}
	revoked := authRequest(t, handler, http.MethodPost, "/api/v1/auth/sessions/revoke", map[string]string{"sessionId": oldSessionID}, []*http.Cookie{accessTwo})
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("session revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	if old := authRequest(t, handler, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{accessOne}); old.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status=%d body=%s", old.Code, old.Body.String())
	}

	var deviceList struct {
		Devices []struct {
			ID string `json:"id"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(devices.Body.Bytes(), &deviceList); err != nil || len(deviceList.Devices) == 0 {
		t.Fatalf("decode devices: count=%d err=%v", len(deviceList.Devices), err)
	}
	deviceRevoked := authRequest(t, handler, http.MethodPost, "/api/v1/auth/devices/revoke", map[string]string{"deviceId": deviceList.Devices[0].ID}, []*http.Cookie{accessTwo})
	if deviceRevoked.Code != http.StatusNoContent {
		t.Fatalf("device revoke status=%d body=%s", deviceRevoked.Code, deviceRevoked.Body.String())
	}
	if current := authRequest(t, handler, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{accessTwo}); current.Code != http.StatusUnauthorized {
		t.Fatalf("device session remained active status=%d body=%s", current.Code, current.Body.String())
	}

	thirdLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "sessions@example.com", "password": password}, nil)
	accessThree := cookieByName(t, thirdLogin, cfg.Settings.Security.AccessCookieName)
	refreshThree := cookieByName(t, thirdLogin, cfg.Settings.Security.RefreshCookieName)
	loggedOut := authRequest(t, handler, http.MethodPost, "/api/v1/auth/logout", nil, []*http.Cookie{accessThree, refreshThree})
	if loggedOut.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", loggedOut.Code, loggedOut.Body.String())
	}
	if after := authRequest(t, handler, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{accessThree}); after.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out session status=%d body=%s", after.Code, after.Body.String())
	}
}

func TestSprint1MFAChallengeAndRecoveryCodeContracts(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	email := "mfa-contract@example.com"
	password := "PrivilegedPassword!42"
	cfg := authTestConfig(email)
	handler := New(store, cfg).Handler
	registered := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": email, "password": password, "country": "ZA", "dateOfBirth": "1988-02-01", "acceptTerms": true, "acceptFairPlay": true}, nil)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registered.Code, registered.Body.String())
	}
	if verified := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": emailTokenFromLatestJob(t, store)}, nil); verified.Code != http.StatusNoContent {
		t.Fatalf("verify status=%d body=%s", verified.Code, verified.Body.String())
	}
	login := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	access := cookieByName(t, login, cfg.Settings.Security.AccessCookieName)
	setup := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/setup", nil, []*http.Cookie{access})
	var setupBody map[string]string
	if setup.Code != http.StatusOK || json.Unmarshal(setup.Body.Bytes(), &setupBody) != nil {
		t.Fatalf("setup status=%d body=%s", setup.Code, setup.Body.String())
	}
	confirmed := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/confirm", map[string]string{"code": totpCode(setupBody["secret"], time.Now())}, []*http.Cookie{access})
	var recovery struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if confirmed.Code != http.StatusOK || json.Unmarshal(confirmed.Body.Bytes(), &recovery) != nil || len(recovery.RecoveryCodes) != 10 {
		t.Fatalf("confirm status=%d body=%s", confirmed.Code, confirmed.Body.String())
	}

	challengeLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	var challenge map[string]any
	if challengeLogin.Code != http.StatusAccepted || json.Unmarshal(challengeLogin.Body.Bytes(), &challenge) != nil {
		t.Fatalf("challenge login status=%d body=%s", challengeLogin.Code, challengeLogin.Body.String())
	}
	challengeToken, _ := challenge["challengeToken"].(string)
	invalid := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/challenge", map[string]string{"challengeToken": challengeToken, "code": "000000"}, nil)
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid MFA status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	valid := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/challenge", map[string]string{"challengeToken": challengeToken, "code": totpCode(setupBody["secret"], time.Now())}, nil)
	if valid.Code != http.StatusOK {
		t.Fatalf("valid MFA status=%d body=%s", valid.Code, valid.Body.String())
	}

	recoveryLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	challenge = map[string]any{}
	_ = json.Unmarshal(recoveryLogin.Body.Bytes(), &challenge)
	recoveryToken, _ := challenge["challengeToken"].(string)
	recovered := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/challenge", map[string]string{"challengeToken": recoveryToken, "recoveryCode": recovery.RecoveryCodes[0]}, nil)
	if recovered.Code != http.StatusOK {
		t.Fatalf("recovery MFA status=%d body=%s", recovered.Code, recovered.Body.String())
	}
	reuseLogin := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": password}, nil)
	challenge = map[string]any{}
	_ = json.Unmarshal(reuseLogin.Body.Bytes(), &challenge)
	reused := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/challenge", map[string]string{"challengeToken": fmt.Sprint(challenge["challengeToken"]), "recoveryCode": recovery.RecoveryCodes[0]}, nil)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("reused recovery code status=%d body=%s", reused.Code, reused.Body.String())
	}
}

func TestSprint1PlayerCanEnableAndDisableMFA(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler
	password := "PlayerPassword!42"
	access, _ := registerVerifyLogin(t, handler, store, [2]string{cfg.Settings.Security.AccessCookieName, cfg.Settings.Security.RefreshCookieName}, "optional-mfa@example.com", password)
	setup := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/setup", nil, []*http.Cookie{access})
	var setupBody map[string]string
	if setup.Code != http.StatusOK || json.Unmarshal(setup.Body.Bytes(), &setupBody) != nil {
		t.Fatalf("setup status=%d body=%s", setup.Code, setup.Body.String())
	}
	code := totpCode(setupBody["secret"], time.Now())
	confirmed := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/confirm", map[string]string{"code": code}, []*http.Cookie{access})
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm status=%d body=%s", confirmed.Code, confirmed.Body.String())
	}
	disabled := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/disable", map[string]string{"password": password, "code": totpCode(setupBody["secret"], time.Now())}, []*http.Cookie{access})
	if disabled.Code != http.StatusNoContent {
		t.Fatalf("disable status=%d body=%s", disabled.Code, disabled.Body.String())
	}
	user, err := store.GetUserByEmail(context.Background(), "optional-mfa@example.com")
	if err != nil {
		t.Fatal(err)
	}
	setting, err := store.GetMFASettings(context.Background(), user.ID)
	if err != nil || setting.Enabled {
		t.Fatalf("MFA remained enabled: setting=%#v err=%v", setting, err)
	}
}

func TestSprint1InvalidExpiryRateLimitAndAuthorizationContracts(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := New(store, cfg).Handler

	protected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/logout"}, {http.MethodGet, "/api/v1/auth/session"}, {http.MethodGet, "/api/v1/auth/sessions"},
		{http.MethodPost, "/api/v1/auth/sessions/revoke"}, {http.MethodGet, "/api/v1/auth/devices"}, {http.MethodPost, "/api/v1/auth/devices/revoke"},
		{http.MethodPost, "/api/v1/auth/mfa/setup"}, {http.MethodPost, "/api/v1/auth/mfa/confirm"}, {http.MethodPost, "/api/v1/auth/mfa/disable"},
		{http.MethodPost, "/api/v1/devices/fingerprint"},
	}
	for _, endpoint := range protected {
		response := authRequest(t, handler, endpoint.method, endpoint.path, map[string]string{}, nil)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated %s %s status=%d body=%s", endpoint.method, endpoint.path, response.Code, response.Body.String())
		}
	}

	verificationExpired := expiredSignedToken(cfg.JWTSecret, models.AuthTokenPurposeEmailVerification)
	verification := authRequest(t, handler, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": verificationExpired}, nil)
	if verification.Code != http.StatusBadRequest || !strings.Contains(verification.Body.String(), "AUTH_TOKEN_EXPIRED") {
		t.Fatalf("expired verification status=%d body=%s", verification.Code, verification.Body.String())
	}
	resetExpired := expiredSignedToken(cfg.JWTSecret, models.AuthTokenPurposePasswordReset)
	reset := authRequest(t, handler, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]string{"token": resetExpired, "password": "ReplacementPassword!43", "confirmPassword": "ReplacementPassword!43"}, nil)
	if reset.Code != http.StatusBadRequest || !strings.Contains(reset.Body.String(), "AUTH_TOKEN_EXPIRED") {
		t.Fatalf("expired reset status=%d body=%s", reset.Code, reset.Body.String())
	}
	expiredChallenge := expiredSignedToken(cfg.JWTSecret, models.AuthTokenPurposeMFAChallenge)
	mfa := authRequest(t, handler, http.MethodPost, "/api/v1/auth/mfa/challenge", map[string]string{"challengeToken": expiredChallenge, "code": "123456"}, nil)
	if mfa.Code != http.StatusUnauthorized || !strings.Contains(mfa.Body.String(), "AUTH_TOKEN_EXPIRED") {
		t.Fatalf("expired MFA status=%d body=%s", mfa.Code, mfa.Body.String())
	}

	underage := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": "minor@example.com", "password": "StrongPassword!42", "country": "ZA", "dateOfBirth": time.Now().AddDate(-17, 0, 0).Format("2006-01-02"), "acceptTerms": true, "acceptFairPlay": true}, nil)
	if underage.Code != http.StatusBadRequest {
		t.Fatalf("underage registration status=%d body=%s", underage.Code, underage.Body.String())
	}
	weak := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": "weak@example.com", "password": "password", "country": "ZA", "dateOfBirth": "1990-01-01", "acceptTerms": true, "acceptFairPlay": true}, nil)
	if weak.Code != http.StatusBadRequest || !strings.Contains(weak.Body.String(), "AUTH_PASSWORD_POLICY") {
		t.Fatalf("weak-password registration status=%d body=%s", weak.Code, weak.Body.String())
	}

	resendRegistration := authRequest(t, handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": "resend@example.com", "password": "StrongPassword!42", "country": "ZA", "dateOfBirth": "1990-01-01", "acceptTerms": true, "acceptFairPlay": true}, nil)
	if resendRegistration.Code != http.StatusCreated {
		t.Fatalf("resend registration status=%d body=%s", resendRegistration.Code, resendRegistration.Body.String())
	}
	resend := authRequest(t, handler, http.MethodPost, "/api/v1/auth/resend-verification", map[string]string{"email": "resend@example.com"}, nil)
	if resend.Code != http.StatusAccepted {
		t.Fatalf("resend status=%d body=%s", resend.Code, resend.Body.String())
	}

	limited := false
	for attempt := 0; attempt < 30; attempt++ {
		response := authRequest(t, handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "missing@example.com", "password": "wrong"}, nil)
		if response.Code == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("login rate limit was not enforced")
	}
}

func TestSprint1PublicEntryAndHealthContracts(t *testing.T) {
	store, err := db.New(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, authTestConfig("")).Handler
	checks := []struct {
		path     string
		contains string
	}{
		{"/health", `"status":"ready"`},
		{"/health/live", `"status":"alive"`},
		{"/health/ready", `"identity":"ready"`},
		{"/api/v1/config/features", "MazeArena"},
		{"/api/v1/platform/stats", "currentSeason"},
		{"/api/v1/platform/puzzle-preview", "lines"},
	}
	for _, check := range checks {
		request := httptest.NewRequest(http.MethodGet, check.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), check.contains) {
			t.Errorf("GET %s status=%d body=%s", check.path, response.Code, response.Body.String())
		}
	}
}
