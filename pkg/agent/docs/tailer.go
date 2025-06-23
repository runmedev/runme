package docs

import "strings"

func tailLines(s string, maxLen int) string {
	lines := strings.Split(s, "\n")

	startIndex := len(lines) - 1

	length := len(lines[len(lines)-1])

	for ; startIndex >= 1; startIndex-- {
		nextIndex := startIndex - 1
		if len(lines[nextIndex])+length > maxLen {
			break
		}

		length += len(lines[nextIndex])
	}

	if startIndex < 0 {
		startIndex = 0
	}

	return strings.Join(lines[startIndex:], "\n")
}
