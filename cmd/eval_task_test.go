package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type fakeEvalTaskNewer struct {
	err error
}

func (n fakeEvalTaskNewer) Run([]string) error {
	return n.err
}

func TestEvalTaskNewCmdPassesOptionsToHarborTaskNewer(t *testing.T) {
	var gotOpts harbor.EvalTaskNewOptions
	var gotArgs []string
	cmd := evalCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"task",
		"new",
		"runmedev/my-task",
		"--tasks-dir", "custom/tasks",
		"--org", "ignored",
		"--description", "A useful task",
		"--author", "Alice <alice@example.com>",
		"--author", "Bob",
		"--no-solution",
		"--force",
	})
	restoreEvalTaskNewer(t, func(opts harbor.EvalTaskNewOptions) evalTaskNewer {
		gotOpts = opts
		return evalTaskNewerFunc(func(args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		})
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotArgs, []string{"runmedev/my-task"}) {
		t.Fatalf("args = %#v", gotArgs)
	}
	if gotOpts.TasksDir != "custom/tasks" ||
		gotOpts.Org != "ignored" ||
		gotOpts.Description != "A useful task" ||
		!reflect.DeepEqual(gotOpts.Authors, []string{"Alice <alice@example.com>", "Bob"}) ||
		!gotOpts.NoSolution ||
		!gotOpts.Force ||
		gotOpts.Stdout != &stdout ||
		gotOpts.Stderr != &stderr {
		t.Fatalf("options = %#v", gotOpts)
	}
}

func TestEvalTaskNewCmdRejectsMissingName(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{"task", "new"})
	restoreEvalTaskNewer(t, func(harbor.EvalTaskNewOptions) evalTaskNewer {
		return fakeEvalTaskNewer{}
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalTaskNewCmdRejectsExtraArgs(t *testing.T) {
	cmd := evalCmd()
	cmd.SetArgs([]string{"task", "new", "one", "two"})
	restoreEvalTaskNewer(t, func(harbor.EvalTaskNewOptions) evalTaskNewer {
		return fakeEvalTaskNewer{}
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalTaskNewCmdHelpMentionsTasksDefault(t *testing.T) {
	cmd := evalCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"task", "new", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{
		"Usage:\n  eval task new <org/name> [flags]",
		"Create a Harbor eval task scaffold for Runme.",
		"When --tasks-dir is omitted, runme eval task new writes under ./" + harbor.DefaultEvalDatasetPath + ".",
		"--tasks-dir string",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}
}

type evalTaskNewerFunc func([]string) error

func (f evalTaskNewerFunc) Run(args []string) error {
	return f(args)
}

func restoreEvalTaskNewer(t *testing.T, fn func(harbor.EvalTaskNewOptions) evalTaskNewer) {
	t.Helper()
	previous := newEvalTaskNewer
	newEvalTaskNewer = fn
	t.Cleanup(func() {
		newEvalTaskNewer = previous
	})
}
