package harbor

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEvalTaskNewerCreatesExpectedScaffold(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	tasksDir := filepath.Join(tmp, DefaultEvalDatasetPath)
	var stdout bytes.Buffer
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:    tasksDir,
		Description: "A useful task",
		Authors:     []string{"Alice <alice@example.com>", "Bob"},
		GitConfig:   noGitConfig,
		Stdout:      &stdout,
	})

	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}

	taskDir := filepath.Join(tasksDir, "my-task")
	for _, path := range []string{
		"README.md",
		"task.toml",
		"instruction.md",
		"environment/Dockerfile",
		"workdir/.gitignore",
		"workdir/.gitkeep",
		"tests/test.sh",
		"solution/solve.sh",
	} {
		if _, err := os.Stat(filepath.Join(taskDir, path)); err != nil {
			t.Fatalf("%s missing: %v", path, err)
		}
	}

	taskTOML := readFile(t, filepath.Join(taskDir, "task.toml"))
	for _, want := range []string{
		`schema_version = "1.3"`,
		`name = "runmedev/my-task"`,
		`description = "A useful task"`,
		`{ name = "Alice", email = "alice@example.com" }`,
		`{ name = "Bob" }`,
		`workdir = "/app/evals/tasks/my-task/workdir"`,
	} {
		if !strings.Contains(taskTOML, want) {
			t.Fatalf("task.toml missing %q:\n%s", want, taskTOML)
		}
	}
	dockerfile := readFile(t, filepath.Join(taskDir, "environment", "Dockerfile"))
	if !strings.Contains(dockerfile, "WORKDIR /app/evals/tasks/my-task/workdir") {
		t.Fatalf("Dockerfile = %s", dockerfile)
	}
	testScript := readFile(t, filepath.Join(taskDir, "tests", "test.sh"))
	for _, want := range []string{
		`RUNME_VERIFIER_DIR`,
		`RUNME_REWARD_PATH`,
		`test-stdout.txt`,
		`tee "$stdout_path"`,
		`{"reward": 0.0}`,
		`/app/evals/tasks/my-task/workdir`,
	} {
		if !strings.Contains(testScript, want) {
			t.Fatalf("tests/test.sh missing %q:\n%s", want, testScript)
		}
	}
	if !strings.Contains(stdout.String(), "Author: Alice <alice@example.com>, Bob") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "- Optional Docker setup (--env docker): "+filepath.Join(taskDir, "environment", "Dockerfile")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestEvalTaskNewerDefaultsTasksDirUnderProjectRoot(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		Org:       "runmedev",
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})

	if err := runner.Run([]string{"my-task"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(tmp, DefaultEvalDatasetPath, "my-task", "task.toml")); err != nil {
		t.Fatal(err)
	}
	taskTOML := readFile(t, filepath.Join(tmp, DefaultEvalDatasetPath, "my-task", "task.toml"))
	if !strings.Contains(taskTOML, `workdir = "/app/evals/tasks/my-task/workdir"`) {
		t.Fatalf("task.toml = %s", taskTOML)
	}
}

func TestEvalTaskNewerUsesRelativeTasksDirForContainerWorkdir(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  filepath.Join("examples", "harbor", "datasets", "custom"),
		Org:       "runmedev",
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})

	if err := runner.Run([]string{"my-task"}); err != nil {
		t.Fatal(err)
	}

	taskDir := filepath.Join(tmp, "examples", "harbor", "datasets", "custom", "my-task")
	want := `/app/examples/harbor/datasets/custom/my-task/workdir`
	taskTOML := readFile(t, filepath.Join(taskDir, "task.toml"))
	if !strings.Contains(taskTOML, `workdir = "`+want+`"`) {
		t.Fatalf("task.toml = %s", taskTOML)
	}
	dockerfile := readFile(t, filepath.Join(taskDir, "environment", "Dockerfile"))
	if !strings.Contains(dockerfile, "WORKDIR "+want) {
		t.Fatalf("Dockerfile = %s", dockerfile)
	}
}

func TestEvalTaskNewerUsesAbsoluteTasksDirInsideWorkspaceForContainerWorkdir(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	tasksDir := filepath.Join(tmp, "custom", "tasks")

	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  tasksDir,
		Org:       "runmedev",
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})

	if err := runner.Run([]string{"my-task"}); err != nil {
		t.Fatal(err)
	}

	taskTOML := readFile(t, filepath.Join(tasksDir, "my-task", "task.toml"))
	if !strings.Contains(taskTOML, `workdir = "/app/custom/tasks/my-task/workdir"`) {
		t.Fatalf("task.toml = %s", taskTOML)
	}
}

func TestEvalTaskNewerRejectsTasksDirOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	externalTasksDir := filepath.Join(t.TempDir(), "tasks")

	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  externalTasksDir,
		Org:       "runmedev",
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})

	err := runner.Run([]string{"my-task"})
	if err == nil || !strings.Contains(err.Error(), "must be under workspace root") {
		t.Fatalf("error = %v", err)
	}
}

