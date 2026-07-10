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
	"fmt"
	"io"
	"math"
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

type tokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
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

type deviceRegisterRequest struct {
	Fingerprint string `json:"fingerprint"`
	DeviceName  string `json:"deviceName,omitempty"`
	OS          string `json:"os,omitempty"`
	Browser     string `json:"browser,omitempty"`
}

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
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return r.RemoteAddr
}

func signAccessToken(user *models.User, cfg *config.Config) (string, error) {
	return signAccessTokenWithMFAState(user, cfg, false, false)
}

func signAccessTokenWithMFAState(user *models.User, cfg *config.Config, mfaVerified, enrollmentOnly bool) (string, error) {
	role := user.Role
	for _, email := range cfg.Settings.Admin.SuperAdminEmails {
		if strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(user.Email)) {
			role = models.RoleSuperAdmin
			break
		}
	}
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": role,
		"typ":  "access",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
	}
	if mfaVerified {
		claims["mfaVerified"] = true
	}
	if enrollmentOnly {
		claims["mfaEnrollmentOnly"] = true
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
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

func enqueueEmail(ctx context.Context, store *db.Store, to, subject, link, template string) {
	_, _ = store.EnqueueJob(ctx, models.JobEmailSend, map[string]string{
		"to":       to,
		"subject":  subject,
		"link":     link,
		"template": template,
	}, time.Now().UTC())
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
	signed, err := signAccessTokenWithMFAState(user, cfg, mfaVerified, enrollmentOnly)
	if err != nil {
		http.Error(w, "failed to sign token", http.StatusInternalServerError)
		return
	}
	refreshToken := db.NewRefreshToken()
	if _, err := store.CreateAuthSession(r.Context(), user.ID, refreshToken, r.UserAgent(), clientIP(r), 30*24*time.Hour); err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	_ = store.RecordLoginSuccess(r.Context(), user.ID, clientIP(r), r.UserAgent())
	_ = store.AppendAuditLog(r.Context(), user.ID, "auth.login.succeeded", user.ID, clientIP(r), map[string]string{"userAgent": r.UserAgent()})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenResponse{Token: signed, RefreshToken: refreshToken, ExpiresIn: int64((15 * time.Minute).Seconds())})
}

func RegisterHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password are required", http.StatusBadRequest)
			return
		}
		if policyErr := passwordPolicyError(req.Password); policyErr != "" {
			http.Error(w, policyErr, http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to secure password", http.StatusInternalServerError)
			return
		}

		user := models.NewUser("", req.Email, string(hash))
		if err := store.CreateUser(r.Context(), user); err != nil {
			http.Error(w, fmt.Sprintf("failed to create user: %v", err), http.StatusInternalServerError)
			return
		}
		_ = store.AppendAuditLog(r.Context(), user.ID, "auth.user.registered", user.ID, clientIP(r), nil)
		rawToken := db.NewAuthToken()
		if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeEmailVerification, rawToken, clientIP(r), 24*time.Hour); err == nil {
			enqueueEmail(r.Context(), store, user.Email, "Verify your Skill Arena email", tokenLink(config.Runtime().Email.BaseURL, "/verify-email", rawToken), "email_verification")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
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
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByEmail(r.Context(), req.Email)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		state, _ := store.LoginSecurityState(r.Context(), user.ID)
		if state.LockedUntil != nil && state.LockedUntil.After(time.Now().UTC()) {
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.login.locked", user.ID, clientIP(r), nil)
			http.Error(w, "account temporarily locked", http.StatusLocked)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			_, _ = store.RecordLoginFailure(r.Context(), user.ID, clientIP(r), r.UserAgent())
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		mfa, _ := store.GetMFASettings(r.Context(), user.ID)
		enrollmentOnly := false
		if privilegedRole(user.Role) && !mfa.Enabled {
			enrollmentOnly = true
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.mfa.enrollment_required", user.ID, clientIP(r), nil)
		}
		mfaRequired := mfa.Enabled
		mfaVerified := false
		if mfaRequired {
			if req.MFACode != "" && mfa.Enabled && mfa.TOTPSecretCiphertext != "" {
				if secret, err := openSecret(mfa.TOTPSecretCiphertext, cfg); err == nil {
					mfaVerified = verifyTOTP(secret, req.MFACode, time.Now().UTC())
				}
			}
			if !mfaVerified && req.RecoveryCode != "" {
				used, _ := store.ConsumeRecoveryCode(r.Context(), user.ID, sha256Hex(req.RecoveryCode), clientIP(r))
				mfaVerified = used
			}
			if !mfaVerified {
				rawChallenge := db.NewAuthToken()
				_, _ = store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeMFAChallenge, rawChallenge, clientIP(r), 5*time.Minute)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(mfaRequiredResponse{MFARequired: true, Challenge: rawChallenge, ExpiresIn: int64((5 * time.Minute).Seconds())})
				return
			}
		}

		issueSession(w, r, store, cfg, user, mfaVerified, enrollmentOnly)
	}
}

func RefreshTokenHandler(store *db.Store, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.RefreshToken == "" {
			http.Error(w, "refreshToken is required", http.StatusBadRequest)
			return
		}

		newRefresh := db.NewRefreshToken()
		user, _, err := store.RotateRefreshToken(r.Context(), req.RefreshToken, newRefresh, r.UserAgent(), clientIP(r), 30*24*time.Hour)
		if err != nil {
			http.Error(w, "invalid refresh token", http.StatusUnauthorized)
			return
		}

		signed, err := signAccessToken(user, cfg)
		if err != nil {
			http.Error(w, "failed to sign token", http.StatusInternalServerError)
			return
		}
		_ = store.AppendAuditLog(r.Context(), user.ID, "auth.token.refreshed", user.ID, clientIP(r), nil)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{Token: signed, RefreshToken: newRefresh, ExpiresIn: int64((15 * time.Minute).Seconds())})
	}
}

func LogoutHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		userID := UserIDFromContext(r.Context())
		if req.RefreshToken != "" {
			_ = store.RevokeRefreshToken(r.Context(), req.RefreshToken, userID, clientIP(r))
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func VerifyEmailHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.URL.Query().Get("token") == "" {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.Token == "" {
			req.Token = r.URL.Query().Get("token")
		}
		if req.Token == "" {
			http.Error(w, "token is required", http.StatusBadRequest)
			return
		}

		_, user, err := store.ConsumeAuthToken(r.Context(), models.AuthTokenPurposeEmailVerification, req.Token, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if user.EmailVerified {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := store.VerifyEmail(r.Context(), user.ID); err != nil {
			http.Error(w, fmt.Sprintf("failed to verify email: %v", err), http.StatusInternalServerError)
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
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		user, err := store.GetUserByEmail(r.Context(), req.Email)
		if err == nil && !user.EmailVerified {
			raw := db.NewAuthToken()
			if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposeEmailVerification, raw, clientIP(r), 24*time.Hour); err == nil {
				enqueueEmail(r.Context(), store, user.Email, "Verify your Skill Arena email", tokenLink(cfg.Settings.Email.BaseURL, "/verify-email", raw), "email_verification")
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
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		user, err := store.GetUserByEmail(r.Context(), req.Email)
		if err == nil {
			raw := db.NewAuthToken()
			if _, err := store.CreateAuthToken(r.Context(), user.ID, models.AuthTokenPurposePasswordReset, raw, clientIP(r), 30*time.Minute); err == nil {
				enqueueEmail(r.Context(), store, user.Email, "Reset your Skill Arena password", tokenLink(cfg.Settings.Email.BaseURL, "/reset-password", raw), "password_reset")
			}
			_ = store.AppendAuditLog(r.Context(), user.ID, "auth.password_reset.requested", user.ID, clientIP(r), nil)
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func PasswordResetConfirmHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req passwordResetConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.Password != req.ConfirmPassword {
			http.Error(w, "password confirmation does not match", http.StatusBadRequest)
			return
		}
		if policyErr := passwordPolicyError(req.Password); policyErr != "" {
			http.Error(w, policyErr, http.StatusBadRequest)
			return
		}
		_, user, err := store.ConsumeAuthToken(r.Context(), models.AuthTokenPurposePasswordReset, req.Token, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		history, _ := store.PasswordHistory(r.Context(), user.ID)
		for _, entry := range history {
			if bcrypt.CompareHashAndPassword([]byte(entry.PasswordHash), []byte(req.Password)) == nil {
				http.Error(w, "password was used recently", http.StatusBadRequest)
				return
			}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to secure password", http.StatusInternalServerError)
			return
		}
		if err := store.UpdatePassword(r.Context(), user.ID, string(hash), sha256Hex(req.Password), clientIP(r)); err != nil {
			http.Error(w, "failed to update password", http.StatusInternalServerError)
			return
		}
		_ = store.RevokeUserSessions(r.Context(), user.ID, user.ID, clientIP(r), "password_reset")
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
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}
		secret, err := randomBase32(20)
		if err != nil {
			http.Error(w, "failed to create mfa secret", http.StatusInternalServerError)
			return
		}
		ciphertext, err := sealSecret(secret, cfg)
		if err != nil {
			http.Error(w, "failed to protect mfa secret", http.StatusInternalServerError)
			return
		}
		setting := &models.MFASettings{UserID: userID, Enabled: false, TOTPSecretCiphertext: ciphertext}
		if err := store.SaveMFASettings(r.Context(), setting, userID, clientIP(r)); err != nil {
			http.Error(w, "failed to save mfa setup", http.StatusInternalServerError)
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
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		setting, _ := store.GetMFASettings(r.Context(), userID)
		secret, err := openSecret(setting.TOTPSecretCiphertext, cfg)
		if err != nil || !verifyTOTP(secret, req.Code, time.Now().UTC()) {
			http.Error(w, "invalid mfa code", http.StatusUnauthorized)
			return
		}
		recoveryCodes := make([]string, 0, 10)
		recoveryHashes := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			code, err := randomRecoveryCode()
			if err != nil {
				http.Error(w, "failed to create recovery codes", http.StatusInternalServerError)
				return
			}
			recoveryCodes = append(recoveryCodes, code)
			recoveryHashes = append(recoveryHashes, sha256Hex(code))
		}
		setting.Enabled = true
		setting.RecoveryCodeHashes = recoveryHashes
		setting.ConfirmedAt = time.Now().UTC()
		if err := store.SaveMFASettings(r.Context(), setting, userID, clientIP(r)); err != nil {
			http.Error(w, "failed to enable mfa", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"recoveryCodes": recoveryCodes})
	}
}

func MFADisableHandler(store *db.Store) http.HandlerFunc {
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
		setting, _ := store.GetMFASettings(r.Context(), userID)
		setting.Enabled = false
		setting.RecoveryCodeHashes = nil
		if err := store.SaveMFASettings(r.Context(), setting, userID, clientIP(r)); err != nil {
			http.Error(w, "failed to disable mfa", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
