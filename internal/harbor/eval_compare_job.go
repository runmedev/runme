package harbor

import (
	"fmt"
	"os"
	"path/filepath"
)

type compareJob struct {
	RelPath   string
	ResultRel string
	Result    promoteJobResult
	Config    promoteJobConfig
	Selection string
}

type compareBaseOptions struct {
	git      compareGit
	baseRef  string
	jobsRoot string
}

type compareCandidateOptions struct {
	jobsDir       string
	job           string
	includeOracle bool
	allowErrors   bool
	git           compareGit
}

func resolveCompareBase(opts compareBaseOptions) (compareJob, error) {
	jobs, err := opts.git.TrackedEvalJobs(opts.baseRef, opts.jobsRoot)
	if err != nil {
		return compareJob{}, err
	}
	for _, job := range jobs {
		if incompletePromoteJobReason(job.Result) == "" {
			return job, nil
		}
	}
	return compareJob{}, fmt.Errorf("no complete Git-tracked eval job found under %s at %s", opts.jobsRoot, opts.baseRef)
}

func resolveCompareCandidate(opts compareCandidateOptions) (compareJob, error) {
	var jobDir, selection string
	var err error
	policy := promoteJobPolicy{
		includeOracle: opts.includeOracle,
		allowErrors:   opts.allowErrors,
	}
	if opts.job == "" {
		jobDir, selection, err = latestPromoteJob(opts.jobsDir, policy)
		if err != nil {
			return compareJob{}, err
		}
	} else {
		invocationCwd, err := os.Getwd()
		if err != nil {
			return compareJob{}, err
		}
		jobDir = opts.job
		if !filepath.IsAbs(jobDir) {
			jobDir = filepath.Join(invocationCwd, jobDir)
		}
		jobDir = cleanExistingPath(jobDir)
		if err := validatePromoteJobDir(opts.jobsDir, jobDir); err != nil {
			return compareJob{}, err
		}
		selection = "explicit --job"
	}

	result, err := readPromoteJobResult(jobDir)
	if err != nil {
		return compareJob{}, err
	}
	config, err := readPromoteJobConfig(jobDir)
	if err != nil {
		return compareJob{}, err
	}
	if err := validatePromoteJobPolicy(result, config, policy); err != nil {
		return compareJob{}, err
	}
	jobRel, err := opts.git.Rel(jobDir)
	if err != nil {
		return compareJob{}, err
	}
	resultRel, err := opts.git.Rel(result.Path)
	if err != nil {
		return compareJob{}, err
	}
	return compareJob{
		RelPath:   jobRel,
		ResultRel: resultRel,
		Result:    result,
		Config:    config,
		Selection: selection,
	}, nil
}
