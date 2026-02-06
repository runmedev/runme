package iam

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func tokenSummary(idToken *jwt.Token) map[string]any {
	if idToken == nil {
		return map[string]any{"valid": false, "reason": "token is nil"}
	}

	summary := map[string]any{
		"valid": idToken.Valid,
	}

	claims, ok := idToken.Claims.(jwt.MapClaims)
	if !ok {
		summary["claimsType"] = fmt.Sprintf("%T", idToken.Claims)
		return summary
	}

	copyIfPresent(summary, claims, "iss", "hd", "azp")
	copyIfPresent(summary, claims, "aud")

	if sub, ok := claims["sub"].(string); ok {
		summary["sub"] = maskValue(sub, 6)
	}
	if email, ok := claims["email"].(string); ok {
		summary["email"] = maskEmail(email)
	}
	if exp, ok := asUnixTime(claims["exp"]); ok {
		summary["exp"] = exp.Format(time.RFC3339)
	}
	if iat, ok := asUnixTime(claims["iat"]); ok {
		summary["iat"] = iat.Format(time.RFC3339)
	}
	if nbf, ok := asUnixTime(claims["nbf"]); ok {
		summary["nbf"] = nbf.Format(time.RFC3339)
	}

	return summary
}

func copyIfPresent(dst map[string]any, src jwt.MapClaims, keys ...string) {
	for _, key := range keys {
		if v, ok := src[key]; ok {
			dst[key] = v
		}
	}
}

func asUnixTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0), true
	case int64:
		return time.Unix(t, 0), true
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return time.Unix(n, 0), true
		}
	}
	return time.Time{}, false
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return maskValue(email, 3)
	}
	local := parts[0]
	domain := parts[1]
	if local == "" {
		return "***@" + domain
	}
	return local[:1] + "***@" + domain
}

func maskValue(v string, keep int) string {
	if keep <= 0 || v == "" {
		return "***"
	}
	if len(v) <= keep {
		return v + "***"
	}
	return v[:keep] + "..."
}
