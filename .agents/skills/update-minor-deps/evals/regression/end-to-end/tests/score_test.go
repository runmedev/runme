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
			text: ".agents/skills/update-minor-deps/scripts/update-go-deps.sh",
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

			if got := scoreDependencyUpdate(tt.files, tt.text); got != tt.want {
				t.Fatalf("scoreDependencyUpdate() = %v, want %v", got, tt.want)
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
			files: []string{".agents/skills/update-minor-deps/evals/regression/end-to-end/tests/test.sh"},
			want:  1.0,
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

			if got := scoreScopedChanges(tt.files); got != tt.want {
				t.Fatalf("scoreScopedChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelevantChangedFiles(t *testing.T) {
	t.Parallel()

	files := []string{
		".agents/skills/update-minor-deps/evals/regression/end-to-end/tests/score.go",
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
			name:  "go sum only with eval harness ignored",
			files: []string{".agents/skills/update-minor-deps/evals/regression/end-to-end/tests/score.go", "go.sum"},
			want:  true,
		},
		{
			name:  "source file changed",
			files: []string{"go.mod", "runner/session.go"},
			want:  false,
		},
		{
			name:  "eval harness only",
			files: []string{".agents/skills/update-minor-deps/evals/regression/end-to-end/tests/score.go"},
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

func TestEvidenceScores(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{
			name: "workflow full credit",
			got: scoreWorkflowEvidence(
				"git status --short contributing.md .agents/skills/update-minor-deps/scripts/update-go-deps.sh go mod tidy",
			),
			want: 1.0,
		},
		{
			name: "skill activation from skill name",
			got:  scoreSkillActivationEvidence("using update-minor-deps"),
			want: 1.0,
		},
		{
			name: "module-only validation final only",
			got:  scoreValidationEvidence("runme run lint test", []string{"go.mod", "go.sum"}),
			want: 1.0,
		},
		{
			name: "module-only split runme validation",
			got:  scoreValidationEvidence("runme run lint\nrunme run test", []string{"go.mod", "go.sum"}),
			want: 1.0,
		},
		{
			name: "module-only focused only",
			got:  scoreValidationEvidence("go test ./...", []string{"go.mod", "go.sum"}),
			want: 0.4,
		},
		{
			name: "source change validation final only",
			got:  scoreValidationEvidence("runme run lint test", []string{"go.mod", "go.sum", "runner/session.go"}),
			want: 0.6,
		},
		{
			name: "validation final output does not count as focused command",
			got: scoreValidationEvidence(
				"runme run lint test\nTZ=UTC go test -ldflags=\"...\" ./...",
				[]string{"go.mod", "go.sum", "runner/session.go"},
			),
			want: 0.6,
		},
		{
			name: "source change focused and final",
			got:  scoreValidationEvidence("go test ./runner runme run lint test", []string{"go.mod", "go.sum", "runner/session.go"}),
			want: 1.0,
		},
		{
			name: "make lint test does not count as required runme validation",
			got:  scoreValidationEvidence("make lint test", []string{"go.mod", "go.sum"}),
			want: 0.0,
		},
		{
			name: "no forbidden commands",
			got:  scoreNoRealPROrCommit("wrote the draft PR summary"),
			want: 1.0,
		},
		{
			name: "git commit forbidden",
			got:  scoreNoRealPROrCommit("git commit -s -m update"),
			want: 0.0,
		},
		{
			name: "documented git commit text is harmless when commands are clean",
			got:  scoreNoRealPROrCommit("sed -n '1,240p' .agents/skills/update-minor-deps/SKILL.md"),
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

Ran .agents/skills/update-minor-deps/scripts/update-go-deps.sh.
Updated go.mod and go.sum.
Compatibility fixes: none.
Focused tests: go test ./runner.
Final validation: runme run lint test.
`
	moduleOnlyDraft := `# chore: update minor and patch dependencies (2026-06-15)

Ran .agents/skills/update-minor-deps/scripts/update-go-deps.sh.
Updated go.mod and go.sum.
No compatibility code or test fixes required.
Final validation: runme run lint test.
`
	splitValidationDraft := `# chore: update minor and patch dependencies (2026-06-15)

Ran .agents/skills/update-minor-deps/scripts/update-go-deps.sh.
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
			want:  0.875,
		},
		{
			name: "title only",
			text: "chore: update minor and patch dependencies",
			want: 0.125,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := scorePRDraftText(tt.text, tt.files); got != tt.want {
				t.Fatalf("scorePRDraftText() = %v, want %v", got, tt.want)
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

	scores := scoreRewards(files, text, commands, prDraft)
	if scores.ValidationEvidence != 0.6 {
		t.Fatalf("ValidationEvidence = %v, want 0.6", scores.ValidationEvidence)
	}
	if scores.PRDraftQuality != 0.875 {
		t.Fatalf("PRDraftQuality = %v, want 0.875", scores.PRDraftQuality)
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
