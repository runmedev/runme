package dagger

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{name: "script arguments", command: "dagger script --debug", isCommand: true},
		{name: "v1 shell arguments", command: "dagger shell --container foo"},
		{name: "legacy shell argument", command: "dagger shell --debug"},
		{name: "wrapper", command: "sudo dagger shell"},
		{name: "substring", command: "not dagger shell"},
		{name: "suffix", command: "dagger shellcheck"},
		{name: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.isCommand, IsCommand(tt.command))
		})
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
		{name: "release selector", command: "dagger --x-release v1.0.0-beta.25 shell", version: "v1.0.0-beta.25", want: "dagger --x-release v1.0.0-beta.25 script", wantVersion: "dagger --x-release v1.0.0-beta.25 version"},
		{name: "script args", command: "dagger script --debug", version: "v1.0.0", want: "dagger script --debug", wantVersion: "dagger version"},
		{name: "v1 terminal shell", command: "dagger shell --container foo", want: "dagger shell --container foo"},
		{name: "unrelated", command: "bash -c true", want: "bash -c true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var gotVersion string
			resolver := newCommandResolver(func(_ context.Context, fields, _ []string) ([]byte, error) {
				gotVersion = strings.Join(fields, " ")
				return []byte(tt.version), tt.probeErr
			})

			fields, err := resolver.resolve(context.Background(), strings.Fields(tt.command), nil)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, strings.Join(fields, " "))
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

func TestCommandResolverCachesVersion(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	resolver := newCommandResolver(func(_ context.Context, _, _ []string) ([]byte, error) {
		calls.Add(1)
		return []byte("v1.0.0-beta.25"), nil
	})

	for range 2 {
		resolved, err := resolver.resolve(context.Background(), []string{"dagger", "shell"}, []string{"PATH=/bin"})
		require.NoError(t, err)
		assert.Equal(t, []string{"dagger", "script"}, resolved)
	}
	assert.Equal(t, int32(1), calls.Load())

	_, err := resolver.resolve(context.Background(), []string{"dagger", "shell"}, []string{"PATH=/different"})
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
