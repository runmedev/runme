//go:build windows

package cmd

import (
	"context"
	"os"
	"time"
)

// waitParentExit blocks until the process identified by pid exits,
// or until ctx is done.
//
// Windows does not re-parent orphaned processes, so instead of polling
// the parent PID we wait on a handle to the parent process.
func waitParentExit(ctx context.Context, pid int, _ time.Duration) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		// The process does not exist anymore.
		return nil
	}

	done := make(chan struct{})
	go func() {
		_, _ = p.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// The server is shutting down. Leave the handle to Wait, which will
		// release it once the process exits; closing it here would race with
		// the pending wait.
		return ctx.Err()
	}
}
