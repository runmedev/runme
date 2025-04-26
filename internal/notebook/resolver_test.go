package notebook

import (
	"context"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"

	parserv1 "github.com/runmedev/runme/v3/pkg/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/document"
	"github.com/runmedev/runme/v3/pkg/document/identity"
)

func TestResolve_GetCellIndexByBlock(t *testing.T) {
	simpleSource := []byte("---\nrunme:\n  id: 01JJDCG2SQSGV0DP55XCR55AYM\n  version: v3\nshell: dagger shell\nterminalRows: 20\n---\n\n# Compose Notebook Pipelines using the Dagger Shell\n\nLet's get upstream artifacts ready. First, compile the Runme kernel binary.\n\n```sh {\"id\":\"01JJDCG2SPRDWGQ1F4Z6EH69EJ\",\"name\":\"KERNEL_BINARY\"}\ngithub.com/purpleclay/daggerverse/golang $(git https://github.com/runmedev/runme | head | tree) |\n  build | \n  file runme\n```\n\nThen, grab the presetup.sh script to provision the build container.\n\n```sh {\"id\":\"01JJDCG2SQSGV0DP55X86EJFSZ\",\"name\":\"PRESETUP\",\"terminalRows\":\"14\"}\ngit https://github.com/stateful/vscode-runme |\n  head |\n  tree |\n  file dagger/scripts/presetup.sh\n```\n\n## Build the Runme VS Code Extension\n\nLet's tie together above's artifacts via their respective cell names to build the Runme VS Code extension.\n\n```sh {\"id\":\"01JJDCG2SQSGV0DP55X8JVYDNR\",\"name\":\"EXTENSION\",\"terminalRows\":\"25\"}\ngithub.com/stateful/vscode-runme |\n  with-remote github.com/stateful/vscode-runme main |\n  with-container $(KERNEL_BINARY) $(PRESETUP) |\n  build-extension GITHUB_TOKEN\n```\n")

	resolver, err := NewResolver(WithSource(simpleSource))
	require.NoError(t, err)

	doc := document.New(simpleSource, identity.NewResolver(identity.DefaultLifecycleIdentity))
	require.NoError(t, err)
	require.NotNil(t, doc)

	node, err := doc.Root()
	require.NoError(t, err)
	require.Len(t, node.Children(), 8)

	expectedValue := []byte("git https://github.com/stateful/vscode-runme |\n  head |\n  tree |\n  file dagger/scripts/presetup.sh")
	expectedIndex := uint32(4)

	block, ok := node.Children()[expectedIndex].Item().(*document.CodeBlock)
	require.True(t, ok)
	require.NotNil(t, block)
	require.Equal(t, expectedValue, block.Content())

	index, err := resolver.GetCellIndexByBlock(block)
	require.NoError(t, err)
	assert.Equal(t, expectedIndex, index)
}

