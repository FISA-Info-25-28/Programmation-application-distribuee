package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"daddy/apps/breezy-api/internal/shared"
)

func init() { gin.SetMode(gin.TestMode) }

func testCfg() gatewayConfig {
	return gatewayConfig{
		JWTSecret:      "test-secret-key",
		JWTIssuer:      "daddy-auth",
		JWTAudience:    "daddy-api",
		JWTAlg:         jwt.SigningMethodHS256.Alg(),
		JWTActiveKID:   "v1",
		JWTLeeway:      30 * time.Second,
		JWTMaxTokenAge: 20 * time.Minute,
	}
}

// signToken forge un JWT signé. mutate permet d'altérer les claims/headers par
// cas de test ; secret et kid sont fournis pour couvrir les rejets de signature.
func signToken(t *testing.T, method jwt.SigningMethod, secret, kid string, mutate func(*gatewayClaims)) string {
	t.Helper()
	now := time.Now().UTC()
	claims := &gatewayClaims{
		Role:     "user",
		Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "usr_1",
			Issuer:    "daddy-auth",
			Audience:  jwt.ClaimStrings{"daddy-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			ID:        "jti_1",
		},
	}
	if mutate != nil {
		mutate(claims)
	}
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestParseJWT(t *testing.T) {
	cfg := testCfg()
	const secret = "test-secret-key"

	t.Run("valid token", func(t *testing.T) {
		claims, ok := parseJWT(signToken(t, jwt.SigningMethodHS256, secret, "v1", nil), cfg)
		if !ok {
			t.Fatal("valid token rejected")
		}
		if claims.Subject != "usr_1" || claims.Role != "user" {
			t.Errorf("claims = %+v, want subject usr_1 / role user", claims)
		}
	})

	t.Run("accepts previous kid during rotation", func(t *testing.T) {
		rotated := cfg
		rotated.JWTPreviousKID = "v0"
		rotated.JWTPreviousSecret = "old-secret"
		_, ok := parseJWT(signToken(t, jwt.SigningMethodHS256, "old-secret", "v0", nil), rotated)
		if !ok {
			t.Fatal("token signed with previous key should still be accepted")
		}
	})

	rejections := []struct {
		name   string
		token  func() string
		altCfg *gatewayConfig
	}{
		{name: "empty token", token: func() string { return "" }},
		{name: "wrong secret", token: func() string { return signToken(t, jwt.SigningMethodHS256, "WRONG", "v1", nil) }},
		{name: "unexpected alg", token: func() string { return signToken(t, jwt.SigningMethodHS384, secret, "v1", nil) }},
		{name: "missing kid", token: func() string { return signToken(t, jwt.SigningMethodHS256, secret, "", nil) }},
		{name: "unknown kid", token: func() string { return signToken(t, jwt.SigningMethodHS256, secret, "v9", nil) }},
		{
			name: "wrong issuer",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) { c.Issuer = "evil" })
			},
		},
		{
			name: "wrong audience",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) { c.Audience = jwt.ClaimStrings{"other"} })
			},
		},
		{
			name: "expired",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) {
					c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Minute))
				})
			},
		},
		{
			name: "issued too long ago (max token age)",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) {
					old := time.Now().Add(-25 * time.Minute)
					c.IssuedAt = jwt.NewNumericDate(old)
					c.NotBefore = jwt.NewNumericDate(old)
					c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(10 * time.Minute))
				})
			},
		},
		{
			name: "missing jti",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) { c.ID = "" })
			},
		},
		{
			name: "missing role",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) { c.Role = "" })
			},
		},
		{
			name: "missing subject",
			token: func() string {
				return signToken(t, jwt.SigningMethodHS256, secret, "v1", func(c *gatewayClaims) { c.Subject = "" })
			},
		},
	}

	for _, tt := range rejections {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := parseJWT(tt.token(), cfg); ok {
				t.Fatalf("%s should have been rejected", tt.name)
			}
		})
	}
}

// echoIdentityRouter monte requireJWT puis un handler qui renvoie l'identité que
// le gateway a injectée dans les headers de requête.
func echoIdentityRouter(cfg gatewayConfig) *gin.Engine {
	r := gin.New()
	r.Use(requireJWT(cfg, nil)) // revoker nil = no-op (pas de Valkey)
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, c.Request.Header.Get(shared.HeaderUserID))
	})
	return r
}

