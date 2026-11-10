package dagger

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/internal/execution"
)

func TestIsCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		command   string
		isCommand bool
	}{
		{name: "legacy", command: "dagger shell", isCommand: true},
		{name: "script", command: "dagger script", isCommand: true},
		{name: "path", command: "/usr/local/bin/dagger shell", isCommand: true},
		{name: "windows path", command: `C:\Tools\dagger.exe shell`, isCommand: true},
		{name: "env", command: "/usr/bin/env dagger shell", isCommand: true},
		{name: "env option", command: "env -S dagger shell", isCommand: true},
		{name: "global flag", command: "dagger --progress=plain shell", isCommand: true},
		{name: "trailing global flag", command: "dagger shell --progress=plain", isCommand: true},
		{name: "trailing global flag value", command: "dagger shell --progress plain", isCommand: true},
		{name: "mixed global flags", command: "dagger --debug shell --progress plain -vv", isCommand: true},
		{name: "trailing debug flag", command: "dagger shell --debug", isCommand: true},
		{name: "trailing release selector", command: "dagger shell --x-release v1.0.0-beta.25", isCommand: true},
		{name: "script arguments", command: "dagger script --debug", isCommand: true},
		{name: "shell arguments", command: "dagger shell --container foo", isCommand: true},
		{name: "shell command", command: "dagger shell -c echo", isCommand: true},
		{name: "shell list", command: "dagger shell --list", isCommand: true},
		{name: "shell target", command: "dagger shell my-container --progress=plain", isCommand: true},
		{name: "unknown flag", command: "dagger shell --future-flag foo", isCommand: true},
		{name: "missing global flag value", command: "dagger shell --progress", isCommand: true},
		{name: "positional after separator", command: "dagger shell -- --debug", isCommand: true},
		{name: "wrapper", command: "sudo dagger shell"},
		{name: "substring", command: "not dagger shell"},
		{name: "suffix", command: "dagger shellcheck"},
		{name: "shell in flag value", command: "dagger --progress=shell"},
		{name: "script in flag value", command: "dagger --future-flag=script"},
		{name: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.isCommand, IsCommand(tt.command))
		})
	}
}

func TestInvocationFlagPermutations(t *testing.T) {
	t.Parallel()

	for _, interpreter := range []string{"shell", "script"} {
		for _, flags := range []struct {
			name          string
			before, after []string
		}{
			{name: "bare"},
			{name: "equals before", before: []string{"--progress=plain"}},
			{name: "equals after", after: []string{"--progress=plain"}},
			{name: "value before", before: []string{"--progress", "plain"}},
			{name: "value after", after: []string{"--progress", "plain"}},
			{name: "both sides", before: []string{"--debug"}, after: []string{"--progress", "plain", "-vv"}},
			{name: "unknown flags", before: []string{"--future-global=value"}, after: []string{"--future-option", "value"}},
		} {
			t.Run(interpreter+"/"+flags.name, func(t *testing.T) {
				t.Parallel()
				args := append(append([]string(nil), flags.before...), interpreter)
				args = append(args, flags.after...)
				assert.True(t, IsCommand("dagger "+strings.Join(args, " ")))
				invocation, ok := parseInvocation("dagger", args)
				require.True(t, ok)
				assert.Equal(t, args, invocation.command.Arguments())
				wantArgs := append(append([]string(nil), flags.before...), "script")
				wantArgs = append(wantArgs, flags.after...)
				assert.Equal(t, wantArgs, invocation.asScript().Arguments())
			})
		}
	}
}

