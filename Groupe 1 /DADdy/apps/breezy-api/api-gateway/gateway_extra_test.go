package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestWithCSRF(t *testing.T) {
	allowed := []string{"http://localhost:5173"}
	router := gin.New()
	router.Use(withCSRF(allowed))
	router.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	router.POST("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	do := func(method string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/x", nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("safe method passes without marker", func(t *testing.T) {
		if w := do(http.MethodGet, nil); w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("mutating without X-Requested-With is 403", func(t *testing.T) {
		if w := do(http.MethodPost, nil); w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("mutating with forbidden origin is 403", func(t *testing.T) {
		w := do(http.MethodPost, map[string]string{
			"Origin":           "http://evil.com",
			"X-Requested-With": "XMLHttpRequest",
		})
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("mutating with marker and allowed origin passes", func(t *testing.T) {
		w := do(http.MethodPost, map[string]string{
			"Origin":           "http://localhost:5173",
			"X-Requested-With": "XMLHttpRequest",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}

func TestBuildJWTVerifyKeys(t *testing.T) {
	hs := jwt.SigningMethodHS256.Alg()
	rs := jwt.SigningMethodRS256.Alg()

	t.Run("HS256 active key", func(t *testing.T) {
		keys, err := buildJWTVerifyKeys(gatewayConfig{JWTAlg: hs, JWTSecret: "s", JWTActiveKID: "v1"})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if _, ok := keys["v1"]; !ok || len(keys) != 1 {
			t.Errorf("keys = %v, want single v1 entry", keys)
		}
	})

	t.Run("HS256 with previous key", func(t *testing.T) {
		keys, err := buildJWTVerifyKeys(gatewayConfig{
			JWTAlg: hs, JWTSecret: "s", JWTActiveKID: "v1",
			JWTPreviousKID: "v0", JWTPreviousSecret: "old",
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(keys) != 2 {
			t.Errorf("keys = %v, want 2 entries", keys)
		}
	})

	t.Run("HS256 missing secret errors", func(t *testing.T) {
		if _, err := buildJWTVerifyKeys(gatewayConfig{JWTAlg: hs, JWTActiveKID: "v1"}); err == nil {
			t.Error("expected error for missing secret")
		}
	})

	t.Run("RS256 missing path errors", func(t *testing.T) {
		if _, err := buildJWTVerifyKeys(gatewayConfig{JWTAlg: rs, JWTActiveKID: "v1"}); err == nil {
			t.Error("expected error for missing public key path")
		}
	})

	t.Run("RS256 loads public key from file", func(t *testing.T) {
		path := writeRSAPublicKey(t)
		keys, err := buildJWTVerifyKeys(gatewayConfig{JWTAlg: rs, JWTActiveKID: "v1", JWTPublicKeyPath: path})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if _, ok := keys["v1"].(*rsa.PublicKey); !ok {
			t.Errorf("expected *rsa.PublicKey, got %T", keys["v1"])
		}
	})

	t.Run("RS256 bad path errors", func(t *testing.T) {
		if _, err := buildJWTVerifyKeys(gatewayConfig{JWTAlg: rs, JWTActiveKID: "v1", JWTPublicKeyPath: "/no/such/key.pem"}); err == nil {
			t.Error("expected error for unreadable key file")
		}
	})

	t.Run("unsupported alg errors", func(t *testing.T) {
		if _, err := buildJWTVerifyKeys(gatewayConfig{JWTAlg: jwt.SigningMethodES512.Alg()}); err == nil {
			t.Error("expected error for unsupported alg")
		}
	})
}

// writeRSAPublicKey génère une paire RSA et écrit la clé publique (PKIX PEM) dans
// un fichier temporaire, renvoyant son chemin.
func writeRSAPublicKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	path := filepath.Join(t.TempDir(), "pub.pem")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestWithRateLimitNilLimiter(t *testing.T) {
	t.Parallel()
	// limiter nil (Valkey non configuré) => fail-open : la requête passe.
	router := gin.New()
	router.Use(withRateLimit(nil, keyByIP("api"), 10))
	router.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fail-open)", w.Code)
	}
}

func TestNewRateLimiter(t *testing.T) {
	t.Parallel()
	if newRateLimiter(gatewayConfig{}) != nil {
		t.Error("empty ValkeyAddr should disable the limiter (nil)")
	}
	if newRateLimiter(gatewayConfig{ValkeyAddr: "localhost:6379"}) == nil {
		t.Error("configured ValkeyAddr should build a limiter")
	}
}

func TestSessionRevocationChecker(t *testing.T) {
	t.Parallel()

	t.Run("nil checker never revokes", func(t *testing.T) {
		t.Parallel()
		var s *sessionRevocationChecker
		if s.isRevoked("usr_1", time.Now()) {
			t.Error("nil checker should fail-open to not revoked")
		}
	})

	t.Run("empty addr yields nil checker", func(t *testing.T) {
		t.Parallel()
		if newSessionRevocationChecker(gatewayConfig{}) != nil {
			t.Error("empty ValkeyAddr should yield nil checker")
		}
	})

	t.Run("unreachable valkey fails open", func(t *testing.T) {
		t.Parallel()
		// Le port 1 rejette tout de suite => Get échoue => fail-open.
		s := newSessionRevocationChecker(gatewayConfig{ValkeyAddr: "127.0.0.1:1"})
		if s == nil {
			t.Fatal("expected non-nil checker for configured addr")
		}
		if s.isRevoked("usr_1", time.Now()) {
			t.Error("unreachable valkey should fail-open (not revoked)")
		}
	})
}

func TestNewGatewayRouter(t *testing.T) {
	// Backend factice : toute requête proxifiée y atterrit.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("backend"))
	}))
	defer backend.Close()

	cfg := testCfg()
	cfg.CORSAllowedOrigins = []string{"http://localhost:5173"}
	cfg.CoreTarget = mustParseURL(backend.URL)
	cfg.UserTarget = mustParseURL(backend.URL)
	cfg.AuthTarget = mustParseURL(backend.URL)
	cfg.PostTarget = mustParseURL(backend.URL)
	cfg.MsgTarget = mustParseURL(backend.URL)
	cfg.NotifTarget = mustParseURL(backend.URL)

	// Servir via un vrai serveur HTTP : le reverse proxy a besoin d'un
	// ResponseWriter réel (httptest.ResponseRecorder n'implémente pas certaines
	// interfaces requises par le proxy/gin).
	gw := httptest.NewServer(newGatewayRouter(cfg))
	defer gw.Close()

	get := func(path string) (int, string) {
		resp, err := http.Get(gw.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}

	t.Run("health is public", func(t *testing.T) {
		if code, _ := get("/health"); code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
	})

	t.Run("unknown route proxies to core backend", func(t *testing.T) {
		code, body := get("/unknown/path")
		if code != http.StatusOK || body != "backend" {
			t.Fatalf("status=%d body=%q, want 200/backend", code, body)
		}
	})
}

// TestGatewayMFAAndOAuthRouting vérifie le routage des chemins auth sociaux/MFA :
// deux backends distincts (auth vs core) prouvent la cible réelle, pas seulement un
// 200. Sans route explicite, ces chemins tomberaient sur NoRoute → core (404 en prod).
func TestGatewayMFAAndOAuthRouting(t *testing.T) {
	newBackend := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
	}
	authBackend := newBackend("auth")
	defer authBackend.Close()
	coreBackend := newBackend("core")
	defer coreBackend.Close()

	cfg := testCfg()
	cfg.AuthTarget = mustParseURL(authBackend.URL)
	cfg.CoreTarget = mustParseURL(coreBackend.URL)
	cfg.UserTarget = mustParseURL(coreBackend.URL)
	cfg.PostTarget = mustParseURL(coreBackend.URL)
	cfg.MsgTarget = mustParseURL(coreBackend.URL)
	cfg.NotifTarget = mustParseURL(coreBackend.URL)

	gw := httptest.NewServer(newGatewayRouter(cfg))
	defer gw.Close()

	req := func(token, path string) (int, string) {
		r, _ := http.NewRequest(http.MethodGet, gw.URL+path, nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}
	token := signToken(t, jwt.SigningMethodHS256, cfg.JWTSecret, "v1", nil)

	t.Run("oauth redirect is anonymous and routed to auth", func(t *testing.T) {
		if code, body := req("", "/auth/oauth/google"); code != http.StatusOK || body != "auth" {
			t.Fatalf("got %d/%q, want 200/auth", code, body)
		}
	})
	t.Run("mfa verify is anonymous and routed to auth", func(t *testing.T) {
		if code, body := req("", "/auth/mfa/verify"); code != http.StatusOK || body != "auth" {
			t.Fatalf("got %d/%q, want 200/auth", code, body)
		}
	})
	t.Run("mfa setup requires a JWT", func(t *testing.T) {
		if code, _ := req("", "/auth/mfa/setup"); code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401 without token", code)
		}
	})
	t.Run("mfa setup with a JWT is routed to auth", func(t *testing.T) {
		if code, body := req(token, "/auth/mfa/setup"); code != http.StatusOK || body != "auth" {
			t.Fatalf("got %d/%q, want 200/auth", code, body)
		}
	})
	t.Run("mfa disable requires a JWT", func(t *testing.T) {
		if code, _ := req("", "/auth/mfa"); code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401 without token", code)
		}
	})
}

func TestLoadGatewayConfig(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("JWT_ISSUER", "issuer-test")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a,http://b")
	cfg := loadGatewayConfig()
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want 9999", cfg.Port)
	}
	if cfg.JWTIssuer != "issuer-test" {
		t.Errorf("JWTIssuer = %q, want issuer-test", cfg.JWTIssuer)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("CORSAllowedOrigins = %v, want 2 entries", cfg.CORSAllowedOrigins)
	}
	if cfg.CoreTarget == nil || cfg.AuthTarget == nil {
		t.Error("service targets should be parsed")
	}
}
