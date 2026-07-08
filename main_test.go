//go:build !windows && test_with_txtar

package main

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/creack/pty"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/internal/testutils"
)

// realHome captures the real HOME before testscript overrides it.
// This is needed for tools like asdf that use shims requiring $HOME.
var realHome = os.Getenv("HOME")

func TestMain(m *testing.M) {
	setNoTelemetryEnv()

	os.Exit(testscript.RunMain(m, map[string]func() int{
		"runme": root,
	}))
}

func setNoTelemetryEnv() {
	_ = os.Setenv("DO_NOT_TRACK", "true")
	_ = os.Setenv("SCARF_NO_ANALYTICS", "true")
}

func setupNoTelemetry(env *testscript.Env) error {
	env.Setenv("DO_NOT_TRACK", "true")
	env.Setenv("SCARF_NO_ANALYTICS", "true")
	return nil
}

// setupWithRealHome provides a Setup function that preserves the real HOME.
// This is needed for tools like asdf/nvm that use shims requiring $HOME.
func setupWithRealHome(env *testscript.Env) error {
	env.Setenv("HOME", realHome)
	return setupNoTelemetry(env)
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
		Setup:           setupNoTelemetry,
	})
}

func TestRunmeTags(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/tags",
		ContinueOnError: true,
		Setup:           setupNoTelemetry,
	})
}

func TestRunmeRunAll(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/runall",
		ContinueOnError: true,
		Setup:           setupNoTelemetry,
	})
}

func TestRunmeBeta(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:             "testdata/beta",
		ContinueOnError: true,
		Setup:           setupNoTelemetry,
	})
}

func TestRunmeFilePermissions(t *testing.T) {
	if testutils.IsRunningInDocker() {
		t.Skip("Test skipped when running inside a Docker container")
	}

	testscript.Run(t, testscript.Params{
		Dir:             "testdata/permissions",
		ContinueOnError: true,
		Setup:           setupNoTelemetry,
	})
}

func TestSkipPromptsWithinAPty(t *testing.T) {
	err := os.Setenv("RUNME_VERBOSE", "false")
	require.NoError(t, err)
	defer os.Unsetenv("RUNME_VERBOSE")

	cmd := exec.Command("go", "run", ".", "run", "skip-prompts-sample", "--chdir", "./examples/frontmatter/skipPrompts")
	cmd.Env = append(os.Environ(), "DO_NOT_TRACK=true", "SCARF_NO_ANALYTICS=true")
	ptmx, err := pty.StartWithAttrs(cmd, &pty.Winsize{Rows: 25, Cols: 80}, &syscall.SysProcAttr{})
	require.NoError(t, err)
	defer ptmx.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(ptmx) // ignoring errors explicitly

	expected := "The content of ENV is <insert-env-here>"
	current := buf.String()
	current = removeAnsiCodes(current)
	current = strings.TrimSpace(current)
	require.Equal(t, expected, current, "output does not match")
}

func removeAnsiCodes(str string) string {
	re := regexp.MustCompile(`\x1b\[.*?[a-zA-Z]|\x1b\].*?\x1b\\`)
	return re.ReplaceAllString(str, "")
}
