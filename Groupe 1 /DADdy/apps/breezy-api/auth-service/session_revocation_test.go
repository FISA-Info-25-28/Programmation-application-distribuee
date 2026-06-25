package main

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRevokeUserSessionsNilNoOp(t *testing.T) {
	t.Parallel()
	// Nil receiver must not panic — used when ValkeyAddr is empty.
	var s *sessionRevoker
	s.revokeUserSessions("user123")
}

func TestRevokeUserSessionsWriteFailureDoesNotPanic(t *testing.T) {
	t.Parallel()
	// Port 1 is always refused; the write will fail.
	// The error must be logged, not propagated — the function must not panic.
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	s := &sessionRevoker{rdb: rdb, ttl: time.Minute}
	s.revokeUserSessions("user123")
}

func TestRevocationKeyPrefix(t *testing.T) {
	t.Parallel()
	// The gateway reads the same prefix; ensure it stays stable.
	const want = "authrevoke:"
	if revocationKeyPrefix != want {
		t.Fatalf("revocationKeyPrefix = %q, want %q", revocationKeyPrefix, want)
	}
}
