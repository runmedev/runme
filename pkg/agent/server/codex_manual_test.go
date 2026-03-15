package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	codexv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/codex/v1"
	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/api"
	"github.com/runmedev/runme/v3/pkg/agent/config"
)

// Test_Manual_CodexChatKitSmoke exists to speed up manual backend iteration for codex integration.
// It starts a fake OIDC issuer and a real Runme server in-process, opens a real /codex/app-server/ws
// proxy connection plus a real /codex/ws notebook bridge, then drives a Codex thread/turn flow that
// must use notebook MCP tools to add a Python code cell. It fails with full response diagnostics if
// either the Codex proxy path or notebook tool bridge path breaks.
func Test_Manual_CodexChatKitSmoke(t *testing.T) {
	SkipIfMissing(t, "RUN_MANUAL_TESTS")
	runmeLogs := installManualTestLogger(t)

	if _, err := exec.LookPath("codex"); err != nil {
		t.Skipf("missing codex binary in PATH: %v", err)
	}

	const (
		clientID = "manual-codex-client"
		email    = "codex-manual@example.com"
		prompt   = "Use the runme-notebooks tools to inspect the current notebook and add a new Python code cell whose exact contents are print('Hello, world!'). Do not execute any cells."
	)

	fakeOIDC := newManualFakeOIDC(t, clientID, email)
	idToken := fakeOIDC.mustGenerateToken(t)

	port := mustGetFreePort(t)
	agentEnabled := true
	serverCfg := &config.AssistantServerConfig{
		BindAddress:     "127.0.0.1",
		Port:            port,
		AgentService:    &agentEnabled,
		RunnerService:   true,
		ParserService:   true,
		RunnerReconnect: true,
		CorsOrigins:     []string{"http://localhost:3000"},
		OIDC: &config.OIDCConfig{
			Generic: &config.GenericOIDCConfig{
				ClientID:     clientID,
				ClientSecret: "unused",
				RedirectURL:  "http://localhost:3000/oidc/callback",
				DiscoveryURL: fakeOIDC.discoveryURL(),
				Issuer:       fakeOIDC.issuerURL,
			},
		},
	}

	iamPolicy := &api.IAMPolicy{
		Bindings: []api.IAMBinding{
			{
				Role: api.AgentUserRole,
				Members: []api.Member{
					{Name: email, Kind: api.UserKind},
				},
			},
			{
				Role: api.RunnerUserRole,
				Members: []api.Member{
					{Name: email, Kind: api.UserKind},
				},
			},
			{
				Role: api.ParserUserRole,
				Members: []api.Member{
					{Name: email, Kind: api.UserKind},
				},
			},
		},
	}

	agent, err := ai.NewAgent(ai.AgentOptions{Client: ai.NewClientWithoutKey()})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	srv, err := NewServer(Options{
		Server:                   serverCfg,
		IAMPolicy:                iamPolicy,
		AssetsFileSystemProvider: manualAssetsFileSystemProvider{},
	}, agent)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- srv.Run()
	}()

	t.Cleanup(func() {
		select {
		case err := <-runErrCh:
			if err != nil {
				t.Logf("server exited with error: %v", err)
			}
			return
		default:
		}

		if srv.shutdownComplete != nil {
			srv.shutdown()
		}

		select {
		case err := <-runErrCh:
			if err != nil {
				t.Logf("server exited with error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Logf("timed out waiting for server shutdown")
		}
	})

	baseURL := fmt.Sprintf("http://%s:%d", serverCfg.BindAddress, serverCfg.Port)
	if err := waitForHTTP200(baseURL+"/metrics", 40*time.Second); err != nil {
		t.Fatalf("server did not become ready: %v", err)
	}

	notebookBridge := newManualNotebookBridge()
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/codex/ws"
	wsHeader := http.Header{}
	wsHeader.Set("Authorization", "Bearer "+idToken)
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, wsHeader)
	if err != nil {
		t.Fatalf("failed to connect codex websocket bridge: %v", err)
	}
	defer wsConn.Close()

	if err := wsConn.WriteJSON(map[string]string{
		"authorization": "Bearer " + idToken,
	}); err != nil {
		t.Fatalf("failed to authorize codex websocket bridge: %v", err)
	}

	wsErrCh := make(chan error, 1)
	go notebookBridge.serve(wsConn, wsErrCh)

	proxyURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/codex/app-server/ws"
	proxyHeader := http.Header{}
	proxyHeader.Set("Authorization", "Bearer "+idToken)
	proxyConn, _, err := websocket.DefaultDialer.Dial(proxyURL, proxyHeader)
	if err != nil {
		t.Fatalf("failed to connect codex app-server proxy: %v", err)
	}
	defer proxyConn.Close()

	recorder := &manualTranscriptRecorder{}
	var conversationPath string
	t.Cleanup(func() {
		path, err := recorder.writeJSONFile(threadConversationArtifact{
			ThreadID: threadIDOrEmpty(recorder),
			Prompt:   prompt,
		})
		if err != nil {
			t.Logf("failed to write manual Codex conversation artifact: %v", err)
			return
		}
		conversationPath = path
		t.Logf("manual Codex conversation JSON written to %s", path)
	})

	threadID, turnTranscript := runManualCodexConversation(
		t,
		proxyConn,
		prompt,
		t.TempDir(),
		"Bearer "+idToken,
		recorder,
	)
	if conversationPath == "" {
		t.Logf("manual Codex conversation JSON will be written during cleanup")
	}

	select {
	case err := <-wsErrCh:
		t.Fatalf("codex websocket bridge failed: %v\nrunme logs:\n%s", err, runmeLogs.String())
	default:
	}

	if threadID == "" {
		t.Fatalf("codex proxy response missing thread id; transcript: %s\nrunme logs:\n%s", turnTranscript, runmeLogs.String())
	}
	if !strings.Contains(turnTranscript, `"method":"item/completed"`) && !strings.Contains(turnTranscript, `"method":"turn/completed"`) {
		t.Fatalf("codex proxy transcript missing turn completion notifications; transcript: %s\nrunme logs:\n%s", turnTranscript, runmeLogs.String())
	}

	listCalls, getCalls, updateCalls := notebookBridge.callCounts()
	if listCalls+getCalls == 0 {
		t.Fatalf("expected codex to inspect notebook before editing; counts list=%d get=%d update=%d\ntranscript: %s\nrunme logs:\n%s", listCalls, getCalls, updateCalls, turnTranscript, runmeLogs.String())
	}
	if updateCalls == 0 {
		t.Fatalf("expected codex to invoke UpdateCells; counts list=%d get=%d update=%d\ntranscript: %s\nrunme logs:\n%s", listCalls, getCalls, updateCalls, turnTranscript, runmeLogs.String())
	}
	if !notebookBridge.containsCodeCell("python", "print('Hello, world!')") {
		t.Fatalf("expected notebook bridge state to contain inserted Python code cell; cells=%#v\ntranscript: %s\nrunme logs:\n%s", notebookBridge.snapshotCells(), turnTranscript, runmeLogs.String())
	}
}

