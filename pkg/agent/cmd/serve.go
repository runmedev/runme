package cmd

import (
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/config"
	"github.com/runmedev/runme/v3/pkg/agent/server"
)

func NewServeCmd(appName string) *cobra.Command {
	cmd := cobra.Command{
		Use:   "serve",
		Short: "Start the Assistant and Runme server",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := application.NewApp(appName)

			// Load the configuration
			if err := app.LoadConfig(cmd); err != nil {
				return err
			}

			if err := app.SetupServerLogging(); err != nil {
				return err
			}

			if keys := app.AppConfig.GetConfig().DeprecatedConfigKeys(); len(keys) > 0 {
				log := zapr.NewLogger(zap.L())
				log.Info("Legacy configuration is deprecated and ignored", "keys", keys)
			}

			if err := app.SetupOTEL(); err != nil {
				return err
			}
			if app.AppConfig.AssistantServer == nil {
				app.AppConfig.AssistantServer = &config.AssistantServerConfig{}
			}

			if err := ensureTLSCertificate(app); err != nil {
				return err
			}

			serverOptions := &server.Options{
				Telemetry: app.AppConfig.Telemetry,
				Server:    app.AppConfig.AssistantServer,
				ConfigDir: app.AppConfig.GetConfigDir(),
				IAMPolicy: app.AppConfig.IAMPolicy,
			}
			s, err := server.NewServer(*serverOptions)
			if err != nil {
				return err
			}

			return s.Run()
		},
	}

	return &cmd
}
