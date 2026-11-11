package dagger

import (
	"strings"

	"github.com/runmedev/runme/v3/internal/execution"
)

// RenderScript returns a self-contained script for an already resolved Dagger
// invocation. Non-Dagger scripts are returned unchanged.
func RenderScript(invocation execution.Invocation, body string) string {
	if _, ok := parseInvocation(invocation.Executable(), invocation.Arguments()); !ok {
		return body
	}

	header := invocation.Shebang()
	if !strings.HasPrefix(body, "#!") {
		if body == "" {
			return header
		}
		return header + "\n" + body
	}

	end := strings.IndexAny(body, "\r\n")
	if end < 0 {
		end = len(body)
	}
	if isDaggerShebang(body[2:end]) {
		return header + body[end:]
	}
	return header + "\n" + body
}

func isDaggerShebang(header string) bool {
	tokens, ok := headerTokens(header)
	if !ok || len(tokens) == 0 {
		return false
	}
	fields := make([]string, len(tokens))
	for i, token := range tokens {
		fields[i] = token.value
	}
	_, ok = parseInvocation(fields[0], fields[1:])
	return ok
}

type headerToken struct {
	value      string
	start, end int
}

// headerTokens removes quotes for recognition while retaining byte spans for edits.
// Backslashes in paths are literal; no shell expansion is performed.
func headerTokens(header string) ([]headerToken, bool) {
	var tokens []headerToken
	for i := 0; i < len(header); {
		if header[i] == ' ' || header[i] == '\t' {
			i++
			continue
		}
		start := i
		var value strings.Builder
		var quote byte
		for i < len(header) {
			c := header[i]
			if quote == 0 && (c == ' ' || c == '\t') {
				break
			}
			if c == '\'' || c == '"' {
				if quote == 0 {
					quote = c
					i++
					continue
				}
				if quote == c {
					quote = 0
					i++
					continue
				}
			}
			_ = value.WriteByte(c)
			i++
		}
		if quote != 0 {
			return nil, false
		}
		tokens = append(tokens, headerToken{value.String(), start, i})
	}
	return tokens, true
}
