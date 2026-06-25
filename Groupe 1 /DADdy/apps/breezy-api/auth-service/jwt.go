package main

import (
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type tokenClaims struct {
	Role     string `json:"role"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

const (
	jwtAlgHS256 = "HS256"
	jwtAlgRS256 = "RS256"
)

func prepareAuthSigningKey(cfg *authConfig) {
	alg := strings.TrimSpace(cfg.JWTAlg)
	if alg == "" {
		alg = jwtAlgHS256
	}
	cfg.JWTAlg = alg

	switch alg {
	case jwtAlgHS256:
		if strings.TrimSpace(cfg.JWTSecret) == "" {
			panic("JWT_SECRET is required for " + jwtAlgHS256)
		}
		cfg.jwtSignKey = []byte(cfg.JWTSecret)
	case jwtAlgRS256:
		if strings.TrimSpace(cfg.JWTKeyPath) == "" {
			panic("JWT_PRIVATE_KEY_PATH is required for " + jwtAlgRS256)
		}
		key, err := loadRSAPrivateKey(cfg.JWTKeyPath)
		if err != nil {
			panic(err)
		}
		cfg.jwtSignKey = key
	default:
		panic(fmt.Sprintf("unsupported JWT_ALG: %s", alg))
	}
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read rsa private key: %w", err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		return nil, fmt.Errorf("parse rsa private key: %w", err)
	}
	return key, nil
}

func signJWT(userID, role, username string, cfg authConfig, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := tokenClaims{
		Role:     role,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    cfg.JWTIssuer,
			Audience:  jwt.ClaimStrings{cfg.JWTAudience},
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        randomID("jti"),
		},
	}
	method := jwt.GetSigningMethod(cfg.JWTAlg)
	if method == nil {
		return "", fmt.Errorf("unsupported signing method %s", cfg.JWTAlg)
	}
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = cfg.JWTKID
	signed, err := token.SignedString(cfg.jwtSignKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}
