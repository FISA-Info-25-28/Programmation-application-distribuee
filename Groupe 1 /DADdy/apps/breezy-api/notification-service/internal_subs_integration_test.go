//go:build integration

package main

import (
	"net/http"
	"testing"
)

func TestInternalSubscribe(t *testing.T) {
	a, router := newTestApp(t)

	t.Run("creates a subscription (204)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/subscriptions", reqOpt{
			internalKey: testInternalKey,
			body:        `{"subscriber_id":"alice","target_user_id":"bob"}`,
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (body %s)", w.Code, w.Body.String())
		}
		var count int64
		a.db.Model(&subscriptionModel{}).Where("subscriber_id = ? AND target_user_id = ?", "alice", "bob").Count(&count)
		if count != 1 {
			t.Errorf("subscription count = %d, want 1", count)
		}
	})

	t.Run("idempotent on conflict (204)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/subscriptions", reqOpt{
			internalKey: testInternalKey,
			body:        `{"subscriber_id":"alice","target_user_id":"bob"}`,
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		var count int64
		a.db.Model(&subscriptionModel{}).Where("subscriber_id = ? AND target_user_id = ?", "alice", "bob").Count(&count)
		if count != 1 {
			t.Errorf("subscription count = %d, want 1 (no duplicate)", count)
		}
	})

	t.Run("self-subscription is a no-op (204)", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/subscriptions", reqOpt{
			internalKey: testInternalKey,
			body:        `{"subscriber_id":"carol","target_user_id":"carol"}`,
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		var count int64
		a.db.Model(&subscriptionModel{}).Where("subscriber_id = ?", "carol").Count(&count)
		if count != 0 {
			t.Errorf("self-subscription should not be stored, got %d", count)
		}
	})

	t.Run("missing field is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/internal/subscriptions", reqOpt{
			internalKey: testInternalKey,
			body:        `{"subscriber_id":"alice"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestInternalUnsubscribe(t *testing.T) {
	a, router := newTestApp(t)
	if err := a.db.Create(&subscriptionModel{SubscriberID: "alice", TargetUserID: "bob"}).Error; err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	t.Run("removes the subscription (204)", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/internal/subscriptions", reqOpt{
			internalKey: testInternalKey,
			body:        `{"subscriber_id":"alice","target_user_id":"bob"}`,
		})
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (body %s)", w.Code, w.Body.String())
		}
		var count int64
		a.db.Model(&subscriptionModel{}).Where("subscriber_id = ? AND target_user_id = ?", "alice", "bob").Count(&count)
		if count != 0 {
			t.Errorf("subscription should be gone, got %d", count)
		}
	})

	t.Run("missing field is 400", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/internal/subscriptions", reqOpt{
			internalKey: testInternalKey,
			body:        `{"subscriber_id":"alice"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
