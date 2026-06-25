//go:build integration

package main

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

// seedReport crée un signalement via l'API et renvoie son ID.
func seedReport(t *testing.T, a *app, router *gin.Engine, postID int64, reporter string) int64 {
	t.Helper()
	if w := do(t, router, http.MethodPost, "/posts/"+itoa(postID)+"/report",
		reqOpt{userID: reporter, body: `{"reason":"spam"}`}); w.Code != http.StatusCreated {
		t.Fatalf("seed report status = %d (body %s)", w.Code, w.Body.String())
	}
	var r reportModel
	if err := a.db.Where("post_id = ? AND reporter_id = ?", postID, reporter).First(&r).Error; err != nil {
		t.Fatalf("load seeded report: %v", err)
	}
	return r.ID
}

func TestUpdateReport(t *testing.T) {
	a, router := newTestApp(t)
	postID := seedPost(t, a, "author", "offensive?")

	t.Run("requires staff role", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/reports/1", reqOpt{userID: "u", role: shared.RoleUser, body: `{"action":"reject"}`})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("invalid id is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/reports/abc", reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"action":"reject"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unknown report is 404", func(t *testing.T) {
		w := do(t, router, http.MethodPatch, "/reports/99999", reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"action":"reject"}`})
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("unknown action is 400", func(t *testing.T) {
		rid := seedReport(t, a, router, postID, "r1")
		w := do(t, router, http.MethodPatch, "/reports/"+itoa(rid), reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"action":"explode"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("reject marks the report rejected", func(t *testing.T) {
		rid := seedReport(t, a, router, postID, "r2")
		w := do(t, router, http.MethodPatch, "/reports/"+itoa(rid), reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"action":"reject"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		var r reportModel
		if err := a.db.First(&r, rid).Error; err != nil {
			t.Fatalf("reload report: %v", err)
		}
		if r.Status != reportStatusRejected {
			t.Errorf("status = %q, want %q", r.Status, reportStatusRejected)
		}
	})

	t.Run("resolve marks the report resolved", func(t *testing.T) {
		rid := seedReport(t, a, router, postID, "r3")
		w := do(t, router, http.MethodPatch, "/reports/"+itoa(rid), reqOpt{userID: "mod", role: shared.RoleAdministrator, body: `{"action":"resolve"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var r reportModel
		if err := a.db.First(&r, rid).Error; err != nil {
			t.Fatalf("reload report: %v", err)
		}
		if r.Status != reportStatusResolved {
			t.Errorf("status = %q, want %q", r.Status, reportStatusResolved)
		}
	})

	t.Run("delete purges the reported post", func(t *testing.T) {
		victim := seedPost(t, a, "author", "to be removed")
		rid := seedReport(t, a, router, victim, "r4")
		w := do(t, router, http.MethodPatch, "/reports/"+itoa(rid), reqOpt{userID: "mod", role: shared.RoleModerator, body: `{"action":"delete"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		var count int64
		a.db.Model(&postModel{}).Where("id = ?", victim).Count(&count)
		if count != 0 {
			t.Errorf("reported post should be purged, got %d", count)
		}
	})
}
