package harbor

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type evalComparison struct {
	Base           evalComparisonJob     `json:"base"`
	Candidate      evalComparisonJob     `json:"candidate"`
	Metadata       evalComparisonMeta    `json:"metadata"`
	MetadataDiffs  []evalComparisonDiff  `json:"metadata_mismatches,omitempty"`
	Job            evalComparisonStats   `json:"job"`
	Results        evalComparisonResults `json:"results"`
	Recommendation string                `json:"recommendation"`
}

type evalComparisonJob struct {
	Path      string `json:"path"`
	Result    string `json:"result"`
	Ref       string `json:"ref,omitempty"`
	Selection string `json:"selection"`
	Timestamp string `json:"timestamp,omitempty"`
}

type evalComparisonMeta struct {
	Dataset     evalComparisonDiff `json:"dataset"`
	Tasks       evalComparisonDiff `json:"tasks"`
	Agent       evalComparisonDiff `json:"agent"`
	Model       evalComparisonDiff `json:"model"`
	Environment evalComparisonDiff `json:"environment"`
}

type evalComparisonStats struct {
	Completed evalComparisonDiff `json:"completed"`
	Errors    evalComparisonDiff `json:"errors"`
	Evals     evalComparisonDiff `json:"evals"`
	CostUSD   evalComparisonDiff `json:"cost_usd,omitempty"`
}

type evalComparisonResults struct {
	BaseOnly      []string               `json:"base_only,omitempty"`
	CandidateOnly []string               `json:"candidate_only,omitempty"`
	Comparisons   []evalComparisonResult `json:"comparisons"`
}

type evalComparisonResult struct {
	Key          string             `json:"key"`
	BaseKey      string             `json:"base_key,omitempty"`
	CandidateKey string             `json:"candidate_key,omitempty"`
	Reward       evalComparisonDiff `json:"reward"`
	RewardStatus string             `json:"reward_status,omitempty"`
	Errors       evalComparisonDiff `json:"errors"`
	Trials       evalComparisonDiff `json:"trials"`
}

type evalComparisonDiff struct {
	Base      interface{} `json:"base"`
	Candidate interface{} `json:"candidate"`
	Delta     interface{} `json:"delta,omitempty"`
}

func buildEvalComparison(base, candidate compareJob, baseRef string) evalComparison {
	baseSummary := promoteMessageData{config: base.Config, result: base.Result}
	candidateSummary := promoteMessageData{config: candidate.Config, result: candidate.Result}

	metadata := evalComparisonMeta{
		Dataset:     stringDiff(baseSummary.datasetSummary(), candidateSummary.datasetSummary()),
		Tasks:       stringDiff(baseSummary.taskSummary(), candidateSummary.taskSummary()),
		Agent:       stringDiff(baseSummary.agentSummary(), candidateSummary.agentSummary()),
		Model:       stringDiff(baseSummary.modelSummary(), candidateSummary.modelSummary()),
		Environment: stringDiff(baseSummary.environmentSummary(), candidateSummary.environmentSummary()),
	}
	job := compareJobStats(base.Result, candidate.Result)
	results := compareResults(base, candidate)
	mismatches := metadataMismatches(metadata)

	comparison := evalComparison{
		Base: evalComparisonJob{
			Path:      base.RelPath,
			Result:    base.ResultRel,
			Ref:       baseRef,
			Selection: base.Selection,
			Timestamp: formatCompareTimestamp(resultTimestamp(base.Result)),
		},
		Candidate: evalComparisonJob{
			Path:      candidate.RelPath,
			Result:    candidate.ResultRel,
			Selection: candidate.Selection,
			Timestamp: formatCompareTimestamp(resultTimestamp(candidate.Result)),
		},
		Metadata:      metadata,
		MetadataDiffs: mismatches,
		Job:           job,
		Results:       results,
	}
	comparison.Recommendation = compareRecommendation(comparison)
	return comparison
}

