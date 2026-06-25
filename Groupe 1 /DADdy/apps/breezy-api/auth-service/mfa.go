package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // TOTP (RFC 6238) mandates SHA-1
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	mfaBackupCodeCount = 8
	mfaTOTPDigits      = 6
	mfaTOTPPeriod      = 30
	mfaPendingTTL      = 5 * time.Minute
	totpURISecretParam = "secret"
)

func generateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate TOTP secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

func totpURI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	params := url.Values{
		totpURISecretParam: {secret},
		"issuer":           {issuer},
		"algorithm":        {"SHA1"},
		"digits":           {fmt.Sprintf("%d", mfaTOTPDigits)},
		"period":           {fmt.Sprintf("%d", mfaTOTPPeriod)},
	}
	return "otpauth://totp/" + label + "?" + params.Encode()
}

// verifyTOTP validates a 6-digit code against the secret, checking ±1 step for clock drift.
func verifyTOTP(secret, code string) bool {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}
	t := time.Now().Unix() / int64(mfaTOTPPeriod)
	for step := int64(-1); step <= 1; step++ {
		if computeTOTP(key, t+step) == code {
			return true
		}
	}
	return false
}

func computeTOTP(key []byte, counter int64) string {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(counter)) //nolint:gosec // G115: TOTP counter is never negative
	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	h := mac.Sum(nil)
	offset := h[len(h)-1] & 0xf
	code := (binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff) % 1_000_000
	return fmt.Sprintf("%06d", code)
}

func generateBackupCodes(n int) ([]string, error) {
	codes := make([]string, n)
	for i := range codes {
		buf := make([]byte, 5) // 10 uppercase hex chars
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("generate backup code: %w", err)
		}
		codes[i] = fmt.Sprintf("%X", buf)
	}
	return codes, nil
}
