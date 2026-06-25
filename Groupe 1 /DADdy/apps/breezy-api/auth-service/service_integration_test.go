//go:build integration

package main

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const strongPassword = "Str0ng!Passphrase"

func TestRegister(t *testing.T) {
	s := newTestService(t)

	t.Run("creates an unverified user with a hashed password", func(t *testing.T) {
		if err := s.register("Alice", "Alice@Example.com", strongPassword, true, currentTermsVersion); err != nil {
			t.Fatalf("register: %v", err)
		}
		var u authUserModel
		if err := s.db.First(&u, "username = ?", "alice").Error; err != nil {
			t.Fatalf("fetch registered user: %v", err)
		}
		if u.Email != "alice@example.com" {
			t.Errorf("email = %q, want normalized lowercase", u.Email)
		}
		if u.EmailVerifiedAt != nil {
			t.Error("new account must start unverified")
		}
		if u.PasswordHash == strongPassword || u.PasswordHash == "" {
			t.Error("password must be hashed, never stored in clear")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(strongPassword)) != nil {
			t.Error("stored hash does not verify against the original password")
		}
	})

	t.Run("duplicate username or email conflicts", func(t *testing.T) {
		if err := s.register("alice", "other@example.com", strongPassword, true, currentTermsVersion); !errors.Is(err, errConflict) {
			t.Fatalf("err = %v, want errConflict", err)
		}
	})

	t.Run("invalid input is a validation error", func(t *testing.T) {
		if err := s.register("ab", "bad@example.com", strongPassword, true, currentTermsVersion); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})

	t.Run("terms must be accepted", func(t *testing.T) {
		if err := s.register("bob", "bob@example.com", strongPassword, false, currentTermsVersion); !errors.Is(err, errValidation) {
			t.Fatalf("err = %v, want errValidation", err)
		}
	})
}

func TestLoginGates(t *testing.T) {
	s := newTestService(t)
	seedVerifiedUser(t, s, "active", "active@example.com", strongPassword, accountStatusActive)
	seedVerifiedUser(t, s, "banned", "banned@example.com", strongPassword, accountStatusBanned)

	// Compte non vérifié inséré directement (EmailVerifiedAt nil).
	hash, _ := bcrypt.GenerateFromPassword([]byte(strongPassword), bcrypt.DefaultCost)
	now := time.Now().UTC()
	if err := s.db.Create(&authUserModel{
		ID: randomID("usr"), Username: "unverified", Email: "unverified@example.com",
		PasswordHash: string(hash), Role: defaultUserRole, Status: accountStatusActive,
		TermsAcceptedAt: &now, TermsVersion: currentTermsVersion,
	}).Error; err != nil {
		t.Fatalf("seed unverified: %v", err)
	}

	t.Run("success returns a token pair", func(t *testing.T) {
		pair, err := s.login("active", strongPassword, "1.2.3.4", "test-agent")
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		if pair.Tokens.AccessToken == "" || pair.Tokens.RefreshToken == "" {
			t.Fatal("token pair must be populated")
		}
	})

	t.Run("login by email also works", func(t *testing.T) {
		if _, err := s.login("active@example.com", strongPassword, "1.2.3.4", "agent"); err != nil {
			t.Fatalf("login by email: %v", err)
		}
	})

	t.Run("wrong password is invalid credentials", func(t *testing.T) {
		if _, err := s.login("active", "WrongPassword1!", "1.2.3.4", "agent"); !errors.Is(err, errInvalidCredentials) {
			t.Fatalf("err = %v, want errInvalidCredentials", err)
		}
	})

	t.Run("unverified email is blocked", func(t *testing.T) {
		if _, err := s.login("unverified", strongPassword, "1.2.3.4", "agent"); !errors.Is(err, errEmailNotVerified) {
			t.Fatalf("err = %v, want errEmailNotVerified", err)
		}
	})

	t.Run("banned account is blocked", func(t *testing.T) {
		if _, err := s.login("banned", strongPassword, "1.2.3.4", "agent"); !errors.Is(err, errAccountSuspended) {
			t.Fatalf("err = %v, want errAccountSuspended", err)
		}
	})
}

