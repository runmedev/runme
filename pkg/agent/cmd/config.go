package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

// NewConfigCmd adds commands to deal with configuration
func NewConfigCmd(appName string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "config",
	}

	cmd.AddCommand(NewGetConfigCmd(appName))
	cmd.AddCommand(NewSetConfigCmd(appName))
	return cmd
}

// NewSetConfigCmd sets a key value pair in the configuration
func NewSetConfigCmd(appName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "set <name>=<value>",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				v := viper.GetViper()

				ac, err := config.NewAppConfig(appName, config.WithViperInstance(v, cmd))
				if err != nil {
					return err
				}

				fConfig, err := ac.UpdateViperConfig(args[0])
				if err != nil {
					return errors.Wrap(err, "Failed to update configuration")
				}

				file := ac.GetConfigFile()
				if file == "" {
					return errors.New("Failed to get configuration file")
				}
				// Persist the configuration
				return fConfig.Write(file)
			}()
			if err != nil {
				fmt.Printf("Failed to set configuration;\n %+v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}

// NewGetConfigCmd  prints out the configuration
func NewGetConfigCmd(appName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Dump agent configuration as YAML",
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				ac, err := config.NewAppConfig(appName, config.WithViper(cmd))
				if err != nil {
					return err
				}
				fConfig := ac.GetConfig()

				if err := yaml.NewEncoder(os.Stdout).Encode(fConfig); err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				fmt.Printf("Failed to get configuration;\n %+v\n", err)
				os.Exit(1)
			}
		},
	}

	return cmd
}
