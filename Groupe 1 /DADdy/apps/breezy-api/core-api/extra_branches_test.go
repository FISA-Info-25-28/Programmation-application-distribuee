//go:build integration

package main

import (
	"net/http"
	"testing"

	"daddy/apps/breezy-api/internal/models"
)

// TestDeletePostInvalidIDWithIdentity covers the ParseInt error path inside
// deletePost when the caller IS authenticated (the unauthenticated branch is
// covered by TestDeletePost).
func TestDeletePostInvalidIDWithIdentity(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")

	w := do(t, router, http.MethodDelete, "/posts/abc", reqOpt{userID: "usr_1"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestLikePostInvalidIDWithIdentity covers the ParseInt error path inside
// likePost when the caller IS authenticated.
func TestLikePostInvalidIDWithIdentity(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")

	w := do(t, router, http.MethodPost, "/posts/abc/like", reqOpt{userID: "usr_1"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestUnlikePostInvalidIDWithIdentity covers the ParseInt error path inside
// unlikePost when the caller IS authenticated.
func TestUnlikePostInvalidIDWithIdentity(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")

	w := do(t, router, http.MethodDelete, "/posts/abc/like", reqOpt{userID: "usr_1"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestCreateCommentInvalidIDWithIdentity covers the ParseInt error path inside
// createComment when the caller IS authenticated.
func TestCreateCommentInvalidIDWithIdentity(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")

	w := do(t, router, http.MethodPost, "/posts/abc/comments", reqOpt{
		userID: "usr_1", body: `{"content":"hi"}`,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestCreateCommentWithParentID covers req.ParentCommentID != nil inside
// createComment (parsing the parent id and setting comment.ParentCommentID)
// and the cmt.ParentCommentID != nil branch inside the getComments response
// loop (building parentIDStr).
func TestCreateCommentWithParentID(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	postID := seedPost(t, s, "usr_1", "hello")

	// Create a top-level comment; its id becomes the parent for the reply.
	w := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/comments", reqOpt{
		userID: "usr_1", body: `{"content":"parent comment"}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create parent comment status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	parentID, _ := decode(t, w)["id"].(string)

	// Create a reply referencing the parent comment (covers ParentCommentID != nil).
	replyW := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/comments", reqOpt{
		userID: "usr_1",
		body:   `{"content":"reply","parentCommentId":"` + parentID + `"}`,
	})
	if replyW.Code != http.StatusCreated {
		t.Fatalf("create reply status = %d, want 201 (body %s)", replyW.Code, replyW.Body.String())
	}
	replyData := decode(t, replyW)
	if gotParent, _ := replyData["parentCommentId"].(string); gotParent != parentID {
		t.Errorf("parentCommentId in create response = %v, want %q", replyData["parentCommentId"], parentID)
	}

	// List comments; the reply must carry parentCommentId, covering the
	// parentIDStr != nil branch in the getComments response loop.
	listW := do(t, router, http.MethodGet, "/posts/"+itoa(postID)+"/comments", reqOpt{})
	if listW.Code != http.StatusOK {
		t.Fatalf("list comments status = %d, want 200", listW.Code)
	}
	data, _ := decode(t, listW)["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("comments count = %d, want 2", len(data))
	}
	reply, _ := data[1].(map[string]any)
	if gotParent, _ := reply["parentCommentId"].(string); gotParent != parentID {
		t.Errorf("reply parentCommentId in list = %v, want %q", reply["parentCommentId"], parentID)
	}
}

// TestGetPostWithParentID seeds a post with ParentID set directly and fetches
// it to cover the post.ParentID != nil branch inside buildPostResponse.
func TestGetPostWithParentID(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	parentID := seedPost(t, s, "usr_1", "parent post")

	reply := models.Post{Content: "reply content", AuthorID: "usr_1", ParentID: &parentID}
	if err := s.db.Create(&reply).Error; err != nil {
		t.Fatalf("seed reply post: %v", err)
	}

	w := do(t, router, http.MethodGet, "/posts/"+itoa(reply.ID), reqOpt{})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	if decode(t, w)["parentId"] == nil {
		t.Error("parentId should be set in response for a reply post")
	}
}

// TestGetPostLikedByMe covers the hasLiked(db, userID, postID) true path
// by first liking a post then fetching it as the same user.
func TestGetPostLikedByMe(t *testing.T) {
	s, router := newTestServer(t)
	seedUser(t, s, "usr_1", "alice")
	id := seedPost(t, s, "usr_1", "hello")

	do(t, router, http.MethodPost, "/posts/"+itoa(id)+"/like", reqOpt{userID: "usr_1"})

	w := do(t, router, http.MethodGet, "/posts/"+itoa(id), reqOpt{userID: "usr_1"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if liked, _ := decode(t, w)["likedByMe"].(bool); !liked {
		t.Error("likedByMe should be true after liking the post")
	}
}

// TestPageParamsClamping covers the page < 1 and limit out-of-range branches
// inside pageParams (executed when feed is requested with invalid query params).
func TestPageParamsClamping(t *testing.T) {
	_, router := newTestServer(t)

	t.Run("page=0 is clamped to 1", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/feed?page=0", reqOpt{}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("limit=0 is clamped to 20", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/feed?limit=0", reqOpt{}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("limit=200 is clamped to 20", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, "/feed?limit=200", reqOpt{}); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}
