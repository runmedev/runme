package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultEvalDatasetPath = "evals/tasks"
	defaultEvalJobsDir     = ".runme/evals/jobs"
)

var errRunmeHarborMissing = errors.New("runme-harbor missing")

type evalOptions struct {
	agent       string
	taskDir     string
	jobsDir     string
	yes         bool
	model       string
	runmeBin    string
	runmeArgs   []string
	runmeHarbor string
	debug       bool
	commandRun  commandRunFunc
	lookPath    func(string) (string, error)
	executable  func() (string, error)
	stdout      io.Writer
	stderr      io.Writer
	extraEnv    []string
	preflight   bool
}

type commandRunFunc func(name string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) error

func evalCmd() *cobra.Command {
	opts := evalOptions{
		agent:      "oracle",
		jobsDir:    defaultEvalJobsDir,
		commandRun: runExternalCommand,
		lookPath:   exec.LookPath,
		executable: os.Executable,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		preflight:  true,
	}

	cmd := &cobra.Command{
		Use:   "eval [dataset-path] [flags] [-- harbor-flags...]",
		Short: "Run Harbor eval tasks with Runme",
		Long: fmt.Sprintf(`Run Harbor eval tasks with Runme.

When dataset-path is omitted, runme eval uses ./%s.`, defaultEvalDatasetPath),
		Args: validateEvalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			return runEval(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.agent, "agent", "oracle", "Harbor agent to use")
	flags.StringVar(&opts.taskDir, "task-dir", "", "Task directory name to include from the Harbor dataset")
	flags.StringVar(&opts.jobsDir, "jobs-dir", defaultEvalJobsDir, "Eval jobs directory")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Confirm Harbor prompts")
	flags.StringVar(&opts.model, "model", "", "Harbor agent model")
	flags.StringVar(&opts.runmeBin, "runme-bin", "", "Runme binary used by the Harbor environment")
	flags.StringArrayVar(&opts.runmeArgs, "runme-arg", nil, "Additional Runme argument used by the Harbor environment")
	flags.StringVar(&opts.runmeHarbor, "runme-harbor-bin", "", "runme-harbor executable")
	flags.BoolVar(&opts.debug, "debug", false, "Print delegated commands")

	return cmd
}

func runEval(opts evalOptions, args []string) error {
	datasetArg, passthrough := splitEvalDatasetArg(args)
	datasetPath, err := filepath.Abs(datasetArg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(datasetPath); err != nil {
		if os.IsNotExist(err) {
			if datasetArg == defaultEvalDatasetPath {
				return fmt.Errorf("dataset path does not exist: %s; create it or pass a dataset path explicitly", datasetArg)
			}
			return fmt.Errorf("dataset path does not exist: %s", datasetArg)
		}
		return err
	}

	if opts.model != "" && containsModelFlag(passthrough) {
		return fmt.Errorf("--model cannot be used together with passthrough --model; use only runme eval --model")
	}

	runmeHarbor, err := resolveRunmeHarbor(opts)
	if err != nil {
		if errors.Is(err, errRunmeHarborMissing) {
			return fmt.Errorf("`runme eval` requires the optional Python package `runme-harbor`.\n\nInstall it with:\n  uv tool install runme-harbor\n\nThen retry:\n  runme eval\n\nOr pass a dataset path explicitly:\n  runme eval <dataset-path>")
		}
		return err
	}

	runmeBin, err := resolveRunmeBin(opts)
	if err != nil {
		return err
	}

	env := os.Environ()
	env = setEnv(env, "RUNME_BIN", runmeBin)
	if len(opts.runmeArgs) > 0 {
		env = setEnv(env, "RUNME_ARGS", joinShellArgs(opts.runmeArgs))
	}
	env = append(env, opts.extraEnv...)

	if opts.preflight {
		request := strings.NewReader("{\"id\":\"preflight\",\"preflight\":{}}\n")
		if err := opts.commandRun(runmeBin, []string{"harbor", "stdio"}, env, request, io.Discard, opts.stderr); err != nil {
			return fmt.Errorf("runme harbor stdio preflight failed: %w", err)
		}
	}

	delegatedArgs := buildRunmeHarborArgs(datasetPath, opts, passthrough)
	if opts.debug {
		_, _ = fmt.Fprintf(opts.stderr, "%s\n", shellCommandString(append([]string{runmeHarbor}, delegatedArgs...)))
	}

	err = opts.commandRun(runmeHarbor, delegatedArgs, env, os.Stdin, opts.stdout, opts.stderr)
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

func splitEvalDatasetArg(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return defaultEvalDatasetPath, args
	}
	return args[0], args[1:]
}

func resolveRunmeHarbor(opts evalOptions) (string, error) {
	candidates := []string{}
	if value := os.Getenv("RUNME_HARBOR_BIN"); value != "" {
		candidates = append(candidates, value)
	}
	if opts.runmeHarbor != "" {
		candidates = append(candidates, opts.runmeHarbor)
	}
	candidates = append(candidates, "runme-harbor")

	for _, candidate := range candidates {
		resolved, err := resolveExecutable(candidate, opts.lookPath)
		if err == nil {
			return resolved, nil
		}
		if candidate != "runme-harbor" {
			return "", fmt.Errorf("runme-harbor executable %q not found: %w", candidate, err)
		}
	}
	return "", errRunmeHarborMissing
}

func resolveRunmeBin(opts evalOptions) (string, error) {
	if opts.runmeBin != "" {
		return opts.runmeBin, nil
	}
	if opts.executable != nil {
		if current, err := opts.executable(); err == nil && current != "" {
			return current, nil
		}
	}
	return "runme", nil
}

func resolveExecutable(name string, lookPath func(string) (string, error)) (string, error) {
	if strings.ContainsRune(name, rune(os.PathSeparator)) {
		info, err := os.Stat(name)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory", name)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
			return "", fmt.Errorf("%s is not executable", name)
		}
		return name, nil
	}
	return lookPath(name)
}

func buildRunmeHarborArgs(datasetPath string, opts evalOptions, passthrough []string) []string {
	args := []string{
		"run",
		datasetPath,
		"--agent", opts.agent,
		"--jobs-dir", opts.jobsDir,
	}
	if opts.taskDir != "" {
		args = append(args, "--task-dir", opts.taskDir)
	}
	if opts.yes {
		args = append(args, "-y")
	}
	if opts.debug {
		args = append(args, "--debug")
	}
	delegatedPassthrough := append([]string(nil), passthrough...)
	if opts.model != "" {
		delegatedPassthrough = append(delegatedPassthrough, "--model", opts.model)
	}
	if len(delegatedPassthrough) > 0 {
		args = append(args, "--")
		args = append(args, delegatedPassthrough...)
	}
	return args
}

func containsModelFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--model" || strings.HasPrefix(arg, "--model=") {
			return true
		}
	}
	return false
}

func runExternalCommand(name string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	return cmd.Run()
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func joinShellArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellCommandString(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return (r < 'A' || r > 'Z') &&
			(r < 'a' || r > 'z') &&
			(r < '0' || r > '9') &&
			!strings.ContainsRune("@%_+=:,./-", r)
	}) == -1 {
		return value
	}
	return strconv.Quote(value)
}
