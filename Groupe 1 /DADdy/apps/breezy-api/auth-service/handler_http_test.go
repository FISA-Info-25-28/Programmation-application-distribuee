//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

func init() { gin.SetMode(gin.TestMode) }

// reqOpt configure une requête de test (identité via headers, corps JSON,
// en-têtes additionnels comme X-Internal-Key).
type reqOpt struct {
	userID  string
	role    string
	body    string
	headers map[string]string
}

// do exécute une requête sur le routeur réel et renvoie le ResponseRecorder.
func do(t *testing.T, router *gin.Engine, method, path string, opt reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(opt.body))
	if opt.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if opt.userID != "" {
		req.Header.Set(shared.HeaderUserID, opt.userID)
	}
	if opt.role != "" {
		req.Header.Set(shared.HeaderUserRole, opt.role)
	}
	for k, v := range opt.headers {
		req.Header.Set(k, v)
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

func newTestRouter(t *testing.T) (*gin.Engine, *authService) {
	t.Helper()
	s := newTestService(t)
	return newAuthRouter(s), s
}

func TestHTTPRegister(t *testing.T) {
	router, _ := newTestRouter(t)

	t.Run("valid registration returns 201", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/register", reqOpt{
			body: `{"username":"alice","email":"alice@example.com","password":"` + strongPassword + `","termsAccepted":true,"termsVersion":"` + currentTermsVersion + `"}`,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body %s)", w.Code, w.Body.String())
		}
		if _, ok := decode(t, w)["message"]; !ok {
			t.Error("expected a message field")
		}
	})

	t.Run("duplicate returns 409", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/register", reqOpt{
			body: `{"username":"alice","email":"other@example.com","password":"` + strongPassword + `","termsAccepted":true,"termsVersion":"` + currentTermsVersion + `"}`,
		})
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/register", reqOpt{body: `{not json`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/register", reqOpt{
			body: `{"username":"ab","email":"bad@example.com","password":"` + strongPassword + `","termsAccepted":true,"termsVersion":"` + currentTermsVersion + `"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestHTTPLogin(t *testing.T) {
	router, s := newTestRouter(t)
	seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)
	seedVerifiedUser(t, s, "banned", "banned@example.com", strongPassword, accountStatusBanned)

	t.Run("success returns 200 with tokens", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"alice","password":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := decode(t, w)
		if body["accessToken"] == "" || body["accessToken"] == nil {
			t.Error("accessToken missing")
		}
		if body["refreshToken"] == "" || body["refreshToken"] == nil {
			t.Error("refreshToken missing")
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"alice","password":"WrongPass1!"}`,
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("banned account returns 403", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"banned","password":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})
}

func TestHTTPRefreshAndLogout(t *testing.T) {
	router, s := newTestRouter(t)
	seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	login := do(t, router, http.MethodPost, "/auth/login", reqOpt{
		body: `{"identifier":"alice","password":"` + strongPassword + `"}`,
	})
	refreshToken, _ := decode(t, login)["refreshToken"].(string)
	if refreshToken == "" {
		t.Fatal("setup: no refresh token from login")
	}

	t.Run("refresh returns 200 with rotated tokens", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
			body: `{"refreshToken":"` + refreshToken + `"}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if newTok, _ := decode(t, w)["refreshToken"].(string); newTok == refreshToken {
			t.Error("refresh token was not rotated")
		}
	})

	t.Run("invalid refresh token returns 401", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{body: `{"refreshToken":"nope"}`})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("logout returns 204", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/logout", reqOpt{body: `{"refreshToken":"anything"}`})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestHTTPMeRequiresIdentity(t *testing.T) {
	router, s := newTestRouter(t)
	userID := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	t.Run("without identity returns 401", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/auth/me", reqOpt{})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("with identity returns the profile", func(t *testing.T) {
		w := do(t, router, http.MethodGet, "/auth/me", reqOpt{userID: userID})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if decode(t, w)["username"] != "alice" {
			t.Errorf("username = %v, want alice", decode(t, w)["username"])
		}
	})
}

func TestHTTPChangePassword(t *testing.T) {
	router, s := newTestRouter(t)
	userID := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	t.Run("without identity returns 401", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
			body: `{"currentPassword":"x","newPassword":"y"}`,
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("wrong current password returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
			userID: userID,
			body:   `{"currentPassword":"WrongCurrent1!","newPassword":"Even!Str0ngerPass"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("success returns 200", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
			userID: userID,
			body:   `{"currentPassword":"` + strongPassword + `","newPassword":"Even!Str0ngerPass"}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})
}

// Les endpoints anti-énumération répondent toujours 202, que le compte existe ou non.
func TestHTTPAntiEnumeration(t *testing.T) {
	router, s := newTestRouter(t)
	seedVerifiedUser(t, s, "known", "known@example.com", strongPassword, accountStatusActive)

	cases := []struct {
		name  string
		path  string
		email string
	}{
		{name: "reset known email", path: "/auth/request-password-reset", email: "known@example.com"},
		{name: "reset unknown email", path: "/auth/request-password-reset", email: "ghost@example.com"},
		{name: "resend known email", path: "/auth/resend-verification", email: "known@example.com"},
		{name: "resend unknown email", path: "/auth/resend-verification", email: "ghost@example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(t, router, http.MethodPost, tc.path, reqOpt{body: `{"email":"` + tc.email + `"}`})
			if w.Code != http.StatusAccepted {
				t.Fatalf("status = %d, want 202 (anti-enumeration)", w.Code)
			}
		})
	}
}

func TestHTTPInternalSetStatus(t *testing.T) {
	router, s := newTestRouter(t)
	userID := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	t.Run("missing internal key is forbidden", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/internal/users/"+userID+"/status", reqOpt{
			body: `{"status":"banned"}`,
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("valid key updates status (204)", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/internal/users/"+userID+"/status", reqOpt{
			body:    `{"status":"suspended"}`,
			headers: map[string]string{"X-Internal-Key": "test-internal-key"},
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		if loadUser(t, s, userID).Status != accountStatusSuspended {
			t.Error("status was not updated to suspended")
		}
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/internal/users/"+userID+"/status", reqOpt{
			body:    `{"status":"bogus"}`,
			headers: map[string]string{"X-Internal-Key": "test-internal-key"},
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unknown user returns 404", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/internal/users/ghost/status", reqOpt{
			body:    `{"status":"banned"}`,
			headers: map[string]string{"X-Internal-Key": "test-internal-key"},
		})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}
