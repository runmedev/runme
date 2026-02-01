package chatkit

import (
	"context"
	"encoding/json"

	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/openai/openai-go/v2/packages/ssestream"
	"github.com/openai/openai-go/v2/responses"
	oaiconstants "github.com/openai/openai-go/v2/shared/constant"
	"github.com/pkg/errors"
	"github.com/runmedev/runme/v3/pkg/agent/ai/tools"
	"go.openai.org/lib/oaigo/telemetry/oailog"
	aisreproto "go.openai.org/oaiproto/aisre"
	"google.golang.org/protobuf/encoding/protojson"
)

type EventSender func(context.Context, ThreadStreamEvent) error

// StreamResponseEvents consumes the OpenAI response stream, emitting ChatKit
// events via sender.
func StreamResponseEvents(
	ctx context.Context,
	threadID string,
	stream *ssestream.Stream[responses.ResponseStreamEventUnion],
	sender EventSender,
) error {
	builder := &responseStreamBuilder{
		threadID:  threadID,
		sender:    sender,
		toolState: make(map[string]*ClientToolCallItem),
	}

	for stream.Next() {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return connect.NewError(connect.CodeDeadlineExceeded, errors.Wrap(err, "request context cancelled"))
			}
			return err
		default:
			if err := builder.handleEvent(ctx, stream.Current()); err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrap(err, "error processing response stream event"))
			}
		}
	}

	if err := stream.Err(); err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrap(err, "error reading response stream"))
	}

	return nil
}

type responseStreamBuilder struct {
	threadID string
	sender   EventSender

	// responseID is the id of the response
	// We need to pass this along to the client so it can be provided on the next request
	responseID string

	message         *AssistantMessageItem
	messageBuffer   strings.Builder
	messageAdded    bool
	messageFinished bool

	// toolState keeps track of the toolcalls
	// currently there can only be one toolcall per turn.
	// It should be a map from itemID to the toolCallState.
	toolState map[string]*ClientToolCallItem

	contentIndex int64
}

type AISREChatkitEvent struct {
	Type string          `json:"type"`
	Item json.RawMessage `json:"item"`
}

func (e AISREChatkitEvent) isThreadStreamEvent() {}

func (b *responseStreamBuilder) handleEvent(ctx context.Context, e responses.ResponseStreamEventUnion) error {
	ctx = oailog.WithContextAttrs(ctx, oailog.String("threadID", b.threadID))
	// Only set response ID if it isn't set
	if id := e.Response.ID; id != "" && b.responseID == "" {
		// Sent the responseID and send an en event message
		b.responseID = id

		state := &aisreproto.ChatkitState{
			ThreadId:           b.threadID,
			PreviousResponseId: b.responseID,
		}
		stateBytes, err := protojson.Marshal(state)
		if err != nil {
			return errors.Wrap(err, "failed to serialize chat kit state")
		}
		raw := json.RawMessage(stateBytes)
		// We send a special type of event to store the thread and response id on the client.
		event := AISREChatkitEvent{
			Type: aisreChatKitState,
			Item: raw,
		}
		if err := b.sender(ctx, event); err != nil {
			return errors.Wrap(err, "failed to send chat kit state")
		}
	}
	ctx = oailog.WithContextAttrs(ctx, oailog.String("responseID", b.responseID))

	oailog.Info(ctx, "Handle event", oailog.Any("eventType", e.Type))
	switch event := e.AsAny().(type) {
	case responses.ResponseTextDeltaEvent:
		return b.handleTextDelta(ctx, e.AsResponseOutputTextDelta())
	case responses.ResponseCompletedEvent:
		return b.handleCompleted(ctx, e)
	case responses.ResponseOutputItemAddedEvent:
		return b.handleOutputItemAdded(ctx, event)
	case responses.ResponseOutputItemDoneEvent:
		return b.handleOutputItemDone(ctx, event)
	case responses.ResponseFunctionCallArgumentsDeltaEvent:
		// We don't do anything with function call argument deltas event
		return nil
	case responses.ResponseFunctionCallArgumentsDoneEvent:
		return b.handleFunctionArgumentsDone(ctx, e.AsResponseFunctionCallArgumentsDone())
	default:
		return nil
	}
}

func (b *responseStreamBuilder) handleTextDelta(ctx context.Context, delta responses.ResponseTextDeltaEvent) error {
	u := AssistantMessageContentPartTextDelta{
		Type:         AssistantMessageContentPartTextDeltaType,
		ContentIndex: b.contentIndex,
		Delta:        delta.Delta,
	}

	e := ThreadItemUpdated{
		Type:   threadItemUpdatedType,
		ItemID: delta.ItemID,
		Update: u,
	}

	return b.sender(ctx, e)
}

