package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type authRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	MFACode      string `json:"mfaCode,omitempty"`
	RecoveryCode string `json:"recoveryCode,omitempty"`
}

type registrationRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	Country        string `json:"country"`
	DateOfBirth    string `json:"dateOfBirth"`
	AcceptTerms    bool   `json:"acceptTerms"`
	AcceptFairPlay bool   `json:"acceptFairPlay"`
}

type tokenResponse struct {
	Authenticated         bool         `json:"authenticated"`
	MFAEnrollmentRequired bool         `json:"mfaEnrollmentRequired,omitempty"`
	ExpiresIn             int64        `json:"expiresIn"`
	User                  *models.User `json:"user"`
}

type mfaRequiredResponse struct {
	MFARequired bool   `json:"mfaRequired"`
	Challenge   string `json:"challengeToken"`
	ExpiresIn   int64  `json:"expiresIn"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type emailRequest struct {
	Email string `json:"email"`
}

type passwordResetConfirmRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type mfaConfirmRequest struct {
	Code string `json:"code"`
}

type mfaChallengeRequest struct {
	ChallengeToken string `json:"challengeToken"`
	Code           string `json:"code,omitempty"`
	RecoveryCode   string `json:"recoveryCode,omitempty"`
}

type mfaDisableRequest struct {
	Password     string `json:"password"`
	Code         string `json:"code,omitempty"`
	RecoveryCode string `json:"recoveryCode,omitempty"`
}

type sessionRevokeRequest struct {
	SessionID string `json:"sessionId"`
}

type deviceRevokeRequest struct {
	DeviceID string `json:"deviceId"`
}

type publicSession struct {
	ID          string     `json:"id"`
	UserAgent   string     `json:"userAgent,omitempty"`
	IPAddress   string     `json:"ipAddress,omitempty"`
	DeviceID    string     `json:"deviceId,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	Current     bool       `json:"current"`
	MFAVerified bool       `json:"mfaVerified"`
}

type deviceRegisterRequest struct {
	Fingerprint string `json:"fingerprint"`
	DeviceName  string `json:"deviceName,omitempty"`
	OS          string `json:"os,omitempty"`
	Browser     string `json:"browser,omitempty"`
}

var dummyPasswordHash = func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("timing-equalization-only"), bcrypt.DefaultCost)
	return hash
}()

