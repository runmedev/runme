package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func NewAgentCmd(appName string) *cobra.Command {
	var cfgFile string
	var level string
	var jsonLog bool

	agentCmd := &cobra.Command{
		Use:    "agent",
		Short:  appName,
		Hidden: true,
	}

	agentCmd.PersistentFlags().StringVar(&cfgFile, config.ConfigFlagName, "", fmt.Sprintf("config file (default is $HOME/.%s/config.yaml)", appName))
	agentCmd.PersistentFlags().StringVarP(&level, config.LevelFlagName, "", "info", "The logging level.")
	agentCmd.PersistentFlags().BoolVarP(&jsonLog, "json-logs", "", false, "Enable json logging.")

	agentCmd.AddCommand(NewVersionCmd(appName, os.Stdout))
	agentCmd.AddCommand(NewConfigCmd(appName))
	agentCmd.AddCommand(NewRunCmd(appName))
	agentCmd.AddCommand(NewServeCmd(appName))
	agentCmd.AddCommand(NewEnvCmd())
	agentCmd.AddCommand(NewEvalCmd(appName))

	serveCmd := NewServeCmd(appName)
	// Make serveCmd the default command.
	agentCmd.RunE = serveCmd.RunE

	serveCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if flag := agentCmd.Flags().Lookup(f.Name); flag == nil {
			agentCmd.Flags().AddFlag(f)
		}
	})

	return agentCmd
}
