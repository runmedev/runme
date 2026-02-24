package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/pkg/agent/ai/chatkit"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	codexSessionTokenHeader = "X-Runme-Codex-Session-Token"

	reqTypeThreadsCreate              = "threads.create"
	reqTypeThreadsList                = "threads.list"
	reqTypeThreadsGetByID             = "threads.get_by_id"
	reqTypeItemsList                  = "items.list"
	reqTypeThreadsAddUserMessage      = "threads.add_user_message"
	reqTypeThreadsAddClientToolOutput = "threads.add_client_tool_output"

	threadCreatedEventType = "thread.created"
	assistantMessageType   = "assistant_message"
	outputTextType         = "output_text"
	aisreChatKitState      = "aisre.chatkit.state"
)

type ChatKitAdapter struct {
	fallback       http.Handler
	processManager codexProcessManager
	tokenManager   *SessionTokenManager
}

type codexProcessManager interface {
	EnsureStarted(ctx context.Context) error
	ConfigureSession(ctx context.Context, cfg SessionConfig) error
	RunTurn(ctx context.Context, req TurnRequest) (*TurnResponse, error)
	Interrupt(ctx context.Context, sessionID, threadID string) error
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

type chatkitBaseRequest struct {
	Type         string          `json:"type"`
	ChatKitState json.RawMessage `json:"chatkit_state"`
}

type TurnRequest struct {
	SessionID          string
	ThreadID           string
	PreviousResponseID string
	Input              *chatkit.UserMessageInput
	ToolOutput         *toolsv1.ToolCallOutput
}

type TurnResponse struct {
	ThreadID           string      `json:"thread_id,omitempty"`
	PreviousResponseID string      `json:"previous_response_id,omitempty"`
	Events             []TurnEvent `json:"events,omitempty"`
}

type TurnEvent struct {
	Type       string `json:"type,omitempty"`
	ItemID     string `json:"item_id,omitempty"`
	Text       string `json:"text,omitempty"`
	Icon       string `json:"icon,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	AllowRetry bool   `json:"allow_retry,omitempty"`
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

	body, err := readRequestBody(r)
	if err != nil {
		logger.Error(err, "failed to read request body")
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	restoreRequestBody(r, body)

	sessionID := extractSessionID(body)
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

	base := &chatkitBaseRequest{}
	if err := json.Unmarshal(body, base); err != nil {
		logger.Error(err, "invalid request payload")
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	chatKitState := &toolsv1.ChatkitState{}
	if len(base.ChatKitState) > 0 {
		if err := protojson.Unmarshal(base.ChatKitState, chatKitState); err != nil {
			logger.Error(err, "failed to unmarshal chatkit_state")
			http.Error(w, "failed to unmarshal chatkit_state", http.StatusBadRequest)
			return
		}
	}

	switch base.Type {
	case reqTypeThreadsList:
		writeChatKitJSON(w, chatkit.ThreadsPage{Data: []chatkit.Thread{}, HasMore: false})
	case reqTypeThreadsGetByID, reqTypeItemsList:
		http.Error(w, "thread not found", http.StatusNotFound)
	case reqTypeThreadsCreate:
		req := &chatkit.ThreadsCreateReq{}
		if err := json.Unmarshal(body, req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		h.handleTurnRequest(w, r, TurnRequest{
			SessionID: sessionID,
			ThreadID:  sessionID,
			Input:     &req.Params.Input,
		}, true)
	case reqTypeThreadsAddUserMessage:
		req := &chatkit.ThreadsAddUserMessageReq{}
		if err := json.Unmarshal(body, req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.Params.ThreadID == "" {
			http.Error(w, "thread_id is required", http.StatusBadRequest)
			return
		}
		h.handleTurnRequest(w, r, TurnRequest{
			SessionID:          sessionID,
			ThreadID:           req.Params.ThreadID,
			PreviousResponseID: chatKitState.GetPreviousResponseId(),
			Input:              &req.Params.Input,
		}, false)
	case reqTypeThreadsAddClientToolOutput:
		req := &chatkit.ThreadsAddClientToolOutputReq{}
		if err := json.Unmarshal(body, req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}
		if req.Params.ThreadID == "" {
			http.Error(w, "thread_id is required", http.StatusBadRequest)
			return
		}
		output := &toolsv1.ToolCallOutput{}
		if err := protojson.Unmarshal(req.Params.Result, output); err != nil {
			http.Error(w, "invalid ToolOutput; result is not a ToolCallOutput", http.StatusBadRequest)
			return
		}
		h.handleTurnRequest(w, r, TurnRequest{
			SessionID:          sessionID,
			ThreadID:           req.Params.ThreadID,
			PreviousResponseID: chatKitState.GetPreviousResponseId(),
			ToolOutput:         output,
		}, false)
	default:
		if h.fallback != nil {
			logger.Info("dispatching unsupported request type via fallback", "type", base.Type)
			h.fallback.ServeHTTP(w, r)
			return
		}
		http.Error(w, fmt.Sprintf("unsupported chatkit request type %q", base.Type), http.StatusBadRequest)
	}
}

func extractSessionID(body []byte) string {
	if len(body) == 0 {
		return uuid.NewString()
	}

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

func (h *ChatKitAdapter) handleTurnRequest(w http.ResponseWriter, r *http.Request, req TurnRequest, includeThreadEvent bool) {
	logger := logs.FromContext(r.Context()).WithValues("threadID", req.ThreadID)
	if h.processManager == nil {
		logger.Info("missing process manager")
		http.Error(w, "codex adapter has no process manager", http.StatusInternalServerError)
		return
	}

	sse, err := newAdapterSSE(w)
	if err != nil {
		logger.Error(err, "failed to create SSE")
		http.Error(w, "failed to create SSE", http.StatusInternalServerError)
		return
	}

	resp, err := h.processManager.RunTurn(r.Context(), req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			interruptCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if interruptErr := h.processManager.Interrupt(interruptCtx, req.SessionID, req.ThreadID); interruptErr != nil {
				logger.Error(interruptErr, "failed to interrupt canceled codex turn")
			} else {
				logger.Info("interrupted canceled codex turn")
			}
		}
		logger.Error(err, "codex turn failed")
		_ = sse.send(r.Context(), errorEvent("codex_turn_failed", err.Error(), true))
		return
	}

	threadID := req.ThreadID
	if resp != nil && resp.ThreadID != "" {
		threadID = resp.ThreadID
	}
	if includeThreadEvent {
		if err := sse.send(r.Context(), threadCreatedEvent(threadID)); err != nil {
			return
		}
	}
	if resp != nil && resp.PreviousResponseID != "" {
		if err := sse.send(r.Context(), stateEvent(threadID, resp.PreviousResponseID)); err != nil {
			return
		}
	}
	if resp == nil {
		return
	}

	for _, event := range resp.Events {
		switch event.Type {
		case "", "assistant_message":
			itemID := event.ItemID
			if itemID == "" {
				itemID = "item_" + uuid.NewString()
			}
			payload := assistantMessageItem(threadID, itemID, event.Text)
			if err := sse.send(r.Context(), threadItemAddedEvent(payload)); err != nil {
				return
			}
			if err := sse.send(r.Context(), threadItemDoneEvent(payload)); err != nil {
				return
			}
		case "progress", "progress_update":
			progress := chatkit.NewProgressUpdateEvent()
			progress.Icon = event.Icon
			progress.Text = event.Text
			if err := sse.send(r.Context(), progress); err != nil {
				return
			}
		case "error":
			if err := sse.send(r.Context(), errorEvent(event.Code, event.Message, event.AllowRetry)); err != nil {
				return
			}
		}
	}

	_ = sse.send(r.Context(), threadItemDoneEvent(chatkit.NewEndOfTurnItem()))
}

func writeChatKitJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

type adapterSSE struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

func newAdapterSSE(w http.ResponseWriter) (*adapterSSE, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported by server")
	}
	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache, no-store")
	headers.Set("Connection", "keep-alive")
	return &adapterSSE{writer: w, flusher: flusher}, nil
}

func (s *adapterSSE) send(ctx context.Context, payload any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte("\n\n")); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func threadCreatedEvent(threadID string) chatkit.ThreadCreatedEvent {
	return chatkit.ThreadCreatedEvent{
		Type: threadCreatedEventType,
		Thread: chatkit.Thread{
			ID:        threadID,
			Title:     "Codex thread",
			CreatedAt: time.Now().UTC(),
			Status:    chatkit.ThreadStatus{Type: "active"},
			Items: chatkit.ThreadItemsPage{
				Data:    []chatkit.ThreadItem{},
				HasMore: false,
			},
		},
	}
}

func stateEvent(threadID, previousResponseID string) chatkit.AISREChatkitEvent {
	state := &toolsv1.ChatkitState{
		ThreadId:           threadID,
		PreviousResponseId: previousResponseID,
	}
	stateBytes, _ := protojson.Marshal(state)
	return chatkit.AISREChatkitEvent{
		Type: aisreChatKitState,
		Item: json.RawMessage(stateBytes),
	}
}

func assistantMessageItem(threadID, itemID, text string) chatkit.AssistantMessageItem {
	return chatkit.AssistantMessageItem{
		ThreadItemBase: chatkit.ThreadItemBase{
			ID:        itemID,
			ThreadID:  threadID,
			CreatedAt: time.Now().UTC(),
		},
		Type: assistantMessageType,
		Content: []chatkit.AssistantMessageContent{
			{
				Type:        outputTextType,
				Text:        text,
				Annotations: []chatkit.Annotation{},
			},
		},
	}
}

func errorEvent(code, message string, allowRetry bool) chatkit.ErrorEvent {
	event := chatkit.NewErrorEvent()
	event.Code = code
	event.Message = message
	event.AllowRetry = allowRetry
	return event
}

func threadItemAddedEvent(item chatkit.ThreadItem) chatkit.ThreadItemAddedEvent {
	event := chatkit.NewThreadItemAddedEvent()
	event.Item = item
	return event
}

func threadItemDoneEvent(item chatkit.ThreadItem) chatkit.ThreadItemDoneEvent {
	event := chatkit.NewThreadItemDoneEvent()
	event.Item = item
	return event
}

// PrepareSessionToken issues a token for a session without handling a chatkit request.
func (h *ChatKitAdapter) PrepareSessionToken(ctx context.Context, sessionID string) (string, error) {
	if h.tokenManager == nil {
		return "", nil
	}
	return h.tokenManager.Issue(sessionID)
}
