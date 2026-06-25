//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestPostHandlerErrorBranches exerce les branches d'erreur des handlers
// (corps invalide, identifiants invalides, ressources absentes) afin de couvrir
// les chemins non atteints par les flux nominaux.
func TestPostHandlerErrorBranches(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "author", "hi")

	t.Run("create post invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("create post with over 4 media is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: `{"content":"x","media_ids":[1,2,3,4,5]}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("create post quoting a missing post is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: `{"content":"quote","quote_id":999999}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("create post combining poll and media is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice",
			body: `{"content":"x","media_ids":[1],"poll":{"options":["a","b"]}}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("reply invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/replies", reqOpt{userID: "alice", body: `{bad`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("reply to missing post is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/999999/replies", reqOpt{userID: "alice", body: `{"content":"hi"}`})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("reply with invalid id is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/abc/replies", reqOpt{userID: "alice", body: `{"content":"hi"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("get missing post is 404", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/posts/999999", reqOpt{userID: "alice"}); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("bookmark missing post is 404", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/posts/999999/bookmarks", reqOpt{userID: "alice"}); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("unbookmark when not bookmarked is 404", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/bookmarks", reqOpt{userID: "alice"}); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("vote on a post without poll is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/poll/vote", reqOpt{userID: "voter", body: `{"option_index":0}`})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("vote invalid body is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/poll/vote", reqOpt{userID: "voter", body: `{bad`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("delete missing post is 404", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/999999", reqOpt{userID: "author"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

// TestPostReadListings couvre les endpoints de listing peu sollicités.
func TestPostReadListings(t *testing.T) {
	a, router := newTestApp(t)
	pid := seedPost(t, a, "author", "hello #trend world")
	// Un hashtag pour les endpoints trending / by-hashtag.
	if w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "author", body: `{"content":"more #trend","author_username":"author"}`}); w.Code != http.StatusCreated {
		t.Fatalf("seed hashtag post status = %d", w.Code)
	}

	t.Run("replies of a post", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/posts/"+itoa(pid)+"/replies", reqOpt{userID: "x"}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("trending hashtags", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/hashtags/trending", reqOpt{}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("posts by hashtag", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/hashtags/trend/posts", reqOpt{userID: "x"}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("user posts listing", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/users/author/posts", reqOpt{userID: "author"}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("user liked listing", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/users/author/liked", reqOpt{userID: "author"}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("user rebreezed listing", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/users/author/rebreezed", reqOpt{userID: "author"}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("search with query", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/posts/search?q=hello", reqOpt{userID: "x"}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}
