package server

import (
	"net/http"
	"testing"
)

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

