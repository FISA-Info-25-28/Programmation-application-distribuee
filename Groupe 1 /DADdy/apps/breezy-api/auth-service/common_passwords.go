package main

import "strings"

// commonPasswords est une blocklist des mots de passe les plus répandus (fuites
// publiques, listes "rockyou" / "Have I Been Pwned" tronquées). La comparaison se
// fait en minuscules : on bloque donc aussi les variations de casse (Password,
// PASSWORD, ...). La liste est volontairement courte et embarquée pour rester sans
// dépendance ; on peut la remplacer plus tard par un appel à un service de breach
// checking (k-anonymity HIBP) sans changer l'interface de validatePassword.
var commonPasswords = map[string]struct{}{}

func init() {
	for _, p := range commonPasswordList {
		commonPasswords[p] = struct{}{}
	}
}

// isCommonPassword indique si le mot de passe figure dans la blocklist. La
// comparaison est insensible à la casse.
func isCommonPassword(password string) bool {
	_, found := commonPasswords[strings.ToLower(password)]
	return found
}

// commonPasswordList regroupe les mots de passe les plus utilisés selon les
// classements publics. Toutes les entrées sont en minuscules.
var commonPasswordList = []string{
	"123456", "123456789", "12345678", "1234567890", "1234567",
	"12345", "123123", "111111", "000000", "654321",
	"password", "password1", "password123", "passw0rd", "p@ssw0rd",
	"p@ssword", "motdepasse", "azerty", "azertyuiop", "qwerty",
	"qwertyuiop", "qwerty123", "1q2w3e4r", "1q2w3e4r5t", "zaq12wsx",
	"qazwsx", "abc123", "a1b2c3", "iloveyou", "admin",
	"administrator", "welcome", "welcome1", "login", "letmein",
	"monkey", "dragon", "sunshine", "princess", "football",
	"baseball", "superman", "batman", "trustno1", "master",
	"shadow", "michael", "jennifer", "soleil", "bonjour",
	"changeme", "secret", "default", "starwars", "whatever",
	"hello", "hello123", "test", "test123", "root",
	"toor", "user", "guest", "qwerty1", "azerty123",
	"123qwe", "qwe123", "asdfgh", "asdf1234", "zxcvbnm",
	"google", "facebook", "instagram", "snapchat", "tiktok",
}
