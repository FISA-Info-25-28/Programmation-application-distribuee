package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// revocationKeyPrefix DOIT rester identique à celui lu par l'api-gateway
// (cf. api-gateway/revocation.go). C'est le contrat partagé via Valkey.
const revocationKeyPrefix = "authrevoke:"

const revocationWriteTimeout = 200 * time.Millisecond

// sessionRevoker publie une borne de révocation par utilisateur dans Valkey.
//
// Un JWT est stateless : révoquer les refresh tokens en base ne suffit pas, les
// access tokens déjà émis pour les AUTRES sessions restent valides jusqu'à leur
// expiration. Après un changement / une réinitialisation de mot de passe, on
// écrit donc authrevoke:<userID> = instant courant ; le gateway rejette tout
// access token émis (iat) avant cette borne, ce qui déconnecte immédiatement les
// autres sessions sans attendre l'expiration naturelle du token.
type sessionRevoker struct {
	rdb *redis.Client
	ttl time.Duration
}

// newSessionRevoker construit le revoker. ValkeyAddr vide => nil (no-op) : la
// révocation retombe alors sur l'expiration naturelle des access tokens. En dev
// avec compose, Valkey est configuré (même instance que le rate limiting).
func newSessionRevoker(cfg authConfig) *sessionRevoker {
	if cfg.ValkeyAddr == "" {
		log.Print("session revoker: désactivé (VALKEY_ADDR vide) — les autres sessions expireront avec leur access token")
		return nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
	})
	// La borne ne doit vivre que tant qu'un access token émis avant elle peut
	// encore être présenté : durée de vie de l'access token + marge (leeway/
	// dérive d'horloge). Au-delà, tous les anciens tokens sont expirés et la clé
	// peut disparaître d'elle-même.
	return &sessionRevoker{rdb: rdb, ttl: cfg.TokenTTL + 2*time.Minute}
}

// revokeUserSessions marque toutes les sessions de userID comme révoquées à
// l'instant présent. Best-effort : à appeler APRÈS le commit en base. Une erreur
// Valkey est tracée dans les logs mais ne fait pas échouer le changement de mot
// de passe (déjà persisté ; les refresh tokens sont révoqués en base de toute façon).
//
// La borne est écrite en nanosecondes (UnixNano) : avec une résolution à la
// seconde, un access token émis dans la même seconde que la révocation passerait
// au travers (iat == borne). Le gateway compare avec la même unité.
func (s *sessionRevoker) revokeUserSessions(userID string) {
	if s == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), revocationWriteTimeout)
	defer cancel()

	now := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	if err := s.rdb.Set(ctx, revocationKeyPrefix+userID, now, s.ttl).Err(); err != nil {
		log.Printf("session revoker: échec d'écriture de la borne pour %s: %v", userID, err)
	}
}
