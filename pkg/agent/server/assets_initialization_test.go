package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func Test_NewServer_WithoutStaticAssets_DoesNotInitializeAssets(t *testing.T) {
	opts := Options{
		Server: &config.AssistantServerConfig{
			RunnerService: true,
		},
	}

	s, err := NewServer(opts)
	if err != nil {
		t.Fatalf("expected NewServer to succeed without static assets: %v", err)
	}
	if s.assetsFS != nil {
		t.Fatalf("expected assets filesystem to remain nil when static assets are not configured")
	}
}

func Test_NewServer_WithStaticAssets_InitializesAssets(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>test</html>"), 0o644); err != nil {
		t.Fatalf("failed to create test index.html: %v", err)
	}

	opts := Options{
		Server: &config.AssistantServerConfig{
			StaticAssets:  dir,
			RunnerService: true,
		},
	}

	s, err := NewServer(opts)
	if err != nil {
		t.Fatalf("expected NewServer to succeed with static assets configured: %v", err)
	}
	if s.assetsFS == nil {
		t.Fatalf("expected assets filesystem to be initialized when static assets are configured")
	}
}
