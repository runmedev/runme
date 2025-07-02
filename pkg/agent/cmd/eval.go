package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"buf.build/go/protovalidate"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent"
	"github.com/runmedev/runme/v3/pkg/agent/ai"
	"github.com/runmedev/runme/v3/pkg/agent/application"
)

func NewEvalCmd(appName string) *cobra.Command {
	var cookieFile string
	cmd := cobra.Command{
		Use:   "eval <yaml-file>",
		Short: "Run evaluation using a single experiment YAML file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cookieFile == "" {
				return fmt.Errorf("--cookie-file flag is required")
			}
			app := application.NewApp(appName)
			if err := app.LoadConfig(cmd); err != nil {
				return err
			}
			if err := app.SetupServerLogging(); err != nil {
				return err
			}

			// Read the experiment YAML file
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var m map[string]interface{}
			if err := yaml.Unmarshal(data, &m); err != nil {
				return err
			}
			jsonBytes, err := json.Marshal(m) // use json as intermediate format
			if err != nil {
				return err
			}
			var experiment agent.Experiment
			if err := protojson.Unmarshal(jsonBytes, &experiment); err != nil {
				return err
			}
			if err := protovalidate.Validate(&experiment); err != nil {
				return fmt.Errorf("failed to validate experiment file %q: %w", args[0], err)
			}
			// Read the cookie file (.env-style)
			cookieData, err := os.ReadFile(cookieFile)
			if err != nil {
				return fmt.Errorf("failed to read cookie file: %w", err)
			}
			cookies := make(map[string]string)
			lines := strings.Split(string(cookieData), "\n")
			client, err := ai.NewClient(*app.AppConfig.OpenAI)
			if err != nil {
				return fmt.Errorf("failed to read OpenAI API key file: %w", err)
			}
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					cookies[parts[0]] = parts[1]
				}
			}
			log := zapr.NewLogger(zap.L())
			_, err = ai.EvalFromExperiment(&experiment, args[0], cookies, client, log)
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&cookieFile, "cookie-file", "", "Path to the cookie file (required)")
	if err := cmd.MarkFlagRequired("cookie-file"); err != nil {
		panic(err)
	}
	return &cmd
}
