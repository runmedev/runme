package cmd

import (
	"fmt"
	"io"

	"github.com/runmedev/runme/v3/pkg/agent/version"

	"github.com/spf13/cobra"
)

func NewVersionCmd(appName string, w io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Return version",
		Example: fmt.Sprintf("%s  version", appName),
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(w, "%s %s, commit %s, built at %s by %s\n", appName, version.Version, version.Commit, version.Date, version.BuiltBy)
		},
	}
	return cmd
}
