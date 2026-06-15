package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	root              = "/app"
	agentLogDir       = "/logs/agent"
	artifactsDir      = "/logs/artifacts"
	verifierDir       = "/logs/verifier"
	prDraft           = artifactsDir + "/pr.md"
	rewardPath        = verifierDir + "/reward.json"
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

type codexLogLine struct {
	Type string `json:"type"`
	Item struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	} `json:"item"`
	Payload struct {
		Type      string          `json:"type"`
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"payload"`
}

type claudeLogLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				Command string `json:"command"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

func commandFromAgentLogLine(line []byte) string {
	if command := commandFromCodexLogLine(line); command != "" {
		return command
	}
	return commandFromClaudeLogLine(line)
}

func commandFromCodexLogLine(line []byte) string {
	var event codexLogLine
	if err := json.Unmarshal(line, &event); err != nil {
		return ""
	}
	if event.Type == "item.completed" &&
		event.Item.Type == "command_execution" &&
		event.Item.Command != "" {
		return event.Item.Command
	}
	if event.Type != "response_item" ||
		event.Payload.Type != "function_call" ||
		event.Payload.Name != "exec_command" {
		return ""
	}
	return execCommandFromArguments(event.Payload.Arguments)
}

func commandFromClaudeLogLine(line []byte) string {
	var event claudeLogLine
	if err := json.Unmarshal(line, &event); err != nil {
		return ""
	}
	if event.Type != "assistant" {
		return ""
	}
	for _, content := range event.Message.Content {
		if content.Type == "tool_use" &&
			content.Name == "Bash" &&
			content.Input.Command != "" {
			return content.Input.Command
		}
	}
	return ""
}

func execCommandFromArguments(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var arguments string
	if err := json.Unmarshal(raw, &arguments); err == nil {
		raw = json.RawMessage(arguments)
	}

	var parsed struct {
		Cmd string `json:"cmd"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ""
	}
	return parsed.Cmd
}

func collectAgentCommands() string {
	var commands []string
	info, err := os.Stat(agentLogDir)
	if err != nil || !info.IsDir() {
		return ""
	}

	var paths []string
	_ = filepath.WalkDir(agentLogDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 || line[0] != '{' {
				continue
			}
			if command := commandFromAgentLogLine(line); command != "" {
				commands = append(commands, command)
			}
		}
		_ = file.Close()
	}

	return strings.ToLower(strings.Join(commands, "\n"))
}

func scoreDependencyUpdate(files []string, text string) float64 {
	rootDepChanged := contains(files, "go.mod") || contains(files, "go.sum")
	daggerUntouched := !contains(files, ".dagger/go.mod") && !contains(files, ".dagger/go.sum")
	updateAttempted := strings.Contains(text, ".agents/skills/update-minor-deps/scripts/update-go-deps.sh") ||
		strings.Contains(text, "go get -t -u ./...") ||
		strings.Contains(text, "update-go-deps.sh")

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

func scoreScopedChanges(files []string) float64 {
	if len(files) == 0 {
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
	for _, file := range files {
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

func scoreWorkflowEvidence(text string) float64 {
	checks := []bool{
		strings.Contains(text, "git status --short") || strings.Contains(text, "git status -s"),
		strings.Contains(text, "contributing.md"),
		strings.Contains(text, ".agents/skills/update-minor-deps/scripts/update-go-deps.sh") ||
			strings.Contains(text, "go get -t -u ./..."),
		strings.Contains(text, "go mod tidy") || strings.Contains(text, "update-go-deps.sh"),
	}
	return scoreChecks(checks)
}

func scoreSkillActivationEvidence(text string) float64 {
	checks := []bool{
		strings.Contains(text, "update-minor-deps"),
		strings.Contains(text, "skill") && (strings.Contains(text, "dependency") || strings.Contains(text, "dependencies")),
		strings.Contains(text, ".agents/skills/update-minor-deps/scripts/update-go-deps.sh"),
	}
	if anyTrue(checks) {
		return 1.0
	}
	return 0.0
}

func scoreValidationEvidence(text string, files []string) float64 {
	commands := text
	focused := goTestRE.MatchString(commands)
	final := finalValidationRan(commands)
	moduleOnly := isModuleOnlyChange(files)
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

func scorePRDraft(files []string) float64 {
	text := strings.ToLower(readText(prDraft))
	return scorePRDraftText(text, files)
}

func scorePRDraftText(text string, files []string) float64 {
	text = strings.ToLower(text)
	if text == "" {
		return 0.0
	}
	focusedValidation := strings.Contains(text, "go test") || strings.Contains(text, "focused")
	if isModuleOnlyChange(files) {
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

func scoreNoRealPROrCommit(text string) float64 {
	for _, re := range forbiddenCmdREs {
		if re.MatchString(text) {
			return 0.0
		}
	}
	return 1.0
}

func scoreRewards(files []string, text string, commands string, prDraftText string) rewardScores {
	return rewardScores{
		DependencyUpdate:        scoreDependencyUpdate(files, text),
		ScopedChanges:           scoreScopedChanges(files),
		SkillActivationEvidence: scoreSkillActivationEvidence(text),
		WorkflowEvidence:        scoreWorkflowEvidence(text),
		ValidationEvidence:      scoreValidationEvidence(commands, files),
		PRDraftQuality:          scorePRDraftText(prDraftText, files),
		NoRealPROrCommit:        scoreNoRealPROrCommit(commands),
	}
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

	files, err := changedFiles()
	if err != nil {
		return err
	}
	text := collectAgentText()
	commands := collectAgentCommands()
	scores := scoreRewards(files, text, commands, readText(prDraft))

	diagnostics := map[string]any{
		"changed_files": files,
		"pr_draft":      prDraft,
		"agent_log_dir": agentLogDir,
	}
	if err := writeJSON(filepath.Join(verifierDir, "diagnostics.json"), diagnostics, false); err != nil {
		return fmt.Errorf("write diagnostics: %w", err)
	}
	if err := writeJSON(rewardPath, scores, true); err != nil {
		return fmt.Errorf("write reward: %w", err)
	}

	output, err := json.MarshalIndent(scores, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
