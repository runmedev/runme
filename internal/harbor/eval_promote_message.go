package harbor

import (
	"fmt"
	"strings"
)

type promoteMessageData struct {
	subject      string
	jobPath      string
	resultPath   string
	evidenceOnly bool
	config       promoteJobConfig
	result       promoteJobResult
	comparison   promoteMessageComparison
}

type promoteMessageComparison struct {
	hasBaseline bool
	basePath    string
	baseRef     string
	baseCommit  string
	job         evalComparisonStats
	results     evalComparisonResults
	gate        promoteCompareGate
	overridden  bool
}

func renderPromoteCommitMessage(data promoteMessageData) string {
	var b strings.Builder
	subject := strings.TrimSpace(data.subject)
	if subject == "" {
		subject = defaultPromoteSubject
	}
	_, _ = fmt.Fprintf(&b, "%s\n\n", subject)
	_, _ = fmt.Fprintf(&b, "Eval-Job: %s\n", data.jobPath)
	_, _ = fmt.Fprintf(&b, "Eval-Result: %s\n", data.resultPath)
	if data.evidenceOnly {
		_, _ = fmt.Fprintln(&b, "Promotion-Mode: eval-evidence-only")
	}
	_, _ = fmt.Fprintf(&b, "Dataset: %s\n", data.datasetSummary())
	if tasks := data.taskSummary(); tasks != "" {
		_, _ = fmt.Fprintf(&b, "Tasks: %s\n", tasks)
	}
	_, _ = fmt.Fprintln(&b)
	if results := data.resultsSummary(); len(results) > 0 {
		_, _ = fmt.Fprintln(&b, "Results:")
		for _, result := range results {
			_, _ = fmt.Fprintf(&b, " %s\n", result)
		}
	}
	_, _ = fmt.Fprintf(&b, "Job: %s\n", data.jobSummary())
	_, _ = fmt.Fprintln(&b)
	data.comparison.render(&b)
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintf(&b, "Agent: %s\n", data.agentSummary())
	_, _ = fmt.Fprintf(&b, "Model: %s\n", data.modelSummary())
	_, _ = fmt.Fprintf(&b, "Environment: %s\n", data.environmentSummary())
	return b.String()
}

func (d promoteMessageData) datasetSummary() string {
	var values []string
	for _, dataset := range d.config.Datasets {
		if dataset.Name != "" {
			values = append(values, dataset.Name)
			continue
		}
		values = append(values, dataset.Path)
	}
	return displayList(values)
}

func (d promoteMessageData) taskSummary() string {
	var values []string
	for _, dataset := range d.config.Datasets {
		values = append(values, dataset.TaskNames...)
	}
	if len(values) == 0 {
		return ""
	}
	return displayList(values)
}

func (d promoteMessageData) agentSummary() string {
	var values []string
	for _, agent := range d.config.Agents {
		if agent.Name != "" {
			values = append(values, agent.Name)
			continue
		}
		values = append(values, agent.ImportPath)
	}
	return displayList(values)
}

func (d promoteMessageData) modelSummary() string {
	var values []string
	for _, agent := range d.config.Agents {
		values = append(values, agent.ModelName)
	}
	return displayList(values)
}

func (d promoteMessageData) environmentSummary() string {
	if d.config.Environment.Type != "" {
		return d.config.Environment.Type
	}
	if d.config.Environment.ImportPath != "" {
		return d.config.Environment.ImportPath
	}
	return "unknown"
}

func (d promoteMessageData) jobSummary() string {
	stats := d.result.Stats
	parts := []string{
		fmt.Sprintf("completed=%d/%d", stats.CompletedTrials, d.result.TotalTrials),
		fmt.Sprintf("errors=%d", stats.ErroredTrials),
	}
	if len(stats.Evals) > 0 {
		parts = append(parts, fmt.Sprintf("evals=%d", len(stats.Evals)))
	}
	return strings.Join(parts, ", ")
}

