package telemetry

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestInstrumentCommandTreePreservesRootPersistentPreRun(t *testing.T) {
	var reportedPath string
	var rootRan bool

	withReportCLIInvocation(t, func(cmd *cobra.Command) bool {
		reportedPath = cmd.CommandPath()
		return true
	})

	root := &cobra.Command{
		Use: "runme",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			rootRan = true
		},
	}
	child := &cobra.Command{
		Use: "run",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(child)
	InstrumentCommandTree(root)
	root.SetArgs([]string{"run"})

	require.NoError(t, root.Execute())
	require.True(t, rootRan)
	require.Equal(t, "runme run", reportedPath)
}

func TestInstrumentCommandTreePreservesPersistentPreRunEError(t *testing.T) {
	wantErr := errors.New("boom")
	var reported bool

	withReportCLIInvocation(t, func(cmd *cobra.Command) bool {
		reported = true
		return true
	})

	root := &cobra.Command{Use: "runme"}
	beta := &cobra.Command{
		Use: "beta",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return wantErr
		},
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(beta)
	InstrumentCommandTree(root)
	root.SetArgs([]string{"beta"})

	err := root.Execute()
	require.ErrorIs(t, err, wantErr)
	require.True(t, reported)
}

func TestInstrumentCommandTreeCoversNestedPersistentPreRunE(t *testing.T) {
	var reportedPath string
	var serverRan bool

	withReportCLIInvocation(t, func(cmd *cobra.Command) bool {
		reportedPath = cmd.CommandPath()
		return true
	})

	root := &cobra.Command{Use: "runme"}
	beta := &cobra.Command{Use: "beta"}
	server := &cobra.Command{
		Use: "server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			serverRan = true
			return nil
		},
	}
	list := &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	server.AddCommand(list)
	beta.AddCommand(server)
	root.AddCommand(beta)
	InstrumentCommandTree(root)
	root.SetArgs([]string{"beta", "server", "list"})

	require.NoError(t, root.Execute())
	require.True(t, serverRan)
	require.Equal(t, "runme beta server list", reportedPath)
}

func TestInstrumentCommandTreeReportsOnce(t *testing.T) {
	var reportCount int

	withReportCLIInvocation(t, func(cmd *cobra.Command) bool {
		reportCount++
		return true
	})

	root := &cobra.Command{
		Use:              "runme",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	child := &cobra.Command{
		Use:              "run",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
		Run:              func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(child)
	InstrumentCommandTree(root)
	root.SetArgs([]string{"run"})

	require.NoError(t, root.Execute())
	require.Equal(t, 1, reportCount)
}

func TestInstrumentCommandTreeSkipsKernelServer(t *testing.T) {
	var reportCount int

	withReportCLIInvocation(t, func(cmd *cobra.Command) bool {
		if newCLIInvocation(cmd).shouldReport() {
			reportCount++
		}
		return true
	})

	root := &cobra.Command{Use: "runme"}
	server := &cobra.Command{
		Use: "server",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(server)
	InstrumentCommandTree(root)
	root.SetArgs([]string{"server"})

	require.NoError(t, root.Execute())
	require.Equal(t, 0, reportCount)
}

func withReportCLIInvocation(t *testing.T, fn func(*cobra.Command) bool) {
	t.Helper()

	previous := reportCLIInvocation
	reportCLIInvocation = fn
	t.Cleanup(func() {
		reportCLIInvocation = previous
	})
}
