//go:build integration

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

const testInternalKey = "test-internal-key"

// newInternalRouter renvoie un routeur dont le service a une clé interne
// configurée (newTestService laisse la clé vide → routes internes en 403).
func newInternalRouter(t *testing.T) (*gin.Engine, *userService) {
	t.Helper()
	base := newTestService(t)
	s := newUserService(base.db, nil, userConfig{InternalKey: testInternalKey})
	return newUserRouter(s), s
}

// doKey exécute une requête avec l'en-tête X-Internal-Key.
func doKey(t *testing.T, router *gin.Engine, method, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("X-Internal-Key", key)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestHTTPFollowRequestsFlow(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "priv", "priv", true)

	// alice demande à suivre un compte privé → demande en attente.
	if w := do(t, router, http.MethodPost, "/users/priv/follow", reqOpt{userID: "alice"}); w.Code != http.StatusAccepted {
		t.Fatalf("follow status = %d, want 202", w.Code)
	}

	t.Run("owner lists pending requests", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/me/follow-requests", reqOpt{userID: "priv"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("accept missing request is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/me/follow-requests/ghost/accept", reqOpt{userID: "priv"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("accept then list is empty", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/users/me/follow-requests/alice/accept", reqOpt{userID: "priv"}); w.Code != http.StatusNoContent {
			t.Fatalf("accept status = %d, want 204", w.Code)
		}
		w := do(t, router, http.MethodGet, "/users/me/follow-requests", reqOpt{userID: "priv"})
		if total, _ := decode(t, w)["total"].(float64); total != 0 {
			t.Errorf("total after accept = %v, want 0", decode(t, w)["total"])
		}
	})

	t.Run("reject is idempotent 204", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/me/follow-requests/whoever", reqOpt{userID: "priv"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestHTTPFollowingFollowers(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)

	if w := do(t, router, http.MethodPost, "/users/bob/follow", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
		t.Fatalf("follow status = %d, want 204", w.Code)
	}

	t.Run("following list", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/alice/following", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("following total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("followers list", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/bob/followers", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("followers total = %v, want 1", decode(t, w)["total"])
		}
	})
}

func TestHTTPBlockListsAndUnblock(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)

	if w := do(t, router, http.MethodPost, "/users/bob/block", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
		t.Fatalf("block status = %d, want 204", w.Code)
	}

	t.Run("blocker sees blocked list", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/me/blocked", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("blocked total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("blocked user sees blocked-by list", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/me/blocked-by", reqOpt{userID: "bob"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if total, _ := decode(t, w)["total"].(float64); total != 1 {
			t.Errorf("blocked-by total = %v, want 1", decode(t, w)["total"])
		}
	})

	t.Run("unblock returns 204", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/bob/block", reqOpt{userID: "alice"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestHTTPListSanctions(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "mod", "mod", false)
	seedUser(t, s, "target", "target", false)

	if w := do(t, router, http.MethodPost, "/users/target/sanctions",
		reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"type":"ban","reason":"spam"}`}); w.Code != http.StatusCreated {
		t.Fatalf("sanction status = %d, want 201", w.Code)
	}

	t.Run("staff lists sanctions", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/target/sanctions", reqOpt{userID: "mod", role: shared.RoleModerator})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		data, _ := decode(t, w)["data"].([]any)
		if len(data) != 1 {
			t.Errorf("sanctions = %d, want 1", len(data))
		}
	})

	t.Run("regular user is forbidden", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/target/sanctions", reqOpt{userID: "mod", role: shared.RoleUser})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})
}

func TestHTTPInternalRoutes(t *testing.T) {
	router, s := newInternalRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)
	seedUser(t, s, "priv", "priv", true)

	t.Run("forbidden without key", func(t *testing.T) {
		w := doKey(t, router, http.MethodGet, "/internal/users/hidden-authors?viewer=alice", "", "")
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("forbidden with wrong key", func(t *testing.T) {
		w := doKey(t, router, http.MethodGet, "/internal/users/hidden-authors?viewer=alice", "wrong", "")
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("create internal user", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users", testInternalKey,
			`{"id":"carol","username":"carol","email":"carol@example.com","role":"user"}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		if _, err := s.getUserByUsername("carol"); err != nil {
			t.Errorf("created user not found: %v", err)
		}
	})

	t.Run("can-view public user", func(t *testing.T) {
		w := doKey(t, router, http.MethodGet, "/internal/users/bob/can-view?viewer=alice", testInternalKey, "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})

	t.Run("can-view unknown user is 404", func(t *testing.T) {
		w := doKey(t, router, http.MethodGet, "/internal/users/ghost/can-view?viewer=alice", testInternalKey, "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("hidden-authors lists private non-followed", func(t *testing.T) {
		w := doKey(t, router, http.MethodGet, "/internal/users/hidden-authors?viewer=alice", testInternalKey, "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		// priv est privé et non suivi par alice → doit apparaître.
		if !strings.Contains(w.Body.String(), "priv") {
			t.Errorf("hidden authors should include priv: %s", w.Body.String())
		}
	})

	t.Run("following-ids lists followees", func(t *testing.T) {
		// alice suit bob (CreatedAt est auto-renseigné par GORM à la création).
		if err := s.db.Create(&followModel{FollowerID: "alice", FolloweeID: "bob"}).Error; err != nil {
			t.Fatalf("seed follow: %v", err)
		}
		w := doKey(t, router, http.MethodGet, "/internal/users/alice/following-ids", testInternalKey, "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "bob") {
			t.Errorf("following should include bob: %s", w.Body.String())
		}
	})

	t.Run("resolve-mentions maps usernames to ids", func(t *testing.T) {
		w := doKey(t, router, http.MethodPost, "/internal/users/resolve-mentions", testInternalKey,
			`{"usernames":["alice","bob","nobody"]}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		data, _ := decode(t, w)["data"].([]any)
		if len(data) != 2 {
			t.Errorf("resolved = %d, want 2 (alice, bob)", len(data))
		}
	})
}
