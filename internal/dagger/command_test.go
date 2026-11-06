package dagger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		command   string
		isDagger  bool
		canonical string
	}{
		{name: "legacy", command: "dagger shell", isDagger: true, canonical: Command},
		{name: "canonical", command: Command, isDagger: true, canonical: Command},
		{name: "path", command: "/usr/local/bin/dagger shell", isDagger: true, canonical: "/usr/local/bin/dagger script"},
		{name: "env", command: "/usr/bin/env dagger shell", isDagger: true, canonical: "/usr/bin/env dagger script"},
		{name: "env option", command: "env -S dagger shell", isDagger: true, canonical: "env -S dagger script"},
		{name: "arguments", command: "dagger shell --debug", isDagger: true, canonical: "dagger script --debug"},
		{name: "unrelated", command: "bash", canonical: "bash"},
		{name: "substring", command: "not dagger shell", canonical: "not dagger shell"},
		{name: "suffix", command: "dagger shellcheck", canonical: "dagger shellcheck"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.isDagger, IsCommand(tt.command))
			assert.Equal(t, tt.canonical, CanonicalizeCommand(tt.command))
		})
	}
}
