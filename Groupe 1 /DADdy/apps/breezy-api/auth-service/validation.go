package main

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

const (
	// passwordMinLength : seuil bas privilégié comme principal facteur de
	// robustesse (préconisation NIST 800-63B : la longueur prime sur la
	// composition). passwordMaxLength est borné par bcrypt (72 octets, le reste
	// est ignoré au hachage).
	passwordMinLength = 12
	passwordMaxLength = 72

	// minCategories : nombre minimal de catégories de caractères exigées parmi
	// minuscule / majuscule / chiffre / spécial (3 sur 4).
	minCategories = 3
)

const (
	passwordLengthError     = "password must be between 12 and 72 characters"
	passwordCategoriesError = "password must contain at least 3 of these: lowercase letter, uppercase letter, number, special character"
	passwordCommonError     = "password is too common"
	passwordIdentifierError = "password must not contain your username or email"
)

func validateRegisterInput(username, email, password string) error {
	if len(username) < 3 || len(username) > 50 {
		return errors.New("username must be between 3 and 50 characters")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("invalid email format")
	}
	return validatePassword(password, username, email)
}

// validatePassword applique la règle commune à l'inscription, à la
// réinitialisation et au changement de mot de passe :
//   - longueur entre passwordMinLength et passwordMaxLength octets (borne haute
//     imposée par bcrypt) ;
//   - au moins minCategories catégories de caractères sur 4 ;
//   - absence dans la blocklist des mots de passe les plus communs ;
//   - ne contient pas le username ni la partie locale de l'email.
//
// username et email peuvent être vides (ex. contexte sans identité connue) :
// le contrôle de containment est alors simplement ignoré.
func validatePassword(password, username, email string) error {
	// Blocklist en premier : un mot de passe notoirement faible doit être rejeté
	// comme tel, même s'il échouerait par ailleurs sur la longueur.
	if isCommonPassword(password) {
		return errors.New(passwordCommonError)
	}

	if len(password) < passwordMinLength || len(password) > passwordMaxLength {
		return errors.New(passwordLengthError)
	}

	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	categories := 0
	for _, present := range []bool{hasLower, hasUpper, hasDigit, hasSpecial} {
		if present {
			categories++
		}
	}
	if categories < minCategories {
		return errors.New(passwordCategoriesError)
	}

	if containsIdentifier(password, username, email) {
		return errors.New(passwordIdentifierError)
	}

	return nil
}

// containsIdentifier détecte si le mot de passe inclut le username ou la partie
// locale de l'email (avant le @). Comparaison insensible à la casse. Les
// identifiants de moins de 3 caractères sont ignorés pour éviter les faux
// positifs sur des fragments trop courts.
func containsIdentifier(password, username, email string) bool {
	lowerPassword := strings.ToLower(password)

	if local := emailLocalPart(email); len(local) >= 3 {
		if strings.Contains(lowerPassword, strings.ToLower(local)) {
			return true
		}
	}

	if len(username) >= 3 && strings.Contains(lowerPassword, strings.ToLower(username)) {
		return true
	}

	return false
}

func emailLocalPart(email string) string {
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}
