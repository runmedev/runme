package iam

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestDomainMatcher_Check(t *testing.T) {
	tests := []struct {
		domain  string
		idToken *jwt.Token
		want    bool
	}{
		{
			domain:  "example.com",
			idToken: &jwt.Token{Claims: jwt.MapClaims{"email": "user@example.com"}},
			want:    true,
		},
		{
			domain:  "example.org",
			idToken: &jwt.Token{Claims: jwt.MapClaims{"email": "user@example.com"}},
			want:    false,
		},
		{
			domain:  "example.com",
			idToken: &jwt.Token{Claims: jwt.MapClaims{"email": "user@sub.example.com"}},
			want:    false,
		},
		{
			domain:  "example.com",
			idToken: &jwt.Token{Claims: jwt.MapClaims{"email": "invalid-email"}},
			want:    false,
		},
		{
			domain:  "example.com",
			idToken: &jwt.Token{Claims: jwt.MapClaims{"email": "another@example.com"}},
			want:    true,
		},
	}

	for _, tt := range tests {
		principal, _ := extractEmailFromIDToken(tt.idToken)
		t.Run(principal, func(t *testing.T) {
			matcher := &domainMatcher{domain: tt.domain}
			if got := matcher.Check(principal); got != tt.want {
				t.Errorf("domainMatcher.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}
