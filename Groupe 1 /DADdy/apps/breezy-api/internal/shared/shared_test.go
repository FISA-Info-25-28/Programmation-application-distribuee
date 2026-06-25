package shared

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestNewHealthResponse(t *testing.T) {
	t.Parallel()
	resp := NewHealthResponse("core-api")
	if resp.Status != "ok" {
		t.Errorf("Status = %q, want ok", resp.Status)
	}
	if resp.Service != "core-api" {
		t.Errorf("Service = %q, want core-api", resp.Service)
	}
	if resp.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestIsStaff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want bool
	}{
		{RoleModerator, true},
		{RoleAdministrator, true},
		{"  moderator  ", true}, // trim
		{RoleUser, false},
		{"", false},
		{"anything", false},
	}
	for _, tt := range tests {
		if got := IsStaff(tt.role); got != tt.want {
			t.Errorf("IsStaff(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

// newCtx fabrique un contexte gin avec les headers d'identité fournis.
func newCtx(headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	return c, w
}

func TestIdentityFromContext(t *testing.T) {
	t.Parallel()

	t.Run("absent userId => not ok", func(t *testing.T) {
		t.Parallel()
		c, _ := newCtx(nil)
		if _, ok := IdentityFromContext(c); ok {
			t.Error("expected ok=false when no user id header")
		}
	})

	t.Run("blank userId => not ok", func(t *testing.T) {
		t.Parallel()
		c, _ := newCtx(map[string]string{HeaderUserID: "   "})
		if _, ok := IdentityFromContext(c); ok {
			t.Error("expected ok=false when blank user id header")
		}
	})

	t.Run("present => trimmed identity", func(t *testing.T) {
		t.Parallel()
		c, _ := newCtx(map[string]string{
			HeaderUserID:       " usr_1 ",
			HeaderUserRole:     " moderator ",
			HeaderUserUsername: " alice ",
		})
		id, ok := IdentityFromContext(c)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if id.UserID != "usr_1" || id.Role != "moderator" || id.Username != "alice" {
			t.Errorf("identity = %+v, want trimmed values", id)
		}
	})
}

func TestAbortError(t *testing.T) {
	t.Parallel()
	c, w := newCtx(nil)
	AbortError(c, http.StatusNotFound, ErrNotFound, "missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var body ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != ErrNotFound || body.Message != "missing" {
		t.Errorf("body = %+v", body)
	}
	if !c.IsAborted() {
		t.Error("context should be aborted")
	}
}

func TestRequireIdentity(t *testing.T) {
	t.Parallel()

	t.Run("401 when unauthenticated", func(t *testing.T) {
		t.Parallel()
		c, w := newCtx(nil)
		RequireIdentity()(c)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
		if !c.IsAborted() {
			t.Error("should abort")
		}
	})

	t.Run("passes and stores identity", func(t *testing.T) {
		t.Parallel()
		c, _ := newCtx(map[string]string{HeaderUserID: "usr_1"})
		RequireIdentity()(c)
		if c.IsAborted() {
			t.Fatal("should not abort with valid identity")
		}
		got, exists := c.Get("identity")
		if !exists {
			t.Fatal("identity not set in context")
		}
		if got.(Identity).UserID != "usr_1" {
			t.Errorf("stored identity = %+v", got)
		}
	})
}

func TestRequireRole(t *testing.T) {
	t.Parallel()

	t.Run("401 when unauthenticated", func(t *testing.T) {
		t.Parallel()
		c, w := newCtx(nil)
		RequireRole(RoleAdministrator)(c)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("403 when role not allowed", func(t *testing.T) {
		t.Parallel()
		c, w := newCtx(map[string]string{HeaderUserID: "usr_1", HeaderUserRole: RoleUser})
		RequireRole(RoleModerator, RoleAdministrator)(c)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("passes when role allowed", func(t *testing.T) {
		t.Parallel()
		c, _ := newCtx(map[string]string{HeaderUserID: "usr_1", HeaderUserRole: RoleModerator})
		RequireRole(RoleModerator, RoleAdministrator)(c)
		if c.IsAborted() {
			t.Fatal("should not abort for allowed role")
		}
		if _, exists := c.Get("identity"); !exists {
			t.Error("identity should be set")
		}
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("returns trimmed value when set", func(t *testing.T) {
		t.Setenv("TEST_GETENV", "  value  ")
		if got := GetEnv("TEST_GETENV", "fallback"); got != "value" {
			t.Errorf("GetEnv = %q, want value", got)
		}
	})

	t.Run("returns fallback when blank", func(t *testing.T) {
		t.Setenv("TEST_GETENV_BLANK", "   ")
		if got := GetEnv("TEST_GETENV_BLANK", "fallback"); got != "fallback" {
			t.Errorf("GetEnv = %q, want fallback", got)
		}
	})

	t.Run("returns fallback when absent", func(t *testing.T) {
		if got := GetEnv("TEST_GETENV_ABSENT_XYZ", "fallback"); got != "fallback" {
			t.Errorf("GetEnv = %q, want fallback", got)
		}
	})
}

func TestGetEnvAny(t *testing.T) {
	t.Run("first non-empty key wins", func(t *testing.T) {
		t.Setenv("TEST_KEY_A", "")
		t.Setenv("TEST_KEY_B", "second")
		if got := GetEnvAny([]string{"TEST_KEY_A", "TEST_KEY_B"}, "fallback"); got != "second" {
			t.Errorf("GetEnvAny = %q, want second", got)
		}
	})

	t.Run("fallback when none set", func(t *testing.T) {
		if got := GetEnvAny([]string{"TEST_NONE_1", "TEST_NONE_2"}, "fallback"); got != "fallback" {
			t.Errorf("GetEnvAny = %q, want fallback", got)
		}
	})
}

func TestGetEnvIntMin(t *testing.T) {
	t.Run("valid value above minimum", func(t *testing.T) {
		t.Setenv("TEST_INT", "42")
		if got := GetEnvIntMin("TEST_INT", 10, 1); got != 42 {
			t.Errorf("GetEnvIntMin = %d, want 42", got)
		}
	})

	t.Run("below minimum falls back", func(t *testing.T) {
		t.Setenv("TEST_INT_LOW", "0")
		if got := GetEnvIntMin("TEST_INT_LOW", 10, 5); got != 10 {
			t.Errorf("GetEnvIntMin = %d, want 10", got)
		}
	})

	t.Run("non-parsable falls back", func(t *testing.T) {
		t.Setenv("TEST_INT_BAD", "abc")
		if got := GetEnvIntMin("TEST_INT_BAD", 10, 1); got != 10 {
			t.Errorf("GetEnvIntMin = %d, want 10", got)
		}
	})

	t.Run("absent falls back", func(t *testing.T) {
		if got := GetEnvIntMin("TEST_INT_ABSENT_XYZ", 10, 1); got != 10 {
			t.Errorf("GetEnvIntMin = %d, want 10", got)
		}
	})
}

func TestIsProduction(t *testing.T) {
	t.Run("true for production case-insensitive", func(t *testing.T) {
		t.Setenv("APP_ENV", "Production")
		if !IsProduction() {
			t.Error("expected IsProduction true")
		}
	})

	t.Run("false otherwise", func(t *testing.T) {
		t.Setenv("APP_ENV", "dev")
		if IsProduction() {
			t.Error("expected IsProduction false")
		}
	})
}

func TestSecretEnv(t *testing.T) {
	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("TEST_SECRET", "s3cr3t")
		if got := SecretEnv("TEST_SECRET", "dev"); got != "s3cr3t" {
			t.Errorf("SecretEnv = %q, want s3cr3t", got)
		}
	})

	t.Run("dev fallback when not production", func(t *testing.T) {
		t.Setenv("APP_ENV", "dev")
		if got := SecretEnv("TEST_SECRET_ABSENT_XYZ", "dev-fallback"); got != "dev-fallback" {
			t.Errorf("SecretEnv = %q, want dev-fallback", got)
		}
	})

	t.Run("panics in production when absent", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when secret missing in production")
			}
		}()
		SecretEnv("TEST_SECRET_ABSENT_PROD_XYZ", "dev-fallback")
	})
}

func TestConnectPostgresEmptyDSN(t *testing.T) {
	t.Parallel()
	if _, err := ConnectPostgres("   "); err == nil {
		t.Error("expected error for empty DSN")
	}
}
