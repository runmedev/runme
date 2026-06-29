package harbor

import (
	"fmt"
	"sort"
	"strings"
)

const evalRewardStatusAmbiguous = "ambiguous"

type evalResultSummary struct {
	Key          string
	RawKey       string
	Stats        promoteEvalStats
	Reward       *float64
	RewardStatus string
}

func summarizeEvalResults(result promoteJobResult, config promoteJobConfig) []evalResultSummary {
	entries := normalizedEvalResultEntries(result, config)
	keys := sortedEvalResultEntryKeys(entries)
	summaries := make([]evalResultSummary, 0, len(keys))
	for _, key := range keys {
		entry := entries[key]
		reward, status := evalReward(entry.stats)
		summaries = append(summaries, evalResultSummary{
			Key:          key,
			RawKey:       entry.key,
			Stats:        entry.stats,
			Reward:       reward,
			RewardStatus: status,
		})
	}
	return summaries
}

type evalResultEntry struct {
	key   string
	stats promoteEvalStats
}

func normalizedEvalResultEntries(result promoteJobResult, config promoteJobConfig) map[string]evalResultEntry {
	keys := sortedEvalKeys(result.Stats.Evals)
	entries := make(map[string]evalResultEntry, len(keys))
	collisions := make(map[string]struct{})
	for _, key := range keys {
		identity := evalResultIdentity(key, config)
		if _, ok := entries[identity]; ok {
			collisions[identity] = struct{}{}
		}
		entries[identity] = evalResultEntry{
			key:   key,
			stats: result.Stats.Evals[key],
		}
	}
	if len(collisions) == 0 {
		return entries
	}

	entries = make(map[string]evalResultEntry, len(keys))
	for _, key := range keys {
		identity := evalResultIdentity(key, config)
		if _, ok := collisions[identity]; ok {
			identity = key
		}
		entries[identity] = evalResultEntry{
			key:   key,
			stats: result.Stats.Evals[key],
		}
	}
	return entries
}

func evalResultIdentity(key string, config promoteJobConfig) string {
	for _, agent := range sortedAgentKeys(config.Agents) {
		if result, ok := strings.CutPrefix(key, agent+"__"); ok {
			return result
		}
	}
	return key
}

func sortedAgentKeys(agents []promoteAgentConfig) []string {
	keys := make([]string, 0, len(agents))
	for _, agent := range agents {
		key := agent.Name
		if key == "" {
			key = agent.ImportPath
		}
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	return keys
}

func sortedEvalResultEntryKeys(entries map[string]evalResultEntry) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedEvalKeys(evals map[string]promoteEvalStats) []string {
	keys := make([]string, 0, len(evals))
	for key := range evals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func evalReward(eval promoteEvalStats) (*float64, string) {
	var rewards []float64
	for _, metric := range eval.Metrics {
		if value, ok := promoteMetricNumber(metric["reward"]); ok {
			rewards = append(rewards, value)
		}
	}
	switch len(rewards) {
	case 0:
		reward := 0.0
		return &reward, ""
	case 1:
		return &rewards[0], ""
	default:
		return nil, evalRewardStatusAmbiguous
	}
}

func combinedRewardStatus(base, candidate string) string {
	if base == "" && candidate == "" {
		return ""
	}
	if base == candidate {
		return base
	}
	if base == "" {
		base = "ok"
	}
	if candidate == "" {
		candidate = "ok"
	}
	return fmt.Sprintf("base %s, candidate %s", base, candidate)
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
