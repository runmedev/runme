package harbor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestEvalComparerComparesTrackedBaseWithLatestLocalCandidate(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	candidateDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithMean(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithMean(t, candidateDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.75)
	t.Chdir(tmp)

	baseJob := localCompareJob(t, tmp, baseDir, "tracked in test")
	var stdout bytes.Buffer
	err := NewEvalComparer(EvalCompareOptions{
		JobsDir: jobsRoot,
		Stdout:  &stdout,
		Git: fakeCompareGit{
			root: cleanExistingPath(tmp),
			jobs: []compareJob{baseJob},
		},
	}).Run(nil)
	if err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	for _, want := range []string{
		"Base:   jobs/2026-06-25__09-00-00  tracked in HEAD",
		"Latest: jobs/2026-06-25__10-00-00  local",
		"mean:      0.500 -> 0.750  +0.250",
		"Recommendation: candidate improved or held steady; promotion looks reasonable after normal review.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestEvalComparerJSONOutputIsSmallComparisonObject(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "old")
	candidateDir := filepath.Join(jobsRoot, "new")
	writePromoteJobWithMean(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithMean(t, candidateDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.75)
	t.Chdir(tmp)

	var stdout bytes.Buffer
	err := NewEvalComparer(EvalCompareOptions{
		JobsDir: jobsRoot,
		Format:  "json",
		Stdout:  &stdout,
		Git: fakeCompareGit{
			root: cleanExistingPath(tmp),
			jobs: []compareJob{localCompareJob(t, tmp, baseDir, "tracked in test")},
		},
	}).Run(nil)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if _, ok := parsed["base"]; !ok {
		t.Fatalf("json missing base: %#v", parsed)
	}
	if _, ok := parsed["candidate"]; !ok {
		t.Fatalf("json missing candidate: %#v", parsed)
	}
	if _, ok := parsed["stats"]; !ok {
		t.Fatalf("json missing stats: %#v", parsed)
	}
	if _, ok := parsed["recommendation"]; !ok {
		t.Fatalf("json missing recommendation: %#v", parsed)
	}
}

func TestBuildEvalComparisonReportsMetadataMismatches(t *testing.T) {
	base := compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"oracle__dataset": {Metrics: []map[string]interface{}{{"mean": 1.0}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "dataset-a", TaskNames: []string{"one"}}},
			Agents:   []promoteAgentConfig{{Name: "oracle", ModelName: "model-a"}},
		},
		Selection: "tracked",
	}
	candidate := base
	candidate.RelPath = ".runme/evals/jobs/candidate"
	candidate.ResultRel = ".runme/evals/jobs/candidate/result.json"
	candidate.Config.Datasets = []promoteDatasetConfig{{Path: "dataset-b", TaskNames: []string{"one"}}}

	comparison := buildEvalComparison(base, candidate, "HEAD")
	if len(comparison.MetadataDiffs) != 1 {
		t.Fatalf("metadata diffs = %#v, want one diff", comparison.MetadataDiffs)
	}
	if !strings.Contains(comparison.Recommendation, "metadata differs") {
		t.Fatalf("recommendation = %q", comparison.Recommendation)
	}
}

func TestBuildEvalComparisonRecommendsNoopForSameJob(t *testing.T) {
	job := compareJob{
		RelPath:   ".runme/evals/jobs/job",
		ResultRel: ".runme/evals/jobs/job/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-codex__dataset": {Metrics: []map[string]interface{}{{"mean": 1.0}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "dataset", TaskNames: []string{"one"}}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex", ModelName: "model"}},
		},
		Selection: "tracked",
	}

	comparison := buildEvalComparison(job, job, "HEAD")
	if comparison.Recommendation != "base and latest are the same eval job; nothing to compare." {
		t.Fatalf("recommendation = %q", comparison.Recommendation)
	}
}

func TestRenderEvalComparisonTextOmitsEmptyTasks(t *testing.T) {
	comparison := buildEvalComparison(compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats:       promoteJobStats{CompletedTrials: 1},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/tasks"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex"}},
		},
		Selection: "tracked",
	}, compareJob{
		RelPath:   ".runme/evals/jobs/candidate",
		ResultRel: ".runme/evals/jobs/candidate/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats:       promoteJobStats{CompletedTrials: 1},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/tasks"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex"}},
		},
		Selection: "local",
	}, "HEAD")

	var stdout bytes.Buffer
	if err := renderEvalComparisonText(&stdout, comparison); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "Tasks:") {
		t.Fatalf("output should not contain Tasks without explicit task names:\n%s", stdout.String())
	}
}

func TestEvalComparerReadsTrackedBaseFromGit(t *testing.T) {
	tmp := t.TempDir()
	repo, err := git.PlainInit(tmp, false)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	jobsRoot := filepath.Join(tmp, ".runme", "evals", "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	candidateDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithMean(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	if _, err := wt.Add(".runme/evals/jobs/2026-06-25__09-00-00/result.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add(".runme/evals/jobs/2026-06-25__09-00-00/config.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("baseline", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Runme Test",
			Email: "runme@example.com",
			When:  time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatal(err)
	}

	writePromoteJobWithMean(t, candidateDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.75)
	t.Chdir(tmp)

	var stdout bytes.Buffer
	if err := NewEvalComparer(EvalCompareOptions{
		Stdout: &stdout,
	}).Run(nil); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Base:   .runme/evals/jobs/2026-06-25__09-00-00  tracked in HEAD") {
		t.Fatalf("output missing tracked base:\n%s", output)
	}
	if !strings.Contains(output, "Latest: .runme/evals/jobs/2026-06-25__10-00-00  local") {
		t.Fatalf("output missing local candidate:\n%s", output)
	}
}

