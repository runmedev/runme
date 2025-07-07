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
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1/agentv1connect"

	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/logs"

	v2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	streamv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/stream/v1"
)

const testAppName = "runme-agent"

func Test_HealthCheck(t *testing.T) {
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
	addr := "https://localhost:9080"
	if err := waitForServer(addr); err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}
	t.Logf("Server started")
}

func Test_GenerateBlocks(t *testing.T) {
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
	addr := fmt.Sprintf("https://localhost:%v", cfg.AssistantServer.Port)

	go func() {
		if err := setupAndRunServer(*cfg); err != nil {
			log.Error(err, "Error running server")
		}
	}()

	// N.B. There's probably a race condition here because the client might start before the server is fully up.
	// Or maybe that's implicitly handled because the connection won't succeed until the server is up?
	if err := waitForServer(addr); err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}

	log.Info("Server started")
	blocks, err := runAIClient(addr)
	if err != nil {
		t.Fatalf("Error running client for addres %v; %v", addr, err)
	}

	if len(blocks) < 2 {
		t.Errorf("Expected at least 2 blocks; got %d blocks", len(blocks))
	}

	// Ensure there is a filesearch results block and that the filenames are set. This is intended
	// to catch various bugs with the SDK;
	hasFSBlock := false
	for _, b := range blocks {
		if b.Kind == agentv1.BlockKind_BLOCK_KIND_FILE_SEARCH_RESULTS {
			hasFSBlock = true

			if len(b.FileSearchResults) <= 0 {
				t.Errorf("FileSearchResults block has no results")
			}

			for _, r := range b.FileSearchResults {
				if r.FileName == "" {
					t.Errorf("FileSearchResults block has empty filename")
				}
			}
		}
	}

	if !hasFSBlock {
		t.Errorf("There was no FileSearch block in the results.")
	}
}

