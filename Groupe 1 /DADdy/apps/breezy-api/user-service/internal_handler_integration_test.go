//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPBatchAvatars(t *testing.T) {
	router, s := newInternalRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)

	t.Run("happy path returns avatar entries", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/batch-avatars", testInternalKey,
			`{"ids":["alice","bob"]}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		var resp struct {
			Data []struct {
				ID       string  `json:"id"`
				Username string  `json:"username"`
				Avatar   *string `json:"avatarUrl"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data) != 2 {
			t.Errorf("data length = %d, want 2", len(resp.Data))
		}
	})

	t.Run("empty ids list returns empty array", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/batch-avatars", testInternalKey,
			`{"ids":[]}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "[]") {
			t.Errorf("expected empty array in body, got: %s", w.Body.String())
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/batch-avatars", testInternalKey, `{bad}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing key returns 403", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/batch-avatars", "", `{"ids":["alice"]}`)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})
}

func TestHTTPResolveMentionsEdgeCases(t *testing.T) {
	router, s := newInternalRouter(t)
	seedUser(t, s, "alice", "alice", false)

	t.Run("missing usernames field returns 400", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/resolve-mentions", testInternalKey, `{}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("empty usernames list returns empty array", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/resolve-mentions", testInternalKey,
			`{"usernames":[]}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "[]") {
			t.Errorf("expected empty array, got: %s", w.Body.String())
		}
	})

	t.Run("unknown usernames are filtered out", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/resolve-mentions", testInternalKey,
			`{"usernames":["alice","nobody"]}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data, _ := resp["data"].([]any)
		if len(data) != 1 {
			t.Errorf("resolved = %d, want 1 (only alice)", len(data))
		}
	})
}

func TestHTTPCreateInternalUserEdgeCases(t *testing.T) {
	router, _ := newInternalRouter(t)

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users", testInternalKey, `{bad}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestHTTPLargePageFollowers(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)

	// Very large page should return 200 with an empty result set, not overflow.
	w := do(t, router, http.MethodGet, "/users/alice/followers?page=999999999&limit=20",
		reqOpt{userID: "alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("large page GET /followers = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data for out-of-range page, got %d items", len(data))
	}
}
