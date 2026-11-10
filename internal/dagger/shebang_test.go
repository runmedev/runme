package dagger

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/runmedev/runme/v3/internal/execution"
)

func TestRewriteShebang(t *testing.T) {
	for _, header := range []string{
		"dagger shell", "dagger --debug shell --quiet",
		"/usr/bin/env -S dagger shell",
		`"/path with spaces/dagger" shell`,
		`'C:\Program Files\Dagger\dagger.exe' shell`,
		`C:\tools\dagger.exe "shell"`,
		"/usr/bin/env -S '/path with spaces/dagger' --debug shell --quiet",
	} {
		for _, newline := range []string{"\n", "\r\n", ""} {
			script := "#!" + header + newline
			if newline != "" {
				script += "echo shell script\r\n"
			}
			want := "#!" + replaceShell(header) + newline
			if newline != "" {
				want += "echo shell script\r\n"
			}
			assert.Equal(t, want, RewriteShebang(script, execution.NewInvocation("/usr/bin/env", []string{"-S", "dagger", "--debug", "script"})))
			assert.Equal(t, script, RewriteShebang(want, execution.NewInvocation("dagger", []string{"shell", "--quiet"})))
		}
	}
	for _, script := range []string{
		"", "echo shell\n", "#!/bin/bash\necho shell\n",
		"#!/usr/bin/env python\n# dagger shell\n",
		"#!dagger shell\n", "#!'unterminated dagger shell\n",
	} {
		assert.Equal(t, script, RewriteShebang(script, execution.NewInvocation("bash", []string{"-c"})))
	}
	for _, script := range []string{"echo shell\n", "#!/bin/bash\necho shell\n", "#!'unterminated dagger shell\n"} {
		assert.Equal(t, script, RewriteShebang(script, execution.NewInvocation("dagger", []string{"script"})))
	}
}

// Only the header's interpreter token changes.
func replaceShell(header string) string {
	for i := 0; i+5 <= len(header); i++ {
		if header[i:i+5] == "shell" {
			return header[:i] + "script" + header[i+5:]
		}
	}
	return header
}
