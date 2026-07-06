package cmd

import (
	"errors"
	"os/exec"
)

type ExitCodeError struct {
	Code int
	Err  error
}

// asExitCodeError maps an error that carries a process exit code into an
// ExitCodeError, but only when the code is non-zero. A zero code from a
// non-nil error must not be reported as success; such errors are returned
// unchanged so the top-level handler exits with a failing status.
func asExitCodeError(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() != 0 {
		return ExitCodeError{Code: exitErr.ExitCode(), Err: err}
	}
	var codeErr interface{ ExitCode() int }
	if errors.As(err, &codeErr) && codeErr.ExitCode() != 0 {
		return ExitCodeError{Code: codeErr.ExitCode(), Err: err}
	}
	return err
}

func (e ExitCodeError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "command failed"
}

func (e ExitCodeError) Unwrap() error {
	return e.Err
}
