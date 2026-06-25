//go:build integration

package main

import "testing"

// Ces tests couvrent les écritures DB pures de gestion d'avatar/bannière
// (setAvatar, clearAvatar, setBanner, clearBanner, updateImageFields). Elles ne
// touchent pas MinIO : seuls les handlers HTTP d'upload (avatar.go) sont exclus
// de la couverture, pas cette logique transactionnelle.

func TestSetAndClearAvatar(t *testing.T) {
	s := newTestService(t)
	uid := seedUser(t, s, "u1", "alice", false)

	// Premier avatar : pas d'ancienne clé.
	user, oldKey, err := s.setAvatar(uid, "avatars/u1/a.png", "http://cdn/a.png")
	if err != nil {
		t.Fatalf("setAvatar: %v", err)
	}
	if oldKey != nil {
		t.Errorf("oldKey = %v, want nil on first set", *oldKey)
	}
	if user.AvatarURL == nil || *user.AvatarURL != "http://cdn/a.png" {
		t.Errorf("returned AvatarURL = %v, want http://cdn/a.png", user.AvatarURL)
	}
	if got := reload(t, s, uid); got.AvatarObjectKey == nil || *got.AvatarObjectKey != "avatars/u1/a.png" {
		t.Errorf("persisted AvatarObjectKey = %v, want avatars/u1/a.png", got.AvatarObjectKey)
	}

	// Deuxième avatar : l'ancienne clé doit être renvoyée pour suppression MinIO.
	_, oldKey, err = s.setAvatar(uid, "avatars/u1/b.png", "http://cdn/b.png")
	if err != nil {
		t.Fatalf("setAvatar (2): %v", err)
	}
	if oldKey == nil || *oldKey != "avatars/u1/a.png" {
		t.Errorf("oldKey = %v, want avatars/u1/a.png", oldKey)
	}

	// clearAvatar renvoie la clé courante et remet les champs à NULL.
	oldKey, err = s.clearAvatar(uid)
	if err != nil {
		t.Fatalf("clearAvatar: %v", err)
	}
	if oldKey == nil || *oldKey != "avatars/u1/b.png" {
		t.Errorf("clearAvatar oldKey = %v, want avatars/u1/b.png", oldKey)
	}
	if got := reload(t, s, uid); got.AvatarURL != nil || got.AvatarObjectKey != nil {
		t.Errorf("avatar not cleared: url=%v key=%v", got.AvatarURL, got.AvatarObjectKey)
	}
}

func TestSetAndClearBanner(t *testing.T) {
	s := newTestService(t)
	uid := seedUser(t, s, "u1", "alice", false)

	oldKey, err := s.setBanner(uid, "banners/u1/a.png", "http://cdn/banner-a.png")
	if err != nil {
		t.Fatalf("setBanner: %v", err)
	}
	if oldKey != nil {
		t.Errorf("oldKey = %v, want nil on first set", *oldKey)
	}
	got := reload(t, s, uid)
	if got.BannerURL == nil || *got.BannerURL != "http://cdn/banner-a.png" {
		t.Errorf("persisted BannerURL = %v", got.BannerURL)
	}
	if got.BannerObjectKey == nil || *got.BannerObjectKey != "banners/u1/a.png" {
		t.Errorf("persisted BannerObjectKey = %v", got.BannerObjectKey)
	}

	oldKey, err = s.clearBanner(uid)
	if err != nil {
		t.Fatalf("clearBanner: %v", err)
	}
	if oldKey == nil || *oldKey != "banners/u1/a.png" {
		t.Errorf("clearBanner oldKey = %v, want banners/u1/a.png", oldKey)
	}
	if got := reload(t, s, uid); got.BannerURL != nil || got.BannerObjectKey != nil {
		t.Errorf("banner not cleared: url=%v key=%v", got.BannerURL, got.BannerObjectKey)
	}
}
