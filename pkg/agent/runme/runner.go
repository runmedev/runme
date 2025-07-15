package runme

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2/runnerv2connect"
	"github.com/runmedev/runme/v3/command"
	"github.com/runmedev/runme/v3/runnerv2service"
)

// Runner lets you run commands using Runme.
type Runner struct {
	Server runnerv2.RunnerServiceServer
}

func NewRunner(logger *zap.Logger) (*Runner, error) {
	factory := command.NewFactory(command.WithLogger(logger))
	server, err := runnerv2service.NewRunnerService(factory, logger)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Runme runner service")
	}
	return &Runner{
		Server: server,
	}, nil
}

var _ runnerv2connect.RunnerServiceHandler = (*Runner)(nil)

func (r *Runner) CreateSession(ctx context.Context, req *connect.Request[runnerv2.CreateSessionRequest]) (*connect.Response[runnerv2.CreateSessionResponse], error) {
	resp, err := r.Server.CreateSession(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (r *Runner) GetSession(ctx context.Context, req *connect.Request[runnerv2.GetSessionRequest]) (*connect.Response[runnerv2.GetSessionResponse], error) {
	resp, err := r.Server.GetSession(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (r *Runner) ListSessions(ctx context.Context, req *connect.Request[runnerv2.ListSessionsRequest]) (*connect.Response[runnerv2.ListSessionsResponse], error) {
	resp, err := r.Server.ListSessions(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (r *Runner) UpdateSession(ctx context.Context, req *connect.Request[runnerv2.UpdateSessionRequest]) (*connect.Response[runnerv2.UpdateSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (r *Runner) DeleteSession(ctx context.Context, req *connect.Request[runnerv2.DeleteSessionRequest]) (*connect.Response[runnerv2.DeleteSessionResponse], error) {
	resp, err := r.Server.DeleteSession(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (r *Runner) MonitorEnvStore(ctx context.Context, req *connect.Request[runnerv2.MonitorEnvStoreRequest], stream *connect.ServerStream[runnerv2.MonitorEnvStoreResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func (r *Runner) Execute(ctx context.Context, req *connect.BidiStream[runnerv2.ExecuteRequest, runnerv2.ExecuteResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("use websocket instead to support duplex stream"))
}

func (r *Runner) ResolveProgram(ctx context.Context, req *connect.Request[runnerv2.ResolveProgramRequest]) (*connect.Response[runnerv2.ResolveProgramResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}
