package codex

import (
	"testing"
	"time"
)

func TestSessionTokenManager_IssueValidateAndExpire(t *testing.T) {
	now := time.Now().UTC()
	manager := NewSessionTokenManager(2 * time.Second)
	manager.nowFn = func() time.Time { return now }

	token, err := manager.Issue("session-1")
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	gotSessionID, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if gotSessionID != "session-1" {
		t.Fatalf("Validate sessionID = %q, want %q", gotSessionID, "session-1")
	}

	now = now.Add(3 * time.Second)
	if _, err := manager.Validate(token); err == nil {
		t.Fatalf("Validate should fail for expired token")
	}
}
