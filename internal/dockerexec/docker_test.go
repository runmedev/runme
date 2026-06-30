//go:build docker_enabled

package dockerexec

import (
	"bytes"
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	"github.com/stretchr/testify/require"
)

func TestDockerCommandContext(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()

	docker, err := New(
		&Options{
			Image: "alpine:3.19",
		},
	)
	require.NoError(t, err)

	// Do not parallelize the warmup step.
	t.Run("Warmup", func(t *testing.T) {
		cmd := docker.CommandContext(context.Background(), "true")
		cmd.Dir = workingDir
		require.NoError(t, cmd.Start())
		require.NoError(t, cmd.Wait())
	})

	t.Run("Cleanup", func(t *testing.T) {
		cmd := docker.CommandContext(context.Background(), "true")
		cmd.Dir = workingDir

		require.NoError(t, cmd.Start())
		require.NotEmpty(t, cmd.containerID)
		_, err := docker.client.ContainerInspect(context.Background(), cmd.containerID)
		require.NoError(t, err)

		require.NoError(t, cmd.Wait())
		_, err = docker.client.ContainerInspect(context.Background(), cmd.containerID)
		require.True(t, errdefs.IsNotFound(err), "expected container to be removed, got %v", err)
	})

	t.Run("DebugLeavesContainer", func(t *testing.T) {
		debugDocker, err := New(
			&Options{
				Debug: true,
				Image: "alpine:3.19",
			},
		)
		require.NoError(t, err)

		cmd := debugDocker.CommandContext(context.Background(), "true")
		cmd.Dir = workingDir

		require.NoError(t, cmd.Start())
		require.NotEmpty(t, cmd.containerID)
		require.NoError(t, cmd.Wait())
		_, err = debugDocker.client.ContainerInspect(context.Background(), cmd.containerID)
		require.NoError(t, err)

		require.NoError(t, debugDocker.client.ContainerRemove(
			context.Background(),
			cmd.containerID,
			container.RemoveOptions{Force: true},
		))
	})

	t.Run("NotTTY", func(t *testing.T) {
		t.Parallel()

		stdout := bytes.NewBuffer(nil)

		cmd := docker.CommandContext(context.Background(), "echo", "hello")
		cmd.Dir = workingDir
		cmd.Stdout = stdout

		require.NoError(t, cmd.Start())
		require.NoError(t, cmd.Wait())
		require.Equal(t, "hello\n", stdout.String())
	})

	t.Run("TTY", func(t *testing.T) {
		t.Parallel()

		stdout := bytes.NewBuffer(nil)

		cmd := docker.CommandContext(context.Background(), "echo", "hello")
		cmd.Dir = workingDir
		cmd.TTY = true
		cmd.Stdout = stdout

		require.NoError(t, cmd.Start())
		require.NoError(t, cmd.Wait())
		require.Equal(t, "hello\r\n", stdout.String())
	})

	t.Run("Shell", func(t *testing.T) {
		t.Parallel()

		stdout := bytes.NewBuffer(nil)

		cmd := docker.CommandContext(context.Background(), "sh", "-c", "echo hello")
		cmd.Dir = workingDir
		cmd.TTY = true
		cmd.Stdout = stdout

		require.NoError(t, cmd.Start())
		require.NoError(t, cmd.Wait())
		require.Equal(t, "hello\r\n", stdout.String())
	})
}
