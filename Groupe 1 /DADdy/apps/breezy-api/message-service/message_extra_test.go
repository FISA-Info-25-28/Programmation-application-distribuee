//go:build integration

package main

import (
	"net/http"
	"testing"
)

// TestListConversationsEmpty verifies that a user who has no conversations
// receives 200 with an empty data array (covers the len(myParts)==0 branch
// in listConversations).
func TestListConversationsEmpty(t *testing.T) {
	_, router := newTestApp(t)

	w := do(t, router, http.MethodGet, "/conversations", reqOpt{userID: "newcomer"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := decode(t, w)
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("data field missing or wrong type: %v", body["data"])
	}
	if len(data) != 0 {
		t.Errorf("len(data) = %d, want 0 (no conversations)", len(data))
	}
}

// TestGetOrCreateConversationInvalidBody covers the ShouldBindJSON error branch
// in getOrCreateConversation.
func TestGetOrCreateConversationInvalidBody(t *testing.T) {
	_, router := newTestApp(t)

	w := do(t, router, http.MethodPost, "/conversations", reqOpt{userID: "alice", body: `{bad`})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestGetConversationInvalidID covers parseConvID returning false in getConversation.
func TestGetConversationInvalidID(t *testing.T) {
	_, router := newTestApp(t)

	w := do(t, router, http.MethodGet, "/conversations/abc", reqOpt{userID: "alice"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestListMessagesInvalidID covers parseConvID returning false in listMessages.
func TestListMessagesInvalidID(t *testing.T) {
	_, router := newTestApp(t)

	w := do(t, router, http.MethodGet, "/conversations/abc/messages", reqOpt{userID: "alice"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestMarkReadInvalidID covers parseConvID returning false in markRead.
func TestMarkReadInvalidID(t *testing.T) {
	_, router := newTestApp(t)

	w := do(t, router, http.MethodPut, "/conversations/abc/read", reqOpt{userID: "alice"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestSendMessageInvalidBody covers the ShouldBindJSON error branch in sendMessage.
func TestSendMessageInvalidBody(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")
	path := "/conversations/" + itoa(convID) + "/messages"

	w := do(t, router, http.MethodPost, path, reqOpt{userID: "alice", body: `{bad`})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestSendMessageMediaURLWithoutMediaType covers the false branch of
// `if body.MediaType != nil` inside sendMessage: when media_url is present
// but media_type is absent, mt stays "" and fails validation → 400.
func TestSendMessageMediaURLWithoutMediaType(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")
	path := "/conversations/" + itoa(convID) + "/messages"

	w := do(t, router, http.MethodPost, path, reqOpt{
		userID: "alice",
		body:   `{"media_url":"https://cdn.example.com/pic.jpg"}`,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (media_url without media_type)", w.Code)
	}
}

// TestSendMessageWithMedia exercises the media success path in sendMessage:
// the message is created with media_url and media_type="image" and returned
// in the response.
func TestSendMessageWithMedia(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")
	path := "/conversations/" + itoa(convID) + "/messages"

	mediaURL := "https://cdn.example.com/pic.jpg"
	body := `{"media_url":"` + mediaURL + `","media_type":"image"}`
	w := do(t, router, http.MethodPost, path, reqOpt{userID: "alice", body: body})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	data, ok := decode(t, w)["data"].(map[string]any)
	if !ok {
		t.Fatalf("data field missing or wrong type")
	}
	if got := data["media_url"]; got != mediaURL {
		t.Errorf("media_url = %v, want %q", got, mediaURL)
	}
	if got := data["media_type"]; got != "image" {
		t.Errorf("media_type = %v, want image", got)
	}
}
