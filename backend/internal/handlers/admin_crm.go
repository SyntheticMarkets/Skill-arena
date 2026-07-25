package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/email"
	"skill-arena/internal/models"
	"skill-arena/internal/payments"
)

type AdminCRMHandlers struct {
	store    *db.Store
	settings *config.RuntimeSettings
	payments *payments.Core
	email    email.Sender
}

func NewAdminCRMHandlers(store *db.Store, settings *config.RuntimeSettings) *AdminCRMHandlers {
	return &AdminCRMHandlers{
		store: store, settings: settings, payments: payments.CoreFromSettings(settings.Payments),
		email: email.NewSender(settings.Email, store.DataDir()),
	}
}

func (h *AdminCRMHandlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	result, err := h.store.BuildCRMDashboard(r.Context())
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminCRMHandlers) Users(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	users, total, err := h.store.SearchCRMUsers(r.Context(), r.URL.Query().Get("query"), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users, "total": total, "limit": limit, "offset": offset})
}

func crmPathID(path, prefix string) (string, string) {
	remaining := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(remaining, "/")
	if len(parts) == 0 {
		return "", ""
	}
	action := ""
	if len(parts) > 1 {
		action = strings.Join(parts[1:], "/")
	}
	return parts[0], action
}

func (h *AdminCRMHandlers) User(w http.ResponseWriter, r *http.Request) {
	userID, action := crmPathID(r.URL.Path, "/api/v1/admin-crm/users/")
	if userID == "" {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "user id is required")
		return
	}
	if action == "" && r.Method == http.MethodGet {
		record, err := h.store.GetCRMUserRecord(r.Context(), userID)
		if err != nil {
			WriteMappedError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, record)
		return
	}
	switch action {
	case "status":
		h.userStatus(w, r, userID)
	case "force-logout":
		h.forceLogout(w, r, userID)
	case "notes":
		h.userNotes(w, r, userID)
	case "restrictions":
		h.userRestrictions(w, r, userID)
	case "role":
		h.userRole(w, r, userID)
	case "mfa/reset":
		h.resetMFA(w, r, userID)
	default:
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "administrator user operation was not found")
	}
}

