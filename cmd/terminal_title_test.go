package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/document"
	"github.com/runmedev/runme/v3/document/identity"
	"github.com/runmedev/runme/v3/internal/terminal"
	"github.com/runmedev/runme/v3/project"
)

func TestSetTerminalTitleDoesNotWriteToBuffer(t *testing.T) {
	var buf bytes.Buffer

	setTerminalTitle(&buf, "runme run build")

	require.Empty(t, buf.String())
}

func TestSanitizeTerminalTitle(t *testing.T) {
	title := terminal.SanitizeTitle(" \x1b]0;bad\x07\nrunme\rrun build\t ")

	require.Equal(t, "]0;badrunmerun build", title)
}

func TestSanitizeTerminalTitleTruncates(t *testing.T) {
	title := terminal.SanitizeTitle(strings.Repeat("a", 101))

	require.Len(t, title, 100)
	require.True(t, strings.HasSuffix(title, "..."))
}

func TestRunTerminalTitle(t *testing.T) {
	tasks := terminalTitleTasks(t, `# Test

`+"```sh { name=build }\n"+`npm run build
`+"```\n\n"+`Another task

`+"```sh { name=test }\n"+`npm test
`+"```\n\n"+`Unnamed task

`+"```sh\n"+`docker compose up
`+"```\n")

	require.Equal(t, "runme run build", runTerminalTitle(tasks[:1], false, false, -1, nil))
	require.Equal(t, "runme run build,test", runTerminalTitle(tasks[:2], false, false, -1, nil))
	require.Equal(t, `runme run build,test +1`, runTerminalTitle(tasks, false, false, -1, nil))
	require.Equal(t, `runme run "docker compose up"`, runTerminalTitle(tasks[2:], false, false, -1, nil))
	require.Equal(t, "runme run #2", runTerminalTitle(tasks[2:], false, true, 2, nil))
	require.Equal(t, "runme run all tasks", runTerminalTitle(tasks, true, false, -1, nil))
	require.Equal(t, "runme run tag ci", runTerminalTitle(tasks, false, false, -1, []string{"ci"}))
	require.Equal(t, "runme run tags ci,release", runTerminalTitle(tasks, false, false, -1, []string{"ci", "release"}))
	require.Equal(t, "runme run tags ci,release +1", runTerminalTitle(tasks, false, false, -1, []string{"ci", "release", "nightly"}))
}

func terminalTitleTasks(t *testing.T, markdown string) []project.Task {
	t.Helper()

	resolver := identity.NewResolver(identity.DefaultLifecycleIdentity)
	doc := document.New([]byte(markdown), resolver)
	root, err := doc.Root()
	require.NoError(t, err)

	blocks := document.CollectCodeBlocks(root)
	tasks := make([]project.Task, len(blocks))
	for i, block := range blocks {
		tasks[i] = project.Task{CodeBlock: block}
	}
	return tasks
}