func (d promoteMessageData) resultsSummary() []string {
	summaries := evalJobRef{
		Result: d.result,
		Config: d.config,
	}.ResultSummaries()
	values := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		reward := "n/a"
		if summary.RewardStatus == evalRewardStatusAmbiguous {
			reward = "n/a (ambiguous)"
		} else if summary.Reward != nil {
			reward = fmt.Sprintf("%.3f", *summary.Reward)
		}
		values = append(values, fmt.Sprintf("%s: reward=%s", summary.Key, reward))
	}
	return values
}

func newPromoteMessageComparison(comparison evalComparison, baseCommit string, overridden bool) promoteMessageComparison {
	return promoteMessageComparison{
		hasBaseline: true,
		basePath:    comparison.Base.Path,
		baseRef:     comparison.Base.Ref,
		baseCommit:  baseCommit,
		job:         comparison.Job,
		results:     comparison.Results,
		gate:        comparison.Gate(),
		overridden:  overridden,
	}
}

func noBaselinePromoteMessageComparison() promoteMessageComparison {
	return promoteMessageComparison{}
}

func (c promoteMessageComparison) shouldRender() bool {
	return c.hasBaseline || c.baseRef != "" || c.baseCommit != ""
}

func (c promoteMessageComparison) render(b *strings.Builder) {
	if !c.shouldRender() {
		_, _ = fmt.Fprintln(b, "Comparison-Base: none")
		_, _ = fmt.Fprintln(b, "Promotion-Gate: not evaluated")
		return
	}

	_, _ = fmt.Fprintf(b, "Comparison-Base: %s tracked in %s", c.basePath, c.baseRef)
	if short := shortCommitHash(c.baseCommit); short != "" {
		_, _ = fmt.Fprintf(b, " (%s)", short)
	}
	_, _ = fmt.Fprintln(b)

	c.renderResultDelta(b)
	c.renderJobDelta(b)
	c.renderPromotionGate(b)
}

func (c promoteMessageComparison) renderResultDelta(b *strings.Builder) {
	if len(c.results.Comparisons) == 0 {
		_, _ = fmt.Fprintln(b, "Result-Delta: no matching eval results")
	} else {
		_, _ = fmt.Fprintln(b, "Result-Delta:")
		for _, result := range c.results.Comparisons {
			_, _ = fmt.Fprintf(
				b,
				" %s: reward %s -> %s  %s\n",
				result.Key,
				displayValue(result.Reward.Base),
				displayValue(result.Reward.Candidate),
				signedDelta(result.Reward.Delta),
			)
		}
	}
	if len(c.results.BaseOnly) > 0 {
		_, _ = fmt.Fprintf(b, "Base-Only-Results: %s\n", strings.Join(c.results.BaseOnly, ", "))
	}
	if len(c.results.CandidateOnly) > 0 {
		_, _ = fmt.Fprintf(b, "Candidate-Only-Results: %s\n", strings.Join(c.results.CandidateOnly, ", "))
	}
}

func (c promoteMessageComparison) renderJobDelta(b *strings.Builder) {
	_, _ = fmt.Fprintf(
		b,
		"Job-Delta: completed %s, errors %s, evals %s\n",
		signedDelta(c.job.Completed.Delta),
		signedDelta(c.job.Errors.Delta),
		signedDelta(c.job.Evals.Delta),
	)
}

func (c promoteMessageComparison) renderPromotionGate(b *strings.Builder) {
	switch {
	case c.gate.Blocking && c.overridden:
		_, _ = fmt.Fprintln(b, "Promotion-Gate: overridden")
		_, _ = fmt.Fprintf(b, "Promotion-Gate-Reason: %s\n", c.gate.Reason)
	case c.gate.Blocking:
		_, _ = fmt.Fprintln(b, "Promotion-Gate: blocked")
		_, _ = fmt.Fprintf(b, "Promotion-Gate-Reason: %s\n", c.gate.Reason)
	default:
		_, _ = fmt.Fprintln(b, "Promotion-Gate: passed")
	}
}

func shortCommitHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) <= 12 {
		return hash
	}
	return hash[:8]
}
