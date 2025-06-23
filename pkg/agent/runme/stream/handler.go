package stream

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

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

	mu   sync.Mutex
	runs map[string]*Multiplexer
}

func NewWebSocketHandler(runner *runme.Runner, auth *iam.AuthContext) *WebSocketHandler {
	return &WebSocketHandler{
		auth:   auth,
		runner: runner,
		runs:   make(map[string]*Multiplexer),
	}
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

	multiplex, err := h.handleConnection(ctx, runID, streamID, sc)
	if err != nil {
		log.Error(err, "Could not handle websocket connection")
		_ = sc.Error("Could not handle websocket connection")
		return
	}

	// If the processor was blocking, we remove the run from the handler.
	wait := multiplex.process()
	if wait {
		h.removeRun(ctx, runID)
	}

	log.Info("Websocket handler finished", "runID", runID, "streamID", streamID, "wait", wait)
}

// handleConnection accepts a websocket connection as a stream into a multiplexer.
func (h *WebSocketHandler) handleConnection(ctx context.Context, runID string, streamID string, sc *Connection) (*Multiplexer, error) {
	log := logs.FromContextWithTrace(ctx)
	log.Info("WebSocketHandler.handleConnection", "runID", runID, "streamID", streamID)

	h.mu.Lock()
	defer h.mu.Unlock()

	// If we already have a run, accept the connection on the existing multiplexer.
	multiplex, ok := h.runs[runID]
	if !ok {
		multiplex = NewMultiplexer(ctx, runID, h.auth, h.runner)
		h.runs[runID] = multiplex
	}

	if err := multiplex.acceptConnection(streamID, sc); err != nil {
		return nil, errors.Wrap(err, "could not accept connection")
	}

	return multiplex, nil
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
