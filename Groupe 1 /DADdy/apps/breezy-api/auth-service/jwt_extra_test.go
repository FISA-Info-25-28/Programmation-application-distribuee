package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func writeTestRSAKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return path
}

func TestPrepareAuthSigningKey(t *testing.T) {
	t.Run("empty alg defaults to HS256", func(t *testing.T) {
		cfg := &authConfig{JWTSecret: "secret"}
		prepareAuthSigningKey(cfg)
		if cfg.JWTAlg != "HS256" {
			t.Fatalf("JWTAlg = %q, want HS256", cfg.JWTAlg)
		}
		if cfg.jwtSignKey == nil {
			t.Fatal("jwtSignKey should be set")
		}
	})

	t.Run("RS256 loads private key", func(t *testing.T) {
		cfg := &authConfig{JWTAlg: "RS256", JWTKeyPath: writeTestRSAKey(t)}
		prepareAuthSigningKey(cfg)
		if _, ok := cfg.jwtSignKey.(*rsa.PrivateKey); !ok {
			t.Fatalf("jwtSignKey type = %T, want *rsa.PrivateKey", cfg.jwtSignKey)
		}
	})

	t.Run("HS256 without secret panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for missing secret")
			}
		}()
		prepareAuthSigningKey(&authConfig{JWTAlg: "HS256"})
	})

	t.Run("RS256 without key path panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for missing key path")
			}
		}()
		prepareAuthSigningKey(&authConfig{JWTAlg: "RS256"})
	})

	t.Run("unsupported alg panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for unsupported alg")
			}
		}()
		prepareAuthSigningKey(&authConfig{JWTAlg: "ES999"})
	})
}

func TestLoadRSAPrivateKey(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		key, err := loadRSAPrivateKey(writeTestRSAKey(t))
		if err != nil || key == nil {
			t.Fatalf("loadRSAPrivateKey: key=%v err=%v", key, err)
		}
	})

	t.Run("missing file errors", func(t *testing.T) {
		if _, err := loadRSAPrivateKey(filepath.Join(t.TempDir(), "nope.pem")); err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid PEM errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.pem")
		if err := os.WriteFile(path, []byte("not a pem"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := loadRSAPrivateKey(path); err == nil {
			t.Fatal("expected error for invalid PEM")
		}
	})
}
