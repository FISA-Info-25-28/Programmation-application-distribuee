package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// captureServer renvoie un serveur de test qui pousse (méthode, chemin) sur le
// canal hits dès qu'il reçoit une requête.
func captureServer(t *testing.T) (*httptest.Server, chan [2]string) {
	t.Helper()
	hits := make(chan [2]string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits <- [2]string{r.Method, r.URL.Path}
	}))
	return srv, hits
}

func waitHit(t *testing.T, hits chan [2]string) [2]string {
	t.Helper()
	select {
	case h := <-hits:
		return h
	case <-time.After(2 * time.Second):
		t.Fatal("downstream service was not called")
		return [2]string{}
	}
}

func TestSyncSubscriptionAsync(t *testing.T) {
	t.Run("empty NotifURL is no-op", func(t *testing.T) {
		s := &userService{cfg: userConfig{}}
		s.syncSubscriptionAsync("f", "g", true) // must not panic / call out
	})

	t.Run("subscribe POSTs to subscriptions", func(t *testing.T) {
		srv, hits := captureServer(t)
		defer srv.Close()
		s := &userService{cfg: userConfig{NotifURL: srv.URL, InternalKey: "k"}}
		s.syncSubscriptionAsync("follower", "followee", true)
		h := waitHit(t, hits)
		if h[0] != http.MethodPost || h[1] != "/internal/subscriptions" {
			t.Fatalf("got %v, want POST /internal/subscriptions", h)
		}
	})

	t.Run("unsubscribe DELETEs subscriptions", func(t *testing.T) {
		srv, hits := captureServer(t)
		defer srv.Close()
		s := &userService{cfg: userConfig{NotifURL: srv.URL, InternalKey: "k"}}
		s.syncSubscriptionAsync("follower", "followee", false)
		h := waitHit(t, hits)
		if h[0] != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", h[0])
		}
	})
}

func TestSyncAuthStatusAsync(t *testing.T) {
	t.Run("empty AuthURL is no-op", func(t *testing.T) {
		s := &userService{cfg: userConfig{}}
		s.syncAuthStatusAsync("u1", "banned")
	})

	t.Run("PATCHes auth status endpoint", func(t *testing.T) {
		srv, hits := captureServer(t)
		defer srv.Close()
		s := &userService{cfg: userConfig{AuthURL: srv.URL, InternalKey: "k"}}
		s.syncAuthStatusAsync("u1", "banned")
		h := waitHit(t, hits)
		if h[0] != http.MethodPatch || h[1] != "/internal/users/u1/status" {
			t.Fatalf("got %v, want PATCH /internal/users/u1/status", h)
		}
	})
}
