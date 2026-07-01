package cmd

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type fakeEvalComparer struct {
	err error
}

func (c fakeEvalComparer) Run([]string) error {
	return c.err
}

func TestEvalCompareCmdPassesOptionsToHarborComparer(t *testing.T) {
	var gotOpts harbor.EvalCompareOptions
	var gotArgs []string
	cmd := evalCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"compare",
		"examples/harbor/datasets/runme-rewardkit",
		"--jobs-dir", "jobs",
		"--job", "jobs/job-1",
		"--base", "HEAD~1",
		"--format", "json",
		"--include-oracle",
		"--allow-errors",
	})
	restoreEvalComparer(t, func(opts harbor.EvalCompareOptions) evalComparer {
		gotOpts = opts
		return evalComparerFunc(func(args []string) error {
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
		gotOpts.Base != "HEAD~1" ||
		gotOpts.Format != "json" ||
		!gotOpts.IncludeOracle ||
		!gotOpts.AllowErrors ||
		gotOpts.Stdout != &stdout ||
		gotOpts.Stderr != &stderr {
		t.Fatalf("options = %#v", gotOpts)
	}
}

func TestEvalCompareCmdRejectsMultipleDatasetPaths(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{"compare", "one", "two"})
	restoreEvalComparer(t, func(harbor.EvalCompareOptions) evalComparer {
		return fakeEvalComparer{}
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalCompareCmdDefaultsBaseAndFormat(t *testing.T) {
	var gotOpts harbor.EvalCompareOptions
	cmd := evalCmd()
	cmd.SetArgs([]string{"compare"})
	restoreEvalComparer(t, func(opts harbor.EvalCompareOptions) evalComparer {
		gotOpts = opts
		return fakeEvalComparer{}
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if gotOpts.Base != "HEAD" {
		t.Fatalf("Base = %q, want HEAD", gotOpts.Base)
	}
	if gotOpts.Format != "text" {
		t.Fatalf("Format = %q, want text", gotOpts.Format)
	}
}

func TestEvalCompareCmdHelpMentionsDatasetDefault(t *testing.T) {
	cmd := evalCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"compare", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{
		"Usage:\n  eval compare [dataset-path] [flags]",
		"Compare eval jobs for a dataset.",
		"When dataset-path is omitted, runme eval compare uses ./" + harbor.DefaultEvalDatasetPath + ".",
		"--job string        Compare against a specific local eval job",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}
}

func TestRunEvalComparePropagatesDelegateExitCode(t *testing.T) {
	restoreEvalComparer(t, func(harbor.EvalCompareOptions) evalComparer {
		return fakeEvalComparer{err: fakeExitError{code: 42}}
	})

	err := runEvalCompare(evalCompareOptions{}, nil)
	var exitErr ExitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want ExitCodeError", err, err)
	}
	if exitErr.Code != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.Code)
	}
}

type evalComparerFunc func([]string) error

func (f evalComparerFunc) Run(args []string) error {
	return f(args)
}

func restoreEvalComparer(t *testing.T, fn func(harbor.EvalCompareOptions) evalComparer) {
	t.Helper()
	previous := newEvalComparer
	newEvalComparer = fn
	t.Cleanup(func() {
		newEvalComparer = previous
	})
}
