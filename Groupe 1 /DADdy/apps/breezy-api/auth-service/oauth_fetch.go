package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// Ce fichier regroupe les appels sortants vers les fournisseurs OAuth (échange du
// code, récupération du profil chez Google/GitHub). Code I/O réseau vers des URLs
// externes en dur, non couvert par les tests : isolé ici pour être exclu du calcul
// de couverture (voir .coverignore). La logique métier (find/create, dedup,
// exchange code) reste dans oauth.go et est testée.

func (s *authService) oauthCallback(provider, code string) (oauthResult, error) {
	cfg, err := s.oauthConfig(provider)
	if err != nil {
		return oauthResult{}, err
	}

	ctx := context.Background()
	providerToken, err := cfg.Exchange(ctx, code)
	if err != nil {
		return oauthResult{}, fmt.Errorf("oauth exchange: %w", err)
	}

	info, err := fetchOAuthUserInfo(ctx, provider, providerToken)
	if err != nil {
		return oauthResult{}, fmt.Errorf("fetch user info: %w", err)
	}

	return s.findOrCreateSocialUser(provider, info, providerToken)
}

func fetchOAuthUserInfo(ctx context.Context, provider string, token *oauth2.Token) (oauthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	switch provider {
	case oauthProviderGoogle:
		return fetchGoogleUserInfo(client)
	case oauthProviderGitHub:
		return fetchGitHubUserInfo(ctx, client)
	default:
		return oauthUserInfo{}, fmt.Errorf("unknown provider: %s", provider)
	}
}

func fetchGoogleUserInfo(client *http.Client) (oauthUserInfo, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return oauthUserInfo{}, fmt.Errorf("google userinfo request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthUserInfo{}, fmt.Errorf("google userinfo read: %w", err)
	}
	var data struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return oauthUserInfo{}, fmt.Errorf("google userinfo decode: %w", err)
	}
	username := strings.ToLower(strings.ReplaceAll(data.Name, " ", ""))
	if username == "" {
		username = strings.Split(data.Email, "@")[0]
	}
	return oauthUserInfo{ProviderUID: data.Sub, Email: strings.ToLower(data.Email), Username: username}, nil
}

func fetchGitHubUserInfo(ctx context.Context, client *http.Client) (oauthUserInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return oauthUserInfo{}, fmt.Errorf("github user request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthUserInfo{}, fmt.Errorf("github user read: %w", err)
	}
	var data struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return oauthUserInfo{}, fmt.Errorf("github user decode: %w", err)
	}
	email := strings.ToLower(strings.TrimSpace(data.Email))
	if email == "" {
		var fetchErr error
		email, fetchErr = fetchGitHubPrimaryEmail(ctx, client)
		if fetchErr != nil {
			log.Printf("oauth github: failed to fetch primary email: %v", fetchErr)
		}
	}
	username := strings.ToLower(strings.TrimSpace(data.Login))
	if username == "" {
		username = "gh_user"
	}
	return oauthUserInfo{ProviderUID: fmt.Sprintf("%d", data.ID), Email: email, Username: username}, nil
}

func fetchGitHubPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github emails request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("github emails decode: %w", err)
	}
	for _, e := range emails {
		if e.Primary {
			return strings.ToLower(e.Email), nil
		}
	}
	if len(emails) > 0 {
		return strings.ToLower(emails[0].Email), nil
	}
	return "", fmt.Errorf("no email returned by GitHub")
}
