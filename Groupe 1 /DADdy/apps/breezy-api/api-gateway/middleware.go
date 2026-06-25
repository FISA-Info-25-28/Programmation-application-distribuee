package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

func withCORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		if isAllowedOrigin(origin, allowedOrigins) {
			if containsWildcard(allowedOrigins) {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Requested-With,X-CSRF-Token")
		c.Header("Access-Control-Expose-Headers", "Content-Length,Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// withCSRF rejects cross-origin state-mutating requests that lack the
// X-Requested-With: XMLHttpRequest marker. Because Bearer tokens cannot be
// attached cross-origin without CORS approval, CSRF is already structurally
// mitigated; this middleware adds a defense-in-depth layer by:
//  1. Verifying the Origin header (when present) is in the allowed list.
//  2. Requiring the X-Requested-With: XMLHttpRequest header, which browsers
//     will not send on cross-site simple requests without a CORS preflight.
//
// OAuth redirect callbacks (GET /auth/oauth/*) are browser navigations and
// are excluded from both checks since they carry no session credentials.
func withCSRF(allowedOrigins []string) gin.HandlerFunc {
	safeMethods := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}
	return func(c *gin.Context) {
		if safeMethods[c.Request.Method] {
			c.Next()
			return
		}

		// Origin present → must be in allowed list.
		if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" {
			if !isAllowedOrigin(origin, allowedOrigins) {
				shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "forbidden origin")
				return
			}
		}

		// All mutating XHR/fetch calls from the SPA must carry this header.
		if !strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Requested-With")), "XMLHttpRequest") {
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "missing X-Requested-With header")
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || strings.EqualFold(origin, allowed) {
			return true
		}
	}

	return false
}

func containsWildcard(allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
	}

	return false
}

// requireJWT valide le Bearer token et injecte l'identité dans les headers
// X-User-Id / X-User-Role. Middleware gin : il s'enchaîne via c.Next(), ce qui
// permet à un rate-limiter placé après lui de lire l'identité (clé par user).
//
// revoker (peut être nil) rejette en plus les tokens invalidés en masse après un
// changement de mot de passe, fermant la fenêtre où les autres sessions
// resteraient valides jusqu'à l'expiration naturelle du JWT.
func requireJWT(cfg gatewayConfig, revoker *sessionRevocationChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		sanitizeIdentityHeaders(c)

		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "missing authorization header")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid authorization scheme")
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		claims, ok := parseJWT(tokenString, cfg)
		if !ok {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid token")
			return
		}

		// parseJWT garantit claims.IssuedAt non nil.
		if revoker.isRevoked(claims.Subject, claims.IssuedAt.Time) {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "session revoked")
			return
		}

		c.Request.Header.Set(shared.HeaderUserID, claims.Subject)
		c.Request.Header.Set(shared.HeaderUserRole, claims.Role)
		c.Request.Header.Set(shared.HeaderUserUsername, claims.Username)
		c.Next()
	}
}

func withIdentityHeaderSanitizer() gin.HandlerFunc {
	return func(c *gin.Context) {
		sanitizeIdentityHeaders(c)
		c.Next()
	}
}

func sanitizeIdentityHeaders(c *gin.Context) {
	for _, header := range []string{
		shared.HeaderUserID,
		shared.HeaderUserRole,
		shared.HeaderUserUsername,
		"X-User-ID",
		"X-User-Role",
		"X-Auth-Verified",
	} {
		c.Request.Header.Del(header)
	}
}
