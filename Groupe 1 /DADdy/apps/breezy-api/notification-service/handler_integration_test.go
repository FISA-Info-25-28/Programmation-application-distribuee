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

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

type reqOpt struct {
	userID      string
	body        string
	internalKey string
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

func TestRequiresIdentity(t *testing.T) {
	_, router := newTestApp(t)
	if w := do(t, router, http.MethodGet, "/notifications", reqOpt{}); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestInternalAuth(t *testing.T) {
	_, router := newTestApp(t)
	body := `{"user_id":"alice","type":"follow","actor_id":"bob"}`

	t.Run("missing key is 401", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/internal/notifications", reqOpt{body: body}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
	t.Run("wrong key is 401", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/internal/notifications", reqOpt{body: body, internalKey: "nope"}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

func TestInternalCreateNotif(t *testing.T) {
	a, router := newTestApp(t)

	t.Run("creates a notification (201)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/notifications", reqOpt{
			internalKey: testInternalKey,
			body:        `{"user_id":"alice","type":"follow","actor_id":"bob","actor_username":"bob","entity_id":"bob","entity_type":"user"}`,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		var count int64
		a.db.Model(&notificationModel{}).Where("user_id = ?", "alice").Count(&count)
		if count != 1 {
			t.Errorf("notifications for alice = %d, want 1", count)
		}
	})

	t.Run("self-notification is a no-op (204)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/notifications", reqOpt{
			internalKey: testInternalKey,
			body:        `{"user_id":"alice","type":"follow","actor_id":"alice"}`,
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("missing required field is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/notifications", reqOpt{
			internalKey: testInternalKey,
			body:        `{"user_id":"alice"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestInternalNewPostFanout(t *testing.T) {
	a, router := newTestApp(t)

	// Deux abonnés au compte "author" + l'auteur lui-même (qui doit être exclu).
	for _, sub := range []string{"f1", "f2", "author"} {
		if err := a.db.Create(&subscriptionModel{SubscriberID: sub, TargetUserID: "author"}).Error; err != nil {
			t.Fatalf("seed sub %s: %v", sub, err)
		}
	}

	t.Run("fans out to subscribers excluding the author", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/notifications/new-post", reqOpt{
			internalKey: testInternalKey,
			body:        `{"author_id":"author","author_username":"author","post_id":"post_1"}`,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		var count int64
		a.db.Model(&notificationModel{}).Where("type = ?", "new_post").Count(&count)
		if count != 2 {
			t.Errorf("new_post notifications = %d, want 2 (author excluded)", count)
		}
	})

	t.Run("no subscribers yields 204", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/notifications/new-post", reqOpt{
			internalKey: testInternalKey,
			body:        `{"author_id":"lonely","author_username":"lonely","post_id":"post_2"}`,
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestListAndUnreadCount(t *testing.T) {
	a, router := newTestApp(t)
	seedNotif(t, a, "alice", "bob")
	seedNotif(t, a, "alice", "carol")
	seedNotif(t, a, "other", "bob") // ne doit jamais apparaître pour alice

	t.Run("list returns only the caller's notifications with envelope", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/notifications?limit=1", reqOpt{userID: "alice"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := decode(t, w)
		if body["total"] != float64(2) {
			t.Errorf("total = %v, want 2", body["total"])
		}
		if data, ok := body["data"].([]any); !ok || len(data) != 1 {
			t.Errorf("data length = %v, want 1 (paginated)", body["data"])
		}
	})

	t.Run("unread count counts only the caller's unread", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/notifications/unread-count", reqOpt{userID: "alice"})
		data := decode(t, w)["data"].(map[string]any)
		if data["count"] != float64(2) {
			t.Errorf("count = %v, want 2", data["count"])
		}
	})
}

func TestMarkRead(t *testing.T) {
	a, router := newTestApp(t)
	id := seedNotif(t, a, "alice", "bob")

	t.Run("invalid id is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPut, "/notifications/abc/read", reqOpt{userID: "alice"}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("another user's notification is 404", func(t *testing.T) {
		if w := do(t, router, http.MethodPut, "/notifications/"+itoa(id)+"/read", reqOpt{userID: "intruder"}); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("owner marks read (204)", func(t *testing.T) {
		if w := do(t, router, http.MethodPut, "/notifications/"+itoa(id)+"/read", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		var n notificationModel
		a.db.First(&n, id)
		if n.ReadAt == nil {
			t.Error("ReadAt should be set after marking read")
		}
	})
}

func TestMarkAllRead(t *testing.T) {
	a, router := newTestApp(t)
	seedNotif(t, a, "alice", "bob")
	seedNotif(t, a, "alice", "carol")

	if w := do(t, router, http.MethodPut, "/notifications/read-all", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	var unread int64
	a.db.Model(&notificationModel{}).Where("user_id = ? AND read_at IS NULL", "alice").Count(&unread)
	if unread != 0 {
		t.Errorf("unread after read-all = %d, want 0", unread)
	}
}

func TestDeleteNotification(t *testing.T) {
	a, router := newTestApp(t)
	id := seedNotif(t, a, "alice", "bob")

	t.Run("another user cannot delete (404)", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/notifications/"+itoa(id), reqOpt{userID: "intruder"}); w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("owner deletes (204)", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/notifications/"+itoa(id), reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		var count int64
		a.db.Model(&notificationModel{}).Where("id = ?", id).Count(&count)
		if count != 0 {
			t.Error("notification should be gone after delete")
		}
	})
}

func TestSubscriptions(t *testing.T) {
	a, router := newTestApp(t)

	t.Run("subscribe to yourself is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/subscriptions", reqOpt{userID: "alice", body: `{"target_user_id":"alice"}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("subscribe then list", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/subscriptions", reqOpt{userID: "alice", body: `{"target_user_id":"bob"}`}); w.Code != http.StatusCreated {
			t.Fatalf("subscribe status = %d, want 201", w.Code)
		}
		// Idempotent : re-subscribe reste 201 sans doublon.
		if w := do(t, router, http.MethodPost, "/subscriptions", reqOpt{userID: "alice", body: `{"target_user_id":"bob"}`}); w.Code != http.StatusCreated {
			t.Fatalf("re-subscribe status = %d, want 201", w.Code)
		}
		var count int64
		a.db.Model(&subscriptionModel{}).Where("subscriber_id = ?", "alice").Count(&count)
		if count != 1 {
			t.Errorf("subscriptions = %d, want 1 (idempotent)", count)
		}

		w := do(t, router, http.MethodGet, "/subscriptions", reqOpt{userID: "alice"})
		if data, ok := decode(t, w)["data"].([]any); !ok || len(data) != 1 || data[0] != "bob" {
			t.Errorf("list = %v, want [bob]", decode(t, w)["data"])
		}
	})

	t.Run("unsubscribe (204) is idempotent", func(t *testing.T) {
		if w := do(t, router, http.MethodDelete, "/subscriptions/bob", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		if w := do(t, router, http.MethodDelete, "/subscriptions/bob", reqOpt{userID: "alice"}); w.Code != http.StatusNoContent {
			t.Fatalf("idempotent unsubscribe status = %d, want 204", w.Code)
		}
	})
}