func runAIClient(baseURL string) (map[string]*agentv1.Block, error) {
	log := zapr.NewLoggerWithOptions(zap.L(), zapr.AllowZapFields(true))

	blocks := make(map[string]*agentv1.Block)

	Block := agentv1.Block{
		Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
		Contents: "This is a block",
	}

	log.Info("Block", logs.ZapProto("block", &Block))

	u, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return blocks, errors.Wrapf(err, "Failed to parse URL")
	}

	var client agentv1connect.BlocksServiceClient

	// Mimic what the frontend does
	options := []connect.ClientOption{connect.WithGRPCWeb()}
	if u.Scheme == "https" {
		// Configure the TLS settings
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Set to true only for testing; otherwise validate the server's certificate
		}

		client = agentv1connect.NewBlocksServiceClient(
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
		client = agentv1connect.NewBlocksServiceClient(
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

	ctx := context.Background()
	genReq := &agentv1.GenerateRequest{
		Blocks: []*agentv1.Block{
			{
				Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
				Role:     agentv1.BlockRole_BLOCK_ROLE_USER,
				Contents: "Show me all the AKS clusters at OpenAI",
			},
		},
	}
	req := connect.NewRequest(genReq)

	stream, err := client.Generate(ctx, req)
	if err != nil {
		return blocks, errors.Wrapf(err, "Failed to create generate stream")
	}

	// Receive responses
	for stream.Receive() {
		response := stream.Msg()

		for _, block := range response.Blocks {
			blocks[block.Id] = block

			options := protojson.MarshalOptions{
				Multiline: true,
				Indent:    "  ", // Two spaces for indentation
			}

			// Marshal the protobuf message to JSON
			jsonData, err := options.Marshal(block)
			if err != nil {
				log.Error(err, "Failed to marshal block to JSON")
			} else {
				log.Info("Block", "block", string(jsonData))
			}
		}

	}

	if stream.Err() != nil {
		return blocks, errors.Wrapf(stream.Err(), "Error receiving response")
	}
	return blocks, nil
}

func Test_ExecuteWithRunme(t *testing.T) {
	SkipIfMissing(t, "RUN_MANUAL_TESTS")

	app := application.NewApp(testAppName)
	err := app.LoadConfig(nil)
	if err != nil {
		t.Fatalf("Error loading config; %v", err)
	}
	cfg := app.AppConfig

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
	// N.B. Server currently needs to be started manually. Should we start it autommatically?
	addr := fmt.Sprintf("http://localhost:%v", cfg.AssistantServer.Port)
	go func() {
		c := app.GetConfig()
		if err := setupAndRunServer(*c); err != nil {
			log.Error(err, "Error running server")
		}
	}()

	// N.B. There's probably a race condition here because the client might start before the server is fully up.
	// Or maybe that's implicitly handled because the connection won't succeed until the server is up?
	if err := waitForServer(addr); err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}

	log.Info("Server started")
	_, err = runRunmeClient(addr)
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
	cfg := app.AppConfig

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
	// N.B. Server currently needs to be started manually. Should we start it autommatically?
	addr := fmt.Sprintf("http://localhost:%v", cfg.AssistantServer.Port)
	go func() {
		c := app.GetConfig()
		if err := setupAndRunServer(*c); err != nil {
			log.Error(err, "Error running server")
		}
	}()

	// N.B. There's probably a race condition here because the client might start before the server is fully up.
	// Or maybe that's implicitly handled because the connection won't succeed until the server is up?
	if err := waitForServer(addr); err != nil {
		t.Fatalf("Error waiting for server; %v", err)
	}

	log.Info("Server started")
	_, err = runRunmeClientConcurrent(addr)
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

func waitForServer(addr string) error {
	log := zapr.NewLogger(zap.L())
	log.Info("Waiting for server to start", "address", addr)
	endTime := time.Now().Add(30 * time.Second)
	wait := 2 * time.Second
	for time.Now().Before(endTime) {

		// 1. Create a custom transport that skips TLS verification
		// Important we need to use http2.Transport to support HTTP/2 and properly handle trailers with gRPCWeb
		customTransport := &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // ⚠️ Accept ANY cert (dangerous in production!)

			},
			DialTLSContext: func(ctx context.Context, network, addr string, config *tls.Config) (net.Conn, error) {
				// Create a secure connection with TLS
				return tls.Dial(network, addr, config)
			},
		}

		// 2. Create an HTTP client using the custom transport
		customHTTPClient := &http.Client{
			Timeout:   10 * time.Second,
			Transport: customTransport,
		}

		client := connect.NewClient[grpc_health_v1.HealthCheckRequest, grpc_health_v1.HealthCheckResponse](
			customHTTPClient,
			// http.DefaultClient,
			addr+"/grpc.health.v1.Health/Check", // Adjust if using a different route
			// N.B. We use GRPCWeb to mimic what the frontend does. The frontend will use GRPCWeb to support streaming.
			connect.WithGRPCWeb(),
		)

		resp, err := client.CallUnary(context.Background(), connect.NewRequest(&grpc_health_v1.HealthCheckRequest{}))
		if err != nil {
			time.Sleep(wait)
			continue
		}

		if resp.Msg.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return nil
		} else {
			log.Info("Server not ready", "status", resp.Msg.GetStatus())
		}
	}
	return errors.Errorf("Server didn't start in time")
}

