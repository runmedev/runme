package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	defaultShutdownTimeout   = 3 * time.Second
	defaultInitializeTimeout = 5 * time.Second
)

const (
	defaultInitializeMethod          = "initialize"
	defaultSessionConfigMethod       = "session/configure"
	defaultTurnStartMethod           = "turn/start"
	defaultThreadInterrupt           = "thread/interrupt"
	defaultInitializeProtocolVersion = "2025-03-26"
	defaultInitializeClientName      = "runme"
	defaultInitializeClientVersion   = "dev"
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
	turnStartMethod     string
	threadInterrupt     string
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
		initializeParams:    defaultInitializeParams(),
		sessionConfigMethod: defaultSessionConfigMethod,
		turnStartMethod:     defaultTurnStartMethod,
		threadInterrupt:     defaultThreadInterrupt,
	}
}

func defaultInitializeParams() map[string]any {
	return map[string]any{
		"protocolVersion": defaultInitializeProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    defaultInitializeClientName,
			"version": defaultInitializeClientVersion,
		},
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

func (p *ProcessManager) RunTurn(ctx context.Context, req TurnRequest) (*TurnResponse, error) {
	logger := logs.FromContextWithTrace(ctx).WithValues(
		"component", "codex-process-manager",
		"sessionID", req.SessionID,
		"threadID", req.ThreadID,
	)
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}

	p.mu.Lock()
	client := p.client
	method := p.turnStartMethod
	p.mu.Unlock()
	if client == nil {
		return nil, errors.New("codex process is not running")
	}

	resp := &TurnResponse{}
	if err := client.Call(ctx, method, buildTurnParams(req), resp); err != nil {
		logger.Error(err, "failed to execute codex turn")
		return nil, err
	}
	logger.Info("codex turn completed", "eventCount", len(resp.Events), "previousResponseID", resp.PreviousResponseID)
	return resp, nil
}

func (p *ProcessManager) Interrupt(ctx context.Context, sessionID, threadID string) error {
	if threadID == "" {
		return errors.New("thread id must not be empty")
	}

	p.mu.Lock()
	client := p.client
	method := p.threadInterrupt
	p.mu.Unlock()
	if client == nil {
		return errors.New("codex process is not running")
	}

	params := map[string]any{
		"session_id": sessionID,
		"sessionId":  sessionID,
		"thread_id":  threadID,
		"threadId":   threadID,
	}
	return client.Notify(ctx, method, params)
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

func buildTurnParams(req TurnRequest) map[string]any {
	params := map[string]any{
		"session_id": req.SessionID,
		"sessionId":  req.SessionID,
		"thread_id":  req.ThreadID,
		"threadId":   req.ThreadID,
	}
	if req.PreviousResponseID != "" {
		params["previous_response_id"] = req.PreviousResponseID
		params["previousResponseId"] = req.PreviousResponseID
	}

	input := buildTurnInput(req)
	if len(input) > 0 {
		params["input"] = input
		params["message"] = flattenTurnInput(input)
	}
	if req.ToolOutput != nil {
		toolOutput := protoJSONValue(req.ToolOutput)
		params["tool_output"] = toolOutput
		params["toolOutput"] = toolOutput
	}
	return params
}

func buildTurnInput(req TurnRequest) []map[string]any {
	out := make([]map[string]any, 0, 2)

	if req.Input != nil {
		for _, content := range req.Input.Content {
			if strings.TrimSpace(content.Text) == "" {
				continue
			}
			out = append(out, map[string]any{
				"type": "text",
				"text": content.Text,
			})
		}
	}

	if req.ToolOutput != nil {
		value := protoJSONValue(req.ToolOutput)
		payload, err := json.Marshal(value)
		if err != nil {
			payload = []byte(`{"error":"failed to marshal tool output"}`)
		}
		out = append(out, map[string]any{
			"type": "text",
			"text": string(payload),
		})
	}

	return out
}

func protoJSONValue(message interface{ ProtoReflect() protoreflect.Message }) any {
	data, err := protojson.Marshal(message)
	if err != nil {
		return map[string]any{"marshal_error": err.Error()}
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{"unmarshal_error": err.Error()}
	}
	return out
}

func flattenTurnInput(input []map[string]any) string {
	parts := make([]string, 0, len(input))
	for _, item := range input {
		text, _ := item["text"].(string)
		if strings.TrimSpace(text) == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
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
