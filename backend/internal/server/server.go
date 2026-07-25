package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/email"
	"skill-arena/internal/handlers"
	"skill-arena/internal/models"
)

type Server struct {
	http.Server
}

func New(store *db.Store, cfg *config.Config) *Server {
	store.ConfigureRuntime(cfg.Settings)
	router := http.NewServeMux()
	financial := handlers.NewFinancialHandlers(store, cfg.Settings)
	crm := handlers.NewAdminCRMHandlers(store, cfg.Settings)
	router.Handle("/health", healthHandler(store, cfg, financial))
	router.HandleFunc("/health/live", liveHandler)
	router.Handle("/health/ready", healthHandler(store, cfg, financial))
	router.Handle("/api/v1/config/features", handlers.FeatureFlagsHandler(cfg.Settings))
	router.Handle("/api/v1/platform/stats", handlers.PlatformStatsHandler(store))
	router.Handle("/api/v1/platform/puzzle-preview", handlers.PlatformPuzzlePreviewHandler())
	router.Handle("/api/v1/catalog/games", handlers.GameCatalogHandler(store))
	router.Handle("/api/v1/catalog/games/", handlers.GameRulesHandler(store))
	router.Handle("/api/v1/support/content", handlers.SupportContentHandler(cfg.Settings.Platform.SupportEmail))
	router.HandleFunc("/api/v1/payments/webhooks/", financial.Webhook)
	router.Handle("/api/v1/auth/register", handlers.RegisterHandler(store, cfg))
	router.Handle("/api/v1/auth/login", handlers.LoginHandler(store, cfg))
	router.Handle("/api/v1/auth/mfa/challenge", handlers.MFAChallengeHandler(store, cfg))
	router.Handle("/api/v1/auth/refresh-token", handlers.RefreshTokenHandler(store, cfg))
	router.Handle("/api/v1/auth/logout", handlers.AuthMiddleware(store, cfg, handlers.LogoutHandler(store, cfg)))
	router.Handle("/api/v1/auth/session", handlers.AuthMiddleware(store, cfg, handlers.SessionStatusHandler(store)))
	router.Handle("/api/v1/auth/sessions", handlers.AuthMiddleware(store, cfg, handlers.SessionsHandler(store)))
	router.Handle("/api/v1/auth/sessions/revoke", handlers.AuthMiddleware(store, cfg, handlers.SessionRevokeHandler(store)))
	router.Handle("/api/v1/auth/devices", handlers.AuthMiddleware(store, cfg, handlers.DevicesHandler(store)))
	router.Handle("/api/v1/auth/devices/revoke", handlers.AuthMiddleware(store, cfg, handlers.DeviceRevokeHandler(store)))
	router.Handle("/api/v1/auth/verify-email", handlers.VerifyEmailHandler(store, cfg))
	router.Handle("/api/v1/auth/resend-verification", handlers.ResendVerificationHandler(store, cfg))
	router.Handle("/api/v1/auth/password-reset/request", handlers.PasswordResetRequestHandler(store, cfg))
	router.Handle("/api/v1/auth/password-reset/confirm", handlers.PasswordResetConfirmHandler(store, cfg))
	router.Handle("/api/v1/auth/mfa/setup", handlers.AuthMiddleware(store, cfg, handlers.MFASetupHandler(store, cfg)))
	router.Handle("/api/v1/auth/mfa/confirm", handlers.AuthMiddleware(store, cfg, handlers.MFAConfirmHandler(store, cfg)))
	router.Handle("/api/v1/auth/mfa/disable", handlers.AuthMiddleware(store, cfg, handlers.MFADisableHandler(store, cfg)))
	router.Handle("/api/v1/admin-crm/auth/login", handlers.AdminCRMLoginHandler(store, cfg))
	router.Handle("/api/v1/admin-crm/auth/mfa/challenge", handlers.AdminCRMMFAChallengeHandler(store, cfg))
	router.Handle("/api/v1/admin-crm/auth/refresh", handlers.AdminCRMRefreshHandler(store, cfg))
	router.Handle("/api/v1/admin-crm/auth/session", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.AdminCRMSessionHandler(store)))
	router.Handle("/api/v1/admin-crm/auth/logout", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.AdminCRMLogoutHandler(store, cfg)))
	router.Handle("/api/v1/admin-crm/auth/mfa/setup", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.MFASetupHandler(store, cfg)))
	router.Handle("/api/v1/admin-crm/auth/mfa/confirm", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.MFAConfirmHandler(store, cfg)))
	router.Handle("/api/v1/admin-crm/dashboard", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionDashboardRead, http.HandlerFunc(crm.Dashboard))))
	router.Handle("/api/v1/admin-crm/users", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionUsersRead, http.HandlerFunc(crm.Users))))
	router.Handle("/api/v1/admin-crm/users/", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionUsersRead, http.HandlerFunc(crm.User))))
	router.Handle("/api/v1/admin-crm/finance", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionFinanceRead, http.HandlerFunc(crm.Finance))))
	router.Handle("/api/v1/admin-crm/finance/withdrawals/", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionWithdrawalsReview, http.HandlerFunc(crm.Withdrawal))))
	router.Handle("/api/v1/admin-crm/compliance/cases", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionKYCRead, http.HandlerFunc(crm.ComplianceCases))))
	router.Handle("/api/v1/admin-crm/compliance/evidence/", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionKYCRead, http.HandlerFunc(crm.ComplianceEvidence))))
	router.Handle("/api/v1/admin-crm/compliance/decisions", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionKYCDecide, http.HandlerFunc(crm.ComplianceDecision))))
	router.Handle("/api/v1/admin-crm/compliance/jurisdictions", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionComplianceRead, http.HandlerFunc(crm.Jurisdictions))))
	router.Handle("/api/v1/admin-crm/support/tickets", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionSupportRead, http.HandlerFunc(crm.Support))))
	router.Handle("/api/v1/admin-crm/support/tickets/", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionSupportManage, http.HandlerFunc(crm.SupportTicket))))
	router.Handle("/api/v1/admin-crm/support/attachments/", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionSupportRead, http.HandlerFunc(crm.SupportAttachment))))
	router.Handle("/api/v1/admin-crm/audit", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionAuditRead, http.HandlerFunc(crm.Audit))))
	router.Handle("/api/v1/admin-crm/announcements", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionNotificationsSend, http.HandlerFunc(crm.Announcements))))
	router.Handle("/api/v1/admin-crm/monitoring", handlers.AdminCRMAuthMiddleware(store, cfg, handlers.RequirePermission(models.PermissionMonitoringRead, http.HandlerFunc(crm.Monitoring))))
	router.Handle("/api/v1/identity/kyc-submit", handlers.AuthMiddleware(store, cfg, handlers.KYCSubmitHandler(store)))
	router.Handle("/api/v1/identity/kyc-status", handlers.AuthMiddleware(store, cfg, handlers.KYCStatusHandler(store)))
	router.Handle("/api/v1/leaderboard", handlers.LeaderboardHandler(store))
	router.Handle("/api/v1/profile", handlers.AuthMiddleware(store, cfg, handlers.ProfileHandler(store)))
	router.Handle("/api/v1/progression", handlers.AuthMiddleware(store, cfg, handlers.ProgressionHandler(store)))
	router.Handle("/api/v1/hub", handlers.AuthMiddleware(store, cfg, handlers.HubHandler(store)))
	router.Handle("/api/v1/notifications", handlers.AuthMiddleware(store, cfg, handlers.NotificationsHandler(store)))
	router.Handle("/api/v1/notifications/read", handlers.AuthMiddleware(store, cfg, handlers.NotificationStatusHandler(store, "read")))
	router.Handle("/api/v1/notifications/archive", handlers.AuthMiddleware(store, cfg, handlers.NotificationStatusHandler(store, "archived")))
	router.Handle("/api/v1/support/tickets", handlers.AuthMiddleware(store, cfg, handlers.SupportTicketsHandler(store)))
	router.Handle("/api/v1/support/attachments", handlers.AuthMiddleware(store, cfg, handlers.SupportAttachmentsHandler(store)))
	router.Handle("/api/v1/support/attachments/", handlers.AuthMiddleware(store, cfg, handlers.SupportAttachmentsHandler(store)))
	router.Handle("/api/v1/achievements", handlers.AuthMiddleware(store, cfg, handlers.AchievementsHandler(store)))
	router.Handle("/api/v1/achievements/catalog", handlers.AuthMiddleware(store, cfg, handlers.AchievementCatalogHandler()))
	router.Handle("/api/v1/seasons/current", handlers.AuthMiddleware(store, cfg, handlers.ActiveSeasonHandler(store)))
	router.Handle("/api/v1/seasons/leaderboard", handlers.AuthMiddleware(store, cfg, handlers.SeasonLeaderboardHandler(store)))
	router.Handle("/api/v1/tournaments", handlers.AuthMiddleware(store, cfg, handlers.TournamentListHandler(store)))
	router.Handle("/api/v1/tournaments/register", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.MaintenanceMiddleware(cfg, handlers.TournamentRegisterHandler(store)))))
	router.Handle("/api/v1/tournaments/submit-match", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.TournamentSubmitMatchHandler(store))))
	router.Handle("/api/v1/tournaments/", handlers.AuthMiddleware(store, cfg, handlers.TournamentDetailHandler(store)))
	router.Handle("/api/v1/pvp/join", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.MaintenanceMiddleware(cfg, handlers.PvPJoinHandler(store)))))
	router.Handle("/api/v1/pvp/progress", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.PvPProgressHandler(store))))
	router.Handle("/api/v1/pvp/submit", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.PvPSubmitHandler(store))))
	router.Handle("/api/v1/pvp/matches", handlers.AuthMiddleware(store, cfg, handlers.PvPMatchesHandler(store)))
	router.Handle("/api/v1/pvp/matches/", handlers.AuthMiddleware(store, cfg, handlers.PvPMatchDetailHandler(store)))
	router.Handle("/api/v1/wallet", handlers.AuthMiddleware(store, cfg, handlers.WalletHandler(store)))
	router.Handle("/api/v1/wallet/transactions", handlers.AuthMiddleware(store, cfg, handlers.WalletTransactionsHandler(store)))
	router.Handle("/api/v1/wallet/balance", handlers.AuthMiddleware(store, cfg, handlers.WalletBalanceHandler(store)))
	router.Handle("/api/v1/wallet/available", handlers.AuthMiddleware(store, cfg, handlers.WalletAvailableHandler(store)))
	router.Handle("/api/v1/financial/overview", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Overview)))
	router.Handle("/api/v1/financial/transactions", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Transactions)))
	router.Handle("/api/v1/financial/transactions/export", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.TransactionExport)))
	router.Handle("/api/v1/financial/statements", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Statement)))
	router.Handle("/api/v1/financial/statements/export", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.StatementExport)))
	router.Handle("/api/v1/financial/artifacts/", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Artifact)))
	router.Handle("/api/v1/financial/evidence", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Evidence)))
	router.Handle("/api/v1/financial/assessment", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Assessment)))
	router.Handle("/api/v1/financial/limits", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Limits)))
	router.Handle("/api/v1/financial/deposits", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Deposit)))
	router.Handle("/api/v1/financial/withdrawals", handlers.AuthMiddleware(store, cfg, http.HandlerFunc(financial.Withdrawal)))
	router.Handle("/api/v1/treasury/status", handlers.AuthMiddleware(store, cfg, handlers.PublicTreasuryStatusHandler(store)))
	router.Handle("/api/v1/devices/fingerprint", handlers.AuthMiddleware(store, cfg, handlers.RegisterDeviceHandler(store)))
	router.Handle("/api/v1/calibration/start", handlers.AuthMiddleware(store, cfg, handlers.StartCalibrationHandler(store)))
	router.Handle("/api/v1/calibration/baseline", handlers.AuthMiddleware(store, cfg, handlers.BaselineHandler(store)))
	router.Handle("/api/v1/games/start", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.MaintenanceMiddleware(cfg, handlers.StartGameHandler(store)))))
	router.Handle("/api/v1/games/finish", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.FinishGameHandler(store))))
	router.Handle("/api/v1/games/history", handlers.AuthMiddleware(store, cfg, handlers.GameHistoryHandler(store)))
	router.Handle("/api/v1/games/", handlers.AuthMiddleware(store, cfg, handlers.GetGameSessionHandler(store)))
	router.Handle("/api/v1/house/tiers", handlers.AuthMiddleware(store, cfg, handlers.HouseTiersHandler(store)))
	router.Handle("/api/v1/house/start", handlers.AuthMiddleware(store, cfg, handlers.ComplianceRestrictionMiddleware(store, []string{"account", "competition", "cooling_off", "self_exclusion"}, handlers.MaintenanceMiddleware(cfg, handlers.StartHouseChallengeHandler(store)))))
	router.Handle("/api/v1/replays", handlers.AuthMiddleware(store, cfg, handlers.ReplayListHandler(store)))
	router.Handle("/api/v1/replays/", handlers.AuthMiddleware(store, cfg, handlers.ReplayDetailHandler(store)))

	return &Server{
		Server: http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           securityHeadersMiddleware(corsMiddleware(cfg, csrfMiddleware(cfg, requestSizeMiddleware(handlers.RateLimitMiddlewareWithStore(store, router))))),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       2 * time.Minute,
			MaxHeaderBytes:    1 << 20,
		},
	}
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func healthHandler(store *db.Store, cfg *config.Config, financial *handlers.FinancialHandlers) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		checks := map[string]string{"api": "ready"}
		ready := true
		if err := store.AuthHealth(ctx); err != nil {
			checks["identity"] = err.Error()
			ready = false
		} else {
			checks["identity"] = "ready"
		}
		sender := email.NewSender(cfg.Settings.Email, store.DataDir())
		if err := sender.Health(ctx); err != nil {
			checks["email"] = err.Error()
			ready = false
		} else {
			checks["email"] = "ready"
		}
		if err := store.ObjectStorageHealth(ctx); err != nil {
			checks["object_storage"] = err.Error()
			ready = false
		} else {
			checks["object_storage"] = "ready"
		}
		if err := financial.ProviderHealth(ctx); err != nil {
			checks["payment_provider"] = err.Error()
			ready = false
		} else {
			checks["payment_provider"] = "ready"
		}
		status := http.StatusOK
		state := "ready"
		if !ready {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": state, "checks": checks})
	})
}

func corsMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !originAllowed(cfg, origin) {
				http.Error(w, "origin is not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Device-Fingerprint, X-Device-Name, X-Device-OS, X-Device-Browser")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func csrfMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.Header.Get("Authorization") != "" {
			next.ServeHTTP(w, r)
			return
		}
		_, accessErr := r.Cookie(cfg.Settings.Security.AccessCookieName)
		_, refreshErr := r.Cookie(cfg.Settings.Security.RefreshCookieName)
		if accessErr != nil && refreshErr != nil {
			next.ServeHTTP(w, r)
			return
		}
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" || !originAllowed(cfg, origin) {
			handlers.WriteAPIError(w, http.StatusForbidden, handlers.ErrForbidden, "request origin could not be verified")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestSizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			limit := int64(1 << 20)
			if r.URL.Path == "/api/v1/financial/evidence" {
				limit = 11 << 20
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func originAllowed(cfg *config.Config, origin string) bool {
	if cfg == nil || cfg.Settings == nil {
		return false
	}
	for _, allowed := range cfg.Settings.CORS.AllowedOrigins {
		if strings.EqualFold(strings.TrimSpace(allowed), strings.TrimSpace(origin)) {
			return true
		}
	}
	return false
}
