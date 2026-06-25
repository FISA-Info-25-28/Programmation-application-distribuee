package main

import (
	"time"

	"daddy/apps/breezy-api/internal/shared"
)

type authConfig struct {
	Port           string
	PostgresDSN    string
	JWTSecret      string
	JWTIssuer      string
	JWTAudience    string
	JWTAlg         string
	JWTKID         string
	JWTKeyPath     string
	TokenTTL       time.Duration
	RefreshTTL     time.Duration
	UserServiceURL string
	CoreAPIURL     string
	InternalKey    string
	jwtSignKey     any

	// Vérification email + envoi SMTP. SMTPHost vide => mailer en mode log (dev).
	SMTPHost             string
	SMTPPort             string
	SMTPUsername         string
	SMTPPassword         string
	MailFrom             string
	AppBaseURL           string
	EmailVerificationTTL time.Duration
	PasswordResetTTL     time.Duration

	// Valkey : rate limiter distribué. ValkeyAddr vide => fallback in-memory.
	ValkeyAddr     string
	ValkeyPassword string

	// OAuth2 providers. ClientID/Secret vides => provider désactivé.
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
}

func loadAuthConfig() authConfig {
	accessTokenTTLMinutes := shared.GetEnvIntMin("ACCESS_TOKEN_TTL_MINUTES", 15, 1)
	refreshTokenTTLDays := shared.GetEnvIntMin("REFRESH_TOKEN_TTL_DAYS", 30, 1)
	emailVerificationTTLHours := shared.GetEnvIntMin("EMAIL_VERIFICATION_TTL_HOURS", 24, 1)
	passwordResetTTLMinutes := shared.GetEnvIntMin("PASSWORD_RESET_TTL_MINUTES", 60, 1)

	cfg := authConfig{
		Port:           shared.GetEnv("PORT", "3104"),
		PostgresDSN:    shared.GetEnvAny([]string{"AUTH_DATABASE_URL", "DATABASE_URL"}, "postgres://postgres:postgres@localhost:5432/daddy?sslmode=disable"),
		JWTSecret:      shared.GetEnv("JWT_SECRET", "dev-jwt-secret"),
		JWTIssuer:      shared.GetEnv("JWT_ISSUER", "daddy-auth"),
		JWTAudience:    shared.GetEnv("JWT_AUDIENCE", "daddy-api"),
		JWTAlg:         shared.GetEnv("JWT_ALG", "HS256"),
		JWTKID:         shared.GetEnv("JWT_ACTIVE_KID", "v1"),
		JWTKeyPath:     shared.GetEnv("JWT_PRIVATE_KEY_PATH", ""),
		TokenTTL:       time.Duration(accessTokenTTLMinutes) * time.Minute,
		RefreshTTL:     time.Duration(refreshTokenTTLDays) * 24 * time.Hour,
		UserServiceURL: shared.GetEnv("USER_SERVICE_URL", "http://localhost:3102"),
		CoreAPIURL:     shared.GetEnv("CORE_API_URL", "http://localhost:3101"),
		InternalKey:    shared.SecretEnv("INTERNAL_API_KEY", "dev-internal-key"),

		ValkeyAddr:     shared.GetEnv("VALKEY_ADDR", ""),
		ValkeyPassword: shared.SecretEnv("VALKEY_PASSWORD", ""),

		GoogleClientID:     shared.GetEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: shared.SecretEnv("GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:     shared.GetEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: shared.SecretEnv("GITHUB_CLIENT_SECRET", ""),

		SMTPHost:             shared.GetEnv("SMTP_HOST", ""),
		SMTPPort:             shared.GetEnv("SMTP_PORT", "587"),
		SMTPUsername:         shared.GetEnv("SMTP_USERNAME", ""),
		SMTPPassword:         shared.SecretEnv("SMTP_PASSWORD", ""),
		MailFrom:             shared.GetEnv("MAIL_FROM", "Breezy <no-reply@breezy.local>"),
		AppBaseURL:           shared.GetEnv("APP_BASE_URL", "http://localhost:5173"),
		EmailVerificationTTL: time.Duration(emailVerificationTTLHours) * time.Hour,
		PasswordResetTTL:     time.Duration(passwordResetTTLMinutes) * time.Minute,
	}

	prepareAuthSigningKey(&cfg)
	return cfg
}
