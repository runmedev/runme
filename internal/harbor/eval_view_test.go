package harbor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

type fakeStartedCommand struct {
	err error
}

func (c fakeStartedCommand) Wait() error {
	return c.err
}

func TestRunEvalViewDefaultsJobsDirFromGitRoot(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	jobsDir := filepath.Join(repoRoot, DefaultEvalJobsDir)
	if err := os.MkdirAll(jobsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	var calls []recordedCommand
	opts := testEvalViewOptions(t, &calls, io.Discard)
	opts.Port = 9090
	opts.Open = false

	if err := NewEvalViewer(opts).Run(nil); err != nil {
		t.Fatal(err)
	}

	want := []string{"view", defaultJobsArg(), "--jobs", "--port", "9090"}
	if !reflect.DeepEqual(calls[0].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[0].args, want)
	}
	if calls[0].workingDir != repoRoot {
		t.Fatalf("workingDir = %q, want %q", calls[0].workingDir, repoRoot)
	}
}

func TestRunEvalViewExplicitJobsDirUsesInvocationCwd(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := git.PlainInit(repoRoot, false); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(repoRoot, "nested", "dir")
	jobsDir := filepath.Join(nested, "custom", "jobs")
	if err := os.MkdirAll(jobsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	var calls []recordedCommand
	opts := testEvalViewOptions(t, &calls, io.Discard)
	opts.Port = 9090
	opts.Open = false

	if err := NewEvalViewer(opts).Run([]string{"custom/jobs"}); err != nil {
		t.Fatal(err)
	}

	want := []string{"view", filepath.Join("custom", "jobs"), "--jobs", "--port", "9090"}
	if !reflect.DeepEqual(calls[0].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[0].args, want)
	}
	if calls[0].workingDir != cleanExistingPath(nested) {
		t.Fatalf("workingDir = %q, want %q", calls[0].workingDir, cleanExistingPath(nested))
	}
}

func TestRunEvalViewDelegatesBundledHarborAndOpensDashboard(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalJobsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	var opened []string
	var ready []string
	var stderr bytes.Buffer
	opts := testEvalViewOptions(t, &calls, &stderr)
	opts.Port = 9090
	opts.Debug = true
	opts.DashboardReady = func(urls []string) (string, error) {
		ready = append(ready, urls...)
		return urls[0], nil
	}
	opts.BrowserOpen = func(url string) error {
		opened = append(opened, url)
		return nil
	}

	if err := NewEvalViewer(opts).Run(nil); err != nil {
		t.Fatal(err)
	}

	bundledHarbor := filepath.Join(filepath.Dir(opts.RunmeHarborBin), "runme-harbor-harbor")
	want := recordedCommand{
		name:       bundledHarbor,
		args:       []string{"view", defaultJobsArg(), "--jobs", "--port", "9090"},
		workingDir: cleanExistingPath(tmp),
	}
	if !sameCommand(calls[0], want) || calls[0].workingDir != want.workingDir {
		t.Fatalf("call = %#v, want %#v", calls[0], want)
	}
	wantURL := "http://127.0.0.1:9090"
	if !reflect.DeepEqual(ready, []string{wantURL}) {
		t.Fatalf("ready = %#v, want %q", ready, wantURL)
	}
	if !reflect.DeepEqual(opened, []string{wantURL}) {
		t.Fatalf("opened = %#v, want %q", opened, wantURL)
	}
	wantDebug := shellCommandString(append([]string{bundledHarbor}, want.args...)) + "\n"
	if stderr.String() != wantDebug {
		t.Fatalf("debug = %q, want %q", stderr.String(), wantDebug)
	}
}

func TestRunEvalViewNoOpenSkipsDashboardOpen(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalJobsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	opts := testEvalViewOptions(t, &calls, io.Discard)
	opts.Port = 9090
	opts.Open = false
	opts.DashboardReady = func([]string) (string, error) {
		t.Fatal("DashboardReady called")
		return "", nil
	}
	opts.BrowserOpen = func(string) error {
		t.Fatal("BrowserOpen called")
		return nil
	}

	if err := NewEvalViewer(opts).Run(nil); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("calls = %#v, want one delegate", calls)
	}
}

