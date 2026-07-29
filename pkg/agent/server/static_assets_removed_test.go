package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func TestServerDoesNotServeDeprecatedStaticAssets(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "index.html"),
		[]byte("<html>legacy app</html>"),
		0o644,
	); err != nil {
		t.Fatalf("write legacy index: %v", err)
	}

	srv, err := NewServer(Options{
		ConfigDir: t.TempDir(),
		Server: &config.AssistantServerConfig{
			RunnerService: true,
			StaticAssets:  dir,
		},
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	handler, err := srv.BuildHandler()
	if err != nil {
		t.Fatalf("build handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected static asset route to be absent, got status %d", rec.Code)
	}
}
