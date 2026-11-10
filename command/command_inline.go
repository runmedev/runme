package command

import (
	"bytes"
	"context"
	"os"

	"go.uber.org/zap"

	"github.com/runmedev/runme/v3/internal/dagger"
)

type inlineCommand struct {
	internalCommand

	logger *zap.Logger
}

func (c *inlineCommand) getPty() *os.File {
	cmd, ok := c.internalCommand.(commandWithPty)
	if !ok {
		return nil
	}
	return cmd.getPty()
}

func (c *inlineCommand) Start(ctx context.Context) error {
	program, args, err := c.PrepareProgram(ctx)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	bw := bulkWriter{Writer: buf}
	cfg := c.ProgramConfig()

	// Write the script from the commands or the script.
	if commands := cfg.GetCommands(); commands != nil {
		for _, cmd := range commands.Items {
			bw.WriteString(cmd)
			bw.WriteByte('\n')
		}
	} else if script := cfg.GetScript(); script != "" {
		bw.WriteString(script)
	}

	// TODO(adamb): "-c" is not supported for all inline programs.
	if val := buf.String(); val != "" {
		c.SetPayloadArgs(dagger.SyncShebang(val, program, args))
	}

	return c.internalCommand.Start(ctx)
}
