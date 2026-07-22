package stream

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/code"

	streamv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/stream/v1"
	"github.com/runmedev/runme/v3/pkg/agent/iam"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/runme"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Implement origin checking as needed
		// TODO(jlewi): Do we need to check ORIGIN?
		return true
	},
}

// WebSocketHandler is a handler for websockets. A single instance is registered with the http server
// to connect websocket requests to RunmeHandlers.
type WebSocketHandler struct {
	auth *iam.AuthContext

	runner *runme.Runner

	// tapFactory creates a StreamTap per run. May be nil (no recording).
	tapFactory TapFactory

	// preprocessor transforms initial ExecuteRequests before execution. May be nil.
	preprocessor RequestPreprocessor

	// clientGracePeriod overrides the default multiplexer client close grace
	// period when set. A zero value disables the grace period.
	clientGracePeriod *time.Duration

	mu   sync.Mutex
	runs map[string]*Multiplexer
}

type WebSocketHandlerOption func(*WebSocketHandler)

func WithClientGracePeriod(d time.Duration) WebSocketHandlerOption {
	return func(h *WebSocketHandler) {
		h.clientGracePeriod = &d
	}
}

func NewWebSocketHandler(runner *runme.Runner, auth *iam.AuthContext, options ...WebSocketHandlerOption) *WebSocketHandler {
	h := &WebSocketHandler{
		auth:   auth,
		runner: runner,
		runs:   make(map[string]*Multiplexer),
	}
	for _, option := range options {
		option(h)
	}
	return h
}

// SetTapFactory configures a factory that creates a StreamTap for each new run.
// If factory is nil or returns nil, recording is disabled for that run.
func (h *WebSocketHandler) SetTapFactory(factory TapFactory) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tapFactory = factory
}

// SetRequestPreprocessor configures a function that transforms initial
// ExecuteRequests (those with Config) before they reach the runner.
// If preprocessor is nil, requests pass through unchanged.
func (h *WebSocketHandler) SetRequestPreprocessor(preprocessor RequestPreprocessor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.preprocessor = preprocessor
}

// Handler is the main handler mounted in a mux to handle websocket connection upgrades.
func (h *WebSocketHandler) Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logs.FromContextWithTrace(ctx)
	log.Info("WebsocketHandler.Handler")

	if h.runner.Server == nil {
		log.Error(errors.New("Runner server is nil"), "Runner server is nil")
		http.Error(w, "Runner server is nil; server is not properly configured", http.StatusInternalServerError)
		return
	}

	// runID is a ulid to identify a run end-to-end.
	runID := r.URL.Query().Get("runID")
	if runID == "" {
		log.Error(errors.New("run id cannot be empty"), "Run id cannot be empty")
		http.Error(w, "Run id cannot be empty", http.StatusBadRequest)
		return
	}

	// streamID is a uuidv4 without dashes to identify a websocket connection.
	streamID := r.URL.Query().Get("id")
	if streamID == "" {
		log.Error(errors.New("stream cannot be empty"), "Stream cannot be empty")
		http.Error(w, "Stream cannot be empty", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err, "Could not upgrade to websocket")
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}
	sc := NewConnection(conn)

	initialRequest, err := sc.ReadWebsocketRequest(ctx)
	if err != nil {
		log.Error(err, "Could not read initial websocket request")
		return
	}

	var multiplex *Multiplexer
	var created bool
	if initialRequest.GetOpenRunRequest() != nil {
		multiplex, created, err = h.handleNegotiatedConnection(ctx, runID, streamID, sc, initialRequest)
		if err != nil {
			log.Error(err, "Could not negotiate websocket connection")
			return
		}
	} else {
		multiplex, created, err = h.handleConnection(ctx, runID, streamID, sc, initialRequest)
	}
	if err != nil {
		log.Error(err, "Could not handle websocket connection")
		_ = sc.Error("Could not handle websocket connection")
		return
	}

	wait := false
	if created {
		wait = multiplex.process()
		h.removeRun(ctx, runID)
	}

	log.Info("Websocket handler finished", "runID", runID, "streamID", streamID, "wait", wait)
}

// handleConnection accepts a websocket connection as a stream into a multiplexer.
func (h *WebSocketHandler) handleConnection(ctx context.Context, runID string, streamID string, sc *Connection, initialRequest *streamv1.WebsocketRequest) (*Multiplexer, bool, error) {
	log := logs.FromContextWithTrace(ctx)
	log.Info("WebSocketHandler.handleConnection", "runID", runID, "streamID", streamID)

	h.mu.Lock()
	defer h.mu.Unlock()

	// If we already have a run, accept the connection on the existing multiplexer.
	created := false
	multiplex, ok := h.runs[runID]
	if !ok {
		multiplex = h.newMultiplexer(ctx, runID)
		h.runs[runID] = multiplex
		created = true
	}

	if err := multiplex.acceptConnection(streamID, sc, initialRequest); err != nil {
		if created {
			delete(h.runs, runID)
		}
		return nil, false, errors.Wrap(err, "could not accept connection")
	}

	return multiplex, created, nil
}

