package main

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

// requireInternalKey protège les routes internes service-à-service. La clé est
// comparée en temps constant ; sans clé configurée, tout est refusé.
func requireInternalKey(expected string) gin.HandlerFunc {
	expectedBytes := []byte(expected)
	return func(c *gin.Context) {
		provided := []byte(c.GetHeader("X-Internal-Key"))
		if len(expectedBytes) == 0 || subtle.ConstantTimeCompare(provided, expectedBytes) != 1 {
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "forbidden")
			return
		}
		c.Next()
	}
}

// setStatusHandler met à jour l'état de modération d'un compte (endpoint interne
// appelé par le user-service lors d'une sanction). Révoque les sessions si le
// compte est désactivé.
func (s *authService) setStatusHandler(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	if err := s.setUserStatus(userID, req.Status); err != nil {
		if errors.Is(err, errValidation) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid status")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to update status")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *authService) registerHandler(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("register", c.ClientIP(), req.Email)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	err := s.register(req.Username, req.Email, req.Password, req.TermsAccepted, req.TermsVersion)
	if errors.Is(err, errConflict) {
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusConflict, shared.ErrConflict, "username or email already exists")
		return
	}
	if errors.Is(err, errValidation) {
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
		return
	}
	if err != nil {
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to create user")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusCreated, messageResponse{
		Message: "account created, please check your email to verify your address",
	})
}

func (s *authService) verifyEmailHandler(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("verify-email", c.ClientIP(), req.Token)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	err := s.verifyEmail(req.Token)
	if err != nil {
		if errors.Is(err, errValidation) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, errInvalidToken) || errors.Is(err, errTokenExpired) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid or expired verification token")
			return
		}
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to verify email")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, messageResponse{Message: "email verified, you can now log in"})
}

// anonymousEmailAction factorise les endpoints anti-énumération qui prennent un
// email, le rate-limitent par IP+email et répondent TOUJOURS 202 (resend
// verification, request password reset). Chaque tentative consomme le quota
// (RegisterFailure) sans jamais RegisterSuccess : sinon un attaquant remettrait
// le compteur à zéro et rendrait l'action illimitée. La réponse étant constante,
// il n'y a aucune notion succès/échec exploitable pour énumérer les comptes.
func (s *authService) anonymousEmailAction(
	c *gin.Context,
	email, action string,
	do func(string) error,
	internalErrMsg, successMsg string,
) {
	rateKey := s.rateLimitKey(action, c.ClientIP(), email)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}
	s.rateLimiter.RegisterFailure(rateKey)

	if err := do(email); err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, internalErrMsg)
		return
	}

	c.JSON(http.StatusAccepted, messageResponse{Message: successMsg})
}

func (s *authService) resendVerificationHandler(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}
	s.anonymousEmailAction(c, req.Email, "resend-verification", s.resendVerification,
		"failed to resend verification email",
		"if the account exists and is unverified, a verification email has been sent")
}

func (s *authService) requestPasswordResetHandler(c *gin.Context) {
	var req requestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}
	s.anonymousEmailAction(c, req.Email, "request-password-reset", s.requestPasswordReset,
		"failed to send password reset email",
		"if an account exists for this email, a password reset link has been sent")
}

func (s *authService) resetPasswordHandler(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("reset-password", c.ClientIP(), req.Token)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	err := s.resetPassword(req.Token, req.Password)
	if err != nil {
		if errors.Is(err, errValidation) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, errInvalidToken) || errors.Is(err, errTokenExpired) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid or expired reset token")
			return
		}
		if errors.Is(err, errSamePassword) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "new password must differ from the current one")
			return
		}
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to reset password")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, messageResponse{Message: "password updated, you can now log in"})
}

func (s *authService) changePasswordHandler(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok || strings.TrimSpace(identity.UserID) == "" {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("change-password", c.ClientIP(), identity.UserID)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	err := s.changePassword(identity.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, errValidation) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "unknown user")
			return
		}
		if errors.Is(err, errInvalidCredentials) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "current password is incorrect")
			return
		}
		if errors.Is(err, errSamePassword) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "new password must differ from the current one")
			return
		}
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to change password")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, messageResponse{Message: "password changed, please log in again"})
}

func (s *authService) loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}

	rateKey := s.rateLimitKey("login", c.ClientIP(), identifier)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	result, err := s.login(identifier, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, errValidation) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, errInvalidCredentials) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid credentials")
			return
		}
		if errors.Is(err, errEmailNotVerified) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "email not verified")
			return
		}
		if errors.Is(err, errAccountSuspended) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "account suspended")
			return
		}
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to login")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	if result.MFARequired {
		c.JSON(http.StatusOK, mfaRequiredResponse{MFARequired: true, PendingToken: result.PendingToken})
		return
	}
	c.JSON(http.StatusOK, authResponse(result.Tokens))
}

func (s *authService) refreshHandler(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("refresh", c.ClientIP(), req.RefreshToken)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	tokens, err := s.refresh(req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, errValidation) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, errInvalidToken) || errors.Is(err, errRefreshRevoked) || errors.Is(err, errRefreshExpired) || errors.Is(err, errRefreshReuse) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid refresh token")
			return
		}
		if errors.Is(err, errEmailNotVerified) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "email not verified")
			return
		}
		if errors.Is(err, errAccountSuspended) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "account suspended")
			return
		}
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to refresh session")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, authResponse(tokens))
}