func TestEvalComparerLatestSkipsOracleOnlyJobsByDefault(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__08-00-00")
	agentDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	oracleDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithMean(t, baseDir, "2026-06-25T08:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithMean(t, agentDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithMeanAgent(t, oracleDir, "2026-06-25T10:00:00Z", "oracle", 1, 1, 0, 1.0)
	t.Chdir(tmp)

	var stdout bytes.Buffer
	if err := NewEvalComparer(EvalCompareOptions{
		JobsDir: jobsRoot,
		Stdout:  &stdout,
		Git: fakeCompareGit{
			root: cleanExistingPath(tmp),
			jobs: []compareJob{localCompareJob(t, tmp, baseDir, "tracked in test")},
		},
	}).Run(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Latest: jobs/2026-06-25__09-00-00  local") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestEvalComparerLatestCanIncludeOracleOnlyJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "2026-06-25__08-00-00")
	agentDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	oracleDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithMean(t, baseDir, "2026-06-25T08:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithMean(t, agentDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithMeanAgent(t, oracleDir, "2026-06-25T10:00:00Z", "oracle", 1, 1, 0, 1.0)
	t.Chdir(tmp)

	var stdout bytes.Buffer
	if err := NewEvalComparer(EvalCompareOptions{
		JobsDir:       jobsRoot,
		IncludeOracle: true,
		Stdout:        &stdout,
		Git: fakeCompareGit{
			root: cleanExistingPath(tmp),
			jobs: []compareJob{localCompareJob(t, tmp, baseDir, "tracked in test")},
		},
	}).Run(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Latest: jobs/2026-06-25__10-00-00  local") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestResolveCompareBaseSkipsIncompleteTrackedJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	completeDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	incompleteDir := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	writePromoteJobWithMean(t, completeDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithMean(t, incompleteDir, "2026-06-25T10:00:00Z", 1, 0, 0, 0.75)

	job, err := resolveCompareBase(compareBaseOptions{
		git: fakeCompareGit{
			root: cleanExistingPath(tmp),
			jobs: []compareJob{
				localCompareJob(t, tmp, incompleteDir, "latest tracked job under --base"),
				localCompareJob(t, tmp, completeDir, "latest tracked job under --base"),
			},
		},
		baseRef:  "HEAD",
		jobsRoot: jobsRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.RelPath != "jobs/2026-06-25__09-00-00" {
		t.Fatalf("base job = %q, want complete tracked job", job.RelPath)
	}
}

type fakeCompareGit struct {
	root string
	jobs []compareJob
}

func (g fakeCompareGit) Rel(path string) (string, error) {
	rel, err := filepath.Rel(cleanExistingPath(g.root), cleanExistingPath(path))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func (g fakeCompareGit) TrackedEvalJobs(string, string) ([]compareJob, error) {
	return append([]compareJob(nil), g.jobs...), nil
}

func localCompareJob(t *testing.T, root, jobDir, selection string) compareJob {
	t.Helper()
	root = cleanExistingPath(root)
	jobDir = cleanExistingPath(jobDir)
	result, err := readPromoteJobResult(jobDir)
	if err != nil {
		t.Fatal(err)
	}
	config, err := readPromoteJobConfig(jobDir)
	if err != nil {
		t.Fatal(err)
	}
	jobRel, err := filepath.Rel(root, jobDir)
	if err != nil {
		t.Fatal(err)
	}
	return compareJob{
		RelPath:   filepath.ToSlash(jobRel),
		ResultRel: filepath.ToSlash(filepath.Join(jobRel, "result.json")),
		Result:    result,
		Config:    config,
		Selection: selection,
	}
}

func writePromoteJobWithMean(t *testing.T, jobDir, finishedAt string, total, completed, errors int, mean float64) {
	t.Helper()
	writePromoteJobWithMeanAgent(t, jobDir, finishedAt, "runme-codex", total, completed, errors, mean)
}

func writePromoteJobWithMeanAgent(t *testing.T, jobDir, finishedAt, agent string, total, completed, errors int, mean float64) {
	t.Helper()
	writePromoteJob(t, jobDir, finishedAt)
	result := fmt.Sprintf(`{
		"started_at": %q,
		"updated_at": %q,
		"finished_at": %q,
		"n_total_trials": %d,
		"stats": {
			"n_completed_trials": %d,
			"n_errored_trials": %d,
			"evals": {
				"%s__dataset": {
					"n_trials": %d,
					"n_errors": %d,
					"metrics": [{"mean": %.3f}]
				}
			}
		}
	}`, finishedAt, finishedAt, finishedAt, total, completed, errors, agent, total, errors, mean)
	writeFile(t, filepath.Join(jobDir, "result.json"), result)
	config := `{
		"datasets": [{"path": "dataset", "task_names": ["task"]}],
		"agents": [{"name": "` + agent + `", "model_name": "model"}],
		"environment": {"type": "runme"}
	}`
	writeFile(t, filepath.Join(jobDir, "config.json"), config)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
