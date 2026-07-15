package cmd

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	owlcmd "github.com/runmedev/owl/cmd"

	"github.com/runmedev/runme/v3/command"
	"github.com/runmedev/runme/v3/runner/client"
)

var newOSEnvironReader = func() (io.Reader, error) {
	return command.NewEnvProducerFromEnv()
}

func environmentCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "env",
		Aliases: []string{"environment"},
		Hidden:  true,
		Short:   "Environment management",
		Long:    "Various commands to manage environments in runme",
	}

	cmd.AddCommand(environmentDumpCmd())
	cmd.AddCommand(storeCmd())

	setDefaultFlags(&cmd)

	return &cmd
}

type envStoreFlags struct {
	serverAddr      string
	sessionID       string
	sessionStrategy string
	tlsDir          string
}

func storeCmd() *cobra.Command {
	var (
		storeFlags    envStoreFlags
		checkAddr     string
		getRunnerOpts func() ([]client.RunnerOption, error)
	)

	cmd := owlcmd.NewStoreCommand(owlcmd.StoreCommandOptions{
		Hidden: true,
		InsecureAllowed: func() bool {
			return fInsecure
		},
		ClientFactory: func(cmd *cobra.Command) (owlcmd.StoreClient, error) {
			return &runmeOwlStoreClient{
				storeFlags:    storeFlags,
				checkAddr:     checkAddr,
				getRunnerOpts: getRunnerOpts,
				stdin:         cmd.InOrStdin(),
				stdout:        cmd.OutOrStdout(),
				stderr:        cmd.ErrOrStderr(),
			}, nil
		},
		ConfigureCheckCommand: func(cmd *cobra.Command) {
			getRunnerOpts = setRunnerFlags(cmd, &checkAddr)
		},
	})
	cmd.Flags().StringVar(&storeFlags.serverAddr, "server-address", os.Getenv("RUNME_SERVER_ADDR"), "The Server ServerAddress to connect to, i.e. 127.0.0.1:7865")
	cmd.Flags().StringVar(&storeFlags.tlsDir, "tls-dir", os.Getenv("RUNME_TLS_DIR"), "Path to tls files")
	cmd.Flags().StringVar(&storeFlags.sessionID, "session", os.Getenv("RUNME_SESSION"), "Session Id")
	cmd.Flags().StringVar(&storeFlags.sessionStrategy, "session-strategy", func() string {
		if val, ok := os.LookupEnv("RUNME_SESSION_STRATEGY"); ok {
			return val
		}
		return "manual"
	}(), "Strategy for session selection. Options are manual, recent. Defaults to manual")

	return cmd
}

func environmentDumpCmd() *cobra.Command {
	cmd := cobra.Command{
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

	setDefaultFlags(&cmd)

	return &cmd
}
