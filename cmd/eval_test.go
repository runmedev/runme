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

	"github.com/go-git/go-git/v5"
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
			"--jobs-dir", defaultJobsDir(t),
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
	if err := os.MkdirAll(defaultEvalDatasetPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, defaultEvalDatasetPath),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalDefaultsDatasetPathWithPassthrough(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(defaultEvalDatasetPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, []string{"--model", "haiku"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, defaultEvalDatasetPath),
		"--agent", "oracle",
		"--jobs-dir", defaultJobsDir(t),
		"--",
		"--model", "haiku",
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalDefaultsUseGitRoot(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, defaultEvalDatasetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		filepath.Join(repoRoot, defaultEvalDatasetPath),
		"--agent", "oracle",
		"--jobs-dir", filepath.Join(repoRoot, defaultEvalJobsDir),
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalExplicitDatasetUsesCwdAndDefaultJobsUseGitRoot(t *testing.T) {
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

	err := runEval(opts, []string{"./custom-dataset"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		dataset,
		"--agent", "oracle",
		"--jobs-dir", filepath.Join(repoRoot, defaultEvalJobsDir),
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalExplicitJobsDirIsUnchanged(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, defaultEvalDatasetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.jobsDir = "custom/jobs"
	opts.jobsDirExplicit = true

	err := runEval(opts, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		filepath.Join(repoRoot, defaultEvalDatasetPath),
		"--agent", "oracle",
		"--jobs-dir", "custom/jobs",
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
			opts.agent = tt.agent
			opts.taskDir = "simple-agent"
			opts.jobsDir = "jobs"
			opts.jobsDirExplicit = true
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
				"--task-dir", "simple-agent",
				"-y",
			}
			if !reflect.DeepEqual(calls[1].args, want) {
				t.Fatalf("args = %#v, want %#v", calls[1].args, want)
			}
		})
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
		"--jobs-dir", defaultJobsDir(t),
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
		"--jobs-dir", defaultJobsDir(t),
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

func TestRunEvalDelegatesRunmeEnvAlias(t *testing.T) {
	path := t.TempDir()
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.env = "runme"

	err := runEval(opts, []string{path})
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
	opts.env = "docker"
	opts.agent = "codex"

	err := runEval(opts, []string{path})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"run",
		mustAbs(t, path),
		"--agent", "codex",
		"--jobs-dir", defaultJobsDir(t),
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
	opts.env = "podman"

	err := runEval(opts, []string{dataset})
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

			err := runEval(opts, append([]string{path}, tt.passthrough...))
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
	opts.runmeHarbor = flagHarbor
	opts.runmeBin = "/flag/runme"
	opts.runmeArgs = []string{"--chdir", "/tmp/with space", "--label=can't-$expand`this`\\path"}

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
	if got := envValue(calls[1].env, "RUNME_ARGS"); got != "--chdir '/tmp/with space' '--label=can'\\''t-$expand`this`\\path'" {
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
		"--jobs-dir", defaultJobsDir(t),
	}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[1].args, want)
	}
}

func TestRunEvalMissingPath(t *testing.T) {
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, []string{filepath.Join(t.TempDir(), "missing")})
	if err == nil || !strings.Contains(err.Error(), "dataset path does not exist") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunEvalMissingDefaultPath(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)

	err := runEval(opts, nil)
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
		jobsDir:     defaultEvalJobsDir,
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

func defaultJobsDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(defaultEvalBaseDir(cwd), defaultEvalJobsDir)
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
