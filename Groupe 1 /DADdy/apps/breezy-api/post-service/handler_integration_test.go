//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

type reqOpt struct {
	userID string
	role   string
	body   string
}

func do(t *testing.T, router *gin.Engine, method, path string, opt reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(opt.body))
	if opt.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if opt.userID != "" {
		req.Header.Set(shared.HeaderUserID, opt.userID)
	}
	if opt.role != "" {
		req.Header.Set(shared.HeaderUserRole, opt.role)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return m
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestCreatePost(t *testing.T) {
	_, router := newTestApp(t)

	t.Run("requires identity", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/posts", reqOpt{body: `{"content":"hi"}`}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("creates a post and extracts hashtags", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{
			userID: "alice", body: `{"content":"hello #Breezy world","author_username":"alice"}`,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		data := decode(t, w)["data"].(map[string]any)
		if data["content"] != "hello #Breezy world" {
			t.Errorf("content = %v", data["content"])
		}
		tags, _ := data["hashtags"].([]any)
		if len(tags) != 1 || tags[0] != "breezy" {
			t.Errorf("hashtags = %v, want [breezy]", data["hashtags"])
		}
	})

	t.Run("empty content without media is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: `{"content":"   "}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("content over 280 chars is 400", func(t *testing.T) {
		long := strings.Repeat("a", 281)
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: `{"content":"` + long + `"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestLikeFlow(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "author", "hi")

	t.Run("like increments counter (201)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "liker"})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		if c := reloadPost(t, a, id).LikesCount; c != 1 {
			t.Errorf("LikesCount = %d, want 1", c)
		}
	})

	t.Run("double like is 409", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "liker"})
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		if c := reloadPost(t, a, id).LikesCount; c != 1 {
			t.Errorf("LikesCount = %d, want still 1", c)
		}
	})

	t.Run("unlike decrements (204)", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "liker"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		if c := reloadPost(t, a, id).LikesCount; c != 0 {
			t.Errorf("LikesCount = %d, want 0", c)
		}
	})

	t.Run("unlike when not liked is 404", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/likes", reqOpt{userID: "liker"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("like a missing post is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/999999/likes", reqOpt{userID: "liker"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("invalid post id is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/abc/likes", reqOpt{userID: "liker"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestRebreezeFlow(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "author", "hi")

	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/rebreezers", reqOpt{userID: "fan"}); w.Code != http.StatusCreated {
		t.Fatalf("rebreeze status = %d, want 201", w.Code)
	}
	if c := reloadPost(t, a, id).RebreezeCount; c != 1 {
		t.Errorf("RebreezeCount = %d, want 1", c)
	}
	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/rebreezers", reqOpt{userID: "fan"}); w.Code != http.StatusConflict {
		t.Fatalf("double rebreeze status = %d, want 409", w.Code)
	}
	if w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/rebreezers", reqOpt{userID: "fan"}); w.Code != http.StatusNoContent {
		t.Fatalf("unrebreeze status = %d, want 204", w.Code)
	}
	if c := reloadPost(t, a, id).RebreezeCount; c != 0 {
		t.Errorf("RebreezeCount = %d, want 0", c)
	}
}

func TestReplyFlow(t *testing.T) {
	a, router := newTestApp(t)
	parent := seedPost(t, a, "author", "parent")

	w := do(t, router, http.MethodPost, "/posts/"+itoa(parent)+"/replies", reqOpt{
		userID: "commenter", body: `{"content":"nice post","author_username":"commenter"}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("reply status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	if c := reloadPost(t, a, parent).CommentsCount; c != 1 {
		t.Errorf("parent CommentsCount = %d, want 1", c)
	}

	// La réponse doit apparaître dans la liste des replies.
	lw := do(t, router, http.MethodGet, "/posts/"+itoa(parent)+"/replies", reqOpt{userID: "x"})
	if data, ok := decode(t, lw)["data"].([]any); !ok || len(data) != 1 {
		t.Errorf("replies length = %v, want 1", decode(t, lw)["data"])
	}
}

func TestDeletePostOwnership(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "author", "mine")

	t.Run("non-owner non-staff is 403", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id), reqOpt{userID: "intruder", role: shared.RoleUser})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("moderator can delete anyone's post", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id), reqOpt{userID: "mod", role: shared.RoleModerator})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("owner can delete own post", func(t *testing.T) {
		own := seedPost(t, a, "author", "again")
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(own), reqOpt{userID: "author", role: shared.RoleUser})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		var count int64
		a.db.Model(&postModel{}).Where("id = ?", own).Count(&count)
		if count != 0 {
			t.Error("post should be gone after delete")
		}
	})
}

