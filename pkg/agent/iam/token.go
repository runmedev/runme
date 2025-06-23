package iam

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// IDTokenKeyType  string is the key used to store the ID token in the context
// We define a type to avoid collisions with other context keys
type IDTokenKeyType string

const (
	IDTokenKey IDTokenKeyType = "idToken"
)

// GetIDToken retrieves the ID token from the context if there is one or nil
func GetIDToken(ctx context.Context) (*jwt.Token, error) {
	idToken := ctx.Value(IDTokenKey)
	if idToken == nil {
		return nil, errors.New("No ID token")
	}
	token, ok := idToken.(*jwt.Token)
	if !ok {
		return nil, errors.New("ID token is not of type *jwt.Token")
	}
	return token, nil
}

// ContextWithIDToken adds the IDToken to the context
func ContextWithIDToken(ctx context.Context, idToken *jwt.Token) context.Context {
	return context.WithValue(ctx, IDTokenKey, idToken)
}
