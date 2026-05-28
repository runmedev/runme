package stream

import (
	v2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
)

// StreamTap receives lifecycle and data events from the multiplexer.
// Implementations must be safe for concurrent use.
//
// The multiplexer calls these methods at the corresponding points in the
// request/response flow. All methods are best-effort: errors are logged
// but do not interrupt the session.
type StreamTap interface {
	// RunStart is called when a new multiplexer run begins processing.
	RunStart(runID string)

	// RunEnd is called when the multiplexer run completes.
	RunEnd()

	// Output records terminal stdout data from an ExecuteResponse.
	Output(data []byte)

	// Stderr records terminal stderr data from an ExecuteResponse.
	// Implementations may also merge this into the output stream for
	// replay compatibility.
	Stderr(data []byte)

	// Input records terminal input data from a WebsocketRequest.
	Input(data []byte)

	// Resize records a terminal resize event.
	Resize(cols, rows uint32)

	// CommandStart is called when an ExecuteRequest with a Config is received,
	// signaling the start of a command execution.
	CommandStart(program, cwd string)

	// CommandEnd is called when command execution finishes.
	CommandEnd(exitCode int)

	// ClientConnect is called when a WebSocket client connects to the multiplexer.
	ClientConnect(streamID string)

	// ClientDisconnect is called when a WebSocket client disconnects.
	ClientDisconnect(streamID string)

	// Close finalizes the recording and releases resources.
	Close() error
}

// TapFactory creates a StreamTap for a given runID.
// If nil is returned, recording is disabled for that run.
type TapFactory func(runID string) StreamTap

// RequestPreprocessor transforms an ExecuteRequest before it reaches
// the runner. Only the initial request (Config != nil) is preprocessed;
// subsequent input-only requests pass through unchanged.
//
// Implementations may modify the request in place and return it, or
// return a new request. If an error is returned, the multiplexer logs
// the failure and falls back to the original request.
//
// Returning a nil request is invalid and is treated as a preprocessor
// failure; the original request is used.
type RequestPreprocessor func(req *v2.ExecuteRequest) (*v2.ExecuteRequest, error)

// noopTap is a StreamTap that discards all events.
type noopTap struct{}

func (noopTap) RunStart(string)             {}
func (noopTap) RunEnd()                     {}
func (noopTap) Output([]byte)               {}
func (noopTap) Stderr([]byte)               {}
func (noopTap) Input([]byte)                {}
func (noopTap) Resize(uint32, uint32)       {}
func (noopTap) CommandStart(string, string) {}
func (noopTap) CommandEnd(int)              {}
func (noopTap) ClientConnect(string)        {}
func (noopTap) ClientDisconnect(string)     {}
func (noopTap) Close() error                { return nil }
