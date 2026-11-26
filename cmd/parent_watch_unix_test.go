//go:build !windows

package cmd

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitParentExit_ParentAlive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := waitParentExit(ctx, os.Getppid(), 10*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWaitParentExit_NotParent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// A PID that is not our parent must be reported as gone immediately.
	err := waitParentExit(ctx, os.Getppid()+1, 10*time.Millisecond)
	require.NoError(t, err)
}

// TestWaitParentExit_Orphaned verifies the watchdog fires when the real
// parent process is killed and the child gets re-parented.
func TestWaitParentExit_Orphaned(t *testing.T) {
	if os.Getenv("RUNME_TEST_PARENT_WATCH_CHILD") == "1" {
		pid, _ := strconv.Atoi(os.Getenv("RUNME_TEST_PARENT_PID"))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := waitParentExit(ctx, pid, 10*time.Millisecond); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}

	// Spawn: sh (parent) -> test binary (child). Then kill sh and expect
	// the child to notice and exit with code 0.
	sh := exec.Command(
		"sh", "-c",
		`RUNME_TEST_PARENT_PID=$$ "$0" -test.run=^TestWaitParentExit_Orphaned$ & echo $!; wait`,
		os.Args[0],
	)
	sh.Env = append(os.Environ(), "RUNME_TEST_PARENT_WATCH_CHILD=1")
	out, err := sh.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, sh.Start())

	buf := make([]byte, 32)
	n, err := out.Read(buf)
	require.NoError(t, err)
	childPID, err := strconv.Atoi(string(buf[:n-1]))
	require.NoError(t, err)

	require.NoError(t, sh.Process.Kill())
	_ = sh.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		// Signal 0 checks for existence; the orphan is reaped by init.
		if p, _ := os.FindProcess(childPID); p.Signal(syscall.Signal(0)) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("child process %d did not exit after its parent was killed", childPID)
}
