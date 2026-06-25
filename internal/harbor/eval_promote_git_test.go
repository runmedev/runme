package harbor

import (
	"os"
	"path/filepath"
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGoGitPromoteClientAddJobDirAddsCompactEvidence(t *testing.T) {
	repoRoot := t.TempDir()
	repo, err := git.PlainInit(repoRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte(".runme/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(".gitignore"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial", &git.CommitOptions{Author: testPromoteSignature()}); err != nil {
		t.Fatal(err)
	}

	jobDir := filepath.Join(repoRoot, ".runme", "evals", "jobs", "job")
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "result.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "job.log"), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	trialDir := filepath.Join(jobDir, "text-stats-reward__abc123")
	for _, dir := range []string{
		filepath.Join(trialDir, "artifacts"),
		filepath.Join(trialDir, "agent"),
		filepath.Join(trialDir, "verifier"),
		filepath.Join(trialDir, "workdir"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string]string{
		filepath.Join(trialDir, "trial.log"):                "trial",
		filepath.Join(trialDir, "artifacts", "result.txt"):  "artifact",
		filepath.Join(trialDir, "agent", "trajectory.json"): "{}",
		filepath.Join(trialDir, "verifier", "reward.json"):  "{}",
		filepath.Join(trialDir, "workdir", "sample.txt"):    "sample",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(repoRoot)

	client, err := newGoGitPromoteClient()
	if err != nil {
		t.Fatal(err)
	}
	files, err := client.JobFiles(jobDir, false)
	if err != nil {
		t.Fatal(err)
	}
	wantFiles := []string{
		".runme/evals/jobs/job/config.json",
		".runme/evals/jobs/job/job.log",
		".runme/evals/jobs/job/result.json",
		".runme/evals/jobs/job/text-stats-reward__abc123/verifier/reward.json",
	}
	if len(files) != len(wantFiles) {
		t.Fatalf("files = %#v, want %#v", files, wantFiles)
	}
	for i := range wantFiles {
		if files[i] != wantFiles[i] {
			t.Fatalf("files = %#v, want %#v", files, wantFiles)
		}
		if wtStatus, err := wt.Status(); err != nil {
			t.Fatal(err)
		} else if wtStatus.File(files[i]).Staging != git.Untracked {
			t.Fatalf("%s staging = %q, want untracked before add; status=%s", files[i], wtStatus.File(files[i]).Staging, wtStatus.String())
		}
	}
	if err := client.AddJobDir(jobDir, false); err != nil {
		t.Fatal(err)
	}
	status, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".runme/evals/jobs/job/result.json",
		".runme/evals/jobs/job/config.json",
		".runme/evals/jobs/job/job.log",
		".runme/evals/jobs/job/text-stats-reward__abc123/verifier/reward.json",
	} {
		if status.File(path).Staging != git.Added {
			t.Fatalf("%s staging = %q, want added; status=%s", path, status.File(path).Staging, status.String())
		}
	}
	for _, path := range []string{
		".runme/evals/jobs/job/text-stats-reward__abc123/trial.log",
		".runme/evals/jobs/job/text-stats-reward__abc123/artifacts/result.txt",
		".runme/evals/jobs/job/text-stats-reward__abc123/agent/trajectory.json",
		".runme/evals/jobs/job/text-stats-reward__abc123/workdir/sample.txt",
	} {
		if status.File(path).Staging != git.Untracked {
			t.Fatalf("%s staging = %q, want untracked; status=%s", path, status.File(path).Staging, status.String())
		}
	}
}

func TestGoGitPromoteClientAddJobDirAddsArtifactsExceptWorkdir(t *testing.T) {
	repoRoot := t.TempDir()
	repo, err := git.PlainInit(repoRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte(".runme/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(".gitignore"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial", &git.CommitOptions{Author: testPromoteSignature()}); err != nil {
		t.Fatal(err)
	}

	jobDir := filepath.Join(repoRoot, ".runme", "evals", "jobs", "job")
	trialDir := filepath.Join(jobDir, "text-stats-reward__abc123")
	for _, dir := range []string{
		filepath.Join(trialDir, "artifacts"),
		filepath.Join(trialDir, "agent"),
		filepath.Join(trialDir, "verifier"),
		filepath.Join(trialDir, "workdir"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string]string{
		filepath.Join(jobDir, "result.json"):                "{}",
		filepath.Join(trialDir, "trial.log"):                "trial",
		filepath.Join(trialDir, "artifacts", "result.txt"):  "artifact",
		filepath.Join(trialDir, "agent", "trajectory.json"): "{}",
		filepath.Join(trialDir, "verifier", "reward.json"):  "{}",
		filepath.Join(trialDir, "workdir", "sample.txt"):    "sample",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(repoRoot)

	client, err := newGoGitPromoteClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := client.AddJobDir(jobDir, true); err != nil {
		t.Fatal(err)
	}
	status, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".runme/evals/jobs/job/result.json",
		".runme/evals/jobs/job/text-stats-reward__abc123/trial.log",
		".runme/evals/jobs/job/text-stats-reward__abc123/artifacts/result.txt",
		".runme/evals/jobs/job/text-stats-reward__abc123/agent/trajectory.json",
		".runme/evals/jobs/job/text-stats-reward__abc123/verifier/reward.json",
	} {
		if status.File(path).Staging != git.Added {
			t.Fatalf("%s staging = %q, want added; status=%s", path, status.File(path).Staging, status.String())
		}
	}
	if path := ".runme/evals/jobs/job/text-stats-reward__abc123/workdir/sample.txt"; status.File(path).Staging != git.Untracked {
		t.Fatalf("%s staging = %q, want untracked; status=%s", path, status.File(path).Staging, status.String())
	}
}

func TestGoGitPromoteClientStagedFilesIgnoresUntrackedEvalJobs(t *testing.T) {
	repoRoot := t.TempDir()
	repo, err := git.PlainInit(repoRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte(".runme/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(".gitignore"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("main.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("initial", &git.CommitOptions{Author: testPromoteSignature()}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("main.go"); err != nil {
		t.Fatal(err)
	}

	for _, job := range []string{
		"2026-06-11__15-16-47",
		"2026-06-24__16-22-06",
		"2026-06-25__09-02-34",
	} {
		workdir := filepath.Join(repoRoot, ".runme", "evals", "jobs", job, "text-stats-reward__abc123", "workdir")
		if err := os.MkdirAll(workdir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workdir, ".gitignore"), []byte("*\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workdir, ".gitkeep"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workdir, "sample.txt"), []byte("sample"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(repoRoot)

	client, err := newGoGitPromoteClient()
	if err != nil {
		t.Fatal(err)
	}
	staged, err := client.StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 || staged[0] != "main.go" {
		t.Fatalf("staged = %#v, want only main.go", staged)
	}
	conflicts, err := client.UnstagedFilesTouching(staged)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("conflicts = %#v, want none", conflicts)
	}
}

func testPromoteSignature() *object.Signature {
	return &object.Signature{Name: "Runme Test", Email: "test@example.com"}
}
