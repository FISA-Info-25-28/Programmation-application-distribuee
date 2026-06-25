package main

import (
	"errors"
	"strings"
)

// On valide la longueur APRÈS TrimSpace, pour rester cohérent avec updateProfile
// qui trimme puis remet à NULL si vide : une saisie composée uniquement d'espaces
// est une demande de vidage, pas un dépassement de longueur.
func validateProfileUpdate(req updateProfileRequest) error {
	if req.DisplayName != nil && len([]rune(strings.TrimSpace(*req.DisplayName))) > 50 {
		return errors.New("displayName must be at most 50 characters")
	}
	if req.Pronouns != nil && len([]rune(strings.TrimSpace(*req.Pronouns))) > 50 {
		return errors.New("pronouns must be at most 50 characters")
	}
	if req.Bio != nil && len([]rune(strings.TrimSpace(*req.Bio))) > 160 {
		return errors.New("bio must be at most 160 characters")
	}
	return nil
}
