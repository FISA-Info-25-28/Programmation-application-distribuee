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
	userID      string
	role        string
	internalKey string
	body        string
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
	if opt.internalKey != "" {
		req.Header.Set("X-Internal-Key", opt.internalKey)
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

func TestHealth(t *testing.T) {
	_, router := newTestServer(t)
	w := do(t, router, http.MethodGet, "/health", reqOpt{})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if decode(t, w)["service"] != "core-api" {
		t.Errorf("service = %v, want core-api", decode(t, w)["service"])
	}
}

func TestInternalUsers(t *testing.T) {
	_, router := newTestServer(t)

	t.Run("forbidden without key", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/users", reqOpt{body: `{"id":"usr_1"}`})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("bad body with key", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/users", reqOpt{internalKey: "test-internal-key", body: `{}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("creates user", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/users", reqOpt{
			internalKey: "test-internal-key",
			body:        `{"id":"usr_1","username":"alice","email":"alice@example.com"}`,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		if decode(t, w)["id"] != "usr_1" {
			t.Errorf("id = %v", decode(t, w)["id"])
		}
	})
}

func TestCreatePost(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")

	t.Run("requires identity", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{body: `{"content":"hi"}`})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "usr_1", body: `not json`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "usr_1", body: `{"content":"   "}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("content over 280", func(t *testing.T) {
		long := strings.Repeat("a", 281)
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "usr_1", body: `{"content":"` + long + `"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unknown author => 422", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "ghost", body: `{"content":"hi"}`})
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
	})

	t.Run("creates with tags", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts", reqOpt{
			userID: "usr_1", body: `{"content":"hello world","tags":["Go","  ","go"]}`,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		var resp postResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Content != "hello world" || resp.Author.Username != "alice" {
			t.Errorf("resp = %+v", resp)
		}
		// "Go" et "go" se normalisent en un seul tag "go"; le blanc est ignoré.
		if len(resp.Tags) != 1 || resp.Tags[0] != "go" {
			t.Errorf("tags = %v, want [go]", resp.Tags)
		}
	})
}

func TestGetPost(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	id := seedPost(t, s, "usr_1", "hello")

	t.Run("invalid id", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/abc", reqOpt{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/9999", reqOpt{})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("found", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/"+itoa(id), reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp postResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.ID != itoa(id) || resp.Content != "hello" {
			t.Errorf("resp = %+v", resp)
		}
	})
}

func TestDeletePost(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	seedUser(t, s, "usr_2", "bob")

	t.Run("requires identity", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/1", reqOpt{})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/9999", reqOpt{userID: "usr_1"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("forbidden for non-owner non-staff", func(t *testing.T) {
		id := seedPost(t, s, "usr_1", "mine")
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id), reqOpt{userID: "usr_2", role: "user"})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("owner deletes", func(t *testing.T) {
		id := seedPost(t, s, "usr_1", "mine")
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id), reqOpt{userID: "usr_1"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("moderator deletes others", func(t *testing.T) {
		id := seedPost(t, s, "usr_1", "mine")
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id), reqOpt{userID: "usr_2", role: "moderator"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestLikeUnlikePost(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	id := seedPost(t, s, "usr_1", "hello")

	t.Run("like requires identity", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/like", reqOpt{})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("like increments once (idempotent)", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/like", reqOpt{userID: "usr_1"})
			if w.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204", w.Code)
			}
		}
		if got := reloadPost(t, s, id).LikesCount; got != 1 {
			t.Fatalf("likes_count = %d, want 1", got)
		}
	})

	t.Run("unlike decrements", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/posts/"+itoa(id)+"/like", reqOpt{userID: "usr_1"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		if got := reloadPost(t, s, id).LikesCount; got != 0 {
			t.Fatalf("likes_count = %d, want 0", got)
		}
	})
}

func TestComments(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	id := seedPost(t, s, "usr_1", "hello")

	t.Run("create requires identity", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/comments", reqOpt{body: `{"content":"hi"}`})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("empty content rejected", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/comments", reqOpt{userID: "usr_1", body: `{"content":"  "}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("create and list", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/comments", reqOpt{userID: "usr_1", body: `{"content":"nice post"}`})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		if got := reloadPost(t, s, id).CommentsCount; got != 1 {
			t.Errorf("comments_count = %d, want 1", got)
		}

		lw := do(t, router, http.MethodGet, "/posts/"+itoa(id)+"/comments", reqOpt{})
		if lw.Code != http.StatusOK {
			t.Fatalf("list status = %d, want 200", lw.Code)
		}
		data, _ := decode(t, lw)["data"].([]any)
		if len(data) != 1 {
			t.Errorf("comments = %d, want 1", len(data))
		}
	})

	t.Run("list invalid post id", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/abc/comments", reqOpt{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestFeedSearchUserPosts(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	seedPost(t, s, "usr_1", "golang rocks")
	seedPost(t, s, "usr_1", "another post")

	t.Run("feed lists root posts", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/feed", reqOpt{userID: "usr_1"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 2 {
			t.Errorf("total = %v, want 2", decode(t, w)["total"])
		}
	})

	t.Run("search requires q", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/search", reqOpt{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("search matches content", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/posts/search?q=golang", reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		data, _ := decode(t, w)["data"].([]any)
		if len(data) != 1 {
			t.Errorf("results = %d, want 1", len(data))
		}
	})

	t.Run("user posts", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/usr_1/posts", reqOpt{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 2 {
			t.Errorf("total = %v, want 2", decode(t, w)["total"])
		}
	})
}
