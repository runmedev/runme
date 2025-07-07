package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openai/openai-go/responses"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
)

func NullOpSender(resp *agentv1.GenerateResponse) error {
	return nil
}

func Test_ProcessEvent(t *testing.T) {
	// This unittest is intended to ensure we properly accumulate events
	type testCase struct {
		name string
		// Preexisting blocks
		blocks map[string]*agentv1.Block
		// Event to process
		event responses.ResponseStreamEventUnion
		// Expected block after processing the event
		expectedBlock *agentv1.Block
	}

	textDeltaEvent := responses.ResponseTextDeltaEvent{
		Delta:  "world",
		ItemID: "abcd",
	}

	textDeltaEventBytes, err := json.Marshal(textDeltaEvent)
	if err != nil {
		t.Fatalf("Failed to marshal textDeltaEvent: %+v", err)
	}

	textDeltaEventUnion := &responses.ResponseStreamEventUnion{}

	if err := textDeltaEventUnion.UnmarshalJSON(textDeltaEventBytes); err != nil {
		t.Fatalf("Failed to unmarshal textDeltaEvent: %+v", err)
	}

	testCases := []testCase{
		{
			name:   "TextDelta-no-block",
			blocks: map[string]*agentv1.Block{},
			event:  *textDeltaEventUnion,
			expectedBlock: &agentv1.Block{
				Id:       "abcd",
				Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
				Role:     agentv1.BlockRole_BLOCK_ROLE_ASSISTANT,
				Contents: "world",
			},
		},
		{
			name: "TextDelta-accumulate",
			blocks: map[string]*agentv1.Block{
				"abcd": {
					Id:       "abcd",
					Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
					Contents: "hello",
				},
			},
			event: *textDeltaEventUnion,
			expectedBlock: &agentv1.Block{
				Id:       "abcd",
				Kind:     agentv1.BlockKind_BLOCK_KIND_MARKUP,
				Contents: "helloworld",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &BlocksBuilder{
				blocks: tc.blocks,
			}
			if err := b.ProcessEvent(context.TODO(), tc.event, NullOpSender); err != nil {
				t.Fatalf("Failed to process event: %+v", err)
			}
			actual, ok := b.blocks[tc.expectedBlock.Id]
			if !ok {
				t.Fatalf("Block %s not found", tc.expectedBlock.Id)
			}

			opts := cmpopts.IgnoreUnexported(agentv1.Block{})
			if d := cmp.Diff(tc.expectedBlock, actual, opts); d != "" {
				t.Fatalf("Unexpected diff in block block:\n%s", d)
			}
		})
	}
}
