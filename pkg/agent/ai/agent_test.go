package ai

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	lru "github.com/hashicorp/golang-lru/v2"
	"google.golang.org/protobuf/testing/protocmp"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

func TestFillInToolcalls(t *testing.T) {
	// Define test cases
	tests := []struct {
		name               string
		previousResponseId string
		cachedResponses    map[string][]string
		cachedCells        map[string]*parserv1.Cell
		request            *agentv1.GenerateRequest
		expected           *agentv1.GenerateRequest
	}{
		{
			name:               "Missing Previous Calls",
			previousResponseId: "abc",
			cachedResponses: map[string][]string{
				"abc": {"block1"},
			},
			cachedCells: map[string]*parserv1.Cell{
				"block1": {
					Id:     "block1",
					Kind:   parserv1.CellKind_CELL_KIND_CODE,
					Value:  "print('Hello, world!')",
					CallId: "call1",
				},
			},
			request: &agentv1.GenerateRequest{
				PreviousResponseId: "abc",
			},
			expected: &agentv1.GenerateRequest{
				PreviousResponseId: "abc",
				Cells: []*parserv1.Cell{
					{
						Id:     "block1",
						Kind:   parserv1.CellKind_CELL_KIND_CODE,
						Value:  "print('Hello, world!')",
						CallId: "call1",
					},
				},
			},
		},
		{
			name:               "Has Previous Calls",
			previousResponseId: "abc",
			cachedResponses: map[string][]string{
				"abc": {"block1"},
			},
			cachedCells: map[string]*parserv1.Cell{
				"block1": {
					Id:     "block1",
					Kind:   parserv1.CellKind_CELL_KIND_CODE,
					Value:  "print('This was the original command!')",
					CallId: "call1",
				},
			},
			request: &agentv1.GenerateRequest{
				PreviousResponseId: "abc",
				Cells: []*parserv1.Cell{
					// We want to ensure that the cell in the request takes precendence over the cache
					{
						Id:     "block1",
						Kind:   parserv1.CellKind_CELL_KIND_CODE,
						Value:  "print('Actual Command')",
						CallId: "call1",
					},
				},
			},
			expected: &agentv1.GenerateRequest{
				PreviousResponseId: "abc",
				Cells: []*parserv1.Cell{
					{
						Id:     "block1",
						Kind:   parserv1.CellKind_CELL_KIND_CODE,
						Value:  "print('Actual Command')",
						CallId: "call1",
					},
				},
			},
		},
	}

	// Run each test case
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize cache with a size large enough for the test
			responseCache, err := lru.New[string, []string](5)
			if err != nil {
				t.Fatalf("Failed to create response cache: %v", err)
			}

			blocksCache, err := lru.New[string, *parserv1.Cell](5)
			if err != nil {
				t.Fatalf("Failed to create blocks cache: %v", err)
			}

			// Populate response cache
			for respID, calls := range tc.cachedResponses {
				responseCache.Add(respID, calls)
			}

			// Populate blocks cache
			for blockID, block := range tc.cachedCells {
				blocksCache.Add(blockID, block)
			}

			if err := fillInToolcalls(context.Background(), responseCache, blocksCache, tc.request); err != nil {
				t.Fatalf("Failed to fill in tool calls: %v", err)
			}

			// Check if the request matches the expected result
			if d := cmp.Diff(tc.expected, tc.request, protocmp.Transform()); d != "" {
				t.Errorf("Request mismatch (-want +got):\n%s", d)
			}
		})
	}
}
