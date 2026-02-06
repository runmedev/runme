package chatkit

import (
	"encoding/json"
	"time"
)

// See the types defined in
// https://github.com/openai/chatkit-python/blob/8e2d2577080112e21b4912b659bddf6ad967bfa4/chatkit/types.py#L534
// We define interfaces to represent the union types in the data model.
//
// Base classes are defined by types that are then embedded in types representing subclasses.

const (
	threadCreatedEventType = "thread.created"
	threadItemAddedType    = "thread.item.added"
	threadItemDoneType     = "thread.item.done"
	errorEventType         = "error"
	// https://github.com/openai/chatkit-python/blob/8e2d2577080112e21b4912b659bddf6ad967bfa4/chatkit/types.py#L318C20-L318C35
	progressUpdateEventType = "progress_update"
	// https://github.com/openai/chatkit-python/blob/8e2d2577080112e21b4912b659bddf6ad967bfa4/chatkit/types.py#L567
	endOfTurnType               = "end_of_turn"
	userMessageType             = "user_message"
	assistantMessageType        = "assistant_message"
	clientToolCallType          = "client_tool_call"
	userTextContentType         = "input_text"
	assistantMessageContentType = "output_text"
)

type StreamingReq interface {
	isStreamingReq()
}

const threadsCreateReqType = "threads.create"

type ThreadsCreateReq struct {
	Type   string             `json:"type"`
	Params ThreadCreateParams `json:"params"`
}

func (req ThreadsCreateReq) isStreamingReq() {}

type ThreadCreateParams struct {
	Input UserMessageInput `json:"input"`
}

const threadsAdduserMessageReqType = "threads.add_user_message"

type ThreadsAddUserMessageReq struct {
	Type   string                     `json:"type"`
	Params ThreadAddUserMessageParams `json:"params"`
}

func (req ThreadsAddUserMessageReq) isStreamingReq() {}

type ThreadAddUserMessageParams struct {
	ThreadID string           `json:"thread_id"`
	Input    UserMessageInput `json:"input"`
}

const threadsAddClientToolOutputReqType = "threads.add_client_tool_output"

type ThreadsAddClientToolOutputReq struct {
	Type   string                          `json:"type"`
	Params ThreadAddClientToolOutputParams `json:"params"`
}

func (req ThreadsAddClientToolOutputReq) isStreamingReq() {}

type ThreadAddClientToolOutputParams struct {
	ThreadID string `json:"thread_id"`
	// We use a json.RawMessage because we will end up deserializing to proto.
	// Alternatively we could store it as the proto and write a custom json marshal function to properly unmarshal
	// it using protojson.
	Result json.RawMessage `json:"result"`
}

// ThreadItemBase represents the common fields for all thread items.
// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L509C7-L509C21
//
// It should be embedded in thread items.
type ThreadItemBase struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (b ThreadItemBase) isThreadItem() {}

type Thread struct {
	ID        string          `json:"id"`
	Title     string          `json:"title,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	Status    ThreadStatus    `json:"status"`
	Items     ThreadItemsPage `json:"items"`
}

type ThreadStatus struct {
	Type string `json:"type"`
}

type ThreadItemsPage struct {
	Data    []ThreadItem `json:"data"`
	HasMore bool         `json:"has_more"`
	After   *string      `json:"after,omitempty"`
}

type ThreadsPage struct {
	Data    []Thread `json:"data"`
	HasMore bool     `json:"has_more"`
	After   *string  `json:"after,omitempty"`
}

// ThreadItem is a union of different types
// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L580
type ThreadItem interface {
	isThreadItem()
}

type UserMessageItem struct {
	ThreadItemBase
	Type             string               `json:"type"`
	Content          []UserMessageContent `json:"content"`
	Attachments      []Attachment         `json:"attachments"`
	QuotedText       *string              `json:"quoted_text,omitempty"`
	InferenceOptions InferenceOptions     `json:"inference_options"`
}

// AssistantMessageItem
// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L527
type AssistantMessageItem struct {
	ThreadItemBase
	Type    string                    `json:"type"`
	Content []AssistantMessageContent `json:"content"`
}

type ClientToolCallItem struct {
	ThreadItemBase
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	CallID    string         `json:"call_id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Output    interface{}    `json:"output,omitempty"`
}

type ThreadCreatedEvent struct {
	Type   string `json:"type"`
	Thread Thread `json:"thread"`
}

func (ThreadCreatedEvent) isThreadStreamEvent() {}

func NewErrorEvent() ErrorEvent {
	return ErrorEvent{
		Type: errorEventType,
	}
}

type ErrorEvent struct {
	Type       string `json:"type"`
	Code       string `json:"code"`
	Message    string `json:"message,omitempty"`
	AllowRetry bool   `json:"allow_retry"`
}

func (ErrorEvent) isThreadStreamEvent() {}

