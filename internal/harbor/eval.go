package harbor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/mattn/go-isatty"

	"github.com/runmedev/runme/v3/project"
)

const (
	DefaultEvalDatasetPath = "evals/tasks"
	DefaultEvalJobsDir     = ".runme/evals/jobs"
)

var ErrRunmeHarborMissing = errors.New("runme-harbor missing")

type EvalOptions struct {
	Agent           string
	TaskDir         string
	JobsDir         string
	Yes             bool
	Model           string
	Env             string
	RunmeBin        string
	RunmeArgs       []string
	RunmeHarborBin  string
	Debug           bool
	JobsDirExplicit bool
	CommandRun      CommandRunFunc
	LookPath        func(string) (string, error)
	Executable      func() (string, error)
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	ExtraEnv        []string
	Preflight       bool
}

type CommandRunFunc func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error

type EvalRunner struct {
	opts EvalOptions
}

func NewEvalRunner(opts EvalOptions) *EvalRunner {
	if opts.Agent == "" {
		opts.Agent = "oracle"
	}
	if opts.JobsDir == "" {
		opts.JobsDir = DefaultEvalJobsDir
	}
	if opts.CommandRun == nil {
		opts.CommandRun = runExternalCommand
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}
	if opts.Executable == nil {
		opts.Executable = os.Executable
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return &EvalRunner{opts: opts}
}

func (r *EvalRunner) Run(args []string) error {
	opts := r.opts
	datasetPath, passthrough, defaultDataset, evalBaseDir, err := resolveEvalPaths(&opts, args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(datasetPath); err != nil {
		if os.IsNotExist(err) {
			if defaultDataset {
				return fmt.Errorf("dataset path does not exist: %s; create it or pass a dataset path explicitly", DefaultEvalDatasetPath)
			}
			return fmt.Errorf("dataset path does not exist: %s", datasetPath)
		}
		return err
	}

	if opts.Model != "" && containsModelFlag(passthrough) {
		return fmt.Errorf("--model cannot be used together with passthrough --model; use only runme eval --model")
	}
	if containsEnvironmentFlag(passthrough) {
		return fmt.Errorf("use runme eval --env instead of passing Harbor environment flags after --")
	}

	runmeHarbor, err := resolveRunmeHarbor(opts)
	if err != nil {
		return err
	}

	runmeBin, err := resolveRunmeBin(opts)
	if err != nil {
		return err
	}

	env := os.Environ()
	env = setEnv(env, "RUNME_BIN", runmeBin)
	if len(opts.RunmeArgs) > 0 {
		env = setEnv(env, "RUNME_ARGS", joinShellArgs(opts.RunmeArgs))
	}
	env = append(env, opts.ExtraEnv...)

	if usesHarborDockerEnvironment(opts.Env) {
		stager, err := NewDockerWorkdirStager(DockerWorkdirStagerOptions{
			Stderr: opts.Stderr,
		})
		if err != nil {
			return err
		}
		if err := stager.StageDataset(datasetPath); err != nil {
			return err
		}
	}

	if opts.Preflight && usesRunmeEnvironment(opts.Env) {
		request := strings.NewReader("{\"id\":\"preflight\",\"preflight\":{}}\n")
		if err := opts.CommandRun(runmeBin, []string{"harbor", "stdio"}, "", env, request, io.Discard, opts.Stderr); err != nil {
			return fmt.Errorf("runme harbor stdio preflight failed: %w", err)
		}
	}

	delegatedArgs := buildRunmeHarborArgs(datasetPath, opts, passthrough)
	if opts.Debug {
		_, _ = fmt.Fprintf(opts.Stderr, "%s\n", shellCommandString(append([]string{runmeHarbor}, delegatedArgs...)))
	}

	harborStdout := NewResultPathWriter(opts.Stdout)
	err = opts.CommandRun(runmeHarbor, delegatedArgs, evalBaseDir, env, opts.Stdin, harborStdout, opts.Stderr)
	if jobDir := JobDirFromResultPath(harborStdout.ResultPath(), evalBaseDir); jobDir != "" {
		PrintExceptionDetails(opts.Stdout, jobDir)
	}
	return err
}

func resolveEvalPaths(opts *EvalOptions, args []string) (string, []string, bool, string, error) {
	datasetArg, defaultDataset, passthrough := splitEvalDatasetArg(args)
	baseDir, err := evalDefaultBaseDir()
	if err != nil {
		return "", nil, false, "", err
	}
	if !opts.JobsDirExplicit {
		opts.JobsDir = filepath.Join(baseDir, DefaultEvalJobsDir)
	}
	if defaultDataset {
		return filepath.Join(baseDir, DefaultEvalDatasetPath), passthrough, true, baseDir, nil
	}

	datasetPath, err := filepath.Abs(datasetArg)
	if err != nil {
		return "", nil, false, "", err
	}
	return datasetPath, passthrough, false, baseDir, nil
}

func evalDefaultBaseDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return defaultEvalBaseDir(cwd), nil
}

func splitEvalDatasetArg(args []string) (string, bool, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return DefaultEvalDatasetPath, true, args
	}
	return args[0], false, args[1:]
}

