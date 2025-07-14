//go:build !windows
// +build !windows

package docs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/agent/testutil"
)

func Test_MarkdownToCells(t *testing.T) {
	type testCase struct {
		name     string
		inFile   string
		expected []*parserv1.Cell
	}

	cases := []testCase{
		{
			name:   "simple",
			inFile: "testdoc.md",
			expected: []*parserv1.Cell{
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "# Section 1",
					Outputs:  nil,
				},
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "This is section 1",
					Outputs:  nil,
				},
				{
					Kind: parserv1.CellKind_CELL_KIND_CODE,
					Metadata: map[string]string{
						"runme.dev/name":          "package-main",
						"runme.dev/nameGenerated": "true",
					},
					TextRange: &parserv1.TextRange{
						Start: 38,
						End:   72,
					},
					LanguageId: "go",
					Value:      "package main\n\nfunc main() {\n...\n}",
					Outputs:    nil,
				},
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "Breaking text",
					Outputs:  nil,
				},
				{
					Kind: parserv1.CellKind_CELL_KIND_CODE,
					Metadata: map[string]string{
						"runme.dev/name":          "echo-hello",
						"runme.dev/nameGenerated": "true",
					},
					TextRange: &parserv1.TextRange{
						Start: 100,
						End:   121,
					},
					LanguageId: "bash",
					Value:      "echo \"Hello, World!\"",
					Outputs: []*parserv1.CellOutput{
						{
							Items: []*parserv1.CellOutputItem{
								{
									Data: []byte("hello, world!"),
								},
							},
						},
					},
				},
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "## Subsection",
					Outputs:  nil,
				},
			},
		},
		{
			name:   "list-nested",
			inFile: "list.md",
			expected: []*parserv1.Cell{
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "Test code cells nested in a list",
					Outputs:  nil,
				},
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "1. First command",
					Outputs:  nil,
				},
				{
					Kind: parserv1.CellKind_CELL_KIND_CODE,
					Metadata: map[string]string{
						"runme.dev/name":          "echo-1",
						"runme.dev/nameGenerated": "true",
					},
					TextRange: &parserv1.TextRange{
						Start: 68,
						End:   75,
					},
					LanguageId: "bash",
					Value:      "echo 1",
					Outputs:    nil,
				},
				{
					Kind:     parserv1.CellKind_CELL_KIND_MARKUP,
					Metadata: make(map[string]string),
					Value:    "2. Second command",
					Outputs:  nil,
				},
				{
					Kind: parserv1.CellKind_CELL_KIND_CODE,
					Metadata: map[string]string{
						"runme.dev/name":          "echo-2",
						"runme.dev/nameGenerated": "true",
					},
					TextRange: &parserv1.TextRange{
						Start: 117,
						End:   124,
					},
					LanguageId: "bash",
					Value:      "echo 2",
					Outputs:    nil,
				},
			},
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fPath := filepath.Join(cwd, "test_data", c.inFile)
			raw, err := os.ReadFile(fPath)
			if err != nil {
				t.Fatalf("Failed to read raw file: %v", err)
			}
			actual, err := MarkdownToCells(string(raw))
			if err != nil {
				t.Fatalf("MarkdownToCells(%v) returned error %v", c.inFile, err)
			}
			if len(actual) != len(c.expected) {
				t.Errorf("Expected %v cells got %v", len(c.expected), len(actual))
			}

			for i, eCell := range c.expected {
				if i >= len(actual) {
					break
				}

				aCell := actual[i]

				opts := cmp.Options{
					// ignore Id because it will be unique each time it gets run
					cmpopts.IgnoreFields(parserv1.Cell{}, "RefId"),
					// ignore TextRange because it varies depending on available space
					cmpopts.IgnoreFields(parserv1.Cell{}, "TextRange"),
				}

				// Zero out the metadata field for id
				delete(aCell.Metadata, "runme.dev/id")

				if d := cmp.Diff(eCell, aCell, testutil.CellComparer, opts); d != "" {
					t.Errorf("Unexpected diff cell %d:\n%s", i, d)
				}
			}
		})
	}
}

func Test_CellToMarkdown(t *testing.T) {
	type testCase struct {
		name      string
		cell      *parserv1.Cell
		maxLength int
		expected  string
	}

	testCases := []testCase{
		{
			name: "markup",
			cell: &parserv1.Cell{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "This is a test",
			},
			expected: "This is a test\n",
		},
		{
			name: "code",
			cell: &parserv1.Cell{
				Kind:  parserv1.CellKind_CELL_KIND_CODE,
				Value: "echo \"something something\"",
				Outputs: []*parserv1.CellOutput{
					{
						Items: []*parserv1.CellOutputItem{
							{
								Data: []byte("something something"),
							},
						},
					},
				},
			},
			expected: "```bash\necho \"something something\"\n```\n```output\nsomething something\n```\n",
		},
		{
			name: "filter-by-mime-type",
			cell: &parserv1.Cell{
				Kind:  parserv1.CellKind_CELL_KIND_CODE,
				Value: "echo \"something something\"",
				Outputs: []*parserv1.CellOutput{
					{
						Items: []*parserv1.CellOutputItem{
							{
								Data: []byte("Should be excluded"),
								Mime: StatefulRunmeOutputItemsMimeType,
							},
							{
								Data: []byte("Terminal be excluded"),
								Mime: StatefulRunmeTerminalMimeType,
							},
							{
								Data: []byte("Should be included"),
								Mime: VSCodeNotebookStdOutMimeType,
							},
						},
					},
				},
			},
			expected: "```bash\necho \"something something\"\n```\n```output\nShould be included\n```\n",
		},
		{
			name: "truncate-output",
			cell: &parserv1.Cell{
				Kind:  parserv1.CellKind_CELL_KIND_CODE,
				Value: "echo line1\nline2",
				Outputs: []*parserv1.CellOutput{
					{
						Items: []*parserv1.CellOutputItem{
							{
								Data: []byte("some really long output"),
							},
						},
					},
				},
			},
			maxLength: 10,
			expected:  "```bash\n<...code was truncated...>\nline2\n```\n```output\nsome r<...stdout was truncated...>\n```\n",
		},
	}
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			actual := CellToMarkdown(c.cell, c.maxLength)
			if d := cmp.Diff(c.expected, actual); d != "" {
				t.Errorf("Unexpected diff:\n%s", d)
			}
		})
	}
}
