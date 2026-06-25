package harbor

import (
	"strings"
	"testing"
)

func TestRenderPromoteCommitMessageUsesJobRollup(t *testing.T) {
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
						Metrics: []map[string]interface{}{{"mean": 0.75}},
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
		"Result: completed=2/2, errors=0, evals=1",
		"Score: mean=0.750",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
}

func TestRenderPromoteCommitMessageSeparatesResultFromScore(t *testing.T) {
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
				Evals: map[string]promoteEvalStats{
					"runme-codex__dataset": {},
				},
			},
		},
	})

	for _, want := range []string{
		"Result: completed=2/2, errors=0, evals=1",
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
}

func TestRenderPromoteCommitMessageIgnoresNamedMetricDimensions(t *testing.T) {
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

	if strings.Contains(msg, "Score:") {
		t.Fatalf("message should not contain Score for named metric dimensions:\n%s", msg)
	}
}

func TestRenderPromoteCommitMessageIgnoresRewardWithoutMean(t *testing.T) {
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
						Metrics: []map[string]interface{}{{"reward": 0.5}},
					},
				},
			},
		},
	})

	if strings.Contains(msg, "Score:") {
		t.Fatalf("message should not contain Score without mean:\n%s", msg)
	}
}
