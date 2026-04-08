package codex

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamableMCPHandler_RequiresBearerToken(t *testing.T) {
	bridge := NewToolBridge(nil)
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
	bridge := NewToolBridge(nil)
	tokens := NewSessionTokenManager(0)
	handler, err := NewStreamableMCPHandler(bridge, tokens)
	if err != nil {
		t.Fatalf("NewStreamableMCPHandler returned error: %v", err)
	}

	token, err := tokens.Issue()
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

func TestStreamableMCPHandler_AllowsQueryTokenInitialize(t *testing.T) {
	bridge := NewToolBridge(nil)
	tokens := NewSessionTokenManager(0)
	handler, err := NewStreamableMCPHandler(bridge, tokens)
	if err != nil {
		t.Fatalf("NewStreamableMCPHandler returned error: %v", err)
	}

	token, err := tokens.Issue()
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

	req := httptest.NewRequest(http.MethodPost, "/mcp/notebooks?"+sessionTokenQueryParam+"="+token, bytes.NewBufferString(initializeReq))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestNewStreamableMCPHandler_RequiresTokenManager(t *testing.T) {
	bridge := NewToolBridge(nil)
	if _, err := NewStreamableMCPHandler(bridge, nil); err == nil {
		t.Fatalf("NewStreamableMCPHandler should require a token manager")
	}
}
