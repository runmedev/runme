package harbor

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const (
	defaultEvalViewPort      = 8080
	defaultEvalViewPortLimit = 8180
)

type EvalViewOptions struct {
	JobsDir        string
	Port           int
	Open           bool
	RunmeHarborBin string
	Debug          bool
	CommandStart   CommandStartFunc
	BrowserOpen    BrowserOpenFunc
	DashboardReady DashboardReadyFunc
	LookPath       func(string) (string, error)
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
	ExtraEnv       []string
}

type CommandStartFunc func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (StartedCommand, error)

type StartedCommand interface {
	Wait() error
}

type BrowserOpenFunc func(url string) error

type DashboardReadyFunc func(urls []string) (string, error)

type EvalViewer struct {
	opts EvalViewOptions
}

type evalViewPaths struct {
	jobsDir         string
	delegateJobsDir string
	workingDir      string
}

func NewEvalViewer(opts EvalViewOptions) *EvalViewer {
	if opts.CommandStart == nil {
		opts.CommandStart = startExternalCommand
	}
	if opts.BrowserOpen == nil {
		opts.BrowserOpen = browser.OpenURL
	}
	if opts.DashboardReady == nil {
		opts.DashboardReady = waitForDashboard
	}
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
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
	return &EvalViewer{opts: opts}
}

func (r *EvalViewer) Run(args []string) error {
	opts := r.opts
	if len(args) > 0 && opts.JobsDir == "" {
		opts.JobsDir = args[0]
	}

	runmeHarbor, err := resolveRunmeHarbor(EvalOptions{
		RunmeHarborBin: opts.RunmeHarborBin,
		LookPath:       opts.LookPath,
	})
	if err != nil {
		return err
	}

	bundledHarbor, err := resolveBundledHarbor(runmeHarbor)
	if err != nil {
		return err
	}

	paths, err := resolveEvalViewPaths(opts.JobsDir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(paths.jobsDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("eval jobs directory does not exist: %s", paths.jobsDir)
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("eval jobs path is not a directory: %s", paths.jobsDir)
	}

	portArg := fmt.Sprintf("%d", opts.Port)
	candidatePorts := []int{opts.Port}
	if opts.Port == 0 {
		portArg = fmt.Sprintf("%d-%d", defaultEvalViewPort, defaultEvalViewPortLimit)
		candidatePorts = nil
		if opts.Open {
			candidatePorts, err = openLocalPorts(defaultEvalViewPort, defaultEvalViewPortLimit)
			if err != nil {
				return err
			}
		}
	}

	delegatedArgs := []string{
		"view",
		paths.delegateJobsDir,
		"--jobs",
		"--port",
		portArg,
	}
	if opts.Debug {
		_, _ = fmt.Fprintf(opts.Stderr, "%s\n", shellCommandString(append([]string{bundledHarbor}, delegatedArgs...)))
	}

	env := append(os.Environ(), opts.ExtraEnv...)
	started, err := opts.CommandStart(bundledHarbor, delegatedArgs, paths.workingDir, env, opts.Stdin, opts.Stdout, opts.Stderr)
	if err != nil {
		return err
	}

	if opts.Open {
		urls := dashboardURLs(candidatePorts)
		url, err := opts.DashboardReady(urls)
		if err != nil {
			_, _ = fmt.Fprintf(opts.Stderr, "warning: dashboard did not become ready at %s: %v\n", strings.Join(urls, ", "), err)
		} else if err := opts.BrowserOpen(url); err != nil {
			_, _ = fmt.Fprintf(opts.Stderr, "warning: failed to open dashboard at %s: %v\n", url, err)
		}
	}

	return started.Wait()
}

func resolveEvalViewPaths(jobsDir string) (evalViewPaths, error) {
	invocationCwd, err := os.Getwd()
	if err != nil {
		return evalViewPaths{}, err
	}
	invocationCwd = cleanExistingPath(invocationCwd)
	baseDir := defaultEvalBaseDir(invocationCwd)

	if jobsDir == "" {
		return evalViewPaths{
			jobsDir:         filepath.Join(baseDir, DefaultEvalJobsDir),
			delegateJobsDir: DefaultEvalJobsDir,
			workingDir:      baseDir,
		}, nil
	}

	resolver := evalPathResolver{invocationCwd: invocationCwd, baseDir: baseDir}
	resolved := resolver.inputPath(jobsDir)
	delegate := resolver.delegatePath(resolved, invocationCwd)
	return evalViewPaths{
		jobsDir:         resolved,
		delegateJobsDir: delegate,
		workingDir:      invocationCwd,
	}, nil
}

func resolveBundledHarbor(runmeHarbor string) (string, error) {
	candidate := filepath.Join(filepath.Dir(runmeHarbor), bundledHarborExecutableName())
	if _, err := resolveExecutable(candidate, exec.LookPath); err != nil {
		return "", fmt.Errorf("runme: Harbor installation is missing a required executable.\n\nReinstall with:\n  uv tool install runme-harbor --force")
	}
	return candidate, nil
}

func bundledHarborExecutableName() string {
	if runtime.GOOS == "windows" {
		return "runme-harbor-harbor.exe"
	}
	return "runme-harbor-harbor"
}

func openLocalPorts(start, limit int) ([]int, error) {
	ports := []int{}
	for port := start; port <= limit; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = listener.Close()
			ports = append(ports, port)
		}
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("no open local port found in range %d-%d", start, limit)
	}
	return ports, nil
}

func dashboardURLs(ports []int) []string {
	urls := make([]string, 0, len(ports))
	for _, port := range ports {
		urls = append(urls, fmt.Sprintf("http://127.0.0.1:%d", port))
	}
	return urls
}

func waitForDashboard(urls []string) (string, error) {
	client := http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		for _, url := range urls {
			resp, err := client.Get(url)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode < 500 {
					return url, nil
				}
				lastErr = fmt.Errorf("%s returned status %s", url, resp.Status)
			} else {
				lastErr = err
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("timed out waiting for dashboard")
}

type externalStartedCommand struct {
	cmd *exec.Cmd
}

func (c externalStartedCommand) Wait() error {
	return waitExternalCommand(c.cmd)
}

func startExternalCommand(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (StartedCommand, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = workingDir
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return externalStartedCommand{cmd: cmd}, nil
}
