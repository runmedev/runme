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
	Ask             bool
	AgentKwargs     []string
	AgentEnv        []string
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
	paths, err := resolveEvalPaths(&opts, args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(paths.datasetPath); err != nil {
		if os.IsNotExist(err) {
			if paths.defaultDataset {
				return fmt.Errorf("dataset path does not exist: %s; create it or pass a dataset path explicitly", DefaultEvalDatasetPath)
			}
			return fmt.Errorf("dataset path does not exist: %s", paths.datasetPath)
		}
		return err
	}

	if opts.Model != "" && containsModelFlag(paths.passthrough) {
		return fmt.Errorf("--model cannot be used together with passthrough --model; use only runme eval --model")
	}
	if len(opts.AgentKwargs) > 0 && containsFlagAlias(paths.passthrough, "--agent-kwarg", "--ak") {
		return fmt.Errorf("--agent-kwarg cannot be used together with passthrough --agent-kwarg/--ak; use only runme eval --agent-kwarg")
	}
	if len(opts.AgentEnv) > 0 && containsFlagAlias(paths.passthrough, "--agent-env", "--ae") {
		return fmt.Errorf("--agent-env cannot be used together with passthrough --agent-env/--ae; use only runme eval --agent-env")
	}
	if containsEnvironmentFlag(paths.passthrough) {
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
		if err := stager.StageDataset(paths.datasetPath); err != nil {
			return err
		}
	}

	if opts.Preflight && usesRunmeEnvironment(opts.Env) {
		request := strings.NewReader("{\"id\":\"preflight\",\"preflight\":{}}\n")
		if err := opts.CommandRun(runmeBin, []string{"harbor", "stdio"}, "", env, request, io.Discard, opts.Stderr); err != nil {
			return fmt.Errorf("runme harbor stdio preflight failed: %w", err)
		}
	}

	delegatedArgs := buildRunmeHarborArgs(paths.delegateDatasetPath, opts, paths.delegateJobsDir, paths.passthrough)
	if opts.Debug {
		_, _ = fmt.Fprintf(opts.Stderr, "%s\n", shellCommandString(append([]string{runmeHarbor}, delegatedArgs...)))
	}

	harborStdout := NewResultPathWriter(opts.Stdout)
	err = opts.CommandRun(runmeHarbor, delegatedArgs, paths.executionCwd, env, opts.Stdin, harborStdout, opts.Stderr)
	if jobDir := JobDirFromResultPath(harborStdout.ResultPath(), paths.executionCwd); jobDir != "" {
		PrintExceptionDetails(opts.Stdout, jobDir)
	}
	return err
}

type evalPaths struct {
	datasetPath         string
	delegateDatasetPath string
	delegateJobsDir     string
	passthrough         []string
	defaultDataset      bool
	invocationCwd       string
	executionCwd        string
}

type evalPathResolver struct {
	opts          *EvalOptions
	invocationCwd string
	baseDir       string
}

func resolveEvalPaths(opts *EvalOptions, args []string) (evalPaths, error) {
	invocationCwd, err := os.Getwd()
	if err != nil {
		return evalPaths{}, err
	}

	resolver := evalPathResolver{
		opts:          opts,
		invocationCwd: cleanExistingPath(invocationCwd),
	}
	resolver.baseDir = defaultEvalBaseDir(resolver.invocationCwd)
	return resolver.resolve(args), nil
}

func (r evalPathResolver) resolve(args []string) evalPaths {
	datasetArg, defaultDataset, passthrough := splitEvalDatasetArg(args)

	var datasetPath string
	if defaultDataset {
		datasetPath = filepath.Join(r.baseDir, DefaultEvalDatasetPath)
	} else {
		datasetPath = r.inputPath(datasetArg)
	}

	executionCwd := r.executionCwd(datasetPath)
	delegateDatasetPath := r.delegatePath(datasetPath, executionCwd)

	if !r.opts.JobsDirExplicit {
		r.opts.JobsDir = filepath.Join(r.baseDir, DefaultEvalJobsDir)
	} else {
		r.opts.JobsDir = r.inputPath(r.opts.JobsDir)
	}
	delegateJobsDir := r.delegateJobsPath(r.opts.JobsDir, executionCwd)

	return evalPaths{
		datasetPath:         datasetPath,
		delegateDatasetPath: delegateDatasetPath,
		delegateJobsDir:     delegateJobsDir,
		passthrough:         passthrough,
		defaultDataset:      defaultDataset,
		invocationCwd:       r.invocationCwd,
		executionCwd:        executionCwd,
	}
}

func (r evalPathResolver) inputPath(path string) string {
	if filepath.IsAbs(path) {
		return cleanExistingPath(path)
	}

	return filepath.Clean(filepath.Join(r.invocationCwd, path))
}

func (r evalPathResolver) executionCwd(datasetPath string) string {
	if relativePathUnder(r.baseDir, datasetPath) != "" {
		return r.baseDir
	}
	return r.invocationCwd
}

func (r evalPathResolver) delegatePath(path, executionCwd string) string {
	delegated := relativePathUnder(executionCwd, path)
	if delegated == "" {
		return filepath.Clean(path)
	}
	return delegated
}

func (r evalPathResolver) delegateJobsPath(path, executionCwd string) string {
	if r.invocationCwd == executionCwd {
		return r.delegatePath(path, executionCwd)
	}
	return filepath.Clean(path)
}

func cleanExistingPath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved)
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		return cleanPathFromExistingParent(abs)
	}
	return filepath.Clean(path)
}

func cleanPathFromExistingParent(path string) string {
	cleaned := filepath.Clean(path)
	current := cleaned
	missing := []string{}

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return cleaned
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func relativePathUnder(base, path string) string {
	relative, err := filepath.Rel(base, path)
	if err != nil {
		return ""
	}
	if relative == "." {
		return "."
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return ""
	}
	return filepath.Clean(relative)
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

func buildRunmeHarborArgs(datasetPath string, opts EvalOptions, jobsDir string, passthrough []string) []string {
	args := []string{
		"run",
		datasetPath,
		"--agent", opts.Agent,
		"--jobs-dir", jobsDir,
	}
	if opts.TaskDir != "" {
		args = append(args, "--task-dir", opts.TaskDir)
	}
	if !opts.Ask {
		args = append(args, "-y")
	}
	if opts.Debug {
		args = append(args, "--debug")
	}
	if opts.Env != "" {
		args = append(args, "--env", opts.Env)
	}
	delegatedPassthrough := append([]string(nil), passthrough...)
	for _, kwarg := range opts.AgentKwargs {
		delegatedPassthrough = append(delegatedPassthrough, "--agent-kwarg", kwarg)
	}
	for _, env := range opts.AgentEnv {
		delegatedPassthrough = append(delegatedPassthrough, "--agent-env", env)
	}
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

func containsFlagAlias(args []string, long, alias string) bool {
	for _, arg := range args {
		if arg == long || arg == alias {
			return true
		}
		if strings.HasPrefix(arg, long+"=") || strings.HasPrefix(arg, alias+"=") {
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

	// Only stdout is bridged through a PTY so Harbor's Rich progress output
	// keeps terminal rendering while eval itself stays non-interactive.
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
	copyErr := <-copyDone
	if waitErr != nil {
		return waitErr
	}
	if copyErr != nil {
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
