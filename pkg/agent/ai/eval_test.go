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
	"github.com/runmedev/runme/v3/pkg/agent/application"
)

func TestAssertions(t *testing.T) {
	type asserter interface {
		Assert(ctx context.Context, assertion *agentv1.Assertion, inputText string, blocks map[string]*agentv1.Block) error
	}

	type testCase struct {
		name              string
		asserter          asserter
		assertion         *agentv1.Assertion
		blocks            map[string]*agentv1.Block
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
			blocks: map[string]*agentv1.Block{
				"1": {
					Kind:     agentv1.BlockKind_BLOCK_KIND_CODE,
					Contents: "kubectl get pods --context test -n default",
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
			blocks: map[string]*agentv1.Block{
				"1": {
					Kind:     agentv1.BlockKind_BLOCK_KIND_CODE,
					Contents: "kubectl get pods --context test",
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
			blocks: map[string]*agentv1.Block{
				"block1": {
					Kind: agentv1.BlockKind_BLOCK_KIND_FILE_SEARCH_RESULTS,
					FileSearchResults: []*agentv1.FileSearchResult{
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
			blocks: map[string]*agentv1.Block{
				"block1": {
					Kind: agentv1.BlockKind_BLOCK_KIND_FILE_SEARCH_RESULTS,
					FileSearchResults: []*agentv1.FileSearchResult{
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
			blocks: map[string]*agentv1.Block{
				"1": {
					Kind:     agentv1.BlockKind_BLOCK_KIND_CODE,
					Contents: "echo hello world",
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
			blocks: map[string]*agentv1.Block{
				"1": {
					Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
					Contents: "This is not a code block.",
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
			blocks: map[string]*agentv1.Block{
				"1": {
					Kind:     agentv1.BlockKind_BLOCK_KIND_CODE,
					Contents: "az aks list --query \"[?name=='unified-60'].{Name:name, Location:location}\" --output table",
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
