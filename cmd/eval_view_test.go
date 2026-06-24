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

type fakeEvalViewer struct {
	err error
}

func (v fakeEvalViewer) Run([]string) error {
	return v.err
}

func TestEvalViewCmdPassesOptionsToHarborRunner(t *testing.T) {
	var gotOpts harbor.EvalViewOptions
	var gotArgs []string
	cmd := evalCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"view",
		"custom/jobs",
		"--port", "9090",
		"--no-open",
		"--runme-harbor-bin", "/bin/runme-harbor",
		"--debug",
	})
	restoreEvalViewer(t, func(opts harbor.EvalViewOptions) evalViewer {
		gotOpts = opts
		return evalViewerFunc(func(args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		})
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotArgs, []string{"custom/jobs"}) {
		t.Fatalf("args = %#v", gotArgs)
	}
	if gotOpts.JobsDir != "custom/jobs" ||
		gotOpts.Port != 9090 ||
		gotOpts.Open ||
		gotOpts.RunmeHarborBin != "/bin/runme-harbor" ||
		!gotOpts.Debug ||
		gotOpts.Stdout != &stdout ||
		gotOpts.Stderr != &stderr {
		t.Fatalf("options = %#v", gotOpts)
	}
}

func TestEvalViewCmdDefaultsOpen(t *testing.T) {
	var gotOpts harbor.EvalViewOptions
	cmd := evalCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"view"})
	restoreEvalViewer(t, func(opts harbor.EvalViewOptions) evalViewer {
		gotOpts = opts
		return fakeEvalViewer{}
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if gotOpts.JobsDir != "" {
		t.Fatalf("jobsDir = %q, want default", gotOpts.JobsDir)
	}
	if !gotOpts.Open {
		t.Fatal("Open = false, want true")
	}
}

func TestRunEvalViewPropagatesDelegateExitCode(t *testing.T) {
	restoreEvalViewer(t, func(harbor.EvalViewOptions) evalViewer {
		return fakeEvalViewer{err: fakeExitError{code: 42}}
	})

	err := runEvalView(evalViewOptions{}, nil)
	var exitErr ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want ExitCodeError", err, err)
	}
	if exitErr.Code != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.Code)
	}
}

func TestRunEvalViewMissingRunmeHarborMessage(t *testing.T) {
	restoreEvalViewer(t, func(harbor.EvalViewOptions) evalViewer {
		return fakeEvalViewer{err: harbor.ErrRunmeHarborMissing}
	})

	err := runEvalView(evalViewOptions{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "uv tool install runme-harbor") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "runme eval view") {
		t.Fatalf("error = %q", err.Error())
	}
}

type evalViewerFunc func([]string) error

func (f evalViewerFunc) Run(args []string) error {
	return f(args)
}

func restoreEvalViewer(t *testing.T, fn func(harbor.EvalViewOptions) evalViewer) {
	t.Helper()
	previous := newEvalViewer
	newEvalViewer = fn
	t.Cleanup(func() {
		newEvalViewer = previous
	})
}
