package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runmedev/runme/v3/session"
)

func TestIsShellProgram_DaggerCommands(t *testing.T) {
	t.Parallel()

	assert.True(t, isShellProgram("dagger shell"))
	assert.True(t, isShellProgram("dagger script"))
	assert.False(t, isShellProgram("not dagger shell"))
}

func TestProgramPathCanonicalizesLegacyDaggerCommand(t *testing.T) {
	sess, err := session.New()
	require.NoError(t, err)

	cmd := &base{
		cfg:     &ProgramConfig{ProgramName: "dagger shell"},
		runtime: dockerRuntime{},
		session: sess,
	}

	program, args, err := cmd.ProgramPath()
	require.NoError(t, err)
	assert.Equal(t, "dagger", program)
	assert.Equal(t, []string{"script"}, args)
}
