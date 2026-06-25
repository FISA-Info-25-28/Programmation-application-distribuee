//go:build integration

package main

import (
	"net/http"
	"testing"
)

// Sans POST_SERVICE_URL configurée (cas des tests), /users/suggestions bascule en
// repli : comptes récents actifs, en excluant soi et les comptes déjà suivis.
func TestSuggestionsFallbackExcludesSelfAndFollowed(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)
	seedUser(t, s, "carol", "carol", false)

	if err := s.db.Create(&followModel{FollowerID: "alice", FolloweeID: "bob"}).Error; err != nil {
		t.Fatalf("seed follow: %v", err)
	}

	w := do(t, router, http.MethodGet, "/users/suggestions", reqOpt{userID: "alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
	}

	data, _ := decode(t, w)["data"].([]any)
	foundCarol := false
	for _, item := range data {
		m, _ := item.(map[string]any)
		switch m["id"] {
		case "alice":
			t.Error("suggestions must not include self")
		case "bob":
			t.Error("suggestions must not include already-followed accounts")
		case "carol":
			foundCarol = true
		}
	}
	if !foundCarol {
		t.Error("expected carol (not followed) in fallback suggestions")
	}
}

func TestSuggestionsRequiresIdentity(t *testing.T) {
	router, _ := newTestRouter(t)
	if w := do(t, router, http.MethodGet, "/users/suggestions", reqOpt{}); w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without identity, got %d", w.Code)
	}
}
