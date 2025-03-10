package teststub

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/require"
)

func Setup(t *testing.T, temp string) Path {
	t.Helper()

	testDataSrc := originalTestDataPath()
	require.NoError(t, copy.Copy(testDataSrc, temp))
	Cleanup(t, temp)

	err := os.Rename(
		filepath.Join(temp, "git-project", ".git.bkp"),
		filepath.Join(temp, "git-project", ".git"),
	)
	require.NoError(t, err)

	err = os.Rename(
		filepath.Join(temp, "git-project", ".gitignore.bkp"),
		filepath.Join(temp, "git-project", ".gitignore"),
	)
	require.NoError(t, err)

	err = os.Rename(
		filepath.Join(temp, "git-project", "nested", ".gitignore.bkp"),
		filepath.Join(temp, "git-project", "nested", ".gitignore"),
	)
	require.NoError(t, err)

	// don't break tests on Windows
	if runtime.GOOS != "windows" {
		cmd := exec.Command("direnv", "allow")
		cmd.Dir = filepath.Join(temp, "direnv-project")
		cmd.Env = os.Environ()
		require.NoError(t, cmd.Run())
	}

	return Path{root: temp}
}

func Cleanup(t *testing.T, temp string) {
	t.Helper()

	cmd := exec.Command("direnv", "prune")
	cmd.Dir = filepath.Join(temp, "direnv-project")
	cmd.Env = os.Environ()
	_ = cmd.Run() // not critical

	err := os.RemoveAll(filepath.Join(temp, "git-project", ".git"))
	require.NoError(t, err)
}

func OriginalPath() Path {
	return Path{root: originalTestDataPath()}
}

type Path struct {
	root string
}

func (p Path) Root() string {
	return p.root
}

func (p Path) Join(elems ...string) string {
	elems = append([]string{p.root}, elems...)
	return filepath.Join(elems...)
}

func (p Path) DirProjectPath() string {
	return p.Join("dir-project")
}

func (p Path) DirEnvProjectPath() string {
	return p.Join("direnv-project")
}

func (p Path) GitProjectPath() string {
	return p.Join("git-project")
}

func (p Path) GitProjectNestedPath() string {
	return p.Join("git-project", "nested")
}

func (p Path) ProjectFilePath() string {
	return p.Join("file-project.md")
}

func originalTestDataPath() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "..", "testdata")
}
