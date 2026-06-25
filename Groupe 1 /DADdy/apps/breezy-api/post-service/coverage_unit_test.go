package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestCanViewProfile(t *testing.T) {
	t.Run("empty userURL allows", func(t *testing.T) {
		a := &app{}
		if !a.canViewProfile("viewer", "target") {
			t.Fatal("empty userURL should allow")
		}
	})

	t.Run("server says can view", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"canView":true}`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if !a.canViewProfile("viewer", "target") {
			t.Fatal("should allow when canView=true")
		}
	})

	t.Run("server says cannot view", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"canView":false}`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if a.canViewProfile("viewer", "target") {
			t.Fatal("should deny when canView=false")
		}
	})

	t.Run("non-200 fails closed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if a.canViewProfile("viewer", "target") {
			t.Fatal("non-200 should fail closed")
		}
	})

	t.Run("bad json fails closed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if a.canViewProfile("viewer", "target") {
			t.Fatal("bad json should fail closed")
		}
	})

	t.Run("network error fails closed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close() // server is now down
		a := &app{userURL: url, internalKey: "k"}
		if a.canViewProfile("viewer", "target") {
			t.Fatal("network error should fail closed")
		}
	})
}

func TestHiddenAuthors(t *testing.T) {
	t.Run("empty userURL returns nil", func(t *testing.T) {
		a := &app{}
		if a.hiddenAuthors("viewer") != nil {
			t.Fatal("empty userURL should return nil")
		}
	})

	t.Run("returns hidden list", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"hidden":["a","b"]}`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		got := a.hiddenAuthors("viewer")
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("hiddenAuthors = %v, want [a b]", got)
		}
	})

	t.Run("non-200 returns nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if a.hiddenAuthors("viewer") != nil {
			t.Fatal("non-200 should return nil")
		}
	})

	t.Run("bad json returns nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if a.hiddenAuthors("viewer") != nil {
			t.Fatal("bad json should return nil")
		}
	})

	t.Run("network error returns nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()
		a := &app{userURL: url, internalKey: "k"}
		if a.hiddenAuthors("viewer") != nil {
			t.Fatal("network error should return nil")
		}
	})
}

func TestResolveMentions(t *testing.T) {
	t.Run("empty userURL returns explicit", func(t *testing.T) {
		a := &app{}
		explicit := []mentionInput{{ID: "1", Username: "bob"}}
		got := a.resolveMentions("hi @alice", explicit)
		if len(got) != 1 || got[0].Username != "bob" {
			t.Fatalf("got %v, want explicit unchanged", got)
		}
	})

	t.Run("no new mentions returns explicit", func(t *testing.T) {
		a := &app{userURL: "http://example.invalid", internalKey: "k"}
		explicit := []mentionInput{{ID: "1", Username: "bob"}}
		// @bob already known, no content mention to resolve.
		got := a.resolveMentions("hi @bob", explicit)
		if len(got) != 1 {
			t.Fatalf("got %v, want explicit unchanged", got)
		}
	})

	t.Run("resolves content mentions via user-service", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"data":[{"id":"u9","username":"alice"}]}`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		got := a.resolveMentions("hey @alice look", nil)
		if len(got) != 1 || got[0].ID != "u9" || got[0].Username != "alice" {
			t.Fatalf("resolveMentions = %v, want alice resolved", got)
		}
	})

	t.Run("bad json returns explicit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`nope`))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		explicit := []mentionInput{{ID: "1", Username: "bob"}}
		got := a.resolveMentions("hey @alice", explicit)
		if len(got) != 1 {
			t.Fatalf("got %v, want explicit on bad json", got)
		}
	})

	t.Run("network error returns explicit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close() // server is now down
		a := &app{userURL: url, internalKey: "k"}
		explicit := []mentionInput{{ID: "1", Username: "bob"}}
		got := a.resolveMentions("hey @alice", explicit)
		if len(got) != 1 {
			t.Fatalf("got %v, want explicit on network error", got)
		}
	})
}

func TestSendNotifAsync(t *testing.T) {
	t.Run("empty notifURL is no-op", func(t *testing.T) {
		a := &app{}
		a.sendNotifAsync(map[string]string{"x": "y"}) // must not panic
	})

	t.Run("posts payload to notif service", func(t *testing.T) {
		hit := make(chan string, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit <- r.URL.Path
		}))
		defer srv.Close()
		a := &app{notifURL: srv.URL, internalKey: "k"}
		a.sendNotifAsync(map[string]string{"type": "like"})
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
		a.sendNotifAsync(map[string]string{"type": "like"})
		select {
		case <-done:
			time.Sleep(50 * time.Millisecond) // let goroutine check resp.StatusCode
		case <-time.After(2 * time.Second):
			t.Fatal("notif service was not called")
		}
	})
}

func TestSendNewPostAsync(t *testing.T) {
	t.Run("empty notifURL is no-op", func(t *testing.T) {
		a := &app{}
		a.sendNewPostAsync("a", "alice", "5")
	})

	t.Run("posts to new-post endpoint", func(t *testing.T) {
		hit := make(chan string, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit <- r.URL.Path
		}))
		defer srv.Close()
		a := &app{notifURL: srv.URL, internalKey: "k"}
		a.sendNewPostAsync("a", "alice", "5")
		select {
		case path := <-hit:
			if path != "/internal/notifications/new-post" {
				t.Fatalf("path = %q", path)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("notif service was not called")
		}
	})

	t.Run("status >= 300 logs unexpected status", func(t *testing.T) {
		done := make(chan struct{}, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			done <- struct{}{}
		}))
		defer srv.Close()
		a := &app{notifURL: srv.URL, internalKey: "k"}
		a.sendNewPostAsync("a", "alice", "5")
		select {
		case <-done:
			time.Sleep(50 * time.Millisecond)
		case <-time.After(2 * time.Second):
			t.Fatal("notif service was not called")
		}
	})
}

func TestValidatePollInput(t *testing.T) {
	tests := []struct {
		name     string
		poll     pollInput
		hasMedia bool
		want     bool
		wantDays int
	}{
		{name: "with media is invalid", poll: pollInput{Options: []string{"a", "b"}}, hasMedia: true, want: false},
		{name: "too few options", poll: pollInput{Options: []string{"a"}}, want: false},
		{name: "too many options", poll: pollInput{Options: []string{"a", "b", "c", "d", "e"}}, want: false},
		{name: "blank option", poll: pollInput{Options: []string{"a", "  "}}, want: false},
		{name: "option too long", poll: pollInput{Options: []string{"a", "this option is way too long to be allowed"}}, want: false},
		{name: "valid clamps zero duration to 1", poll: pollInput{Options: []string{"a", "b"}, DurationDays: 0}, want: true, wantDays: 1},
		{name: "valid clamps over-7 duration to 1", poll: pollInput{Options: []string{"a", "b"}, DurationDays: 9}, want: true, wantDays: 1},
		{name: "valid keeps in-range duration", poll: pollInput{Options: []string{"a", "b", "c"}, DurationDays: 5}, want: true, wantDays: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			poll := tt.poll
			got := validatePollInput(c, &poll, tt.hasMedia)
			if got != tt.want {
				t.Fatalf("validatePollInput = %v, want %v", got, tt.want)
			}
			if tt.want && poll.DurationDays != tt.wantDays {
				t.Fatalf("DurationDays = %d, want %d", poll.DurationDays, tt.wantDays)
			}
		})
	}
}
