//go:build integration

package main

import "testing"

// seedHashtagAffinity donne à un user de l'affinité pour un hashtag (co-engagement
// simulé), matière première de la matrice de proximité puis des clusters.
func seedHashtagAffinity(t *testing.T, a *app, userID, tag string, weight float64) {
	t.Helper()
	if err := a.db.Create(&userAffinityModel{UserID: userID, Kind: affinityKindHashtag, Ref: tag, Weight: weight}).Error; err != nil {
		t.Fatalf("seed hashtag affinity %s/%s: %v", userID, tag, err)
	}
}

func TestEmergentClustersFromCoEngagement(t *testing.T) {
	a, _ := newTestApp(t)
	// Groupe « sport » : 3 users engagent #messi, #sport, #foot ensemble.
	for _, u := range []string{"s1", "s2", "s3"} {
		seedHashtagAffinity(t, a, u, "messi", 5)
		seedHashtagAffinity(t, a, u, "sport", 8)
		seedHashtagAffinity(t, a, u, "foot", 4)
	}
	// Groupe « cuisine » : 3 autres users engagent #cuisine, #ramen.
	for _, u := range []string{"c1", "c2", "c3"} {
		seedHashtagAffinity(t, a, u, "cuisine", 6)
		seedHashtagAffinity(t, a, u, "ramen", 5)
	}

	a.refreshHashtagNeighbors()
	a.refreshHashtagClusters()

	sportTheme := clusterThemeForTags(a.db, []string{"messi"})
	if sportTheme == nil {
		t.Fatal("#messi devrait appartenir à un cluster émergent")
	}
	// #messi et #sport émergent dans le MÊME cluster (proximité par co-engagement).
	if got := clusterThemeForTags(a.db, []string{"sport"}); got == nil || *got != *sportTheme {
		t.Errorf("#messi et #sport devraient partager un cluster : %v vs %v", sportTheme, got)
	}
	// …et #cuisine dans un cluster différent.
	if got := clusterThemeForTags(a.db, []string{"cuisine"}); got != nil && *got == *sportTheme {
		t.Error("#cuisine ne devrait pas être dans le cluster sport")
	}
}

func TestThemeAssignedFromCluster(t *testing.T) {
	a, _ := newTestApp(t)
	for _, u := range []string{"s1", "s2", "s3"} {
		seedHashtagAffinity(t, a, u, "messi", 5)
		seedHashtagAffinity(t, a, u, "sport", 8)
	}
	// Post ne portant que #messi → thème déduit du cluster (pas d'un dictionnaire).
	p := postModel{Content: "allez #messi", AuthorID: "dan", AuthorUsername: "dan"}
	if err := a.db.Create(&p).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	upsertHashtags(a.db, p.ID, extractHashtags(p.Content))

	a.refreshHashtagNeighbors()
	a.refreshHashtagClusters()
	a.runThemeInferencePass()

	want := clusterThemeForTags(a.db, []string{"messi"})
	got := reloadPost(t, a, p.ID)
	if want == nil || got.Theme == nil || *got.Theme != *want || !got.ThemeInferred {
		t.Errorf("post theme=%v inferred=%v, want %v", got.Theme, got.ThemeInferred, want)
	}
}
