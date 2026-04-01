package codex

import (
	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/pkg/agent/ai/chatkit"
)

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
