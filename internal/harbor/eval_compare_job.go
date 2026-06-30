package harbor

import (
	"fmt"
	"os"
	"path/filepath"
)

type evalJobRef struct {
	RelPath   string
	ResultRel string
	Result    promoteJobResult
	Config    promoteJobConfig
	Selection string
}

type compareJob = evalJobRef

func (j evalJobRef) MessageData(subject string, evidenceOnly bool) promoteMessageData {
	return promoteMessageData{
		subject:      subject,
		jobPath:      j.RelPath,
		resultPath:   j.ResultRel,
		evidenceOnly: evidenceOnly,
		config:       j.Config,
		result:       j.Result,
	}
}

func (j evalJobRef) ResultSummaries() []evalResultSummary {
	return summarizeEvalResults(j.Result, j.Config)
}

func (j evalJobRef) ResultSummaryMap() map[string]evalResultSummary {
	summaries := j.ResultSummaries()
	entries := make(map[string]evalResultSummary, len(summaries))
	for _, summary := range summaries {
		entries[summary.Key] = summary
	}
	return entries
}

func (j evalJobRef) ComparisonJob(baseRef string) evalComparisonJob {
	return evalComparisonJob{
		Path:      j.RelPath,
		Result:    j.ResultRel,
		Ref:       baseRef,
		Selection: j.Selection,
		Timestamp: formatCompareTimestamp(resultTimestamp(j.Result)),
	}
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

type evalJobStore struct {
	jobsDir string
	git     compareGit
	policy  promoteJobPolicy
}

func (s evalJobStore) LatestTracked(baseRef string) (evalJobRef, error) {
	jobs, err := s.git.TrackedEvalJobs(baseRef, s.jobsDir)
	if err != nil {
		return evalJobRef{}, err
	}
	for _, job := range jobs {
		if incompletePromoteJobReason(job.Result) == "" {
			return job, nil
		}
	}
	return evalJobRef{}, fmt.Errorf("no complete Git-tracked eval job found under %s at %s", s.jobsDir, baseRef)
}

func (s evalJobStore) LatestLocal() (evalJobRef, string, error) {
	jobDir, selection, err := latestPromoteJob(s.jobsDir, s.policy)
	if err != nil {
		return evalJobRef{}, "", err
	}
	job, err := s.localJob(jobDir, selection)
	return job, jobDir, err
}

func (s evalJobStore) ExplicitLocal(jobDir string) (evalJobRef, string, error) {
	invocationCwd, err := os.Getwd()
	if err != nil {
		return evalJobRef{}, "", err
	}
	if !filepath.IsAbs(jobDir) {
		jobDir = filepath.Join(invocationCwd, jobDir)
	}
	jobDir = cleanExistingPath(jobDir)
	if err := validatePromoteJobDir(s.jobsDir, jobDir); err != nil {
		return evalJobRef{}, "", err
	}
	job, err := s.localJob(jobDir, "explicit --job")
	return job, jobDir, err
}

func (s evalJobStore) localJob(jobDir, selection string) (evalJobRef, error) {
	result, err := readPromoteJobResult(jobDir)
	if err != nil {
		return evalJobRef{}, err
	}
	config, err := readPromoteJobConfig(jobDir)
	if err != nil {
		return evalJobRef{}, err
	}
	if err := validatePromoteJobPolicy(result, config, s.policy); err != nil {
		return evalJobRef{}, err
	}
	jobRel, err := s.git.Rel(jobDir)
	if err != nil {
		return evalJobRef{}, err
	}
	resultRel, err := s.git.Rel(result.Path)
	if err != nil {
		return evalJobRef{}, err
	}
	return evalJobRef{
		RelPath:   jobRel,
		ResultRel: resultRel,
		Result:    result,
		Config:    config,
		Selection: selection,
	}, nil
}

func resolveCompareBase(opts compareBaseOptions) (compareJob, error) {
	return evalJobStore{
		jobsDir: opts.jobsRoot,
		git:     opts.git,
	}.LatestTracked(opts.baseRef)
}

func resolveCompareCandidate(opts compareCandidateOptions) (compareJob, error) {
	policy := promoteJobPolicy{
		includeOracle: opts.includeOracle,
		allowErrors:   opts.allowErrors,
	}
	store := evalJobStore{
		jobsDir: opts.jobsDir,
		git:     opts.git,
		policy:  policy,
	}
	if opts.job == "" {
		job, _, err := store.LatestLocal()
		return job, err
	}
	job, _, err := store.ExplicitLocal(opts.job)
	return job, err
}
