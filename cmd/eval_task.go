package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type evalTaskNewer interface {
	Run(args []string) error
}

var newEvalTaskNewer = func(opts harbor.EvalTaskNewOptions) evalTaskNewer {
	return harbor.NewEvalTaskNewer(opts)
}

type evalTaskNewOptions struct {
	tasksDir    string
	org         string
	description string
	authors     []string
	noSolution  bool
	force       bool
	stdout      io.Writer
	stderr      io.Writer
}

func evalTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage Harbor eval tasks",
	}

	cmd.AddCommand(evalTaskNewCmd())

	return cmd
}

func evalTaskNewCmd() *cobra.Command {
	opts := evalTaskNewOptions{
		tasksDir: harbor.DefaultEvalDatasetPath,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "new <org/name>",
		Short: "Create a Harbor eval task for Runme",
		Long: `Create a Harbor eval task scaffold for Runme.

When --tasks-dir is omitted, runme eval task new writes under ./` + harbor.DefaultEvalDatasetPath + `.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			return runEvalTaskNew(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.tasksDir, "tasks-dir", harbor.DefaultEvalDatasetPath, "Eval tasks directory")
	flags.StringVar(&opts.org, "org", "", "Organization name for bare task names")
	flags.StringVar(&opts.description, "description", "", "Task description")
	flags.StringArrayVar(&opts.authors, "author", nil, "Author in 'Name <email>' or 'Name' format; can be repeated")
	flags.BoolVar(&opts.noSolution, "no-solution", false, "Do not include solution template")
	flags.BoolVar(&opts.force, "force", false, "Overwrite scaffold-owned files in an existing task directory")

	return cmd
}

func runEvalTaskNew(opts evalTaskNewOptions, args []string) error {
	err := newEvalTaskNewer(harbor.EvalTaskNewOptions{
		TasksDir:    opts.tasksDir,
		Org:         opts.org,
		Description: opts.description,
		Authors:     opts.authors,
		NoSolution:  opts.noSolution,
		Force:       opts.force,
		Stdout:      opts.stdout,
		Stderr:      opts.stderr,
	}).Run(args)
	return asExitCodeError(err)
}
