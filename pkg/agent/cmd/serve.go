package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1/agentv1connect"
	"github.com/runmedev/runme/v3/pkg/agent/ai"
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

			if err := app.SetupOTEL(); err != nil {
				return err
			}
			if app.AppConfig.AssistantServer == nil {
				app.AppConfig.AssistantServer = &config.AssistantServerConfig{}
			}

			var agent agentv1connect.MessagesServiceHandler
			if app.AppConfig.AssistantServer.GetAgentService() {
				agentOptions := &ai.AgentOptions{}
				if app.AppConfig.CloudAssistant == nil {
					return errors.New("cloudAssistant config is required when assistantServer.agentService is enabled")
				}
				if err := agentOptions.FromAssistantConfig(*app.AppConfig.CloudAssistant); err != nil {
					return err
				}

				if app.AppConfig.OpenAI == nil {
					// OpenAI access tokens will be provided by the client per request.
					agentOptions.Client = ai.NewClientWithoutKey()
				} else {
					client, err := ai.NewClient(*app.AppConfig.OpenAI)
					if err != nil {
						return err
					}
					agentOptions.Client = client
					agentOptions.OAuthOpenAIOrganization = app.AppConfig.OpenAI.Organization
					agentOptions.OAuthOpenAIProject = app.AppConfig.OpenAI.Project
				}

				var err error
				agent, err = ai.NewAgent(*agentOptions)
				if err != nil {
					return err
				}
			}

			if err := ensureTLSCertificate(app); err != nil {
				return err
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
