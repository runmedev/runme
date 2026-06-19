package harbor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
	"github.com/go-git/go-git/v5"

	"github.com/runmedev/runme/v3/internal/ansi"
)

type recordedCommand struct {
	name       string
	args       []string
	workingDir string
	env        []string
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
	opts.Debug = true

	err := NewEvalRunner(opts).Run([]string{
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
			"--jobs-dir", defaultJobsDir(t),
			"-y",
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

func TestRunEvalDefaultsDatasetPath(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalDatasetPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		defaultDatasetPath(t),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
	if calls[1].workingDir != defaultEvalBaseDir(tmp) {
		t.Fatalf("workingDir = %q, want %q", calls[1].workingDir, defaultEvalBaseDir(tmp))
	}
}

func TestRunEvalDefaultsDatasetPathWithPassthrough(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalDatasetPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run([]string{"--model", "haiku"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		defaultDatasetPath(t),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--",
		"--model", "haiku",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
	if calls[1].workingDir != defaultEvalBaseDir(tmp) {
		t.Fatalf("workingDir = %q, want %q", calls[1].workingDir, defaultEvalBaseDir(tmp))
	}
}

func TestRunEvalDefaultsUseGitRoot(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, DefaultEvalDatasetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		filepath.Join("..", "..", DefaultEvalDatasetPath),
		"--agent", "oracle",
		"--jobs-dir", filepath.Join("..", "..", DefaultEvalJobsDir),
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
	if calls[1].workingDir != nested {
		t.Fatalf("workingDir = %q, want %q", calls[1].workingDir, nested)
	}
}

func TestRunEvalExplicitDatasetUsesInvocationCwdAndDefaultJobsUseGitRoot(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	dataset := filepath.Join(nested, "custom-dataset")
	if err := os.MkdirAll(dataset, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run([]string{"./custom-dataset"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		"custom-dataset",
		"--agent", "oracle",
		"--jobs-dir", filepath.Join("..", "..", DefaultEvalJobsDir),
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
	if calls[1].workingDir != nested {
		t.Fatalf("workingDir = %q, want %q", calls[1].workingDir, nested)
	}
	if _, err := os.Stat(dataset); err != nil {
		t.Fatal(err)
	}
}

func TestRunEvalExplicitJobsDirUsesInvocationCwd(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, DefaultEvalDatasetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.JobsDir = "custom/jobs"
	opts.JobsDirExplicit = true

	err := NewEvalRunner(opts).Run(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		filepath.Join("..", "..", DefaultEvalDatasetPath),
		"--agent", "oracle",
		"--jobs-dir", "custom/jobs",
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
	if calls[1].workingDir != nested {
		t.Fatalf("workingDir = %q, want %q", calls[1].workingDir, nested)
	}
}

func TestRunEvalExplicitAbsoluteJobsDirUnderCwdUsesRelativeDelegatePath(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalDatasetPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.JobsDir = filepath.Join(tmp, "custom", "jobs")
	opts.JobsDirExplicit = true

	err := NewEvalRunner(opts).Run(nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		DefaultEvalDatasetPath,
		"--agent", "oracle",
		"--jobs-dir", filepath.Join("custom", "jobs"),
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalExplicitAbsolutePathsOutsideCwdStayAbsolute(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	dataset := filepath.Join(t.TempDir(), "tasks")
	if err := os.MkdirAll(dataset, 0o755); err != nil {
		t.Fatal(err)
	}
	jobsDir := filepath.Join(t.TempDir(), "jobs")
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.JobsDir = jobsDir
	opts.JobsDirExplicit = true

	err := NewEvalRunner(opts).Run([]string{dataset})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		dataset,
		"--agent", "oracle",
		"--jobs-dir", jobsDir,
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
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
			opts.Agent = tt.agent
			opts.TaskDir = "simple-agent"
			opts.JobsDir = "jobs"
			opts.JobsDirExplicit = true

			err := NewEvalRunner(opts).Run([]string{path})
			if err != nil {
				t.Fatal(err)
			}

			want := []string{
				"run",
				mustAbs(t, path),
				"--agent", tt.agent,
				"--jobs-dir", "jobs",
				"--task-dir", "simple-agent",
				"-y",
			}
			if !reflect.DeepEqual(calls[1].args, want) {
				t.Fatalf("args = %#v, want %#v", calls[1].args, want)
			}
		})
	}
}

func TestRunEvalAskDoesNotDelegateYes(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Ask = true

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalDelegatesModel(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Model = "haiku"

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--",
		"--model", "haiku",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalDelegatesAgentKwargs(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.AgentKwargs = []string{"reasoning_effort=xhigh", "sandbox_mode=workspace-write"}

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--",
		"--agent-kwarg", "reasoning_effort=xhigh",
		"--agent-kwarg", "sandbox_mode=workspace-write",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalPreservesPassthroughModel(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run([]string{path, "--model", "haiku"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--",
		"--model", "haiku",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalPreservesPassthroughAgentKwargs(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run([]string{path, "--ak", "reasoning_effort=xhigh", "--agent-kwarg=sandbox_mode=workspace-write"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--",
		"--ak", "reasoning_effort=xhigh",
		"--agent-kwarg=sandbox_mode=workspace-write",
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
			opts.Model = "haiku"

			err := NewEvalRunner(opts).Run(append([]string{path}, tt.passthrough...))
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

func TestRunEvalRejectsDuplicateAgentKwargs(t *testing.T) {
	for _, tt := range []struct {
		name        string
		passthrough []string
	}{
		{name: "long separate arg", passthrough: []string{"--agent-kwarg", "sandbox_mode=workspace-write"}},
		{name: "long equals arg", passthrough: []string{"--agent-kwarg=sandbox_mode=workspace-write"}},
		{name: "alias separate arg", passthrough: []string{"--ak", "sandbox_mode=workspace-write"}},
		{name: "alias equals arg", passthrough: []string{"--ak=sandbox_mode=workspace-write"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := t.TempDir()
			var calls []recordedCommand
			opts := testEvalOptions(t, &calls, io.Discard)
			opts.AgentKwargs = []string{"reasoning_effort=xhigh"}

			err := NewEvalRunner(opts).Run(append([]string{path}, tt.passthrough...))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "--agent-kwarg cannot be used together with passthrough") {
				t.Fatalf("error = %q", err.Error())
			}
			if len(calls) != 0 {
				t.Fatalf("calls = %#v, want none", calls)
			}
		})
	}
}

func TestRunEvalDelegatesRunmeEnvAlias(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Env = "runme"

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	wantPreflight := recordedCommand{
		name: "/bin/runme",
		args: []string{"harbor", "stdio"},
	}
	wantDelegate := []string{
		"run",
		mustAbs(t, path),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--env", "runme",
	}
	if !sameCommand(calls[0], wantPreflight) {
		t.Fatalf("preflight = %#v, want %#v", calls[0], wantPreflight)
	}
	if !reflect.DeepEqual(calls[1].args, wantDelegate) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, wantDelegate)
	}
}

func TestRunEvalDelegatesNonRunmeEnvWithoutPreflight(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Env = "docker"
	opts.Agent = "codex"

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "codex",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
		"--env", "docker",
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %#v, want delegate only", calls)
	}
	if !reflect.DeepEqual(calls[0].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[0].args, want)
	}
}

func TestRunEvalDoesNotStageNonDockerWorkdir(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	dataset, workdir, target := makeHarborDockerDataset(t, workspace, "/app/source/workdir")
	if err := os.WriteFile(filepath.Join(workdir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Env = "podman"

	err := NewEvalRunner(opts).Run([]string{dataset})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target stat err = %v, want not exist", err)
	}
}

func TestRunEvalRejectsPassthroughEnvironmentFlags(t *testing.T) {
	for _, tt := range []struct {
		name        string
		passthrough []string
	}{
		{name: "env separate", passthrough: []string{"--env", "docker"}},
		{name: "env equals", passthrough: []string{"--env=docker"}},
		{name: "env shorthand separate", passthrough: []string{"-e", "docker"}},
		{name: "env shorthand joined", passthrough: []string{"-edocker"}},
		{name: "import path separate", passthrough: []string{"--environment-import-path", "pkg:Env"}},
		{name: "import path equals", passthrough: []string{"--environment-import-path=pkg:Env"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := t.TempDir()
			var calls []recordedCommand
			opts := testEvalOptions(t, &calls, io.Discard)

			err := NewEvalRunner(opts).Run(append([]string{path}, tt.passthrough...))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "use runme eval --env") {
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
	opts.RunmeHarborBin = flagHarbor
	opts.RunmeBin = "/flag/runme"
	opts.RunmeArgs = []string{"--chdir", "/tmp/with space", "--label=can't-$expand`this`\\path"}

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}

	if calls[1].name != envHarbor {
		t.Fatalf("runme-harbor = %q, want env override", calls[1].name)
	}
	if got := envValue(calls[1].env, "RUNME_BIN"); got != "/flag/runme" {
		t.Fatalf("RUNME_BIN = %q, want /flag/runme", got)
	}
	if got := envValue(calls[1].env, "RUNME_ARGS"); got != "--chdir '/tmp/with space' '--label=can'\\''t-$expand`this`\\path'" {
		t.Fatalf("RUNME_ARGS = %q", got)
	}
}

func TestRunEvalPreservesExistingRunmeArgs(t *testing.T) {
	t.Setenv("RUNME_ARGS", "--existing")
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run([]string{path})
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
	opts.LookPath = func(string) (string, error) {
		return "", os.ErrNotExist
	}

	err := NewEvalRunner(opts).Run([]string{path})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRunmeHarborMissing) {
		t.Fatalf("error = %q, want ErrRunmeHarborMissing", err.Error())
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func TestRunEvalDelegatesUnknownAgent(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Agent = "bad"

	err := NewEvalRunner(opts).Run([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "bad",
		"--jobs-dir", defaultJobsDir(t),
		"-y",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalMissingPath(t *testing.T) {
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run([]string{filepath.Join(t.TempDir(), "missing")})
	if err == nil || !strings.Contains(err.Error(), "dataset path does not exist") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunEvalMissingDefaultPath(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := NewEvalRunner(opts).Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dataset path does not exist: evals/tasks") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "pass a dataset path explicitly") {
		t.Fatalf("error = %q", err.Error())
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func TestRunEvalPropagatesDelegateExitCode(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.CommandRun = func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
		calls = append(calls, recordedCommand{name: name, args: append([]string(nil), args...), workingDir: workingDir, env: env})
		if len(calls) == 2 {
			return fakeExitError{code: 42}
		}
		return nil
	}

	err := NewEvalRunner(opts).Run([]string{path})
	var exitErr fakeExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want fakeExitError", err, err)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("exit code = %d, want 42", exitErr.ExitCode())
	}
}

func TestRunEvalPrintsExceptionDetailsAfterHarborOutput(t *testing.T) {
	path := t.TempDir()
	jobsDir := filepath.Join(t.TempDir(), "jobs")
	jobDir := filepath.Join(jobsDir, "2026-06-18__10-51-14")
	resultPath := filepath.Join(jobDir, "result.json")
	var stdout bytes.Buffer
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.JobsDir = jobsDir
	opts.JobsDirExplicit = true
	opts.Stdout = &stdout
	opts.CommandRun = func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdoutWriter, stderr io.Writer) error {
		calls = append(calls, recordedCommand{name: name, args: append([]string(nil), args...), workingDir: workingDir, env: env})
		if len(calls) == 2 {
			_, _ = fmt.Fprintf(stdoutWriter, "Exception           Count\nHarborProtocolError 1\nResults written to %s\n", resultPath)
			writeEvalException(t, jobDir, "end-to-end__abc/exception.txt", "HarborProtocolError: runme failed with useful context\n")
			return fakeExitError{code: 7}
		}
		return nil
	}

	err := NewEvalRunner(opts).Run([]string{path})
	var exitErr fakeExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want fakeExitError", err, err)
	}

	output := string(ansi.Strip(stdout.Bytes()))
	if !strings.Contains(output, "Exception           Count\nHarborProtocolError 1\nResults written to "+resultPath+"\nHarbor Exception Details\n") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "File: end-to-end__abc/exception.txt") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "HarborProtocolError: runme failed with useful context") {
		t.Fatalf("output = %q", output)
	}
}

func TestRunEvalPrintsExceptionDetailsForDockerEnvironment(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	dataset, _, _ := makeHarborDockerDataset(t, workspace, "/app/source/workdir")
	jobsDir := filepath.Join(t.TempDir(), "jobs")
	jobDir := filepath.Join(jobsDir, "2026-06-18__10-51-14")
	resultPath := filepath.Join(jobDir, "result.json")
	var stdout bytes.Buffer
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.Env = "docker"
	opts.JobsDir = jobsDir
	opts.JobsDirExplicit = true
	opts.Stdout = &stdout
	opts.CommandRun = func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdoutWriter, stderr io.Writer) error {
		calls = append(calls, recordedCommand{name: name, args: append([]string(nil), args...), workingDir: workingDir, env: env})
		_, _ = fmt.Fprintf(stdoutWriter, "Exception           Count\nHarborProtocolError 1\nResults written to %s\n", resultPath)
		writeEvalException(t, jobDir, "end-to-end__abc/exception.txt", "ValueError: docker detail should be visible\n")
		return fakeExitError{code: 7}
	}

	err := NewEvalRunner(opts).Run([]string{dataset})
	var exitErr fakeExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want fakeExitError", err, err)
	}
	output := string(ansi.Strip(stdout.Bytes()))
	if !strings.Contains(output, "Harbor Exception Details") || !strings.Contains(output, "ValueError: docker detail should be visible") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestRunEvalOnlyPrintsExceptionDetailsForReportedJob(t *testing.T) {
	path := t.TempDir()
	jobsDir := filepath.Join(t.TempDir(), "jobs")
	jobDir := filepath.Join(jobsDir, "current")
	resultPath := filepath.Join(jobDir, "result.json")
	writeEvalException(t, jobsDir, "other/attempt/exception.txt", "other detail\n")
	writeEvalException(t, jobDir, "attempt/exception.txt", "current detail\n")
	var stdout bytes.Buffer
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.JobsDir = jobsDir
	opts.JobsDirExplicit = true
	opts.Stdout = &stdout
	opts.CommandRun = func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdoutWriter, stderr io.Writer) error {
		calls = append(calls, recordedCommand{name: name, args: append([]string(nil), args...), workingDir: workingDir, env: env})
		if len(calls) == 2 {
			_, _ = fmt.Fprintf(stdoutWriter, "Exception           Count\nHarborProtocolError 1\nResults written to %s\n", resultPath)
			return fakeExitError{code: 7}
		}
		return nil
	}

	err := NewEvalRunner(opts).Run([]string{path})
	var exitErr fakeExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %v, want fakeExitError", err, err)
	}
	output := string(ansi.Strip(stdout.Bytes()))
	if !strings.Contains(output, "Harbor Exception Details") || !strings.Contains(output, "current detail") {
		t.Fatalf("output = %q", stdout.String())
	}
	if strings.Contains(output, "other detail") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestRunExternalCommandUsesPtyWhenStdoutIsTerminal(t *testing.T) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Skipf("open pty: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()

	stdout := &terminalCapture{file: tty}
	err = runExternalCommand(
		testShell(t),
		[]string{"-c", "if [ -t 1 ]; then printf tty; else printf pipe; fi"},
		"",
		os.Environ(),
		nil,
		stdout,
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "tty" {
		t.Fatalf("stdout = %q, want tty", got)
	}
}

func TestRunExternalCommandWithPtyStdoutReturnsExitError(t *testing.T) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Skipf("open pty: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()

	stdout := &terminalCapture{file: tty}
	err = runExternalCommand(
		testShell(t),
		[]string{"-c", "printf before-exit; exit 7"},
		"",
		os.Environ(),
		nil,
		stdout,
		io.Discard,
	)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T %[1]v, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("exit code = %d, want 7", exitErr.ExitCode())
	}
	if got := stdout.String(); got != "before-exit" {
		t.Fatalf("stdout = %q, want before-exit", got)
	}
}

func testEvalOptions(t *testing.T, calls *[]recordedCommand, stderr io.Writer) EvalOptions {
	t.Helper()
	return EvalOptions{
		Agent:          "oracle",
		JobsDir:        DefaultEvalJobsDir,
		RunmeBin:       "/bin/runme",
		CommandRun:     recordCommand(calls),
		LookPath:       fakeLookPath,
		Executable:     func() (string, error) { return "/bin/current-runme", nil },
		Stdout:         io.Discard,
		Stderr:         stderr,
		Preflight:      true,
		RunmeHarborBin: "",
	}
}

type terminalCapture struct {
	bytes.Buffer
	file *os.File
}

func (w *terminalCapture) StdoutFile() *os.File {
	return w.file
}

func testShell(t *testing.T) string {
	t.Helper()
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found")
	}
	return shell
}

func recordCommand(calls *[]recordedCommand) CommandRunFunc {
	return func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
		*calls = append(*calls, recordedCommand{
			name:       name,
			args:       append([]string(nil), args...),
			workingDir: workingDir,
			env:        append([]string(nil), env...),
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

func defaultJobsDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path, err := relativePathFrom(cwd, filepath.Join(defaultEvalBaseDir(cwd), DefaultEvalJobsDir))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func defaultDatasetPath(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path, err := relativePathFrom(cwd, filepath.Join(defaultEvalBaseDir(cwd), DefaultEvalDatasetPath))
	if err != nil {
		t.Fatal(err)
	}
	return path
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

func makeHarborDockerDataset(t *testing.T, workspace string, remoteWorkdir string) (string, string, string) {
	t.Helper()
	dataset := filepath.Join(workspace, "evals", "tasks")
	task := filepath.Join(dataset, "example-task")
	workdir := filepath.Join(workspace, "source", "workdir")
	target := filepath.Join(task, "environment", "workdir")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(task, "environment"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "schema_version = \"1.1\"\n\n[environment]\nworkdir = \"" + remoteWorkdir + "\"\n"
	if err := os.WriteFile(filepath.Join(task, "task.toml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return dataset, workdir, target
}

func writeEvalException(t *testing.T, jobsDir, relativePath, content string) {
	t.Helper()
	path := filepath.Join(jobsDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
