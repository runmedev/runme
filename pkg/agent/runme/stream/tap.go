package stream

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
	// exitCode is the final process exit code (or -1 if unknown).
	RunEnd(exitCode int)

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

// noopTap is a StreamTap that discards all events.
type noopTap struct{}

func (noopTap) RunStart(string)             {}
func (noopTap) RunEnd(int)                  {}
func (noopTap) Output([]byte)               {}
func (noopTap) Stderr([]byte)               {}
func (noopTap) Input([]byte)                {}
func (noopTap) Resize(uint32, uint32)       {}
func (noopTap) CommandStart(string, string) {}
func (noopTap) CommandEnd(int)              {}
func (noopTap) ClientConnect(string)        {}
func (noopTap) ClientDisconnect(string)     {}
func (noopTap) Close() error                { return nil }
