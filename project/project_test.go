package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/project/teststub"
)

func TestExtractDataFromLoadEvent(t *testing.T) {
	t.Run("MatchingTypes", func(t *testing.T) {
		event := LoadEvent{
			Type: LoadEventFoundDir,
			Data: LoadEventFoundDirData{
				Path: "/some/path",
			},
		}

		data := ExtractDataFromLoadEvent[LoadEventFoundDirData](event)
		assert.Equal(t, "/some/path", data.Path)
	})

	t.Run("NotMatchingTypes", func(t *testing.T) {
		event := LoadEvent{
			Type: LoadEventFoundDir,
			Data: LoadEventFoundDirData{
				Path: "/some/path",
			},
		}

		require.Panics(t, func() {
			ExtractDataFromLoadEvent[LoadEventStartedWalkData](event)
		})
	})
}

func assertEqualPath(t *testing.T, expected, actual string) {
	t.Helper()

	expected = filepath.Clean(expected)
	actual = filepath.Clean(actual)
	if expected == actual {
		return
	}

	expectedInfo, expectedErr := os.Stat(expected)
	actualInfo, actualErr := os.Stat(actual)
	if expectedErr == nil && actualErr == nil && os.SameFile(expectedInfo, actualInfo) {
		return
	}

	assert.Equal(t, expected, actual)
}

