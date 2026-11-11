package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/runmedev/runme/v3/internal/dagger"
)

type ShellRaw struct {
	*Shell
}

var _ Executable = (*ShellRaw)(nil)

func (s ShellRaw) DryRun(ctx context.Context, w io.Writer) error {
	var b bytes.Buffer

	_, _ = b.WriteString(fmt.Sprintf("// run in %q\n\n", s.Dir))
	_, _ = b.WriteString(prepareScript(strings.Join(s.Cmds, "\n"), s.ShellType()))
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

func (s ShellRaw) Run(ctx context.Context) error {
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
			Commands:    nil,
			Script:      strings.Join(s.Cmds, "\n"),
			KnownName:   s.KnownName,
			Logger:      s.Logger,
		},
	)
	if err != nil {
		return err
	}
	return s.run(ctx, cmd)
}
