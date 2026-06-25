package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestToPostID(t *testing.T) {
	t.Parallel()
	if got := toPostID(42); got != "42" {
		t.Errorf("toPostID(42) = %q, want 42", got)
	}
	if got := toPostID(0); got != "0" {
		t.Errorf("toPostID(0) = %q, want 0", got)
	}
}

func TestPageParams(t *testing.T) {
	t.Parallel()

	newCtx := func(query string) *gin.Context {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		return c
	}

	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
	}{
		{name: "defaults", query: "", wantPage: 1, wantLimit: 20},
		{name: "explicit values", query: "page=3&limit=10", wantPage: 3, wantLimit: 10},
		{name: "page below 1 clamps", query: "page=0", wantPage: 1, wantLimit: 20},
		{name: "negative page clamps", query: "page=-5", wantPage: 1, wantLimit: 20},
		{name: "limit below 1 falls back", query: "limit=0", wantPage: 1, wantLimit: 20},
		{name: "limit over 100 falls back", query: "limit=500", wantPage: 1, wantLimit: 20},
		{name: "non-numeric falls back", query: "page=x&limit=y", wantPage: 1, wantLimit: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, limit := pageParams(newCtx(tt.query))
			if page != tt.wantPage || limit != tt.wantLimit {
				t.Fatalf("pageParams(%q) = (%d, %d), want (%d, %d)", tt.query, page, limit, tt.wantPage, tt.wantLimit)
			}
		})
	}
}
