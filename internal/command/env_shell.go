package command

import (
	"io"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

const StoreStdoutEnvName = "__"

func CreateEnv(key, value string) string {
	return createEnv(key, value)
}

func createEnv(key, value string) string {
	return key + "=" + value
}

// ScriptEnvSetter returns a shell script that installs itself and
// collects environment variables to provided pre- and post-paths.
type ScriptEnvSetter struct {
	debug       bool
	dumpCommand string
	prePath     string
	postPath    string
}

func NewScriptEnvSetter(prePath, postPath string, debug bool) ScriptEnvSetter {
	return ScriptEnvSetter{
		debug:       debug,
		dumpCommand: envDumpCommand,
		prePath:     prePath,
		postPath:    postPath,
	}
}

func (s ScriptEnvSetter) SetOnShell(shell io.Writer) error {
	if err := s.validate(); err != nil {
		return err
	}
	return setOnShell(shell, s.dumpCommand, false, true, s.debug, s.prePath, s.postPath)
}

func (s ScriptEnvSetter) validate() (err error) {
	if s.prePath == "" {
		err = multierr.Append(err, errors.New("pre-path is required"))
	}
	if s.postPath == "" {
		err = multierr.Append(err, errors.New("post-path is required"))
	}
	return
}

func setOnShell(
	shell io.Writer,
	dumpCommand string,
	skipShellHistory bool,
	asFile bool,
	debug bool,
	prePath string,
	postPath string,
) error {
	prefix := ""
	if skipShellHistory {
		prefix = " " // space avoids polluting the shell history
	}

	w := bulkWriter{Writer: shell}

	if asFile {
		w.WriteString("#!/bin/sh\n")
	}

	if debug {
		w.WriteString("set -euxo pipefail\n")
	}

	// Dump all env at the beginning, so that a diff can be calculated.
	w.WriteString(prefix + dumpCommand + " > " + prePath + "\n")
	// Then, set a trap on EXIT to dump all env at the end.
	w.WriteString(prefix + "__cleanup() {\nrv=$?\n" + (envDumpCommand + " > " + postPath) + "\nexit $rv\n}\n")
	w.WriteString(prefix + "trap -- \"__cleanup\" EXIT\n")

	if debug {
		w.WriteString("set +euxo pipefail\n")
	}

	_, err := w.Done()
	return err
}