type crmReasonRequest struct {
	Status        string     `json:"status,omitempty"`
	Reason        string     `json:"reason"`
	Body          string     `json:"body,omitempty"`
	Type          string     `json:"type,omitempty"`
	Role          string     `json:"role,omitempty"`
	RestrictionID string     `json:"restrictionId,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}

func (h *AdminCRMHandlers) userStatus(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost || !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionUsersManage) {
		WriteAPIError(w, http.StatusForbidden, ErrForbidden, "user management permission is required")
		return
	}
	var request crmReasonRequest
	if decodeJSON(r, &request) != nil {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "status and reason are required")
		return
	}
	user, err := h.store.SetCRMUserStatus(r.Context(), UserIDFromContext(r.Context()), userID, request.Status, request.Reason, clientIP(r), r.UserAgent())
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AdminCRMHandlers) forceLogout(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost || !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionUsersManage) {
		WriteAPIError(w, http.StatusForbidden, ErrForbidden, "user management permission is required")
		return
	}
	var request crmReasonRequest
	if decodeJSON(r, &request) != nil || strings.TrimSpace(request.Reason) == "" {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "reason is required")
		return
	}
	if err := h.store.ForceLogoutCRMUser(r.Context(), UserIDFromContext(r.Context()), userID, request.Reason, clientIP(r), r.UserAgent()); err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminCRMHandlers) userNotes(w http.ResponseWriter, r *http.Request, userID string) {
	switch r.Method {
	case http.MethodGet:
		notes, err := h.store.ListCRMInternalNotes(r.Context(), userID)
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
	case http.MethodPost:
		if !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionUsersManage) &&
			!models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionSupportManage) {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "user or support management permission is required")
			return
		}
		var request crmReasonRequest
		if decodeJSON(r, &request) != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "note body is required")
			return
		}
		note, err := h.store.AddCRMInternalNote(r.Context(), UserIDFromContext(r.Context()), userID, request.Body)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if !h.requireAudit(w, r, "admin.user.note.created", note.ID, request.Body, "", "") {
			return
		}
		writeJSON(w, http.StatusCreated, note)
	default:
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
	}
}

func (h *AdminCRMHandlers) userRestrictions(w http.ResponseWriter, r *http.Request, userID string) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.store.ListCRMRestrictions(r.Context(), userID, r.URL.Query().Get("status"))
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"restrictions": items})
	case http.MethodPost:
		if !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionComplianceManage) {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "compliance management permission is required")
			return
		}
		var request crmReasonRequest
		if decodeJSON(r, &request) != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "restriction type and reason are required")
			return
		}
		item, err := h.store.CreateCRMRestriction(r.Context(), models.CRMRestriction{
			UserID: userID, Type: request.Type, Reason: request.Reason, ExpiresAt: request.ExpiresAt,
			CreatedBy: UserIDFromContext(r.Context()),
		})
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if !h.requireAudit(w, r, "admin.compliance.restriction.created", item.ID, request.Reason, "", item.Status) {
			return
		}
		writeJSON(w, http.StatusCreated, item)
	case http.MethodPatch:
		if !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionComplianceManage) {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "compliance management permission is required")
			return
		}
		var request crmReasonRequest
		if decodeJSON(r, &request) != nil || request.RestrictionID == "" || strings.TrimSpace(request.Reason) == "" {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "restriction id and reason are required")
			return
		}
		item, err := h.store.UpdateCRMRestrictionStatus(r.Context(), userID, request.RestrictionID, "lifted")
		if err != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "active restriction was not found for this user")
			return
		}
		if !h.requireAudit(w, r, "admin.compliance.restriction.lifted", item.ID, request.Reason, "active", "lifted") {
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
	}
}

func (h *AdminCRMHandlers) userRole(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost || !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionAdminRolesManage) {
		WriteAPIError(w, http.StatusForbidden, ErrForbidden, "administrator role management permission is required")
		return
	}
	var request crmReasonRequest
	if decodeJSON(r, &request) != nil || request.Role == "" || strings.TrimSpace(request.Reason) == "" {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "role and reason are required")
		return
	}
	user, err := h.store.UpdateUserRole(r.Context(), UserIDFromContext(r.Context()), userID, request.Role, clientIP(r))
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	if !h.requireAudit(w, r, "admin.user.role.changed", userID, request.Reason, "", request.Role) {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AdminCRMHandlers) resetMFA(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost || !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionAdminRolesManage) {
		WriteAPIError(w, http.StatusForbidden, ErrForbidden, "administrator role management permission is required")
		return
	}
	var request crmReasonRequest
	if decodeJSON(r, &request) != nil || strings.TrimSpace(request.Reason) == "" {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "reason is required")
		return
	}
	if err := h.store.ResetAdminMFA(r.Context(), UserIDFromContext(r.Context()), userID, clientIP(r)); err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.store.ForceLogoutCRMUser(r.Context(), UserIDFromContext(r.Context()), userID, request.Reason, clientIP(r), r.UserAgent()); err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminCRMHandlers) Finance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	result, err := h.store.CRMFinanceWorkspace(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	for _, provider := range h.payments.ProviderNames() {
		status := models.CRMProviderStatus{ID: provider, Status: "healthy"}
		if err := h.providerHealth(r.Context(), provider); err != nil {
			status.Status = "unavailable"
			status.Details = err.Error()
		}
		result.Providers = append(result.Providers, status)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminCRMHandlers) providerHealth(ctx context.Context, provider string) error {
	for _, country := range h.settings.Financial.Jurisdictions {
		if _, err := h.payments.Balance(ctx, provider, country.Currency); err == nil {
			return nil
		}
	}
	return h.payments.Health(ctx)
}

type crmWithdrawalDecisionRequest struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func (h *AdminCRMHandlers) Withdrawal(w http.ResponseWriter, r *http.Request) {
	withdrawalID, action := crmPathID(r.URL.Path, "/api/v1/admin-crm/finance/withdrawals/")
	if r.Method != http.MethodPost || withdrawalID == "" || action != "decision" {
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "withdrawal operation was not found")
		return
	}
	var request crmWithdrawalDecisionRequest
	if decodeJSON(r, &request) != nil || strings.TrimSpace(request.Reason) == "" ||
		(request.Decision != "approve" && request.Decision != "reject") {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "approve or reject decision and reason are required")
		return
	}
	targetStatus := models.FinancialWithdrawalStatusApproved
	if request.Decision == "reject" {
		targetStatus = models.FinancialWithdrawalStatusRejected
	}
	result, err := h.store.TransitionFinancialWithdrawal(r.Context(), withdrawalID, targetStatus, "administrator", UserIDFromContext(r.Context()), request.Reason, "")
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	if !h.requireAudit(w, r, "admin.withdrawal."+request.Decision, withdrawalID, request.Reason, models.FinancialWithdrawalStatusPendingReview, targetStatus) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminCRMHandlers) ComplianceCases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	items, err := h.store.ListCRMComplianceCases(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cases": items})
}

func (h *AdminCRMHandlers) ComplianceEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	evidenceID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin-crm/compliance/evidence/"), "/")
	item, data, err := h.store.GetFinancialEvidenceForReview(r.Context(), evidenceID)
	if err != nil {
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "compliance evidence was not found")
		return
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="evidence-`+item.ID+`"`)
	w.Header().Set("X-Content-SHA256", item.SHA256)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

