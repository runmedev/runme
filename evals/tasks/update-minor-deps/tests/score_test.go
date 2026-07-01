package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewardScoresJSONShape(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(rewardScores{
		Reward:                  1.0,
		DependencyUpdate:        1.0,
		ScopedChanges:           1.0,
		SkillActivationEvidence: 1.0,
		WorkflowEvidence:        1.0,
		ValidationEvidence:      1.0,
		PRDraftQuality:          1.0,
		NoRealPROrCommit:        1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	var scores map[string]float64
	if err := json.Unmarshal(data, &scores); err != nil {
		t.Fatal(err)
	}

	wantKeys := []string{
		"reward",
		"dependency_update",
		"scoped_changes",
		"skill_activation_evidence",
		"workflow_evidence",
		"validation_evidence",
		"pr_draft_quality",
		"no_real_pr_or_commit",
	}
	if len(scores) != len(wantKeys) {
		t.Fatalf("got %d keys, want %d: %#v", len(scores), len(wantKeys), scores)
	}
	for _, key := range wantKeys {
		if _, ok := scores[key]; !ok {
			t.Fatalf("missing reward key %q in %#v", key, scores)
		}
	}
}

func TestRollupReward(t *testing.T) {
	t.Parallel()

	scores := rewardScores{
		DependencyUpdate:        1.0,
		ScopedChanges:           0.5,
		SkillActivationEvidence: 1.0,
		WorkflowEvidence:        0.75,
		ValidationEvidence:      0.6,
		PRDraftQuality:          0.875,
		NoRealPROrCommit:        1.0,
	}

	want := (1.0 + 0.5 + 1.0 + 0.75 + 0.6 + 0.875 + 1.0) / 7.0
	if got := rollupReward(scores); got != want {
		t.Fatalf("rollupReward() = %v, want %v", got, want)
	}
}

func TestRewardDetailsJSONShape(t *testing.T) {
	t.Parallel()

	scores := rewardScores{
		Reward:                  0.8178571428571428,
		DependencyUpdate:        1.0,
		ScopedChanges:           0.5,
		SkillActivationEvidence: 1.0,
		WorkflowEvidence:        0.75,
		ValidationEvidence:      0.6,
		PRDraftQuality:          0.875,
		NoRealPROrCommit:        1.0,
	}
	data, err := json.Marshal(rewardDetails(scores))
	if err != nil {
		t.Fatal(err)
	}

	var details map[string]struct {
		Score    float64 `json:"score"`
		Criteria []struct {
			Name        string  `json:"name"`
			Value       float64 `json:"value"`
			Raw         float64 `json:"raw"`
			Weight      float64 `json:"weight"`
			Description string  `json:"description"`
		} `json:"criteria"`
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &details); err != nil {
		t.Fatal(err)
	}

	wantScores := map[string]float64{
		"dependency_update":         1.0,
		"scoped_changes":            0.5,
		"skill_activation_evidence": 1.0,
		"workflow_evidence":         0.75,
		"validation_evidence":       0.6,
		"pr_draft_quality":          0.875,
		"no_real_pr_or_commit":      1.0,
	}
	if len(details) != len(wantScores) {
		t.Fatalf("got %d details, want %d: %#v", len(details), len(wantScores), details)
	}
	for name, wantScore := range wantScores {
		detail, ok := details[name]
		if !ok {
			t.Fatalf("missing reward detail %q in %#v", name, details)
		}
		if detail.Score != wantScore {
			t.Fatalf("%s score = %v, want %v", name, detail.Score, wantScore)
		}
		if detail.Kind != "programmatic" {
			t.Fatalf("%s kind = %q, want programmatic", name, detail.Kind)
		}
		if len(detail.Criteria) != 1 {
			t.Fatalf("%s criteria len = %d, want 1", name, len(detail.Criteria))
		}
		criterion := detail.Criteria[0]
		if criterion.Name != name || criterion.Description != name {
			t.Fatalf("%s criterion metadata = %#v", name, criterion)
		}
		if criterion.Value != wantScore || criterion.Raw != wantScore || criterion.Weight != 1.0 {
			t.Fatalf("%s criterion scores = %#v, want score/raw %v and weight 1", name, criterion, wantScore)
		}
	}
}

func TestTaskRoot(t *testing.T) {
	t.Run("uses env", func(t *testing.T) {
		t.Setenv(taskWorkdirEnv, "/tmp/task-workdir")

		if got := taskRoot(); got != "/tmp/task-workdir" {
			t.Fatalf("taskRoot() = %q, want /tmp/task-workdir", got)
		}
	})

	t.Run("falls back to default", func(t *testing.T) {
		t.Setenv(taskWorkdirEnv, "")

		if got := taskRoot(); got != defaultRoot {
			t.Fatalf("taskRoot() = %q, want %q", got, defaultRoot)
		}
	})
}

func TestRuntimePaths(t *testing.T) {
	t.Run("uses env", func(t *testing.T) {
		t.Setenv(agentLogDirEnv, "/tmp/agent")
		t.Setenv(artifactsDirEnv, "/tmp/artifacts")
		t.Setenv(verifierDirEnv, "/tmp/verifier")
		t.Setenv(rewardPathEnv, "/tmp/reward.json")
		t.Setenv(rewardDetailsPathEnv, "/tmp/reward-details.json")

		if got := agentLogDir(); got != "/tmp/agent" {
			t.Fatalf("agentLogDir() = %q, want /tmp/agent", got)
		}
		if got := artifactsDir(); got != "/tmp/artifacts" {
			t.Fatalf("artifactsDir() = %q, want /tmp/artifacts", got)
		}
		if got := verifierDir(); got != "/tmp/verifier" {
			t.Fatalf("verifierDir() = %q, want /tmp/verifier", got)
		}
		if got := prDraftPath(); got != "/tmp/artifacts/pr.md" {
			t.Fatalf("prDraftPath() = %q, want /tmp/artifacts/pr.md", got)
		}
		if got := rewardPath(); got != "/tmp/reward.json" {
			t.Fatalf("rewardPath() = %q, want /tmp/reward.json", got)
		}
		if got := rewardDetailsPath(); got != "/tmp/reward-details.json" {
			t.Fatalf("rewardDetailsPath() = %q, want /tmp/reward-details.json", got)
		}
	})

	t.Run("falls back to defaults", func(t *testing.T) {
		t.Setenv(agentLogDirEnv, "")
		t.Setenv(artifactsDirEnv, "")
		t.Setenv(verifierDirEnv, "")
		t.Setenv(rewardPathEnv, "")
		t.Setenv(rewardDetailsPathEnv, "")

		if got := agentLogDir(); got != defaultAgentLogDir {
			t.Fatalf("agentLogDir() = %q, want %q", got, defaultAgentLogDir)
		}
		if got := artifactsDir(); got != defaultArtifactsDir {
			t.Fatalf("artifactsDir() = %q, want %q", got, defaultArtifactsDir)
		}
		if got := verifierDir(); got != defaultVerifierDir {
			t.Fatalf("verifierDir() = %q, want %q", got, defaultVerifierDir)
		}
		if got := prDraftPath(); got != defaultArtifactsDir+"/pr.md" {
			t.Fatalf("prDraftPath() = %q, want %q", got, defaultArtifactsDir+"/pr.md")
		}
		if got := rewardPath(); got != defaultRewardPath {
			t.Fatalf("rewardPath() = %q, want %q", got, defaultRewardPath)
		}
		if got := rewardDetailsPath(); got != defaultRewardDetails {
			t.Fatalf("rewardDetailsPath() = %q, want %q", got, defaultRewardDetails)
		}
	})
}

func TestScoreDependencyUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		text  string
		want  float64
	}{
		{
			name:  "root deps changed",
			files: []string{"go.mod", "go.sum"},
			want:  1.0,
		},
		{
			name: "update attempted without root dep changes",
			text: "runme run update-go-deps",
			want: 0.5,
		},
		{
			name:  "dagger deps changed too",
			files: []string{"go.mod", ".dagger/go.mod"},
			want:  0.5,
		},
		{
			name: "no evidence",
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := (scorer{files: tt.files, text: tt.text}).dependencyUpdate(); got != tt.want {
				t.Fatalf("dependencyUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreScopedChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		want  float64
	}{
		{
			name: "no changes",
			want: 1.0,
		},
		{
			name:  "allowed dependency and go changes",
			files: []string{"go.mod", "go.sum", "runner/session.go"},
			want:  1.0,
		},
		{
			name:  "eval regression changes are allowed",
			files: []string{"evals/tasks/update-minor-deps/tests/test.sh"},
			want:  1.0,
		},
		{
			name: "skill harness changes are allowed",
			files: []string{
				".agents/skills/update-minor-deps/README.md",
				".agents/skills/update-minor-deps/evals/README.md",
				"go.mod",
				"go.sum",
			},
			want: 1.0,
		},
		{
			name:  "one unrelated file gets partial credit",
			files: []string{"docs/usage.md"},
			want:  0.5,
		},
		{
			name:  "more than two unrelated files gets no credit",
			files: []string{"docs/usage.md", "README.md", "web/app.ts"},
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := (scorer{files: tt.files}).scopedChanges(); got != tt.want {
				t.Fatalf("scopedChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelevantChangedFiles(t *testing.T) {
	t.Parallel()

	files := []string{
		".agents/skills/update-minor-deps/README.md",
		"evals/tasks/update-minor-deps/tests/score.go",
		"go.mod",
		"go.sum",
	}
	got := relevantChangedFiles(files)
	want := []string{"go.mod", "go.sum"}

	if len(got) != len(want) {
		t.Fatalf("relevantChangedFiles() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("relevantChangedFiles() = %#v, want %#v", got, want)
		}
	}
}

func TestIsModuleOnlyChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{
			name:  "go mod only",
			files: []string{"go.mod"},
			want:  true,
		},
		{
			name: "go sum only with harness changes ignored",
			files: []string{
				".agents/skills/update-minor-deps/README.md",
				"evals/tasks/update-minor-deps/tests/score.go",
				"go.sum",
			},
			want: true,
		},
		{
			name:  "source file changed",
			files: []string{"go.mod", "runner/session.go"},
			want:  false,
		},
		{
			name:  "eval harness only",
			files: []string{"evals/tasks/update-minor-deps/tests/score.go"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isModuleOnlyChange(tt.files); got != tt.want {
				t.Fatalf("isModuleOnlyChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandsFromATIF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want []string
	}{
		{
			name: "codex style cmd argument",
			data: `{
				"schema_version": "1.5",
				"steps": [
					{
						"tool_calls": [
							{
								"function_name": "exec_command",
								"arguments": {
									"cmd": "runme run lint test",
									"workdir": "/repo"
								}
							}
						]
					}
				]
			}`,
			want: []string{"runme run lint test"},
		},
		{
			name: "claude style command argument",
			data: `{
				"schema_version": "1.2",
				"steps": [
					{
						"tool_calls": [
							{
								"function_name": "Bash",
								"arguments": {
									"command": "runme run test"
								}
							}
						]
					}
				]
			}`,
			want: []string{"runme run test"},
		},
		{
			name: "mixed tool calls collect commands in order",
			data: `{
				"schema_version": "1.5",
				"steps": [
					{
						"tool_calls": [
							{
								"function_name": "Read",
								"arguments": {
									"file_path": "CONTRIBUTING.md"
								}
							},
							{
								"function_name": "exec_command",
								"arguments": {
									"cmd": "git status --short"
								}
							}
						]
					},
					{
						"tool_calls": [
							{
								"function_name": "write_stdin",
								"arguments": {
									"chars": ""
								}
							},
							{
								"function_name": "Bash",
								"arguments": {
									"command": "go test ./runner"
								}
							},
							{
								"function_name": "Skill",
								"arguments": {
									"name": "update-minor-deps"
								}
							}
						]
					}
				]
			}`,
			want: []string{"git status --short", "go test ./runner"},
		},
		{
			name: "malformed json ignored",
			data: `not-json`,
			want: nil,
		},
		{
			name: "no extractable command calls",
			data: `{
				"schema_version": "1.5",
				"steps": [
					{
						"tool_calls": [
							{
								"function_name": "apply_patch",
								"arguments": {
									"patch": "*** Begin Patch"
								}
							},
							{
								"function_name": "exec_command",
								"arguments": {
									"cmd": "   "
								}
							}
						]
					}
				]
			}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := commandsFromATIF([]byte(tt.data))
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Fatalf("commandsFromATIF() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCollectAgentCommandsFromFile(t *testing.T) {
	t.Parallel()

	if got := collectAgentCommandsFromFile(filepath.Join(t.TempDir(), "missing.json")); got != "" {
		t.Fatalf("collectAgentCommandsFromFile(missing) = %q, want empty string", got)
	}

	path := filepath.Join(t.TempDir(), "trajectory.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"1.5","steps":[{"tool_calls":[{"function_name":"exec_command","arguments":{"cmd":"Runme Run Lint Test"}}]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := collectAgentCommandsFromFile(path); got != "runme run lint test" {
		t.Fatalf("collectAgentCommandsFromFile() = %q, want %q", got, "runme run lint test")
	}
}

func TestCollectAgentShellTraceCommands(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "oracle.txt"), []byte(`Using update-minor-deps skill workflow
+ git status --short
+ runme run update-go-deps
+ go mod tidy
+ runme run lint test
`), 0o600); err != nil {
		t.Fatal(err)
	}

	want := strings.Join([]string{
		"git status --short",
		"runme run update-go-deps",
		"go mod tidy",
		"runme run lint test",
	}, "\n")
	if got := collectAgentShellTraceCommandsFromFile(filepath.Join(dir, "oracle.txt")); got != want {
		t.Fatalf("collectAgentShellTraceCommandsFromFile() = %q, want %q", got, want)
	}
}

func TestCollectAgentCommandsRequiresOracleForShellTraceFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	trace := []byte("+ runme run lint test\n")
	if err := os.WriteFile(filepath.Join(dir, "agent.txt"), trace, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := collectAgentCommandsFromDir(dir); got != "" {
		t.Fatalf("collectAgentCommandsFromDir() = %q, want empty string without oracle log", got)
	}

	if err := os.WriteFile(filepath.Join(dir, "oracle.txt"), trace, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := collectAgentCommandsFromDir(dir); got != "runme run lint test" {
		t.Fatalf("collectAgentCommandsFromDir() = %q, want %q", got, "runme run lint test")
	}
}

func TestEvidenceScores(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{
			name: "workflow full credit",
			got: (scorer{
				text: "git status --short contributing.md runme run update-go-deps go mod tidy",
			}).workflowEvidence(),
			want: 1.0,
		},
		{
			name: "skill activation from skill name",
			got:  (scorer{text: "using update-minor-deps"}).skillActivationEvidence(),
			want: 1.0,
		},
		{
			name: "module-only validation final only",
			got:  (scorer{commands: "runme run lint test", files: []string{"go.mod", "go.sum"}}).validationEvidence(),
			want: 1.0,
		},
		{
			name: "module-only split runme validation",
			got:  (scorer{commands: "runme run lint\nrunme run test", files: []string{"go.mod", "go.sum"}}).validationEvidence(),
			want: 1.0,
		},
		{
			name: "module-only focused only",
			got:  (scorer{commands: "go test ./...", files: []string{"go.mod", "go.sum"}}).validationEvidence(),
			want: 0.4,
		},
		{
			name: "source change validation final only",
			got: (scorer{
				commands: "runme run lint test",
				files:    []string{"go.mod", "go.sum", "runner/session.go"},
			}).validationEvidence(),
			want: 0.6,
		},
		{
			name: "validation final output does not count as focused command",
			got: (scorer{
				commands: "runme run lint test\nTZ=UTC go test -ldflags=\"...\" ./...",
				files:    []string{"go.mod", "go.sum", "runner/session.go"},
			}).validationEvidence(),
			want: 0.6,
		},
		{
			name: "source change focused and final",
			got: (scorer{
				commands: "go test ./runner runme run lint test",
				files:    []string{"go.mod", "go.sum", "runner/session.go"},
			}).validationEvidence(),
			want: 1.0,
		},
		{
			name: "make lint test does not count as required runme validation",
			got:  (scorer{commands: "make lint test", files: []string{"go.mod", "go.sum"}}).validationEvidence(),
			want: 0.0,
		},
		{
			name: "no forbidden commands",
			got:  (scorer{commands: "wrote the draft PR summary"}).noRealPROrCommit(),
			want: 1.0,
		},
		{
			name: "git commit forbidden",
			got:  (scorer{commands: "git commit -s -m update"}).noRealPROrCommit(),
			want: 0.0,
		},
		{
			name: "documented git commit text is harmless when commands are clean",
			got:  (scorer{commands: "sed -n '1,240p' .agents/skills/update-minor-deps/SKILL.md"}).noRealPROrCommit(),
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("score = %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestScorePRDraftText(t *testing.T) {
	t.Parallel()

	fullDraft := `# chore: update minor and patch dependencies (2026-06-15)

Ran runme run update-go-deps and go mod tidy.
Updated go.mod and go.sum.
Compatibility fixes: none.
Focused tests: go test ./runner.
Final validation: runme run lint test.
`
	moduleOnlyDraft := `# chore: update minor and patch dependencies (2026-06-15)

Ran runme run update-go-deps and go mod tidy.
Updated go.mod and go.sum.
No compatibility code or test fixes required.
Final validation: runme run lint test.
`
	splitValidationDraft := `# chore: update minor and patch dependencies (2026-06-15)

Ran runme run update-go-deps and go mod tidy.
Updated go.mod and go.sum.
No compatibility code or test fixes required.
Validation: runme run lint and runme run test.
`

	tests := []struct {
		name  string
		text  string
		files []string
		want  float64
	}{
		{
			name: "empty draft",
			want: 0.0,
		},
		{
			name:  "complete draft",
			text:  fullDraft,
			files: []string{"go.mod", "go.sum", "runner/session.go"},
			want:  1.0,
		},
		{
			name:  "module-only draft without focused tests",
			text:  moduleOnlyDraft,
			files: []string{"go.mod", "go.sum"},
			want:  1.0,
		},
		{
			name:  "module-only draft with split runme validation",
			text:  splitValidationDraft,
			files: []string{"go.mod", "go.sum"},
			want:  1.0,
		},
		{
			name:  "non-module draft without focused tests",
			text:  moduleOnlyDraft,
			files: []string{"go.mod", "go.sum", "runner/session.go"},
			want:  8.0 / 9.0,
		},
		{
			name: "title only",
			text: "chore: update minor and patch dependencies",
			want: 1.0 / 9.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := (scorer{prDraftText: tt.text, files: tt.files}).prDraftQuality(); got != tt.want {
				t.Fatalf("prDraftQuality() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNegativeControlSourceChangeFinalOnlyFixture(t *testing.T) {
	t.Parallel()

	fixture := filepath.Join("fixtures", "source-change-final-only")
	files := readFixtureLines(t, filepath.Join(fixture, "changed_files.txt"))
	trajectory := readFixture(t, filepath.Join(fixture, "trajectory.json"))
	prDraft := readFixture(t, filepath.Join(fixture, "pr.md"))
	commands := strings.ToLower(strings.Join(commandsFromATIF([]byte(trajectory)), "\n"))
	text := strings.ToLower(trajectory + "\n" + prDraft)

	scores := (scorer{
		files:       files,
		text:        text,
		commands:    commands,
		prDraftText: prDraft,
	}).scores()
	if scores.ValidationEvidence != 0.6 {
		t.Fatalf("ValidationEvidence = %v, want 0.6", scores.ValidationEvidence)
	}
	if scores.PRDraftQuality != 8.0/9.0 {
		t.Fatalf("PRDraftQuality = %v, want %v", scores.PRDraftQuality, 8.0/9.0)
	}

	fullCreditChecks := map[string]float64{
		"DependencyUpdate":        scores.DependencyUpdate,
		"ScopedChanges":           scores.ScopedChanges,
		"SkillActivationEvidence": scores.SkillActivationEvidence,
		"WorkflowEvidence":        scores.WorkflowEvidence,
		"NoRealPROrCommit":        scores.NoRealPROrCommit,
	}
	for name, got := range fullCreditChecks {
		if got != 1.0 {
			t.Fatalf("%s = %v, want 1.0", name, got)
		}
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func readFixtureLines(t *testing.T, path string) []string {
	t.Helper()

	var lines []string
	for _, line := range strings.Split(readFixture(t, path), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
