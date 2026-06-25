//go:build integration

package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestHTTPOAuthRedirect(t *testing.T) {
	router, s := newTestRouter(t)

	t.Run("disabled provider is 404", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/auth/oauth/google", reqOpt{})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("enabled provider redirects with state cookie", func(t *testing.T) {
		s.cfg.GoogleClientID = "gid"
		w := do(t, router, http.MethodGet, "/auth/oauth/google", reqOpt{})
		if w.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", w.Code)
		}
		if loc := w.Header().Get("Location"); loc == "" {
			t.Error("expected a redirect Location to the provider")
		}
		if cookie := w.Header().Get("Set-Cookie"); cookie == "" {
			t.Error("expected an oauth_state cookie")
		}
	})
}

func TestHTTPOAuthCallbackErrors(t *testing.T) {
	router, _ := newTestRouter(t)

	t.Run("missing state cookie redirects to error", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/auth/oauth/google/callback?state=x&code=y", reqOpt{})
		if w.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", w.Code)
		}
		// Doit pointer sur la route SPA existante /auth/oauth/callback avec un
		// `reason` (OAuthCallbackPage le relaie vers /login?oauthError=…). Une
		// route /auth/oauth/error n'existe pas côté front → le motif serait perdu.
		if loc := w.Header().Get("Location"); !strings.Contains(loc, "/auth/oauth/callback?reason=state") {
			t.Errorf("Location = %q, want it to contain /auth/oauth/callback?reason=state", loc)
		}
	})
}

func TestHTTPOAuthExchange(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "oex", "oex@example.com", strongPassword, accountStatusActive)

	t.Run("invalid body is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/oauth/exchange", reqOpt{body: `{bad`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("empty code is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/oauth/exchange", reqOpt{body: `{"code":""}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unknown code is 401", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/oauth/exchange", reqOpt{body: `{"code":"ghost"}`})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("valid code returns tokens", func(t *testing.T) {
		code, err := s.issueOAuthExchangeCode(uid)
		if err != nil {
			t.Fatalf("issueOAuthExchangeCode: %v", err)
		}
		w := do(t, router, http.MethodPost, "/auth/oauth/exchange", reqOpt{body: `{"code":"` + code + `"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		if decode(t, w)["accessToken"] == nil {
			t.Error("expected an accessToken in the response")
		}
	})
}
