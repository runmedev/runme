package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/server"
	"github.com/runmedev/runme/v3/pkg/agent/tlsbuilder"
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

			if err := app.SetupOTEL(); err != nil {
				return err
			}
			agentOptions := &ai.AgentOptions{}

			if err := agentOptions.FromAssistantConfig(*app.AppConfig.CloudAssistant); err != nil {
				return err
			}

			client, err := ai.NewClient(*app.AppConfig.OpenAI)
			if err != nil {
				return err
			}

			agentOptions.Client = client

			agent, err := ai.NewAgent(*agentOptions)
			if err != nil {
				return err
			}

			// Setup the defaults for the TLSConfig
			if app.AppConfig.AssistantServer.TLSConfig != nil && app.AppConfig.AssistantServer.TLSConfig.Generate {
				// Set the default values for the TLSConfig
				if app.AppConfig.AssistantServer.TLSConfig.KeyFile == "" {
					app.AppConfig.AssistantServer.TLSConfig.KeyFile = filepath.Join(app.AppConfig.GetConfigDir(), tlsbuilder.KeyPEMFile)
				}

				if app.AppConfig.AssistantServer.TLSConfig.CertFile == "" {
					app.AppConfig.AssistantServer.TLSConfig.CertFile = filepath.Join(app.AppConfig.GetConfigDir(), tlsbuilder.CertPEMFile)
				}
			}

			serverOptions := &server.Options{
				Telemetry: app.AppConfig.Telemetry,
				Server:    app.AppConfig.AssistantServer,
				IAMPolicy: app.AppConfig.IAMPolicy,
				WebApp:    app.AppConfig.WebApp,
			}
			s, err := server.NewServer(*serverOptions, agent)
			if err != nil {
				return err
			}

			return s.Run()
		},
	}

	return &cmd
}
