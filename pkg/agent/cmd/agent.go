package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func NewAgentCmd() *cobra.Command {
	var cfgFile string
	var level string
	var jsonLog bool
	agentCmd := &cobra.Command{
		Use:    "agent",
		Short:  config.AppName,
		Hidden: true,
	}

	agentCmd.PersistentFlags().StringVar(&cfgFile, config.ConfigFlagName, "", fmt.Sprintf("config file (default is $HOME/.%s/config.yaml)", config.AppName))
	agentCmd.PersistentFlags().StringVarP(&level, config.LevelFlagName, "", "info", "The logging level.")
	agentCmd.PersistentFlags().BoolVarP(&jsonLog, "json-logs", "", false, "Enable json logging.")

	agentCmd.AddCommand(NewVersionCmd(os.Stdout))
	agentCmd.AddCommand(NewConfigCmd())
	agentCmd.AddCommand(NewRunCmd())
	agentCmd.AddCommand(NewServeCmd())
	agentCmd.AddCommand(NewEnvCmd())
	agentCmd.AddCommand(NewEvalCmd())

	serveCmd := NewServeCmd()
	// Make serveCmd the default command.
	agentCmd.RunE = serveCmd.RunE

	serveCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if flag := agentCmd.Flags().Lookup(f.Name); flag == nil {
			agentCmd.Flags().AddFlag(f)
		}
	})

	return agentCmd
}
