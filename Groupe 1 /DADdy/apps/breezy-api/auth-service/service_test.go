package main

import (
	"errors"
	"testing"
	"time"
)

func TestValidateSessionUser(t *testing.T) {
	verifiedAt := time.Now().UTC()

	tests := []struct {
		name    string
		user    authUserModel
		wantErr error
	}{
		{
			name:    "rejects unverified account",
			user:    authUserModel{Status: accountStatusActive},
			wantErr: errEmailNotVerified,
		},
		{
			name:    "rejects suspended account",
			user:    authUserModel{Status: accountStatusSuspended, EmailVerifiedAt: &verifiedAt},
			wantErr: errAccountSuspended,
		},
		{
			name:    "rejects banned account",
			user:    authUserModel{Status: accountStatusBanned, EmailVerifiedAt: &verifiedAt},
			wantErr: errAccountSuspended,
		},
		{
			name: "allows verified active account",
			user: authUserModel{Status: accountStatusActive, EmailVerifiedAt: &verifiedAt},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSessionUser(test.user)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("validateSessionUser() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