func TestRunEvalViewAutoPortDelegatesRange(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalJobsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	var ready []string
	opts := testEvalViewOptions(t, &calls, io.Discard)
	opts.Port = 0
	opts.DashboardReady = func(urls []string) (string, error) {
		ready = append(ready, urls...)
		return "http://127.0.0.1:8081", nil
	}

	if err := NewEvalViewer(opts).Run(nil); err != nil {
		t.Fatal(err)
	}

	want := []string{"view", defaultJobsArg(), "--jobs", "--port", "8080-8180"}
	if !reflect.DeepEqual(calls[0].args, want) {
		t.Fatalf("args = %#v, want %#v", calls[0].args, want)
	}
	if len(ready) == 0 {
		t.Fatal("DashboardReady did not receive candidate URLs")
	}
}

func TestRunEvalViewWarnsWhenDashboardOpenFails(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalJobsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	var stderr bytes.Buffer
	opts := testEvalViewOptions(t, &calls, &stderr)
	opts.Port = 9090
	opts.DashboardReady = func(urls []string) (string, error) { return urls[0], nil }
	opts.BrowserOpen = func(string) error { return errors.New("no opener") }

	if err := NewEvalViewer(opts).Run(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "warning: failed to open dashboard") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunEvalViewMissingRunmeHarbor(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalJobsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	opts := testEvalViewOptions(t, &calls, io.Discard)
	opts.RunmeHarborBin = ""
	opts.LookPath = func(string) (string, error) {
		return "", os.ErrNotExist
	}

	err := NewEvalViewer(opts).Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRunmeHarborMissing) {
		t.Fatalf("error = %q, want ErrRunmeHarborMissing", err.Error())
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func TestRunEvalViewMissingBundledHarbor(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	if err := os.MkdirAll(DefaultEvalJobsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []recordedCommand
	opts := testEvalViewOptions(t, &calls, io.Discard)
	if err := os.Remove(filepath.Join(filepath.Dir(opts.RunmeHarborBin), "runme-harbor-harbor")); err != nil {
		t.Fatal(err)
	}

	err := NewEvalViewer(opts).Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "uv tool install runme-harbor --force") {
		t.Fatalf("error = %q", err.Error())
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func TestOpenLocalPortsSkipsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	occupied := listener.Addr().(*net.TCPAddr).Port
	if occupied == 65535 {
		t.Skip("occupied port has no next port")
	}
	next := occupied + 1
	probe, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", next))
	if err != nil {
		t.Skipf("next port %d is unavailable: %v", next, err)
	}
	_ = probe.Close()

	ports, err := openLocalPorts(occupied, occupied+1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ports, []int{next}) {
		t.Fatalf("ports = %#v, want [%d]", ports, next)
	}
}

func TestRunEvalMissingHarborBeforeMissingDefaultDataset(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	var calls []recordedCommand
	opts := testEvalOptions(t, &calls, io.Discard)
	opts.LookPath = func(string) (string, error) {
		return "", os.ErrNotExist
	}

	err := NewEvalRunner(opts).Run(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRunmeHarborMissing) {
		t.Fatalf("error = %q, want ErrRunmeHarborMissing", err.Error())
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want none", calls)
	}
}

func testEvalViewOptions(t *testing.T, calls *[]recordedCommand, stderr io.Writer) EvalViewOptions {
	t.Helper()
	binDir := t.TempDir()
	runmeHarbor := filepath.Join(binDir, "runme-harbor")
	bundledHarbor := filepath.Join(binDir, "runme-harbor-harbor")
	for _, path := range []string{runmeHarbor, bundledHarbor} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return EvalViewOptions{
		Port:           9090,
		Open:           true,
		RunmeHarborBin: runmeHarbor,
		CommandStart:   recordStartedCommand(calls, nil),
		BrowserOpen:    func(string) error { return nil },
		DashboardReady: func(urls []string) (string, error) { return urls[0], nil },
		Stdin:          nil,
		Stdout:         io.Discard,
		Stderr:         stderr,
	}
}

func recordStartedCommand(calls *[]recordedCommand, err error) CommandStartFunc {
	return func(name string, args []string, workingDir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (StartedCommand, error) {
		*calls = append(*calls, recordedCommand{
			name:       name,
			args:       append([]string(nil), args...),
			workingDir: workingDir,
			env:        append([]string(nil), env...),
		})
		return fakeStartedCommand{err: err}, nil
	}
}
