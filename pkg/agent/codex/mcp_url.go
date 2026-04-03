package codex

import (
	"fmt"
	"net/http"
	"strings"
)

func mcpServerURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		part := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if part != "" {
			scheme = part
		}
	}
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	return fmt.Sprintf("%s://%s/mcp/notebooks", scheme, host)
}
