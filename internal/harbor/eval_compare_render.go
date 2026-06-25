package harbor

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type evalComparison struct {
	Base           evalComparisonJob    `json:"base"`
	Candidate      evalComparisonJob    `json:"candidate"`
	Metadata       evalComparisonMeta   `json:"metadata"`
	MetadataDiffs  []evalComparisonDiff `json:"metadata_mismatches,omitempty"`
	Stats          evalComparisonStats  `json:"stats"`
	Recommendation string               `json:"recommendation"`
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
	Mean      evalComparisonDiff `json:"mean,omitempty"`
	Evals     evalComparisonDiff `json:"evals"`
	CostUSD   evalComparisonDiff `json:"cost_usd,omitempty"`
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
	stats := compareStats(base.Result, candidate.Result)
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
		Stats:         stats,
	}
	comparison.Recommendation = compareRecommendation(comparison)
	return comparison
}

func renderEvalComparisonText(w io.Writer, comparison evalComparison) error {
	_, _ = fmt.Fprintf(w, "Base:   %s  tracked in %s\n", comparison.Base.Path, comparison.Base.Ref)
	_, _ = fmt.Fprintf(w, "Latest: %s  local\n\n", comparison.Candidate.Path)

	_, _ = fmt.Fprintf(w, "Dataset: %s\n", comparison.Metadata.Dataset.Candidate)
	if tasks := strings.TrimSpace(fmt.Sprint(comparison.Metadata.Tasks.Candidate)); tasks != "" {
		_, _ = fmt.Fprintf(w, "Tasks: %s\n", tasks)
	}
	_, _ = fmt.Fprintf(w, "Agent: %s\n", comparison.Metadata.Agent.Candidate)
	_, _ = fmt.Fprintf(w, "Model: %s\n", comparison.Metadata.Model.Candidate)
	_, _ = fmt.Fprintf(w, "Environment: %s\n\n", comparison.Metadata.Environment.Candidate)

	if len(comparison.MetadataDiffs) > 0 {
		_, _ = fmt.Fprintln(w, "Metadata mismatches:")
		for _, diff := range comparison.MetadataDiffs {
			_, _ = fmt.Fprintf(w, "  %s: %v -> %v\n", diff.Delta, diff.Base, diff.Candidate)
		}
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w, "Score:")
	_, _ = fmt.Fprintf(w, "  completed: %v -> %v  %s\n", comparison.Stats.Completed.Base, comparison.Stats.Completed.Candidate, signedDelta(comparison.Stats.Completed.Delta))
	_, _ = fmt.Fprintf(w, "  errors:    %v -> %v  %s\n", comparison.Stats.Errors.Base, comparison.Stats.Errors.Candidate, signedDelta(comparison.Stats.Errors.Delta))
	if comparison.Stats.Mean.Base != nil || comparison.Stats.Mean.Candidate != nil {
		_, _ = fmt.Fprintf(w, "  mean:      %s -> %s  %s\n", displayValue(comparison.Stats.Mean.Base), displayValue(comparison.Stats.Mean.Candidate), signedDelta(comparison.Stats.Mean.Delta))
	}
	_, _ = fmt.Fprintf(w, "  evals:     %v -> %v  %s\n\n", comparison.Stats.Evals.Base, comparison.Stats.Evals.Candidate, signedDelta(comparison.Stats.Evals.Delta))
	_, _ = fmt.Fprintf(w, "Recommendation: %s\n", comparison.Recommendation)
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

func compareStats(base, candidate promoteJobResult) evalComparisonStats {
	baseMean, baseHasMean := singlePromoteMean(base.Stats)
	candidateMean, candidateHasMean := singlePromoteMean(candidate.Stats)
	var baseMeanPtr, candidateMeanPtr *float64
	if baseHasMean {
		baseMeanPtr = &baseMean
	}
	if candidateHasMean {
		candidateMeanPtr = &candidateMean
	}
	return evalComparisonStats{
		Completed: intDiff(base.Stats.CompletedTrials, candidate.Stats.CompletedTrials),
		Errors:    intDiff(base.Stats.ErroredTrials, candidate.Stats.ErroredTrials),
		Mean:      floatDiff(baseMeanPtr, candidateMeanPtr),
		Evals:     intDiff(len(base.Stats.Evals), len(candidate.Stats.Evals)),
		CostUSD:   floatDiff(base.Stats.CostUSD, candidate.Stats.CostUSD),
	}
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
	if len(comparison.MetadataDiffs) > 0 {
		return "metadata differs; review mismatches before promotion."
	}
	stats := comparison.Stats
	if intDelta(stats.Errors) > 0 || intDelta(stats.Completed) < 0 || floatDelta(stats.Mean) < 0 {
		return "candidate regressed; rerun or inspect job/task details before promotion."
	}
	if stats.Mean.Base == nil || stats.Mean.Candidate == nil {
		return "summary changed; inspect job/task details before promotion."
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
