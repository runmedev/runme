package harbor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type promoteJobOptions struct {
	jobsDir       string
	job           string
	latest        bool
	includeOracle bool
	allowErrors   bool
}

type promoteJobResult struct {
	Path         string
	StartedAt    time.Time
	UpdatedAt    time.Time
	FinishedAt   time.Time
	TotalTrials  int
	Stats        promoteJobStats
	RawTimestamp string
}

type promoteJobStats struct {
	CompletedTrials int                         `json:"n_completed_trials"`
	ErroredTrials   int                         `json:"n_errored_trials"`
	RunningTrials   int                         `json:"n_running_trials"`
	PendingTrials   int                         `json:"n_pending_trials"`
	CancelledTrials int                         `json:"n_cancelled_trials"`
	Retries         int                         `json:"n_retries"`
	Evals           map[string]promoteEvalStats `json:"evals"`
	InputTokens     *int                        `json:"n_input_tokens"`
	CacheTokens     *int                        `json:"n_cache_tokens"`
	OutputTokens    *int                        `json:"n_output_tokens"`
	CostUSD         *float64                    `json:"cost_usd"`
}

type promoteEvalStats struct {
	Trials  int                      `json:"n_trials"`
	Errors  int                      `json:"n_errors"`
	Metrics []map[string]interface{} `json:"metrics"`
}

type promoteJobConfig struct {
	Datasets    []promoteDatasetConfig `json:"datasets"`
	Agents      []promoteAgentConfig   `json:"agents"`
	Environment promoteEnvConfig       `json:"environment"`
}

type promoteDatasetConfig struct {
	Path      string   `json:"path"`
	Name      string   `json:"name"`
	TaskNames []string `json:"task_names"`
}

type promoteAgentConfig struct {
	Name       string `json:"name"`
	ImportPath string `json:"import_path"`
	ModelName  string `json:"model_name"`
}

type promoteEnvConfig struct {
	Type       string `json:"type"`
	ImportPath string `json:"import_path"`
}

func resolvePromoteJob(opts promoteJobOptions) (jobsRoot, jobDir, selection string, err error) {
	paths, err := resolveEvalViewPaths(opts.jobsDir)
	if err != nil {
		return "", "", "", err
	}
	jobsRoot = paths.jobsDir

	if opts.latest {
		jobDir, selection, err = latestPromoteJob(jobsRoot, promoteJobPolicy{
			includeOracle: opts.includeOracle,
			allowErrors:   opts.allowErrors,
		})
		if err != nil {
			return "", "", "", err
		}
		return jobsRoot, jobDir, selection, nil
	}

	invocationCwd, err := os.Getwd()
	if err != nil {
		return "", "", "", err
	}
	jobDir = opts.job
	if !filepath.IsAbs(jobDir) {
		jobDir = filepath.Join(invocationCwd, jobDir)
	}
	jobDir = cleanExistingPath(jobDir)
	if err := validatePromoteJobDir(jobsRoot, jobDir); err != nil {
		return "", "", "", err
	}
	return jobsRoot, jobDir, "explicit --job", nil
}

type promoteJobPolicy struct {
	includeOracle bool
	allowErrors   bool
}

func latestPromoteJob(jobsRoot string, policy promoteJobPolicy) (string, string, error) {
	entries, err := os.ReadDir(jobsRoot)
	if err != nil {
		return "", "", err
	}
	var candidates []promoteJobCandidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(jobsRoot, entry.Name())
		result, err := readPromoteJobResult(dir)
		if err != nil {
			continue
		}
		config, err := readPromoteJobConfig(dir)
		if err != nil {
			continue
		}
		if err := validatePromoteJobPolicy(result, config, policy); err != nil {
			continue
		}
		timestamp := resultTimestamp(result)
		info, _ := entry.Info()
		candidates = append(candidates, promoteJobCandidate{
			dir:       dir,
			name:      entry.Name(),
			timestamp: timestamp,
			modTime:   info.ModTime(),
		})
	}
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("no promotable eval jobs found under %s; use --include-oracle for oracle-only jobs or --allow-errors for errored jobs", jobsRoot)
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if !left.timestamp.IsZero() || !right.timestamp.IsZero() {
			if !left.timestamp.Equal(right.timestamp) {
				return left.timestamp.After(right.timestamp)
			}
		}
		if left.name != right.name {
			return left.name > right.name
		}
		return left.modTime.After(right.modTime)
	})
	return candidates[0].dir, "latest job under --jobs-dir", nil
}

func validatePromoteJobPolicy(result promoteJobResult, config promoteJobConfig, policy promoteJobPolicy) error {
	if reason := incompletePromoteJobReason(result); reason != "" {
		return fmt.Errorf("eval job is incomplete: %s", reason)
	}
	if result.Stats.ErroredTrials > 0 && !policy.allowErrors {
		return fmt.Errorf("eval job has %d errored trial(s); use --allow-errors to promote it anyway", result.Stats.ErroredTrials)
	}
	if isOracleOnlyPromoteJob(config, result) && !policy.includeOracle {
		return fmt.Errorf("eval job only used Harbor's oracle agent; use --include-oracle to promote it anyway")
	}
	return nil
}

func newerPromotableJobWarning(jobsRoot, selectedJobDir string, selectedResult promoteJobResult, policy promoteJobPolicy) string {
	entries, err := os.ReadDir(jobsRoot)
	if err != nil {
		return ""
	}
	selectedJobDir = cleanExistingPath(selectedJobDir)
	selectedName := filepath.Base(selectedJobDir)
	selectedTimestamp := resultTimestamp(selectedResult)
	var foundNewer bool
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := cleanExistingPath(filepath.Join(jobsRoot, entry.Name()))
		if dir == selectedJobDir {
			continue
		}
		result, err := readPromoteJobResult(dir)
		if err != nil {
			continue
		}
		if !isPromoteJobNewer(result, entry.Name(), selectedTimestamp, selectedName) {
			continue
		}
		foundNewer = true
		config, err := readPromoteJobConfig(dir)
		if err != nil {
			continue
		}
		if validatePromoteJobPolicy(result, config, policy) == nil {
			return ""
		}
	}
	if !foundNewer {
		return ""
	}
	return "no newer complete promotable eval job found"
}

