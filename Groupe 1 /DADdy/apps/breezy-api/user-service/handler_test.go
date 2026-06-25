package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

// newQueryContext fabrique un *gin.Context portant la query string fournie, pour
// tester parsePagination sans démarrer de serveur.
func newQueryContext(query string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	return c
}

func TestParsePagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
	}{
		{name: "defaults when empty", query: "", wantPage: defaultPage, wantLimit: defaultLimit},
		{name: "valid values", query: "page=3&limit=10", wantPage: 3, wantLimit: 10},
		{name: "zero page falls back to default", query: "page=0", wantPage: defaultPage, wantLimit: defaultLimit},
		{name: "negative page falls back to default", query: "page=-5", wantPage: defaultPage, wantLimit: defaultLimit},
		{name: "non-numeric page falls back", query: "page=abc", wantPage: defaultPage, wantLimit: defaultLimit},
		{name: "zero limit falls back", query: "limit=0", wantPage: defaultPage, wantLimit: defaultLimit},
		{name: "limit capped at maxLimit", query: "limit=9999", wantPage: defaultPage, wantLimit: maxLimit},
		{name: "limit exactly at cap", query: "limit=100", wantPage: defaultPage, wantLimit: maxLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page, limit := parsePagination(newQueryContext(tt.query))
			if page != tt.wantPage || limit != tt.wantLimit {
				t.Fatalf("parsePagination(%q) = (%d, %d), want (%d, %d)",
					tt.query, page, limit, tt.wantPage, tt.wantLimit)
			}
		})
	}
}

func TestToUserResponse(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 6, 18, 9, 30, 0, 0, time.UTC)
	base := userModel{
		ID:        "usr_abc",
		Username:  "johndoe",
		Email:     "john@example.com",
		Role:      "user",
		Status:    "active",
		CreatedAt: created,
	}

	t.Run("excludes email when includeEmail false", func(t *testing.T) {
		t.Parallel()
		resp := toUserResponse(base, false, nil, nil, nil)
		if resp.Email != nil {
			t.Fatalf("Email = %v, want nil", *resp.Email)
		}
	})

	t.Run("includes copied email when includeEmail true", func(t *testing.T) {
		t.Parallel()
		resp := toUserResponse(base, true, nil, nil, nil)
		if resp.Email == nil || *resp.Email != base.Email {
			t.Fatalf("Email = %v, want %q", resp.Email, base.Email)
		}
		// La copie doit être indépendante du modèle source.
		if resp.Email == &base.Email {
			t.Fatal("Email should point to a copy, not the model field")
		}
	})

	t.Run("formats createdAt as RFC3339 UTC", func(t *testing.T) {
		t.Parallel()
		resp := toUserResponse(base, false, nil, nil, nil)
		if want := "2026-06-18T09:30:00Z"; resp.CreatedAt != want {
			t.Fatalf("CreatedAt = %q, want %q", resp.CreatedAt, want)
		}
	})

	t.Run("passes through optional relation flags", func(t *testing.T) {
		t.Parallel()
		yes := true
		resp := toUserResponse(base, false, &yes, &yes, &yes)
		if resp.IsFollowedByMe == nil || !*resp.IsFollowedByMe {
			t.Fatal("IsFollowedByMe not propagated")
		}
		if resp.IsBlockedByMe == nil || !*resp.IsBlockedByMe {
			t.Fatal("IsBlockedByMe not propagated")
		}
		if resp.HasBlockedMe == nil || !*resp.HasBlockedMe {
			t.Fatal("HasBlockedMe not propagated")
		}
	})
}

func TestToSanctionResponse(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	created := time.Date(2026, 6, 18, 10, 0, 1, 0, time.UTC)

	t.Run("active sanction has nil endedAt", func(t *testing.T) {
		t.Parallel()
		resp := toSanctionResponse(sanctionModel{
			ID:        1,
			UserID:    "usr_abc",
			Type:      "ban",
			StartedAt: started,
			CreatedAt: created,
		})
		if resp.EndedAt != nil {
			t.Fatalf("EndedAt = %v, want nil", *resp.EndedAt)
		}
		if resp.StartedAt != "2026-06-18T10:00:00Z" {
			t.Fatalf("StartedAt = %q", resp.StartedAt)
		}
	})

	t.Run("lifted sanction formats endedAt", func(t *testing.T) {
		t.Parallel()
		ended := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
		resp := toSanctionResponse(sanctionModel{
			ID:        2,
			UserID:    "usr_abc",
			Type:      "suspension",
			StartedAt: started,
			EndedAt:   &ended,
			CreatedAt: created,
		})
		if resp.EndedAt == nil || *resp.EndedAt != "2026-06-19T12:00:00Z" {
			t.Fatalf("EndedAt = %v, want 2026-06-19T12:00:00Z", resp.EndedAt)
		}
	})
}
