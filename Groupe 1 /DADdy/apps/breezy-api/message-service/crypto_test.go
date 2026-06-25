package main

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	if err := initEncryption("test-secret"); err != nil {
		t.Fatalf("initEncryption: %v", err)
	}
	for _, plain := range []string{"hello", "accentué é€ 🙂", strings.Repeat("x", 2000)} {
		enc := encryptContent(plain)
		if !strings.HasPrefix(enc, encPrefix) {
			t.Fatalf("expected ciphertext to be prefixed, got %q", enc)
		}
		if enc == plain || strings.Contains(enc, plain) {
			t.Fatalf("ciphertext leaks plaintext for %q", plain)
		}
		if got := decryptContent(enc); got != plain {
			t.Fatalf("round trip mismatch: want %q got %q", plain, got)
		}
	}
}

func TestEncryptEmptyIsNoop(t *testing.T) {
	_ = initEncryption("test-secret")
	if got := encryptContent(""); got != "" {
		t.Fatalf("empty content should stay empty, got %q", got)
	}
}

func TestDecryptLegacyPlaintext(t *testing.T) {
	_ = initEncryption("test-secret")
	// Un message stocké en clair (sans préfixe) doit être renvoyé tel quel.
	const legacy = "old plaintext message"
	if got := decryptContent(legacy); got != legacy {
		t.Fatalf("legacy plaintext mangled: want %q got %q", legacy, got)
	}
}

func TestDecryptMalformedBase64ReturnsStored(t *testing.T) {
	_ = initEncryption("test-secret")
	// Legacy fallback: invalid base64 after the prefix must not panic and must
	// return the stored value as-is (so reads never fail due to encryption).
	stored := encPrefix + "not-base64!!!"
	if got := decryptContent(stored); got != stored {
		t.Fatalf("expected stored value for malformed base64, got %q", got)
	}
}

func TestDecryptWrongKeyReturnsStored(t *testing.T) {
	_ = initEncryption("key-A")
	enc := encryptContent("secret")
	// Avec une autre clé, le déchiffrement échoue : on renvoie la valeur stockée
	// sans planter.
	_ = initEncryption("key-B")
	if got := decryptContent(enc); got != enc {
		t.Fatalf("expected stored value on decrypt failure, got %q", got)
	}
}
