//go:build !windows && test_with_txtar

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/internal/testutils"
)

// realHome captures the real HOME before testscript overrides it.
// This is needed for tools like asdf that use shims requiring $HOME.
var realHome = os.Getenv("HOME")

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"runme": root,
	}))
}

// setupWithRealHome provides a Setup function that preserves the real HOME.
// This is needed for tools like asdf/nvm that use shims requiring $HOME.
func setupWithRealHome(env *testscript.Env) error {
	env.Setenv("HOME", realHome)
	return nil
}

// TestRunme tests runme end-to-end using testscript.
// Check out the package from "import" to learn more.
// More comprehensive tutorial can be found here:
// https://bitfieldconsulting.com/golang/test-scripts
func TestRunme(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/script",
		ContinueOnError: true,
		Setup:           setupWithRealHome,
	})
}

func TestRunmeFlags(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/flags",
		ContinueOnError: true,
	})
}

func TestRunmeTags(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/tags",
		ContinueOnError: true,
	})
}

func TestRunmeRunAll(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/runall",
		ContinueOnError: true,
	})
}

func TestRunmeBeta(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/beta",
		ContinueOnError: true,
	})
}

func TestRunmeFilePermissions(t *testing.T) {
	if testutils.IsRunningInDocker() {
		t.Skip("Test skipped when running inside a Docker container")
	}

	testscript.Run(t, testscript.Params{
		Dir:             "testdata/permissions",
		ContinueOnError: true,
	})
}

func TestSkipPromptsWithinAPty(t *testing.T) {
	err := os.Setenv("RUNME_VERBOSE", "false")
	require.NoError(t, err)
	defer os.Unsetenv("RUNME_VERBOSE")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", ".", "run", "skip-prompts-sample", "--chdir", "./examples/frontmatter/skipPrompts")
	ptmx, err := pty.StartWithAttrs(cmd, &pty.Winsize{Rows: 25, Cols: 80}, &syscall.SysProcAttr{})
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	readDone := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(ptmx) // best-effort read of PTY output until close
		close(readDone)
	}()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case err := <-waitDone:
		require.NoError(t, err)
	case <-ctx.Done():
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		current := strings.TrimSpace(removeAnsiCodes(buf.String()))
		t.Fatalf("command timed out waiting for PTY output: %q", current)
	}
	_ = ptmx.Close()
	<-readDone

	expected := "The content of ENV is <insert-env-here>"
	current := buf.String()
	current = removeAnsiCodes(current)
	current = stripNoiseLines(current)
	current = strings.TrimSpace(current)
	require.Equal(t, expected, current, "output does not match")
}

func removeAnsiCodes(str string) string {
	re := regexp.MustCompile(`\x1b\[.*?[a-zA-Z]|\x1b\].*?\x1b\\`)
	return re.ReplaceAllString(str, "")
}

func stripNoiseLines(str string) string {
	lines := strings.Split(str, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "(eval):") && strings.Contains(line, "no matches found: **Error:**") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
