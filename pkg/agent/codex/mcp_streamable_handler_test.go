package codex

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestStreamableMCPHandler_RequiresBearerToken(t *testing.T) {
	bridge := NewToolBridge()
	tokens := NewSessionTokenManager(0)
	handler, err := NewStreamableMCPHandler(bridge, tokens)
	if err != nil {
		t.Fatalf("NewStreamableMCPHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp/notebooks", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestStreamableMCPHandler_AllowsAuthorizedInitialize(t *testing.T) {
	bridge := NewToolBridge()
	tokens := NewSessionTokenManager(0)
	handler, err := NewStreamableMCPHandler(bridge, tokens)
	if err != nil {
		t.Fatalf("NewStreamableMCPHandler returned error: %v", err)
	}

	token, err := tokens.Issue("session-1")
	if err != nil {
		t.Fatalf("Issue token failed: %v", err)
	}

	initializeReq := `{
	  "jsonrpc":"2.0",
	  "id":1,
	  "method":"initialize",
	  "params":{
	    "protocolVersion":"2025-03-26",
	    "capabilities":{},
	    "clientInfo":{"name":"test-client","version":"1.0.0"}
	  }
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/notebooks", bytes.NewBufferString(initializeReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestParseApprovedRefIDs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "single", in: "cell-1", want: []string{"cell-1"}},
		{name: "csv with spaces", in: "cell-1, cell-2 , ,cell-3", want: []string{"cell-1", "cell-2", "cell-3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseApprovedRefIDs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseApprovedRefIDs(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
