package command

import (
	"slices"
	"testing"
)

func TestAppendKnownNameEnv(t *testing.T) {
	t.Parallel()

	t.Run("Named", func(t *testing.T) {
		t.Parallel()

		env := []string{"PATH=/bin", EnvNameKnownName + "=ambient"}
		got := AppendKnownNameEnv(env, "my-cell")

		want := []string{"PATH=/bin", EnvNameKnownName + "=my-cell"}
		if !slices.Equal(got, want) {
			t.Fatalf("AppendKnownNameEnv() = %q, want %q", got, want)
		}
	})

	t.Run("Unnamed", func(t *testing.T) {
		t.Parallel()

		env := []string{"PATH=/bin", EnvNameKnownName + "=ambient"}
		got := AppendKnownNameEnv(env, "")

		if !slices.Equal(got, env) {
			t.Fatalf("AppendKnownNameEnv() = %q, want %q", got, env)
		}
	})
}