func runManualCodexConversation(
	t *testing.T,
	conn *websocket.Conn,
	prompt,
	cwd,
	authorization string,
	recorder *manualTranscriptRecorder,
) (string, string) {
	t.Helper()

	mustWriteManualProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "manual-codex-test",
				"version": "1.0.0",
			},
			"authorization": authorization,
		},
	}, recorder)
	_ = mustReadManualProxyResponse(t, conn, 1, recorder)
	mustWriteManualProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]any{},
	}, recorder)

	mustWriteManualProxyMessage(t, conn, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "thread/start",
		"params": map[string]any{
			"cwd":   cwd,
			"model": "gpt-5.1-codex",
		},
	}, recorder)
	threadStart := mustReadManualProxyResponse(t, conn, 2, recorder)
	threadResult, _ := threadStart["result"].(map[string]any)
	threadValue, _ := threadResult["thread"].(map[string]any)
	threadID, _ := threadValue["id"].(string)
	if strings.TrimSpace(threadID) == "" {
		t.Fatalf("thread/start response missing thread id: %#v", threadStart)
	}

	const maxAttempts = 2
	var transcript string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		mustWriteManualProxyMessage(t, conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      3,
			"method":  "turn/start",
			"params": map[string]any{
				"threadId": threadID,
				"input": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		}, recorder)

		turnResp := mustReadManualProxyResponse(t, conn, 3, recorder)
		transcript = recorder.ndjson()
		if !isRetriableManualCodexTurnFailure(transcript, turnResp) || attempt == maxAttempts {
			if errValue, ok := turnResp["error"]; ok && errValue != nil {
				t.Fatalf("turn/start returned error: %#v\ntranscript: %s", errValue, transcript)
			}
			return threadID, transcript
		}
		t.Logf("retrying transient codex upstream failure on attempt %d/%d", attempt, maxAttempts)
	}

	return threadID, transcript
}

