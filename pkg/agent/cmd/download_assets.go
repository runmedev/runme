package cmd

import (
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/pkg/agent/application"
	"github.com/runmedev/runme/v3/pkg/agent/assets"
)

// NewDownloadAssetsCmd downloads and unpacks the web app assets from an OCI image.
func NewDownloadAssetsCmd(appName string) *cobra.Command {
	var imageRef string

	cmd := &cobra.Command{
		Use:   "download-assets",
		Short: "Download and unpack web app assets",
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				if imageRef == "" {
					return errors.New("image reference is required; set --image")
				}

				app := application.NewApp(appName)
				if err := app.LoadConfig(cmd); err != nil {
					return err
				}
				if err := app.SetupLogging(); err != nil {
					return err
				}

				cfg := app.AppConfig.GetConfig()
				if cfg.AssistantServer == nil || cfg.AssistantServer.StaticAssets == "" {
					return errors.New("assistantServer.staticAssets must be set in config to download assets")
				}

				log := zapr.NewLogger(zap.L())
				log.Info("Downloading assets image", "image", imageRef, "dir", cfg.AssistantServer.StaticAssets)

				if err := assets.DownloadFromImage(cmd.Context(), imageRef, cfg.AssistantServer.StaticAssets); err != nil {
					return err
				}

				log.Info("Assets download complete", "dir", cfg.AssistantServer.StaticAssets)
				return nil
			}()
			if err != nil {
				fmt.Printf("Failed to download assets;\n %+v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&imageRef, "image", "ghcr.io/runmedev/app-assets:latest", "OCI image reference to download (e.g. ghcr.io/runmedev/app-assets:latest)")

	return cmd
}
