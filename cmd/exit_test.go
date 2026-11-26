package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/runner"
)

type codeError struct {
	code int
}

func (e codeError) Error() string { return "code error" }

func (e codeError) ExitCode() int { return e.code }

func TestAsExitCodeError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.NoError(t, asExitCodeError(nil))
	})

	t.Run("non-zero code is wrapped", func(t *testing.T) {
		err := asExitCodeError(codeError{code: 2})
		var exitErr ExitCodeError
		require.ErrorAs(t, err, &exitErr)
		require.Equal(t, 2, exitErr.Code)
	})

	t.Run("zero code is not reported as success", func(t *testing.T) {
		src := codeError{code: 0}
		err := asExitCodeError(src)
		var exitErr ExitCodeError
		require.False(t, errors.As(err, &exitErr), "zero exit code must not be wrapped as ExitCodeError")
		require.ErrorIs(t, err, src)
	})

	t.Run("remote runner exit code without a wrapped process error", func(t *testing.T) {
		src := &runner.ExitError{Code: 42}
		err := asExitCodeError(src)
		var exitErr ExitCodeError
		require.ErrorAs(t, err, &exitErr)
		require.Equal(t, 42, exitErr.Code)
		require.ErrorIs(t, err, src)
	})

	t.Run("plain error passes through", func(t *testing.T) {
		src := errors.New("boom")
		require.ErrorIs(t, asExitCodeError(src), src)
	})
}
