package beta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/document"
	"github.com/runmedev/runme/v3/document/identity"
	"github.com/runmedev/runme/v3/project"
)

func TestCreateProjectFilterFromPatterns(t *testing.T) {
	tasks := betaSelectorTasks(t, "```sh {\"name\":\"build\"}\necho build\n```\n\n```sh {\"name\":\"builder\"}\necho builder\n```\n\n```sh {\"name\":\"generate-code\"}\necho generate\n```\n")

	t.Run("decorated exact name", func(t *testing.T) {
		filter, err := createProjectFilterFromPatterns(tasks, []string{"build*"})
		require.NoError(t, err)
		matched, err := project.FilterTasksByFn(tasks, filter)
		require.NoError(t, err)
		require.Len(t, matched, 1)
		assert.Equal(t, "build", matched[0].CodeBlock.Name())
	})

	t.Run("glob fallback", func(t *testing.T) {
		filter, err := createProjectFilterFromPatterns(tasks, []string{"generate-*"})
		require.NoError(t, err)
		matched, err := project.FilterTasksByFn(tasks, filter)
		require.NoError(t, err)
		require.Len(t, matched, 1)
		assert.Equal(t, "generate-code", matched[0].CodeBlock.Name())
	})
}

func betaSelectorTasks(t *testing.T, source string) []project.Task {
	t.Helper()
	doc := document.New([]byte(source), identity.NewResolver(identity.DefaultLifecycleIdentity))
	root, err := doc.Root()
	require.NoError(t, err)
	blocks := document.CollectCodeBlocks(root)
	tasks := make([]project.Task, len(blocks))
	for index, block := range blocks {
		tasks[index] = project.Task{CodeBlock: block}
	}
	return tasks
}
