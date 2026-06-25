package shared

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
}

func NewHealthResponse(service string) HealthResponse {
	return HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Service:   service,
	}
}

// ───────────────────── Identité (contrat gateway → services) ─────────────────────
//
// Le gateway valide le JWT et injecte l'identité dans ces headers. Les services
// ne parsent JAMAIS le JWT : ils font confiance à ces headers. Le gateway doit
// donc supprimer toute valeur entrante de ces headers avant de poser les siennes.

const (
	HeaderUserID       = "X-User-Id"
	HeaderUserRole     = "X-User-Role"
	HeaderUserUsername = "X-User-Username"
)

type Identity struct {
	UserID   string
	Role     string
	Username string
}

// Rôles applicatifs. Le rôle voyage du JWT (signé par l'auth-service) jusqu'aux
// services via le header X-User-Role posé par le gateway.
const (
	RoleUser          = "user"
	RoleModerator     = "moderator"
	RoleAdministrator = "administrator"
)

// IsStaff indique si le rôle dispose des droits de modération (modérateur ou
// administrateur).
func IsStaff(role string) bool {
	role = strings.TrimSpace(role)
	return role == RoleModerator || role == RoleAdministrator
}

// IdentityFromContext lit l'identité injectée par le gateway.
// Retourne false si aucun userId n'est présent (requête non authentifiée).
func IdentityFromContext(c *gin.Context) (Identity, bool) {
	id := strings.TrimSpace(c.GetHeader(HeaderUserID))
	if id == "" {
		return Identity{}, false
	}
	return Identity{
		UserID:   id,
		Role:     strings.TrimSpace(c.GetHeader(HeaderUserRole)),
		Username: strings.TrimSpace(c.GetHeader(HeaderUserUsername)),
	}, true
}

// ───────────────────── Erreurs homogènes (cf. schema Error de l'OpenAPI) ─────────────────────

type ErrorCode string

const (
	ErrValidation      ErrorCode = "VALIDATION_ERROR"
	ErrUnauthenticated ErrorCode = "UNAUTHENTICATED"
	ErrForbidden       ErrorCode = "FORBIDDEN"
	ErrNotFound        ErrorCode = "NOT_FOUND"
	ErrConflict        ErrorCode = "CONFLICT"
	ErrRateLimited     ErrorCode = "RATE_LIMITED"
	ErrInternal        ErrorCode = "INTERNAL_ERROR"
)

type ErrorResponse struct {
	Error   ErrorCode `json:"error"`
	Message string    `json:"message"`
}

// AbortError répond avec le format d'erreur standardisé et interrompt la chaîne.
func AbortError(c *gin.Context, status int, code ErrorCode, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: code, Message: message})
}

// RequireIdentity garde un handler : 401 standardisé si pas d'identité, sinon
// dépose l'identité dans le contexte gin sous la clé "identity".
func RequireIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		identity, ok := IdentityFromContext(c)
		if !ok {
			AbortError(c, http.StatusUnauthorized, ErrUnauthenticated, "authentication required")
			return
		}
		c.Set("identity", identity)
		c.Next()
	}
}

// RequireRole garde un handler : 401 si non authentifié, 403 si le rôle de
// l'appelant n'est pas dans la liste autorisée. À chaîner APRÈS le gateway qui
// a posé l'identité dans les headers. Dépose aussi l'identité dans le contexte.
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = true
	}
	return func(c *gin.Context) {
		identity, ok := IdentityFromContext(c)
		if !ok {
			AbortError(c, http.StatusUnauthorized, ErrUnauthenticated, "authentication required")
			return
		}
		if !allowedSet[identity.Role] {
			AbortError(c, http.StatusForbidden, ErrForbidden, "insufficient privileges")
			return
		}
		c.Set("identity", identity)
		c.Next()
	}
}

func GetEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func GetEnvAny(keys []string, fallback string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return fallback
}

// GetEnvIntMin lit un entier >= minValue depuis l'environnement. Renvoie
// fallback si la variable est absente, vide, non parsable, ou inférieure à
// minValue. Le plancher évite qu'une valeur aberrante (TTL nul/négatif…) ne
// produise une config incohérente : on retombe alors sur le défaut sûr.
func GetEnvIntMin(key string, fallback, minValue int) int {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= minValue {
			return v
		}
	}
	return fallback
}

// IsProduction indique si le service tourne en production (APP_ENV=production).
func IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

// SecretEnv lit un secret (clé JWT, clé interne…). Un secret ne doit jamais
// avoir de valeur de repli silencieuse en production : si la variable est
// absente alors que APP_ENV=production, on panique au démarrage plutôt que de
// tourner sur un secret de dev connu (donc forgeable). Hors production, on
// retombe sur devFallback pour garder un `docker compose up` sans config.
func SecretEnv(key, devFallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	if IsProduction() {
		panic(fmt.Sprintf("environment variable %s is required when APP_ENV=production", key))
	}
	return devFallback
}

func ConnectPostgres(dsn string) (*gorm.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("postgres DSN is empty")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB, models ...any) error {
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	return nil
}
