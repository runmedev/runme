package harbor

import (
	"strings"
	"testing"
)

func TestRenderPromoteCommitMessageUsesJobAndResultRollups(t *testing.T) {
	msg := renderPromoteCommitMessage(promoteMessageData{
		subject:    defaultPromoteSubject,
		jobPath:    ".runme/evals/jobs/job",
		resultPath: ".runme/evals/jobs/job/result.json",
		config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{
				Path:      "evals/tasks",
				TaskNames: []string{"one", "two"},
			}},
			Agents: []promoteAgentConfig{{
				Name:      "codex",
				ModelName: "gpt-5-codex",
			}},
			Environment: promoteEnvConfig{ImportPath: runmeEnvironmentImportPath},
		},
		result: promoteJobResult{
			TotalTrials: 2,
			Stats: promoteJobStats{
				CompletedTrials: 2,
				ErroredTrials:   0,
				Evals: map[string]promoteEvalStats{
					"codex__dataset": {
						Metrics: []map[string]interface{}{{"reward": 0.75}},
					},
				},
			},
		},
	})

	for _, want := range []string{
		"Promote changes verified by task eval",
		"Eval-Job: .runme/evals/jobs/job",
		"Eval-Result: .runme/evals/jobs/job/result.json",
		"Dataset: evals/tasks",
		"Tasks: one, two",
		"Agent: codex",
		"Model: gpt-5-codex",
		"Environment: " + runmeEnvironmentImportPath,
		"Job: completed=2/2, errors=0, evals=1",
		"Results:",
		"dataset: reward=0.750",
		"\nResults:\n dataset: reward=0.750\nJob: completed=2/2, errors=0, evals=1\n\nAgent: codex\n",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
	for _, unwanted := range []string{
		"Result: completed=",
		"Score:",
		"mean",
		"codex__dataset: reward=",
	} {
		if strings.Contains(msg, unwanted) {
			t.Fatalf("message should not contain %q:\n%s", unwanted, msg)
		}
	}
	assertPromoteMessageOrder(t, msg,
		"Dataset: evals/tasks",
		"Results:",
		"dataset: reward=0.750",
		"Job: completed=2/2, errors=0, evals=1",
		"Agent: codex",
		"Model: gpt-5-codex",
		"Environment: "+runmeEnvironmentImportPath,
	)
}

func TestRenderPromoteCommitMessageOmitsResultsWithoutEvals(t *testing.T) {
	msg := renderPromoteCommitMessage(promoteMessageData{
		subject:    defaultPromoteSubject,
		jobPath:    ".runme/evals/jobs/job",
		resultPath: ".runme/evals/jobs/job/result.json",
		config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/tasks"}},
			Agents: []promoteAgentConfig{{
				Name:      "runme-codex",
				ModelName: "gpt-5.4-mini",
			}},
		},
		result: promoteJobResult{
			TotalTrials: 2,
			Stats: promoteJobStats{
				CompletedTrials: 2,
				ErroredTrials:   0,
			},
		},
	})

	for _, want := range []string{
		"Job: completed=2/2, errors=0",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "Tasks:") {
		t.Fatalf("message should not contain Tasks without explicit task names:\n%s", msg)
	}
	if strings.Contains(msg, "Score:") {
		t.Fatalf("message should not contain Score without score metric:\n%s", msg)
	}
	if strings.Contains(msg, "Results:") {
		t.Fatalf("message should not contain Results without eval summaries:\n%s", msg)
	}
}

func TestRenderPromoteCommitMessageDefaultsMissingRewardToZero(t *testing.T) {
	msg := renderPromoteCommitMessage(promoteMessageData{
		subject:    defaultPromoteSubject,
		jobPath:    ".runme/evals/jobs/job",
		resultPath: ".runme/evals/jobs/job/result.json",
		config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/tasks"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex"}},
		},
		result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-codex__dataset": {
						Metrics: []map[string]interface{}{{
							"structure":   1.0,
							"correctness": 0.5,
						}},
					},
				},
			},
		},
	})

	for _, want := range []string{
		"Results:",
		"dataset: reward=0.000",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "Score:") || strings.Contains(msg, "mean") {
		t.Fatalf("message should use reward results, not score/mean:\n%s", msg)
	}
}

func assertPromoteMessageOrder(t *testing.T, msg string, values ...string) {
	t.Helper()
	previous := -1
	for _, value := range values {
		index := strings.Index(msg, value)
		if index == -1 {
			t.Fatalf("message missing %q:\n%s", value, msg)
		}
		if index <= previous {
			t.Fatalf("message has %q out of order:\n%s", value, msg)
		}
		previous = index
	}
}

func TestRenderPromoteCommitMessageMarksAmbiguousReward(t *testing.T) {
	msg := renderPromoteCommitMessage(promoteMessageData{
		subject:    defaultPromoteSubject,
		jobPath:    ".runme/evals/jobs/job",
		resultPath: ".runme/evals/jobs/job/result.json",
		config: promoteJobConfig{
			Datasets: []promoteDatasetConfig{{Path: "evals/tasks"}},
			Agents:   []promoteAgentConfig{{Name: "runme-codex"}},
		},
		result: promoteJobResult{
			TotalTrials: 1,
			Stats: promoteJobStats{
				CompletedTrials: 1,
				Evals: map[string]promoteEvalStats{
					"runme-codex__dataset": {
						Metrics: []map[string]interface{}{
							{"reward": 0.5},
							{"reward": 0.75},
						},
					},
				},
			},
		},
	})

	if !strings.Contains(msg, "dataset: reward=n/a (ambiguous)") {
		t.Fatalf("message should mark ambiguous rewards:\n%s", msg)
	}
	if strings.Contains(msg, "Score:") || strings.Contains(msg, "mean") {
		t.Fatalf("message should use reward results, not score/mean:\n%s", msg)
	}
}
