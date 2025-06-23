package iam

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/runmedev/runme/v3/pkg/agent/api"
)

func TestChecker_Check(t *testing.T) {
	complexPolicy := api.IAMPolicy{
		Bindings: []api.IAMBinding{
			{
				Role: api.RunnerUserRole,
				Members: []api.Member{
					{
						Name: "user1",
						Kind: api.UserKind,
					},
					{
						Name: "acme.com",
						Kind: api.DomainKind,
					},
				},
			},
			{
				Role: api.AgentUserRole,
				Members: []api.Member{
					{
						Name: "user2",
						Kind: api.UserKind,
					},
				},
			},
		},
	}

	testCases := []struct {
		name     string
		idToken  *jwt.Token
		role     string
		policy   api.IAMPolicy
		expected bool
	}{
		{
			name:     "User doesn't have role",
			idToken:  &jwt.Token{Claims: jwt.MapClaims{"email": "bob@beta.com"}},
			role:     api.RunnerUserRole,
			policy:   complexPolicy,
			expected: false,
		},
		{
			name:     "Domain has role",
			idToken:  &jwt.Token{Claims: jwt.MapClaims{"email": "alice@acme.com"}},
			role:     api.RunnerUserRole,
			policy:   complexPolicy,
			expected: true,
		},
		{
			// This test is intended to verify that the domain rule gets correctly
			// scoped to the role. Alice should have runner access but not agent access
			name:     "Role doesn't have domain rule but other does",
			idToken:  &jwt.Token{Claims: jwt.MapClaims{"email": "alice@acme.com"}},
			role:     api.AgentUserRole,
			policy:   complexPolicy,
			expected: false,
		},
		{
			name:     "User1 is allowed under member rule not domain rule",
			idToken:  &jwt.Token{Claims: jwt.MapClaims{"email": "user1"}},
			role:     api.RunnerUserRole,
			policy:   complexPolicy,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker, err := NewChecker(tc.policy)
			if err != nil {
				t.Fatalf("failed to create checker: %v", err)
			}

			principal, err := checker.GetPrincipal(tc.idToken)
			if err != nil {
				t.Fatalf("failed to extract principal from idToken: %v", err)
			}

			result := checker.Check(principal, tc.role)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
