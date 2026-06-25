//go:build integration

package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCanView(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false) // public
	priv := seedUser(t, s, "priv", "priv", true)    // private
	viewer := seedUser(t, s, "viewer", "viewer", false)

	t.Run("target not found returns errNotFound", func(t *testing.T) {
		_, _, err := s.canView("alice", "ghost")
		if !errors.Is(err, errNotFound) {
			t.Fatalf("err = %v, want errNotFound", err)
		}
	})

	t.Run("public user visible to anonymous viewer", func(t *testing.T) {
		isPrivate, visible, err := s.canView("", alice)
		if err != nil {
			t.Fatalf("canView: %v", err)
		}
		if isPrivate {
			t.Error("isPrivate = true, want false")
		}
		if !visible {
			t.Error("visible = false, want true")
		}
	})

	t.Run("public user visible to logged-in viewer", func(t *testing.T) {
		_, visible, err := s.canView(viewer, alice)
		if err != nil {
			t.Fatalf("canView: %v", err)
		}
		if !visible {
			t.Error("visible = false, want true")
		}
	})

	t.Run("private user visible to self", func(t *testing.T) {
		isPrivate, visible, err := s.canView(priv, priv)
		if err != nil {
			t.Fatalf("canView: %v", err)
		}
		if !isPrivate {
			t.Error("isPrivate = false, want true")
		}
		if !visible {
			t.Error("visible = false, want true (self)")
		}
	})

	t.Run("private user invisible to anonymous viewer", func(t *testing.T) {
		isPrivate, visible, err := s.canView("", priv)
		if err != nil {
			t.Fatalf("canView: %v", err)
		}
		if !isPrivate {
			t.Error("isPrivate = false, want true")
		}
		if visible {
			t.Error("visible = true, want false (anonymous + private)")
		}
	})

	t.Run("private user invisible to non-follower", func(t *testing.T) {
		isPrivate, visible, err := s.canView(viewer, priv)
		if err != nil {
			t.Fatalf("canView: %v", err)
		}
		if !isPrivate {
			t.Error("isPrivate = false, want true")
		}
		if visible {
			t.Error("visible = true, want false (not following)")
		}
	})

	t.Run("private user visible to follower", func(t *testing.T) {
		if err := s.db.Create(&followModel{
			FollowerID: viewer,
			FolloweeID: priv,
		}).Error; err != nil {
			t.Fatalf("seed follow: %v", err)
		}
		isPrivate, visible, err := s.canView(viewer, priv)
		if err != nil {
			t.Fatalf("canView: %v", err)
		}
		if !isPrivate {
			t.Error("isPrivate = false, want true")
		}
		if !visible {
			t.Error("visible = false, want true (follower)")
		}
	})
}

// TestHiddenAuthorIDsNoPrivate is a separate top-level test so it gets its own
// clean database state (newTestService calls truncateAll).
func TestHiddenAuthorIDsNoPrivate(t *testing.T) {
	s := newTestService(t)
	seedUser(t, s, "only-pub", "only-pub", false)

	ids, err := s.hiddenAuthorIDs("")
	if err != nil {
		t.Fatalf("hiddenAuthorIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("len = %d, want 0 (no private users)", len(ids))
	}
}

// TestHTTPHiddenAuthorsNoPrivate exercises the ids == nil branch in
// hiddenAuthorsHandler when the DB contains no private users.
func TestHTTPHiddenAuthorsNoPrivate(t *testing.T) {
	router, s := newInternalRouter(t)
	seedUser(t, s, "only-pub", "only-pub", false)

	w := doKey(t, router, "GET", "/internal/users/hidden-authors", testInternalKey, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"hidden":[]`) && !strings.Contains(w.Body.String(), `"hidden": []`) {
		t.Errorf("expected empty hidden array, got: %s", w.Body.String())
	}
}

func TestHiddenAuthorIDs(t *testing.T) {
	s := newTestService(t)
	seedUser(t, s, "pub1", "pub1", false)
	priv1 := seedUser(t, s, "priv1", "priv1", true)
	priv2 := seedUser(t, s, "priv2", "priv2", true)
	viewer := seedUser(t, s, "viewer", "viewer", false)

	t.Run("anonymous viewer gets all private users as hidden", func(t *testing.T) {
		ids, err := s.hiddenAuthorIDs("")
		if err != nil {
			t.Fatalf("hiddenAuthorIDs: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("len = %d, want 2 (priv1 and priv2)", len(ids))
		}
	})

	t.Run("viewer following priv1 sees only priv2 as hidden", func(t *testing.T) {
		if err := s.db.Create(&followModel{
			FollowerID: viewer,
			FolloweeID: priv1,
		}).Error; err != nil {
			t.Fatalf("seed follow: %v", err)
		}
		ids, err := s.hiddenAuthorIDs(viewer)
		if err != nil {
			t.Fatalf("hiddenAuthorIDs: %v", err)
		}
		if len(ids) != 1 || ids[0] != priv2 {
			t.Errorf("ids = %v, want [%s]", ids, priv2)
		}
	})

	t.Run("private user not in own hidden list", func(t *testing.T) {
		ids, err := s.hiddenAuthorIDs(priv1)
		if err != nil {
			t.Fatalf("hiddenAuthorIDs: %v", err)
		}
		for _, id := range ids {
			if id == priv1 {
				t.Error("priv1 should not appear in its own hidden list")
			}
		}
	})
}
