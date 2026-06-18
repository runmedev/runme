package ansi

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/mgutz/ansi"
)

func DisableColors() bool {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return true
	}

	return false
}

var IsColorDisabled = DisableColors()

// Color is a wrapper around ansi.Color that respects the NO_COLOR environment variable
func Color(s string, style string) string {
	if IsColorDisabled {
		return s
	}

	return ansi.Color(s, style)
}

// ColorForWriter colors s only when w is connected to a terminal.
func ColorForWriter(w io.Writer, s, style string) string {
	if !WriterSupportsColor(w) {
		return s
	}
	return Color(s, style)
}

// WriterSupportsColor reports whether w is connected to a terminal that can display ANSI colors.
func WriterSupportsColor(w io.Writer) bool {
	if file, ok := w.(*os.File); ok {
		return isTerminal(file.Fd())
	}
	if provider, ok := w.(interface{ StdoutFile() *os.File }); ok {
		file := provider.StdoutFile()
		return file != nil && isTerminal(file.Fd())
	}
	return false
}

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
