package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// TestWithRateLimitFailOpen verifies that withRateLimit is fail-open: when the
// Redis limiter is unreachable the request still goes through (200, not 429).
func TestWithRateLimitFailOpen(t *testing.T) {
	t.Parallel()
	// Port 1 is closed; the Redis client returns a connection-refused error on
	// every Allow() call, exercising the fail-open branch inside withRateLimit.
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	limiter := redis_rate.NewLimiter(client)

	router := gin.New()
	router.Use(withRateLimit(limiter, keyByIP("test"), 100))
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fail-open on Redis error)", w.Code)
	}
}

// TestKeyByUser exercises both branches of keyByUser: with and without a user ID.
func TestKeyByUser(t *testing.T) {
	t.Parallel()

	t.Run("with userID header uses user scope", func(t *testing.T) {
		t.Parallel()
		fn := keyByUser("scope")
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("X-User-Id", "usr_42")

		key := fn(c)
		if key != "scope:user:usr_42" {
			t.Errorf("key = %q, want scope:user:usr_42", key)
		}
	})

	t.Run("without userID header falls back to IP scope", func(t *testing.T) {
		t.Parallel()
		fn := keyByUser("scope")
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		key := fn(c)
		if key == "" {
			t.Error("key should not be empty")
		}
		if key == "scope:user:" {
			t.Errorf("key = %q, should fall back to IP scope, not empty user scope", key)
		}
	})
}
