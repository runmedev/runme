package cmd

import (
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/command"
)

var newOSEnvironReader = func() (io.Reader, error) {
	return command.NewEnvProducerFromEnv()
}

var fInsecure bool

func NewEnvCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "env",
		Aliases: []string{"environment"},
		Hidden:  true,
		Short:   "Environment management",
		Long:    "Various commands to manage environments in runme",
	}

	dumpCmd := cobra.Command{
		Use:   "dump",
		Short: "Dump environment variables to stdout",
		Long:  "Dumps all environment variables to stdout as a list of K=V separated by null terminators",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !fInsecure {
				return errors.New("must be run in insecure mode to prevent misuse; enable by adding --insecure flag")
			}

			producer, err := newOSEnvironReader()
			if err != nil {
				return err
			}

			_, _ = io.Copy(cmd.OutOrStdout(), producer)

			return nil
		},
	}

	dumpCmd.Flags().BoolVar(&fInsecure, "insecure", false, "Explicitly allow insecure operations to prevent misuse")

	cmd.AddCommand(&dumpCmd)

	return &cmd
}
