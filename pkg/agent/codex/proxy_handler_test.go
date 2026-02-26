package codex

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type fakeProxyProcessManager struct {
	ensureStartedCalls int
	callRawCalls       int
	initializeResult   json.RawMessage
	lastMethod         string
	lastParams         any
	onCall             func(method string, params any, onNotification func(jsonRPCNotification) error) (json.RawMessage, error)
}

func (f *fakeProxyProcessManager) EnsureStarted(context.Context) error {
	f.ensureStartedCalls++
	return nil
}

func (f *fakeProxyProcessManager) InitializeResult() json.RawMessage {
	return append(json.RawMessage(nil), f.initializeResult...)
}

func (f *fakeProxyProcessManager) CallRaw(_ context.Context, method string, params any, onNotification func(jsonRPCNotification) error) (json.RawMessage, error) {
	f.callRawCalls++
	f.lastMethod = method
	f.lastParams = params
	if f.onCall != nil {
		return f.onCall(method, params, onNotification)
	}
	return json.RawMessage(`{}`), nil
}

type fakeProxyTokenManager struct {
	token      string
	issueCalls int
}

func (f *fakeProxyTokenManager) Issue() (string, error) {
	f.issueCalls++
	return f.token, nil
}

func TestAppServerProxyHandler_InitializeHandledLocally(t *testing.T) {
	process := &fakeProxyProcessManager{
		initializeResult: json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"codex"}}`),
	}
	handler, err := NewAppServerProxyHandler(process, &fakeProxyTokenManager{token: "token-1"})
	if err != nil {
		t.Fatalf("NewAppServerProxyHandler returned error: %v", err)
	}

	ts := newTCP4TestServer(t, http.HandlerFunc(handler.ServeHTTP))
	defer ts.Close()

	conn := mustDialProxyWebsocket(t, ts.URL)
	defer conn.Close()

	mustWriteProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": defaultInitializeProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test-client", "version": "1.0.0"},
		},
	})

	msg := mustReadProxyMessage(t, conn)
	if got := msg["id"]; got != float64(1) {
		t.Fatalf("initialize response id = %#v, want 1", got)
	}
	result, ok := msg["result"].(map[string]any)
	if !ok {
		t.Fatalf("initialize response missing result object: %#v", msg)
	}
	if got := result["protocolVersion"]; got != defaultInitializeProtocolVersion {
		t.Fatalf("initialize protocolVersion = %#v, want %q", got, defaultInitializeProtocolVersion)
	}
	if process.callRawCalls != 0 {
		t.Fatalf("CallRaw calls = %d, want 0", process.callRawCalls)
	}
}

func TestAppServerProxyHandler_ThreadStartInjectsMCPConfig(t *testing.T) {
	process := &fakeProxyProcessManager{
		initializeResult: json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"codex"}}`),
		onCall: func(method string, params any, _ func(jsonRPCNotification) error) (json.RawMessage, error) {
			return json.RawMessage(`{"thread":{"id":"thread-1"}}`), nil
		},
	}
	handler, err := NewAppServerProxyHandler(process, &fakeProxyTokenManager{token: "token-1"})
	if err != nil {
		t.Fatalf("NewAppServerProxyHandler returned error: %v", err)
	}

	ts := newTCP4TestServer(t, http.HandlerFunc(handler.ServeHTTP))
	defer ts.Close()

	conn := mustDialProxyWebsocket(t, ts.URL)
	defer conn.Close()
	mustInitializeProxyConnection(t, conn)

	mustWriteProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  defaultThreadStartMethod,
		"params": map[string]any{
			"cwd":                   "/tmp/project",
			"model":                 "gpt-5.1-codex",
			"developerInstructions": "User project guidance.",
		},
	})
	msg := mustReadResponseByID(t, conn, 2)
	if _, ok := msg["result"].(map[string]any); !ok {
		t.Fatalf("thread/start response missing result: %#v", msg)
	}

	if process.lastMethod != defaultThreadStartMethod {
		t.Fatalf("CallRaw method = %q, want %q", process.lastMethod, defaultThreadStartMethod)
	}
	params, ok := process.lastParams.(map[string]any)
	if !ok {
		t.Fatalf("CallRaw params type = %T, want map[string]any", process.lastParams)
	}
	if got := params["cwd"]; got != "/tmp/project" {
		t.Fatalf("cwd = %#v, want /tmp/project", got)
	}
	if got := params["model"]; got != "gpt-5.1-codex" {
		t.Fatalf("model = %#v, want gpt-5.1-codex", got)
	}
	if got := params["approvalPolicy"]; got != "never" {
		t.Fatalf("approvalPolicy = %#v, want never", got)
	}
	developerInstructions, _ := params["developerInstructions"].(string)
	if !strings.Contains(developerInstructions, "User project guidance.") || !strings.Contains(developerInstructions, "runme-notebooks") {
		t.Fatalf("developerInstructions = %q, want user guidance plus Runme instructions", developerInstructions)
	}
	config, ok := params["config"].(map[string]any)
	if !ok {
		t.Fatalf("config type = %T, want map[string]any", params["config"])
	}
	mcpServers, ok := config["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("config.mcpServers type = %T, want map[string]any", config["mcpServers"])
	}
	runmeNotebooks, ok := mcpServers["runme-notebooks"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers[runme-notebooks] type = %T, want map[string]any", mcpServers["runme-notebooks"])
	}
	urlValue, _ := runmeNotebooks["url"].(string)
	if !strings.Contains(urlValue, "/mcp/notebooks") || !strings.Contains(urlValue, sessionTokenQueryParam+"=token-1") {
		t.Fatalf("runme-notebooks url = %q, want /mcp/notebooks with session token", urlValue)
	}
}

