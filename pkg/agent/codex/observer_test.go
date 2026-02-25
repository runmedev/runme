package codex

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mark3labs/mcp-go/mcp"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

type startupEvent struct {
	status   string
	duration time.Duration
}

type toolEvent struct {
	tool     string
	outcome  string
	duration time.Duration
}

type fakeObserver struct {
	startups    []startupEvent
	toolCalls   []toolEvent
	disconnects []string
}

func (f *fakeObserver) ObserveAppServerStartup(duration time.Duration, status string) {
	f.startups = append(f.startups, startupEvent{
		status:   status,
		duration: duration,
	})
}

func (f *fakeObserver) ObserveMCPToolCall(tool string, duration time.Duration, outcome string) {
	f.toolCalls = append(f.toolCalls, toolEvent{
		tool:     tool,
		outcome:  outcome,
		duration: duration,
	})
}

func (f *fakeObserver) IncBridgeDisconnect(reason string) {
	f.disconnects = append(f.disconnects, reason)
}

func TestProcessManager_EnsureStartedObservesStartup(t *testing.T) {
	obs := &fakeObserver{}
	t.Cleanup(setObserverForTest(obs))

	pm := NewProcessManager(
		os.Args[0],
		[]string{"-test.run=TestProcessManagerHelper", "--"},
		[]string{"GO_WANT_PROCESS_MANAGER_HELPER=1"},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := pm.EnsureStarted(ctx); err != nil {
		t.Fatalf("EnsureStarted returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = pm.Stop(context.Background())
	})

	if len(obs.startups) != 1 {
		t.Fatalf("startup events = %d, want 1", len(obs.startups))
	}
	if obs.startups[0].status != startupStatusSuccess {
		t.Fatalf("startup status = %q, want %q", obs.startups[0].status, startupStatusSuccess)
	}
	if obs.startups[0].duration <= 0 {
		t.Fatalf("startup duration = %s, want > 0", obs.startups[0].duration)
	}
}

func TestProcessManager_EnsureStartedObservesFailure(t *testing.T) {
	obs := &fakeObserver{}
	t.Cleanup(setObserverForTest(obs))

	pm := NewProcessManager(
		os.Args[0],
		[]string{"-test.run=TestProcessManagerHelper", "--"},
		[]string{
			"GO_WANT_PROCESS_MANAGER_HELPER=1",
			"GO_HELPER_FAIL_INITIALIZE=1",
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := pm.EnsureStarted(ctx)
	if err == nil {
		t.Fatalf("EnsureStarted should fail")
	}
	if len(obs.startups) != 1 {
		t.Fatalf("startup events = %d, want 1", len(obs.startups))
	}
	if obs.startups[0].status != startupStatusError {
		t.Fatalf("startup status = %q, want %q", obs.startups[0].status, startupStatusError)
	}
}

func TestNotebookMCPBridge_ObservesToolCallOutcome(t *testing.T) {
	obs := &fakeObserver{}
	t.Cleanup(setObserverForTest(obs))

	bridge := &fakeBridge{
		nextOutput: &toolsv1.ToolCallOutput{
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
			Output: &toolsv1.ToolCallOutput_ListCells{
				ListCells: &toolsv1.ListCellsResponse{},
			},
		},
	}
	nb := NewNotebookMCPBridge(bridge)
	if _, err := nb.handleListCells(context.Background(), mcp.CallToolRequest{}, listCellsArgs{}); err != nil {
		t.Fatalf("handleListCells returned error: %v", err)
	}
	if len(obs.toolCalls) != 1 {
		t.Fatalf("tool call events = %d, want 1", len(obs.toolCalls))
	}
	if obs.toolCalls[0].tool != "ListCells" {
		t.Fatalf("tool = %q, want ListCells", obs.toolCalls[0].tool)
	}
	if obs.toolCalls[0].outcome != toolOutcomeSuccess {
		t.Fatalf("tool outcome = %q, want %q", obs.toolCalls[0].outcome, toolOutcomeSuccess)
	}
}

func TestToolCallOutcomeClassifiesBridgeErrors(t *testing.T) {
	outcome := toolCallOutcome(nil, errors.New("boom"))
	if outcome != toolOutcomeBridgeErr {
		t.Fatalf("outcome = %q, want %q", outcome, toolOutcomeBridgeErr)
	}
}

func TestDisconnectReasonClassification(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "shutdown",
			err:  nil,
			want: disconnectReasonShutdown,
		},
		{
			name: "replaced",
			err:  errors.New("websocket: close 1000 (normal): replaced by force_replace"),
			want: disconnectReasonReplaced,
		},
		{
			name: "normal closure",
			err:  &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "normal closure"},
			want: disconnectReasonClient,
		},
		{
			name: "other",
			err:  errors.New("i/o timeout"),
			want: disconnectReasonReadError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := disconnectReason(tt.err)
			if got != tt.want {
				t.Fatalf("disconnectReason(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
