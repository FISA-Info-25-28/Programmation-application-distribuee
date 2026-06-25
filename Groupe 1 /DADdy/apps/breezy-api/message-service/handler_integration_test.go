//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

type reqOpt struct {
	userID string
	body   string
}

func do(t *testing.T, router *gin.Engine, method, path string, opt reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(opt.body))
	if opt.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if opt.userID != "" {
		req.Header.Set(shared.HeaderUserID, opt.userID)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return m
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestRequiresIdentity(t *testing.T) {
	_, router := newTestApp(t)
	if w := do(t, router, http.MethodGet, "/conversations", reqOpt{}); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestGetOrCreateConversation(t *testing.T) {
	_, router := newTestApp(t)

	t.Run("self conversation is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/conversations", reqOpt{userID: "alice", body: `{"participant_id":"alice"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing participant_id is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/conversations", reqOpt{userID: "alice", body: `{"participant_id":"  "}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("creates then returns the same conversation (idempotent)", func(t *testing.T) {
		first := do(t, router, http.MethodPost, "/conversations", reqOpt{userID: "alice", body: `{"participant_id":"bob"}`})
		if first.Code != http.StatusCreated {
			t.Fatalf("first status = %d, want 201 (body %s)", first.Code, first.Body.String())
		}
		id1 := decode(t, first)["data"].(map[string]any)["id"]

		// Deuxième appel (même paire, sens inverse) → 200 et MÊME conversation.
		second := do(t, router, http.MethodPost, "/conversations", reqOpt{userID: "bob", body: `{"participant_id":"alice"}`})
		if second.Code != http.StatusOK {
			t.Fatalf("second status = %d, want 200", second.Code)
		}
		id2 := decode(t, second)["data"].(map[string]any)["id"]
		if id1 != id2 {
			t.Errorf("conversation ids differ: %v != %v (should reuse)", id1, id2)
		}
	})
}

func TestSendMessageAccessControl(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")

	t.Run("non-participant is forbidden", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/conversations/"+itoa(convID)+"/messages", reqOpt{userID: "intruder", body: `{"content":"hi"}`})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("missing conversation is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/conversations/999999/messages", reqOpt{userID: "alice", body: `{"content":"hi"}`})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("invalid conversation id is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/conversations/abc/messages", reqOpt{userID: "alice", body: `{"content":"hi"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestSendMessageValidation(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")
	path := "/conversations/" + itoa(convID) + "/messages"

	t.Run("empty content without media is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, path, reqOpt{userID: "alice", body: `{"content":"   "}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("content over 2000 chars is 400", func(t *testing.T) {
		long := strings.Repeat("a", 2001)
		if w := do(t, router, http.MethodPost, path, reqOpt{userID: "alice", body: `{"content":"` + long + `"}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("media without valid media_type is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, path, reqOpt{userID: "alice", body: `{"media_url":"http://x/y.bin","media_type":"pdf"}`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// Le contenu est chiffré en base mais l'API renvoie le clair (round-trip AES-GCM).
func TestSendMessageEncryptionRoundTrip(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")

	w := do(t, router, http.MethodPost, "/conversations/"+itoa(convID)+"/messages", reqOpt{userID: "alice", body: `{"content":"secret hello"}`})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	// La réponse API contient le clair.
	if got := decode(t, w)["data"].(map[string]any)["content"]; got != "secret hello" {
		t.Errorf("response content = %v, want plaintext", got)
	}
	// La ligne en base est chiffrée (préfixe enc:v1:), jamais le clair.
	var stored messageModel
	if err := a.db.Where("conversation_id = ?", convID).First(&stored).Error; err != nil {
		t.Fatalf("load stored message: %v", err)
	}
	if !strings.HasPrefix(stored.Content, "enc:v1:") {
		t.Errorf("stored content not encrypted: %q", stored.Content)
	}
	if strings.Contains(stored.Content, "secret hello") {
		t.Error("plaintext leaked into the database")
	}
}

func TestListMessagesAndUnread(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")
	send := "/conversations/" + itoa(convID) + "/messages"

	// Alice envoie 2 messages.
	for _, txt := range []string{"first", "second"} {
		if w := do(t, router, http.MethodPost, send, reqOpt{userID: "alice", body: `{"content":"` + txt + `"}`}); w.Code != http.StatusCreated {
			t.Fatalf("send %q: status %d", txt, w.Code)
		}
	}

	t.Run("participant lists messages", func(t *testing.T) {
		w := do(t, router, http.MethodGet, send, reqOpt{userID: "bob"})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if data, ok := decode(t, w)["data"].([]any); !ok || len(data) != 2 {
			t.Errorf("messages = %v, want 2", decode(t, w)["data"])
		}
	})

	t.Run("non-participant cannot list", func(t *testing.T) {
		if w := do(t, router, http.MethodGet, send, reqOpt{userID: "intruder"}); w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("unread count reflects the other party's messages, then resets on read", func(t *testing.T) {
		// Bob a 2 messages non lus d'Alice.
		w := do(t, router, http.MethodGet, "/conversations/"+itoa(convID), reqOpt{userID: "bob"})
		conv := decode(t, w)["data"].(map[string]any)
		if conv["unread_count"] != float64(2) {
			t.Errorf("unread_count = %v, want 2", conv["unread_count"])
		}

		// Bob marque comme lu → 204.
		if rw := do(t, router, http.MethodPut, "/conversations/"+itoa(convID)+"/read", reqOpt{userID: "bob"}); rw.Code != http.StatusNoContent {
			t.Fatalf("mark read status = %d, want 204", rw.Code)
		}

		w2 := do(t, router, http.MethodGet, "/conversations/"+itoa(convID), reqOpt{userID: "bob"})
		conv2 := decode(t, w2)["data"].(map[string]any)
		if conv2["unread_count"] != float64(0) {
			t.Errorf("unread_count after read = %v, want 0", conv2["unread_count"])
		}
	})
}

func TestListConversations(t *testing.T) {
	a, router := newTestApp(t)
	seedConversation(t, a, "alice", "bob")
	seedConversation(t, a, "alice", "carol")
	seedConversation(t, a, "bob", "carol") // alice n'en fait pas partie

	w := do(t, router, http.MethodGet, "/conversations", reqOpt{userID: "alice"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if data, ok := decode(t, w)["data"].([]any); !ok || len(data) != 2 {
		t.Errorf("conversations for alice = %v, want 2", decode(t, w)["data"])
	}
}

func TestGetConversationAccessControl(t *testing.T) {
	a, router := newTestApp(t)
	convID := seedConversation(t, a, "alice", "bob")

	if w := do(t, router, http.MethodGet, "/conversations/"+itoa(convID), reqOpt{userID: "alice"}); w.Code != http.StatusOK {
		t.Fatalf("participant status = %d, want 200", w.Code)
	}
	if w := do(t, router, http.MethodGet, "/conversations/"+itoa(convID), reqOpt{userID: "intruder"}); w.Code != http.StatusForbidden {
		t.Fatalf("non-participant status = %d, want 403", w.Code)
	}
}
