package codex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	defaultThreadResumeMethod = "thread/resume"
)

type proxyProcessManager interface {
	EnsureStarted(ctx context.Context) error
	InitializeResult() json.RawMessage
	CallRaw(ctx context.Context, method string, params any, onNotification func(jsonRPCNotification) error) (json.RawMessage, error)
}

type proxyTokenManager interface {
	Issue() (string, error)
}

type AppServerProxyHandler struct {
	processManager proxyProcessManager
	tokenManager   proxyTokenManager
	upgrader       websocket.Upgrader
}

type proxyJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type proxyJSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type proxyJSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func NewAppServerProxyHandler(processManager proxyProcessManager, tokenManager proxyTokenManager) (*AppServerProxyHandler, error) {
	if processManager == nil {
		return nil, errors.New("process manager is nil")
	}
	if tokenManager == nil {
		return nil, errors.New("token manager is nil")
	}
	return &AppServerProxyHandler{
		processManager: processManager,
		tokenManager:   tokenManager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}, nil
}

func (h *AppServerProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logs.FromContextWithTrace(ctx).WithValues("component", "codex-app-server-proxy")
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}
	ctx = logr.NewContext(ctx, logger)

	if err := h.processManager.EnsureStarted(ctx); err != nil {
		logger.Error(err, "failed to start codex app-server")
		http.Error(w, "failed to start codex app-server: "+err.Error(), http.StatusBadGateway)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "failed to upgrade codex app-server proxy websocket")
		return
	}
	defer conn.Close()

	token, err := h.tokenManager.Issue()
	if err != nil {
		logger.Error(err, "failed to issue codex mcp token")
		_ = writeProxyResponse(conn, proxyJSONRPCResponse{
			JSONRPC: "2.0",
			Error: &jsonRPCError{
				Code:    -32603,
				Message: "failed to issue codex mcp token",
			},
		})
		return
	}

	writeMu := &sync.Mutex{}
	state := proxyConnectionState{
		initialized: false,
		sessionConfig: SessionConfig{
			SessionID:    defaultSessionTokenScope,
			MCPServerURL: mcpServerURL(r),
			BearerToken:  token,
		},
	}

	for {
		req, err := readProxyRequest(conn)
		if err != nil {
			logger.Error(err, "codex app-server proxy websocket closed")
			return
		}
		if err := h.handleRequest(ctx, conn, writeMu, &state, req); err != nil {
			logger.Error(err, "codex app-server proxy request failed", "method", req.Method)
			if len(req.ID) > 0 {
				_ = writeProxyResponse(conn, proxyJSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &jsonRPCError{
						Code:    -32603,
						Message: err.Error(),
					},
				})
			}
		}
	}
}

type proxyConnectionState struct {
	initialized   bool
	sessionConfig SessionConfig
}

func (h *AppServerProxyHandler) handleRequest(
	ctx context.Context,
	conn *websocket.Conn,
	writeMu *sync.Mutex,
	state *proxyConnectionState,
	req *proxyJSONRPCRequest,
) error {
	if req == nil {
		return errors.New("request is nil")
	}
	if req.JSONRPC != "2.0" {
		return errors.New("jsonrpc must be 2.0")
	}

	if len(req.ID) == 0 {
		switch req.Method {
		case defaultInitializedMethod:
			state.initialized = true
			return nil
		default:
			return nil
		}
	}

	switch req.Method {
	case defaultInitializeMethod:
		result := h.processManager.InitializeResult()
		if len(result) == 0 {
			result = json.RawMessage(`{}`)
		}
		return writeProxyResponse(conn, proxyJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		})
	}

	if !state.initialized {
		return writeProxyResponse(conn, proxyJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jsonRPCError{
				Code:    -32600,
				Message: "Invalid request: browser proxy connection is not initialized",
			},
		})
	}

	params, err := parseProxyParams(req.Params)
	if err != nil {
		return writeProxyResponse(conn, proxyJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jsonRPCError{
				Code:    -32600,
				Message: "Invalid request: " + err.Error(),
			},
		})
	}
	if req.Method == defaultThreadStartMethod || req.Method == defaultThreadResumeMethod {
		params, err = mergeProxyThreadParams(params, state.sessionConfig, req.Method == defaultThreadStartMethod)
		if err != nil {
			return writeProxyResponse(conn, proxyJSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &jsonRPCError{
					Code:    -32600,
					Message: "Invalid request: " + err.Error(),
				},
			})
		}
	}

	result, err := h.processManager.CallRaw(ctx, req.Method, params, func(note jsonRPCNotification) error {
		return writeProxyNotification(conn, writeMu, note)
	})
	if err != nil {
		return err
	}
	return writeProxyResponse(conn, proxyJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

func parseProxyParams(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var params any
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}
	return params, nil
}

func mergeProxyThreadParams(params any, cfg SessionConfig, includeInstructions bool) (map[string]any, error) {
	merged := map[string]any{}
	if params != nil {
		cast, ok := params.(map[string]any)
		if !ok {
			return nil, errors.New("thread params must be an object")
		}
		for k, v := range cast {
			merged[k] = v
		}
	}

	merged["approvalPolicy"] = "never"
	if includeInstructions {
		merged["developerInstructions"] = mergeDeveloperInstructions(merged["developerInstructions"])
	}
	configMap := map[string]any{}
	if existing, ok := merged["config"].(map[string]any); ok {
		for k, v := range existing {
			configMap[k] = v
		}
	}
	for k, v := range buildSessionConfigParams(cfg) {
		configMap[k] = v
	}
	merged["config"] = configMap
	return merged, nil
}

func mergeDeveloperInstructions(value any) string {
	base := strings.TrimSpace(defaultThreadDeveloperInstructions)
	if existing, ok := value.(string); ok && strings.TrimSpace(existing) != "" {
		return strings.TrimSpace(existing) + "\n\n" + base
	}
	return base
}

func readProxyRequest(conn *websocket.Conn) (*proxyJSONRPCRequest, error) {
	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if messageType != websocket.TextMessage {
		return nil, errors.New("proxy only supports text websocket messages")
	}
	req := &proxyJSONRPCRequest{}
	if err := json.Unmarshal(payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

func writeProxyNotification(conn *websocket.Conn, writeMu *sync.Mutex, note jsonRPCNotification) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteJSON(proxyJSONRPCNotification{
		JSONRPC: "2.0",
		Method:  note.Method,
		Params:  note.Params,
	})
}

func writeProxyResponse(conn *websocket.Conn, response proxyJSONRPCResponse) error {
	return conn.WriteJSON(response)
}