type UserMessageContent struct {
	Type        string         `json:"type"`
	Text        string         `json:"text,omitempty"`
	ID          string         `json:"id,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Interactive bool           `json:"interactive,omitempty"`
}

type UserMessageInput struct {
	Content          []UserMessageContent `json:"content"`
	Attachments      []string             `json:"attachments"`
	QuotedText       *string              `json:"quoted_text,omitempty"`
	InferenceOptions InferenceOptions     `json:"inference_options"`
}

type InferenceOptions struct {
	ToolChoice *ToolChoice `json:"tool_choice,omitempty"`
	Model      string      `json:"model,omitempty"`
}

type ToolChoice struct {
	ID string `json:"id"`
}

type Attachment struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type AssistantMessageContent struct {
	Type        string       `json:"type"`
	Text        string       `json:"text"`
	Annotations []Annotation `json:"annotations"`
}

type Annotation struct {
	Type   string           `json:"type"`
	Source AnnotationSource `json:"source,omitempty"`
	Index  *int             `json:"index,omitempty"`
}

func NewAnnotation() Annotation {
	return Annotation{
		Type: "annotation",
	}
}

// ThreadStreamEvent is a union of all the different streaming events
// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L344C1-L344C18
type ThreadStreamEvent interface {
	isThreadStreamEvent()
}

func NewThreadItemAddedEvent() ThreadItemAddedEvent {
	return ThreadItemAddedEvent{
		Type: threadItemAddedType,
	}
}

type ThreadItemAddedEvent struct {
	Type string     `json:"type"`
	Item ThreadItem `json:"item"`
}

func (e ThreadItemAddedEvent) isThreadStreamEvent() {}

const (
	threadItemUpdatedType = "thread.item.updated"
)

func NewThreadItemUpdated() ThreadItemUpdated {
	return ThreadItemUpdated{
		Type: threadItemUpdatedType,
	}
}

// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L286
type ThreadItemUpdated struct {
	Type   string           `json:"type"`
	ItemID string           `json:"item_id"`
	Update ThreadItemUpdate `json:"update"`
}

func (e ThreadItemUpdated) isThreadStreamEvent() {
}

type ThreadItemUpdate interface {
	isThreadItemUpdate()
}

const (
	AssistantMessageContentPartTextDeltaType = "assistant_message.content_part.text_delta"
)

type AssistantMessageContentPartTextDelta struct {
	Type         string `json:"type"`
	ContentIndex int64  `json:"content_index"`
	Delta        string `json:"delta"`
}

// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L447
func (e AssistantMessageContentPartTextDelta) isThreadItemUpdate() {
}

const AssistantMessageContentPartAddedType = "assistant_message.content_part.added"

func NewAssistantMessageContentPartAdded() AssistantMessageContentPartAdded {
	return AssistantMessageContentPartAdded{
		Type: AssistantMessageContentPartAddedType,
	}
}

type AssistantMessageContentPartAdded struct {
	Type         string                  `json:"type"`
	ContentIndex int64                   `json:"content_index"`
	Content      AssistantMessageContent `json:"content"`
}

// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L447
func (e AssistantMessageContentPartAdded) isThreadItemUpdate() {
}

const AssistantMessageContentPartDoneType = "assistant_message.content_part.done"

func NewAssistantMessageContentPartDone() AssistantMessageContentPartDone {
	return AssistantMessageContentPartDone{
		Type: AssistantMessageContentPartDoneType,
	}
}

type AssistantMessageContentPartDone struct {
	Type         string                  `json:"type"`
	ContentIndex int64                   `json:"content_index"`
	Content      AssistantMessageContent `json:"content"`
}

// https://github.com/openai/chatkit-python/blob/71966ad024ec89e56ae4767b4d8c7e4fdb652a16/chatkit/types.py#L447
func (e AssistantMessageContentPartDone) isThreadItemUpdate() {
}

func NewEndOfTurnItem() EndOfTurnItem {
	return EndOfTurnItem{
		Type: endOfTurnType,
	}
}

type EndOfTurnItem struct {
	Type string `json:"type"`
}

func (e EndOfTurnItem) isThreadItem() {}

func NewThreadItemDoneEvent() ThreadItemDoneEvent {
	return ThreadItemDoneEvent{
		Type: threadItemDoneType,
	}
}

type ThreadItemDoneEvent struct {
	Type string     `json:"type"`
	Item ThreadItem `json:"item"`
}

func (e ThreadItemDoneEvent) isThreadStreamEvent() {}

func NewProgressUpdateEvent() ProgressUpdateEvent {
	return ProgressUpdateEvent{
		Type: progressUpdateEventType,
	}
}

type ProgressUpdateEvent struct {
	Type string `json:"type"`
	Icon string `json:"icon,omitempty"`
	Text string `json:"text,omitempty"`
}

func (e ProgressUpdateEvent) isThreadStreamEvent() {}

// AnnotationSource is the union type for the different Annotation sources
// https://openai.github.io/chatkit-python/api/chatkit/types/#chatkit.types.SourceBase
type AnnotationSource interface {
	isAnnotationSource()
}

type SourceBase struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Group       string `json:"group,omitempty"`
}

type UrlSource struct {
	SourceBase
	Url         string `json:"url"`
	Type        string `json:"type"`
	Attribution string `json:"attribution,omitempty"`
}

func (s UrlSource) isAnnotationSource() {}

func NewUrlSource() UrlSource {
	return UrlSource{
		Type: "url",
	}
}