func TestBookmarkFlow(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "author", "hi")

	if w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/bookmarks", reqOpt{userID: "alice"}); w.Code != http.StatusCreated {
		t.Fatalf("bookmark status = %d, want 201", w.Code)
	}
	// La liste des bookmarks de l'appelant contient le post.
	lw := do(t, router, http.MethodGet, "/users/alice/bookmarks", reqOpt{userID: "alice"})
	if lw.Code != http.StatusOK {
		t.Fatalf("list bookmarks status = %d, want 200", lw.Code)
	}
	if data, ok := decode(t, lw)["data"].([]any); !ok || len(data) != 1 {
		t.Errorf("bookmarks length = %v, want 1", decode(t, lw)["data"])
	}
	if w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/bookmarks", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
		t.Fatalf("unbookmark status = %d, want 204", w.Code)
	}
}

func TestReportFlow(t *testing.T) {
	a, router := newTestApp(t)
	id := seedPost(t, a, "author", "offensive?")

	t.Run("cannot report your own post", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/report", reqOpt{userID: "author", body: `{"reason":"x"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("empty reason is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/report", reqOpt{userID: "reporter", body: `{"reason":"  "}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("valid report (201) then duplicate (409)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/report", reqOpt{userID: "reporter", body: `{"reason":"spam"}`})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		dup := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/report", reqOpt{userID: "reporter", body: `{"reason":"spam"}`})
		if dup.Code != http.StatusConflict {
			t.Fatalf("duplicate status = %d, want 409", dup.Code)
		}
	})

	t.Run("listing reports requires staff", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/reports", reqOpt{userID: "reporter", role: shared.RoleUser}); w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
		if w := do(t, router, http.MethodGet, "/reports", reqOpt{userID: "mod", role: shared.RoleModerator}); w.Code != http.StatusOK {
			t.Fatalf("staff status = %d, want 200", w.Code)
		}
	})
}

func TestVotePoll(t *testing.T) {
	_, router := newTestApp(t)

	// Crée un post avec sondage via l'API (assure la cohérence poll/options).
	cw := do(t, router, http.MethodPost, "/posts", reqOpt{
		userID: "author",
		body:   `{"content":"pick one","author_username":"author","poll":{"options":["A","B"],"duration_days":1}}`,
	})
	if cw.Code != http.StatusCreated {
		t.Fatalf("create poll post status = %d, want 201 (body %s)", cw.Code, cw.Body.String())
	}
	postID := int64(decode(t, cw)["data"].(map[string]any)["id"].(float64))

	t.Run("author cannot vote on own poll", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/poll/vote", reqOpt{userID: "author", body: `{"option_index":0}`})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("voter casts a vote then double vote is 409", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/poll/vote", reqOpt{userID: "voter", body: `{"option_index":0}`})
		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Fatalf("vote status = %d, want 200/201 (body %s)", w.Code, w.Body.String())
		}
		dup := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/poll/vote", reqOpt{userID: "voter", body: `{"option_index":1}`})
		if dup.Code != http.StatusConflict {
			t.Fatalf("double vote status = %d, want 409", dup.Code)
		}
	})

	t.Run("invalid option index is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/poll/vote", reqOpt{userID: "voter2", body: `{"option_index":99}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
