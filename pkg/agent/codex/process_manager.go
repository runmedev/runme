package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
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
	defaultThreadStartMethod         = "thread/start"
	defaultThreadReadMethod          = "thread/read"
	defaultTurnStartMethod           = "turn/start"
	defaultTurnInterrupt             = "turn/interrupt"
	defaultInitializeProtocolVersion = "2025-03-26"
	defaultInitializeClientName      = "runme"
	defaultInitializeClientVersion   = "dev"
)

const defaultThreadDeveloperInstructions = "You are working inside a Runme notebook. When asked to inspect or modify the notebook, use the runme-notebooks MCP tools (ListCells, GetCells, UpdateCells) instead of only describing the change. Use ListCells first to inspect the current notebook before editing cells."

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

	client            *Client
	initializeMethod  string
	initializeParams  any
	threadStartMethod string
	threadReadMethod  string
	turnStartMethod   string
	turnInterrupt     string
	sessionConfigs    map[string]SessionConfig
	lastTurnIDs       map[string]string
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

		initializeMethod:  defaultInitializeMethod,
		initializeParams:  defaultInitializeParams(),
		threadStartMethod: defaultThreadStartMethod,
		threadReadMethod:  defaultThreadReadMethod,
		turnStartMethod:   defaultTurnStartMethod,
		turnInterrupt:     defaultTurnInterrupt,
		sessionConfigs:    make(map[string]SessionConfig, 4),
		lastTurnIDs:       make(map[string]string, 16),
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
	if client != nil {
		p.sessionConfigs[cfg.SessionID] = cfg
	}
	p.mu.Unlock()
	if client == nil {
		return errors.New("codex process is not running")
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
	threadStartMethod := p.threadStartMethod
	turnStartMethod := p.turnStartMethod
	sessionCfg, hasSessionCfg := p.sessionConfigs[req.SessionID]
	p.mu.Unlock()
	if client == nil {
		return nil, errors.New("codex process is not running")
	}

	threadID := req.ThreadID
	if threadID == "" {
		startResp := &threadStartResponse{}
		if err := client.Call(ctx, threadStartMethod, buildThreadStartParams(sessionCfg, hasSessionCfg), startResp); err != nil {
			logger.Error(err, "failed to start codex thread")
			return nil, err
		}
		threadID = strings.TrimSpace(startResp.Thread.ID)
		if threadID == "" {
			return nil, errors.New("thread/start response missing thread id")
		}
	}

	notifications := make([]jsonRPCNotification, 0, 16)
	startTurnResp := &turnStartResponse{}
	if err := client.CallUntil(
		ctx,
		turnStartMethod,
		buildTurnParams(req, threadID),
		startTurnResp,
		func(note jsonRPCNotification) error {
			notifications = append(notifications, note)
			return nil
		},
		func() bool {
			return turnNotificationsComplete(notifications, startTurnResp.Turn.ID)
		},
	); err != nil {
		logger.Error(err, "failed to execute codex turn")
		return nil, err
	}

	turnID := strings.TrimSpace(startTurnResp.Turn.ID)
	if turnID != "" {
		p.mu.Lock()
		p.lastTurnIDs[threadID] = turnID
		p.mu.Unlock()
	}

	resp := &TurnResponse{
		ThreadID:           threadID,
		PreviousResponseID: turnID,
	}
	completion := turnCompletionFromNotifications(notifications, turnID)
	if completion.Status == "failed" {
		msg := strings.TrimSpace(completion.Error.Message)
		if msg == "" {
			msg = "codex turn failed"
		}
		return nil, errors.New(msg)
	}
	resp.Events = extractTurnEventsFromNotifications(notifications, turnID)
	logger.Info("codex turn completed", "eventCount", len(resp.Events), "previousResponseID", resp.PreviousResponseID)
	return resp, nil
}

func (p *ProcessManager) Interrupt(ctx context.Context, sessionID, threadID string) error {
	_ = sessionID
	if threadID == "" {
		return errors.New("thread id must not be empty")
	}

	p.mu.Lock()
	client := p.client
	method := p.turnInterrupt
	turnID := p.lastTurnIDs[threadID]
	p.mu.Unlock()
	if client == nil {
		return errors.New("codex process is not running")
	}
	if turnID == "" {
		return errors.New("turn id must not be empty")
	}

	params := map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	}
	return client.Call(ctx, method, params, nil)
}

func buildSessionConfigParams(cfg SessionConfig) map[string]any {
	return map[string]any{
		"approval_policy": "never",
		"approvalPolicy":  "never",
		"mcp_servers":     buildMCPServersConfig(cfg),
		"mcpServers":      buildMCPServersConfig(cfg),
	}
}

func buildMCPServersConfig(cfg SessionConfig) map[string]any {
	mcpServer := map[string]any{
		"url": sessionScopedMCPServerURL(cfg),
	}
	mcpServers := map[string]any{
		"runme-notebooks": mcpServer,
	}
	return mcpServers
}

