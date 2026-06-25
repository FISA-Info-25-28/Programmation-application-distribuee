package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

const (
	oauthStateLength    = 32
	oauthExchangeTTL    = 2 * time.Minute
	oauthProviderGoogle = "google"
	oauthProviderGitHub = "github"
)

var errOAuthProviderDisabled = errors.New("oauth_provider_disabled")

type oauthUserInfo struct {
	ProviderUID string
	Email       string
	Username    string // suggested username, may need deduplication
}

func (s *authService) oauthConfig(provider string) (*oauth2.Config, error) {
	baseURL := strings.TrimRight(s.cfg.AppBaseURL, "/")
	redirectURL := fmt.Sprintf("%s/api/auth/oauth/%s/callback", baseURL, provider)

	switch provider {
	case oauthProviderGoogle:
		if s.cfg.GoogleClientID == "" {
			return nil, errOAuthProviderDisabled
		}
		return &oauth2.Config{
			ClientID:     s.cfg.GoogleClientID,
			ClientSecret: s.cfg.GoogleClientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}, nil

	case oauthProviderGitHub:
		if s.cfg.GitHubClientID == "" {
			return nil, errOAuthProviderDisabled
		}
		return &oauth2.Config{
			ClientID:     s.cfg.GitHubClientID,
			ClientSecret: s.cfg.GitHubClientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %s", errOAuthProviderDisabled, provider)
	}
}

func (s *authService) oauthRedirectURL(provider string) (authURL, state string, err error) {
	cfg, err := s.oauthConfig(provider)
	if err != nil {
		return "", "", err
	}
	rawState, err := randomToken()
	if err != nil {
		return "", "", err
	}
	state = rawState[:oauthStateLength]
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), state, nil
}

type oauthResult struct {
	UserID string
	Tokens tokenPair
}

// findOrCreateSocialUser finds an existing social link or creates/links the user.
// Priority:
//  1. Existing social account with this provider+UID → log in
//  2. Existing auth user with same email → link provider + log in
//  3. New user → create auth record + social link
func (s *authService) findOrCreateSocialUser(provider string, info oauthUserInfo, providerToken *oauth2.Token) (oauthResult, error) {
	now := time.Now().UTC()

	// 1. Existing social link
	var social authSocialAccountModel
	err := s.db.First(&social, "provider = ? AND provider_uid = ?", provider, info.ProviderUID).Error
	if err == nil {
		user, err := s.loadSessionUser(social.UserID)
		if err != nil {
			return oauthResult{}, err
		}
		s.db.Model(&social).Updates(map[string]any{
			"access_token":  providerToken.AccessToken,
			"refresh_token": providerToken.RefreshToken,
			"expires_at":    providerToken.Expiry,
			"updated_at":    now,
		})
		tokens, err := s.issueTokenPair(user, "", "")
		return oauthResult{UserID: user.ID, Tokens: tokens}, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return oauthResult{}, err
	}

	// 2. Existing user with same email → link
	if info.Email != "" {
		var user authUserModel
		if err := s.db.First(&user, "email = ?", info.Email).Error; err == nil {
			if err := s.linkSocialAccount(user.ID, provider, info, providerToken, now); err != nil {
				return oauthResult{}, err
			}
			sessionUser, err := s.loadSessionUser(user.ID)
			if err != nil {
				return oauthResult{}, err
			}
			tokens, err := s.issueTokenPair(sessionUser, "", "")
			return oauthResult{UserID: user.ID, Tokens: tokens}, err
		}
	}

	// 3. Create new user
	return s.createSocialUser(provider, info, providerToken, now)
}

