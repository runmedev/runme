package ai

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/pkg/agent/ai/e2etests"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/agent/application"
)

func TestAssertions(t *testing.T) {
	type asserter interface {
		Assert(ctx context.Context, assertion *agentv1.Assertion, inputText string, cells map[string]*parserv1.Cell) error
	}

	type testCase struct {
		name              string
		asserter          asserter
		assertion         *agentv1.Assertion
		cells             map[string]*parserv1.Cell
		expectedAssertion *agentv1.Assertion
		inputText         string
	}
	isGHA := os.Getenv("GITHUB_ACTIONS") == "true"
	if !isGHA {
		// todo(sebastian): can't run due to lack of access to O3
		e2etests.SkipIfMissing(t, "RUN_MANUAL_TESTS")
	}
	ghaOwner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	if ghaOwner == "runmedev" {
		t.Skip("Skipping eval tests in runmedev repository")
	}

	// todo(sebastian): we might want to use a different app name for agents/evals
	app := application.NewApp("runme-agent-test")
	if err := app.LoadConfig(nil); err != nil {
		t.Fatal(err)
	}
	cfg := app.GetConfig()
	var apiKey string
	if !isGHA {
		// When running locally create the OpenAI client using the config
		apiKeyFile := cfg.OpenAI.APIKeyFile
		apiKeyBytes, err := os.ReadFile(apiKeyFile)
		if err != nil {
			t.Fatalf("Failed to read API key file; %v", err)
		}
		apiKey = string(apiKeyBytes)
	} else {
		// In GHA we get the API key from the environment variable
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			t.Fatal("OPENAI_API_KEY environment variable is not set")
		}
	}
	apiKey = strings.TrimSpace(apiKey)
	client, err := NewClientWithKey(apiKey)
	if err != nil {
		t.Fatalf("Failed to create client from API key; %v", err)
	}

	testCases := []testCase{
		{
			name:     "kubectl-required-flags-present",
			asserter: shellRequiredFlag{},
			assertion: &agentv1.Assertion{
				Name: "test-pass",
				Type: agentv1.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agentv1.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agentv1.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"1": {
					Kind:       parserv1.CellKind_CELL_KIND_CODE,
					Value:      "kubectl get pods --context test -n default",
					LanguageId: "bash",
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "test-pass",
				Type: agentv1.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agentv1.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agentv1.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
				Result: agentv1.Assertion_RESULT_TRUE,
			},
		},
		{
			name:     "kubectl-required-flag-missing",
			asserter: shellRequiredFlag{},
			assertion: &agentv1.Assertion{
				Name: "test-fail",
				Type: agentv1.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agentv1.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agentv1.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"1": {
					Kind:       parserv1.CellKind_CELL_KIND_CODE,
					Value:      "kubectl get pods --context test",
					LanguageId: "bash",
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "test-fail",
				Type: agentv1.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agentv1.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agentv1.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
				Result: agentv1.Assertion_RESULT_FALSE,
			},
		},
		{
			name:     "file-search-file-found",
			asserter: fileRetrieved{},
			assertion: &agentv1.Assertion{
				Name: "file-found",
				Type: agentv1.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agentv1.Assertion_FileRetrieval_{
					FileRetrieval: &agentv1.Assertion_FileRetrieval{
						FileId:   "file-123",
						FileName: "test.txt",
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"cell1": {
					Kind: parserv1.CellKind_CELL_KIND_DOC_RESULTS,
					DocResults: []*parserv1.DocResult{
						{FileId: "file-123", FileName: "test.txt"},
					},
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "file-found",
				Type: agentv1.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agentv1.Assertion_FileRetrieval_{
					FileRetrieval: &agentv1.Assertion_FileRetrieval{
						FileId:   "file-123",
						FileName: "test.txt",
					},
				},
				Result: agentv1.Assertion_RESULT_TRUE,
			},
		},
		{
			name:     "file-search-file-not-found",
			asserter: fileRetrieved{},
			assertion: &agentv1.Assertion{
				Name: "file-not-found",
				Type: agentv1.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agentv1.Assertion_FileRetrieval_{
					FileRetrieval: &agentv1.Assertion_FileRetrieval{
						FileId:   "file-999",
						FileName: "notfound.txt",
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"cell1": {
					Kind: parserv1.CellKind_CELL_KIND_DOC_RESULTS,
					DocResults: []*parserv1.DocResult{
						{FileId: "file-123", FileName: "test.txt"},
					},
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "file-not-found",
				Type: agentv1.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agentv1.Assertion_FileRetrieval_{
					FileRetrieval: &agentv1.Assertion_FileRetrieval{
						FileId:   "file-999",
						FileName: "notfound.txt",
					},
				},
				Result: agentv1.Assertion_RESULT_FALSE,
			},
		},
		{
			name:     "tool-invocation-shell-command",
			asserter: toolInvocation{},
			assertion: &agentv1.Assertion{
				Name: "shell-invoked",
				Type: agentv1.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agentv1.Assertion_ToolInvocation_{
					ToolInvocation: &agentv1.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"1": {
					Kind:       parserv1.CellKind_CELL_KIND_CODE,
					Value:      "echo hello world",
					LanguageId: "bash",
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "shell-invoked",
				Type: agentv1.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agentv1.Assertion_ToolInvocation_{
					ToolInvocation: &agentv1.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
				Result: agentv1.Assertion_RESULT_TRUE,
			},
		},
		{
			name:     "tool-invocation-no-shell-command",
			asserter: toolInvocation{},
			assertion: &agentv1.Assertion{
				Name: "shell-not-invoked",
				Type: agentv1.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agentv1.Assertion_ToolInvocation_{
					ToolInvocation: &agentv1.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"1": {
					Kind:       parserv1.CellKind_CELL_KIND_MARKUP,
					Value:      "This is not a code cell.",
					LanguageId: "markdown",
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "shell-not-invoked",
				Type: agentv1.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agentv1.Assertion_ToolInvocation_{
					ToolInvocation: &agentv1.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
				Result: agentv1.Assertion_RESULT_FALSE,
			},
		},
		{
			name:     "llm-judge-basic",
			asserter: llmJudge{client: client},
			assertion: &agentv1.Assertion{
				Name: "basic_llm_judge",
				Type: agentv1.Assertion_TYPE_LLM_JUDGE,
				Payload: &agentv1.Assertion_LlmJudge{
					LlmJudge: &agentv1.Assertion_LLMJudge{
						Prompt: "Do you think the LLM's command is mostly correct?",
					},
				},
			},
			cells: map[string]*parserv1.Cell{
				"1": {
					Kind:       parserv1.CellKind_CELL_KIND_CODE,
					Value:      "az aks list --query \"[?name=='unified-60'].{Name:name, Location:location}\" --output table",
					LanguageId: "bash",
				},
			},
			expectedAssertion: &agentv1.Assertion{
				Name: "basic_llm_judge",
				Type: agentv1.Assertion_TYPE_LLM_JUDGE,
				Payload: &agentv1.Assertion_LlmJudge{
					LlmJudge: &agentv1.Assertion_LLMJudge{
						Prompt: "Do you think the LLM's command is mostly correct?",
					},
				},
				Result: agentv1.Assertion_RESULT_TRUE,
			},
			inputText: "What region is cluster unified-60 in?",
		},
	}

	log := zapr.NewLogger(zap.L())
	ctx := logr.NewContext(context.Background(), log)
	opts := cmp.Options{
		cmpopts.IgnoreUnexported(
			agentv1.Assertion{},
			agentv1.Assertion_ShellRequiredFlag{},
			agentv1.Assertion_ToolInvocation{},
			agentv1.Assertion_FileRetrieval{},
			agentv1.Assertion_CodeblockRegex{},
			agentv1.Assertion_LLMJudge{},
		),
		cmpopts.IgnoreFields(agentv1.Assertion{}, "FailureReason"),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.asserter.Assert(ctx, tc.assertion, tc.inputText, tc.cells)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d := cmp.Diff(tc.expectedAssertion, tc.assertion, opts); d != "" {
				t.Fatalf("unexpected diff in assertion (-want +got):\n%s", d)
			}
		})
	}
}
