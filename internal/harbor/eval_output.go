package harbor

import (
	"io"

	"github.com/runmedev/runme/v3/internal/ansi"
)

const evalOutputLabelStyle = "+b"

func evalOutputLabel(w io.Writer, label string) string {
	return ansi.ColorForWriter(w, label, evalOutputLabelStyle)
}
