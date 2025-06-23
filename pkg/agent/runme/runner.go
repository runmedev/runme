package runme

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
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
