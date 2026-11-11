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

func TestInvocationShebang(t *testing.T) {
	for _, tt := range []struct {
		name       string
		invocation Invocation
		want       string
	}{
		{name: "empty", invocation: NewInvocation("", nil), want: ""},
		{name: "single field", invocation: NewInvocation("/bin/bash", nil), want: "#!/bin/bash"},
		{name: "multiple fields", invocation: NewInvocation("dagger", []string{"--debug", "script"}), want: "#!/usr/bin/env -S dagger --debug script"},
		{name: "path with spaces", invocation: NewInvocation("/tmp/dagger bin/dagger", []string{"script"}), want: "#!/usr/bin/env -S '/tmp/dagger bin/dagger' script"},
		{name: "empty and spaced arguments", invocation: NewInvocation("dagger", []string{"script", "", "two words"}), want: "#!/usr/bin/env -S dagger script '' 'two words'"},
		{name: "env", invocation: NewInvocation("env", []string{"dagger", "script"}), want: "#!/usr/bin/env -S dagger script"},
		{name: "normalized env", invocation: NewInvocation("/usr/bin/env", []string{"-S", "dagger", "script"}), want: "#!/usr/bin/env -S dagger script"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.invocation.Shebang())
		})
	}
}
