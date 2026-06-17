package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/harbor"
)

func harborCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "harbor",
		Hidden: true,
	}
	cmd.AddCommand(harborStdioCmd())
	return cmd
}

func harborStdioCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "stdio",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := harbor.NewServer(harbor.Options{})
			if err != nil {
				return err
			}
			return server.ServeStdio(cmd.Context(), os.Stdin, os.Stdout)
		},
	}
}