func (b *responseStreamBuilder) handleCompleted(ctx context.Context, e responses.ResponseStreamEventUnion) error {
	oailog.Info(ctx, "Response completed",
		oailog.String("threadID", b.threadID),
		oailog.String("responseID", b.responseID),
	)
	if b.message == nil || b.messageFinished {
		return nil
	}

	b.checkCallIDsUnchanged(ctx, e)

	// TODO(jlewi): I'm not really sure if endofturn should be sent as threadItemDoneEvent but who knows.
	chatEvent := ThreadItemDoneEvent{
		Type: threadItemDoneType,
		Item: NewEndOfTurnItem(),
	}
	// Notify chatkit the turn has ended.
	if err := b.sender(ctx, chatEvent); err != nil {
		return err
	}
	b.messageFinished = true
	return nil
}

func (b *responseStreamBuilder) handleOutputItemAdded(ctx context.Context, event responses.ResponseOutputItemAddedEvent) error {
	// Reset messageAdded; this is a bit of a hack.
	b.messageAdded = false
	oailog.Info(ctx, "Response item added",
		oailog.String("threadID", b.threadID),
		oailog.String("responseID", b.responseID),
		oailog.Any("type", event.Type),
		oailog.String("itemType", event.Item.Type),
	)

	var fSearchCall oaiconstants.FileSearchCall
	var functionCall oaiconstants.FunctionCall
	var customToolCall oaiconstants.CustomToolCall
	var messageType oaiconstants.Message
	progressEvent := NewProgressUpdateEvent()
	progressEvent.Icon = "sparkle"

	switch event.Item.Type {
	case string(messageType.Default()):
		// Looks like not much happens with the assistant message ThreadItemAddedEvent
		// https://github.com/openai/openai/blob/bc4df8345fd83949b6996cc2fa6deb5475b8b4e5/project/chatkit-web-inner/src/stores/ChatStore.tsx#L1729
		assistantItem := AssistantMessageItem{
			ThreadItemBase: ThreadItemBase{
				ID:        event.Item.ID,
				ThreadID:  b.threadID,
				CreatedAt: time.Now(),
			},
			Type:    assistantMessageType,
			Content: []AssistantMessageContent{},
		}

		chatEvent := NewThreadItemAddedEvent()
		chatEvent.Item = assistantItem
		if err := b.sender(ctx, chatEvent); err != nil {
			return err
		}

		// Add a content part maybe we should wait for text delta and see if we've already done it
		// https://github.com/openai/openai/blob/bc4df8345fd83949b6996cc2fa6deb5475b8b4e5/project/chatkit-web-inner/src/stores/ChatStore.tsx#L1907
		assistantAdd := NewAssistantMessageContentPartAdded()

		assistantAdd.ContentIndex = b.contentIndex
		assistantAdd.Content = AssistantMessageContent{
			Type:        assistantMessageContentType,
			Text:        "",
			Annotations: []Annotation{},
		}
		update := NewThreadItemUpdated()
		update.ItemID = event.Item.ID
		update.Update = assistantAdd

		if err := b.sender(ctx, update); err != nil {
			return err
		}
	case string(functionCall.Default()), string(customToolCall.Default()):
		progressEvent.Text = "generating toolcall " + event.Item.Name

		state := &ClientToolCallItem{
			ThreadItemBase: ThreadItemBase{
				ID:        event.Item.ID,
				ThreadID:  b.threadID,
				CreatedAt: time.Now().UTC(),
			},
			Type:   clientToolCallType,
			Status: "pending",
			Name:   event.Item.Name,
			CallID: event.Item.CallID,
		}

		b.toolState[event.Item.ID] = state
	case string(fSearchCall.Default()):
		progressEvent.Text = "searching internal knowledge"
	}

	if progressEvent.Text != "" {
		if err := b.sender(ctx, progressEvent); err != nil {
			return err
		}
	}
	return nil
}

