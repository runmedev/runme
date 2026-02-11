package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
	var assetsDirFlag string

	cmd := &cobra.Command{
		Use:   "download-assets",
		Short: "Download and unpack web app assets",
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				if imageRef == "" {
					return errors.New("image reference is required; set --image")
				}

				var assetsDir string
				if assetsDirFlag != "" {
					assetsDir = assetsDirFlag
				} else {
					app := application.NewApp(appName)
					if err := app.LoadConfig(cmd); err != nil {
						return err
					}
					if err := app.SetupLogging(); err != nil {
						return err
					}

					cfg := app.AppConfig.GetConfig()
					if cfg.AssistantServer == nil {
						return errors.New("assistantServer config must be set to download assets")
					}

					assetsDir = cfg.AssistantServer.StaticAssets
					if assetsDir == "" {
						homeDir, err := os.UserHomeDir()
						if err != nil {
							return errors.Wrap(err, "failed to resolve home directory for assets")
						}
						assetsDir = filepath.Join(homeDir, "."+appName, "assets")
					} else if !filepath.IsAbs(assetsDir) {
						homeDir, err := os.UserHomeDir()
						if err != nil {
							return errors.Wrap(err, "failed to resolve home directory for assets")
						}
						assetsDir = filepath.Join(homeDir, assetsDir)
					}
				}

				log := zapr.NewLogger(zap.L())
				log.Info("Downloading assets image", "image", imageRef, "dir", assetsDir)

				if err := assets.DownloadFromImage(cmd.Context(), imageRef, assetsDir); err != nil {
					return err
				}

				log.Info("Assets download complete", "dir", assetsDir)
				return nil
			}()
			if err != nil {
				fmt.Printf("Failed to download assets;\n %+v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&imageRef, "image", "ghcr.io/runmedev/app-assets:latest", "OCI image reference to download (e.g. ghcr.io/runmedev/app-assets:latest)")
	cmd.Flags().StringVar(&assetsDirFlag, "assets-dir", "", "Directory to download and unpack assets (skips config)")

	return cmd
}
