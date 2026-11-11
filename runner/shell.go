package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/internal/dagger"
	"github.com/runmedev/runme/v3/internal/execution"
	"github.com/runmedev/runme/v3/internal/system"
	"github.com/runmedev/runme/v3/project"
)

type Shell struct {
	*ExecutableConfig
	command     *command
	Cmds        []string
	CustomShell string
}

var _ Executable = (*Shell)(nil)

func (s Shell) ProgramPath() string {
	return resolveShellPath(s.CustomShell)
}

func (s Shell) ShellType() string {
	return ShellFromShellPath(s.ProgramPath())
}

func (s Shell) DryRun(ctx context.Context, w io.Writer) error {
	var b bytes.Buffer

	_, _ = b.WriteString(fmt.Sprintf("# run in %q\n\n", s.Dir))
	_, _ = b.WriteString(prepareScriptFromCommands(s.Cmds, s.ShellType()))
	body := b.String()
	if dagger.IsCommand(s.CustomShell) {
		invocation, err := resolveDryRunInvocation(ctx, s.CustomShell, s.Session)
		if err != nil {
			return err
		}
		body = dagger.RenderScript(invocation, "\n"+body)
	} else {
		body = fmt.Sprintf("#!%s\n\n%s", s.ProgramPath(), body)
	}
	_, err := io.WriteString(w, body)
	return err
}

func resolveDryRunInvocation(ctx context.Context, command string, session *Session) (execution.Invocation, error) {
	program, arguments := parseFileProgram(command)
	env := os.Environ()
	if session != nil {
		var err error
		env, err = session.Envs()
		if err != nil {
			return execution.Invocation{}, err
		}
	}
	return dagger.ResolveInvocation(ctx, execution.NewInvocation(program, arguments), env)
}

func (s *Shell) Run(ctx context.Context) error {
	cmd, err := newCommand(
		ctx,
		&commandConfig{
			ProgramName: s.ProgramPath(),
			Directory:   s.Dir,
			Session:     s.Session,
			Tty:         s.Tty,
			Stdin:       s.Stdin,
			Stdout:      s.Stdout,
			Stderr:      s.Stderr,
			CommandMode: CommandModeInlineShell,
			Commands:    s.Cmds,
			Script:      "",
			KnownName:   s.KnownName,
			Logger:      s.Logger,
		},
	)
	if err != nil {
		return err
	}
	s.command = cmd
	return s.run(ctx, cmd)
}

func (s Shell) ExitCode() int {
	if s.command == nil || s.command.cmd == nil {
		return -1
	}

	return s.command.cmd.ProcessState.ExitCode()
}

func (s Shell) run(ctx context.Context, cmd *command) error {
	opts := &startOpts{}
	if s.Tty {
		opts.DisableEcho = true
	}

	if err := cmd.StartWithOpts(ctx, opts); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		var exiterr *exec.ExitError
		// Ignore errors caused by SIGINT.
		if errors.As(err, &exiterr) {
			var rerr error = ExitErrorFromExec(exiterr)

			if exiterr.ProcessState.Sys().(syscall.WaitStatus).Signal() != os.Kill {
				msg := "failed to run command"
				if len(s.KnownName) > 0 {
					msg += " " + strconv.Quote(s.KnownName)
				}
				return errors.Wrap(rerr, msg)
			}

			return rerr
		}
	}

	return nil
}

func IsShellLanguage(languageID string) bool {
	switch strings.ToLower(languageID) {
	// shellscripts
	case "sh", "bash", "zsh", "ksh", "shell", "shellscript":
		return true

	// dos
	case "bat", "cmd":
		return true

	// powershell
	case "powershell", "pwsh":
		return true

	// fish
	case "fish":
		return true

	default:
		return false
	}
}

func GetShellProgram(baseShell string, task project.Task) (intent execution.Intent, commandMode CommandMode) {
	customShell, err := task.CustomShell(baseShell)
	if err != nil {
		program, arguments := parseFileProgram(baseShell)
		return execution.NewIntent(program, arguments), CommandModeTempFile
	}

	program := ""
	switch {
	case dagger.IsCommand(customShell):
		program = customShell
		commandMode = CommandModeDaggerShell
	case IsShellLanguage(task.CodeBlock.Language()):
		program = customShell
		commandMode = CommandModeInlineShell
	default:
		commandMode = CommandModeTempFile
	}

	if interpreter := task.BlockInterpreter(); interpreter != "" {
		program = interpreter
	}

	program, arguments := parseFileProgram(program)
	return execution.NewIntent(program, arguments), commandMode
}

func resolveShellPath(customShell string) string {
	if customShell != "" {
		programParts := strings.Split(customShell, " ")
		args := []string{}
		if len(programParts) > 1 {
			args = programParts[1:]
		}
		if path, err := system.LookPath(programParts[0]); err == nil {
			return strings.Join(append([]string{path}, args...), " ")
		}
	}

	return globalShellPath()
}

func globalShellPath() string {
	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		shell = "sh"
	}
	if path, err := system.LookPath(shell); err == nil {
		return path
	}
	return "/bin/sh"
}

// TODO(mxs): this method for determining shell is not strong, since shells can
// be aliased. we should probably run the shell to get this information
func ShellFromShellPath(programPath string) string {
	programFile := filepath.Base(programPath)
	return programFile[:len(programFile)-len(filepath.Ext(programFile))]
}

func PrepareScriptFromCommands(cmds []string, shell string) string {
	return prepareScriptFromCommands(cmds, shell)
}

func prepareScriptFromCommands(cmds []string, shell string) string {
	var b strings.Builder

	_, _ = b.WriteString(getShellOptions(shell))

	for _, cmd := range cmds {
		_, _ = b.WriteString(cmd)
		_, _ = b.WriteRune('\n')
	}

	_, _ = b.WriteRune('\n')

	return b.String()
}

func prepareScript(script string, shell string) string {
	var b strings.Builder

	_, _ = b.WriteString(getShellOptions(shell))

	_, _ = b.WriteString(script)
	_, _ = b.WriteRune('\n')

	return b.String()
}

func getShellOptions(shell string) (res string) {
	// TODO(mxs): powershell, DOS
	switch shell {
	case "zsh", "ksh", "bash":
		res += "set -e -o pipefail"
	case "sh":
		res += "set -e"
	}

	res += "\n"
	return
}
