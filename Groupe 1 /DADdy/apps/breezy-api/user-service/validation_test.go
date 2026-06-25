package main

import (
	"strings"
	"testing"
)

func strptr(s string) *string { return &s }

func TestValidateProfileUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     updateProfileRequest
		wantErr bool
	}{
		{name: "empty request", req: updateProfileRequest{}},
		{name: "all nil pointers", req: updateProfileRequest{}},
		{
			name: "displayName at limit",
			req:  updateProfileRequest{DisplayName: strptr(strings.Repeat("a", 50))},
		},
		{
			name:    "displayName over limit",
			req:     updateProfileRequest{DisplayName: strptr(strings.Repeat("a", 51))},
			wantErr: true,
		},
		{
			name: "displayName over limit but spaces trimmed back under",
			req:  updateProfileRequest{DisplayName: strptr("  " + strings.Repeat("a", 50) + "  ")},
		},
		{
			name: "displayName only spaces is a clear request",
			req:  updateProfileRequest{DisplayName: strptr(strings.Repeat(" ", 80))},
		},
		{
			name: "pronouns at limit",
			req:  updateProfileRequest{Pronouns: strptr(strings.Repeat("x", 50))},
		},
		{
			name:    "pronouns over limit",
			req:     updateProfileRequest{Pronouns: strptr(strings.Repeat("x", 51))},
			wantErr: true,
		},
		{
			name: "bio at limit",
			req:  updateProfileRequest{Bio: strptr(strings.Repeat("b", 160))},
		},
		{
			name:    "bio over limit",
			req:     updateProfileRequest{Bio: strptr(strings.Repeat("b", 161))},
			wantErr: true,
		},
		{
			name: "unicode counted by runes not bytes",
			req:  updateProfileRequest{DisplayName: strptr(strings.Repeat("é", 50))},
		},
		{
			name:    "unicode over limit by runes",
			req:     updateProfileRequest{DisplayName: strptr(strings.Repeat("é", 51))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateProfileUpdate(tt.req)
			if tt.wantErr && err == nil {
				t.Fatalf("validateProfileUpdate() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateProfileUpdate() = %v, want nil", err)
			}
		})
	}
}
