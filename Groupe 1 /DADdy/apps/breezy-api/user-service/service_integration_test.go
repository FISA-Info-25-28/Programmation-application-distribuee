//go:build integration

package main

import (
	"errors"
	"strings"
	"testing"
)

func TestCreateProfile(t *testing.T) {
	s := newTestService(t)

	t.Run("normalizes and applies defaults", func(t *testing.T) {
		if err := s.createProfile("  usr_1 ", "  JohnDoe ", "  John@Example.COM ", ""); err != nil {
			t.Fatalf("createProfile: %v", err)
		}
		u := reload(t, s, "usr_1")
		if u.Username != "johndoe" {
			t.Errorf("Username = %q, want lowercased+trimmed", u.Username)
		}
		if u.Email != "john@example.com" {
			t.Errorf("Email = %q, want lowercased+trimmed", u.Email)
		}
		if u.Role != roleUser {
			t.Errorf("Role = %q, want default %q", u.Role, roleUser)
		}
		if u.DisplayName == nil || *u.DisplayName != "johndoe" {
			t.Errorf("DisplayName = %v, want default = username", u.DisplayName)
		}
	})

	t.Run("idempotent on id", func(t *testing.T) {
		if err := s.createProfile("usr_1", "different", "other@example.com", "admin"); err != nil {
			t.Fatalf("second createProfile: %v", err)
		}
		u := reload(t, s, "usr_1")
		if u.Username != "johndoe" {
			t.Errorf("Username = %q, want unchanged (ON CONFLICT DO NOTHING)", u.Username)
		}
	})

	t.Run("rejects missing required fields", func(t *testing.T) {
		err := s.createProfile("", "x", "x@example.com", "")
		if !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})
}

func TestFollowPublic(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false)
	bob := seedUser(t, s, "bob", "bob", false)

	outcome, err := s.follow(alice, bob)
	if err != nil {
		t.Fatalf("follow: %v", err)
	}
	if outcome != outcomeFollowed {
		t.Fatalf("outcome = %v, want outcomeFollowed", outcome)
	}
	if c := reload(t, s, bob).FollowersCount; c != 1 {
		t.Errorf("bob.FollowersCount = %d, want 1", c)
	}
	if c := reload(t, s, alice).FollowingCount; c != 1 {
		t.Errorf("alice.FollowingCount = %d, want 1", c)
	}

	t.Run("idempotent: re-follow does not double counters", func(t *testing.T) {
		if _, err := s.follow(alice, bob); err != nil {
			t.Fatalf("re-follow: %v", err)
		}
		if c := reload(t, s, bob).FollowersCount; c != 1 {
			t.Errorf("bob.FollowersCount = %d, want still 1", c)
		}
	})
}

func TestFollowErrors(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false)

	t.Run("cannot follow self", func(t *testing.T) {
		if _, err := s.follow(alice, alice); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("target not found", func(t *testing.T) {
		if _, err := s.follow(alice, "ghost"); !errors.Is(err, errNotFound) {
			t.Fatalf("err = %v, want errNotFound", err)
		}
	})
}

func TestFollowPrivateCreatesRequest(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false)
	priv := seedUser(t, s, "priv", "priv", true)

	outcome, err := s.follow(alice, priv)
	if err != nil {
		t.Fatalf("follow private: %v", err)
	}
	if outcome != outcomeRequested {
		t.Fatalf("outcome = %v, want outcomeRequested", outcome)
	}
	// Aucun compteur ne bouge tant que la demande n'est pas acceptée.
	if c := reload(t, s, priv).FollowersCount; c != 0 {
		t.Errorf("priv.FollowersCount = %d, want 0", c)
	}
	var reqCount int64
	s.db.Model(&followRequestModel{}).Where("requester_id = ? AND target_id = ?", alice, priv).Count(&reqCount)
	if reqCount != 1 {
		t.Errorf("follow_requests = %d, want 1", reqCount)
	}

	t.Run("accept request creates follow and bumps counters", func(t *testing.T) {
		if err := s.acceptFollowRequest(priv, alice); err != nil {
			t.Fatalf("accept: %v", err)
		}
		if c := reload(t, s, priv).FollowersCount; c != 1 {
			t.Errorf("priv.FollowersCount = %d, want 1", c)
		}
		s.db.Model(&followRequestModel{}).Where("requester_id = ? AND target_id = ?", alice, priv).Count(&reqCount)
		if reqCount != 0 {
			t.Errorf("follow_requests = %d, want 0 after accept", reqCount)
		}
	})
}

func TestAcceptFollowRequestNotFound(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false)
	priv := seedUser(t, s, "priv", "priv", true)

	if err := s.acceptFollowRequest(priv, alice); !errors.Is(err, errNotFound) {
		t.Fatalf("err = %v, want errNotFound", err)
	}
}