// handleNegotiatedConnection binds a stream only after the client explicitly
// identifies whether it is starting or resuming a run. Protocol-level errors
// are sent to the client before this method returns.
func (h *WebSocketHandler) handleNegotiatedConnection(ctx context.Context, runID string, streamID string, sc *Connection, req *streamv1.WebsocketRequest) (*Multiplexer, bool, error) {
	if err := validateRequestEnvelope(ctx, h.auth, runID, req); err != nil {
		sc.ErrorMessage(ctx, code.Code_PERMISSION_DENIED, err)
		return nil, false, err
	}

	intent := req.GetOpenRunRequest().GetIntent()
	h.mu.Lock()
	defer h.mu.Unlock()

	multiplex, exists := h.runs[runID]
	created := false
	state := streamv1.RunState_RUN_STATE_UNSPECIFIED
	switch intent {
	case streamv1.RunIntent_RUN_INTENT_START:
		if exists {
			err := errors.New("run already exists")
			sc.ErrorMessage(ctx, code.Code_ALREADY_EXISTS, err)
			return nil, false, err
		}
		multiplex = h.newMultiplexer(ctx, runID)
		h.runs[runID] = multiplex
		created = true
		state = streamv1.RunState_RUN_STATE_CREATED
	case streamv1.RunIntent_RUN_INTENT_RESUME:
		if !exists {
			err := errors.New("run not found")
			sc.ErrorMessage(ctx, code.Code_NOT_FOUND, err)
			return nil, false, err
		}
		state = streamv1.RunState_RUN_STATE_RUNNING
	default:
		err := errors.New("run intent must be START or RESUME")
		sc.ErrorMessage(ctx, code.Code_INVALID_ARGUMENT, err)
		return nil, false, err
	}

	if err := multiplex.streams.authorizeRequest(ctx, streamID, runID, sc, req); err != nil {
		if created {
			delete(h.runs, runID)
		}
		return nil, false, err
	}
	if err := multiplex.acceptConnection(streamID, sc, nil); err != nil {
		if created {
			delete(h.runs, runID)
		}
		sc.ErrorMessage(ctx, code.Code_INTERNAL, err)
		return nil, false, errors.Wrap(err, "could not accept negotiated connection")
	}

	resp := &streamv1.WebsocketResponse{
		Status: &streamv1.WebsocketStatus{Code: code.Code_OK},
		Payload: &streamv1.WebsocketResponse_OpenRunResponse{
			OpenRunResponse: &streamv1.OpenRunResponse{State: state},
		},
	}
	if err := sc.WriteWebsocketResponse(ctx, resp); err != nil {
		multiplex.streams.removeStream(ctx, streamID)
		if created {
			delete(h.runs, runID)
		}
		return nil, false, errors.Wrap(err, "could not acknowledge negotiated connection")
	}

	return multiplex, created, nil
}

// newMultiplexer constructs a run while h.mu is held by the caller.
func (h *WebSocketHandler) newMultiplexer(ctx context.Context, runID string) *Multiplexer {
	var tap StreamTap
	if h.tapFactory != nil {
		tap = h.tapFactory(runID)
	}
	var options *MultiplexerOptions
	if h.clientGracePeriod != nil {
		options = &MultiplexerOptions{ClientGracePeriod: *h.clientGracePeriod}
	}
	return NewMultiplexer(ctx, runID, h.auth, h.runner, tap, h.preprocessor, options)
}

// Shutdown cancels all active multiplexers and waits until each one completes
// its close path or ctx expires.
func (h *WebSocketHandler) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	multiplexers := make([]*Multiplexer, 0, len(h.runs))
	for _, m := range h.runs {
		multiplexers = append(multiplexers, m)
		m.cancel()
	}
	h.mu.Unlock()

	for _, m := range multiplexers {
		if err := m.Wait(ctx); err != nil {
			return fmt.Errorf("wait for run %s finalization: %w", m.runID, err)
		}
	}
	return nil
}

// removeRun removes a run from the handler. It is called when the processor is done.
func (h *WebSocketHandler) removeRun(ctx context.Context, runID string) {
	log := logs.FromContextWithTrace(ctx)
	log.Info("WebSocketHandler.removeRun", "runID", runID)

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.runs, runID)
	log.Info("WebSocketHandler.removeRun: run deleted", "runID", runID)
}
