//go:build integration

package main

import (
	"net/http"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TestHTTPLoginExtraBranches covers loginHandler error paths not reached by the
// main login test: unverified email, empty identifier, invalid body, MFA required.
func TestHTTPLoginExtraBranches(t *testing.T) {
	router, s := newTestRouter(t)

	t.Run("invalid body returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{body: `{bad`})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("empty identifier returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"","password":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("unverified email returns 403", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte(strongPassword), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("hash: %v", err)
		}
		now := time.Now().UTC()
		u := authUserModel{
			ID:              randomID("usr"),
			Username:        "unverified",
			Email:           "unverified@example.com",
			PasswordHash:    string(hash),
			Role:            defaultUserRole,
			Status:          accountStatusActive,
			TermsAcceptedAt: &now,
			TermsVersion:    currentTermsVersion,
			// EmailVerifiedAt deliberately nil
		}
		if err := s.db.Create(&u).Error; err != nil {
			t.Fatalf("seed unverified user: %v", err)
		}
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"unverified","password":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body %s)", w.Code, w.Body.String())
		}
	})

	t.Run("mfa required returns 200 with mfaRequired and pendingToken", func(t *testing.T) {
		uid := seedVerifiedUser(t, s, "mfauser", "mfa@example.com", strongPassword, accountStatusActive)
		secret, err := generateTOTPSecret()
		if err != nil {
			t.Fatalf("generateTOTPSecret: %v", err)
		}
		now := time.Now().UTC()
		if err := s.db.Create(&authMFAModel{
			ID:         randomID("mfa"),
			UserID:     uid,
			TOTPSecret: secret,
			Enabled:    true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}).Error; err != nil {
			t.Fatalf("seed mfa: %v", err)
		}
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"mfauser","password":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
		}
		body := decode(t, w)
		if mfaReq, _ := body["mfaRequired"].(bool); !mfaReq {
			t.Errorf("mfaRequired = %v, want true", body["mfaRequired"])
		}
		if pt, _ := body["pendingToken"].(string); pt == "" {
			t.Error("pendingToken missing")
		}
	})
}

