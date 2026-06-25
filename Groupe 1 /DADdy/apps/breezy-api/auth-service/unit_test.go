package main

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateRegisterInput(t *testing.T) {
	t.Parallel()

	const goodPw = "Str0ng!Passphrase"

	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{name: "valid", username: "alice", email: "alice@example.com", password: goodPw},
		{name: "username too short", username: "ab", email: "alice@example.com", password: goodPw, wantErr: true},
		{name: "username too long", username: strings.Repeat("a", 51), email: "a@example.com", password: goodPw, wantErr: true},
		{name: "invalid email", username: "alice", email: "not-an-email", password: goodPw, wantErr: true},
		{name: "weak password propagates", username: "alice", email: "alice@example.com", password: "short", wantErr: true},
		{name: "password containing username", username: "alice", email: "x@example.com", password: "aliceStr0ng!", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRegisterInput(tt.username, tt.email, tt.password)
			if tt.wantErr && err == nil {
				t.Fatalf("validateRegisterInput() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateRegisterInput() = %v, want nil", err)
			}
		})
	}
}

func TestContainsIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		username string
		email    string
		want     bool
	}{
		{name: "contains username case-insensitive", password: "xxALICExx", username: "alice", want: true},
		{name: "contains email local part", password: "myjohndoepw", email: "johndoe@example.com", want: true},
		{name: "short username ignored", password: "abxyz", username: "ab", want: false},
		{name: "no match", password: "totallyunrelated", username: "alice", email: "alice@example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := containsIdentifier(tt.password, tt.username, tt.email); got != tt.want {
				t.Fatalf("containsIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailLocalPart(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"alice@example.com": "alice",
		"no-at-sign":        "no-at-sign",
		"@leading":          "@leading",
	}
	for in, want := range cases {
		if got := emailLocalPart(in); got != want {
			t.Errorf("emailLocalPart(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRateLimitKey(t *testing.T) {
	t.Parallel()
	s := &authService{}

	// Le sujet est haché : il n'apparaît jamais en clair dans la clé.
	key := s.rateLimitKey("login", "1.2.3.4", "Alice@Example.com")
	if strings.Contains(strings.ToLower(key), "alice") {
		t.Fatalf("key %q must not contain the raw subject", key)
	}

	// Normalisation : casse et espaces ne doivent pas changer la clé.
	if got := s.rateLimitKey("login", "1.2.3.4", "  alice@example.com "); got != key {
		t.Errorf("key not normalized: %q != %q", got, key)
	}

	// Sujet vide => bucket "anon" partagé.
	anon1 := s.rateLimitKey("login", "1.2.3.4", "")
	anon2 := s.rateLimitKey("login", "1.2.3.4", "   ")
	if anon1 != anon2 {
		t.Errorf("empty subjects should map to same anon key: %q != %q", anon1, anon2)
	}
	if anon1 == key {
		t.Error("anon key collides with a real subject key")
	}
}

func TestMemoryRateLimiter(t *testing.T) {
	t.Parallel()
	rl := newMemoryRateLimiter()
	const key = "login|1.2.3.4|abc"

	if !rl.Allow(key) {
		t.Fatal("fresh key should be allowed")
	}

	// Sous le seuil (authRLThreshold), toujours autorisé.
	for i := 0; i < authRLThreshold-1; i++ {
		rl.RegisterFailure(key)
	}
	if !rl.Allow(key) {
		t.Fatalf("should still be allowed after %d failures", authRLThreshold-1)
	}

	// Au seuil, le blocage s'arme.
	rl.RegisterFailure(key)
	if rl.Allow(key) {
		t.Fatalf("should be blocked after %d failures", authRLThreshold)
	}

	// Un succès remet complètement le compteur à zéro.
	rl.RegisterSuccess(key)
	if !rl.Allow(key) {
		t.Fatal("should be allowed again after RegisterSuccess")
	}
}

func TestHashToken(t *testing.T) {
	t.Parallel()
	h1 := hashToken("secret-token")
	h2 := hashToken("secret-token")
	if h1 != h2 {
		t.Fatal("hashToken must be deterministic")
	}
	if len(h1) != 64 {
		t.Fatalf("sha256 hex length = %d, want 64", len(h1))
	}
	if h1 == "secret-token" {
		t.Fatal("hashToken must not return the raw token")
	}
	if hashToken("other") == h1 {
		t.Fatal("different inputs must produce different hashes")
	}
}

func TestTrimUserAgent(t *testing.T) {
	t.Parallel()
	if got := trimUserAgent("  Mozilla/5.0  "); got != "Mozilla/5.0" {
		t.Errorf("trimUserAgent did not trim: %q", got)
	}
	long := strings.Repeat("u", 600)
	if got := trimUserAgent(long); len(got) != 512 {
		t.Errorf("trimUserAgent length = %d, want 512", len(got))
	}
}

func TestIsCommonPassword(t *testing.T) {
	t.Parallel()

	common := []string{"password", "123456", "AZERTY", "Passw0rd", "qwerty"}
	for _, p := range common {
		if !isCommonPassword(p) {
			t.Errorf("isCommonPassword(%q) = false, want true", p)
		}
	}

	notCommon := []string{"xK9#mQr2vL!", "correct horse battery staple", "MyUn1queP@ss!"}
	for _, p := range notCommon {
		if isCommonPassword(p) {
			t.Errorf("isCommonPassword(%q) = true, want false", p)
		}
	}
}

func TestMemoryRateLimiterBlockExpiry(t *testing.T) {
	t.Parallel()
	rl := newMemoryRateLimiter()
	const key = "ratelimit|expiry"

	for i := 0; i < authRLThreshold; i++ {
		rl.RegisterFailure(key)
	}
	if rl.Allow(key) {
		t.Fatal("should be blocked after threshold failures")
	}

	// Manually expire the block to simulate the sliding-window edge case where
	// BlockedTill expires exactly during Allow's evaluation.
	rl.mu.Lock()
	e := rl.entries[key]
	e.BlockedTill = time.Now().UTC().Add(-time.Millisecond)
	rl.entries[key] = e
	rl.mu.Unlock()

	if !rl.Allow(key) {
		t.Fatal("should be allowed again once BlockedTill has passed")
	}
}

func TestSignJWT(t *testing.T) {
	t.Parallel()
	cfg := authConfig{
		JWTSecret:   "test-secret-key",
		JWTAlg:      "HS256",
		JWTIssuer:   "daddy-auth-test",
		JWTAudience: "daddy-api-test",
		JWTKID:      "v1",
	}
	prepareAuthSigningKey(&cfg)

	signed, err := signJWT("usr_1", "moderator", "alice", cfg, time.Hour)
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}

	parsed, err := jwt.Parse(signed, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwtAlgHS256 {
			t.Fatalf("unexpected alg %s", token.Method.Alg())
		}
		return []byte("test-secret-key"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("parse signed token: %v (valid=%v)", err, parsed.Valid)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("unexpected claims type %T", parsed.Claims)
	}
	if claims["sub"] != "usr_1" {
		t.Errorf("sub = %v, want usr_1", claims["sub"])
	}
	if claims["role"] != "moderator" {
		t.Errorf("role = %v, want moderator", claims["role"])
	}
	if claims["username"] != "alice" {
		t.Errorf("username = %v, want alice", claims["username"])
	}
	if claims["iss"] != "daddy-auth-test" {
		t.Errorf("iss = %v, want daddy-auth-test", claims["iss"])
	}
	if parsed.Header["kid"] != "v1" {
		t.Errorf("kid header = %v, want v1", parsed.Header["kid"])
	}
}
