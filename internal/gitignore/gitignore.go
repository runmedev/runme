package gitignore

import (
	"github.com/go-git/go-billy/v5"
	gogitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"go.uber.org/zap"
)

func NewMatcher(fs billy.Filesystem, respectGitignore bool, ignoreFilePatterns []string, logger *zap.Logger) gogitignore.Matcher {
	return gogitignore.NewMatcher(Patterns(fs, respectGitignore, ignoreFilePatterns, logger))
}

func Patterns(fs billy.Filesystem, respectGitignore bool, ignoreFilePatterns []string, logger *zap.Logger) []gogitignore.Pattern {
	if logger == nil {
		logger = zap.NewNop()
	}

	ignorePatterns := []gogitignore.Pattern{
		gogitignore.ParsePattern(".git", nil),
	}

	if respectGitignore {
		sysPatterns, err := gogitignore.LoadSystemPatterns(fs)
		if err != nil {
			logger.Info("failed to load system ignore patterns", zap.Error(err))
		}
		ignorePatterns = append(ignorePatterns, sysPatterns...)

		globPatterns, err := gogitignore.LoadGlobalPatterns(fs)
		if err != nil {
			logger.Info("failed to load global ignore patterns", zap.Error(err))
		}
		ignorePatterns = append(ignorePatterns, globPatterns...)

		patterns, err := gogitignore.ReadPatterns(fs, nil)
		if err != nil {
			logger.Info("failed to load local ignore patterns", zap.Error(err))
		}
		ignorePatterns = append(ignorePatterns, patterns...)
	}

	for _, pattern := range ignoreFilePatterns {
		ignorePatterns = append(ignorePatterns, gogitignore.ParsePattern(pattern, nil))
	}

	return ignorePatterns
}
