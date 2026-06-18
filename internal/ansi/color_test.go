package ansi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stdoutFileWriter struct {
	file *os.File
}

func (w stdoutFileWriter) Write(p []byte) (int, error) {
	return w.file.Write(p)
}

func (w stdoutFileWriter) StdoutFile() *os.File {
	return w.file
}

func TestColorForWriterLeavesNonTerminalOutputPlain(t *testing.T) {
	var stdout bytes.Buffer

	got := ColorForWriter(&stdout, "plain", "red+b")

	if got != "plain" {
		t.Fatalf("ColorForWriter() = %q, want plain", got)
	}
}

func TestWriterSupportsColorRejectsNonTerminalFile(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	if WriterSupportsColor(file) {
		t.Fatal("WriterSupportsColor() = true, want false for regular file")
	}
	if WriterSupportsColor(stdoutFileWriter{file: file}) {
		t.Fatal("WriterSupportsColor() = true, want false for wrapper around regular file")
	}
	if strings.Contains(ColorForWriter(stdoutFileWriter{file: file}, "plain", "red+b"), "\x1b[") {
		t.Fatal("ColorForWriter() returned ANSI escapes for wrapper around regular file")
	}
}
