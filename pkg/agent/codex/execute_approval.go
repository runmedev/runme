package codex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultApprovalTTL = 5 * time.Minute

type PendingExecuteApproval struct {
	SessionID string    `json:"session_id"`
	RefIDs    []string  `json:"ref_ids"`
	CreatedAt time.Time `json:"created_at"`
}

type ExecuteApprovalManager struct {
	mu        sync.Mutex
	now       func() time.Time
	ttl       time.Duration
	pending   map[string]map[string]PendingExecuteApproval
	approvals map[string]map[string]time.Time
}

func NewExecuteApprovalManager(ttl time.Duration) *ExecuteApprovalManager {
	if ttl <= 0 {
		ttl = defaultApprovalTTL
	}
	return &ExecuteApprovalManager{
		now:       time.Now,
		ttl:       ttl,
		pending:   make(map[string]map[string]PendingExecuteApproval),
		approvals: make(map[string]map[string]time.Time),
	}
}

func (m *ExecuteApprovalManager) RequireApproval(sessionID string, refIDs []string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session id is required")
	}
	refIDs = normalizeRefIDs(refIDs)
	if len(refIDs) == 0 {
		return errors.New("at least one ref id is required")
	}
	key := refIDKey(refIDs)

	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	m.gcLocked(now)

	sessionApprovals := m.approvals[sessionID]
	if sessionApprovals != nil {
		if expiresAt, ok := sessionApprovals[key]; ok && now.Before(expiresAt) {
			delete(sessionApprovals, key)
			return nil
		}
	}
	sessionPending := m.pending[sessionID]
	if sessionPending == nil {
		sessionPending = make(map[string]PendingExecuteApproval)
		m.pending[sessionID] = sessionPending
	}
	if _, ok := sessionPending[key]; !ok {
		sessionPending[key] = PendingExecuteApproval{
			SessionID: sessionID,
			RefIDs:    append([]string(nil), refIDs...),
			CreatedAt: now,
		}
	}
	return errors.New("missing execute approvals")
}

func (m *ExecuteApprovalManager) Approve(sessionID string, refIDs []string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session id is required")
	}
	refIDs = normalizeRefIDs(refIDs)
	if len(refIDs) == 0 {
		return errors.New("at least one ref id is required")
	}
	key := refIDKey(refIDs)

	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	m.gcLocked(now)

	sessionApprovals := m.approvals[sessionID]
	if sessionApprovals == nil {
		sessionApprovals = make(map[string]time.Time)
		m.approvals[sessionID] = sessionApprovals
	}
	sessionApprovals[key] = now.Add(m.ttl)

	if sessionPending := m.pending[sessionID]; sessionPending != nil {
		delete(sessionPending, key)
	}
	return nil
}

func (m *ExecuteApprovalManager) ListPending(sessionID string) []PendingExecuteApproval {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(m.now())
	sessionPending := m.pending[sessionID]
	if len(sessionPending) == 0 {
		return nil
	}
	out := make([]PendingExecuteApproval, 0, len(sessionPending))
	for _, value := range sessionPending {
		out = append(out, PendingExecuteApproval{
			SessionID: value.SessionID,
			RefIDs:    append([]string(nil), value.RefIDs...),
			CreatedAt: value.CreatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

func (m *ExecuteApprovalManager) gcLocked(now time.Time) {
	for sessionID, sessionApprovals := range m.approvals {
		for key, expiresAt := range sessionApprovals {
			if !now.Before(expiresAt) {
				delete(sessionApprovals, key)
			}
		}
		if len(sessionApprovals) == 0 {
			delete(m.approvals, sessionID)
		}
	}
	for sessionID, sessionPending := range m.pending {
		if len(sessionPending) == 0 {
			delete(m.pending, sessionID)
		}
	}
}

func normalizeRefIDs(refIDs []string) []string {
	out := make([]string, 0, len(refIDs))
	for _, id := range refIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, id)
	}
	return out
}

func refIDKey(refIDs []string) string {
	return strings.Join(refIDs, "\x1f")
}

type executeApprovalApprover struct {
	manager *ExecuteApprovalManager
}

func (a executeApprovalApprover) AllowExecute(ctx context.Context, refIDs []string) error {
	approvedViaHeader := approvedRefIDsFromContext(ctx)
	if len(approvedViaHeader) > 0 {
		headerApprover := contextExecuteApprover{}
		if err := headerApprover.AllowExecute(ctx, refIDs); err == nil {
			return nil
		}
	}
	if a.manager == nil {
		return errors.New("missing execute approval manager")
	}
	return a.manager.RequireApproval(SessionIDFromContext(ctx), refIDs)
}

type ExecuteApprovalHTTPHandler struct {
	manager *ExecuteApprovalManager
}

func NewExecuteApprovalHTTPHandler(manager *ExecuteApprovalManager) *ExecuteApprovalHTTPHandler {
	return &ExecuteApprovalHTTPHandler{manager: manager}
}

func (h *ExecuteApprovalHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id is required", http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{
			"pending": h.manager.ListPending(sessionID),
		})
	case http.MethodPost:
		payload := struct {
			SessionID string   `json:"session_id"`
			RefIDs    []string `json:"ref_ids"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if err := h.manager.Approve(payload.SessionID, payload.RefIDs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
