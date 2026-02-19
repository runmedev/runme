package codex

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
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

func TestProcessManagerHelper(t *testing.T) {
	if os.Getenv("GO_WANT_PROCESS_MANAGER_HELPER") != "1" {
		return
	}

	if os.Getenv("GO_HELPER_EXPECT_PARENT_ENV") == "1" && os.Getenv("PROCESS_MANAGER_PARENT_MARKER") == "" {
		os.Exit(2)
	}

	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			os.Exit(0)
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
