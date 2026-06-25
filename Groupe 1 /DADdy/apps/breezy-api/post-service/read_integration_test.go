//go:build integration

package main

import (
	"net/http"
	"testing"
)

func TestGlobalFeed(t *testing.T) {
	a, router := newTestApp(t)
	seedPost(t, a, "alice", "first")
	seedPost(t, a, "alice", "second")

	w := do(t, router, http.MethodGet, "/posts", reqOpt{})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	if total, _ := decode(t, w)["total"].(float64); total != 2 {
		t.Errorf("total = %v, want 2", decode(t, w)["total"])
	}
}

func TestSearchPosts(t *testing.T) {
	a, router := newTestApp(t)
	seedPost(t, a, "alice", "golang is great")
	seedPost(t, a, "alice", "unrelated thoughts")

	t.Run("missing q is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/posts/search", reqOpt{}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("matches content", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/search?q=golang", reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		data, _ := decode(t, w)["data"].([]any)
		if len(data) != 1 {
			t.Errorf("results = %d, want 1", len(data))
		}
	})
}

func TestGetPost(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "alice", "hello")

	t.Run("invalid id is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/posts/abc", reqOpt{}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing post is 404", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/posts/999999", reqOpt{}); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("found", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/"+itoa(id), reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		post, _ := decode(t, w)["data"].(map[string]any)
		if post["content"] != "hello" {
			t.Errorf("content = %v, want hello", post["content"])
		}
	})
}

func TestUserPosts(t *testing.T) {
	a, router := newTestApp(t)
	seedPost(t, a, "alice", "a1")
	seedPost(t, a, "alice", "a2")
	seedPost(t, a, "bob", "b1")

	w := do(t, router, http.MethodGet, "/users/alice/posts", reqOpt{})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if total, _ := decode(t, w)["total"].(float64); total != 2 {
		t.Errorf("total = %v, want 2", decode(t, w)["total"])
	}
}

func TestUserLikedAndRebreezed(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "bob", "likeable")

	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("like status = %d (body %s)", w.Code, w.Body.String())
	}
	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/rebreezers", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("rebreeze status = %d (body %s)", w.Code, w.Body.String())
	}

	t.Run("liked", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/alice/liked", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("liked total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("rebreezed", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/alice/rebreezed", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("rebreezed total = %v, want 1", decode(t, w)["total"])
		}
	})
}

func TestUserBookmarks(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "bob", "bookmarkable")

	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/bookmarks", reqOpt{userID: "alice"}); w.Code >= 300 {
		t.Fatalf("bookmark status = %d (body %s)", w.Code, w.Body.String())
	}

	t.Run("owner sees own bookmarks", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/alice/bookmarks", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("bookmarks total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("cannot read another user's bookmarks", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/alice/bookmarks", reqOpt{userID: "bob"})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})
}

func TestListReplies(t *testing.T) {
	a, router := newTestApp(t)
	parentID := seedPost(t, a, "alice", "parent")

	if w := do(t, router, http.MethodPost, "/posts/"+itoa(parentID)+"/replies", reqOpt{
		userID: "bob", body: `{"content":"a reply"}`,
	}); w.Code >= 300 {
		t.Fatalf("reply status = %d (body %s)", w.Code, w.Body.String())
	}

	w := do(t, router, http.MethodGet, "/posts/"+itoa(parentID)+"/replies", reqOpt{})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	data, _ := decode(t, w)["data"].([]any)
	if len(data) != 1 {
		t.Errorf("replies = %d, want 1", len(data))
	}
}

func TestHashtagEndpoints(t *testing.T) {
	_, router := newTestApp(t)

	// createPost extrait les hashtags du contenu.
	if w := do(t, router, http.MethodPost, "/posts", reqOpt{
		userID: "alice", body: `{"content":"loving #breezy today","author_username":"alice"}`,
	}); w.Code != http.StatusCreated {
		t.Fatalf("create status = %d (body %s)", w.Code, w.Body.String())
	}

	t.Run("posts by hashtag", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/hashtags/breezy/posts", reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("trending hashtags", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/hashtags/trending", reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		data, _ := decode(t, w)["data"].([]any)
		if len(data) < 1 {
			t.Errorf("trending should list at least one hashtag, got %v", data)
		}
	})
}