func isRetriableManualCodexTurnFailure(transcript string, response map[string]any) bool {
	if strings.Contains(transcript, "stream disconnected before completion") {
		return true
	}
	if response == nil {
		return false
	}
	errValue, ok := response["error"].(map[string]any)
	if !ok {
		return false
	}
	message, _ := errValue["message"].(string)
	return strings.Contains(message, "stream disconnected before completion")
}

func mustWriteManualProxyMessage(t *testing.T, conn *websocket.Conn, payload any, recorder *manualTranscriptRecorder) {
	t.Helper()
	recorder.record("request", payload)
	if err := conn.WriteJSON(payload); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
}

func mustReadManualProxyResponse(t *testing.T, conn *websocket.Conn, id int, args ...any) map[string]any {
	t.Helper()

	var recorder *manualTranscriptRecorder
	var collected *[]map[string]any
	for _, arg := range args {
		switch typed := arg.(type) {
		case *manualTranscriptRecorder:
			recorder = typed
		case *[]map[string]any:
			collected = typed
		}
	}

	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			t.Fatalf("SetReadDeadline failed: %v", err)
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage failed: %v", err)
		}
		msg := map[string]any{}
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("failed to unmarshal proxy message %q: %v", string(payload), err)
		}
		if recorder != nil {
			recorder.record(messageKind(msg), msg)
		}
		if collected != nil {
			*collected = append(*collected, msg)
		}
		if gotID, ok := msg["id"].(float64); ok && int(gotID) == id {
			return msg
		}
	}
	t.Fatalf("timed out waiting for proxy response id %d", id)
	return nil
}

type manualTranscriptRecorder struct {
	events []map[string]any
}

func (r *manualTranscriptRecorder) record(kind string, payload any) {
	if r == nil {
		return
	}
	r.events = append(r.events, map[string]any{
		"kind":    kind,
		"payload": payload,
	})
}

func (r *manualTranscriptRecorder) ndjson() string {
	if r == nil || len(r.events) == 0 {
		return ""
	}
	encoded := make([]string, 0, len(r.events))
	for _, event := range r.events {
		payload, err := json.Marshal(event)
		if err != nil {
			continue
		}
		encoded = append(encoded, string(payload))
	}
	return strings.Join(encoded, "\n")
}

func (r *manualTranscriptRecorder) writeJSONFile(artifact threadConversationArtifact) (string, error) {
	if r == nil {
		return "", fmt.Errorf("manual transcript recorder is nil")
	}
	artifact.Events = append([]map[string]any(nil), r.events...)
	file, err := os.CreateTemp("/tmp", "runme-codex-conversation-*.json")
	if err != nil {
		return "", err
	}
	defer file.Close()

	payload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", err
	}
	if _, err := file.Write(payload); err != nil {
		return "", err
	}
	return file.Name(), nil
}

type threadConversationArtifact struct {
	ThreadID string           `json:"thread_id"`
	Prompt   string           `json:"prompt"`
	Events   []map[string]any `json:"events"`
}

func threadIDOrEmpty(recorder *manualTranscriptRecorder) string {
	if recorder == nil {
		return ""
	}
	for _, event := range recorder.events {
		payload, ok := event["payload"].(map[string]any)
		if !ok {
			continue
		}
		result, ok := payload["result"].(map[string]any)
		if !ok {
			continue
		}
		thread, ok := result["thread"].(map[string]any)
		if !ok {
			continue
		}
		threadID, _ := thread["id"].(string)
		if strings.TrimSpace(threadID) != "" {
			return threadID
		}
	}
	return ""
}

func messageKind(message map[string]any) string {
	if _, ok := message["id"]; ok {
		return "response"
	}
	return "notification"
}

type manualAssetsFileSystemProvider struct{}

