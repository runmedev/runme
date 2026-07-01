package cmd

import (
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type evalPromoter interface {
	Run(args []string) error
}

var newEvalPromoter = func(opts harbor.EvalPromoteOptions) evalPromoter {
	return harbor.NewEvalPromoter(opts)
}

type evalPromoteOptions struct {
	jobsDir       string
	job           string
	datasetPath   string
	latest        bool
	dryRun        bool
	evidenceOnly  bool
	artifacts     bool
	includeOracle bool
	allowErrors   bool
	promoteAnyway bool
	message       string
	stdout        io.Writer
	stderr        io.Writer
}

func evalPromoteCmd() *cobra.Command {
	opts := evalPromoteOptions{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "promote [dataset-path]",
		Short: "Commit staged changes with eval job evidence",
		Long: `Commit staged changes with eval job evidence.

When dataset-path is omitted, runme eval promote uses ./` + harbor.DefaultEvalDatasetPath + `.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			if len(args) > 0 {
				opts.datasetPath = args[0]
			}
			return runEvalPromote(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.jobsDir, "jobs-dir", "", "Eval jobs directory; defaults to .runme/evals/jobs under the project root")
	flags.StringVar(&opts.job, "job", "", "Eval job directory to promote")
	flags.BoolVar(&opts.latest, "latest", false, "Promote the latest eval job under --jobs-dir")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Print what would be committed without staging or committing")
	flags.BoolVar(&opts.evidenceOnly, "evidence-only", false, "Commit only the selected eval job evidence when no source changes are staged")
	flags.BoolVar(&opts.artifacts, "artifacts", false, "Include full eval artifacts such as logs and trial outputs; may contain sensitive information")
	flags.BoolVar(&opts.includeOracle, "include-oracle", false, "Allow promoting eval jobs that only used Harbor's oracle agent")
	flags.BoolVar(&opts.allowErrors, "allow-errors", false, "Allow promoting eval jobs with errored trials")
	flags.BoolVar(&opts.promoteAnyway, "promote-anyway", false, "Promote even when eval comparison blocks promotion")
	flags.StringVar(&opts.message, "message", "", "Commit subject line; eval evidence is added to the commit body")

	return cmd
}

func runEvalPromote(opts evalPromoteOptions, args []string) error {
	err := newEvalPromoter(harbor.EvalPromoteOptions{
		JobsDir:       opts.jobsDir,
		Job:           opts.job,
		DatasetPath:   opts.datasetPath,
		Latest:        opts.latest,
		DryRun:        opts.dryRun,
		EvidenceOnly:  opts.evidenceOnly,
		Artifacts:     opts.artifacts,
		IncludeOracle: opts.includeOracle,
		AllowErrors:   opts.allowErrors,
		PromoteAnyway: opts.promoteAnyway,
		Message:       opts.message,
		Stdout:        opts.stdout,
		Stderr:        opts.stderr,
	}).Run(args)
	if err == nil {
		return nil
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
