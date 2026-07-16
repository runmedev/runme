package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommandMatchesVersionFlag(t *testing.T) {
	flagStdout, flagStderr := executeRootCommand(t, "--version")
	commandStdout, commandStderr := executeRootCommand(t, "version")

	require.Empty(t, flagStderr)
	require.Empty(t, commandStderr)
	require.Equal(t, flagStdout, commandStdout)
}

func executeRootCommand(t *testing.T, args ...string) (string, string) {
	t.Helper()

	cmd := Root()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	require.NoError(t, cmd.Execute())

	return stdout.String(), stderr.String()
}
