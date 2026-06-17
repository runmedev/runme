package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	root              = "/app"
	agentLogDir       = "/logs/agent"
	artifactsDir      = "/logs/artifacts"
	verifierDir       = "/logs/verifier"
	prDraft           = artifactsDir + "/pr.md"
	rewardPath        = verifierDir + "/reward.json"
	rewardDetailsPath = verifierDir + "/reward-details.json"
	evalHarnessPrefix = ".agents/skills/update-minor-deps/evals/regression/"
)

var (
	prTitleWithDateRE = regexp.MustCompile(`chore:\s*update minor and patch dependencies.*\b\d{4}-\d{2}-\d{2}\b`)
	goTestRE          = regexp.MustCompile(`go test\s+(\./|\S*/)`)
	forbiddenCmdREs   = []*regexp.Regexp{
		regexp.MustCompile(`\bgit\s+commit\b`),
		regexp.MustCompile(`\bgit\s+push\b`),
		regexp.MustCompile(`\bgh\s+pr\s+create\b`),
		regexp.MustCompile(`\bhub\s+pull-request\b`),
	}
)

type rewardScores struct {
	DependencyUpdate        float64 `json:"dependency_update"`
	ScopedChanges           float64 `json:"scoped_changes"`
	SkillActivationEvidence float64 `json:"skill_activation_evidence"`
	WorkflowEvidence        float64 `json:"workflow_evidence"`
	ValidationEvidence      float64 `json:"validation_evidence"`
	PRDraftQuality          float64 `json:"pr_draft_quality"`
	NoRealPROrCommit        float64 `json:"no_real_pr_or_commit"`
}

