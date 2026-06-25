package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "lowercases", in: "Alice", want: "alice"},
		{name: "strips invalid chars", in: "a.b-c!d e", want: "abcde"},
		{name: "keeps digits and underscore", in: "Bob_99", want: "bob_99"},
		{name: "empty falls back to default", in: "***", want: defaultUserRole},
		{name: "truncates to 30", in: strings.Repeat("x", 40), want: strings.Repeat("x", 30)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeUsername(tt.in); got != tt.want {
				t.Fatalf("sanitizeUsername(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestOAuthConfig(t *testing.T) {
	t.Run("google disabled without client id", func(t *testing.T) {
		s := &authService{cfg: authConfig{AppBaseURL: "http://localhost:5173"}}
		if _, err := s.oauthConfig(oauthProviderGoogle); err != errOAuthProviderDisabled {
			t.Fatalf("err = %v, want errOAuthProviderDisabled", err)
		}
	})

	t.Run("github disabled without client id", func(t *testing.T) {
		s := &authService{cfg: authConfig{AppBaseURL: "http://localhost:5173"}}
		if _, err := s.oauthConfig(oauthProviderGitHub); err != errOAuthProviderDisabled {
			t.Fatalf("err = %v, want errOAuthProviderDisabled", err)
		}
	})

	t.Run("unknown provider is disabled", func(t *testing.T) {
		s := &authService{cfg: authConfig{}}
		if _, err := s.oauthConfig("twitter"); err == nil {
			t.Fatal("unknown provider should error")
		}
	})

	t.Run("google enabled returns config with redirect", func(t *testing.T) {
		s := &authService{cfg: authConfig{AppBaseURL: "http://localhost:5173/", GoogleClientID: "gid"}}
		cfg, err := s.oauthConfig(oauthProviderGoogle)
		if err != nil {
			t.Fatalf("oauthConfig: %v", err)
		}
		if cfg.ClientID != "gid" {
			t.Errorf("ClientID = %q", cfg.ClientID)
		}
		want := "http://localhost:5173/api/auth/oauth/google/callback"
		if cfg.RedirectURL != want {
			t.Errorf("RedirectURL = %q, want %q", cfg.RedirectURL, want)
		}
	})

	t.Run("github enabled returns config", func(t *testing.T) {
		s := &authService{cfg: authConfig{AppBaseURL: "http://localhost:5173", GitHubClientID: "ghid"}}
		cfg, err := s.oauthConfig(oauthProviderGitHub)
		if err != nil {
			t.Fatalf("oauthConfig: %v", err)
		}
		if cfg.ClientID != "ghid" {
			t.Errorf("ClientID = %q", cfg.ClientID)
		}
	})
}

func TestOAuthRedirectURL(t *testing.T) {
	t.Run("disabled provider errors", func(t *testing.T) {
		s := &authService{cfg: authConfig{}}
		if _, _, err := s.oauthRedirectURL(oauthProviderGoogle); err == nil {
			t.Fatal("disabled provider should error")
		}
	})

	t.Run("enabled provider returns auth url and state", func(t *testing.T) {
		s := &authService{cfg: authConfig{AppBaseURL: "http://localhost:5173", GoogleClientID: "gid"}}
		authURL, state, err := s.oauthRedirectURL(oauthProviderGoogle)
		if err != nil {
			t.Fatalf("oauthRedirectURL: %v", err)
		}
		if len(state) != oauthStateLength {
			t.Errorf("state length = %d, want %d", len(state), oauthStateLength)
		}
		if !strings.Contains(authURL, "state="+state) || !strings.Contains(authURL, "client_id=gid") {
			t.Errorf("authURL missing state/client_id: %s", authURL)
		}
	})
}

func TestSyncToService(t *testing.T) {
	user := authUserModel{ID: "u1", Username: "alice", Email: "a@example.com", Role: "user"}

	t.Run("disabled when both URLs empty", func(t *testing.T) {
		s := &authService{cfg: authConfig{}}
		if err := s.syncUserProfile(user); err != nil {
			t.Fatalf("expected nil when sync disabled, got %v", err)
		}
	})

	t.Run("201 created is success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal-Key") != "k" {
				t.Errorf("missing internal key header")
			}
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()
		s := &authService{cfg: authConfig{UserServiceURL: srv.URL, InternalKey: "k"}}
		if err := s.syncUserProfile(user); err != nil {
			t.Fatalf("expected success on 201, got %v", err)
		}
	})

	t.Run("409 conflict is success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
		}))
		defer srv.Close()
		s := &authService{cfg: authConfig{CoreAPIURL: srv.URL, InternalKey: "k"}}
		if err := s.syncUserProfile(user); err != nil {
			t.Fatalf("expected success on 409, got %v", err)
		}
	})

	t.Run("other status is error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		s := &authService{cfg: authConfig{UserServiceURL: srv.URL, InternalKey: "k"}}
		if err := s.syncUserProfile(user); err == nil {
			t.Fatal("expected error on 500")
		}
	})

	t.Run("network error is error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()
		s := &authService{cfg: authConfig{UserServiceURL: url, InternalKey: "k"}}
		if err := s.syncUserProfile(user); err == nil {
			t.Fatal("expected error when service unreachable")
		}
	})
}
