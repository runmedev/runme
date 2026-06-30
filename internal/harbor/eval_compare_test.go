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
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithReward(t, candidateDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.75)
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
		"Job:",
		"Results:",
		"dataset: reward 0.500 -> 0.750  +0.250",
		"Recommendation: candidate improved or held steady; promotion looks reasonable after normal review.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "Score:") {
		t.Fatalf("output should use Job/Results instead of Score:\n%s", output)
	}
}

func TestEvalComparerJSONOutputIsSmallComparisonObject(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	baseDir := filepath.Join(jobsRoot, "old")
	candidateDir := filepath.Join(jobsRoot, "new")
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithReward(t, candidateDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.75)
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
	if _, ok := parsed["job"]; !ok {
		t.Fatalf("json missing job: %#v", parsed)
	}
	if _, ok := parsed["results"]; !ok {
		t.Fatalf("json missing results: %#v", parsed)
	}
	if _, ok := parsed["stats"]; ok {
		t.Fatalf("json should not keep legacy stats shape: %#v", parsed)
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
					"oracle__dataset": {Metrics: []map[string]interface{}{{"reward": 1.0}}},
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

func TestBuildEvalComparisonMatchesResultsAcrossDifferentAgents(t *testing.T) {
	comparison := buildEvalComparison(compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-codex__regression": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.5}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/regression"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex", ModelName: "gpt-5.5"}},
		},
		Selection: "tracked",
	}, compareJob{
		RelPath:   ".runme/evals/jobs/candidate",
		ResultRel: ".runme/evals/jobs/candidate/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-claude-code__regression": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.75}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/regression"}},
			Agents:   []promoteAgentConfig{{Name: "runme-claude-code", ModelName: "claude-opus-4-8"}},
		},
		Selection: "local",
	}, "HEAD")

	if len(comparison.Results.Comparisons) != 1 {
		t.Fatalf("comparisons = %#v, want one normalized match", comparison.Results.Comparisons)
	}
	result := comparison.Results.Comparisons[0]
	if result.Key != "regression" {
		t.Fatalf("result key = %q, want normalized regression", result.Key)
	}
	if result.BaseKey != "runme-codex__regression" || result.CandidateKey != "runme-claude-code__regression" {
		t.Fatalf("raw result keys = %q/%q", result.BaseKey, result.CandidateKey)
	}
	if len(comparison.Results.BaseOnly) != 0 || len(comparison.Results.CandidateOnly) != 0 {
		t.Fatalf("unexpected non-overlap rows: %#v", comparison.Results)
	}

	var stdout bytes.Buffer
	if err := renderEvalComparisonText(&stdout, comparison); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "regression: reward 0.500 -> 0.750  +0.250") {
		t.Fatalf("output missing normalized result row:\n%s", output)
	}
	if strings.Contains(output, "no matching eval results") ||
		strings.Contains(output, "base only:") ||
		strings.Contains(output, "latest only:") {
		t.Fatalf("output should not report non-overlap for normalized agent keys:\n%s", output)
	}
	if comparison.Recommendation != "metadata differs; review mismatches before promotion." {
		t.Fatalf("recommendation = %q", comparison.Recommendation)
	}
}

func TestBuildEvalComparisonTreatsMissingRewardAsZero(t *testing.T) {
	comparison := buildEvalComparison(compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-codex__regression": {Trials: 1, Metrics: []map[string]interface{}{{"source_quality": 1.0}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/regression"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex", ModelName: "gpt-5.5"}},
		},
		Selection: "tracked",
	}, compareJob{
		RelPath:   ".runme/evals/jobs/candidate",
		ResultRel: ".runme/evals/jobs/candidate/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-codex__regression": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.75}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/regression"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex", ModelName: "gpt-5.5"}},
		},
		Selection: "local",
	}, "HEAD")

	if len(comparison.Results.Comparisons) != 1 {
		t.Fatalf("comparisons = %#v, want one match", comparison.Results.Comparisons)
	}
	result := comparison.Results.Comparisons[0]
	if result.RewardStatus != "" {
		t.Fatalf("reward status = %q, want empty status", result.RewardStatus)
	}
	if got := displayValue(result.Reward.Base); got != "0.000" {
		t.Fatalf("base reward = %s, want 0.000", got)
	}
	if got := displayValue(result.Reward.Candidate); got != "0.750" {
		t.Fatalf("candidate reward = %s, want 0.750", got)
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
					"runme-codex__dataset": {Metrics: []map[string]interface{}{{"reward": 1.0}}},
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

func TestRenderEvalComparisonTextShowsOverlappingResultsFirst(t *testing.T) {
	comparison := buildEvalComparison(compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 2,
			Stats: promoteJobStats{
				CompletedTrials: 2,
				Evals: map[string]promoteEvalStats{
					"base-only": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.1}}},
					"shared":    {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.5}}},
				},
			},
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
			TotalTrials: 2,
			Stats: promoteJobStats{
				CompletedTrials: 2,
				Evals: map[string]promoteEvalStats{
					"candidate-only": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.9}}},
					"shared":         {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.75}}},
				},
			},
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
	output := stdout.String()
	sharedIndex := strings.Index(output, "shared: reward 0.500 -> 0.750  +0.250")
	baseOnlyIndex := strings.Index(output, "base only: base-only")
	latestOnlyIndex := strings.Index(output, "latest only: candidate-only")
	if sharedIndex < 0 || baseOnlyIndex < 0 || latestOnlyIndex < 0 {
		t.Fatalf("output missing expected result rows:\n%s", output)
	}
	if !(sharedIndex < baseOnlyIndex && baseOnlyIndex < latestOnlyIndex) {
		t.Fatalf("expected overlapping results before non-overlap rows:\n%s", output)
	}
	if !strings.Contains(output, "Recommendation: candidate improved or held steady") {
		t.Fatalf("partial overlap alone should not downgrade recommendation:\n%s", output)
	}
}

