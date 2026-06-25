//go:build integration

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrendingPersonalizedByTheme(t *testing.T) {
	a, router := newTestApp(t)
	seedThemedPost(t, a, "bob", "le sport avant tout #alpha", "sport")
	seedThemedPost(t, a, "bob", "la cuisine du jour #beta", "cuisine")

	// alice aime le sport → #alpha (porté par un post sport) doit passer devant
	// #beta, à comptage égal.
	if err := a.db.Create(&userAffinityModel{UserID: "alice", Kind: affinityKindTheme, Ref: "sport", Weight: 10}).Error; err != nil {
		t.Fatalf("seed affinity: %v", err)
	}

	w := do(t, router, http.MethodGet, "/hashtags/trending", reqOpt{userID: "alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
	}
	data, _ := decode(t, w)["data"].([]any)
	if len(data) < 2 {
		t.Fatalf("expected 2 trends, got %d", len(data))
	}
	first, _ := data[0].(map[string]any)
	if first["name"] != "alpha" {
		t.Errorf("expected #alpha (sport) ranked first for a sport fan, got %v", first["name"])
	}
}

func TestSuggestedAuthorsByTheme(t *testing.T) {
	a, router := newTestApp(t)
	seedThemedPost(t, a, "carol", "recette du jour #x", "cuisine")
	seedThemedPost(t, a, "carol", "encore une recette #y", "cuisine")
	seedThemedPost(t, a, "dan", "debug sans fin #z", "tech")

	// alice aime la cuisine → carol (autrice cuisine) doit ressortir en tête.
	if err := a.db.Create(&userAffinityModel{UserID: "alice", Kind: affinityKindTheme, Ref: "cuisine", Weight: 8}).Error; err != nil {
		t.Fatalf("seed affinity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/internal/posts/suggested-authors?viewer=alice&limit=10", nil)
	req.Header.Set("X-Internal-Key", "test-internal-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
	}
	authors, _ := decode(t, w)["authors"].([]any)
	if len(authors) == 0 || authors[0] != "carol" {
		t.Errorf("expected carol (cuisine) ranked first, got %v", authors)
	}
}

func TestSuggestedAuthorsRequiresInternalKey(t *testing.T) {
	_, router := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/internal/posts/suggested-authors?viewer=alice", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 without internal key, got %d", w.Code)
	}
}
