package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

func newGatewayRouter(cfg gatewayConfig) *gin.Engine {
	limiter := newRateLimiter(cfg)

	coreProxy := httputil.NewSingleHostReverseProxy(cfg.CoreTarget)
	userProxy := httputil.NewSingleHostReverseProxy(cfg.UserTarget)
	authProxy := httputil.NewSingleHostReverseProxy(cfg.AuthTarget)
	postProxy := httputil.NewSingleHostReverseProxy(cfg.PostTarget)
	msgProxy := httputil.NewSingleHostReverseProxy(cfg.MsgTarget)
	notifProxy := httputil.NewSingleHostReverseProxy(cfg.NotifTarget)
	notifProxy.FlushInterval = -1 // flush immediately for SSE streams

	router := gin.New()

	// Proxies de confiance pour c.ClientIP() : sans configuration explicite, Gin
	// fait confiance à tous les proxies et l'IP cliente peut être usurpée via
	// X-Forwarded-For/X-Real-IP → contournement du rate limiting par IP. Liste
	// vide → SetTrustedProxies(nil) : seule l'adresse réelle de la connection est
	// utilisée. En prod, renseigner TRUSTED_PROXIES avec les IP/CIDR amont.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}

	csrf := withCSRF(cfg.CORSAllowedOrigins)
	router.Use(withCORS(cfg.CORSAllowedOrigins), gin.Recovery(), gin.Logger(), withIdentityHeaderSanitizer(), csrf)
	router.HandleMethodNotAllowed = true

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, shared.NewHealthResponse("api-gateway"))
	})

	// Middlewares réutilisables. L'ordre dans chaque route est :
	//   logReq → [jwt] → rate-limit → proxy
	// jwt placé AVANT apiLimit pour que keyByUser voie l'identité.
	logReq := requestLogging()
	jwt := requireJWT(cfg, newSessionRevocationChecker(cfg))
	authLimit := withRateLimit(limiter, keyByIP("auth"), cfg.RateLimitAuthPerMin) // anonyme → par IP
	apiLimit := withRateLimit(limiter, keyByUser("api"), cfg.RateLimitAPIPerMin)  // authentifié → par user

	core := proxyHandler(coreProxy)
	user := proxyHandler(userProxy)
	auth := proxyHandler(authProxy)
	post := proxyHandler(postProxy)
	msg := proxyHandler(msgProxy)
	notif := proxyHandler(notifProxy)

	// ── Auth (anonyme) : limite stricte par IP (anti brute-force) ──
	router.Any("/auth/register", logReq, authLimit, auth)
	router.Any("/auth/verify-email", logReq, authLimit, auth)
	router.Any("/auth/resend-verification", logReq, authLimit, auth)
	router.Any("/auth/request-password-reset", logReq, authLimit, auth)
	router.Any("/auth/reset-password", logReq, authLimit, auth)
	router.Any("/auth/login", logReq, authLimit, auth)
	router.Any("/auth/refresh", logReq, authLimit, auth)
	router.Any("/auth/logout", logReq, authLimit, auth)
	// OAuth2 social login (anonyme) : redirection initiale, callback du provider
	// et échange du code court-lived. Tout passe par l'auth-service.
	router.Any("/auth/oauth/*path", logReq, authLimit, auth)
	// MFA. `verify` est une étape du login (anonyme, basée sur pendingToken) →
	// limite stricte par IP comme /auth/login (anti brute-force). setup/confirm/
	// disable gèrent le MFA d'un compte déjà connecté → JWT requis ; l'auth-service
	// lit l'identité injectée par le gateway via RequireIdentity.
	router.Any("/auth/mfa/verify", logReq, authLimit, auth)
	router.Any("/auth/mfa/setup", logReq, jwt, apiLimit, auth)
	router.Any("/auth/mfa/confirm", logReq, jwt, apiLimit, auth)
	router.Any("/auth/mfa", logReq, jwt, apiLimit, auth)
	router.Any("/auth/me", logReq, jwt, apiLimit, auth)
	// Changement de mot de passe (connecté) : exige un JWT valide ; le gateway
	// injecte l'identité que l'auth-service lit via RequireIdentity.
	router.Any("/auth/change-password", logReq, jwt, apiLimit, auth)

	// ── Users / posts (authentifié) : limite par utilisateur ──
	router.Any("/users", logReq, jwt, apiLimit, user)
	router.Any("/users/*path", logReq, jwt, apiLimit, func(c *gin.Context) {
		// GET /users/:id/{posts,rebreezed,bookmarks} sont servis par post-service.
		path := c.Param("path")
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		postServiceSubs := map[string]bool{"posts": true, "rebreezed": true, "liked": true, "bookmarks": true}
		if c.Request.Method == http.MethodGet && len(parts) == 2 && parts[0] != "" && postServiceSubs[parts[1]] {
			post(c)
		} else {
			user(c)
		}
	})

	router.Any("/posts", logReq, jwt, apiLimit, post)
	router.Any("/posts/*path", logReq, jwt, apiLimit, post)
	// Signalements (modération) : servis par le post-service.
	router.Any("/reports", logReq, jwt, apiLimit, post)
	router.Any("/reports/*path", logReq, jwt, apiLimit, post)
	router.Any("/media/*path", logReq, jwt, apiLimit, post)
	router.Any("/hashtags/*path", logReq, jwt, apiLimit, post)

	router.Any("/conversations", logReq, jwt, apiLimit, msg)
	router.Any("/conversations/*path", logReq, jwt, apiLimit, msg)

	router.Any("/notifications", logReq, jwt, apiLimit, notif)
	router.Any("/notifications/*path", logReq, jwt, apiLimit, notif)
	router.Any("/subscriptions", logReq, jwt, apiLimit, notif)
	router.Any("/subscriptions/*path", logReq, jwt, apiLimit, notif)

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	router.NoRoute(logReq, core)

	return router
}