func TestResolveDaggerShell(t *testing.T) {
	ctx := context.Background()

	// fake notebook with dagger shell cells
	daggerShellNotebook := &parserv1.Notebook{
		Cells: []*parserv1.Cell{
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"id":             "01JJDCG2SPRDWGQ1F4Z6EH69EJ",
					"name":           "KERNEL_BINARY",
					"runme.dev/id":   "01JJDCG2SPRDWGQ1F4Z6EH69EJ",
					"runme.dev/name": "KERNEL_BINARY",
				},
				Value: "github.com/purpleclay/daggerverse/golang $(git https://github.com/runmedev/runme | head | tree) |\n  build | \n  file runme",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"id":             "01JJDCG2SQSGV0DP55X86EJFSZ",
					"name":           "PRESETUP",
					"runme.dev/id":   "01JJDCG2SQSGV0DP55X86EJFSZ",
					"runme.dev/name": "PRESETUP",
				},
				Value: "git https://github.com/stateful/vscode-runme |\n  head |\n  tree |\n  file dagger/scripts/presetup.sh",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"id":             "01JJDCG2SQSGV0DP55X8JVYDNR",
					"name":           "EXTENSION",
					"runme.dev/id":   "01JJDCG2SQSGV0DP55X8JVYDNR",
					"runme.dev/name": "EXTENSION",
				},
				Value: "github.com/stateful/vscode-runme |\n  with-remote github.com/stateful/vscode-runme main |\n  with-container $(KERNEL_BINARY) $(PRESETUP) |\n  build-extension GITHUB_TOKEN",
			},
		},
		Metadata: map[string]string{
			"runme.dev/frontmatter": "---\nrunme:\n  id: 01JJDCG2SQSGV0DP55XCR55AYM\n  version: v3\nshell: dagger shell\nterminalRows: 20\n---",
		},
	}

	resolver, err := NewResolver(WithNotebook(daggerShellNotebook))
	require.NoError(t, err)

	definition := `#!/usr/bin/env dagger shell
KERNEL_BINARY()
{
  github.com/purpleclay/daggerverse/golang $(git https://github.com/runmedev/runme | head | tree) \
    | build \
    | file runme
}
PRESETUP()
{
  git https://github.com/stateful/vscode-runme | head | tree \
    | file dagger/scripts/presetup.sh
}
EXTENSION()
{
  github.com/stateful/vscode-runme | with-remote github.com/stateful/vscode-runme main | with-container $(KERNEL_BINARY) $(PRESETUP) | build-extension GITHUB_TOKEN
}
`

	expectedScripts := []string{
		definition + "KERNEL_BINARY\n",
		definition + "PRESETUP\n",
		definition + "EXTENSION\n",
	}

	for cellIndex, expectedScript := range expectedScripts {
		script, err := resolver.ResolveDaggerShell(ctx, uint32(cellIndex))
		require.NoError(t, err)
		assert.Equal(t, expectedScript, script)
	}
}

func TestResolveDaggerShell_CellDaggerShell(t *testing.T) {
	ctx := context.Background()

	// fake notebook with mixed cells
	daggerShellNotebook := &parserv1.Notebook{
		Cells: []*parserv1.Cell{
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "## Mixed with System Shell as Default",
			},
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "Dagger Call:",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"runme.dev/id":            "01JPR4JA36V7M9ZTVZBZB2DPF5",
					"runme.dev/name":          "dagger-core",
					"runme.dev/nameGenerated": "true",
				},
				Value: "dagger core \\\n  git --url github.com/runmedev/vscode-runme \\\n    tag --name main \\\n    tree \\\n        file --path dagger/scripts/presetup.sh",
			},
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "Dagger Shell:",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"interpreter":             "dagger shell",
					"name":                    "Presetup",
					"runme.dev/id":            "01JPR4JA37X07H529VAC49HEZJ",
					"runme.dev/name":          "Presetup",
					"runme.dev/nameGenerated": "false",
				},
				Value: "git github.com/runmedev/vscode-runme |\n  tag main |\n  tree |\n    file dagger/scripts/presetup.sh",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"interpreter":             "dagger shell",
					"runme.dev/id":            "01JPR4JA37X07H529VADC0XFRF",
					"runme.dev/name":          "echo-presetup",
					"runme.dev/nameGenerated": "true",
				},
				Value: ".echo $(Presetup)",
			},
		},
		Metadata: map[string]string{
			"runme.dev/cacheId":         "01JPR4JA37X07H529VAG28EPBG",
			"runme.dev/finalLineBreaks": "0",
			"runme.dev/frontmatter":     "---\nterminalRows: 10\n---",
			"runme.dev/id":              "01JPR4JA37X07H529VAG28EPBG",
		},
		Frontmatter: &parserv1.Frontmatter{
			TerminalRows: "10",
		},
	}

	resolver, err := NewResolver(WithNotebook(daggerShellNotebook))
	require.NoError(t, err)

	// Test that the resolver can resolve vanilla shell
	script, err := resolver.ResolveDaggerShell(ctx, uint32(2))
	require.NoError(t, err)
	assert.Equal(t, "dagger core \\\n  git --url github.com/runmedev/vscode-runme \\\n    tag --name main \\\n    tree \\\n        file --path dagger/scripts/presetup.sh", script)

	// Test that the resolver can resolve dagger shell, skipping vanilla shell cells
	script, err = resolver.ResolveDaggerShell(ctx, uint32(4))
	require.NoError(t, err)
	assert.Equal(t, "#!/usr/bin/env dagger shell\nPresetup()\n{\n  git github.com/runmedev/vscode-runme \\\n    | tag main \\\n    | tree \\\n    | file dagger/scripts/presetup.sh\n}\nDAGGER_01JPR4JA37X07H529VADC0XFRF()\n{\n  .echo $(Presetup)\n}\nPresetup\n", script)
}

