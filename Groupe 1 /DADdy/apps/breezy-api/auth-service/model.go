package main

import "time"

// authMFAModel stores the TOTP secret for a user. Enabled=false means setup
// has been initiated but not yet confirmed with a valid OTP.
type authMFAModel struct {
	ID         string    `gorm:"primaryKey;size:64"`
	UserID     string    `gorm:"size:64;uniqueIndex;not null"`
	TOTPSecret string    `gorm:"size:512;not null"` // base32-encoded
	Enabled    bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}

// authMFABackupCodeModel holds a single hashed backup code. SHA-256 of the
// raw code is stored; the plain code is shown to the user exactly once.
type authMFABackupCodeModel struct {
	ID        string     `gorm:"primaryKey;size:64"`
	UserID    string     `gorm:"size:64;index;not null"`
	CodeHash  string     `gorm:"size:128;uniqueIndex;not null"`
	UsedAt    *time.Time `gorm:"index"`
	CreatedAt time.Time  `gorm:"not null"`
}

// authMFAPendingModel holds a short-lived token issued after credential
// verification when the user has MFA enabled. The front-end must exchange it
// for a real token pair via POST /auth/mfa/verify within ExpiresAt.
type authMFAPendingModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	UserID    string    `gorm:"size:64;index;not null"`
	TokenHash string    `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	IP        string    `gorm:"size:64"`
	UserAgent string    `gorm:"size:512"`
	CreatedAt time.Time `gorm:"not null"`
}

// authSocialAccountModel links a social provider identity to an auth user.
// One user can have multiple providers (Google + GitHub).
type authSocialAccountModel struct {
	ID           string `gorm:"primaryKey;size:64"`
	UserID       string `gorm:"size:64;index;not null"`
	Provider     string `gorm:"size:20;not null;uniqueIndex:idx_provider_uid"`  // "google" | "github"
	ProviderUID  string `gorm:"size:255;not null;uniqueIndex:idx_provider_uid"` // provider's user ID
	Email        string `gorm:"size:255;not null"`                              // email from provider
	AccessToken  string `gorm:"size:2048"`                                      // provider access token (optional, for future API calls)
	RefreshToken string `gorm:"size:2048"`                                      // provider refresh token
	ExpiresAt    *time.Time
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// authOAuthExchangeModel is a short-lived code issued after a successful OAuth
// callback so the frontend can exchange it (via XHR) for a real token pair.
// This avoids embedding tokens in URL parameters after the redirect.
type authOAuthExchangeModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	UserID    string    `gorm:"size:64;index;not null"`
	CodeHash  string    `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

type authUserModel struct {
	ID           string `gorm:"primaryKey;size:64"`
	Username     string `gorm:"size:50;uniqueIndex;not null"`
	Email        string `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	Role         string `gorm:"size:20;not null;default:user"`
	// Status reflète l'état de modération du compte. Un compte 'banned' ou
	// 'suspended' ne peut plus se connecter (cf. login). Synchronisé par le
	// user-service via l'endpoint interne lors d'une sanction.
	Status string `gorm:"size:20;not null;default:active"`
	// EmailVerifiedAt nil = email non vérifié : le login est refusé tant que
	// l'utilisateur n'a pas confirmé via le lien reçu par mail.
	EmailVerifiedAt *time.Time `gorm:"index"`
	// Les comptes antérieurs au CLUF restent null/vides après la migration.
	// Toute nouvelle inscription renseigne obligatoirement les deux champs.
	TermsAcceptedAt *time.Time `gorm:"index"`
	TermsVersion    string     `gorm:"size:20"`
	CreatedAt       time.Time  `gorm:"not null"`
	UpdatedAt       time.Time  `gorm:"not null"`
}

// authEmailVerificationModel stocke le SHA-256 du token de vérification (jamais
// le token en clair, calqué sur les refresh tokens). Token à usage unique
// (ConsumedAt) et à durée de vie courte (ExpiresAt).
type authEmailVerificationModel struct {
	ID         string     `gorm:"primaryKey;size:64"`
	UserID     string     `gorm:"size:64;index;not null"`
	TokenHash  string     `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt  time.Time  `gorm:"index;not null"`
	ConsumedAt *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"not null"`
}

// authPasswordResetModel stocke le SHA-256 du token de réinitialisation (jamais
// le token en clair, calqué sur authEmailVerificationModel). Token à usage unique
// (ConsumedAt) et à durée de vie courte (ExpiresAt).
type authPasswordResetModel struct {
	ID         string     `gorm:"primaryKey;size:64"`
	UserID     string     `gorm:"size:64;index;not null"`
	TokenHash  string     `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt  time.Time  `gorm:"index;not null"`
	ConsumedAt *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"not null"`
}

type authRefreshTokenModel struct {
	ID            string     `gorm:"primaryKey;size:64"`
	UserID        string     `gorm:"size:64;index;not null"`
	SessionID     string     `gorm:"size:64;index;not null"`
	FamilyID      string     `gorm:"size:64;index;not null"`
	RotatedFromID string     `gorm:"size:64;index"`
	TokenHash     string     `gorm:"size:128;uniqueIndex;not null"`
	ExpiresAt     time.Time  `gorm:"index;not null"`
	LastSeenAt    *time.Time `gorm:"index"`
	LastIP        string     `gorm:"size:64"`
	UserAgent     string     `gorm:"size:512"`
	RevokedAt     *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
}
