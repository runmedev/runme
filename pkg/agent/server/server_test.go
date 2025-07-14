package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-logr/zapr"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jlewi/monogo/networking"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1/agentv1connect"

	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/logs"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	v2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	streamv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/stream/v1"
)

const testAppName = "runme-agent"

func Test_HealthCheck(t *testing.T) {
	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))
	// Try sending a healthcheck to the given server.
	// This is solely for the purpose of trying to reproduce the grpc-trailer issue
	SkipIfMissing(t, "RUN_MANUAL_TESTS")
	app := application.NewApp(testAppName)
	err := app.LoadConfig(nil)
	if err != nil {
		t.Fatalf("Error loading config; %v", err)
	}

	if err := app.SetupLogging(); err != nil {
		t.Fatalf("Error setting up logging; %v", err)
	}
	cfg := app.GetConfig()
	c := *cfg
	c.AssistantServer = &config.AssistantServerConfig{
		Port:         9080,
		StaticAssets: cfg.AssistantServer.StaticAssets,
	}

	go func() {
		if err := setupAndRunServer(c); err != nil {
			log.Error(err, "Error running server")
		}
	}()
	if _, err := waitForServer(c); err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}
	t.Logf("Server started")
}

func Test_GenerateCells(t *testing.T) {
	SkipIfMissing(t, "RUN_MANUAL_TESTS")

	app := application.NewApp(testAppName)
	err := app.LoadConfig(nil)
	if err != nil {
		t.Fatalf("Error loading config; %v", err)
	}
	cfg := app.GetConfig()

	if err := app.SetupLogging(); err != nil {
		t.Fatalf("Error setting up logging; %v", err)
	}

	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))

	port, err := networking.GetFreePort()
	if err != nil {
		t.Fatalf("Error getting free port; %v", err)
	}

	if cfg.AssistantServer == nil {
		cfg.AssistantServer = &config.AssistantServerConfig{}
	}
	cfg.AssistantServer.Port = port

	go func() {
		if err := setupAndRunServer(*cfg); err != nil {
			log.Error(err, "Error running server")
		}
	}()

	// N.B. There's probably a race condition here because the client might start before the server is fully up.
	// Or maybe that's implicitly handled because the connection won't succeed until the server is up?
	addr, err := waitForServer(*cfg)
	if err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}

	log.Info("Server started")
	cells, err := runAIClient(addr)
	if err != nil {
		t.Fatalf("Error running client for addres %v; %v", addr, err)
	}

	if len(cells) < 2 {
		t.Errorf("Expected at least 2 cells; got %d cells", len(cells))
	}

	// Ensure there is a filesearch results cell and that the filenames are set. This is intended
	// to catch various bugs with the SDK;
	hasFSCell := false
	for _, b := range cells {
		if b.Kind == parserv1.CellKind_CELL_KIND_DOC_RESULTS {
			hasFSCell = true

			if len(b.DocResults) <= 0 {
				t.Errorf("FileSearchResults cell has no results")
			}

			for _, r := range b.DocResults {
				if r.FileName == "" {
					t.Errorf("DocResults cell has empty filename")
				}
			}
		}
	}

	if !hasFSCell {
		t.Errorf("There was no DocResults cell in the results.")
	}
}

func runAIClient(baseURL string) (map[string]*parserv1.Cell, error) {
	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))

	cells := make(map[string]*parserv1.Cell)

	Cell := parserv1.Cell{
		Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
		Value: "This is a cell",
	}

	log.Info("Cell", logs.ZapProto("cell", &Cell))

	u, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return cells, errors.Wrapf(err, "Failed to parse URL")
	}

	var client agentv1connect.MessagesServiceClient

	// Mimic what the frontend does
	options := []connect.ClientOption{connect.WithGRPCWeb()}
	if u.Scheme == "https" {
		// Configure the TLS settings
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Set to true only for testing; otherwise validate the server's certificate
		}

		client = agentv1connect.NewMessagesServiceClient(
			&http.Client{
				Transport: &http2.Transport{
					TLSClientConfig: tlsConfig,
					DialTLSContext: func(ctx context.Context, network, addr string, config *tls.Config) (net.Conn, error) {
						// Create a secure connection with TLS
						return tls.Dial(network, addr, config)
					},
				},
			},
			baseURL,
			options...,
		)
	} else {
		client = agentv1connect.NewMessagesServiceClient(
			&http.Client{
				Transport: &http2.Transport{
					AllowHTTP: true,
					DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
						// Use the standard Dial function to create a plain TCP connection
						return net.Dial(network, u.Host)
					},
				},
			},
			baseURL,
			options...,
		)
	}

	contents := "Show me all the AKS clusters at OpenAI"
	if os.Getenv("GITHUB_REPOSITORY_OWNER") == "runmedev" {
		contents = "launch a psql session against the staging database as documented"
	}

	ctx := context.Background()
	genReq := &agentv1.GenerateRequest{
		Cells: []*parserv1.Cell{
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Role:  parserv1.CellRole_CELL_ROLE_USER,
				Value: contents,
			},
		},
	}
	req := connect.NewRequest(genReq)

	stream, err := client.Generate(ctx, req)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to create generate stream")
	}

	// Receive responses
	for stream.Receive() {
		response := stream.Msg()

		for _, cell := range response.Cells {
			cells[cell.RefId] = cell

			options := protojson.MarshalOptions{
				Multiline: true,
				Indent:    "  ", // Two spaces for indentation
			}

			// Marshal the protobuf message to JSON
			jsonData, err := options.Marshal(cell)
			if err != nil {
				log.Error(err, "Failed to marshal cell to JSON")
			} else {
				log.Info("Cell", "cell", string(jsonData))
			}
		}

	}

	if stream.Err() != nil {
		return cells, errors.Wrapf(stream.Err(), "Error receiving response")
	}
	return cells, nil
}

