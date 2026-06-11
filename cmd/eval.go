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

const defaultHarborJobsDir = ".runme/harbor/jobs"

var errRunmeHarborMissing = errors.New("runme-harbor missing")

type evalOptions struct {
	agent       string
	task        string
	jobsDir     string
	yes         bool
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
		jobsDir:    defaultHarborJobsDir,
		commandRun: runExternalCommand,
		lookPath:   exec.LookPath,
		executable: os.Executable,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		preflight:  true,
	}

	cmd := &cobra.Command{
		Use:   "eval <path> [flags] [-- harbor-flags...]",
		Short: "Run Harbor evals against Runme",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stdout = cmd.OutOrStdout()
			opts.stderr = cmd.ErrOrStderr()
			return runEval(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.agent, "agent", "oracle", "Harbor agent to use: oracle, codex, or claude")
	flags.StringVar(&opts.task, "task", "", "Harbor task name to include")
	flags.StringVar(&opts.jobsDir, "jobs-dir", defaultHarborJobsDir, "Harbor jobs directory")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Confirm Harbor prompts")
	flags.StringVar(&opts.runmeBin, "runme-bin", "", "Runme binary used by the Harbor environment")
	flags.StringArrayVar(&opts.runmeArgs, "runme-arg", nil, "Additional Runme argument used by the Harbor environment")
	flags.StringVar(&opts.runmeHarbor, "runme-harbor-bin", "", "runme-harbor executable")
	flags.BoolVar(&opts.debug, "debug", false, "Print delegated commands")

	return cmd
}

func runEval(opts evalOptions, args []string) error {
	if err := validateEvalAgent(opts.agent); err != nil {
		return err
	}

	taskPath, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	if _, err := os.Stat(taskPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("task path does not exist: %s", args[0])
		}
		return err
	}

	runmeHarbor, err := resolveRunmeHarbor(opts)
	if err != nil {
		if errors.Is(err, errRunmeHarborMissing) {
			return fmt.Errorf("`runme eval` requires the optional Python package `runme-harbor`.\n\nInstall it with:\n  uv tool install runme-harbor\n\nThen retry:\n  runme eval <path>")
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

	delegatedArgs := buildRunmeHarborArgs(taskPath, opts, args[1:])
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

func validateEvalAgent(agent string) error {
	switch agent {
	case "oracle", "codex", "claude":
		return nil
	default:
		return fmt.Errorf("invalid --agent %q: expected oracle, codex, or claude", agent)
	}
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

func buildRunmeHarborArgs(path string, opts evalOptions, passthrough []string) []string {
	args := []string{
		"run",
		path,
		"--agent", opts.agent,
		"--jobs-dir", opts.jobsDir,
	}
	if opts.task != "" {
		args = append(args, "--task", opts.task)
	}
	if opts.yes {
		args = append(args, "-y")
	}
	if opts.debug {
		args = append(args, "--debug")
	}
	if len(passthrough) > 0 {
		args = append(args, "--")
		args = append(args, passthrough...)
	}
	return args
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
