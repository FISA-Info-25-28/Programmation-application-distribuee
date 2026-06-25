package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// Paramètres communs aux deux implémentations du rate limiter d'auth.
const (
	authRLWindow    = 15 * time.Minute // fenêtre glissante de comptage des échecs
	authRLThreshold = 5                // nb d'échecs avant déclenchement du blocage
	authRLBlockStep = 30 * time.Second // pas de blocage (progressif au-delà du seuil)
	authRLBlockCap  = 30 * time.Minute // plafond de blocage
)

// rateLimiter protège les routes sensibles (login/register/refresh) contre le
// brute-force, par couple (route, IP, compte). Deux implémentations :
//   - memoryRateLimiter : in-memory, suffisant pour une instance unique ;
//   - valkeyRateLimiter : adossé à Valkey, distribué + résistant aux redémarrages.
type rateLimiter interface {
	// Allow indique si une nouvelle tentative est autorisée pour cette clé.
	Allow(key string) bool
	// RegisterFailure comptabilise une tentative échouée (et arme le blocage).
	RegisterFailure(key string)
	// RegisterSuccess remet le compteur à zéro après une réussite.
	RegisterSuccess(key string)
}

type memoryRateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
}

type rateLimitEntry struct {
	Failures    int
	WindowStart time.Time
	BlockedTill time.Time
}

func newMemoryRateLimiter() *memoryRateLimiter {
	return &memoryRateLimiter{entries: map[string]rateLimitEntry{}}
}

func (r *memoryRateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	entry := r.entries[key]
	if !entry.BlockedTill.IsZero() && now.Before(entry.BlockedTill) {
		return false
	}
	if now.Sub(entry.WindowStart) > authRLWindow {
		entry = rateLimitEntry{WindowStart: now}
		r.entries[key] = entry
	}
	return true
}

func (r *memoryRateLimiter) RegisterFailure(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	entry := r.entries[key]
	if entry.WindowStart.IsZero() || now.Sub(entry.WindowStart) > authRLWindow {
		entry = rateLimitEntry{WindowStart: now}
	}

	entry.Failures++
	if entry.Failures >= authRLThreshold {
		block := time.Duration(entry.Failures-(authRLThreshold-1)) * authRLBlockStep
		if block > authRLBlockCap {
			block = authRLBlockCap
		}
		entry.BlockedTill = now.Add(block)
	}
	r.entries[key] = entry
}

func (r *memoryRateLimiter) RegisterSuccess(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, key)
}

func (s *authService) rateLimitKey(route, ip, subject string) string {
	normalizedIP := strings.TrimSpace(strings.ToLower(ip))
	normalizedSubject := strings.TrimSpace(strings.ToLower(subject))
	if normalizedSubject == "" {
		normalizedSubject = "anon"
	}
	hash := sha256.Sum256([]byte(normalizedSubject))
	return route + "|" + normalizedIP + "|" + hex.EncodeToString(hash[:8])
}