func TestVerifyEmail(t *testing.T) {
	s := newTestService(t)
	userID := seedVerifiedUser(t, s, "tmp", "tmp@example.com", strongPassword, accountStatusActive)
	// On repasse le compte en non vérifié pour tester la vérification.
	if err := s.db.Model(&authUserModel{}).Where("id = ?", userID).Update("email_verified_at", nil).Error; err != nil {
		t.Fatalf("reset verification: %v", err)
	}

	// Crée un token de vérification dont on connaît la valeur brute.
	raw := "raw-verification-token"
	now := time.Now().UTC()
	if err := s.db.Create(&authEmailVerificationModel{
		ID: randomID("evt"), UserID: userID, TokenHash: hashToken(raw),
		ExpiresAt: now.Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed token: %v", err)
	}

	t.Run("valid token verifies the account and is single-use", func(t *testing.T) {
		if err := s.verifyEmail(raw); err != nil {
			t.Fatalf("verifyEmail: %v", err)
		}
		if loadUser(t, s, userID).EmailVerifiedAt == nil {
			t.Error("EmailVerifiedAt should be set after verification")
		}
		// Rejouer le même token doit échouer (déjà consommé).
		if err := s.verifyEmail(raw); !errors.Is(err, errInvalidToken) {
			t.Fatalf("replay err = %v, want errInvalidToken", err)
		}
	})

	t.Run("unknown token is invalid", func(t *testing.T) {
		if err := s.verifyEmail("does-not-exist"); !errors.Is(err, errInvalidToken) {
			t.Fatalf("err = %v, want errInvalidToken", err)
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		expiredRaw := "expired-token"
		if err := s.db.Create(&authEmailVerificationModel{
			ID: randomID("evt"), UserID: userID, TokenHash: hashToken(expiredRaw),
			ExpiresAt: now.Add(-time.Hour),
		}).Error; err != nil {
			t.Fatalf("seed expired token: %v", err)
		}
		if err := s.verifyEmail(expiredRaw); !errors.Is(err, errTokenExpired) {
			t.Fatalf("err = %v, want errTokenExpired", err)
		}
	})
}

func TestRefreshRotationAndReuse(t *testing.T) {
	s := newTestService(t)
	seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	pair, err := s.login("alice", strongPassword, "1.2.3.4", "agent")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	t.Run("refresh rotates the token and revokes the old one", func(t *testing.T) {
		newPair, err := s.refresh(pair.Tokens.RefreshToken, "1.2.3.4", "agent")
		if err != nil {
			t.Fatalf("refresh: %v", err)
		}
		if newPair.RefreshToken == pair.Tokens.RefreshToken {
			t.Fatal("refresh token should be rotated to a new value")
		}

		// Réutiliser l'ancien token (déjà révoqué) doit être détecté comme reuse.
		if _, err := s.refresh(pair.Tokens.RefreshToken, "1.2.3.4", "agent"); !errors.Is(err, errRefreshReuse) {
			t.Fatalf("reuse err = %v, want errRefreshReuse", err)
		}

		// La détection de reuse révoque toute la famille : le token roté est mort aussi.
		if _, err := s.refresh(newPair.RefreshToken, "1.2.3.4", "agent"); !errors.Is(err, errRefreshReuse) {
			t.Fatalf("post-reuse rotated token err = %v, want errRefreshReuse (family revoked)", err)
		}
	})

	t.Run("unknown refresh token is invalid", func(t *testing.T) {
		if _, err := s.refresh("not-a-real-token", "1.2.3.4", "agent"); !errors.Is(err, errInvalidToken) {
			t.Fatalf("err = %v, want errInvalidToken", err)
		}
	})
}

func TestChangePassword(t *testing.T) {
	s := newTestService(t)
	const newPw = "Even!Str0ngerPass"
	userID := seedVerifiedUser(t, s, "alice", "alice@example.com", strongPassword, accountStatusActive)

	t.Run("wrong current password is rejected", func(t *testing.T) {
		if err := s.changePassword(userID, "WrongCurrent1!", newPw); !errors.Is(err, errInvalidCredentials) {
			t.Fatalf("err = %v, want errInvalidCredentials", err)
		}
	})

	t.Run("same password is rejected", func(t *testing.T) {
		if err := s.changePassword(userID, strongPassword, strongPassword); !errors.Is(err, errSamePassword) {
			t.Fatalf("err = %v, want errSamePassword", err)
		}
	})

	t.Run("success updates the hash", func(t *testing.T) {
		if err := s.changePassword(userID, strongPassword, newPw); err != nil {
			t.Fatalf("changePassword: %v", err)
		}
		u := loadUser(t, s, userID)
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(newPw)) != nil {
			t.Error("new password should verify against the stored hash")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(strongPassword)) == nil {
			t.Error("old password should no longer verify")
		}
	})
}
