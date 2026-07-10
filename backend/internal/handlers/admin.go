package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"skill-arena/internal/db"
	"skill-arena/internal/workers"
)

type approveKYCRequest struct {
	UserID string `json:"userId"`
}

type reviewTransitionRequest struct {
	CaseID   string `json:"caseId"`
	Status   string `json:"status"`
	Decision string `json:"decision,omitempty"`
}

type adminRoleRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role,omitempty"`
}

type enqueueJobRequest struct {
	Type    string            `json:"type"`
	Payload map[string]string `json:"payload,omitempty"`
}

type jobActionRequest struct {
	JobID string `json:"jobId"`
}

type restoreBackupRequest struct {
	Path string `json:"path"`
}

func AdminUsersHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		users, err := store.ListUsers(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list users: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

func AdminAuditLogsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		logs, err := store.GetAuditLogs(r.Context(), 100)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load audit logs: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
	}
}

func AdminApproveKYCHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req approveKYCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.UserID) == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		actorID := UserIDFromContext(r.Context())
		if err := store.ApproveKYC(r.Context(), actorID, req.UserID, clientIP(r)); err != nil {
			http.Error(w, fmt.Sprintf("failed to approve KYC: %v", err), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func AdminReviewCasesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		cases, err := store.ListReviewCases(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load review cases: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cases)
	}
}

func AdminReviewTransitionHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req reviewTransitionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.CaseID == "" || req.Status == "" {
			http.Error(w, "caseId and status are required", http.StatusBadRequest)
			return
		}
		reviewCase, err := store.TransitionReviewCase(r.Context(), UserIDFromContext(r.Context()), req.CaseID, req.Status, req.Decision, clientIP(r))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to transition review case: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reviewCase)
	}
}

func AdminMetricsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		metrics, err := store.Metrics(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load metrics: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}

func AdminUpdateRoleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req adminRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.UserID == "" || req.Role == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "userId and role are required")
			return
		}
		user, err := store.UpdateUserRole(r.Context(), UserIDFromContext(r.Context()), req.UserID, req.Role, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func AdminSuspendHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req adminRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.UserID == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "userId is required")
			return
		}
		user, err := store.SuspendAdmin(r.Context(), UserIDFromContext(r.Context()), req.UserID, clientIP(r))
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func AdminResetMFAHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req adminRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.UserID == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "userId is required")
			return
		}
		if err := store.ResetAdminMFA(r.Context(), UserIDFromContext(r.Context()), req.UserID, clientIP(r)); err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func AdminSystemHealthHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		health, err := store.SystemHealth(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}
}

func AdminJobsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			jobs, err := store.ListJobs(r.Context(), r.URL.Query().Get("status"))
			if err != nil {
				WriteMappedError(w, http.StatusInternalServerError, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(jobs)
		case http.MethodPost:
			var req enqueueJobRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
				return
			}
			job, err := store.EnqueueJob(r.Context(), req.Type, req.Payload, time.Time{})
			if err != nil {
				WriteMappedError(w, http.StatusBadRequest, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(job)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func AdminJobActionHandler(store *db.Store, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req jobActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.JobID == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "jobId is required")
			return
		}
		var (
			job any
			err error
		)
		switch action {
		case "retry", "requeue":
			job, err = store.RetryJob(r.Context(), req.JobID)
		case "cancel":
			job, err = store.CancelJob(r.Context(), req.JobID)
		default:
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "unsupported job action")
			return
		}
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		_ = store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "admin.job."+action, req.JobID, clientIP(r), nil)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	}
}

func AdminJobStatsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		stats, err := store.QueueStats(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func AdminBackupsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			backups, err := store.ListBackupRecords(r.Context())
			if err != nil {
				WriteMappedError(w, http.StatusInternalServerError, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(backups)
		case http.MethodPost:
			job, err := store.EnqueueJob(r.Context(), "backup_run", map[string]string{"trigger": "manual", "actorId": UserIDFromContext(r.Context())}, time.Time{})
			if err != nil {
				WriteMappedError(w, http.StatusBadRequest, err)
				return
			}
			_ = store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "admin.backup.requested", job.ID, clientIP(r), nil)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(job)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func AdminBackupRestoreHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req restoreBackupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request payload")
			return
		}
		if req.Path == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "path is required")
			return
		}
		report, err := workers.ValidateRecovery(r.Context(), req.Path)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		_ = store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), "admin.backup.restore_validated", req.Path, clientIP(r), nil)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}
