package obs

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	iam "github.com/runmedev/runme/v3/pkg/agent/iam"
)

func TestGetPrincipal(t *testing.T) {
	tt := []struct {
		name     string
		token    *jwt.Token
		expected string
	}{
		{
			name: "email present",
			token: jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
				"email": "user@example.com",
			}),
			expected: "user@example.com",
		},
		{
			name:     "no token",
			token:    nil,
			expected: "",
		},
		{
			name: "missing email claim",
			token: jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
				"sub": "user-123",
			}),
			expected: "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.token != nil {
				ctx = iam.ContextWithIDToken(ctx, tc.token)
			}

			got := GetPrincipal(ctx)
			if got != tc.expected {
				t.Fatalf("GetPrincipal() = %q; want %q", got, tc.expected)
			}
		})
	}
}
