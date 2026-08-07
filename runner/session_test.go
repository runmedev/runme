package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/runmedev/owl/pkg/owl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/project"
)

func Test_SessionList(t *testing.T) {
	t.Parallel()

	createSession := func() (*Session, error) {
		return NewSession(nil, nil, zap.NewNop())
	}

	t.Run("UpdatedOnCreate", func(t *testing.T) {
		list := newSessionList()

		session1, err := createSession()
		require.NoError(t, err)
		err = list.Add(session1)
		require.NoError(t, err)

		session2, err := createSession()
		require.NoError(t, err)
		err = list.Add(session2)
		require.NoError(t, err)

		mostRecent, ok := list.Newest()
		require.Equal(t, true, ok)
		assert.Equal(t, session2.ID, mostRecent.ID)
	})

	t.Run("GetSession", func(t *testing.T) {
		list := newSessionList()

		session1, err := createSession()
		require.NoError(t, err)
		err = list.Add(session1)
		require.NoError(t, err)

		session2, err := createSession()
		require.NoError(t, err)

		assert.NotEqual(t, session1.ID, session2.ID)

		err = list.Add(session2)
		require.NoError(t, err)

		found, ok := list.Get(session1)
		require.Equal(t, true, ok)
		assert.Equal(t, session1, found)

		newest, ok := list.Newest()
		require.Equal(t, true, ok)
		assert.Equal(t, session1.ID, newest.ID)
	})

	t.Run("CreateAndAddEntry", func(t *testing.T) {
		list := newSessionList()

		session1, err := list.CreateAndAdd(createSession)
		require.NoError(t, err)

		session2, err := list.CreateAndAdd(createSession)
		require.NoError(t, err)

		assert.NotEqual(t, session1.ID, session2.ID)

		sessions := list.List()

		expected := []string{session1.ID, session2.ID}
		actual := []string{}

		for _, session := range sessions {
			actual = append(actual, session.ID)
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("DeleteEntry", func(t *testing.T) {
		list := newSessionList()

		session1, err := list.CreateAndAdd(createSession)
		require.NoError(t, err)

		session2, err := list.CreateAndAdd(createSession)
		require.NoError(t, err)

		assert.NotEqual(t, session1.ID, session2.ID)

		{
			sessionList := list.List()
			assert.Equal(t, 2, len(sessionList))
		}

		deleted := list.Delete(session2)
		require.Equal(t, true, deleted)

		{
			sessionList := list.List()
			assert.Equal(t, 1, len(sessionList))

			expected := []string{session1.ID}
			actual := []string{}

			for _, session := range sessionList {
				actual = append(actual, session.ID)
			}

			assert.Equal(t, expected, actual)
		}

		deleted = list.Delete(session1)
		require.Equal(t, true, deleted)

		{
			sessionList := list.List()
			assert.Equal(t, 0, len(sessionList))
		}

		list.Delete(session2)
	})
}

func TestOwlSessionTreatsEmptySensitiveEnvAsMasked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".env.spec"),
		[]byte("RUNME_TEST_TOKEN=\"The Runme test token to use for integration tests.\" # Secret\n"),
		0o600,
	))
	proj, err := project.NewDirProject(dir)
	require.NoError(t, err)

	sess, err := NewSessionWithStore([]string{"RUNME_TEST_TOKEN="}, proj, true, zap.NewNop())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	snapshotc := make(chan []owl.SnapshotItem)
	require.NoError(t, sess.Subscribe(ctx, snapshotc))

	snapshot := <-snapshotc
	byName := snapshotItemsByName(snapshot)
	require.Contains(t, byName, "RUNME_TEST_TOKEN")

	assert.Equal(t, "[masked]", byName["RUNME_TEST_TOKEN"].Value)
	assert.Empty(t, byName["RUNME_TEST_TOKEN"].OriginalValue)
	assert.Equal(t, owl.VisibilityMasked, byName["RUNME_TEST_TOKEN"].Visibility)
	assert.Equal(t, "[process]", byName["RUNME_TEST_TOKEN"].Source.Name)
}

