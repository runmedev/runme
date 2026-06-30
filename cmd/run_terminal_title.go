package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/runmedev/runme/v3/internal/terminal"
	"github.com/runmedev/runme/v3/project"
)

func setTerminalTitle(w io.Writer, title string) {
	terminal.SetTitle(w, title)
}

func runTerminalTitle(tasks []project.Task, runAll, runWithIndex bool, runIndex int, tags []string) string {
	title := runTerminalTitleOptions{
		tasks:   tasks,
		all:     runAll,
		indexed: runWithIndex,
		index:   runIndex,
		tags:    tags,
	}.Title()

	return string(title)
}

type runTerminalTitleOptions struct {
	tasks   []project.Task
	all     bool
	indexed bool
	index   int
	tags    []string
}

func (o runTerminalTitleOptions) Title() terminal.Title {
	if len(o.tags) > 0 {
		return terminal.Title("runme run " + runTerminalTitleTags(o.tags).String())
	}

	if o.all {
		return "runme run all tasks"
	}

	return terminal.Title("runme run " + o.taskLabels().String())
}

func (o runTerminalTitleOptions) taskLabels() terminalTitleItems {
	names := make(terminalTitleItems, 0, len(o.tasks))
	for i, task := range o.tasks {
		label := runTerminalTitleTask{
			task:    task,
			indexed: o.indexed && i == 0,
			index:   o.index,
		}
		names = append(names, label.String())
	}
	return names
}

type runTerminalTitleTags []string

func (t runTerminalTitleTags) String() string {
	label := "tag"
	if len(t) > 1 {
		label = "tags"
	}
	return label + " " + terminalTitleItems(t).String()
}

type runTerminalTitleTask struct {
	task    project.Task
	indexed bool
	index   int
}

func (t runTerminalTitleTask) String() string {
	if t.task.CodeBlock == nil {
		return "unnamed"
	}

	if t.indexed && t.task.CodeBlock.IsUnnamed() {
		return fmt.Sprintf("#%d", t.index)
	}

	if !t.task.CodeBlock.IsUnnamed() && strings.TrimSpace(t.task.CodeBlock.Name()) != "" {
		return t.task.CodeBlock.Name()
	}

	if firstLine := strings.TrimSpace(t.task.CodeBlock.FirstLine()); firstLine != "" {
		return fmt.Sprintf("%q", firstLine)
	}

	return "unnamed"
}

type terminalTitleItems []string

func (items terminalTitleItems) String() string {
	items = items.Compact()
	if len(items) == 0 {
		return "unnamed"
	}

	if len(items) <= 2 {
		return strings.Join(items, ",")
	}

	return fmt.Sprintf("%s +%d", strings.Join(items[:2], ","), len(items)-2)
}

func (items terminalTitleItems) Compact() terminalTitleItems {
	compacted := make(terminalTitleItems, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		compacted = append(compacted, item)
	}
	return compacted
}
