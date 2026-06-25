package main

import "errors"

const (
	currentTermsVersion  = "2026-06-12"
	termsAcceptanceError = "current terms must be accepted"
)

func validateTermsAcceptance(accepted bool, version string) error {
	if !accepted || version != currentTermsVersion {
		return errors.New(termsAcceptanceError)
	}
	return nil
}
