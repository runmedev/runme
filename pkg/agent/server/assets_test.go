package server

import (
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func TestServerServesStaticAssets(t *testing.T) {
	dir := t.TempDir()
	index := "<html>unchanged app</html>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("create assets directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("export {};"), 0o644); err != nil {
		t.Fatalf("write JavaScript asset: %v", err)
	}

	srv, err := NewServer(Options{
		ConfigDir: t.TempDir(),
		Server: &config.AssistantServerConfig{
			StaticAssets: dir,
		},
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	handler, err := srv.BuildHandler()
	if err != nil {
		t.Fatalf("build handler: %v", err)
	}

	tests := []struct {
		name        string
		method      string
		path        string
		wantBody    string
		contentType string
	}{
		{name: "index", method: http.MethodGet, path: "/", wantBody: index, contentType: "text/html"},
		{name: "explicit index", method: http.MethodGet, path: "/index.html", wantBody: index, contentType: "text/html"},
		{name: "SPA route", method: http.MethodGet, path: "/notebooks/example", wantBody: index, contentType: "text/html"},
		{name: "asset", method: http.MethodGet, path: "/assets/app.js", wantBody: "export {};", contentType: mime.TypeByExtension(".js")},
		{name: "head", method: http.MethodHead, path: "/assets/app.js", wantBody: "", contentType: mime.TypeByExtension(".js")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}
			if rec.Body.String() != tt.wantBody {
				t.Fatalf("expected body %q, got %q", tt.wantBody, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tt.contentType) {
				t.Fatalf("expected Content-Type containing %q, got %q", tt.contentType, got)
			}
		})
	}
}

func TestServerStaticAssetsAreOptional(t *testing.T) {
	tests := []struct {
		name         string
		staticAssets func(t *testing.T) string
	}{
		{name: "empty", staticAssets: func(t *testing.T) string { return "" }},
		{name: "missing directory", staticAssets: func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "missing")
		}},
		{name: "file instead of directory", staticAssets: func(t *testing.T) string {
			file := filepath.Join(t.TempDir(), "app")
			if err := os.WriteFile(file, []byte("not a directory"), 0o644); err != nil {
				t.Fatalf("write file: %v", err)
			}
			return file
		}},
		{name: "missing index", staticAssets: func(t *testing.T) string { return t.TempDir() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, err := NewServer(Options{
				ConfigDir: t.TempDir(),
				Server: &config.AssistantServerConfig{
					RunnerService: true,
					StaticAssets:  tt.staticAssets(t),
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
				t.Fatalf("expected static route to be absent, got status %d", rec.Code)
			}
		})
	}
}

func TestStaticAssetsDoNotShadowAPIRoutes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
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

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics response: %v", err)
	}
	if strings.Contains(string(body), "<html>app</html>") {
		t.Fatal("expected metrics handler to take precedence over static assets")
	}
}

func TestStaticAssetsCORS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	srv, err := NewServer(Options{
		ConfigDir: t.TempDir(),
		Server: &config.AssistantServerConfig{
			CorsOrigins:  []string{"https://example.com", "*"},
			StaticAssets: dir,
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
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected configured CORS origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected static assets not to allow credentials, got %q", got)
	}
}
