package harbor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	git           compareGit
	baseRef       string
	jobsRoot      string
	datasetFilter evalDatasetFilter
}

type compareCandidateOptions struct {
	jobsDir       string
	job           string
	includeOracle bool
	allowErrors   bool
	git           compareGit
	datasetFilter evalDatasetFilter
}

type evalJobStore struct {
	jobsDir       string
	git           compareGit
	policy        promoteJobPolicy
	datasetFilter evalDatasetFilter
}

func (s evalJobStore) LatestTracked(baseRef string) (evalJobRef, error) {
	jobs, err := s.git.TrackedEvalJobs(baseRef, s.jobsDir)
	if err != nil {
		return evalJobRef{}, err
	}
	for _, job := range jobs {
		if incompletePromoteJobReason(job.Result) == "" && s.datasetFilter.matches(job.Config) {
			return job, nil
		}
	}
	return evalJobRef{}, fmt.Errorf("no complete Git-tracked eval job found for dataset %s under %s at %s", s.datasetFilter.displayPath(), s.jobsDir, baseRef)
}

func (s evalJobStore) LatestLocal() (evalJobRef, string, error) {
	jobDir, selection, err := latestPromoteJob(s.jobsDir, s.policy, s.datasetFilter)
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
	if !s.datasetFilter.matches(config) {
		return evalJobRef{}, fmt.Errorf("eval job dataset does not match requested dataset %s", s.datasetFilter.displayPath())
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
		jobsDir:       opts.jobsRoot,
		git:           opts.git,
		datasetFilter: opts.datasetFilter,
	}.LatestTracked(opts.baseRef)
}

func resolveCompareCandidate(opts compareCandidateOptions) (compareJob, error) {
	policy := promoteJobPolicy{
		includeOracle: opts.includeOracle,
		allowErrors:   opts.allowErrors,
	}
	store := evalJobStore{
		jobsDir:       opts.jobsDir,
		git:           opts.git,
		policy:        policy,
		datasetFilter: opts.datasetFilter,
	}
	if opts.job == "" {
		job, _, err := store.LatestLocal()
		return job, err
	}
	job, _, err := store.ExplicitLocal(opts.job)
	return job, err
}

type evalDatasetFilter struct {
	paths []string
}

func newEvalDatasetFilter(path string, git interface{ Rel(string) (string, error) }) (evalDatasetFilter, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultEvalDatasetPath
	}

	var paths []string
	addPath := func(candidate string) {
		candidate = normalizeEvalDatasetConfigPath(candidate)
		if candidate == "" {
			return
		}
		for _, existing := range paths {
			if existing == candidate {
				return
			}
		}
		paths = append(paths, candidate)
	}

	if filepath.IsAbs(path) {
		rel, err := git.Rel(path)
		if err != nil {
			return evalDatasetFilter{}, err
		}
		addPath(rel)
	} else {
		addPath(path)
		if normalizeEvalDatasetConfigPath(path) != normalizeEvalDatasetConfigPath(DefaultEvalDatasetPath) {
			cwd, err := os.Getwd()
			if err != nil {
				return evalDatasetFilter{}, err
			}
			if rel, err := git.Rel(filepath.Join(cwd, path)); err == nil {
				addPath(rel)
			}
		}
	}
	return evalDatasetFilter{paths: paths}, nil
}

func (f evalDatasetFilter) matches(config promoteJobConfig) bool {
	if len(f.paths) == 0 {
		return true
	}
	for _, dataset := range config.Datasets {
		path := normalizeEvalDatasetConfigPath(dataset.Path)
		for _, want := range f.paths {
			if path == want {
				return true
			}
		}
	}
	return false
}

func (f evalDatasetFilter) displayPath() string {
	if len(f.paths) == 0 {
		return DefaultEvalDatasetPath
	}
	return f.paths[0]
}

func normalizeEvalDatasetConfigPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
}
