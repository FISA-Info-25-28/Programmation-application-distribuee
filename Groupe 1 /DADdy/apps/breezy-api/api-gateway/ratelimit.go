package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"daddy/apps/breezy-api/internal/shared"
)

// newRateLimiter construit un limiter adossé à Valkey (GCRA, atomique côté
// serveur). Retourne nil si VALKEY_ADDR n'est pas configuré → le rate limiting
// est alors désactivé proprement (cf. withRateLimit).
func newRateLimiter(cfg gatewayConfig) *redis_rate.Limiter {
	if cfg.ValkeyAddr == "" {
		log.Print("rate limiting disabled (VALKEY_ADDR empty)")
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
	})

	return redis_rate.NewLimiter(client)
}

// withRateLimit limite le débit par clé (perMinute requêtes/minute) via Valkey.
//
// Fail-open : si le limiter est nil (non configuré) ou si Valkey est injoignable,
// on laisse passer la requête. Le cache ne doit jamais devenir un point de panne
// qui coupe l'API ; un rate limiter indisponible dégrade la protection, il ne
// doit pas dégrader la disponibilité.
func withRateLimit(limiter *redis_rate.Limiter, keyFn func(*gin.Context) string, perMinute int) gin.HandlerFunc {
	if limiter == nil {
		return func(c *gin.Context) { c.Next() }
	}

	limit := redis_rate.PerMinute(perMinute)

	return func(c *gin.Context) {
		res, err := limiter.Allow(c.Request.Context(), keyFn(c), limit)
		if err != nil {
			log.Printf("rate limiter unavailable, allowing request: %v", err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))

		if res.Allowed == 0 {
			seconds := int(res.RetryAfter / time.Second)
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			shared.AbortError(c, http.StatusTooManyRequests, shared.ErrRateLimited, "rate limit exceeded")
			return
		}

		c.Next()
	}
}

// keyByIP : clé de rate limiting basée sur l'IP cliente. Pour les routes
// anonymes (login, register…) où aucun utilisateur n'est encore identifié →
// protection anti brute-force.
func keyByIP(scope string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		return scope + ":ip:" + c.ClientIP()
	}
}

// keyByUser : clé basée sur l'utilisateur authentifié (header posé par
// requireJWT). Équitable et insensible au NAT / à la rotation d'IP. Retombe sur
// l'IP si l'identité est absente (sécurité : on limite toujours quelque chose).
func keyByUser(scope string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		if uid := c.GetHeader(shared.HeaderUserID); uid != "" {
			return scope + ":user:" + uid
		}
		return scope + ":ip:" + c.ClientIP()
	}
}
