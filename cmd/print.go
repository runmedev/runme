package cmd

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/runner/client"
)

func printCmd() *cobra.Command {
	fRaw := false
	fNoNewLine := false

	cmd := cobra.Command{
		Use:               "print",
		Short:             "Print a selected snippet",
		Long:              "Print will display the details of the corresponding command block based on its name.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: validCmdNames,
		RunE: func(cmd *cobra.Command, args []string) error {
		generateBlocks:
			tasks, err := getProjectTasks(cmd)
			if err != nil {
				return err
			}

			task, err := lookupTaskWithPrompt(cmd, args[0], tasks)
			if isTaskNotFoundError(err) && !fAllowUnnamed {
				fAllowUnnamed = true
				goto generateBlocks
			} else if err != nil {
				return err
			}

			baseShell := "" // not necessary for printing only
			_, lines, _, err := client.GetTaskProgram(baseShell, task)
			if err != nil {
				return err
			}
			value := []byte(strings.Join(lines, "\n"))

			if fRaw {
				value = task.CodeBlock.Value()
			}

			if !fNoNewLine {
				value = append(value, '\n')
			}

			w := bulkWriter{
				Writer: cmd.OutOrStdout(),
			}
			w.Write(value)
			return errors.Wrap(w.Err(), "failed to write to stdout")
		},
	}

	setDefaultFlags(&cmd)
	cmd.Flags().BoolVarP(&fRaw, "raw", "r", false, "Print the raw command without transforming it")
	cmd.Flags().BoolVarP(&fNoNewLine, "skip-newline", "n", false, "Do not print newline after the command")

	return &cmd
}

type bulkWriter struct {
	io.Writer
	n   int
	err error
}

func (w *bulkWriter) Err() error {
	return w.err
}

func (w *bulkWriter) Result() (int, error) {
	return w.n, w.err
}

func (w *bulkWriter) Write(p []byte) {
	if w.err != nil {
		return
	}
	n, err := w.Writer.Write(p)
	w.n += n
	w.err = err
}
