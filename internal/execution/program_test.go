package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgramValuesOwnArguments(t *testing.T) {
	arguments := []string{"--debug"}
	intent := NewIntent("dagger shell", arguments)
	invocation := NewInvocation("/usr/bin/dagger", arguments)
	arguments[0] = "changed"

	assert.Equal(t, "dagger shell", intent.ProgramName())
	assert.Equal(t, []string{"--debug"}, intent.Arguments())
	assert.Equal(t, "/usr/bin/dagger", invocation.Executable())
	assert.Equal(t, []string{"--debug"}, invocation.Arguments())

	resolved := invocation.WithArguments("script.dsh")
	assert.Equal(t, []string{"--debug"}, invocation.Arguments())
	assert.Equal(t, []string{"--debug", "script.dsh"}, resolved.Arguments())
}
