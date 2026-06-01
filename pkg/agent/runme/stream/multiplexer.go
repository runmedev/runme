package stream

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/protobuf/encoding/protojson"

	streamv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/stream/v1"
	"github.com/runmedev/runme/v3/pkg/agent/iam"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/runme"
)

// Stream client grace period (package-level, for testability).
var ClientGracePeriod = 30 * time.Second

// MultiplexerOptions configures runtime behavior for a Multiplexer.
type MultiplexerOptions struct {
	// ClientGracePeriod is how long to wait in close() before force-closing
	// websocket streams, giving clients a chance to close first.
	// If options are nil, defaults to ClientGracePeriod.
	ClientGracePeriod time.Duration
}

// Multiplexer timeout and interval for inactivity (package-level, for testability).
var (
	MultiplexerTimeout  = 20 * time.Minute
	MultiplexerInterval = 30 * time.Second
)

// Multiplexer manages websocket connections, runme.Runner Execution bidirectional processing, and request/response multiplexing
// for a given runID. It handles multiple streams and clients, coordinating authenticated requests and responses between them
// and the Runme runner. The same multiplexer bridges the v2.ExecuteRequest and v2.ExecuteResponse for a run in runme.Runner
// for one or many Console DOM element with the same runID.
// Todo(sebastian): Deduplicate Cell ID to the runID to peg the run to a specific cell.
type Multiplexer struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	runID string

	auth    *iam.AuthContext
	runner  *runme.Runner
	streams *Streams

	// tap receives session lifecycle and data events. Never nil (uses noopTap as default).
	tap StreamTap

	// preprocessor transforms initial ExecuteRequests before they reach the runner. May be nil.
	preprocessor RequestPreprocessor

	// authedWebsocketRequests is a channel that receives socket requests from authenticated clients.
	authedWebsocketRequests chan *streamv1.WebsocketRequest

	// clientGracePeriod controls the close grace period for websocket clients.
	clientGracePeriod time.Duration

	mu sync.Mutex
	// p is the processor that is currently processing messages. If p is nil then no run against runme.Runner is currently processing
	p *Processor
}

// NewMultiplexer creates a new Multiplexer (see description above).
// tap may be nil, in which case recording is disabled.
// preprocessor may be nil, in which case requests pass through unchanged.
func NewMultiplexer(ctx context.Context, runID string, auth *iam.AuthContext, runner *runme.Runner, tap StreamTap, preprocessor RequestPreprocessor, options *MultiplexerOptions) *Multiplexer {
	if tap == nil {
		tap = noopTap{}
	}
	clientGracePeriod := ClientGracePeriod
	if options != nil {
		clientGracePeriod = options.ClientGracePeriod
	}

	ctx, cancel := context.WithCancel(ctx)
	m := &Multiplexer{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),

		runID:             runID,
		auth:              auth,
		runner:            runner,
		tap:               tap,
		preprocessor:      preprocessor,
		clientGracePeriod: clientGracePeriod,
	}

	m.authedWebsocketRequests = make(chan *streamv1.WebsocketRequest, 100)
	streams := NewStreams(ctx, auth, m.authedWebsocketRequests)
	m.streams = streams

	return m
}

func (m *Multiplexer) acceptConnection(streamID string, sc *Connection) error {
	log := logs.FromContextWithTrace(m.ctx)

	if err := m.streams.createStream(streamID, sc); err != nil {
		log.Error(err, "Could not create stream")
		return err
	}

	m.tap.ClientConnect(streamID)

	// Start a goroutine to receive requests for a specific stream.
	go m.receiveRequests(streamID, sc)

	// Start a goroutine to enforce a timeout with an interval for inactivity per stream.
	go m.startInactivityTimeout(MultiplexerTimeout, MultiplexerInterval)

	return nil
}