func TestOwlSessionRevealsSnapshotWithInsecurePolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	proj, err := project.NewDirProject(dir)
	require.NoError(t, err)

	sess, err := NewSessionWithStore([]string{"TERMINFO=/tmp/terminfo"}, proj, true, zap.NewNop())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	snapshotc := make(chan []owl.SnapshotItem)
	require.NoError(t, sess.SubscribeWithPolicy(ctx, snapshotc, owl.SnapshotPolicy{Reveal: true}))

	snapshot := <-snapshotc
	byName := snapshotItemsByName(snapshot)
	require.Contains(t, byName, "TERMINFO")

	assert.Equal(t, "/tmp/terminfo", byName["TERMINFO"].Value)
	assert.Equal(t, owl.VisibilityLiteral, byName["TERMINFO"].Visibility)
	assert.Equal(t, "[process]", byName["TERMINFO"].Source.Name)
}

func TestOwlSessionSeedsDotenvSource(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".env.example"),
		[]byte("FILE_ONLY=\"File only\" # Plain\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".env"),
		[]byte("FILE_ONLY=from-dotenv\n"),
		0o600,
	))
	proj, err := project.NewDirProject(dir)
	require.NoError(t, err)

	snapshot := owlSessionSnapshot(t, nil, proj, owl.SnapshotPolicy{})
	env := snapshotItemsByName(snapshot)["FILE_ONLY"]

	assert.Equal(t, "from-dotenv", env.Value)
	assert.Equal(t, owl.VisibilityLiteral, env.Visibility)
	assert.Equal(t, "dotenv", env.Source.Kind)
	assert.Equal(t, ".env", filepath.Base(env.Source.Name))
}

func TestOwlSessionAttributesDirenvMatchingObservedValue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake direnv shell script is POSIX-only")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".env.example"),
		[]byte("DIRENV_ONLY=\"Direnv only\" # Plain\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".envrc"),
		[]byte("export DIRENV_ONLY=from-direnv\n"),
		0o600,
	))

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(binDir, "direnv"),
		[]byte("#!/bin/sh\necho '{\"DIRENV_ONLY\":\"from-direnv\"}'\n"),
		0o700,
	))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	proj, err := project.NewDirProject(dir, project.WithEnvDirEnv(true))
	require.NoError(t, err)

	snapshot := owlSessionSnapshot(t, []string{"DIRENV_ONLY=from-direnv"}, proj, owl.SnapshotPolicy{})
	env := snapshotItemsByName(snapshot)["DIRENV_ONLY"]

	assert.Equal(t, "from-direnv", env.Value)
	assert.Equal(t, owl.VisibilityLiteral, env.Visibility)
	assert.Equal(t, ".envrc", env.Source.Name)
	assert.Equal(t, "direnv", env.Source.Kind)
}

func TestOwlSessionKeepsProcessSourceWhenDirenvDiffers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake direnv shell script is POSIX-only")
	}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".env.example"),
		[]byte("DIRENV_ONLY=\"Direnv only\" # Plain\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".envrc"),
		[]byte("export DIRENV_ONLY=from-direnv\n"),
		0o600,
	))

	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(binDir, "direnv"),
		[]byte("#!/bin/sh\necho '{\"DIRENV_ONLY\":\"from-direnv\"}'\n"),
		0o700,
	))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	proj, err := project.NewDirProject(dir, project.WithEnvDirEnv(true))
	require.NoError(t, err)

	snapshot := owlSessionSnapshot(t, []string{"DIRENV_ONLY=from-process"}, proj, owl.SnapshotPolicy{})
	env := snapshotItemsByName(snapshot)["DIRENV_ONLY"]

	assert.Equal(t, "from-process", env.Value)
	assert.Equal(t, owl.VisibilityLiteral, env.Visibility)
	assert.Equal(t, "[process]", env.Source.Name)
	assert.Equal(t, "process", env.Source.Kind)
}

func owlSessionSnapshot(t *testing.T, envs []string, proj *project.Project, policy owl.SnapshotPolicy) []owl.SnapshotItem {
	t.Helper()

	sess, err := NewSessionWithStore(envs, proj, true, zap.NewNop())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	snapshotc := make(chan []owl.SnapshotItem)
	require.NoError(t, sess.SubscribeWithPolicy(ctx, snapshotc, policy))

	return <-snapshotc
}

func snapshotItemsByName(items []owl.SnapshotItem) map[string]owl.SnapshotItem {
	result := make(map[string]owl.SnapshotItem, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}
