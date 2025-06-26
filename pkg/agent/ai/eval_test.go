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

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent"
	"github.com/runmedev/runme/v3/pkg/agent/application"
)

func TestAssertions(t *testing.T) {
	type asserter interface {
		Assert(ctx context.Context, assertion *agent.Assertion, inputText string, blocks map[string]*agent.Block) error
	}

	type testCase struct {
		name              string
		asserter          asserter
		assertion         *agent.Assertion
		blocks            map[string]*agent.Block
		expectedAssertion *agent.Assertion
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

	app := application.NewApp()
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
			assertion: &agent.Assertion{
				Name: "test-pass",
				Type: agent.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agent.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agent.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
			},
			blocks: map[string]*agent.Block{
				"1": {
					Kind:     agent.BlockKind_CODE,
					Contents: "kubectl get pods --context test -n default",
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "test-pass",
				Type: agent.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agent.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agent.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
				Result: agent.Assertion_RESULT_TRUE,
			},
		},
		{
			name:     "kubectl-required-flag-missing",
			asserter: shellRequiredFlag{},
			assertion: &agent.Assertion{
				Name: "test-fail",
				Type: agent.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agent.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agent.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
			},
			blocks: map[string]*agent.Block{
				"1": {
					Kind:     agent.BlockKind_CODE,
					Contents: "kubectl get pods --context test",
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "test-fail",
				Type: agent.Assertion_TYPE_SHELL_REQUIRED_FLAG,
				Payload: &agent.Assertion_ShellRequiredFlag_{
					ShellRequiredFlag: &agent.Assertion_ShellRequiredFlag{
						Command: "kubectl",
						Flags:   []string{"--context", "-n"},
					},
				},
				Result: agent.Assertion_RESULT_FALSE,
			},
		},
		{
			name:     "file-search-file-found",
			asserter: fileRetrieved{},
			assertion: &agent.Assertion{
				Name: "file-found",
				Type: agent.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agent.Assertion_FileRetrieval_{
					FileRetrieval: &agent.Assertion_FileRetrieval{
						FileId:   "file-123",
						FileName: "test.txt",
					},
				},
			},
			blocks: map[string]*agent.Block{
				"block1": {
					Kind: agent.BlockKind_FILE_SEARCH_RESULTS,
					FileSearchResults: []*agent.FileSearchResult{
						{FileID: "file-123", FileName: "test.txt"},
					},
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "file-found",
				Type: agent.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agent.Assertion_FileRetrieval_{
					FileRetrieval: &agent.Assertion_FileRetrieval{
						FileId:   "file-123",
						FileName: "test.txt",
					},
				},
				Result: agent.Assertion_RESULT_TRUE,
			},
		},
		{
			name:     "file-search-file-not-found",
			asserter: fileRetrieved{},
			assertion: &agent.Assertion{
				Name: "file-not-found",
				Type: agent.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agent.Assertion_FileRetrieval_{
					FileRetrieval: &agent.Assertion_FileRetrieval{
						FileId:   "file-999",
						FileName: "notfound.txt",
					},
				},
			},
			blocks: map[string]*agent.Block{
				"block1": {
					Kind: agent.BlockKind_FILE_SEARCH_RESULTS,
					FileSearchResults: []*agent.FileSearchResult{
						{FileID: "file-123", FileName: "test.txt"},
					},
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "file-not-found",
				Type: agent.Assertion_TYPE_FILE_RETRIEVED,
				Payload: &agent.Assertion_FileRetrieval_{
					FileRetrieval: &agent.Assertion_FileRetrieval{
						FileId:   "file-999",
						FileName: "notfound.txt",
					},
				},
				Result: agent.Assertion_RESULT_FALSE,
			},
		},
		{
			name:     "tool-invocation-shell-command",
			asserter: toolInvocation{},
			assertion: &agent.Assertion{
				Name: "shell-invoked",
				Type: agent.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agent.Assertion_ToolInvocation_{
					ToolInvocation: &agent.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
			},
			blocks: map[string]*agent.Block{
				"1": {
					Kind:     agent.BlockKind_CODE,
					Contents: "echo hello world",
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "shell-invoked",
				Type: agent.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agent.Assertion_ToolInvocation_{
					ToolInvocation: &agent.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
				Result: agent.Assertion_RESULT_TRUE,
			},
		},
		{
			name:     "tool-invocation-no-shell-command",
			asserter: toolInvocation{},
			assertion: &agent.Assertion{
				Name: "shell-not-invoked",
				Type: agent.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agent.Assertion_ToolInvocation_{
					ToolInvocation: &agent.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
			},
			blocks: map[string]*agent.Block{
				"1": {
					Kind:     agent.BlockKind_MARKUP,
					Contents: "This is not a code block.",
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "shell-not-invoked",
				Type: agent.Assertion_TYPE_TOOL_INVOKED,
				Payload: &agent.Assertion_ToolInvocation_{
					ToolInvocation: &agent.Assertion_ToolInvocation{
						ToolName: "shell",
					},
				},
				Result: agent.Assertion_RESULT_FALSE,
			},
		},
		{
			name:     "llm-judge-basic",
			asserter: llmJudge{client: client},
			assertion: &agent.Assertion{
				Name: "basic_llm_judge",
				Type: agent.Assertion_TYPE_LLM_JUDGE,
				Payload: &agent.Assertion_LlmJudge{
					LlmJudge: &agent.Assertion_LLMJudge{
						Prompt: "Do you think the LLM's command is mostly correct?",
					},
				},
			},
			blocks: map[string]*agent.Block{
				"1": {
					Kind:     agent.BlockKind_CODE,
					Contents: "az aks list --query \"[?name=='unified-60'].{Name:name, Location:location}\" --output table",
				},
			},
			expectedAssertion: &agent.Assertion{
				Name: "basic_llm_judge",
				Type: agent.Assertion_TYPE_LLM_JUDGE,
				Payload: &agent.Assertion_LlmJudge{
					LlmJudge: &agent.Assertion_LLMJudge{
						Prompt: "Do you think the LLM's command is mostly correct?",
					},
				},
				Result: agent.Assertion_RESULT_TRUE,
			},
			inputText: "What region is cluster unified-60 in?",
		},
	}

	log := zapr.NewLogger(zap.L())
	ctx := logr.NewContext(context.Background(), log)
	opts := cmp.Options{
		cmpopts.IgnoreUnexported(
			agent.Assertion{},
			agent.Assertion_ShellRequiredFlag{},
			agent.Assertion_ToolInvocation{},
			agent.Assertion_FileRetrieval{},
			agent.Assertion_CodeblockRegex{},
			agent.Assertion_LLMJudge{},
		),
		cmpopts.IgnoreFields(agent.Assertion{}, "FailureReason"),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.asserter.Assert(ctx, tc.assertion, tc.inputText, tc.blocks)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d := cmp.Diff(tc.expectedAssertion, tc.assertion, opts); d != "" {
				t.Fatalf("unexpected diff in assertion (-want +got):\n%s", d)
			}
		})
	}
}
