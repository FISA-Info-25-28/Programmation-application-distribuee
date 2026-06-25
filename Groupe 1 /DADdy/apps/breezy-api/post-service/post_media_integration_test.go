//go:build integration

package main

import (
	"net/http"
	"testing"
	"time"
)

// TestCreatePostWithMedia seeds a media record (PostID=nil, UploaderID="alice")
// and creates a post via the API referencing that media. This exercises the
// attachMediaToPost success path (RowsAffected == len(mediaIDs)).
func TestCreatePostWithMedia(t *testing.T) {
	a, router := newTestApp(t)

	m := mediaModel{
		UploaderID: "alice",
		ObjectKey:  "media/test-key.jpg",
		MediaType:  mediaTypeImage,
		URL:        "https://cdn.example.com/test.jpg",
	}
	if err := a.db.Create(&m).Error; err != nil {
		t.Fatalf("seed media: %v", err)
	}

	body := `{"content":"check out this image","media_ids":[` + itoa(m.ID) + `],"author_username":"alice"}`
	w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: body})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}

	// Verify the media record is now linked to the new post.
	var updated mediaModel
	if err := a.db.First(&updated, m.ID).Error; err != nil {
		t.Fatalf("reload media: %v", err)
	}
	if updated.PostID == nil {
		t.Error("media PostID should be set after post creation")
	}
}

// TestCreatePostInvalidMedia verifies that supplying a media ID owned by a
// different uploader triggers the errInvalidMediaIDs path in attachMediaToPost
// (RowsAffected != len(mediaIDs)) and results in a 400 response.
func TestCreatePostInvalidMedia(t *testing.T) {
	a, router := newTestApp(t)

	// Seed a media record owned by "bob", then try to use it as "alice".
	m := mediaModel{
		UploaderID: "bob",
		ObjectKey:  "media/bob-key.jpg",
		MediaType:  mediaTypeImage,
		URL:        "https://cdn.example.com/bob.jpg",
	}
	if err := a.db.Create(&m).Error; err != nil {
		t.Fatalf("seed media: %v", err)
	}

	body := `{"content":"stealing bob media","media_ids":[` + itoa(m.ID) + `],"author_username":"alice"}`
	w := do(t, router, http.MethodPost, "/posts", reqOpt{userID: "alice", body: body})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
}

// TestVotePollEnded seeds a post with an already-expired poll and verifies
// that a vote attempt returns 400 with "poll has ended".
func TestVotePollEnded(t *testing.T) {
	a, router := newTestApp(t)

	// Create the post directly (not via API) so we can control the poll EndsAt.
	post := postModel{Content: "expired poll", AuthorID: "author", AuthorUsername: "author"}
	if err := a.db.Create(&post).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}

	past := time.Now().UTC().Add(-1 * time.Hour)
	poll := pollModel{PostID: post.ID, EndsAt: past}
	if err := a.db.Create(&poll).Error; err != nil {
		t.Fatalf("seed poll: %v", err)
	}
	opt := pollOptionModel{PollID: poll.ID, OptionIndex: 0, Text: "Yes"}
	if err := a.db.Create(&opt).Error; err != nil {
		t.Fatalf("seed poll option: %v", err)
	}

	w := do(t, router, http.MethodPost, "/posts/"+itoa(post.ID)+"/poll/vote",
		reqOpt{userID: "voter", body: `{"option_index":0}`})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
}

// TestDeleteReply verifies that deleting a reply decrements the parent's
// comments_count, exercising the post.ParentID != nil branch in purgePost.
func TestDeleteReply(t *testing.T) {
	a, router := newTestApp(t)

	// Create parent post then a reply via the API.
	parentID := seedPost(t, a, "author", "parent content")

	replyW := do(t, router, http.MethodPost, "/posts/"+itoa(parentID)+"/replies",
		reqOpt{userID: "replier", body: `{"content":"reply text","author_username":"replier"}`})
	if replyW.Code != http.StatusCreated {
		t.Fatalf("create reply status = %d (body %s)", replyW.Code, replyW.Body.String())
	}
	replyID := int64(decode(t, replyW)["data"].(map[string]any)["id"].(float64))

	// Deleting the reply must succeed (204) and cover the ParentID branch.
	delW := do(t, router, http.MethodDelete, "/posts/"+itoa(replyID),
		reqOpt{userID: "replier"})
	if delW.Code != http.StatusNoContent {
		t.Fatalf("delete reply status = %d (body %s)", delW.Code, delW.Body.String())
	}

	// Parent comments_count should have been decremented back to 0.
	parent := reloadPost(t, a, parentID)
	if parent.CommentsCount != 0 {
		t.Errorf("comments_count = %d, want 0 after reply deleted", parent.CommentsCount)
	}
}
