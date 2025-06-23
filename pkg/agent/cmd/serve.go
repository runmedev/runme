package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/server"
	"github.com/runmedev/runme/v3/pkg/agent/tlsbuilder"
)

func NewServeCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "serve",
		Short: "Start the Assistant and Runme server",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := application.NewApp()

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

			if err := agentOptions.FromAssistantConfig(*app.Config.CloudAssistant); err != nil {
				return err
			}

			client, err := ai.NewClient(*app.Config.OpenAI)
			if err != nil {
				return err
			}

			agentOptions.Client = client

			agent, err := ai.NewAgent(*agentOptions)
			if err != nil {
				return err
			}

			// Setup the defaults for the TLSConfig
			if app.Config.AssistantServer.TLSConfig != nil && app.Config.AssistantServer.TLSConfig.Generate {
				// Set the default values for the TLSConfig
				if app.Config.AssistantServer.TLSConfig.KeyFile == "" {
					app.Config.AssistantServer.TLSConfig.KeyFile = filepath.Join(app.Config.GetConfigDir(), tlsbuilder.KeyPEMFile)
				}

				if app.Config.AssistantServer.TLSConfig.CertFile == "" {
					app.Config.AssistantServer.TLSConfig.CertFile = filepath.Join(app.Config.GetConfigDir(), tlsbuilder.CertPEMFile)
				}
			}

			serverOptions := &server.Options{
				Telemetry: app.Config.Telemetry,
				Server:    app.Config.AssistantServer,
				IAMPolicy: app.Config.IAMPolicy,
				WebApp:    app.Config.WebApp,
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
