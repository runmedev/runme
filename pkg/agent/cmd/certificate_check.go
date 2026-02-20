package cmd

import (
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/tlsbuilder"
)

func ensureTLSCertificate(app *application.App) error {
	if app.AppConfig == nil || app.AppConfig.AssistantServer == nil {
		return nil
	}

	tlsConfig := app.AppConfig.AssistantServer.TLSConfig
	if tlsConfig == nil || !tlsConfig.Generate {
		return nil
	}

	_, err := tlsbuilder.LoadOrGenerateConfig(tlsConfig.CertFile, tlsConfig.KeyFile, zap.L())
	return err
}

func NewCertificateCheckCmd(appName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certificate-check",
		Short: "Check or create generated TLS certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := application.NewApp(appName)
			if err := app.LoadConfig(cmd); err != nil {
				return err
			}
			if err := app.SetupLogging(); err != nil {
				return err
			}

			log := zapr.NewLogger(zap.L())
			if app.AppConfig == nil || app.AppConfig.AssistantServer == nil || app.AppConfig.AssistantServer.TLSConfig == nil || !app.AppConfig.AssistantServer.TLSConfig.Generate {
				log.Info("TLS certificate generation is disabled; nothing to do")
				return nil
			}

			if err := ensureTLSCertificate(app); err != nil {
				return err
			}

			log.Info("TLS certificate check complete")
			return nil
		},
	}

	return cmd
}