func TestResolveDaggerShell_FrontmatterWithVanillaCells(t *testing.T) {
	ctx := context.Background()

	// fake notebook with mixed cells
	daggerShellNotebook := &parserv1.Notebook{
		Cells: []*parserv1.Cell{
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "## Mixed with Dagger Shell as Default",
			},
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "Dagger Call:",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"interpreter":             "bash",
					"runme.dev/id":            "01JPR4JA36V7M9ZTVZBZB2DPF5",
					"runme.dev/name":          "dagger-core",
					"runme.dev/nameGenerated": "true",
				},
				Value: "dagger core \\\n  git --url github.com/runmedev/vscode-runme \\\n    tag --name main \\\n    tree \\\n        file --path dagger/scripts/presetup.sh",
			},
			{
				Kind:  parserv1.CellKind_CELL_KIND_MARKUP,
				Value: "Dagger Shell:",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"name":                    "Presetup",
					"runme.dev/id":            "01JPR4JA37X07H529VAC49HEZJ",
					"runme.dev/name":          "Presetup",
					"runme.dev/nameGenerated": "false",
				},
				Value: "git github.com/runmedev/vscode-runme |\n  tag main |\n  tree |\n    file dagger/scripts/presetup.sh",
			},
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"runme.dev/id":            "01JPR4JA37X07H529VADC0XFRF",
					"runme.dev/name":          "echo-presetup",
					"runme.dev/nameGenerated": "true",
				},
				Value: ".echo $(Presetup)",
			},
		},
		Metadata: map[string]string{
			"runme.dev/frontmatter": "---\nrunme:\n  id: 01JJDCG2SQSGV0DP55XCR55AYM\n  version: v3\nshell: dagger shell\nterminalRows: 20\n---",
		},
		Frontmatter: &parserv1.Frontmatter{
			TerminalRows: "20",
			Shell:        "dagger shell",
		},
	}

	resolver, err := NewResolver(WithNotebook(daggerShellNotebook))
	require.NoError(t, err)

	// Test that the resolver can resolve vanilla shell
	script, err := resolver.ResolveDaggerShell(ctx, uint32(2))
	require.NoError(t, err)
	assert.Equal(t, "dagger core \\\n  git --url github.com/runmedev/vscode-runme \\\n    tag --name main \\\n    tree \\\n        file --path dagger/scripts/presetup.sh", script)

	// Test that the resolver can resolve dagger shell, skipping vanilla shell cells
	script, err = resolver.ResolveDaggerShell(ctx, uint32(4))
	require.NoError(t, err)
	assert.Equal(t, "#!/usr/bin/env dagger shell\nPresetup()\n{\n  git github.com/runmedev/vscode-runme \\\n    | tag main \\\n    | tree \\\n    | file dagger/scripts/presetup.sh\n}\nDAGGER_01JPR4JA37X07H529VADC0XFRF()\n{\n  .echo $(Presetup)\n}\nPresetup\n", script)
}

func TestResolveDaggerShell_Source(t *testing.T) {
	simpleSource := "---\nshell: dagger shell\n---\n\n```sh {\"name\":\"SimpleDagger\",\"terminalRows\":\"18\"}\n### Exported in runme.dev as SimpleDagger\ngit github.com/runmedev/runme |\n    head |\n    tree |\n    file examples/README.md\n```\n"

	resolver, err := NewResolver(WithSource([]byte(simpleSource)))
	require.NoError(t, err)

	script, err := resolver.ResolveDaggerShell(context.Background(), uint32(0))
	require.NoError(t, err)

	assert.Equal(t, "#!/usr/bin/env dagger shell\nSimpleDagger()\n{\n  git github.com/runmedev/runme \\\n    | head \\\n    | tree \\\n    | file examples/README.md\n}\nSimpleDagger\n", script)
}

