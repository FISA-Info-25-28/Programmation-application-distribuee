package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Timeout par appel : si Valkey est lent/injoignable, on ne bloque pas le login.
const authRLTimeout = 200 * time.Millisecond

// newRateLimiter choisit l'implémentation selon la config : Valkey si configuré
// (distribué, qui survit aux redémarrages), sinon in-memory (fallback dev /
// instance unique). On ne désactive jamais la protection : sans Valkey on garde
// au moins le limiter mémoire.
func newRateLimiter(cfg authConfig) rateLimiter {
	if cfg.ValkeyAddr == "" {
		log.Print("auth rate limiter: in-memory (VALKEY_ADDR vide)")
		return newMemoryRateLimiter()
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
	})
	log.Printf("auth rate limiter: valkey (%s)", cfg.ValkeyAddr)
	return newValkeyRateLimiter(rdb)
}

type valkeyRateLimiter struct {
	rdb    *redis.Client
	prefix string
}

func newValkeyRateLimiter(rdb *redis.Client) *valkeyRateLimiter {
	return &valkeyRateLimiter{rdb: rdb, prefix: "authrl:"}
}

// Scripts Lua exécutés atomiquement côté Valkey : le read-modify-write reste sûr
// même avec plusieurs instances d'auth-service. L'horloge de référence est celle
// de Valkey (redis.call("TIME")) pour éviter toute dérive entre instances.

// authAllowScript : retourne 0 si la clé est bloquée, 1 sinon. Remet à zéro la
// fenêtre (échecs + blocage) si elle est expirée.
var authAllowScript = redis.NewScript(`
local key = KEYS[1]
local window = tonumber(ARGV[1])
local now = tonumber(redis.call("TIME")[1])

local blocked_till = tonumber(redis.call("HGET", key, "blocked_till") or "0")
if blocked_till > now then
  return 0
end

local window_start = tonumber(redis.call("HGET", key, "window_start") or "0")
if window_start == 0 or (now - window_start) > window then
  redis.call("HSET", key, "window_start", now, "failures", 0)
  redis.call("HDEL", key, "blocked_till")
  redis.call("EXPIRE", key, window)
end
return 1
`)

// authFailureScript : incrémente le compteur d'échecs, arme le blocage progressif
// au-delà du seuil, et retourne le nombre d'échecs courant.
var authFailureScript = redis.NewScript(`
local key = KEYS[1]
local window = tonumber(ARGV[1])
local threshold = tonumber(ARGV[2])
local step = tonumber(ARGV[3])
local cap = tonumber(ARGV[4])
local now = tonumber(redis.call("TIME")[1])

local window_start = tonumber(redis.call("HGET", key, "window_start") or "0")
local failures
if window_start == 0 or (now - window_start) > window then
  redis.call("HSET", key, "window_start", now)
  redis.call("HDEL", key, "blocked_till")
  failures = 0
else
  failures = tonumber(redis.call("HGET", key, "failures") or "0")
end

failures = failures + 1
redis.call("HSET", key, "failures", failures)

if failures >= threshold then
  local block = (failures - (threshold - 1)) * step
  if block > cap then block = cap end
  redis.call("HSET", key, "blocked_till", now + block)
end

redis.call("EXPIRE", key, window + cap)
return failures
`)

func (r *valkeyRateLimiter) Allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), authRLTimeout)
	defer cancel()

	allowed, err := authAllowScript.Run(ctx, r.rdb, []string{r.prefix + key},
		int(authRLWindow.Seconds())).Int()
	if err != nil {
		// Fail-open : un Valkey indisponible ne doit pas empêcher de se connecter.
		log.Printf("auth rate limiter (valkey) Allow indisponible, on autorise: %v", err)
		return true
	}
	return allowed == 1
}

func (r *valkeyRateLimiter) RegisterFailure(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), authRLTimeout)
	defer cancel()

	if err := authFailureScript.Run(ctx, r.rdb, []string{r.prefix + key},
		int(authRLWindow.Seconds()), authRLThreshold,
		int(authRLBlockStep.Seconds()), int(authRLBlockCap.Seconds())).Err(); err != nil {
		log.Printf("auth rate limiter (valkey) RegisterFailure: %v", err)
	}
}

func (r *valkeyRateLimiter) RegisterSuccess(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), authRLTimeout)
	defer cancel()

	if err := r.rdb.Del(ctx, r.prefix+key).Err(); err != nil {
		log.Printf("auth rate limiter (valkey) RegisterSuccess: %v", err)
	}
}