// receiveRequests handles receiving socket requests for a specific stream in a goroutine.
func (m *Multiplexer) receiveRequests(streamID string, sc *Connection) {
	tracer := otel.Tracer("github.com/runmedev/runme/v3/pkg/agent/runme/stream")
	ctx, span := tracer.Start(m.ctx, "Multiplexer.receiveRequests")
	// todo(sebastian): ideally we set attributes from the context so we don't have set them every time.
	span.SetAttributes(
		attribute.String("streamID", streamID),
		attribute.String("runID", m.runID),
	)
	defer span.End()

	defer func() {
		m.tap.ClientDisconnect(streamID)
		m.streams.removeStream(ctx, streamID)
	}()
	log := logs.FromContextWithTrace(ctx)

	if err := m.streams.receive(ctx, streamID, m.runID, sc); err != nil {
		closeErr, ok := err.(*websocket.CloseError)
		if !ok {
			log.Error(err, "Unexpected error while receiving socket requests")
			return
		}

		log.Info("Connection closed", "streamID", streamID, "closeCode", closeErr.Code, "closeText", closeErr.Error())
	}
}

// startInactivityTimeout enforces a timeout for inactivity (not actively executing requests) using an interval.
func (m *Multiplexer) startInactivityTimeout(timeout time.Duration, interval time.Duration) {
	log := logs.FromContextWithTrace(m.ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-m.ctx.Done():
			// Multiplexer already canceled, exit goroutine.
			return
		case <-ticker.C:
			if p := m.getInflight(); p != nil && p.ActiveRequests {
				// Processor actively executing requests, exit goroutine.
				return
			}
			if time.Now().After(deadline) {
				// Timeout reached, processor not executing requests, cancel multiplexer.
				log.Info("Inactivity timeout reached, canceling multiplexer", "runID", m.runID)
				m.streams.error(m.ctx, code.Code_DEADLINE_EXCEEDED, errors.New("inactivity timeout"))
				m.cancel() // will grant 30s grace period for clients to close connection.
				return
			}
		}
	}
}

// close shuts down the RunmeMultiplexer.
func (m *Multiplexer) close() {
	defer close(m.done)

	p := m.getInflight()
	if p != nil {
		p.close()
	}
	m.setInflight(nil)

	// Finalize the recording before client grace. Client grace is about
	// disconnect/reconnect semantics and should not delay RunEnd delivery.
	m.tap.RunEnd()
	_ = m.tap.Close()

	// Wait for the client grace period to give the client a chance to close the connection.
	// Intentionally do not short-circuit on m.ctx.Done(); this grace period gives
	// clients a chance to receive disconnect signaling triggered by cancellation.
	time.Sleep(m.clientGracePeriod)
	// With Runme's execution finished we can close all websocket connections.
	m.streams.close(m.ctx)
}

