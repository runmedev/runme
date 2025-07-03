package stream

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/runme"

	v2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
)

// Processor handles the v2.ExecuteRequest and v2.ExecuteResponse for a run in runme.Runner.
type Processor struct {
	Ctx   context.Context
	RunID string
	// ActiveRequests is used to signal to the multiplexer that the processor is actively executing requests.
	ActiveRequests   bool
	ExecuteRequests  chan *v2.ExecuteRequest
	ExecuteResponses chan *v2.ExecuteResponse
	// StopReading is used to signal to the readMessages goroutine that it should stop reading messages
	StopReading chan bool

	Runner *runme.Runner
}

// NewProcessor creates a new Processor that handles the execution requests and responses for a run.
func NewProcessor(ctx context.Context, runID string) *Processor {
	p := &Processor{
		Ctx:   ctx,
		RunID: runID,
		// Create a channel to buffer requests
		ExecuteRequests:  make(chan *v2.ExecuteRequest, 100),
		ExecuteResponses: make(chan *v2.ExecuteResponse, 100),
		StopReading:      make(chan bool, 1),
	}

	return p
}

func (p *Processor) close() {
	// Close the requests channel to signal to the Runme that no more requests are expected
	close(p.ExecuteRequests)
	// Close the responses channel to signal to the Runme that no more responses are expected
	close(p.ExecuteResponses)
}

// Recv reads a v2.ExecuteRequest inside Runme.Runner for the Execute operation from the
// channel until it is closed.
func (p *Processor) Recv() (*v2.ExecuteRequest, error) {
	log := logs.FromContextWithTrace(p.Ctx)

	req, ok := <-p.ExecuteRequests
	if !ok {
		log.Info("Channel closed", "runID", p.RunID)
		// We return io.EOF to indicate the stream is closed by the client per the grpc Bidi spec.
		return nil, io.EOF
	}
	return req, nil
}

// Send sends a response message to the client.  The server handler may
// call Send multiple times to send multiple messages to the client.  An
// error is returned if the stream was terminated unexpectedly, and the
// handler method should return, as the stream is no longer usable.
func (p *Processor) Send(res *v2.ExecuteResponse) error {
	p.ActiveRequests = true
	p.ExecuteResponses <- res
	return nil
}

func (p *Processor) SetHeader(md metadata.MD) error {
	log := logs.FromContextWithTrace(p.Ctx)
	log.Info("Set called", "md", md, "runID", p.RunID)
	return nil
}

func (p *Processor) SendHeader(md metadata.MD) error {
	log := logs.FromContextWithTrace(p.Ctx)
	log.Info("SendHeader called", "md", md, "runID", p.RunID)
	return nil
}

func (p *Processor) SetTrailer(md metadata.MD) {
	log := logs.FromContextWithTrace(p.Ctx)
	log.Info("SetTrailer called", "md", md, "runID", p.RunID)
}

func (p *Processor) Context() context.Context {
	return p.Ctx
}

func (p *Processor) SendMsg(m any) error {
	err := errors.New("SendMsg is not implemented")
	log := logs.FromContextWithTrace(p.Ctx)
	log.Error(err, "SendMsg is not implemented")
	return err
}

func (p *Processor) RecvMsg(m any) error {
	err := errors.New("RecvMsg is not implemented")
	log := logs.FromContextWithTrace(p.Ctx)
	log.Error(err, "RecvMsg is not implemented")
	return err
}
