package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/api"
	"github.com/runmedev/runme/v3/pkg/agent/config"
)

// Test_Manual_CodexChatKitSmoke exists to speed up manual backend iteration for codex integration.
// It starts a fake OIDC issuer and a real Runme server in-process, then sends a /chatkit-codex
// request ("Write hello world in python") and fails with full response diagnostics if the turn path breaks.
func Test_Manual_CodexChatKitSmoke(t *testing.T) {
	SkipIfMissing(t, "RUN_MANUAL_TESTS")
	runmeLogs := installManualTestLogger(t)

	if _, err := exec.LookPath("codex"); err != nil {
		t.Skipf("missing codex binary in PATH: %v", err)
	}

	const (
		clientID = "manual-codex-client"
		email    = "codex-manual@example.com"
		prompt   = "Write hello world in python"
	)

	fakeOIDC := newManualFakeOIDC(t, clientID, email)
	idToken := fakeOIDC.mustGenerateToken(t)

	port := mustGetFreePort(t)
	agentEnabled := true
	serverCfg := &config.AssistantServerConfig{
		BindAddress:     "127.0.0.1",
		Port:            port,
		AgentService:    &agentEnabled,
		RunnerService:   true,
		ParserService:   true,
		RunnerReconnect: true,
		CorsOrigins:     []string{"http://localhost:3000"},
		OIDC: &config.OIDCConfig{
			Generic: &config.GenericOIDCConfig{
				ClientID:     clientID,
				ClientSecret: "unused",
				RedirectURL:  "http://localhost:3000/oidc/callback",
				DiscoveryURL: fakeOIDC.discoveryURL(),
				Issuer:       fakeOIDC.issuerURL,
			},
		},
	}

	iamPolicy := &api.IAMPolicy{
		Bindings: []api.IAMBinding{
			{
				Role: api.AgentUserRole,
				Members: []api.Member{
					{Name: email, Kind: api.UserKind},
				},
			},
			{
				Role: api.RunnerUserRole,
				Members: []api.Member{
					{Name: email, Kind: api.UserKind},
				},
			},
			{
				Role: api.ParserUserRole,
				Members: []api.Member{
					{Name: email, Kind: api.UserKind},
				},
			},
		},
	}

	agent, err := ai.NewAgent(ai.AgentOptions{Client: ai.NewClientWithoutKey()})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	srv, err := NewServer(Options{
		Server:                   serverCfg,
		IAMPolicy:                iamPolicy,
		AssetsFileSystemProvider: manualAssetsFileSystemProvider{},
	}, agent)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- srv.Run()
	}()

	t.Cleanup(func() {
		select {
		case err := <-runErrCh:
			if err != nil {
				t.Logf("server exited with error: %v", err)
			}
			return
		default:
		}

		if srv.shutdownComplete != nil {
			srv.shutdown()
		}

		select {
		case err := <-runErrCh:
			if err != nil {
				t.Logf("server exited with error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Logf("timed out waiting for server shutdown")
		}
	})

	baseURL := fmt.Sprintf("http://%s:%d", serverCfg.BindAddress, serverCfg.Port)
	if err := waitForHTTP200(baseURL+"/metrics", 40*time.Second); err != nil {
		t.Fatalf("server did not become ready: %v", err)
	}

	body, err := json.Marshal(map[string]any{
		"type": "threads.create",
		"chatkit_state": map[string]any{
			"thread_id":            "",
			"previous_response_id": "",
		},
		"params": map[string]any{
			"input": map[string]any{
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": prompt,
					},
				},
				"attachments":       []any{},
				"inference_options": map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/chatkit-codex", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+idToken)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 120 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request to /chatkit-codex failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read /chatkit-codex response: %v", err)
	}
	respText := string(respBody)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chatkit-codex status %d; body: %s\nrunme logs:\n%s", resp.StatusCode, respText, runmeLogs.String())
	}

	if strings.Contains(respText, "codex_turn_failed") {
		t.Fatalf("chatkit-codex returned codex_turn_failed; body: %s\nrunme logs:\n%s", respText, runmeLogs.String())
	}

	if !strings.Contains(respText, `"type":"thread.created"`) {
		t.Fatalf("chatkit-codex response missing thread.created event; body: %s\nrunme logs:\n%s", respText, runmeLogs.String())
	}
	if !strings.Contains(respText, `"type":"thread.item.added"`) {
		t.Fatalf("chatkit-codex response missing assistant item event; body: %s\nrunme logs:\n%s", respText, runmeLogs.String())
	}
}

type manualAssetsFileSystemProvider struct{}

func (manualAssetsFileSystemProvider) GetAssetsFileSystem() (fs.FS, error) {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><html><body>manual</body></html>"),
			Mode: 0o644,
		},
	}, nil
}

type manualFakeOIDC struct {
	privateKey *rsa.PrivateKey
	issuerURL  string
	clientID   string
	email      string
	kid        string
	server     *httptest.Server
}

func newManualFakeOIDC(t *testing.T, clientID, email string) *manualFakeOIDC {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	fake := &manualFakeOIDC{
		privateKey: privateKey,
		clientID:   clientID,
		email:      email,
		kid:        "manual-test-key",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeManualJSON(w, map[string]any{
			"issuer":                 fake.issuerURL,
			"jwks_uri":               fake.issuerURL + "/jwks",
			"authorization_endpoint": fake.issuerURL + "/auth",
			"token_endpoint":         fake.issuerURL + "/token",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeManualJSON(w, map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"alg": "RS256",
					"use": "sig",
					"kid": fake.kid,
					"n":   base64.RawURLEncoding.EncodeToString(fake.privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(fake.privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	fake.server = httptest.NewServer(mux)
	fake.issuerURL = fake.server.URL
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *manualFakeOIDC) discoveryURL() string {
	return f.issuerURL + "/.well-known/openid-configuration"
}

func (f *manualFakeOIDC) mustGenerateToken(t *testing.T) string {
	t.Helper()
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   f.issuerURL,
		"aud":   f.clientID,
		"sub":   f.email,
		"email": f.email,
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
	})
	token.Header["kid"] = f.kid
	signed, err := token.SignedString(f.privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func writeManualJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func mustGetFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to acquire free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForHTTP200(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %s", url)
}

func installManualTestLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	logBuf := &bytes.Buffer{}
	encCfg := zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), zapcore.AddSync(logBuf), zap.DebugLevel)
	logger := zap.New(core)

	restore := zap.ReplaceGlobals(logger)
	t.Cleanup(func() {
		restore()
		_ = logger.Sync()
	})
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("runme server logs:\n%s", logBuf.String())
		}
	})
	return logBuf
}
