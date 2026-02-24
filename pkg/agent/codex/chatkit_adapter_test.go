package codex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

type fakeCodexProcessManager struct {
	ensureStartedCalls   int
	configureSessionCall int
	runTurnCalls         int
	interruptCalls       int
	lastSessionConfig    SessionConfig
	lastTurnRequest      TurnRequest
	lastInterruptThread  string
	ensureErr            error
	configureErr         error
	runTurnErr           error
	runTurnResp          *TurnResponse
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

func (f *fakeCodexProcessManager) RunTurn(_ context.Context, req TurnRequest) (*TurnResponse, error) {
	f.runTurnCalls++
	f.lastTurnRequest = req
	if f.runTurnResp == nil {
		f.runTurnResp = &TurnResponse{}
	}
	return f.runTurnResp, f.runTurnErr
}

func (f *fakeCodexProcessManager) Interrupt(_ context.Context, _ string, threadID string) error {
	f.interruptCalls++
	f.lastInterruptThread = threadID
	return nil
}

func TestChatKitAdapter_ThreadsCreateStreamsCodexTurn(t *testing.T) {
	fakePM := &fakeCodexProcessManager{}
	tokenManager := NewSessionTokenManager(10 * time.Minute)
	fakePM.runTurnResp = &TurnResponse{
		ThreadID:           "thread-1",
		PreviousResponseID: "resp-1",
		Events: []TurnEvent{
			{Type: "progress_update", Icon: "sparkle", Text: "Working"},
			{Type: "assistant_message", ItemID: "item-1", Text: "Hello from codex"},
		},
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		ProcessManager: fakePM,
		TokenManager:   tokenManager,
	})

	body := `{"type":"threads.create","chatkit_state":{"thread_id":"","previous_response_id":""},"params":{"input":{"content":[{"type":"input_text","text":"hello"}],"attachments":[],"inference_options":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "https://example.test/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
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
	if fakePM.runTurnCalls != 1 {
		t.Fatalf("RunTurn calls = %d, want 1", fakePM.runTurnCalls)
	}
	if fakePM.lastSessionConfig.SessionID == "" {
		t.Fatalf("SessionID should not be empty")
	}
	if fakePM.lastSessionConfig.BearerToken != token {
		t.Fatalf("BearerToken mismatch between response header and configure payload")
	}
	if fakePM.lastSessionConfig.MCPServerURL != "https://example.test/mcp/notebooks" {
		t.Fatalf("MCPServerURL = %q, want https://example.test/mcp/notebooks", fakePM.lastSessionConfig.MCPServerURL)
	}
	if fakePM.lastTurnRequest.Input == nil {
		t.Fatalf("expected input to be passed to RunTurn")
	}
	events := decodeSSEPayloads(t, rr.Body.String())
	if len(events) != 6 {
		t.Fatalf("event count = %d, want 6; body=%s", len(events), rr.Body.String())
	}
	if events[0]["type"] != threadCreatedEventType {
		t.Fatalf("first event type = %v, want %s", events[0]["type"], threadCreatedEventType)
	}
	if events[1]["type"] != aisreChatKitState {
		t.Fatalf("second event type = %v, want %s", events[1]["type"], aisreChatKitState)
	}
	if events[2]["type"] != "progress_update" {
		t.Fatalf("third event type = %v, want progress_update", events[2]["type"])
	}
	if events[3]["type"] != "thread.item.added" {
		t.Fatalf("fourth event type = %v, want thread.item.added", events[3]["type"])
	}
}

func TestChatKitAdapter_ConfigureSessionFailureReturnsBadGateway(t *testing.T) {
	fakePM := &fakeCodexProcessManager{
		configureErr: context.DeadlineExceeded,
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
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

func TestChatKitAdapter_AddUserMessagePassesPreviousResponseID(t *testing.T) {
	fakePM := &fakeCodexProcessManager{
		runTurnResp: &TurnResponse{
			ThreadID:           "thread-1",
			PreviousResponseID: "resp-2",
		},
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		ProcessManager: fakePM,
		TokenManager:   NewSessionTokenManager(10 * time.Minute),
	})
	body := `{"type":"threads.add_user_message","chatkit_state":{"thread_id":"thread-1","previous_response_id":"resp-1"},"params":{"thread_id":"thread-1","input":{"content":[{"type":"input_text","text":"next"}],"attachments":[],"inference_options":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if got := fakePM.lastTurnRequest.PreviousResponseID; got != "resp-1" {
		t.Fatalf("PreviousResponseID = %q, want resp-1", got)
	}
}

func TestChatKitAdapter_AddClientToolOutputParsesProto(t *testing.T) {
	fakePM := &fakeCodexProcessManager{
		runTurnResp: &TurnResponse{
			ThreadID:           "thread-1",
			PreviousResponseID: "resp-2",
		},
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		ProcessManager: fakePM,
		TokenManager:   NewSessionTokenManager(10 * time.Minute),
	})

	toolOutput := &toolsv1.ToolCallOutput{
		CallId: "call-1",
		Output: &toolsv1.ToolCallOutput_ListCells{
			ListCells: &toolsv1.ListCellsResponse{},
		},
		Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
	}
	toolOutputJSON, err := protojson.Marshal(toolOutput)
	if err != nil {
		t.Fatalf("Marshal tool output: %v", err)
	}
	body := `{"type":"threads.add_client_tool_output","chatkit_state":{"thread_id":"thread-1","previous_response_id":"resp-1"},"params":{"thread_id":"thread-1","result":` + string(toolOutputJSON) + `}}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if fakePM.lastTurnRequest.ToolOutput == nil {
		t.Fatalf("expected tool output to be passed to RunTurn")
	}
	if got := fakePM.lastTurnRequest.ToolOutput.GetCallId(); got != "call-1" {
		t.Fatalf("ToolOutput.CallId = %q, want call-1", got)
	}
}

func TestChatKitAdapter_RunTurnFailureSendsErrorEvent(t *testing.T) {
	fakePM := &fakeCodexProcessManager{
		runTurnErr: context.DeadlineExceeded,
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		ProcessManager: fakePM,
		TokenManager:   NewSessionTokenManager(10 * time.Minute),
	})
	body := `{"type":"threads.create","chatkit_state":{"thread_id":"","previous_response_id":""},"params":{"input":{"content":[{"type":"input_text","text":"hello"}],"attachments":[],"inference_options":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	events := decodeSSEPayloads(t, rr.Body.String())
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1; body=%s", len(events), rr.Body.String())
	}
	if events[0]["type"] != "error" {
		t.Fatalf("event type = %v, want error", events[0]["type"])
	}
}

func TestChatKitAdapter_CanceledTurnTriggersInterrupt(t *testing.T) {
	fakePM := &fakeCodexProcessManager{
		runTurnErr: context.Canceled,
	}
	adapter := NewChatKitAdapter(ChatKitAdapterOptions{
		ProcessManager: fakePM,
		TokenManager:   NewSessionTokenManager(10 * time.Minute),
	})
	body := `{"type":"threads.add_user_message","chatkit_state":{"thread_id":"thread-1","previous_response_id":"resp-1"},"params":{"thread_id":"thread-1","input":{"content":[{"type":"input_text","text":"next"}],"attachments":[],"inference_options":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost/chatkit-codex", strings.NewReader(body))
	rr := httptest.NewRecorder()

	adapter.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if fakePM.interruptCalls != 1 {
		t.Fatalf("Interrupt calls = %d, want 1", fakePM.interruptCalls)
	}
	if fakePM.lastInterruptThread != "thread-1" {
		t.Fatalf("last interrupt thread = %q, want thread-1", fakePM.lastInterruptThread)
	}
}

func decodeSSEPayloads(t *testing.T, body string) []map[string]any {
	t.Helper()

	parts := strings.Split(strings.TrimSpace(body), "\n\n")
	events := make([]map[string]any, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.HasPrefix(part, "data: ") {
			t.Fatalf("unexpected SSE chunk: %q", part)
		}
		raw := strings.TrimPrefix(part, "data: ")
		event := map[string]any{}
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			t.Fatalf("failed to unmarshal event %q: %v", raw, err)
		}
		events = append(events, event)
	}
	return events
}
