package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

func newAuthRouter(service *authService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.HandleMethodNotAllowed = true

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, shared.NewHealthResponse("auth-service"))
	})

	router.POST("/auth/register", service.registerHandler)
	router.POST("/auth/verify-email", service.verifyEmailHandler)
	router.POST("/auth/resend-verification", service.resendVerificationHandler)
	router.POST("/auth/request-password-reset", service.requestPasswordResetHandler)
	router.POST("/auth/reset-password", service.resetPasswordHandler)
	router.POST("/auth/change-password", shared.RequireIdentity(), service.changePasswordHandler)
	router.POST("/auth/login", service.loginHandler)
	router.POST("/auth/refresh", service.refreshHandler)
	router.POST("/auth/logout", service.logoutHandler)
	router.GET("/auth/me", shared.RequireIdentity(), service.meHandler)

	// MFA: setup (generate secret + backup codes), confirm (enable), disable, verify (login step)
	router.POST("/auth/mfa/setup", shared.RequireIdentity(), service.mfaSetupHandler)
	router.POST("/auth/mfa/confirm", shared.RequireIdentity(), service.mfaConfirmHandler)
	router.DELETE("/auth/mfa", shared.RequireIdentity(), service.mfaDisableHandler)
	router.POST("/auth/mfa/verify", service.mfaVerifyHandler)

	// OAuth2 social login: redirect to provider, handle callback, exchange short-lived code
	router.GET("/auth/oauth/:provider", service.oauthRedirectHandler)
	router.GET("/auth/oauth/:provider/callback", service.oauthCallbackHandler)
	router.POST("/auth/oauth/exchange", service.oauthExchangeHandler)

	// Route interne service-à-service (jamais exposée par le gateway) : le
	// user-service y propage l'état de modération d'un compte.
	router.PATCH("/internal/users/:id/status", requireInternalKey(service.cfg.InternalKey), service.setStatusHandler)

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	return router
}
