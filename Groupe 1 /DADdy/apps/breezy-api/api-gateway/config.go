package main

import (
	"log"
	"net/url"
	"strings"
	"time"

	"daddy/apps/breezy-api/internal/shared"
)

type gatewayConfig struct {
	Port               string
	JWTSecret          string
	JWTPreviousSecret  string
	JWTIssuer          string
	JWTAudience        string
	JWTAlg             string
	JWTActiveKID       string
	JWTPreviousKID     string
	JWTPublicKeyPath   string
	JWTPrevPublicPath  string
	JWTLeeway          time.Duration
	JWTMaxTokenAge     time.Duration
	CORSAllowedOrigins []string
	TrustedProxies     []string

	ValkeyAddr          string
	ValkeyPassword      string
	RateLimitAuthPerMin int
	RateLimitAPIPerMin  int

	CoreTarget  *url.URL
	UserTarget  *url.URL
	AuthTarget  *url.URL
	PostTarget  *url.URL
	MsgTarget   *url.URL
	NotifTarget *url.URL
}

func loadGatewayConfig() gatewayConfig {
	jwtLeewaySeconds := shared.GetEnvIntMin("JWT_LEEWAY_SECONDS", 30, 0)
	jwtMaxTokenAgeMinutes := shared.GetEnvIntMin("JWT_MAX_TOKEN_AGE_MINUTES", 16, 1)

	return gatewayConfig{
		Port:               shared.GetEnv("PORT", "3001"),
		JWTSecret:          shared.SecretEnv("JWT_SECRET", "dev-jwt-secret"),
		JWTPreviousSecret:  shared.GetEnv("JWT_PREVIOUS_SECRET", ""),
		JWTIssuer:          shared.GetEnv("JWT_ISSUER", "daddy-auth"),
		JWTAudience:        shared.GetEnv("JWT_AUDIENCE", "daddy-api"),
		JWTAlg:             shared.GetEnv("JWT_ALG", "HS256"),
		JWTActiveKID:       shared.GetEnv("JWT_ACTIVE_KID", "v1"),
		JWTPreviousKID:     shared.GetEnv("JWT_PREVIOUS_KID", ""),
		JWTPublicKeyPath:   shared.GetEnv("JWT_PUBLIC_KEY_PATH", ""),
		JWTPrevPublicPath:  shared.GetEnv("JWT_PREVIOUS_PUBLIC_KEY_PATH", ""),
		JWTLeeway:          time.Duration(jwtLeewaySeconds) * time.Second,
		JWTMaxTokenAge:     time.Duration(jwtMaxTokenAgeMinutes) * time.Minute,
		CORSAllowedOrigins: splitCSV(shared.GetEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),

		// Proxies de confiance pour c.ClientIP(). Vide → SetTrustedProxies(nil) :
		// Gin ignore X-Forwarded-For/X-Real-IP et n'utilise que l'adresse réelle
		// de la connection, empêchant le spoofing d'IP (contournement du rate
		// limiting par IP). En prod, lister les IP/CIDR des proxies amont.
		TrustedProxies: splitCSV(shared.GetEnv("TRUSTED_PROXIES", "")),

		// Valkey (cache) + rate limiting. ValkeyAddr vide → rate limiting désactivé.
		ValkeyAddr:          shared.GetEnv("VALKEY_ADDR", ""),
		ValkeyPassword:      shared.SecretEnv("VALKEY_PASSWORD", ""),
		RateLimitAuthPerMin: shared.GetEnvIntMin("RATE_LIMIT_AUTH_PER_MIN", 10, 1),
		RateLimitAPIPerMin:  shared.GetEnvIntMin("RATE_LIMIT_API_PER_MIN", 120, 1),
		CoreTarget:          mustParseURL(shared.GetEnv("CORE_API_URL", "http://localhost:3101")),
		UserTarget:          mustParseURL(shared.GetEnv("USER_SERVICE_URL", "http://localhost:3102")),
		AuthTarget:          mustParseURL(shared.GetEnv("AUTH_SERVICE_URL", "http://localhost:3104")),
		PostTarget:          mustParseURL(shared.GetEnv("POST_SERVICE_URL", "http://localhost:3103")),
		MsgTarget:           mustParseURL(shared.GetEnv("MSG_SERVICE_URL", "http://localhost:3105")),
		NotifTarget:         mustParseURL(shared.GetEnv("NOTIF_SERVICE_URL", "http://localhost:3106")),
	}
}

// splitCSV découpe une liste CSV en valeurs non vides (espaces rognés).
func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		values = append(values, value)
	}

	return values
}

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("invalid service URL %q: %v", raw, err)
	}
	return parsed
}