func TestCommandResolver(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		command     string
		version     string
		probeErr    error
		want        string
		wantErr     string
		wantVersion string
	}{
		{name: "legacy stable", command: "dagger shell", version: "dagger v0.21.11 darwin/arm64", want: "dagger shell", wantVersion: "dagger version"},
		{name: "legacy v1 prerelease", command: "dagger shell", version: "version: v1.0.0-beta.25", want: "dagger script", wantVersion: "dagger version"},
		{name: "script v1", command: "dagger script", version: "version: v1.0.0-beta.25", want: "dagger script", wantVersion: "dagger version"},
		{name: "script stable", command: "dagger script", version: "dagger v0.21.11", wantErr: "requires Dagger v1 or newer", wantVersion: "dagger version"},
		{name: "legacy probe failure", command: "dagger shell", probeErr: errors.New("boom"), want: "dagger shell", wantVersion: "dagger version"},
		{name: "script probe failure", command: "dagger script", probeErr: errors.New("boom"), wantErr: "determine Dagger CLI version", wantVersion: "dagger version"},
		{name: "path", command: "/opt/dagger shell", version: "v1.0.0", want: "/opt/dagger script", wantVersion: "/opt/dagger version"},
		{name: "env", command: "env -S dagger shell", version: "v1.0.0", want: "env -S dagger script", wantVersion: "env -S dagger version"},
		{name: "global flag", command: "dagger --progress=plain shell", version: "v1.0.0", want: "dagger --progress=plain script", wantVersion: "dagger version"},
		{name: "trailing global flag legacy", command: "dagger shell --progress=plain", version: "v0.21.11", want: "dagger shell --progress=plain", wantVersion: "dagger version"},
		{name: "trailing global flag v1", command: "dagger shell --progress=plain", version: "v1.0.0", want: "dagger script --progress=plain", wantVersion: "dagger version"},
		{name: "trailing global flag value", command: "dagger shell --progress plain", version: "v1.0.0", want: "dagger script --progress plain", wantVersion: "dagger version"},
		{name: "trailing global flag probe failure", command: "dagger shell --debug", probeErr: errors.New("boom"), want: "dagger shell --debug", wantVersion: "dagger version"},
		{name: "mixed global flags", command: "dagger --debug shell --progress plain -vv", version: "v1.0.0", want: "dagger --debug script --progress plain -vv", wantVersion: "dagger version"},
		{name: "release selector", command: "dagger --x-release v1.0.0-beta.25 shell", version: "v1.0.0-beta.25", want: "dagger --x-release v1.0.0-beta.25 script", wantVersion: "dagger --x-release v1.0.0-beta.25 version"},
		{name: "trailing release selector", command: "dagger shell --x-release v1.0.0-beta.25", version: "v1.0.0-beta.25", want: "dagger script --x-release v1.0.0-beta.25", wantVersion: "dagger --x-release v1.0.0-beta.25 version"},
		{name: "trailing release selector equals", command: "dagger shell --x-release=v1.0.0-beta.25", version: "v1.0.0-beta.25", want: "dagger script --x-release=v1.0.0-beta.25", wantVersion: "dagger --x-release=v1.0.0-beta.25 version"},
		{name: "script args", command: "dagger script --debug", version: "v1.0.0", want: "dagger script --debug", wantVersion: "dagger version"},
		{name: "shell arguments v1", command: "dagger shell --container foo", version: "v1.0.0", want: "dagger script --container foo", wantVersion: "dagger version"},
		{name: "unknown flag v1", command: "dagger shell --future-flag foo", version: "v1.0.0", want: "dagger script --future-flag foo", wantVersion: "dagger version"},
		{name: "unknown flag legacy", command: "dagger shell --future-flag foo", version: "v0.21.11", want: "dagger shell --future-flag foo", wantVersion: "dagger version"},
		{name: "missing flag value", command: "dagger shell --progress", version: "v1.0.0", want: "dagger script --progress", wantVersion: "dagger version"},
		{name: "positional arguments", command: "dagger shell example.dsh", version: "v1.0.0", want: "dagger script example.dsh", wantVersion: "dagger version"},
		{name: "release selector after separator", command: "dagger shell -- --x-release=foo", version: "v1.0.0", want: "dagger script -- --x-release=foo", wantVersion: "dagger version"},
		{name: "unrelated", command: "bash -c true", want: "bash -c true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var gotVersion string
			resolver := newResolver(func(_ context.Context, cmd execution.Invocation, _ []string) ([]byte, error) {
				gotVersion = cmd.String()
				return []byte(tt.version), tt.probeErr
			})

			fields := strings.Fields(tt.command)
			if len(fields) == 0 {
				assert.Empty(t, tt.want)
				return
			}
			invocation, ok := parseInvocation(fields[0], fields[1:])
			if !ok {
				assert.Equal(t, tt.want, invocationFromFields(fields).String())
				return
			}
			resolved, err := resolver.resolve(context.Background(), invocation, nil)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, resolved.String())
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

func TestInvocationArgumentsWithPaths(t *testing.T) {
	t.Parallel()
	for _, program := range []string{"/Applications/Dagger CLI/dagger", `C:\Program Files\Dagger\dagger.exe`} {
		t.Run(program, func(t *testing.T) {
			t.Parallel()
			args := []string{"shell", "--progress=plain", "--workspace", `C:\Workspaces\my project`}
			invocation, ok := parseInvocation(program, args)
			require.True(t, ok)
			resolved := invocation.asScript()
			assert.Equal(t, program, resolved.Executable())
			assert.Equal(t, []string{"script", "--progress=plain", "--workspace", `C:\Workspaces\my project`}, resolved.Arguments())
			assert.Equal(t, "shell", args[0])
		})
	}
}

func TestCommandResolverCachesVersion(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resolver := newResolver(func(_ context.Context, _ execution.Invocation, _ []string) ([]byte, error) {
		calls.Add(1)
		return []byte("v1.0.0-beta.25"), nil
	})
	invocation, ok := parseInvocation("dagger", []string{"shell"})
	require.True(t, ok)

	for range 2 {
		resolved, err := resolver.resolve(context.Background(), invocation, []string{"PATH=/bin"})
		require.NoError(t, err)
		assert.Equal(t, "dagger script", resolved.String())
	}
	assert.Equal(t, int32(1), calls.Load())

	_, err := resolver.resolve(context.Background(), invocation, []string{"PATH=/different"})
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	for _, output := range []string{
		"dagger v0.21.11 (image://registry.dagger.io/engine:v0.21.11) darwin/arm64/v8",
		"version:     v1.0.0-beta.25\ncommit: a45241b2",
	} {
		version, err := parseVersion(output)
		require.NoError(t, err)
		assert.NotEmpty(t, version.String())
	}
}
