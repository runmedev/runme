package runnerservice

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/command"
	"github.com/runmedev/runme/v3/runnerv2service"
)

func New(t *testing.T) (_ *bufconn.Listener, stop func()) {
	t.Helper()

	logger := zap.NewNop()
	factory := command.NewFactory(command.WithLogger(logger))

	runnerService, err := runnerv2service.NewRunnerService(factory, logger)
	require.NoError(t, err)

	server := grpc.NewServer()
	runnerv2.RegisterRunnerServiceServer(server, runnerService)

	lis := bufconn.Listen(1 << 20) // 1 MB
	go server.Serve(lis)

	stop = func() {
		_ = logger.Sync()
		server.Stop()
	}

	return lis, stop
}
