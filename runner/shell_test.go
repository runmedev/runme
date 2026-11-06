package runner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestParseFileProgramExistingPathWithSpaces(t *testing.T) {
	program := filepath.Join(t.TempDir(), "Program Files", "bash")
	require.NoError(t, os.MkdirAll(filepath.Dir(program), 0o700))
	require.NoError(t, os.WriteFile(program, nil, 0o700))

	gotProgram, gotArgs := parseFileProgram(program)
	assert.Equal(t, program, gotProgram)
	assert.Empty(t, gotArgs)
}

func TestShellKnownNameEnv(t *testing.T) {
	t.Setenv("RUNME_PARENT_ENV", "inherited")

	sess, err := NewSession([]string{}, nil, zaptest.NewLogger(t))
	require.NoError(t, err)

	stdout := new(bytes.Buffer)
	shell := &Shell{
		ExecutableConfig: &ExecutableConfig{
			KnownName: "my-cell",
			Stdout:    stdout,
			Stderr:    new(bytes.Buffer),
			Session:   sess,
			Logger:    zaptest.NewLogger(t),
		},
		Cmds:        []string{`printf '%s|%s' "$RUNME_NAME" "$RUNME_PARENT_ENV"`},
		CustomShell: "bash",
	}

	require.NoError(t, shell.Run(context.Background()))
	assert.Equal(t, "my-cell|inherited", stdout.String())
}

func TestShellKnownNameError(t *testing.T) {
	shell := &Shell{
		ExecutableConfig: &ExecutableConfig{
			KnownName: "my-cell",
			Stdout:    new(bytes.Buffer),
			Stderr:    new(bytes.Buffer),
			Logger:    zaptest.NewLogger(t),
		},
		Cmds:        []string{"exit 1"},
		CustomShell: "bash",
	}

	err := shell.Run(context.Background())
	require.ErrorContains(t, err, `failed to run command "my-cell"`)
}

func TestTempFileKnownNameEnv(t *testing.T) {
	t.Parallel()

	sess, err := NewSession(os.Environ(), nil, zaptest.NewLogger(t))
	require.NoError(t, err)

	stdout := new(bytes.Buffer)
	tempFile := &TempFile{
		ExecutableConfig: &ExecutableConfig{
			KnownName: "my-cell",
			Stdout:    stdout,
			Stderr:    new(bytes.Buffer),
			Session:   sess,
			Logger:    zaptest.NewLogger(t),
		},
		Script:      "echo -n $RUNME_NAME",
		ProgramName: "bash",
		LanguageID:  "sh",
	}

	require.NoError(t, tempFile.Run(context.Background()))
	assert.Equal(t, "my-cell", stdout.String())
}

func TestGoKnownNameEnv(t *testing.T) {
	t.Parallel()

	stdout := new(bytes.Buffer)
	goProgram := &Go{
		ExecutableConfig: &ExecutableConfig{
			KnownName: "my-cell",
			Stdout:    stdout,
			Stderr:    new(bytes.Buffer),
		},
		Source: `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Print(os.Getenv("RUNME_NAME"))
}`,
	}

	require.NoError(t, goProgram.Run(context.Background()))
	assert.Equal(t, "my-cell", stdout.String())
}

func TestPrepareScript(t *testing.T) {
	script := prepareScriptFromCommands([]string{
		`# macOS`,
		`brew bundle --no-lock`,
		`brew upgrade`,
	}, "bash")
	assert.Equal(t, "set -e -o pipefail\n# macOS\nbrew bundle --no-lock\nbrew upgrade\n\n", script)

	script = prepareScriptFromCommands([]string{
		"deno install \\",
		"--allow-read --allow-write \\",
		"--allow-env --allow-net --allow-run \\",
		"--no-check \\",
		"-r -f https://deno.land/x/deploy/deployctl.ts",
	}, "bash")
	assert.Equal(t, "set -e -o pipefail\ndeno install \\\n--allow-read --allow-write \\\n--allow-env --allow-net --allow-run \\\n--no-check \\\n-r -f https://deno.land/x/deploy/deployctl.ts\n\n", script)

	script = prepareScriptFromCommands([]string{
		`pipenv run bash -c 'echo "Some message"'`,
	}, "bash")
	assert.Equal(t, "set -e -o pipefail\npipenv run bash -c 'echo \"Some message\"'\n\n", script)

	script = prepareScriptFromCommands([]string{
		`brew bundle --no-lock`,
		`brew upgrade`,
	}, "sh")
	assert.Equal(t, "set -e\nbrew bundle --no-lock\nbrew upgrade\n\n", script)

	script = prepareScriptFromCommands([]string{
		`brew bundle --no-lock`,
		`brew upgrade`,
	}, "pwsh")
	assert.Equal(t, "\nbrew bundle --no-lock\nbrew upgrade\n\n", script)
}

func TestShellFromShellPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t,
			"pwsh",
			ShellFromShellPath("c:\\System32\\pwsh.exe"),
		)
	} else {
		assert.Equal(t,
			"sh",
			ShellFromShellPath("/bin/sh"),
		)

		assert.Equal(t,
			"bash",
			ShellFromShellPath("/bin/bash"),
		)

		assert.Equal(t,
			"zsh",
			ShellFromShellPath("/bin/zsh"),
		)
	}
}