type crmComplianceDecisionRequest struct {
	UserID             string `json:"userId"`
	Decision           string `json:"decision"`
	RiskClassification string `json:"riskClassification"`
	VerificationStatus string `json:"verificationStatus"`
	Reason             string `json:"reason"`
}

func (h *AdminCRMHandlers) ComplianceDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	var request crmComplianceDecisionRequest
	if decodeJSON(r, &request) != nil || request.UserID == "" || strings.TrimSpace(request.Reason) == "" {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "user, decision, risk, verification, and reason are required")
		return
	}
	allowed := map[string]bool{"complete": true, "restricted": true, "in_review": true}
	if !allowed[request.Decision] {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "compliance decision is invalid")
		return
	}
	result, err := h.store.ReviewFinancialAssessment(r.Context(), request.UserID, request.Decision, request.RiskClassification, request.VerificationStatus)
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	kycStatus := "pending"
	evidenceStatus := "received"
	message := "Your identity and financial assessment remains under review."
	switch request.Decision {
	case models.AssessmentStatusComplete:
		kycStatus, evidenceStatus = "approved", "verified"
		message = "Your identity and financial assessment has been approved."
	case models.AssessmentStatusRestricted:
		kycStatus, evidenceStatus = "rejected", "rejected"
		message = "Your identity assessment could not be approved. Live financial activity remains unavailable."
	case models.AssessmentStatusInReview:
		if request.VerificationStatus == "more_information" {
			kycStatus, evidenceStatus = "more_information", "more_information"
			message = "More information is required to complete your identity assessment."
		}
	}
	if err := h.store.SetUserKYCStatus(r.Context(), request.UserID, kycStatus); err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.store.SetFinancialEvidenceStatus(r.Context(), request.UserID, evidenceStatus); err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.store.CreateNotification(r.Context(), &models.Notification{
		UserID: request.UserID, Category: "compliance", Title: "Identity assessment updated",
		Message: message, ActionURL: "/wallet/verification",
	}); err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	if !h.requireAudit(w, r, "admin.compliance.assessment.decided", request.UserID, request.Reason, "in_review", request.Decision) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminCRMHandlers) Jurisdictions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.store.ListCRMJurisdictions(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jurisdictions": items})
	case http.MethodPut:
		if !models.HasAdminPermission(UserRoleFromContext(r.Context()), models.PermissionComplianceManage) {
			WriteAPIError(w, http.StatusForbidden, ErrForbidden, "compliance management permission is required")
			return
		}
		var policy models.CRMJurisdictionPolicy
		if decodeJSON(r, &policy) != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "jurisdiction policy is invalid")
			return
		}
		policy.UpdatedBy = UserIDFromContext(r.Context())
		result, err := h.store.SaveCRMJurisdiction(r.Context(), policy)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if !h.requireAudit(w, r, "admin.compliance.jurisdiction.updated", policy.Country, "policy update", "", policy.Currency) {
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
	}
}

