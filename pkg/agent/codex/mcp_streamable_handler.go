package codex

import (
	"context"
	"errors"
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

type contextKey int

const (
	sessionIDContextKey contextKey = iota
	approvedRefIDsContextKey
)

// SessionIDFromContext returns the session id authenticated for an MCP HTTP request.
func SessionIDFromContext(ctx context.Context) string {
	sid, _ := ctx.Value(sessionIDContextKey).(string)
	return sid
}

const executeApprovalHeader = "X-Runme-Codex-Execute-Approved"

func approvedRefIDsFromContext(ctx context.Context) []string {
	ids, _ := ctx.Value(approvedRefIDsContextKey).([]string)
	return ids
}

type StreamableMCPHandler struct {
	inner  http.Handler
	tokens *SessionTokenManager
}

func NewStreamableMCPHandler(bridge *ToolBridge, tokens *SessionTokenManager) (*StreamableMCPHandler, error) {
	if bridge == nil {
		return nil, errors.New("bridge is nil")
	}
	if tokens == nil {
		return nil, errors.New("token manager is nil")
	}
	nbBridge := NewNotebookMCPBridge(bridge)
	nbBridge.SetExecuteApprover(contextExecuteApprover{})
	mcpServer := nbBridge.NewServer()
	streamable := mcpserver.NewStreamableHTTPServer(mcpServer)
	return &StreamableMCPHandler{
		inner:  streamable,
		tokens: tokens,
	}, nil
}

func (h *StreamableMCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logs.FromContextWithTrace(r.Context()).WithValues("component", "codex-mcp-handler")
	if principal := obs.GetPrincipal(r.Context()); principal != "" {
		logger = logger.WithValues("principal", principal)
	}

	sid, err := h.authenticate(r)
	if err != nil {
		logger.Error(err, "mcp request authentication failed")
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	logger = logger.WithValues("sessionID", sid)
	logger.Info("authenticated codex mcp request")
	ctx := context.WithValue(r.Context(), sessionIDContextKey, sid)
	ctx = context.WithValue(ctx, approvedRefIDsContextKey, parseApprovedRefIDs(r.Header.Get(executeApprovalHeader)))
	h.inner.ServeHTTP(w, r.WithContext(ctx))
}

func (h *StreamableMCPHandler) authenticate(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("authorization must use bearer token")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("bearer token is empty")
	}
	return h.tokens.Validate(token)
}

func parseApprovedRefIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