type rewardCriterion struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Raw         float64 `json:"raw"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

type rewardDetail struct {
	Score    float64           `json:"score"`
	Criteria []rewardCriterion `json:"criteria"`
	Kind     string            `json:"kind"`
}

type rewardScore struct {
	Name        string
	Score       float64
	Description string
}

type scorer struct {
	files       []string
	text        string
	commands    string
	prDraftText string
}

func newScorer() (*scorer, error) {
	files, err := changedFiles()
	if err != nil {
		return nil, err
	}

	return &scorer{
		files:       files,
		text:        collectAgentText(),
		commands:    collectAgentCommands(),
		prDraftText: readText(prDraft),
	}, nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

func changedFiles() ([]string, error) {
	files := make(map[string]struct{})
	for _, args := range [][]string{
		{"diff", "--name-only"},
		{"diff", "--cached", "--name-only"},
	} {
		output, err := runGit(args...)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(output, "\n") {
			file := strings.TrimSpace(line)
			if file != "" {
				files[file] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	sort.Strings(result)
	return result, nil
}

func relevantChangedFiles(files []string) []string {
	var relevant []string
	for _, file := range files {
		if strings.HasPrefix(file, evalHarnessPrefix) {
			continue
		}
		relevant = append(relevant, file)
	}
	return relevant
}

func isModuleOnlyChange(files []string) bool {
	relevant := relevantChangedFiles(files)
	if len(relevant) == 0 {
		return false
	}
	for _, file := range relevant {
		if file != "go.mod" && file != "go.sum" {
			return false
		}
	}
	return true
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func collectAgentText() string {
	var chunks []string
	info, err := os.Stat(agentLogDir)
	if err == nil && info.IsDir() {
		var paths []string
		_ = filepath.WalkDir(agentLogDir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil || info.Size() > 5_000_000 {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		sort.Strings(paths)
		for _, path := range paths {
			chunks = append(chunks, readText(path))
		}
	}

	chunks = append(chunks, readText(prDraft))
	return strings.ToLower(strings.Join(chunks, "\n"))
}

type atifTrajectory struct {
	SchemaVersion string     `json:"schema_version"`
	Steps         []atifStep `json:"steps"`
}

type atifStep struct {
	ToolCalls []atifToolCall `json:"tool_calls"`
}

type atifToolCall struct {
	FunctionName string                     `json:"function_name"`
	Arguments    map[string]json.RawMessage `json:"arguments"`
}

func commandsFromATIF(data []byte) []string {
	var trajectory atifTrajectory
	if err := json.Unmarshal(data, &trajectory); err != nil {
		return nil
	}

	var commands []string
	for _, step := range trajectory.Steps {
		for _, toolCall := range step.ToolCalls {
			if command := commandFromATIFArguments(toolCall.Arguments); command != "" {
				commands = append(commands, command)
			}
		}
	}
	return commands
}

func commandFromATIFArguments(arguments map[string]json.RawMessage) string {
	for _, key := range []string{"command", "cmd"} {
		raw, ok := arguments[key]
		if !ok {
			continue
		}
		var command string
		if err := json.Unmarshal(raw, &command); err != nil {
			continue
		}
		if strings.TrimSpace(command) != "" {
			return command
		}
	}
	return ""
}

func collectAgentCommands() string {
	return collectAgentCommandsFromDir(agentLogDir)
}

func collectAgentCommandsFromDir(dir string) string {
	commands := collectAgentCommandsFromFile(filepath.Join(dir, "trajectory.json"))
	if commands != "" {
		return commands
	}
	oracleLogPath := filepath.Join(dir, "oracle.txt")
	if _, err := os.Stat(oracleLogPath); err != nil {
		return ""
	}
	return collectAgentShellTraceCommandsFromFile(oracleLogPath)
}

func collectAgentCommandsFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	commands := commandsFromATIF(data)
	return strings.ToLower(strings.Join(commands, "\n"))
}

func collectAgentShellTraceCommandsFromFile(path string) string {
	text := readText(path)
	if text == "" {
		return ""
	}

	var commands []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if command, ok := strings.CutPrefix(line, "+ "); ok {
			command = strings.TrimSpace(command)
			if command != "" {
				commands = append(commands, command)
			}
		}
	}
	return strings.ToLower(strings.Join(commands, "\n"))
}

func (s scorer) dependencyUpdate() float64 {
	rootDepChanged := contains(s.files, "go.mod") || contains(s.files, "go.sum")
	daggerUntouched := !contains(s.files, ".dagger/go.mod") && !contains(s.files, ".dagger/go.sum")
	updateAttempted := strings.Contains(s.text, ".agents/skills/update-minor-deps/scripts/update-go-deps.sh") ||
		strings.Contains(s.text, "go get -t -u ./...") ||
		strings.Contains(s.text, "update-go-deps.sh")

	switch {
	case rootDepChanged && daggerUntouched:
		return 1.0
	case daggerUntouched && updateAttempted:
		return 0.5
	case rootDepChanged:
		return 0.5
	default:
		return 0.0
	}
}

func (s scorer) scopedChanges() float64 {
	if len(s.files) == 0 {
		return 1.0
	}

	forbiddenPrefixes := []string{
		"api/gen/",
		"app/",
		"docs/",
		"examples/",
		"integrations/",
		"web/",
		".github/",
		".dagger/",
	}
	forbiddenNames := map[string]struct{}{
		"README.md":       {},
		"CONTRIBUTING.md": {},
		"Makefile":        {},
		"package.json":    {},
		"pnpm-lock.yaml":  {},
		"yarn.lock":       {},
		"bun.lock":        {},
	}

	var unrelated []string
	for _, file := range s.files {
		switch {
		case file == "go.mod" || file == "go.sum":
			continue
		case strings.HasSuffix(file, ".go"):
			continue
		case strings.HasPrefix(file, ".agents/skills/update-minor-deps/evals/regression/"):
			continue
		}

		if _, ok := forbiddenNames[file]; ok {
			unrelated = append(unrelated, file)
			continue
		}
		if hasAnyPrefix(file, forbiddenPrefixes) {
			unrelated = append(unrelated, file)
			continue
		}
		unrelated = append(unrelated, file)
	}

	switch {
	case len(unrelated) == 0:
		return 1.0
	case len(unrelated) <= 2:
		return 0.5
	default:
		return 0.0
	}
}

func (s scorer) workflowEvidence() float64 {
	checks := []bool{
		strings.Contains(s.text, "git status --short") || strings.Contains(s.text, "git status -s"),
		strings.Contains(s.text, "contributing.md"),
		strings.Contains(s.text, ".agents/skills/update-minor-deps/scripts/update-go-deps.sh") ||
			strings.Contains(s.text, "go get -t -u ./..."),
		strings.Contains(s.text, "go mod tidy") || strings.Contains(s.text, "update-go-deps.sh"),
	}
	return scoreChecks(checks)
}

func (s scorer) skillActivationEvidence() float64 {
	checks := []bool{
		strings.Contains(s.text, "update-minor-deps"),
		strings.Contains(s.text, "skill") && (strings.Contains(s.text, "dependency") || strings.Contains(s.text, "dependencies")),
		strings.Contains(s.text, ".agents/skills/update-minor-deps/scripts/update-go-deps.sh"),
	}
	if anyTrue(checks) {
		return 1.0
	}
	return 0.0
}

func (s scorer) validationEvidence() float64 {
	focused := goTestRE.MatchString(s.commands)
	final := finalValidationRan(s.commands)
	moduleOnly := isModuleOnlyChange(s.files)
	switch {
	case final && (focused || moduleOnly):
		return 1.0
	case final:
		return 0.6
	case focused:
		return 0.4
	default:
		return 0.0
	}
}

func finalValidationRan(commands string) bool {
	return strings.Contains(commands, "runme run lint test") ||
		(strings.Contains(commands, "runme run lint") && strings.Contains(commands, "runme run test"))
}

func (s scorer) prDraftQuality() float64 {
	text := strings.ToLower(s.prDraftText)
	if text == "" {
		return 0.0
	}
	focusedValidation := strings.Contains(text, "go test") || strings.Contains(text, "focused")
	if isModuleOnlyChange(s.files) {
		focusedValidation = focusedValidation || hasNoCompatibilityFixesNeeded(text)
	}
	checks := []bool{
		strings.Contains(text, "chore: update minor and patch dependencies"),
		prTitleWithDateRE.MatchString(text),
		strings.Contains(text, "update-go-deps.sh") || strings.Contains(text, "go get -t -u ./..."),
		strings.Contains(text, "go.mod"),
		strings.Contains(text, "go.sum"),
		strings.Contains(text, "compat") || strings.Contains(text, "fix") || strings.Contains(text, "none"),
		focusedValidation,
		finalValidationRan(text),
	}
	return scoreChecks(checks)
}

func hasNoCompatibilityFixesNeeded(text string) bool {
	noFixes := strings.Contains(text, "no compatibility") ||
		strings.Contains(text, "compatibility fixes: none") ||
		strings.Contains(text, "compatibility code") ||
		strings.Contains(text, "code or test fixes")
	return noFixes && (strings.Contains(text, "not required") || strings.Contains(text, "none") || strings.Contains(text, "no "))
}

func (s scorer) noRealPROrCommit() float64 {
	for _, re := range forbiddenCmdREs {
		if re.MatchString(s.commands) {
			return 0.0
		}
	}
	return 1.0
}

func (s scorer) scores() rewardScores {
	return rewardScores{
		DependencyUpdate:        s.dependencyUpdate(),
		ScopedChanges:           s.scopedChanges(),
		SkillActivationEvidence: s.skillActivationEvidence(),
		WorkflowEvidence:        s.workflowEvidence(),
		ValidationEvidence:      s.validationEvidence(),
		PRDraftQuality:          s.prDraftQuality(),
		NoRealPROrCommit:        s.noRealPROrCommit(),
	}
}

func rewardScoreList(scores rewardScores) []rewardScore {
	return []rewardScore{
		{
			Name:        "dependency_update",
			Score:       scores.DependencyUpdate,
			Description: "dependency_update",
		},
		{
			Name:        "scoped_changes",
			Score:       scores.ScopedChanges,
			Description: "scoped_changes",
		},
		{
			Name:        "skill_activation_evidence",
			Score:       scores.SkillActivationEvidence,
			Description: "skill_activation_evidence",
		},
		{
			Name:        "workflow_evidence",
			Score:       scores.WorkflowEvidence,
			Description: "workflow_evidence",
		},
		{
			Name:        "validation_evidence",
			Score:       scores.ValidationEvidence,
			Description: "validation_evidence",
		},
		{
			Name:        "pr_draft_quality",
			Score:       scores.PRDraftQuality,
			Description: "pr_draft_quality",
		},
		{
			Name:        "no_real_pr_or_commit",
			Score:       scores.NoRealPROrCommit,
			Description: "no_real_pr_or_commit",
		},
	}
}

func rewardDetails(scores rewardScores) map[string]rewardDetail {
	details := make(map[string]rewardDetail)
	for _, score := range rewardScoreList(scores) {
		details[score.Name] = rewardDetail{
			Score: score.Score,
			Criteria: []rewardCriterion{
				{
					Name:        score.Name,
					Value:       score.Score,
					Raw:         score.Score,
					Weight:      1.0,
					Description: score.Description,
				},
			},
			Kind: "programmatic",
		}
	}
	return details
}

func printRewardSummary(scores rewardScores) {
	for _, score := range rewardScoreList(scores) {
		fmt.Printf("%s: %s\n", score.Name, formatRewardFloat(score.Score))
	}
}

func formatRewardFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(formatted, ".") {
		return formatted + ".0"
	}
	return formatted
}

func scoreChecks(checks []bool) float64 {
	var passed int
	for _, check := range checks {
		if check {
			passed++
		}
	}
	return float64(passed) / float64(len(checks))
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func anyTrue(values []bool) bool {
	for _, value := range values {
		if value {
			return true
		}
	}
	return false
}

func writeJSON(path string, value any, trailingNewline bool) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if trailingNewline {
		data = append(data, '\n')
	}
	return os.WriteFile(path, data, 0o600)
}

func run() error {
	if err := os.MkdirAll(verifierDir, 0o750); err != nil {
		return fmt.Errorf("create verifier dir: %w", err)
	}
	if err := os.MkdirAll(artifactsDir, 0o750); err != nil {
		return fmt.Errorf("create artifacts dir: %w", err)
	}

	s, err := newScorer()
	if err != nil {
		return err
	}
	scores := s.scores()

	diagnostics := map[string]any{
		"changed_files": s.files,
		"pr_draft":      prDraft,
		"agent_log_dir": agentLogDir,
	}
	if err := writeJSON(filepath.Join(verifierDir, "diagnostics.json"), diagnostics, false); err != nil {
		return fmt.Errorf("write diagnostics: %w", err)
	}
	if err := writeJSON(rewardPath, scores, true); err != nil {
		return fmt.Errorf("write reward: %w", err)
	}
	if err := writeJSON(rewardDetailsPath, rewardDetails(scores), true); err != nil {
		return fmt.Errorf("write reward details: %w", err)
	}
	printRewardSummary(scores)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