func Test_ExecuteWithRunmeStream(t *testing.T) {
	SkipIfMissing(t, "RUN_MANUAL_TESTS")

	app := application.NewApp(testAppName)
	err := app.LoadConfig(nil)
	if err != nil {
		t.Fatalf("Error loading config; %v", err)
	}

	if err := app.SetupLogging(); err != nil {
		t.Fatalf("Error setting up logging; %v", err)
	}

	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))

	// N.B. Server currently needs to be started manually. Should we start it autommatically?
	cfg := app.GetConfig()

	// N.B. There's probably a race condition here because the client might start before the server is fully up.
	// Or maybe that's implicitly handled because the connection won't succeed until the server is up?
	addr, err := waitForServer(*cfg)
	if err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}

	log.Info("Server started")
	_, err = runWebsocketClient(addr)
	if err != nil {
		t.Fatalf("Error running client for addres %v; %v", addr, err)
	}
}

func Test_ExecuteWithRunmeConcurrent(t *testing.T) {
	// The purpose of this test test is to verify that if we use to ExecuteRequests without waiting for the first to
	// finish that the server can handle this. Specifically, we want to make sure we don't have concurrent writes
	// to the websocket on the backend because that causes a panic
	SkipIfMissing(t, "RUN_MANUAL_TESTS")

	app := application.NewApp(testAppName)
	err := app.LoadConfig(nil)
	if err != nil {
		t.Fatalf("Error loading config; %v", err)
	}

	if err := app.SetupLogging(); err != nil {
		t.Fatalf("Error setting up logging; %v", err)
	}

	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))

	// N.B. Server currently needs to be started manually. Should we start it autommatically?
	cfg := app.GetConfig()

	// N.B. There's probably a race condition here because the client might start before the server is fully up.
	// Or maybe that's implicitly handled because the connection won't succeed until the server is up?
	addr, err := waitForServer(*cfg)
	if err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}

	log.Info("Server started")
	_, err = runWebsocketClientConcurrent(addr)
	if err != nil {
		t.Fatalf("Error running client for addres %v; %v", addr, err)
	}
}

func setupAndRunServer(cfg config.Config) error {
	log := zapr.NewLogger(zap.L())

	client, err := ai.NewClient(*cfg.OpenAI)
	if err != nil {
		return errors.Wrap(err, "Failed to create client")
	}

	agentOptions := &ai.AgentOptions{}

	if err := agentOptions.FromAssistantConfig(*cfg.CloudAssistant); err != nil {
		return err
	}

	agentOptions.Client = client

	agent, err := ai.NewAgent(*agentOptions)
	if err != nil {
		return err
	}

	serverOptions := &Options{
		Telemetry: cfg.Telemetry,
		Server:    cfg.AssistantServer,
	}
	srv, err := NewServer(*serverOptions, agent)
	if err != nil {
		return errors.Wrap(err, "Failed to create server")
	}
	go func() {
		if err := srv.Run(); err != nil {
			log.Error(err, "Error running server")
		}
		log.Info("Shutting down server...")
		srv.shutdown()
	}()
	log.Info("Server stopped")
	return nil
}

