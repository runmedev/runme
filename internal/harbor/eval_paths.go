package harbor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/runmedev/runme/v3/project"
)

type evalPaths struct {
	datasetPath         string
	delegateDatasetPath string
	delegateJobsDir     string
	passthrough         []string
	defaultDataset      bool
	invocationCwd       string
	executionCwd        string
}

type evalPathResolver struct {
	opts          *EvalOptions
	invocationCwd string
	baseDir       string
}

func resolveEvalPaths(opts *EvalOptions, args []string) (evalPaths, error) {
	invocationCwd, err := os.Getwd()
	if err != nil {
		return evalPaths{}, err
	}

	resolver := evalPathResolver{
		opts:          opts,
		invocationCwd: cleanExistingPath(invocationCwd),
	}
	resolver.baseDir = defaultEvalBaseDir(resolver.invocationCwd)
	return resolver.resolve(args), nil
}

func (r evalPathResolver) resolve(args []string) evalPaths {
	datasetArg, defaultDataset, passthrough := splitEvalDatasetArg(args)

	var datasetPath string
	if defaultDataset {
		datasetPath = filepath.Join(r.baseDir, DefaultEvalDatasetPath)
	} else {
		datasetPath = r.inputPath(datasetArg)
	}

	executionCwd := r.executionCwd(datasetPath)
	delegateDatasetPath := r.delegatePath(datasetPath, executionCwd)

	if !r.opts.JobsDirExplicit {
		r.opts.JobsDir = filepath.Join(r.baseDir, DefaultEvalJobsDir)
	} else {
		r.opts.JobsDir = r.inputPath(r.opts.JobsDir)
	}
	delegateJobsDir := r.delegateJobsPath(r.opts.JobsDir, executionCwd)

	return evalPaths{
		datasetPath:         datasetPath,
		delegateDatasetPath: delegateDatasetPath,
		delegateJobsDir:     delegateJobsDir,
		passthrough:         passthrough,
		defaultDataset:      defaultDataset,
		invocationCwd:       r.invocationCwd,
		executionCwd:        executionCwd,
	}
}

func (r evalPathResolver) inputPath(path string) string {
	if filepath.IsAbs(path) {
		return cleanExistingPath(path)
	}

	return filepath.Clean(filepath.Join(r.invocationCwd, path))
}

func (r evalPathResolver) executionCwd(datasetPath string) string {
	if relativePathUnder(r.baseDir, datasetPath) != "" {
		return r.baseDir
	}
	return r.invocationCwd
}

func (r evalPathResolver) delegatePath(path, executionCwd string) string {
	delegated := relativePathUnder(executionCwd, path)
	if delegated == "" {
		return filepath.Clean(path)
	}
	return delegated
}

func (r evalPathResolver) delegateJobsPath(path, executionCwd string) string {
	if r.invocationCwd == executionCwd {
		return r.delegatePath(path, executionCwd)
	}
	return filepath.Clean(path)
}

func cleanExistingPath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved)
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		return cleanPathFromExistingParent(abs)
	}
	return filepath.Clean(path)
}

func cleanPathFromExistingParent(path string) string {
	cleaned := filepath.Clean(path)
	current := cleaned
	missing := []string{}

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return cleaned
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func relativePathUnder(base, path string) string {
	relative, err := filepath.Rel(base, path)
	if err != nil {
		return ""
	}
	if relative == "." {
		return "."
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return ""
	}
	return filepath.Clean(relative)
}

func splitEvalDatasetArg(args []string) (string, bool, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return DefaultEvalDatasetPath, true, args
	}
	return args[0], false, args[1:]
}

func defaultEvalBaseDir(cwd string) string {
	proj, err := project.NewDirProject(
		cwd,
		project.WithFindRepoUpward(),
		project.WithAllowUnsupportedGitExtensions(true),
		project.WithRespectGitignore(false),
	)
	if err != nil {
		return cwd
	}
	return proj.Root()
}
