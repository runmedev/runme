package cmd

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type fakeEvalRunner struct {
	err error
}

func (r fakeEvalRunner) Run([]string) error {
	return r.err
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

func TestEvalCmdPassesOptionsToHarborRunner(t *testing.T) {
	var gotOpts harbor.EvalOptions
	var gotArgs []string
	cmd := evalCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"dataset",
		"--agent", "codex",
		"--task-dir", "simple-agent",
		"--jobs-dir", "jobs",
		"--ask",
		"--model", "haiku",
		"--env", "docker",
		"--runme-bin", "/bin/runme",
		"--runme-arg", "--skip-prompts",
		"--runme-harbor-bin", "/bin/runme-harbor",
		"--debug",
		"--",
		"--extra",
	})
	restoreEvalRunner(t, func(opts harbor.EvalOptions) evalRunner {
		gotOpts = opts
		return evalRunnerFunc(func(args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		})
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotArgs, []string{"dataset", "--extra"}) {
		t.Fatalf("args = %#v", gotArgs)
	}
	if gotOpts.Agent != "codex" ||
		gotOpts.TaskDir != "simple-agent" ||
		gotOpts.JobsDir != "jobs" ||
		!gotOpts.JobsDirExplicit ||
		!gotOpts.Ask ||
		gotOpts.Model != "haiku" ||
		gotOpts.Env != "docker" ||
		gotOpts.RunmeBin != "/bin/runme" ||
		!reflect.DeepEqual(gotOpts.RunmeArgs, []string{"--skip-prompts"}) ||
		gotOpts.RunmeHarborBin != "/bin/runme-harbor" ||
		!gotOpts.Debug ||
		gotOpts.Stdout != &stdout ||
		gotOpts.Stderr != &stderr ||
		!gotOpts.Preflight {
		t.Fatalf("options = %#v", gotOpts)
	}
}

func TestEvalCmdRejectsTaskFlag(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{t.TempDir(), "--task", "simple-agent"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --task") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestEvalCmdRejectsTaskNameFlag(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{t.TempDir(), "--task-name", "simple-agent"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --task-name") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestEvalCmdRejectsYesFlags(t *testing.T) {
	for _, args := range [][]string{
		{t.TempDir(), "-y"},
		{t.TempDir(), "--yes"},
	} {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			cmd := evalCmd()
			cmd.SetArgs(args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "unknown shorthand flag: 'y'") &&
				!strings.Contains(err.Error(), "unknown flag: --yes") {
				t.Fatalf("error = %q", err.Error())
			}
		})
	}
}

func TestEvalCmdHelpIncludesDefaultDatasetPath(t *testing.T) {
	cmd := evalCmd()
	var stdout bytes.Buffer
	cmd.SetArgs([]string{"--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "When dataset-path is omitted, runme eval uses ./evals/tasks.") {
		t.Fatalf("help output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `-e, --env string`) {
		t.Fatalf("help output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `Defaults to "runme"`) {
		t.Fatalf("help output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `--ask`) {
		t.Fatalf("help output = %q", stdout.String())
	}
}

func TestEvalCmdRejectsMultipleDatasetPaths(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{"first", "second"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "accepts at most 1 dataset path") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRunEvalPropagatesDelegateExitCode(t *testing.T) {
	restoreEvalRunner(t, func(harbor.EvalOptions) evalRunner {
		return fakeEvalRunner{err: fakeExitError{code: 42}}
	})

	err := runEval(evalOptions{}, []string{"dataset"})
	var exitErr ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want ExitCodeError", err, err)
	}
	if exitErr.Code != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.Code)
	}
}

func TestRunEvalMissingRunmeHarborMessage(t *testing.T) {
	restoreEvalRunner(t, func(harbor.EvalOptions) evalRunner {
		return fakeEvalRunner{err: harbor.ErrRunmeHarborMissing}
	})

	err := runEval(evalOptions{}, []string{"dataset"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "uv tool install runme-harbor") {
		t.Fatalf("error = %q", err.Error())
	}
}

type evalRunnerFunc func([]string) error

func (f evalRunnerFunc) Run(args []string) error {
	return f(args)
}

func restoreEvalRunner(t *testing.T, fn func(harbor.EvalOptions) evalRunner) {
	t.Helper()
	previous := newEvalRunner
	newEvalRunner = fn
	t.Cleanup(func() {
		newEvalRunner = previous
	})
}
