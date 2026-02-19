package codex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"google.golang.org/protobuf/encoding/protojson"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

const (
	errCodeCodexWSAlreadyConnected = "codex_ws_already_connected"
	requestTypeToolCall            = "notebook_tool_call_request"
	responseTypeToolResult         = "notebook_tool_call_response"
)

var (
	ErrBridgeUnavailable = errors.New("codex bridge websocket is not connected")
	ErrBridgeClosed      = errors.New("codex bridge websocket closed")
)

type NotebookToolCallRequest struct {
	Type         string          `json:"type"`
	BridgeCallID string          `json:"bridge_call_id"`
	Input        json.RawMessage `json:"input"`
}

type NotebookToolCallResponse struct {
	Type         string          `json:"type"`
	BridgeCallID string          `json:"bridge_call_id"`
	Output       json.RawMessage `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
}

type pendingBridgeCall struct {
	result chan bridgeCallResult
}

type bridgeCallResult struct {
	output *toolsv1.ToolCallOutput
	err    error
}

type ToolBridge struct {
	upgrader websocket.Upgrader

	mu      sync.Mutex
	conn    *websocket.Conn
	connSeq uint64
	pending map[string]pendingBridgeCall

	writeMu sync.Mutex
}

func NewToolBridge() *ToolBridge {
	return &ToolBridge{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		pending: make(map[string]pendingBridgeCall, 8),
	}
}

// HandleWebsocket upgrades and handles the singleton codex bridge websocket.
func (b *ToolBridge) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logs.FromContextWithTrace(ctx)
	forceReplace := r.URL.Query().Get("force_replace") == "true"

	b.mu.Lock()
	existingConn := b.conn
	b.mu.Unlock()
	if existingConn != nil && !forceReplace {
		writeBridgeConflict(w)
		return
	}

	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "failed to upgrade /codex/ws connection")
		return
	}

	connSeq := b.installConnection(conn)
	if existingConn != nil {
		_ = existingConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "replaced by force_replace"), time.Now().Add(3*time.Second))
		_ = existingConn.Close()
	}

	logger = logger.WithValues("connectionSeq", connSeq)
	ctx = logr.NewContext(ctx, logger)
	logger.Info("codex bridge websocket connected")

	b.readLoop(ctx, conn, connSeq)
}

func writeBridgeConflict(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    errCodeCodexWSAlreadyConnected,
		"message": "codex bridge websocket is already connected",
	})
}

func (b *ToolBridge) installConnection(conn *websocket.Conn) uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connSeq++
	b.conn = conn
	return b.connSeq
}

// Call dispatches a notebook tool call request over the active websocket and waits for the response.
func (b *ToolBridge) Call(ctx context.Context, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error) {
	if input == nil {
		return nil, errors.New("tool call input must not be nil")
	}

	bridgeCallID := uuid.NewString()
	if input.CallId == "" {
		input.CallId = bridgeCallID
	}

	pending := pendingBridgeCall{
		result: make(chan bridgeCallResult, 1),
	}

	conn, err := b.registerPending(bridgeCallID, pending)
	if err != nil {
		return nil, err
	}

	inputJSON, err := protojson.Marshal(input)
	if err != nil {
		return nil, err
	}

	req := NotebookToolCallRequest{
		Type:         requestTypeToolCall,
		BridgeCallID: bridgeCallID,
		Input:        inputJSON,
	}
	if err := b.writeJSON(conn, req); err != nil {
		b.unregisterPending(bridgeCallID)
		return nil, err
	}

	select {
	case <-ctx.Done():
		b.unregisterPending(bridgeCallID)
		return nil, ctx.Err()
	case result := <-pending.result:
		return result.output, result.err
	}
}

func (b *ToolBridge) registerPending(bridgeCallID string, pending pendingBridgeCall) (*websocket.Conn, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn == nil {
		return nil, ErrBridgeUnavailable
	}
	b.pending[bridgeCallID] = pending
	return b.conn, nil
}

func (b *ToolBridge) unregisterPending(bridgeCallID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.pending, bridgeCallID)
}

func (b *ToolBridge) writeJSON(conn *websocket.Conn, payload any) error {
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	return conn.WriteJSON(payload)
}

func (b *ToolBridge) readLoop(ctx context.Context, conn *websocket.Conn, connSeq uint64) {
	logger := logs.FromContext(ctx)
	for {
		var response NotebookToolCallResponse
		if err := conn.ReadJSON(&response); err != nil {
			logger.Info("codex bridge websocket disconnected", "connectionSeq", connSeq, "error", err.Error())
			b.handleDisconnect(conn, err)
			return
		}

		if response.BridgeCallID == "" {
			logger.Info("ignoring codex bridge response without bridge_call_id")
			continue
		}
		if response.Type != "" && response.Type != responseTypeToolResult {
			logger.Info("ignoring codex bridge response with unsupported type", "type", response.Type, "bridgeCallID", response.BridgeCallID)
			continue
		}
		b.resolvePending(response)
	}
}

func (b *ToolBridge) resolvePending(response NotebookToolCallResponse) {
	b.mu.Lock()
	pending, ok := b.pending[response.BridgeCallID]
	if ok {
		delete(b.pending, response.BridgeCallID)
	}
	b.mu.Unlock()
	if !ok {
		return
	}

	if response.Error != "" {
		pending.result <- bridgeCallResult{err: errors.New(response.Error)}
		return
	}
	if len(response.Output) == 0 {
		pending.result <- bridgeCallResult{err: errors.New("bridge response did not include output")}
		return
	}
	output := &toolsv1.ToolCallOutput{}
	if err := protojson.Unmarshal(response.Output, output); err != nil {
		pending.result <- bridgeCallResult{err: err}
		return
	}
	pending.result <- bridgeCallResult{output: output}
}

func (b *ToolBridge) handleDisconnect(conn *websocket.Conn, connErr error) {
	b.mu.Lock()
	if b.conn != conn {
		b.mu.Unlock()
		_ = conn.Close()
		return
	}
	b.conn = nil
	pending := b.pending
	b.pending = make(map[string]pendingBridgeCall, 4)
	b.mu.Unlock()

	_ = conn.Close()

	bridgeErr := ErrBridgeClosed
	if connErr != nil {
		bridgeErr = errors.Join(ErrBridgeClosed, connErr)
	}
	for _, p := range pending {
		p.result <- bridgeCallResult{err: bridgeErr}
	}
}

// Shutdown closes the active websocket connection and fails all pending tool calls.
func (b *ToolBridge) Shutdown() {
	b.mu.Lock()
	conn := b.conn
	b.conn = nil
	pending := b.pending
	b.pending = make(map[string]pendingBridgeCall, 4)
	b.mu.Unlock()

	if conn != nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server shutdown"), time.Now().Add(3*time.Second))
		_ = conn.Close()
	}
	for _, p := range pending {
		p.result <- bridgeCallResult{err: ErrBridgeClosed}
	}
}