// TestHTTPRefreshExtraBranches covers error paths in refreshHandler: token reuse
// and suspended account returning 403.
func TestHTTPRefreshExtraBranches(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	login := do(t, router, http.MethodPost, "/auth/login", reqOpt{
		body: `{"identifier":"alice","password":"` + strongPassword + `"}`,
	})
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.Code)
	}
	originalToken, _ := decode(t, login)["refreshToken"].(string)
	if originalToken == "" {
		t.Fatal("setup: no refreshToken from login")
	}

	t.Run("reuse of already-rotated token returns 401", func(t *testing.T) {
		rotate := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
			body: `{"refreshToken":"` + originalToken + `"}`,
		})
		if rotate.Code != http.StatusOK {
			t.Fatalf("first refresh status = %d, want 200", rotate.Code)
		}
		reuse := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
			body: `{"refreshToken":"` + originalToken + `"}`,
		})
		if reuse.Code != http.StatusUnauthorized {
			t.Fatalf("reuse status = %d, want 401 (body %s)", reuse.Code, reuse.Body.String())
		}
	})

	t.Run("suspended account refresh returns 403", func(t *testing.T) {
		// Get a fresh token before suspending.
		l := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"alice","password":"` + strongPassword + `"}`,
		})
		if l.Code != http.StatusOK {
			t.Fatalf("login status = %d", l.Code)
		}
		freshToken, _ := decode(t, l)["refreshToken"].(string)

		if err := s.db.Model(&authUserModel{}).
			Where("id = ?", uid).
			Update("status", accountStatusSuspended).Error; err != nil {
			t.Fatalf("suspend user: %v", err)
		}
		w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
			body: `{"refreshToken":"` + freshToken + `"}`,
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body %s)", w.Code, w.Body.String())
		}
	})
}

// TestHTTPRefreshExpiredToken seeds an expired refresh token directly in the DB
// and verifies that the refresh handler returns 401.
func TestHTTPRefreshExpiredToken(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	rawToken, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken: %v", err)
	}
	now := time.Now().UTC()
	past := now.Add(-24 * time.Hour)
	if err := s.db.Create(&authRefreshTokenModel{
		ID:        randomID("rt"),
		UserID:    uid,
		SessionID: randomID("sid"),
		FamilyID:  randomID("fam"),
		TokenHash: hashToken(rawToken),
		ExpiresAt: past, // expired
		CreatedAt: past,
		UpdatedAt: past,
	}).Error; err != nil {
		t.Fatalf("seed expired token: %v", err)
	}

	w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
		body: `{"refreshToken":"` + rawToken + `"}`,
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", w.Code, w.Body.String())
	}
}

// TestHTTPRefreshEmptyToken verifies that an empty refreshToken string triggers
// the errValidation path inside the refresh service method.
func TestHTTPRefreshEmptyToken(t *testing.T) {
	router, _ := newTestRouter(t)
	w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
		body: `{"refreshToken":""}`,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestHTTPLogoutWithRealToken exercises the token-found-and-revoked path inside
// logout (contrast with the existing test that uses a non-existent "anything" token).
func TestHTTPLogoutWithRealToken(t *testing.T) {
	router, s := newTestRouter(t)
	seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	login := do(t, router, http.MethodPost, "/auth/login", reqOpt{
		body: `{"identifier":"alice","password":"` + strongPassword + `"}`,
	})
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.Code)
	}
	refreshToken, _ := decode(t, login)["refreshToken"].(string)
	if refreshToken == "" {
		t.Fatal("no refreshToken from login")
	}

	w := do(t, router, http.MethodPost, "/auth/logout", reqOpt{
		body: `{"refreshToken":"` + refreshToken + `"}`,
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body %s)", w.Code, w.Body.String())
	}
}

// TestHTTPRateLimitBranches exercises the rate-limit-exceeded (429) branches in
// loginHandler and changePasswordHandler by exhausting the in-memory limiter.
// authRLThreshold = 5, so the 6th attempt from the same IP+identifier is blocked.
func TestHTTPRateLimitBranches(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "limited", "limited@example.com", strongPassword, accountStatusActive)

	t.Run("login 429 after 5 failures", func(t *testing.T) {
		for i := 0; i < authRLThreshold; i++ {
			do(t, router, http.MethodPost, "/auth/login", reqOpt{
				body: `{"identifier":"limited","password":"WrongPass1!"}`,
			})
		}
		w := do(t, router, http.MethodPost, "/auth/login", reqOpt{
			body: `{"identifier":"limited","password":"WrongPass1!"}`,
		})
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429 after rate limit exceeded", w.Code)
		}
	})

	t.Run("change-password 429 after 5 failures", func(t *testing.T) {
		for i := 0; i < authRLThreshold; i++ {
			do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
				userID: uid,
				body:   `{"currentPassword":"WrongCurrent1!","newPassword":"` + strongPassword + `"}`,
			})
		}
		w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
			userID: uid,
			body:   `{"currentPassword":"WrongCurrent1!","newPassword":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429 after rate limit exceeded", w.Code)
		}
	})

	t.Run("refresh 429 after 5 failures with same token", func(t *testing.T) {
		const fakeToken = "rate-limit-test-token-always-invalid"
		for i := 0; i < authRLThreshold; i++ {
			do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
				body: `{"refreshToken":"` + fakeToken + `"}`,
			})
		}
		w := do(t, router, http.MethodPost, "/auth/refresh", reqOpt{
			body: `{"refreshToken":"` + fakeToken + `"}`,
		})
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429 after rate limit exceeded", w.Code)
		}
	})
}

// TestHTTPSetStatusInvalidBody covers the ShouldBindJSON error path in
// setStatusHandler (distinct from the errValidation path triggered by an
// unrecognised status string, which is tested in TestHTTPInternalSetStatus).
func TestHTTPSetStatusInvalidBody(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	w := do(t, router, http.MethodPatch, "/internal/users/"+uid+"/status", reqOpt{
		body:    `{bad`,
		headers: map[string]string{"X-Internal-Key": "test-internal-key"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
}

// TestHTTPAnonymousEmailAction429 exercises the rate-limit branch inside
// anonymousEmailAction (shared by resend-verification and request-password-reset).
// Five consecutive requests with the same email exhaust the budget; the sixth
// returns 429.
func TestHTTPAnonymousEmailAction429(t *testing.T) {
	router, s := newTestRouter(t)
	seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	for i := 0; i < authRLThreshold; i++ {
		do(t, router, http.MethodPost, "/auth/resend-verification", reqOpt{
			body: `{"email":"alice@example.com"}`,
		})
	}
	w := do(t, router, http.MethodPost, "/auth/resend-verification", reqOpt{
		body: `{"email":"alice@example.com"}`,
	})
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 after rate limit exhausted", w.Code)
	}
}

// TestHTTPChangePasswordExtraBranches covers the same-password and weak-new-password
// error paths in changePasswordHandler.
func TestHTTPChangePasswordExtraBranches(t *testing.T) {
	router, s := newTestRouter(t)
	uid := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	t.Run("new password same as current returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
			userID: uid,
			body:   `{"currentPassword":"` + strongPassword + `","newPassword":"` + strongPassword + `"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
		}
	})

	t.Run("new password fails validation returns 400", func(t *testing.T) {
		w := do(t, router, http.MethodPost, "/auth/change-password", reqOpt{
			userID: uid,
			body:   `{"currentPassword":"` + strongPassword + `","newPassword":"weak"}`,
		})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
		}
	})
}
