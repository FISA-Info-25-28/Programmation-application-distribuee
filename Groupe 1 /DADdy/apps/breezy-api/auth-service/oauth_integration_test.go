//go:build integration

package main

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestFindOrCreateSocialUser(t *testing.T) {
	t.Run("creates a new user when no link or email match", func(t *testing.T) {
		s := newTestService(t)
		info := oauthUserInfo{ProviderUID: "g-123", Email: "newbie@example.com", Username: "New.Bie!"}
		res, err := s.findOrCreateSocialUser(oauthProviderGoogle, info, &oauth2.Token{AccessToken: "at"})
		if err != nil {
			t.Fatalf("findOrCreateSocialUser: %v", err)
		}
		if res.UserID == "" || res.Tokens.AccessToken == "" {
			t.Fatalf("expected user id and tokens, got %+v", res)
		}
		u := loadUser(t, s, res.UserID)
		if u.Username != "newbie" {
			t.Errorf("username = %q, want sanitized 'newbie'", u.Username)
		}
		if u.EmailVerifiedAt == nil {
			t.Error("social user should be email-verified")
		}
	})

	t.Run("returning user with existing social link logs in", func(t *testing.T) {
		s := newTestService(t)
		info := oauthUserInfo{ProviderUID: "g-777", Email: "again@example.com", Username: "again"}
		first, err := s.findOrCreateSocialUser(oauthProviderGoogle, info, &oauth2.Token{AccessToken: "at1"})
		if err != nil {
			t.Fatalf("first call: %v", err)
		}
		second, err := s.findOrCreateSocialUser(oauthProviderGoogle, info, &oauth2.Token{AccessToken: "at2"})
		if err != nil {
			t.Fatalf("second call: %v", err)
		}
		if first.UserID != second.UserID {
			t.Errorf("expected same user, got %s then %s", first.UserID, second.UserID)
		}
	})

	t.Run("links to existing account with same email", func(t *testing.T) {
		s := newTestService(t)
		uid := seedVerifiedUser(t, s, "linkme", "link@example.com", "Str0ng!Passphrase", "active")
		info := oauthUserInfo{ProviderUID: "gh-1", Email: "link@example.com", Username: "linkme"}
		res, err := s.findOrCreateSocialUser(oauthProviderGitHub, info, &oauth2.Token{AccessToken: "at"})
		if err != nil {
			t.Fatalf("findOrCreateSocialUser: %v", err)
		}
		if res.UserID != uid {
			t.Errorf("should link to existing user %s, got %s", uid, res.UserID)
		}
		var count int64
		s.db.Model(&authSocialAccountModel{}).Where("user_id = ?", uid).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 linked social account, got %d", count)
		}
	})

	t.Run("creates synthetic email when provider omits it", func(t *testing.T) {
		s := newTestService(t)
		info := oauthUserInfo{ProviderUID: "gh-noemail", Username: "ghost"}
		res, err := s.findOrCreateSocialUser(oauthProviderGitHub, info, &oauth2.Token{})
		if err != nil {
			t.Fatalf("findOrCreateSocialUser: %v", err)
		}
		u := loadUser(t, s, res.UserID)
		if u.Email != "github_gh-noemail@oauth.breezy.local" {
			t.Errorf("synthetic email = %q", u.Email)
		}
	})
}

func TestDeduplicateUsername(t *testing.T) {
	s := newTestService(t)
	seedVerifiedUser(t, s, "taken", "taken@example.com", "Str0ng!Passphrase", "active")

	if got := deduplicateUsername(s.db, "free"); got != "free" {
		t.Errorf("free username should be unchanged, got %q", got)
	}
	if got := deduplicateUsername(s.db, "taken"); got != "taken1" {
		t.Errorf("taken should become taken1, got %q", got)
	}
}

func TestOAuthExchangeCode(t *testing.T) {
	s := newTestService(t)
	uid := seedVerifiedUser(t, s, "exch", "exch@example.com", "Str0ng!Passphrase", "active")

	t.Run("issue then redeem returns a token pair once", func(t *testing.T) {
		code, err := s.issueOAuthExchangeCode(uid)
		if err != nil {
			t.Fatalf("issueOAuthExchangeCode: %v", err)
		}
		tokens, err := s.redeemOAuthExchangeCode(code)
		if err != nil {
			t.Fatalf("redeemOAuthExchangeCode: %v", err)
		}
		if tokens.AccessToken == "" {
			t.Error("expected an access token")
		}
		// Second redemption of the same code must fail (single-use).
		if _, err := s.redeemOAuthExchangeCode(code); err == nil {
			t.Error("second redemption should fail")
		}
	})

	t.Run("empty code is a validation error", func(t *testing.T) {
		if _, err := s.redeemOAuthExchangeCode("   "); err == nil {
			t.Error("empty code should error")
		}
	})

	t.Run("unknown code is invalid", func(t *testing.T) {
		if _, err := s.redeemOAuthExchangeCode("does-not-exist"); err == nil {
			t.Error("unknown code should error")
		}
	})

	t.Run("expired code is rejected", func(t *testing.T) {
		code, err := s.issueOAuthExchangeCode(uid)
		if err != nil {
			t.Fatalf("issueOAuthExchangeCode: %v", err)
		}
		// Force expiry in the past.
		s.db.Model(&authOAuthExchangeModel{}).
			Where("code_hash = ?", hashToken(code)).
			Update("expires_at", time.Now().UTC().Add(-time.Minute))
		if _, err := s.redeemOAuthExchangeCode(code); err == nil {
			t.Error("expired code should be rejected")
		}
	})
}
