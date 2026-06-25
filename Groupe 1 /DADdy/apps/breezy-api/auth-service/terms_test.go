package main

import "testing"

func TestValidateTermsAcceptance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		accepted bool
		version  string
		wantErr  string
	}{
		{name: "accepted", accepted: true, version: currentTermsVersion},
		{name: "not accepted", version: currentTermsVersion, wantErr: termsAcceptanceError},
		{name: "missing version", accepted: true, wantErr: termsAcceptanceError},
		{name: "stale version", accepted: true, version: "2026-01-01", wantErr: termsAcceptanceError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTermsAcceptance(tt.accepted, tt.version)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateTermsAcceptance() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("validateTermsAcceptance() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