func (h *AdminCRMHandlers) Support(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	items, err := h.store.ListCRMSupportTickets(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tickets": items})
}

type crmSupportUpdateRequest struct {
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	AssignedTo string `json:"assignedTo,omitempty"`
	Escalated  bool   `json:"escalated"`
	Reply      string `json:"reply,omitempty"`
	Internal   bool   `json:"internal"`
}

func (h *AdminCRMHandlers) SupportTicket(w http.ResponseWriter, r *http.Request) {
	ticketID, _ := crmPathID(r.URL.Path, "/api/v1/admin-crm/support/tickets/")
	if r.Method != http.MethodPatch || ticketID == "" {
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "support operation was not found")
		return
	}
	var request crmSupportUpdateRequest
	if decodeJSON(r, &request) != nil {
		WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "support update is invalid")
		return
	}
	result, err := h.store.UpdateCRMSupportTicket(r.Context(), UserIDFromContext(r.Context()), ticketID, request.Status, request.Priority, request.AssignedTo, request.Escalated, request.Reply, request.Internal)
	if err != nil {
		WriteMappedError(w, http.StatusBadRequest, err)
		return
	}
	if !h.requireAudit(w, r, "admin.support.ticket.updated", ticketID, request.Reply, "", request.Status) {
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AdminCRMHandlers) SupportAttachment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	attachmentID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin-crm/support/attachments/"), "/")
	item, data, err := h.store.GetSupportAttachment(r.Context(), attachmentID)
	if err != nil {
		WriteAPIError(w, http.StatusNotFound, ErrNotFound, "support attachment was not found")
		return
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+item.FileName+`"`)
	w.Header().Set("X-Content-SHA256", item.SHA256)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *AdminCRMHandlers) Audit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := h.store.GetAuditLogs(r.Context(), limit)
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	chainStatus := "verified"
	if err := h.store.VerifyCRMAuditChain(r.Context()); err != nil {
		chainStatus = "invalid"
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs, "chainStatus": chainStatus})
}

func (h *AdminCRMHandlers) Announcements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.store.ListCRMAnnouncements(r.Context())
		if err != nil {
			WriteMappedError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"announcements": items})
	case http.MethodPost:
		var request models.CRMAnnouncement
		if decodeJSON(r, &request) != nil {
			WriteAPIError(w, http.StatusBadRequest, ErrInvalidRequest, "announcement is invalid")
			return
		}
		request.CreatedBy = UserIDFromContext(r.Context())
		result, err := h.store.CreateCRMAnnouncement(r.Context(), request)
		if err != nil {
			WriteMappedError(w, http.StatusBadRequest, err)
			return
		}
		if !h.requireAudit(w, r, "admin.notification.broadcast.sent", result.ID, result.Message, "", result.Audience) {
			return
		}
		writeJSON(w, http.StatusCreated, result)
	default:
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
	}
}

func (h *AdminCRMHandlers) Monitoring(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, ErrInvalidRequest, "method is not allowed")
		return
	}
	system, err := h.store.SystemHealth(r.Context())
	if err != nil {
		WriteMappedError(w, http.StatusInternalServerError, err)
		return
	}
	queue, _ := h.store.QueueStats(r.Context())
	dependencies := map[string]string{}
	alerts := []string{}
	check := func(name string, health func(context.Context) error) {
		dependencies[name] = "healthy"
		if err := health(r.Context()); err != nil {
			dependencies[name] = "unavailable"
			alerts = append(alerts, name+": "+err.Error())
		}
	}
	check("database", h.store.AuthHealth)
	if h.store.Redis() == nil {
		dependencies["redis"] = "unavailable"
		alerts = append(alerts, "redis: client is not configured")
	} else {
		check("redis", h.store.Redis().Health)
	}
	check("storage", h.store.ObjectStorageHealth)
	check("paymentProviders", h.payments.Health)
	check("email", h.email.Health)
	writeJSON(w, http.StatusOK, map[string]any{
		"system": system, "queue": queue, "dependencies": dependencies, "alerts": alerts, "readOnly": true,
	})
}

func (h *AdminCRMHandlers) audit(r *http.Request, action, target, reason, previous, next string) error {
	metadata := map[string]string{
		"reason": reason, "previous": previous, "new": next,
		"device": r.UserAgent(), "resource": target,
	}
	return h.store.AppendAuditLog(r.Context(), UserIDFromContext(r.Context()), action, target, clientIP(r), metadata)
}

func (h *AdminCRMHandlers) requireAudit(w http.ResponseWriter, r *http.Request, action, target, reason, previous, next string) bool {
	if err := h.audit(r, action, target, reason, previous, next); err != nil {
		WriteAPIError(w, http.StatusInternalServerError, ErrInternal, "the operation could not be audit logged")
		return false
	}
	return true
}
