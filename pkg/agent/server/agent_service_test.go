package server

import (
	"io/fs"
	"testing"

	"github.com/pkg/errors"

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
