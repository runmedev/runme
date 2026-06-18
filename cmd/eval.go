package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/spf13/cobra"

	"github.com/runmedev/runme/v3/internal/ansi"
	"github.com/runmedev/runme/v3/internal/harbor"
	"github.com/runmedev/runme/v3/project"
)

const (
	defaultEvalDatasetPath = "evals/tasks"
	defaultEvalJobsDir     = ".runme/evals/jobs"
	maxHarborResultLineLen = 64 * 1024
)

var errRunmeHarborMissing = errors.New("runme-harbor missing")

type evalOptions struct {
	agent           string
	taskDir         string
	jobsDir         string
	yes             bool
	model           string
	env             string
	runmeBin        string
	runmeArgs       []string
	runmeHarbor     string
	debug           bool
	jobsDirExplicit bool
	evalBaseDir     string
	commandRun      commandRunFunc
	lookPath        func(string) (string, error)
	executable      func() (string, error)
	stdout          io.Writer
	stderr          io.Writer
	extraEnv        []string
	preflight       bool
}

type commandRunFunc func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) error

type harborResultPathWriter struct {
	dst         io.Writer
	line        []byte
	lineTooLong bool
	resultPath  string
}

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
			opts.jobsDirExplicit = cmd.Flags().Changed("jobs-dir")
			return runEval(opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.agent, "agent", "oracle", "Harbor agent to use")
	flags.StringVar(&opts.taskDir, "task-dir", "", "Task directory name to include from the Harbor dataset")
	flags.StringVar(&opts.jobsDir, "jobs-dir", defaultEvalJobsDir, "Eval jobs directory")
	flags.BoolVarP(&opts.yes, "yes", "y", false, "Confirm Harbor prompts")
	flags.StringVar(&opts.model, "model", "", "Harbor agent model")
	flags.StringVarP(&opts.env, "env", "e", "", `Harbor environment to use. Defaults to "runme"`)
	flags.StringVar(&opts.runmeBin, "runme-bin", "", "Runme binary used by the Harbor environment")
	flags.StringArrayVar(&opts.runmeArgs, "runme-arg", nil, "Additional Runme argument used by the Harbor environment")
	flags.StringVar(&opts.runmeHarbor, "runme-harbor-bin", "", "runme-harbor executable")
	flags.BoolVar(&opts.debug, "debug", false, "Print delegated commands")

	return cmd
}

