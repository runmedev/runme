package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1/agentv1connect"
	"github.com/runmedev/runme/v3/pkg/agent/config"
)

type panicAssetsProvider struct{}

func (p *panicAssetsProvider) GetAssetsFileSystem() (fs.FS, error) {
	return nil, errors.New("assets provider should not be called")
}

func Test_NewServer_AgentNil_DoesNotInitializeAssets(t *testing.T) {
	opts := Options{
		Server: &config.AssistantServerConfig{
			RunnerService: true,
		},
		AssetsFileSystemProvider: &panicAssetsProvider{},
	}

	s, err := NewServer(opts, nil)
	if err != nil {
		t.Fatalf("expected NewServer to succeed with nil agent when runner service is enabled: %v", err)
	}
	if s.assetsFS != nil {
		t.Fatalf("expected assets filesystem to remain nil when agent is nil")
	}
}

func Test_NewServer_AgentEnabled_WithoutStaticAssets_DoesNotInitializeAssets(t *testing.T) {
	opts := Options{
		Server: &config.AssistantServerConfig{},
	}

	s, err := NewServer(opts, &agentv1connect.UnimplementedMessagesServiceHandler{})
	if err != nil {
		t.Fatalf("expected NewServer to succeed without static assets when agent service is enabled: %v", err)
	}
	if s.assetsFS != nil {
		t.Fatalf("expected assets filesystem to remain nil when static assets are not configured")
	}
}

func Test_NewServer_AgentEnabled_WithStaticAssets_InitializesAssets(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>test</html>"), 0o644); err != nil {
		t.Fatalf("failed to create test index.html: %v", err)
	}

	opts := Options{
		Server: &config.AssistantServerConfig{
			StaticAssets: dir,
		},
	}

	s, err := NewServer(opts, &agentv1connect.UnimplementedMessagesServiceHandler{})
	if err != nil {
		t.Fatalf("expected NewServer to succeed with static assets configured: %v", err)
	}
	if s.assetsFS == nil {
		t.Fatalf("expected assets filesystem to be initialized when static assets are configured")
	}
}
