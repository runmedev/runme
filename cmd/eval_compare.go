package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type evalComparer interface {
	Run(args []string) error
}

var newEvalComparer = func(opts harbor.EvalCompareOptions) evalComparer {
	return harbor.NewEvalComparer(opts)
}

type evalCompareOptions struct {
	jobsDir       string
	job           string
	datasetPath   string
	base          string
	format        string
	includeOracle bool
	allowErrors   bool
	stdout        io.Writer
	stderr        io.Writer
}

func evalCompareCmd() *cobra.Command {
	opts := evalCompareOptions{
		base:   "HEAD",
		format: "text",
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "compare [dataset-path]",
		Short: "Compare the latest Git-tracked eval job with the latest local eval job",
		Long: `Compare eval jobs for a dataset.

When dataset-path is omitted, runme eval compare uses ./` + harbor.DefaultEvalDatasetPath + `.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			if len(args) > 0 {
				opts.datasetPath = args[0]
			}
			return runEvalCompare(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.jobsDir, "jobs-dir", "", "Eval jobs directory; defaults to .runme/evals/jobs under the project root")
	flags.StringVar(&opts.job, "job", "", "Compare against a specific local eval job")
	flags.StringVar(&opts.base, "base", "HEAD", "Git ref used to find the tracked baseline eval job")
	flags.StringVar(&opts.format, "format", "text", "Output format: text or json")
	flags.BoolVar(&opts.includeOracle, "include-oracle", false, "Allow comparing eval jobs that only used Harbor's oracle agent")
	flags.BoolVar(&opts.allowErrors, "allow-errors", false, "Allow comparing eval jobs with errored trials")

	return cmd
}

func runEvalCompare(opts evalCompareOptions, args []string) error {
	err := newEvalComparer(harbor.EvalCompareOptions{
		JobsDir:       opts.jobsDir,
		Job:           opts.job,
		DatasetPath:   opts.datasetPath,
		Base:          opts.base,
		Format:        opts.format,
		IncludeOracle: opts.includeOracle,
		AllowErrors:   opts.allowErrors,
		Stdout:        opts.stdout,
		Stderr:        opts.stderr,
	}).Run(args)
	return asExitCodeError(err)
}