func runEval(opts evalOptions, args []string) error {
	datasetPath, passthrough, defaultDataset, err := resolveEvalPaths(&opts, args)
	if err != nil {
		return err
	}
	if _, err := os.Stat(datasetPath); err != nil {
		if os.IsNotExist(err) {
			if defaultDataset {
				return fmt.Errorf("dataset path does not exist: %s; create it or pass a dataset path explicitly", defaultEvalDatasetPath)
			}
			return fmt.Errorf("dataset path does not exist: %s", datasetPath)
		}
		return err
	}

	if opts.model != "" && containsModelFlag(passthrough) {
		return fmt.Errorf("--model cannot be used together with passthrough --model; use only runme eval --model")
	}
	if containsEnvironmentFlag(passthrough) {
		return fmt.Errorf("use runme eval --env instead of passing Harbor environment flags after --")
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

	if usesHarborDockerEnvironment(opts.env) {
		stager, err := harbor.NewDockerWorkdirStager(harbor.DockerWorkdirStagerOptions{
			Stderr: opts.stderr,
		})
		if err != nil {
			return err
		}
		if err := stager.StageDataset(datasetPath); err != nil {
			return err
		}
	}

	if opts.preflight && usesRunmeEnvironment(opts.env) {
		request := strings.NewReader("{\"id\":\"preflight\",\"preflight\":{}}\n")
		if err := opts.commandRun(runmeBin, []string{"harbor", "stdio"}, "", env, request, io.Discard, opts.stderr); err != nil {
			return fmt.Errorf("runme harbor stdio preflight failed: %w", err)
		}
	}

	delegatedArgs := buildRunmeHarborArgs(datasetPath, opts, passthrough)
	if opts.debug {
		_, _ = fmt.Fprintf(opts.stderr, "%s\n", shellCommandString(append([]string{runmeHarbor}, delegatedArgs...)))
	}

	harborStdout := &harborResultPathWriter{dst: opts.stdout}
	err = opts.commandRun(runmeHarbor, delegatedArgs, opts.evalBaseDir, env, os.Stdin, harborStdout, opts.stderr)
	if jobDir := evalJobDirFromResultPath(harborStdout.ResultPath(), opts.evalBaseDir); jobDir != "" {
		printEvalExceptionDetails(opts.stdout, jobDir)
	}
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

func resolveEvalPaths(opts *evalOptions, args []string) (string, []string, bool, error) {
	datasetArg, defaultDataset, passthrough := splitEvalDatasetArg(args)
	if defaultDataset {
		baseDir, err := evalDefaultBaseDir()
		if err != nil {
			return "", nil, false, err
		}
		if !opts.jobsDirExplicit {
			opts.jobsDir = filepath.Join(baseDir, defaultEvalJobsDir)
		}
		opts.evalBaseDir = baseDir
		return filepath.Join(baseDir, defaultEvalDatasetPath), passthrough, true, nil
	}

	datasetPath, err := filepath.Abs(datasetArg)
	if err != nil {
		return "", nil, false, err
	}
	if !opts.jobsDirExplicit {
		baseDir, err := evalDefaultBaseDir()
		if err != nil {
			return "", nil, false, err
		}
		opts.jobsDir = filepath.Join(baseDir, defaultEvalJobsDir)
		opts.evalBaseDir = baseDir
	} else {
		baseDir, err := evalDefaultBaseDir()
		if err != nil {
			return "", nil, false, err
		}
		opts.evalBaseDir = baseDir
	}
	return datasetPath, passthrough, false, nil
}

func evalDefaultBaseDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return defaultEvalBaseDir(cwd), nil
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

func splitEvalDatasetArg(args []string) (string, bool, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return defaultEvalDatasetPath, true, args
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
	if opts.env != "" {
		args = append(args, "--env", opts.env)
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

func (w *harborResultPathWriter) Write(p []byte) (int, error) {
	// Forward Harbor output before inspecting it so progress and tables keep streaming.
	n, err := w.dst.Write(p)
	remaining := p[:n]
	for len(remaining) > 0 {
		index := bytes.IndexByte(remaining, '\n')
		if index < 0 {
			w.appendLine(remaining)
			break
		}
		w.appendLine(remaining[:index])
		if !w.lineTooLong {
			w.recordLine(w.line)
		}
		w.line = w.line[:0]
		w.lineTooLong = false
		remaining = remaining[index+1:]
	}
	return n, err
}

func (w *harborResultPathWriter) ResultPath() string {
	if len(w.line) > 0 && !w.lineTooLong {
		w.recordLine(w.line)
		w.line = nil
	}
	return w.resultPath
}

func (w *harborResultPathWriter) StdoutFile() *os.File {
	file, _ := w.dst.(*os.File)
	return file
}

func (w *harborResultPathWriter) appendLine(p []byte) {
	if w.lineTooLong {
		return
	}
	if len(w.line)+len(p) > maxHarborResultLineLen {
		w.line = w.line[:0]
		w.lineTooLong = true
		return
	}
	w.line = append(w.line, p...)
}

func (w *harborResultPathWriter) recordLine(line []byte) {
	const prefix = "Results written to "
	text := strings.TrimSpace(string(ansi.Strip(line)))
	index := strings.Index(text, prefix)
	if index == -1 {
		return
	}
	w.resultPath = strings.TrimSpace(text[index+len(prefix):])
}

func evalJobDirFromResultPath(resultPath, evalBaseDir string) string {
	if resultPath == "" {
		return ""
	}
	if !filepath.IsAbs(resultPath) && evalBaseDir != "" {
		resultPath = filepath.Join(evalBaseDir, resultPath)
	}
	return filepath.Dir(resultPath)
}

func printEvalExceptionDetails(w io.Writer, jobDir string) {
	paths := evalExceptionFiles(jobDir)
	if len(paths) == 0 {
		return
	}

	printed := false
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		detail := strings.TrimSpace(string(content))
		if detail == "" {
			continue
		}
		if !printed {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, ansi.Color("Harbor Exception Details", "red+b"))
			printed = true
		}
		_, _ = fmt.Fprintf(w, "\nFile: %s\n%s\n", evalExceptionDisplayPath(jobDir, path), detail)
	}
}

func evalExceptionDisplayPath(jobsDir, path string) string {
	relative, err := filepath.Rel(jobsDir, path)
	if err != nil || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || relative == ".." {
		return path
	}
	return relative
}

func evalExceptionFiles(jobDir string) []string {
	var paths []string
	_ = filepath.WalkDir(jobDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(path) != "exception.txt" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)
	return paths
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