func TestUnfollow(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false)
	bob := seedUser(t, s, "bob", "bob", false)
	if _, err := s.follow(alice, bob); err != nil {
		t.Fatalf("setup follow: %v", err)
	}

	t.Run("unfollow decrements counters", func(t *testing.T) {
		if err := s.unfollow(alice, bob); err != nil {
			t.Fatalf("unfollow: %v", err)
		}
		if c := reload(t, s, bob).FollowersCount; c != 0 {
			t.Errorf("bob.FollowersCount = %d, want 0", c)
		}
		if c := reload(t, s, alice).FollowingCount; c != 0 {
			t.Errorf("alice.FollowingCount = %d, want 0", c)
		}
	})

	t.Run("idempotent: unfollow again keeps counters at zero floor", func(t *testing.T) {
		if err := s.unfollow(alice, bob); err != nil {
			t.Fatalf("second unfollow: %v", err)
		}
		if c := reload(t, s, bob).FollowersCount; c != 0 {
			t.Errorf("bob.FollowersCount = %d, want 0 (no negative)", c)
		}
	})
}

func TestBlockUser(t *testing.T) {
	s := newTestService(t)
	alice := seedUser(t, s, "alice", "alice", false)
	bob := seedUser(t, s, "bob", "bob", false)

	t.Run("cannot block self", func(t *testing.T) {
		if err := s.blockUser(alice, alice); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("block missing target is not found", func(t *testing.T) {
		if err := s.blockUser(alice, "ghost"); !errors.Is(err, errNotFound) {
			t.Fatalf("err = %v, want errNotFound", err)
		}
	})

	t.Run("block then idempotent re-block", func(t *testing.T) {
		if err := s.blockUser(alice, bob); err != nil {
			t.Fatalf("block: %v", err)
		}
		if err := s.blockUser(alice, bob); err != nil {
			t.Fatalf("re-block: %v", err)
		}
		var count int64
		s.db.Model(&blockModel{}).Where("blocker_id = ? AND blocked_id = ?", alice, bob).Count(&count)
		if count != 1 {
			t.Errorf("blocks = %d, want 1", count)
		}
	})

	t.Run("unblock removes the relation", func(t *testing.T) {
		if err := s.unblockUser(alice, bob); err != nil {
			t.Fatalf("unblock: %v", err)
		}
		var count int64
		s.db.Model(&blockModel{}).Where("blocker_id = ? AND blocked_id = ?", alice, bob).Count(&count)
		if count != 0 {
			t.Errorf("blocks = %d, want 0", count)
		}
	})
}

func TestSanctionUser(t *testing.T) {
	s := newTestService(t)
	mod := seedUser(t, s, "mod", "mod", false)
	target := seedUser(t, s, "target", "target", false)

	t.Run("rejects invalid type", func(t *testing.T) {
		if _, err := s.sanctionUser(mod, target, sanctionRequest{Type: "warn"}); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("cannot sanction self", func(t *testing.T) {
		if _, err := s.sanctionUser(mod, mod, sanctionRequest{Type: sanctionTypeBan}); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("ban sets status and truncates long reason", func(t *testing.T) {
		long := strings.Repeat("x", 300)
		sanction, err := s.sanctionUser(mod, target, sanctionRequest{Type: sanctionTypeBan, Reason: &long})
		if err != nil {
			t.Fatalf("sanction: %v", err)
		}
		if sanction.Reason == nil || len(*sanction.Reason) != 255 {
			t.Errorf("reason length = %v, want 255", sanction.Reason)
		}
		if st := reload(t, s, target).Status; st != statusBanned {
			t.Errorf("target.Status = %q, want %q", st, statusBanned)
		}
	})

	t.Run("new sanction closes the previous active one", func(t *testing.T) {
		if _, err := s.sanctionUser(mod, target, sanctionRequest{Type: sanctionTypeSuspension}); err != nil {
			t.Fatalf("second sanction: %v", err)
		}
		var active int64
		s.db.Model(&sanctionModel{}).Where("user_id = ? AND ended_at IS NULL", target).Count(&active)
		if active != 1 {
			t.Errorf("active sanctions = %d, want exactly 1", active)
		}
		if st := reload(t, s, target).Status; st != statusSuspended {
			t.Errorf("target.Status = %q, want %q", st, statusSuspended)
		}
	})

	t.Run("lift sanction reactivates account", func(t *testing.T) {
		if err := s.liftSanction(target); err != nil {
			t.Fatalf("lift: %v", err)
		}
		if st := reload(t, s, target).Status; st != statusActive {
			t.Errorf("target.Status = %q, want %q", st, statusActive)
		}
		var active int64
		s.db.Model(&sanctionModel{}).Where("user_id = ? AND ended_at IS NULL", target).Count(&active)
		if active != 0 {
			t.Errorf("active sanctions = %d, want 0", active)
		}
	})
}
