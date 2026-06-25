package main

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// revocationKeyPrefix DOIT rester identique à celui écrit par l'auth-service
// (cf. auth-service/session_revocation.go). C'est le contrat partagé via Valkey.
const revocationKeyPrefix = "authrevoke:"

// revocationCheckTimeout : un Valkey lent ne doit pas ralentir chaque requête.
const revocationCheckTimeout = 200 * time.Millisecond

// sessionRevocationChecker rejette les access tokens invalidés en masse après un
// changement / une réinitialisation de mot de passe. L'auth-service écrit
// authrevoke:<userID> = instant de révocation ; tout token dont l'iat est
// antérieur est refusé, ce qui déconnecte les autres sessions sans attendre
// l'expiration naturelle du JWT (stateless, donc non révocable autrement).
type sessionRevocationChecker struct {
	rdb *redis.Client
}

// newSessionRevocationChecker construit le checker. ValkeyAddr vide => nil
// (no-op) : on retombe alors sur l'expiration naturelle des access tokens.
func newSessionRevocationChecker(cfg gatewayConfig) *sessionRevocationChecker {
	if cfg.ValkeyAddr == "" {
		log.Print("session revocation check disabled (VALKEY_ADDR empty)")
		return nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
	})
	return &sessionRevocationChecker{rdb: rdb}
}

// isRevoked indique si un token émis à issuedAt pour userID a été révoqué.
//
// Fail-open : en cas d'erreur Valkey, on n'invalide pas (cohérent avec la
// philosophie du gateway — le cache ne doit pas devenir un point de panne — et la
// fenêtre d'exposition reste bornée par la courte durée de vie de l'access token).
func (s *sessionRevocationChecker) isRevoked(userID string, issuedAt time.Time) bool {
	if s == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), revocationCheckTimeout)
	defer cancel()

	val, err := s.rdb.Get(ctx, revocationKeyPrefix+userID).Result()
	if errors.Is(err, redis.Nil) {
		return false // aucune révocation enregistrée pour cet utilisateur
	}
	if err != nil {
		log.Printf("session revocation check unavailable, allowing request: %v", err)
		return false
	}

	revokedAtNano, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		log.Printf("session revocation: borne illisible pour %s (%q): %v", userID, val, err)
		return false
	}

	// Comparaison en nanosecondes (l'auth-service écrit la borne en UnixNano) :
	// avec une résolution à la seconde, un access token émis la même seconde que
	// la révocation (iat == borne) passerait au travers. Strictement antérieur :
	// un token émis avant la borne est révoqué ; une reconnexion postérieure
	// (iat >= borne) est acceptée.
	return issuedAt.UnixNano() < revokedAtNano
}
