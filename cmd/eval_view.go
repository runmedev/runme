package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type evalViewer interface {
	Run(args []string) error
}

var newEvalViewer = func(opts harbor.EvalViewOptions) evalViewer {
	return harbor.NewEvalViewer(opts)
}

type evalViewOptions struct {
	jobsDir     string
	port        int
	noOpen      bool
	runmeHarbor string
	debug       bool
	stdout      io.Writer
	stderr      io.Writer
}

func evalViewCmd() *cobra.Command {
	opts := evalViewOptions{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "view [jobs-dir]",
		Short: "View Harbor eval jobs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			if len(args) > 0 {
				opts.jobsDir = args[0]
			}
			return runEvalView(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&opts.port, "port", 0, "Dashboard port; defaults to the first open port from 8080")
	flags.BoolVar(&opts.noOpen, "no-open", false, "Do not open the dashboard in the default browser")
	flags.StringVar(&opts.runmeHarbor, "runme-harbor-bin", "", "runme-harbor executable")
	flags.BoolVar(&opts.debug, "debug", false, "Print delegated commands")

	return cmd
}

func runEvalView(opts evalViewOptions, args []string) error {
	err := newEvalViewer(harbor.EvalViewOptions{
		JobsDir:        opts.jobsDir,
		Port:           opts.port,
		Open:           !opts.noOpen,
		RunmeHarborBin: opts.runmeHarbor,
		Debug:          opts.debug,
		Stdout:         opts.stdout,
		Stderr:         opts.stderr,
	}).Run(args)
	if err == nil {
		return nil
	}
	if errors.Is(err, harbor.ErrRunmeHarborMissing) {
		return fmt.Errorf("`runme eval view` requires the optional Python package `runme-harbor`.\n\nInstall it with:\n  uv tool install runme-harbor\n\nThen retry:\n  runme eval view")
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return ExitCodeError{Code: exitErr.ExitCode(), Err: err}
	}
	var codeErr interface{ ExitCode() int }
	if errors.As(err, &codeErr) {
		return ExitCodeError{Code: codeErr.ExitCode(), Err: err}
	}
	return err
}