func waitForServer(cfg config.Config) (string, error) {
	addr := fmt.Sprintf("https://localhost:%d", cfg.AssistantServer.Port)
	if cfg.AssistantServer.TLSConfig == nil {
		addr = fmt.Sprintf("http://localhost:%d", cfg.AssistantServer.Port)
	}

	u, err := url.Parse(addr)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse URL")
	}

	log := zapr.NewLogger(zap.L())
	log.Info("Waiting for server to start", "address", addr)
	endTime := time.Now().Add(30 * time.Second)
	wait := 2 * time.Second
	for time.Now().Before(endTime) {
		var customHTTPClient *http.Client
		if u.Scheme == "https" {
			customTransport := &http2.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // ⚠️ Accept ANY cert (dangerous in production!)
				},
				DialTLSContext: func(ctx context.Context, network, addr string, config *tls.Config) (net.Conn, error) {
					// Create a secure connection with TLS
					return tls.Dial(network, addr, config)
				},
			}
			customHTTPClient = &http.Client{
				Timeout:   10 * time.Second,
				Transport: customTransport,
			}
		} else {
			customTransport := &http.Transport{}
			customHTTPClient = &http.Client{
				Timeout:   10 * time.Second,
				Transport: customTransport,
			}
		}

		client := connect.NewClient[grpc_health_v1.HealthCheckRequest, grpc_health_v1.HealthCheckResponse](
			customHTTPClient,
			addr+"/grpc.health.v1.Health/Check", // Adjust if using a different route
			connect.WithGRPCWeb(),
		)

		resp, err := client.CallUnary(context.Background(), connect.NewRequest(&grpc_health_v1.HealthCheckRequest{}))
		if err != nil {
			time.Sleep(wait)
			continue
		}

		if resp.Msg.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return addr, nil
		} else {
			log.Info("Server not ready", "status", resp.Msg.GetStatus())
		}
	}
	return "", errors.Errorf("Server didn't start in time")
}

func dialWebsocketConn(runID string, base *url.URL) (*websocket.Conn, error) {
	rawQuery := fmt.Sprintf("runID=%s&id=%s", runID, uuid.NewString())
	u := url.URL{Scheme: "ws", Host: base.Host, Path: "/ws", RawQuery: rawQuery}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to dial; %v", err)
	}
	return c, nil
}

func runWebsocketClient(baseURL string) (map[string]any, error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log := logs.NewLogger()

	cells := make(map[string]any)

	base, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return cells, errors.Wrapf(err, "Failed to parse URL")
	}

	run1 := uuid.NewString()
	c1, err := dialWebsocketConn(run1, base)
	if err != nil {
		return cells, err
	}
	defer func() {
		if err := c1.Close(); err != nil {
			log.Error(err, "Could not close websocket")
		}
	}()

	// Send one command
	if err := sendExecuteRequest(c1, newExecuteRequest(run1, []string{"ls -la"})); err != nil {
		return cells, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	// Wait for the command to finish.
	cell, err := waitForCommandToFinish(c1)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to wait for command to finish; %v", err)
	}

	log.Info("Cell", "cell", logs.ZapProto("cell", cell))

	run2 := uuid.NewString()
	c2, err := dialWebsocketConn(run2, base)
	if err != nil {
		return cells, err
	}
	defer func() {
		if err := c2.Close(); err != nil {
			log.Error(err, "Could not close websocket")
		}
	}()

	// Send second command
	if err := sendExecuteRequest(c2, newExecuteRequest(run2, []string{"echo The date is $(DATE)"})); err != nil {
		return cells, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	// Wait for the command to finish.
	cell, err = waitForCommandToFinish(c2)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to wait for command to finish; %v", err)
	}

	log.Info("Cell", "cell", logs.ZapProto("cell", cell))

	return cells, nil
}

