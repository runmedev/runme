package harbor

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runmedev/runme/v3/internal/ansi"
)

const maxResultLineLen = 64 * 1024

type ResultPathWriter struct {
	dst         io.Writer
	line        []byte
	lineTooLong bool
	resultPath  string
}

// NewResultPathWriter returns a writer that forwards Harbor output while capturing the result path.
func NewResultPathWriter(dst io.Writer) *ResultPathWriter {
	return &ResultPathWriter{dst: dst}
}

func (w *ResultPathWriter) Write(p []byte) (int, error) {
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

// ResultPath finalizes any pending output line and returns the last result path seen.
func (w *ResultPathWriter) ResultPath() string {
	if len(w.line) > 0 && !w.lineTooLong {
		w.recordLine(w.line)
		w.line = nil
	}
	return w.resultPath
}

// StdoutFile returns the forwarded stdout file when Harbor output is connected to one.
func (w *ResultPathWriter) StdoutFile() *os.File {
	file, _ := w.dst.(*os.File)
	return file
}

func (w *ResultPathWriter) appendLine(p []byte) {
	if w.lineTooLong {
		return
	}
	if len(w.line)+len(p) > maxResultLineLen {
		w.line = w.line[:0]
		w.lineTooLong = true
		return
	}
	w.line = append(w.line, p...)
}

func (w *ResultPathWriter) recordLine(line []byte) {
	// Harbor currently reports result locations only through this human-readable line.
	// Keep this coupling local until Harbor exposes a machine-readable marker.
	const prefix = "Results written to "
	text := strings.TrimSpace(string(ansi.Strip(line)))
	index := strings.Index(text, prefix)
	if index == -1 {
		return
	}
	w.resultPath = strings.TrimSpace(text[index+len(prefix):])
}

// JobDirFromResultPath returns the eval job directory containing Harbor's result file.
func JobDirFromResultPath(resultPath, evalBaseDir string) string {
	if resultPath == "" {
		return ""
	}
	if !filepath.IsAbs(resultPath) && evalBaseDir != "" {
		resultPath = filepath.Join(evalBaseDir, resultPath)
	}
	return filepath.Dir(resultPath)
}

// PrintExceptionDetails appends Harbor exception file details for an eval job.
func PrintExceptionDetails(w io.Writer, jobDir string) {
	paths := exceptionFiles(jobDir)
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
			_, _ = fmt.Fprintln(w, exceptionDetailsHeading(w))
			printed = true
		}
		_, _ = fmt.Fprintf(w, "\nFile: %s\n%s\n", exceptionDisplayPath(jobDir, path), detail)
	}
}

func exceptionDetailsHeading(w io.Writer) string {
	const heading = "Harbor Exception Details"
	if file, ok := w.(*os.File); ok && isTerminal(file.Fd()) {
		return ansi.Color(heading, "red+b")
	}
	if provider, ok := w.(interface{ StdoutFile() *os.File }); ok {
		if file := provider.StdoutFile(); file != nil && isTerminal(file.Fd()) {
			return ansi.Color(heading, "red+b")
		}
	}
	return heading
}

func exceptionDisplayPath(jobDir, path string) string {
	relative, err := filepath.Rel(jobDir, path)
	if err != nil || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || relative == ".." {
		return path
	}
	return relative
}

func exceptionFiles(jobDir string) []string {
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
