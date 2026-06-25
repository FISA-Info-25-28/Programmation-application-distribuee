//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestHTTPHandlerErrorBranches exerce les branches d'erreur des handlers
// (corps invalide, identité manquante, états MFA incohérents) afin de couvrir
// les chemins non atteints par les tests de happy-path.
func TestHTTPHandlerErrorBranches(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "err", "err@example.com", strongPassword, accountStatusActive)

	t.Run("refresh invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("logout invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/logout", reqOpt{body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("logout empty token is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/logout", reqOpt{body: `{"refreshToken":""}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("change-password invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{userID: uid, body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("mfa confirm without identity is 401", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{body: `{"code":"123456"}`}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("mfa confirm invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{userID: uid, body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("mfa confirm before setup is 400 (not initiated)", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{userID: uid, body: `{"code":"123456"}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("mfa disable without identity is 401", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/auth/mfa", reqOpt{body: `{"code":"123456"}`}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("mfa disable invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/auth/mfa", reqOpt{userID: uid, body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("mfa disable when not enabled is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/auth/mfa", reqOpt{userID: uid, body: `{"code":"123456"}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("mfa verify invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/mfa/verify", reqOpt{body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("mfa verify unknown pending token is 401", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/mfa/verify", reqOpt{body: `{"pendingToken":"ghost","code":"123456"}`})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("me with unknown identity is 401", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/auth/me", reqOpt{userID: "ghost-user"}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

// TestHTTPMFASetupConflict couvre le 409 quand le MFA est déjà activé.
func TestHTTPMFASetupConflict(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "mc", "mc@example.com", strongPassword, accountStatusActive)

	// setup + confirm pour activer le MFA.
	w := do(t, router, http.MethodPost, "/auth/mfa/setup", reqOpt{userID: uid})
	if w.Code != http.StatusOK {
		t.Fatalf("setup status = %d, want 200", w.Code)
	}
	secret, _ := decode(t, w)["secret"].(string)
	if cw := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{userID: uid, body: `{"code":"` + currentTOTP(t, secret) + `"}`}); cw.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200", cw.Code)
	}

	// Un nouveau setup doit échouer en 409 (déjà activé).
	if again := do(t, router, http.MethodPost, "/auth/mfa/setup", reqOpt{userID: uid}); again.Code != http.StatusConflict {
		t.Fatalf("second setup status = %d, want 409", again.Code)
	}

	// confirm sur un MFA déjà activé doit aussi renvoyer 409.
	if cw := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{userID: uid, body: `{"code":"` + currentTOTP(t, secret) + `"}`}); cw.Code != http.StatusConflict {
		t.Fatalf("confirm-again status = %d, want 409", cw.Code)
	}
}
