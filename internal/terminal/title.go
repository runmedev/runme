package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-isatty"
)

const (
	noTitleEnv = "RUNME_NO_TERMINAL_TITLE"
	titleMax   = 100
)

type Title string

func SetTitle(w io.Writer, title string) {
	TitleWriter{w: w}.Set(Title(title))
}

type TitleWriter struct {
	w io.Writer
}

func (t TitleWriter) Set(title Title) {
	if os.Getenv(noTitleEnv) != "" || os.Getenv("TERM") == "dumb" {
		return
	}

	f, ok := t.w.(*os.File)
	if !ok || !isTerminal(f.Fd()) {
		return
	}

	title = title.Sanitize()
	if title.Empty() {
		return
	}

	_, _ = fmt.Fprintf(t.w, "\x1b]0;%s\x1b\\", title)
}

func (t Title) Empty() bool {
	return t == ""
}

func (t Title) Sanitize() Title {
	return Title(SanitizeTitle(string(t)))
}

func SanitizeTitle(title string) string {
	title = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, title)

	title = strings.TrimSpace(title)
	if utf8.RuneCountInString(title) <= titleMax {
		return title
	}

	runes := []rune(title)
	return string(runes[:titleMax-3]) + "..."
}

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
