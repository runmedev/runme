package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/osfs"
	gogitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/pelletier/go-toml/v2"

	internalgitignore "github.com/runmedev/runme/v3/internal/gitignore"
)

type harborDockerTaskConfig struct {
	Environment struct {
		Workdir string `toml:"workdir"`
	} `toml:"environment"`
}

func stageHarborDockerWorkdirs(datasetPath string, stderr io.Writer) error {
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return err
	}
	workspaceRoot, err = filepath.Abs(workspaceRoot)
	if err != nil {
		return err
	}
	workspaceRoot, err = filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		return err
	}

	ignoreMatcher := internalgitignore.NewMatcher(osfs.New(workspaceRoot), true, nil, nil)

	return filepath.WalkDir(datasetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "task.toml" {
			return nil
		}
		return stageHarborDockerTaskWorkdir(workspaceRoot, ignoreMatcher, path, stderr)
	})
}

func stageHarborDockerTaskWorkdir(workspaceRoot string, ignoreMatcher gogitignore.Matcher, taskConfigPath string, stderr io.Writer) error {
	config, err := readHarborDockerTaskConfig(taskConfigPath)
	if err != nil {
		return err
	}

	source, ok, err := harborDockerWorkdirSource(workspaceRoot, config.Environment.Workdir)
	if err != nil || !ok {
		return err
	}

	info, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	target := filepath.Join(filepath.Dir(taskConfigPath), "environment", "workdir")
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := copyHarborDockerWorkdir(workspaceRoot, ignoreMatcher, source, source, target); err != nil {
		return err
	}
	if !isGitIgnored(workspaceRoot, ignoreMatcher, target, true) {
		_, _ = fmt.Fprintf(
			stderr,
			"warning: staged Harbor Docker workdir %s is not ignored by git; add **/environment/workdir/ to .gitignore\n",
			target,
		)
	}
	return nil
}

func readHarborDockerTaskConfig(path string) (*harborDockerTaskConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config harborDockerTaskConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("read Harbor task config %s: %w", path, err)
	}
	return &config, nil
}

func harborDockerWorkdirSource(workspaceRoot string, remoteWorkdir string) (string, bool, error) {
	if remoteWorkdir == "" || remoteWorkdir == "/app" {
		return "", false, nil
	}
	const appPrefix = "/app/"
	if !strings.HasPrefix(remoteWorkdir, appPrefix) {
		return "", false, nil
	}

	rel := strings.TrimPrefix(remoteWorkdir, appPrefix)
	source := filepath.Join(workspaceRoot, filepath.FromSlash(rel))
	source, err := filepath.Abs(source)
	if err != nil {
		return "", false, err
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if !isPathWithin(source, workspaceRoot) {
		return "", false, fmt.Errorf("harbor Docker workdir %s maps outside workspace root %s", remoteWorkdir, workspaceRoot)
	}
	return source, true, nil
}

func copyHarborDockerWorkdir(workspaceRoot string, ignoreMatcher gogitignore.Matcher, source string, logicalSource string, target string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o750); err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		logicalPath := filepath.Join(logicalSource, entry.Name())
		targetPath := filepath.Join(target, entry.Name())

		info, err := os.Lstat(sourcePath)
		if err != nil {
			return err
		}
		isDir := info.IsDir()
		resolvedSource := sourcePath
		if info.Mode()&os.ModeSymlink != 0 {
			resolvedSource, err = filepath.EvalSymlinks(sourcePath)
			if err != nil {
				return err
			}
			info, err = os.Stat(resolvedSource)
			if err != nil {
				return err
			}
			isDir = info.IsDir()
		}

		if isGitIgnored(workspaceRoot, ignoreMatcher, logicalPath, isDir) {
			continue
		}

		if isDir {
			if err := copyHarborDockerWorkdir(workspaceRoot, ignoreMatcher, resolvedSource, logicalPath, targetPath); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := copyHarborDockerFile(resolvedSource, targetPath, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func copyHarborDockerFile(source string, target string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func isGitIgnored(workspaceRoot string, ignoreMatcher gogitignore.Matcher, path string, isDir bool) bool {
	rel, err := filepath.Rel(workspaceRoot, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return false
	}
	return ignoreMatcher.Match(strings.Split(rel, string(filepath.Separator)), isDir)
}

func isPathWithin(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}
