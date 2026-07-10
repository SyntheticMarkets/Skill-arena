package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/models"

	"github.com/golang-jwt/jwt/v4"
)

type contextKey string

const userIDKey contextKey = "userId"
const userRoleKey contextKey = "userRole"
const mfaEnrollmentOnlyKey contextKey = "mfaEnrollmentOnly"

var rateLimiter = newMemoryRateLimiter()

type memoryRateLimiter struct {
	mu      sync.Mutex
	events  map[string][]time.Time
	limiter map[string]rateLimitRule
}

type rateLimitRule struct {
	Limit  int
	Window time.Duration
}

func newMemoryRateLimiter() *memoryRateLimiter {
	settings := config.Runtime().RateLimit
	return &memoryRateLimiter{
		events: map[string][]time.Time{},
		limiter: map[string]rateLimitRule{
			"/api/v1/auth/login":                  {Limit: settings.LoginLimit, Window: settings.DefaultWindow},
			"/api/v1/auth/register":               {Limit: settings.RegisterLimit, Window: settings.DefaultWindow},
			"/api/v1/auth/resend-verification":    {Limit: settings.RegisterLimit, Window: settings.DefaultWindow},
			"/api/v1/auth/password-reset/request": {Limit: settings.RegisterLimit, Window: settings.DefaultWindow},
			"/api/v1/auth/password-reset/confirm": {Limit: settings.LoginLimit, Window: settings.DefaultWindow},
			"/api/v1/auth/mfa/confirm":            {Limit: settings.LoginLimit, Window: settings.DefaultWindow},
			"/api/v1/pvp/join":                    {Limit: settings.MatchCreationLimit, Window: settings.DefaultWindow},
			"/api/v1/games/start":                 {Limit: settings.MatchCreationLimit, Window: settings.DefaultWindow},
			"/api/v1/replays":                     {Limit: settings.ReplayLimit, Window: settings.DefaultWindow},
			"/api/v1/wallet/withdraw":             {Limit: settings.WithdrawalLimit, Window: settings.DefaultWindow},
			"default":                             {Limit: settings.DefaultLimit, Window: settings.DefaultWindow},
		},
	}
}

func (l *memoryRateLimiter) allow(key string, rule rateLimitRule, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-rule.Window)
	events := l.events[key]
	kept := events[:0]
	for _, event := range events {
		if event.After(cutoff) {
			kept = append(kept, event)
		}
	}
	if len(kept) >= rule.Limit {
		l.events[key] = kept
		return false
	}
	kept = append(kept, now)
	l.events[key] = kept
	return true
}

func (l *memoryRateLimiter) ruleFor(path string) rateLimitRule {
	if rule, ok := l.limiter[path]; ok {
		return rule
	}
	if strings.HasPrefix(path, "/api/v1/replays/") {
		return l.limiter["/api/v1/replays"]
	}
	return l.limiter["default"]
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		rule := rateLimiter.ruleFor(r.URL.Path)
		key := clientIP(r) + ":" + r.URL.Path
		if !rateLimiter.allow(key, rule, time.Now().UTC()) {
			WriteAPIError(w, http.StatusTooManyRequests, ErrRateLimited, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RateLimitMiddlewareWithStore(store *db.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		rule := rateLimiter.ruleFor(r.URL.Path)
		key := clientIP(r) + ":" + r.URL.Path
		allowed, err := redisRateLimitAllow(r.Context(), store, key, rule, time.Now().UTC())
		if err != nil {
			allowed = rateLimiter.allow(key, rule, time.Now().UTC())
		}
		if !allowed {
			WriteAPIError(w, http.StatusTooManyRequests, ErrRateLimited, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func redisRateLimitAllow(ctx context.Context, store *db.Store, key string, rule rateLimitRule, now time.Time) (bool, error) {
	if store == nil || store.Redis() == nil {
		return false, errors.New("redis unavailable")
	}
	redisKey := "rate:" + key
	value, ok, err := store.Redis().Get(ctx, redisKey)
	if err != nil {
		return false, err
	}
	events := []time.Time{}
	if ok && value != "" {
		if err := json.Unmarshal([]byte(value), &events); err != nil {
			events = []time.Time{}
		}
	}
	cutoff := now.Add(-rule.Window)
	kept := events[:0]
	for _, event := range events {
		if event.After(cutoff) {
			kept = append(kept, event)
		}
	}
	if len(kept) >= rule.Limit {
		data, _ := json.Marshal(kept)
		_ = store.Redis().Set(ctx, redisKey, string(data), rule.Window)
		return false, nil
	}
	kept = append(kept, now)
	data, err := json.Marshal(kept)
	if err != nil {
		return false, err
	}
	return true, store.Redis().Set(ctx, redisKey, string(data), rule.Window)
}

func MaintenanceMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	blockedPaths := map[string]bool{
		"/api/v1/games/start":          true,
		"/api/v1/house/start":          true,
		"/api/v1/pvp/join":             true,
		"/api/v1/tournaments/register": true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		settings := cfg.Settings
		if settings == nil {
			settings = config.Runtime()
		}
		if !settings.Maintenance.Enabled || !blockedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		if settings.Maintenance.AllowSuperAdmins && UserRoleFromContext(r.Context()) == models.RoleSuperAdmin {
			next.ServeHTTP(w, r)
			return
		}
		WriteAPIError(w, http.StatusServiceUnavailable, ErrMaintenance, settings.Maintenance.Message)
	})
}

func AuthMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenValue := r.Header.Get("Authorization")
		if tokenValue == "" {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "authorization token is required")
			return
		}

		if strings.HasPrefix(strings.ToLower(tokenValue), "bearer ") {
			tokenValue = strings.TrimSpace(tokenValue[7:])
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid authorization token")
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			WriteAPIError(w, http.StatusUnauthorized, ErrUnauthorized, "invalid authorization token")
			return
		}

		role, _ := claims["role"].(string)
		if role == "" {
			role = "player"
		}

		ctx := context.WithValue(r.Context(), userIDKey, sub)
		ctx = context.WithValue(ctx, userRoleKey, role)
		if enrollmentOnly, _ := claims["mfaEnrollmentOnly"].(bool); enrollmentOnly {
			ctx = context.WithValue(ctx, mfaEnrollmentOnlyKey, true)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

func UserRoleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userRoleKey).(string); ok {
		return v
	}
	return "player"
}

func RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v, _ := r.Context().Value(mfaEnrollmentOnlyKey).(bool); v {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "mfa enrollment required before privileged access")
			return
		}
		if models.RoleRank(UserRoleFromContext(r.Context())) < models.RoleRank(role) {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "insufficient permissions")
			return
		}
		next.ServeHTTP(w, r)
	})
}