func TestAppServerProxyHandler_ForwardsTurnNotifications(t *testing.T) {
	process := &fakeProxyProcessManager{
		initializeResult: json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"serverInfo":{"name":"codex"}}`),
		onCall: func(method string, params any, onNotification func(jsonRPCNotification) error) (json.RawMessage, error) {
			if method != defaultTurnStartMethod {
				return json.RawMessage(`{}`), nil
			}
			if err := onNotification(jsonRPCNotification{
				Method: "turn/completed",
				Params: json.RawMessage(`{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed"}}`),
			}); err != nil {
				return nil, err
			}
			return json.RawMessage(`{"turn":{"id":"turn-1"}}`), nil
		},
	}
	handler, err := NewAppServerProxyHandler(process, &fakeProxyTokenManager{token: "token-1"})
	if err != nil {
		t.Fatalf("NewAppServerProxyHandler returned error: %v", err)
	}

	ts := newTCP4TestServer(t, http.HandlerFunc(handler.ServeHTTP))
	defer ts.Close()

	conn := mustDialProxyWebsocket(t, ts.URL)
	defer conn.Close()
	mustInitializeProxyConnection(t, conn)

	mustWriteProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  defaultTurnStartMethod,
		"params": map[string]any{
			"threadId": "thread-1",
			"input":    []map[string]any{{"type": "text", "text": "hello"}},
		},
	})

	sawNotification := false
	for {
		msg := mustReadProxyMessage(t, conn)
		if msg["method"] == "turn/completed" {
			sawNotification = true
			continue
		}
		if msg["id"] == float64(3) {
			break
		}
	}
	if !sawNotification {
		t.Fatalf("expected proxy to forward turn/completed notification")
	}
}

func mustDialProxyWebsocket(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	return conn
}

func mustInitializeProxyConnection(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	mustWriteProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  defaultInitializeMethod,
		"params": map[string]any{
			"protocolVersion": defaultInitializeProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test-client", "version": "1.0.0"},
		},
	})
	_ = mustReadResponseByID(t, conn, 1)
	mustWriteProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"method":  defaultInitializedMethod,
		"params":  map[string]any{},
	})
}

func mustWriteProxyMessage(t *testing.T, conn *websocket.Conn, payload any) {
	t.Helper()
	if err := conn.WriteJSON(payload); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
}

func mustReadResponseByID(t *testing.T, conn *websocket.Conn, id int) map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		msg := mustReadProxyMessage(t, conn)
		if msg["id"] == float64(id) {
			return msg
		}
	}
	t.Fatalf("timed out waiting for response id %d", id)
	return nil
}

func mustReadProxyMessage(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline failed: %v", err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	msg := map[string]any{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("Unmarshal proxy message failed: %v", err)
	}
	return msg
}
