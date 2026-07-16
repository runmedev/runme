package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		// Avoid root's terminal-title PersistentPreRun so output matches --version.
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
		},
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", cmd.Root().Name(), cmd.Root().Version)
		},
	}

	setDefaultFlags(&cmd)

	return &cmd
}
