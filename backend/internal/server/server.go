package server

import (
	"net/http"
	"strings"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/handlers"
)

type Server struct {
	http.Server
}

func New(store *db.Store, cfg *config.Config) *Server {
	router := http.NewServeMux()
	router.HandleFunc("/health", healthHandler)
	router.Handle("/api/v1/config/features", handlers.FeatureFlagsHandler(cfg.Settings))
	router.Handle("/api/v1/platform/stats", handlers.PlatformStatsHandler(store))
	router.Handle("/api/v1/platform/puzzle-preview", handlers.PlatformPuzzlePreviewHandler())
	router.Handle("/api/v1/auth/register", handlers.RegisterHandler(store))
	router.Handle("/api/v1/auth/login", handlers.LoginHandler(store, cfg))
	router.Handle("/api/v1/auth/refresh-token", handlers.RefreshTokenHandler(store, cfg))
	router.Handle("/api/v1/auth/logout", handlers.AuthMiddleware(cfg, handlers.LogoutHandler(store)))
	router.Handle("/api/v1/auth/verify-email", handlers.VerifyEmailHandler(store))
	router.Handle("/api/v1/auth/resend-verification", handlers.ResendVerificationHandler(store, cfg))
	router.Handle("/api/v1/auth/password-reset/request", handlers.PasswordResetRequestHandler(store, cfg))
	router.Handle("/api/v1/auth/password-reset/confirm", handlers.PasswordResetConfirmHandler(store))
	router.Handle("/api/v1/auth/mfa/setup", handlers.AuthMiddleware(cfg, handlers.MFASetupHandler(store, cfg)))
	router.Handle("/api/v1/auth/mfa/confirm", handlers.AuthMiddleware(cfg, handlers.MFAConfirmHandler(store, cfg)))
	router.Handle("/api/v1/auth/mfa/disable", handlers.AuthMiddleware(cfg, handlers.MFADisableHandler(store)))
	router.Handle("/api/v1/identity/kyc-submit", handlers.AuthMiddleware(cfg, handlers.KYCSubmitHandler(store)))
	router.Handle("/api/v1/identity/kyc-status", handlers.AuthMiddleware(cfg, handlers.KYCStatusHandler(store)))
	router.Handle("/api/v1/leaderboard", handlers.LeaderboardHandler(store))
	router.Handle("/api/v1/profile", handlers.AuthMiddleware(cfg, handlers.ProfileHandler(store)))
	router.Handle("/api/v1/progression", handlers.AuthMiddleware(cfg, handlers.ProgressionHandler(store)))
	router.Handle("/api/v1/achievements", handlers.AuthMiddleware(cfg, handlers.AchievementsHandler(store)))
	router.Handle("/api/v1/achievements/catalog", handlers.AuthMiddleware(cfg, handlers.AchievementCatalogHandler()))
	router.Handle("/api/v1/seasons/current", handlers.AuthMiddleware(cfg, handlers.ActiveSeasonHandler(store)))
	router.Handle("/api/v1/seasons/leaderboard", handlers.AuthMiddleware(cfg, handlers.SeasonLeaderboardHandler(store)))
	router.Handle("/api/v1/tournaments", handlers.AuthMiddleware(cfg, handlers.TournamentListHandler(store)))
	router.Handle("/api/v1/tournaments/register", handlers.AuthMiddleware(cfg, handlers.MaintenanceMiddleware(cfg, handlers.TournamentRegisterHandler(store))))
	router.Handle("/api/v1/tournaments/submit-match", handlers.AuthMiddleware(cfg, handlers.TournamentSubmitMatchHandler(store)))
	router.Handle("/api/v1/tournaments/", handlers.AuthMiddleware(cfg, handlers.TournamentDetailHandler(store)))
	router.Handle("/api/v1/pvp/join", handlers.AuthMiddleware(cfg, handlers.MaintenanceMiddleware(cfg, handlers.PvPJoinHandler(store))))
	router.Handle("/api/v1/pvp/progress", handlers.AuthMiddleware(cfg, handlers.PvPProgressHandler(store)))
	router.Handle("/api/v1/pvp/submit", handlers.AuthMiddleware(cfg, handlers.PvPSubmitHandler(store)))
	router.Handle("/api/v1/pvp/matches", handlers.AuthMiddleware(cfg, handlers.PvPMatchesHandler(store)))
	router.Handle("/api/v1/pvp/matches/", handlers.AuthMiddleware(cfg, handlers.PvPMatchDetailHandler(store)))
	router.Handle("/api/v1/wallet", handlers.AuthMiddleware(cfg, handlers.WalletHandler(store)))
	router.Handle("/api/v1/wallet/transactions", handlers.AuthMiddleware(cfg, handlers.WalletTransactionsHandler(store)))
	router.Handle("/api/v1/wallet/balance", handlers.AuthMiddleware(cfg, handlers.WalletBalanceHandler(store)))
	router.Handle("/api/v1/wallet/available", handlers.AuthMiddleware(cfg, handlers.WalletAvailableHandler(store)))
	router.Handle("/api/v1/wallet/deposit", handlers.AuthMiddleware(cfg, handlers.WalletDepositHandler(store)))
	router.Handle("/api/v1/wallet/withdraw", handlers.AuthMiddleware(cfg, handlers.WalletWithdrawHandler(store)))
	router.Handle("/api/v1/wallet/lock-tokens", handlers.AuthMiddleware(cfg, handlers.WalletLockHandler(store)))
	router.Handle("/api/v1/wallet/unlock-tokens", handlers.AuthMiddleware(cfg, handlers.WalletUnlockHandler(store)))
	router.Handle("/api/v1/treasury/status", handlers.AuthMiddleware(cfg, handlers.PublicTreasuryStatusHandler(store)))
	router.Handle("/api/v1/devices/fingerprint", handlers.AuthMiddleware(cfg, handlers.RegisterDeviceHandler(store)))
	router.Handle("/api/v1/calibration/start", handlers.AuthMiddleware(cfg, handlers.StartCalibrationHandler(store)))
	router.Handle("/api/v1/calibration/baseline", handlers.AuthMiddleware(cfg, handlers.BaselineHandler(store)))
	router.Handle("/api/v1/games/start", handlers.AuthMiddleware(cfg, handlers.MaintenanceMiddleware(cfg, handlers.StartGameHandler(store))))
	router.Handle("/api/v1/games/finish", handlers.AuthMiddleware(cfg, handlers.FinishGameHandler(store)))
	router.Handle("/api/v1/games/history", handlers.AuthMiddleware(cfg, handlers.GameHistoryHandler(store)))
	router.Handle("/api/v1/games/", handlers.AuthMiddleware(cfg, handlers.GetGameSessionHandler(store)))
	router.Handle("/api/v1/house/tiers", handlers.AuthMiddleware(cfg, handlers.HouseTiersHandler(store)))
	router.Handle("/api/v1/house/start", handlers.AuthMiddleware(cfg, handlers.MaintenanceMiddleware(cfg, handlers.StartHouseChallengeHandler(store))))
	router.Handle("/api/v1/replays", handlers.AuthMiddleware(cfg, handlers.ReplayListHandler(store)))
	router.Handle("/api/v1/replays/", handlers.AuthMiddleware(cfg, handlers.ReplayDetailHandler(store)))
	router.Handle("/api/v1/admin/users", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminUsersHandler(store))))
	router.Handle("/api/v1/admin/roles/update", handlers.AuthMiddleware(cfg, handlers.RequireRole("super_admin", handlers.AdminUpdateRoleHandler(store))))
	router.Handle("/api/v1/admin/roles/suspend", handlers.AuthMiddleware(cfg, handlers.RequireRole("super_admin", handlers.AdminSuspendHandler(store))))
	router.Handle("/api/v1/admin/mfa/reset", handlers.AuthMiddleware(cfg, handlers.RequireRole("super_admin", handlers.AdminResetMFAHandler(store))))
	router.Handle("/api/v1/admin/audit-logs", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminAuditLogsHandler(store))))
	router.Handle("/api/v1/admin/kyc/approve", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminApproveKYCHandler(store))))
	router.Handle("/api/v1/admin/review-cases", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminReviewCasesHandler(store))))
	router.Handle("/api/v1/admin/review-cases/transition", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminReviewTransitionHandler(store))))
	router.Handle("/api/v1/admin/metrics", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminMetricsHandler(store))))
	router.Handle("/api/v1/admin/system-health", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminSystemHealthHandler(store))))
	router.Handle("/api/v1/admin/jobs", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminJobsHandler(store))))
	router.Handle("/api/v1/admin/jobs/stats", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminJobStatsHandler(store))))
	router.Handle("/api/v1/admin/jobs/retry", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminJobActionHandler(store, "retry"))))
	router.Handle("/api/v1/admin/jobs/cancel", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminJobActionHandler(store, "cancel"))))
	router.Handle("/api/v1/admin/jobs/requeue", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminJobActionHandler(store, "requeue"))))
	router.Handle("/api/v1/admin/backups", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminBackupsHandler(store))))
	router.Handle("/api/v1/admin/backups/restore", handlers.AuthMiddleware(cfg, handlers.RequireRole("super_admin", handlers.AdminBackupRestoreHandler(store))))
	router.Handle("/api/v1/admin/replays/", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminReplayDetailHandler(store))))
	router.Handle("/api/v1/admin/treasury/health", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.TreasuryHealthHandler(store))))
	router.Handle("/api/v1/admin/treasury/withdrawals/approve", handlers.AuthMiddleware(cfg, handlers.RequireRole("treasury_manager", handlers.TreasuryApproveWithdrawalHandler(store))))
	router.Handle("/api/v1/admin/treasury/withdrawals/reject", handlers.AuthMiddleware(cfg, handlers.RequireRole("treasury_manager", handlers.TreasuryRejectWithdrawalHandler(store))))
	router.Handle("/api/v1/admin/treasury/withdrawals/settle", handlers.AuthMiddleware(cfg, handlers.RequireRole("treasury_manager", handlers.TreasurySettleWithdrawalHandler(store))))
	router.Handle("/api/v1/admin/house-risk/", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.HouseRiskHandler(store))))
	router.Handle("/api/v1/admin/baselines", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminBaselinesHandler(store))))
	router.Handle("/api/v1/admin/tournaments/bracket", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminGenerateTournamentBracketHandler(store))))
	router.Handle("/api/v1/admin/tournaments/result", handlers.AuthMiddleware(cfg, handlers.RequireRole("admin", handlers.AdminReportTournamentResultHandler(store))))

	return &Server{
		Server: http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: corsMiddleware(cfg, handlers.RateLimitMiddlewareWithStore(store, router)),
		},
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func corsMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && originAllowed(cfg, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
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
