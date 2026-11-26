//go:build !windows

package cmd

import (
	"context"
	"os"
	"time"
)

// waitParentExit blocks until the process identified by pid is no longer
// the parent of the current process, or until ctx is done.
//
// On Unix, when the parent dies the child is re-parented (to init/launchd or
// a subreaper), so comparing os.Getppid() against the expected PID is
// a reliable signal that is not affected by PID reuse.
func waitParentExit(ctx context.Context, pid int, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if os.Getppid() != pid {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
