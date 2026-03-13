package codex

import (
	"encoding/json"
	"testing"
)

func TestSanitizeRawJSON(t *testing.T) {
	payload := json.RawMessage(`{
		"authorization":"Bearer test-token",
		"config":{
			"mcpServers":[
				{
					"url":"http://localhost:5191/mcp/notebooks?session_token=abc123"
				}
			]
		}
	}`)

	sanitized := sanitizeRawJSON(payload)
	record, ok := sanitized.(map[string]any)
	if !ok {
		t.Fatalf("sanitizeRawJSON() type = %T, want map[string]any", sanitized)
	}
	if got := record["authorization"]; got != "[REDACTED]" {
		t.Fatalf("authorization = %#v, want redacted", got)
	}
	config, _ := record["config"].(map[string]any)
	servers, _ := config["mcpServers"].([]any)
	server, _ := servers[0].(map[string]any)
	urlValue, _ := server["url"].(string)
	if urlValue != "http://localhost:5191/mcp/notebooks?session_token=%5BREDACTED%5D" {
		t.Fatalf("url = %q, want redacted session token", urlValue)
	}
}

func TestExtractProxyIdentifiers(t *testing.T) {
	threadID, turnID, itemID := extractProxyIdentifiers(map[string]any{
		"thread": map[string]any{"id": "thread-1"},
		"turn":   map[string]any{"id": "turn-1"},
		"item":   map[string]any{"id": "item-1"},
	})
	if threadID != "thread-1" || turnID != "turn-1" || itemID != "item-1" {
		t.Fatalf("extractProxyIdentifiers() = (%q, %q, %q)", threadID, turnID, itemID)
	}
}
