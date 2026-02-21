package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	defaultShutdownTimeout   = 3 * time.Second
	defaultInitializeTimeout = 5 * time.Second
)

const (
	defaultInitializeMethod    = "initialize"
	defaultSessionConfigMethod = "session/configure"
)

type SessionConfig struct {
	SessionID    string
	MCPServerURL string
	BearerToken  string
}

type ProcessManager struct {
	mu sync.Mutex

	command string
	args    []string
	env     []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	client              *Client
	initializeMethod    string
	initializeParams    any
	sessionConfigMethod string
}

func NewProcessManager(command string, args []string, env []string) *ProcessManager {
	if command == "" {
		command = "codex"
	}
	if len(args) == 0 {
		args = []string{"app-server"}
	}
	return &ProcessManager{
		command: command,
		args:    append([]string(nil), args...),
		env:     append([]string(nil), env...),

		initializeMethod:    defaultInitializeMethod,
		initializeParams:    map[string]any{},
		sessionConfigMethod: defaultSessionConfigMethod,
	}
}

func (p *ProcessManager) EnsureStarted(ctx context.Context) error {
	start := time.Now()
	logger := logs.FromContextWithTrace(ctx).WithValues("component", "codex-process-manager")
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}

	p.mu.Lock()
	if p.cmd != nil && p.cmd.Process != nil && p.cmd.ProcessState == nil {
		p.mu.Unlock()
		return nil
	}

	cmd := exec.Command(p.command, p.args...) //nolint:gosec // command is configured by server.
	if len(p.env) > 0 {
		cmd.Env = append(os.Environ(), p.env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		p.mu.Unlock()
		startErr := fmt.Errorf("create stdin pipe: %w", err)
		observeAppServerStartup(time.Since(start), startErr)
		logger.Error(startErr, "failed to create stdin pipe")
		return startErr
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.mu.Unlock()
		startErr := fmt.Errorf("create stdout pipe: %w", err)
		observeAppServerStartup(time.Since(start), startErr)
		logger.Error(startErr, "failed to create stdout pipe")
		return startErr
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.mu.Unlock()
		startErr := fmt.Errorf("create stderr pipe: %w", err)
		observeAppServerStartup(time.Since(start), startErr)
		logger.Error(startErr, "failed to create stderr pipe")
		return startErr
	}
	if err := cmd.Start(); err != nil {
		p.mu.Unlock()
		startErr := fmt.Errorf("start codex app-server: %w", err)
		observeAppServerStartup(time.Since(start), startErr)
		logger.Error(startErr, "failed to start codex app-server process")
		return startErr
	}

	p.cmd = cmd
	p.stdin = stdin
	p.stdout = stdout
	p.stderr = stderr
	p.client = NewClient(stdout, stdin)
	client := p.client
	initializeMethod := p.initializeMethod
	initializeParams := p.initializeParams
	p.mu.Unlock()

	healthCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		healthCtx, cancel = context.WithTimeout(ctx, defaultInitializeTimeout)
		defer cancel()
	}
	if err := client.Call(healthCtx, initializeMethod, initializeParams, nil); err != nil {
		_ = p.Stop(context.Background())
		startErr := fmt.Errorf("initialize codex app-server: %w", err)
		observeAppServerStartup(time.Since(start), startErr)
		logger.Error(startErr, "codex app-server initialize call failed")
		return startErr
	}
	observeAppServerStartup(time.Since(start), nil)
	logger.Info("codex app-server started", "startupLatencyMs", time.Since(start).Milliseconds())
	return nil
}

func (p *ProcessManager) StdIO() (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.stdin == nil || p.stdout == nil || p.stderr == nil {
		return nil, nil, nil, errors.New("codex process is not running")
	}
	return p.stdin, p.stdout, p.stderr, nil
}

func (p *ProcessManager) ConfigureSession(ctx context.Context, cfg SessionConfig) error {
	logger := logs.FromContextWithTrace(ctx).WithValues(
		"component", "codex-process-manager",
		"sessionID", cfg.SessionID,
	)
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}

	if cfg.SessionID == "" {
		return errors.New("session id must not be empty")
	}
	if cfg.MCPServerURL == "" {
		return errors.New("mcp server url must not be empty")
	}
	if cfg.BearerToken == "" {
		return errors.New("bearer token must not be empty")
	}

	p.mu.Lock()
	client := p.client
	method := p.sessionConfigMethod
	p.mu.Unlock()
	if client == nil {
		return errors.New("codex process is not running")
	}

	params := buildSessionConfigParams(cfg)
	if err := client.Notify(ctx, method, params); err != nil {
		logger.Error(err, "failed to send codex session configuration")
		return err
	}
	logger.Info("configured codex session")
	return nil
}

func buildSessionConfigParams(cfg SessionConfig) map[string]any {
	mcpServer := map[string]any{
		"transport": map[string]any{
			"type": "streamable_http",
			"url":  cfg.MCPServerURL,
		},
		"headers": map[string]any{
			"Authorization": "Bearer " + cfg.BearerToken,
		},
	}
	mcpServers := map[string]any{
		"runme-notebooks": mcpServer,
	}
	return map[string]any{
		"session_id":      cfg.SessionID,
		"sessionId":       cfg.SessionID,
		"approval_policy": "never",
		"approvalPolicy":  "never",
		"mcp_servers":     mcpServers,
		"mcpServers":      mcpServers,
	}
}

func (p *ProcessManager) Stop(ctx context.Context) error {
	p.mu.Lock()
	cmd := p.cmd
	p.cmd = nil
	p.stdin = nil
	p.stdout = nil
	p.stderr = nil
	p.client = nil
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	_ = cmd.Process.Signal(syscall.SIGINT)

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	timeout := defaultShutdownTimeout
	if d, ok := ctx.Deadline(); ok {
		remaining := time.Until(d)
		if remaining > 0 {
			timeout = remaining
		}
	}

	select {
	case err := <-waitCh:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return <-waitCh
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-waitCh
		return ctx.Err()
	}
}

func (p *ProcessManager) MarshalSessionConfig(cfg SessionConfig) ([]byte, error) {
	return json.Marshal(buildSessionConfigParams(cfg))
}
