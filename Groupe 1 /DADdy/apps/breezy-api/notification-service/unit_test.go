package main

import (
	"strings"
	"testing"
	"time"
)

func TestUserChannel(t *testing.T) {
	t.Parallel()

	ch := userChannel("usr_abc")

	// Identifiant Postgres sûr : préfixe 'n' + 16 hex = 17 chars, sous NAMEDATALEN (63).
	if len(ch) != 17 {
		t.Fatalf("channel length = %d, want 17", len(ch))
	}
	if !strings.HasPrefix(ch, "n") {
		t.Errorf("channel %q should start with 'n'", ch)
	}

	// Déterministe : même entrée → même canal.
	if ch != userChannel("usr_abc") {
		t.Error("userChannel must be deterministic")
	}
	// Entrées différentes → canaux différents.
	if ch == userChannel("usr_xyz") {
		t.Error("different users should map to different channels")
	}
}

func TestToNotifResponse(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 6, 18, 9, 30, 0, 0, time.UTC)
	base := notificationModel{
		ID:            7,
		Type:          "follow",
		ActorID:       "usr_actor",
		ActorUsername: "bob",
		EntityID:      "usr_actor",
		EntityType:    "user",
		CreatedAt:     created,
	}

	t.Run("unread when ReadAt nil", func(t *testing.T) {
		t.Parallel()
		resp := toNotifResponse(base)
		if resp.Read {
			t.Error("Read should be false when ReadAt is nil")
		}
		if resp.CreatedAt != "2026-06-18T09:30:00Z" {
			t.Errorf("CreatedAt = %q, want RFC3339 UTC", resp.CreatedAt)
		}
		if resp.ID != 7 || resp.Type != "follow" || resp.ActorUsername != "bob" {
			t.Errorf("fields not mapped: %+v", resp)
		}
	})

	t.Run("read when ReadAt set", func(t *testing.T) {
		t.Parallel()
		read := created.Add(time.Hour)
		n := base
		n.ReadAt = &read
		if !toNotifResponse(n).Read {
			t.Error("Read should be true when ReadAt is set")
		}
	})
}
