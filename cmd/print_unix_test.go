//go:build !windows

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/internal/execution"
	"github.com/runmedev/runme/v3/runner"
	"github.com/runmedev/runme/v3/runner/client"
)

func TestResolvePrintInvocation(t *testing.T) {
	dir := t.TempDir()
	daggerPath := filepath.Join(dir, "dagger")
	require.NoError(t, os.WriteFile(daggerPath, []byte("#!/bin/sh\necho v1.0.0\n"), 0o700))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	program := client.TaskProgram{
		Intent:      execution.NewIntent("dagger", []string{"--debug", "shell"}),
		CommandMode: runner.CommandModeDaggerShell,
	}
	invocation, err := resolvePrintInvocation(context.Background(), program, os.Environ())
	require.NoError(t, err)
	assert.Equal(t, "dagger", invocation.Executable())
	assert.Equal(t, []string{"--debug", "script"}, invocation.Arguments())
}
