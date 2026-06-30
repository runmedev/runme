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
	gitClient := &recordingPromoteGit{
		root: tmp,
		jobFiles: []string{
			"jobs/2026-06-25__10-00-00/result.json",
			"jobs/2026-06-25__10-00-00/trial/verifier/reward.json",
		},
	}
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
	if !gitClient.jobFilesCalled {
		t.Fatal("JobFiles was not called during dry-run")
	}
	if gitClient.addCalled {
		t.Fatal("AddJobDir was called during dry-run")
	}
	if got := stdout.String(); !strings.Contains(got, "Selected eval job: jobs/2026-06-25__10-00-00") {
		t.Fatalf("stdout = %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "Selection: latest job under jobs") {
		t.Fatalf("stdout = %q", got)
	}
	if got := stdout.String(); strings.HasSuffix(got, "\n\n") {
		t.Fatalf("stdout has trailing blank line: %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "Evidence mode: compact\nFiles to add:\n  jobs/2026-06-25__10-00-00/result.json\n  jobs/2026-06-25__10-00-00/trial/verifier/reward.json\n\nComparison: no tracked baseline found\n\nProposed commit message:\n\nPromote changes verified by task eval") {
		t.Fatalf("stdout = %q", got)
	}
	assertPlainProposedCommitMessage(t, stdout.String())
}

func TestEvalPromoterDryRunShowsArtifactsModeAndWorkdirExclusion(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	var stdout bytes.Buffer
	gitClient := &recordingPromoteGit{
		root:     tmp,
		jobFiles: []string{"jobs/2026-06-25__10-00-00/result.json"},
	}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir:   jobsRoot,
		Latest:    true,
		DryRun:    true,
		Artifacts: true,
		Stdout:    &stdout,
		Git:       gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if !gitClient.includeArtifacts {
		t.Fatal("includeArtifacts = false, want true")
	}
	if got := stdout.String(); !strings.Contains(got, "Evidence mode: artifacts\nFiles to add:\n  jobs/2026-06-25__10-00-00/result.json\nExcluded: */workdir/*\n\nComparison: no tracked baseline found\n\nProposed commit message:\n\nPromote changes verified by task eval") {
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
	if got := stderr.String(); !strings.Contains(got, "warning: using latest complete promotable eval job under jobs; newer eval jobs were skipped") {
		t.Fatalf("stderr = %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "warning: using latest complete promotable eval job under jobs; newer eval jobs were skipped\n\n") {
		t.Fatalf("stderr missing trailing blank line after warning: %q", got)
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
	if got := stderr.String(); strings.Contains(got, "newer eval jobs were skipped") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestEvalPromoterDryRunShowsCompareGateWithoutFailing(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithReward(t, jobDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.5)

	var stdout bytes.Buffer
	gitClient := &recordingPromoteGit{
		root:        tmp,
		trackedJobs: []evalJobRef{localCompareJob(t, tmp, baseDir, "tracked in test")},
		jobFiles:    []string{"jobs/2026-06-25__10-00-00/result.json"},
	}
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
	output := stdout.String()
	for _, want := range []string{
		"Comparison:",
		"dataset: reward 0.750 -> 0.500  -0.250",
		"Promotion gate: blocked",
		"Reason: candidate regressed; rerun, inspect job/task details, or pass --promote-anyway to promote anyway",
		"Proposed commit message:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	if gitClient.addCalled || gitClient.commitCalled {
		t.Fatal("dry-run should not add or commit")
	}
}

func TestEvalPromoterBlocksRegressedComparison(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithReward(t, jobDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.5)

	gitClient := &recordingPromoteGit{
		root:        tmp,
		staged:      []string{"main.go"},
		trackedJobs: []evalJobRef{localCompareJob(t, tmp, baseDir, "tracked in test")},
	}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir: jobsRoot,
		Latest:  true,
		Git:     gitClient,
	})

	err := promoter.Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "promotion blocked by eval comparison: candidate regressed") {
		t.Fatalf("error = %q", err.Error())
	}
	if gitClient.addCalled || gitClient.commitCalled {
		t.Fatal("blocked promotion should not add or commit")
	}
}

func TestEvalPromoterPromoteAnywayCommitsRegressedComparison(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithReward(t, jobDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.5)

	gitClient := &recordingPromoteGit{
		root:        tmp,
		staged:      []string{"main.go"},
		trackedJobs: []evalJobRef{localCompareJob(t, tmp, baseDir, "tracked in test")},
	}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir:       jobsRoot,
		Latest:        true,
		PromoteAnyway: true,
		Git:           gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if !gitClient.addCalled || !gitClient.commitCalled {
		t.Fatal("expected add and commit")
	}
}

func TestEvalPromoterEvidenceOnlyUsesCompareGate(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithReward(t, jobDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.5)

	gitClient := &recordingPromoteGit{
		root:        tmp,
		trackedJobs: []evalJobRef{localCompareJob(t, tmp, baseDir, "tracked in test")},
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
	if !strings.Contains(err.Error(), "promotion blocked by eval comparison: candidate regressed") {
		t.Fatalf("error = %q", err.Error())
	}
	if gitClient.addCalled || gitClient.commitCalled {
		t.Fatal("blocked evidence-only promotion should not add or commit")
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
	if gitClient.includeArtifacts {
		t.Fatal("includeArtifacts = true, want false")
	}
	if !strings.Contains(gitClient.message, "Promotion-Mode: eval-evidence-only") {
		t.Fatalf("commit message missing evidence-only mode:\n%s", gitClient.message)
	}
}

func TestEvalPromoterArtifactsCommitsFullJobEvidence(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T10:00:00Z")

	gitClient := &recordingPromoteGit{
		root:   tmp,
		staged: []string{"main.go"},
	}
	promoter := NewEvalPromoter(EvalPromoteOptions{
		JobsDir:   jobsRoot,
		Latest:    true,
		Artifacts: true,
		Git:       gitClient,
	})

	if err := promoter.Run(nil); err != nil {
		t.Fatal(err)
	}
	if !gitClient.addCalled || !gitClient.commitCalled {
		t.Fatal("expected add and commit")
	}
	if !gitClient.includeArtifacts {
		t.Fatal("includeArtifacts = false, want true")
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
	root             string
	staged           []string
	trackedJobs      []evalJobRef
	stagedCalled     bool
	addCalled        bool
	jobFilesCalled   bool
	commitCalled     bool
	includeArtifacts bool
	jobFiles         []string
	message          string
}

func assertPlainProposedCommitMessage(t *testing.T, output string) {
	t.Helper()
	parts := strings.SplitN(output, "Proposed commit message:\n\n", 2)
	if len(parts) != 2 {
		t.Fatalf("output missing proposed commit message:\n%s", output)
	}
	if strings.Contains(parts[1], "\x1b[") {
		t.Fatalf("proposed commit message contains ANSI formatting:\n%q", parts[1])
	}
}

func (g *recordingPromoteGit) TrackedEvalJobs(string, string) ([]evalJobRef, error) {
	return append([]evalJobRef(nil), g.trackedJobs...), nil
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

func (g *recordingPromoteGit) AddJobDir(_ string, includeArtifacts bool) error {
	g.addCalled = true
	g.includeArtifacts = includeArtifacts
	return nil
}

func (g *recordingPromoteGit) JobFiles(_ string, includeArtifacts bool) ([]string, error) {
	g.jobFilesCalled = true
	g.includeArtifacts = includeArtifacts
	return append([]string(nil), g.jobFiles...), nil
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
