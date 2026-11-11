package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/document"
	"github.com/runmedev/runme/v3/document/identity"
)

func TestNormalizeDecoratedTaskName(t *testing.T) {
	doc := document.New([]byte("```sh {\"name\":\"build\"}\necho build\n```\n\n```sh {\"name\":\"skipped\",\"excludeFromRunAll\":\"true\"}\necho skipped\n```\n"), identity.NewResolver(identity.DefaultLifecycleIdentity))
	root, err := doc.Root()
	require.NoError(t, err)
	blocks := document.CollectCodeBlocks(root)
	tasks := make([]Task, len(blocks))
	for index, block := range blocks {
		tasks[index] = Task{CodeBlock: block}
	}

	for _, tt := range []struct {
		selector string
		want     string
		ok       bool
	}{
		{selector: "build*", want: "build", ok: true},
		{selector: "build", want: "build", ok: false},
		{selector: "skipped*", want: "skipped*", ok: false},
		{selector: "generate-*", want: "generate-*", ok: false},
	} {
		t.Run(tt.selector, func(t *testing.T) {
			got, ok := NormalizeDecoratedTaskName(tasks, tt.selector)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.ok, ok)
		})
	}
}
