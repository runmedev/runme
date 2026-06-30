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