func TestRenderEvalComparisonTextReportsNoOverlappingResults(t *testing.T) {
	comparison := buildEvalComparison(compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"base-only": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.5}}},
				},
			},
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
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"candidate-only": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.75}}},
				},
			},
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
	output := stdout.String()
	for _, want := range []string{
		"no matching eval results",
		"base only: base-only",
		"latest only: candidate-only",
		"Recommendation: no matching eval results; compare job selection before promotion.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestBuildEvalComparisonRecommendsRegressionForMatchedResultRewardDrop(t *testing.T) {
	comparison := buildEvalComparison(compareJob{
		RelPath:   ".runme/evals/jobs/base",
		ResultRel: ".runme/evals/jobs/base/result.json",
		Result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"shared": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.75}}},
				},
			},
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
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"shared": {Trials: 1, Metrics: []map[string]interface{}{{"reward": 0.5}}},
				},
			},
		},
		Config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/tasks"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex"}},
		},
		Selection: "local",
	}, "HEAD")

	if !strings.Contains(comparison.Recommendation, "candidate regressed") {
		t.Fatalf("recommendation = %q", comparison.Recommendation)
	}
}

func TestEvalComparisonResultRewardDeltaStyle(t *testing.T) {
	for _, tc := range []struct {
		name  string
		delta interface{}
		want  string
	}{
		{name: "regressed", delta: -0.25, want: evalResultRegressedStyle},
		{name: "improved", delta: 0.25, want: evalResultImprovedStyle},
		{name: "unchanged", delta: 0.0, want: ""},
		{name: "missing", delta: nil, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := evalComparisonResult{
				Reward: evalComparisonDiff{Delta: tc.delta},
			}
			if got := result.rewardDeltaStyle(); got != tc.want {
				t.Fatalf("style = %q, want %q", got, tc.want)
			}
		})
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
	writePromoteJobWithReward(t, baseDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
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

	writePromoteJobWithReward(t, candidateDir, "2026-06-25T10:00:00Z", 1, 1, 0, 0.75)
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
	writePromoteJobWithReward(t, baseDir, "2026-06-25T08:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithReward(t, agentDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithRewardAgent(t, oracleDir, "2026-06-25T10:00:00Z", "oracle", 1, 1, 0, 1.0)
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
	writePromoteJobWithReward(t, baseDir, "2026-06-25T08:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithReward(t, agentDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.75)
	writePromoteJobWithRewardAgent(t, oracleDir, "2026-06-25T10:00:00Z", "oracle", 1, 1, 0, 1.0)
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
	writePromoteJobWithReward(t, completeDir, "2026-06-25T09:00:00Z", 1, 1, 0, 0.5)
	writePromoteJobWithReward(t, incompleteDir, "2026-06-25T10:00:00Z", 1, 0, 0, 0.75)

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

func writePromoteJobWithReward(t *testing.T, jobDir, finishedAt string, total, completed, errors int, reward float64) {
	t.Helper()
	writePromoteJobWithRewardAgent(t, jobDir, finishedAt, "runme-codex", total, completed, errors, reward)
}

func writePromoteJobWithRewardAgent(t *testing.T, jobDir, finishedAt, agent string, total, completed, errors int, reward float64) {
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
					"metrics": [{"reward": %.3f}]
				}
			}
		}
	}`, finishedAt, finishedAt, finishedAt, total, completed, errors, agent, total, errors, reward)
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