func TestResolveDaggerShell_EmptyRunmeMetadata(t *testing.T) {
	ctx := context.Background()

	// fake notebook with dagger shell cells
	daggerShellNotebook := &parserv1.Notebook{
		Cells: []*parserv1.Cell{
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata:   nil,
				Value:      "git github.com/runmedev/runme |\n    head |\n    tree |\n    file examples/README.md",
			},
		},
		Metadata: map[string]string{
			"runme.dev/frontmatter": "---\nshell: dagger shell\n---",
		},
	}

	resolver, err := NewResolver(WithNotebook(daggerShellNotebook))
	require.NoError(t, err)

	stub := `{
  git github.com/runmedev/runme \
    | head \
    | tree \
    | file examples/README.md
}
DAGGER_`

	script, err := resolver.ResolveDaggerShell(ctx, uint32(0))
	require.NoError(t, err)
	require.Contains(t, script, stub)
}

func TestResolveDaggerShell_InvalidShellFunctionName(t *testing.T) {
	ctx := context.Background()

	// fake notebook with invalid name for dagger shell
	invalidNameNotebook := &parserv1.Notebook{
		Cells: []*parserv1.Cell{
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"runme.dev/id":            "01JPR4JA36V7M9ZTVZBZB2DPF5",
					"name":                    "dagger-core",
					"runme.dev/nameGenerated": "false",
				},
				Value: "git github.com/runmedev/runme |\n    head |\n    tree |\n    file examples/README.md",
			},
		},
		Metadata: map[string]string{
			"runme.dev/frontmatter": "---\nshell: dagger shell\n---",
		},
	}

	resolver, err := NewResolver(WithNotebook(invalidNameNotebook))
	require.NoError(t, err)

	_, err = resolver.ResolveDaggerShell(ctx, uint32(0))
	require.Error(t, err)
	require.Contains(t, err.Error(), "dagger shell integration requires cell name to be a valid shell function name, got dagger-core")

	// fake notebook with valid name for dagger shell
	validNameNotebook := &parserv1.Notebook{
		Cells: []*parserv1.Cell{
			{
				Kind:       parserv1.CellKind_CELL_KIND_CODE,
				LanguageId: "sh",
				Metadata: map[string]string{
					"runme.dev/id":            "01JPR4JA36V7M9ZTVZBZB2DPF5",
					"name":                    "ValidName",
					"runme.dev/nameGenerated": "false",
				},
				Value: "git github.com/runmedev/runme |\n    head |\n    tree |\n    file examples/README.md",
			},
		},
		Metadata: map[string]string{
			"runme.dev/frontmatter": "---\nshell: dagger shell\n---",
		},
	}

	resolver, err = NewResolver(WithNotebook(validNameNotebook))
	require.NoError(t, err)

	script, err := resolver.ResolveDaggerShell(ctx, uint32(0))
	require.NoError(t, err)
	require.Contains(t, script, "ValidName()\n{\n  git github.com/runmedev/runme \\\n    | head \\\n    | tree \\\n    | file examples/README.md\n}\nValidName\n")
}

func TestIsValidShellFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid simple name",
			input:    "my_function",
			expected: true,
		},
		{
			name:     "valid name with numbers",
			input:    "function123",
			expected: true,
		},
		{
			name:     "valid name starting with underscore",
			input:    "_my_function",
			expected: true,
		},
		{
			name:     "valid name with mixed case",
			input:    "MyFunction",
			expected: true,
		},
		{
			name:     "invalid name starting with number",
			input:    "123function",
			expected: false,
		},
		{
			name:     "invalid name with hyphen",
			input:    "my-function",
			expected: false,
		},
		{
			name:     "invalid name with space",
			input:    "my function",
			expected: false,
		},
		{
			name:     "invalid name with special characters",
			input:    "my@function",
			expected: false,
		},
		{
			name:     "empty name",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidShellFunctionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
