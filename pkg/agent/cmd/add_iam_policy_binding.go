package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/runmedev/runme/v3/pkg/agent/config"
)

func NewAddIAMPolicyBindingCmd(appName string) *cobra.Command {
	var member string
	var role string

	cmd := &cobra.Command{
		Use:   "add-iam-policy-binding",
		Short: "Add an IAM policy binding to the agent config",
		Run: func(cmd *cobra.Command, args []string) {
			err := func() error {
				member = strings.TrimSpace(member)
				role = strings.TrimSpace(role)
				if member == "" {
					return errors.New("flag --member is required")
				}
				if role == "" {
					return errors.New("flag --role is required")
				}

				ac, err := config.NewAppConfig(appName, config.WithViperInstance(viper.GetViper(), cmd))
				if err != nil {
					return err
				}

				added, err := ac.AddIAMPolicyBinding(role, member)
				if err != nil {
					return err
				}

				if added {
					fmt.Fprintf(cmd.OutOrStdout(), "Added IAM policy binding for %q with role %q in %s\n", member, role, ac.GetConfigFile())
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "IAM policy binding for %q with role %q already exists in %s\n", member, role, ac.GetConfigFile())
				}

				return nil
			}()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to add IAM policy binding:\n %+v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&member, "member", "", "IAM member (for example user:jlewi@openai.com)")
	cmd.Flags().StringVar(&role, "role", "", "IAM role (for example role/runner.user)")

	return cmd
}
