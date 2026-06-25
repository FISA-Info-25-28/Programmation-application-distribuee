package main

import (
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		username string
		email    string
		wantErr  string
	}{
		{name: "valid four categories", input: "Secure1!pass"},
		{name: "valid three categories no special", input: "Secure1password"},
		{name: "valid three categories no digit", input: "Secure!password"},
		{name: "valid unicode", input: "Sécurité123!"},
		{name: "too short", input: "Secure1!", wantErr: passwordLengthError},
		{name: "too long", input: "Aa1!" + strings.Repeat("x", 69), wantErr: passwordLengthError},
		{name: "only two categories", input: "lowercaseUPPER", wantErr: passwordCategoriesError},
		{name: "lowercase and digits only is two categories", input: "lowercase1234", wantErr: passwordCategoriesError},
		{name: "common password", input: commonPasswordList[0], wantErr: passwordCommonError},
		{name: "common password uppercased", input: "Password123", wantErr: passwordCommonError},
		{
			name:     "contains username",
			input:    "johndoeSecret1",
			username: "johndoe",
			wantErr:  passwordIdentifierError,
		},
		{
			name:    "contains email local part",
			input:   "Secret1johndoe",
			email:   "johndoe@example.com",
			wantErr: passwordIdentifierError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validatePassword(tt.input, tt.username, tt.email)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validatePassword() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("validatePassword() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
