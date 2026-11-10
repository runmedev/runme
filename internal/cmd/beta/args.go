package beta

import (
	"github.com/gobwas/glob"
	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/project"
)

func createProjectFilterFromPatterns(tasks []project.Task, patterns []string) (project.Filter, error) {
	if len(patterns) == 0 {
		return func(t project.Task) (bool, error) { return true, nil }, nil
	}

	normalized := make([]string, len(patterns))
	for index, pattern := range patterns {
		normalized[index], _ = project.NormalizeDecoratedTaskName(tasks, pattern)
	}

	globs, err := parseGlobs(normalized)
	if err != nil {
		return nil, err
	}

	return func(t project.Task) (bool, error) {
		for _, g := range globs {
			if g.Match(t.CodeBlock.Name()) {
				return true, nil
			}
		}
		return false, nil
	}, nil
}

func parseGlobs(patterns []string) ([]*glob.Pattern, error) {
	globs := make([]*glob.Pattern, 0, len(patterns))
	for _, item := range patterns {
		g, err := glob.Compile(item)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		globs = append(globs, g)
	}
	return globs, nil
}
