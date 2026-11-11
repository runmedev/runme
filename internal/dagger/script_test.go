package dagger

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/runmedev/runme/v3/internal/execution"
)

func TestRenderScript(t *testing.T) {
	tests := []struct {
		name       string
		invocation execution.Invocation
		body       string
		want       string
	}{
		{
			name:       "legacy",
			invocation: execution.NewInvocation("dagger", []string{"--debug", "shell"}),
			body:       "echo shell\n",
			want:       "#!/usr/bin/env -S dagger --debug shell\necho shell\n",
		},
		{
			name:       "script",
			invocation: execution.NewInvocation("dagger", []string{"--debug", "script"}),
			body:       "echo shell",
			want:       "#!/usr/bin/env -S dagger --debug script\necho shell",
		},
		{
			name:       "replace legacy",
			invocation: execution.NewInvocation("dagger", []string{"--debug", "script"}),
			body:       "#!/usr/bin/env -S dagger --debug shell\necho shell\n",
			want:       "#!/usr/bin/env -S dagger --debug script\necho shell\n",
		},
		{
			name:       "replace script",
			invocation: execution.NewInvocation("dagger", []string{"--debug", "shell"}),
			body:       "#!/usr/bin/env -S dagger --debug script\necho shell\n",
			want:       "#!/usr/bin/env -S dagger --debug shell\necho shell\n",
		},
		{
			name:       "recognize quoted header and preserve CRLF",
			invocation: execution.NewInvocation("C:\\Program Files\\dagger.exe", []string{"--x-release", "main", "script", "--debug"}),
			body:       "#!'/usr/bin/env' -S \"C:\\Program Files\\dagger.exe\" --x-release main shell --debug\r\necho shell\r\n",
			want:       "#!/usr/bin/env -S 'C:\\Program Files\\dagger.exe' --x-release main script --debug\r\necho shell\r\n",
		},
		{
			name:       "empty body",
			invocation: execution.NewInvocation("dagger", []string{"script"}),
			want:       "#!/usr/bin/env -S dagger script",
		},
		{
			name:       "non-Dagger invocation",
			invocation: execution.NewInvocation("bash", nil),
			body:       "echo shell",
			want:       "echo shell",
		},
		{
			name:       "preserve non-Dagger header",
			invocation: execution.NewInvocation("dagger", []string{"script"}),
			body:       "#!/bin/bash\necho shell\n",
			want:       "#!/usr/bin/env -S dagger script\n#!/bin/bash\necho shell\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderScript(tt.invocation, tt.body)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, got, RenderScript(tt.invocation, got))
		})
	}
}
