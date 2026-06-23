package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/runmedev/runme/v3/internal/harbor"
)

type evalOptions struct {
	agent           string
	taskDir         string
	jobsDir         string
	ask             bool
	agentKwargs     []string
	agentEnv        []string
	model           string
	env             string
	runmeBin        string
	runmeArgs       []string
	runmeHarbor     string
	debug           bool
	jobsDirExplicit bool
	stdout          io.Writer
	stderr          io.Writer
}

type evalRunner interface {
	Run(args []string) error
}

var newEvalRunner = func(opts harbor.EvalOptions) evalRunner {
	return harbor.NewEvalRunner(opts)
}

func evalCmd() *cobra.Command {
	opts := evalOptions{
		agent:   "oracle",
		jobsDir: harbor.DefaultEvalJobsDir,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}

	cmd := &cobra.Command{
		Use:   "eval [dataset-path] [flags] [-- harbor-flags...]",
		Short: "Run Harbor eval tasks with Runme",
		Long: fmt.Sprintf(`Run Harbor eval tasks with Runme.

When dataset-path is omitted, runme eval uses ./%s.`, harbor.DefaultEvalDatasetPath),
		Args: validateEvalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			opts.jobsDirExplicit = cmd.Flags().Changed("jobs-dir")
			return runEval(opts, args)
		},
	}

	flags := cmd.Flags()
	previousNormalize := flags.GetNormalizeFunc()
	flags.SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "ak":
			name = "agent-kwarg"
		case "ae":
			name = "agent-env"
		}
		return previousNormalize(f, name)
	})
	flags.StringVar(&opts.agent, "agent", "oracle", "Harbor agent to use")
	flags.StringVar(&opts.taskDir, "task-dir", "", "Task directory name to include from the Harbor dataset")
	flags.StringVar(&opts.jobsDir, "jobs-dir", harbor.DefaultEvalJobsDir, "Eval jobs directory")
	flags.BoolVar(&opts.ask, "ask", false, "Do not auto-accept Harbor confirmation prompts")
	flags.StringArrayVar(&opts.agentKwargs, "agent-kwarg", nil, "Harbor agent kwarg; can be repeated; alias: --ak")
	flags.StringArrayVar(&opts.agentEnv, "agent-env", nil, "Environment variable to pass to the agent in KEY=VALUE format; can be repeated; alias: --ae")
	flags.StringVar(&opts.model, "model", "", "Harbor agent model")
	flags.StringVarP(&opts.env, "env", "e", "", `Harbor environment to use. Defaults to "runme"`)
	flags.StringVar(&opts.runmeBin, "runme-bin", "", "Runme binary used by the Harbor environment")
	flags.StringArrayVar(&opts.runmeArgs, "runme-arg", nil, "Additional Runme argument used by the Harbor environment")
	flags.StringVar(&opts.runmeHarbor, "runme-harbor-bin", "", "runme-harbor executable")
	flags.BoolVar(&opts.debug, "debug", false, "Print delegated commands")

	return cmd
}

func runEval(opts evalOptions, args []string) error {
	err := newEvalRunner(harbor.EvalOptions{
		Agent:           opts.agent,
		TaskDir:         opts.taskDir,
		JobsDir:         opts.jobsDir,
		Ask:             opts.ask,
		AgentKwargs:     opts.agentKwargs,
		AgentEnv:        opts.agentEnv,
		Model:           opts.model,
		Env:             opts.env,
		RunmeBin:        opts.runmeBin,
		RunmeArgs:       opts.runmeArgs,
		RunmeHarborBin:  opts.runmeHarbor,
		Debug:           opts.debug,
		JobsDirExplicit: opts.jobsDirExplicit,
		Stdout:          opts.stdout,
		Stderr:          opts.stderr,
		Preflight:       true,
	}).Run(args)
	if err == nil {
		return nil
	}
	if errors.Is(err, harbor.ErrRunmeHarborMissing) {
		return fmt.Errorf("`runme eval` requires the optional Python package `runme-harbor`.\n\nInstall it with:\n  uv tool install runme-harbor\n\nThen retry:\n  runme eval\n\nOr pass a dataset path explicitly:\n  runme eval <dataset-path>")
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

func validateEvalArgs(_ *cobra.Command, args []string) error {
	if len(args) < 2 {
		return nil
	}
	if strings.HasPrefix(args[0], "-") {
		return nil
	}
	if strings.HasPrefix(args[1], "-") {
		return nil
	}
	return fmt.Errorf("accepts at most 1 dataset path, received %d", len(args))
}