func defaultEvalBaseDir(cwd string) string {
	proj, err := project.NewDirProject(
		cwd,
		project.WithFindRepoUpward(),
		project.WithAllowUnsupportedGitExtensions(true),
		project.WithRespectGitignore(false),
	)
	if err != nil {
		return cwd
	}
	return proj.Root()
}

func resolveRunmeHarbor(opts EvalOptions) (string, error) {
	candidates := []string{}
	if value := os.Getenv("RUNME_HARBOR_BIN"); value != "" {
		candidates = append(candidates, value)
	}
	if opts.RunmeHarborBin != "" {
		candidates = append(candidates, opts.RunmeHarborBin)
	}
	candidates = append(candidates, "runme-harbor")

	for _, candidate := range candidates {
		resolved, err := resolveExecutable(candidate, opts.LookPath)
		if err == nil {
			return resolved, nil
		}
		if candidate != "runme-harbor" {
			return "", fmt.Errorf("runme-harbor executable %q not found: %w", candidate, err)
		}
	}
	return "", ErrRunmeHarborMissing
}

func resolveRunmeBin(opts EvalOptions) (string, error) {
	if opts.RunmeBin != "" {
		return opts.RunmeBin, nil
	}
	if opts.Executable != nil {
		if current, err := opts.Executable(); err == nil && current != "" {
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

func buildRunmeHarborArgs(datasetPath string, opts EvalOptions, passthrough []string) []string {
	args := []string{
		"run",
		datasetPath,
		"--agent", opts.Agent,
		"--jobs-dir", opts.JobsDir,
	}
	if opts.TaskDir != "" {
		args = append(args, "--task-dir", opts.TaskDir)
	}
	if opts.Yes {
		args = append(args, "-y")
	}
	if opts.Debug {
		args = append(args, "--debug")
	}
	if opts.Env != "" {
		args = append(args, "--env", opts.Env)
	}
	delegatedPassthrough := append([]string(nil), passthrough...)
	if opts.Model != "" {
		delegatedPassthrough = append(delegatedPassthrough, "--model", opts.Model)
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

func containsEnvironmentFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--env" || arg == "-e" || arg == "--environment-import-path" {
			return true
		}
		if strings.HasPrefix(arg, "--env=") || strings.HasPrefix(arg, "-e") && len(arg) > 2 {
			return true
		}
		if strings.HasPrefix(arg, "--environment-import-path=") {
			return true
		}
	}
	return false
}

func usesRunmeEnvironment(env string) bool {
	return env == "" || env == "runme"
}

func usesHarborDockerEnvironment(env string) bool {
	return env == "docker"
}

func runExternalCommand(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workingDir
	cmd.Env = env
	if provider, ok := stdout.(interface{ StdoutFile() *os.File }); ok {
		if file := provider.StdoutFile(); file != nil && isTerminal(file.Fd()) {
			return runExternalCommandWithPtyStdout(cmd, stdin, stdout, stderr, file)
		}
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	return cmd.Run()
}

func runExternalCommandWithPtyStdout(cmd *exec.Cmd, stdin io.Reader, stdout, stderr io.Writer, stdoutFile *os.File) error {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return err
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()
	_ = pty.InheritSize(stdoutFile, ptmx)

	cmd.Stdout = tty
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = tty.Close()

	copyDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdout, ptmx)
		copyDone <- err
	}()

	waitErr := cmd.Wait()
	_ = ptmx.Close()
	copyErr := <-copyDone
	if waitErr != nil {
		return waitErr
	}
	if copyErr != nil && !errors.Is(copyErr, os.ErrClosed) {
		if errors.Is(copyErr, syscall.EIO) {
			return nil
		}
		return copyErr
	}
	return nil
}

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
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
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
