//go:build integration

package main

import (
	"encoding/base32"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// currentTOTP calcule un code TOTP valide pour le secret à l'instant présent.
func currentTOTP(t *testing.T, secret string) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}
	return computeTOTP(key, time.Now().Unix()/int64(mfaTOTPPeriod))
}

// seedResetToken insère un token de réinitialisation et renvoie sa valeur brute.
func seedResetToken(t *testing.T, s *authService, userID string, expiry time.Time) string {
	t.Helper()
	raw := randomID("rawreset")
	if err := s.db.Create(&authPasswordResetModel{
		ID: randomID("prt"), UserID: userID, TokenHash: hashToken(raw), ExpiresAt: expiry,
	}).Error; err != nil {
		t.Fatalf("seed reset token: %v", err)
	}
	return raw
}

func TestResetPassword(t *testing.T) {
	s := newTestService(t)
	const newPw = "Even!Str0ngerPass"
	now := time.Now().UTC()

	t.Run("empty token is validation error", func(t *testing.T) {
		if err := s.resetPassword("  ", newPw); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("unknown token is invalid", func(t *testing.T) {
		if err := s.resetPassword("nope", newPw); !errors.Is(err, errInvalidToken) {
			t.Fatalf("err = %v, want errInvalidToken", err)
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		uid := seedVerifiedUser(t, s, "exp", "exp@example.com", strongPassword, accountStatusActive)
		raw := seedResetToken(t, s, uid, now.Add(-time.Hour))
		if err := s.resetPassword(raw, newPw); !errors.Is(err, errTokenExpired) {
			t.Fatalf("err = %v, want errTokenExpired", err)
		}
	})

	t.Run("weak password does not burn the token", func(t *testing.T) {
		uid := seedVerifiedUser(t, s, "weak", "weak@example.com", strongPassword, accountStatusActive)
		raw := seedResetToken(t, s, uid, now.Add(time.Hour))
		if err := s.resetPassword(raw, "short"); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
		// Le token doit rester utilisable après un échec de validation.
		if err := s.resetPassword(raw, newPw); err != nil {
			t.Fatalf("token should survive a validation failure: %v", err)
		}
	})

	t.Run("reusing current password is rejected", func(t *testing.T) {
		uid := seedVerifiedUser(t, s, "same", "same@example.com", strongPassword, accountStatusActive)
		raw := seedResetToken(t, s, uid, now.Add(time.Hour))
		if err := s.resetPassword(raw, strongPassword); !errors.Is(err, errSamePassword) {
			t.Fatalf("err = %v, want errSamePassword", err)
		}
	})

	t.Run("success updates hash and is single-use", func(t *testing.T) {
		uid := seedVerifiedUser(t, s, "ok", "ok@example.com", strongPassword, accountStatusActive)
		raw := seedResetToken(t, s, uid, now.Add(time.Hour))
		if err := s.resetPassword(raw, newPw); err != nil {
			t.Fatalf("resetPassword: %v", err)
		}
		u := loadUser(t, s, uid)
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(newPw)) != nil {
			t.Error("new password should verify against the stored hash")
		}
		// Rejouer le même token doit échouer (consommé).
		if err := s.resetPassword(raw, "Another!Str0ng1"); !errors.Is(err, errInvalidToken) {
			t.Fatalf("replay err = %v, want errInvalidToken", err)
		}
	})
}

func TestRequestPasswordResetAntiEnumeration(t *testing.T) {
	s := newTestService(t)
	seedVerifiedUser(t, s, "known", "known@example.com", strongPassword, accountStatusActive)

	t.Run("unknown email returns nil without creating a token", func(t *testing.T) {
		if err := s.requestPasswordReset("ghost@example.com"); err != nil {
			t.Fatalf("err = %v, want nil (anti-enumeration)", err)
		}
		var count int64
		s.db.Model(&authPasswordResetModel{}).Count(&count)
		if count != 0 {
			t.Errorf("no token should be created for unknown email, got %d", count)
		}
	})

	t.Run("known email issues a reset token", func(t *testing.T) {
		if err := s.requestPasswordReset("known@example.com"); err != nil {
			t.Fatalf("requestPasswordReset: %v", err)
		}
		var count int64
		s.db.Model(&authPasswordResetModel{}).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 reset token, got %d", count)
		}
	})
}

func TestResendVerification(t *testing.T) {
	s := newTestService(t)

	t.Run("unknown email is a no-op", func(t *testing.T) {
		if err := s.resendVerification("ghost@example.com"); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("already verified is a no-op without new token", func(t *testing.T) {
		seedVerifiedUser(t, s, "verif", "verif@example.com", strongPassword, accountStatusActive)
		if err := s.resendVerification("verif@example.com"); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		var count int64
		s.db.Model(&authEmailVerificationModel{}).Where("user_id IN (SELECT id FROM auth_user_models WHERE email = ?)", "verif@example.com").Count(&count)
		if count != 0 {
			t.Errorf("verified account should not get a new token, got %d", count)
		}
	})

	t.Run("unverified account gets a fresh token", func(t *testing.T) {
		uid := seedVerifiedUser(t, s, "unv", "unv@example.com", strongPassword, accountStatusActive)
		if err := s.db.Model(&authUserModel{}).Where("id = ?", uid).Update("email_verified_at", nil).Error; err != nil {
			t.Fatalf("reset verification: %v", err)
		}
		if err := s.resendVerification("unv@example.com"); err != nil {
			t.Fatalf("resendVerification: %v", err)
		}
		var count int64
		s.db.Model(&authEmailVerificationModel{}).Where("user_id = ?", uid).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 verification token, got %d", count)
		}
	})
}

func TestMFASetupConfirmDisable(t *testing.T) {
	s := newTestService(t)
	uid := seedVerifiedUser(t, s, "mfauser", "mfa@example.com", strongPassword, accountStatusActive)

	setup, err := s.setupMFA(uid)
	if err != nil {
		t.Fatalf("setupMFA: %v", err)
	}
	if setup.Secret == "" || setup.URI == "" || len(setup.BackupCodes) != mfaBackupCodeCount {
		t.Fatalf("setup result incomplete: %+v", setup)
	}

	t.Run("confirm with wrong code fails", func(t *testing.T) {
		if err := s.confirmMFA(uid, "000000"); !errors.Is(err, errMFAInvalidCode) {
			t.Fatalf("err = %v, want errMFAInvalidCode", err)
		}
	})

	t.Run("confirm with valid code enables", func(t *testing.T) {
		if err := s.confirmMFA(uid, currentTOTP(t, setup.Secret)); err != nil {
			t.Fatalf("confirmMFA: %v", err)
		}
		var mfa authMFAModel
		if err := s.db.First(&mfa, "user_id = ?", uid).Error; err != nil || !mfa.Enabled {
			t.Fatalf("MFA should be enabled: %+v err=%v", mfa, err)
		}
	})

	t.Run("setup again when enabled is rejected", func(t *testing.T) {
		if _, err := s.setupMFA(uid); !errors.Is(err, errMFAAlreadyEnabled) {
			t.Fatalf("err = %v, want errMFAAlreadyEnabled", err)
		}
	})

	t.Run("disable with backup code removes MFA", func(t *testing.T) {
		if err := s.disableMFA(uid, setup.BackupCodes[0]); err != nil {
			t.Fatalf("disableMFA: %v", err)
		}
		var count int64
		s.db.Model(&authMFAModel{}).Where("user_id = ?", uid).Count(&count)
		if count != 0 {
			t.Errorf("MFA row should be gone, got %d", count)
		}
	})

	t.Run("disable when not enabled fails", func(t *testing.T) {
		if err := s.disableMFA(uid, "whatever"); !errors.Is(err, errMFANotEnabled) {
			t.Fatalf("err = %v, want errMFANotEnabled", err)
		}
	})
}

func TestVerifyMFALogin(t *testing.T) {
	s := newTestService(t)
	uid := seedVerifiedUser(t, s, "mfalogin", "mfalogin@example.com", strongPassword, accountStatusActive)
	setup, err := s.setupMFA(uid)
	if err != nil {
		t.Fatalf("setupMFA: %v", err)
	}
	if err := s.confirmMFA(uid, currentTOTP(t, setup.Secret)); err != nil {
		t.Fatalf("confirmMFA: %v", err)
	}

	t.Run("empty inputs are validation errors", func(t *testing.T) {
		if _, err := s.verifyMFALogin("", "", "1.2.3.4", "agent"); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("unknown pending token is invalid", func(t *testing.T) {
		if _, err := s.verifyMFALogin("nope", "123456", "1.2.3.4", "agent"); !errors.Is(err, errInvalidToken) {
			t.Fatalf("err = %v, want errInvalidToken", err)
		}
	})

	t.Run("wrong code is rejected", func(t *testing.T) {
		pending, err := s.issuePendingMFAToken(uid, "1.2.3.4", "agent")
		if err != nil {
			t.Fatalf("issuePendingMFAToken: %v", err)
		}
		if _, err := s.verifyMFALogin(pending, "000000", "1.2.3.4", "agent"); !errors.Is(err, errMFAInvalidCode) {
			t.Fatalf("err = %v, want errMFAInvalidCode", err)
		}
	})

	t.Run("valid TOTP issues a token pair", func(t *testing.T) {
		pending, err := s.issuePendingMFAToken(uid, "1.2.3.4", "agent")
		if err != nil {
			t.Fatalf("issuePendingMFAToken: %v", err)
		}
		pair, err := s.verifyMFALogin(pending, currentTOTP(t, setup.Secret), "1.2.3.4", "agent")
		if err != nil {
			t.Fatalf("verifyMFALogin: %v", err)
		}
		if pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatal("token pair must be populated")
		}
		// Le pending token est à usage unique.
		if _, err := s.verifyMFALogin(pending, currentTOTP(t, setup.Secret), "1.2.3.4", "agent"); !errors.Is(err, errInvalidToken) {
			t.Fatalf("replay err = %v, want errInvalidToken", err)
		}
	})
}
