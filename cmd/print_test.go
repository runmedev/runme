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

func TestRenderTaskProgram(t *testing.T) {
	t.Run("Dagger", func(t *testing.T) {
		program := client.TaskProgram{
			Intent:      execution.NewIntent("dagger", []string{"--debug", "shell"}),
			Lines:       []string{"Build()", "{", "  .build", "}", "Build", ""},
			CommandMode: runner.CommandModeDaggerShell,
		}

		invocation := execution.NewInvocation("dagger", []string{"--debug", "script"})
		assert.Equal(t, "#!/usr/bin/env -S dagger --debug script\nBuild()\n{\n  .build\n}\nBuild\n", string(renderTaskProgram(program, invocation)))
	})

	t.Run("Other", func(t *testing.T) {
		program := client.TaskProgram{
			Intent:      execution.NewIntent("bash", nil),
			Lines:       []string{"echo hello"},
			CommandMode: runner.CommandModeInlineShell,
		}

		invocation := execution.NewInvocation("bash", nil)
		assert.Equal(t, "echo hello", string(renderTaskProgram(program, invocation)))
	})
}

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
