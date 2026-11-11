package command

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/internal/execution"
	"github.com/runmedev/runme/v3/session"
)

type preparationLauncher struct {
	*base
	launches int
	program  string
	args     []string
}

func (c *preparationLauncher) Start(ctx context.Context) error {
	invocation, err := c.prepareProgram(ctx)
	if err != nil {
		return err
	}
	return c.startProgram(ctx, invocation)
}

func (c *preparationLauncher) startProgram(_ context.Context, invocation execution.Invocation) error {
	c.launches++
	c.program = invocation.Executable()
	c.args = invocation.Arguments()
	return nil
}

type preparationCollector struct{}

func (preparationCollector) ExtraEnv() []string { return []string{"PREPARATION_ENV=present"} }
func (preparationCollector) SetOnShell(w io.Writer) error {
	_, err := io.WriteString(w, "# preamble\n")
	return err
}
func (preparationCollector) Diff() ([]string, []string, error) { return nil, nil, nil }

func TestSourcePreparation(t *testing.T) {
	for _, kind := range []string{"inline", "shell", "file"} {
		for _, commands := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "Script", true: "Commands"}[commands], func(t *testing.T) {
				sess, err := session.New()
				require.NoError(t, err)
				source := "echo shell"
				cfg := &ProgramConfig{ProgramName: "dagger shell", Arguments: []string{"--debug"}, Env: []string{"CONFIG_ENV=present"}}
				cfg.Source = &runnerv2.ProgramConfig_Script{Script: source}
				if commands {
					cfg.Source = &runnerv2.ProgramConfig_Commands{Commands: &runnerv2.ProgramConfig_CommandList{Items: []string{"echo shell"}}}
				}
				calls := 0
				b := &base{cfg: cfg, session: sess, runtime: dockerRuntime{}, logger: zap.NewNop()}
				b.resolveInvocation = func(_ context.Context, invocation execution.Invocation, env []string) (execution.Invocation, error) {
					calls++
					require.Equal(t, "dagger", invocation.Executable())
					require.Equal(t, []string{"shell", "--debug"}, invocation.Arguments())
					require.Contains(t, env, "CONFIG_ENV=present")
					if kind == "shell" {
						require.Contains(t, env, "PREPARATION_ENV=present")
					}
					return execution.NewInvocation(invocation.Executable(), []string{"script", "--debug"}), nil
				}
				launch := &preparationLauncher{base: b}
				var cmd internalCommand
				switch kind {
				case "inline":
					cmd = &inlineCommand{internalCommand: launch}
				case "shell":
					cmd = &inlineShellCommand{internalCommand: launch, logger: zap.NewNop(), envCollector: preparationCollector{}}
				case "file":
					f := &fileCommand{internalCommand: launch, logger: zap.NewNop()}
					t.Cleanup(func() { require.NoError(t, f.removeTempDir()) })
					cmd = f
				}
				require.NoError(t, cmd.Start(context.Background()))
				require.Equal(t, []string{"--debug"}, cfg.Arguments)
				require.Equal(t, []string{"script", "--debug"}, launch.args[:2])
				payload := launch.args[len(launch.args)-1]
				if kind == "file" {
					data, err := os.ReadFile(filepath.Join(cfg.Directory, payload))
					require.NoError(t, err)
					payload = string(data)
				}
				require.Contains(t, payload, "echo shell")
				require.True(t, strings.HasPrefix(payload, "#!/usr/bin/env -S dagger script --debug\n"))
				if kind == "shell" {
					require.Contains(t, payload, "\n# preamble\n")
				}
				if kind == "shell" {
					require.Len(t, launch.args, 4)
				} else {
					require.Len(t, launch.args, 3)
				}
				require.Equal(t, 1, launch.launches)
				require.Equal(t, 1, calls)
			})
		}
	}
}

func TestNonDaggerSourcePreparationIsUnchanged(t *testing.T) {
	sess, err := session.New()
	require.NoError(t, err)
	cfg := &ProgramConfig{
		ProgramName: "bash",
		Source:      &runnerv2.ProgramConfig_Script{Script: "echo shell"},
	}
	b := &base{cfg: cfg, session: sess, runtime: dockerRuntime{}, logger: zap.NewNop()}
	b.resolveInvocation = func(_ context.Context, invocation execution.Invocation, _ []string) (execution.Invocation, error) {
		return invocation, nil
	}
	launch := &preparationLauncher{base: b}
	cmd := &inlineCommand{internalCommand: launch}
	require.NoError(t, cmd.Start(context.Background()))
	require.Equal(t, "echo shell", launch.args[len(launch.args)-1])
}

func TestPreparationFailureDoesNotLaunch(t *testing.T) {
	sess, err := session.New()
	require.NoError(t, err)
	expected := errors.New("resolution failed")
	calls := 0
	b := &base{cfg: &ProgramConfig{ProgramName: "dagger script"}, runtime: dockerRuntime{}, session: sess, logger: zap.NewNop()}
	b.resolveInvocation = func(context.Context, execution.Invocation, []string) (execution.Invocation, error) {
		calls++
		return execution.Invocation{}, expected
	}
	launch := &preparationLauncher{base: b}
	cmd := &inlineCommand{internalCommand: launch}
	require.ErrorIs(t, cmd.Start(context.Background()), expected)
	require.Equal(t, 1, calls)
	require.Zero(t, launch.launches)
}