func (b *responseStreamBuilder) handleOutputItemDone(ctx context.Context, event responses.ResponseOutputItemDoneEvent) error {
	// Reset messageAdded; this is a bit of a hack.
	b.messageAdded = false
	oailog.Info(ctx, "Response item done",
		oailog.String("threadID", b.threadID),
		oailog.String("responseID", b.responseID),
		oailog.Any("type", event.Type),
		oailog.String("itemType", event.Item.Type),
	)

	var messageType oaiconstants.Message

	switch event.Item.Type {
	case string(messageType.Default()):
		// We need to send an update done message
		// https://github.com/openai/openai/blob/bc4df8345fd83949b6996cc2fa6deb5475b8b4e5/project/chatkit-web-inner/src/stores/ChatStore.tsx#L1929
		contentDone := NewAssistantMessageContentPartDone()
		contentDone.ContentIndex = b.contentIndex
		contentDone.Content = AssistantMessageContent{
			Type:        assistantMessageContentType,
			Text:        "",
			Annotations: []Annotation{},
		}
		for _, o := range event.Item.Content {
			contentDone.Content.Text += o.Text

			// Handling annotations when an output item is done seems to work pretty well. In particular
			// I think annotations get added when the output item is done; i.e. at the end of some block of text.
			// I tried also handling responses.ResponseOutputTextAnnotationAddedEvent but that didn't seem to help.
			// It looks like OutputTextDelta events don't have annotations.
			for _, a := range o.Annotations {
				annotation := maybeConvertAnnotation(a)
				if annotation != nil {
					contentDone.Content.Annotations = append(contentDone.Content.Annotations, *annotation)
				}
			}
		}

		// Frontend uses an update event
		// https://github.com/openai/openai/blob/bc4df8345fd83949b6996cc2fa6deb5475b8b4e5/project/chatkit-web-inner/src/stores/ChatStore.tsx#L1929
		chatEvent := NewThreadItemUpdated()
		chatEvent.ItemID = event.Item.ID
		chatEvent.Update = contentDone
		if err := b.sender(ctx, chatEvent); err != nil {
			return err
		}
	}

	return nil
}

func (b *responseStreamBuilder) handleFunctionArgumentsDone(ctx context.Context, event responses.ResponseFunctionCallArgumentsDoneEvent) error {
	state, ok := b.toolState[event.ItemID]
	if !ok {
		return errors.Errorf("No toolcall was initialized for itemID: %s", event.ItemID)
	}

	if state.CallID == "" {
		oailog.Error(ctx, "CallID not set on toolCall", oailog.String("itemID", event.ItemID))
	}

	// OpenAI JSON needs to be converted to proto json
	callInput, err := tools.ArgsToToolCallInput(ctx, state.Name, state.CallID, event.Arguments)

	if err != nil {
		return err
	}

	// We need to wrap the ToolCall with our message which includes the response id and call_id
	// Because the response id gets sent to the frontend and then passed back.
	callInput.PreviousResponseId = b.responseID

	// Now serialize the proto to a map so that we can send it to Chatkit.
	state.Arguments, err = tools.ProtoToMap(callInput)
	if err != nil {
		return err
	}

	// N.B. I don't think we want to emit a threadItemAddedType for function calls because I think both
	// threadItemAddedType and done events will trigger the tool call in the client which leads to double calling the
	// tool  which leads to two inflight requests. The client can't do much with a partial toolcall so we just wait
	// for it to complete before sending it.

	// Emit a ClientToolCallItem when we are done with the function arguments.
	doneEvent := NewThreadItemDoneEvent()
	doneEvent.Item = *state
	if err := b.sender(ctx, doneEvent); err != nil {
		return err
	}
	return nil
}

func (b *responseStreamBuilder) checkCallIDsUnchanged(ctx context.Context, e responses.ResponseStreamEventUnion) {
	// Has the call_id changed?
	// This appears to be happening due to a bug in the responses API.
	// We try to detect it and log it; I'm not sure how we could fix it because we don't want to wait till the
	// end of the response to stream toolcalls to the client
	outputs := e.Response.Output
	for _, output := range outputs {
		// N.B. We assume for now that call id is only set for functional calls so if there is no id
		// then its not a function call.
		if output.CallID == "" {
			continue
		}

		itemID := output.ID

		if itemID == "" {
			oailog.Info(ctx, "ResponseCompletedEvent has output with callID but not itemID", oailog.String("callID", output.CallID))
			continue
		}

		toolCall, ok := b.toolState[itemID]
		if !ok {
			oailog.Info(ctx, "No toolCall for item", oailog.String("itemID", itemID), oailog.String("callID", output.CallID))
			continue
		}

		if toolCall.CallID != output.CallID {
			// Log an error because the subsequent response will fail because the callID won't be found.
			oailog.Error(ctx, "CallID changed",
				oailog.String("itemID", itemID),
				oailog.String("oldCallID", toolCall.CallID),
				oailog.String("newCallID", output.CallID),
			)
		}
	}
}
