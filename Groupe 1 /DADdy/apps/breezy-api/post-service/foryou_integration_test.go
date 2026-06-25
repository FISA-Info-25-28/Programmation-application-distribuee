//go:build integration

package main

import (
	"net/http"
	"testing"
)

// seedThemedPost insère un post avec un thème et ses hashtags, pour exercer le
// scoring du feed Pour Toi (affinités thème + hashtag + auteur).
func seedThemedPost(t *testing.T, a *app, author, content, theme string) int64 {
	t.Helper()
	th := theme
	p := postModel{Content: content, AuthorID: author, AuthorUsername: author, Theme: &th}
	if err := a.db.Create(&p).Error; err != nil {
		t.Fatalf("seed themed post: %v", err)
	}
	upsertHashtags(a.db, p.ID, extractHashtags(content))
	return p.ID
}

func TestForYouRanksByAffinity(t *testing.T) {
	a, router := newTestApp(t)
	seedThemedPost(t, a, "carol", "le #dev au quotidien", "tech")
	scubaID := seedThemedPost(t, a, "bob", "plongée à 20m #scuba", "plongee")

	// alice like le post de plongée → construit une préférence (thème plongee,
	// auteur bob, hashtag scuba) matérialisée dans user_affinity.
	if w := do(t, router, http.MethodPost, "/posts/"+itoa(scubaID)+"/likes", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("like status = %d (body %s)", w.Code, w.Body.String())
	}

	// Le feed Pour Toi d'alice doit faire remonter le post de plongée en tête.
	w := do(t, router, http.MethodGet, "/posts/for-you", reqOpt{userID: "alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
	}
	data, _ := decode(t, w)["data"].([]any)
	if len(data) == 0 {
		t.Fatalf("expected posts, got none")
	}
	first, _ := data[0].(map[string]any)
	if id, _ := first["id"].(float64); int64(id) != scubaID {
		t.Errorf("expected scuba post #%d ranked first, got id=%v", scubaID, first["id"])
	}
	// Le thème n'est pas exposé dans la réponse (signal back-only) : on vérifie le
	// classement, pas le champ theme.
}

func TestForYouFallsBackWithoutHistory(t *testing.T) {
	a, router := newTestApp(t)
	seedPost(t, a, "bob", "hello")

	// dave n'a aucun historique d'affinité → repli sur le feed global (200, 1 post).
	w := do(t, router, http.MethodGet, "/posts/for-you", reqOpt{userID: "dave"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
	}
	if total, _ := decode(t, w)["total"].(float64); total != 1 {
		t.Errorf("total = %v, want 1", total)
	}
}

func TestForYouAnonymousFallsBack(t *testing.T) {
	a, router := newTestApp(t)
	seedPost(t, a, "bob", "hello")

	// Visiteur anonyme (pas d'en-tête identité) → repli sur le feed global.
	w := do(t, router, http.MethodGet, "/posts/for-you", reqOpt{})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
	}
	if total, _ := decode(t, w)["total"].(float64); total != 1 {
		t.Errorf("total = %v, want 1", total)
	}
}

func TestInferThemeFromEngagers(t *testing.T) {
	a, router := newTestApp(t)
	// Post sans hashtag → thème indéterminé à la création.
	id := seedPost(t, a, "bob", "un texte tout simple sans aucun hashtag")

	// Trois « fans de plongée » (affinité thème plongee) interagissent avec lui.
	for _, u := range []string{"alice", "carol", "dan"} {
		if err := a.db.Create(&userAffinityModel{UserID: u, Kind: affinityKindTheme, Ref: "plongee", Weight: 5}).Error; err != nil {
			t.Fatalf("seed affinity %s: %v", u, err)
		}
		if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/likes", reqOpt{userID: u}); w.Code >= 300 {
			t.Fatalf("like %s: %d", u, w.Code)
		}
	}

	if got := inferThemeFromEngagers(a.db, id); got == nil || *got != "plongee" {
		t.Fatalf("inferThemeFromEngagers = %v, want plongee", got)
	}

	// La passe périodique doit étiqueter le post en base (theme_inferred = true).
	a.runThemeInferencePass()
	p := reloadPost(t, a, id)
	if p.Theme == nil || *p.Theme != "plongee" || !p.ThemeInferred {
		t.Errorf("post theme=%v inferred=%v, want plongee/true", p.Theme, p.ThemeInferred)
	}
}

func TestInferThemeNeedsEnoughEngagers(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "bob", "encore un texte sans hashtag")
	// Un seul interactant → sous le seuil de vérification croisée → pas de thème.
	if err := a.db.Create(&userAffinityModel{UserID: "alice", Kind: affinityKindTheme, Ref: "plongee", Weight: 5}).Error; err != nil {
		t.Fatalf("seed affinity: %v", err)
	}
	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("like: %d", w.Code)
	}
	if got := inferThemeFromEngagers(a.db, id); got != nil {
		t.Errorf("inferThemeFromEngagers = %v, want nil (signal insuffisant)", *got)
	}
}

func TestUnlikeDecrementsAffinity(t *testing.T) {
	a, router := newTestApp(t)
	id := seedThemedPost(t, a, "bob", "plongée #scuba", "plongee")

	// L'affinité-THÈME est désormais dérivée (cf. refreshThemeAffinity), pas bumpée
	// au like : on vérifie la dimension HASHTAG, qui l'est.
	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("like status = %d", w.Code)
	}
	var afterLike int64
	a.db.Model(&userAffinityModel{}).Where("user_id = ? AND kind = ? AND ref = ?", "alice", affinityKindHashtag, "scuba").Count(&afterLike)
	if afterLike != 1 {
		t.Fatalf("expected a scuba hashtag-affinity row after like, got %d", afterLike)
	}

	if w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("unlike status = %d", w.Code)
	}
	var weight float64
	a.db.Model(&userAffinityModel{}).
		Where("user_id = ? AND kind = ? AND ref = ?", "alice", affinityKindHashtag, "scuba").
		Select("COALESCE(SUM(weight), 0)").Scan(&weight)
	if weight != 0 {
		t.Errorf("expected scuba hashtag-affinity weight back to 0 after unlike, got %v", weight)
	}
}
