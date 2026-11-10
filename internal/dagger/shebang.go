package dagger

import "strings"

// SyncShebang aligns an existing Dagger header with an already resolved invocation.
// It never resolves executables or probes versions.
func SyncShebang(script, program string, args []string) string {
	resolved, ok := parseInvocation(program, args)
	if !ok || !strings.HasPrefix(script, "#!") {
		return script
	}
	end := strings.IndexAny(script, "\r\n")
	if end < 0 {
		end = len(script)
	}
	tokens, ok := headerTokens(script[2:end])
	if !ok || len(tokens) == 0 {
		return script
	}
	fields := make([]string, len(tokens))
	for i, token := range tokens {
		fields[i] = token.value
	}
	header, ok := parseInvocation(fields[0], fields[1:])
	if !ok || header.interpreter == resolved.interpreter {
		return script
	}
	token := tokens[header.subcommandIndex]
	spelling := resolved.command.fields()[resolved.subcommandIndex]
	raw := script[2+token.start : 2+token.end]
	// Preserve quotes around the interpreter token.
	var replacement strings.Builder
	for _, c := range raw {
		if c == '\'' || c == '"' {
			_, _ = replacement.WriteRune(c)
		} else if spelling != "" {
			_, _ = replacement.WriteString(spelling)
			spelling = ""
		}
	}
	return script[:2+token.start] + replacement.String() + script[2+token.end:]
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
