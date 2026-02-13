package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func TestEnsureTLSCertificate_GenerateFalse_NoFilesCreated(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	app := &application.App{
		AppConfig: &config.AppConfig{
			Config: &config.Config{
				AssistantServer: &config.AssistantServerConfig{
					TLSConfig: &config.TLSConfig{
						Generate: false,
						CertFile: certFile,
						KeyFile:  keyFile,
					},
				},
			},
		},
	}

	if err := ensureTLSCertificate(app); err != nil {
		t.Fatalf("ensureTLSCertificate returned error: %v", err)
	}

	if _, err := os.Stat(certFile); !os.IsNotExist(err) {
		t.Fatalf("expected cert file not to exist; err=%v", err)
	}
	if _, err := os.Stat(keyFile); !os.IsNotExist(err) {
		t.Fatalf("expected key file not to exist; err=%v", err)
	}
}

func TestEnsureTLSCertificate_GenerateTrue_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	app := &application.App{
		AppConfig: &config.AppConfig{
			Config: &config.Config{
				AssistantServer: &config.AssistantServerConfig{
					TLSConfig: &config.TLSConfig{
						Generate: true,
						CertFile: certFile,
						KeyFile:  keyFile,
					},
				},
			},
		},
	}

	if err := ensureTLSCertificate(app); err != nil {
		t.Fatalf("ensureTLSCertificate returned error: %v", err)
	}

	if _, err := os.Stat(certFile); err != nil {
		t.Fatalf("expected cert file to exist: %v", err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		t.Fatalf("expected key file to exist: %v", err)
	}
}

func TestEnsureTLSCertificate_NilConfig_NoError(t *testing.T) {
	app := &application.App{}
	if err := ensureTLSCertificate(app); err != nil {
		t.Fatalf("expected no error for nil config: %v", err)
	}
}
