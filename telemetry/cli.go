package telemetry

import (
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/internal/version"
)

const cliInvocationEvent = "cli_invocation"

type cliInvocation struct {
	command        string
	commandPath    string
	version        string
	goos           string
	arch           string
	installChannel string
}

func reportCLIInvocationDefault(cmd *cobra.Command) bool {
	invocation := newCLIInvocation(cmd)
	if !invocation.shouldReport() {
		return false
	}

	return newReporter(zap.NewNop()).report(invocation.event())
}

func newCLIInvocation(cmd *cobra.Command) cliInvocation {
	if cmd == nil {
		return cliInvocation{}
	}

	path := cmd.CommandPath()

	return cliInvocation{
		command:        topLevelCommand(path),
		commandPath:    coarseCommandPath(path),
		version:        version.BuildVersion,
		goos:           runtime.GOOS,
		arch:           runtime.GOARCH,
		installChannel: installChannel(),
	}
}

func (i cliInvocation) shouldReport() bool {
	switch i.commandPath {
	case "", "server", "beta server start", "harbor stdio":
		return false
	default:
		return true
	}
}

func (i cliInvocation) event() event {
	props := map[string]string{
		"component":       "cli",
		"version":         i.version,
		"os":              i.goos,
		"arch":            i.arch,
		"command":         i.command,
		"command_path":    i.commandPath,
		"install_channel": i.installChannel,
	}

	return event{
		client:  clientCLI,
		name:    cliInvocationEvent,
		props:   props,
		timeout: 2 * time.Second,
	}
}

func topLevelCommand(commandPath string) string {
	parts := strings.Fields(commandPath)
	if len(parts) < 2 {
		return "tui"
	}

	return parts[1]
}

func coarseCommandPath(commandPath string) string {
	parts := strings.Fields(commandPath)
	if len(parts) <= 1 {
		return "runme"
	}

	return strings.Join(parts[1:], " ")
}
