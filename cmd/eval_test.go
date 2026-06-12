package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recordedCommand struct {
	name string
	args []string
	env  []string
}

type fakeExitError struct {
	code int
}

func (e fakeExitError) Error() string {
	return "failed"
}

func (e fakeExitError) ExitCode() int {
	return e.code
}

func TestRunEvalDelegatesOracle(t *testing.T) {
	path := t.TempDir()
	var stderr bytes.Buffer
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, &stderr)
	opts.debug = true

	err := runEval(opts, []string{
		path,
		"--extra",
		"value",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantPreflight := recordedCommand{
		name: "/bin/runme",
		args: []string{"harbor", "stdio"},
	}
	wantDelegate := recordedCommand{
		name: "/bin/runme-harbor",
		args: []string{
			"run",
			mustAbs(t, path),
			"--agent", "oracle",
			"--jobs-dir", defaultHarborJobsDir,
			"--debug",
			"--",
			"--extra",
			"value",
		},
	}
	if !sameCommand(calls[0], wantPreflight) {
		t.Fatalf("preflight = %#v, want %#v", calls[0], wantPreflight)
	}
	if !sameCommand(calls[1], wantDelegate) {
		t.Fatalf("delegate = %#v, want %#v", calls[1], wantDelegate)
	}
	wantDebug := shellCommandString(append([]string{wantDelegate.name}, wantDelegate.args...)) + "\n"
	if stderr.String() != wantDebug {
		t.Fatalf("debug output = %q, want %q", stderr.String(), wantDebug)
	}
	if got := envValue(calls[1].env, "RUNME_BIN"); got != "/bin/runme" {
		t.Fatalf("RUNME_BIN = %q, want /bin/runme", got)
	}
}

func TestRunEvalDelegatesCodexAndClaudeOptions(t *testing.T) {
	for _, tt := range []struct {
		name  string
		agent string
	}{
		{name: "codex", agent: "codex"},
		{name: "claude-code", agent: "claude-code"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := t.TempDir()
			var calls []recordedCommand
			opts := testEvalOptions(t, &calls, io.Discard)
			opts.agent = tt.agent
			opts.task = "simple-agent"
			opts.jobsDir = "jobs"
			opts.yes = true

			err := runEval(opts, []string{path})
			if err != nil {
				t.Fatal(err)
			}

			want := []string{
				"run",
				mustAbs(t, path),
				"--agent", tt.agent,
				"--jobs-dir", "jobs",
				"--task", "simple-agent",
				"-y",
			}
			if !reflect.DeepEqual(calls[1].args, want) {
				t.Fatalf("args = %#v, want %#v", calls[1].args, want)
			}
		})
	}
}

func TestRunEvalDelegatesModel(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.model = "haiku"

	err := runEval(opts, []string{path})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultHarborJobsDir,
		"--",
		"--model", "haiku",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalPreservesPassthroughModel(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, []string{path, "--model", "haiku"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultHarborJobsDir,
		"--",
		"--model", "haiku",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalRejectsDuplicateModel(t *testing.T) {
	for _, tt := range []struct {
		name        string
		passthrough []string
	}{
		{name: "separate arg", passthrough: []string{"--model", "sonnet"}},
		{name: "equals arg", passthrough: []string{"--model=sonnet"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := t.TempDir()
			var calls []recordedCommand
			opts := testEvalOptions(t, &calls, io.Discard)
			opts.model = "haiku"

			err := runEval(opts, append([]string{path}, tt.passthrough...))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "--model cannot be used together with passthrough --model") {
				t.Fatalf("error = %q", err.Error())
			}
			if len(calls) != 0 {
				t.Fatalf("calls = %#v, want none", calls)
			}
		})
	}
}

func TestRunEvalUsesEnvAndRunmeArgs(t *testing.T) {
	envHarbor := makeExecutable(t, "env-runme-harbor")
	flagHarbor := makeExecutable(t, "flag-runme-harbor")
	t.Setenv("RUNME_HARBOR_BIN", envHarbor)
	t.Setenv("RUNME_ARGS", "--existing")
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.runmeHarbor = flagHarbor
	opts.runmeBin = "/flag/runme"
	opts.runmeArgs = []string{"--chdir", "/tmp/with space"}

	err := runEval(opts, []string{path})
	if err != nil {
		t.Fatal(err)
	}

	if calls[1].name != envHarbor {
		t.Fatalf("runme-harbor = %q, want env override", calls[1].name)
	}
	if got := envValue(calls[1].env, "RUNME_BIN"); got != "/flag/runme" {
		t.Fatalf("RUNME_BIN = %q, want /flag/runme", got)
	}
	if got := envValue(calls[1].env, "RUNME_ARGS"); got != `--chdir "/tmp/with space"` {
		t.Fatalf("RUNME_ARGS = %q", got)
	}
}

func TestRunEvalPreservesExistingRunmeArgs(t *testing.T) {
	t.Setenv("RUNME_ARGS", "--existing")
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, []string{path})
	if err != nil {
		t.Fatal(err)
	}

	if got := envValue(calls[1].env, "RUNME_ARGS"); got != "--existing" {
		t.Fatalf("RUNME_ARGS = %q, want preserved env", got)
	}
}

func TestRunEvalMissingRunmeHarbor(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.lookPath = func(string) (string, error) {
		return "", os.ErrNotExist
	}

	err := runEval(opts, []string{path})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "uv tool install runme-harbor") {
		t.Fatalf("error = %q", err.Error())
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func TestRunEvalDelegatesUnknownAgent(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.agent = "bad"

	err := runEval(opts, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "bad",
		"--jobs-dir", defaultHarborJobsDir,
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalMissingPath(t *testing.T) {
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, []string{filepath.Join(t.TempDir(), "missing")})
	if err == nil || !strings.Contains(err.Error(), "task path does not exist") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunEvalPropagatesDelegateExitCode(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.commandRun = func(name string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
		calls = append(calls, recordedCommand{name: name, args: append([]string(nil), args...), env: env})
		if len(calls) == 2 {
			return fakeExitError{code: 42}
		}
		return nil
	}

	err := runEval(opts, []string{path})
	var exitErr ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want ExitCodeError", err, err)
	}
	if exitErr.Code != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.Code)
	}
}

func testEvalOptions(t *testing.T, calls *[]recordedCommand, stderr io.Writer) evalOptions {
	t.Helper()
	return evalOptions{
		agent:       "oracle",
		jobsDir:     defaultHarborJobsDir,
		runmeBin:    "/bin/runme",
		commandRun:  recordCommand(calls),
		lookPath:    fakeLookPath,
		executable:  func() (string, error) { return "/bin/current-runme", nil },
		stdout:      io.Discard,
		stderr:      stderr,
		preflight:   true,
		runmeHarbor: "",
	}
}

func recordCommand(calls *[]recordedCommand) commandRunFunc {
	return func(name string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
		*calls = append(*calls, recordedCommand{
			name: name,
			args: append([]string(nil), args...),
			env:  append([]string(nil), env...),
		})
		return nil
	}
}

func fakeLookPath(name string) (string, error) {
	switch name {
	case "runme-harbor":
		return "/bin/runme-harbor", nil
	case "/env/runme-harbor", "/flag/runme-harbor":
		return name, nil
	default:
		return "", os.ErrNotExist
	}
}

func sameCommand(got, want recordedCommand) bool {
	return got.name == want.name && reflect.DeepEqual(got.args, want.args)
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func makeExecutable(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
