package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type userProfilePayload struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// syncUserProfile crée le profil correspondant dans le user-service et le
// core-api. Appel synchrone à timeout court : si l'un des services est
// indisponible, le register échoue. Renvoie nil si la synchro est désactivée.
func (s *authService) syncUserProfile(user authUserModel) error {
	payload, err := json.Marshal(userProfilePayload{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	})
	if err != nil {
		return fmt.Errorf("marshal user profile: %w", err)
	}

	if s.cfg.UserServiceURL != "" {
		if err := s.syncToService(s.cfg.UserServiceURL+"/internal/users", payload); err != nil {
			return fmt.Errorf("user-service sync: %w", err)
		}
	}

	if s.cfg.CoreAPIURL != "" {
		if err := s.syncToService(s.cfg.CoreAPIURL+"/internal/users", payload); err != nil {
			return fmt.Errorf("core-api sync: %w", err)
		}
	}

	return nil
}

func (s *authService) syncToService(url string, payload []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", s.cfg.InternalKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 201 = créé, 409 = déjà présent (retry) : les deux sont des succès.
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	return nil
}
