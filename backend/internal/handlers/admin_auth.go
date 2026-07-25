package handlers

import (
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const adminAudience = "skill-arena-admin-crm"

type adminTokenResponse struct {
	Authenticated         bool                 `json:"authenticated"`
	MFAEnrollmentRequired bool                 `json:"mfaEnrollmentRequired,omitempty"`
	ExpiresIn             int64                `json:"expiresIn"`
	Admin                 models.AdminIdentity `json:"admin"`
}

func adminIdentity(user *models.User) models.AdminIdentity {
	return models.AdminIdentity{
		ID: user.ID, Email: user.Email, DisplayName: user.DisplayName,
		Role: user.Role, Permissions: models.AdminPermissions(user.Role),
	}
}

func signAdminAccessToken(user *models.User, session *models.AuthSession, cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"sub": user.ID, "sid": session.ID, "jti": db.NewAuthToken(),
		"role": user.Role, "permissions": models.AdminPermissions(user.Role),
		"typ": "access", "iss": "skill-arena-api", "aud": adminAudience,
		"iat": time.Now().UTC().Unix(), "exp": time.Now().Add(cfg.Settings.Admin.AccessTTL).Unix(),
	}
	if session.MFAVerified {
		claims["mfaVerified"] = true
	}
	if session.EnrollmentOnly {
		claims["mfaEnrollmentOnly"] = true
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecret))
}

func adminCookie(cfg *config.Config, name, value, path string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: path, Domain: cfg.Settings.Security.CookieDomain,
		MaxAge: maxAge, HttpOnly: true, Secure: cfg.Settings.Security.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
}

func setAdminCookies(w http.ResponseWriter, cfg *config.Config, access, refresh string) {
	http.SetCookie(w, adminCookie(cfg, cfg.Settings.Admin.AccessCookieName, access, "/", int(cfg.Settings.Admin.AccessTTL.Seconds())))
	http.SetCookie(w, adminCookie(cfg, cfg.Settings.Admin.RefreshCookieName, refresh, "/api/v1/admin-crm/auth", int(cfg.Settings.Admin.RefreshTTL.Seconds())))
}

func clearAdminCookies(w http.ResponseWriter, cfg *config.Config) {
	http.SetCookie(w, adminCookie(cfg, cfg.Settings.Admin.AccessCookieName, "", "/", -1))
	http.SetCookie(w, adminCookie(cfg, cfg.Settings.Admin.RefreshCookieName, "", "/api/v1/admin-crm/auth", -1))
}

func markAdminSessionActive(r *http.Request, store *db.Store, cfg *config.Config, sessionID string) error {
	return store.Redis().Set(r.Context(), "admin:session:active:"+sessionID, "1", cfg.Settings.Admin.IdleTimeout)
}

func issueAdminSession(w http.ResponseWriter, r *http.Request, store *db.Store, cfg *config.Config, user *models.User, mfaVerified, enrollmentOnly bool) {
	refresh := db.NewRefreshToken()
	session, err := store.CreateAuthSessionForDevice(
		r.Context(), user.ID, refresh, r.UserAgent(), clientIP(r), "",
		cfg.Settings.Admin.RefreshTTL, mfaVerified, enrollmentOnly,
	)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create administrator session")
		return
	}
	access, err := signAdminAccessToken(user, session, cfg)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to sign administrator session")
		return
	}
	if err := markAdminSessionActive(r, store, cfg, session.ID); err != nil {
		_ = store.RevokeAuthSession(r.Context(), session.ID, user.ID, user.ID, clientIP(r), "admin_session_cache_failure")
		WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "administrator session service is unavailable")
		return
	}
	_ = store.RecordLoginSuccess(r.Context(), user.ID, clientIP(r), r.UserAgent())
	_ = store.AppendAuditLog(r.Context(), user.ID, "admin.auth.login.succeeded", user.ID, clientIP(r), map[string]string{
		"userAgent": r.UserAgent(), "application": adminAudience,
	})
	setAdminCookies(w, cfg, access, refresh)
	writeJSON(w, http.StatusOK, adminTokenResponse{
		Authenticated: true, MFAEnrollmentRequired: enrollmentOnly,
		ExpiresIn: int64(cfg.Settings.Admin.AccessTTL.Seconds()), Admin: adminIdentity(user),
	})
}

func AdminCRMLoginHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		var request authRequest
		if err := decodeJSON(r, &request); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		user, err := store.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(request.Email)))
		if err != nil {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(request.Password))
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid administrator credentials")
			return
		}
		if !models.IsAdministratorRole(user.Role) {
			_ = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
			_ = store.AppendAuditLog(r.Context(), user.ID, "admin.auth.login.denied", user.ID, clientIP(r), map[string]string{"reason": "role"})
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "administrator access is not permitted")
			return
		}
		if user.Status != "active" || !user.EmailVerified {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "administrator account is not active and verified")
			return
		}
		state, _ := store.LoginSecurityState(r.Context(), user.ID)
		if state.LockedUntil != nil && state.LockedUntil.After(time.Now().UTC()) {
			WriteAPIError(w, http.StatusLocked, ErrAccountLocked, "administrator account is temporarily locked")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)) != nil {
			_, _ = store.RecordLoginFailure(r.Context(), user.ID, clientIP(r), r.UserAgent())
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid administrator credentials")
			return
		}
		mfa, _ := store.GetMFASettings(r.Context(), user.ID)
		if !mfa.Enabled {
			_ = store.AppendAuditLog(r.Context(), user.ID, "admin.auth.mfa.enrollment_required", user.ID, clientIP(r), nil)
			issueAdminSession(w, r, store, cfg, user, false, true)
			return
		}
		challenge, err := newSignedAuthToken(cfg, models.AuthTokenPurposeAdminMFAChallenge, 5*time.Minute)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create MFA challenge")
			return
		}
		if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeAdminMFAChallenge, challenge, clientIP(r), 5*time.Minute); err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create MFA challenge")
			return
		}
		writeJSON(w, http.StatusAccepted, mfaRequiredResponse{MFARequired: true, Challenge: challenge, ExpiresIn: 300})
	}
}

func AdminCRMMFAChallengeHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		var request mfaChallengeRequest
		if err := decodeJSON(r, &request); err != nil || request.ChallengeToken == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "challengeToken and an MFA code are required")
			return
		}
		if err := verifySignedAuthToken(cfg, models.AuthTokenPurposeAdminMFAChallenge, request.ChallengeToken); err != nil {
			WriteMappedError(w, http.StatusUnauthorized, err)
			return
		}
		_, user, err := store.InspectAuthToken(r.Context(), models.AuthTokenPurposeAdminMFAChallenge, request.ChallengeToken)
		if err != nil || !models.IsAdministratorRole(user.Role) {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "administrator challenge is invalid")
			return
		}
		setting, err := store.GetMFASettings(r.Context(), user.ID)
		if err != nil || !setting.Enabled {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "MFA is not configured")
			return
		}
		verified := false
		if request.Code != "" {
			if secret, openErr := openSecret(setting.TOTPSecretCiphertext, cfg); openErr == nil {
				verified = verifyTOTP(secret, request.Code, time.Now().UTC())
			}
		}
		if !verified && request.RecoveryCode != "" {
			verified, err = store.ConsumeRecoveryCode(r.Context(), user.ID, sha256Hex(strings.TrimSpace(request.RecoveryCode)), clientIP(r))
		}
		if err != nil || !verified {
			_ = store.AppendAuditLog(r.Context(), user.ID, "admin.auth.mfa.failed", user.ID, clientIP(r), nil)
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid MFA code")
			return
		}
		if _, _, err := store.ConsumeAuthToken(r.Context(), models.AuthTokenPurposeAdminMFAChallenge, request.ChallengeToken, clientIP(r)); err != nil {
			WriteMappedError(w, http.StatusUnauthorized, err)
			return
		}
		issueAdminSession(w, r, store, cfg, user, true, false)
	}
}

func AdminCRMRefreshHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		cookie, err := r.Cookie(cfg.Settings.Admin.RefreshCookieName)
		if err != nil || cookie.Value == "" {
			clearAdminCookies(w, cfg)
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "administrator recovery token is required")
			return
		}
		nextRefresh := db.NewRefreshToken()
		user, session, err := store.RotateRefreshToken(r.Context(), cookie.Value, nextRefresh, r.UserAgent(), clientIP(r), cfg.Settings.Admin.RefreshTTL)
		if err != nil || !models.IsAdministratorRole(user.Role) || !session.MFAVerified {
			clearAdminCookies(w, cfg)
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "administrator session is expired or revoked")
			return
		}
		access, err := signAdminAccessToken(user, session, cfg)
		if err != nil || markAdminSessionActive(r, store, cfg, session.ID) != nil {
			WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "administrator session service is unavailable")
			return
		}
		setAdminCookies(w, cfg, access, nextRefresh)
		writeJSON(w, http.StatusOK, adminTokenResponse{
			Authenticated: true, ExpiresIn: int64(cfg.Settings.Admin.AccessTTL.Seconds()), Admin: adminIdentity(user),
		})
	}
}

func AdminCRMLogoutHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		userID, sessionID := UserIDFromContext(r.Context()), SessionIDFromContext(r.Context())
		if sessionID != "" {
			_ = store.RevokeAuthSession(r.Context(), sessionID, userID, userID, clientIP(r), "admin_logout")
			_ = store.Redis().Del(r.Context(), "admin:session:active:"+sessionID)
		}
		clearAdminCookies(w, cfg)
		w.WriteHeader(http.StatusNoContent)
	}
}

func AdminCRMSessionHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
			return
		}
		user, err := store.GetUserByID(r.Context(), UserIDFromContext(r.Context()))
		if err != nil {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "administrator session is unavailable")
			return
		}
		mfa, _ := store.GetMFASettings(r.Context(), user.ID)
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true, "admin": adminIdentity(user), "mfaEnabled": mfa.Enabled,
			"mfaEnrollmentRequired": MFAEnrollmentOnlyFromContext(r.Context()),
		})
	}
}
