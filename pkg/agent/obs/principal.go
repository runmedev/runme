package obs

import (
	"context"

	"github.com/golang-jwt/jwt/v5"

	"github.com/runmedev/runme/v3/pkg/agent/iam"
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

	// TODO(jlewi): This is a place holder method to attach the principal to the context as a logging attribute
	// so that we can log it as part of all messages. We currently don't have a solution for passing around
	// logging attributes as part of the context.
	// would https://github.com/veqryn/slog-context?
	// The problem with attaching loggers to the context is that its impossible to know whether a particular field
	// has already been added as an attribute to the logger; leading to duplicates.
	// With context attributes it should be possible to implement last one wins semantics.
	// return slog.NewContext().WithContextAttrs(ctx, oailog.String("principal", principal))
	return ctx
}
