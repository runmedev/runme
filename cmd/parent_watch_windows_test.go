//go:build windows

package cmd

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWaitParentExit_ProcessExit verifies that the watchdog returns once the
// observed process exits. Windows does not re-parent processes, so the
// watchdog waits on a process handle rather than polling the parent PID.
func TestWaitParentExit_ProcessExit(t *testing.T) {
	p := exec.Command("cmd", "/c", "ping -n 60 127.0.0.1 > nul")
	require.NoError(t, p.Start())
	t.Cleanup(func() {
		_ = p.Process.Kill()
		_, _ = p.Process.Wait()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- waitParentExit(ctx, p.Process.Pid, 10*time.Millisecond) }()

	// Give the watchdog a moment to open the process handle.
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, p.Process.Kill())
	_, _ = p.Process.Wait()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("waitParentExit did not return after the process exited")
	}
}

// TestWaitParentExit_ProcessGone verifies that a PID that no longer exists is
// reported as gone right away instead of blocking forever.
func TestWaitParentExit_ProcessGone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// PID values well above the range Windows hands out.
	err := waitParentExit(ctx, 0x7ffffff0, 10*time.Millisecond)
	require.NoError(t, err)
}
