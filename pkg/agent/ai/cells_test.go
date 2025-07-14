package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openai/openai-go/responses"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

func NullOpSender(resp *agentv1.GenerateResponse) error {
	return nil
}

func Test_ProcessEvent(t *testing.T) {
	// This unittest is intended to ensure we properly accumulate events
	type testCase struct {
		name string
		// Preexisting cells
		cells map[string]*parserv1.Cell
		// Event to process
		event responses.ResponseStreamEventUnion
		// Expected cell after processing the event
		expectedCell *parserv1.Cell
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
			name:  "TextDelta-no-cell",
			cells: map[string]*parserv1.Cell{},
			event: *textDeltaEventUnion,
			expectedCell: &parserv1.Cell{
				RefId:    "abcd",
				Metadata: map[string]string{"id": "abcd", "runme.dev/id": "abcd"},
				Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
				Role:     parserv1.CellRole_CELL_ROLE_ASSISTANT,
				Value:    "world",
			},
		},
		{
			name: "TextDelta-accumulate",
			cells: map[string]*parserv1.Cell{
				"abcd": {
					RefId:    "abcd",
					Metadata: map[string]string{"id": "abcd", "runme.dev/id": "abcd"},
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Value:    "hello",
				},
			},
			event: *textDeltaEventUnion,
			expectedCell: &parserv1.Cell{
				RefId:    "abcd",
				Metadata: map[string]string{"id": "abcd", "runme.dev/id": "abcd"},
				Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
				Value:    "helloworld",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := &CellsBuilder{
				cells: tc.cells,
			}
			if err := b.ProcessEvent(context.TODO(), tc.event, NullOpSender); err != nil {
				t.Fatalf("Failed to process event: %+v", err)
			}
			actual, ok := b.cells[tc.expectedCell.RefId]
			if !ok {
				t.Fatalf("Cell %s not found", tc.expectedCell.RefId)
			}

			opts := cmpopts.IgnoreUnexported(parserv1.Cell{})
			if d := cmp.Diff(tc.expectedCell, actual, opts); d != "" {
				t.Fatalf("Unexpected diff in cell cell:\n%s", d)
			}
		})
	}
}