func (manualAssetsFileSystemProvider) GetAssetsFileSystem() (fs.FS, error) {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><html><body>manual</body></html>"),
			Mode: 0o644,
		},
	}, nil
}

type manualFakeOIDC struct {
	privateKey *rsa.PrivateKey
	issuerURL  string
	clientID   string
	email      string
	kid        string
	server     *httptest.Server
}

func newManualFakeOIDC(t *testing.T, clientID, email string) *manualFakeOIDC {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	fake := &manualFakeOIDC{
		privateKey: privateKey,
		clientID:   clientID,
		email:      email,
		kid:        "manual-test-key",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeManualJSON(w, map[string]any{
			"issuer":                 fake.issuerURL,
			"jwks_uri":               fake.issuerURL + "/jwks",
			"authorization_endpoint": fake.issuerURL + "/auth",
			"token_endpoint":         fake.issuerURL + "/token",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeManualJSON(w, map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": fake.kid,
					"n":   base64.RawURLEncoding.EncodeToString(fake.privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(fake.privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start fake OIDC listener: %v", err)
	}
	fake.server = httptest.NewUnstartedServer(mux)
	fake.server.Listener = listener
	fake.server.Start()
	fake.issuerURL = fake.server.URL
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *manualFakeOIDC) discoveryURL() string {
	return f.issuerURL + "/.well-known/openid-configuration"
}

func (f *manualFakeOIDC) mustGenerateToken(t *testing.T) string {
	t.Helper()
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   f.issuerURL,
		"aud":   f.clientID,
		"sub":   f.email,
		"email": f.email,
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
	})
	token.Header["kid"] = f.kid
	signed, err := token.SignedString(f.privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func writeManualJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func mustGetFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to acquire free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForHTTP200(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %s", url)
}

func installManualTestLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	logBuf := &bytes.Buffer{}
	encCfg := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), zapcore.AddSync(logBuf), zap.DebugLevel)
	logger := zap.New(core)

	restore := zap.ReplaceGlobals(logger)
	t.Cleanup(func() {
		restore()
		_ = logger.Sync()
	})
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("runme server logs:\n%s", logBuf.String())
		}
	})
	return logBuf
}

type manualNotebookBridge struct {
	mu          sync.Mutex
	cells       []*parserv1.Cell
	listCalls   int
	getCalls    int
	updateCalls int
	nextID      int
}

func newManualNotebookBridge() *manualNotebookBridge {
	return &manualNotebookBridge{
		cells: []*parserv1.Cell{
			{
				RefId: "intro-cell",
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "Notebook setup notes",
			},
			{
				RefId:      "seed-code",
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "python",
				Value:      "print('seed cell')",
			},
		},
		nextID: 1,
	}
}

func (b *manualNotebookBridge) serve(conn *websocket.Conn, errCh chan<- error) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			select {
			case errCh <- err:
			default:
			}
			return
		}

		req := &codexv1.WebsocketResponse{}
		if err := protojson.Unmarshal(message, req); err != nil {
			select {
			case errCh <- fmt.Errorf("unmarshal websocket request: %w", err):
			default:
			}
			return
		}

		toolReq := req.GetNotebookToolCallRequest()
		if toolReq == nil {
			continue
		}

		resp, err := b.handleToolCall(toolReq)
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
			return
		}

		payload, err := protojson.Marshal(resp)
		if err != nil {
			select {
			case errCh <- fmt.Errorf("marshal websocket response: %w", err):
			default:
			}
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			select {
			case errCh <- fmt.Errorf("write websocket response: %w", err):
			default:
			}
			return
		}
	}
}

func (b *manualNotebookBridge) handleToolCall(req *codexv1.NotebookToolCallRequest) (*codexv1.WebsocketRequest, error) {
	if req.GetBridgeCallId() == "" {
		return nil, fmt.Errorf("bridge request missing bridge_call_id")
	}
	input := req.GetInput()
	if input == nil {
		return nil, fmt.Errorf("bridge request missing input")
	}

	output, err := b.toolOutputForInput(req.GetBridgeCallId(), input)
	if err != nil {
		return nil, err
	}
	return &codexv1.WebsocketRequest{
		Payload: &codexv1.WebsocketRequest_NotebookToolCallResponse{
			NotebookToolCallResponse: &codexv1.NotebookToolCallResponse{
				BridgeCallId: req.GetBridgeCallId(),
				Output:       output,
			},
		},
	}, nil
}