func TestEvalTaskNewerDefaultsAuthorFromGitConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir: tmp,
		GitConfig: func(key string) (string, error) {
			switch key {
			case "user.name":
				return "Sebastian", nil
			case "user.email":
				return "sebastian@example.com", nil
			default:
				return "", os.ErrNotExist
			}
		},
		Stdout: &bytes.Buffer{},
	})

	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}

	taskTOML := readFile(t, filepath.Join(tmp, "my-task", "task.toml"))
	if !strings.Contains(taskTOML, `{ name = "Sebastian", email = "sebastian@example.com" }`) {
		t.Fatalf("task.toml = %s", taskTOML)
	}
}

func TestEvalTaskNewerExplicitAuthorSkipsGitConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	called := false
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir: tmp,
		Authors:  []string{"Alice"},
		GitConfig: func(string) (string, error) {
			called = true
			return "", nil
		},
		Stdout: &bytes.Buffer{},
	})

	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("GitConfig was called")
	}
}

func TestEvalTaskNewerRequiresOrgForBareName(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  tmp,
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})

	err := runner.Run([]string{"my-task"})
	if err == nil || !strings.Contains(err.Error(), "missing an organization") {
		t.Fatalf("error = %v", err)
	}
}

func TestEvalTaskNewerRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{
		"runmedev/../bad",
		"runmedev/",
		"/bad",
		"too/many/slashes",
	} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Chdir(tmp)
			runner := NewEvalTaskNewer(EvalTaskNewOptions{
				TasksDir:  tmp,
				GitConfig: noGitConfig,
				Stdout:    &bytes.Buffer{},
			})
			if err := runner.Run([]string{name}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestEvalTaskNewerExistingTargetRequiresForce(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.Mkdir(filepath.Join(tmp, "my-task"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  tmp,
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})

	err := runner.Run([]string{"runmedev/my-task"})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %v", err)
	}
}

func TestEvalTaskNewerForceOverwritesOwnedFilesAndKeepsUnknownFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	taskDir := filepath.Join(tmp, "my-task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "task.toml"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "notes.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  tmp,
		Force:     true,
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})
	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}

	if got := readFile(t, filepath.Join(taskDir, "notes.txt")); got != "keep" {
		t.Fatalf("notes.txt = %q", got)
	}
	if got := readFile(t, filepath.Join(taskDir, "task.toml")); got == "old" {
		t.Fatal("task.toml was not overwritten")
	}
}

func TestEvalTaskNewerNoSolutionSkipsSolution(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:   tmp,
		NoSolution: true,
		GitConfig:  noGitConfig,
		Stdout:     &bytes.Buffer{},
	})
	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "my-task", "solution", "solve.sh")); !os.IsNotExist(err) {
		t.Fatalf("solution exists or stat failed unexpectedly: %v", err)
	}
}

func TestEvalTaskNewerScriptsAreExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows filesystems do not report Unix executable bits")
	}

	tmp := t.TempDir()
	t.Chdir(tmp)
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  tmp,
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})
	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"tests/test.sh", "solution/solve.sh"} {
		info, err := os.Stat(filepath.Join(tmp, "my-task", path))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("%s is not executable: %s", path, info.Mode())
		}
	}
}

func TestEvalTaskNewerGeneratedVerifierWritesRewardAndStdout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("generated verifier script requires a Unix shell")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}

	tmp := t.TempDir()
	t.Chdir(tmp)
	runner := NewEvalTaskNewer(EvalTaskNewOptions{
		TasksDir:  tmp,
		GitConfig: noGitConfig,
		Stdout:    &bytes.Buffer{},
	})
	if err := runner.Run([]string{"runmedev/my-task"}); err != nil {
		t.Fatal(err)
	}

	verifierDir := filepath.Join(tmp, "verifier")
	rewardPath := filepath.Join(verifierDir, "reward.json")
	workdir := filepath.Join(tmp, "my-task", "workdir")
	cmd := exec.Command(bash, filepath.Join(tmp, "my-task", "tests", "test.sh"))
	cmd.Env = append(
		os.Environ(),
		"RUNME_VERIFIER_DIR="+verifierDir,
		"RUNME_REWARD_PATH="+rewardPath,
		"RUNME_TASK_NAME=my-task",
		"RUNME_TASK_WORKDIR="+workdir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated verifier failed: %v\n%s", err, output)
	}

	var rewards map[string]float64
	if err := json.Unmarshal([]byte(readFile(t, rewardPath)), &rewards); err != nil {
		t.Fatal(err)
	}
	if got := rewards["reward"]; got != 0.0 {
		t.Fatalf("reward = %v, want 0.0", got)
	}

	stdout := readFile(t, filepath.Join(verifierDir, "test-stdout.txt"))
	for _, want := range []string{
		"Verifier started for my-task",
		"Task workdir: " + workdir,
		"Reward written to: " + rewardPath,
		"Reward: 0.0",
		"Verifier completed successfully",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("test-stdout.txt missing %q:\n%s", want, stdout)
		}
	}
	if string(output) != stdout {
		t.Fatalf("stdout mirror mismatch\noutput:\n%s\nfile:\n%s", output, stdout)
	}
}

func noGitConfig(string) (string, error) {
	return "", os.ErrNotExist
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