func isPromoteJobNewer(result promoteJobResult, name string, selectedTimestamp time.Time, selectedName string) bool {
	timestamp := resultTimestamp(result)
	if !timestamp.IsZero() || !selectedTimestamp.IsZero() {
		if !timestamp.Equal(selectedTimestamp) {
			return timestamp.After(selectedTimestamp)
		}
	}
	return name > selectedName
}

func incompletePromoteJobReason(result promoteJobResult) string {
	var reasons []string
	if strings.TrimSpace(result.RawTimestamp) == "" {
		reasons = append(reasons, "finished_at is missing")
	} else if result.FinishedAt.IsZero() {
		reasons = append(reasons, "finished_at is invalid")
	}
	if result.TotalTrials <= 0 {
		reasons = append(reasons, fmt.Sprintf("total trials is %d", result.TotalTrials))
	}
	stats := result.Stats
	if stats.RunningTrials != 0 {
		reasons = append(reasons, fmt.Sprintf("%d running", stats.RunningTrials))
	}
	if stats.PendingTrials != 0 {
		reasons = append(reasons, fmt.Sprintf("%d pending", stats.PendingTrials))
	}
	if stats.CancelledTrials != 0 {
		reasons = append(reasons, fmt.Sprintf("%d cancelled", stats.CancelledTrials))
	}
	if stats.CompletedTrials+stats.ErroredTrials != result.TotalTrials {
		reasons = append(reasons, fmt.Sprintf(
			"%d/%d completed, %d errored",
			stats.CompletedTrials,
			result.TotalTrials,
			stats.ErroredTrials,
		))
	}
	return strings.Join(reasons, ", ")
}

func isOracleOnlyPromoteJob(config promoteJobConfig, result promoteJobResult) bool {
	var agents []string
	for _, agent := range config.Agents {
		if agent.Name != "" {
			agents = append(agents, agent.Name)
			continue
		}
		if agent.ImportPath != "" {
			agents = append(agents, agent.ImportPath)
		}
	}
	if len(agents) == 0 {
		for key := range result.Stats.Evals {
			agent, _, _ := strings.Cut(key, "__")
			if agent != "" {
				agents = append(agents, agent)
			}
		}
	}
	if len(agents) == 0 {
		return false
	}
	for _, agent := range agents {
		if strings.TrimSpace(agent) != "oracle" {
			return false
		}
	}
	return true
}

type promoteJobCandidate struct {
	dir       string
	name      string
	timestamp time.Time
	modTime   time.Time
}

func validatePromoteJobDir(jobsRoot, jobDir string) error {
	if relativePathUnder(jobsRoot, jobDir) == "" {
		return fmt.Errorf("eval job directory %s is outside jobs directory %s", jobDir, jobsRoot)
	}
	info, err := os.Stat(jobDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("eval job directory does not exist: %s", jobDir)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("eval job path is not a directory: %s", jobDir)
	}
	resultPath := filepath.Join(jobDir, "result.json")
	if _, err := os.Stat(resultPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("eval job result does not exist: %s", resultPath)
		}
		return err
	}
	return nil
}

func readPromoteJobResult(jobDir string) (promoteJobResult, error) {
	resultPath := filepath.Join(jobDir, "result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return promoteJobResult{}, err
	}
	var raw struct {
		StartedAt   string          `json:"started_at"`
		UpdatedAt   string          `json:"updated_at"`
		FinishedAt  string          `json:"finished_at"`
		TotalTrials int             `json:"n_total_trials"`
		Stats       promoteJobStats `json:"stats"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return promoteJobResult{}, err
	}
	return promoteJobResult{
		Path:         resultPath,
		StartedAt:    parsePromoteTime(raw.StartedAt),
		UpdatedAt:    parsePromoteTime(raw.UpdatedAt),
		FinishedAt:   parsePromoteTime(raw.FinishedAt),
		TotalTrials:  raw.TotalTrials,
		Stats:        raw.Stats,
		RawTimestamp: raw.FinishedAt,
	}, nil
}

func readPromoteJobConfig(jobDir string) (promoteJobConfig, error) {
	data, err := os.ReadFile(filepath.Join(jobDir, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return promoteJobConfig{}, nil
		}
		return promoteJobConfig{}, err
	}
	var config promoteJobConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return promoteJobConfig{}, err
	}
	return config, nil
}

func resultTimestamp(result promoteJobResult) time.Time {
	if !result.FinishedAt.IsZero() {
		return result.FinishedAt
	}
	if !result.UpdatedAt.IsZero() {
		return result.UpdatedAt
	}
	return result.StartedAt
}

func parsePromoteTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func stalePromoteWarning(result promoteJobResult, latestStagedMod time.Time) string {
	timestamp := resultTimestamp(result)
	if timestamp.IsZero() {
		return "eval job timestamp could not be determined"
	}
	if !latestStagedMod.IsZero() && timestamp.Before(latestStagedMod) {
		return fmt.Sprintf("eval job finished at %s before staged changes modified at %s", timestamp.Format(time.RFC3339), latestStagedMod.Format(time.RFC3339))
	}
	return ""
}

func displayList(values []string) string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, value)
		}
	}
	if len(filtered) == 0 {
		return "unknown"
	}
	return strings.Join(filtered, ", ")
}
