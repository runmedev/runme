package harbor

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestResolvePromoteJobExplicitPathUnderJobsRoot(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	writePromoteJob(t, jobDir, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	gotJobsRoot, gotJobDir, selection, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: "jobs",
		job:     filepath.Join("jobs", "2026-06-25__09-00-00"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobsRoot != cleanExistingPath(jobsRoot) {
		t.Fatalf("jobsRoot = %q, want %q", gotJobsRoot, cleanExistingPath(jobsRoot))
	}
	if gotJobDir != cleanExistingPath(jobDir) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(jobDir))
	}
	if selection != "explicit --job" {
		t.Fatalf("selection = %q", selection)
	}
}

func TestResolvePromoteJobRejectsOutsideJobsRoot(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	jobDir := filepath.Join(tmp, "outside")
	writePromoteJob(t, jobDir, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	_, _, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: jobsRoot,
		job:     jobDir,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "outside jobs directory") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestResolvePromoteJobLatestUsesHarborTimestamps(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	oldJob := filepath.Join(jobsRoot, "z-name-but-older")
	newJob := filepath.Join(jobsRoot, "a-name-but-newer")
	writePromoteJob(t, oldJob, "2026-06-25T09:00:00Z")
	writePromoteJob(t, newJob, "2026-06-25T10:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, selection, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: "jobs",
		latest:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobDir != cleanExistingPath(newJob) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(newJob))
	}
	if selection != "latest job under --jobs-dir" {
		t.Fatalf("selection = %q", selection)
	}
}

func TestResolvePromoteJobLatestFallsBackToNameSort(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	writePromoteJob(t, filepath.Join(jobsRoot, "2026-06-25__09-00-00"), "2026-06-25T10:00:00Z")
	writePromoteJob(t, filepath.Join(jobsRoot, "2026-06-25__10-00-00"), "2026-06-25T10:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: "jobs",
		latest:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := cleanExistingPath(filepath.Join(jobsRoot, "2026-06-25__10-00-00"))
	if gotJobDir != want {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, want)
	}
}

func TestResolvePromoteJobLatestSkipsIncompleteJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	incompleteJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	completeJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	writeIncompletePromoteJob(t, incompleteJob, "2026-06-25T10:00:00Z", 0, 1, 0)
	writePromoteJob(t, completeJob, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: "jobs",
		latest:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobDir != cleanExistingPath(completeJob) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(completeJob))
	}
}

func TestValidatePromoteJobPolicyRejectsIncompleteJob(t *testing.T) {
	result := promoteJobResult{
		RawTimestamp: "2026-06-25T10:00:00Z",
		FinishedAt:   parsePromoteTime("2026-06-25T10:00:00Z"),
		TotalTrials:  1,
		Stats: promoteJobStats{
			PendingTrials: 1,
		},
	}

	err := validatePromoteJobPolicy(result, promoteJobConfig{}, promoteJobPolicy{
		allowErrors:   true,
		includeOracle: true,
	})
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

func TestValidatePromoteJobPolicyRejectsMissingFinishedAt(t *testing.T) {
	result := promoteJobResult{
		TotalTrials: 1,
		Stats: promoteJobStats{
			CompletedTrials: 1,
		},
	}

	err := validatePromoteJobPolicy(result, promoteJobConfig{}, promoteJobPolicy{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "finished_at is missing") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestResolvePromoteJobLatestSkipsOracleOnlyJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	oracleJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	agentJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	writeOraclePromoteJob(t, oracleJob, "2026-06-25T10:00:00Z")
	writePromoteJob(t, agentJob, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: "jobs",
		latest:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobDir != cleanExistingPath(agentJob) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(agentJob))
	}
}

func TestResolvePromoteJobLatestCanIncludeOracleOnlyJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	oracleJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	agentJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	writeOraclePromoteJob(t, oracleJob, "2026-06-25T10:00:00Z")
	writePromoteJob(t, agentJob, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir:       "jobs",
		latest:        true,
		includeOracle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobDir != cleanExistingPath(oracleJob) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(oracleJob))
	}
}

func TestResolvePromoteJobLatestSkipsErroredJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	erroredJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	passingJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	writeErroredPromoteJob(t, erroredJob, "2026-06-25T10:00:00Z")
	writePromoteJob(t, passingJob, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir: "jobs",
		latest:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobDir != cleanExistingPath(passingJob) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(passingJob))
	}
}

func TestResolvePromoteJobLatestCanAllowErroredJobs(t *testing.T) {
	tmp := t.TempDir()
	jobsRoot := filepath.Join(tmp, "jobs")
	erroredJob := filepath.Join(jobsRoot, "2026-06-25__10-00-00")
	passingJob := filepath.Join(jobsRoot, "2026-06-25__09-00-00")
	writeErroredPromoteJob(t, erroredJob, "2026-06-25T10:00:00Z")
	writePromoteJob(t, passingJob, "2026-06-25T09:00:00Z")
	t.Chdir(tmp)

	_, gotJobDir, _, err := resolvePromoteJob(promoteJobOptions{
		jobsDir:     "jobs",
		latest:      true,
		allowErrors: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotJobDir != cleanExistingPath(erroredJob) {
		t.Fatalf("jobDir = %q, want %q", gotJobDir, cleanExistingPath(erroredJob))
	}
}

func writePromoteJob(t *testing.T, jobDir, finishedAt string) {
	writePromoteJobWithAgent(t, jobDir, finishedAt, "codex", 0)
}

func writeOraclePromoteJob(t *testing.T, jobDir, finishedAt string) {
	writePromoteJobWithAgent(t, jobDir, finishedAt, "oracle", 0)
}

func writeErroredPromoteJob(t *testing.T, jobDir, finishedAt string) {
	writePromoteJobWithAgent(t, jobDir, finishedAt, "codex", 1)
}

func writePromoteJobWithAgent(t *testing.T, jobDir, finishedAt, agent string, errors int) {
	t.Helper()
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatal(err)
	}
	completed := 1
	if errors > 0 {
		completed = 0
	}
	result := `{
		"started_at": "` + finishedAt + `",
		"updated_at": "` + finishedAt + `",
		"finished_at": "` + finishedAt + `",
		"n_total_trials": 1,
		"stats": {
			"n_completed_trials": ` + strconv.Itoa(completed) + `,
			"n_errored_trials": ` + strconv.Itoa(errors) + `,
			"evals": {
				"` + agent + `__dataset": {
					"n_trials": 1,
					"n_errors": ` + strconv.Itoa(errors) + `,
					"metrics": [{"mean": 1.0}]
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(jobDir, "result.json"), []byte(result), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"agents":[{"name":"` + agent + `"}]}`
	if err := os.WriteFile(filepath.Join(jobDir, "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeIncompletePromoteJob(t *testing.T, jobDir, finishedAt string, running, pending, cancelled int) {
	t.Helper()
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatal(err)
	}
	result := `{
		"started_at": "` + finishedAt + `",
		"updated_at": "` + finishedAt + `",
		"finished_at": "` + finishedAt + `",
		"n_total_trials": 1,
		"stats": {
			"n_completed_trials": 0,
			"n_errored_trials": 0,
			"n_running_trials": ` + strconv.Itoa(running) + `,
			"n_pending_trials": ` + strconv.Itoa(pending) + `,
			"n_cancelled_trials": ` + strconv.Itoa(cancelled) + `,
			"evals": {}
		}
	}`
	if err := os.WriteFile(filepath.Join(jobDir, "result.json"), []byte(result), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "config.json"), []byte(`{"agents":[{"name":"codex"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
}
