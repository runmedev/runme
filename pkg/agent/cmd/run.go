package cmd

import (
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/jlewi/monogo/helpers"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/version"
)

// NewRunCmd returns a command to run the server
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "run",
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				app := application.NewApp()
				if err := app.LoadConfig(cmd); err != nil {
					return err
				}
				if err := app.SetupLogging(); err != nil {
					return err
				}
				version.LogVersion()
				log := zapr.NewLogger(zap.L())
				log.Info("Starting application")
				defer helpers.DeferIgnoreError(app.Shutdown)
				return nil
			}()
			if err != nil {
				fmt.Printf("Error running request;\n %+v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}
