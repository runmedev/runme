//go:build !windows

package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
)

func pidAlive(pid int) bool {
	// On Unix, sending signal 0 to a process checks for existence.
	return syscall.Kill(pid, 0) == nil
}

// readPIDFile reads a PID from a file, retrying until available or timeout.
func readPIDFile(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && len(bytes.TrimSpace(data)) > 0 {
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for PID in %s", path)
	return 0
}

func TestProcessLifecycle_NativeLinked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	pidFile1 := filepath.Join(tmpDir, "pid1")

	factory := NewFactory(
		WithLogger(zaptest.NewLogger(t)),
		WithProcessLifecycle(ProcessLifecycleLinked),
	)

	// Use NoShell to bypass inline shell wrapper and its FIFO-based env collection.
	cfg := &ProgramConfig{
		ProgramName: "bash",
		Arguments: []string{"-c", fmt.Sprintf(
			`sleep 300 & echo $! > %s; wait`,
			pidFile1,
		)},
		Mode: runnerv2.CommandMode_COMMAND_MODE_INLINE,
	}

	cmd, err := factory.Build(cfg, CommandOptions{NoShell: true})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, cmd.Start(ctx))

	childPid := readPIDFile(t, pidFile1, 5*time.Second)
	time.Sleep(100 * time.Millisecond)

	// With Linked lifecycle, the child should be in the same process
	// group as the parent (no Setpgid). OS signals propagate naturally.
	parentPgid, err := syscall.Getpgid(os.Getpid())
	require.NoError(t, err)
	cmdPgid, err := syscall.Getpgid(cmd.Pid())
	require.NoError(t, err)
	assert.Equal(t, parentPgid, cmdPgid, "command should share parent's process group")

	childPgid, err := syscall.Getpgid(childPid)
	require.NoError(t, err)
	assert.Equal(t, parentPgid, childPgid, "grandchild should share parent's process group")

	// Clean up.
	cancel()
	_ = cmd.Wait(context.Background())
	_ = syscall.Kill(childPid, syscall.SIGKILL)
}

func TestProcessLifecycle_NativeIsolated(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	pidFile1 := filepath.Join(tmpDir, "pid1")
	pidFile2 := filepath.Join(tmpDir, "pid2")

	factory := NewFactory(WithProcessLifecycle(ProcessLifecycleIsolated))

	cfg := &ProgramConfig{
		ProgramName: "bash",
		Arguments: []string{
			"-c",
			`sleep 300 & echo $! > "$1"; sleep 300 & echo $! > "$2"; wait`,
			"bash", pidFile1, pidFile2,
		},
		Mode: runnerv2.CommandMode_COMMAND_MODE_INLINE,
	}

	// A real file lets Wait reap Bash while its children still hold the output
	// descriptors open.
	output, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer output.Close()

	cmd, err := factory.Build(cfg, CommandOptions{NoShell: true, Stdout: output, Stderr: output})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	require.NoError(t, cmd.Start(ctx))

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		_ = cmd.Wait(context.Background())
	}()

	var isolatedPgid int
	t.Cleanup(func() {
		if isolatedPgid > 0 {
			err := syscall.Kill(-isolatedPgid, syscall.SIGKILL)
			if err != nil && err != syscall.ESRCH {
				t.Errorf("kill isolated process group %d: %v", isolatedPgid, err)
			}
		}
		cancel()

		select {
		case <-waitDone:
		case <-time.After(5 * time.Second):
			t.Error("timed out waiting for the isolated command after cleanup")
		}
	})

	mainPid := cmd.Pid()
	require.Greater(t, mainPid, 0)
	pgid, err := syscall.Getpgid(mainPid)
	require.NoError(t, err)
	require.Equal(t, mainPid, pgid, "command should lead its own process group")
	isolatedPgid = pgid

	pid1 := readPIDFile(t, pidFile1, 5*time.Second)
	pid2 := readPIDFile(t, pidFile2, 5*time.Second)

	// Kill only Bash and observe its exit before checking the descendants.
	require.NoError(t, syscall.Kill(mainPid, syscall.SIGKILL))
	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Bash to exit after SIGKILL")
	}

	assert.True(t, pidAlive(pid1), "child1 should still be alive with ISOLATED lifecycle")
	assert.True(t, pidAlive(pid2), "child2 should still be alive with ISOLATED lifecycle")
}

func TestProcessLifecycle_VirtualLinked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	pidFile1 := filepath.Join(tmpDir, "pid1")
	pidFile2 := filepath.Join(tmpDir, "pid2")

	factory := NewFactory(
		WithLogger(zaptest.NewLogger(t)),
		WithProcessLifecycle(ProcessLifecycleLinked),
	)

	cfg := &ProgramConfig{
		ProgramName: "bash",
		Arguments: []string{"-c", fmt.Sprintf(
			`sleep 300 & echo $! > %s; sleep 300 & echo $! > %s; wait`,
			pidFile1, pidFile2,
		)},
		Interactive: true,
		Mode:        runnerv2.CommandMode_COMMAND_MODE_INLINE,
	}

	cmd, err := factory.Build(cfg, CommandOptions{NoShell: true})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, cmd.Start(ctx))

	pid1 := readPIDFile(t, pidFile1, 5*time.Second)
	pid2 := readPIDFile(t, pidFile2, 5*time.Second)

	time.Sleep(100 * time.Millisecond)

	require.True(t, pidAlive(pid1), "child1 should be alive before cancel")
	require.True(t, pidAlive(pid2), "child2 should be alive before cancel")

	cancel()
	_ = cmd.Wait(context.Background())

	time.Sleep(200 * time.Millisecond)

	assert.False(t, pidAlive(pid1), "child1 should be dead after cancel with LINKED lifecycle")
	assert.False(t, pidAlive(pid2), "child2 should be dead after cancel with LINKED lifecycle")
}

func TestProcessLifecycle_CLIModeLinked(t *testing.T) {
	t.Parallel()

	// Factory defaults to Isolated, but CLI mode overrides the lifecycle
	// to Linked so the child stays in the parent's process group.
	factory := NewFactory(
		WithLogger(zaptest.NewLogger(t)),
	)

	cfg := &ProgramConfig{
		ProgramName: "echo",
		Arguments:   []string{"-n", "test"},
		Mode:        runnerv2.CommandMode_COMMAND_MODE_CLI,
	}

	stdout := bytes.NewBuffer(nil)
	cmd, err := factory.Build(cfg, CommandOptions{Stdout: stdout})
	require.NoError(t, err)

	require.NoError(t, cmd.Start(context.Background()))
	require.NoError(t, cmd.Wait(context.Background()))
	assert.Equal(t, "test", stdout.String())
}
