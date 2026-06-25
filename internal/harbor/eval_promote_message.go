package harbor

import (
	"fmt"
	"sort"
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
		_, _ = fmt.Fprintf(&b, "Promotion-Mode: eval-evidence-only\n")
	}
	_, _ = fmt.Fprintf(&b, "Dataset: %s\n", data.datasetSummary())
	if tasks := data.taskSummary(); tasks != "" {
		_, _ = fmt.Fprintf(&b, "Tasks: %s\n", tasks)
	}
	_, _ = fmt.Fprintln(&b)
	_, _ = fmt.Fprintf(&b, "Agent: %s\n", data.agentSummary())
	_, _ = fmt.Fprintf(&b, "Model: %s\n", data.modelSummary())
	_, _ = fmt.Fprintf(&b, "Environment: %s\n\n", data.environmentSummary())
	_, _ = fmt.Fprintf(&b, "Result: %s\n", data.resultSummary())
	if score := data.scoreSummary(); score != "" {
		_, _ = fmt.Fprintf(&b, "Score: %s\n", score)
	}
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

func (d promoteMessageData) resultSummary() string {
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

func (d promoteMessageData) scoreSummary() string {
	if score, ok := singlePromoteMean(d.result.Stats); ok {
		return fmt.Sprintf("mean=%.3f", score)
	}
	return ""
}

func singlePromoteMean(stats promoteJobStats) (float64, bool) {
	var means []float64
	keys := make([]string, 0, len(stats.Evals))
	for key := range stats.Evals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, metric := range stats.Evals[key].Metrics {
			if value, ok := promoteMetricNumber(metric["mean"]); ok {
				means = append(means, value)
			}
		}
	}
	if len(means) != 1 {
		return 0, false
	}
	return means[0], true
}

func promoteMetricNumber(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	default:
		return 0, false
	}
}