func renderEvalComparisonText(w io.Writer, comparison evalComparison) error {
	_, _ = fmt.Fprintf(w, "%s   %s  tracked in %s\n", evalOutputLabel(w, "Base:"), comparison.Base.Path, comparison.Base.Ref)
	_, _ = fmt.Fprintf(w, "%s %s  local\n\n", evalOutputLabel(w, "Latest:"), comparison.Candidate.Path)

	_, _ = fmt.Fprintf(w, "%s %s\n", evalOutputLabel(w, "Dataset:"), comparison.Metadata.Dataset.Candidate)
	if tasks := strings.TrimSpace(fmt.Sprint(comparison.Metadata.Tasks.Candidate)); tasks != "" {
		_, _ = fmt.Fprintf(w, "%s %s\n", evalOutputLabel(w, "Tasks:"), tasks)
	}
	_, _ = fmt.Fprintf(w, "%s %s\n", evalOutputLabel(w, "Agent:"), comparison.Metadata.Agent.Candidate)
	_, _ = fmt.Fprintf(w, "%s %s\n", evalOutputLabel(w, "Model:"), comparison.Metadata.Model.Candidate)
	_, _ = fmt.Fprintf(w, "%s %s\n\n", evalOutputLabel(w, "Environment:"), comparison.Metadata.Environment.Candidate)

	if len(comparison.MetadataDiffs) > 0 {
		_, _ = fmt.Fprintln(w, evalOutputLabel(w, "Metadata mismatches:"))
		for _, diff := range comparison.MetadataDiffs {
			_, _ = fmt.Fprintf(w, "  %s %v -> %v\n", evalOutputLabel(w, fmt.Sprintf("%s:", diff.Delta)), diff.Base, diff.Candidate)
		}
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w, evalOutputLabel(w, "Job:"))
	_, _ = fmt.Fprintf(w, "  %s %v -> %v  %s\n", evalOutputLabel(w, "completed:"), comparison.Job.Completed.Base, comparison.Job.Completed.Candidate, signedDelta(comparison.Job.Completed.Delta))
	_, _ = fmt.Fprintf(w, "  %s    %v -> %v  %s\n", evalOutputLabel(w, "errors:"), comparison.Job.Errors.Base, comparison.Job.Errors.Candidate, signedDelta(comparison.Job.Errors.Delta))
	_, _ = fmt.Fprintf(w, "  %s     %v -> %v  %s\n\n", evalOutputLabel(w, "evals:"), comparison.Job.Evals.Base, comparison.Job.Evals.Candidate, signedDelta(comparison.Job.Evals.Delta))

	_, _ = fmt.Fprintln(w, evalOutputLabel(w, "Results:"))
	if len(comparison.Results.Comparisons) == 0 {
		_, _ = fmt.Fprintln(w, "  no matching eval results")
	}
	for _, result := range comparison.Results.Comparisons {
		_, _ = fmt.Fprintf(w, "  %s reward %s -> %s  %s\n", evalOutputLabel(w, result.Key+":"), displayValue(result.Reward.Base), displayValue(result.Reward.Candidate), signedDelta(result.Reward.Delta))
	}
	if len(comparison.Results.BaseOnly) > 0 {
		_, _ = fmt.Fprintf(w, "  %s %s\n", evalOutputLabel(w, "base only:"), strings.Join(comparison.Results.BaseOnly, ", "))
	}
	if len(comparison.Results.CandidateOnly) > 0 {
		_, _ = fmt.Fprintf(w, "  %s %s\n", evalOutputLabel(w, "latest only:"), strings.Join(comparison.Results.CandidateOnly, ", "))
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s %s\n", evalOutputLabel(w, "Recommendation:"), comparison.Recommendation)
	return nil
}

func renderEvalComparisonJSON(w io.Writer, comparison evalComparison) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(comparison)
}

func stringDiff(base, candidate string) evalComparisonDiff {
	return evalComparisonDiff{Base: base, Candidate: candidate}
}

func intDiff(base, candidate int) evalComparisonDiff {
	return evalComparisonDiff{Base: base, Candidate: candidate, Delta: candidate - base}
}

func floatDiff(base, candidate *float64) evalComparisonDiff {
	diff := evalComparisonDiff{Base: base, Candidate: candidate}
	if base != nil && candidate != nil {
		diff.Delta = *candidate - *base
	}
	return diff
}

func compareJobStats(base, candidate promoteJobResult) evalComparisonStats {
	return evalComparisonStats{
		Completed: intDiff(base.Stats.CompletedTrials, candidate.Stats.CompletedTrials),
		Errors:    intDiff(base.Stats.ErroredTrials, candidate.Stats.ErroredTrials),
		Evals:     intDiff(len(base.Stats.Evals), len(candidate.Stats.Evals)),
		CostUSD:   floatDiff(base.Stats.CostUSD, candidate.Stats.CostUSD),
	}
}

