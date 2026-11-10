package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/internal/execution"
	"github.com/runmedev/runme/v3/session"
)

type inlineShellCommand struct {
	internalCommand

	debug        bool
	envCollector envCollector
	logger       *zap.Logger
	session      *session.Session
}

func (c *inlineShellCommand) getPty() *os.File {
	cmd, ok := c.internalCommand.(commandWithPty)
	if !ok {
		return nil
	}
	return cmd.getPty()
}

func (c *inlineShellCommand) Start(ctx context.Context) error {
	cfg := c.ProgramConfig()
	if c.envCollector != nil {
		cfg.Env = append(cfg.Env, c.envCollector.ExtraEnv()...)
	}
	invocation, err := c.prepareProgram(ctx)
	if err != nil {
		return err
	}
	script, err := c.build(invocation)
	if err != nil {
		return err
	}
	c.logger.Debug("inline shell script", zap.String("script", script))

	if script != "" {
		invocation = invocation.WithArguments("-c", script)
	}

	return c.internalCommand.startProgram(ctx, invocation)
}

func (c *inlineShellCommand) Wait(ctx context.Context) error {
	err := c.internalCommand.Wait(ctx)

	if c.envCollector != nil {
		c.logger.Info(
			"collecting the environment after the script execution",
			zap.Int("count", len(c.session.GetAllEnv())), // TODO(adamb): change to session.Size()
		)

		// todo(sebastian): in v1 we don't collect unless successful. what's correct?
		cErr := c.collectEnv(ctx)

		c.logger.Info(
			"collected the environment after the script execution",
			zap.Int("count", len(c.session.GetAllEnv())), // TODO(adamb): change to session.Size()
			zap.Error(cErr),
		)

		if cErr != nil && err == nil {
			err = cErr
		}
	}

	return err
}

func (c *inlineShellCommand) build(invocation execution.Invocation) (string, error) {
	buf := bytes.NewBuffer(nil)
	bw := bulkWriter{Writer: buf}

	options := c.shellOptions(invocation)
	if options != "" {
		bw.WriteString(options)
		bw.WriteString("\n\n")
	}

	// If the session is provided, we need to collect the environment before and after the script execution.
	// Here, we dump env before the script execution and use trap on EXIT to collect the env after the script execution.
	if c.envCollector != nil {
		if err := c.envCollector.SetOnShell(buf); err != nil {
			return "", err
		}
	}

	cfg := c.ProgramConfig()

	if commands := cfg.GetCommands(); commands != nil {
		for _, cmd := range commands.Items {
			bw.WriteString(cmd)
			bw.WriteByte('\n')
		}
	} else if script := cfg.GetScript(); script != "" {
		bw.WriteString(script)
	}

	return buf.String(), nil
}

func (c *inlineShellCommand) collectEnv(ctx context.Context) error {
	if c.envCollector == nil {
		return nil
	}

	changed, deleted, err := c.envCollector.Diff()
	if err != nil {
		return err
	}

	err = c.session.SetEnv(ctx, changed...)
	if err != nil {
		return errors.WithMessage(err, "failed to set the new or updated env")
	}

	return c.session.DeleteEnv(ctx, deleted...)
}

func (c *inlineShellCommand) shellOptions(invocation execution.Invocation) string {
	shell := filepath.Base(invocation.Executable())

	// TODO(mxs): powershell and DOS are missing
	switch shell {
	case "zsh", "ksh", "bash":
		result := "set -e -o pipefail"
		if c.debug {
			result += " -x"
		}
		return result
	case "sh":
		return "set -e"
	default:
		return ""
	}
}
