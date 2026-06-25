package main

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPSecret(t *testing.T) {
	t.Parallel()
	s1, err := generateTOTPSecret()
	if err != nil {
		t.Fatalf("generateTOTPSecret: %v", err)
	}
	if s1 == "" {
		t.Fatal("secret is empty")
	}
	// Doit être un base32 sans padding décodable.
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s1); err != nil {
		t.Errorf("secret %q is not valid base32: %v", s1, err)
	}
	// Aléatoire : deux appels diffèrent.
	s2, _ := generateTOTPSecret()
	if s1 == s2 {
		t.Error("two secrets should differ")
	}
}

func TestTOTPURI(t *testing.T) {
	t.Parallel()
	uri := totpURI("Breezy", "alice@example.com", "ABC234")
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Errorf("uri = %q, want otpauth scheme", uri)
	}
	for _, want := range []string{"secret=ABC234", "issuer=Breezy", "algorithm=SHA1", "digits=6", "period=30"} {
		if !strings.Contains(uri, want) {
			t.Errorf("uri %q missing %q", uri, want)
		}
	}
}

func TestVerifyTOTP(t *testing.T) {
	t.Parallel()
	secret, err := generateTOTPSecret()
	if err != nil {
		t.Fatalf("secret: %v", err)
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Code calculé pour la fenêtre courante => accepté.
	now := time.Now().Unix() / int64(mfaTOTPPeriod)
	valid := computeTOTP(key, now)
	if !verifyTOTP(secret, valid) {
		t.Error("current-window code should verify")
	}

	// Tolérance d'horloge : la fenêtre précédente est acceptée.
	prev := computeTOTP(key, now-1)
	if !verifyTOTP(secret, prev) {
		t.Error("previous-window code should verify (clock drift)")
	}

	// Code hors fenêtre => refusé.
	if verifyTOTP(secret, computeTOTP(key, now-5)) {
		t.Error("far-out-of-window code should not verify")
	}
	// Mauvais code => refusé.
	if verifyTOTP(secret, "000000") && valid != "000000" {
		t.Error("wrong code should not verify")
	}
	// Secret non décodable => refusé sans paniquer.
	if verifyTOTP("not!base32!", valid) {
		t.Error("invalid secret should not verify")
	}
}

func TestComputeTOTPDeterministic(t *testing.T) {
	t.Parallel()
	key := []byte("12345678901234567890")
	code := computeTOTP(key, 1)
	if len(code) != mfaTOTPDigits {
		t.Errorf("code length = %d, want %d", len(code), mfaTOTPDigits)
	}
	if code != computeTOTP(key, 1) {
		t.Error("computeTOTP must be deterministic")
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Errorf("code %q must be all digits", code)
		}
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	t.Parallel()
	codes, err := generateBackupCodes(mfaBackupCodeCount)
	if err != nil {
		t.Fatalf("generateBackupCodes: %v", err)
	}
	if len(codes) != mfaBackupCodeCount {
		t.Fatalf("got %d codes, want %d", len(codes), mfaBackupCodeCount)
	}
	seen := make(map[string]bool, len(codes))
	for _, c := range codes {
		if len(c) != 10 { // 5 octets => 10 hex chars
			t.Errorf("code %q length = %d, want 10", c, len(c))
		}
		if c != strings.ToUpper(c) {
			t.Errorf("code %q should be uppercase hex", c)
		}
		seen[c] = true
	}
	if len(seen) != len(codes) {
		t.Error("backup codes should be unique")
	}
}
