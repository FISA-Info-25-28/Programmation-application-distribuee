//go:build integration

package main

import (
	"net/http"
	"testing"
	"time"
)

func TestHTTPVerifyEmail(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "ev", "ev@example.com", strongPassword, accountStatusActive)
	if err := s.db.Model(&authUserModel{}).Where("id = ?", uid).Update("email_verified_at", nil).Error; err != nil {
		t.Fatalf("reset verification: %v", err)
	}
	raw := "raw-verify-token-http"
	if err := s.db.Create(&authEmailVerificationModel{
		ID: randomID("evt"), UserID: uid, TokenHash: hashToken(raw), ExpiresAt: time.Now().UTC().Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed token: %v", err)
	}

	t.Run("invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/verify-email", reqOpt{body: `nope`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unknown token is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/verify-email", reqOpt{body: `{"token":"ghost"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("valid token verifies", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/verify-email", reqOpt{body: `{"token":"` + raw + `"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})
}

func TestHTTPResendAndRequestReset(t *testing.T) {
	router, s := newTestRouter(t)
	seedVerifiedUser(t, s, "anon", "anon@example.com", strongPassword, accountStatusActive)

	// Anti-énumération : 202 que l'email existe ou non.
	t.Run("resend verification always 202", func(t *testing.T) {
		for _, email := range []string{"anon@example.com", "ghost@example.com"} {
			w := do(t, router, http.MethodPost, "/auth/resend-verification", reqOpt{body: `{"email":"` + email + `"}`})
			if w.Code != http.StatusAccepted {
				t.Fatalf("status for %s = %d, want 202", email, w.Code)
			}
		}
	})

	t.Run("request password reset always 202", func(t *testing.T) {
		for _, email := range []string{"anon@example.com", "ghost@example.com"} {
			w := do(t, router, http.MethodPost, "/auth/request-password-reset", reqOpt{body: `{"email":"` + email + `"}`})
			if w.Code != http.StatusAccepted {
				t.Fatalf("status for %s = %d, want 202", email, w.Code)
			}
		}
	})

	t.Run("invalid body is 400", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/request-password-reset", reqOpt{body: `x`}); w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestHTTPResetPassword(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "rp", "rp@example.com", strongPassword, accountStatusActive)
	raw := randomID("rawhttp")
	if err := s.db.Create(&authPasswordResetModel{
		ID: randomID("prt"), UserID: uid, TokenHash: hashToken(raw), ExpiresAt: time.Now().UTC().Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed reset token: %v", err)
	}

	t.Run("unknown token is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/reset-password", reqOpt{body: `{"token":"ghost","password":"Even!Str0ngerPass"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("valid token resets", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/reset-password", reqOpt{body: `{"token":"` + raw + `","password":"Even!Str0ngerPass"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})
}

func TestHTTPMFAFlow(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "mfah", "mfah@example.com", strongPassword, accountStatusActive)

	t.Run("setup requires identity", func(t *testing.T) {
		if w := do(t, router, http.MethodPost, "/auth/mfa/setup", reqOpt{}); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	var secret string
	t.Run("setup returns secret and backup codes", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/mfa/setup", reqOpt{userID: uid})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		body := decode(t, w)
		secret, _ = body["secret"].(string)
		if secret == "" {
			t.Fatalf("secret missing in response: %v", body)
		}
		if codes, _ := body["backupCodes"].([]any); len(codes) != mfaBackupCodeCount {
			t.Errorf("backupCodes = %v, want %d", body["backupCodes"], mfaBackupCodeCount)
		}
	})

	t.Run("confirm with wrong code is 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{userID: uid, body: `{"code":"000000"}`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("confirm with valid code enables", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/mfa/confirm", reqOpt{userID: uid, body: `{"code":"` + currentTOTP(t, secret) + `"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})

	t.Run("MFA login via pending token", func(t *testing.T) {
		pending, err := s.issuePendingMFAToken(uid, "1.2.3.4", "agent")
		if err != nil {
			t.Fatalf("issuePendingMFAToken: %v", err)
		}
		w := do(t, router, http.MethodPost, "/auth/mfa/verify",
			reqOpt{body: `{"pendingToken":"` + pending + `","code":"` + currentTOTP(t, secret) + `"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})

	t.Run("disable with valid code", func(t *testing.T) {
		w := do(t, router, http.MethodDelete, "/auth/mfa", reqOpt{userID: uid, body: `{"code":"` + currentTOTP(t, secret) + `"}`})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
	})
}
