package codex

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeCodexProcessManager struct {
	ensureStartedCalls   int
	configureSessionCall int
	lastSessionConfig    SessionConfig
	ensureErr            error
	configureErr         error
}

func (f *fakeCodexProcessManager) EnsureStarted(context.Context) error {
	f.ensureStartedCalls++
	return f.ensureErr
}

func (f *fakeCodexProcessManager) ConfigureSession(_ context.Context, cfg SessionConfig) error {
	f.configureSessionCall++
	f.lastSessionConfig = cfg
	return f.configureErr
}

func TestChatKitAdapter_ConfiguresCodexSessionAndRestoresBody(t *testing.T) {
	fakePM := &fakeCodexProcessManager{}
	tokenManager := NewSessionTokenManager(10 * time.Minute)
	var fallbackBody string
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read fallback request body: %v", err)
		}
		fallbackBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	})
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		Fallback:       fallback,
		ProcessManager: fakePM,
		TokenManager:   tokenManager,
	})

	body := `{"type":"threads.create","chatkit_state":{"thread_id":"thread-1"},"params":{"input":{"content":[],"attachments":[],"inference_options":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "https://example.test/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if fakePM.ensureStartedCalls != 1 {
		t.Fatalf("EnsureStarted calls = %d, want 1", fakePM.ensureStartedCalls)
	}
	if fakePM.configureSessionCall != 1 {
		t.Fatalf("ConfigureSession calls = %d, want 1", fakePM.configureSessionCall)
	}
	token := rr.Header().Get(codexSessionTokenHeader)
	if token == "" {
		t.Fatalf("missing %s header", codexSessionTokenHeader)
	}
	if fakePM.lastSessionConfig.SessionID != "thread-1" {
		t.Fatalf("SessionID = %q, want thread-1", fakePM.lastSessionConfig.SessionID)
	}
	if fakePM.lastSessionConfig.BearerToken != token {
		t.Fatalf("BearerToken mismatch between response header and configure payload")
	}
	if fakePM.lastSessionConfig.MCPServerURL != "https://example.test/mcp/notebooks" {
		t.Fatalf("MCPServerURL = %q, want https://example.test/mcp/notebooks", fakePM.lastSessionConfig.MCPServerURL)
	}
	if !strings.Contains(fallbackBody, `"thread_id":"thread-1"`) {
		t.Fatalf("fallback body was not restored correctly: %q", fallbackBody)
	}
}

func TestChatKitAdapter_ConfigureSessionFailureReturnsBadGateway(t *testing.T) {
	fakePM := &fakeCodexProcessManager{
		configureErr: context.DeadlineExceeded,
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		Fallback:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		ProcessManager: fakePM,
		TokenManager:   NewSessionTokenManager(10 * time.Minute),
	})
	body := `{"type":"threads.create","chatkit_state":{"thread_id":"thread-1"}}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadGateway, rr.Body.String())
	}
}

func TestChatKitAdapter_RejectsNonPOST(t *testing.T) {
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{})
	req := httptest.NewRequest(http.MethodGet, "http://localhost/chatkit-codex", nil)
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