func (s *authService) linkSocialAccount(userID, provider string, info oauthUserInfo, tok *oauth2.Token, now time.Time) error {
	var expiry *time.Time
	if !tok.Expiry.IsZero() {
		t := tok.Expiry
		expiry = &t
	}
	return s.db.Create(&authSocialAccountModel{
		ID:           randomID("soc"),
		UserID:       userID,
		Provider:     provider,
		ProviderUID:  info.ProviderUID,
		Email:        info.Email,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    expiry,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error
}

func (s *authService) createSocialUser(provider string, info oauthUserInfo, tok *oauth2.Token, now time.Time) (oauthResult, error) {
	username := deduplicateUsername(s.db, sanitizeUsername(info.Username))
	email := info.Email
	if email == "" {
		email = fmt.Sprintf("%s_%s@oauth.breezy.local", provider, info.ProviderUID)
	}

	verified := now
	user := authUserModel{
		ID:              randomID("usr"),
		Username:        username,
		Email:           email,
		PasswordHash:    "",
		Role:            defaultUserRole,
		EmailVerifiedAt: &verified,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		var expiry *time.Time
		if !tok.Expiry.IsZero() {
			t := tok.Expiry
			expiry = &t
		}
		return tx.Create(&authSocialAccountModel{
			ID:           randomID("soc"),
			UserID:       user.ID,
			Provider:     provider,
			ProviderUID:  info.ProviderUID,
			Email:        info.Email,
			AccessToken:  tok.AccessToken,
			RefreshToken: tok.RefreshToken,
			ExpiresAt:    expiry,
			CreatedAt:    now,
			UpdatedAt:    now,
		}).Error
	}); err != nil {
		return oauthResult{}, fmt.Errorf("create social user transaction: %w", err)
	}

	if err := s.syncUserProfile(user); err != nil {
		log.Printf("oauth: failed to sync user profile for %s: %v", user.ID, err)
	}

	tokens, err := s.issueTokenPair(user, "", "")
	return oauthResult{UserID: user.ID, Tokens: tokens}, err
}

func sanitizeUsername(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(raw) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) > 30 {
		s = s[:30]
	}
	if s == "" {
		s = defaultUserRole
	}
	return s
}

func deduplicateUsername(db *gorm.DB, base string) string {
	candidate := base
	for i := 1; i <= 999; i++ {
		var count int64
		db.Model(&authUserModel{}).Where("username = ?", candidate).Count(&count)
		if count == 0 {
			return candidate
		}
		candidate = fmt.Sprintf("%s%d", base, i)
	}
	return base + "_" + randomID("u")
}

// issueOAuthExchangeCode creates a short-lived single-use code that the frontend
// exchanges for a real token pair after the OAuth redirect completes.
func (s *authService) issueOAuthExchangeCode(userID string) (string, error) {
	rawCode, err := randomToken()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := s.db.Create(&authOAuthExchangeModel{
		ID:        randomID("oex"),
		UserID:    userID,
		CodeHash:  hashToken(rawCode),
		ExpiresAt: now.Add(oauthExchangeTTL),
		CreatedAt: now,
	}).Error; err != nil {
		return "", err
	}
	return rawCode, nil
}

// redeemOAuthExchangeCode exchanges a one-time code for a token pair.
func (s *authService) redeemOAuthExchangeCode(rawCode string) (tokenPair, error) {
	rawCode = strings.TrimSpace(rawCode)
	if rawCode == "" {
		return tokenPair{}, fmt.Errorf("%w: code is required", errValidation)
	}

	codeHash := hashToken(rawCode)
	now := time.Now().UTC()

	var record authOAuthExchangeModel
	if err := s.db.First(&record, "code_hash = ?", codeHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tokenPair{}, errInvalidToken
		}
		return tokenPair{}, err
	}
	if now.After(record.ExpiresAt) {
		return tokenPair{}, errTokenExpired
	}

	if err := s.db.Delete(&authOAuthExchangeModel{}, "id = ?", record.ID).Error; err != nil {
		return tokenPair{}, err
	}

	user, err := s.loadSessionUser(record.UserID)
	if err != nil {
		return tokenPair{}, err
	}
	return s.issueTokenPair(user, "", "")
}
