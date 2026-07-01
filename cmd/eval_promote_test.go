package cmd

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type fakeEvalPromoter struct {
	err error
}

func (p fakeEvalPromoter) Run([]string) error {
	return p.err
}

func TestEvalPromoteCmdPassesOptionsToHarborPromoter(t *testing.T) {
	var gotOpts harbor.EvalPromoteOptions
	var gotArgs []string
	cmd := evalCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"promote",
		"examples/harbor/datasets/runme-rewardkit",
		"--jobs-dir", "jobs",
		"--job", "jobs/job-1",
		"--dry-run",
		"--evidence-only",
		"--artifacts",
		"--include-oracle",
		"--allow-errors",
		"--promote-anyway",
		"--message", "Custom subject",
	})
	restoreEvalPromoter(t, func(opts harbor.EvalPromoteOptions) evalPromoter {
		gotOpts = opts
		return evalPromoterFunc(func(args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		})
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotArgs, []string{"examples/harbor/datasets/runme-rewardkit"}) {
		t.Fatalf("args = %#v", gotArgs)
	}
	if gotOpts.JobsDir != "jobs" ||
		gotOpts.Job != "jobs/job-1" ||
		gotOpts.DatasetPath != "examples/harbor/datasets/runme-rewardkit" ||
		!gotOpts.DryRun ||
		!gotOpts.EvidenceOnly ||
		!gotOpts.Artifacts ||
		!gotOpts.IncludeOracle ||
		!gotOpts.AllowErrors ||
		!gotOpts.PromoteAnyway ||
		gotOpts.Latest ||
		gotOpts.Message != "Custom subject" ||
		gotOpts.Stdout != &stdout ||
		gotOpts.Stderr != &stderr {
		t.Fatalf("options = %#v", gotOpts)
	}
}

func TestEvalPromoteCmdRejectsMultipleDatasetPaths(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{"promote", "one", "two"})
	restoreEvalPromoter(t, func(harbor.EvalPromoteOptions) evalPromoter {
		return fakeEvalPromoter{}
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalPromoteCmdPassesLatest(t *testing.T) {
	var gotOpts harbor.EvalPromoteOptions
	cmd := evalCmd()
	cmd.SetArgs([]string{"promote", "--latest"})
	restoreEvalPromoter(t, func(opts harbor.EvalPromoteOptions) evalPromoter {
		gotOpts = opts
		return fakeEvalPromoter{}
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !gotOpts.Latest {
		t.Fatal("Latest = false, want true")
	}
}

func TestEvalPromoteCmdHelpMentionsDatasetDefault(t *testing.T) {
	cmd := evalCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"promote", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{
		"Usage:\n  eval promote [dataset-path] [flags]",
		"Commit staged changes with eval job evidence.",
		"When dataset-path is omitted, runme eval promote uses ./" + harbor.DefaultEvalDatasetPath + ".",
		"--job string        Eval job directory to promote",
		"Promote the latest eval job under --jobs-dir",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}
}

func TestRunEvalPromotePropagatesDelegateExitCode(t *testing.T) {
	restoreEvalPromoter(t, func(harbor.EvalPromoteOptions) evalPromoter {
		return fakeEvalPromoter{err: fakeExitError{code: 42}}
	})

	err := runEvalPromote(evalPromoteOptions{}, nil)
	var exitErr ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want ExitCodeError", err, err)
	}
	if exitErr.Code != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.Code)
	}
}

type evalPromoterFunc func([]string) error

func (f evalPromoterFunc) Run(args []string) error {
	return f(args)
}

func restoreEvalPromoter(t *testing.T, fn func(harbor.EvalPromoteOptions) evalPromoter) {
	t.Helper()
	previous := newEvalPromoter
	newEvalPromoter = fn
	t.Cleanup(func() {
		newEvalPromoter = previous
	})
}
