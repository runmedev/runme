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

func TestIntentShebang(t *testing.T) {
	for _, tt := range []struct {
		name   string
		intent Intent
		want   string
	}{
		{name: "empty", intent: NewIntent("", nil), want: ""},
		{name: "single field", intent: NewIntent("/bin/bash", nil), want: "#!/bin/bash"},
		{name: "multiple fields", intent: NewIntent("dagger", []string{"--debug", "shell"}), want: "#!/usr/bin/env -S dagger --debug shell"},
		{name: "env", intent: NewIntent("env", []string{"dagger", "shell"}), want: "#!/usr/bin/env -S dagger shell"},
		{name: "normalized env", intent: NewIntent("/usr/bin/env", []string{"-S", "dagger", "shell"}), want: "#!/usr/bin/env -S dagger shell"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.intent.Shebang())
		})
	}
}