func runRunmeClient(baseURL string) (map[string]any, error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log := logs.NewLogger()

	blocks := make(map[string]any)

	base, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return blocks, errors.Wrapf(err, "Failed to parse URL")
	}

	u := url.URL{Scheme: "ws", Host: base.Host, Path: "/ws"}
	log.Info("connecting to", "host", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return blocks, errors.Wrapf(err, "Failed to dial; %v", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Error(err, "Could not close websocket")
		}
	}()

	// Send one command
	if err := sendExecuteRequest(c, newExecuteRequest([]string{"ls -la"})); err != nil {
		return blocks, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	// Wait for the command to finish.
	block, err := waitForCommandToFinish(c)
	if err != nil {
		return blocks, errors.Wrapf(err, "Failed to wait for command to finish; %v", err)
	}

	log.Info("Block", "block", logs.ZapProto("block", block))

	// Send second command
	if err := sendExecuteRequest(c, newExecuteRequest([]string{"echo The date is $(DATE)"})); err != nil {
		return blocks, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	// Wait for the command to finish.
	block, err = waitForCommandToFinish(c)
	if err != nil {
		return blocks, errors.Wrapf(err, "Failed to wait for command to finish; %v", err)
	}

	log.Info("Block", "block", logs.ZapProto("block", block))

	return blocks, nil
}

func runRunmeClientConcurrent(baseURL string) (map[string]any, error) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log := logs.NewLogger()

	blocks := make(map[string]any)

	base, err := url.Parse(baseURL)
	if err != nil {
		log.Error(err, "Failed to parse URL")
		return blocks, errors.Wrapf(err, "Failed to parse URL")
	}

	u := url.URL{Scheme: "ws", Host: base.Host, Path: "/ws"}
	log.Info("connecting to", "host", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return blocks, errors.Wrapf(err, "Failed to dial; %v", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Error(err, "Could not close websocket")
		}
	}()

	req := newExecuteRequest([]string{`
for i in {1..10}
do
  echo "hello world - from 1"
  sleep 1
done`})

	// Send one command
	if err := sendExecuteRequest(c, req); err != nil {
		return blocks, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	req2 := newExecuteRequest([]string{`
for i in {1..10}
do
  echo "hello world - from 2"
  sleep 1
done`})

	// Send second command
	if err := sendExecuteRequest(c, req2); err != nil {
		return blocks, errors.Wrapf(err, "Failed to send execute request; %v", err)
	}

	// Wait for the command to finish.
	block, err := waitForCommandToFinish(c)
	if err != nil {
		return blocks, errors.Wrapf(err, "Failed to wait for command to finish; %v", err)
	}

	log.Info("Block", "block", logs.ZapProto("block", block))

	return blocks, nil
}

// newExecuteRequest is a helper function to create an ExecuteRequest.
func newExecuteRequest(commands []string) *v2.ExecuteRequest {
	executeRequest := &v2.ExecuteRequest{
		Config: &v2.ProgramConfig{
			ProgramName:   "/bin/zsh",
			Arguments:     make([]string, 0),
			LanguageId:    "sh",
			Background:    false,
			FileExtension: "",
			Env: []string{
				`RUNME_ID=${blockID}`,
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
			KnownId:     uuid.NewString(),
		},
		Winsize: &v2.Winsize{Rows: 34, Cols: 100, X: 0, Y: 0},
	}
	return executeRequest
}

// sendExecuteRequest sends an ExecuteRequest to the server.
func sendExecuteRequest(c *websocket.Conn, executeRequest *v2.ExecuteRequest) error {
	socketRequest := &streamv1.WebsocketRequest{
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

func waitForCommandToFinish(c *websocket.Conn) (*agentv1.Block, error) {
	log := logs.NewLogger()

	block := &agentv1.Block{
		Outputs: make([]*agentv1.BlockOutput, 0),
	}

	block.Outputs = append(block.Outputs, &agentv1.BlockOutput{
		Items: []*agentv1.BlockOutputItem{
			{
				TextData: "",
			},
			{
				TextData: "",
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
			return block, errors.Wrapf(err, "Failed to unmarshal message; %v", err)
		}
		if response.GetExecuteResponse() != nil {
			resp := response.GetExecuteResponse()

			block.Outputs[0].Items[0].TextData += string(resp.StdoutData)
			block.Outputs[0].Items[1].TextData += string(resp.StderrData)

			if resp.GetExitCode() != nil {
				// Use ExitCode to determine if the message indicates the end of the program
				return block, nil
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
