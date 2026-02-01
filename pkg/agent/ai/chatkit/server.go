package chatkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/runmedev/runme/v3/pkg/agent/ai/tools"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	aisreproto "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/pkg/agent/ai"
)

const (
	// aisreChatKitState defines a special event to keep track of chatkit state
	aisreChatKitState = "aisre.chatkit.state"
)

var (
	errChatKitNoUserText = errors.New("no user text provided")
)

type ChatKitHandler struct {
	agent *ai.Agent
}

func NewChatKitHandler(agent *ai.Agent) *ChatKitHandler {
	return &ChatKitHandler{
		agent: agent,
	}
}

func (h *ChatKitHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()
	logger := logs.FromContext(ctx).WithValues("traceId", traceID)
	ctx = logr.NewContext(ctx, logger)
	ctx = obs.NewContextWithPrincipal(ctx)
	// Add the trace-id to the response headers so that in the chrome console we can get the trace-id.
	// This will make it easier to debug why individual requests to the AISRE failed.
	// TODO(jlewi): I asked chatgpt if there was a way to do this with interceptors or existing infra we could reuse
	// but it didn't come up with anything.
	w.Header().Add("trace-id", traceID)
	logger.Info("Handling chatkit request")

	if r.Method != http.MethodPost {
		logger.Info("Method not allowed")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(err, "failed to read request body")
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var base struct {
		Type string `json:"type"`

		// This is injected by AISRE custom fetch
		ChatKitState json.RawMessage `json:"chatkit_state"`
	}
	if err := json.Unmarshal(body, &base); err != nil {
		logger.Error(err, "invalid request payload", "body", string(body))
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	if len(base.ChatKitState) == 0 {
		err := errors.New("chatkit_state is not in the request body; did the request come from the AISRE webapp")
		logger.Error(err, "missing chatkit_state")
		http.Error(w, "missing chatkit_state", http.StatusBadRequest)
		return
	}

	logger.Info("ChatKitHandler.Handle",
		"type", base.Type,
		"chatkitstate", string(base.ChatKitState),
		"body", string(body),
	)

	chatKitState := &aisreproto.ChatkitState{}
	if err := protojson.Unmarshal(base.ChatKitState, chatKitState); err != nil {
		logger.Error(err, "failed to unmarshal chatkit_state")
		http.Error(w, "failed to unmarshal chatkit_state", http.StatusBadRequest)
		return
	}

	oaiAccessToken := r.Header.Get(OpenAIAccessTokenHeader)
	if oaiAccessToken != "" {
		logger.Info("Got OpenAIAccessToken")
	}

	// We start building a generate request is a container for the parameters that affect how the AI gets called
	req := &aisreproto.GenerateRequest{
		OpenaiAccessToken:  oaiAccessToken,
		PreviousResponseId: chatKitState.GetPreviousResponseId(),
	}

	// Chatkit works by having a single chatkit message and then making the payload a union
	// of request types which determines how it should be processed.
	var handleErr error
	switch base.Type {
	case "threads.list":
		h.handleThreadsList(w)
	case "threads.get_by_id":
		h.handleThreadsGetByID(w, body)
	case "items.list":
		h.handleItemsList(w, body)
	case threadsCreateReqType:
		sse, err := newChatKitSSE(w)
		if err != nil {
			logger.Error(err, "failed to create SSE")
			http.Error(w, "failed to create SSE", http.StatusInternalServerError)
			return
		}
		handleErr = h.handleThreadsCreate(ctx, sse, body, req)
	case threadsAdduserMessageReqType:
		sse, err := newChatKitSSE(w)
		if err != nil {
			logger.Error(err, "failed to create SSE")
			http.Error(w, "failed to create SSE", http.StatusInternalServerError)
			return
		}
		handleErr = h.handleThreadsAddUserMessage(ctx, sse, body, req)
	case threadsAddClientToolOutputReqType:
		sse, err := newChatKitSSE(w)
		if err != nil {
			logger.Error(err, "failed to create SSE")
			http.Error(w, "failed to create SSE", http.StatusInternalServerError)
			return
		}
		handleErr = h.handleThreadsAddClientToolOutput(ctx, sse, body, req)
	default:
		logger.Info("Unsupported chatkit request type", "type", base.Type)
		http.Error(w, fmt.Sprintf("unsupported chatkit request type %q", base.Type), http.StatusBadRequest)
	}

	if handleErr != nil {
		logger.Error(handleErr, "failed to handle request", "type", base.Type, "body", string(body))
		hErr := &HTTPError{}
		if errors.Is(handleErr, &HTTPError{}) && errors.As(handleErr, &hErr) {
			http.Error(w, hErr.Message, hErr.Code)
		} else {
			http.Error(w, handleErr.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *ChatKitHandler) handleThreadsList(w http.ResponseWriter) {
	// TODO(https://linear.app/openai/issue/AISRE-122/ui-store-threadsconversations-using-conversations-api):
	// Could we use the ConversationsAPI to store these
	page := ThreadsPage{
		Data:    []Thread{},
		HasMore: false,
	}
	writeChatKitJSON(w, page)
}

func (h *ChatKitHandler) handleThreadsGetByID(w http.ResponseWriter, _ []byte) {
	// We currently don't persist threads anywhere so just return 404 since no one should be looking up threads
	// by id.
	// TODO(jlewi): In the future we might store these using the conversations API
	http.Error(w, "thread not found", http.StatusNotFound)
}

func (h *ChatKitHandler) handleItemsList(w http.ResponseWriter, _ []byte) {
	// We currently don't persist threads anywhere so just return 404 since no one should be looking up threads
	// by id.
	// TODO(jlewi): In the future we might store these using the conversations API
	http.Error(w, "thread not found", http.StatusNotFound)
}

func (h *ChatKitHandler) handleThreadsCreate(ctx context.Context, sse *chatKitSSE, body []byte, genReq *aisreproto.GenerateRequest) error {
	// When a thread is created we just start a conversation with that message since we don't persist threads anywhere.
	threadCreateReq := &ThreadsCreateReq{}
	if err := json.Unmarshal(body, &threadCreateReq); err != nil {
		return NewHTTPError(http.StatusBadRequest, "invalid request payload")
	}

	threadID := "thread_" + uuid.NewString()
	h.streamConversation(ctx, sse, threadID, threadCreateReq.Params.Input, true, genReq)
	return nil
}

func (h *ChatKitHandler) handleThreadsAddUserMessage(ctx context.Context, sse *chatKitSSE, body []byte, genReq *aisreproto.GenerateRequest) error {
	req := &ThreadsAddUserMessageReq{}
	if err := json.Unmarshal(body, &req); err != nil {
		return NewHTTPError(http.StatusBadRequest, "invalid request payload")
	}

	h.streamConversation(ctx, sse, req.Params.ThreadID, req.Params.Input, false, genReq)
	return nil
}

func (h *ChatKitHandler) handleThreadsAddClientToolOutput(ctx context.Context, sse *chatKitSSE, body []byte, genReq *aisreproto.GenerateRequest) error {
	req := &ThreadsAddClientToolOutputReq{}
	if err := json.Unmarshal(body, &req); err != nil {
		return NewHTTPError(http.StatusBadRequest, "invalid request payload")

	}

	if req.Params.ThreadID == "" {
		return NewHTTPError(http.StatusBadRequest, "thread_id is required")
	}

	output := &aisreproto.ToolCallOutput{}
	if err := protojson.Unmarshal(req.Params.Result, output); err != nil {
		return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid ToolOutput; result is not a ToolCallOutput; threadID: %s", req.Params.ThreadID))
	}

	logger := logr.FromContext(ctx).WithValues(
		"threadID", req.Params.ThreadID,
		"callID", output.CallId,
		"previousResponseID", output.PreviousResponseId,
	)
	ctx = logr.NewContext(ctx, logger)
	logger.Info("handleThreadsAddClientToolOutput")

	streamErr := h.generateAssistantResponse(ctx, req.Params.ThreadID, genReq, output, sse.send)
	if streamErr != nil {
		logger.Error(streamErr, "assistant follow-up after tool output failed")
		return sse.send(ctx, ErrorEvent{
			Type:       errorEventType,
			Code:       "generation_failed",
			Message:    "The assistant could not continue after the tool output.",
			AllowRetry: true,
		})
	}

	return nil
}

func (h *ChatKitHandler) streamConversation(ctx context.Context, sse *chatKitSSE, threadID string, input UserMessageInput, includeThreadEvent bool, req *aisreproto.GenerateRequest) {
	logger := logr.FromContext(ctx).WithValues("threadID", threadID)
	ctx = logr.NewContext(ctx, logger)

	if includeThreadEvent {
		thread := Thread{
			ID:        threadID,
			Title:     "AISRE thread",
			CreatedAt: time.Now(),
		}
		if err := sse.send(ctx, ThreadCreatedEvent{
			Type:   threadCreatedEventType,
			Thread: thread,
		}); err != nil {
			return
		}
	}

	userItem, text, err := newUserMessageItem(threadID, input)
	if err != nil {
		logger.Error(err, "failed to build user message")
		_ = sse.send(ctx, ErrorEvent{
			Type:       errorEventType,
			Code:       "invalid_input",
			Message:    "Your message could not be processed.",
			AllowRetry: false,
		})
		return
	}

	// TODO(jlewi): Do we really need to send user item events back to the client

	addEvent := NewThreadItemAddedEvent()
	addEvent.Item = *userItem
	if err := sse.send(ctx, addEvent); err != nil {
		return
	}
	doneEvent := NewThreadItemDoneEvent()
	doneEvent.Item = *userItem
	if err := sse.send(ctx, doneEvent); err != nil {
		return
	}

	req.Message = text
	req.Model = input.InferenceOptions.Model

	streamErr := h.generateAssistantResponse(ctx, threadID, req, nil, sse.send)
	if streamErr != nil {
		logger := logr.FromContext(ctx)
		logger.Error(streamErr, "assistant generation failed")
		_ = sse.send(ctx, ErrorEvent{
			Type:       errorEventType,
			Code:       "generation_failed",
			Message:    "The assistant could not generate a response. Try again.",
			AllowRetry: true,
		})
		return
	}
}

// generateAssistantResponse creates a response.
func (h *ChatKitHandler) generateAssistantResponse(ctx context.Context, threadID string, req *aisreproto.GenerateRequest, toolCallOutput *aisreproto.ToolCallOutput, emit EventSender) error {
	if req == nil {
		return NewHTTPError(http.StatusInternalServerError, "GenerateRequest is nil")
	}

	req.Context = aisreproto.GenerateRequest_CONTEXT_WEBAPP

	if req.Model == "" {
		req.Model = openai.ChatModelGPT4oMini
	}

	createResponse, opts, err := h.agent.BuildResponseParams(ctx, req)

	if err != nil {
		return err
	}

	if toolCallOutput != nil {
		if req.PreviousResponseId != "" && toolCallOutput.PreviousResponseId != req.PreviousResponseId {
			return errors.Errorf("req previous response id %s != toolcall previous response id %s", req.PreviousResponseId, toolCallOutput.PreviousResponseId)
		}

		if err := tools.AddToolCallOutputToResponse(ctx, toolCallOutput, createResponse); err != nil {
			return err
		}
	}

	if req.PreviousResponseId != "" {
		createResponse.PreviousResponseID = openai.Opt(req.PreviousResponseId)
	}

	if toolCallOutput != nil && toolCallOutput.PreviousResponseId != "" {
		createResponse.PreviousResponseID = openai.Opt(toolCallOutput.PreviousResponseId)
	}

	// Right now chatKit can only handle one tool call at a time. So disable parallel toolCalls.
	createResponse.ParallelToolCalls = openai.Opt(false)

	toolCallOutputCallID := ""
	if toolCallOutput != nil {
		toolCallOutputCallID = toolCallOutput.CallId
	}
	logger := logr.FromContext(ctx).WithValues(
		"threadID", threadID,
		"toolCallOutputCallID", toolCallOutputCallID,
		"hasOpenAIAccessToken", req.OpenaiAccessToken != "",
	)
	ctx = logr.NewContext(ctx, logger)
	logger.Info("Creating response", "previousResponseId", createResponse.PreviousResponseID)
	stream := h.agent.Client.Responses.NewStreaming(ctx, *createResponse, opts...)
	if err := StreamResponseEvents(ctx, threadID, stream, emit); err != nil {
		return err
	}
	return nil
}

func writeChatKitJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// --- Helpers ---

func newUserMessageItem(threadID string, input UserMessageInput) (*UserMessageItem, string, error) {
	text := firstContentText(input.Content)
	if text == "" {
		return nil, "", errChatKitNoUserText
	}
	now := time.Now().UTC()
	return &UserMessageItem{
		Type: userMessageType,
		ThreadItemBase: ThreadItemBase{
			ID:        "msg_" + uuid.NewString(),
			ThreadID:  threadID,
			CreatedAt: now,
		},
		Content:          cloneUserContent(input.Content),
		Attachments:      []Attachment{},
		QuotedText:       input.QuotedText,
		InferenceOptions: input.InferenceOptions,
	}, text, nil
}

func firstContentText(contents []UserMessageContent) string {
	for _, c := range contents {
		if c.Type == userTextContentType && strings.TrimSpace(c.Text) != "" {
			return c.Text
		}
	}
	return ""
}

func cloneUserContent(contents []UserMessageContent) []UserMessageContent {
	if len(contents) == 0 {
		return []UserMessageContent{}
	}
	cloned := make([]UserMessageContent, len(contents))
	for i, c := range contents {
		copy := c
		if c.Data != nil {
			copy.Data = make(map[string]any, len(c.Data))
			for k, v := range c.Data {
				copy.Data[k] = v
			}
		}
		cloned[i] = copy
	}
	return cloned
}
