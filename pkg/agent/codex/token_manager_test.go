package codex

import (
	"testing"
	"time"
)

func TestSessionTokenManager_IssueReusesAppServerToken(t *testing.T) {
	now := time.Now().UTC()
	manager := NewSessionTokenManager(0)
	manager.nowFn = func() time.Time { return now }

	first, err := manager.Issue()
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	second, err := manager.Issue()
	if err != nil {
		t.Fatalf("second Issue returned error: %v", err)
	}
	if first != second {
		t.Fatalf("Issue should reuse the same token; got %q then %q", first, second)
	}

	scope, err := manager.Validate(first)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if scope != defaultSessionTokenScope {
		t.Fatalf("Validate scope = %q, want %q", scope, defaultSessionTokenScope)
	}
}

func TestSessionTokenManager_RotatesExpiredToken(t *testing.T) {
	now := time.Now().UTC()
	manager := NewSessionTokenManager(2 * time.Second)
	manager.nowFn = func() time.Time { return now }

	first, err := manager.Issue()
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	now = now.Add(3 * time.Second)
	if _, err := manager.Validate(first); err == nil {
		t.Fatalf("Validate should fail for expired token")
	}

	second, err := manager.Issue()
	if err != nil {
		t.Fatalf("Issue after expiration returned error: %v", err)
	}
	if second == first {
		t.Fatalf("Issue should rotate expired tokens; got %q twice", second)
	}
}