func sessionScopedMCPServerURL(cfg SessionConfig) string {
	baseURL := strings.TrimSpace(cfg.MCPServerURL)
	if baseURL == "" || strings.TrimSpace(cfg.BearerToken) == "" {
		return baseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	query := parsed.Query()
	query.Set(sessionTokenQueryParam, cfg.BearerToken)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func buildThreadStartParams(cfg SessionConfig, includeConfig bool) map[string]any {
	params := map[string]any{
		"approvalPolicy":        "never",
		"developerInstructions": defaultThreadDeveloperInstructions,
	}
	if includeConfig {
		params["config"] = buildSessionConfigParams(cfg)
	}
	return params
}

func buildTurnParams(req TurnRequest, threadID string) map[string]any {
	input := buildTurnInput(req)
	params := map[string]any{
		"threadId": threadID,
		"input":    input,
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

type threadStartResponse struct {
	Thread struct {
		ID string `json:"id"`
	} `json:"thread"`
}

type turnStartResponse struct {
	Turn struct {
		ID string `json:"id"`
	} `json:"turn"`
}

type threadReadResponse struct {
	Thread struct {
		Turns []codexTurn `json:"turns"`
	} `json:"thread"`
}

type codexTurn struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Items  []codexTurnItem `json:"items"`
	Error  codexTurnError  `json:"error"`
}

type codexTurnError struct {
	Message string `json:"message"`
}

type codexTurnItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

type itemCompletedNotification struct {
	ThreadID string        `json:"threadId"`
	TurnID   string        `json:"turnId"`
	Item     codexTurnItem `json:"item"`
}

type agentMessageDeltaNotification struct {
	Delta    string `json:"delta"`
	ItemID   string `json:"itemId"`
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type turnCompletedNotification struct {
	ThreadID string    `json:"threadId"`
	Turn     codexTurn `json:"turn"`
}

func selectTurn(turns []codexTurn, turnID string) (codexTurn, bool) {
	if len(turns) == 0 {
		return codexTurn{}, false
	}
	if turnID != "" {
		for _, turn := range turns {
			if turn.ID == turnID {
				return turn, true
			}
		}
	}
	return turns[len(turns)-1], true
}

func extractTurnEvents(items []codexTurnItem) []TurnEvent {
	events := make([]TurnEvent, 0, len(items))
	for _, item := range items {
		if item.Type != "agentMessage" {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		events = append(events, TurnEvent{
			Type:   "assistant_message",
			ItemID: item.ID,
			Text:   text,
		})
	}
	return events
}

func turnNotificationsComplete(notifications []jsonRPCNotification, turnID string) bool {
	if strings.TrimSpace(turnID) == "" {
		return false
	}
	completion := turnCompletionFromNotifications(notifications, turnID)
	return completion.ID != ""
}

func turnCompletionFromNotifications(notifications []jsonRPCNotification, turnID string) codexTurn {
	for _, note := range notifications {
		if note.Method != "turn/completed" {
			continue
		}
		payload := &turnCompletedNotification{}
		if err := json.Unmarshal(note.Params, payload); err != nil {
			continue
		}
		if payload.Turn.ID == turnID {
			return payload.Turn
		}
	}
	return codexTurn{}
}

func extractTurnEventsFromNotifications(notifications []jsonRPCNotification, turnID string) []TurnEvent {
	itemEvents := make([]TurnEvent, 0, 4)
	deltaByItem := make(map[string]string, 4)

	for _, note := range notifications {
		switch note.Method {
		case "item/agentMessage/delta":
			payload := &agentMessageDeltaNotification{}
			if err := json.Unmarshal(note.Params, payload); err != nil {
				continue
			}
			if payload.TurnID != turnID || payload.ItemID == "" {
				continue
			}
			deltaByItem[payload.ItemID] += payload.Delta
		case "item/completed":
			payload := &itemCompletedNotification{}
			if err := json.Unmarshal(note.Params, payload); err != nil {
				continue
			}
			if payload.TurnID != turnID {
				continue
			}
			if payload.Item.Type != "agentMessage" {
				continue
			}
			text := strings.TrimSpace(payload.Item.Text)
			if text == "" {
				text = strings.TrimSpace(deltaByItem[payload.Item.ID])
			}
			if text == "" {
				continue
			}
			itemEvents = append(itemEvents, TurnEvent{
				Type:   "assistant_message",
				ItemID: payload.Item.ID,
				Text:   text,
			})
			delete(deltaByItem, payload.Item.ID)
		}
	}

	for itemID, text := range deltaByItem {
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		itemEvents = append(itemEvents, TurnEvent{
			Type:   "assistant_message",
			ItemID: itemID,
			Text:   text,
		})
	}

	return itemEvents
}

func waitForTurn(ctx context.Context, client *Client, threadReadMethod, threadID, turnID string) (codexTurn, error) {
	deadline := time.Now().Add(8 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	var lastErr error
	for {
		readResp := &threadReadResponse{}
		err := client.Call(ctx, threadReadMethod, map[string]any{
			"threadId":     threadID,
			"includeTurns": true,
		}, readResp)
		if err == nil {
			if selectedTurn, ok := selectTurn(readResp.Thread.Turns, turnID); ok {
				return selectedTurn, nil
			}
			lastErr = errors.New("thread/read response missing turn")
		} else {
			lastErr = err
			if !retryableThreadReadErr(err) {
				return codexTurn{}, err
			}
		}

		if time.Now().After(deadline) {
			break
		}

		select {
		case <-ctx.Done():
			return codexTurn{}, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	if lastErr == nil {
		lastErr = errors.New("timed out waiting for turn")
	}
	return codexTurn{}, lastErr
}

func retryableThreadReadErr(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not materialized yet") ||
		strings.Contains(msg, "includeturns is unavailable before first user message")
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
