package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	codexSessionTokenHeader = "X-Runme-Codex-Session-Token"
)

type ChatKitAdapter struct {
	fallback       http.Handler
	processManager codexProcessManager
	tokenManager   *SessionTokenManager
}

type codexProcessManager interface {
	EnsureStarted(ctx context.Context) error
	ConfigureSession(ctx context.Context, cfg SessionConfig) error
}

type ChatKitAdapterOptions struct {
	Fallback       http.Handler
	ProcessManager codexProcessManager
	TokenManager   *SessionTokenManager
}

func NewChatKitAdapter(opts ChatKitAdapterOptions) *ChatKitAdapter {
	return &ChatKitAdapter{
		fallback:       opts.Fallback,
		processManager: opts.ProcessManager,
		tokenManager:   opts.TokenManager,
	}
}

func (h *ChatKitAdapter) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logs.FromContextWithTrace(ctx).WithValues("component", "codex-chatkit-adapter")
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}
	ctx = logr.NewContext(ctx, logger)
	r = r.WithContext(ctx)

	if r.Method != http.MethodPost {
		logger.Info("method not allowed", "method", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.processManager != nil {
		if err := h.processManager.EnsureStarted(r.Context()); err != nil {
			logger.Error(err, "failed to start codex app-server")
			http.Error(w, "failed to start codex app-server: "+err.Error(), http.StatusBadGateway)
			return
		}
	}

	sessionID := extractSessionIDAndRestoreBody(r)
	logger = logger.WithValues("sessionID", sessionID)
	ctx = logr.NewContext(ctx, logger)
	r = r.WithContext(ctx)

	token := ""
	if h.tokenManager != nil {
		var err error
		token, err = h.tokenManager.Issue(sessionID)
		if err != nil {
			logger.Error(err, "failed to issue codex session token")
			http.Error(w, "failed to issue codex session token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set(codexSessionTokenHeader, token)
	}
	if h.processManager != nil && token != "" {
		cfg := SessionConfig{
			SessionID:    sessionID,
			MCPServerURL: mcpServerURL(r),
			BearerToken:  token,
		}
		if err := h.processManager.ConfigureSession(r.Context(), cfg); err != nil {
			logger.Error(err, "failed to configure codex session")
			http.Error(w, "failed to configure codex session: "+err.Error(), http.StatusBadGateway)
			return
		}
	}

	if h.fallback == nil {
		logger.Info("missing fallback handler")
		http.Error(w, "codex adapter has no backing handler", http.StatusInternalServerError)
		return
	}
	logger.Info("dispatching chatkit request via fallback handler")
	h.fallback.ServeHTTP(w, r)
}

func extractSessionIDAndRestoreBody(r *http.Request) string {
	if r.Body == nil {
		return uuid.NewString()
	}

	body, err := readRequestBody(r)
	if err != nil {
		return uuid.NewString()
	}
	defer restoreRequestBody(r, body)

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return uuid.NewString()
	}

	if chatkitState, ok := raw["chatkit_state"].(map[string]any); ok {
		if threadID, ok := chatkitState["thread_id"].(string); ok && threadID != "" {
			return threadID
		}
	}
	if params, ok := raw["params"].(map[string]any); ok {
		if threadID, ok := params["thread_id"].(string); ok && threadID != "" {
			return threadID
		}
	}

	return uuid.NewString()
}

func readRequestBody(r *http.Request) ([]byte, error) {
	body := new(bytes.Buffer)
	if _, err := body.ReadFrom(r.Body); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func restoreRequestBody(r *http.Request, body []byte) {
	r.Body = ioNopCloser(bytes.NewReader(body))
}

type nopCloser struct {
	*bytes.Reader
}

func (n nopCloser) Close() error { return nil }

func ioNopCloser(reader *bytes.Reader) nopCloser {
	return nopCloser{Reader: reader}
}

func mcpServerURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		part := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if part != "" {
			scheme = part
		}
	}
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	return fmt.Sprintf("%s://%s/mcp/notebooks", scheme, host)
}

// PrepareSessionToken issues a token for a session without handling a chatkit request.
func (h *ChatKitAdapter) PrepareSessionToken(ctx context.Context, sessionID string) (string, error) {
	if h.tokenManager == nil {
		return "", nil
	}
	return h.tokenManager.Issue(sessionID)
}
