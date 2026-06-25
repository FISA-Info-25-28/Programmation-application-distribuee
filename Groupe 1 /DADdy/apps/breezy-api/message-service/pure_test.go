package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func strptr(s string) *string { return &s }

func TestSendNotifAsync(t *testing.T) {
	t.Run("empty notifURL is no-op", func(t *testing.T) {
		a := &app{}
		a.sendNotifAsync("u", "actor", "alice", "3") // must not panic
	})

	t.Run("posts a new_message notification", func(t *testing.T) {
		hit := make(chan string, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit <- r.URL.Path
		}))
		defer srv.Close()
		a := &app{notifURL: srv.URL, internalKey: "k"}
		a.sendNotifAsync("u", "actor", "alice", "3")
		select {
		case path := <-hit:
			if path != "/internal/notifications" {
				t.Fatalf("path = %q", path)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("notif service was not called")
		}
	})

	t.Run("status >= 300 logs unexpected status", func(t *testing.T) {
		done := make(chan struct{}, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			done <- struct{}{}
		}))
		defer srv.Close()
		a := &app{notifURL: srv.URL, internalKey: "k"}
		a.sendNotifAsync("u", "actor", "alice", "3")
		select {
		case <-done:
			time.Sleep(50 * time.Millisecond)
		case <-time.After(2 * time.Second):
			t.Fatal("notif service was not called")
		}
	})
}

func TestToMsgResponse(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 6, 18, 9, 30, 0, 0, time.UTC)
	m := messageModel{
		ID: 9, ConversationID: 3, SenderID: "usr_1",
		Content:   "hello", // clair (legacy) : decryptContent le renvoie tel quel
		MediaURL:  strptr("http://x/y.png"),
		MediaType: strptr("image"),
		CreatedAt: created,
	}
	resp := toMsgResponse(m)

	if resp.ID != 9 || resp.ConversationID != 3 || resp.SenderID != "usr_1" {
		t.Errorf("ids not mapped: %+v", resp)
	}
	if resp.Content != "hello" {
		t.Errorf("content = %q, want hello", resp.Content)
	}
	if resp.MediaURL == nil || *resp.MediaURL != *m.MediaURL || resp.MediaType == nil || *resp.MediaType != *m.MediaType {
		t.Errorf("media not mapped: %+v", resp)
	}
	if resp.CreatedAt != "2026-06-18T09:30:00Z" {
		t.Errorf("CreatedAt = %q, want RFC3339 UTC", resp.CreatedAt)
	}
}

func TestParsePagination(t *testing.T) {
	t.Parallel()

	newCtx := func(query string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		return c
	}

	tests := []struct {
		name       string
		query      string
		wantOffset int
		wantLimit  int
	}{
		{name: "defaults", query: "", wantOffset: 0, wantLimit: 20},
		{name: "page 2 default limit", query: "page=2", wantOffset: 20, wantLimit: 20},
		{name: "custom limit", query: "page=3&limit=10", wantOffset: 20, wantLimit: 10},
		{name: "invalid page falls back", query: "page=0", wantOffset: 0, wantLimit: 20},
		{name: "limit over cap falls back", query: "limit=9999", wantOffset: 0, wantLimit: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			offset, limit := parsePagination(newCtx(tt.query))
			if offset != tt.wantOffset || limit != tt.wantLimit {
				t.Fatalf("parsePagination(%q) = (%d, %d), want (%d, %d)", tt.query, offset, limit, tt.wantOffset, tt.wantLimit)
			}
		})
	}
}
