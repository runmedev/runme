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

func TestProjectDotenvPrecedence(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".env"), []byte("SHARED=base\nBASE_ONLY=yes\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".env.local"), []byte("SHARED=local\n"), 0o600))
	proj, err := project.NewDirProject(root, project.WithEnvFilesReadOrder([]string{".env", ".env.local"}))
	require.NoError(t, err)

	t.Run("map", func(t *testing.T) {
		sess, err := New(WithProject(proj), WithSeedEnv([]string{"SHARED=system", "SYSTEM_ONLY=yes"}))
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"SHARED=local", "BASE_ONLY=yes", "SYSTEM_ONLY=yes"}, sess.GetAllEnv())
	})

	t.Run("owl", func(t *testing.T) {
		sess, err := New(WithProject(proj), WithOwl(true), WithSeedEnv([]string{"SHARED=system", "SYSTEM_ONLY=yes"}))
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"SHARED=local", "BASE_ONLY=yes", "SYSTEM_ONLY=yes"}, sess.GetAllEnv())
	})
}
