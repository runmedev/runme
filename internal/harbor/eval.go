package harbor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	DefaultEvalDatasetPath = "evals/tasks"
	DefaultEvalJobsDir     = ".runme/evals/jobs"
)

var ErrRunmeHarborMissing = errors.New("runme-harbor missing")

type EvalOptions struct {
	Agent              string
	TaskDir            string
	JobsDir            string
	Ask                bool
	AgentKwargs        []string
	AgentEnv           []string
	VerifierEnv        []string
	VerifierEnvDefault []string
	Model              string
	Env                string
	RunmeBin           string
	RunmeArgs          []string
	RunmeHarborBin     string
	Debug              bool
	JobsDirExplicit    bool
	CommandRun         CommandRunFunc
	LookPath           func(string) (string, error)
	Executable         func() (string, error)
	Stdin              io.Reader
	Stdout             io.Writer
	Stderr             io.Writer
	ExtraEnv           []string
	Preflight          bool
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

	runmeHarbor, err := resolveRunmeHarbor(opts)
	if err != nil {
		return err
	}
	bundledHarbor, err := resolveBundledHarborWithLookPath(runmeHarbor, opts.LookPath)
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
	env, err = applyEnvDefaults(env, opts.VerifierEnvDefault)
	if err != nil {
		return fmt.Errorf("apply verifier environment defaults: %w", err)
	}

	delegatedArgs, err := harborRunArgsBuilder{
		datasetPath: paths.delegateDatasetPath,
		jobsDir:     paths.delegateJobsDir,
		opts:        opts,
		passthrough: paths.passthrough,
	}.Build()
	if err != nil {
		return err
	}

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

	if opts.Debug {
		_, _ = fmt.Fprintf(opts.Stderr, "%s\n", shellCommandString(append([]string{bundledHarbor}, delegatedArgs...)))
	}

	harborStdout := NewResultPathWriter(opts.Stdout)
	err = opts.CommandRun(bundledHarbor, delegatedArgs, paths.executionCwd, env, opts.Stdin, harborStdout, opts.Stderr)
	if jobDir := JobDirFromResultPath(harborStdout.ResultPath(), paths.executionCwd); jobDir != "" {
		PrintExceptionDetails(opts.Stdout, jobDir)
	}
	r.syncMetadata(runmeHarbor, paths.delegateJobsDir, paths.executionCwd, env)
	return err
}

func applyEnvDefaults(env []string, defaults []string) ([]string, error) {
	for _, assignment := range defaults {
		key, value, ok := strings.Cut(assignment, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid environment assignment %q: expected KEY=VALUE", assignment)
		}
		prefix := key + "="
		configured := false
		for _, item := range env {
			if strings.HasPrefix(item, prefix) {
				configured = true
				break
			}
		}
		if !configured {
			env = append(env, prefix+value)
		}
	}
	return env, nil
}

func (r *EvalRunner) syncMetadata(runmeHarbor, jobsDir, workingDir string, env []string) {
	opts := r.opts
	if skipMetadataSync() {
		return
	}
	syncErr := opts.CommandRun(
		runmeHarbor,
		[]string{"sync-metadata", "--jobs-dir", jobsDir},
		workingDir,
		env,
		nil,
		io.Discard,
		opts.Stderr,
	)
	if syncErr != nil {
		_, _ = fmt.Fprintf(opts.Stderr, "warning: failed to sync Harbor job metadata: %v\n", syncErr)
	}
}
