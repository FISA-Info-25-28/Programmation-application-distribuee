//go:build integration

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newAppWithMocks construit une app branchée sur des user/notif-services factices,
// pour exercer resolveMentions (appel HTTP au user-service) et les notifications.
func newAppWithMocks(t *testing.T) (*app, *httptest.Server, *httptest.Server) {
	t.Helper()
	base, _ := newTestApp(t) // connexion + truncate

	userMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"usr_bob","username":"bob"}]}`))
	}))
	notifMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(func() { userMock.Close(); notifMock.Close() })

	a := &app{db: base.db, internalKey: "test-internal-key", userURL: userMock.URL, notifURL: notifMock.URL}
	return a, userMock, notifMock
}

func TestCreatePostResolvesMentions(t *testing.T) {
	a, _, _ := newAppWithMocks(t)
	router := newRouter(a)

	// Le contenu mentionne @bob (non fourni dans body.mentions) → resolveMentions
	// interroge le user-service factice qui renvoie l'ID de bob.
	w := do(t, router, http.MethodPost, "/posts", reqOpt{
		userID: "alice", body: `{"content":"hey @bob look at this","author_username":"alice"}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}

	data := decode(t, w)["data"].(map[string]any)
	postID := int64(data["id"].(float64))

	var count int64
	a.db.Model(&postMentionModel{}).Where("post_id = ?", postID).Count(&count)
	if count != 1 {
		t.Errorf("resolved mentions persisted = %d, want 1 (bob)", count)
	}
}

func TestCreateAndReadQuotePost(t *testing.T) {
	a, _, _ := newAppWithMocks(t)
	router := newRouter(a)

	quotedID := seedPost(t, a, "bob", "original thought")

	// Création d'un post qui en cite un autre → attachQuotedPost dans la réponse.
	w := do(t, router, http.MethodPost, "/posts", reqOpt{
		userID: "alice", body: `{"content":"great point","author_username":"alice","quote_id":` + itoa(quotedID) + `}`,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
	}
	data := decode(t, w)["data"].(map[string]any)
	quotingID := int64(data["id"].(float64))

	// Lecture : enrichPosts → embedQuotedPosts embarque le post cité.
	rw := do(t, router, http.MethodGet, "/posts/"+itoa(quotingID), reqOpt{userID: "alice"})
	if rw.Code != http.StatusOK {
		t.Fatalf("read status = %d, want 200", rw.Code)
	}
}
