package harbor

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvalPromoterDryRunDoesNotRequireStagedChanges(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	var stdout bytes.Buffer
	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Latest:  true,
		DryRun:  true,
		Stdout:  &stdout,
		Git:     gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if gitClient.stagedCalled {
		t.Fatal("StagedFiles was called during dry-run")
	}
	if got := stdout.String(); !strings.Contains(got, "Selected eval job: jobs/2026-06-25__10-00-00") {
		t.Fatalf("stdout = %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "Selection: latest job under jobs") {
		t.Fatalf("stdout = %q", got)
	}
}

func TestEvalPromoterWarnsWhenNewerJobsAreNotPromotable(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	selectedJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	incompleteJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, selectedJob, "2026-06-25T09:00:00Z")
	writeIncompletePromoteJob(t, incompleteJob, "2026-06-25T10:00:00Z", 0, 1, 0)

	var stderr bytes.Buffer
	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Latest:  true,
		DryRun:  true,
		Stderr:  &stderr,
		Git:     gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if got := stderr.String(); !strings.Contains(got, "warning: no newer complete promotable eval job found under jobs") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestEvalPromoterDoesNotWarnWhenNewerPromotableJobExists(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	selectedJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	newerJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, selectedJob, "2026-06-25T09:00:00Z")
	writePromoteJob(t, newerJob, "2026-06-25T10:00:00Z")

	var stderr bytes.Buffer
	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Job:     selectedJob,
		DryRun:  true,
		Stderr:  &stderr,
		Git:     gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if got := stderr.String(); strings.Contains(got, "no newer complete promotable eval job") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestEvalPromoterSuggestsEvidenceOnlyWhenNothingIsStaged(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Latest:  true,
		Git:     gitClient,
	})

	err := promoter.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "use --evidence-only") {
		t.Fatalf("error = %q", err.Error())
	}
	if gitClient.addCalled || gitClient.commitCalled {
		t.Fatal("expected no add or commit")
	}
}

func TestEvalPromoterEvidenceOnlyCommitsJobWithoutStagedChanges(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir:      jobsRoot,
		Latest:       true,
		EvidenceOnly: true,
		Git:          gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if !gitClient.addCalled || !gitClient.commitCalled {
		t.Fatal("expected add and commit")
	}
	if !strings.Contains(gitClient.message, "Promotion-Mode: eval-evidence-only") {
		t.Fatalf("commit message missing evidence-only mode:\n%s", gitClient.message)
	}
}

func TestEvalPromoterEvidenceOnlyRejectsStagedChanges(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	gitClient := &recordingPromoteGit{
		root:   tmp,
		staged: []string{"main.go"},
	}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir:      jobsRoot,
		Latest:       true,
		EvidenceOnly: true,
		Git:          gitClient,
	})

	err := promoter.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--evidence-only cannot be used with staged changes") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestEvalPromoterExplicitJobRejectsOracleOnlyJob(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writeOraclePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Job:     jobDir,
		DryRun:  true,
		Git:     gitClient,
	})

	err := promoter.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "use --include-oracle") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestEvalPromoterExplicitJobRejectsErroredJob(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writeErroredPromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Job:     jobDir,
		DryRun:  true,
		Git:     gitClient,
	})

	err := promoter.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "use --allow-errors") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestEvalPromoterExplicitJobRejectsIncompleteJob(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writeIncompletePromoteJob(t, jobDir, "2026-06-25T10:00:00Z", 0, 1, 0)

	gitClient := &recordingPromoteGit{root: tmp}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir:     jobsRoot,
		Job:         jobDir,
		DryRun:      true,
		AllowErrors: true,
		Git:         gitClient,
	})

	err := promoter.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "eval job is incomplete:") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "0/1 completed, 0 errored") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "1 pending") {
		t.Fatalf("error = %q", err.Error())
	}
}

type recordingPromoteGit struct {
	root         string
	staged       []string
	stagedCalled bool
	addCalled    bool
	commitCalled bool
	message      string
}

func (g *recordingPromoteGit) StagedFiles() ([]string, error) {
	g.stagedCalled = true
	return append([]string(nil), g.staged...), nil
}

func (g *recordingPromoteGit) UnstagedFilesTouching([]string) ([]string, error) {
	return nil, nil
}

func (g *recordingPromoteGit) LatestModTime([]string) (time.Time, error) {
	return time.Time{}, nil
}

func (g *recordingPromoteGit) AddJobDir(string) error {
	g.addCalled = true
	return nil
}

func (g *recordingPromoteGit) Commit(message string) (string, error) {
	g.commitCalled = true
	g.message = message
	return "commit", nil
}

func (g *recordingPromoteGit) Rel(path string) (string, error) {
	root := cleanExistingPath(g.root)
	path = cleanExistingPath(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is outside git root %s", path, root)
	}
	return filepath.ToSlash(rel), nil
}
