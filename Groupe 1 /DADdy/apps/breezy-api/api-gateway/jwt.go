package main

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type gatewayClaims struct {
	Role     string `json:"role"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func parseJWT(tokenString string, cfg gatewayConfig) (*gatewayClaims, bool) {
	if tokenString == "" {
		return nil, false
	}

	keys, err := buildJWTVerifyKeys(cfg)
	if err != nil {
		return nil, false
	}

	claims := &gatewayClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method == nil || token.Method.Alg() != cfg.JWTAlg {
				return nil, errors.New("invalid signing algorithm")
			}
			kid, _ := token.Header["kid"].(string)
			kid = strings.TrimSpace(kid)
			if kid == "" {
				return nil, errors.New("missing kid")
			}
			key, ok := keys[kid]
			if !ok {
				return nil, errors.New("unknown kid")
			}
			return key, nil
		},
		jwt.WithIssuer(cfg.JWTIssuer),
		jwt.WithAudience(cfg.JWTAudience),
		jwt.WithLeeway(cfg.JWTLeeway),
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{cfg.JWTAlg}),
	)
	if err != nil || !token.Valid {
		return nil, false
	}

	now := time.Now().UTC()
	if claims.IssuedAt == nil || now.Sub(claims.IssuedAt.Time) > cfg.JWTMaxTokenAge {
		return nil, false
	}
	if strings.TrimSpace(claims.ID) == "" {
		return nil, false
	}

	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Role) == "" {
		return nil, false
	}

	return claims, true
}

func buildJWTVerifyKeys(cfg gatewayConfig) (map[string]any, error) {
	alg := strings.TrimSpace(cfg.JWTAlg)
	keys := map[string]any{}

	switch alg {
	case "HS256":
		if strings.TrimSpace(cfg.JWTSecret) == "" {
			return nil, errors.New("missing JWT_SECRET")
		}
		keys[cfg.JWTActiveKID] = []byte(cfg.JWTSecret)
		if strings.TrimSpace(cfg.JWTPreviousKID) != "" && strings.TrimSpace(cfg.JWTPreviousSecret) != "" {
			keys[cfg.JWTPreviousKID] = []byte(cfg.JWTPreviousSecret)
		}
	case "RS256":
		if strings.TrimSpace(cfg.JWTPublicKeyPath) == "" {
			return nil, errors.New("missing JWT_PUBLIC_KEY_PATH")
		}
		activeKey, err := loadRSAPublicKey(cfg.JWTPublicKeyPath)
		if err != nil {
			return nil, err
		}
		keys[cfg.JWTActiveKID] = activeKey
		if strings.TrimSpace(cfg.JWTPreviousKID) != "" && strings.TrimSpace(cfg.JWTPrevPublicPath) != "" {
			previousKey, err := loadRSAPublicKey(cfg.JWTPrevPublicPath)
			if err != nil {
				return nil, err
			}
			keys[cfg.JWTPreviousKID] = previousKey
		}
	default:
		return nil, fmt.Errorf("unsupported JWT_ALG %s", alg)
	}

	return keys, nil
}

func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read rsa public key: %w", err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return nil, fmt.Errorf("parse rsa public key: %w", err)
	}
	return key, nil
}