func (s *authService) logoutHandler(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	if err := s.logout(req.RefreshToken); err != nil {
		if errors.Is(err, errValidation) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to logout")
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *authService) mfaSetupHandler(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok || strings.TrimSpace(identity.UserID) == "" {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}

	result, err := s.setupMFA(identity.UserID)
	if err != nil {
		if errors.Is(err, errMFAAlreadyEnabled) {
			shared.AbortError(c, http.StatusConflict, shared.ErrConflict, "MFA already enabled")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to set up MFA")
		return
	}

	c.JSON(http.StatusOK, mfaSetupResponse(result))
}

func (s *authService) mfaConfirmHandler(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok || strings.TrimSpace(identity.UserID) == "" {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}

	var req mfaConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("mfa-confirm", c.ClientIP(), identity.UserID)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	if err := s.confirmMFA(identity.UserID, req.Code); err != nil {
		if errors.Is(err, errMFAInvalidCode) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid OTP code")
			return
		}
		if errors.Is(err, errMFAAlreadyEnabled) {
			shared.AbortError(c, http.StatusConflict, shared.ErrConflict, "MFA already enabled")
			return
		}
		if errors.Is(err, errMFANotEnabled) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "MFA setup not initiated")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to confirm MFA")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, messageResponse{Message: "MFA enabled"})
}

func (s *authService) mfaDisableHandler(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok || strings.TrimSpace(identity.UserID) == "" {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}

	var req mfaDisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("mfa-disable", c.ClientIP(), identity.UserID)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	if err := s.disableMFA(identity.UserID, req.Code); err != nil {
		if errors.Is(err, errMFAInvalidCode) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid OTP or backup code")
			return
		}
		if errors.Is(err, errMFANotEnabled) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "MFA not enabled")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to disable MFA")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, messageResponse{Message: "MFA disabled"})
}

func (s *authService) mfaVerifyHandler(c *gin.Context) {
	var req mfaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	rateKey := s.rateLimitKey("mfa-verify", c.ClientIP(), req.PendingToken)
	if !s.rateLimiter.Allow(rateKey) {
		shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "too many attempts, retry later")
		return
	}

	tokens, err := s.verifyMFALogin(req.PendingToken, req.Code, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, errValidation) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		if errors.Is(err, errMFAInvalidCode) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid OTP code")
			return
		}
		if errors.Is(err, errInvalidToken) || errors.Is(err, errTokenExpired) {
			s.rateLimiter.RegisterFailure(rateKey)
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid or expired MFA session")
			return
		}
		s.rateLimiter.RegisterFailure(rateKey)
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to verify MFA")
		return
	}
	s.rateLimiter.RegisterSuccess(rateKey)

	c.JSON(http.StatusOK, authResponse(tokens))
}

func (s *authService) meHandler(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok || strings.TrimSpace(identity.UserID) == "" {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}

	user, err := s.me(identity.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "unknown user")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch user")
		return
	}

	var mfa authMFAModel
	mfaEnabled := s.db.First(&mfa, "user_id = ? AND enabled = true", user.ID).Error == nil

	c.JSON(http.StatusOK, meResponse{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		Role:           user.Role,
		Status:         user.Status,
		Language:       "fr",
		Theme:          "system",
		Bio:            nil,
		AvatarUrl:      nil,
		FollowersCount: 0,
		FollowingCount: 0,
		MFAEnabled:     mfaEnabled,
		CreatedAt:      user.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      user.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (s *authService) oauthRedirectHandler(c *gin.Context) {
	provider := c.Param("provider")
	authURL, state, err := s.oauthRedirectURL(provider)
	if err != nil {
		if errors.Is(err, errOAuthProviderDisabled) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "provider not available")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to build OAuth URL")
		return
	}

	// Store state in a short-lived cookie so the callback can verify it
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	c.Redirect(http.StatusFound, authURL)
}

func (s *authService) oauthCallbackHandler(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState == "" || storedState != state {
		c.Redirect(http.StatusFound, s.cfg.AppBaseURL+"/auth/oauth/callback?reason=state")
		return
	}
	c.SetCookie("oauth_state", "", -1, "/", "", false, true) // clear

	if code == "" {
		c.Redirect(http.StatusFound, s.cfg.AppBaseURL+"/auth/oauth/callback?reason=denied")
		return
	}

	result, err := s.oauthCallback(provider, code)
	if err != nil {
		c.Redirect(http.StatusFound, s.cfg.AppBaseURL+"/auth/oauth/callback?reason=provider")
		return
	}

	exchangeCode, err := s.issueOAuthExchangeCode(result.UserID)
	if err != nil {
		c.Redirect(http.StatusFound, s.cfg.AppBaseURL+"/auth/oauth/callback?reason=internal")
		return
	}

	c.Redirect(http.StatusFound, s.cfg.AppBaseURL+"/auth/oauth/callback?code="+exchangeCode)
}

func (s *authService) oauthExchangeHandler(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	tokens, err := s.redeemOAuthExchangeCode(req.Code)
	if err != nil {
		if errors.Is(err, errInvalidToken) || errors.Is(err, errTokenExpired) {
			shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "invalid or expired exchange code")
			return
		}
		if errors.Is(err, errValidation) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to exchange code")
		return
	}

	c.JSON(http.StatusOK, authResponse(tokens))
}
