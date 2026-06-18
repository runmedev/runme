package harbor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDockerWorkdirStagerStagesWorkdir(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(workspace, ".gitignore"),
		[]byte("ignored.txt\n**/environment/workdir/\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	dataset, workdir, target := makeDockerWorkdirDataset(t, workspace, "/app/source/workdir")
	if err := os.WriteFile(filepath.Join(workdir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "ignored.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "stale.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	stager := newTestDockerWorkdirStager(t, workspace, &stderr)
	if err := stager.StageDataset(dataset); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(target, "keep.txt")); err != nil {
		t.Fatalf("staged file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "ignored.txt")); !os.IsNotExist(err) {
		t.Fatalf("ignored file stat err = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(target, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale file stat err = %v, want not exist", err)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want no warning", stderr.String())
	}
}

func TestDockerWorkdirStagerWarnsWhenStagedWorkdirIsNotIgnored(t *testing.T) {
	workspace := t.TempDir()
	dataset, workdir, _ := makeDockerWorkdirDataset(t, workspace, "/app/source/workdir")
	if err := os.WriteFile(filepath.Join(workdir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	stager := newTestDockerWorkdirStager(t, workspace, &stderr)
	if err := stager.StageDataset(dataset); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stderr.String(), "is not ignored by git") {
		t.Fatalf("stderr = %q, want git ignore warning", stderr.String())
	}
}

func TestDockerWorkdirStagerSkipsUnsupportedWorkdirs(t *testing.T) {
	for _, workdir := range []string{"", "relative/workdir", "/app", "/tmp/workdir"} {
		t.Run(workdir, func(t *testing.T) {
			workspace := t.TempDir()
			dataset, _, target := makeDockerWorkdirDataset(t, workspace, workdir)
			stager := newTestDockerWorkdirStager(t, workspace, nil)

			if err := stager.StageDataset(dataset); err != nil {
				t.Fatal(err)
			}

			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatalf("target stat err = %v, want not exist", err)
			}
		})
	}
}

func TestDockerWorkdirStagerStagesSymlinkAsRealFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".gitignore"), []byte("**/environment/workdir/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dataset, workdir, target := makeDockerWorkdirDataset(t, workspace, "/app/source/workdir")
	shared := filepath.Join(workspace, "source", "shared.txt")
	if err := os.WriteFile(shared, []byte("shared"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(shared, filepath.Join(workdir, "linked.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	stager := newTestDockerWorkdirStager(t, workspace, nil)
	if err := stager.StageDataset(dataset); err != nil {
		t.Fatal(err)
	}

	staged := filepath.Join(target, "linked.txt")
	info, err := os.Lstat(staged)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("staged file is a symlink, want regular file")
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "shared" {
		t.Fatalf("staged data = %q, want shared", string(data))
	}
}

func newTestDockerWorkdirStager(t *testing.T, workspace string, stderr *bytes.Buffer) *DockerWorkdirStager {
	t.Helper()
	stager, err := NewDockerWorkdirStager(DockerWorkdirStagerOptions{
		WorkspaceRoot: workspace,
		Stderr:        stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	return stager
}

func makeDockerWorkdirDataset(t *testing.T, workspace string, remoteWorkdir string) (string, string, string) {
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
