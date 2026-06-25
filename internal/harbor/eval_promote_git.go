package harbor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
)

type promoteGit interface {
	StagedFiles() ([]string, error)
	UnstagedFilesTouching([]string) ([]string, error)
	LatestModTime([]string) (time.Time, error)
	AddJobDir(string, bool) error
	JobFiles(string, bool) ([]string, error)
	Commit(string) (string, error)
	Rel(string) (string, error)
}

type goGitPromoteClient struct {
	repo *git.Repository
	wt   *git.Worktree
	root string
}

func newGoGitPromoteClient() (*goGitPromoteClient, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return &goGitPromoteClient{
		repo: repo,
		wt:   wt,
		root: cleanExistingPath(wt.Filesystem.Root()),
	}, nil
}

func (c *goGitPromoteClient) StagedFiles() ([]string, error) {
	status, err := c.wt.Status()
	if err != nil {
		return nil, err
	}
	var staged []string
	for path, file := range status {
		if file.Staging != git.Unmodified && file.Staging != git.Untracked {
			staged = append(staged, filepath.ToSlash(path))
		}
	}
	sort.Strings(staged)
	return staged, nil
}

func (c *goGitPromoteClient) UnstagedFilesTouching(staged []string) ([]string, error) {
	status, err := c.wt.Status()
	if err != nil {
		return nil, err
	}
	stagedSet := map[string]struct{}{}
	for _, path := range staged {
		stagedSet[filepath.ToSlash(path)] = struct{}{}
	}
	var conflicts []string
	for path, file := range status {
		if file.Worktree == git.Unmodified {
			continue
		}
		path = filepath.ToSlash(path)
		if _, ok := stagedSet[path]; ok {
			conflicts = append(conflicts, path)
		}
	}
	sort.Strings(conflicts)
	return conflicts, nil
}

func (c *goGitPromoteClient) LatestModTime(paths []string) (time.Time, error) {
	var latest time.Time
	for _, path := range paths {
		info, err := os.Stat(filepath.Join(c.root, filepath.FromSlash(path)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return time.Time{}, err
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest, nil
}

func (c *goGitPromoteClient) AddJobDir(jobDir string, includeArtifacts bool) error {
	files, err := c.JobFiles(jobDir, includeArtifacts)
	if err != nil {
		return err
	}
	for _, rel := range files {
		if err := c.wt.AddWithOptions(&git.AddOptions{
			Path:       filepath.FromSlash(rel),
			SkipStatus: true,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *goGitPromoteClient) JobFiles(jobDir string, includeArtifacts bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(jobDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if shouldSkipPromoteDir(jobDir, path, d) {
			return filepath.SkipDir
		}
		if d.IsDir() || !shouldAddPromoteFile(jobDir, path, includeArtifacts) {
			return nil
		}
		rel, err := c.Rel(path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func shouldSkipPromoteDir(jobDir, path string, d os.DirEntry) bool {
	if !d.IsDir() || cleanExistingPath(path) == cleanExistingPath(jobDir) {
		return false
	}
	return d.Name() == "workdir"
}

func shouldAddPromoteFile(jobDir, path string, includeArtifacts bool) bool {
	if includeArtifacts {
		return true
	}
	rel, err := filepath.Rel(jobDir, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return false
	}
	if !strings.Contains(rel, string(os.PathSeparator)) {
		return true
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	for _, part := range parts[:len(parts)-1] {
		if part == "verifier" {
			return true
		}
	}
	return false
}

func (c *goGitPromoteClient) Commit(message string) (string, error) {
	hash, err := c.wt.Commit(message, &git.CommitOptions{})
	if err != nil {
		return "", err
	}
	return hash.String(), nil
}

func (c *goGitPromoteClient) Rel(path string) (string, error) {
	path = cleanExistingPath(path)
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s is outside git root %s", path, c.root)
	}
	return filepath.ToSlash(rel), nil
}

func joinDisplay(values []string) string {
	return strings.Join(values, ", ")
}
