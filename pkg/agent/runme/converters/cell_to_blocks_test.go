package converters

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/runmedev/runme/v3/pkg/agent/testutil"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

type testCase struct {
	name     string
	Notebook *parserv1.Notebook
	Doc      *Doc
}

var cases = []testCase{
	{
		name: "Simple",
		Notebook: &parserv1.Notebook{
			Cells: []*parserv1.Cell{
				{
					Metadata: map[string]string{
						"id":          "1234",
						"interactive": "false",
					},
					Kind:       parserv1.CellKind_CELL_KIND_CODE,
					LanguageId: "python",
					Value:      "print('Hello World')",
					Outputs: []*parserv1.CellOutput{
						{
							Items: []*parserv1.CellOutputItem{
								{
									Data: []byte("Hello World\n"),
									Mime: "text/plain",
								},
							},
						},
					},
				},
			},
		},
		Doc: &Doc{
			Blocks: []*agent.Block{
				{
					Id:       "1234",
					Language: "python",
					Contents: "print('Hello World')",
					Metadata: map[string]string{
						"id":          "1234",
						"interactive": "false",
					},
					Kind: agent.BlockKind_CODE,
					Outputs: []*agent.BlockOutput{
						{
							Items: []*agent.BlockOutputItem{
								{
									TextData: "Hello World\n",
									Mime:     "text/plain",
								},
							},
						},
					},
				},
			},
		},
	},
	{
		// This test case we don't set interactive explicitly.
		// It verifies its not getting added
		name: "no-interactive",
		Notebook: &parserv1.Notebook{
			Cells: []*parserv1.Cell{
				{
					Metadata: map[string]string{
						"id": "1234",
					},
					Kind:       parserv1.CellKind_CELL_KIND_CODE,
					LanguageId: "python",
					Value:      "print('Hello World')",
					Outputs: []*parserv1.CellOutput{
						{
							Items: []*parserv1.CellOutputItem{
								{
									Data: []byte("Hello World\n"),
									Mime: "text/plain",
								},
							},
						},
					},
				},
			},
		},
		Doc: &Doc{
			Blocks: []*agent.Block{
				{
					Id:       "1234",
					Language: "python",
					Contents: "print('Hello World')",
					Metadata: map[string]string{
						"id": "1234",
					},
					Kind: agent.BlockKind_CODE,
					Outputs: []*agent.BlockOutput{
						{
							Items: []*agent.BlockOutputItem{
								{
									TextData: "Hello World\n",
									Mime:     "text/plain",
								},
							},
						},
					},
				},
			},
		},
	},
}

func Test_NotebookToDoc(t *testing.T) {
	for i, c := range cases {
		t.Run(fmt.Sprintf("Case %d", i), func(t *testing.T) {
			actual, err := NotebookToDoc(c.Notebook)
			if err != nil {
				t.Errorf("Case %v: Error %v", i, err)
				return
			}

			if diff := cmp.Diff(c.Doc, actual, testutil.BlockComparer); diff != "" {
				t.Errorf("Unexpected Diff:\n%v", diff)
			}
		})
	}
}