// Wait blocks until the multiplexer has finished its close path, including
// tap finalization. Shutdown callers should use this after canceling the
// multiplexer when they need RunEnd/Close delivery to complete.
func (m *Multiplexer) Wait(ctx context.Context) error {
	select {
	case <-m.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// process manages request processing for a runID. Returns false if a run is already in flight.
// Launches goroutines to execute requests and broadcast responses, then forwards ExecuteRequests
// to the processor until context cancellation or channel closure. Handles cleanup on exit.
// Todo(sebastian): Can we get away without the wait flag? Had premature closing issues without it.
func (m *Multiplexer) process() (wait bool) {
	wait = true

	tracer := otel.Tracer("github.com/runmedev/runme/v3/pkg/agent/runme/stream")
	ctx, span := tracer.Start(m.ctx, "Multiplexer.process")
	defer span.End()
	log := logs.FromContextWithTrace(ctx)

	// todo(sebastian): Still have to decide what to do if a user tries to send a new request
	// before the current's done as below. The cleanest solution might be to SIGINT the run in runme.Runner.
	p := m.getInflight()
	if p != nil {
		log.Info("Already have a run in flight", "runID", m.runID)
		wait = false
		return
	}
	p = NewProcessor(ctx, m.runID)
	m.setInflight(p)

	m.tap.RunStart(m.runID)

	// Start a goroutine to execute requests against runme server.
	go m.execute(p)
	// Start a separate goroutine to broadcast responses to all clients.
	go m.broadcastResponses(p)

	// TODO(jlewi): What should we do if a user tries to send a new request before the current one has finished?
	// How can we detect if its a new request? Should we check if anything other than a "Stop" request is sent
	// after the first request? Right now we are just passing it along to RunME. Hopefully, RunMe handles it.

	// Put the request on the channel
	// Access the local variable to ensure its always set at this point and avoid race conditions.

	// When the authedWebsocketRequests channel closes Runme finished executing the command.
	defer m.close()

	for {
		select {
		case <-m.ctx.Done():
			log.Info("Context done, no need to process more requests")
			return
		case req, ok := <-m.authedWebsocketRequests:
			if !ok {
				log.Info("Closing authedWebsocketRequests channel")
				return
			}
			if req.GetExecuteRequest() == nil {
				log.Info("Received message doesn't contain an ExecuteRequest")
				continue
			}

			execReq := req.GetExecuteRequest()

			// Record input data.
			if len(execReq.GetInputData()) > 0 {
				m.tap.Input(execReq.GetInputData())
			}

			// Record winsize changes.
			if ws := execReq.GetWinsize(); ws != nil {
				m.tap.Resize(ws.GetCols(), ws.GetRows())
			}

			// Preprocess the initial request (Config present) before recording and execution.
			if cfg := execReq.GetConfig(); cfg != nil && m.preprocessor != nil {
				modified, ppErr := m.preprocessor(execReq)
				if ppErr != nil {
					log.Error(ppErr, "Request preprocessor failed, passing original request")
				} else if modified == nil {
					log.Error(errors.New("request preprocessor returned nil request"), "Passing original request")
				} else {
					execReq = modified
				}
			}

			// Record command start when a Config is present (first request).
			if cfg := execReq.GetConfig(); cfg != nil {
				program := cfg.GetProgramName()
				if program == "" {
					if cmds := cfg.GetCommands(); cmds != nil && len(cmds.GetItems()) > 0 {
						program = cmds.GetItems()[0]
					}
				}
				m.tap.CommandStart(program, cfg.GetDirectory())
			}

			p.ExecuteRequests <- execReq
		}
	}
}

func (m *Multiplexer) getInflight() *Processor {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.p
}

func (m *Multiplexer) setInflight(p *Processor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p = p
}

// execute invokes the Runme runner to execute the request.
// It returns when the request has been processed by Runme.
func (m *Multiplexer) execute(p *Processor) {
	tracer := otel.Tracer("github.com/runmedev/runme/v3/pkg/agent/runme/stream")
	ctx, span := tracer.Start(m.ctx, "Multiplexer.execute")
	defer span.End()

	// On exit we cancel the context because Runme execution is finished.
	defer m.cancel()

	log := logs.FromContextWithTrace(ctx)
	// Send the request to the runner
	if err := m.runner.Server.Execute(p); err != nil {
		log.Error(err, "Failed to execute request")
		return
	}
}

// broadcastResponses listens for all the responses and sends them over the websocket connection.
func (m *Multiplexer) broadcastResponses(p *Processor) {
	tracer := otel.Tracer("github.com/runmedev/runme/v3/pkg/agent/runme/stream")
	ctx, span := tracer.Start(m.ctx, "Multiplexer.broadcastResponses")
	log := logs.FromContextWithTrace(ctx)
	defer span.End()

	for {
		res, ok := <-p.ExecuteResponses
		if !ok {
			log.Info("Channel to SocketProcessor closed")
			// The channel is closed, no more responses to broadcast.
			return
		}

		// Record stdout data.
		if len(res.GetStdoutData()) > 0 {
			m.tap.Output(res.GetStdoutData())
		}
		// Record stderr data.
		if len(res.GetStderrData()) > 0 {
			m.tap.Stderr(res.GetStderrData())
		}
		// Record command end when exit code is present.
		if res.GetExitCode() != nil {
			m.tap.CommandEnd(int(res.GetExitCode().GetValue()))
		}

		response := &streamv1.WebsocketResponse{
			Status: &streamv1.WebsocketStatus{
				Code: code.Code_OK,
			},
			Payload: &streamv1.WebsocketResponse_ExecuteResponse{
				ExecuteResponse: res,
			},
		}
		responseData, err := protojson.Marshal(response)
		if err != nil {
			log.Error(err, "Could not marshal response")
		}

		if err := m.streams.broadcast(ctx, responseData); err != nil {
			log.Error(err, "Could not broadcast response")
		}
	}
}
