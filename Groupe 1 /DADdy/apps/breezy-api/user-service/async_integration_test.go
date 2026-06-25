//go:build integration

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendNotifAsyncIntegration(t *testing.T) {
	s := newTestService(t)
	seedUser(t, s, "actor1", "actor1", false)

	t.Run("empty NotifURL is no-op", func(t *testing.T) {
		s.cfg.NotifURL = ""
		s.sendNotifAsync("recipient", "actor1", notifTypeFollow) // no panic, no call
	})

	t.Run("posts enriched notification with actor username", func(t *testing.T) {
		type payload map[string]string
		got := make(chan payload, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var p payload
			_ = json.Unmarshal(body, &p)
			got <- p
		}))
		defer srv.Close()

		s.cfg.NotifURL = srv.URL
		s.cfg.InternalKey = "k"
		s.sendNotifAsync("recipient", "actor1", notifTypeFollow)

		select {
		case p := <-got:
			if p["user_id"] != "recipient" || p["actor_id"] != "actor1" {
				t.Errorf("payload = %v", p)
			}
			if p["actor_username"] != "actor1" {
				t.Errorf("actor_username = %q, want resolved from db", p["actor_username"])
			}
			if p["type"] != notifTypeFollow {
				t.Errorf("type = %q", p["type"])
			}
		case <-time.After(2 * time.Second):
			t.Fatal("notification service was not called")
		}
	})
}
