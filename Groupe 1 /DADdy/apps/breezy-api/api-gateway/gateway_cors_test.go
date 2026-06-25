package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestWithCORSNoOrigin covers the origin=="" branch inside withCORS:
//   - OPTIONS without an Origin header → 204 (early abort)
//   - Non-OPTIONS without an Origin header → passes through (c.Next())
func TestWithCORSNoOrigin(t *testing.T) {
	router := gin.New()
	router.Use(withCORS([]string{"http://localhost:5173"}))
	router.OPTIONS("/x", func(c *gin.Context) {})
	router.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	t.Run("OPTIONS without Origin returns 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("GET without Origin passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}

// TestWithCORSWildcard covers the containsWildcard branch inside withCORS:
// when "*" is in allowedOrigins, the response header is "Access-Control-Allow-Origin: *"
// rather than the echoed origin.
func TestWithCORSWildcard(t *testing.T) {
	router := gin.New()
	router.Use(withCORS([]string{"*"}))
	router.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://anywhere.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

// TestWithCSRFNoOriginWithXRW covers the path where no Origin header is present
// but X-Requested-With: XMLHttpRequest IS present — the request should pass
// through (c.Next()) with 200.
func TestWithCSRFNoOriginWithXRW(t *testing.T) {
	router := gin.New()
	router.Use(withCSRF([]string{"http://localhost:5173"}))
	router.POST("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (no Origin + XRW present)", w.Code)
	}
}