func passwordPolicyError(password string) string {
	if len(password) < 12 {
		return "password must be at least 12 characters"
	}
	var hasUpper, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsNumber(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if !hasUpper || !hasNumber || !hasSymbol {
		return "password must include uppercase letters, numbers, and symbols"
	}
	return ""
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func signAccessToken(user *models.User, session *models.AuthSession, cfg *config.Config) (string, error) {
	role := user.Role
	for _, email := range cfg.Settings.Admin.SuperAdminEmails {
		if strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(user.Email)) {
			role = models.RoleSuperAdmin
			break
		}
	}
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"sid":  session.ID,
		"jti":  db.NewAuthToken(),
		"role": role,
		"typ":  "access",
		"iss":  "skill-arena-api",
		"aud":  "skill-arena-web",
		"iat":  time.Now().UTC().Unix(),
		"exp":  time.Now().Add(cfg.Settings.Security.AccessTTL).Unix(),
	}
	if session.MFAVerified {
		claims["mfaVerified"] = true
	}
	if session.EnrollmentOnly {
		claims["mfaEnrollmentOnly"] = true
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func authCookie(cfg *config.Config, name, value, path string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: path, Domain: cfg.Settings.Security.CookieDomain,
		MaxAge: maxAge, HttpOnly: true, Secure: cfg.Settings.Security.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	}
}

func setSessionCookies(w http.ResponseWriter, cfg *config.Config, accessToken, refreshToken string) {
	http.SetCookie(w, authCookie(cfg, cfg.Settings.Security.AccessCookieName, accessToken, "/", int(cfg.Settings.Security.AccessTTL.Seconds())))
	http.SetCookie(w, authCookie(cfg, cfg.Settings.Security.RefreshCookieName, refreshToken, "/api/v1/auth", int(cfg.Settings.Security.RefreshTTL.Seconds())))
}

func clearSessionCookies(w http.ResponseWriter, cfg *config.Config) {
	http.SetCookie(w, authCookie(cfg, cfg.Settings.Security.AccessCookieName, "", "/", -1))
	http.SetCookie(w, authCookie(cfg, cfg.Settings.Security.RefreshCookieName, "", "/api/v1/auth", -1))
}

func privilegedRole(role string) bool {
	return role == models.RoleSuperAdmin || role == models.RoleAdmin || role == models.RoleTreasuryManager || role == models.RoleFraudAnalyst
}

func tokenLink(baseURL, path, token string) string {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Scheme == "" {
		return path + "?token=" + url.QueryEscape(token)
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

func newSignedAuthToken(cfg *config.Config, purpose string, ttl time.Duration) (string, error) {
	nonce, err := randomBase32(32)
	if err != nil {
		return "", err
	}
	expires := strconv.FormatInt(time.Now().UTC().Add(ttl).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(cfg.JWTSecret))
	_, _ = mac.Write([]byte(purpose + "\x00" + nonce + "\x00" + expires))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return nonce + "." + expires + "." + signature, nil
}

func verifySignedAuthToken(cfg *config.Config, purpose, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return errors.New("token not found")
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return errors.New("token not found")
	}
	if time.Now().UTC().Unix() >= expires {
		return errors.New("token expired")
	}
	provided, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return errors.New("token not found")
	}
	mac := hmac.New(sha256.New, []byte(cfg.JWTSecret))
	_, _ = mac.Write([]byte(purpose + "\x00" + parts[0] + "\x00" + parts[1]))
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return errors.New("token not found")
	}
	return nil
}

func enqueueEmail(ctx context.Context, store *db.Store, to, subject, link, template string) error {
	_, err := store.EnqueueJob(ctx, models.JobEmailSend, map[string]string{
		"to":       to,
		"subject":  subject,
		"link":     link,
		"template": template,
	}, time.Now().UTC())
	return err
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}

func encryptionKey(cfg *config.Config) []byte {
	key := cfg.Settings.MFA.EncryptionKey
	if key == "" {
		key = cfg.JWTSecret
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

func sealSecret(secret string, cfg *config.Config) (string, error) {
	block, err := aes.NewCipher(encryptionKey(cfg))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func openSecret(ciphertext string, cfg *config.Config) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey(cfg))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid ciphertext")
	}
	nonce, data := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func randomBase32(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "="), nil
}

func randomRecoveryCode() (string, error) {
	buf := make([]byte, 5)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)), nil
}

func verifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		secretBytes, err = base32.StdEncoding.DecodeString(strings.ToUpper(secret))
		if err != nil {
			return false
		}
	}
	for offset := int64(-1); offset <= 1; offset++ {
		counter := uint64(int64(math.Floor(float64(now.Unix())/30)) + offset)
		var msg [8]byte
		binary.BigEndian.PutUint64(msg[:], counter)
		mac := hmac.New(sha1.New, secretBytes)
		mac.Write(msg[:])
		sum := mac.Sum(nil)
		trunc := sum[len(sum)-1] & 0x0f
		bin := (uint32(sum[trunc])&0x7f)<<24 | (uint32(sum[trunc+1])&0xff)<<16 | (uint32(sum[trunc+2])&0xff)<<8 | (uint32(sum[trunc+3]) & 0xff)
		expected := strconv.Itoa(int(bin % 1000000))
		expected = strings.Repeat("0", 6-len(expected)) + expected
		if hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

func issueSession(w http.ResponseWriter, r *http.Request, store *db.Store, cfg *config.Config, user *models.User, mfaVerified, enrollmentOnly bool) {
	refreshToken := db.NewRefreshToken()
	deviceID := ""
	if fingerprint := strings.TrimSpace(r.Header.Get("X-Device-Fingerprint")); fingerprint != "" {
		device, err := store.RegisterDevice(r.Context(), user.ID, fingerprint, r.Header.Get("X-Device-Name"), r.Header.Get("X-Device-OS"), r.Header.Get("X-Device-Browser"))
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to register device")
			return
		}
		deviceID = device.ID
	}
	session, err := store.CreateAuthSessionForDevice(r.Context(), user.ID, refreshToken, r.UserAgent(), clientIP(r), deviceID, cfg.Settings.Security.RefreshTTL, mfaVerified, enrollmentOnly)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create session")
		return
	}
	signed, err := signAccessToken(user, session, cfg)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to sign token")
		return
	}
	_ = store.RecordLoginSuccess(r.Context(), user.ID, clientIP(r), r.UserAgent())
	_ = store.AppendAuditLog(r.Context(), user.ID, "auth.login.succeeded", user.ID, clientIP(r), map[string]string{"userAgent": r.UserAgent()})

	w.Header().Set("Content-Type", "application/json")
	setSessionCookies(w, cfg, signed, refreshToken)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResponse{Authenticated: true, MFAEnrollmentRequired: enrollmentOnly, User: user, ExpiresIn: int64(cfg.Settings.Security.AccessTTL.Seconds())})
}

func adultOn(date time.Time, now time.Time) bool {
	eighteenth := date.AddDate(18, 0, 0)
	return !eighteenth.After(now)
}

func RegisterHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req registrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		req.Country = strings.ToUpper(strings.TrimSpace(req.Country))
		if req.Email == "" || req.Password == "" || len(req.Country) != 2 {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "email, password, and ISO country code are required")
			return
		}
		birthDate, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err != nil || !adultOn(birthDate, time.Now().UTC()) {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "you must be at least 18 years old")
			return
		}
		if !req.AcceptTerms || !req.AcceptFairPlay {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "terms and fair play rules must be accepted")
			return
		}
		if policyErr := passwordPolicyError(req.Password); policyErr != "" {
			WriteAPIError(w, http.StatusBadRequest, ErrPasswordPolicy, policyErr)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to secure password")
			return
		}

		user := models.NewUser("", req.Email, string(hash))
		acceptedAt := time.Now().UTC()
		user.Country = req.Country
		user.DateOfBirth = &birthDate
		user.TermsAcceptedAt = &acceptedAt
		if err := store.CreateUser(r.Context(), user); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "exists") {
				WriteAPIError(w, http.StatusConflict, ErrConflict, "an account already exists for this email")
				return
			}
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create account")
			return
		}
		_ = store.AppendAuditLog(r.Context(), user.ID, "auth.user.registered", user.ID, clientIP(r), nil)
		rawToken, err := newSignedAuthToken(cfg, models.AuthTokenPurposeEmailVerification, 24*time.Hour)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create verification token")
			return
		}
		if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeEmailVerification, rawToken, clientIP(r), 24*time.Hour); err != nil {
			WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "verification delivery is temporarily unavailable")
			return
		}
		if err := enqueueEmail(r.Context(), store, user.Email, "Verify your Skill Arena email", tokenLink(cfg.Settings.Email.BaseURL, "/auth/verify-email", rawToken), "email_verification"); err != nil {
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.email.delivery_failed", user.ID, clientIP(r), map[string]string{"template": "email_verification"})
			WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "verification delivery is temporarily unavailable")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"status": "verification_required", "email": user.Email})
	}
}

func LoginHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}

		user, err := store.GetUserByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
		if err != nil {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(req.Password))
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid email or password")
			return
		}
		if user.Status != "" && user.Status != "active" {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "account is not active")
			return
		}
		state, _ := store.LoginSecurityState(r.Context(), user.ID)
		if state.LockedUntil != nil && state.LockedUntil.After(time.Now().UTC()) {
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.login.locked", user.ID, clientIP(r), nil)
			WriteAPIError(w, http.StatusLocked, ErrAccountLocked, "account temporarily locked")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			_, _ = store.RecordLoginFailure(r.Context(), user.ID, clientIP(r), r.UserAgent())
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid email or password")
			return
		}
		if !user.EmailVerified {
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.login.email_unverified", user.ID, clientIP(r), nil)
			WriteAPIError(w, http.StatusForbidden, ErrEmailUnverified, "verify your email before signing in")
			return
		}

		mfa, _ := store.GetMFASettings(r.Context(), user.ID)
		enrollmentOnly := false
		if privilegedRole(user.Role) && !mfa.Enabled {
			enrollmentOnly = true
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.mfa.enrollment_required", user.ID, clientIP(r), nil)
		}
		if mfa.Enabled {
			rawChallenge, err := newSignedAuthToken(cfg, models.AuthTokenPurposeMFAChallenge, 5*time.Minute)
			if err != nil {
				WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create MFA challenge")
				return
			}
			if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeMFAChallenge, rawChallenge, clientIP(r), 5*time.Minute); err != nil {
				WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create MFA challenge")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(mfaRequiredResponse{MFARequired: true, Challenge: rawChallenge, ExpiresIn: int64((5 * time.Minute).Seconds())})
			return
		}

		issueSession(w, r, store, cfg, user, false, enrollmentOnly)
	}
}

func MFAChallengeHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req mfaChallengeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ChallengeToken == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "challengeToken and an MFA code are required")
			return
		}
		if err := verifySignedAuthToken(cfg, models.AuthTokenPurposeMFAChallenge, req.ChallengeToken); err != nil {
			WriteMappedError(w, http.StatusUnauthorized, err)
			return
		}
		_, user, err := store.InspectAuthToken(r.Context(), models.AuthTokenPurposeMFAChallenge, req.ChallengeToken)
		if err != nil {
			WriteMappedError(w, http.StatusUnauthorized, err)
			return
		}
		setting, err := store.GetMFASettings(r.Context(), user.ID)
		if err != nil || !setting.Enabled {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "MFA is not configured")
			return
		}
		verified := false
		if req.Code != "" && setting.TOTPSecretCiphertext != "" {
			if secret, err := openSecret(setting.TOTPSecretCiphertext, cfg); err == nil {
				verified = verifyTOTP(secret, req.Code, time.Now().UTC())
			}
		}
		if !verified && req.RecoveryCode != "" {
			verified, err = store.ConsumeRecoveryCode(r.Context(), user.ID, sha256Hex(strings.TrimSpace(req.RecoveryCode)), clientIP(r))
			if err != nil {
				WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to validate recovery code")
				return
			}
		}
		if !verified {
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.mfa.challenge.failed", user.ID, clientIP(r), nil)
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid MFA code")
			return
		}
		if _, _, err := store.ConsumeAuthToken(r.Context(), models.AuthTokenPurposeMFAChallenge, req.ChallengeToken, clientIP(r)); err != nil {
			WriteMappedError(w, http.StatusUnauthorized, err)
			return
		}
		issueSession(w, r, store, cfg, user, true, false)
	}
}

func RefreshTokenHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if cookie, err := r.Cookie(cfg.Settings.Security.RefreshCookieName); err == nil && cookie.Value != "" {
			req.RefreshToken = cookie.Value
		}
		if req.RefreshToken == "" {
			clearSessionCookies(w, cfg)
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "session recovery token is required")
			return
		}

		newRefresh := db.NewRefreshToken()
		user, session, err := store.RotateRefreshToken(r.Context(), req.RefreshToken, newRefresh, r.UserAgent(), clientIP(r), cfg.Settings.Security.RefreshTTL)
		if err != nil {
			clearSessionCookies(w, cfg)
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "session is expired or revoked")
			return
		}

		signed, err := signAccessToken(user, session, cfg)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to sign token")
			return
		}
		_ = store.AppendAuditLog(r.Context(), user.ID, "auth.token.refreshed", user.ID, clientIP(r), nil)

		w.Header().Set("Content-Type", "application/json")
		setSessionCookies(w, cfg, signed, newRefresh)
		json.NewEncoder(w).Encode(tokenResponse{Authenticated: true, MFAEnrollmentRequired: session.EnrollmentOnly, User: user, ExpiresIn: int64(cfg.Settings.Security.AccessTTL.Seconds())})
	}
}

func LogoutHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req refreshRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if cookie, err := r.Cookie(cfg.Settings.Security.RefreshCookieName); err == nil && cookie.Value != "" {
			req.RefreshToken = cookie.Value
		}
		userID := UserIDFromContext(r.Context())
		if req.RefreshToken != "" {
			_ = store.RevokeRefreshToken(r.Context(), req.RefreshToken, userID, clientIP(r))
		} else if sessionID := SessionIDFromContext(r.Context()); sessionID != "" {
			_ = store.RevokeAuthSession(r.Context(), sessionID, userID, userID, clientIP(r), "logout")
		}
		clearSessionCookies(w, cfg)
		w.WriteHeader(http.StatusNoContent)
	}
}

func VerifyEmailHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.URL.Query().Get("token") == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.Token == "" {
			req.Token = r.URL.Query().Get("token")
		}
		if err := verifySignedAuthToken(cfg, models.AuthTokenPurposeEmailVerification, req.Token); err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if req.Token == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "token is required")
			return
		}

		_, user, err := store.ConsumeAuthToken(r.Context(), models.AuthTokenPurposeEmailVerification, req.Token, clientIP(r))
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "token already used") && user != nil && user.EmailVerified {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if user.EmailVerified {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := store.VerifyEmail(r.Context(), user.ID); err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to verify email")
			return
		}
		_ = store.AppendAuditLog(r.Context(), user.ID, "auth.email.verified", user.ID, clientIP(r), nil)

		w.WriteHeader(http.StatusNoContent)
	}
}

func ResendVerificationHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req emailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		user, err := store.GetUserByEmail(r.Context(), req.Email)
		if err == nil && !user.EmailVerified {
			raw, tokenErr := newSignedAuthToken(cfg, models.AuthTokenPurposeEmailVerification, 24*time.Hour)
			if tokenErr != nil {
				WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "verification delivery is temporarily unavailable")
				return
			}
			if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeEmailVerification, raw, clientIP(r), 24*time.Hour); err != nil || enqueueEmail(r.Context(), store, user.Email, "Verify your Skill Arena email", tokenLink(cfg.Settings.Email.BaseURL, "/auth/verify-email", raw), "email_verification") != nil {
				_ = store.AppendAuditLog(r.Context(), user.ID, "auth.email.delivery_failed", user.ID, clientIP(r), map[string]string{"template": "email_verification"})
				WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "verification delivery is temporarily unavailable")
				return
			}
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.email.verification_resent", user.ID, clientIP(r), nil)
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func PasswordResetRequestHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req emailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		user, err := store.GetUserByEmail(r.Context(), req.Email)
		if err == nil {
			raw, tokenErr := newSignedAuthToken(cfg, models.AuthTokenPurposePasswordReset, 30*time.Minute)
			if tokenErr != nil {
				WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "password recovery is temporarily unavailable")
				return
			}
			if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposePasswordReset, raw, clientIP(r), 30*time.Minute); err != nil || enqueueEmail(r.Context(), store, user.Email, "Reset your Skill Arena password", tokenLink(cfg.Settings.Email.BaseURL, "/auth/reset-password", raw), "password_reset") != nil {
				_ = store.AppendAuditLog(r.Context(), user.ID, "auth.email.delivery_failed", user.ID, clientIP(r), map[string]string{"template": "password_reset"})
				WriteAPIError(w, http.StatusServiceUnavailable, ErrInternal, "password recovery is temporarily unavailable")
				return
			}
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.password_reset.requested", user.ID, clientIP(r), nil)
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func PasswordResetConfirmHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req passwordResetConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.Password != req.ConfirmPassword {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "password confirmation does not match")
			return
		}
		if policyErr := passwordPolicyError(req.Password); policyErr != "" {
			WriteAPIError(w, http.StatusBadRequest, ErrPasswordPolicy, policyErr)
			return
		}
		if err := verifySignedAuthToken(cfg, models.AuthTokenPurposePasswordReset, req.Token); err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		_, user, err := store.InspectAuthToken(r.Context(), models.AuthTokenPurposePasswordReset, req.Token)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		history, _ := store.PasswordHistory(r.Context(), user.ID)
		for _, entry := range history {
			if bcrypt.CompareHashAndPassword([]byte(entry.PasswordHash), []byte(req.Password)) == nil {
				WriteAPIError(w, http.StatusBadRequest, ErrPasswordPolicy, "password was used recently")
				return
			}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to secure password")
			return
		}
		if err := store.CompletePasswordReset(r.Context(), req.Token, string(hash), clientIP(r)); err != nil {
			if strings.Contains(err.Error(), "token") {
				WriteMappedError(w, http.StatusBadRequest, err)
				return
			}
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to update password")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func KYCSubmitHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if err := store.SubmitKYC(r.Context(), userID); err != nil {
			http.Error(w, fmt.Sprintf("failed to submit KYC: %v", err), http.StatusInternalServerError)
			return
		}
		_ = store.AppendAuditLog(r.Context(), userID, "identity.kyc.submitted", userID, clientIP(r), nil)

		w.WriteHeader(http.StatusAccepted)
	}
}

func KYCStatusHandler(store *db.Store) http.HandlerFunc {
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

		status, err := store.GetKYCStatus(r.Context(), userID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load KYC status: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"kycStatus": status})
	}
}

func RegisterDeviceHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req deviceRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Fingerprint == "" {
			http.Error(w, "fingerprint is required", http.StatusBadRequest)
			return
		}

		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		device, err := store.RegisterDevice(r.Context(), userID, req.Fingerprint, req.DeviceName, req.OS, req.Browser)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to register device: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(device)
	}
}

func SessionStatusHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		user, err := store.GetUserByID(r.Context(), UserIDFromContext(r.Context()))
		if err != nil {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "session is not available")
			return
		}
		mfa, _ := store.GetMFASettings(r.Context(), user.ID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"authenticated":         true,
			"user":                  user,
			"mfaEnabled":            mfa.Enabled,
			"mfaEnrollmentRequired": MFAEnrollmentOnlyFromContext(r.Context()),
		})
	}
}

func SessionsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromContext(r.Context())
		sessions, err := store.ListAuthSessions(r.Context(), userID)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to load sessions")
			return
		}
		currentID := SessionIDFromContext(r.Context())
		result := make([]publicSession, 0, len(sessions))
		for _, session := range sessions {
			result = append(result, publicSession{ID: session.ID, UserAgent: session.UserAgent, IPAddress: session.IPAddress, DeviceID: session.DeviceID, CreatedAt: session.CreatedAt, ExpiresAt: session.ExpiresAt, RevokedAt: session.RevokedAt, Current: session.ID == currentID, MFAVerified: session.MFAVerified})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sessions": result})
	}
}

func SessionRevokeHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req sessionRevokeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "sessionId is required")
			return
		}
		userID := UserIDFromContext(r.Context())
		if err := store.RevokeAuthSession(r.Context(), req.SessionID, userID, userID, clientIP(r), "user_revoked"); err != nil {
			WriteMappedError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func DevicesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		devices, err := store.ListDevices(r.Context(), UserIDFromContext(r.Context()))
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to load devices")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"devices": devices})
	}
}

func DeviceRevokeHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req deviceRevokeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DeviceID == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "deviceId is required")
			return
		}
		userID := UserIDFromContext(r.Context())
		if err := store.RevokeDevice(r.Context(), userID, req.DeviceID, userID, clientIP(r)); err != nil {
			WriteMappedError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func MFASetupHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to load user")
			return
		}
		existing, _ := store.GetMFASettings(r.Context(), userID)
		if existing.Enabled {
			WriteAPIError(w, http.StatusConflict, ErrConflict, "MFA is already enabled")
			return
		}
		secret, err := randomBase32(20)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create MFA secret")
			return
		}
		ciphertext, err := sealSecret(secret, cfg)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to protect MFA secret")
			return
		}
		setting := &models.MFASettings{UserID: userID, Enabled: false, TOTPSecretCiphertext: ciphertext}
		if err := store.SaveMFASettings(r.Context(), setting, userID, clientIP(r)); err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to save MFA setup")
			return
		}
		issuer := cfg.Settings.MFA.Issuer
		otpauth := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", url.QueryEscape(issuer), url.QueryEscape(user.Email), secret, url.QueryEscape(issuer))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"secret": secret, "otpauthUrl": otpauth})
	}
}

func MFAConfirmHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var req mfaConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		setting, _ := store.GetMFASettings(r.Context(), userID)
		secret, err := openSecret(setting.TOTPSecretCiphertext, cfg)
		if err != nil || !verifyTOTP(secret, req.Code, time.Now().UTC()) {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid MFA code")
			return
		}
		recoveryCodes := make([]string, 0, 10)
		recoveryHashes := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			code, err := randomRecoveryCode()
			if err != nil {
				WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to create recovery codes")
				return
			}
			recoveryCodes = append(recoveryCodes, code)
			recoveryHashes = append(recoveryHashes, sha256Hex(code))
		}
		setting.Enabled = true
		setting.RecoveryCodeHashes = recoveryHashes
		setting.ConfirmedAt = time.Now().UTC()
		if err := store.SaveMFASettings(r.Context(), setting, userID, clientIP(r)); err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to enable MFA")
			return
		}
		if sessionID := SessionIDFromContext(r.Context()); sessionID != "" {
			if err := store.MarkSessionMFA(r.Context(), sessionID, userID, true, false); err != nil {
				WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to complete MFA session")
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"recoveryCodes": recoveryCodes})
	}
}

func MFADisableHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		userID := UserIDFromContext(r.Context())
		if userID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "account not found")
			return
		}
		if privilegedRole(user.Role) {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "MFA is mandatory for privileged accounts")
			return
		}
		var req mfaDisableRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "password and MFA proof are required")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid security confirmation")
			return
		}
		setting, _ := store.GetMFASettings(r.Context(), userID)
		verified := false
		if req.Code != "" && setting.TOTPSecretCiphertext != "" {
			if secret, err := openSecret(setting.TOTPSecretCiphertext, cfg); err == nil {
				verified = verifyTOTP(secret, req.Code, time.Now().UTC())
			}
		}
		if !verified && req.RecoveryCode != "" {
			verified, _ = store.ConsumeRecoveryCode(r.Context(), userID, sha256Hex(strings.TrimSpace(req.RecoveryCode)), clientIP(r))
		}
		if !verified {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid MFA proof")
			return
		}
		setting.Enabled = false
		setting.TOTPSecretCiphertext = ""
		setting.RecoveryCodeHashes = nil
		if err := store.SaveMFASettings(r.Context(), setting, userID, clientIP(r)); err != nil {
			WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "failed to disable MFA")
			return
		}
		_ = store.RevokeUserSessions(r.Context(), userID, userID, clientIP(r), "mfa_disabled")
		clearSessionCookies(w, cfg)
		w.WriteHeader(http.StatusNoContent)
	}
}