func runWebsocketClientConcurrent(baseURL string) (map[string]any, error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log := logs.NewLogger()

	cells := make(map[string]any)

	base, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return cells, errors.Wrapf(err, "Failed to parse URL")
	}

	run1 := uuid.NewString()
	c1, err := dialWebsocketConn(run1, base)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to dial; %v", err)
	}
	defer func() {
		if err := c1.Close(); err != nil {
			log.Error(err, "Could not close websocket")
		}
	}()

	req1 := newExecuteRequest(run1, []string{`
for i in {1..5}
do
  echo "hello world - from 1"
  sleep 1
done`})

	// Send one command
	if err := sendExecuteRequest(c1, req1); err != nil {
		return cells, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	go func() {
		cell1, err := waitForCommandToFinish(c1)
		if err != nil {
			log.Error(err, "Failed to wait for command to finish (non-celling)")
			return
		}
		log.Info("Cell", "cell1", logs.ZapProto("cell", cell1))
		cells[run1] = cell1
	}()

	run2 := uuid.NewString()
	c2, err := dialWebsocketConn(run2, base)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to dial; %v", err)
	}
	defer func() {
		if err := c2.Close(); err != nil {
			log.Error(err, "Could not close websocket")
		}
	}()
	req2 := newExecuteRequest(run2, []string{`
for i in {1..10}
do
  echo "hello world - from 2"
  sleep 1
done`})

	// Send second command
	if err := sendExecuteRequest(c2, req2); err != nil {
		return cells, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	// Wait for the command to finish.
	cell, err := waitForCommandToFinish(c2)
	if err != nil {
		return cells, errors.Wrapf(err, "Failed to wait for command to finish; %v", err)
	}

	log.Info("Cell", "cell2", logs.ZapProto("cell", cell))
	cells[run2] = cell

	if len(cells) != 2 {
		return cells, errors.Errorf("Expected 2 cells; got %d", len(cells))
	}

	return cells, nil
}

// newExecuteRequest is a helper function to create an ExecuteRequest.
func newExecuteRequest(runID string, commands []string) *v2.ExecuteRequest {
	knownID := uuid.NewString()
	executeRequest := &v2.ExecuteRequest{
		Config: &v2.ProgramConfig{
			RunId:         runID,
			ProgramName:   "/bin/zsh",
			Arguments:     make([]string, 0),
			LanguageId:    "sh",
			Background:    false,
			FileExtension: "",
			Env: []string{
				"RUNME_ID=" + knownID,
				"RUNME_RUNNER=v2",
				"TERM=xterm-256color",
			},
			Source: &v2.ProgramConfig_Commands{
				Commands: &v2.ProgramConfig_CommandList{
					Items: commands,
				},
			},
			Interactive: true,
			Mode:        v2.CommandMode_COMMAND_MODE_INLINE,
			KnownId:     knownID,
		},
		Winsize: &v2.Winsize{Rows: 34, Cols: 100, X: 0, Y: 0},
	}
	return executeRequest
}

// sendExecuteRequest sends an ExecuteRequest to the server.
func sendExecuteRequest(c *websocket.Conn, executeRequest *v2.ExecuteRequest) error {
	socketRequest := &streamv1.WebsocketRequest{
		KnownId: executeRequest.GetConfig().GetKnownId(),
		RunId:   executeRequest.GetConfig().GetRunId(),
		Payload: &streamv1.WebsocketRequest_ExecuteRequest{
			ExecuteRequest: executeRequest,
		},
	}

	message, err := protojson.Marshal(socketRequest)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal message; %v", err)
	}

	err = c.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		return errors.Wrapf(err, "Failed to write message; %v", err)
	}
	return nil
}

func waitForCommandToFinish(c *websocket.Conn) (*parserv1.Cell, error) {
	log := logs.NewLogger()

	cell := &parserv1.Cell{
		Outputs: make([]*parserv1.CellOutput, 0),
	}

	cell.Outputs = append(cell.Outputs, &parserv1.CellOutput{
		Items: []*parserv1.CellOutputItem{
			{
				Data: []byte(""),
			},
			{
				Data: []byte(""),
			},
		},
	})
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Error(err, "read error")
		}

		response := &streamv1.WebsocketResponse{}
		if err := protojson.Unmarshal(message, response); err != nil {
			log.Error(err, "Failed to unmarshal message")
			return cell, errors.Wrapf(err, "Failed to unmarshal message; %v", err)
		}
		if response.GetStatus() != nil && response.GetStatus().GetCode() != code.Code_OK {
			return cell, errors.New(response.GetStatus().GetMessage())
		}
		if response.GetExecuteResponse() != nil {
			resp := response.GetExecuteResponse()

			cell.Outputs[0].Items[0].Data = append(cell.Outputs[0].Items[0].Data, resp.StdoutData...)
			cell.Outputs[0].Items[1].Data = append(cell.Outputs[0].Items[1].Data, resp.StderrData...)

			if resp.GetExitCode() != nil {
				// Use ExitCode to determine if the message indicates the end of the program
				return cell, nil
			}
			log.Info("Command Response", "stdout", string(resp.StdoutData), "stderr", string(resp.StderrData), "exitCode", resp.ExitCode)
		} else {
			log.Info("received", "message", string(message))
		}
	}
}

func SkipIfMissing(t *testing.T, env string) string {
	t.Helper()
	if value, ok := os.LookupEnv(env); ok {
		return value
	}
	t.Skipf("missing %s", env)
	return ""
}
