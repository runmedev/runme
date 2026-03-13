package codex

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/zap"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

var sessionTokenPattern = regexp.MustCompile(`session_token=([^&\s]+)`)

func logProxyRequest(logger logr.Logger, req *proxyJSONRPCRequest, payload any) {
	if req == nil {
		return
	}
	attrs := buildProxyLogAttrs(req.Method, req.ID, payload)
	logger.Info("forwarding codex proxy request", attrs...)
}

func logProxyResponse(logger logr.Logger, method string, id json.RawMessage, payload json.RawMessage) {
	attrs := buildProxyLogAttrs(method, id, sanitizeRawJSON(payload))
	logger.Info("received codex proxy response", attrs...)
}

func logProxyNotification(logger logr.Logger, note jsonRPCNotification) {
	attrs := buildProxyLogAttrs(note.Method, nil, sanitizeRawJSON(note.Params))
	logger.Info("received codex proxy notification", attrs...)
}

func buildProxyLogAttrs(method string, id json.RawMessage, payload any) []any {
	sanitized := sanitizeJSONValue(payload)
	threadID, turnID, itemID := extractProxyIdentifiers(sanitized)
	attrs := []any{
		"method", method,
		zap.Any("payload", sanitized),
	}
	if requestID := rawMessageString(id); requestID != "" {
		attrs = append(attrs, "requestID", requestID)
	}
	if threadID != "" {
		attrs = append(attrs, "threadId", threadID)
	}
	if turnID != "" {
		attrs = append(attrs, "turnId", turnID)
	}
	if itemID != "" {
		attrs = append(attrs, "itemId", itemID)
	}
	return attrs
}

func sanitizeRawJSON(payload json.RawMessage) any {
	if len(payload) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return map[string]any{
			"unmarshalError": err.Error(),
			"raw":            string(payload),
		}
	}
	return sanitizeJSONValue(decoded)
}

func sanitizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, nested := range typed {
			normalized := strings.ToLower(key)
			if normalized == "authorization" ||
				normalized == "session_token" ||
				normalized == "sessiontoken" ||
				normalized == "bearertoken" ||
				normalized == "cookie" ||
				normalized == "cookies" ||
				normalized == "set-cookie" {
				sanitized[key] = "[REDACTED]"
				continue
			}
			sanitized[key] = sanitizeJSONValue(nested)
		}
		return sanitized
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeJSONValue(item))
		}
		return items
	case string:
		return redactSensitiveString(typed)
	default:
		return value
	}
}

func redactSensitiveString(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return "Bearer [REDACTED]"
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" {
		query := parsed.Query()
		if query.Has("session_token") {
			query.Set("session_token", "[REDACTED]")
			parsed.RawQuery = query.Encode()
			return parsed.String()
		}
	}
	return sessionTokenPattern.ReplaceAllString(value, "session_token=[REDACTED]")
}

func rawMessageString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return string(raw)
	}
	return fmt.Sprint(decoded)
}

func extractProxyIdentifiers(value any) (threadID, turnID, itemID string) {
	record, _ := value.(map[string]any)
	if record == nil {
		return "", "", ""
	}
	thread := mapValue(record["thread"])
	turn := mapValue(record["turn"])
	item := mapValue(record["item"])
	msg := mapValue(record["msg"])

	threadID = stringValue(record, "threadId", "thread_id")
	if threadID == "" {
		threadID = stringValue(thread, "id", "threadId", "thread_id")
	}
	if threadID == "" {
		threadID = stringValue(msg, "threadId", "thread_id")
	}

	turnID = stringValue(record, "turnId", "turn_id", "responseId", "response_id")
	if turnID == "" {
		turnID = stringValue(turn, "id", "turnId", "turn_id")
	}
	if turnID == "" {
		turnID = stringValue(msg, "turnId", "turn_id", "responseId", "response_id")
	}

	itemID = stringValue(record, "itemId", "item_id")
	if itemID == "" {
		itemID = stringValue(item, "id", "itemId", "item_id")
	}
	if itemID == "" {
		itemID = stringValue(msg, "itemId", "item_id")
	}
	return threadID, turnID, itemID
}

func mapValue(value any) map[string]any {
	record, _ := value.(map[string]any)
	if record == nil {
		return map[string]any{}
	}
	return record
}

func stringValue(record map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := record[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func logBridgeToolInput(logger logr.Logger, input *toolsv1.ToolCallInput) {
	if input == nil {
		return
	}
	logger.Info("codex bridge tool request payload", logs.ZapProto("input", input))
}

func logBridgeToolOutput(logger logr.Logger, output *toolsv1.ToolCallOutput) {
	if output == nil {
		return
	}
	logger.Info("codex bridge tool response payload", logs.ZapProto("output", output))
}
