package codex

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

const defaultSessionTokenTTL = 10 * time.Minute

var (
	errSessionTokenInvalid = errors.New("invalid session token")
	errSessionTokenExpired = errors.New("expired session token")
)

type sessionToken struct {
	sessionID string
	expiresAt time.Time
}

// SessionTokenManager mints and validates short-lived bearer tokens scoped to a codex session.
type SessionTokenManager struct {
	mu     sync.RWMutex
	tokens map[string]sessionToken
	nowFn  func() time.Time
	ttl    time.Duration
}

func NewSessionTokenManager(ttl time.Duration) *SessionTokenManager {
	if ttl <= 0 {
		ttl = defaultSessionTokenTTL
	}
	return &SessionTokenManager{
		tokens: make(map[string]sessionToken, 8),
		nowFn:  time.Now,
		ttl:    ttl,
	}
}

func (m *SessionTokenManager) Issue(sessionID string) (string, error) {
	if sessionID == "" {
		return "", errors.New("session id must not be empty")
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	now := m.nowFn()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)
	m.tokens[token] = sessionToken{
		sessionID: sessionID,
		expiresAt: now.Add(m.ttl),
	}
	return token, nil
}

func (m *SessionTokenManager) Validate(token string) (string, error) {
	now := m.nowFn()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gcLocked(now)

	record, ok := m.tokens[token]
	if !ok {
		return "", errSessionTokenInvalid
	}
	if now.After(record.expiresAt) {
		delete(m.tokens, token)
		return "", errSessionTokenExpired
	}
	return record.sessionID, nil
}

func (m *SessionTokenManager) gcLocked(now time.Time) {
	for token, record := range m.tokens {
		if now.After(record.expiresAt) {
			delete(m.tokens, token)
		}
	}
}
