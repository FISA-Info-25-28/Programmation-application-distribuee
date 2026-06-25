package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// Chiffrement des messages privés au repos.
//
// Le contenu textuel des messages est chiffré en AES-256-GCM avant insertion en
// base et déchiffré à la lecture. La clé dérive d'un secret serveur
// (MESSAGE_ENCRYPTION_KEY) via SHA-256, ce qui accepte n'importe quel format de
// secret tout en garantissant une clé de 32 octets.
//
// Ce n'est pas du chiffrement de bout en bout (le serveur peut déchiffrer) mais
// une protection au repos : une fuite de la base seule n'expose pas les messages.

// encPrefix marque un contenu chiffré. Les anciens messages, stockés en clair,
// n'ont pas ce préfixe et sont renvoyés tels quels (rétro-compatibilité).
const encPrefix = "enc:v1:"

var contentAEAD cipher.AEAD

// initEncryption dérive la clé AES-256 du secret et prépare le mode GCM.
func initEncryption(secret string) error {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM cipher: %w", err)
	}
	contentAEAD = aead
	return nil
}

// encryptContent chiffre un contenu en clair. Le contenu vide (message média
// seul) reste vide. En l'absence de clé initialisée, renvoie le clair (no-op).
func encryptContent(plain string) string {
	if plain == "" || contentAEAD == nil {
		return plain
	}
	nonce := make([]byte, contentAEAD.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return plain
	}
	sealed := contentAEAD.Seal(nonce, nonce, []byte(plain), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed)
}

// decryptContent déchiffre un contenu. Sans préfixe (message légataire en clair)
// ou en cas d'échec, renvoie la valeur stockée sans planter — la lecture ne doit
// jamais échouer à cause du chiffrement.
func decryptContent(stored string) string {
	if contentAEAD == nil || !strings.HasPrefix(stored, encPrefix) {
		return stored
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return stored
	}
	ns := contentAEAD.NonceSize()
	if len(raw) < ns {
		return stored
	}
	plain, err := contentAEAD.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return stored
	}
	return string(plain)
}