func TestNewDirProject(t *testing.T) {
	temp := t.TempDir()
	testData := teststub.Setup(t, temp)

	t.Run("ProperDirProject", func(t *testing.T) {
		_, err := NewDirProject(testData.DirProjectPath())
		require.NoError(t, err)
	})

	t.Run("ProperGitProject", func(t *testing.T) {
		// git-based project is also a dir-based project.
		_, err := NewDirProject(testData.GitProjectPath())
		require.NoError(t, err)
	})

	t.Run("UnknownDir", func(t *testing.T) {
		unknownDir := testData.Join("unknown-project")
		_, err := NewDirProject(unknownDir)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("RelativePathConvertedToAbsolute", func(t *testing.T) {
		cwd, err := os.Getwd()
		require.NoError(t, err)

		projectDir, err := filepath.Rel(
			cwd,
			teststub.OriginalPath().DirProjectPath(),
		)
		require.NoError(t, err)

		proj, err := NewDirProject(projectDir)
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(proj.Root()), "project root is not absolute: %s", proj.Root())
	})
}

func TestNewDirProject_FallbackOnUnsupportedGitExtensions(t *testing.T) {
	repoDir := t.TempDir()
	_, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	configPath := filepath.Join(repoDir, ".git", "config")
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString("\n[extensions]\n\tworktreeConfig = true\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = NewDirProject(repoDir)
	require.Error(t, err)

	p, err := NewDirProject(repoDir, WithAllowUnsupportedGitExtensions(true))
	require.NoError(t, err)
	require.Nil(t, p.repo)
	assertEqualPath(t, repoDir, p.Root())
}

func TestNewFileProject(t *testing.T) {
	temp := t.TempDir()
	testData := teststub.Setup(t, temp)

	t.Run("UnknownFile", func(t *testing.T) {
		fileProject := testData.Join("unknown-file.md")
		_, err := NewFileProject(fileProject)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("UnknownFileAndRelativePath", func(t *testing.T) {
		_, err := NewFileProject("unknown-file.md")
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("RelativePathConvertedToAbsolute", func(t *testing.T) {
		cwd, err := os.Getwd()
		require.NoError(t, err)

		fileProject, err := filepath.Rel(
			cwd,
			teststub.OriginalPath().ProjectFilePath(),
		)
		require.NoError(t, err)

		proj, err := NewFileProject(fileProject)
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(proj.Root()), "project root is not absolute: %s", proj.Root())
	})

	t.Run("ProperFileProject", func(t *testing.T) {
		_, err := NewFileProject(testData.ProjectFilePath())
		require.NoError(t, err)
	})
}

func TestProjectRoot(t *testing.T) {
	temp := t.TempDir()
	testData := teststub.Setup(t, temp)

	t.Run("GitProject", func(t *testing.T) {
		gitProjectDir := testData.GitProjectPath()
		p, err := NewDirProject(gitProjectDir)
		require.NoError(t, err)
		assertEqualPath(t, gitProjectDir, p.Root())
		assert.True(t, filepath.IsAbs(p.Root()), "project root is not absolute: %s", p.Root())
	})

	t.Run("FileProject", func(t *testing.T) {
		fileProject := testData.ProjectFilePath()
		p, err := NewFileProject(fileProject)
		require.NoError(t, err)
		assert.Equal(t, testData.Root(), p.Root())
		assert.True(t, filepath.IsAbs(p.Root()), "project root is not absolute: %s", p.Root())
	})
}

func TestProjectLoad(t *testing.T) {
	temp := t.TempDir()
	testData := teststub.Setup(t, temp)

	gitProjectDir := testData.GitProjectPath()

	t.Run("GitProject", func(t *testing.T) {
		p, err := NewDirProject(
			gitProjectDir,
			WithIgnoreFilePatterns(".git.bkp"),
			WithIgnoreFilePatterns(".gitignore.bkp"),
		)
		require.NoError(t, err)

		eventc := make(chan LoadEvent)

		events := make([]LoadEvent, 0)
		doneReadingEvents := make(chan struct{})
		go func() {
			defer close(doneReadingEvents)
			for e := range eventc {
				events = append(events, e)
			}
		}()

		p.Load(context.Background(), eventc, false)
		<-doneReadingEvents

		expectedEvents := []LoadEventType{
			LoadEventStartedWalk,
			LoadEventFoundDir,  // "."
			LoadEventFoundFile, // "git-ignored.md"
			LoadEventFoundFile, // "ignored.md"
			LoadEventFoundDir,  // "nested"
			LoadEventFoundFile, // "nested/git-ignored.md"
			LoadEventFoundFile, // "readme.md"
			LoadEventFinishedWalk,
			LoadEventStartedParsingDocument,  // "git-ignored.md"
			LoadEventFinishedParsingDocument, // "git-ignored.md"
			LoadEventFoundTask,
			LoadEventStartedParsingDocument,  // "nested/git-ignored.md"
			LoadEventFinishedParsingDocument, // "nested/git-ignored.md"
			LoadEventFoundTask,
			LoadEventStartedParsingDocument,  // "ignored.md"
			LoadEventFinishedParsingDocument, // "ignored.md"
			LoadEventFoundTask,
			LoadEventStartedParsingDocument,  // "readme.md"
			LoadEventFinishedParsingDocument, // "readme.md"
			LoadEventFoundTask,
			LoadEventFoundTask,
		}
		require.EqualValues(
			t,
			expectedEvents,
			mapLoadEvents(events, func(le LoadEvent) LoadEventType { return le.Type }),
			"collected events: %+v",
			events,
		)
		assert.Equal(t, LoadEventFoundDir, events[1].Type)
		assertEqualPath(t, gitProjectDir, ExtractDataFromLoadEvent[LoadEventFoundDirData](events[1]).Path)
		assert.Equal(t, LoadEventFoundFile, events[2].Type)
		assertEqualPath(t, filepath.Join(gitProjectDir, "git-ignored.md"), ExtractDataFromLoadEvent[LoadEventFoundFileData](events[2]).Path)
		assert.Equal(t, LoadEventFoundFile, events[3].Type)
		assertEqualPath(t, filepath.Join(gitProjectDir, "ignored.md"), ExtractDataFromLoadEvent[LoadEventFoundFileData](events[3]).Path)
		assert.Equal(t, LoadEventFoundDir, events[4].Type)
		assertEqualPath(t, filepath.Join(gitProjectDir, "nested"), ExtractDataFromLoadEvent[LoadEventFoundDirData](events[4]).Path)
		assert.Equal(t, LoadEventFoundFile, events[5].Type)
		assertEqualPath(t, filepath.Join(gitProjectDir, "nested", "git-ignored.md"), ExtractDataFromLoadEvent[LoadEventFoundFileData](events[5]).Path)
		assert.Equal(t, LoadEventFoundFile, events[6].Type)
		assertEqualPath(t, filepath.Join(gitProjectDir, "readme.md"), ExtractDataFromLoadEvent[LoadEventFoundFileData](events[6]).Path)
		assertEqualPath(t, filepath.Join(gitProjectDir, "git-ignored.md"), ExtractDataFromLoadEvent[LoadEventFoundTaskData](events[10]).Task.DocumentPath)
		assertEqualPath(t, filepath.Join(gitProjectDir, "ignored.md"), ExtractDataFromLoadEvent[LoadEventFoundTaskData](events[13]).Task.DocumentPath)
		assertEqualPath(t, filepath.Join(gitProjectDir, "nested", "git-ignored.md"), ExtractDataFromLoadEvent[LoadEventFoundTaskData](events[16]).Task.DocumentPath)
		// Unnamed task
		{
			data := ExtractDataFromLoadEvent[LoadEventFoundTaskData](events[19])

			assertEqualPath(t, filepath.Join(gitProjectDir, "readme.md"), data.Task.DocumentPath)
			assert.Equal(t, "echo-hello", data.Task.CodeBlock.Name())
			assert.True(t, data.Task.CodeBlock.IsUnnamed())
		}
		// Named task
		{
			data := ExtractDataFromLoadEvent[LoadEventFoundTaskData](events[20])

			assertEqualPath(t, filepath.Join(gitProjectDir, "readme.md"), data.Task.DocumentPath)
			assert.Equal(t, "my-task", data.Task.CodeBlock.Name())
			assert.False(t, data.Task.CodeBlock.IsUnnamed())
		}
	})

	gitProjectNestedDir := testData.GitProjectNestedPath()

	t.Run("GitProjectWithNested", func(t *testing.T) {
		pRoot1, err := NewDirProject(
			gitProjectDir,
			WithFindRepoUpward(), // not needed, but let's check if it's noop in this case
			WithIgnoreFilePatterns(".git.bkp"),
			WithIgnoreFilePatterns(".gitignore.bkp"),
		)
		require.NoError(t, err)

		pRoot2, err := NewDirProject(
			gitProjectDir,
			WithIgnoreFilePatterns(".git.bkp"),
			WithIgnoreFilePatterns(".gitignore.bkp"),
		)
		require.NoError(t, err)

		pNested, err := NewDirProject(gitProjectNestedDir,
			WithFindRepoUpward(),
			WithIgnoreFilePatterns(".git.bkp"),
			WithIgnoreFilePatterns(".gitignore.bkp"),
		)
		require.NoError(t, err)

		assertEqualPath(t, pRoot1.fs.Root(), pRoot2.fs.Root())
		assertEqualPath(t, pRoot1.fs.Root(), pNested.fs.Root())
	})

	t.Run("DirProjectWithRespectGitignoreAndIgnorePatterns", func(t *testing.T) {
		p, err := NewDirProject(
			gitProjectDir,
			WithRespectGitignore(true),
			WithIgnoreFilePatterns(".git.bkp"),
			WithIgnoreFilePatterns(".gitignore.bkp"),
			WithIgnoreFilePatterns("ignored.md"),
		)
		require.NoError(t, err)

		eventc := make(chan LoadEvent)

		events := make([]LoadEvent, 0)
		doneReadingEvents := make(chan struct{})
		go func() {
			defer close(doneReadingEvents)
			for e := range eventc {
				events = append(events, e)
			}
		}()

		p.Load(context.Background(), eventc, false)
		<-doneReadingEvents

		expectedEvents := []LoadEventType{
			LoadEventStartedWalk,
			LoadEventFoundDir,  // "."
			LoadEventFoundDir,  // "nested"
			LoadEventFoundFile, // "readme.md"
			LoadEventFinishedWalk,
			LoadEventStartedParsingDocument,  // "readme.md"
			LoadEventFinishedParsingDocument, // "readme.md"
			LoadEventFoundTask,               // unnamed; echo-hello
			LoadEventFoundTask,               // named; my-task
		}
		require.EqualValues(
			t,
			expectedEvents,
			mapLoadEvents(events, func(le LoadEvent) LoadEventType { return le.Type }),
			"found events: %#+v", events,
		)
	})

	projectDir := testData.DirProjectPath()

	t.Run("DirProject", func(t *testing.T) {
		p, err := NewDirProject(projectDir)
		require.NoError(t, err)

		eventc := make(chan LoadEvent)

		events := make([]LoadEvent, 0)
		doneReadingEvents := make(chan struct{})
		go func() {
			defer close(doneReadingEvents)
			for e := range eventc {
				events = append(events, e)
			}
		}()

		p.Load(context.Background(), eventc, false)
		<-doneReadingEvents

		expectedEvents := []LoadEventType{
			LoadEventStartedWalk,
			LoadEventFoundDir,  // "."
			LoadEventFoundFile, // "ignored.md"
			LoadEventFoundFile, // "readme.md"
			LoadEventFoundFile, // "session-01HJS35FZ2K0JBWPVAXPMMVTGN.md"
			LoadEventFinishedWalk,
			LoadEventStartedParsingDocument,  // "ignored.md"
			LoadEventFinishedParsingDocument, // "ignored.md"
			LoadEventFoundTask,
			LoadEventStartedParsingDocument,  // "readme.md"
			LoadEventFinishedParsingDocument, // "readme.md"
			LoadEventFoundTask,               // unnamed; echo-hello
			LoadEventFoundTask,               // named; my-task
			LoadEventStartedParsingDocument,  // "session-01HJS35FZ2K0JBWPVAXPMMVTGN.md"
			LoadEventFinishedParsingDocument, // "session-01HJS35FZ2K0JBWPVAXPMMVTGN.md"
		}
		require.EqualValues(
			t,
			expectedEvents,
			mapLoadEvents(events, func(le LoadEvent) LoadEventType { return le.Type }),
		)
	})

	t.Run("DirProjectWithRespectGitignoreAndIgnorePatterns", func(t *testing.T) {
		p, err := NewDirProject(
			projectDir,
			WithIgnoreFilePatterns("ignored.md"),
		)
		require.NoError(t, err)

		eventc := make(chan LoadEvent)

		events := make([]LoadEvent, 0)
		doneReadingEvents := make(chan struct{})
		go func() {
			defer close(doneReadingEvents)
			for e := range eventc {
				events = append(events, e)
			}
		}()

		p.Load(context.Background(), eventc, false)
		<-doneReadingEvents

		expectedEvents := []LoadEventType{
			LoadEventStartedWalk,
			LoadEventFoundDir,  // "."
			LoadEventFoundFile, // "readme.md"
			LoadEventFoundFile, // "session-01HJS35FZ2K0JBWPVAXPMMVTGN.md"
			LoadEventFinishedWalk,
			LoadEventStartedParsingDocument,  // "readme.md"
			LoadEventFinishedParsingDocument, // "readme.md"
			LoadEventFoundTask,               // unnamed; echo-hello
			LoadEventFoundTask,               // named; my-task
			LoadEventStartedParsingDocument,  // "session-01HJS35FZ2K0JBWPVAXPMMVTGN.md"
			LoadEventFinishedParsingDocument, // "session-01HJS35FZ2K0JBWPVAXPMMVTGN.md"
		}
		require.EqualValues(
			t,
			expectedEvents,
			mapLoadEvents(events, func(le LoadEvent) LoadEventType { return le.Type }),
		)
	})

	fileProject := testData.ProjectFilePath()

	t.Run("FileProject", func(t *testing.T) {
		p, err := NewFileProject(fileProject)
		require.NoError(t, err)

		eventc := make(chan LoadEvent)

		events := make([]LoadEvent, 0)
		doneReadingEvents := make(chan struct{})
		go func() {
			defer close(doneReadingEvents)
			for e := range eventc {
				events = append(events, e)
			}
		}()

		p.Load(context.Background(), eventc, false)
		<-doneReadingEvents

		expectedEvents := []LoadEventType{
			LoadEventStartedWalk,
			LoadEventFoundFile, // "file-project.md"
			LoadEventFinishedWalk,
			LoadEventStartedParsingDocument,  // "file-project.md"
			LoadEventFinishedParsingDocument, // "file-project.md"
			LoadEventFoundTask,
		}
		require.EqualValues(
			t,
			expectedEvents,
			mapLoadEvents(events, func(le LoadEvent) LoadEventType { return le.Type }),
		)

		assert.Equal(
			t,
			LoadEvent{
				Type: LoadEventFoundFile,
				Data: LoadEventFoundFileData{Path: fileProject},
			},
			events[1],
		)
		assert.Equal(
			t,
			fileProject,
			ExtractDataFromLoadEvent[LoadEventFoundTaskData](events[5]).Task.DocumentPath,
		)
	})
}

func TestLoadTasks(t *testing.T) {
	temp := t.TempDir()
	testData := teststub.Setup(t, temp)

	gitProjectDir := testData.GitProjectPath()
	p, err := NewDirProject(gitProjectDir, WithIgnoreFilePatterns(".*.bkp"))
	require.NoError(t, err)

	tasks, err := LoadTasks(context.Background(), p)
	require.NoError(t, err)
	assert.Len(t, tasks, 5)
}

func TestLoadEnv(t *testing.T) {
	temp := t.TempDir()
	testData := teststub.Setup(t, temp)

	gitProjectDir := testData.GitProjectPath()
	p, err := NewDirProject(gitProjectDir, WithIgnoreFilePatterns(".*.bkp"), WithEnvFilesReadOrder([]string{".env"}))
	require.NoError(t, err)

	env, err := p.LoadEnv()
	require.NoError(t, err)
	assert.Len(t, env, 1)
	assert.Equal(t, "PROJECT_ENV_FROM_DOTFILE=1", env[0])
}

func mapLoadEvents[T any](events []LoadEvent, fn func(LoadEvent) T) []T {
	result := make([]T, 0, len(events))

	for _, e := range events {
		result = append(result, fn(e))
	}

	return result
}
