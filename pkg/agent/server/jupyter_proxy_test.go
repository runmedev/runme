package server

import (
	"net/http"
	"net/url"
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
