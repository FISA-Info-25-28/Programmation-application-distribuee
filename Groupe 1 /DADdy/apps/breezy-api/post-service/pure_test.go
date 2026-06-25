package main

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractHashtags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "none", in: "just some text", want: nil},
		{name: "simple", in: "hello #World", want: []string{"world"}},
		{name: "lowercased and deduplicated", in: "#Go #go #GO rocks", want: []string{"go"}},
		{name: "multiple distinct", in: "#a loves #b", want: []string{"a", "b"}},
		{name: "unicode letters", in: "#café #naïve", want: []string{"café", "naïve"}},
		{name: "underscore and digits", in: "#go_1 #2026", want: []string{"go_1", "2026"}},
		{name: "bare hash ignored", in: "# not a tag", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractHashtags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractHashtags(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestToPostResponse(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 6, 18, 9, 30, 0, 0, time.UTC)
	p := postModel{
		ID: 5, Content: "hi", AuthorID: "usr_1", AuthorUsername: "alice",
		LikesCount: 3, CommentsCount: 1, RebreezeCount: 2, CreatedAt: created,
	}
	resp := toPostResponse(p)

	if resp.ID != 5 || resp.Content != "hi" || resp.AuthorUsername != "alice" {
		t.Errorf("fields not mapped: %+v", resp)
	}
	if resp.LikesCount != 3 || resp.CommentsCount != 1 || resp.RebreezeCount != 2 {
		t.Errorf("counters not mapped: %+v", resp)
	}
	if resp.CreatedAt != "2026-06-18T09:30:00Z" {
		t.Errorf("CreatedAt = %q, want RFC3339 UTC", resp.CreatedAt)
	}
	// Les slices doivent être non-nil (sérialisées en [] et non null).
	if resp.Media == nil || resp.Hashtags == nil || resp.Mentions == nil {
		t.Errorf("slices should be non-nil: media=%v hashtags=%v mentions=%v", resp.Media, resp.Hashtags, resp.Mentions)
	}
}

func TestPostIDs(t *testing.T) {
	t.Parallel()
	got := postIDs([]postModel{{ID: 1}, {ID: 2}, {ID: 3}})
	want := []int64{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("postIDs = %v, want %v", got, want)
	}
	if len(postIDs(nil)) != 0 {
		t.Error("postIDs(nil) should be empty")
	}
}
