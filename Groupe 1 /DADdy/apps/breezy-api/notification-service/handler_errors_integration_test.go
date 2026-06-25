//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestNotifHandlerErrorBranches couvre les branches d'erreur résiduelles des
// handlers (identifiants invalides, corps manquant).
func TestNotifHandlerErrorBranches(t *testing.T) {
	_, router := newTestApp(t)

	t.Run("delete with invalid id is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/notifications/abc", reqOpt{userID: "alice"}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("subscribe with invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/subscriptions", reqOpt{userID: "alice", body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("subscribe with missing target is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/subscriptions", reqOpt{userID: "alice", body: `{}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
