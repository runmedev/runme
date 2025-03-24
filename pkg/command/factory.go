package command

import (
	"go.uber.org/zap"

	"github.com/stateful/runme/v3/internal/command"
)

type (
	FactoryOption command.FactoryOption
	Factory       command.Factory
)

func WithDebug() FactoryOption {
	return FactoryOption(command.WithDebug())
}

func WithLogger(logger *zap.Logger) FactoryOption {
	return FactoryOption(command.WithLogger(logger))
}

func NewFactory(opts ...FactoryOption) Factory {
	newOpts := make([]command.FactoryOption, 0, len(opts))
	for _, opt := range opts {
		newOpts = append(newOpts, command.FactoryOption(opt))
	}
	return command.NewFactory(newOpts...)
}
