package codex

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

const defaultSessionTokenScope = "codex-app-server"

var (
	errSessionTokenInvalid = errors.New("invalid session token")
	errSessionTokenExpired = errors.New("expired session token")
)

type issuedToken struct {
	value     string
	expiresAt time.Time
}

// SessionTokenManager mints and validates MCP bearer tokens for a codex app-server instance.
// The same token is reused across requests so Codex does not need token refresh or per-thread auth state.
type SessionTokenManager struct {
	mu    sync.Mutex
	token issuedToken
	nowFn func() time.Time
	ttl   time.Duration
	scope string
}

func NewSessionTokenManager(ttl time.Duration) *SessionTokenManager {
	return &SessionTokenManager{
		nowFn: time.Now,
		ttl:   ttl,
		scope: defaultSessionTokenScope,
	}
}

func (m *SessionTokenManager) Issue() (string, error) {
	now := m.nowFn()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.token.value != "" && (m.ttl <= 0 || now.Before(m.token.expiresAt)) {
		return m.token.value, nil
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	m.token = issuedToken{value: token}
	if m.ttl > 0 {
		m.token.expiresAt = now.Add(m.ttl)
	}
	return token, nil
}

func (m *SessionTokenManager) Validate(token string) (string, error) {
	now := m.nowFn()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.token.value == "" || token != m.token.value {
		return "", errSessionTokenInvalid
	}
	if m.ttl > 0 && !m.token.expiresAt.IsZero() && now.After(m.token.expiresAt) {
		m.token = issuedToken{}
		return "", errSessionTokenExpired
	}
	return m.scope, nil
}
