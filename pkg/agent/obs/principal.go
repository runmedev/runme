package obs

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	iam "github.com/runmedev/runme/v3/pkg/agent/iam"
	"go.openai.org/lib/oaigo/telemetry/oailog"
)

// GetPrincipal returns the email associated with the ID token in ctx, or an empty string if unavailable.
func GetPrincipal(ctx context.Context) string {
	idToken, err := iam.GetIDToken(ctx)
	if err != nil || idToken == nil {
		return ""
	}

	claims, ok := idToken.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}

	if email, ok := claims["email"].(string); ok {
		return email
	}

	return ""
}

func NewContextWithPrincipal(ctx context.Context) context.Context {
	principal := GetPrincipal(ctx)
	if principal == "" {
		// Return the same ctx
		return ctx
	}

	return oailog.WithContextAttrs(ctx, oailog.String("principal", principal))
}
