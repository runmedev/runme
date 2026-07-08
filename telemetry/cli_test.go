package telemetry

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCLIInvocation(t *testing.T) {
	cmd := &cobra.Command{Use: "run"}
	root := &cobra.Command{Use: "runme"}
	root.AddCommand(cmd)

	invocation := newCLIInvocation(cmd)
	require.Equal(t, "run", invocation.command)
	require.Equal(t, "run", invocation.commandPath)
	require.NotEmpty(t, invocation.version)
	require.NotEmpty(t, invocation.goos)
	require.NotEmpty(t, invocation.arch)
	require.NotEmpty(t, invocation.installChannel)

	event := invocation.event()
	require.Equal(t, clientCLI, event.client)
	require.Equal(t, cliInvocationEvent, event.name)
	require.Equal(t, "cli", event.props["component"])
	require.Equal(t, "run", event.props["command"])
	require.Equal(t, "run", event.props["command_path"])
	require.NotEmpty(t, event.props["version"])
	require.NotEmpty(t, event.props["os"])
	require.NotEmpty(t, event.props["arch"])
	require.NotEmpty(t, event.props["install_channel"])
}

func TestCLIInvocationShouldReport(t *testing.T) {
	t.Run("RegularCommand", func(t *testing.T) {
		cmd := &cobra.Command{Use: "run"}
		root := &cobra.Command{Use: "runme"}
		root.AddCommand(cmd)

		require.True(t, newCLIInvocation(cmd).shouldReport())
	})

	t.Run("KernelServer", func(t *testing.T) {
		cmd := &cobra.Command{Use: "server"}
		root := &cobra.Command{Use: "runme"}
		root.AddCommand(cmd)

		require.False(t, newCLIInvocation(cmd).shouldReport())
	})

	t.Run("BetaKernelServerStart", func(t *testing.T) {
		start := &cobra.Command{Use: "start"}
		server := &cobra.Command{Use: "server"}
		beta := &cobra.Command{Use: "beta"}
		root := &cobra.Command{Use: "runme"}
		server.AddCommand(start)
		beta.AddCommand(server)
		root.AddCommand(beta)

		require.False(t, newCLIInvocation(start).shouldReport())
	})

	t.Run("HarborStdio", func(t *testing.T) {
		stdio := &cobra.Command{Use: "stdio"}
		harbor := &cobra.Command{Use: "harbor"}
		root := &cobra.Command{Use: "runme"}
		harbor.AddCommand(stdio)
		root.AddCommand(harbor)

		require.False(t, newCLIInvocation(stdio).shouldReport())
	})

	t.Run("NilCommand", func(t *testing.T) {
		require.False(t, newCLIInvocation(nil).shouldReport())
	})
}

func TestInstallChannelFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "HomebrewAppleSilicon",
			path: "/opt/homebrew/Cellar/runme/3.9.0/bin/runme",
			want: installChannelHomebrew,
		},
		{
			name: "HomebrewIntel",
			path: "/usr/local/Cellar/runme/3.9.0/bin/runme",
			want: installChannelHomebrew,
		},
		{
			name: "NPMGlobal",
			path: "/usr/local/lib/node_modules/runme/.bin/runme",
			want: installChannelNPM,
		},
		{
			name: "NPMNpxCache",
			path: "/Users/person/.npm/_npx/abc123/node_modules/runme/.bin/runme",
			want: installChannelNPM,
		},
		{
			name: "GoInstall",
			path: "/Users/person/go/bin/runme",
			want: installChannelGoInstall,
		},
		{
			name: "LinuxPackage",
			path: "/usr/bin/runme",
			want: installChannelLinuxPackage,
		},
		{
			name: "Unknown",
			path: "/tmp/random/runme",
			want: installChannelUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, installChannelFromPath(tt.path))
		})
	}
}
