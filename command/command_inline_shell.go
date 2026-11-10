package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/internal/dagger"
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
	if _, _, err := c.PrepareProgram(ctx); err != nil {
		return err
	}
	script, err := c.build(ctx)
	if err != nil {
		return err
	}
	c.logger.Debug("inline shell script", zap.String("script", script))

	if script != "" {
		c.SetPayloadArgs("-c", script)
	}

	return c.internalCommand.Start(ctx)
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

func (c *inlineShellCommand) build(ctx context.Context) (string, error) {
	buf := bytes.NewBuffer(nil)
	bw := bulkWriter{Writer: buf}

	options, err := c.shellOptions(ctx)
	if err != nil {
		return "", err
	}
	if options != "" {
		bw.WriteString(options)
		bw.WriteString("\n\n")
	}

	// If the session is provided, we need to collect the environment before and after the script execution.
	// Here, we dump env before the script execution and use trap on EXIT to collect the env after the script execution.
	if c.envCollector != nil {
		err = c.envCollector.SetOnShell(buf)
		if err != nil {
			return "", err
		}
	}

	cfg := c.ProgramConfig()

	program, args, err := c.PrepareProgram(ctx)
	if err != nil {
		return "", err
	}
	var source strings.Builder
	if commands := cfg.GetCommands(); commands != nil {
		for _, cmd := range commands.Items {
			_, _ = source.WriteString(cmd)
			_ = source.WriteByte('\n')
		}
	} else {
		_, _ = source.WriteString(cfg.GetScript())
	}
	bw.WriteString(dagger.SyncShebang(source.String(), program, args))

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

func (c *inlineShellCommand) shellOptions(ctx context.Context) (string, error) {
	program, _, err := c.PrepareProgram(ctx)
	if err != nil {
		return "", err
	}

	shell := filepath.Base(program)

	// TODO(mxs): powershell and DOS are missing
	switch shell {
	case "zsh", "ksh", "bash":
		result := "set -e -o pipefail"
		if c.debug {
			result += " -x"
		}
		return result, nil
	case "sh":
		return "set -e", nil
	default:
		return "", nil
	}
}