func (b *manualNotebookBridge) toolOutputForInput(callID string, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error) {
	switch {
	case input.GetListCells() != nil:
		b.mu.Lock()
		b.listCalls++
		cells := cloneManualCells(b.cells)
		b.mu.Unlock()
		return &toolsv1.ToolCallOutput{
			CallId: callID,
			Output: &toolsv1.ToolCallOutput_ListCells{
				ListCells: &toolsv1.ListCellsResponse{Cells: cells},
			},
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
		}, nil
	case input.GetGetCells() != nil:
		b.mu.Lock()
		b.getCalls++
		cells := b.getCellsLocked(input.GetGetCells().GetRefIds())
		b.mu.Unlock()
		return &toolsv1.ToolCallOutput{
			CallId: callID,
			Output: &toolsv1.ToolCallOutput_GetCells{
				GetCells: &toolsv1.GetCellsResponse{Cells: cells},
			},
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
		}, nil
	case input.GetUpdateCells() != nil:
		b.mu.Lock()
		b.updateCalls++
		updated := b.applyUpdateLocked(input.GetUpdateCells().GetCells())
		b.mu.Unlock()
		return &toolsv1.ToolCallOutput{
			CallId: callID,
			Output: &toolsv1.ToolCallOutput_UpdateCells{
				UpdateCells: &toolsv1.UpdateCellsResponse{Cells: updated},
			},
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
		}, nil
	case input.GetExecuteCells() != nil:
		return &toolsv1.ToolCallOutput{
			CallId: callID,
			Output: &toolsv1.ToolCallOutput_ExecuteCells{
				ExecuteCells: &toolsv1.NotebookServiceExecuteCellsResponse{},
			},
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported tool call payload %T", input.GetInput())
	}
}

func (b *manualNotebookBridge) getCellsLocked(refIDs []string) []*parserv1.Cell {
	if len(refIDs) == 0 {
		return cloneManualCells(b.cells)
	}

	cells := make([]*parserv1.Cell, 0, len(refIDs))
	for _, refID := range refIDs {
		for _, cell := range b.cells {
			if cell.GetRefId() != refID {
				continue
			}
			cells = append(cells, cloneManualCell(cell))
			break
		}
	}
	return cells
}

func (b *manualNotebookBridge) applyUpdateLocked(incoming []*parserv1.Cell) []*parserv1.Cell {
	updated := make([]*parserv1.Cell, 0, len(incoming))
	for _, cell := range incoming {
		next := cloneManualCell(cell)
		if strings.TrimSpace(next.GetRefId()) == "" {
			next.RefId = fmt.Sprintf("manual-cell-%d", b.nextID)
			b.nextID++
		}
		replaced := false
		for i, existing := range b.cells {
			if existing.GetRefId() != next.GetRefId() {
				continue
			}
			b.cells[i] = next
			replaced = true
			break
		}
		if !replaced {
			b.cells = append(b.cells, next)
		}
		updated = append(updated, cloneManualCell(next))
	}
	return updated
}

func (b *manualNotebookBridge) callCounts() (listCalls, getCalls, updateCalls int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.listCalls, b.getCalls, b.updateCalls
}

func (b *manualNotebookBridge) containsCodeCell(languageID, value string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	normalize := func(input string) string {
		return strings.ReplaceAll(strings.TrimSpace(input), "\"", "'")
	}
	expected := normalize(value)
	for _, cell := range b.cells {
		if cell.GetLanguageId() != languageID {
			continue
		}
		if strings.Contains(normalize(cell.GetValue()), expected) {
			return true
		}
	}
	return false
}

func (b *manualNotebookBridge) snapshotCells() []*parserv1.Cell {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneManualCells(b.cells)
}

func cloneManualCells(cells []*parserv1.Cell) []*parserv1.Cell {
	out := make([]*parserv1.Cell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, cloneManualCell(cell))
	}
	return out
}

func cloneManualCell(cell *parserv1.Cell) *parserv1.Cell {
	if cell == nil {
		return nil
	}
	clone, _ := proto.Clone(cell).(*parserv1.Cell)
	return clone
}
