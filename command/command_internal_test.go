//go:build !windows

package command

import (
	"bytes"
	"strings"
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

			if tc.LanguageID == "typescript" && tc.ProgramName == "" {
				assertTypeScriptInterpreter(t, programPath, args)
				return
			}

			// Check that the program path contains the expected pattern
			assert.Contains(t, programPath, tc.ExpectedProgramPathPattern,
				"ProgramName should contain expected pattern")

			// Check args
			assert.Equal(t, tc.ExpectedArgs, args,
				"Arguments should match expected value")
		})
	}
}

func assertTypeScriptInterpreter(t *testing.T, programPath string, args []string) {
	t.Helper()

	switch {
	case strings.Contains(programPath, "ts-node"):
		assert.Nil(t, args)
	case strings.Contains(programPath, "deno"):
		assert.Equal(t, []string{"run"}, args)
	case strings.Contains(programPath, "bun"):
		assert.Equal(t, []string{"run"}, args)
	case strings.Contains(programPath, "cat"):
		assert.Nil(t, args)
	default:
		t.Fatalf("unexpected TypeScript interpreter resolution: programPath=%q args=%#v", programPath, args)
	}
}
