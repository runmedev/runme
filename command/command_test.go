package command

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/command/testdata"
	"github.com/runmedev/runme/v3/session"
)

func init() {
	SetEnvDumpCommandForTesting()
}

func testExecuteCommand(
	t *testing.T,
	cfg *ProgramConfig,
	input io.Reader,
	expectedStdout string,
	expectedStderr string,
) {
	t.Helper()

	testExecuteCommandWithSession(t, cfg, nil, input, expectedStdout, expectedStderr)
}

func testExecuteCommandWithSession(
	t *testing.T,
	cfg *ProgramConfig,
	session *session.Session,
	input io.Reader,
	expectedStdout string,
	expectedStderr string,
) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	factory := NewFactory(WithLogger(zaptest.NewLogger(t)))
	options := CommandOptions{
		Session: session,
		Stdout:  stdout,
		Stderr:  stderr,
		Stdin:   input,
	}
	command, err := factory.Build(cfg, options)
	require.NoError(t, err)
	err = command.Start(context.Background())
	require.NoError(t, err)
	err = command.Wait(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedStdout, stdout.String())
	assert.Equal(t, expectedStderr, stderr.String())
}

func Test_InternalCommand_DetectProgramPath(t *testing.T) {
	t.Parallel()

	for _, tc := range testdata.DetectProgramPathTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			stdout := bytes.NewBuffer(nil)
			stderr := bytes.NewBuffer(nil)

			f := &commandFactory{
				logger: zaptest.NewLogger(t),
			}

			cfg := &ProgramConfig{
				ProgramName: tc.ProgramName,
				LanguageId:  tc.LanguageID,
				Mode:        runnerv2.CommandMode_COMMAND_MODE_FILE,
			}
			if isShell(cfg) {
				cfg.Mode = runnerv2.CommandMode_COMMAND_MODE_INLINE
			}

			opts := CommandOptions{
				Stdout: stdout,
				Stderr: stderr,
			}
			base := f.buildBase(cfg, opts)
			internal := f.buildInternal(base, opts)

			programPath, args, err := internal.ProgramPath()
			if tc.ExpectError {
				require.Error(t, err)
				return
			}

			// Check for no error
			require.NoError(t, err)

			// Check that the program path contains the expected pattern
			assert.Contains(t, programPath, tc.ExpectedProgramPathPattern,
				"ProgramName should contain expected pattern")

			// Check args
			assert.Equal(t, tc.ExpectedArgs, args,
				"Arguments should match expected value")
		})
	}
}
