package harbor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type compareGit interface {
	Rel(string) (string, error)
	TrackedEvalJobs(string, string) ([]compareJob, error)
}

type goGitCompareClient struct {
	repo *git.Repository
	root string
}

func newGoGitCompareClient() (*goGitCompareClient, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return &goGitCompareClient{
		repo: repo,
		root: cleanExistingPath(wt.Filesystem.Root()),
	}, nil
}

func (c *goGitCompareClient) Rel(path string) (string, error) {
	path = cleanExistingPath(path)
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s is outside git root %s", path, c.root)
	}
	return filepath.ToSlash(rel), nil
}

func (c *goGitCompareClient) TrackedEvalJobs(baseRef, jobsRoot string) ([]compareJob, error) {
	jobsRootRel, err := c.Rel(jobsRoot)
	if err != nil {
		return nil, err
	}
	tree, err := c.tree(baseRef)
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimSuffix(jobsRootRel, "/") + "/"
	files := tree.Files()
	defer files.Close()

	var jobs []compareJob
	err = files.ForEach(func(file *object.File) error {
		if !strings.HasPrefix(file.Name, prefix) {
			return nil
		}
		rest := strings.TrimPrefix(file.Name, prefix)
		parts := strings.Split(rest, "/")
		if len(parts) != 2 || parts[1] != "result.json" {
			return nil
		}

		resultData, err := file.Contents()
		if err != nil {
			return err
		}
		result, err := parsePromoteJobResult(file.Name, []byte(resultData))
		if err != nil {
			return nil
		}

		jobRel := prefix + parts[0]
		config := promoteJobConfig{}
		if configFile, err := tree.File(jobRel + "/config.json"); err == nil {
			configData, err := configFile.Contents()
			if err != nil {
				return err
			}
			parsed, err := parsePromoteJobConfig([]byte(configData))
			if err == nil {
				config = parsed
			}
		}

		jobs = append(jobs, compareJob{
			RelPath:   jobRel,
			ResultRel: file.Name,
			Result:    result,
			Config:    config,
			Selection: "latest tracked job under --base",
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortCompareJobs(jobs)
	return jobs, nil
}

func (c *goGitCompareClient) tree(ref string) (*object.Tree, error) {
	hash, err := c.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, err
	}
	commit, err := c.repo.CommitObject(*hash)
	if err != nil {
		return nil, err
	}
	return commit.Tree()
}

func sortCompareJobs(jobs []compareJob) {
	sort.Slice(jobs, func(i, j int) bool {
		left, right := jobs[i], jobs[j]
		leftTime, rightTime := resultTimestamp(left.Result), resultTimestamp(right.Result)
		if !leftTime.IsZero() || !rightTime.IsZero() {
			if !leftTime.Equal(rightTime) {
				return leftTime.After(rightTime)
			}
		}
		return left.RelPath > right.RelPath
	})
}
