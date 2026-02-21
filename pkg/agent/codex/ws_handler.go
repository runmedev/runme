package codex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	codexv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/codex/v1"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	errCodeCodexWSAlreadyConnected = "codex_ws_already_connected"
)

var (
	ErrBridgeUnavailable = errors.New("codex bridge websocket is not connected")
	ErrBridgeClosed      = errors.New("codex bridge websocket closed")
)

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
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}
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
	logger := logs.FromContextWithTrace(ctx).WithValues(
		"bridgeCallID", bridgeCallID,
		"tool", toolNameFromInput(input),
		"sessionID", SessionIDFromContext(ctx),
	)
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}

	pending := pendingBridgeCall{
		result: make(chan bridgeCallResult, 1),
	}

	conn, err := b.registerPending(bridgeCallID, pending)
	if err != nil {
		return nil, err
	}

	req := &codexv1.WebsocketResponse{
		Payload: &codexv1.WebsocketResponse_NotebookToolCallRequest{
			NotebookToolCallRequest: &codexv1.NotebookToolCallRequest{
				BridgeCallId: bridgeCallID,
				Input:        input,
			},
		},
	}
	if err := b.writeProtoJSON(conn, req); err != nil {
		b.unregisterPending(bridgeCallID)
		logger.Error(err, "failed to write bridge tool call request")
		return nil, err
	}
	logger.Info("dispatched bridge tool call")

	select {
	case <-ctx.Done():
		b.unregisterPending(bridgeCallID)
		logger.Error(ctx.Err(), "bridge tool call canceled")
		return nil, ctx.Err()
	case result := <-pending.result:
		if result.err != nil {
			logger.Error(result.err, "bridge tool call failed")
		} else {
			logger.Info("bridge tool call completed", "status", result.output.GetStatus().String())
		}
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

func (b *ToolBridge) writeProtoJSON(conn *websocket.Conn, payload proto.Message) error {
	data, err := protojson.Marshal(payload)
	if err != nil {
		return err
	}
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (b *ToolBridge) readLoop(ctx context.Context, conn *websocket.Conn, connSeq uint64) {
	logger := logs.FromContext(ctx)
	for {
		req, err := readWebsocketRequest(conn)
		if err != nil {
			reason := disconnectReason(err)
			logger.Info("codex bridge websocket disconnected", "connectionSeq", connSeq, "error", err.Error(), "reason", reason)
			b.handleDisconnect(conn, err)
			return
		}
		response := req.GetNotebookToolCallResponse()
		if response == nil {
			continue
		}

		if response.GetBridgeCallId() == "" {
			logger.Info("ignoring codex bridge response without bridge_call_id")
			continue
		}
		logger.Info("received codex bridge response", "bridgeCallID", response.GetBridgeCallId())
		b.resolvePending(response)
	}
}

func readWebsocketRequest(conn *websocket.Conn) (*codexv1.WebsocketRequest, error) {
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	msg := &codexv1.WebsocketRequest{}
	switch messageType {
	case websocket.TextMessage:
		if err := protojson.Unmarshal(data, msg); err != nil {
			return nil, err
		}
	case websocket.BinaryMessage:
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported websocket message type")
	}
	return msg, nil
}

func (b *ToolBridge) resolvePending(response *codexv1.NotebookToolCallResponse) {
	b.mu.Lock()
	pending, ok := b.pending[response.GetBridgeCallId()]
	if ok {
		delete(b.pending, response.GetBridgeCallId())
	}
	b.mu.Unlock()
	if !ok {
		return
	}

	if response.GetError() != "" {
		pending.result <- bridgeCallResult{err: errors.New(response.GetError())}
		return
	}
	output := response.GetOutput()
	if output == nil {
		pending.result <- bridgeCallResult{err: errors.New("bridge response did not include output")}
		return
	}
	pending.result <- bridgeCallResult{output: output}
}

func (b *ToolBridge) handleDisconnect(conn *websocket.Conn, connErr error) {
	observeBridgeDisconnect(disconnectReason(connErr))

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
	observeBridgeDisconnect(disconnectReasonShutdown)

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

func toolNameFromInput(input *toolsv1.ToolCallInput) string {
	if input == nil {
		return ""
	}
	switch {
	case input.GetListCells() != nil:
		return "ListCells"
	case input.GetGetCells() != nil:
		return "GetCells"
	case input.GetUpdateCells() != nil:
		return "UpdateCells"
	case input.GetExecuteCells() != nil:
		return "ExecuteCells"
	default:
		return "Unknown"
	}
}

func disconnectReason(err error) string {
	if err == nil {
		return disconnectReasonShutdown
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "replaced by force_replace"):
		return disconnectReasonReplaced
	case strings.Contains(msg, "server shutdown"):
		return disconnectReasonShutdown
	case websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway):
		return disconnectReasonClient
	default:
		return disconnectReasonReadError
	}
}