func TestRequireJWTMiddleware(t *testing.T) {
	cfg := testCfg()
	router := echoIdentityRouter(cfg)

	doReq := func(setup func(*http.Request)) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		if setup != nil {
			setup(req)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("missing authorization is 401", func(t *testing.T) {
		if w := doReq(nil); w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("non-bearer scheme is 401", func(t *testing.T) {
		w := doReq(func(r *http.Request) { r.Header.Set("Authorization", "Basic abc") })
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("invalid token is 401", func(t *testing.T) {
		w := doReq(func(r *http.Request) { r.Header.Set("Authorization", "Bearer not.a.jwt") })
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})

	t.Run("valid token injects identity from claims", func(t *testing.T) {
		tok := signToken(t, jwt.SigningMethodHS256, cfg.JWTSecret, "v1", nil)
		w := doReq(func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tok) })
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if w.Body.String() != "usr_1" {
			t.Errorf("injected X-User-Id = %q, want usr_1", w.Body.String())
		}
	})

	// Sécurité : un X-User-Id fourni par le client doit être ignoré et remplacé
	// par celui dérivé du token vérifié — sinon n'importe qui usurperait une identité.
	t.Run("client-supplied identity header is overridden by claims", func(t *testing.T) {
		tok := signToken(t, jwt.SigningMethodHS256, cfg.JWTSecret, "v1", nil)
		w := doReq(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+tok)
			r.Header.Set(shared.HeaderUserID, "admin")
		})
		if w.Body.String() != "usr_1" {
			t.Fatalf("spoofed identity leaked: got %q, want usr_1", w.Body.String())
		}
	})

	t.Run("forged identity without token is rejected", func(t *testing.T) {
		w := doReq(func(r *http.Request) { r.Header.Set(shared.HeaderUserID, "admin") })
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

func TestIdentityHeaderSanitizerOnPublicRoute(t *testing.T) {
	r := gin.New()
	r.Use(withIdentityHeaderSanitizer())
	r.GET("/public", func(c *gin.Context) {
		c.String(http.StatusOK, "id=%q role=%q", c.Request.Header.Get(shared.HeaderUserID), c.Request.Header.Get(shared.HeaderUserRole))
	})

	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.Header.Set(shared.HeaderUserID, "admin")
	req.Header.Set(shared.HeaderUserRole, "administrator")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Body.String(); got != `id="" role=""` {
		t.Fatalf("sanitizer left forged headers: %s", got)
	}
}

func TestIsAllowedOriginAndWildcard(t *testing.T) {
	t.Parallel()
	allowed := []string{"http://localhost:5173", "https://app.breezy.com"}

	if !isAllowedOrigin("http://localhost:5173", allowed) {
		t.Error("exact origin should be allowed")
	}
	if !isAllowedOrigin("HTTP://LOCALHOST:5173", allowed) {
		t.Error("origin match should be case-insensitive")
	}
	if isAllowedOrigin("http://evil.com", allowed) {
		t.Error("unlisted origin should be denied")
	}
	if containsWildcard(allowed) {
		t.Error("no wildcard expected")
	}
	if !isAllowedOrigin("http://anything", []string{"*"}) || !containsWildcard([]string{"*"}) {
		t.Error("wildcard should allow any origin")
	}
}

func TestWithCORS(t *testing.T) {
	allowed := []string{"http://localhost:5173"}
	router := gin.New()
	router.Use(withCORS(allowed))
	router.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	t.Run("allowed origin gets ACAO header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Fatalf("ACAO = %q, want the origin echoed back", got)
		}
	})

	t.Run("disallowed origin gets no ACAO header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "http://evil.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("ACAO = %q, want empty for disallowed origin", got)
		}
	})

	t.Run("preflight OPTIONS is short-circuited with 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})
}

func TestSplitCSV(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want []string
	}{
		{in: "", want: []string{}},
		{in: "a,b,c", want: []string{"a", "b", "c"}},
		{in: " a , b ,, c ,", want: []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		got := splitCSV(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("splitCSV(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("splitCSV(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestMustParseURL(t *testing.T) {
	t.Parallel()
	u := mustParseURL("http://localhost:3102")
	if u.Scheme != "http" || u.Host != "localhost:3102" {
		t.Fatalf("parsed = %+v, want scheme http host localhost:3102", u)
	}
}

func TestRateLimitKeys(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if got := keyByIP("api")(c); got == "" || got[:7] != "api:ip:" {
		t.Errorf("keyByIP = %q, want api:ip:<addr>", got)
	}

	// Sans identité, keyByUser retombe sur l'IP.
	if got := keyByUser("auth")(c); got[:8] != "auth:ip:" {
		t.Errorf("keyByUser without identity = %q, want auth:ip: fallback", got)
	}

	// Avec identité, keyByUser cible l'utilisateur.
	c.Request.Header.Set(shared.HeaderUserID, "usr_42")
	if got := keyByUser("auth")(c); got != "auth:user:usr_42" {
		t.Errorf("keyByUser with identity = %q, want auth:user:usr_42", got)
	}
}
