package ansi

import (
	"os"

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
