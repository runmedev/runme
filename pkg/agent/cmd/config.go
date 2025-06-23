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
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "config",
	}

	cmd.AddCommand(NewGetConfigCmd())
	cmd.AddCommand(NewSetConfigCmd())
	return cmd
}

// NewSetConfigCmd sets a key value pair in the configuration
func NewSetConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "set <name>=<value>",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				v := viper.GetViper()

				if err := config.InitViperInstance(v, cmd); err != nil {
					return err
				}

				fConfig, err := config.UpdateViperConfig(v, args[0])
				if err != nil {
					return errors.Wrap(err, "Failed to update configuration")
				}

				file := fConfig.GetConfigFile()
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
func NewGetConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Dump Foyle configuration as YAML",
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				if err := config.InitViper(cmd); err != nil {
					return err
				}
				fConfig := config.GetConfig()

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