func compareResults(base, candidate compareJob) evalComparisonResults {
	baseEntries := evalResultSummaryMap(base.Result, base.Config)
	candidateEntries := evalResultSummaryMap(candidate.Result, candidate.Config)
	baseKeys := sortedEvalResultSummaryKeys(baseEntries)
	candidateKeys := sortedEvalResultSummaryKeys(candidateEntries)
	candidateSet := make(map[string]struct{}, len(candidateKeys))
	for _, key := range candidateKeys {
		candidateSet[key] = struct{}{}
	}
	baseSet := make(map[string]struct{}, len(baseKeys))
	for _, key := range baseKeys {
		baseSet[key] = struct{}{}
	}

	var results evalComparisonResults
	for _, key := range baseKeys {
		if _, ok := candidateSet[key]; !ok {
			results.BaseOnly = append(results.BaseOnly, key)
			continue
		}
		baseEntry := baseEntries[key]
		candidateEntry := candidateEntries[key]
		results.Comparisons = append(results.Comparisons, evalComparisonResult{
			Key:          key,
			BaseKey:      baseEntry.RawKey,
			CandidateKey: candidateEntry.RawKey,
			Reward:       floatDiff(baseEntry.Reward, candidateEntry.Reward),
			RewardStatus: combinedRewardStatus(baseEntry.RewardStatus, candidateEntry.RewardStatus),
			Errors:       intDiff(baseEntry.Stats.Errors, candidateEntry.Stats.Errors),
			Trials:       intDiff(baseEntry.Stats.Trials, candidateEntry.Stats.Trials),
		})
	}
	for _, key := range candidateKeys {
		if _, ok := baseSet[key]; !ok {
			results.CandidateOnly = append(results.CandidateOnly, key)
		}
	}
	return results
}

func evalResultSummaryMap(result promoteJobResult, config promoteJobConfig) map[string]evalResultSummary {
	summaries := summarizeEvalResults(result, config)
	entries := make(map[string]evalResultSummary, len(summaries))
	for _, summary := range summaries {
		entries[summary.Key] = summary
	}
	return entries
}

func sortedEvalResultSummaryKeys(entries map[string]evalResultSummary) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func metadataMismatches(meta evalComparisonMeta) []evalComparisonDiff {
	checks := []struct {
		name string
		diff evalComparisonDiff
	}{
		{"dataset", meta.Dataset},
		{"tasks", meta.Tasks},
		{"agent", meta.Agent},
		{"model", meta.Model},
		{"environment", meta.Environment},
	}
	var mismatches []evalComparisonDiff
	for _, check := range checks {
		if check.diff.Base != check.diff.Candidate {
			check.diff.Delta = check.name
			mismatches = append(mismatches, check.diff)
		}
	}
	return mismatches
}

func compareRecommendation(comparison evalComparison) string {
	if sameComparisonJob(comparison.Base, comparison.Candidate) {
		return "base and latest are the same eval job; nothing to compare."
	}
	job := comparison.Job
	if intDelta(job.Errors) > 0 || intDelta(job.Completed) < 0 {
		return "candidate regressed; rerun or inspect job/task details before promotion."
	}
	if len(comparison.Results.Comparisons) == 0 {
		return "no matching eval results; compare job selection before promotion."
	}
	if len(comparison.MetadataDiffs) > 0 {
		return "metadata differs; review mismatches before promotion."
	}
	for _, result := range comparison.Results.Comparisons {
		if floatDelta(result.Reward) < 0 {
			return "candidate regressed; rerun or inspect job/task details before promotion."
		}
	}
	for _, result := range comparison.Results.Comparisons {
		if result.RewardStatus != "" {
			return "summary changed; inspect job/task details before promotion."
		}
	}
	return "candidate improved or held steady; promotion looks reasonable after normal review."
}

func sameComparisonJob(base, candidate evalComparisonJob) bool {
	if base.Result != "" && base.Result == candidate.Result {
		return true
	}
	return base.Path != "" && base.Path == candidate.Path
}

func intDelta(diff evalComparisonDiff) int {
	value, _ := diff.Delta.(int)
	return value
}

func floatDelta(diff evalComparisonDiff) float64 {
	value, _ := diff.Delta.(float64)
	return value
}

func signedDelta(value interface{}) string {
	switch typed := value.(type) {
	case int:
		return fmt.Sprintf("%+d", typed)
	case float64:
		return fmt.Sprintf("%+.3f", typed)
	default:
		return ""
	}
}

func displayValue(value interface{}) string {
	if value == nil {
		return "n/a"
	}
	switch typed := value.(type) {
	case *float64:
		if typed == nil {
			return "n/a"
		}
		return fmt.Sprintf("%.3f", *typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func formatCompareTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
