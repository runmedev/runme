package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/runmedev/runme/v3/project"
)

func TestSessionList(t *testing.T) {
	seedEnv := os.Environ()

	t.Run("ConcurrentAdd", func(t *testing.T) {
		l := NewSessionList()

		var g errgroup.Group

		for i := 0; i < 10; i++ {
			g.Go(func() error {
				s, err := New(WithSeedEnv(seedEnv))
				require.NoError(t, err)
				err = l.Add(s)
				require.NoError(t, err)
				return nil
			})
		}

		require.NoError(t, g.Wait())
		require.Equal(t, l.Size(), 10)
	})

	t.Run("AddAndRetrieveNewest", func(t *testing.T) {
		l := NewSessionList()

		s, err := New(WithSeedEnv(seedEnv))
		require.NoError(t, err)
		err = l.Add(s)
		require.NoError(t, err)

		newest, ok := l.Newest()
		require.True(t, ok)
		require.Equal(t, s.ID, newest.ID)
	})

	t.Run("EvictOnOverflow", func(t *testing.T) {
		l := NewSessionList()

		for i := 0; i < SessionListCapacity+10; i++ {
			s, err := New(WithSeedEnv(seedEnv))
			require.NoError(t, err)
			err = l.Add(s)
			require.NoError(t, err)
		}

		require.Equal(t, l.Size(), SessionListCapacity)
	})
}

func TestNewPlainSessionSeedsRequestEnvLast(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("TOKEN=dotenv\nDOTENV_ONLY=dotenv\n"), 0o600))

	proj, err := project.NewDirProject(
		dir,
		project.WithEnvFilesReadOrder([]string{".env"}),
		project.WithAllowUnsupportedGitExtensions(true),
	)
	require.NoError(t, err)

	sess, err := New(
		WithProject(proj),
		WithSeedEnv([]string{"TOKEN=system"}),
		WithRequestEnv([]string{"TOKEN=request"}),
	)
	require.NoError(t, err)

	value, ok := sess.GetEnv("TOKEN")
	require.True(t, ok)
	require.Equal(t, "request", value)

	value, ok = sess.GetEnv("DOTENV_ONLY")
	require.True(t, ok)
	require.Equal(t, "dotenv", value)
}

func TestNewOwlSessionUsesSeedStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("TOKEN=dotenv\nDOTENV_ONLY=dotenv\n"), 0o600))

	proj, err := project.NewDirProject(
		dir,
		project.WithEnvFilesReadOrder([]string{".env"}),
		project.WithAllowUnsupportedGitExtensions(true),
	)
	require.NoError(t, err)

	sess, err := New(
		WithOwl(true),
		WithProject(proj),
		WithSeedEnv([]string{"TOKEN=system"}),
		WithRequestEnv([]string{"TOKEN=request"}),
	)
	require.NoError(t, err)

	value, ok := sess.GetEnv("TOKEN")
	require.True(t, ok)
	require.Equal(t, "request", value)

	value, ok = sess.GetEnv("DOTENV_ONLY")
	require.True(t, ok)
	require.Equal(t, "dotenv", value)
}

func TestNewProjectlessOwlSessionUsesObservedEnvOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("CWD_DOTENV=dotenv\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".envrc"), []byte("export CWD_DIRENV=direnv\n"), 0o600))
	t.Chdir(dir)

	sess, err := New(
		WithOwl(true),
		WithSeedEnv([]string{"SYSTEM_ENV=system"}),
		WithRequestEnv([]string{"REQUEST_ENV=request"}),
	)
	require.NoError(t, err)

	value, ok := sess.GetEnv("SYSTEM_ENV")
	require.True(t, ok)
	require.Equal(t, "system", value)

	value, ok = sess.GetEnv("REQUEST_ENV")
	require.True(t, ok)
	require.Equal(t, "request", value)

	_, ok = sess.GetEnv("CWD_DOTENV")
	require.False(t, ok)

	_, ok = sess.GetEnv("CWD_DIRENV")
	require.False(t, ok)
}
