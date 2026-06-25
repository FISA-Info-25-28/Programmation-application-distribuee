//go:build integration

package main

import (
	"net/http"
	"testing"

	"daddy/apps/breezy-api/internal/shared"
)

// TestHTTPUserHandlerErrorBranches couvre les branches d'erreur des handlers
// non atteintes par les flux nominaux (auto-action interdite, corps invalide,
// cibles inexistantes).
func TestHTTPUserHandlerErrorBranches(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "target", "target", false)

	t.Run("unfollow self is 400", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/alice/follow", reqOpt{userID: "alice"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unblock self is 400", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/alice/block", reqOpt{userID: "alice"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unblock a non-blocked user is idempotent 204", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/target/block", reqOpt{userID: "alice"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("sanction invalid body is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/target/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator, body: `{bad`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("sanction unknown user is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/ghost/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"type":"ban"}`})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("lift unknown user is 404", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/ghost/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("lift existing user with no active sanction is idempotent 204", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/target/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("list sanctions unknown user is 404", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/ghost/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}
