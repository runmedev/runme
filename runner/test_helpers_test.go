//go:build !windows

package runner

import (
	"os/exec"
	"strings"
	"testing"
)

func requireExecutable(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("skipping test: %q is not available in PATH", name)
	}
}

func requireWorkingBabashka(t *testing.T) {
	t.Helper()
	requireExecutable(t, "bb")

	cmd := exec.Command("bb", "-e", `(println "ok")`)
	if err := cmd.Run(); err != nil {
		t.Skipf("skipping test: bb is present but not runnable: %v", err)
	}
}

func stripKnownBashStartupNoise(str string) string {
	lines := strings.Split(str, "\n")
	filtered := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "bash: complete: nosort: invalid option name" {
			continue
		}
		if trimmed == "bash: /.local/bin/env: No such file or directory" {
			continue
		}
		if trimmed == "bash: /.cargo/env: No such file or directory" {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}
