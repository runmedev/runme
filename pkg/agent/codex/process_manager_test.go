package codex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/pkg/agent/ai/chatkit"
)

func TestProcessManager_EnsureStartedInitializesServer(t *testing.T) {
	t.Setenv("PROCESS_MANAGER_PARENT_MARKER", "present")
	pm := NewProcessManager(
		os.Args[0],
		[]string{"-test.run=TestProcessManagerHelper", "--"},
		[]string{
			"GO_WANT_PROCESS_MANAGER_HELPER=1",
			"GO_HELPER_EXPECT_PARENT_ENV=1",
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := pm.EnsureStarted(ctx); err != nil {
		t.Fatalf("EnsureStarted returned error: %v", err)
	}
	defer func() {
		_ = pm.Stop(context.Background())
	}()
}

func TestProcessManager_EnsureStartedReturnsInitializeError(t *testing.T) {
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
		t.Fatalf("EnsureStarted should fail when initialize fails")
	}
	if !strings.Contains(err.Error(), "initialize codex app-server") {
		t.Fatalf("error %q should mention initialize", err)
	}
}

func TestProcessManager_ConfigureSessionRequiresRunningProcess(t *testing.T) {
	pm := NewProcessManager("codex", nil, nil)
	err := pm.ConfigureSession(context.Background(), SessionConfig{
		SessionID:    "session-1",
		MCPServerURL: "http://localhost/mcp/notebooks",
		BearerToken:  "token-1",
	})
	if err == nil {
		t.Fatalf("ConfigureSession should fail if process is not running")
	}
}

func TestProcessManager_RunTurnRequiresRunningProcess(t *testing.T) {
	pm := NewProcessManager("codex", nil, nil)
	_, err := pm.RunTurn(context.Background(), TurnRequest{
		SessionID: "session-1",
		ThreadID:  "thread-1",
	})
	if err == nil {
		t.Fatalf("RunTurn should fail if process is not running")
	}
}

func TestProcessManager_MarshalSessionConfigIncludesApprovalPolicyAndAuthHeader(t *testing.T) {
	pm := NewProcessManager("codex", nil, nil)
	data, err := pm.MarshalSessionConfig(SessionConfig{
		SessionID:    "session-1",
		MCPServerURL: "http://localhost/mcp/notebooks",
		BearerToken:  "token-1",
	})
	if err != nil {
		t.Fatalf("MarshalSessionConfig returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal session config payload: %v", err)
	}
	if payload["approval_policy"] != "never" {
		t.Fatalf("approval_policy = %v, want never", payload["approval_policy"])
	}
}

func TestBuildTurnParamsIncludesInputAndToolOutput(t *testing.T) {
	params := buildTurnParams(TurnRequest{
		SessionID:          "session-1",
		ThreadID:           "thread-1",
		PreviousResponseID: "resp-1",
		Input: &chatkit.UserMessageInput{
			Content: []chatkit.UserMessageContent{
				{Type: "input_text", Text: "hello"},
				{Type: "input_text", Text: "world"},
			},
		},
		ToolOutput: &toolsv1.ToolCallOutput{
			CallId: "call-1",
			Output: &toolsv1.ToolCallOutput_ListCells{
				ListCells: &toolsv1.ListCellsResponse{},
			},
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
		},
	})
	if params["session_id"] != "session-1" {
		t.Fatalf("session_id = %v, want session-1", params["session_id"])
	}
	if params["thread_id"] != "thread-1" {
		t.Fatalf("thread_id = %v, want thread-1", params["thread_id"])
	}
	if params["previous_response_id"] != "resp-1" {
		t.Fatalf("previous_response_id = %v, want resp-1", params["previous_response_id"])
	}
	input, ok := params["input"].([]map[string]any)
	if !ok {
		t.Fatalf("input type = %T, want []map[string]any", params["input"])
	}
	if len(input) != 3 {
		t.Fatalf("input len = %d, want 3", len(input))
	}
	if input[0]["type"] != "text" || input[0]["text"] != "hello" {
		t.Fatalf("input[0] = %#v, want text item hello", input[0])
	}
	if input[1]["type"] != "text" || input[1]["text"] != "world" {
		t.Fatalf("input[1] = %#v, want text item world", input[1])
	}
	if input[2]["type"] != "text" {
		t.Fatalf("input[2].type = %v, want text", input[2]["type"])
	}
	message, _ := params["message"].(string)
	if !strings.Contains(message, "hello\n\nworld") {
		t.Fatalf("message = %q, want user text content", message)
	}
	if !strings.Contains(message, "\"callId\":\"call-1\"") {
		t.Fatalf("message = %q, want serialized tool output", message)
	}
	if params["tool_output"] == nil {
		t.Fatalf("tool_output should be present")
	}
}

func TestProcessManager_RunTurnAndInterruptDispatchMethods(t *testing.T) {
	captureFile := filepath.Join(t.TempDir(), "rpc-requests.jsonl")
	pm := NewProcessManager(
		os.Args[0],
		[]string{"-test.run=TestProcessManagerHelper", "--"},
		[]string{
			"GO_WANT_PROCESS_MANAGER_HELPER=1",
			"GO_HELPER_CAPTURE_FILE=" + captureFile,
		},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := pm.EnsureStarted(ctx); err != nil {
		t.Fatalf("EnsureStarted returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = pm.Stop(context.Background())
	})

	resp, err := pm.RunTurn(ctx, TurnRequest{
		SessionID:          "session-1",
		ThreadID:           "thread-1",
		PreviousResponseID: "resp-1",
		Input: &chatkit.UserMessageInput{
			Content: []chatkit.UserMessageContent{
				{Type: "input_text", Text: "hello from test"},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunTurn returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("RunTurn response must not be nil")
	}
	if err := pm.Interrupt(ctx, "session-1", "thread-1"); err != nil {
		t.Fatalf("Interrupt returned error: %v", err)
	}

	methods := waitForCapturedMethods(t, captureFile, 2*time.Second)

	if !containsMethod(methods, defaultInitializeMethod) {
		t.Fatalf("captured methods %v do not include %q", methods, defaultInitializeMethod)
	}
	if !containsMethod(methods, defaultTurnStartMethod) {
		t.Fatalf("captured methods %v do not include %q", methods, defaultTurnStartMethod)
	}
	if !containsMethod(methods, defaultThreadInterrupt) {
		t.Fatalf("captured methods %v do not include %q", methods, defaultThreadInterrupt)
	}
}

func TestProcessManager_EnsureStartedSendsInitializeParams(t *testing.T) {
	captureFile := filepath.Join(t.TempDir(), "rpc-requests.jsonl")
	pm := NewProcessManager(
		os.Args[0],
		[]string{"-test.run=TestProcessManagerHelper", "--"},
		[]string{
			"GO_WANT_PROCESS_MANAGER_HELPER=1",
			"GO_HELPER_CAPTURE_FILE=" + captureFile,
		},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := pm.EnsureStarted(ctx); err != nil {
		t.Fatalf("EnsureStarted returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = pm.Stop(context.Background())
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		requests, err := readCapturedRequests(captureFile)
		if err == nil {
			for _, req := range requests {
				method, _ := req["method"].(string)
				if method != defaultInitializeMethod {
					continue
				}

				params, ok := req["params"].(map[string]any)
				if !ok {
					t.Fatalf("initialize params type = %T, want map[string]any", req["params"])
				}
				if params["protocolVersion"] != defaultInitializeProtocolVersion {
					t.Fatalf("protocolVersion = %v, want %q", params["protocolVersion"], defaultInitializeProtocolVersion)
				}
				if _, ok := params["capabilities"].(map[string]any); !ok {
					t.Fatalf("capabilities type = %T, want map[string]any", params["capabilities"])
				}
				clientInfo, ok := params["clientInfo"].(map[string]any)
				if !ok {
					t.Fatalf("clientInfo type = %T, want map[string]any", params["clientInfo"])
				}
				if clientInfo["name"] != defaultInitializeClientName {
					t.Fatalf("clientInfo.name = %v, want %q", clientInfo["name"], defaultInitializeClientName)
				}
				if clientInfo["version"] != defaultInitializeClientVersion {
					t.Fatalf("clientInfo.version = %v, want %q", clientInfo["version"], defaultInitializeClientVersion)
				}
				return
			}
		}

		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("read capture file: %v", err)
			}
			t.Fatalf("initialize request not captured in %s", captureFile)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func containsMethod(methods []string, want string) bool {
	for _, method := range methods {
		if method == want {
			return true
		}
	}
	return false
}

func waitForCapturedMethods(t *testing.T, captureFile string, timeout time.Duration) []string {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		methods, err := readCapturedMethods(captureFile)
		if err == nil && containsMethod(methods, defaultThreadInterrupt) {
			return methods
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("read capture file: %v", err)
			}
			return methods
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func readCapturedRequests(captureFile string) ([]map[string]any, error) {
	data, err := os.ReadFile(captureFile)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	requests := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		req := map[string]any{}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func readCapturedMethods(captureFile string) ([]string, error) {
	requests, err := readCapturedRequests(captureFile)
	if err != nil {
		return nil, err
	}
	methods := make([]string, 0, len(requests))
	for _, req := range requests {
		method, _ := req["method"].(string)
		if method != "" {
			methods = append(methods, method)
		}
	}
	return methods, nil
}

func TestProcessManagerHelper(t *testing.T) {
	if os.Getenv("GO_WANT_PROCESS_MANAGER_HELPER") != "1" {
		return
	}

	if os.Getenv("GO_HELPER_EXPECT_PARENT_ENV") == "1" && os.Getenv("PROCESS_MANAGER_PARENT_MARKER") == "" {
		os.Exit(2)
	}

	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	captureFile := os.Getenv("GO_HELPER_CAPTURE_FILE")

	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			os.Exit(0)
		}
		if captureFile != "" {
			raw, _ := json.Marshal(req)
			f, err := os.OpenFile(captureFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil {
				_, _ = f.Write(append(raw, '\n'))
				_ = f.Close()
			}
		}
		method, _ := req["method"].(string)
		id, hasID := req["id"]
		if !hasID {
			continue
		}
		if method == defaultInitializeMethod {
			if os.Getenv("GO_HELPER_FAIL_INITIALIZE") == "1" {
				_ = enc.Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"error": map[string]any{
						"code":    -32000,
						"message": "initialize failed",
					},
				})
			} else {
				_ = enc.Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]any{
						"ok": true,
					},
				})
			}
			continue
		}
		_ = enc.Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  map[string]any{},
		})
	}
}
