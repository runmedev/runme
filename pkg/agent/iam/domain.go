package iam

import "strings"

// domainMatcher checks whether the principal is a member of a domain
type domainMatcher struct {
	domain string
}

func (m *domainMatcher) Check(principal string) bool {
	pieces := strings.Split(principal, "@")
	if len(pieces) != 2 {
		return false
	}

	return pieces[1] == m.domain
}
