package dagger

import (
	"path/filepath"
	"strings"
)

const (
	LegacyCommand = "dagger shell"
	Command       = "dagger script"
)

// IsCommand reports whether command invokes Dagger's legacy shell command or
// its replacement, script.
func IsCommand(command string) bool {
	_, ok := subcommandIndex(strings.Fields(command))
	return ok
}

// CanonicalizeCommand returns command with Dagger's legacy shell subcommand
// replaced by script. Unrecognized commands are returned unchanged.
func CanonicalizeCommand(command string) string {
	fields := strings.Fields(command)
	index, ok := subcommandIndex(fields)
	if !ok {
		return command
	}

	fields[index] = "script"
	return strings.Join(fields, " ")
}

func subcommandIndex(fields []string) (int, bool) {
	if len(fields) < 2 {
		return 0, false
	}

	daggerIndex := 0
	if filepath.Base(fields[0]) == "env" {
		daggerIndex++
		for daggerIndex < len(fields) && (strings.HasPrefix(fields[daggerIndex], "-") || strings.Contains(fields[daggerIndex], "=")) {
			daggerIndex++
		}
	}

	if daggerIndex+1 >= len(fields) || filepath.Base(fields[daggerIndex]) != "dagger" {
		return 0, false
	}

	switch fields[daggerIndex+1] {
	case "shell", "script":
		return daggerIndex + 1, true
	default:
		return 0, false
	}
}
