//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

// reqOpt configure une requête de test (identité injectée, corps JSON).
type reqOpt struct {
	userID string
	role   string
	body   string
}

// do exécute une requête sur le routeur réel et renvoie le ResponseRecorder.
// Les headers X-User-Id/X-User-Role reproduisent ce que pose le gateway.
func do(t *testing.T, router *gin.Engine, method, path string, opt reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	var body *strings.Reader
	if opt.body != "" {
		body = strings.NewReader(opt.body)
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
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

// decode lit le corps JSON de la réponse dans une map générique.
func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return m
}

func newTestRouter(t *testing.T) (*gin.Engine, *userService) {
	t.Helper()
	s := newTestService(t)
	return newUserRouter(s), s
}

func TestHTTPRequiresIdentity(t *testing.T) {
	router, _ := newTestRouter(t)

	// Sans header X-User-Id, RequireIdentity doit rejeter en 401.
	w := do(t, router, http.MethodGet, "/users", reqOpt{})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /users without identity = %d, want 401", w.Code)
	}
	if got := decode(t, w)["error"]; got != string(shared.ErrUnauthenticated) {
		t.Errorf("error code = %v, want %s", got, shared.ErrUnauthenticated)
	}
}

func TestHTTPGetProfile(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)

	t.Run("self shows email and view flags", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/alice", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := decode(t, w)
		if body["email"] != "alice@example.com" {
			t.Errorf("email = %v, want present for self", body["email"])
		}
		if body["canViewPosts"] != true {
			t.Errorf("canViewPosts = %v, want true", body["canViewPosts"])
		}
		if _, ok := body["followRequested"]; !ok {
			t.Error("followRequested missing on GET /users/:id")
		}
	})

	t.Run("other hides email", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/bob", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if _, ok := decode(t, w)["email"]; ok {
			t.Error("email should be omitted when viewing another profile")
		}
	})

	t.Run("unknown id is 404", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/ghost", reqOpt{userID: "alice"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

func TestHTTPLookup(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)

	t.Run("found", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/lookup?username=alice", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("missing param is 400", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/lookup", reqOpt{userID: "alice"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unknown username is 404", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/users/lookup?username=nobody", reqOpt{userID: "alice"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

func TestHTTPUpdateProfile(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)

	t.Run("valid patch returns 200 with email", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/users/me", reqOpt{userID: "alice", body: `{"bio":"hello world"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if decode(t, w)["bio"] != "hello world" {
			t.Errorf("bio not updated: %v", decode(t, w)["bio"])
		}
	})

	t.Run("invalid json is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/users/me", reqOpt{userID: "alice", body: `{not json`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("bio over limit is 400", func(t *testing.T) {
		long := strings.Repeat("a", 161)
		w := do(t, router, http.MethodPatch, "/users/me", reqOpt{userID: "alice", body: `{"bio":"` + long + `"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestHTTPFollow(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)
	seedUser(t, s, "priv", "priv", true)

	t.Run("follow public returns 204", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/bob/follow", reqOpt{userID: "alice"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("follow private returns 202 requested", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/priv/follow", reqOpt{userID: "alice"})
		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", w.Code)
		}
		if decode(t, w)["status"] != "requested" {
			t.Errorf("status field = %v, want requested", decode(t, w)["status"])
		}
	})

	t.Run("follow self returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/alice/follow", reqOpt{userID: "alice"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("follow ghost returns 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/ghost/follow", reqOpt{userID: "alice"})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("unfollow returns 204", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/users/bob/follow", reqOpt{userID: "alice"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestHTTPBlock(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)

	t.Run("block returns 204", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/bob/block", reqOpt{userID: "alice"})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("block self returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/alice/block", reqOpt{userID: "alice"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestHTTPListUsersEnvelope(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "alice", "alice", false)
	seedUser(t, s, "bob", "bob", false)

	w := do(t, router, http.MethodGet, "/users?limit=1", reqOpt{userID: "alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := decode(t, w)
	for _, k := range []string{"data", "page", "limit", "total"} {
		if _, ok := body[k]; !ok {
			t.Errorf("envelope missing key %q", k)
		}
	}
	if body["limit"] != float64(1) {
		t.Errorf("limit = %v, want 1", body["limit"])
	}
	if body["total"] != float64(2) {
		t.Errorf("total = %v, want 2", body["total"])
	}
	if data, ok := body["data"].([]any); !ok || len(data) != 1 {
		t.Errorf("data length = %v, want 1 (paginated)", body["data"])
	}
}

func TestHTTPSanctionAuthorization(t *testing.T) {
	router, s := newTestRouter(t)
	seedUser(t, s, "mod", "mod", false)
	seedUser(t, s, "target", "target", false)

	t.Run("regular user is forbidden", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/target/sanctions",
			reqOpt{userID: "mod", role: shared.RoleUser, body: `{"type":"ban"}`})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("moderator can sanction and lift", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/target/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"type":"ban","reason":"spam"}`})
		if w.Code != http.StatusCreated {
			t.Fatalf("sanction status = %d, want 201", w.Code)
		}
		if decode(t, w)["type"] != "ban" {
			t.Errorf("type = %v, want ban", decode(t, w)["type"])
		}

		wl := do(t, router, http.MethodDelete, "/users/target/sanctions",
			reqOpt{userID: "mod", role: shared.RoleModerator})
		if wl.Code != http.StatusNoContent {
			t.Fatalf("lift status = %d, want 204", wl.Code)
		}
	})

	t.Run("invalid sanction type is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/users/target/sanctions",
			reqOpt{userID: "mod", role: shared.RoleAdministrator, body: `{"type":"warn"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
