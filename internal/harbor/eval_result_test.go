package harbor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runmedev/runme/v3/internal/ansi"
)

func TestResultPathWriterStreamsBeforeNewline(t *testing.T) {
	var stdout bytes.Buffer
	writer := NewResultPathWriter(&stdout)

	if _, err := writer.Write([]byte("1/1 Mean: 0.000")); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "1/1 Mean: 0.000" {
		t.Fatalf("stdout = %q, want streamed partial line", got)
	}
	if got := writer.ResultPath(); got != "" {
		t.Fatalf("result path = %q, want empty before result line", got)
	}
}

func TestResultPathWriterRecordsResultPath(t *testing.T) {
	var stdout bytes.Buffer
	writer := NewResultPathWriter(&stdout)
	resultPath := filepath.Join(t.TempDir(), "result.json")

	if _, err := writer.Write([]byte("Results written to " + resultPath + "\n")); err != nil {
		t.Fatal(err)
	}
	if got := writer.ResultPath(); got != resultPath {
		t.Fatalf("result path = %q, want %q", got, resultPath)
	}
}

func TestJobDirFromResultPathResolvesRelativeToBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	resultPath := filepath.Join("..", "jobs", "current", "result.json")
	want := filepath.Join(filepath.Dir(baseDir), "jobs", "current")

	if got := JobDirFromResultPath(resultPath, baseDir); got != want {
		t.Fatalf("job dir = %q, want %q", got, want)
	}
}

func TestPrintExceptionDetailsOnlyUsesReportedJob(t *testing.T) {
	jobsDir := filepath.Join(t.TempDir(), "jobs")
	jobDir := filepath.Join(jobsDir, "current")
	writeException(t, filepath.Join(jobsDir, "other", "attempt", "exception.txt"), "other detail\n")
	writeException(t, filepath.Join(jobDir, "attempt", "exception.txt"), "current detail\n")

	var stdout bytes.Buffer
	PrintExceptionDetails(&stdout, jobDir)

	output := string(ansi.Strip(stdout.Bytes()))
	if !strings.Contains(output, "Harbor Exception Details") || !strings.Contains(output, "current detail") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "other detail") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "File: attempt/exception.txt") {
		t.Fatalf("output = %q", output)
	}
}

func TestPrintExceptionDetailsDoesNotColorNonTerminalOutput(t *testing.T) {
	jobDir := t.TempDir()
	writeException(t, filepath.Join(jobDir, "attempt", "exception.txt"), "current detail\n")

	var stdout bytes.Buffer
	PrintExceptionDetails(&stdout, jobDir)

	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatalf("output contains ANSI escape sequence: %q", stdout.String())
	}
	if strings.HasPrefix(stdout.String(), "\n") {
		t.Fatalf("output starts with extra newline: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Harbor Exception Details") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func writeException(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
