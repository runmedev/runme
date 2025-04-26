package daggershell

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/require"
)

func TestDaggerShell_FuncDecl(t *testing.T) {
	script := NewScript()

	err := script.DefineFunc("DAGGER_FUNCTION", `echo "Dagger Function Placeholder"`)
	require.NoError(t, err)

	t.Run("WithDaggerShebang", func(t *testing.T) {
		var rendered bytes.Buffer
		err = script.Render(&rendered, "dagger shell")
		require.NoError(t, err)

		const expected = `#!/usr/bin/env dagger shell
DAGGER_FUNCTION()
{
  echo "Dagger Function Placeholder"
}
`
		assert.Equal(t,
			expected,
			rendered.String(),
		)
	})

	t.Run("WithEmptyShebang", func(t *testing.T) {
		var rendered bytes.Buffer
		err = script.Render(&rendered, "")
		require.NoError(t, err)

		const expected = `DAGGER_FUNCTION()
{
  echo "Dagger Function Placeholder"
}
`
		assert.Equal(t,
			expected,
			rendered.String(),
		)
	})
}

func TestDaggerShell_Script(t *testing.T) {
	// can't use map because order is not guaranteed
	fakeCells := []struct {
		Name string
		Body string
	}{
		{"DAGGER_01JJDCG2SQSGV0DP55X86EJFSZ", `echo "Use known ID"; date;`},
		{"PRESETUP", `echo "This is PRESETUP" | xxd`},
		{"EXTENSION", `echo "This is EXTENSION" | less`},
		{"KERNEL_BINARY", `echo "This is KERNEL_BINARY"`},
	}

	expected := `#!/usr/bin/env dagger shell
DAGGER_01JJDCG2SQSGV0DP55X86EJFSZ()
{
  echo "Use known ID"
  date
}
PRESETUP()
{
  echo "This is PRESETUP" | xxd
}
EXTENSION()
{
  echo "This is EXTENSION" | less
}
KERNEL_BINARY()
{
  echo "This is KERNEL_BINARY"
}
`

	t.Run("Render", func(t *testing.T) {
		script := NewScript()
		for _, entry := range fakeCells {
			script.DefineFunc(entry.Name, entry.Body)
		}

		var rendered bytes.Buffer
		err := script.Render(&rendered, "dagger shell")
		require.NoError(t, err)

		assert.Equal(t, expected, rendered.String())
	})

	t.Run("RenderWithTarget", func(t *testing.T) {
		script := NewScript()
		for _, entry := range fakeCells {
			err := script.DefineFunc(entry.Name, entry.Body)
			require.NoError(t, err)
		}

		for _, entry := range fakeCells {
			var renderedWithCall bytes.Buffer
			err := script.RenderWithTarget(&renderedWithCall, "dagger shell", entry.Name)
			require.NoError(t, err)

			// add function call padded by new lines
			expectedBytesWithCall := strings.Join([]string{expected[:len(expected)-1], entry.Name, ""}, "\n")
			assert.Equal(t, expectedBytesWithCall, renderedWithCall.String())
		}
	})

	t.Run("RenderWithTarget_Invalid", func(t *testing.T) {
		script := NewScript()
		for _, entry := range fakeCells {
			err := script.DefineFunc(entry.Name, entry.Body)
			require.NoError(t, err)
		}

		var renderedWithCall bytes.Buffer
		err := script.RenderWithTarget(&renderedWithCall, "/usr/bin/env dagger shell", "INVALID")
		require.Error(t, err)
	})
}
