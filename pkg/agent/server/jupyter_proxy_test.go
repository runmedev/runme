package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCopyProxyRequestHeaders_StripsSensitiveHeaders(t *testing.T) {
	src := http.Header{}
	src.Set("Authorization", "Bearer test")
	src.Set("Origin", "http://localhost:5173")
	src.Set("Referer", "http://localhost:5173/")
	src.Set("Cookie", "session=abc123")
	src.Set("X-XSRFToken", "token")
	src.Set("Content-Type", "application/json")
	src.Set("X-Test", "ok")

	dst := http.Header{}
	copyProxyRequestHeaders(dst, src)

	for _, key := range []string{"Authorization", "Origin", "Referer", "Cookie", "X-XSRFToken"} {
		if got := dst.Get(key); got != "" {
			t.Fatalf("expected %s to be filtered, got %q", key, got)
		}
	}

	if got := dst.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type to be copied, got %q", got)
	}
	if got := dst.Get("X-Test"); got != "ok" {
		t.Fatalf("expected X-Test to be copied, got %q", got)
	}
}

func TestCopyProxyResponseHeaders_StripsCORSHeaders(t *testing.T) {
	src := http.Header{}
	src.Set("Access-Control-Allow-Origin", "")
	src.Set("Access-Control-Allow-Credentials", "true")
	src.Set("Access-Control-Allow-Headers", "authorization")
	src.Set("Access-Control-Allow-Methods", "GET")
	src.Set("Access-Control-Expose-Headers", "x-custom")
	src.Set("Access-Control-Max-Age", "7200")
	src.Set("Content-Type", "application/json")
	src.Set("X-Test", "ok")

	dst := http.Header{}
	copyProxyResponseHeaders(dst, src)

	if got := dst.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected Access-Control-Allow-Origin to be filtered, got %q", got)
	}
	if got := dst.Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected Access-Control-Allow-Credentials to be filtered, got %q", got)
	}
	if got := dst.Get("Access-Control-Allow-Headers"); got != "" {
		t.Fatalf("expected Access-Control-Allow-Headers to be filtered, got %q", got)
	}
	if got := dst.Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("expected Access-Control-Allow-Methods to be filtered, got %q", got)
	}
	if got := dst.Get("Access-Control-Expose-Headers"); got != "" {
		t.Fatalf("expected Access-Control-Expose-Headers to be filtered, got %q", got)
	}
	if got := dst.Get("Access-Control-Max-Age"); got != "" {
		t.Fatalf("expected Access-Control-Max-Age to be filtered, got %q", got)
	}

	if got := dst.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type to be copied, got %q", got)
	}
	if got := dst.Get("X-Test"); got != "ok" {
		t.Fatalf("expected X-Test to be copied, got %q", got)
	}
}

func TestSetUpstreamAuthToken(t *testing.T) {
	upstreamURL, err := url.Parse("http://localhost:8888/api/kernels?foo=bar")
	if err != nil {
		t.Fatalf("failed to parse url: %v", err)
	}

	setUpstreamAuthToken(upstreamURL, "abc123")
	got := upstreamURL.Query().Get("token")
	if got != "abc123" {
		t.Fatalf("expected token query param to be set, got %q", got)
	}
	if got := upstreamURL.Query().Get("foo"); got != "bar" {
		t.Fatalf("expected existing query params to be preserved, got %q", got)
	}
}

func TestSanitizeClientQueryForUpstream_RemovesAuthorization(t *testing.T) {
	query := "authorization=Bearer+abc123&session_id=s1&Authorization=Bearer+ignored"
	got := sanitizeClientQueryForUpstream(query)
	parsed, err := url.ParseQuery(got)
	if err != nil {
		t.Fatalf("failed to parse sanitized query: %v", err)
	}

	if auth := parsed.Get("authorization"); auth != "" {
		t.Fatalf("expected authorization query to be removed, got %q", auth)
	}
	if auth := parsed.Get("Authorization"); auth != "" {
		t.Fatalf("expected Authorization query to be removed, got %q", auth)
	}
	if sessionID := parsed.Get("session_id"); sessionID != "s1" {
		t.Fatalf("expected session_id to be preserved, got %q", sessionID)
	}
}

func TestSanitizeClientQueryForUpstream_InvalidQueryPassthrough(t *testing.T) {
	const raw = "%zz=1"
	if got := sanitizeClientQueryForUpstream(raw); got != raw {
		t.Fatalf("expected invalid query to pass through, got %q", got)
	}
}

func TestForwardHTTP_ReloadsTokenAfterForbidden(t *testing.T) {
	var tokens []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		tokens = append(tokens, token)

		if got := r.Header.Get("Authorization"); got != "token "+token {
			t.Errorf("Authorization header = %q, want %q", got, "token "+token)
		}
		if token == "old-token" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if token != "new-token" {
			t.Errorf("unexpected token %q", token)
			http.Error(w, "unexpected token", http.StatusBadRequest)
			return
		}
		if got := r.URL.Path; got != "/api/kernels" {
			t.Errorf("upstream path = %q, want /api/kernels", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read upstream request body: %v", err)
		}
		if got := strings.TrimSpace(string(body)); got != `{"name":"python3"}` {
			t.Errorf("upstream request body = %q, want %q", got, `{"name":"python3"}`)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"kernel-1","name":"python3"}`))
	}))
	defer upstream.Close()

	configDir := t.TempDir()
	writeJupyterServerConfig(t, configDir, "port-8890", upstream.URL, "old-token")
	handler, err := newJupyterProxyHandler(configDir)
	if err != nil {
		t.Fatalf("newJupyterProxyHandler failed: %v", err)
	}
	writeJupyterServerConfig(t, configDir, "port-8890", upstream.URL, "new-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/jupyter/servers/port-8890/kernels",
		strings.NewReader(`{"name":"python3"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"id":"kernel-1","name":"python3"}` {
		t.Fatalf("body = %q, want created kernel JSON", got)
	}
	wantTokens := []string{"old-token", "new-token"}
	if !reflect.DeepEqual(tokens, wantTokens) {
		t.Fatalf("upstream tokens = %v, want %v", tokens, wantTokens)
	}
}

func writeJupyterServerConfig(t *testing.T, configDir, name, baseURL, token string) {
	t.Helper()
	jupyterDir := filepath.Join(configDir, jupyterConfigDir)
	if err := os.MkdirAll(jupyterDir, 0o755); err != nil {
		t.Fatalf("failed to create jupyter config dir: %v", err)
	}
	payload := map[string]string{
		"base_url": baseURL,
		"token":    token,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal jupyter config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jupyterDir, name+".json"), raw, 0o600); err != nil {
		t.Fatalf("failed to write jupyter config: %v", err)
	}
}
