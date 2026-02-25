package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestExecuteApprovalManager_RequireApproveConsume(t *testing.T) {
	manager := NewExecuteApprovalManager(10 * time.Minute)

	err := manager.RequireApproval("session-1", []string{"cell-1", "cell-2"})
	if err == nil {
		t.Fatalf("RequireApproval should fail before explicit approval")
	}

	pending := manager.ListPending("session-1")
	if len(pending) != 1 {
		t.Fatalf("pending approvals = %d, want 1", len(pending))
	}
	if !reflect.DeepEqual(pending[0].RefIDs, []string{"cell-1", "cell-2"}) {
		t.Fatalf("pending refIDs = %#v, want %#v", pending[0].RefIDs, []string{"cell-1", "cell-2"})
	}

	if err := manager.Approve("session-1", []string{"cell-1", "cell-2"}); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if got := manager.ListPending("session-1"); len(got) != 0 {
		t.Fatalf("pending approvals after approve = %d, want 0", len(got))
	}

	if err := manager.RequireApproval("session-1", []string{"cell-1", "cell-2"}); err != nil {
		t.Fatalf("RequireApproval after approval should pass once: %v", err)
	}
	if err := manager.RequireApproval("session-1", []string{"cell-1", "cell-2"}); err == nil {
		t.Fatalf("RequireApproval should fail after one-time approval is consumed")
	}
}

func TestExecuteApprovalManager_ApprovalExpires(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	manager := NewExecuteApprovalManager(time.Minute)
	manager.now = func() time.Time { return now }

	if err := manager.Approve("session-1", []string{"cell-1"}); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}

	now = now.Add(2 * time.Minute)
	if err := manager.RequireApproval("session-1", []string{"cell-1"}); err == nil {
		t.Fatalf("RequireApproval should fail after approval TTL expires")
	}
}

func TestExecuteApprovalApprover_UsesContextHeaderApprovals(t *testing.T) {
	approver := executeApprovalApprover{}
	ctx := context.WithValue(context.Background(), approvedRefIDsContextKey, []string{"cell-1", "cell-2"})

	if err := approver.AllowExecute(ctx, []string{"cell-1"}); err != nil {
		t.Fatalf("AllowExecute should succeed with approved ref IDs from context: %v", err)
	}
}

func TestExecuteApprovalApprover_UsesManagerApprovals(t *testing.T) {
	manager := NewExecuteApprovalManager(10 * time.Minute)
	approver := executeApprovalApprover{manager: manager}
	ctx := context.WithValue(context.Background(), sessionIDContextKey, "session-1")

	if err := approver.AllowExecute(ctx, []string{"cell-1"}); err == nil {
		t.Fatalf("AllowExecute should fail before approval")
	}
	if err := manager.Approve("session-1", []string{"cell-1"}); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if err := approver.AllowExecute(ctx, []string{"cell-1"}); err != nil {
		t.Fatalf("AllowExecute should succeed after approval: %v", err)
	}
}

func TestExecuteApprovalHTTPHandler_ListAndApprove(t *testing.T) {
	manager := NewExecuteApprovalManager(10 * time.Minute)
	_ = manager.RequireApproval("session-1", []string{"cell-1"})
	handler := NewExecuteApprovalHTTPHandler(manager)

	getReq := httptest.NewRequest(http.MethodGet, "/codex/execute-approvals?session_id=session-1", nil)
	getRes := httptest.NewRecorder()
	handler.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getRes.Code, http.StatusOK)
	}
	var getBody struct {
		Pending []PendingExecuteApproval `json:"pending"`
	}
	if err := json.Unmarshal(getRes.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("unmarshal GET body failed: %v", err)
	}
	if len(getBody.Pending) != 1 {
		t.Fatalf("pending approvals = %d, want 1", len(getBody.Pending))
	}
	if !reflect.DeepEqual(getBody.Pending[0].RefIDs, []string{"cell-1"}) {
		t.Fatalf("pending refIDs = %#v, want %#v", getBody.Pending[0].RefIDs, []string{"cell-1"})
	}

	postBody := []byte(`{"session_id":"session-1","ref_ids":["cell-1"]}`)
	postReq := httptest.NewRequest(http.MethodPost, "/codex/execute-approvals", bytes.NewReader(postBody))
	postRes := httptest.NewRecorder()
	handler.ServeHTTP(postRes, postReq)
	if postRes.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d", postRes.Code, http.StatusOK)
	}

	getReq = httptest.NewRequest(http.MethodGet, "/codex/execute-approvals?session_id=session-1", nil)
	getRes = httptest.NewRecorder()
	handler.ServeHTTP(getRes, getReq)
	if err := json.Unmarshal(getRes.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("unmarshal GET body failed: %v", err)
	}
	if len(getBody.Pending) != 0 {
		t.Fatalf("pending approvals after approve = %d, want 0", len(getBody.Pending))
	}
}
