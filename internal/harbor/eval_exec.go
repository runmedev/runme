package harbor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/mattn/go-isatty"
)

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
	if err := cmd.Start(); err != nil {
		return err
	}
	return waitExternalCommand(cmd)
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

	waitErr := waitExternalCommand(cmd)
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

func waitExternalCommand(cmd *exec.Cmd) error {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(signals)
		close(done)
	}()

	go func() {
		for {
			select {
			case sig := <-signals:
				if sig != os.Interrupt && cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	return cmd.Wait()
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
