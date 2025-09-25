//go:build !windows

package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/command/testdata"
)

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
