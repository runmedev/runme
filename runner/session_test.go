package runner

import (
	"context"
	"os"
	"path/filepath"
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

func snapshotItemsByName(items []owl.SnapshotItem) map[string]owl.SnapshotItem {
	result := make(map[string]owl.SnapshotItem, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}
