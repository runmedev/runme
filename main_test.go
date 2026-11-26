//go:build !windows && test_with_txtar

package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func TestRunmeRedirectedStdinCancellation(t *testing.T) {
	for _, test := range []struct {
		language string
		script   string
	}{
		{language: "sh", script: "sh -c 'echo ready; exec cat'"},
		{language: "go", script: `package main
import ("fmt"; "os")
func main() {
	fmt.Println("ready")
	os.Stdin.Read(make([]byte, 1))
}`},
	} {
		t.Run(test.language, func(t *testing.T) {
			testRedirectedStdinCancellation(t, test.language, test.script)
		})
	}
}

func testRedirectedStdinCancellation(t *testing.T, language, script string) {
	t.Helper()
	dir := t.TempDir()
	readme := "```" + language + " {\"name\":\"read-input\"}\n" + script + "\n```\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o600))

	input, producer, err := os.Pipe()
	require.NoError(t, err)
	defer input.Close()
	defer producer.Close()

	output, childOutput, err := os.Pipe()
	require.NoError(t, err)
	defer output.Close()
	defer childOutput.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "runme", "run", "-y", "--server=", "read-input")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SHELL=sh", "RUNME_VERBOSE=false", "RUNMESHELL=")
	cmd.Stdin = input
	cmd.Stdout = childOutput
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Wait() })
	require.NoError(t, childOutput.Close())

	require.NoError(t, output.SetReadDeadline(time.Now().Add(30*time.Second)))
	reader := bufio.NewReader(output)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	require.Error(t, cmd.Wait())

	// The producer is still open. A surviving subprocess would also hold
	// its copy of the output descriptor open while it waits for input.
	require.NoError(t, output.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err = reader.ReadByte()
	require.ErrorIs(t, err, io.EOF)
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
